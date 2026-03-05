"""Score de performance RELATIF par match (0-100).

Compare la performance d'un match à l'historique personnel du joueur.
- 50 = match dans ta moyenne
- 100 = meilleur match de ton historique
- 0 = pire match de ton historique

Configuration centralisée dans : src/analysis/performance_config.py
"""

from __future__ import annotations

from typing import Any

import polars as pl

from src.analysis.performance_config import (
    MIN_MATCHES_FOR_RELATIVE,
    RELATIVE_WEIGHTS,
)
from src.data.domain.refdata import Outcome
from src.utils.safe_types import clamp as _clamp
from src.utils.safe_types import safe_float as _safe_float

# =============================================================================
# Utilitaires
# =============================================================================


def _normalize_df(df: pl.DataFrame | Any) -> pl.DataFrame:
    """Convertit un DataFrame Pandas en Polars si nécessaire."""
    if isinstance(df, pl.DataFrame):
        return df
    return pl.from_pandas(df)


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
    """Ajoute un percentile à la collection si données valides.

    DRY helper pour le pattern répété 10× dans compute_relative_performance_score.
    """
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


def _prepare_history_metrics(df_history: pl.DataFrame) -> pl.DataFrame:  # noqa: PLR0912
    """Prépare les métriques normalisées par minute pour l'historique.

    Returns:
        DataFrame avec colonnes: kpm, dpm_deaths, apm, kda, accuracy,
        pspm, dpm_damage, rank_perf_diff, kills_vs_expected, deaths_vs_expected.
    """
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

    # Durée du match en secondes
    duration_col = None
    for col in ["time_played_seconds", "duration_seconds", "match_duration_seconds"]:
        if col in df_history.columns:
            duration_col = col
            break

    if duration_col is None:
        df = df_history.with_columns(pl.lit(600.0).alias("_duration"))
    else:
        df = df_history.with_columns(
            pl.when(pl.col(duration_col).cast(pl.Float64, strict=False).fill_null(0.0) <= 0)
            .then(600.0)
            .otherwise(pl.col(duration_col).cast(pl.Float64, strict=False).fill_null(600.0))
            .alias("_duration")
        )

    minutes_expr = pl.col("_duration") / 60.0
    df = df.with_columns(
        [
            (_safe_col(df, "kills") / minutes_expr).alias("kpm"),
            (_safe_col(df, "deaths") / minutes_expr).alias("dpm_deaths"),
            (_safe_col(df, "assists") / minutes_expr).alias("apm"),
        ]
    )

    # KDA
    if "kda" in df.columns:
        df = df.with_columns(pl.col("kda").cast(pl.Float64, strict=False).alias("kda"))
    else:
        k = _safe_col(df, "kills")
        d = pl.when(_safe_col(df, "deaths") < 1.0).then(1.0).otherwise(_safe_col(df, "deaths"))
        a = _safe_col(df, "assists")
        df = df.with_columns(((k + a) / d).alias("kda"))

    # Accuracy
    if "accuracy" in df.columns:
        df = df.with_columns(pl.col("accuracy").cast(pl.Float64, strict=False).alias("accuracy"))
    else:
        df = df.with_columns(pl.lit(None).cast(pl.Float64).alias("accuracy"))

    # PSPM — Personal Score Per Minute (v4)
    if "personal_score" in df.columns:
        df = df.with_columns((_safe_col(df, "personal_score") / minutes_expr).alias("pspm"))
    else:
        df = df.with_columns(pl.lit(None).cast(pl.Float64).alias("pspm"))

    # DPM Damage (v4)
    if "damage_dealt" in df.columns:
        df = df.with_columns((_safe_col(df, "damage_dealt") / minutes_expr).alias("dpm_damage"))
    else:
        df = df.with_columns(pl.lit(None).cast(pl.Float64).alias("dpm_damage"))

    # Rank Performance Diff (v4)
    df = _add_rank_perf_diff_column(df)

    # Kills vs Expected (v5)
    df = _add_vs_expected_column(
        df,
        "kills",
        "kills_expected",
        "kills_vs_expected",
        inverse=False,
    )

    # Deaths vs Expected (v5) — inversé : expected/actual
    df = _add_vs_expected_column(
        df,
        "deaths",
        "deaths_expected",
        "deaths_vs_expected",
        inverse=True,
    )

    return df.select(output_cols)


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
            .then(
                expected / safe_actual,
            )
            .otherwise(pl.lit(None))
        )
    else:
        expr = (
            pl.when(expected.is_not_null() & (expected > 0.0))
            .then(
                actual / expected,
            )
            .otherwise(pl.lit(None))
        )

    return df.with_columns(expr.alias(output_col))


# =============================================================================
# Calcul du score relatif
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
        return None


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

    percentiles: dict[str, float] = {}
    weights_used: dict[str, float] = {}

    # Métriques standards (10 composantes)
    _add_percentile("kpm", metrics["kpm"], history_metrics, percentiles, weights_used)
    _add_percentile(
        "dpm_deaths",
        metrics["dpm_deaths"],
        history_metrics,
        percentiles,
        weights_used,
        inverse=True,
    )
    _add_percentile("apm", metrics["apm"], history_metrics, percentiles, weights_used)
    _add_percentile("kda", metrics["kda"], history_metrics, percentiles, weights_used)
    _add_percentile("accuracy", metrics["accuracy"], history_metrics, percentiles, weights_used)
    _add_percentile("pspm", metrics["pspm"], history_metrics, percentiles, weights_used)
    _add_percentile("dpm_damage", metrics["dpm_damage"], history_metrics, percentiles, weights_used)

    # Rank Performance (cas spécial : calcul via _compute_rank_performance)
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

    _add_percentile(
        "kills_vs_expected",
        metrics["kills_vs_expected"],
        history_metrics,
        percentiles,
        weights_used,
    )
    _add_percentile(
        "deaths_vs_expected",
        metrics["deaths_vs_expected"],
        history_metrics,
        percentiles,
        weights_used,
    )

    if not percentiles:
        return None

    total_weight = sum(weights_used.values())
    if total_weight <= 0:
        return None

    score = sum(percentiles[k] * weights_used[k] for k in percentiles) / total_weight

    # Bonus/indulgence bot coéquipier
    if had_bot_teammate:
        score = _apply_bot_bonus(score, row)

    return round(score, 1)


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


# =============================================================================
# Série de scores pour un DataFrame
# =============================================================================


def compute_performance_series(
    df: pl.DataFrame | Any,
    df_history: pl.DataFrame | Any | None = None,
) -> pl.Series | Any:
    """Calcule le score de performance pour chaque match d'un DataFrame.

    Args:
        df: DataFrame des matchs à évaluer.
        df_history: Historique complet pour le calcul relatif. Si None, utilise df.

    Returns:
        Series avec les scores de performance.
    """
    was_pandas = not isinstance(df, pl.DataFrame)
    df_pl = _normalize_df(df)
    history_pl = _normalize_df(df_history) if df_history is not None else None

    if df_pl.is_empty():
        result = pl.Series("performance", [], dtype=pl.Float64)
        return result.to_pandas() if was_pandas else result

    history = history_pl if history_pl is not None else df_pl

    if len(history) < MIN_MATCHES_FOR_RELATIVE:
        result = _fallback_kda_percentile(df_pl)
        return result.to_pandas() if was_pandas else result

    scores = [
        compute_relative_performance_score(row_dict, history)
        for row_dict in df_pl.iter_rows(named=True)
    ]

    result = pl.Series("performance", scores, dtype=pl.Float64)
    return result.to_pandas() if was_pandas else result


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
