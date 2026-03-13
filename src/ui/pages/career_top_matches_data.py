"""Page Carrière — Chargement Top 10 meilleurs / pires matchs."""

from __future__ import annotations

import logging

from src.analysis._medal_verdicts import DominanceFlag
from src.data.domain.refdata import Outcome
from src.utils.db import duckdb_read_only
from src.utils.paths import get_shared_matches_path

logger = logging.getLogger(__name__)

# Durée minimale d'un match "valide" (en secondes).
# Les matchs plus courts sont exclus (rage quit, crash serveur).
MIN_MATCH_DURATION_SECONDS: int = 180


_TOP_MATCHES_SQL = """
WITH enriched AS (
    SELECT
        mv.match_id,
        mv.start_time,
        mv.map_name,
        mv.playlist_name,
        mv.game_variant_name,
        mv.outcome,
        mv.kills,
        mv.deaths,
        mv.assists,
        mv.kda,
        mv.time_played_seconds,
        mv.my_team_score,
        mv.enemy_team_score,
        COALESCE(pme.dominance_flag, 0) AS dominance_flag,
        COALESCE(pme.had_bot_teammate, FALSE) AS had_bot_teammate
    FROM mv_player_matches mv
    LEFT JOIN player.player_match_enrichment pme
        ON pme.match_id = mv.match_id
    WHERE mv.xuid = ?
      AND mv.outcome IN (?, ?)
      AND COALESCE(mv.time_played_seconds, 0) >= ?
      AND COALESCE(pme.had_bot_teammate, FALSE) = FALSE
      AND COALESCE(mv.is_firefight, FALSE) = FALSE
)
SELECT * FROM enriched
WHERE outcome = ?
ORDER BY
    (dominance_flag = ?) DESC,
    time_played_seconds ASC,
    ABS(COALESCE(my_team_score, 0) - COALESCE(enemy_team_score, 0)) DESC
LIMIT 10
"""


def _load_top_matches(
    db_path: str,
    xuid: str,
    *,
    best: bool,
) -> list[dict]:
    """Charge le Top 10 meilleurs ou pires matchs.

    Args:
        db_path: Chemin vers stats.duckdb du joueur.
        xuid: XUID du joueur.
        best: True pour les meilleures perf, False pour les pires.

    Returns:
        Liste de dicts avec les colonnes du match.
    """
    shared_path = get_shared_matches_path()
    if not shared_path.exists():
        logger.debug("shared_matches.duckdb introuvable pour top matches")
        return []

    target_outcome = int(Outcome.WIN) if best else int(Outcome.LOSS)
    dom_flag = int(DominanceFlag.DOMINATION) if best else int(DominanceFlag.HUMILIATION)

    try:
        with duckdb_read_only(str(shared_path)) as conn:
            conn.execute(f"ATTACH '{db_path}' AS player (READ_ONLY)")
            result = conn.execute(
                _TOP_MATCHES_SQL,
                [
                    xuid,
                    int(Outcome.WIN),
                    int(Outcome.LOSS),
                    MIN_MATCH_DURATION_SECONDS,
                    target_outcome,
                    dom_flag,
                ],
            )
            columns = [desc[0] for desc in result.description]
            rows = result.fetchall()
            matches = [dict(zip(columns, row, strict=True)) for row in rows]
            logger.debug("Top matches chargés (best=%s): %d résultats", best, len(matches))
            return matches
    except Exception as e:
        logger.warning("Erreur chargement top matches (best=%s): %s", best, e)
        return []


def load_top_best_matches(db_path: str, xuid: str) -> list[dict]:
    """Top 10 meilleures performances (victoires dominantes)."""
    return _load_top_matches(db_path, xuid, best=True)


def load_top_worst_matches(db_path: str, xuid: str) -> list[dict]:
    """Top 10 pires performances (défaites humiliantes)."""
    return _load_top_matches(db_path, xuid, best=False)
