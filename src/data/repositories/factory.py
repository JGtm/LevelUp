"""
Factory pour la création des repositories.
(Factory for repository creation)

HOW IT WORKS:
Ce module fournit une interface simple pour créer le bon repository.
Depuis la v4, seul DuckDBRepository est utilisé.

Architecture v4 (actuelle):
    Utiliser get_repository_from_profile() pour auto-détection.

Usage recommandé:
    from src.data.repositories.factory import get_repository_from_profile

    # Auto-détection depuis db_profiles.json (recommandé)
    repo = get_repository_from_profile("SpartanC")

Usage explicite:
    from src.data import get_repository

    # DuckDB natif (architecture v4)
    repo = get_repository(
        "data/players/SpartanC/stats.duckdb",
        xuid,
    )
"""

from __future__ import annotations

import json
from enum import Enum
from pathlib import Path
from typing import TYPE_CHECKING, Any

import polars as pl

from src.data.repositories.duckdb_repo import DuckDBRepository

if TYPE_CHECKING:
    from src.data.domain.models.stats import MatchRow
    from src.ports.repository import DataRepository


class RepositoryMode(Enum):
    """
    Modes de repository disponibles.
    (Available repository modes)

    Depuis v4, seul DUCKDB est supporté.
    """

    DUCKDB = "duckdb"


def get_repository(  # noqa: PLR0913
    db_path: str,
    xuid: str,
    *,
    mode: RepositoryMode | str = RepositoryMode.DUCKDB,
    warehouse_path: str | Path | None = None,  # @deprecated - ignoré
    shared_db_path: str | Path | None = None,
    gamertag: str | None = None,
) -> DataRepository:
    """
    Crée et retourne le repository approprié.
    (Create and return the appropriate repository)

    Depuis v4, seul le mode DUCKDB est supporté.

    Args:
        db_path: Chemin vers la DB DuckDB (stats.duckdb)
        xuid: XUID du joueur principal
        mode: Mode de repository (seul DUCKDB est supporté)
        warehouse_path: @deprecated - Ignoré depuis v4
        shared_db_path: Chemin vers shared_matches_v2.duckdb (auto-détecté si None)
        gamertag: Gamertag du joueur (optionnel)

    Returns:
        Instance de DataRepository (DuckDBRepository)

    Raises:
        ValueError: Si un mode différent de DUCKDB est demandé

    Exemple:
        # Mode DuckDB natif (v4)
        repo = get_repository(
            "data/players/SpartanC/stats.duckdb",
            "1234567890",
        )
    """
    # Normalise et valide le mode
    if isinstance(mode, str):
        normalized = mode.lower().strip()
        if normalized != RepositoryMode.DUCKDB.value:
            raise ValueError(
                f"Mode '{mode}' non supporté. Utilisez '{RepositoryMode.DUCKDB.value}'."
            )
        mode = RepositoryMode.DUCKDB

    if mode != RepositoryMode.DUCKDB:
        raise ValueError(
            f"Mode '{mode.value}' non supporté. Utilisez '{RepositoryMode.DUCKDB.value}'."
        )

    # Mode DuckDB natif - le db_path pointe vers stats.duckdb
    return DuckDBRepository(
        player_db_path=db_path,
        xuid=xuid,
        shared_db_path=shared_db_path,
        gamertag=gamertag,
    )


def load_db_profiles(profiles_path: str | Path | None = None) -> dict[str, Any]:
    """
    Charge la configuration des profils depuis db_profiles.json.
    (Load profile configuration from db_profiles.json)

    Returns:
        dict avec version, warehouse_path, profiles, etc.
    """
    if profiles_path is None:
        # Cherche dans le répertoire racine du projet
        profiles_path = Path(__file__).parent.parent.parent.parent / "db_profiles.json"

    profiles_path = Path(profiles_path)

    if not profiles_path.exists():
        return {"version": "1.0", "profiles": {}}

    with open(profiles_path, encoding="utf-8") as f:
        return json.load(f)


def get_repository_from_profile(
    gamertag: str,
    *,
    mode: RepositoryMode | str | None = None,  # @deprecated - ignoré, toujours DUCKDB
    profiles_path: str | Path | None = None,
    shared_db_path: str | Path | None = None,
) -> DataRepository:
    """
    Crée un repository à partir du profil d'un joueur.
    (Create repository from player profile)

    Lit db_profiles.json et crée un DuckDBRepository.

    Args:
        gamertag: Gamertag du joueur
        mode: @deprecated - Ignoré depuis v4, toujours DUCKDB
        profiles_path: Chemin vers db_profiles.json
        shared_db_path: Chemin vers shared_matches_v2.duckdb (auto-détecté si None)

    Returns:
        Instance de DuckDBRepository configurée

    Exemple:
        repo = get_repository_from_profile("SpartanC")
    """
    profiles = load_db_profiles(profiles_path)

    if gamertag not in profiles.get("profiles", {}):
        raise ValueError(f"Profil non trouvé pour: {gamertag}")

    profile = profiles["profiles"][gamertag]
    xuid = profile.get("xuid", "")

    return DuckDBRepository(
        player_db_path=profile["db_path"],
        xuid=xuid,
        shared_db_path=shared_db_path,
        gamertag=gamertag,
    )


def matches_to_polars(matches: list[MatchRow]) -> pl.DataFrame:
    """Convertit une liste de MatchRow en DataFrame Polars.

    Format normalisé — utilisé pour les services et tests.

    Args:
        matches: Liste de MatchRow.

    Returns:
        pl.DataFrame avec colonnes typées (inclut stats par minute).
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

    minutes = pl.col("time_played_seconds").cast(pl.Float64) / 60.0
    df = df.with_columns(
        (pl.col("kills").cast(pl.Float64) / minutes).alias("kills_per_min"),
        (pl.col("deaths").cast(pl.Float64) / minutes).alias("deaths_per_min"),
        (pl.col("assists").cast(pl.Float64) / minutes).alias("assists_per_min"),
    )

    return df
