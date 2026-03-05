"""Indexation des médias en arrière-plan (non-bloquant).

Extrait de ``streamlit_app.py`` pour réduire la taille du point d'entrée.
"""

from __future__ import annotations

import logging
import os
import threading

import streamlit as st

logger = logging.getLogger(__name__)


def background_media_indexing(settings, db_path: str) -> None:  # noqa: C901, PLR0915
    """Lance l'indexation des médias en arrière-plan (non-bloquant).

    Dossier par joueur : base_dir/{gamertag}/. Indexe tous les joueurs connus.
    """
    if not bool(getattr(settings, "media_enabled", True)):
        logger.debug("Indexation médias désactivée dans les paramètres")
        return

    base_dir = str(getattr(settings, "media_captures_base_dir", "") or "").strip()
    # Fallback legacy
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

    def worker():  # noqa: C901, PLR0912, PLR0915
        try:
            from pathlib import Path

            from src.data.media_indexer import MediaIndexer
            from src.utils.paths import PLAYER_DB_FILENAME, PLAYERS_DIR

            def _index_media_for_player(
                db_file: Path, gamertag: str, captures_dir: Path, tolerance: int
            ) -> None:
                """Indexe les médias d'un joueur (scan, association, thumbnails)."""
                indexer = MediaIndexer(db_file)
                result = indexer.scan_and_index(
                    player_captures_dir=captures_dir,
                    force_rescan=False,
                )
                n_associated = indexer.associate_with_matches(tolerance_minutes=tolerance)
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

            tolerance = int(getattr(settings, "media_tolerance_minutes", 5) or 5)
            base_path = Path(base_dir) if base_dir else None

            if base_path is not None and base_path.exists():
                # Nouvelle logique : indexer tous les joueurs ayant base_dir/gamertag
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
                    # Libérer TOUTES les connexions read_only pour éviter le conflit
                    # DuckDB "different configuration".
                    try:
                        from src.data.repositories.duckdb_repo import (
                            release_all_db_connections,
                        )

                        n_closed = release_all_db_connections()
                        if n_closed > 0:
                            logger.debug(
                                "🔓 %d connexion(s) libérée(s) avant indexation %s",
                                n_closed,
                                gamertag,
                            )
                    except Exception:
                        pass

                    # Retry loop : 3 tentatives avec délai croissant
                    last_err: Exception | None = None
                    for _attempt in range(3):
                        try:
                            _index_media_for_player(db_file, gamertag, player_captures, tolerance)
                            last_err = None
                            break
                        except Exception as player_err:
                            last_err = player_err
                            if "different configuration" in str(player_err).lower():
                                import time as _time

                                try:
                                    from src.data.repositories.duckdb_repo import (
                                        release_all_db_connections,
                                    )

                                    release_all_db_connections()
                                except Exception:
                                    pass
                                _time.sleep((_attempt + 1) * 0.5)
                            else:
                                break  # Erreur non liée au conflit → pas de retry
                    if last_err is not None:
                        if "different configuration" in str(last_err).lower():
                            logger.info(
                                "⏭️ Indexation médias %s ignorée (DB occupée par Streamlit)",
                                gamertag,
                            )
                        else:
                            logger.warning(
                                "⏭️ Indexation médias %s ignorée: %s",
                                gamertag,
                                last_err,
                            )
            else:
                # Legacy : deux dossiers globaux, DB courante uniquement
                videos_path = (
                    Path(videos_dir) if videos_dir and os.path.exists(videos_dir) else None
                )
                screens_path = (
                    Path(screens_dir) if screens_dir and os.path.exists(screens_dir) else None
                )
                if not videos_path and not screens_path:
                    logger.warning("Aucun dossier média valide trouvé")
                    return
                indexer = MediaIndexer(Path(db_path))
                result = indexer.scan_and_index(
                    videos_dir=videos_path,
                    screens_dir=screens_path,
                    force_rescan=False,
                )
                n_associated = indexer.associate_with_matches(tolerance_minutes=tolerance)
                n_thumb_gen, n_thumb_err = indexer.generate_thumbnails_for_new(
                    videos_dir=videos_path,
                    screens_dir=screens_path,
                )
                logger.info(
                    "✅ Scan: %d scannés, %d assoc., %d thumbs",
                    result.n_scanned,
                    n_associated,
                    n_thumb_gen,
                )
            logger.info("✅ Indexation médias terminée")
        except Exception as e:
            logger.exception("❌ Erreur indexation médias: %s", e)

    thread = threading.Thread(target=worker, daemon=True, name="media-indexer")
    thread.start()
