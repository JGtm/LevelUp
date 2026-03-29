"""Module d'analyse des performances cumulées avec Polars.

Sprint 6: Fonctions pour calculer les séries cumulées (net score, K/D, objectifs).
"""

from __future__ import annotations

import logging
from typing import Any

import polars as pl

logger = logging.getLogger(__name__)

# Re-exports pour compatibilité descendante
from src.analysis._cumulative_results import CumulativeMetricsResult, CumulativeSeriesResult
from src.analysis._cumulative_series import (
    compute_cumulative_kd_series_polars,
    compute_cumulative_kda_series_polars,
    compute_cumulative_net_score_series_polars,
    compute_cumulative_objective_score_series_polars,
    compute_rolling_kd_polars,
)

__all__ = [
    "CumulativeSeriesResult",
    "CumulativeMetricsResult",
    "compute_cumulative_net_score_series_polars",
    "compute_cumulative_kd_series_polars",
    "compute_cumulative_kda_series_polars",
    "compute_cumulative_objective_score_series_polars",
    "compute_cumulative_metrics_polars",
    "compute_rolling_kd_polars",
    "compute_session_trend_polars",
    "cumulative_series_to_dicts",
]


# =============================================================================
# Fonctions Polars - Métriques agrégées et tendances
# =============================================================================


def compute_cumulative_metrics_polars(
    match_stats_df: pl.DataFrame,
) -> CumulativeMetricsResult:
    """Calcule les métriques cumulées finales pour une session."""
    if match_stats_df.is_empty():
        return CumulativeMetricsResult(
            total_kills=0,
            total_deaths=0,
            total_assists=0,
            cumulative_net_score=0,
            cumulative_kd=0.0,
            cumulative_efficiency=0.0,
            matches_count=0,
        )

    totals = match_stats_df.select(
        [
            pl.col("kills").fill_null(0).sum().alias("total_kills"),
            pl.col("deaths").fill_null(0).sum().alias("total_deaths"),
            pl.col("assists").fill_null(0).sum().alias("total_assists"),
            pl.len().alias("matches_count"),
        ]
    ).row(0, named=True)

    total_kills = int(totals["total_kills"])
    total_deaths = int(totals["total_deaths"])
    total_assists = int(totals["total_assists"])
    matches_count = int(totals["matches_count"])

    net_score = total_kills - total_deaths
    kd = total_kills / max(1, total_deaths)
    efficiency = (total_kills + total_assists) / max(1, total_deaths)

    return CumulativeMetricsResult(
        total_kills=total_kills,
        total_deaths=total_deaths,
        total_assists=total_assists,
        cumulative_net_score=net_score,
        cumulative_kd=round(kd, 2),
        cumulative_efficiency=round(efficiency, 2),
        matches_count=matches_count,
    )


def compute_session_trend_polars(
    match_stats_df: pl.DataFrame,
) -> dict[str, Any]:
    """Calcule la tendance d'une session (amélioration ou dégradation).

    Compare la première et la seconde moitié de la session.
    """
    if match_stats_df.is_empty() or len(match_stats_df) < 4:
        return {
            "trend": "stable",
            "first_half_kd": None,
            "second_half_kd": None,
            "kd_change": None,
            "kd_change_pct": None,
        }

    df = match_stats_df.sort("start_time")
    mid = len(df) // 2

    first_half = df.head(mid)
    first_kills = first_half.select(pl.col("kills").fill_null(0).sum()).item()
    first_deaths = first_half.select(pl.col("deaths").fill_null(0).sum()).item()
    first_kd = first_kills / max(1, first_deaths)

    second_half = df.tail(len(df) - mid)
    second_kills = second_half.select(pl.col("kills").fill_null(0).sum()).item()
    second_deaths = second_half.select(pl.col("deaths").fill_null(0).sum()).item()
    second_kd = second_kills / max(1, second_deaths)

    kd_change = second_kd - first_kd
    kd_change_pct = (kd_change / first_kd * 100) if first_kd > 0 else 0

    if kd_change_pct > 10:
        trend = "improving"
    elif kd_change_pct < -10:
        trend = "declining"
    else:
        trend = "stable"

    return {
        "trend": trend,
        "first_half_kd": round(first_kd, 2),
        "second_half_kd": round(second_kd, 2),
        "kd_change": round(kd_change, 2),
        "kd_change_pct": round(kd_change_pct, 1),
    }


def cumulative_series_to_dicts(
    df: pl.DataFrame,
) -> list[dict[str, Any]]:
    """Convertit un DataFrame Polars en liste de dicts pour Plotly."""
    if df.is_empty():
        logger.debug("cumulative_series_to_dicts: df vide, retour []")
        return []
    return df.to_dicts()
