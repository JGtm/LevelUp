"""Cache Streamlit — Chargement et pagination de matchs.

Fonctions cached_load_recent_matches, cached_load_matches_paginated et
cached_get_match_count_duckdb extraites de cache_filters.py.
"""

from __future__ import annotations

import logging

import polars as pl
import streamlit as st

from src.ui.cache_loaders import PARIS_TZ_NAME  # noqa: F401 (compat)

logger = logging.getLogger(__name__)


def _get_user_tz_name() -> str:
    """TZ utilisateur dynamique, fallback Paris."""
    try:
        from src.ui.tz import get_tz_name

        return get_tz_name()
    except Exception:
        return PARIS_TZ_NAME


# 8bis.A4 : TTL supprimé, l'invalidation se fait via db_key (mtime + size)
@st.cache_data(show_spinner=False)
def cached_load_recent_matches(
    player_db_path: str,
    xuid: str,
    limit: int = 50,
    db_key: tuple[int, int] | None = None,
    tz_name: str = "",
) -> pl.DataFrame:
    """Charge les N matchs les plus récents via DuckDB (lazy loading).

    Optimisé pour le chargement initial rapide de l'UI.
    Utilise le DuckDBRepository si disponible, sinon fallback.

    Args:
        player_db_path: Chemin vers stats.duckdb du joueur.
        xuid: XUID du joueur.
        limit: Nombre de matchs à charger.
        db_key: Clé de cache pour invalidation.

    Returns:
        DataFrame Polars des matchs récents.
    """
    _ = db_key  # Pour invalidation du cache Streamlit

    try:
        from pathlib import Path

        from src.data.repositories.duckdb_repo import DuckDBRepository

        db_path = Path(player_db_path)
        if not db_path.exists():
            return pl.DataFrame()

        repo = DuckDBRepository(db_path, xuid)
        try:
            matches = repo.load_recent_matches(limit=limit)
        finally:
            repo.close()

        if not matches:
            return pl.DataFrame()

        df = _matches_to_dataframe(matches)
        return _convert_timezone(df, tz_name or _get_user_tz_name())

    except ImportError:
        logger.warning("Import DuckDBRepository indisponible pour recent_matches")
        return pl.DataFrame()
    except Exception:
        logger.warning("Erreur chargement recent_matches", exc_info=True)
        return pl.DataFrame()


# 8bis.A4 : TTL supprimé, l'invalidation se fait via db_key (mtime + size)
@st.cache_data(show_spinner=False)
def cached_load_matches_paginated(  # noqa: PLR0913
    player_db_path: str,
    xuid: str,
    page: int = 1,
    page_size: int = 50,
    db_key: tuple[int, int] | None = None,
    tz_name: str = "",
) -> tuple[pl.DataFrame, int]:
    """Charge les matchs avec pagination via DuckDB.

    Args:
        player_db_path: Chemin vers stats.duckdb du joueur.
        xuid: XUID du joueur.
        page: Numéro de page (1-indexed).
        page_size: Nombre de matchs par page.
        db_key: Clé de cache pour invalidation.

    Returns:
        Tuple (DataFrame Polars des matchs, nombre total de pages).
    """
    _ = db_key

    try:
        from pathlib import Path

        from src.data.repositories.duckdb_repo import DuckDBRepository

        db_path = Path(player_db_path)
        if not db_path.exists():
            return pl.DataFrame(), 1

        repo = DuckDBRepository(db_path, xuid)
        try:
            matches, total_pages = repo.load_matches_paginated(
                page=page,
                page_size=page_size,
                order_desc=True,
            )
        finally:
            repo.close()

        if not matches:
            return pl.DataFrame(), total_pages

        df = _matches_to_dataframe(matches)
        return _convert_timezone(df, tz_name or _get_user_tz_name()), total_pages

    except ImportError:
        logger.warning("Import DuckDBRepository indisponible pour matches_paginated")
        return pl.DataFrame(), 1
    except Exception:
        logger.warning(
            "Erreur chargement matches_paginated (db=%s page=%d)",
            player_db_path,
            page,
            exc_info=True,
        )
        return pl.DataFrame(), 1


@st.cache_data(show_spinner=False, ttl=600)
def cached_get_match_count_duckdb(
    player_db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None = None,
) -> int:
    """Récupère le nombre total de matchs via DuckDB.

    Args:
        player_db_path: Chemin vers stats.duckdb du joueur.
        xuid: XUID du joueur.
        db_key: Clé de cache pour invalidation.

    Returns:
        Nombre total de matchs.
    """
    _ = db_key

    try:
        from pathlib import Path

        from src.data.repositories.duckdb_repo import DuckDBRepository

        db_path = Path(player_db_path)
        if not db_path.exists():
            return 0

        repo = DuckDBRepository(db_path, xuid)
        try:
            count = repo.get_match_count()
            return count
        finally:
            repo.close()

    except Exception:
        logger.debug("Erreur récupération match_count", exc_info=True)
        return 0


# ---------------------------------------------------------------------------
# Helpers internes (DRY — logique partagée par recent + paginated)
# ---------------------------------------------------------------------------

_MATCH_COLUMNS = [
    "match_id",
    "start_time",
    "map_id",
    "map_name",
    "playlist_id",
    "playlist_name",
    "pair_id",
    "pair_name",
    "game_variant_id",
    "game_variant_name",
    "outcome",
    "kda",
    "my_team_score",
    "enemy_team_score",
    "max_killing_spree",
    "headshot_kills",
    "average_life_seconds",
    "time_played_seconds",
    "kills",
    "deaths",
    "assists",
    "accuracy",
    "ratio",
    "team_mmr",
    "enemy_mmr",
]


def _matches_to_dataframe(matches: list) -> pl.DataFrame:
    """Convertit une liste de MatchResult en DataFrame Polars."""
    return pl.DataFrame(
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


def _convert_timezone(df: pl.DataFrame, tz_name: str = "") -> pl.DataFrame:
    """Convertit start_time UTC → timezone utilisateur (naïve) et ajoute colonne date."""
    if not tz_name:
        tz_name = _get_user_tz_name()
    try:
        df = df.with_columns(
            pl.col("start_time")
            .cast(pl.Datetime("us", "UTC"))
            .dt.convert_time_zone(tz_name)
            .dt.replace_time_zone(None)
        )
    except Exception:
        logger.debug("Erreur conversion timezone start_time (tz=%s)", tz_name, exc_info=True)
        import contextlib

        with contextlib.suppress(Exception):
            df = df.with_columns(
                pl.col("start_time")
                .dt.replace_time_zone("UTC")
                .dt.convert_time_zone(tz_name)
                .dt.replace_time_zone(None)
            )
    return df.with_columns(pl.col("start_time").cast(pl.Date).alias("date"))
