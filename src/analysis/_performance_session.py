"""Score de performance par SESSION (0-100).

Évalue la performance globale d'une session de jeu (plusieurs matchs)
via des composantes pondérées : K/D, victoires, précision, KPM, survie,
objectif, performance MMR.
"""

from __future__ import annotations

import math
from collections.abc import Callable
from dataclasses import dataclass
from typing import Any

import polars as pl

from src.analysis._performance_relative import _normalize_df
from src.data.domain.refdata import Outcome
from src.utils.safe_types import clamp as _clamp

# =============================================================================
# Helpers
# =============================================================================


def _mean_numeric(df: pl.DataFrame, column: str) -> float | None:
    """Moyenne d'une colonne numérique, None si absente ou vide."""
    if column not in df.columns:
        return None
    values = df.get_column(column).cast(pl.Float64, strict=False).drop_nulls()
    if values.is_empty():
        return None
    return float(values.mean())


def _sum_int(df: pl.DataFrame, column: str) -> int:
    """Somme entière d'une colonne, 0 si absente."""
    if column not in df.columns:
        return 0
    values = df.get_column(column).cast(pl.Float64, strict=False).fill_null(0)
    return int(values.sum())


def _count_wins(df: pl.DataFrame) -> int:
    """Compte les victoires (outcome == Outcome.WIN)."""
    if "outcome" not in df.columns:
        return 0
    return int(
        df.get_column("outcome").cast(pl.Float64, strict=False).fill_null(0).eq(Outcome.WIN).sum()
    )


def _saturation_score(x: float, scale: float) -> float:
    """Score 0–100 avec saturation exponentielle.

    x=0 → 0, x ~= scale*ln(2) → 50, x → +inf → 100.
    """
    if scale <= 0 or x <= 0:
        return 0.0
    return _clamp(100.0 * (1.0 - math.exp(-x / scale)))


# =============================================================================
# Composantes de score
# =============================================================================


@dataclass(frozen=True)
class ScoreComponent:
    """Une composante de score (0–100) avec une pondération."""

    key: str
    label: str
    weight: float
    compute: Callable[[pl.DataFrame], tuple[float | None, dict[str, Any]]]


def _compute_kd_component(df: pl.DataFrame) -> tuple[float | None, dict[str, Any]]:
    """Composante K/D."""
    kills = _sum_int(df, "kills")
    deaths = _sum_int(df, "deaths")
    if kills == 0 and deaths == 0:
        return None, {"kd_ratio": None}
    kd_ratio = (kills / deaths) if deaths > 0 else float(kills)
    return _clamp(kd_ratio * 50.0), {"kd_ratio": round(kd_ratio, 2)}


def _compute_win_component(df: pl.DataFrame) -> tuple[float | None, dict[str, Any]]:
    """Composante victoires."""
    if "outcome" not in df.columns:
        return None, {"win_rate": None}
    n = len(df)
    if n <= 0:
        return None, {"win_rate": None}
    wins = _count_wins(df)
    win_rate = wins / n
    return _clamp(win_rate * 100.0), {"win_rate": round(win_rate * 100.0, 1)}


def _compute_accuracy_component(df: pl.DataFrame) -> tuple[float | None, dict[str, Any]]:
    """Composante précision."""
    acc = _mean_numeric(df, "accuracy") or _mean_numeric(df, "shots_accuracy")
    if acc is None:
        return None, {"accuracy": None}
    return _clamp(acc), {"accuracy": round(acc, 1)}


def _compute_kpm_component(df: pl.DataFrame) -> tuple[float | None, dict[str, Any]]:
    """Composante kills par minute."""
    kpm = _mean_numeric(df, "kills_per_min")
    if kpm is None:
        return None, {"kills_per_min": None}
    return _saturation_score(kpm, scale=0.8), {"kills_per_min": round(kpm, 2)}


def _compute_life_component(df: pl.DataFrame) -> tuple[float | None, dict[str, Any]]:
    """Composante survie (durée de vie moyenne)."""
    life = _mean_numeric(df, "average_life_seconds")
    if life is None:
        return None, {"avg_life_seconds": None}
    return _saturation_score(life, scale=50.0), {"avg_life_seconds": round(life, 1)}


_OBJECTIVE_COLUMN_WEIGHTS: dict[str, float] = {
    "flag_captures": 3.0,
    "flag_returns": 1.0,
    "zones_captured": 2.0,
    "zones_defended": 1.0,
    "ball_time_seconds": 1.0 / 30.0,
    "time_with_ball_seconds": 1.0 / 30.0,
    "hill_time_seconds": 1.0 / 30.0,
    "time_in_hill_seconds": 1.0 / 30.0,
    "core_captures": 3.0,
    "objective_carries": 1.0,
}


def _compute_objective_component(df: pl.DataFrame) -> tuple[float | None, dict[str, Any]]:
    """Composante objectif (CTF, Strongholds, Oddball, etc.)."""
    used: dict[str, float] = {}
    total_points = 0.0
    for col, w in _OBJECTIVE_COLUMN_WEIGHTS.items():
        if col not in df.columns:
            continue
        values = df.get_column(col).cast(pl.Float64, strict=False).drop_nulls()
        if values.is_empty():
            continue
        mean_val = float(values.mean())
        if mean_val <= 0:
            continue
        used[col] = w
        total_points += mean_val * w

    if not used:
        return None, {
            "objective_score": None,
            "objective_points_per_match": None,
            "objective_columns": [],
        }
    score = _saturation_score(total_points, scale=3.0)
    return score, {
        "objective_score": round(score, 1),
        "objective_points_per_match": round(total_points, 2),
        "objective_columns": sorted(used.keys()),
    }


def _compute_mmr_performance_component(  # noqa: C901
    df: pl.DataFrame,
) -> tuple[float | None, dict[str, Any]]:
    """Composante performance vs MMR attendu (style Elo)."""
    team_mmr = _mean_numeric(df, "team_mmr")
    enemy_mmr = _mean_numeric(df, "enemy_mmr")
    _empty_meta = {
        "expected_win_rate": None,
        "actual_win_rate": None,
        "performance_vs_expected": None,
    }

    if team_mmr is None or enemy_mmr is None:
        return None, _empty_meta

    mmr_diff = team_mmr - enemy_mmr
    expected_win_rate = 1.0 / (1.0 + math.pow(10, -mmr_diff / 400.0))

    if "outcome" not in df.columns or len(df) <= 0:
        return None, {
            "expected_win_rate": round(expected_win_rate * 100, 1),
            "actual_win_rate": None,
            "performance_vs_expected": None,
        }

    wins = _count_wins(df)
    actual_win_rate = wins / len(df)
    performance_diff = actual_win_rate - expected_win_rate
    score = _clamp(50.0 + (performance_diff * 100.0), 0.0, 100.0)

    return score, {
        "expected_win_rate": round(expected_win_rate * 100, 1),
        "actual_win_rate": round(actual_win_rate * 100, 1),
        "performance_vs_expected": round(performance_diff * 100, 1),
        "mmr_diff": round(mmr_diff, 1),
    }


def _compute_mmr_aggregates(df: pl.DataFrame) -> dict[str, float | None]:
    """Calcule les agrégats MMR."""
    team = _mean_numeric(df, "team_mmr")
    enemy = _mean_numeric(df, "enemy_mmr")
    delta = (team - enemy) if (team is not None and enemy is not None) else None
    return {
        "team_mmr_avg": round(team, 1) if team is not None else None,
        "enemy_mmr_avg": round(enemy, 1) if enemy is not None else None,
        "delta_mmr_avg": round(delta, 1) if delta is not None else None,
    }


def _mmr_difficulty_multiplier(delta_mmr_avg: float | None) -> float:
    """Applique un ajustement léger selon la difficulté."""
    if delta_mmr_avg is None:
        return 1.0
    adj = _clamp((-delta_mmr_avg / 300.0) * 5.0, lo=-5.0, hi=5.0) / 100.0
    return 1.0 + adj


# =============================================================================
# Score de session v1 (legacy)
# =============================================================================


_EMPTY_V1_RESULT: dict[str, Any] = {
    "score": None,
    "kd_ratio": None,
    "efficiency": None,
    "win_rate": None,
    "accuracy": None,
    "avg_score": None,
    "avg_life_seconds": None,
    "matches": 0,
    "kills": 0,
    "deaths": 0,
    "assists": 0,
    "team_mmr_avg": None,
    "enemy_mmr_avg": None,
    "delta_mmr_avg": None,
}


def compute_session_performance_score_v1(df_session: pl.DataFrame | Any) -> dict[str, Any]:
    """Version historique du score de session (0-100)."""
    df_session = _normalize_df(df_session)
    if df_session is None or df_session.is_empty():
        return dict(_EMPTY_V1_RESULT)

    total_kills = _sum_int(df_session, "kills")
    total_deaths = _sum_int(df_session, "deaths")
    total_assists = _sum_int(df_session, "assists")
    n_matches = len(df_session)

    kd_ratio = total_kills / total_deaths if total_deaths > 0 else float(total_kills)
    kd_score = _clamp(kd_ratio * 50.0)
    efficiency = (
        (total_kills + total_assists) / total_deaths
        if total_deaths > 0
        else float(total_kills + total_assists)
    )

    wins = _count_wins(df_session) if "outcome" in df_session.columns else 0
    win_rate = wins / n_matches if n_matches > 0 else 0.0
    win_score = win_rate * 100.0

    accuracy = _mean_numeric(df_session, "accuracy") or _mean_numeric(
        df_session,
        "shots_accuracy",
    )
    acc_score = accuracy if accuracy is not None else 50.0

    avg_life_seconds = _mean_numeric(df_session, "average_life_seconds")
    avg_score = _mean_numeric(df_session, "match_score")
    score_pts = _clamp((avg_score or 10.0) * 5.0) if avg_score is not None else 50.0

    mmr = _compute_mmr_aggregates(df_session)
    final_score = (kd_score * 0.30) + (win_score * 0.25) + (acc_score * 0.25) + (score_pts * 0.20)

    return {
        "score": round(final_score, 1),
        "kd_ratio": round(kd_ratio, 2),
        "efficiency": round(efficiency, 2),
        "win_rate": round(win_rate * 100.0, 1),
        "accuracy": round(accuracy, 1) if accuracy is not None else None,
        "avg_score": round(avg_score, 1) if avg_score is not None else None,
        "avg_life_seconds": round(avg_life_seconds, 1) if avg_life_seconds is not None else None,
        "matches": n_matches,
        "kills": total_kills,
        "deaths": total_deaths,
        "assists": total_assists,
        **mmr,
    }


# =============================================================================
# Score de session v2 (modulaire)
# =============================================================================

_SESSION_COMPONENTS: list[ScoreComponent] = [
    ScoreComponent(key="kd", label="F/D", weight=0.20, compute=_compute_kd_component),
    ScoreComponent(key="win", label="Victoires", weight=0.15, compute=_compute_win_component),
    ScoreComponent(key="acc", label="Précision", weight=0.15, compute=_compute_accuracy_component),
    ScoreComponent(key="kpm", label="Kills/min", weight=0.15, compute=_compute_kpm_component),
    ScoreComponent(key="life", label="Survie", weight=0.10, compute=_compute_life_component),
    ScoreComponent(key="obj", label="Objectif", weight=0.10, compute=_compute_objective_component),
    ScoreComponent(
        key="mmr_perf",
        label="MMR Performance",
        weight=0.15,
        compute=_compute_mmr_performance_component,
    ),
]


def compute_session_performance_score_v2(
    df_session: pl.DataFrame | Any,
    *,
    include_mmr_adjustment: bool = True,
) -> dict[str, Any]:
    """Score de performance modulaire (0–100).

    N'utilise que les composantes disponibles et renormalise les poids.

    Args:
        df_session: DataFrame des matchs de la session.
        include_mmr_adjustment: Inclure l'ajustement MMR.

    Returns:
        Dict avec score, composantes, confiance, et détails.
    """
    df_session = _normalize_df(df_session)

    if df_session is None or df_session.is_empty():
        base = compute_session_performance_score_v1(df_session)
        base.update(
            {
                "components": {},
                "weights_used": {},
                "confidence": 0.0,
                "confidence_label": "faible",
                "objective_score": None,
                "objective_points_per_match": None,
                "objective_columns": [],
                "version": "v2",
            }
        )
        return base

    total_kills = _sum_int(df_session, "kills")
    total_deaths = _sum_int(df_session, "deaths")
    total_assists = _sum_int(df_session, "assists")
    n_matches = len(df_session)

    kd_ratio = (total_kills / total_deaths) if total_deaths > 0 else float(total_kills)
    efficiency = (
        (total_kills + total_assists) / total_deaths
        if total_deaths > 0
        else float(total_kills + total_assists)
    )
    avg_life_seconds = _mean_numeric(df_session, "average_life_seconds")
    accuracy = _mean_numeric(df_session, "accuracy") or _mean_numeric(
        df_session,
        "shots_accuracy",
    )
    mmr = _compute_mmr_aggregates(df_session)

    computed_scores, component_meta, weights_used = _evaluate_components(df_session)

    final_score = _weighted_score(computed_scores, weights_used, mmr, include_mmr_adjustment)

    confidence = _clamp((n_matches / 10.0) * 100.0, lo=0.0, hi=100.0) / 100.0
    confidence_label = "faible" if n_matches < 4 else ("moyenne" if n_matches < 10 else "élevée")
    obj_meta = component_meta.get("obj", {})

    return _build_v2_result(
        final_score=final_score,
        kd_ratio=kd_ratio,
        efficiency=efficiency,
        accuracy=accuracy,
        avg_life_seconds=avg_life_seconds,
        n_matches=n_matches,
        total_kills=total_kills,
        total_deaths=total_deaths,
        total_assists=total_assists,
        mmr=mmr,
        obj_meta=obj_meta,
        computed_scores=computed_scores,
        weights_used=weights_used,
        confidence=confidence,
        confidence_label=confidence_label,
        component_meta=component_meta,
    )


def _build_v2_result(  # noqa: PLR0913
    *,
    final_score: float | None,
    kd_ratio: float,
    efficiency: float,
    accuracy: float | None,
    avg_life_seconds: float | None,
    n_matches: int,
    total_kills: int,
    total_deaths: int,
    total_assists: int,
    mmr: dict,
    obj_meta: dict,
    computed_scores: dict[str, float],
    weights_used: dict[str, float],
    confidence: float,
    confidence_label: str,
    component_meta: dict,
) -> dict[str, Any]:
    """Construit le dict résultat v2."""
    return {
        "score": round(final_score, 1) if final_score is not None else None,
        "kd_ratio": round(kd_ratio, 2),
        "efficiency": round(efficiency, 2),
        "win_rate": component_meta.get("win", {}).get("win_rate"),
        "accuracy": round(accuracy, 1) if accuracy is not None else None,
        "avg_score": None,
        "avg_life_seconds": round(avg_life_seconds, 1) if avg_life_seconds is not None else None,
        "matches": n_matches,
        "kills": total_kills,
        "deaths": total_deaths,
        "assists": total_assists,
        **mmr,
        "objective_score": obj_meta.get("objective_score"),
        "objective_points_per_match": obj_meta.get("objective_points_per_match"),
        "objective_columns": obj_meta.get("objective_columns", []),
        "components": {k: round(v, 1) for k, v in computed_scores.items()},
        "weights_used": weights_used,
        "confidence": round(confidence, 2),
        "confidence_label": confidence_label,
        "version": "v2",
    }


def _evaluate_components(
    df_session: pl.DataFrame,
) -> tuple[dict[str, float], dict[str, Any], dict[str, float]]:
    """Évalue toutes les composantes de score."""
    computed_scores: dict[str, float] = {}
    component_meta: dict[str, Any] = {}
    weights_used: dict[str, float] = {}
    for comp in _SESSION_COMPONENTS:
        score, meta = comp.compute(df_session)
        component_meta[comp.key] = meta
        if score is None:
            continue
        computed_scores[comp.key] = float(score)
        weights_used[comp.key] = float(comp.weight)
    return computed_scores, component_meta, weights_used


def _weighted_score(
    computed_scores: dict[str, float],
    weights_used: dict[str, float],
    mmr: dict[str, float | None],
    include_mmr_adjustment: bool,
) -> float | None:
    """Calcule le score final pondéré."""
    total_weight = sum(weights_used.values())
    if total_weight <= 0:
        return None
    final_score = sum(computed_scores[key] * (w / total_weight) for key, w in weights_used.items())
    if include_mmr_adjustment:
        final_score *= _mmr_difficulty_multiplier(mmr.get("delta_mmr_avg"))
    return _clamp(final_score)
