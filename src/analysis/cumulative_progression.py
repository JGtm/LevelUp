"""Analyse de progression avancée — EWMA, IC 90%, régression linéaire, net score/heure.

Module complémentaire à cumulative.py, focalisé sur la mesure de L'amélioration
dans le temps plutôt que sur les simples cumuls.
"""

from __future__ import annotations

import logging
from typing import Any

import polars as pl

from src.data.domain.refdata import Outcome

logger = logging.getLogger(__name__)

# =============================================================================
# EWMA K/D
# =============================================================================


def compute_ewma_kd_polars(
    match_stats_df: pl.DataFrame,
    alpha: float = 0.2,
) -> pl.DataFrame:
    """Calcule le K/D lissé par moyenne mobile exponentielle (EWMA).

    EWMA_t = α · kd_t + (1-α) · EWMA_{t-1}

    Args:
        match_stats_df: DataFrame des matchs triés par start_time.
        alpha: Facteur de lissage (0 = très lisse, 1 = très réactif). Défaut 0.2.

    Returns:
        DataFrame avec colonnes : match_id (si présent), start_time, kd, ewma_kd,
        outcome (si présent).
    """
    if match_stats_df.is_empty():
        return pl.DataFrame(
            schema={
                "match_id": pl.Utf8,
                "start_time": pl.Utf8,
                "kd": pl.Float64,
                "ewma_kd": pl.Float64,
            }
        )

    logger.debug("compute_ewma_kd_polars: %d matchs, alpha=%.2f", len(match_stats_df), alpha)
    result = (
        match_stats_df.sort("start_time")
        .with_columns(
            (
                pl.col("kills").fill_null(0).cast(pl.Float64)
                / pl.when(pl.col("deaths").fill_null(0) == 0)
                .then(pl.lit(1.0))
                .otherwise(pl.col("deaths").fill_null(0).cast(pl.Float64))
            ).alias("kd")
        )
        .with_columns(pl.col("kd").ewm_mean(alpha=alpha, adjust=True).alias("ewma_kd"))
    )

    output_cols = ["start_time", "kd", "ewma_kd"]
    if "match_id" in result.columns:
        output_cols = ["match_id"] + output_cols
    if "outcome" in result.columns:
        output_cols.append("outcome")

    return result.select(output_cols)


# =============================================================================
# K/D cumulé + Intervalle de confiance 90%
# =============================================================================


def compute_cumulative_kd_with_ci(
    match_stats_df: pl.DataFrame,
    z: float = 1.645,
) -> pl.DataFrame:
    """Calcule le K/D cumulé avec intervalle de confiance à 90%.

    Utilise l'erreur standard de la moyenne expansive des K/D individuels :
    SE_t = sqrt(Var_expansive(kd) / t)
    CI = kd_cumul ± z * SE_t

    La bande se rétrécit avec le nombre de matchs, visualisant la convergence.

    Args:
        match_stats_df: DataFrame des matchs.
        z: Z-score (1.645 pour 90 %, 1.96 pour 95 %).

    Returns:
        DataFrame avec colonnes : match_id (si présent), start_time, kd,
        cumulative_kd, ci_lower, ci_upper.
    """
    if match_stats_df.is_empty():
        return pl.DataFrame(
            schema={
                "match_id": pl.Utf8,
                "start_time": pl.Utf8,
                "kd": pl.Float64,
                "cumulative_kd": pl.Float64,
                "ci_lower": pl.Float64,
                "ci_upper": pl.Float64,
            }
        )

    result = (
        match_stats_df.sort("start_time")
        .with_columns(
            (
                pl.col("kills").fill_null(0).cast(pl.Float64)
                / pl.when(pl.col("deaths").fill_null(0) == 0)
                .then(pl.lit(1.0))
                .otherwise(pl.col("deaths").fill_null(0).cast(pl.Float64))
            ).alias("kd"),
            pl.col("kills").fill_null(0).cum_sum().alias("cumulative_kills"),
            pl.col("deaths").fill_null(0).cum_sum().alias("cumulative_deaths"),
        )
        .with_columns(
            (
                pl.col("cumulative_kills").cast(pl.Float64)
                / pl.when(pl.col("cumulative_deaths") == 0)
                .then(pl.lit(1.0))
                .otherwise(pl.col("cumulative_deaths").cast(pl.Float64))
            ).alias("cumulative_kd"),
            # Indice 1-based pour SE = sqrt(Var / n)
            (pl.int_range(pl.len()) + 1).cast(pl.Float64).alias("_n"),
        )
        .with_columns(
            # Moyenne expansive = cum_sum / n
            (pl.col("kd").cum_sum() / pl.col("_n")).alias("_e_x"),
            (pl.col("kd").pow(2).cum_sum() / pl.col("_n")).alias("_e_x2"),
        )
        .with_columns(
            ((pl.col("_e_x2") - pl.col("_e_x").pow(2)).clip(lower_bound=0.0) / pl.col("_n")).alias(
                "_variance"
            )
        )
        .with_columns((pl.col("_variance").sqrt() * z).alias("_margin"))
        .with_columns(
            (pl.col("cumulative_kd") - pl.col("_margin")).clip(lower_bound=0.0).alias("ci_lower"),
            (pl.col("cumulative_kd") + pl.col("_margin")).alias("ci_upper"),
        )
    )

    output_cols = ["start_time", "kd", "cumulative_kd", "ci_lower", "ci_upper"]
    if "match_id" in result.columns:
        output_cols = ["match_id"] + output_cols

    return result.select(output_cols)


# =============================================================================
# Régression linéaire sur K/D et win rate
# =============================================================================


def compute_linear_regression_kd(
    series_df: pl.DataFrame,
    kd_col: str = "ewma_kd",
) -> dict[str, Any]:
    """Calcule la régression linéaire OLS sur une série K/D.

    x = indice du match (0, 1, 2, …)
    y = valeur de kd_col

    Args:
        series_df: DataFrame avec start_time et kd_col.
        kd_col: Colonne à régresser (par défaut « ewma_kd »).

    Returns:
        Dict avec les clés :
        - slope        : pente (Δ K/D par match)
        - intercept    : valeur initiale estimée
        - r_squared    : R² ∈ [0, 1]
        - y_hat        : valeurs prédites (list[float])
        - is_significant : bool (r_squared ≥ 0.3)
        - trend        : "improving" | "declining" | "stable"
        - x_labels     : liste start_time pour l'axe X
        - win_rate_slope (si colonne outcome présente)
        - win_rate_r2   (si colonne outcome présente)
        - win_y_hat     (si colonne outcome présente)
    """
    if series_df.is_empty() or kd_col not in series_df.columns or len(series_df) < 3:
        logger.debug(
            "compute_linear_regression_kd: données insuffisantes (n=%d, col=%s)",
            len(series_df),
            kd_col,
        )
        return _empty_regression()

    df = series_df.sort("start_time")
    y_values = df[kd_col].fill_null(0.0).to_list()
    n = len(y_values)

    slope, intercept, y_hat, r_squared = _ols(y_values)
    mean_y = sum(y_values) / n
    is_significant = r_squared >= 0.3

    # Variation relative sur toute la session (slope * n / moyenne)
    # Calculée inconditionnellement — le R² indique si la tendance est fiable
    kd_relative_change = round(slope * n / mean_y, 4) if mean_y > 0 else 0.0

    # Tendance : variation relative sur toute la période
    if mean_y > 0 and is_significant:
        if kd_relative_change > 0.05:
            trend = "improving"
        elif kd_relative_change < -0.05:
            trend = "declining"
        else:
            trend = "stable"
    else:
        trend = "stable"

    result: dict[str, Any] = {
        "slope": round(slope, 4),
        "intercept": round(intercept, 4),
        "r_squared": round(r_squared, 3),
        "y_hat": y_hat,
        "is_significant": is_significant,
        "trend": trend,
        "x_labels": df["start_time"].to_list() if "start_time" in df.columns else list(range(n)),
        "kd_relative_change": kd_relative_change,
    }

    if "outcome" in df.columns:
        wins = [1.0 if v == Outcome.WIN else 0.0 for v in df["outcome"].to_list()]
        wr_slope, _, wr_y_hat, wr_r2 = _ols(wins)
        mean_wins = sum(wins) / n  # noqa: F841 (conservé pour lisibilité)
        # Variation absolue (pp) — pas relative : diviser par mean_wins amplifiait
        # démesurément les sessions avec peu de victoires (ex : 1W/10 → 200%).
        wr_relative_change = round(wr_slope * n, 4)
        result["win_rate_slope"] = round(wr_slope, 4)
        result["win_rate_r2"] = round(wr_r2, 3)
        result["win_y_hat"] = wr_y_hat
        result["wr_relative_change"] = wr_relative_change

    return result


def _ols(y_values: list[float]) -> tuple[float, float, list[float], float]:
    """OLS minimal : retourne (slope, intercept, y_hat, r_squared)."""
    n = len(y_values)
    if n < 2:
        return 0.0, y_values[0] if y_values else 0.0, y_values, 0.0

    x_values = list(range(n))
    sum_x = sum(x_values)
    sum_y = sum(y_values)
    sum_xy = sum(x * y for x, y in zip(x_values, y_values, strict=False))
    sum_x2 = sum(x * x for x in x_values)

    denom = n * sum_x2 - sum_x**2
    if denom == 0:
        mean_y = sum_y / n
        return 0.0, mean_y, [mean_y] * n, 0.0

    slope = (n * sum_xy - sum_x * sum_y) / denom
    intercept = (sum_y - slope * sum_x) / n
    y_hat = [intercept + slope * x for x in x_values]

    mean_y = sum_y / n
    ss_res = sum((y - yh) ** 2 for y, yh in zip(y_values, y_hat, strict=False))
    ss_tot = sum((y - mean_y) ** 2 for y in y_values)
    r_squared = max(0.0, 1 - ss_res / ss_tot) if ss_tot > 0 else 0.0

    return slope, intercept, y_hat, r_squared


def _empty_regression() -> dict[str, Any]:
    return {
        "slope": 0.0,
        "intercept": 0.0,
        "r_squared": 0.0,
        "y_hat": [],
        "is_significant": False,
        "trend": "stable",
        "x_labels": [],
    }


# =============================================================================
# Net score normalisé par heure (rolling adaptatif)
# =============================================================================


def compute_rolling_net_score_per_hour(
    match_stats_df: pl.DataFrame,
    window_size: int | None = None,
) -> pl.DataFrame:
    """Calcule le net score normalisé par heure de jeu, en rolling adaptatif.

    Efficacité(t) = (frags − morts) / durée_joué(h), lissée sur N matchs.
    La fenêtre est adaptative : max(3, 10 % des matchs).

    Args:
        match_stats_df: DataFrame avec kills, deaths, time_played_seconds, start_time.
        window_size: Taille de fenêtre explicite (None = adaptatif).

    Returns:
        DataFrame avec start_time, net_score, net_per_hour, rolling_net_per_hour.
        Retourne DataFrame vide si time_played_seconds est absent.
    """
    _empty_schema = {
        "start_time": pl.Utf8,
        "net_score": pl.Int64,
        "net_per_hour": pl.Float64,
        "rolling_net_per_hour": pl.Float64,
    }
    if match_stats_df.is_empty():
        return pl.DataFrame(schema=_empty_schema)
    if "time_played_seconds" not in match_stats_df.columns:
        logger.debug("compute_rolling_net_score_per_hour: colonne time_played_seconds absente")
        return pl.DataFrame(schema=_empty_schema)

    n = len(match_stats_df)
    win = window_size if window_size is not None else max(3, n * 10 // 100)

    result = (
        match_stats_df.sort("start_time")
        .with_columns(
            (pl.col("kills").fill_null(0) - pl.col("deaths").fill_null(0)).alias("net_score"),
            pl.col("time_played_seconds").fill_null(0).cast(pl.Float64).alias("tps"),
        )
        .with_columns(
            pl.when(pl.col("tps") > 0)
            .then(pl.col("net_score").cast(pl.Float64) / (pl.col("tps") / 3600.0))
            .otherwise(0.0)
            .alias("net_per_hour"),
        )
        .with_columns(
            pl.col("net_per_hour").rolling_mean(window_size=win).alias("rolling_net_per_hour"),
        )
    )

    output_cols = ["start_time", "net_score", "net_per_hour", "rolling_net_per_hour"]
    if "match_id" in result.columns:
        output_cols = ["match_id"] + output_cols
    if "outcome" in result.columns:
        output_cols.append("outcome")

    return result.select(output_cols)
