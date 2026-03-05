"""Chargement de données pour la page Match View — section joueurs.

Ce module isole les accès DuckDB/repository utilisés par
``match_view_players.py`` et ``match_view_scoreboard.py``.
"""

from __future__ import annotations

import logging
from typing import Any

from src.utils.db import is_duckdb_v4_path as _is_duckdb_v4_path

logger = logging.getLogger(__name__)


def has_table_duckdb(db_path: str, table_name: str) -> bool:
    """Vérifie si une table existe dans une DB DuckDB (locale ou shared).

    Utilise le repository pour vérifier la table locale puis shared (v5.1).

    Args:
        db_path: Chemin vers la DB joueur.
        table_name: Nom de la table à vérifier.

    Returns:
        True si la table existe (locale ou shared).
    """
    if not _is_duckdb_v4_path(db_path):
        return False
    try:
        from src.data.repositories.duckdb_repo import DuckDBRepository

        repo = DuckDBRepository(db_path, xuid="", read_only=True)
        return repo.has_table(table_name) or repo._has_shared_table(table_name)
    except Exception:
        logger.debug(
            "has_table_duckdb: échec vérification table=%s db=%s",
            table_name,
            db_path,
        )
        return False


def load_match_players_stats(db_path: str, match_id: str) -> list[dict[str, Any]]:
    """Charge les stats des joueurs d'un match.

    Args:
        db_path: Chemin vers la DB joueur.
        match_id: Identifiant du match.

    Returns:
        Liste de dicts joueur ou liste vide en cas d'erreur.
    """
    if not _is_duckdb_v4_path(db_path):
        return []
    try:
        from src.data.repositories.duckdb_repo import DuckDBRepository

        repo = DuckDBRepository(db_path, xuid="", read_only=True)
        return repo.load_match_players_stats(match_id)
    except Exception:
        logger.warning("Chargement stats joueurs échoué match=%s", match_id)
        return []


def load_match_scoreboard(db_path: str, match_id: str) -> list[dict[str, Any]]:
    """Charge le tableau de bord complet d'un match.

    Inclut toutes les stats + frags parfaits.

    Args:
        db_path: Chemin vers la DB joueur.
        match_id: Identifiant du match.

    Returns:
        Liste de dicts joueur ou liste vide en cas d'erreur.
    """
    if not _is_duckdb_v4_path(db_path):
        return []
    try:
        from src.data.repositories.duckdb_repo import DuckDBRepository

        repo = DuckDBRepository(db_path, xuid="", read_only=True)
        return repo.load_match_scoreboard(match_id)
    except Exception:
        logger.warning("Chargement scoreboard échoué match=%s", match_id)
        return []
