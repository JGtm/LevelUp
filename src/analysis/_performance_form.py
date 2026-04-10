"""Score de forme — historique rolling_mean(14) vs rolling_mean(90).

Calcul pur sans accès DB : entrée DataFrame, sortie DataFrame enrichi.
"""

from __future__ import annotations

import logging

import polars as pl

logger = logging.getLogger(__name__)

_WINDOW_SHORT = 14
_WINDOW_LONG = 90


def compute_form_score_history(df: pl.DataFrame) -> pl.DataFrame:
    """Calcule le score de forme match par match sur l'historique complet.

    form_score = avg_perf(14 derniers matchs) - avg_perf(90 derniers matchs)

    Positif → en forme (récents au-dessus de l'habituel).
    Négatif → creux de forme.

    Args:
        df: DataFrame avec colonnes start_time et performance_score.

    Returns:
        DataFrame trié par start_time avec avg_14, avg_90, form_score ajoutés.
        Retourné tel quel si colonnes manquantes ou vide.
    """
    if df.is_empty():
        logger.debug("compute_form_score_history: DataFrame vide, retour immédiat.")
        return df
    missing = {"performance_score", "start_time"} - set(df.columns)
    if missing:
        logger.debug(
            "compute_form_score_history: colonnes manquantes %s, retour immédiat.", missing
        )
        return df

    logger.debug("compute_form_score_history: calcul sur %d matchs.", len(df))
    result = (
        df.sort("start_time")
        .with_columns(
            pl.col("performance_score")
            .rolling_mean(window_size=_WINDOW_SHORT, min_samples=1)
            .alias("avg_14"),
            pl.col("performance_score")
            .rolling_mean(window_size=_WINDOW_LONG, min_samples=1)
            .alias("avg_90"),
        )
        .with_columns((pl.col("avg_14") - pl.col("avg_90")).alias("form_score"))
    )
    logger.debug(
        "compute_form_score_history: form_score range [%.2f, %.2f].",
        result["form_score"].min() or 0.0,
        result["form_score"].max() or 0.0,
    )
    return result
