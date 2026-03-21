"""Score de performance d'une escouade (0-100).

Calcule un score collectif à partir des scores individuels de session v2,
avec des bonus pour le win rate d'équipe, la cohésion K/D et l'équilibre des frags.
"""

from __future__ import annotations

import logging

import polars as pl

from src.analysis.performance_config import resolve_squad_grade
from src.utils.safe_types import clamp as _clamp

logger = logging.getLogger(__name__)


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
        logger.debug("compute_squad_performance_score: no valid scores → returning None")
        return {"score": None, "grade": "N/A", "components": {}}

    base = sum(s["score"] for s in valid) / len(valid)
    bonuses, comps = _compute_squad_bonuses(valid)
    final = _clamp(base + bonuses, lo=0.0, hi=100.0)
    logger.debug(
        "Squad score: base=%.1f bonuses=%.1f final=%.1f grade=%s",
        base,
        bonuses,
        final,
        comps,
    )
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


def _build_base_cols(df: pl.DataFrame) -> list:
    """Sélectionne les colonnes contextuelles disponibles dans df."""
    _CASTS: dict[str, object] = {
        "team_mmr": pl.Float64,
        "session_label": pl.Utf8,
    }
    cols: list = [pl.col("match_id").cast(pl.Utf8)]
    for name in ("start_time", "session_id", "session_label", "team_mmr"):
        if name in df.columns:
            dtype = _CASTS.get(name)
            cols.append(pl.col(name).cast(dtype, strict=False) if dtype else pl.col(name))
    return cols


def _join_perf_frames(dfs: list[pl.DataFrame]) -> pl.DataFrame:
    """Joint les DataFrames des joueurs sur match_id et calcule squad_perf moyen."""
    joined = dfs[0].select(
        *_build_base_cols(dfs[0]),
        pl.col("performance_score").cast(pl.Float64, strict=False).alias("perf_0"),
    )
    joined = joined.filter(pl.col("perf_0").is_not_null())
    for i, df in enumerate(dfs[1:], 1):
        part = df.select(
            pl.col("match_id").cast(pl.Utf8),
            pl.col("performance_score").cast(pl.Float64, strict=False).alias(f"perf_{i}"),
        ).filter(pl.col(f"perf_{i}").is_not_null())
        joined = joined.join(part, on="match_id", how="inner")
    if joined.is_empty():
        return pl.DataFrame()
    if "team_mmr" not in joined.columns:
        joined = joined.with_columns(pl.lit(None).cast(pl.Float64).alias("team_mmr"))
    perf_cols = [f"perf_{i}" for i in range(len(dfs))]
    return joined.with_columns(
        pl.concat_list([pl.col(c) for c in perf_cols]).list.mean().alias("squad_perf")
    )


def _group_by_session(joined: pl.DataFrame) -> pl.DataFrame | None:
    """Regroupe par session_id (1 barre par session). Retourne None si pas de session_id."""
    if "session_id" not in joined.columns or joined["session_id"].drop_nulls().len() == 0:
        return None
    has_label = "session_label" in joined.columns
    # Extraire JJ/MM (5 premiers caractères du format "DD/MM/YYYY HH:MM–HH:MM (n)")
    label_expr = (
        pl.col("session_label").first().str.slice(0, 5)
        if has_label
        else pl.col("session_id").first().cast(pl.Utf8)
    ).alias("bucket_label")
    return (
        joined.group_by("session_id")
        .agg(
            [
                pl.col("squad_perf").mean(),
                pl.col("team_mmr").mean().alias("team_mmr_avg"),
                pl.len().alias("match_count"),
                pl.col("session_id").cast(pl.Int64, strict=False).min().alias("_sort"),
                label_expr,
            ]
        )
        .sort("_sort")
        .select("bucket_label", "squad_perf", "team_mmr_avg", "match_count")
    )


def _group_by_time_period(joined: pl.DataFrame, max_buckets: int) -> pl.DataFrame:
    """Regroupe par période temporelle (fallback). Nécessite start_time."""
    for trunc in ("1d", "1w", "1mo"):
        grouped = (
            joined.with_columns(pl.col("start_time").dt.truncate(trunc).alias("bucket"))
            .group_by("bucket")
            .agg(
                pl.col("squad_perf").mean(),
                pl.col("team_mmr").mean().alias("team_mmr_avg"),
                pl.len().alias("match_count"),
            )
            .sort("bucket")
        )
        if len(grouped) <= max_buckets:
            fmt = "%d/%m" if trunc in ("1d", "1w") else "%m/%Y"
            return grouped.with_columns(
                pl.col("bucket").dt.strftime(fmt).alias("bucket_label")
            ).select("bucket_label", "squad_perf", "team_mmr_avg", "match_count")
    return grouped.with_columns(pl.col("bucket").dt.strftime("%m/%Y").alias("bucket_label")).select(
        "bucket_label", "squad_perf", "team_mmr_avg", "match_count"
    )


def compute_squad_timeseries(
    series: list[tuple[str, pl.DataFrame]],
    *,
    max_buckets: int = 20,
) -> pl.DataFrame:
    """Agrège la performance d'escouade par session ou période de temps.

    Args:
        series: Liste de (nom, DataFrame). Premier élément = joueur principal.
                Tous doivent avoir match_id + performance_score.
        max_buckets: Nombre max de buckets pour le fallback temporel.

    Returns:
        DataFrame avec: bucket_label, squad_perf, team_mmr_avg, match_count.
    """
    dfs = [df for _, df in series if not df.is_empty() and "performance_score" in df.columns]
    if len(dfs) < 2:
        return pl.DataFrame()
    joined = _join_perf_frames(dfs)
    if joined.is_empty():
        return pl.DataFrame()
    by_session = _group_by_session(joined)
    if by_session is not None:
        return by_session
    if "start_time" not in joined.columns:
        return pl.DataFrame()
    return _group_by_time_period(joined.sort("start_time"), max_buckets)


__all__ = ["compute_squad_performance_score", "compute_squad_timeseries"]
