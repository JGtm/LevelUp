"""Indexation des médias en arrière-plan (non-bloquant).

Extrait de ``streamlit_app.py`` pour réduire la taille du point d'entrée.
"""

from __future__ import annotations

import logging
import os
import threading
import time
from pathlib import Path

import streamlit as st

logger = logging.getLogger(__name__)


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
            videos_dir=captures_dir, screens_dir=captures_dir,
        )
    logger.info(
        "✅ %s: %d médias, %d assoc., %d thumbs",
        gamertag, result.n_new + result.n_updated, n_associated, n_thumb_gen,
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
            videos_dir=videos_path, screens_dir=screens_path, force_rescan=False,
        )
        n_associated = indexer.associate_with_matches(tolerance_minutes=tolerance)

    # Phase 2 : thumbnails (write lease séparé)
    with db_write_lease(legacy_db_file):
        release_db_connections(legacy_db_file)
        n_thumb_gen, _n_thumb_err = indexer.generate_thumbnails_for_new(
            videos_dir=videos_path, screens_dir=screens_path,
        )
    logger.info(
        "✅ Scan: %d scannés, %d assoc., %d thumbs",
        result.n_scanned, n_associated, n_thumb_gen,
    )


def _index_with_retry(db_file: Path, gamertag: str, captures_dir: Path, tolerance: int) -> None:
    """Indexe un joueur avec 3 tentatives (erreurs OS transitoires Windows)."""
    last_err: Exception | None = None
    for attempt in range(3):
        try:
            _index_media_for_player(db_file, gamertag, captures_dir, tolerance)
            return
        except Exception as err:
            last_err = err
            time.sleep((attempt + 1) * 0.5)
    logger.warning("⏭️ Indexation médias %s ignorée: %s", gamertag, last_err)


def background_media_indexing(settings, db_path: str) -> None:
    """Lance l'indexation des médias en arrière-plan (non-bloquant).

    Dossier par joueur : base_dir/{gamertag}/. Indexe tous les joueurs connus.
    """
    if not bool(getattr(settings, "media_enabled", True)):
        logger.debug("Indexation médias désactivée dans les paramètres")
        return

    base_dir = str(getattr(settings, "media_captures_base_dir", "") or "").strip()
    if not base_dir:
        videos_dir = str(getattr(settings, "media_videos_dir", "") or "").strip()
        screens_dir = str(getattr(settings, "media_screens_dir", "") or "").strip()
        if not videos_dir and not screens_dir:
            logger.debug("Aucun dossier média configuré - indexation ignorée")
            return
    else:
        videos_dir = screens_dir = ""

    if not db_path or not db_path.endswith(".duckdb"):
        logger.debug("DB non DuckDB ou invalide - indexation ignorée")
        return

    if st.session_state.get("_media_indexing_started"):
        logger.debug("Indexation médias déjà démarrée dans cette session")
        return

    st.session_state["_media_indexing_started"] = True
    logger.info("🚀 Démarrage indexation médias en arrière-plan")

    def worker() -> None:
        try:
            tolerance = int(getattr(settings, "media_tolerance_minutes", 5) or 5)
            base_path = Path(base_dir) if base_dir else None

            if base_path is not None and base_path.exists():
                _index_all_players(base_path, tolerance)
            else:
                _index_media_legacy(db_path, videos_dir, screens_dir, tolerance)
            logger.info("✅ Indexation médias terminée")
        except Exception as e:
            logger.exception("❌ Erreur indexation médias: %s", e)

    thread = threading.Thread(target=worker, daemon=True, name="media-indexer")
    thread.start()


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
