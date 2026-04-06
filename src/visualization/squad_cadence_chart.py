"""Profils de tempo synchronisés — barres groupées par joueur.

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
    color_map: dict[str, str] | None = None,
) -> go.Figure | None:
    """Construit le graphe barres groupées de tempo synchronisé.

    Args:
        profiles_df: DataFrame avec colonne ``phase`` (0-9) +
                     une colonne par joueur (avg kills/phase).
        player_names: Noms des joueurs (dans l'ordre des colonnes).
        opts: Options de rendu V3.
        color_map: Mapping {nom: couleur_hex} issu d'``assign_player_colors``.
                   Si absent, la palette interne est utilisée.

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
    # Axe numérique : chaque barre est centrée à i*pct_step + pct_step/2
    # Les ticks sont placés exactement aux frontières : 0, 10, 20 … 100
    x_centers = [i * pct_step + pct_step / 2 for i in range(n_buckets)]
    x_tickvals = list(range(0, n_buckets * pct_step + 1, pct_step))
    x_ticktext = [f"{v}%" for v in x_tickvals]
    bar_width = pct_step  # chaque groupe occupe exactement une tranche

    # Calcul du range Y depuis les données pour amplifier les variations
    player_cols = [n for n in player_names if n in profiles_df.columns]
    y_max = profiles_df.select(player_cols).max().to_numpy().max() if player_cols else 1.0
    y_range = [0, max(float(y_max) * 1.25, 1.0)]

    fig = go.Figure()

    n_players = sum(1 for n in player_names if n in profiles_df.columns)
    for i, name in enumerate(player_names):
        if name not in profiles_df.columns:
            continue
        color = color_map.get(name) if color_map else _PLAYER_COLORS[i % len(_PLAYER_COLORS)]
        values = profiles_df[name].to_list()

        fig.add_trace(
            go.Bar(
                x=x_centers,
                y=values,
                name=name,
                width=[bar_width * 0.9 / max(n_players, 1)] * n_buckets,
                offset=[(i - n_players / 2) * bar_width * 0.9 / max(n_players, 1)] * n_buckets,
                marker_color=color,
                marker_line={"color": color, "width": 0.5},
                hovertemplate=f"{name}: %{{y:.1f}} kills<extra></extra>",
            )
        )

    fig.update_layout(
        barmode="overlay",
        height=opts.height_px,
        plot_bgcolor=theme.bg_plot,
        paper_bgcolor=theme.bg_plot,
        font={"color": theme.font_color, "size": 12},
        margin={"l": 40, "r": 20, "t": 10, "b": 80},
        legend={
            "orientation": "h",
            "yanchor": "top",
            "y": -0.15,
            "xanchor": "center",
            "x": 0.5,
            "font": {"size": 11},
        },
        xaxis={
            "title": viz_t("axis_intensity_phase", lang),
            "gridcolor": theme.grid_color,
            "tickvals": x_tickvals,
            "ticktext": x_ticktext,
            "range": [0, n_buckets * pct_step],
        },
        yaxis={
            "title": viz_t("axis_cadence_kills", lang),
            "gridcolor": theme.grid_color,
            "zeroline": True,
            "zerolinecolor": theme.zero_line_color,
            "zerolinewidth": 1,
            "range": y_range,
            "dtick": 0.1,
            "tickformat": ".1f",
        },
    )

    return fig
