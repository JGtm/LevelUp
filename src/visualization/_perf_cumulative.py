"""Graphiques cumulatifs K/D et net score.

Extraits de performance.py — regroupe les charts cumulatifs séquentiels.
"""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl

from src.config import HALO_COLORS, THEME_COLORS
from src.ui.i18n.viz import viz_t
from src.visualization.theme import apply_halo_plot_style

# =============================================================================
# Configuration des couleurs
# =============================================================================

# Couleurs mises à jour pour respecter la palette Okabe-Ito (accessibilité daltonisme).
PERFORMANCE_COLORS = {
    "positive": HALO_COLORS.green,  # Vert néon pour positif
    "negative": HALO_COLORS.red,  # Rouge pour négatif
    "neutral": HALO_COLORS.cyan,  # Cyan pour neutre
    "kills": "#009E73",  # Vert bleuté Okabe-Ito (visible deuteranopes)
    "deaths": "#D55E00",  # Vermillon Okabe-Ito (distinct du vert bleuté)
    "kd_line": "#E69F00",  # Orange Okabe-Ito pour K/D
    "cumulative": "#56B4E9",  # Bleu ciel Okabe-Ito pour cumulatif
    "rolling": "#CC79A7",  # Rose mauve Okabe-Ito pour rolling
    "trend_up": "#009E73",  # Vert bleuté Okabe-Ito pour amélioration
    "trend_down": "#D55E00",  # Vermillon Okabe-Ito pour dégradation
    "baseline": "#999999",  # Gris neutre
}


# =============================================================================
# Helpers internes
# =============================================================================


def _add_duration_markers(
    fig: go.Figure,
    x_values: list,
    time_played_seconds: list[int | float] | None,
    marker_interval_minutes: float,
) -> None:
    """Ajoute des lignes verticales marquant les intervalles de durée cumulée."""
    if not time_played_seconds or not x_values:
        return

    if len(time_played_seconds) != len(x_values):
        return

    interval_seconds = marker_interval_minutes * 60
    cumul_time = 0.0
    next_marker = interval_seconds

    for i, tps in enumerate(time_played_seconds):
        if tps is None:
            continue
        cumul_time += float(tps)
        while cumul_time >= next_marker:
            total_min = int(next_marker / 60)
            x_val = x_values[i]
            fig.add_shape(
                type="line",
                x0=x_val,
                x1=x_val,
                y0=0,
                y1=1,
                yref="paper",
                line={"color": PERFORMANCE_COLORS["baseline"], "width": 1, "dash": "dot"},
                opacity=0.5,
            )
            fig.add_annotation(
                x=x_val,
                y=1.0,
                yref="paper",
                text=f"{total_min} min",
                showarrow=False,
                font={"size": 9, "color": PERFORMANCE_COLORS["baseline"]},
                yanchor="bottom",
            )
            next_marker += interval_seconds


# =============================================================================
# Graphiques de performance cumulée
# =============================================================================


def _add_cumulative_score_traces(
    fig: go.Figure,
    x_values: list,
    y_cumulative: list,
    y_match: list,
    lang: str,
) -> None:
    """Ajoute les traces scatter cumulé et bar par match."""
    line_color = (
        PERFORMANCE_COLORS["positive"] if y_cumulative[-1] >= 0 else PERFORMANCE_COLORS["negative"]
    )
    fig.add_trace(
        go.Scatter(
            x=x_values,
            y=y_cumulative,
            mode="lines+markers",
            name=viz_t("trace_net_score_cumul", lang),
            line={"color": line_color, "width": 3},
            marker={"size": 8, "color": line_color},
            hovertemplate=viz_t("hover_cumul_score", lang),
        )
    )
    bar_colors = [
        PERFORMANCE_COLORS["positive"] if v >= 0 else PERFORMANCE_COLORS["negative"]
        for v in y_match
    ]
    fig.add_trace(
        go.Bar(
            x=x_values,
            y=y_match,
            name=viz_t("trace_net_score_match", lang),
            marker_color=bar_colors,
            opacity=0.5,
            hovertemplate="<b>%{x}</b><br>Match: %{y:+d}<extra></extra>",
        )
    )


def plot_cumulative_net_score(  # noqa: PLR0913
    cumulative_df: pl.DataFrame,
    *,
    title: str | None = None,
    height: int = 400,
    show_zero_line: bool = True,
    time_played_seconds: list[int | float] | None = None,
    duration_marker_minutes: float = 8.0,
    lang: str = "fr",
) -> go.Figure:
    """Crée un graphique du net score cumulé au fil des matchs."""
    if title is None:
        title = viz_t("title_cumul_net_score", lang)
    fig = go.Figure()

    if cumulative_df.is_empty():
        fig.add_annotation(
            text=viz_t("empty_no_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 16, "color": THEME_COLORS.text_primary},
        )
        return apply_halo_plot_style(fig, title=title, height=height)

    data = cumulative_df.to_dicts()
    x_values = [d.get("start_time", "") for d in data]
    y_cumulative = [d.get("cumulative_net_score", 0) for d in data]
    y_match = [d.get("net_score", 0) for d in data]

    _add_cumulative_score_traces(fig, x_values, y_cumulative, y_match, lang)

    if show_zero_line:
        fig.add_hline(
            y=0,
            line_width=2,
            line_color="rgba(255,255,255,0.75)",
            annotation_text=viz_t("label_balance", lang),
            annotation_position="right",
        )

    fig.update_layout(
        yaxis_title=viz_t("axis_net_score", lang),
        xaxis_title="Match",
        hovermode="x unified",
        showlegend=True,
        legend={"orientation": "h", "yanchor": "bottom", "y": 1.02, "xanchor": "center", "x": 0.5},
    )

    _add_duration_markers(fig, x_values, time_played_seconds, duration_marker_minutes)

    return apply_halo_plot_style(fig, title=title, height=height)


def plot_cumulative_kd(  # noqa: PLR0913
    cumulative_df: pl.DataFrame,
    *,
    title: str | None = None,
    height: int = 400,
    show_target: float | None = 1.0,
    time_played_seconds: list[int | float] | None = None,
    duration_marker_minutes: float = 8.0,
    lang: str = "fr",
) -> go.Figure:
    """Crée un graphique du K/D cumulé au fil des matchs."""
    if title is None:
        title = viz_t("title_cumul_kd", lang)
    fig = go.Figure()

    if cumulative_df.is_empty():
        fig.add_annotation(
            text=viz_t("empty_no_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 16, "color": THEME_COLORS.text_primary},
        )
        return apply_halo_plot_style(fig, title=title, height=height)

    data = cumulative_df.to_dicts()
    x_values = [d.get("start_time", "") for d in data]
    y_cumulative = [d.get("cumulative_kd", 0) for d in data]
    y_match = [d.get("kd", 0) for d in data]

    fig.add_trace(
        go.Scatter(
            x=x_values,
            y=y_cumulative,
            mode="lines+markers",
            name=viz_t("trace_kd_cumul", lang),
            line={"color": PERFORMANCE_COLORS["kd_line"], "width": 3},
            marker={"size": 8, "color": PERFORMANCE_COLORS["kd_line"]},
            hovertemplate=viz_t("hover_kd_cumul_line", lang),
        )
    )

    fig.add_trace(
        go.Scatter(
            x=x_values,
            y=y_match,
            mode="markers",
            name=viz_t("trace_kd_match", lang),
            marker={
                "size": 10,
                "color": PERFORMANCE_COLORS["neutral"],
                "opacity": 0.6,
                "symbol": "circle-open",
            },
            hovertemplate=viz_t("hover_kd_match", lang),
        )
    )

    if show_target is not None:
        fig.add_hline(
            y=show_target,
            line_dash="dash",
            line_color=PERFORMANCE_COLORS["baseline"],
            annotation_text=viz_t("label_target", lang, value=show_target),
            annotation_position="right",
        )

    fig.update_layout(
        yaxis_title=viz_t("axis_kd_ratio", lang),
        xaxis_title="Match",
        hovermode="x unified",
        showlegend=True,
        legend={"orientation": "h", "yanchor": "bottom", "y": 1.02, "xanchor": "center", "x": 0.5},
    )

    _add_duration_markers(fig, x_values, time_played_seconds, duration_marker_minutes)

    return apply_halo_plot_style(fig, title=title, height=height)


def plot_rolling_kd(
    rolling_df: pl.DataFrame,
    *,
    window_size: int = 5,
    title: str | None = None,
    height: int = 400,
    lang: str = "fr",
) -> go.Figure:
    """Crée un graphique du K/D glissant."""
    if title is None:
        title = viz_t("title_rolling_kd", lang, window=window_size)

    fig = go.Figure()

    if rolling_df.is_empty():
        fig.add_annotation(
            text=viz_t("empty_no_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 16, "color": THEME_COLORS.text_primary},
        )
        return apply_halo_plot_style(fig, title=title, height=height)

    data = rolling_df.to_dicts()
    x_values = [d.get("start_time", "") for d in data]
    y_rolling = [d.get("rolling_kd", 0) for d in data]
    y_match = [d.get("kd", 0) for d in data]

    fig.add_trace(
        go.Scatter(
            x=x_values,
            y=y_match,
            mode="lines",
            name=viz_t("trace_kd_match", lang),
            line={"color": PERFORMANCE_COLORS["neutral"], "width": 1},
            opacity=0.4,
            hovertemplate=viz_t("hover_kd_match", lang),
        )
    )

    fig.add_trace(
        go.Scatter(
            x=x_values,
            y=y_rolling,
            mode="lines+markers",
            name=viz_t("trace_kd_rolling", lang, window=window_size),
            line={"color": PERFORMANCE_COLORS["rolling"], "width": 3},
            marker={"size": 6, "color": PERFORMANCE_COLORS["rolling"]},
            hovertemplate=viz_t("hover_kd_rolling_line", lang),
        )
    )

    fig.add_hline(
        y=1.0,
        line_dash="dash",
        line_color=PERFORMANCE_COLORS["baseline"],
        annotation_text=viz_t("label_kd_ref", lang),
        annotation_position="right",
    )

    fig.update_layout(
        yaxis_title=viz_t("axis_kd_ratio", lang),
        xaxis_title="Match",
        hovermode="x unified",
        showlegend=True,
        legend={"orientation": "h", "yanchor": "bottom", "y": 1.02, "xanchor": "center", "x": 0.5},
    )

    return apply_halo_plot_style(fig, title=title, height=height)
