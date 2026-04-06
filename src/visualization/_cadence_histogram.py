"""Histogramme de cadence de kills par tranche — barres empilées + moyenne glissante.

Architecture V3 : PlotOptions + ChartTheme.
"""

from __future__ import annotations

import plotly.graph_objects as go

from src.analysis.match_cadence import CadenceBucket, compute_cadence_moving_avg
from src.ui.i18n.viz import viz_t
from src.visualization._plot_options import PlotOptions
from src.visualization._team_dominance_helpers import (
    ENEMY_COLOR,
    ENEMY_RGBA,
    MY_TEAM_COLOR,
    MY_TEAM_RGBA,
)

_MA_MY_TEAM_COLOR = "rgba(0, 114, 178, 0.85)"  # MY_TEAM bleu semi-opaque
_MA_ENEMY_COLOR = "rgba(213, 94, 0, 0.85)"  # ENEMY vermillion semi-opaque


def _format_time_label(seconds: float) -> str:
    """Convertit des secondes en label lisible (ex: '1:30')."""
    m, s = divmod(int(seconds), 60)
    return f"{m}:{s:02d}"


def _add_cadence_traces(
    fig: go.Figure,
    x_labels: list[str],
    buckets: list[CadenceBucket],
    lang: str,
) -> None:
    """Ajoute les 4 traces (barres + moyennes glissantes par équipe)."""
    fig.add_trace(
        go.Bar(
            x=x_labels,
            y=[b.my_kills for b in buckets],
            name=viz_t("trace_cadence_my_team", lang),
            marker_color=MY_TEAM_RGBA,
            marker_line={"color": MY_TEAM_COLOR, "width": 1},
            hovertemplate="%{y} " + viz_t("hover_cadence_kills", lang) + "<extra></extra>",
        )
    )
    fig.add_trace(
        go.Bar(
            x=x_labels,
            y=[b.enemy_kills for b in buckets],
            name=viz_t("trace_cadence_enemy", lang),
            marker_color=ENEMY_RGBA,
            marker_line={"color": ENEMY_COLOR, "width": 1},
            hovertemplate="%{y} " + viz_t("hover_cadence_kills", lang) + "<extra></extra>",
        )
    )
    ma_my = compute_cadence_moving_avg(buckets, window=3, field="my_kills")
    ma_enemy = compute_cadence_moving_avg(buckets, window=3, field="enemy_kills")
    # Bordures blanches fines (traces fantômes sous les courbes MA)
    _outline = {"color": "rgba(255,255,255,0.8)", "width": 5}
    for ma_vals in (ma_my, ma_enemy):
        fig.add_trace(
            go.Scatter(
                x=x_labels,
                y=ma_vals,
                mode="lines",
                line=_outline,
                hoverinfo="skip",
                showlegend=False,
            )
        )
    fig.add_trace(
        go.Scatter(
            x=x_labels,
            y=ma_my,
            mode="lines",
            name=viz_t("trace_cadence_ma_my_team", lang),
            line={"color": _MA_MY_TEAM_COLOR, "width": 3},
            hovertemplate="%{y:.1f}<extra></extra>",
        )
    )
    fig.add_trace(
        go.Scatter(
            x=x_labels,
            y=ma_enemy,
            mode="lines",
            name=viz_t("trace_cadence_ma_enemy", lang),
            line={"color": _MA_ENEMY_COLOR, "width": 3},
            hovertemplate="%{y:.1f}<extra></extra>",
        )
    )


def plot_match_cadence_histogram(
    buckets: list[CadenceBucket],
    duration_s: float,
    opts: PlotOptions | None = None,
) -> go.Figure | None:
    """Construit l'histogramme de cadence bicolore (mon équipe / adverse).

    Args:
        buckets: Tranches de cadence avec my_kills / enemy_kills.
        duration_s: Durée totale du match en secondes.
        opts: Options de rendu V3.

    Returns:
        Figure Plotly ou None si données insuffisantes.
    """
    if not buckets or sum(b.total for b in buckets) < 3:
        return None

    opts = opts or PlotOptions()
    theme = opts.theme
    lang = opts.lang
    x_labels = [_format_time_label(b.t_center_s) for b in buckets]

    fig = go.Figure()
    _add_cadence_traces(fig, x_labels, buckets, lang)

    fig.update_layout(
        barmode="stack",
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
            "title": viz_t("axis_cadence_time", lang),
            "gridcolor": theme.grid_color,
            "showgrid": False,
        },
        yaxis={
            "title": viz_t("axis_cadence_kills", lang),
            "gridcolor": theme.grid_color,
            "zeroline": True,
            "zerolinecolor": theme.zero_line_color,
            "zerolinewidth": 1,
        },
        bargap=0.15,
    )

    max_total = max(b.total for b in buckets)
    if max_total > 0:
        peak_idx = next(i for i, b in enumerate(buckets) if b.total == max_total)
        fig.add_annotation(
            x=x_labels[peak_idx],
            y=max_total,
            text=viz_t("label_cadence_peak", lang),
            showarrow=True,
            arrowhead=2,
            arrowsize=0.8,
            arrowcolor=theme.annotation_color,
            font={"size": 10, "color": theme.annotation_color},
            yshift=10,
        )

    return fig
