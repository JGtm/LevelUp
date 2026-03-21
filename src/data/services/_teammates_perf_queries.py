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
    return {k: v for k, v, *_ in _load_perf_enrichment(db_path, match_ids)}


def load_perf_enrichment_with_session(db_path: str, match_ids: list[str]) -> dict[str, dict]:
    """Charge performance_score + session_id + session_label depuis player_match_enrichment.

    Returns:
        Mapping match_id → {"perf": float, "session_id": str|None, "session_label": str|None}.
    """
    return {
        str(mid): {"perf": float(perf), "session_id": sid, "session_label": slabel}
        for mid, perf, sid, slabel in _load_perf_enrichment(db_path, match_ids)
    }


def load_team_mmr_by_match(
    shared_db_path: str, gamertag: str, match_ids: list[str]
) -> dict[str, float]:
    """Charge team_mmr depuis shared.match_participants pour un gamertag et des match_ids.

    Returns:
        Mapping match_id → team_mmr (float).
    """
    from src.utils.db import duckdb_read_only

    if not match_ids or not Path(shared_db_path).exists():
        return {}
    try:
        ph = ", ".join(["?"] * len(match_ids))
        with duckdb_read_only(shared_db_path) as conn:
            rows = conn.execute(
                f"SELECT mp.match_id, mp.team_mmr"
                f" FROM match_participants mp"
                f" JOIN xuid_aliases xa ON mp.xuid = xa.xuid"
                f" WHERE xa.gamertag = ?"
                f" AND mp.match_id IN ({ph})"
                f" AND mp.team_mmr IS NOT NULL",
                [gamertag, *match_ids],
            ).fetchall()
        return {str(mid): float(mmr) for mid, mmr in rows}
    except Exception as e:
        logger.debug("Erreur team_mmr depuis %s pour %s: %s", shared_db_path, gamertag, e)
        return {}


def _load_perf_enrichment(db_path: str, match_ids: list[str]) -> list[tuple]:
    """Charge performance_score, session_id, session_label depuis player_match_enrichment.

    session_id et session_label sont NULL si les colonnes n'existent pas encore
    (compatibilité avec les DB créées avant la migration sessions).
    """
    from src.utils.db import duckdb_read_only

    if not match_ids or not Path(db_path).exists():
        return []
    try:
        ph = ", ".join(["?" for _ in match_ids])
        with duckdb_read_only(db_path) as conn:
            existing = {
                r[0]
                for r in conn.execute(
                    "SELECT column_name FROM information_schema.columns"
                    " WHERE table_name = 'player_match_enrichment'"
                ).fetchall()
            }
            sid = "session_id" if "session_id" in existing else "NULL"
            slabel = "session_label" if "session_label" in existing else "NULL"
            rows = conn.execute(
                f"SELECT match_id, performance_score, {sid}, {slabel}"
                f" FROM player_match_enrichment"
                f" WHERE match_id IN ({ph}) AND performance_score IS NOT NULL",
                match_ids,
            ).fetchall()
        return rows
    except Exception as e:
        logger.debug("Erreur perf_enrichment depuis %s: %s", db_path, e)
        return []
