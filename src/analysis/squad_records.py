"""Calcul des records historiques par joueur pour la page Escouade.

Fonctions pures : entrée Polars DataFrame, sortie scalaires ou dicts.
Aucun accès DB, aucun import Streamlit.
"""

from __future__ import annotations

import logging

import polars as pl

logger = logging.getLogger(__name__)


def get_dominant_pair_name(dfs: list[pl.DataFrame]) -> str | None:
    """Retourne le pair_name le plus fréquent parmi tous les DataFrames fournis.

    Args:
        dfs: Liste de DataFrames joueurs (doivent avoir une colonne pair_name).

    Returns:
        Le pair_name dominant (mode), ou None si aucune donnée disponible.
    """
    parts: list[pl.Series] = []
    for df in dfs:
        if df is not None and not df.is_empty() and "pair_name" in df.columns:
            parts.append(df["pair_name"].drop_nulls())
    if not parts:
        logger.debug("get_dominant_pair_name: aucun DataFrame avec colonne pair_name")
        return None
    combined = pl.concat(parts)
    if combined.is_empty():
        return None
    counts = combined.value_counts(sort=True)
    return str(counts[0, "pair_name"])


def compute_player_record(
    df: pl.DataFrame,
    metric: str,
    pair_name: str | None,
    *,
    is_negative: bool = False,
) -> float | None:
    """Calcule le record historique d'un joueur pour une métrique.

    Args:
        df: DataFrame du joueur (colonnes : pair_name, metric, ...).
        metric: Colonne à analyser.
        pair_name: Filtre exact sur pair_name. Si None, utilise tous les matchs.
        is_negative: Si True, record = min (le plus proche de 0 est le meilleur).

    Returns:
        Valeur record (max ou min selon is_negative), ou None si données vides.
    """
    if df is None or df.is_empty() or metric not in df.columns:
        logger.debug("compute_player_record: données absentes pour metric=%s", metric)
        return None
    sub = df
    if pair_name is not None and "pair_name" in df.columns:
        sub = df.filter(pl.col("pair_name") == pair_name)
        if sub.is_empty():
            logger.debug(
                "compute_player_record: aucune ligne pour pair_name=%r metric=%s",
                pair_name,
                metric,
            )
            return None
    sub = sub.select(metric).drop_nulls()
    if sub.is_empty():
        return None
    col = sub[metric]
    value = col.min() if is_negative else col.max()
    if value is None:
        return None
    return float(value)


def compute_squad_records(
    players: list[tuple[str, pl.DataFrame]],
    metrics: list[tuple[str, bool]],
    dominant_pair: str | None,
) -> dict[str, dict[str, float | None]]:
    """Calcule les records escouade pour N joueurs et M métriques.

    Args:
        players: Liste de (nom_joueur, df_historique_complet).
        metrics: Liste de (nom_colonne, is_negative).
        dominant_pair: pair_name filtrant (ex: "Arena:4v4 Slayer").

    Returns:
        Dict {nom_joueur: {nom_métrique: valeur_record | None}}.
    """
    result: dict[str, dict[str, float | None]] = {}
    for name, df in players:
        player_records: dict[str, float | None] = {}
        for metric, is_neg in metrics:
            player_records[metric] = compute_player_record(
                df, metric, dominant_pair, is_negative=is_neg
            )
        result[name] = player_records
    return result
