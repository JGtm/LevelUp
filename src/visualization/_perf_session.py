"""Graphiques de session et indicateurs de performance.

Extraits de performance.py — regroupe les charts de tendance de session
et les indicateurs de métriques agrégées.
"""

from __future__ import annotations

from typing import Any

import plotly.graph_objects as go
import polars as pl
from plotly.subplots import make_subplots

from src.config import THEME_COLORS
from src.ui.i18n.viz import viz_t
from src.visualization._perf_cumulative import PERFORMANCE_COLORS
from src.visualization.theme import apply_halo_plot_style


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

    first_kd = trend_data.get("first_half_kd", 0) or 0
    second_kd = trend_data.get("second_half_kd", 0) or 0
    change_pct = trend_data.get("kd_change_pct", 0) or 0
    trend = trend_data.get("trend", "stable")

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

    fig.add_trace(
        go.Indicator(
            mode="number",
            value=first_kd,
            number={
                "font": {"size": 40, "color": PERFORMANCE_COLORS["neutral"]},
                "suffix": "",
                "valueformat": ".2f",
            },
            title={"text": viz_t("title_kd_timeline", lang), "font": {"size": 14}},
        ),
        row=1,
        col=1,
    )

    fig.add_trace(
        go.Indicator(
            mode="number",
            value=second_kd,
            number={
                "font": {"size": 40, "color": PERFORMANCE_COLORS["kd_line"]},
                "suffix": "",
                "valueformat": ".2f",
            },
            title={"text": viz_t("title_kd_timeline", lang), "font": {"size": 14}},
        ),
        row=1,
        col=2,
    )

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


def _perf_color(p: float | None, fallback: str) -> str:
    """Retourne la couleur Plotly selon le score de performance."""
    if p is None:
        return fallback
    return "#27AE60" if float(p) >= 70 else "#F39C12" if float(p) >= 45 else "#E74C3C"


def _add_session_trace(  # noqa: PLR0913
    fig: go.Figure,
    df: pl.DataFrame,
    label: str,
    color: str,
    lang: str,
    perf_scores: list[float | None] | None = None,
) -> None:
    """Ajoute la trace du score cumulé d'une session sur la figure."""
    from src.analysis.cumulative import compute_cumulative_net_score_series_polars

    if df.is_empty():
        return
    cumul = compute_cumulative_net_score_series_polars(df)
    if cumul.is_empty():
        return
    data = cumul.to_dicts()
    x_values = list(range(len(data)))
    y_values = [d.get("cumulative_net_score", 0) for d in data]
    n = len(data)

    if perf_scores and len(perf_scores) >= n:
        marker_colors: list[str] | str = [_perf_color(perf_scores[i], color) for i in range(n)]
        customdata = [[f"{float(p):.0f}" if p is not None else "—"] for p in perf_scores[:n]]
        hover = (
            f"<b>{label}</b><br>Match #%{{x}}<br>Cumulé: %{{y:+d}}"
            "<br>⭐ Perf: %{customdata[0]}<extra></extra>"
        )
    else:
        marker_colors = color
        customdata = None
        hover = viz_t("hover_match_cumul", lang, label=label)

    trace_kwargs = {
        "x": x_values,
        "y": y_values,
        "mode": "lines+markers",
        "name": label,
        "line": {"color": color, "width": 2},
        "marker": {"size": 6, "color": marker_colors},
        "hovertemplate": hover,
    }
    if customdata is not None:
        trace_kwargs["customdata"] = customdata
    fig.add_trace(go.Scatter(**trace_kwargs))


def _add_rank_trace(
    fig: go.Figure,
    ranks: list[float | None],
    label: str,
    color: str,
    rank_label: str,
) -> None:
    """Ajoute la trace LUSR/CSR (axe Y secondaire) sur la figure."""
    valid = [(i, r) for i, r in enumerate(ranks) if r is not None and float(r) > 0]
    if not valid:
        return
    fig.add_trace(
        go.Scatter(
            x=[v[0] for v in valid],
            y=[float(v[1]) for v in valid],
            mode="lines+markers",
            name=f"{label} {rank_label}",
            line={"color": color, "width": 1, "dash": "dot"},
            marker={"size": 5, "symbol": "diamond", "color": color},
            yaxis="y2",
            opacity=0.7,
            hovertemplate=f"{label} {rank_label}: %{{y:.0f}}<extra></extra>",
        )
    )


def plot_cumulative_comparison(  # noqa: PLR0913
    session_a_df: pl.DataFrame,
    session_b_df: pl.DataFrame,
    *,
    label_a: str = "Session A",
    label_b: str = "Session B",
    title: str | None = None,
    height: int = 420,
    lang: str = "fr",
    rank_a: list[float | None] | None = None,
    rank_b: list[float | None] | None = None,
    rank_label: str = "CSR",
    perf_a: list[float | None] | None = None,
    perf_b: list[float | None] | None = None,
) -> go.Figure:
    """Compare deux sessions avec leurs courbes de net score cumulé."""
    if title is None:
        title = viz_t("title_session_comparison", lang)

    fig = go.Figure()
    _add_session_trace(fig, session_a_df, label_a, PERFORMANCE_COLORS["neutral"], lang, perf_a)
    _add_session_trace(fig, session_b_df, label_b, PERFORMANCE_COLORS["kd_line"], lang, perf_b)

    has_rank = bool(rank_a or rank_b)
    if rank_a:
        _add_rank_trace(fig, rank_a, label_a, PERFORMANCE_COLORS["neutral"], rank_label)
    if rank_b:
        _add_rank_trace(fig, rank_b, label_b, PERFORMANCE_COLORS["kd_line"], rank_label)

    fig.add_hline(
        y=0,
        line_width=2,
        line_color="rgba(255,255,255,0.75)",
        annotation_text=viz_t("label_balance", lang),
        annotation_position="right",
    )

    layout_kwargs: dict = {
        "xaxis_title": viz_t("axis_match_number", lang),
        "yaxis_title": viz_t("axis_cumul_net_score", lang),
        "hovermode": "x unified",
        "showlegend": True,
        "legend": {"orientation": "h", "yanchor": "top", "y": -0.18, "xanchor": "center", "x": 0.5},
    }
    if has_rank:
        layout_kwargs["yaxis2"] = {
            "title": rank_label,
            "overlaying": "y",
            "side": "right",
            "showgrid": False,
            "rangemode": "tozero",
        }
    fig.update_layout(**layout_kwargs)
    return apply_halo_plot_style(fig, title=title, height=height)


def create_cumulative_metrics_indicator(
    metrics: Any,  # CumulativeMetricsResult
    *,
    show_trend: bool = True,
    height: int = 150,
    lang: str = "fr",
) -> go.Figure:
    """Crée un indicateur compact des métriques cumulées."""
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


def get_performance_colors() -> dict[str, str]:
    """Retourne le dictionnaire des couleurs de performance."""
    return PERFORMANCE_COLORS.copy()


# =============================================================================
# Indicateurs de régression (pente K/D, win rate, R²)
# =============================================================================


def _kd_progression_indicator(kd_relative_change: float, trend: str, lang: str) -> go.Indicator:
    """Indicateur de progression F/D sur la session (variation relative en %)."""
    if trend == "improving":
        color = PERFORMANCE_COLORS["trend_up"]
    elif trend == "declining":
        color = PERFORMANCE_COLORS["trend_down"]
    else:
        color = PERFORMANCE_COLORS["baseline"]
    pct = max(-200.0, min(200.0, round(kd_relative_change * 100, 1)))
    return go.Indicator(
        mode="number+delta",
        value=pct,
        number={
            "font": {"size": 32, "color": color},
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
        title={"text": viz_t("label_kd_slope", lang), "font": {"size": 13}},
    )


def _r2_indicator(r_sq: float, is_sig: bool, lang: str) -> go.Indicator:
    """Retourne l'indicateur Plotly pour la qualité d'ajustement R²."""
    color = PERFORMANCE_COLORS["kd_line"] if is_sig else PERFORMANCE_COLORS["baseline"]
    suffix = "" if is_sig else f" {viz_t('label_not_significant', lang)}"
    return go.Indicator(
        mode="number",
        value=round(r_sq, 2),
        number={"font": {"size": 32, "color": color}, "suffix": suffix, "valueformat": ".0%"},
        title={"text": viz_t("label_r_squared", lang), "font": {"size": 13}},
    )


def _wr_progression_indicator(wr_relative_change: float, lang: str) -> go.Indicator:
    """Indicateur de progression taux de victoires sur la session (variation absolue en pp)."""
    if wr_relative_change > 0.05:
        color = PERFORMANCE_COLORS["trend_up"]
    elif wr_relative_change < -0.05:
        color = PERFORMANCE_COLORS["trend_down"]
    else:
        color = PERFORMANCE_COLORS["baseline"]
    pct = max(-100.0, min(100.0, round(wr_relative_change * 100, 1)))
    return go.Indicator(
        mode="number+delta",
        value=pct,
        number={
            "font": {"size": 32, "color": color},
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
        title={"text": viz_t("label_win_rate_slope", lang), "font": {"size": 13}},
    )


def plot_regression_trend(
    regression_data: dict,
    *,
    title: str | None = None,
    height: int = 160,
    lang: str = "fr",
) -> go.Figure:
    """Indicateurs compacts : pente K/D, pente win rate et R².

    Remplace la comparaison début/fin par une régression sur tous les matchs.
    R² < 0.3 est signalé ⚠ "tendance non significative".

    Args:
        regression_data: Dict issu de compute_linear_regression_kd.
        title: Titre (auto si None).
        height: Hauteur en pixels.
        lang: Langue.
    """
    if title is None:
        title = viz_t("title_regression_trend", lang)

    slope = regression_data.get("slope", 0.0) or 0.0
    r_sq = regression_data.get("r_squared", 0.0) or 0.0
    is_sig = regression_data.get("is_significant", False)
    trend = regression_data.get("trend", "stable")
    kd_relative_change = regression_data.get("kd_relative_change")
    wr_relative_change = regression_data.get("wr_relative_change")

    n_cols = 3 if wr_relative_change is not None else 2
    fig = make_subplots(rows=1, cols=n_cols, specs=[[{"type": "indicator"}] * n_cols])

    if kd_relative_change is not None:
        fig.add_trace(_kd_progression_indicator(kd_relative_change, trend, lang), row=1, col=1)
    else:
        fig.add_trace(_kd_progression_indicator(slope * 10, trend, lang), row=1, col=1)
    fig.add_trace(_r2_indicator(r_sq, is_sig, lang), row=1, col=2)
    if wr_relative_change is not None:
        fig.add_trace(_wr_progression_indicator(wr_relative_change, lang), row=1, col=n_cols)

    # Marges serrées : le titre est géré côté Streamlit (st.markdown) pour
    # éviter tout débordement avec height=160.
    fig.update_layout(margin={"l": 10, "r": 10, "t": 10, "b": 10})
    return apply_halo_plot_style(fig, title=None, height=height)
