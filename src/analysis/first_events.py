"""Calcul de tendance — premiers événements de match (premier frag / première mort).

Module pur : pas d'accès base de données.
Entrée : pl.DataFrame avec colonnes xuid, start_time, first_kill_s, first_death_s.
Sortie : mêmes données enrichies des colonnes de rolling average.
"""

from __future__ import annotations

import polars as pl

_ROLL_MIN_SAMPLES: int = 3
"""Nombre minimum de matchs requis avant d'émettre une rolling average."""


def compute_first_events_rolling(
    df: pl.DataFrame,
    window: int = 10,
) -> pl.DataFrame:
    """Ajoute les colonnes de rolling average des premiers événements par joueur.

    Pour chaque xuid distinct, trie les matchs par start_time et applique
    une moyenne glissante sur first_kill_s et first_death_s.

    Args:
        df: DataFrame avec colonnes xuid, start_time, first_kill_s, first_death_s.
        window: Taille de la fenêtre de lissage en nombre de matchs.

    Returns:
        DataFrame avec colonnes first_kill_roll et first_death_roll ajoutées.
        Les valeurs sont None si moins de _ROLL_MIN_SAMPLES matchs précèdent.
    """
    if df.is_empty():
        return df.with_columns(
            pl.lit(None, dtype=pl.Float64).alias("first_kill_roll"),
            pl.lit(None, dtype=pl.Float64).alias("first_death_roll"),
        )

    w = max(1, window)
    result: list[pl.DataFrame] = []
    for xuid_val in df["xuid"].unique().to_list():
        sub = df.filter(pl.col("xuid") == xuid_val).sort("start_time")
        sub = sub.with_columns(
            pl.col("first_kill_s")
            .rolling_mean(window_size=w, min_samples=_ROLL_MIN_SAMPLES)
            .alias("first_kill_roll"),
            pl.col("first_death_s")
            .rolling_mean(window_size=w, min_samples=_ROLL_MIN_SAMPLES)
            .alias("first_death_roll"),
        )
        result.append(sub)
    return pl.concat(result) if result else df
