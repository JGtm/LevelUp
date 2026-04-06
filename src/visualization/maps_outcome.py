"""Graphiques par carte centrés sur l'outcome (lollipop, timeline, bullet, perf vs historique).

Extrait de maps.py pour respecter la limite de 500 lignes.
Ces 4 fonctions partagent la dimension outcome (W/D/L) et complètent
plot_map_ratio_with_winloss pour les vues session courte et longue période.
"""

from __future__ import annotations

import plotly.graph_objects as go

from src.analysis.performance_config import SCORE_THRESHOLDS
from src.config import HALO_COLORS, PLOT_CONFIG
from src.visualization._compat import DataFrameLike, ensure_polars, to_pandas_for_plotly
from src.visualization._maps_outcome_bullet import (  # noqa: F401 — re-export
    _sort_by_map_order,
    plot_map_winrate_bullet,
)
from src.visualization._maps_outcome_history import (  # noqa: F401 — re-export
    plot_map_perf_vs_history,
)
from src.visualization._maps_outcome_timeline import (  # noqa: F401 — re-export
    plot_map_outcome_timeline,
)
from src.visualization.theme import apply_halo_plot_style

__all__ = [
    "plot_map_lollipop",
    "plot_map_outcome_timeline",
    "plot_map_perf_vs_history",
    "plot_map_winrate_bullet",
]

# ─── Helpers privés ─────────────────────────────────────────────────────────


def _perf_color(v: float) -> str:
    """Retourne la couleur de performance selon SCORE_THRESHOLDS."""
    c = HALO_COLORS.as_dict()
    if v >= SCORE_THRESHOLDS["excellent"]:
        return c["green"]
    if v >= SCORE_THRESHOLDS["good"]:
        return c["cyan"]
    if v >= SCORE_THRESHOLDS["average"]:
        return c["amber"]
    if v >= SCORE_THRESHOLDS["below_average"]:
        return c.get("orange", "#FF8C00")
    return c["red"]


def _empty_map_figure() -> go.Figure:
    """Figure vide standardisée pour les graphiques de carte."""
    fig = go.Figure()
    fig.update_layout(
        height=PLOT_CONFIG.default_height, margin={"l": 40, "r": 20, "t": 30, "b": 40}
    )
    return apply_halo_plot_style(fig, height=PLOT_CONFIG.default_height)


# ─── Option A — Lollipop ────────────────────────────────────────────────────


def plot_map_lollipop(
    df_breakdown: DataFrameLike,
    lang: str = "fr",
    map_order: list[str] | None = None,
    color_by_perf: bool = False,
) -> go.Figure:
    """Lollipop chart : win rate par carte (outcome-focused).

    Chaque carte = tige grise + cercle coloré.
    Couleur par défaut : vert >= 50 %, rouge sinon. Si color_by_perf=True: gamme performance.
    Taille du cercle proportionnelle au nombre de matchs.

    Args:
        df_breakdown: DataFrame issu de compute_map_breakdown.
        lang: Langue cible.
        map_order: Ordre chronologique des cartes (oldest=index 0). Si None, tri par win_rate.
        color_by_perf: Si True, couleur selon le score de performance (gamme globale).

    Returns:
        Figure Plotly.
    """
    df_pl = ensure_polars(df_breakdown).drop_nulls(subset=["win_rate", "matches"])
    if df_pl.is_empty():
        return _empty_map_figure()

    if map_order is not None:
        df_pl = _sort_by_map_order(df_pl, map_order)
    else:
        df_pl = df_pl.sort("win_rate")
    d = to_pandas_for_plotly(df_pl)
    colors = HALO_COLORS.as_dict()
    win_rates = list(d["win_rate"])
    map_names = list(d["map_name"])
    match_counts = list(d["matches"])

    stems_x: list[float | None] = []
    stems_y: list[str | None] = []
    for wr, mn in zip(win_rates, map_names, strict=False):
        stems_x += [0.0, wr, None]
        stems_y += [mn, mn, None]

    if color_by_perf and "performance_avg" in d.columns:
        dot_colors = [_perf_color(v if v == v else 45.0) for v in d["performance_avg"]]
    else:
        dot_colors = [colors["green"] if wr >= 0.5 else colors["red"] for wr in win_rates]
    sizes = [min(24, 12 + int(n**0.5 * 3)) for n in match_counts]
    texts = [
        f"{wr:.0%}" if n > 1 else ("V" if wr >= 0.5 else "D")
        for wr, n in zip(win_rates, match_counts, strict=False)
    ]
    fig = go.Figure()
    fig.add_trace(
        go.Scatter(
            x=stems_x,
            y=stems_y,
            mode="lines",
            line={"color": "rgba(160,160,160,0.4)", "width": 2},
            showlegend=False,
            hoverinfo="skip",
        )
    )
    fig.add_trace(
        go.Scatter(
            x=win_rates,
            y=map_names,
            mode="markers+text",
            marker={"color": dot_colors, "size": sizes, "line": {"color": "white", "width": 1.5}},
            text=texts,
            textposition="middle center",
            textfont={"size": 9, "color": "white", "weight": "bold"},
            customdata=list(zip(match_counts, win_rates, strict=False)),
            hovertemplate="%{y}<br>Win=%{customdata[1]:.0%}  N=%{customdata[0]}<extra></extra>",
            showlegend=False,
        )
    )
    fig.add_vline(x=0.5, line={"dash": "dot", "color": "rgba(180,180,180,0.5)", "width": 1})
    fig.update_layout(height=PLOT_CONFIG.tall_height, margin={"l": 40, "r": 30, "t": 30, "b": 40})
    fig.update_xaxes(tickformat=".0%", range=[-0.05, 1.15])
    return apply_halo_plot_style(fig, height=PLOT_CONFIG.tall_height)
