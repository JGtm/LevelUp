"""Requêtes DB pour l'enrichissement performance_score des coéquipiers."""

from __future__ import annotations

import logging
from pathlib import Path

logger = logging.getLogger(__name__)


def load_performance_scores_from_player_db(db_path: str, match_ids: list[str]) -> dict[str, float]:
    """Charge performance_score depuis player_match_enrichment pour un ensemble de matchs.

    Args:
        db_path: Chemin vers la DB individuelle du joueur (stats.duckdb).
        match_ids: Liste des match_id à charger.

    Returns:
        Mapping match_id → performance_score (float). Vide si DB absente ou erreur.
    """
    from src.utils.db import duckdb_read_only

    if not match_ids or not Path(db_path).exists():
        return {}
    try:
        ph = ", ".join(["?" for _ in match_ids])
        with duckdb_read_only(db_path) as conn:
            rows = conn.execute(
                f"SELECT match_id, performance_score FROM player_match_enrichment"
                f" WHERE match_id IN ({ph}) AND performance_score IS NOT NULL",
                match_ids,
            ).fetchall()
        return {str(r[0]): float(r[1]) for r in rows}
    except Exception as e:
        logger.debug("Erreur performance_score depuis %s: %s", db_path, e)
        return {}
