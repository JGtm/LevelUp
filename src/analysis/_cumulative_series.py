"""Séries cumulatives Polars (K/D, net score, KDA, objectifs, rolling K/D)."""

from __future__ import annotations

import polars as pl

from src.config import CORE_STAT_COLUMNS


def compute_cumulative_net_score_series_polars(
    match_stats_df: pl.DataFrame,
) -> pl.DataFrame:
    """Calcule la série cumulative du net score avec Polars.

    Le net score est défini comme : kills - deaths.

    Args:
        match_stats_df: DataFrame Polars avec colonnes start_time, kills, deaths.

    Returns:
        DataFrame avec colonnes: match_id, start_time, net_score, cumulative_net_score.

    Raises:
        ValueError: Si Polars n'est pas disponible.

    Example:
        >>> df = repo.query_df("SELECT * FROM match_stats ORDER BY start_time")
        >>> result = compute_cumulative_net_score_series_polars(df)
        >>> print(result.head())
    """
    if match_stats_df.is_empty():
        return pl.DataFrame(
            schema={
                "match_id": pl.Utf8,
                "start_time": pl.Utf8,
                "net_score": pl.Int64,
                "cumulative_net_score": pl.Int64,
            }
        )

    # S'assurer que les colonnes requises existent
    required_cols = set(CORE_STAT_COLUMNS)
    available_cols = set(match_stats_df.columns)
    if not required_cols.issubset(available_cols):
        missing = required_cols - available_cols
        msg = f"Colonnes manquantes: {missing}"
        raise ValueError(msg)

    # Calculer net score et cumul
    result = (
        match_stats_df.sort("start_time")
        .with_columns(
            [
                # Net score = kills - deaths
                (pl.col("kills").fill_null(0) - pl.col("deaths").fill_null(0)).alias("net_score"),
            ]
        )
        .with_columns(
            [
                # Cumul du net score
                pl.col("net_score").cum_sum().alias("cumulative_net_score"),
            ]
        )
    )

    # Sélectionner les colonnes de sortie
    output_cols = ["start_time", "net_score", "cumulative_net_score"]
    if "match_id" in result.columns:
        output_cols = ["match_id"] + output_cols

    return result.select(output_cols)


def compute_cumulative_kd_series_polars(
    match_stats_df: pl.DataFrame,
) -> pl.DataFrame:
    """Calcule la série cumulative du K/D avec Polars.

    Le K/D cumulé est calculé comme: sum(kills) / max(1, sum(deaths)).

    Args:
        match_stats_df: DataFrame Polars avec colonnes start_time, kills, deaths.

    Returns:
        DataFrame avec colonnes: match_id, start_time, kd, cumulative_kills,
        cumulative_deaths, cumulative_kd.
    """
    if match_stats_df.is_empty():
        return pl.DataFrame(
            schema={
                "match_id": pl.Utf8,
                "start_time": pl.Utf8,
                "kd": pl.Float64,
                "cumulative_kills": pl.Int64,
                "cumulative_deaths": pl.Int64,
                "cumulative_kd": pl.Float64,
            }
        )

    result = (
        match_stats_df.sort("start_time")
        .with_columns(
            [
                # K/D du match
                (
                    pl.col("kills").fill_null(0)
                    / pl.when(pl.col("deaths").fill_null(0) == 0)
                    .then(1)
                    .otherwise(pl.col("deaths").fill_null(0))
                ).alias("kd"),
                # Cumuls
                pl.col("kills").fill_null(0).cum_sum().alias("cumulative_kills"),
                pl.col("deaths").fill_null(0).cum_sum().alias("cumulative_deaths"),
            ]
        )
        .with_columns(
            [
                # K/D cumulé
                (
                    pl.col("cumulative_kills")
                    / pl.when(pl.col("cumulative_deaths") == 0)
                    .then(1)
                    .otherwise(pl.col("cumulative_deaths"))
                ).alias("cumulative_kd"),
            ]
        )
    )

    # Sélectionner les colonnes de sortie
    output_cols = [
        "start_time",
        "kd",
        "cumulative_kills",
        "cumulative_deaths",
        "cumulative_kd",
    ]
    if "match_id" in result.columns:
        output_cols = ["match_id"] + output_cols

    return result.select(output_cols)


def compute_cumulative_kda_series_polars(
    match_stats_df: pl.DataFrame,
) -> pl.DataFrame:
    """Calcule la série cumulative du KDA avec Polars.

    Le KDA cumulé est calculé comme: (sum(kills) + sum(assists)) / max(1, sum(deaths)).

    Args:
        match_stats_df: DataFrame Polars avec colonnes start_time, kills, deaths, assists.

    Returns:
        DataFrame avec colonnes: match_id, start_time, kda, cumulative_kda.
    """
    if match_stats_df.is_empty():
        return pl.DataFrame(
            schema={
                "match_id": pl.Utf8,
                "start_time": pl.Utf8,
                "kda": pl.Float64,
                "cumulative_kda": pl.Float64,
            }
        )

    result = (
        match_stats_df.sort("start_time")
        .with_columns(
            [
                # KDA du match: (K + A) / max(1, D)
                (
                    (pl.col("kills").fill_null(0) + pl.col("assists").fill_null(0))
                    / pl.when(pl.col("deaths").fill_null(0) == 0)
                    .then(1)
                    .otherwise(pl.col("deaths").fill_null(0))
                ).alias("kda"),
                # Cumuls
                pl.col("kills").fill_null(0).cum_sum().alias("_cum_kills"),
                pl.col("deaths").fill_null(0).cum_sum().alias("_cum_deaths"),
                pl.col("assists").fill_null(0).cum_sum().alias("_cum_assists"),
            ]
        )
        .with_columns(
            [
                # KDA cumulé
                (
                    (pl.col("_cum_kills") + pl.col("_cum_assists"))
                    / pl.when(pl.col("_cum_deaths") == 0).then(1).otherwise(pl.col("_cum_deaths"))
                ).alias("cumulative_kda"),
            ]
        )
    )

    # Sélectionner les colonnes de sortie
    output_cols = ["start_time", "kda", "cumulative_kda"]
    if "match_id" in result.columns:
        output_cols = ["match_id"] + output_cols

    return result.select(output_cols)


def compute_cumulative_objective_score_series_polars(
    awards_df: pl.DataFrame,
    match_stats_df: pl.DataFrame,
) -> pl.DataFrame:
    """Calcule la série cumulative du score objectifs avec Polars.

    Utilise les personal_score_awards pour calculer le score objectifs.

    Args:
        awards_df: DataFrame des personal_score_awards.
        match_stats_df: DataFrame des match_stats (pour start_time).

    Returns:
        DataFrame avec colonnes: match_id, start_time, objective_score, cumulative_objective.
    """
    if awards_df.is_empty() or match_stats_df.is_empty():
        return pl.DataFrame(
            schema={
                "match_id": pl.Utf8,
                "start_time": pl.Utf8,
                "objective_score": pl.Float64,
                "cumulative_objective": pl.Float64,
            }
        )

    # Catégories d'objectifs (depuis refdata)
    objective_categories = ["objective", "mode"]

    # Calculer score objectifs par match
    objective_by_match = (
        awards_df.filter(pl.col("score_category").is_in(objective_categories))
        .group_by("match_id")
        .agg(pl.col("points").sum().alias("objective_score"))
    )

    # Joindre avec match_stats pour avoir start_time
    result = (
        match_stats_df.select(["match_id", "start_time"])
        .join(objective_by_match, on="match_id", how="left")
        .with_columns([pl.col("objective_score").fill_null(0)])
        .sort("start_time")
        .with_columns([pl.col("objective_score").cum_sum().alias("cumulative_objective")])
    )

    return result.select(["match_id", "start_time", "objective_score", "cumulative_objective"])


def compute_rolling_kd_polars(
    match_stats_df: pl.DataFrame,
    window_size: int = 5,
) -> pl.DataFrame:
    """Calcule le K/D glissant sur une fenêtre de matchs.

    Args:
        match_stats_df: DataFrame Polars des matchs.
        window_size: Taille de la fenêtre glissante.

    Returns:
        DataFrame avec colonnes: match_id, start_time, kd, rolling_kd.
    """
    if match_stats_df.is_empty():
        return pl.DataFrame(
            schema={
                "match_id": pl.Utf8,
                "start_time": pl.Utf8,
                "kd": pl.Float64,
                "rolling_kd": pl.Float64,
            }
        )

    result = (
        match_stats_df.sort("start_time")
        .with_columns(
            [
                # K/D du match
                (
                    pl.col("kills").fill_null(0)
                    / pl.when(pl.col("deaths").fill_null(0) == 0)
                    .then(1)
                    .otherwise(pl.col("deaths").fill_null(0))
                ).alias("kd"),
                # Rolling sum
                pl.col("kills")
                .fill_null(0)
                .rolling_sum(window_size=window_size)
                .alias("_rolling_kills"),
                pl.col("deaths")
                .fill_null(0)
                .rolling_sum(window_size=window_size)
                .alias("_rolling_deaths"),
            ]
        )
        .with_columns(
            [
                # K/D glissant
                (
                    pl.col("_rolling_kills")
                    / pl.when(pl.col("_rolling_deaths") == 0)
                    .then(1)
                    .otherwise(pl.col("_rolling_deaths"))
                ).alias("rolling_kd"),
            ]
        )
    )

    # Sélectionner les colonnes de sortie
    output_cols = ["start_time", "kd", "rolling_kd"]
    if "match_id" in result.columns:
        output_cols = ["match_id"] + output_cols

    return result.select(output_cols)
