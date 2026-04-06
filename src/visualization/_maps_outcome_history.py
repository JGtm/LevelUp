"""Performance vs historique par carte (extraction de maps_outcome.py).

Barres horizontales groupées comparant la session actuelle à l'historique.
"""

from __future__ import annotations

import plotly.graph_objects as go

from src.analysis.performance_config import SCORE_THRESHOLDS
from src.config import HALO_COLORS, PLOT_CONFIG
from src.visualization._compat import DataFrameLike, ensure_polars, to_pandas_for_plotly
from src.visualization._maps_outcome_bullet import _sort_by_map_order
from src.visualization.theme import apply_halo_plot_style, get_legend_horizontal_bottom

# ─── Helper privé ───────────────────────────────────────────────────────────


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


# ─── Fonction publique ──────────────────────────────────────────────────────


def plot_map_perf_vs_history(
    bd_current: DataFrameLike,
    bd_history: DataFrameLike,
    lang: str = "fr",
    map_order: list[str] | None = None,
) -> go.Figure | None:
    """Barres horizontales groupées : performance session vs historique par carte.

    Les barres session sont colorées selon la gamme de performance (vert/cyan/ambre/orange/rouge).
    Les barres historique restent grises (référence neutre).

    Args:
        bd_current: Breakdown session (map_name, performance_avg, matches).
        bd_history: Breakdown historique (map_name, performance_avg, matches).
        lang: Langue.
        map_order: Ordre chronologique des cartes (oldest=index 0). Si None, tri par perf.

    Returns:
        Figure Plotly ou None si aucune carte commune.
    """
    bd_curr = ensure_polars(bd_current).drop_nulls(subset=["performance_avg"])
    bd_hist = (
        ensure_polars(bd_history)
        .drop_nulls(subset=["performance_avg"])
        .select(["map_name", "performance_avg", "matches"])
        .rename({"performance_avg": "_hist_perf", "matches": "_hist_n"})
    )
    joined = bd_curr.join(bd_hist, on="map_name", how="inner")
    if joined.is_empty():
        return None

    if map_order is not None:
        joined = _sort_by_map_order(joined, map_order)
    else:
        joined = joined.sort("performance_avg", descending=False).head(20)
    d = to_pandas_for_plotly(joined)
    hist_lbl = "Historique" if lang == "fr" else "History"
    sess_lbl = "Session actuelle" if lang == "fr" else "Current session"

    # Couleurs session basées sur la gamme de performance
    bar_colors = [_perf_color(v) for v in d["performance_avg"]]

    fig = go.Figure()
    fig.add_trace(
        go.Bar(
            x=d["_hist_perf"],
            y=d["map_name"],
            orientation="h",
            name=hist_lbl,
            marker_color="rgba(120,120,120,0.45)",
            customdata=d["_hist_n"],
            hovertemplate=f"%{{y}}<br>{hist_lbl}=%{{x:.1f}} (N=%{{customdata}})<extra></extra>",
        )
    )
    fig.add_trace(
        go.Bar(
            x=d["performance_avg"],
            y=d["map_name"],
            orientation="h",
            name=sess_lbl,
            marker_color=bar_colors,
            opacity=0.85,
            customdata=d["matches"],
            hovertemplate=f"%{{y}}<br>{sess_lbl}=%{{x:.1f}} (N=%{{customdata}})<extra></extra>",
        )
    )
    fig.add_vline(x=0.0, line={"dash": "dot", "color": "rgba(180,180,180,0.6)", "width": 1})
    fig.update_layout(
        barmode="group",
        height=PLOT_CONFIG.tall_height,
        margin={"l": 40, "r": 20, "t": 30, "b": 70},
        legend=get_legend_horizontal_bottom(),
    )
    return apply_halo_plot_style(fig, height=PLOT_CONFIG.tall_height)
