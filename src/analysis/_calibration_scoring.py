"""Métriques de scoring pour la calibration LUSR (MAE et corrélation de Pearson)."""

from __future__ import annotations

import polars as pl


def _normalize(series: pl.Series) -> pl.Series:
    """Normalise une série à [0, 1] par min-max."""
    lo = series.min()
    hi = series.max()
    if hi is None or lo is None or hi == lo:
        return pl.Series([0.5] * len(series))
    return (series - lo) / (hi - lo)


def _score_mae(
    df_ratings: pl.DataFrame,
    team_mmr_map: dict[str, float],
) -> float:
    """Calcule MAE normalisé entre LUSR et individual_mmr décorrélé (plus bas = meilleur)."""
    rows = [
        (mid, rv, team_mmr_map[mid])
        for mid, rv in zip(
            df_ratings["match_id"].to_list(),
            df_ratings["rating_value"].to_list(),
            strict=False,
        )
        if mid in team_mmr_map
    ]
    if len(rows) < 10:
        return 1.0

    r_series = pl.Series([r[1] for r in rows], dtype=pl.Float64)
    t_series = pl.Series([r[2] for r in rows], dtype=pl.Float64)

    r_norm = _normalize(r_series)
    t_norm = _normalize(t_series)
    return float((r_norm - t_norm).abs().mean())


def _score_corr(
    df_ratings: pl.DataFrame,
    team_mmr_map: dict[str, float],
) -> float:
    """Corrélation de Pearson entre LUSR et individual_mmr décorrélé (plus haut = meilleur)."""
    rows = [
        (mid, rv, team_mmr_map[mid])
        for mid, rv in zip(
            df_ratings["match_id"].to_list(),
            df_ratings["rating_value"].to_list(),
            strict=False,
        )
        if mid in team_mmr_map
    ]
    if len(rows) < 10:
        return -1.0

    r_series = pl.Series([r[1] for r in rows], dtype=pl.Float64)
    t_series = pl.Series([r[2] for r in rows], dtype=pl.Float64)

    corr = r_series.pearson_corr(t_series)
    return float(corr) if corr is not None else 0.0
