"""Migration : colonnes team_0_ps_score / team_1_ps_score (shared_matches_v2.duckdb).

Ces colonnes stockent la somme des scores personnels (personal_score) de chaque
équipe, calculée depuis match_participants.

Contexte : l'API Halo Infinite retourne parfois dans CoreStats.Score la somme
des scores personnels pour une équipe et le score objectif (captures CTF, ticks
de zone) pour l'autre, rendant team_0_score / team_1_score incohérents dans
les modes BTB CTF / Total Control. Les colonnes ps_score sont toujours cohérentes
entre les deux équipes car elles sont recalculées depuis match_participants.
"""

from __future__ import annotations

import logging

import duckdb

from src.data.migration.registry import Migration, register
from src.data.sync.migrations import ensure_team_ps_scores

logger = logging.getLogger(__name__)


def _backfill_team_ps_scores(conn: duckdb.DuckDBPyConnection) -> None:
    """Calcule team_0/1_ps_score depuis match_participants pour tous les matchs.

    Idempotente : ne recalcule que les lignes avec ps_score NULL.
    """
    if not _table_exists(conn, "match_registry") or not _table_exists(conn, "match_participants"):
        logger.warning("backfill_team_ps_scores : tables manquantes, abandon")
        return

    result = conn.execute("""
        UPDATE match_registry
        SET
            team_0_ps_score = sub.s0,
            team_1_ps_score = sub.s1
        FROM (
            SELECT
                match_id,
                SUM(CASE WHEN team_id = 0 THEN COALESCE(score, 0) ELSE 0 END) AS s0,
                SUM(CASE WHEN team_id = 1 THEN COALESCE(score, 0) ELSE 0 END) AS s1
            FROM match_participants
            GROUP BY match_id
        ) sub
        WHERE match_registry.match_id = sub.match_id
          AND (match_registry.team_0_ps_score IS NULL
               OR match_registry.team_1_ps_score IS NULL)
    """).fetchone()
    count = result[0] if result else "?"
    logger.info("✅ backfill_team_ps_scores : %s matchs mis à jour", count)


def _table_exists(conn: duckdb.DuckDBPyConnection, name: str) -> bool:
    try:
        rows = conn.execute(
            "SELECT 1 FROM information_schema.tables WHERE table_name = ?", [name]
        ).fetchall()
        return len(rows) > 0
    except Exception:
        return False


register(
    Migration(
        name="add_team_ps_scores",
        target_db="shared",
        description=(
            "Colonnes team_0_ps_score/team_1_ps_score (somme personal_score par équipe) "
            "pour corriger l'incohérence CoreStats.Score API entre modes objectifs et Slayer"
        ),
        apply_schema=ensure_team_ps_scores,
        apply_backfill=_backfill_team_ps_scores,
    )
)
