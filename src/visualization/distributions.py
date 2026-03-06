"""Graphiques de distributions et répartitions.

Les fonctions liées aux outcome (outcomes_over_time, stacked_outcomes,
heatmap, matches_at_top) ont été extraites dans distributions_outcomes.py
(Sprint 16).  Elles sont ré-exportées ici pour compatibilité.
"""

from __future__ import annotations

from typing import TYPE_CHECKING

import numpy as np
import plotly.graph_objects as go
import polars as pl

from src.config import HALO_COLORS, PLOT_CONFIG
from src.ui.i18n.viz import viz_t
from src.visualization._compat import (
    DataFrameLike,
    ensure_polars,
    ensure_polars_series,
)
from src.visualization.theme import (
    apply_halo_plot_style,
    get_legend_horizontal_bottom,  # noqa: F401 – re-export implicite
)

if TYPE_CHECKING:
    import pandas as pd


def plot_kda_distribution(df: DataFrameLike, lang: str = "fr") -> go.Figure:
    """Graphique de distribution du KDA (FDA) avec KDE.

    Args:
        df: DataFrame (Pandas ou Polars) avec colonne kda.

    Returns:
        Figure Plotly avec densité KDE et rug plot.
    """
    df = ensure_polars(df)

    colors = HALO_COLORS.as_dict()
    d = df.drop_nulls(subset=["kda"])
    x = d.get_column("kda").cast(pl.Float64, strict=False).drop_nulls().to_numpy()

    if x.size == 0:
        fig = go.Figure()
        fig.update_layout(
            height=PLOT_CONFIG.default_height, margin={"l": 40, "r": 20, "t": 30, "b": 40}
        )
        fig.update_xaxes(title_text=viz_t("axis_fda", lang))
        fig.update_yaxes(title_text=viz_t("trace_density_short", lang))
        return apply_halo_plot_style(fig, height=PLOT_CONFIG.default_height)

    # KDE gaussien (règle de Silverman)
    n = int(x.size)
    std = float(np.std(x, ddof=1)) if n > 1 else 0.0
    q25, q75 = np.percentile(x, [25, 75]).tolist() if n > 1 else [0.0, 0.0]
    iqr = float(q75 - q25)
    sigma = min(std, iqr / 1.34) if (std > 0 and iqr > 0) else max(std, iqr / 1.34)
    bw = (1.06 * sigma * (n ** (-1.0 / 5.0))) if sigma and sigma > 0 else 0.3
    bw = float(max(bw, 0.05))

    xmin = float(np.min(x))
    xmax = float(np.max(x))
    span = max(0.25, xmax - xmin)
    pad = 0.15 * span
    grid = np.linspace(xmin - pad, xmax + pad, 256)
    z = (grid[:, None] - x[None, :]) / bw
    dens = np.exp(-0.5 * (z**2)).sum(axis=1) / (n * bw * np.sqrt(2.0 * np.pi))

    fig = go.Figure()
    fig.add_trace(
        go.Scatter(
            x=grid,
            y=dens,
            mode="lines",
            name=viz_t("trace_density", lang),
            line={"width": PLOT_CONFIG.line_width, "color": colors["cyan"]},
            fill="tozeroy",
            fillcolor="rgba(53,208,255,0.18)",
            hovertemplate=viz_t("hover_kde", lang),
        )
    )

    # Rug plot
    fig.add_trace(
        go.Scatter(
            x=x,
            y=np.zeros_like(x),
            mode="markers",
            name=viz_t("trace_matches", lang),
            marker={"symbol": "line-ns-open", "size": 10, "color": "rgba(255,255,255,0.45)"},
            hovertemplate=viz_t("hover_kda_rug", lang),
        )
    )

    fig.add_vline(x=0, line_width=1, line_dash="dot", line_color="rgba(255,255,255,0.35)")

    # Ligne médiane
    if len(x) > 0:
        median_val = float(np.median(x))
        fig.add_vline(
            x=median_val,
            line_dash="dash",
            line_color="#ffaa00",
            annotation_text=viz_t("annot_median", lang, val=f"{median_val:.2f}"),
            annotation_position="top right",
            annotation_font_color="#ffaa00",
        )

    fig.update_layout(
        height=PLOT_CONFIG.default_height, margin={"l": 40, "r": 20, "t": 30, "b": 40}
    )
    fig.update_xaxes(title_text=viz_t("axis_fda", lang), zeroline=True)
    fig.update_yaxes(title_text=viz_t("trace_density_short", lang), rangemode="tozero")

    return apply_halo_plot_style(fig, height=PLOT_CONFIG.default_height)


def plot_top_weapons(
    weapons_data: list[dict],
    *,
    title: str | None = None,
    top_n: int = 10,
    lang: str = "fr",
) -> go.Figure:
    """Graphique des armes les plus utilisées.

    Args:
        weapons_data: Liste de dicts avec weapon_name, total_kills, headshot_rate, accuracy.
        title: Titre optionnel.
        top_n: Nombre d'armes à afficher.

    Returns:
        Figure Plotly avec barres horizontales.
    """
    colors = HALO_COLORS.as_dict()

    if not weapons_data:
        fig = go.Figure()
        fig.update_layout(height=PLOT_CONFIG.default_height)
        return apply_halo_plot_style(fig, title=title)

    # Limiter et trier
    data = sorted(weapons_data, key=lambda x: x.get("total_kills", 0), reverse=True)[:top_n]

    names = [w.get("weapon_name", "?") for w in data][::-1]
    kills = [w.get("total_kills", 0) for w in data][::-1]
    hs_rates = [w.get("headshot_rate", 0) for w in data][::-1]
    accuracies = [w.get("accuracy", 0) for w in data][::-1]

    fig = go.Figure()

    fig.add_trace(
        go.Bar(
            x=kills,
            y=names,
            orientation="h",
            name=viz_t("trace_kills", lang),
            marker_color=colors["cyan"],
            opacity=0.85,
            text=[viz_t("text_kills_count", lang, k=k) for k in kills],
            textposition="outside",
            customdata=list(zip(hs_rates, accuracies, strict=False)),
            hovertemplate=viz_t("hover_weapons", lang),
        )
    )

    height = max(PLOT_CONFIG.default_height, 30 * len(names) + 80)

    fig.update_layout(
        height=height,
        margin={"l": 120, "r": 60, "t": 60 if title else 30, "b": 40},
    )
    fig.update_xaxes(title_text=viz_t("axis_kills", lang))
    fig.update_yaxes(title_text="")

    return apply_halo_plot_style(fig, title=title, height=height)


def plot_histogram(  # noqa: PLR0913
    values: pd.Series | pl.Series | np.ndarray,
    *,
    title: str | None = None,
    x_label: str = "Valeur",
    y_label: str | None = None,
    bins: int | str = "auto",
    color: str | None = None,
    show_kde: bool = False,
    lang: str = "fr",
) -> go.Figure:
    """Histogramme générique avec option KDE.

    Args:
        values: Série (Pandas ou Polars) ou array de valeurs numériques.
        title: Titre optionnel.
        x_label: Label de l'axe X.
        y_label: Label de l'axe Y.
        bins: Nombre de bins ou "auto".
        color: Couleur des barres (défaut: cyan).
        show_kde: Afficher la courbe KDE superposée.

    Returns:
        Figure Plotly avec histogramme.
    """
    colors = HALO_COLORS.as_dict()
    bar_color = color or colors["cyan"]
    if y_label is None:
        y_label = viz_t("axis_frequency", lang)

    if isinstance(values, np.ndarray):
        x = values[~np.isnan(values)].astype(float)
    else:
        s = ensure_polars_series(values)
        x = s.cast(pl.Float64, strict=False).drop_nulls().to_numpy()

    if x.size == 0:
        fig = go.Figure()
        fig.update_layout(height=PLOT_CONFIG.default_height)
        return apply_halo_plot_style(fig, title=title)

    # Calculer les bins
    n_bins = min(50, max(10, int(np.sqrt(x.size)))) if bins == "auto" else int(bins)

    fig = go.Figure()

    fig.add_trace(
        go.Histogram(
            x=x,
            nbinsx=n_bins,
            name=x_label,
            marker_color=bar_color,
            opacity=0.75,
            hovertemplate=f"{x_label}: %{{x}}<br>{y_label}: %{{y}}<extra></extra>",
        )
    )

    if show_kde and x.size > 10:
        # KDE simple
        n = int(x.size)
        std = float(np.std(x, ddof=1)) if n > 1 else 0.0
        if std > 0:
            bw = 1.06 * std * (n ** (-1.0 / 5.0))
            bw = max(bw, 0.01)

            xmin, xmax = float(np.min(x)), float(np.max(x))
            pad = 0.1 * (xmax - xmin)
            grid = np.linspace(xmin - pad, xmax + pad, 128)
            z = (grid[:, None] - x[None, :]) / bw
            dens = np.exp(-0.5 * (z**2)).sum(axis=1) / (n * bw * np.sqrt(2 * np.pi))

            # Normaliser pour matcher l'histogramme
            hist_counts, hist_edges = np.histogram(x, bins=n_bins)
            bin_width = hist_edges[1] - hist_edges[0]
            dens_scaled = dens * n * bin_width

            fig.add_trace(
                go.Scatter(
                    x=grid,
                    y=dens_scaled,
                    mode="lines",
                    name=viz_t("trace_density_short", lang),
                    line={"color": colors["amber"], "width": 2},
                    hoverinfo="skip",
                )
            )

    # Ligne médiane
    if len(x) > 0:
        median_val = float(np.median(x))
        fig.add_vline(
            x=median_val,
            line_dash="dash",
            line_color="#ffaa00",
            annotation_text=viz_t("annot_median", lang, val=f"{median_val:.1f}"),
            annotation_position="top right",
            annotation_font_color="#ffaa00",
        )

    fig.update_layout(
        height=PLOT_CONFIG.default_height,
        margin={"l": 40, "r": 20, "t": 60 if title else 30, "b": 40},
        bargap=0.05,
    )
    fig.update_xaxes(title_text=x_label)
    fig.update_yaxes(title_text=y_label)

    return apply_halo_plot_style(fig, title=title, height=PLOT_CONFIG.default_height)


def plot_medals_distribution(
    medals_data: list[tuple[int, int]],
    medal_names: dict[int, str],
    *,
    title: str | None = None,
    top_n: int = 20,
    lang: str = "fr",
) -> go.Figure:
    """Graphique de distribution des médailles (barres horizontales).

    Args:
        medals_data: Liste de tuples (medal_name_id, count).
        medal_names: Dictionnaire {medal_name_id: nom_traduit}.
        title: Titre optionnel.
        top_n: Nombre de médailles à afficher.

    Returns:
        Figure Plotly avec barres horizontales.
    """
    if not medals_data:
        fig = go.Figure()
        fig.update_layout(height=PLOT_CONFIG.default_height)
        return apply_halo_plot_style(fig, title=title)

    # Trier et limiter
    sorted_medals = sorted(medals_data, key=lambda x: x[1], reverse=True)[:top_n]

    names = [
        medal_names.get(m[0], f"Medal #{m[0]}" if lang == "en" else f"Médaille #{m[0]}")
        for m in sorted_medals
    ]
    counts = [m[1] for m in sorted_medals]

    # Inverser pour afficher du plus grand au plus petit (haut -> bas)
    names = names[::-1]
    counts = counts[::-1]

    # Dégradé de couleurs basé sur le rang
    n = len(counts)
    gradient_colors = [f"rgba(53, 208, 255, {0.4 + 0.5 * (i / max(1, n - 1))})" for i in range(n)]

    fig = go.Figure(
        data=go.Bar(
            x=counts,
            y=names,
            orientation="h",
            marker_color=gradient_colors,
            text=counts,
            textposition="outside",
            hovertemplate="%{y}<br>Nombre: %{x}<extra></extra>",
        )
    )

    height = max(PLOT_CONFIG.default_height, 25 * len(names) + 80)

    fig.update_layout(
        height=height,
        margin={"l": 40, "r": 60, "t": 60 if title else 30, "b": 40},
    )
    fig.update_xaxes(title_text=viz_t("axis_count", lang))
    fig.update_yaxes(title_text="")

    return apply_halo_plot_style(fig, title=title, height=height)


# ---------------------------------------------------------------------------
# Re-exports depuis _distributions_advanced (compat backward)
# ---------------------------------------------------------------------------
from src.visualization._distributions_advanced import (  # noqa: E402, F401
    plot_correlation_scatter,
    plot_first_event_distribution,
)

# ---------------------------------------------------------------------------
# Re-exports depuis distributions_outcomes (compat backward — Sprint 16)
# ---------------------------------------------------------------------------
from src.visualization.distributions_outcomes import (  # noqa: E402, F401
    plot_matches_at_top_by_week,
    plot_outcomes_over_time,
    plot_stacked_outcomes_by_category,
    plot_win_ratio_heatmap,
)
