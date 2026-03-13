"""Helpers internes pour le calcul du score de performance relatif.

Regroupe les utilitaires de percentile, la préparation de l'historique
et les fonctions annexes utilisées par _performance_relative.py.
"""

from __future__ import annotations

import logging
from typing import Any

import polars as pl

from src.analysis.performance_config import RELATIVE_WEIGHTS
from src.data.domain.refdata import Outcome
from src.utils.safe_types import clamp as _clamp

_log = logging.getLogger(__name__)

# =============================================================================
# Utilitaires de base
# =============================================================================


def _normalize_df(df: pl.DataFrame | Any) -> pl.DataFrame:
    """Valide que le DataFrame est Polars (conversion Pandas supprimée v5.7)."""
    if isinstance(df, pl.DataFrame):
        return df
    _log.warning("Conversion Pandas→Polars inattendue dans _normalize_df")
    return pl.from_pandas(df)  # garde de sécurité pour appelants externes


def _percentile_rank(value: float, series: pl.Series) -> float:
    """Calcule le percentile d'une valeur dans une série (0-100)."""
    if series.is_empty() or len(series) < 2:
        return 50.0
    below_or_equal = (series <= value).sum()
    percentile = (below_or_equal / len(series)) * 100.0
    return _clamp(percentile, 0.0, 100.0)


def _percentile_rank_inverse(value: float, series: pl.Series) -> float:
    """Percentile inversé (pour les morts: moins = mieux)."""
    if series.is_empty() or len(series) < 2:
        return 50.0
    above_or_equal = (series >= value).sum()
    percentile = (above_or_equal / len(series)) * 100.0
    return _clamp(percentile, 0.0, 100.0)


def _safe_col(df: pl.DataFrame, col: str, default: float = 0.0) -> pl.Expr:
    """Retourne une expression pour une colonne, ou un literal si absente."""
    if col in df.columns:
        return pl.col(col).cast(pl.Float64, strict=False).fill_null(default)
    return pl.lit(default)


def _add_percentile(  # noqa: PLR0913
    key: str,
    value: float | None,
    history_metrics: pl.DataFrame,
    percentiles: dict[str, float],
    weights_used: dict[str, float],
    *,
    inverse: bool = False,
) -> None:
    """Ajoute un percentile à la collection si données valides."""
    if value is None:
        return
    if key not in history_metrics.columns:
        return
    series = history_metrics.get_column(key).drop_nulls()
    if series.is_empty():
        return
    rank_fn = _percentile_rank_inverse if inverse else _percentile_rank
    percentiles[key] = rank_fn(value, series)
    weights_used[key] = RELATIVE_WEIGHTS[key]


# =============================================================================
# Préparation de l'historique
# =============================================================================


_DEFAULT_DURATION = 600.0

_DURATION_CANDIDATES = ("time_played_seconds", "duration_seconds", "match_duration_seconds")


def _resolve_duration_column(df: pl.DataFrame) -> pl.DataFrame:
    """Ajoute la colonne _duration normalisée (secondes, toujours > 0)."""
    duration_col = next((c for c in _DURATION_CANDIDATES if c in df.columns), None)
    if duration_col is None:
        return df.with_columns(pl.lit(_DEFAULT_DURATION).alias("_duration"))
    return df.with_columns(
        pl.when(pl.col(duration_col).cast(pl.Float64, strict=False).fill_null(0.0) <= 0)
        .then(_DEFAULT_DURATION)
        .otherwise(pl.col(duration_col).cast(pl.Float64, strict=False).fill_null(_DEFAULT_DURATION))
        .alias("_duration")
    )


def _add_rank_perf_diff_column(df: pl.DataFrame) -> pl.DataFrame:
    """Ajoute la colonne rank_perf_diff au DataFrame."""
    has_rank = "rank" in df.columns
    has_mmrs = "team_mmr" in df.columns and "enemy_mmr" in df.columns
    if has_rank and has_mmrs:
        delta_mmr = _safe_col(df, "team_mmr") - _safe_col(df, "enemy_mmr")
        expected_rank = pl.lit(4.5) - (delta_mmr / pl.lit(100.0)) * pl.lit(0.5)
        actual_rank = _safe_col(df, "rank")
        return df.with_columns((expected_rank - actual_rank).alias("rank_perf_diff"))
    return df.with_columns(pl.lit(None).cast(pl.Float64).alias("rank_perf_diff"))


def _add_vs_expected_column(
    df: pl.DataFrame,
    actual_col: str,
    expected_col: str,
    output_col: str,
    *,
    inverse: bool = False,
) -> pl.DataFrame:
    """Ajoute une colonne ratio actual/expected (ou expected/actual si inverse)."""
    if expected_col not in df.columns:
        return df.with_columns(pl.lit(None).cast(pl.Float64).alias(output_col))

    actual = _safe_col(df, actual_col)
    expected = pl.col(expected_col).cast(pl.Float64, strict=False)

    if inverse:
        safe_actual = pl.when(actual < 1.0).then(1.0).otherwise(actual)
        expr = (
            pl.when(expected.is_not_null() & (expected > 0.0))
            .then(expected / safe_actual)
            .otherwise(pl.lit(None))
        )
    else:
        expr = (
            pl.when(expected.is_not_null() & (expected > 0.0))
            .then(actual / expected)
            .otherwise(pl.lit(None))
        )

    return df.with_columns(expr.alias(output_col))


def _prepare_history_metrics(df_history: pl.DataFrame) -> pl.DataFrame:
    """Prépare les métriques normalisées par minute pour l'historique."""
    output_cols = [
        "kpm",
        "dpm_deaths",
        "apm",
        "kda",
        "accuracy",
        "pspm",
        "dpm_damage",
        "rank_perf_diff",
        "kills_vs_expected",
        "deaths_vs_expected",
    ]

    if df_history.is_empty():
        return pl.DataFrame(schema=dict.fromkeys(output_cols, pl.Float64))

    df = _resolve_duration_column(df_history)
    minutes_expr = pl.col("_duration") / 60.0
    df = df.with_columns(
        [
            (_safe_col(df, "kills") / minutes_expr).alias("kpm"),
            (_safe_col(df, "deaths") / minutes_expr).alias("dpm_deaths"),
            (_safe_col(df, "assists") / minutes_expr).alias("apm"),
        ]
    )

    if "kda" in df.columns:
        df = df.with_columns(pl.col("kda").cast(pl.Float64, strict=False).alias("kda"))
    else:
        k = _safe_col(df, "kills")
        d = pl.when(_safe_col(df, "deaths") < 1.0).then(1.0).otherwise(_safe_col(df, "deaths"))
        a = _safe_col(df, "assists")
        df = df.with_columns(((k + a) / d).alias("kda"))

    if "accuracy" in df.columns:
        df = df.with_columns(pl.col("accuracy").cast(pl.Float64, strict=False).alias("accuracy"))
    else:
        df = df.with_columns(pl.lit(None).cast(pl.Float64).alias("accuracy"))

    if "personal_score" in df.columns:
        df = df.with_columns((_safe_col(df, "personal_score") / minutes_expr).alias("pspm"))
    else:
        df = df.with_columns(pl.lit(None).cast(pl.Float64).alias("pspm"))

    if "damage_dealt" in df.columns:
        df = df.with_columns((_safe_col(df, "damage_dealt") / minutes_expr).alias("dpm_damage"))
    else:
        df = df.with_columns(pl.lit(None).cast(pl.Float64).alias("dpm_damage"))

    df = _add_rank_perf_diff_column(df)
    df = _add_vs_expected_column(df, "kills", "kills_expected", "kills_vs_expected", inverse=False)
    df = _add_vs_expected_column(
        df, "deaths", "deaths_expected", "deaths_vs_expected", inverse=True
    )

    return df.select(output_cols)


# =============================================================================
# Fonctions de score annexes
# =============================================================================


def _compute_rank_performance(
    rank: int | float,
    team_mmr: float,
    enemy_mmr: float,
    history_metrics: pl.DataFrame,
) -> float | None:
    """Calcule le percentile de performance du rang contextualisé par MMR."""
    if rank is None or team_mmr is None or enemy_mmr is None:
        return None
    delta_mmr = float(team_mmr) - float(enemy_mmr)
    expected_rank = 4.5 - (delta_mmr / 100.0) * 0.5
    rank_diff = expected_rank - float(rank)

    if "rank_perf_diff" not in history_metrics.columns:
        return None
    rank_diff_series = history_metrics.get_column("rank_perf_diff").drop_nulls()
    if rank_diff_series.is_empty():
        return None
    return _percentile_rank(rank_diff, rank_diff_series)


def _apply_bot_bonus(score: float, row: dict[str, Any]) -> float:
    """Applique le bonus/indulgence pour coéquipier bot."""
    _outcome_val = row.get("outcome")
    _is_win = _outcome_val is not None and float(_outcome_val) == Outcome.WIN

    _team_mmr = row.get("team_mmr")
    _enemy_mmr = row.get("enemy_mmr")
    _mmr_gap_factor = 0.0
    if _team_mmr and _enemy_mmr and float(_team_mmr) > 0:
        _gap_pct = (float(_enemy_mmr) - float(_team_mmr)) / float(_team_mmr)
        _mmr_gap_factor = max(-1.0, min(1.0, _gap_pct / 0.30))

    _bonus = max(0.5, 3.0 + _mmr_gap_factor * 2.0) if _is_win else 5.0 + _mmr_gap_factor * 2.0
    return min(100.0, score + _bonus)


def _fallback_kda_percentile(df_pl: pl.DataFrame) -> pl.Series:
    """Score simple basé sur le percentile KDA (historique trop court)."""
    if len(df_pl) == 0:
        return pl.Series("performance", [], dtype=pl.Float64)

    kills = (
        df_pl.get_column("kills").cast(pl.Float64, strict=False)
        if "kills" in df_pl.columns
        else pl.Series([0.0] * len(df_pl), dtype=pl.Float64)
    ).fill_null(0.0)
    deaths = (
        df_pl.get_column("deaths").cast(pl.Float64, strict=False)
        if "deaths" in df_pl.columns
        else pl.Series([0.0] * len(df_pl), dtype=pl.Float64)
    ).fill_null(0.0)
    assists = (
        df_pl.get_column("assists").cast(pl.Float64, strict=False)
        if "assists" in df_pl.columns
        else pl.Series([0.0] * len(df_pl), dtype=pl.Float64)
    ).fill_null(0.0)

    deaths_safe = deaths.clip(lower_bound=1.0)
    derived_kda = (kills + assists) / deaths_safe

    if "kda" in df_pl.columns:
        kda_series = df_pl.get_column("kda").cast(pl.Float64, strict=False)
        kda_series = kda_series.fill_null(derived_kda)
    else:
        kda_series = derived_kda

    kda_series = kda_series.fill_null(0.0)
    n = len(kda_series)
    if n <= 1:
        return pl.Series("performance", [50.0] * len(df_pl), dtype=pl.Float64)

    ranks = kda_series.rank(method="average")
    return ((ranks - 1.0) / float(n - 1) * 100.0).alias("performance")
