"""Enrichissement i18n centralisé des DataFrames de matchs.

Ce module est la source de vérité unique pour le calcul de ``map_ui``,
``mode_ui`` et ``playlist_ui``. Il doit être appelé **une seule fois**
juste après ``load_df_optimized()``/``load_match_dataframe()``, avant
de passer ``df`` à quoi que ce soit d'autre.

Règle : 0 import Streamlit, 0 accès DB — module pur testable.
"""

from __future__ import annotations

import logging
import re

import polars as pl

logger = logging.getLogger(__name__)


def _strip_mode_map_suffix(s: str | None) -> str | None:
    """Normalise un label de mode déjà traduit : supprime ' on NomCarte' + suffixes Forge/Ranked.

    Appliqué sur ``pair_name_fr`` (déjà traduit depuis ``v_match_full``) sans accès DB.
    """
    if not s:
        return None
    s = str(s).strip()
    if " on " in s:
        s = s.split(" on ", 1)[0].strip()
    s = re.sub(r"\s*-\s*Forge\b", "", s, flags=re.IGNORECASE).strip()
    s = re.sub(r"\s*-\s*Ranked\b", "", s, flags=re.IGNORECASE).strip()
    return s or None


def add_i18n_display_columns(df: pl.DataFrame, lang: str = "fr") -> pl.DataFrame:
    """Ajoute ``map_ui``, ``mode_ui``, ``playlist_ui`` au DataFrame.

    Utilise les colonnes ``*_fr`` déjà présentes (depuis ``v_match_full``) comme
    source primaire — aucun appel SQL, aucun callback.  Les colonnes sont
    ajoutées de façon idempotente : si elles existent déjà, elles ne sont
    pas recalculées.

    Priorité pour chaque colonne (lang == "fr") :
    - ``map_ui``      : ``map_name_fr``      → ``map_name``
    - ``mode_ui``     : ``_strip_mode_map_suffix(coalesce(pair_name_fr, pair_name))``
    - ``playlist_ui`` : ``playlist_name_fr`` → ``playlist_name``

    ``mode_ui`` est normalisé via ``_strip_mode_map_suffix`` (sans accès DB) :
    suppression du suffixe " on NomCarte" et des variantes Forge/Ranked.

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
            # pair_name_fr est déjà traduit — normaliser sans accès DB (supp. ' on Carte', Forge, Ranked)
            exprs.append(
                pl.coalesce([
                    pl.col("pair_name_fr").cast(pl.Utf8),
                    pl.col("pair_name").cast(pl.Utf8),
                ])
                .map_elements(_strip_mode_map_suffix, return_dtype=pl.Utf8)
                .alias("mode_ui")
            )
        else:
            # EN : passthrough pair_name (les fonctions aval normalisent si besoin)
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
