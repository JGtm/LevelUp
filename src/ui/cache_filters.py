"""Fonctions de cache Streamlit — Transformations, analytics et pagination.

Ce module regroupe les fonctions @st.cache_data qui effectuent
des calculs de sessions, transformations, agrégations DuckDB analytics,
et pagination.

Extrait de cache.py lors du Sprint 17 (découpage <800L).
Sous-modules : _cache_sessions.py, _cache_loading.py.
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING

import polars as pl
import streamlit as st

from src.ui import translate_pair_name, translate_playlist_name

# Re-exports depuis sous-modules
from src.ui._cache_loading import (  # noqa: F401
    cached_get_match_count_duckdb,
    cached_load_matches_paginated,
    cached_load_recent_matches,
)
from src.ui._cache_sessions import cached_compute_sessions_db  # noqa: F401
from src.ui.cache_loaders import (
    cached_query_matches_with_friend,
)
from src.ui.vectorize_helpers import build_mapping

if TYPE_CHECKING:
    pass

logger = logging.getLogger(__name__)


_FRIEND_DF_EMPTY_SCHEMA: dict[str, pl.PolarsDataType] = {
    "match_id": pl.Utf8,
    "start_time": pl.Datetime,
    "playlist_name": pl.Utf8,
    "playlist_name_fr": pl.Utf8,
    "pair_name": pl.Utf8,
    "pair_name_fr": pl.Utf8,
    "map_name": pl.Utf8,
    "same_team": pl.Boolean,
    "my_team_id": pl.Int64,
    "my_outcome": pl.Utf8,
    "friend_team_id": pl.Int64,
    "friend_outcome": pl.Utf8,
}


def _build_friend_df_from_match_ids_v4(  # noqa: PLR0913
    db_path: str,
    self_xuid: str,
    friend_xuid: str,
    match_ids: list[str],
    same_team_only: bool,
    tz_name: str = "Europe/Paris",
) -> pl.DataFrame:
    """Construit le DataFrame des matchs partagés pour DuckDB v4.

    Requête shared.match_registry + shared.match_participants à partir
    d'une liste de match_ids (list[str]) et applique la conversion UTC→Paris.
    """
    try:
        from src.ui._cache_core import get_cached_repository_st

        repo = get_cached_repository_st(db_path, self_xuid)
        dfr = repo.load_friend_match_details(friend_xuid, match_ids)
    except Exception:
        logger.debug("_build_friend_df_from_match_ids_v4 erreur", exc_info=True)
        return pl.DataFrame(schema=_FRIEND_DF_EMPTY_SCHEMA)

    if dfr.is_empty():
        return pl.DataFrame(schema=_FRIEND_DF_EMPTY_SCHEMA)

    if same_team_only:
        dfr = dfr.filter(pl.col("same_team") == True)  # noqa: E712

    if dfr.is_empty():
        return pl.DataFrame(schema=_FRIEND_DF_EMPTY_SCHEMA)

    # Traduction des libellés playlist / pair
    dfr = _translate_playlist_pair_columns(dfr)

    # Conversion timezone UTC → tz_name → naïve
    dfr = _convert_start_time_timezone(dfr, tz_name)

    return dfr.sort("start_time", descending=True)


def _translate_playlist_pair_columns(dfr: pl.DataFrame) -> pl.DataFrame:
    """Traduit les colonnes playlist_name et pair_name en utilisant les colonnes FR si disponibles."""
    if "playlist_name_fr" in dfr.columns:
        dfr = dfr.with_columns(
            pl.coalesce([pl.col("playlist_name_fr"), pl.col("playlist_name")]).alias("playlist_name")
        )
    else:
        _pl_map = build_mapping(dfr["playlist_name"], translate_playlist_name)
        dfr = dfr.with_columns(
            pl.col("playlist_name").replace_strict(
                _pl_map, default=pl.col("playlist_name"), return_dtype=pl.Utf8
            )
        )
    if "pair_name_fr" in dfr.columns:
        dfr = dfr.with_columns(
            pl.coalesce([pl.col("pair_name_fr"), pl.col("pair_name")]).alias("pair_name")
        )
    else:
        _pair_map = build_mapping(dfr["pair_name"], translate_pair_name)
        dfr = dfr.with_columns(
            pl.col("pair_name").replace_strict(
                _pair_map, default=pl.col("pair_name"), return_dtype=pl.Utf8
            )
        )
    return dfr


def _convert_start_time_timezone(dfr: pl.DataFrame, tz_name: str) -> pl.DataFrame:
    """Convertit start_time de UTC vers le fuseau local (naïf)."""
    try:
        return dfr.with_columns(
            pl.col("start_time")
            .cast(pl.Datetime("us", "UTC"))
            .dt.convert_time_zone(tz_name)
            .dt.replace_time_zone(None)
        )
    except Exception:
        import contextlib

        with contextlib.suppress(Exception):
            dfr = dfr.with_columns(
                pl.col("start_time")
                .dt.replace_time_zone("UTC")
                .dt.convert_time_zone(tz_name)
                .dt.replace_time_zone(None)
            )
        return dfr

    return dfr.sort("start_time", descending=True)


@st.cache_data(show_spinner=False)
def cached_friend_matches_df(  # noqa: PLR0913
    db_path: str,
    self_xuid: str,
    friend_xuid: str,
    same_team_only: bool,
    db_key: tuple[int, int] | None,
    tz_name: str = "Europe/Paris",
) -> pl.DataFrame:
    """Retourne un DataFrame des matchs joués avec un ami (cache)."""
    rows = cached_query_matches_with_friend(db_path, self_xuid, friend_xuid, db_key=db_key)
    if not rows:
        return pl.DataFrame(schema=_FRIEND_DF_EMPTY_SCHEMA)

    return _build_friend_df_from_match_ids_v4(
        db_path, self_xuid, friend_xuid, list(rows), same_team_only, tz_name
    )
