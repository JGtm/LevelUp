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
    load_df_optimized,
)
from src.ui.vectorize_helpers import build_mapping
from src.utils.db import is_duckdb_v4_path as _is_duckdb_v4_path

if TYPE_CHECKING:
    pass

logger = logging.getLogger(__name__)


_FRIEND_DF_EMPTY_SCHEMA: dict[str, pl.PolarsDataType] = {
    "match_id": pl.Utf8,
    "start_time": pl.Datetime,
    "playlist_name": pl.Utf8,
    "pair_name": pl.Utf8,
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
    """Traduit les colonnes playlist_name et pair_name."""
    _pl_map = build_mapping(dfr["playlist_name"], translate_playlist_name)
    _pair_map = build_mapping(dfr["pair_name"], translate_pair_name)
    return dfr.with_columns(
        pl.col("playlist_name").replace_strict(
            _pl_map, default=pl.col("playlist_name"), return_dtype=pl.Utf8
        ),
        pl.col("pair_name").replace_strict(
            _pair_map, default=pl.col("pair_name"), return_dtype=pl.Utf8
        ),
    )


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

    # DuckDB v4 : rows est une list[str] de match_ids — requête shared DB
    if _is_duckdb_v4_path(db_path) and isinstance(rows[0], str):
        return _build_friend_df_from_match_ids_v4(
            db_path, self_xuid, friend_xuid, list(rows), same_team_only, tz_name
        )

    # Chemin legacy : rows contient des objets avec attributs .same_team, .match_id…
    if same_team_only:
        rows = [r for r in rows if r.same_team]
    if not rows:
        return pl.DataFrame(schema=_FRIEND_DF_EMPTY_SCHEMA)

    dfr = pl.DataFrame(
        {
            "match_id": [r.match_id for r in rows],
            "start_time": [r.start_time for r in rows],
            "playlist_name": [translate_playlist_name(r.playlist_name) for r in rows],
            "pair_name": [translate_pair_name(r.pair_name) for r in rows],
            "same_team": [r.same_team for r in rows],
            "my_team_id": [r.my_team_id for r in rows],
            "my_outcome": [r.my_outcome for r in rows],
            "friend_team_id": [r.friend_team_id for r in rows],
            "friend_outcome": [r.friend_outcome for r in rows],
        }
    )
    # Conversion timezone : UTC → tz_name → naïf
    try:
        dfr = dfr.with_columns(
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
    return dfr.sort("start_time", descending=True)


# =============================================================================
# Fonctions utilisant l'architecture hybride (Phase 2+)
# =============================================================================


def _get_repository_mode() -> str:
    """Récupère le mode de repository depuis les settings."""
    try:
        settings = st.session_state.get("app_settings")
        if settings and hasattr(settings, "repository_mode"):
            return str(settings.repository_mode).lower()
    except Exception:
        pass
    return "duckdb"


def _is_duckdb_analytics_enabled() -> bool:
    """Vérifie si les analytics DuckDB sont activées."""
    try:
        settings = st.session_state.get("app_settings")
        if settings and hasattr(settings, "enable_duckdb_analytics"):
            return bool(settings.enable_duckdb_analytics)
    except Exception:
        pass
    return False


@st.cache_data(show_spinner=False)
def load_df_hybrid(
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None = None,
    include_firefight: bool = True,
) -> pl.DataFrame:
    """Charge les matchs via le système hybride (Parquet + DuckDB).

    Utilise le DataRepository configuré selon le mode dans les settings.
    Fallback automatique sur legacy si le mode hybride échoue.

    Args:
        db_path: Chemin vers la DB.
        xuid: XUID du joueur.
        db_key: Clé de cache (mtime, size).
        include_firefight: Inclure les matchs PvE.

    Returns:
        DataFrame Polars enrichi avec toutes les colonnes calculées.
    """
    _ = db_key  # Utilisé pour invalidation du cache Streamlit

    try:
        from src.data.integration import get_repository_mode_from_settings, load_matches_polars

        mode = get_repository_mode_from_settings()

        # Polars natif — plus de roundtrip Pandas→Polars
        df_pl = load_matches_polars(
            db_path,
            xuid,
            include_firefight=include_firefight,
            mode=mode,
        )

        if not df_pl.is_empty():
            return df_pl

        # Fallback sur legacy si vide (pas de données Parquet)
        return load_df_optimized(db_path, xuid, db_key=db_key, include_firefight=include_firefight)

    except ImportError:
        # Module d'intégration non disponible, utiliser legacy
        return load_df_optimized(db_path, xuid, db_key=db_key, include_firefight=include_firefight)
    except Exception:
        # Erreur inattendue, fallback sur legacy
        return load_df_optimized(db_path, xuid, db_key=db_key, include_firefight=include_firefight)


# 8bis.A4 : TTL supprimé, l'invalidation se fait via db_key (mtime + size)
@st.cache_data(show_spinner=False)
def cached_get_global_stats_duckdb(
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None = None,
) -> dict | None:
    """Récupère les stats globales via DuckDB (haute performance).

    Utilise le QueryEngine pour des agrégations ultra-rapides sur Parquet.
    Retourne None si DuckDB n'est pas disponible ou pas de données.
    """
    if not _is_duckdb_analytics_enabled():
        return None

    try:
        from src.data.integration import check_hybrid_available, get_analytics_for_ui

        if not check_hybrid_available(db_path, xuid):
            return None

        engine, analytics = get_analytics_for_ui(db_path, xuid)
        try:
            stats = analytics.get_global_stats()
            return {
                "total_matches": stats.total_matches,
                "total_kills": stats.total_kills,
                "total_deaths": stats.total_deaths,
                "total_assists": stats.total_assists,
                "total_time_hours": stats.total_time_hours,
                "avg_kda": stats.avg_kda,
                "avg_accuracy": stats.avg_accuracy,
                "win_rate": stats.win_rate,
                "loss_rate": stats.loss_rate,
                "wins": stats.wins,
                "losses": stats.losses,
            }
        finally:
            engine.close()
    except Exception:
        return None


# 8bis.A4 : TTL supprimé, l'invalidation se fait via db_key (mtime + size)
@st.cache_data(show_spinner=False)
def cached_get_kda_trend_duckdb(
    db_path: str,
    xuid: str,
    window_size: int = 20,
    last_n: int = 500,
    db_key: tuple[int, int] | None = None,
) -> list[dict] | None:
    """Récupère l'évolution du KDA via DuckDB (haute performance).

    Utilise le TrendAnalyzer pour calculer des moyennes mobiles
    ultra-rapidement sur les fichiers Parquet.
    """
    if not _is_duckdb_analytics_enabled():
        return None

    try:
        from src.data.integration import check_hybrid_available, get_trends_for_ui

        if not check_hybrid_available(db_path, xuid):
            return None

        engine, trends = get_trends_for_ui(db_path, xuid)
        try:
            return trends.get_rolling_kda(window_size=window_size, last_n=last_n)
        finally:
            engine.close()
    except Exception:
        return None


# 8bis.A4 : TTL supprimé, l'invalidation se fait via db_key (mtime + size)
@st.cache_data(show_spinner=False)
def cached_get_performance_by_map_duckdb(
    db_path: str,
    xuid: str,
    min_matches: int = 3,
    db_key: tuple[int, int] | None = None,
) -> list[dict] | None:
    """Récupère les performances par carte via DuckDB."""
    if not _is_duckdb_analytics_enabled():
        return None

    try:
        from src.data.integration import check_hybrid_available, get_analytics_for_ui

        if not check_hybrid_available(db_path, xuid):
            return None

        engine, analytics = get_analytics_for_ui(db_path, xuid)
        try:
            return analytics.get_performance_by_map(min_matches=min_matches)
        finally:
            engine.close()
    except Exception:
        return None


@st.cache_data(show_spinner=False, ttl=3600)
def cached_get_migration_status(
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None = None,
) -> dict:
    """Récupère l'état de la migration vers le système hybride."""
    try:
        from src.data.integration import get_migration_status

        return get_migration_status(db_path, xuid)
    except Exception as e:
        return {
            "error": str(e),
            "legacy_count": 0,
            "hybrid_count": 0,
            "progress_percent": 0,
            "is_complete": False,
        }
