"""Score de performance d'une escouade (0-100).

Calcule un score collectif à partir des scores individuels de session v2,
avec des bonus pour le win rate d'équipe, la cohésion K/D et l'équilibre des frags.
"""

from __future__ import annotations

from src.analysis.performance_config import SQUAD_GRADE_THRESHOLDS, resolve_squad_grade
from src.utils.safe_types import clamp as _clamp


def compute_squad_performance_score(scores: list[dict]) -> dict:
    """Calcule le score collectif d'une escouade.

    Args:
        scores: liste de dicts issus de compute_session_performance_score_v2()
                (un par joueur, champ 'score' float 0-100).

    Returns:
        dict avec: score (float|None), grade (str), components (dict).
    """
    valid = [s for s in scores if s.get("score") is not None]
    if not valid:
        return {"score": None, "grade": "N/A", "components": {}}

    base = sum(s["score"] for s in valid) / len(valid)
    bonuses, comps = _compute_squad_bonuses(valid)
    final = _clamp(base + bonuses, lo=0.0, hi=100.0)
    return {
        "score": round(final, 1),
        "grade": resolve_squad_grade(final),
        "components": {"base_avg": round(base, 1), **comps},
    }


def _compute_squad_bonuses(scores: list[dict]) -> tuple[float, dict]:
    """Calcule les bonus collectifs (win rate, cohésion, équilibre).

    Returns:
        (bonus_total, composantes_dict).
    """
    bonus = 0.0
    comps: dict[str, float | None] = {}

    # Bonus win rate : +5 si win_rate_équipe > 60 %
    win_rates = [s["win_rate"] for s in scores if s.get("win_rate") is not None]
    if win_rates:
        team_win_rate = sum(win_rates) / len(win_rates)
        comps["team_win_rate"] = round(team_win_rate, 1)
        if team_win_rate > 60.0:
            bonus += 5.0

    # Bonus cohésion : +5 si min(K/D) > 1.0
    kd_ratios = [s["kd_ratio"] for s in scores if s.get("kd_ratio") is not None]
    if kd_ratios and min(kd_ratios) > 1.0:
        comps["min_kd"] = round(min(kd_ratios), 2)
        bonus += 5.0

    # Bonus équilibre : +3 si écart-type des frags < 3.0
    kill_lists = [s["kills"] for s in scores if s.get("kills") is not None]
    if len(kill_lists) >= 2:
        mean_kills = sum(kill_lists) / len(kill_lists)
        variance = sum((k - mean_kills) ** 2 for k in kill_lists) / len(kill_lists)
        std_kills = variance**0.5
        comps["kills_std"] = round(std_kills, 1)
        if std_kills < 3.0:
            bonus += 3.0

    return bonus, comps


__all__ = ["compute_squad_performance_score", "SQUAD_GRADE_THRESHOLDS"]
