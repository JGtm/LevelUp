"""Score composite continu [0, 1] pour TrueSkill 2.

Extrait de skill_rating.py — calcul du score de performance normalisé
qui sert de résultat dans la mise à jour TrueSkill.
"""

from __future__ import annotations

from typing import Any

from src.analysis.skill_rating_config import COMPOSITE_WEIGHTS
from src.utils.safe_types import clamp as _clamp
from src.utils.safe_types import safe_float as _safe_float


def _sigmoid_ratio(numerator: float, denominator: float) -> float:
    """Mappe ratio positif → [0,1] via sigmoid symétrique.

    ratio = 1.0 → 0.5 (performance attendue)
    ratio = 2.0 → ~0.67 (sur-performance)
    ratio = 0.5 → ~0.33 (sous-performance)
    """
    if denominator <= 0:
        return 0.5
    ratio = numerator / denominator
    return _clamp(ratio / (1.0 + ratio), 0.0, 1.0)


def _kills_vs_expected(
    row: dict[str, Any], teammate_avg_ke: float | None, _w: dict[str, float]
) -> tuple[str, float, float] | None:
    """Composante kills vs expected. Retourne (clé, score, poids) ou None."""
    kills = _safe_float(row.get("kills"))
    kills_expected = _safe_float(row.get("kills_expected"))
    if kills is None or kills_expected is None or kills_expected <= 0:
        return None
    score = _sigmoid_ratio(kills, kills_expected)
    if teammate_avg_ke is not None and kills_expected > 0 and teammate_avg_ke > 0:
        carry_ratio = kills_expected / teammate_avg_ke
        carry_adj = _clamp(carry_ratio, 0.5, 2.0)
        score = _clamp(score * (1.0 / carry_adj) + 0.5 * (1.0 - 1.0 / carry_adj), 0.0, 1.0)
    return "kills_vs_expected", score, _w["kills_vs_expected"]


def _deaths_vs_expected(
    row: dict[str, Any], _w: dict[str, float]
) -> tuple[str, float, float] | None:
    """Composante deaths vs expected (inversée). Retourne (clé, score, poids) ou None."""
    deaths = _safe_float(row.get("deaths"))
    deaths_expected = _safe_float(row.get("deaths_expected"))
    if deaths is None or deaths_expected is None or deaths_expected <= 0:
        return None
    return (
        "deaths_vs_expected",
        _sigmoid_ratio(deaths_expected, max(1.0, deaths)),
        _w["deaths_vs_expected"],
    )


def _win_outcome_factor(
    row: dict[str, Any], _w: dict[str, float]
) -> tuple[str, float, float] | None:
    """Composante win/loss outcome. Retourne (clé, score, poids) ou None."""
    outcome = row.get("outcome")
    if outcome is None:
        return None
    try:
        outcome_int = int(outcome)
    except (ValueError, TypeError):
        return None
    win_score = {2: 1.0, 1: 0.5, 3: 0.0, 4: 0.15}.get(outcome_int)
    return ("win_factor", win_score, _w["win_factor"]) if win_score is not None else None


def _damage_efficiency_component(
    row: dict[str, Any], avg_damage_eff: float | None, _w: dict[str, float]
) -> tuple[str, float, float] | None:
    """Composante efficacité des dégâts. Retourne (clé, score, poids) ou None."""
    damage_dealt = _safe_float(row.get("damage_dealt"))
    damage_taken = _safe_float(row.get("damage_taken"))
    if damage_dealt is None or damage_taken is None:
        return None
    total = damage_dealt + damage_taken
    if total <= 0:
        return None
    raw_eff = _clamp(damage_dealt / total, 0.0, 1.0)
    score_eff = (
        _sigmoid_ratio(raw_eff, avg_damage_eff)
        if avg_damage_eff and avg_damage_eff > 0
        else raw_eff
    )
    return "damage_efficiency", score_eff, _w["damage_efficiency"]


def _accuracy_delta_component(
    row: dict[str, Any], avg_accuracy: float | None, _w: dict[str, float]
) -> tuple[str, float, float] | None:
    """Composante delta précision. Retourne (clé, score, poids) ou None."""
    accuracy = _safe_float(row.get("accuracy"))
    if accuracy is None or avg_accuracy is None or avg_accuracy <= 0:
        return None
    return "accuracy_delta", _sigmoid_ratio(accuracy, avg_accuracy), _w["accuracy_delta"]


def compute_composite_score(  # noqa: PLR0913
    row: dict[str, Any],
    avg_accuracy: float | None,
    teammate_avg_ke: float | None,
    enemy_avg_ke: float | None,
    avg_damage_eff: float | None = None,
    *,
    weights: dict[str, float] | None = None,
) -> float:
    """Calcule le score composite continu [0, 1] d'un match.

    Ce score est l'équivalent du "résultat" dans TrueSkill 2 — il combine
    plusieurs métriques en un indicateur unique de performance.

    La valeur 0.5 correspond à une performance parfaitement attendue.
    > 0.5 : sur-performance, < 0.5 : sous-performance.

    Graceful degradation : si des métriques sont manquantes, les poids sont
    renormalisés automatiquement sur les métriques disponibles.
    """
    _w = weights if weights is not None else COMPOSITE_WEIGHTS
    entries = [
        _kills_vs_expected(row, teammate_avg_ke, _w),
        _deaths_vs_expected(row, _w),
        _win_outcome_factor(row, _w),
        _damage_efficiency_component(row, avg_damage_eff, _w),
        _accuracy_delta_component(row, avg_accuracy, _w),
    ]
    valid = [e for e in entries if e is not None]
    if not valid:
        return 0.5
    total_weight = sum(w for _, _, w in valid)
    if total_weight < 1e-12:
        return 0.5
    return _clamp(sum(s * w for _, s, w in valid) / total_weight, 0.0, 1.0)
