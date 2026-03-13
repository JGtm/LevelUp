"""Score de performance RELATIF par match (0-100).

Compare la performance d'un match à l'historique personnel du joueur.
- 50 = match dans ta moyenne
- 100 = meilleur match de ton historique
- 0 = pire match de ton historique

Configuration centralisée dans : src/analysis/performance_config.py
"""

from __future__ import annotations

import logging
from typing import Any

import polars as pl

from src.analysis._performance_relative_helpers import (
    _DEFAULT_DURATION,  # noqa: F401
    _DURATION_CANDIDATES,  # noqa: F401
    _add_percentile,
    _add_rank_perf_diff_column,  # noqa: F401
    _add_vs_expected_column,  # noqa: F401
    _apply_bot_bonus,
    _compute_rank_performance,
    _fallback_kda_percentile,
    _normalize_df,
    _percentile_rank,  # noqa: F401
    _percentile_rank_inverse,  # noqa: F401
    _prepare_history_metrics,
    _resolve_duration_column,  # noqa: F401
    _safe_col,  # noqa: F401
)
from src.analysis.performance_config import MIN_MATCHES_FOR_RELATIVE, RELATIVE_WEIGHTS
from src.utils.safe_types import safe_float as _safe_float

logger = logging.getLogger(__name__)


# =============================================================================
# Extraction des métriques dun match
# =============================================================================


def _extract_match_values(row: dict[str, Any]) -> dict[str, Any] | None:
    """Extrait les métriques d'un match pour le calcul du score relatif.

    Returns:
        Dict avec les métriques calculées, ou None si extraction échoue.
    """
    try:
        duration = None
        for col in ["time_played_seconds", "duration_seconds", "match_duration_seconds"]:
            val = row.get(col)
            if val is not None:
                try:
                    duration = float(val)
                    if duration > 0:
                        break
                except (ValueError, TypeError):
                    pass
        if duration is None or duration <= 0:
            duration = 600.0

        minutes = duration / 60.0
        kills = float(row.get("kills") or 0)
        deaths = float(row.get("deaths") or 0)
        assists = float(row.get("assists") or 0)

        kda = row.get("kda")
        if kda is not None:
            try:
                kda = float(kda)
            except (ValueError, TypeError):
                kda = (kills + assists) / max(1, deaths)
        else:
            kda = (kills + assists) / max(1, deaths)

        accuracy = _safe_float(row.get("accuracy"))
        personal_score = _safe_float(row.get("personal_score"))
        damage_dealt = _safe_float(row.get("damage_dealt"))
        rank = _safe_float(row.get("rank"))
        team_mmr = _safe_float(row.get("team_mmr"))
        enemy_mmr = _safe_float(row.get("enemy_mmr"))
        kills_expected = _safe_float(row.get("kills_expected"))
        deaths_expected = _safe_float(row.get("deaths_expected"))

        return {
            "kpm": kills / minutes,
            "dpm_deaths": deaths / minutes,
            "apm": assists / minutes,
            "kda": kda,
            "accuracy": accuracy,
            "pspm": personal_score / minutes if personal_score is not None else None,
            "dpm_damage": damage_dealt / minutes if damage_dealt is not None else None,
            "rank": rank,
            "team_mmr": team_mmr,
            "enemy_mmr": enemy_mmr,
            "kills_vs_expected": (
                kills / kills_expected
                if kills_expected is not None and kills_expected > 0
                else None
            ),
            "deaths_vs_expected": (
                deaths_expected / max(1.0, deaths)
                if deaths_expected is not None and deaths_expected > 0
                else None
            ),
        }
    except Exception:
        logger.debug("Erreur extraction métriques match", exc_info=True)
        return None


_STANDARD_METRICS = ["kpm", "apm", "kda", "accuracy", "pspm", "dpm_damage"]
_INVERSE_METRICS = {"dpm_deaths"}
_VS_EXPECTED_METRICS = ["kills_vs_expected", "deaths_vs_expected"]


def _collect_percentiles(
    metrics: dict[str, Any],
    history_metrics: pl.DataFrame,
) -> tuple[dict[str, float], dict[str, float]]:
    """Collecte les percentiles et poids pour toutes les métriques."""
    percentiles: dict[str, float] = {}
    weights_used: dict[str, float] = {}

    for m in _STANDARD_METRICS:
        _add_percentile(m, metrics[m], history_metrics, percentiles, weights_used)
    _add_percentile(
        "dpm_deaths",
        metrics["dpm_deaths"],
        history_metrics,
        percentiles,
        weights_used,
        inverse=True,
    )

    if metrics["rank"] is not None and metrics["team_mmr"] and metrics["enemy_mmr"]:
        rank_perf = _compute_rank_performance(
            metrics["rank"],
            metrics["team_mmr"],
            metrics["enemy_mmr"],
            history_metrics,
        )
        if rank_perf is not None:
            percentiles["rank_perf"] = rank_perf
            weights_used["rank_perf"] = RELATIVE_WEIGHTS["rank_perf"]

    for m in _VS_EXPECTED_METRICS:
        _add_percentile(m, metrics[m], history_metrics, percentiles, weights_used)
    return percentiles, weights_used


# =============================================================================
# API publique
# =============================================================================


def compute_relative_performance_score(
    row: dict[str, Any],
    df_history: pl.DataFrame | Any,
    *,
    had_bot_teammate: bool = False,
) -> float | None:
    """Calcule le score de performance RELATIF d'un match (v4).

    Compare le match à l'historique personnel du joueur.
    Utilise 10 métriques avec graceful degradation si certaines sont absentes.

    Args:
        row: Dict du match avec kills, deaths, assists, kda, accuracy, etc.
        df_history: DataFrame de l'historique complet du joueur.
        had_bot_teammate: Si True et match perdu, applique un bonus de +5 pts.

    Returns:
        Score 0-100 ou None si pas assez de données.
    """
    df_history = _normalize_df(df_history)
    if df_history is None or df_history.is_empty():
        return None
    if len(df_history) < MIN_MATCHES_FOR_RELATIVE:
        return None

    history_metrics = _prepare_history_metrics(df_history)
    metrics = _extract_match_values(row)
    if metrics is None:
        return None

    percentiles, weights_used = _collect_percentiles(metrics, history_metrics)

    if not percentiles:
        return None

    total_weight = sum(weights_used.values())
    if total_weight <= 0:
        return None

    score = sum(percentiles[k] * weights_used[k] for k in percentiles) / total_weight

    if had_bot_teammate:
        score = _apply_bot_bonus(score, row)

    return round(score, 1)


def compute_performance_series(
    df: pl.DataFrame,
    df_history: pl.DataFrame | None = None,
) -> pl.Series:
    """Calcule le score de performance pour chaque match d'un DataFrame.

    Args:
        df: DataFrame Polars des matchs à évaluer.
        df_history: Historique complet pour le calcul relatif. Si None, utilise df.

    Returns:
        Series Polars avec les scores de performance.
    """
    if df.is_empty():
        return pl.Series("performance", [], dtype=pl.Float64)

    history = df_history if df_history is not None else df

    if len(history) < MIN_MATCHES_FOR_RELATIVE:
        return _fallback_kda_percentile(df)

    scores = [
        compute_relative_performance_score(row_dict, history)
        for row_dict in df.iter_rows(named=True)
    ]

    return pl.Series("performance", scores, dtype=pl.Float64)
