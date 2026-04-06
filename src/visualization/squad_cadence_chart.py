"""Profils de tempo synchronisés — multi-lignes par joueur.

Architecture V3 : PlotOptions + ChartTheme.
"""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl

from src.ui.i18n.viz import viz_t
from src.visualization._plot_options import PlotOptions

# Palette Okabe-Ito étendue pour les joueurs
_PLAYER_COLORS = [
    "#0072B2",  # Bleu — joueur principal
    "#009E73",  # Vert
    "#CC79A7",  # Rose mauve
    "#E69F00",  # Orange
]


def plot_squad_cadence_profiles(
    profiles_df: pl.DataFrame,
    player_names: list[str],
    opts: PlotOptions | None = None,
) -> go.Figure | None:
    """Construit le graphe multi-lignes de tempo synchronisé.

    Args:
        profiles_df: DataFrame avec colonne ``phase`` (0-9) +
                     une colonne par joueur (avg kills/phase).
        player_names: Noms des joueurs (dans l'ordre des colonnes).
        opts: Options de rendu V3.

    Returns:
        Figure Plotly ou None si données insuffisantes.
    """
    if profiles_df.is_empty() or not player_names:
        return None

    opts = opts or PlotOptions()
    theme = opts.theme
    lang = opts.lang
    n_buckets = len(profiles_df)

    pct_step = 100 // n_buckets
    x_labels = [f"{i * pct_step}–{(i + 1) * pct_step}%" for i in range(n_buckets)]

    fig = go.Figure()

    for i, name in enumerate(player_names):
        if name not in profiles_df.columns:
            continue
        color = _PLAYER_COLORS[i % len(_PLAYER_COLORS)]
        values = profiles_df[name].to_list()

        fig.add_trace(
            go.Scatter(
                x=x_labels,
                y=values,
                mode="lines+markers",
                name=name,
                line={"color": color, "width": 2.5},
                marker={"color": color, "size": 6},
                hovertemplate=f"{name}: %{{y:.1f}} kills<extra></extra>",
            )
        )

    fig.update_layout(
        height=opts.height_px,
        plot_bgcolor=theme.bg_plot,
        paper_bgcolor=theme.bg_plot,
        font={"color": theme.font_color, "size": 12},
        margin={"l": 40, "r": 20, "t": 30, "b": 40},
        legend={
            "orientation": "h",
            "yanchor": "bottom",
            "y": 1.02,
            "xanchor": "center",
            "x": 0.5,
            "font": {"size": 11},
        },
        xaxis={
            "title": viz_t("axis_intensity_phase", lang),
            "gridcolor": theme.grid_color,
        },
        yaxis={
            "title": viz_t("axis_cadence_kills", lang),
            "gridcolor": theme.grid_color,
            "zeroline": True,
            "zerolinecolor": theme.zero_line_color,
            "zerolinewidth": 1,
        },
    )

    return fig
