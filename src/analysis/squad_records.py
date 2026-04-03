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


def compute_player_pm_records(
    df: pl.DataFrame,
    pair_name: str | None,
) -> tuple[float | None, float | None, float | None]:
    """Calcule les records de stats par minute pour un joueur.

    Args:
        df: DataFrame joueur avec kills, deaths, assists, time_played_seconds.
        pair_name: Filtre exact sur pair_name (None = tous les matchs).

    Returns:
        Tuple (record max kills/min, record min deaths/min, record max assists/min).
        Chaque valeur est None si les données sont insuffisantes.
    """
    sub = df
    if pair_name is not None and "pair_name" in df.columns:
        sub = df.filter(pl.col("pair_name") == pair_name)
    # Alias duration_seconds → time_played_seconds si nécessaire
    if "time_played_seconds" not in sub.columns and "duration_seconds" in sub.columns:
        sub = sub.with_columns(pl.col("duration_seconds").alias("time_played_seconds"))
    needed = {"kills", "deaths", "assists", "time_played_seconds"}
    if sub.is_empty() or not needed.issubset(sub.columns):
        logger.debug("compute_player_pm_records: données insuffisantes pour pair_name=%r", pair_name)
        return None, None, None
    sub = sub.filter(
        pl.col("time_played_seconds").is_not_null() & (pl.col("time_played_seconds") > 0)
    )
    if sub.is_empty():
        return None, None, None
    pm = sub.with_columns([
        (pl.col("kills").cast(pl.Float64) / (pl.col("time_played_seconds") / 60)).alias("_kpm"),
        (pl.col("deaths").cast(pl.Float64) / (pl.col("time_played_seconds") / 60)).alias("_dpm"),
        (pl.col("assists").cast(pl.Float64) / (pl.col("time_played_seconds") / 60)).alias("_apm"),
    ])
    return pm["_kpm"].max(), pm["_dpm"].min(), pm["_apm"].max()


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


def compute_squad_records_per_map(
    players: list[tuple[str, pl.DataFrame]],
    metrics: list[tuple[str, bool]],
    dominant_pair: str | None,
) -> dict[str, dict[str, dict[str, float | None]]]:
    """Calcule les records par carte pour N joueurs et M métriques.

    Args:
        players: Liste de (nom_joueur, df_historique_complet).
            Les DataFrames doivent contenir une colonne map_name.
        metrics: Liste de (nom_colonne, is_negative).
        dominant_pair: pair_name filtrant. Si None, utilise tous les matchs.

    Returns:
        Dict {nom_joueur: {nom_métrique: {map_name: valeur | None}}}.
        La structure est nom_joueur → métrique → carte → valeur pour
        permettre aux graphiques de trancher d'abord par métrique.
    """
    result: dict[str, dict[str, dict[str, float | None]]] = {}
    for name, df in players:
        _map_col = "map_ui" if df is not None and "map_ui" in df.columns else "map_name"
        if df is None or df.is_empty() or _map_col not in df.columns:
            result[name] = {}
            continue
        sub = df
        if dominant_pair is not None and "pair_name" in df.columns:
            sub = df.filter(pl.col("pair_name") == dominant_pair)
        if sub.is_empty():
            result[name] = {}
            continue
        map_names = sub[_map_col].drop_nulls().unique().to_list()
        # Initialise metric → {} for each metric
        by_metric: dict[str, dict[str, float | None]] = {m: {} for m, _ in metrics}
        for map_name in map_names:
            if not map_name:
                continue
            map_sub = sub.filter(pl.col(_map_col) == map_name)
            for metric, is_neg in metrics:
                if metric not in map_sub.columns:
                    continue
                vals = map_sub.select(metric).drop_nulls()
                if vals.is_empty():
                    continue
                col = vals[metric]
                val = col.min() if is_neg else col.max()
                if val is not None:
                    by_metric[metric][map_name] = float(val)
        result[name] = by_metric
    return result
