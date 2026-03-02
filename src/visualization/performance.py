"""Graphiques de performance cumulée avec Plotly.

Sprint 6: Visualisations des séries cumulatives (net score, K/D, tendances).
"""

from __future__ import annotations

from typing import Any

import plotly.graph_objects as go
import polars as pl
from plotly.subplots import make_subplots

from src.config import HALO_COLORS, THEME_COLORS
from src.ui.i18n.viz import viz_t
from src.visualization.theme import apply_halo_plot_style

# =============================================================================
# Configuration des couleurs
# =============================================================================

# Couleurs mises à jour pour respecter la palette Okabe-Ito (accessibilité daltonisme).
# Anciens code hexadécimaux (deuteranopie/protanopie-incompatibles) :
#   kills:      #00ff00 (vert néon)   → #009E73 (vert bleuté)
#   deaths:     #ff4444 (rouge néon)  → #D55E00 (vermillon)
#   kd_line:    #ffaa00 (orange chaud) → #E69F00 (orange Okabe-Ito)
#   cumulative: #00ccff (cyan)         → #56B4E9 (bleu ciel)
#   rolling:    #ff66ff (magenta)      → #CC79A7 (rose mauve)
#   trend_up:   #00ff88 (vert clair)   → #009E73 (vert bleuté)
#   trend_down: #ff6666 (rouge clair)  → #D55E00 (vermillon)
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
    """Ajoute des lignes verticales marquant les intervalles de durée cumulée.

    Par exemple, si marker_interval_minutes=8, des lignes verticales sont
    tracées tous les ~8 min de jeu cumulé.

    Args:
        fig: Figure Plotly à modifier en place.
        x_values: Valeurs de l'axe X (start_time des matchs).
        time_played_seconds: Durée de chaque match en secondes.
        marker_interval_minutes: Intervalle entre marqueurs (en minutes).
    """
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
            # Utiliser add_shape + add_annotation au lieu de add_vline
            # pour éviter les erreurs avec les axes datetime/string
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


def plot_cumulative_net_score(
    cumulative_df: pl.DataFrame,
    *,
    title: str | None = None,
    height: int = 400,
    show_zero_line: bool = True,
    time_played_seconds: list[int | float] | None = None,
    duration_marker_minutes: float = 8.0,
    lang: str = "fr",
) -> go.Figure:
    """Crée un graphique du net score cumulé au fil des matchs.

    Le graphique montre la progression du net score (kills - deaths) avec
    une coloration positive/négative.

    Args:
        cumulative_df: DataFrame avec colonnes start_time, net_score, cumulative_net_score.
        title: Titre du graphique.
        height: Hauteur en pixels.
        show_zero_line: Afficher la ligne de référence à zéro.
        time_played_seconds: Durée de chaque match (secondes) pour marqueurs de durée.
        duration_marker_minutes: Intervalle en minutes entre les marqueurs de durée.

    Returns:
        Figure Plotly avec le graphique.

    Example:
        >>> df = compute_cumulative_net_score_series_polars(match_stats)
        >>> fig = plot_cumulative_net_score(df)
        >>> st.plotly_chart(fig, config={"displayModeBar": False})
    """
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

    # Convertir en dicts pour Plotly
    data = cumulative_df.to_dicts()

    # Extraire les données
    x_values = [d.get("start_time", "") for d in data]
    y_cumulative = [d.get("cumulative_net_score", 0) for d in data]
    y_match = [d.get("net_score", 0) for d in data]

    # Couleur selon positif/négatif
    line_color = (
        PERFORMANCE_COLORS["positive"] if y_cumulative[-1] >= 0 else PERFORMANCE_COLORS["negative"]
    )

    # Ligne du cumul
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

    # Barres pour net score par match
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

    # Ligne de référence à zéro
    if show_zero_line:
        fig.add_hline(
            y=0,
            line_dash="dash",
            line_color=PERFORMANCE_COLORS["baseline"],
            annotation_text=viz_t("label_balance", lang),
            annotation_position="right",
        )

    # Layout
    fig.update_layout(
        yaxis_title=viz_t("axis_net_score", lang),
        xaxis_title="Match",
        hovermode="x unified",
        showlegend=True,
        legend={"orientation": "h", "yanchor": "bottom", "y": 1.02, "xanchor": "center", "x": 0.5},
    )

    # Marqueurs de durée cumulée (Sprint 6 - 6.4)
    _add_duration_markers(fig, x_values, time_played_seconds, duration_marker_minutes)

    return apply_halo_plot_style(fig, title=title, height=height)


def plot_cumulative_kd(
    cumulative_df: pl.DataFrame,
    *,
    title: str | None = None,
    height: int = 400,
    show_target: float | None = 1.0,
    time_played_seconds: list[int | float] | None = None,
    duration_marker_minutes: float = 8.0,
    lang: str = "fr",
) -> go.Figure:
    """Crée un graphique du K/D cumulé au fil des matchs.

    Args:
        cumulative_df: DataFrame avec colonnes start_time, kd, cumulative_kd.
        title: Titre du graphique.
        height: Hauteur en pixels.
        show_target: Afficher une ligne cible (ex: 1.0 pour K/D équilibré).
        time_played_seconds: Durée de chaque match (secondes) pour marqueurs de durée.
        duration_marker_minutes: Intervalle en minutes entre les marqueurs de durée.

    Returns:
        Figure Plotly avec le graphique.
    """
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

    # K/D cumulé
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

    # K/D par match (points)
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

    # Ligne cible
    if show_target is not None:
        fig.add_hline(
            y=show_target,
            line_dash="dash",
            line_color=PERFORMANCE_COLORS["baseline"],
            annotation_text=viz_t("label_target", lang, value=show_target),
            annotation_position="right",
        )

    # Layout
    fig.update_layout(
        yaxis_title=viz_t("axis_kd_ratio", lang),
        xaxis_title="Match",
        hovermode="x unified",
        showlegend=True,
        legend={"orientation": "h", "yanchor": "bottom", "y": 1.02, "xanchor": "center", "x": 0.5},
    )

    # Marqueurs de durée cumulée (Sprint 6 - 6.4)
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
    """Crée un graphique du K/D glissant.

    Args:
        rolling_df: DataFrame avec colonnes start_time, kd, rolling_kd.
        window_size: Taille de la fenêtre (pour le titre).
        title: Titre personnalisé (par défaut: "K/D Glissant (5 matchs)").
        height: Hauteur en pixels.

    Returns:
        Figure Plotly avec le graphique.
    """
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

    # K/D par match (en fond)
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

    # K/D glissant
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

    # Ligne de référence à 1.0
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


def plot_session_trend(
    match_stats_df: pl.DataFrame,
    *,
    title: str | None = None,
    height: int = 350,
    lang: str = "fr",
) -> go.Figure:
    """Crée un graphique montrant la tendance de la session.

    Compare la première et la seconde moitié avec une indication visuelle
    de l'amélioration ou de la dégradation.

    Args:
        match_stats_df: DataFrame des matchs triés par start_time.
        title: Titre du graphique.
        height: Hauteur en pixels.

    Returns:
        Figure Plotly avec indicateurs de tendance.
    """
    if title is None:
        title = viz_t("title_session_trend", lang)
    fig = go.Figure()

    if match_stats_df.is_empty() or len(match_stats_df) < 4:
        fig.add_annotation(
            text=viz_t("empty_not_enough_matches", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 14, "color": THEME_COLORS.text_primary},
        )
        return apply_halo_plot_style(fig, title=title, height=height)

    # Import local pour éviter les imports circulaires
    from src.analysis.cumulative import compute_session_trend_polars

    trend_data = compute_session_trend_polars(match_stats_df)

    # Créer un graphique à indicateurs
    first_kd = trend_data.get("first_half_kd", 0) or 0
    second_kd = trend_data.get("second_half_kd", 0) or 0
    change_pct = trend_data.get("kd_change_pct", 0) or 0
    trend = trend_data.get("trend", "stable")

    # Couleurs selon la tendance
    if trend == "improving":
        delta_color = PERFORMANCE_COLORS["trend_up"]
        trend_symbol = "▲"
        trend_text = viz_t("label_improving", lang)
    elif trend == "declining":
        delta_color = PERFORMANCE_COLORS["trend_down"]
        trend_symbol = "▼"
        trend_text = viz_t("label_declining", lang)
    else:
        delta_color = PERFORMANCE_COLORS["baseline"]
        trend_symbol = "◆"
        trend_text = viz_t("label_stable", lang)

    # Créer les indicateurs côte à côte
    fig = make_subplots(
        rows=1,
        cols=3,
        specs=[[{"type": "indicator"}, {"type": "indicator"}, {"type": "indicator"}]],
        subplot_titles=[
            viz_t("label_session_start", lang),
            viz_t("label_session_end", lang),
            viz_t("trace_trend", lang),
        ],
    )

    # Indicateur première moitié
    fig.add_trace(
        go.Indicator(
            mode="number",
            value=first_kd,
            number={
                "font": {"size": 40, "color": PERFORMANCE_COLORS["neutral"]},
                "suffix": "",
                "valueformat": ".2f",
            },
            title={"text": "K/D", "font": {"size": 14}},
        ),
        row=1,
        col=1,
    )

    # Indicateur seconde moitié
    fig.add_trace(
        go.Indicator(
            mode="number",
            value=second_kd,
            number={
                "font": {"size": 40, "color": PERFORMANCE_COLORS["kd_line"]},
                "suffix": "",
                "valueformat": ".2f",
            },
            title={"text": "K/D", "font": {"size": 14}},
        ),
        row=1,
        col=2,
    )

    # Indicateur de tendance avec delta
    fig.add_trace(
        go.Indicator(
            mode="number+delta",
            value=change_pct,
            number={
                "font": {"size": 32, "color": delta_color},
                "suffix": "%",
                "valueformat": "+.1f",
            },
            delta={
                "reference": 0,
                "relative": False,
                "valueformat": ".1f",
                "increasing": {"color": PERFORMANCE_COLORS["trend_up"]},
                "decreasing": {"color": PERFORMANCE_COLORS["trend_down"]},
            },
            title={"text": f"{trend_symbol} {trend_text}", "font": {"size": 14}},
        ),
        row=1,
        col=3,
    )

    return apply_halo_plot_style(fig, title=title, height=height)


def plot_cumulative_comparison(
    session_a_df: pl.DataFrame,
    session_b_df: pl.DataFrame,
    *,
    label_a: str = "Session A",
    label_b: str = "Session B",
    title: str | None = None,
    height: int = 400,
    lang: str = "fr",
) -> go.Figure:
    """Compare deux sessions avec leurs courbes de net score cumulé.

    Args:
        session_a_df: DataFrame de la première session.
        session_b_df: DataFrame de la seconde session.
        label_a: Label pour la session A.
        label_b: Label pour la session B.
        title: Titre du graphique.
        height: Hauteur en pixels.

    Returns:
        Figure Plotly avec les deux courbes superposées.
    """
    fig = go.Figure()

    if title is None:
        title = viz_t("title_session_comparison", lang)

    # Import local
    from src.analysis.cumulative import compute_cumulative_net_score_series_polars

    # Calculer les séries cumulées
    def add_session_trace(df: pl.DataFrame, label: str, color: str) -> None:
        if df.is_empty():
            return

        cumul = compute_cumulative_net_score_series_polars(df)
        if cumul.is_empty():
            return

        data = cumul.to_dicts()
        # Normaliser l'index des matchs (0, 1, 2, ...)
        x_values = list(range(len(data)))
        y_values = [d.get("cumulative_net_score", 0) for d in data]

        fig.add_trace(
            go.Scatter(
                x=x_values,
                y=y_values,
                mode="lines+markers",
                name=label,
                line={"color": color, "width": 2},
                marker={"size": 6, "color": color},
                hovertemplate=viz_t("hover_match_cumul", lang, label=label),
            )
        )

    add_session_trace(session_a_df, label_a, PERFORMANCE_COLORS["neutral"])
    add_session_trace(session_b_df, label_b, PERFORMANCE_COLORS["kd_line"])

    # Ligne de référence
    fig.add_hline(
        y=0,
        line_dash="dash",
        line_color=PERFORMANCE_COLORS["baseline"],
        annotation_text=viz_t("label_balance", lang),
        annotation_position="right",
    )

    fig.update_layout(
        xaxis_title=viz_t("axis_match_number", lang),
        yaxis_title=viz_t("axis_cumul_net_score", lang),
        hovermode="x unified",
        showlegend=True,
        legend={"orientation": "h", "yanchor": "bottom", "y": 1.02, "xanchor": "center", "x": 0.5},
    )

    return apply_halo_plot_style(fig, title=title, height=height)


# =============================================================================
# Indicateurs de performance
# =============================================================================


def create_cumulative_metrics_indicator(
    metrics: Any,  # CumulativeMetricsResult
    *,
    show_trend: bool = True,
    height: int = 150,
    lang: str = "fr",
) -> go.Figure:
    """Crée un indicateur compact des métriques cumulées.

    Args:
        metrics: CumulativeMetricsResult avec les métriques.
        show_trend: Afficher la tendance si disponible.
        height: Hauteur en pixels.

    Returns:
        Figure Plotly avec les indicateurs.
    """
    fig = make_subplots(
        rows=1,
        cols=4,
        specs=[[{"type": "indicator"}] * 4],
        subplot_titles=[
            viz_t("trace_kills", lang),
            viz_t("trace_deaths", lang),
            viz_t("trace_net_score_match", lang),
            viz_t("label_kd_fm", lang),
        ],
    )

    # Kills
    fig.add_trace(
        go.Indicator(
            mode="number",
            value=metrics.total_kills if hasattr(metrics, "total_kills") else 0,
            number={
                "font": {"size": 28, "color": PERFORMANCE_COLORS["kills"]},
            },
        ),
        row=1,
        col=1,
    )

    # Deaths
    fig.add_trace(
        go.Indicator(
            mode="number",
            value=metrics.total_deaths if hasattr(metrics, "total_deaths") else 0,
            number={
                "font": {"size": 28, "color": PERFORMANCE_COLORS["deaths"]},
            },
        ),
        row=1,
        col=2,
    )

    # Net Score
    net_score = metrics.cumulative_net_score if hasattr(metrics, "cumulative_net_score") else 0
    net_color = PERFORMANCE_COLORS["positive"] if net_score >= 0 else PERFORMANCE_COLORS["negative"]
    fig.add_trace(
        go.Indicator(
            mode="number",
            value=net_score,
            number={
                "font": {"size": 28, "color": net_color},
                "valueformat": "+d",
            },
        ),
        row=1,
        col=3,
    )

    # K/D
    kd = metrics.cumulative_kd if hasattr(metrics, "cumulative_kd") else 0
    fig.add_trace(
        go.Indicator(
            mode="number",
            value=kd,
            number={
                "font": {"size": 28, "color": PERFORMANCE_COLORS["kd_line"]},
                "valueformat": ".2f",
            },
        ),
        row=1,
        col=4,
    )

    return apply_halo_plot_style(fig, height=height)


# =============================================================================
# Fonctions utilitaires
# =============================================================================


def get_performance_colors() -> dict[str, str]:
    """Retourne le dictionnaire des couleurs de performance.

    Returns:
        Dict avec les couleurs configurées.
    """
    return PERFORMANCE_COLORS.copy()
