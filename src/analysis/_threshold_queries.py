"""Requêtes SQL et agrégation des seuils radar globaux.

Sous-fonctions privées extraites de compute_global_radar_thresholds
pour respecter les limites de complexité.
"""

from __future__ import annotations

import logging
import statistics
from collections.abc import Callable
from dataclasses import dataclass, field
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    pass

logger = logging.getLogger(__name__)

_FACTOR = 0.85

_EXCLUDE_SQL = """
    AND match_id IN (
        SELECT match_id FROM shared.match_registry
        WHERE (LOWER(COALESCE(pair_name,'')) NOT LIKE '%firefight%')
          AND (LOWER(COALESCE(pair_name,'')) NOT LIKE '%btb%')
          AND (LOWER(COALESCE(pair_name,'')) NOT LIKE '%big team%')
          AND (LOWER(COALESCE(pair_name,'')) NOT LIKE '%grande équipe%')
    )
"""


@dataclass
class _CollectedStats:
    """Stats agrégées sur l'ensemble des DBs joueur."""

    max_kill: float = 0.0
    max_obj: float = 0.0
    max_assist: float = 0.0
    max_score: float = 0.0
    max_impact: float = 0.0
    seen_any: bool = False
    mode_scores: dict[str, list[float]] = field(default_factory=dict)

    def merge(self, other: _CollectedStats) -> None:
        """Fusionne les stats d'un autre joueur dans cet agrégat."""
        self.max_kill = max(self.max_kill, other.max_kill)
        self.max_obj = max(self.max_obj, other.max_obj)
        self.max_assist = max(self.max_assist, other.max_assist)
        self.max_score = max(self.max_score, other.max_score)
        self.max_impact = max(self.max_impact, other.max_impact)
        self.seen_any = self.seen_any or other.seen_any
        for family, scores in other.mode_scores.items():
            self.mode_scores.setdefault(family, []).extend(scores)


def query_player_db_stats(
    conn: object,
    shared_path_sql: str,
    get_mode_family_fn: Callable[[str | None], str],
) -> _CollectedStats:
    """Exécute les 4 requêtes de seuils sur une connexion DuckDB joueur ouverte.

    Args:
        conn: Connexion DuckDB déjà ouverte (shared ATTACH déjà effectué).
        shared_path_sql: Chemin SQL-escaped de shared_matches.duckdb (non utilisé ici,
            présent pour cohérence d'interface).
        get_mode_family_fn: Fonction de résolution de famille de mode.

    Returns:
        _CollectedStats avec les maximaux et scores per-mode.
    """
    stats = _CollectedStats()

    # Requête 1 : max par catégorie hors Firefight/BTB
    r = conn.execute(f"""  # type: ignore[union-attr]
        SELECT award_category, MAX(total) as m FROM (
            SELECT p.match_id, p.award_category, SUM(p.award_score) as total
            FROM personal_score_awards p
            WHERE p.award_category IN ('kill','assist','objective','vehicle')
            {_EXCLUDE_SQL}
            GROUP BY p.match_id, p.award_category
        ) GROUP BY award_category
    """).fetchall()
    for cat, m in r or []:
        m = float(m or 0)
        if cat == "kill":
            stats.max_kill = max(stats.max_kill, m)
        elif cat == "assist":
            stats.max_assist = max(stats.max_assist, m)
        elif cat == "objective":
            stats.max_obj = max(stats.max_obj, m)
        stats.seen_any = True

    # Requête 2 : max score total positif par match
    r2 = conn.execute(f"""  # type: ignore[union-attr]
        SELECT MAX(s) FROM (
            SELECT p.match_id,
                GREATEST(0, SUM(CASE WHEN p.award_score > 0 THEN p.award_score ELSE 0 END)) as s
            FROM personal_score_awards p
            WHERE 1=1 {_EXCLUDE_SQL}
            GROUP BY p.match_id
        )
    """).fetchone()
    if r2 and r2[0] is not None:
        stats.max_score = max(stats.max_score, float(r2[0]))
        stats.seen_any = True

    # Requête 3 : max impact (pts/min)
    try:
        r3 = conn.execute(f"""  # type: ignore[union-attr]
            SELECT MAX(agg.total_pos / NULLIF(ms.duration_seconds / 60.0, 0)) FROM (
                SELECT p.match_id,
                    SUM(CASE WHEN p.award_category IN ('kill','assist','objective','vehicle')
                        AND p.award_score > 0 THEN p.award_score ELSE 0 END) as total_pos
                FROM personal_score_awards p
                WHERE 1=1 {_EXCLUDE_SQL}
                GROUP BY p.match_id
            ) agg
            JOIN shared.match_registry ms ON agg.match_id = ms.match_id
            WHERE ms.duration_seconds > 0
            AND (LOWER(COALESCE(ms.pair_name,'')) NOT LIKE '%firefight%')
            AND (LOWER(COALESCE(ms.pair_name,'')) NOT LIKE '%btb%')
        """).fetchone()
        if r3 and r3[0] is not None and float(r3[0]) > 0:
            stats.max_impact = max(stats.max_impact, float(r3[0]))
            stats.seen_any = True
    except Exception as e:
        logger.debug("radar_thresholds: calcul impact échoué: %s", e)

    # Requête 4 : p90 des scores par mode
    try:
        mode_rows = conn.execute("""  # type: ignore[union-attr]
            SELECT r.pair_name, p.award_category, SUM(p.award_score) AS score
            FROM personal_score_awards p
            JOIN shared.match_registry r ON p.match_id = r.match_id
            WHERE p.award_category IN ('objective', 'kill')
              AND (LOWER(COALESCE(r.pair_name,'')) NOT LIKE '%firefight%')
            GROUP BY p.match_id, r.pair_name, p.award_category
        """).fetchall()
        for pn, cat, sc in mode_rows or []:
            family = get_mode_family_fn(pn)
            is_family_obj = family not in ("slayer", "fiesta", "other")
            if (is_family_obj and cat == "objective") or (not is_family_obj and cat == "kill"):
                stats.mode_scores.setdefault(family, []).append(float(sc or 0))
    except Exception as e:
        logger.debug("radar_thresholds: per-mode échoué: %s", e)

    return stats


def build_thresholds_result(
    agg: _CollectedStats,
    fallback: dict[str, float],
    fallback_per_mode: dict[str, float],
) -> dict[str, float]:
    """Calcule le dict final de seuils à partir des stats agrégées.

    Args:
        agg: Stats collectées sur toutes les DBs joueur.
        fallback: RADAR_THRESHOLDS (valeurs par défaut globales).
        fallback_per_mode: RADAR_THRESHOLDS_PER_MODE.

    Returns:
        Dict de seuils prêt à mettre en cache.
    """
    objectifs = agg.max_obj if agg.max_obj > 0 else fallback["objectifs"]
    per_mode: dict[str, float] = dict(fallback_per_mode)
    for family, scores in agg.mode_scores.items():
        if len(scores) >= 2:
            per_mode[family] = max(1.0, float(statistics.quantiles(scores, n=100)[89]))
        elif scores:
            per_mode[family] = max(1.0, scores[0])
    logger.info(
        "radar_thresholds: p90/mode — %s",
        {k: round(v) for k, v in per_mode.items() if k in agg.mode_scores},
    )
    return {
        "objectifs": objectifs * _FACTOR,
        "combat": max(agg.max_kill, 1.0) * _FACTOR,
        "support": max(agg.max_assist, 1.0) * _FACTOR,
        "score": max(agg.max_score, 1.0) * _FACTOR,
        "impact_pts_per_min": max(agg.max_impact, 1.0) * _FACTOR,
        "survie_deaths_per_min_ref": fallback["survie_deaths_per_min_ref"],
        "survie_avg_life_ref_seconds": fallback.get("survie_avg_life_ref_seconds", 90.0),
        "per_mode": per_mode,
    }
