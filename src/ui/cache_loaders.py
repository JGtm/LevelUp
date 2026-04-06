"""Fonctions de cache Streamlit — Chargement atomique depuis DuckDB.

Ce module regroupe les fonctions de chargement et enrichissement des matchs.
Les requêtes unitaires cachées sont dans ``_cache_queries.py``,
l'infrastructure partagée (repository, constantes) dans ``_cache_core.py``.

Extrait de cache.py lors du Sprint 17 (découpage <800L).
Redécoupé en 3 modules pour respecter la limite de 500L.
"""

from __future__ import annotations

import logging

import polars as pl
import streamlit as st

from src.ui.tz import get_tz_name as _get_tz_name  # résolution dynamique depuis settings
from src.utils.db import is_duckdb_v4_path as _is_duckdb_v4_path

# ─── Réexports depuis _cache_core ──────────────────────────────────────────
from ._cache_core import (  # noqa: F401
    COLUMNS_COMMON,
    COLUMNS_COMPUTED,
    PARIS_TZ_NAME,
    _resolve_player_xuid,
    db_cache_key,
    get_cached_repository_st,
)

# ─── Réexports depuis _cache_queries ───────────────────────────────────────
from ._cache_queries import (  # noqa: F401
    cached_get_cache_stats,
    cached_get_match_skill_rank,
    cached_has_cache_tables,
    cached_list_local_dbs,
    cached_list_other_xuids,
    cached_list_top_teammates,
    cached_load_highlight_events_for_match,
    cached_load_match_medals_for_player,
    cached_load_match_player_gamertags,
    cached_load_match_rosters,
    cached_load_player_match_result,
    cached_load_top_medals,
    cached_query_matches_with_friend,
    cached_same_team_match_ids_with_friend,
    clear_app_caches,
    top_medals_smart,
)

logger = logging.getLogger(__name__)


# ─── Chargement des matchs ─────────────────────────────────────────────────


def _load_matches_duckdb_v4(db_path: str, include_firefight: bool = True) -> list:
    """Charge les matchs depuis une DB DuckDB v4 (legacy — retourne MatchRow).

    Préférer _load_matches_duckdb_v4_polars() pour le chemin optimisé.
    Utilise le repository caché (v5.1 perf) pour éviter les reconnexions.
    """
    from src.ui._cache_core import SharedDBUnavailableError

    try:
        player_xuid = _resolve_player_xuid(db_path)
        if not player_xuid:
            raise SharedDBUnavailableError(
                "XUID non résolu (DB probablement verrouillée) — retry au prochain rerun"
            )
        repo = get_cached_repository_st(db_path, player_xuid)
        return repo.load_matches(include_firefight=include_firefight)
    except SharedDBUnavailableError:
        raise
    except Exception:
        return []


def _load_matches_duckdb_v4_polars(
    db_path: str,
    include_firefight: bool = True,
    columns: list[str] | None = None,
    max_matches: int | None = None,
) -> pl.DataFrame:
    """Charge les matchs depuis une DB DuckDB v4 en Polars via Arrow zero-copy.

    Chemin optimisé Sprint 19 : DuckDB → Arrow → Polars sans intermédiaire
    MatchRow. ~3× plus rapide que _load_matches_duckdb_v4 + reconstruction.
    Utilise le repository caché (v5.1 perf) pour éviter les reconnexions.

    Args:
        db_path: Chemin vers la DB DuckDB.
        include_firefight: Inclure les matchs PvE.
        columns: Liste de colonnes à projeter (None = toutes).
        max_matches: Limite SQL sur les N matchs les plus récents (P3).

    Returns:
        DataFrame Polars. Vide en cas d'erreur.
    """
    from src.ui._cache_core import SharedDBUnavailableError

    try:
        player_xuid = _resolve_player_xuid(db_path)
        if not player_xuid:
            raise SharedDBUnavailableError(
                "XUID non résolu (DB probablement verrouillée) — retry au prochain rerun"
            )
        repo = get_cached_repository_st(db_path, player_xuid)
        return repo.load_matches_as_polars(
            include_firefight=include_firefight,
            columns=columns,
            max_matches=max_matches,
        )
    except SharedDBUnavailableError:
        raise  # Ne pas cacher — retry automatique au prochain appel
    except Exception:
        logger.warning("load_matches_as_polars échoué, fallback MatchRow", exc_info=True)
        return pl.DataFrame()


# ─── Enrichissement des matchs ─────────────────────────────────────────────


def _enrich_matches_df(df: pl.DataFrame) -> pl.DataFrame:
    """Enrichit un DataFrame Polars de matchs avec timezone et colonnes calculées.

    Applique les transformations standard :
    - Conversion timezone UTC → Paris → naïf
    - Extraction colonne date
    - Calcul kills/deaths/assists par minute

    Args:
        df: DataFrame Polars brut avec au minimum start_time, kills, deaths, assists,
            time_played_seconds.

    Returns:
        DataFrame enrichi.
    """
    if df.is_empty():
        return df

    # Conversion timezone start_time
    if "start_time" in df.columns:
        _tz = _get_tz_name()  # résolution dynamique depuis app_settings
        start_time_dtype = df.schema.get("start_time")
        if start_time_dtype in (
            pl.Datetime,
            pl.Datetime("us"),
            pl.Datetime("ns"),
            pl.Datetime("ms"),
        ):
            try:
                df = df.with_columns(
                    pl.col("start_time")
                    .dt.replace_time_zone(PARIS_TZ_NAME)
                    .dt.convert_time_zone(_tz)
                    .dt.replace_time_zone(None)
                    .alias("start_time")
                )
            except Exception:
                logger.debug(
                    "_enrich_matches_df: conversion timezone échouée (tz=%s), retry",
                    _tz,
                    exc_info=True,
                )
                df = df.with_columns(
                    pl.col("start_time")
                    .dt.replace_time_zone(PARIS_TZ_NAME)
                    .dt.convert_time_zone(_tz)
                    .dt.replace_time_zone(None)
                    .alias("start_time")
                )
        elif start_time_dtype == pl.Utf8:
            df = df.with_columns(
                pl.col("start_time")
                .str.to_datetime(time_zone=PARIS_TZ_NAME)
                .dt.convert_time_zone(_tz)
                .dt.replace_time_zone(None)
                .alias("start_time")
            )

        # Extraire la date
        df = df.with_columns(pl.col("start_time").dt.date().alias("date"))

    # Stats par minute
    if "time_played_seconds" in df.columns:
        df = df.with_columns(
            (pl.col("time_played_seconds").cast(pl.Float64) / 60.0)
            .clip(lower_bound=0.0)
            .alias("minutes")
        )
        per_min_cols = []
        if "kills" in df.columns:
            per_min_cols.append(
                (pl.col("kills").cast(pl.Float64) / pl.col("minutes")).alias("kills_per_min")
            )
        if "deaths" in df.columns:
            per_min_cols.append(
                (pl.col("deaths").cast(pl.Float64) / pl.col("minutes")).alias("deaths_per_min")
            )
        if "assists" in df.columns:
            per_min_cols.append(
                (pl.col("assists").cast(pl.Float64) / pl.col("minutes")).alias("assists_per_min")
            )
        if per_min_cols:
            df = df.with_columns(per_min_cols)
        df = df.drop("minutes")

    return df


@st.cache_data(show_spinner=False)
def load_df_optimized(  # noqa: PLR0913
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None = None,
    include_firefight: bool = True,
    cache_buster: int = 0,
    max_matches: int | None = None,
) -> pl.DataFrame:
    """Charge les matchs avec fallback intelligent.

    Supporte:
    - DuckDB v4: data/players/{gamertag}/stats.duckdb
    - Legacy SQLite: MatchCache puis MatchStats

    Mécanisme d'invalidation du cache (Sprint 19 — tâche 19.4) :
    - db_key (mtime_ns, size) : détecte les modifications du fichier DB
      (sync externe, modification directe). Invalidation automatique.
    - cache_buster (int) : incrémenté dans session_state après un sync réussi.
      Force le rechargement même si db_key n'a pas encore changé (race condition).
    Les deux paramètres sont passés à @st.cache_data comme clés de hash,
    et ne sont pas lus dans le corps de la fonction.

    Args:
        db_path: Chemin vers la DB.
        xuid: XUID du joueur (ignoré pour DuckDB v4).
        db_key: Clé de cache (mtime, size) — None si fichier inexistant.
        include_firefight: Inclure les matchs PvE.
        cache_buster: Token pour forcer l'invalidation du cache après sync.
        max_matches: Limite SQL sur les N matchs les plus récents (P3).
                     None = pas de limite (comportement par défaut).

    Returns:
        DataFrame Polars enrichi avec toutes les colonnes calculées.
    """
    _ = db_key  # Utilisé pour invalidation du cache Streamlit
    _ = cache_buster  # Utilisé pour forcer le rechargement après sync
    # COLUMNS_SCHEMA_VERSION=2 : playlist_name_fr, map_name_fr, pair_name_fr ajoutées à COLUMNS_COMMON
    logger.debug(
        "load_df_optimized: xuid=%s..., cache_buster=%s",
        str(xuid or "")[:8],
        cache_buster,
    )

    # Détecter le type de DB
    if _is_duckdb_v4_path(db_path):
        # Sprint 19 : chemin optimisé DuckDB → Arrow → Polars (zero-copy)
        # P5 : projection COLUMNS_COMMON pour réduire l'empreinte mémoire
        #      (exclut game_variant_id, team_id, rank — non utilisés en hot-path)
        # P3 : LIMIT SQL via max_matches pour limiter le chargement initial
        df = _load_matches_duckdb_v4_polars(
            db_path,
            include_firefight=include_firefight,
            columns=COLUMNS_COMMON,
            max_matches=max_matches,
        )
        if not df.is_empty():
            # Enrichissement standard (timezone, colonnes calculées)
            df = _enrich_matches_df(df)
            return df

        # Fallback legacy : MatchRow → reconstruction DataFrame
        matches = _load_matches_duckdb_v4(db_path, include_firefight=include_firefight)

    if not matches:
        return pl.DataFrame()

    # Construire le DataFrame Polars depuis MatchRow (fallback legacy)
    df = pl.DataFrame(
        {
            "match_id": [m.match_id for m in matches],
            "start_time": [m.start_time for m in matches],
            "map_id": [m.map_id for m in matches],
            "map_name": [m.map_name for m in matches],
            "playlist_id": [m.playlist_id for m in matches],
            "playlist_name": [m.playlist_name for m in matches],
            "pair_id": [m.map_mode_pair_id for m in matches],
            "pair_name": [m.map_mode_pair_name for m in matches],
            "game_variant_id": [m.game_variant_id for m in matches],
            "game_variant_name": [m.game_variant_name for m in matches],
            "outcome": [m.outcome for m in matches],
            "kda": [m.kda for m in matches],
            "my_team_score": [m.my_team_score for m in matches],
            "enemy_team_score": [m.enemy_team_score for m in matches],
            "max_killing_spree": [m.max_killing_spree for m in matches],
            "headshot_kills": [m.headshot_kills for m in matches],
            "average_life_seconds": [m.average_life_seconds for m in matches],
            "time_played_seconds": [m.time_played_seconds for m in matches],
            "kills": [m.kills for m in matches],
            "deaths": [m.deaths for m in matches],
            "assists": [m.assists for m in matches],
            "accuracy": [m.accuracy for m in matches],
            "ratio": [m.ratio for m in matches],
            "team_mmr": [m.team_mmr for m in matches],
            "enemy_mmr": [m.enemy_mmr for m in matches],
        }
    )

    return _enrich_matches_df(df)


# ─── Fonctions sociales réexportées depuis cache_social.py ─────────────────
from src.ui.cache_social import (  # noqa: E402
    cached_load_top_teammates_optimized,
)

__all__ = [
    "cached_load_top_teammates_optimized",
]
