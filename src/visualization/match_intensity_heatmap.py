"""Heatmap d'intensité : profil de kills normalisé par phase pour chaque match.

Architecture V3 : PlotOptions + ChartTheme.
"""

from __future__ import annotations

import plotly.graph_objects as go

from src.analysis.match_intensity import IntensityProfile
from src.ui.i18n.viz import viz_t
from src.visualization._plot_options import PlotOptions


def _build_heatmap_trace(
    z_data: object,
    x_labels: list[str],
    y_labels: list[str],
    lang: str,
) -> go.Heatmap:
    """Construit la trace Heatmap avec colorscale cyan→jaune→rouge."""
    return go.Heatmap(
        z=z_data,
        x=x_labels,
        y=y_labels,
        colorscale=[
            [0.0, "#38C8C8"],
            [0.25, "#FFFF00"],
            [0.5, "#FFB300"],
            [0.75, "#FF5500"],
            [1.0, "#FF1A00"],
        ],
        colorbar={
            "title": {
                "text": viz_t("axis_cadence_kills", lang),
                "font": {"size": 11},
            },
            "thickness": 12,
            "len": 0.8,
        },
        hovertemplate=(
            viz_t("axis_intensity_phase", lang)
            + ": %{x}<br>"
            + viz_t("axis_intensity_match", lang)
            + ": %{y}<br>"
            + viz_t("axis_cadence_kills", lang)
            + ": %{z}"
            + "<extra></extra>"
        ),
    )


def plot_match_intensity_heatmap(
    profile: IntensityProfile,
    opts: PlotOptions | None = None,
) -> go.Figure | None:
    """Construit la heatmap d'intensité matches × phases.

    Args:
        profile: IntensityProfile avec colonnes match_id + phase_0…phase_9.
        opts: Options de rendu V3.

    Returns:
        Figure Plotly ou None si données insuffisantes.
    """
    if profile.df.is_empty() or len(profile.df) < 2:
        return None

    opts = opts or PlotOptions()
    theme = opts.theme
    lang = opts.lang
    n = profile.n_buckets

    phase_cols = [f"phase_{i}" for i in range(n)]
    z_data = profile.df.select(phase_cols).to_numpy()
    y_labels = [f"#{i + 1}" for i in range(len(profile.df))]
    pct_step = 100 // n
    x_labels = [f"{i * pct_step}–{(i + 1) * pct_step}%" for i in range(n)]

    fig = go.Figure(_build_heatmap_trace(z_data, x_labels, y_labels, lang))
    height = min(max(200, len(profile.df) * 22 + 80), 600)

    fig.update_layout(
        height=height,
        plot_bgcolor=theme.bg_plot,
        paper_bgcolor=theme.bg_plot,
        font={"color": theme.font_color, "size": 12},
        margin={"l": 50, "r": 20, "t": 10, "b": 40},
        xaxis={
            "title": viz_t("axis_intensity_phase", lang),
            "side": "bottom",
        },
        yaxis={
            "title": "",
            "autorange": "reversed",
        },
    )

    return fig
