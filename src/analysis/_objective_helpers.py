"""Helpers DRY pour l'analyse de participation aux objectifs.

Factorise les patterns répétés dans les fonctions d'analyse :
- Filtrage par match_id / xuid
- Calcul de scores par catégorie (objectif, assist, kill, négatif)
- Création de DataFrames vides avec schéma prédéfini
"""

from __future__ import annotations

import polars as pl

from src.data.domain.refdata import (
    ASSIST_SCORES,
    KILL_SCORES,
    NEGATIVE_SCORES,
    OBJECTIVE_SCORES,
)


def filter_awards(
    df: pl.DataFrame,
    *,
    match_id: str | None = None,
    xuid: str | None = None,
) -> pl.DataFrame:
    """Filtre un DataFrame d'awards par match_id et/ou xuid."""
    if match_id:
        df = df.filter(pl.col("match_id") == match_id)
    if xuid and "xuid" in df.columns:
        df = df.filter(pl.col("xuid") == xuid)
    return df


# Mapping catégorie → set d'IDs
CATEGORY_IDS: dict[str, set] = {
    "objective": OBJECTIVE_SCORES,
    "assist": ASSIST_SCORES,
    "kill": KILL_SCORES,
    "negative": NEGATIVE_SCORES,
}


def compute_category_scores(
    df: pl.DataFrame,
) -> dict[str, int]:
    """Calcule les scores par catégorie (objectif, assist, kill, négatif).

    Args:
        df: DataFrame filtré avec colonnes award_name_id, total_points, count.

    Returns:
        Dict avec clés objective_score, assist_score, kill_score, negative_score,
        objective_count, assist_count, kill_count, total_score.
    """
    result: dict[str, int] = {}
    total = 0

    for cat_name, cat_ids in CATEGORY_IDS.items():
        ids_list = list(cat_ids)
        cat_df = df.filter(pl.col("award_name_id").is_in(ids_list))

        score = cat_df.select(pl.col("total_points").sum()).item() or 0
        count = cat_df.select(pl.col("count").sum()).item() or 0

        result[f"{cat_name}_score"] = int(score)
        result[f"{cat_name}_count"] = int(count)
        total += score

    result["total_score"] = int(total)
    return result


def empty_summary_df() -> pl.DataFrame:
    """Retourne un DataFrame vide avec le schéma de résumé par match."""
    return pl.DataFrame(
        {
            "match_id": [],
            "objective_score": [],
            "assist_score": [],
            "kill_score": [],
            "total_score": [],
            "objective_ratio": [],
        }
    )


def empty_frequency_df() -> pl.DataFrame:
    """Retourne un DataFrame vide avec le schéma de fréquence d'awards."""
    return pl.DataFrame(
        {
            "award_name_id": [],
            "display_name": [],
            "count": [],
            "total_points": [],
        }
    )
