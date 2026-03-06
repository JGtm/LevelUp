"""Résumés et fréquences des objectifs par match.

Fonctions d'agrégation pour la visualisation des tendances objectives
et la distribution des types d'awards.
"""

from __future__ import annotations

import polars as pl

from src.analysis._objective_helpers import (
    CATEGORY_IDS,
    empty_frequency_df,
    empty_summary_df,
    filter_awards,
)
from src.data.domain.refdata import (
    ASSIST_SCORES,
    OBJECTIVE_SCORES,
    PERSONAL_SCORE_DISPLAY_NAMES,
    PERSONAL_SCORE_POINTS,
    get_personal_score_display_name,
)


def _pivot_awards_by_category(filtered_df: pl.DataFrame) -> pl.DataFrame:
    """Pivote les awards filtrés par catégorie (objective/assist/kill/other)."""
    objective_ids = list(CATEGORY_IDS["objective"])
    assist_ids = list(CATEGORY_IDS["assist"])
    kill_ids = list(CATEGORY_IDS["kill"])

    categorized = filtered_df.with_columns(
        pl.when(pl.col("award_name_id").is_in(objective_ids))
        .then(pl.lit("objective"))
        .when(pl.col("award_name_id").is_in(assist_ids))
        .then(pl.lit("assist"))
        .when(pl.col("award_name_id").is_in(kill_ids))
        .then(pl.lit("kill"))
        .otherwise(pl.lit("other"))
        .alias("category")
    )

    summary = (
        categorized.group_by(["match_id", "category"])
        .agg(pl.col("total_points").sum().alias("points"))
        .pivot(
            values="points",
            index="match_id",
            on="category",
            aggregate_function="sum",
        )
        .fill_null(0)
    )

    for col in ["objective", "assist", "kill", "other"]:
        if col not in summary.columns:
            summary = summary.with_columns(pl.lit(0).alias(col))

    return summary


def compute_objective_summary_by_match_polars(
    awards_df: pl.DataFrame,
    *,
    xuid: str | None = None,
) -> pl.DataFrame:
    """Calcule un résumé des objectifs par match avec Polars.

    Args:
        awards_df: DataFrame Polars avec les awards.
        xuid: Filtrer pour un joueur spécifique (optionnel).

    Returns:
        DataFrame Polars avec match_id, objective_score, assist_score,
        kill_score, total_score, objective_ratio.
    """
    if awards_df.is_empty():
        return empty_summary_df()

    filtered_df = filter_awards(awards_df, xuid=xuid)
    if filtered_df.is_empty():
        return empty_summary_df()

    summary = _pivot_awards_by_category(filtered_df)

    return (
        summary.rename(
            {
                "objective": "objective_score",
                "assist": "assist_score",
                "kill": "kill_score",
            }
        )
        .with_columns(
            (pl.col("objective_score") + pl.col("assist_score") + pl.col("kill_score")).alias(
                "total_score"
            )
        )
        .with_columns(
            (pl.col("objective_score") / pl.col("total_score"))
            .fill_nan(0)
            .fill_null(0)
            .alias("objective_ratio")
        )
        .select(
            "match_id",
            "objective_score",
            "assist_score",
            "kill_score",
            "total_score",
            "objective_ratio",
        )
    )


def compute_award_frequency_polars(
    awards_df: pl.DataFrame,
    *,
    category: str | None = None,
    top_n: int = 20,
) -> pl.DataFrame:
    """Calcule la fréquence des awards par type.

    Args:
        awards_df: DataFrame Polars avec les awards.
        category: Filtrer par catégorie ("objective", "assist", "kill", ou None).
        top_n: Nombre de types à retourner.

    Returns:
        DataFrame Polars avec award_name_id, display_name, count, total_points.
    """
    if awards_df.is_empty():
        return empty_frequency_df()

    filtered_df = awards_df

    # Filtrer par catégorie si spécifié
    if category and category in CATEGORY_IDS:
        filtered_df = filtered_df.filter(
            pl.col("award_name_id").is_in(list(CATEGORY_IDS[category]))
        )

    if filtered_df.is_empty():
        return empty_frequency_df()

    # Agréger par award_name_id
    aggregated = (
        filtered_df.group_by("award_name_id")
        .agg(
            [
                pl.col("count").sum().alias("count"),
                pl.col("total_points").sum().alias("total_points"),
            ]
        )
        .sort("total_points", descending=True)
        .head(top_n)
    )

    # Ajouter les noms d'affichage
    award_ids = aggregated["award_name_id"].to_list()
    display_names = [get_personal_score_display_name(aid) for aid in award_ids]

    return aggregated.with_columns(pl.Series("display_name", display_names)).select(
        ["award_name_id", "display_name", "count", "total_points"]
    )


# =============================================================================
# Fonctions utilitaires
# =============================================================================


def get_objective_mode_awards() -> dict[int, str]:
    """Retourne les awards liés aux modes à objectifs avec leurs noms."""
    return {
        int(score_id): PERSONAL_SCORE_DISPLAY_NAMES.get(int(score_id), "Inconnu")
        for score_id in OBJECTIVE_SCORES
    }


def get_assist_awards_with_points() -> dict[int, tuple[str, int]]:
    """Retourne les awards d'assistance avec leurs noms et points."""
    return {
        int(score_id): (
            PERSONAL_SCORE_DISPLAY_NAMES.get(int(score_id), "Inconnu"),
            PERSONAL_SCORE_POINTS.get(int(score_id), 0),
        )
        for score_id in ASSIST_SCORES
    }


def is_objective_mode_match(game_variant_category: int) -> bool:
    """Vérifie si un match est un mode à objectifs basé sur la catégorie."""
    from src.data.domain.refdata import OBJECTIVE_MODE_CATEGORIES

    return game_variant_category in OBJECTIVE_MODE_CATEGORIES
