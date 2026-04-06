"""Heatmap de performance par joueur × carte — section squad.

Extrait de friends_impact_heatmap.py pour respecter la limite de 500 lignes.
Contient : plot_squad_map_heatmap et ses helpers privés.
"""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl

from src.analysis.performance_config import SCORE_THRESHOLDS
from src.config import HALO_COLORS
from src.visualization._compat import DataFrameLike, ensure_polars
from src.visualization.theme import apply_halo_plot_style


def _top_maps_by_frequency(all_bd: pl.DataFrame) -> list[str]:
    """Retourne les 15 cartes les plus jouées."""
    return (
        all_bd.group_by("map_name")
        .agg(pl.col("matches").sum())
        .sort("matches", descending=True)
        .head(15)["map_name"]
        .to_list()
    )


def _order_maps_by_first_seen(
    series: list[tuple[str, DataFrameLike]], maps: list[str]
) -> list[str]:
    """Trie les cartes par première apparition chronologique dans les données brutes."""
    raw_frames: list[pl.DataFrame] = []
    for _, df in series:
        df_pl = ensure_polars(df)
        if df_pl.is_empty():
            continue
        # Préférer map_ui (traduit) si disponible pour l'ordre chronologique
        _map_col = (
            "map_ui"
            if "map_ui" in df_pl.columns
            else ("map_name" if "map_name" in df_pl.columns else None)
        )
        if _map_col is not None and "start_time" in df_pl.columns:
            raw_frames.append(df_pl.select([pl.col(_map_col).alias("map_name"), "start_time"]))

    if not raw_frames:
        return maps

    first_seen = (
        pl.concat(raw_frames, how="diagonal_relaxed")
        .filter(pl.col("map_name").is_in(maps))
        .group_by("map_name")
        .agg(pl.col("start_time").min().alias("first_seen"))
        .sort("first_seen")
    )
    ordered = first_seen["map_name"].to_list()
    return ordered + [m for m in maps if m not in ordered]


def _build_perf_matrix(
    all_bd: pl.DataFrame,
    player_names: list[str],
    maps: list[str],
) -> list[list[object]]:
    """Construit la matrice joueur × carte des performances."""
    matrix: list[list[object]] = []
    for name in player_names:
        row_bd = all_bd.filter(pl.col("_player") == name)
        perf_map = dict(
            zip(row_bd["map_name"].to_list(), row_bd["performance_avg"].to_list(), strict=False)
        )
        matrix.append([perf_map.get(m) for m in maps])
    return matrix


def _discrete_perf_colorscale() -> list[list[object]]:
    """Colorscale à paliers discrets cohérente avec _perf_color."""
    c = HALO_COLORS.as_dict()
    _max = 100.0
    _eps = 1e-4
    _orange = "#FF8C00"
    return [
        [0.0, c["red"]],
        [SCORE_THRESHOLDS["below_average"] / _max - _eps, c["red"]],
        [SCORE_THRESHOLDS["below_average"] / _max, _orange],
        [SCORE_THRESHOLDS["average"] / _max - _eps, _orange],
        [SCORE_THRESHOLDS["average"] / _max, c["amber"]],
        [SCORE_THRESHOLDS["good"] / _max - _eps, c["amber"]],
        [SCORE_THRESHOLDS["good"] / _max, c["cyan"]],
        [SCORE_THRESHOLDS["excellent"] / _max - _eps, c["cyan"]],
        [SCORE_THRESHOLDS["excellent"] / _max, c["green"]],
        [1.0, c["green"]],
    ]


# ─── Feature 1 — Heatmap joueur × carte ─────────────────────────────────────


def plot_squad_map_heatmap(
    series: list[tuple[str, DataFrameLike]],
    lang: str = "fr",
) -> go.Figure | None:
    """Heatmap de performance par joueur × carte.

    Chaque cellule = performance_avg du joueur sur cette carte (top 15 cartes par fréquence).

    Args:
        series: Liste de (nom_joueur, df_matchs) pour chaque membre de l'escouade.
        lang: Langue.

    Returns:
        Figure Plotly ou None si données insuffisantes.
    """
    from src.analysis import compute_map_breakdown  # noqa: PLC0415

    if not series:
        return None

    player_bds: list[pl.DataFrame] = []
    for name, df in series:
        df_pl = ensure_polars(df)
        if df_pl.is_empty():
            continue
        # Ajouter map_ui si absent mais map_name_fr disponible (lang=fr)
        # Garantit des noms FR cohérents sur l'axe X de la heatmap.
        if "map_ui" not in df_pl.columns and "map_name_fr" in df_pl.columns and lang == "fr":
            df_pl = df_pl.with_columns(
                pl.coalesce(
                    [pl.col("map_name_fr").cast(pl.Utf8), pl.col("map_name").cast(pl.Utf8)]
                ).alias("map_ui")
            )
        bd = compute_map_breakdown(df_pl)
        if not bd.is_empty():
            player_bds.append(bd.with_columns(pl.lit(name).alias("_player")))

    if not player_bds:
        return None

    all_bd = pl.concat(player_bds, how="diagonal_relaxed")
    top_maps = _order_maps_by_first_seen(series, _top_maps_by_frequency(all_bd))

    all_bd = all_bd.filter(pl.col("map_name").is_in(top_maps))
    if all_bd.is_empty():
        return None

    player_names = [n for n, _ in series]
    matrix = _build_perf_matrix(all_bd, player_names, top_maps)

    colorscale = _discrete_perf_colorscale()
    perf_lbl = "Perf"
    height = max(180, len(player_names) * 60 + 100)
    fig = go.Figure(
        go.Heatmap(
            z=matrix,
            x=top_maps,
            y=player_names,
            colorscale=colorscale,
            zmin=0,
            zmax=100,
            hovertemplate=f"%{{y}} — %{{x}}<br>{perf_lbl}=%{{z:.1f}}<extra></extra>",
            colorbar={"title": perf_lbl, "thickness": 15},
        )
    )
    fig.update_layout(
        height=height,
        margin={"l": 40, "r": 80, "t": 30, "b": 120},
        xaxis={"tickangle": -35},
    )
    return apply_halo_plot_style(fig, height=height)
