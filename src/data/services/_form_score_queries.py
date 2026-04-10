"""Requêtes DB — historique complet performance_score pour le score de forme."""

from __future__ import annotations

import logging
from pathlib import Path

import polars as pl

logger = logging.getLogger(__name__)


def load_full_performance_history(player_db_path: str) -> pl.DataFrame:
    """Charge l'historique complet de performance_score d'un joueur.

    Joint player_match_enrichment (stats.duckdb) avec match_registry (shared)
    pour obtenir start_time sur tous les matchs enrichis du joueur.

    Args:
        player_db_path: Chemin vers stats.duckdb du joueur.

    Returns:
        DataFrame (match_id: Utf8, start_time: Datetime, performance_score: Float64)
        trié par start_time. Vide si DB absente ou erreur.
    """
    from src.utils.db import duckdb_read_only
    from src.utils.paths import get_shared_matches_path_from_player

    player_path = Path(player_db_path)
    if not player_path.exists():
        return pl.DataFrame()

    shared_path = get_shared_matches_path_from_player(player_db_path)
    if not shared_path or not shared_path.exists():
        return pl.DataFrame()

    shared_path_sql = str(shared_path).replace("'", "''")
    try:
        with duckdb_read_only(str(player_path)) as conn:
            conn.execute(f"ATTACH '{shared_path_sql}' AS shared (READ_ONLY)")
            rows = conn.execute(
                "SELECT pme.match_id, mr.start_time, pme.performance_score"
                " FROM player_match_enrichment pme"
                " JOIN shared.match_registry mr ON pme.match_id = mr.match_id"
                " WHERE pme.performance_score IS NOT NULL"
                " ORDER BY mr.start_time"
            ).fetchall()
        if not rows:
            return pl.DataFrame()
        return pl.DataFrame(
            {
                "match_id": [str(r[0]) for r in rows],
                "start_time": [r[1] for r in rows],
                "performance_score": [float(r[2]) for r in rows],
            }
        )
    except Exception as exc:
        logger.debug("load_full_performance_history(%s): %s", player_db_path, exc)
        return pl.DataFrame()
