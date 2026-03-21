"""Filtres et options UI de la bibliothèque médias."""

from __future__ import annotations

import contextlib
import os
from dataclasses import dataclass

import streamlit as st

from src.ui.i18n import t
from src.ui.pages.match_view_helpers import index_media_dir
from src.ui.settings import AppSettings


@dataclass(frozen=True)
class _MediaDirs:
    """Répertoires médias configurés."""

    screens_dir: str
    videos_dir: str


@dataclass(frozen=True)
class MediaFilterState:
    """État des filtres médias sélectionnés par l'utilisateur."""

    group_by_match: bool
    show_unassigned: bool
    cols_per_row: int
    max_items: int
    kinds: list[str]
    name_filter: str


def coerce_dirs(settings: AppSettings) -> _MediaDirs:
    """Extrait et nettoie les répertoires médias depuis les settings."""
    return _MediaDirs(
        screens_dir=settings.media_screens_dir.strip(),
        videos_dir=settings.media_videos_dir.strip(),
    )


def render_media_filters(
    settings: AppSettings,
    dirs: _MediaDirs,
    db_path: str,
) -> MediaFilterState:
    """Rend le panneau d'options médias et retourne l'état des filtres.

    Gère aussi les boutons re-scan et génération de thumbnails.
    """
    with st.expander(t("ml_options"), expanded=True):
        c1, c2, c3, c4 = st.columns([1, 1, 1, 1])
        group_by_match = c1.toggle(t("ml_group_by_match"), value=True)
        show_unassigned = c2.toggle(t("ml_show_unassociated"), value=True)
        cols_per_row = c3.slider(t("media_columns"), min_value=2, max_value=6, value=4, step=1)
        max_items = c4.slider(
            t("ml_max_media"),
            min_value=50,
            max_value=2000,
            value=400,
            step=50,
        )

        kinds = st.multiselect(
            t("ml_types"),
            options=["image", "video"],
            default=["image", "video"],
        )
        name_filter = st.text_input(t("ml_filter_filename"), value="", placeholder="ex: 2026-01")

        col_scan, col_thumbs = st.columns(2)
        with col_scan:
            _handle_rescan_button(settings, dirs, db_path)

        with col_thumbs:
            _handle_thumbnails_button(dirs, db_path)

    return MediaFilterState(
        group_by_match=group_by_match,
        show_unassigned=show_unassigned,
        cols_per_row=cols_per_row,
        max_items=max_items,
        kinds=list(kinds),
        name_filter=name_filter,
    )


def _handle_rescan_button(settings: AppSettings, dirs: _MediaDirs, db_path: str) -> None:
    """Gère le bouton re-scan des médias."""
    if not st.button(t("ml_rescan"), width="stretch"):
        return

    with contextlib.suppress(Exception):
        index_media_dir.clear()

    # Forcer re-indexation en BDD
    if "_media_indexing_started" in st.session_state:
        del st.session_state["_media_indexing_started"]

    # Lancer l'indexation manuellement si DB DuckDB disponible
    if db_path and db_path.endswith(".duckdb"):
        try:
            from pathlib import Path

            from src.data.media_indexer import MediaIndexer

            videos_path = (
                Path(dirs.videos_dir)
                if dirs.videos_dir and os.path.exists(dirs.videos_dir)
                else None
            )
            screens_path = (
                Path(dirs.screens_dir)
                if dirs.screens_dir and os.path.exists(dirs.screens_dir)
                else None
            )

            if videos_path or screens_path:
                with st.spinner(t("media_scanning")):
                    indexer = MediaIndexer(Path(db_path))
                    result = indexer.scan_and_index(
                        videos_dir=videos_path,
                        screens_dir=screens_path,
                        force_rescan=True,
                    )
                    tolerance = settings.media_tolerance_minutes
                    n_associated = indexer.associate_with_matches(tolerance_minutes=tolerance)
                    n_thumb_gen, n_thumb_err = 0, 0
                    if videos_path:
                        n_thumb_gen, n_thumb_err = indexer.generate_thumbnails_for_new(videos_path)
                    msg = t(
                        "ml_indexing_done",
                        n_new=result.n_new,
                        n_updated=result.n_updated,
                        n_associated=n_associated,
                    )
                    if n_thumb_gen or n_thumb_err:
                        msg += t(
                            "ml_indexing_thumbnails",
                            n_gen=n_thumb_gen,
                            n_err=n_thumb_err,
                        )
                    st.success(msg)
        except Exception as e:
            st.error(t("media_error_indexing", error=e))

    st.rerun()


def _handle_thumbnails_button(dirs: _MediaDirs, db_path: str) -> None:
    """Gère le bouton de génération de thumbnails."""
    if not st.button(
        t("ml_generate_thumbnails"),
        width="stretch",
        help=t("ml_generate_help"),
    ):
        return

    if (
        db_path
        and db_path.endswith(".duckdb")
        and dirs.videos_dir
        and os.path.exists(dirs.videos_dir)
    ):
        try:
            from pathlib import Path

            from src.data.media_indexer import MediaIndexer

            with st.spinner(t("media_generating_thumbnails")):
                indexer = MediaIndexer(Path(db_path))
                n_gen, n_err = indexer.generate_thumbnails_for_new(Path(dirs.videos_dir))
                st.success(
                    t("ml_thumbnails_generated", n_gen=n_gen)
                    + (t("ml_indexing_thumbnails", n_gen=0, n_err=n_err) if n_err else "")
                )
        except Exception as e:
            st.error(t("error_loading", error=e))
        st.rerun()
    else:
        st.warning(t("media_configure_video"))
