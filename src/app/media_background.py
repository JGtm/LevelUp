"""Indexation des médias en arrière-plan (non-bloquant).

Extrait de ``streamlit_app.py`` pour réduire la taille du point d'entrée.
"""

from __future__ import annotations

import logging
import os
import platform
import threading
import time
from pathlib import Path

logger = logging.getLogger(__name__)

# Guard process-level : un seul thread périodique par process, indépendant des sessions.
_PERIODIC_LOCK = threading.Lock()
_PERIODIC_STARTED: bool = False


def _index_media_for_player(
    db_file: Path, gamertag: str, captures_dir: Path, tolerance: int
) -> None:
    """Indexe les médias d'un joueur (scan, association, thumbnails).

    Le write lease est découpé en deux phases courtes pour ne pas
    bloquer _get_connection() des repos read-only pendant les
    opérations I/O longues (thumbnails).
    """
    from src.data.media_indexer import MediaIndexer
    from src.data.repositories.duckdb_repo import db_write_lease, release_db_connections

    # Phase 1 : scan + association (DB R/W, rapide)
    with db_write_lease(db_file):
        release_db_connections(db_file)
        indexer = MediaIndexer(db_file)
        result = indexer.scan_and_index(player_captures_dir=captures_dir, force_rescan=False)
        n_associated = indexer.associate_with_matches(tolerance_minutes=tolerance)

    # Phase 2 : thumbnails (DB R/W + I/O fichiers, potentiellement long)
    with db_write_lease(db_file):
        release_db_connections(db_file)
        n_thumb_gen, _n_thumb_err = indexer.generate_thumbnails_for_new(
            videos_dir=captures_dir,
            screens_dir=captures_dir,
        )
    logger.info(
        "✅ %s: %d médias, %d assoc., %d thumbs",
        gamertag,
        result.n_new + result.n_updated,
        n_associated,
        n_thumb_gen,
    )


def _index_media_legacy(db_path: str, videos_dir: str, screens_dir: str, tolerance: int) -> None:
    """Indexe les médias en mode legacy (deux dossiers globaux, DB courante)."""
    from src.data.media_indexer import MediaIndexer
    from src.data.repositories.duckdb_repo import db_write_lease, release_db_connections

    videos_path = Path(videos_dir) if videos_dir and os.path.exists(videos_dir) else None
    screens_path = Path(screens_dir) if screens_dir and os.path.exists(screens_dir) else None
    if not videos_path and not screens_path:
        logger.warning("Aucun dossier média valide trouvé")
        return

    legacy_db_file = Path(db_path)
    # Phase 1 : scan + association
    with db_write_lease(legacy_db_file):
        release_db_connections(legacy_db_file)
        indexer = MediaIndexer(legacy_db_file)
        result = indexer.scan_and_index(
            videos_dir=videos_path,
            screens_dir=screens_path,
            force_rescan=False,
        )
        n_associated = indexer.associate_with_matches(tolerance_minutes=tolerance)

    # Phase 2 : thumbnails (write lease séparé)
    with db_write_lease(legacy_db_file):
        release_db_connections(legacy_db_file)
        n_thumb_gen, _n_thumb_err = indexer.generate_thumbnails_for_new(
            videos_dir=videos_path,
            screens_dir=screens_path,
        )
    logger.info(
        "✅ Scan: %d scannés, %d assoc., %d thumbs",
        result.n_scanned,
        n_associated,
        n_thumb_gen,
    )


def _index_with_retry(db_file: Path, gamertag: str, captures_dir: Path, tolerance: int) -> None:
    """Indexe un joueur avec 3 tentatives (erreurs OS transitoires Windows)."""
    last_err: Exception | None = None
    for attempt in range(3):
        try:
            _index_media_for_player(db_file, gamertag, captures_dir, tolerance)
            if attempt > 0:
                logger.info(
                    "✅ Indexation médias %s réussie après %d tentative(s)", gamertag, attempt + 1
                )
            return
        except Exception as err:
            last_err = err
            time.sleep((attempt + 1) * 0.5)
    logger.warning("⏭️ Indexation médias %s ignorée: %s", gamertag, last_err)


def background_media_indexing(settings, db_path: str) -> None:
    """Lance l'indexation périodique des médias (une seule fois par process).

    Dossier par joueur : base_dir/{gamertag}/. Indexe tous les joueurs connus.
    Le thread worker boucle toutes les ``media_indexing_interval_hours`` heures.
    Si l'intervalle vaut 0, il tourne une seule fois.
    """
    global _PERIODIC_STARTED  # noqa: PLW0603

    if not settings.media_enabled:
        logger.debug("Indexation médias désactivée dans les paramètres")
        return

    base_dir = settings.media_captures_base_dir.strip()
    if not base_dir:
        videos_dir = settings.media_videos_dir.strip()
        screens_dir = settings.media_screens_dir.strip()
        if not videos_dir and not screens_dir:
            logger.debug("Aucun dossier média configuré - indexation ignorée")
            return
    else:
        videos_dir = screens_dir = ""

    if not db_path or not db_path.endswith(".duckdb"):
        logger.debug("DB non DuckDB ou invalide - indexation ignorée")
        return

    base_path = Path(base_dir) if base_dir else None

    # Guard process-level — un seul thread/watcher par process, quel que soit l'OS
    with _PERIODIC_LOCK:
        if _PERIODIC_STARTED:
            logger.debug("Indexation médias déjà active (process-level guard)")
            return
        _PERIODIC_STARTED = True

    # Sur Linux + dossier base configuré + watcher activé → inotify (zéro polling)
    if (
        platform.system() == "Linux"
        and base_path is not None
        and base_path.exists()
        and getattr(settings, "media_watcher_enabled", True)
    ):
        from src.app.media_watcher import start_media_watcher

        start_media_watcher(base_path, settings)
        logger.info("👁️ Mode indexation médias : watcher inotify (Linux)")
        return

    # Windows ou mode legacy (deux dossiers séparés) → thread périodique

    interval_hours: int = getattr(settings, "media_indexing_interval_hours", 4)
    logger.info(
        "🚀 Démarrage indexation médias (intervalle: %dh)",
        interval_hours if interval_hours > 0 else 0,
    )

    def worker() -> None:
        while True:
            try:
                tolerance = settings.media_tolerance_minutes
                if base_path is not None and base_path.exists():
                    _index_all_players(base_path, tolerance)
                else:
                    _index_media_legacy(db_path, videos_dir, screens_dir, tolerance)
                logger.info("✅ Indexation médias terminée")
            except Exception as e:
                logger.exception("❌ Erreur indexation médias: %s", e)

            if interval_hours <= 0:
                break
            logger.debug("⏳ Prochaine indexation médias dans %dh", interval_hours)
            time.sleep(interval_hours * 3600)

    thread = threading.Thread(target=worker, daemon=True, name="media-indexer")
    thread.start()


def reindex_media_after_sync(settings, db_path: str) -> None:
    """Lance un scan one-shot des médias après un sync (thread indépendant).

    N'interfère pas avec le thread périodique (_PERIODIC_STARTED).
    Ignoré si media_enabled=False ou media_reindex_after_sync=False.
    """
    if not settings.media_enabled:
        return
    if not getattr(settings, "media_reindex_after_sync", True):
        return

    base_dir = settings.media_captures_base_dir.strip()
    if not base_dir:
        videos_dir = settings.media_videos_dir.strip()
        screens_dir = settings.media_screens_dir.strip()
        if not videos_dir and not screens_dir:
            return
    else:
        videos_dir = screens_dir = ""

    if not db_path or not db_path.endswith(".duckdb"):
        return

    logger.info("🔄 Réindexation médias post-sync démarrée")

    def worker() -> None:
        try:
            tolerance = settings.media_tolerance_minutes
            base_path = Path(base_dir) if base_dir else None
            if base_path is not None and base_path.exists():
                _index_all_players(base_path, tolerance)
            else:
                _index_media_legacy(db_path, videos_dir, screens_dir, tolerance)
            logger.info("✅ Réindexation médias post-sync terminée")
        except Exception as e:
            logger.exception("❌ Erreur réindexation post-sync: %s", e)

    threading.Thread(target=worker, daemon=True, name="media-indexer-post-sync").start()


def _index_all_players(base_path: Path, tolerance: int) -> None:
    """Indexe les médias de tous les joueurs ayant un dossier captures."""
    from src.utils.paths import PLAYER_DB_FILENAME, PLAYERS_DIR

    for player_dir in sorted(PLAYERS_DIR.iterdir(), key=lambda p: p.name):
        if not player_dir.is_dir():
            continue
        db_file = player_dir / PLAYER_DB_FILENAME
        if not db_file.exists():
            continue
        gamertag = player_dir.name
        player_captures = base_path / gamertag
        if not player_captures.exists():
            continue
        _index_with_retry(db_file, gamertag, player_captures, tolerance)
