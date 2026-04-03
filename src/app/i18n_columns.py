"""Enrichissement i18n centralisé des DataFrames de matchs.

Ce module est la source de vérité unique pour le calcul de ``map_ui``,
``mode_ui`` et ``playlist_ui``. Il doit être appelé **une seule fois**
juste après ``load_df_optimized()``/``load_match_dataframe()``, avant
de passer ``df`` à quoi que ce soit d'autre.

Règle : 0 import Streamlit, 0 accès DB — module pur testable.
"""

from __future__ import annotations

import logging

import polars as pl

logger = logging.getLogger(__name__)


def add_i18n_display_columns(df: pl.DataFrame, lang: str = "fr") -> pl.DataFrame:
    """Ajoute ``map_ui``, ``mode_ui``, ``playlist_ui`` au DataFrame.

    Utilise les colonnes ``*_fr`` déjà présentes (depuis ``v_match_full``) comme
    source primaire — aucun appel SQL, aucun callback.  Les colonnes sont
    ajoutées de façon idempotente : si elles existent déjà, elles ne sont
    pas recalculées.

    Priorité pour chaque colonne (lang == "fr") :
    - ``map_ui``      : ``map_name_fr``     → ``map_name``
    - ``mode_ui``     : ``pair_name_fr``    → ``pair_name``
    - ``playlist_ui`` : ``playlist_name_fr`` → ``playlist_name``

    En lang == "en", on utilise directement les colonnes EN (passthrough).

    Args:
        df:   DataFrame Polars (sortie de ``load_df_optimized``).
        lang: Code de langue courant ("fr" ou "en").

    Returns:
        DataFrame enrichi (même objet ou nouveau si des colonnes sont ajoutées).
    """
    if df.is_empty():
        return df

    exprs: list[pl.Expr] = []

    # --- map_ui ---
    if "map_ui" not in df.columns and "map_name" in df.columns:
        if lang == "fr" and "map_name_fr" in df.columns:
            exprs.append(
                pl.coalesce([
                    pl.col("map_name_fr").cast(pl.Utf8),
                    pl.col("map_name").cast(pl.Utf8),
                ]).alias("map_ui")
            )
        else:
            exprs.append(pl.col("map_name").cast(pl.Utf8).alias("map_ui"))

    # --- mode_ui ---
    if "mode_ui" not in df.columns and "pair_name" in df.columns:
        if lang == "fr" and "pair_name_fr" in df.columns:
            exprs.append(
                pl.coalesce([
                    pl.col("pair_name_fr").cast(pl.Utf8),
                    pl.col("pair_name").cast(pl.Utf8),
                ]).alias("mode_ui")
            )
        else:
            exprs.append(pl.col("pair_name").cast(pl.Utf8).alias("mode_ui"))

    # --- playlist_ui ---
    if "playlist_ui" not in df.columns and "playlist_name" in df.columns:
        if lang == "fr" and "playlist_name_fr" in df.columns:
            exprs.append(
                pl.coalesce([
                    pl.col("playlist_name_fr").cast(pl.Utf8),
                    pl.col("playlist_name").cast(pl.Utf8),
                ]).alias("playlist_ui")
            )
        else:
            exprs.append(pl.col("playlist_name").cast(pl.Utf8).alias("playlist_ui"))

    if exprs:
        df = df.with_columns(exprs)
        logger.debug(
            "add_i18n_display_columns: %d colonnes ajoutées (lang=%s, rows=%d)",
            len(exprs),
            lang,
            len(df),
        )

    return df
