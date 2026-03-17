"""Page Bibliothèque médias — orchestrateur.

L'association média → match se fait par proximité temporelle :
- on indexe les fichiers (mtime)
- on calcule pour chaque match une fenêtre [start - tol ; end + tol]
- on associe un média au match dont la fenêtre contient son mtime

Architecture :
    - media_library_filters.py : filtres et options UI
    - media_library_render.py  : rendu grille et affichage groupé
    - media_library_data.py    : chargement et association de données
"""

from __future__ import annotations

import polars as pl
import streamlit as st

from src.ui.i18n import t
from src.ui.pages.media_library_data import (
    associate_media_to_matches as _associate_media_to_matches,
)
from src.ui.pages.media_library_data import (
    compute_match_windows as _compute_match_windows,
)
from src.ui.pages.media_library_data import (
    gamertag_from_db_path as _gamertag_from_db_path,
)
from src.ui.pages.media_library_data import (
    index_all_media as _index_all_media,
)
from src.ui.pages.media_library_data import (
    load_match_windows_from_db as _load_match_windows_from_db,
)
from src.ui.pages.media_library_data import (
    load_media_from_db as _load_media_from_db,
)
from src.ui.pages.media_library_filters import (
    coerce_dirs,
    render_media_filters,
)
from src.ui.pages.media_library_render import render_grouped_media
from src.ui.settings import AppSettings
from src.visualization._compat import DataFrameLike


def render_media_library_page(*, df_full: DataFrameLike, settings: AppSettings) -> None:
    """Rend la page Bibliothèque médias."""
    st.subheader(t("media_library_title"))

    if not settings.media_enabled:
        st.info(t("media_disabled"))
        return

    dirs = coerce_dirs(settings)
    if not dirs.screens_dir and not dirs.videos_dir:
        st.info(t("media_no_folder"))
        return

    # Récupérer le XUID du joueur actuel
    db_path = st.session_state.get("db_path", "")
    xuid_input = st.session_state.get("xuid_input", "")

    from src.app.profile import resolve_xuid
    from src.app.state import get_default_identity

    identity = get_default_identity()
    xuid = (
        resolve_xuid(xuid_input or "SpartanC", db_path, identity)
        or identity.xuid
        or identity.xuid_fallback
    )

    # Filtres et options
    filters = render_media_filters(settings, dirs, db_path)

    # Chargement des données
    media_df, using_db, windows_df = _load_media(db_path, xuid, df_full, settings)

    # Diagnostic
    if using_db and not media_df.is_empty():
        unassigned_count = media_df["match_id"].is_null().sum()
        if unassigned_count > 0:
            st.info(t("ml_unassigned_from_db", count=unassigned_count))

    if media_df.is_empty():
        st.info(t("media_no_files"))
        return

    # Appliquer les filtres
    assigned, unassigned = _apply_media_filters(media_df, filters, using_db, windows_df, settings)

    # Rendu
    render_grouped_media(
        assigned,
        unassigned,
        group_by_match=filters.group_by_match,
        show_unassigned=filters.show_unassigned,
        cols_per_row=filters.cols_per_row,
    )


def _load_media(
    db_path: str,
    xuid: str,
    df_full: DataFrameLike,
    settings: AppSettings,
) -> tuple[pl.DataFrame, bool, pl.DataFrame]:
    """Charge les médias depuis la BDD ou le disque.

    Returns:
        Tuple (assoc_df, using_db, windows_df).
    """
    media_df = pl.DataFrame()
    using_db = False
    windows_df = pl.DataFrame()

    if db_path and db_path.endswith(".duckdb"):
        gamertag = _gamertag_from_db_path(db_path)
        media_df = _load_media_from_db(db_path, xuid=xuid, gamertag=gamertag)
        using_db = not media_df.is_empty()

    # Fallback sur scan disque si BDD vide
    if media_df.is_empty():
        media_df = _index_all_media(settings)
        if not media_df.is_empty():
            windows_df = _compute_match_windows(df_full, settings)
            assoc_df = _associate_media_to_matches(media_df, windows_df)
        else:
            assoc_df = pl.DataFrame()
    else:
        assoc_df = media_df.clone()
        if "match_id" not in assoc_df.columns:
            assoc_df = assoc_df.with_columns(pl.lit(None).alias("match_id"))
        if "match_start_time" not in assoc_df.columns:
            assoc_df = assoc_df.with_columns(pl.lit(None).alias("match_start_time"))
        windows_df = _load_match_windows_from_db(db_path, xuid) if db_path else pl.DataFrame()

    return assoc_df, using_db, windows_df


def _apply_media_filters(
    assoc_df: pl.DataFrame,
    filters: MediaFilterState,  # noqa: F821
    using_db: bool,
    windows_df: pl.DataFrame,
    settings: AppSettings,
) -> tuple[pl.DataFrame, pl.DataFrame]:
    """Applique les filtres et retourne (assigned, unassigned).

    Affiche aussi les diagnostics.
    """
    from src.ui.pages.media_library_filters import MediaFilterState  # noqa: F811

    assert isinstance(filters, MediaFilterState)

    assoc_df = assoc_df.head(int(filters.max_items))

    if filters.kinds:
        assoc_df = assoc_df.filter(pl.col("kind").is_in([str(k) for k in filters.kinds]))

    if filters.name_filter.strip():
        nf = filters.name_filter.strip().lower()
        assoc_df = assoc_df.filter(
            pl.col("basename").cast(pl.Utf8).str.to_lowercase().str.contains(nf, literal=True)
        )

    assigned = assoc_df.filter(pl.col("match_id").is_not_null())
    unassigned = assoc_df.filter(pl.col("match_id").is_null())

    # Dédupliquer
    if not assigned.is_empty():
        assigned = assigned.unique(subset=["path", "match_id"], keep="first")
    if not unassigned.is_empty():
        unassigned = unassigned.unique(subset=["path"], keep="first")

    # Diagnostic unifié
    if not using_db:
        st.info(t("ml_disk_fallback"))
    elif windows_df.is_empty() and assigned.is_empty():
        st.warning(t("ml_no_match_windows"))
    elif assigned.is_empty() and not unassigned.is_empty() and using_db:
        tolerance = settings.media_tolerance_minutes
        st.warning(t("ml_no_associations", tol=tolerance))

    return assigned, unassigned
