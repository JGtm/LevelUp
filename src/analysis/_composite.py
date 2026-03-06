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


def compute_composite_score(  # noqa: C901, PLR0912, PLR0913
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
    components: dict[str, float] = {}
    weights_used: dict[str, float] = {}
    _w = weights if weights is not None else COMPOSITE_WEIGHTS

    # ── 1. Kills vs Expected ──
    kills = _safe_float(row.get("kills"))
    kills_expected = _safe_float(row.get("kills_expected"))
    if kills is not None and kills_expected is not None and kills_expected > 0:
        score = _sigmoid_ratio(kills, kills_expected)
        if teammate_avg_ke is not None and kills_expected > 0 and teammate_avg_ke > 0:
            carry_ratio = kills_expected / teammate_avg_ke
            carry_adj = _clamp(carry_ratio, 0.5, 2.0)
            score = _clamp(score * (1.0 / carry_adj) + 0.5 * (1.0 - 1.0 / carry_adj), 0.0, 1.0)
        components["kills_vs_expected"] = score
        weights_used["kills_vs_expected"] = _w["kills_vs_expected"]

    # ── 2. Deaths vs Expected (inversé) ──
    deaths = _safe_float(row.get("deaths"))
    deaths_expected = _safe_float(row.get("deaths_expected"))
    if deaths is not None and deaths_expected is not None and deaths_expected > 0:
        deaths_safe = max(1.0, deaths)
        score = _sigmoid_ratio(deaths_expected, deaths_safe)
        components["deaths_vs_expected"] = score
        weights_used["deaths_vs_expected"] = _w["deaths_vs_expected"]

    # ── 3. Win factor ──
    outcome = row.get("outcome")
    if outcome is not None:
        try:
            outcome_int = int(outcome)
        except (ValueError, TypeError):
            outcome_int = -1
        win_map = {2: 1.0, 1: 0.5, 3: 0.0, 4: 0.15}
        win_score = win_map.get(outcome_int)
        if win_score is not None:
            components["win_factor"] = win_score
            weights_used["win_factor"] = _w["win_factor"]

    # ── 4. Damage efficiency ──
    damage_dealt = _safe_float(row.get("damage_dealt"))
    damage_taken = _safe_float(row.get("damage_taken"))
    if damage_dealt is not None and damage_taken is not None:
        total = damage_dealt + damage_taken
        if total > 0:
            raw_eff = _clamp(damage_dealt / total, 0.0, 1.0)
            if avg_damage_eff is not None and avg_damage_eff > 0:
                score_eff = _sigmoid_ratio(raw_eff, avg_damage_eff)
            else:
                score_eff = raw_eff
            components["damage_efficiency"] = score_eff
            weights_used["damage_efficiency"] = _w["damage_efficiency"]

    # ── 5. Accuracy delta ──
    accuracy = _safe_float(row.get("accuracy"))
    if accuracy is not None and avg_accuracy is not None and avg_accuracy > 0:
        score = _sigmoid_ratio(accuracy, avg_accuracy)
        components["accuracy_delta"] = score
        weights_used["accuracy_delta"] = _w["accuracy_delta"]

    if not components:
        return 0.5

    total_weight = sum(weights_used.values())
    if total_weight < 1e-12:
        return 0.5

    composite = sum(components[k] * weights_used[k] for k in components) / total_weight
    return _clamp(composite, 0.0, 1.0)
