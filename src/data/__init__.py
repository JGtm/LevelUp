"""
Module data : Architecture DuckDB native.
(Data module: Native DuckDB architecture)

Architecture v4 : Toutes les données sont stockées dans DuckDB.
Les anciens modes (Legacy, Hybrid, Shadow) ont été supprimés.

HOW IT WORKS:
1. DataRepository : Interface abstraite définissant le contrat d'accès aux données
2. DuckDBRepository : Implémentation v4 utilisant DuckDB natif (stats.duckdb)

Usage recommandé:
    from src.data import get_repository_from_profile

    # Via gamertag
    repo = get_repository_from_profile("SpartanC")
    matches = repo.load_matches()

Usage explicite:
    from src.data import get_repository

    # Mode DuckDB natif (v4)
    repo = get_repository(
        "data/players/SpartanC/stats.duckdb",
        xuid,
    )
"""

from src.data.repositories.factory import (
    RepositoryMode,
    get_repository,
    get_repository_from_profile,
    load_db_profiles,
)
from src.data.repositories.protocol import DataRepository

__all__ = [
    "get_repository",
    "get_repository_from_profile",
    "load_db_profiles",
    "RepositoryMode",
    "DataRepository",
]
