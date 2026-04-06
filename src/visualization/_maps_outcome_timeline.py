"""Timeline des outcomes par carte (extraction de maps_outcome.py).

Visualisation chronologique des matchs par carte avec mise en évidence de la session.
"""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl

from src.config import HALO_COLORS, PLOT_CONFIG
from src.data.domain.refdata import Outcome
from src.visualization._compat import DataFrameLike, ensure_polars
from src.visualization.theme import apply_halo_plot_style, get_legend_horizontal_bottom

# ─── Constantes ─────────────────────────────────────────────────────────────

_MAX_MAPS_TIMELINE = 15

# (outcome_group_int, colors_key, label_fr, label_en)
_TIMELINE_OUTCOMES: list[tuple[int, str, str, str]] = [
    (int(Outcome.WIN), "green", "Victoire", "Win"),
    (int(Outcome.LOSS), "red", "Défaite", "Loss"),
    (0, "violet", "Autre", "Other"),
]

# ─── Helpers privés ─────────────────────────────────────────────────────────


def _prepare_timeline_df(df_pl: pl.DataFrame, session_ids: set[str]) -> pl.DataFrame:
    """Trie et enrichit le DataFrame pour la timeline (index temporel, groupe outcome)."""
    return (
        df_pl.filter(
            pl.col("map_name").is_not_null() & (pl.col("map_name").str.strip_chars() != "")
        )
        .sort("start_time")
        .with_row_index("_global_idx")
        .with_columns(
            (pl.col("_global_idx") - pl.col("_global_idx").min().over("map_name")).alias("_idx"),
            pl.when(pl.col("outcome") == int(Outcome.WIN))
            .then(int(Outcome.WIN))
            .when(pl.col("outcome") == int(Outcome.LOSS))
            .then(int(Outcome.LOSS))
            .otherwise(0)
            .alias("_outcome_group"),
            pl.col("match_id").cast(pl.Utf8).is_in(list(session_ids)).alias("_is_session"),
        )
    )


def _add_timeline_traces(fig: go.Figure, sub: pl.DataFrame, color: str, label: str) -> None:
    """Ajoute les traces historique (petits) + session (grands, anneau or) pour un outcome."""
    sub_hist = sub.filter(~pl.col("_is_session"))
    sub_sess = sub.filter(pl.col("_is_session"))
    if not sub_hist.is_empty():
        fig.add_trace(
            go.Scatter(
                x=sub_hist["_idx"].to_list(),
                y=sub_hist["map_name"].to_list(),
                mode="markers",
                name=label,
                legendgroup=label,
                marker={"color": color, "size": 8, "opacity": 0.4},
                showlegend=True,
            )
        )
    if not sub_sess.is_empty():
        fig.add_trace(
            go.Scatter(
                x=sub_sess["_idx"].to_list(),
                y=sub_sess["map_name"].to_list(),
                mode="markers",
                name=label,
                legendgroup=label,
                marker={
                    "color": color,
                    "size": 14,
                    "opacity": 1.0,
                    "line": {"color": "white", "width": 2},
                },
                showlegend=False,
            )
        )


# ─── Fonction publique ──────────────────────────────────────────────────────


def plot_map_outcome_timeline(
    df_matches: DataFrameLike,
    session_match_ids: list[str] | None = None,
    lang: str = "fr",
) -> go.Figure | None:
    """Timeline des outcomes par carte : un cercle coloré par match, ordonné chronologiquement.

    Les matchs de la sélection courante sont mis en évidence (plus grands, contour blanc).

    Args:
        df_matches: DataFrame brut (map_name, start_time, outcome, match_id).
        session_match_ids: IDs des matchs a mettre en evidence.
        lang: Langue.

    Returns:
        Figure Plotly ou None si donnees insuffisantes.
    """
    required = {"map_name", "start_time", "outcome", "match_id"}
    df_pl = ensure_polars(df_matches)
    if df_pl.is_empty() or not required.issubset(set(df_pl.columns)):
        return None

    # Utiliser map_ui (traduit) si disponible pour les labels des cartes
    if "map_ui" in df_pl.columns:
        df_pl = df_pl.with_columns(pl.col("map_ui").alias("map_name"))

    session_ids: set[str] = set(session_match_ids or [])
    top_maps = (
        df_pl.filter(pl.col("map_name").is_not_null())
        .group_by("map_name")
        .agg(pl.len().alias("_cnt"))
        .sort("_cnt", descending=True)
        .head(_MAX_MAPS_TIMELINE)["map_name"]
        .to_list()
    )
    df_pl = df_pl.filter(pl.col("map_name").is_in(top_maps))
    if df_pl.is_empty():
        return None

    df_proc = _prepare_timeline_df(df_pl, session_ids)
    colors = HALO_COLORS.as_dict()
    fig = go.Figure()
    for outcome_val, color_key, label_fr, label_en in _TIMELINE_OUTCOMES:
        label = label_fr if lang == "fr" else label_en
        sub = df_proc.filter(pl.col("_outcome_group") == outcome_val)
        if not sub.is_empty():
            _add_timeline_traces(fig, sub, colors[color_key], label)

    if not fig.data:
        return None

    # Overlay doré : met en évidence TOUS les matchs de la session (quel que soit l'outcome)
    sess_all = df_proc.filter(pl.col("_is_session"))
    if not sess_all.is_empty():
        sess_lbl = "Session actuelle" if lang == "fr" else "Current session"
        fig.add_trace(
            go.Scatter(
                x=sess_all["_idx"].to_list(),
                y=sess_all["map_name"].to_list(),
                mode="markers",
                name=sess_lbl,
                marker={
                    "symbol": "circle-open",
                    "color": "rgba(0,0,0,0)",
                    "size": 18,
                    "line": {"color": "#FFD700", "width": 2.5},
                },
                showlegend=True,
            )
        )

    height = max(PLOT_CONFIG.default_height, len(top_maps) * 34 + 80)
    fig.update_layout(
        height=height,
        margin={"l": 40, "r": 20, "t": 30, "b": 80},
        xaxis={"visible": False},
        legend=get_legend_horizontal_bottom(),
    )
    return apply_halo_plot_style(fig, height=height)
