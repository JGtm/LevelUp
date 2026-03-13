"""
Bridge entre le nouveau système de données et l'UI Streamlit.
(Bridge between new data system and Streamlit UI)

# v5.7 : frontière Pandas légitime — migration évaluée pour v5.8+
# Ce module utilise pd.to_datetime().dt.tz_convert() pour la timezone,
# pas d'équivalent simple en Polars pur.

HOW IT WORKS:
Ce module fournit des fonctions qui :
1. Créent les repositories/engines avec le bon mode
2. Convertissent les résultats en DataFrames Pandas
3. Gèrent le cache et l'état Streamlit

Les fonctions retournent des types compatibles avec l'UI existante
(pd.DataFrame, list, dict) plutôt que les nouveaux types (MatchRow, etc.)
"""

from __future__ import annotations

import os
from pathlib import Path
from typing import Any

import polars as pl

# Import direct depuis factory pour éviter l'import circulaire avec src.data
from src.data.domain.models.stats import MatchRow
from src.data.repositories.factory import (
    RepositoryMode,
    get_repository,
    get_repository_from_profile,
    load_db_profiles,
)
from src.data.repositories.protocol import DataRepository

# Timezone — fallback legacy, préférer get_tz_name() pour la TZ utilisateur
PARIS_TZ_NAME = "Europe/Paris"


def _get_display_tz_name() -> str:
    """Retourne la timezone utilisateur, fallback 'Europe/Paris'."""
    try:
        from src.ui.tz import get_tz_name

        return get_tz_name()
    except Exception:
        return PARIS_TZ_NAME


def get_repository_mode_from_settings() -> RepositoryMode:
    """
    Récupère le mode de repository depuis les settings ou l'environnement.
    (Get repository mode from settings or environment)

    Priorité:
    1. Variable d'environnement LEVELUP_REPOSITORY_MODE (duckdb)
    2. Paramètre dans st.session_state.app_settings.repository_mode (duckdb)
    3. Auto-détection depuis db_profiles.json
    4. Défaut: DUCKDB
    """
    # 1. Variable d'environnement
    env_mode = os.environ.get("LEVELUP_REPOSITORY_MODE", "").lower().strip()
    if env_mode == RepositoryMode.DUCKDB.value:
        return RepositoryMode.DUCKDB

    # 2. Settings Streamlit (si disponible)
    try:
        import streamlit as st

        settings = st.session_state.get("app_settings")
        if settings and hasattr(settings, "repository_mode"):
            mode_str = str(settings.repository_mode).lower().strip()
            if mode_str == RepositoryMode.DUCKDB.value:
                return RepositoryMode.DUCKDB
    except Exception:
        pass

    # 3. Auto-détection depuis db_profiles.json
    try:
        profiles = load_db_profiles()
        if profiles.get("version", "1.0") >= "2.0":
            return RepositoryMode.DUCKDB
    except Exception:
        pass

    # 4. Défaut
    return RepositoryMode.DUCKDB


def get_warehouse_path(db_path: str) -> Path:
    """
    Détermine le chemin du warehouse à partir du chemin de la DB.
    (Determine warehouse path from DB path)

    Convention: warehouse est dans le même dossier que la DB.
    """
    return Path(db_path).parent / "warehouse"


def get_repository_for_ui(
    db_path: str,
    xuid: str,
    *,
    mode: RepositoryMode | None = None,
    gamertag: str | None = None,
) -> DataRepository:
    """
    Crée un repository configuré pour l'UI.
    (Create a repository configured for UI)

    Args:
        db_path: Chemin vers la base de données
        xuid: XUID du joueur
        mode: Mode explicite (sinon récupéré depuis settings)
        gamertag: Gamertag du joueur (optionnel)

    Returns:
        Instance de DataRepository
    """
    if mode is None:
        mode = get_repository_mode_from_settings()

    return get_repository(
        db_path,
        xuid,
        mode=mode,
        gamertag=gamertag,
    )


def get_repository_for_player(
    gamertag: str,
    *,
    mode: RepositoryMode | None = None,
) -> DataRepository:
    """
    Crée un repository à partir du gamertag via db_profiles.json.
    (Create repository from gamertag using db_profiles.json)

    C'est la méthode recommandée pour l'UI Streamlit car elle
    gère automatiquement les chemins et le mode de repository.

    Args:
        gamertag: Gamertag du joueur
        mode: Mode explicite (sinon auto-détecté)

    Returns:
        Instance de DataRepository

    Raises:
        ValueError: Si le gamertag n'est pas trouvé dans db_profiles.json
    """
    return get_repository_from_profile(gamertag, mode=mode)


def matches_to_dataframe(matches: list[MatchRow]) -> Any:
    """Convertit une liste de MatchRow en DataFrame Pandas.

    Construit en interne un DataFrame Polars puis convertit en Pandas
    à la frontière UI. ``import pandas`` est différé dans cette fonction.

    Format identique à load_df_optimized() pour compatibilité.
    """
    import pandas as pd  # frontière UI uniquement

    if not matches:
        return pd.DataFrame()

    # Construire d'abord en Polars (moteur natif)
    df_pl = pl.DataFrame(
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
            "personal_score": [m.personal_score for m in matches],
        }
    )

    # Stats par minute (en Polars)
    minutes = pl.col("time_played_seconds").cast(pl.Float64) / 60.0
    df_pl = df_pl.with_columns(
        (pl.col("kills").cast(pl.Float64) / minutes).alias("kills_per_min"),
        (pl.col("deaths").cast(pl.Float64) / minutes).alias("deaths_per_min"),
        (pl.col("assists").cast(pl.Float64) / minutes).alias("assists_per_min"),
    )

    # Convertir en Pandas à la frontière
    df = df_pl.to_pandas()

    # Conversions timezone (API Pandas requise par l'UI)
    df["start_time"] = (
        pd.to_datetime(df["start_time"], utc=True)
        .dt.tz_convert(_get_display_tz_name())
        .dt.tz_localize(None)
    )
    df["date"] = df["start_time"].dt.date

    return df


def load_matches_df(  # noqa: PLR0913
    db_path: str,
    xuid: str,
    *,
    include_firefight: bool = True,
    playlist_filter: str | None = None,
    map_filter: str | None = None,
    mode: RepositoryMode | None = None,
) -> Any:
    """
    Charge les matchs en DataFrame via le DataRepository.
    (Load matches as DataFrame via DataRepository)

    Fonction principale d'intégration, remplaçant load_df_optimized()
    pour les cas où on veut utiliser le nouveau système.

    Args:
        db_path: Chemin vers la base de données
        xuid: XUID du joueur
        include_firefight: Inclure les matchs PvE
        playlist_filter: Filtre sur playlist_id
        map_filter: Filtre sur map_id
        mode: Mode de repository (défaut: depuis settings)

    Returns:
        DataFrame Pandas compatible avec l'UI existante
    """
    repo = get_repository_for_ui(db_path, xuid, mode=mode)

    try:
        matches = repo.load_matches(
            playlist_filter=playlist_filter,
            map_filter=map_filter,
            include_firefight=include_firefight,
        )
        return matches_to_dataframe(matches)
    finally:
        # Fermer si c'est un HybridRepository ou ShadowRepository
        if hasattr(repo, "close"):
            repo.close()


def get_analytics_for_ui(
    db_path: str,
    xuid: str,
):
    """
    Crée une instance d'AnalyticsQueries pour l'UI.
    (Create AnalyticsQueries instance for UI)

    Returns:
        Tuple (engine, analytics) à fermer après utilisation
    """
    from src.data.query import AnalyticsQueries, QueryEngine

    warehouse_path = get_warehouse_path(db_path)
    engine = QueryEngine(warehouse_path)
    analytics = AnalyticsQueries(engine, xuid)

    return engine, analytics


def get_trends_for_ui(
    db_path: str,
    xuid: str,
):
    """
    Crée une instance de TrendAnalyzer pour l'UI.
    (Create TrendAnalyzer instance for UI)

    Returns:
        Tuple (engine, trends) à fermer après utilisation
    """
    from src.data.query import QueryEngine, TrendAnalyzer

    warehouse_path = get_warehouse_path(db_path)
    engine = QueryEngine(warehouse_path)
    trends = TrendAnalyzer(engine, xuid)

    return engine, trends


def matches_to_polars(matches: list[MatchRow]) -> pl.DataFrame:
    """Convertit une liste de MatchRow en DataFrame Polars.

    Format normalisé v4.5 — privilégier cette fonction pour les services.

    Args:
        matches: Liste de MatchRow.

    Returns:
        pl.DataFrame avec colonnes typées.
    """
    if not matches:
        return pl.DataFrame()

    data = {
        "match_id": [m.match_id for m in matches],
        "start_time": [m.start_time for m in matches],
        "map_id": [m.map_id for m in matches],
        "map_name": [m.map_name for m in matches],
        "playlist_id": [m.playlist_id for m in matches],
        "playlist_name": [m.playlist_name for m in matches],
        "pair_id": [m.map_mode_pair_id for m in matches],
        "pair_name": [m.map_mode_pair_name for m in matches],
        "outcome": [m.outcome for m in matches],
        "kills": [m.kills for m in matches],
        "deaths": [m.deaths for m in matches],
        "assists": [m.assists for m in matches],
        "accuracy": [m.accuracy for m in matches],
        "ratio": [m.ratio for m in matches],
        "kda": [m.kda for m in matches],
        "time_played_seconds": [m.time_played_seconds for m in matches],
        "average_life_seconds": [m.average_life_seconds for m in matches],
        "personal_score": [m.personal_score for m in matches],
        "team_mmr": [m.team_mmr for m in matches],
        "enemy_mmr": [m.enemy_mmr for m in matches],
    }

    df = pl.DataFrame(data)

    # Ajouter stats par minute
    minutes = pl.col("time_played_seconds").cast(pl.Float64) / 60.0
    df = df.with_columns(
        (pl.col("kills").cast(pl.Float64) / minutes).alias("kills_per_min"),
        (pl.col("deaths").cast(pl.Float64) / minutes).alias("deaths_per_min"),
        (pl.col("assists").cast(pl.Float64) / minutes).alias("assists_per_min"),
    )

    return df


def load_matches_polars(  # noqa: PLR0913
    db_path: str,
    xuid: str,
    *,
    include_firefight: bool = True,
    playlist_filter: str | None = None,
    map_filter: str | None = None,
    mode: RepositoryMode | None = None,
) -> pl.DataFrame:
    """Charge les matchs en pl.DataFrame via le DataRepository.

    Version Polars-native de load_matches_df(). Retour normalisé v4.5.

    Args:
        db_path: Chemin vers la base de données.
        xuid: XUID du joueur.
        include_firefight: Inclure les matchs PvE.
        playlist_filter: Filtre sur playlist_id.
        map_filter: Filtre sur map_id.
        mode: Mode de repository (défaut: depuis settings).

    Returns:
        pl.DataFrame compatible avec les services et la couche analysis.
    """
    repo = get_repository_for_ui(db_path, xuid, mode=mode)

    try:
        matches = repo.load_matches(
            playlist_filter=playlist_filter,
            map_filter=map_filter,
            include_firefight=include_firefight,
        )
        return matches_to_polars(matches)
    finally:
        if hasattr(repo, "close"):
            repo.close()


def check_hybrid_available(db_path: str, xuid: str) -> bool:
    """
    @deprecated Vérifie si les données hybrides (Parquet) sont disponibles.

    Depuis v4, cette fonction retourne toujours False car Parquet n'est plus utilisé.
    """
    return False


def get_migration_status(db_path: str, xuid: str) -> dict[str, Any]:
    """
    @deprecated Retourne l'état de la migration pour un joueur.

    Depuis v4, les migrations legacy → Parquet ne sont plus supportées.
    Utilisez scripts/migrate_to_duckdb.py pour migrer vers DuckDB.
    """
    return {
        "legacy_count": 0,
        "hybrid_count": 0,
        "progress_percent": 100,
        "is_complete": True,
        "message": "Migration v4 (DuckDB) - Pas de migration Parquet nécessaire",
    }
