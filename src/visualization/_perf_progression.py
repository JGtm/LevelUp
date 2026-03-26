"""Graphiques de progression avancée — EWMA, IC 90 %, net score/heure.

Extraits de _perf_cumulative et _perf_session pour respecter la limite de 500L.
Contient aussi le helper _add_outcome_markers partagé.
"""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl

from src.config import THEME_COLORS
from src.ui.i18n.viz import viz_t
from src.visualization._perf_cumulative import PERFORMANCE_COLORS
from src.visualization.theme import apply_halo_plot_style, iqr_yrange

# =============================================================================
# Helper partagé — marqueurs résultat (V/D) sur l'axe X
# =============================================================================


def _add_outcome_markers(
    fig: go.Figure,
    x_values: list,
    outcome_values: list[int | None] | None,
    lang: str = "fr",
) -> None:
    """Ajoute des marqueurs V/D/Égalité en bas de figure sur un axe secondaire.

    Les symboles (▲ vert = victoire, ▼ rouge = défaite, ◆ gris = autre) sont
    tracés à y2=0.03, dans les 3 % inférieurs de la zone de tracé, sans
    perturber l'axe principal.

    Args:
        fig: Figure Plotly à enrichir (modifiée en place).
        x_values: Liste des valeurs X (dates ou indices).
        outcome_values: Liste d'outcomes (entiers Outcome enum) ou None.
        lang: Langue pour les libellés hover.
    """
    if not outcome_values or not x_values or len(outcome_values) != len(x_values):
        return

    from src.data.domain.refdata import Outcome

    symbols: list[str] = []
    colors: list[str] = []
    hover_texts: list[str] = []

    win_label = viz_t("label_win", lang)
    loss_label = viz_t("label_loss", lang)
    tie_label = viz_t("label_tie", lang)

    for v in outcome_values:
        if v == Outcome.WIN:
            symbols.append("triangle-up")
            colors.append(PERFORMANCE_COLORS["trend_up"])
            hover_texts.append(win_label)
        elif v == Outcome.LOSS:
            symbols.append("triangle-down")
            colors.append(PERFORMANCE_COLORS["trend_down"])
            hover_texts.append(loss_label)
        else:
            symbols.append("diamond")
            colors.append(PERFORMANCE_COLORS["baseline"])
            hover_texts.append(tie_label)

    fig.add_trace(
        go.Scatter(
            x=x_values,
            y=[0.03] * len(x_values),
            mode="markers",
            yaxis="y2",
            marker={"symbol": symbols, "color": colors, "size": 8},
            name=viz_t("trace_outcome_markers", lang),
            hovertemplate="%{text}<extra></extra>",
            text=hover_texts,
            showlegend=False,
        )
    )
    fig.update_layout(
        yaxis2={
            "range": [0, 1],
            "overlaying": "y",
            "showticklabels": False,
            "showgrid": False,
            "showline": False,
            "zeroline": False,
            "fixedrange": True,
        }
    )


# =============================================================================
# K/D cumulé + intervalle de confiance 90 %
# =============================================================================


def _add_kd_ci_traces(  # noqa: PLR0913
    fig: go.Figure,
    x_values: list,
    y_cumul: list,
    y_upper: list,
    y_lower: list,
    y_match: list,
    lang: str,
) -> None:
    """Ajoute bande IC, points match et courbe K/D cumulé sur `fig`."""
    fig.add_trace(
        go.Scatter(
            x=x_values + list(reversed(x_values)),
            y=y_upper + list(reversed(y_lower)),
            fill="toself",
            fillcolor="rgba(86,180,233,0.18)",
            line={"color": "rgba(0,0,0,0)"},
            name=viz_t("trace_kd_ci_band", lang),
            hoverinfo="skip",
            showlegend=True,
        )
    )
    fig.add_trace(
        go.Scatter(
            x=x_values,
            y=y_match,
            mode="markers",
            name=viz_t("trace_kd_match", lang),
            marker={
                "size": 7,
                "color": PERFORMANCE_COLORS["neutral"],
                "opacity": 0.5,
                "symbol": "circle-open",
            },
            hovertemplate=viz_t("hover_kd_match", lang),
        )
    )
    fig.add_trace(
        go.Scatter(
            x=x_values,
            y=y_cumul,
            mode="lines+markers",
            name=viz_t("trace_kd_cumul", lang),
            line={"color": PERFORMANCE_COLORS["kd_line"], "width": 3},
            marker={"size": 7, "color": PERFORMANCE_COLORS["kd_line"]},
            hovertemplate=viz_t("hover_kd_cumul_line", lang),
        )
    )


def plot_cumulative_kd_with_ci(
    cumul_ci_df: pl.DataFrame,
    *,
    title: str | None = None,
    height: int = 400,
    outcome_values: list[int | None] | None = None,
    lang: str = "fr",
) -> go.Figure:
    """Graphique du K/D cumulé avec bande d'intervalle de confiance à 90 %.

    La bande ombragée se rétrécit avec le nombre de matchs rencontrés.

    Args:
        cumul_ci_df: DataFrame issu de compute_cumulative_kd_with_ci.
        title: Titre du graphique (auto si None).
        height: Hauteur en pixels.
        outcome_values: Liste d'Outcome pour les marqueurs V/D (optionnel).
        lang: Langue.
    """
    if title is None:
        title = viz_t("title_cumul_kd_ci", lang)
    fig = go.Figure()

    if cumul_ci_df.is_empty():
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

    data = cumul_ci_df.to_dicts()
    x_values = [d.get("start_time", "") for d in data]
    _add_kd_ci_traces(
        fig,
        x_values,
        y_cumul=[d.get("cumulative_kd", 0.0) for d in data],
        y_upper=[d.get("ci_upper", 0.0) for d in data],
        y_lower=[d.get("ci_lower", 0.0) for d in data],
        y_match=[d.get("kd", 0.0) for d in data],
        lang=lang,
    )
    fig.add_hline(
        y=1.0,
        line_dash="dash",
        line_color=PERFORMANCE_COLORS["baseline"],
        annotation_text=viz_t("label_target", lang, value=1.0),
        annotation_position="right",
    )
    fig.update_layout(
        yaxis_title=viz_t("axis_kd_ratio", lang),
        xaxis_title="Match",
        hovermode="x unified",
        showlegend=True,
        legend={"orientation": "h", "yanchor": "bottom", "y": 1.02, "xanchor": "center", "x": 0.5},
    )
    _add_outcome_markers(fig, x_values, outcome_values, lang=lang)
    return apply_halo_plot_style(fig, title=title, height=height)


# =============================================================================
# Net score par heure (rolling adaptatif)
# =============================================================================


def _add_nph_traces(
    fig: go.Figure,
    x_values: list,
    y_raw: list,
    y_rolling: list,
    lang: str,
) -> None:
    """Ajoute les 4 traces net score/heure : zones ± et courbes brute/rolling."""
    y_pos = [max(0.0, v) if v is not None else 0.0 for v in y_rolling]
    y_neg = [min(0.0, v) if v is not None else 0.0 for v in y_rolling]
    fig.add_trace(
        go.Scatter(
            x=x_values,
            y=y_pos,
            fill="tozeroy",
            fillcolor="rgba(0,158,115,0.25)",
            line={"color": "rgba(0,0,0,0)"},
            name=viz_t("trace_nph_positive", lang),
            hoverinfo="skip",
            showlegend=False,
        )
    )
    fig.add_trace(
        go.Scatter(
            x=x_values,
            y=y_neg,
            fill="tozeroy",
            fillcolor="rgba(213,94,0,0.25)",
            line={"color": "rgba(0,0,0,0)"},
            name=viz_t("trace_nph_negative", lang),
            hoverinfo="skip",
            showlegend=False,
        )
    )
    fig.add_trace(
        go.Scatter(
            x=x_values,
            y=y_raw,
            mode="markers",
            name=viz_t("trace_nph_raw", lang),
            marker={"size": 5, "color": PERFORMANCE_COLORS["neutral"], "opacity": 0.35},
            hovertemplate=viz_t("hover_nph_raw", lang),
        )
    )
    fig.add_trace(
        go.Scatter(
            x=x_values,
            y=y_rolling,
            mode="lines+markers",
            name=viz_t("trace_nph_rolling", lang),
            line={"color": PERFORMANCE_COLORS["cumulative"], "width": 3},
            marker={"size": 6, "color": PERFORMANCE_COLORS["cumulative"]},
            hovertemplate=viz_t("hover_nph_rolling", lang),
        )
    )


def plot_net_score_per_hour(
    nph_df: pl.DataFrame,
    *,
    title: str | None = None,
    height: int = 350,
    outcome_values: list[int | None] | None = None,
    lang: str = "fr",
) -> go.Figure:
    """Graphique du net score normalisé par heure de jeu (rolling).

    Contrairement au net score brut cumulé (toujours croissant si on joue assez),
    cette courbe mesure l'*efficacité* : haute = frags/heure élevés.

    Args:
        nph_df: DataFrame issu de compute_rolling_net_score_per_hour.
        title: Titre (auto si None).
        height: Hauteur en pixels.
        outcome_values: Liste d'Outcome pour les marqueurs V/D.
        lang: Langue.
    """
    if title is None:
        title = viz_t("title_net_score_per_hour", lang)
    fig = go.Figure()

    if nph_df.is_empty():
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

    data = nph_df.sort("start_time").to_dicts()
    x_values = [d.get("start_time", "") for d in data]
    y_raw = [d.get("net_per_hour", 0.0) for d in data]
    y_rolling = [d.get("rolling_net_per_hour") for d in data]
    _add_nph_traces(fig, x_values, y_raw=y_raw, y_rolling=y_rolling, lang=lang)
    fig.add_hline(
        y=0,
        line_width=2,
        line_color="rgba(255,255,255,0.75)",
        annotation_text=viz_t("label_balance", lang),
        annotation_position="right",
    )
    y_range = iqr_yrange(y_raw + [v for v in y_rolling if v is not None])
    layout: dict = {
        "yaxis_title": viz_t("axis_net_per_hour", lang),
        "xaxis_title": "Match",
        "hovermode": "x unified",
        "showlegend": True,
        "legend": {
            "orientation": "h",
            "yanchor": "bottom",
            "y": 1.02,
            "xanchor": "center",
            "x": 0.5,
        },
    }
    if y_range:
        layout["yaxis_range"] = list(y_range)
    fig.update_layout(**layout)
    _add_outcome_markers(fig, x_values, outcome_values, lang=lang)
    return apply_halo_plot_style(fig, title=title, height=height)


# =============================================================================
# K/D EWMA + droite de régression
# =============================================================================


def _add_ewma_traces(  # noqa: PLR0913
    fig: go.Figure,
    x_values: list,
    y_kd: list,
    y_ewma: list,
    regression_data: dict | None,
    lang: str,
) -> None:
    """Ajoute K/D brut, courbe EWMA et droite de régression (si significative)."""
    fig.add_trace(
        go.Scatter(
            x=x_values,
            y=y_kd,
            mode="lines",
            name=viz_t("trace_kd_match", lang),
            line={"color": PERFORMANCE_COLORS["neutral"], "width": 1},
            opacity=0.35,
            hovertemplate=viz_t("hover_kd_match", lang),
        )
    )
    fig.add_trace(
        go.Scatter(
            x=x_values,
            y=y_ewma,
            mode="lines+markers",
            name=viz_t("trace_kd_ewma", lang),
            line={"color": PERFORMANCE_COLORS["rolling"], "width": 3},
            marker={"size": 6, "color": PERFORMANCE_COLORS["rolling"]},
            hovertemplate=viz_t("hover_kd_ewma", lang),
        )
    )
    if regression_data and regression_data.get("is_significant") and regression_data.get("y_hat"):
        trend = regression_data.get("trend", "stable")
        reg_color = (
            PERFORMANCE_COLORS["trend_up"]
            if trend == "improving"
            else PERFORMANCE_COLORS["trend_down"]
            if trend == "declining"
            else PERFORMANCE_COLORS["baseline"]
        )
        fig.add_trace(
            go.Scatter(
                x=x_values,
                y=regression_data["y_hat"],
                mode="lines",
                name=viz_t("trace_regression_line", lang),
                line={"color": reg_color, "width": 2, "dash": "dot"},
                hoverinfo="skip",
                showlegend=True,
            )
        )


def plot_ewma_kd(  # noqa: PLR0913
    ewma_df: pl.DataFrame,
    *,
    title: str | None = None,
    height: int = 400,
    regression_data: dict | None = None,
    outcome_values: list[int | None] | None = None,
    lang: str = "fr",
) -> go.Figure:
    """Graphique du K/D lissé (EWMA) avec droite de régression optionnelle.

    L'EWMA donne plus de poids aux matchs récents : amélioration visible dès
    3 parties. La droite de tendance (si R² ≥ 0.3) confirme la direction.

    Args:
        ewma_df: DataFrame issu de compute_ewma_kd_polars.
        title: Titre (auto si None).
        height: Hauteur en pixels.
        regression_data: Dict issu de compute_linear_regression_kd (optionnel).
        outcome_values: Liste d'Outcome pour les marqueurs V/D.
        lang: Langue.
    """
    if title is None:
        title = viz_t("title_ewma_kd", lang)
    fig = go.Figure()

    if ewma_df.is_empty():
        fig.add_annotation(
            text=viz_t("empty_no_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 14, "color": THEME_COLORS.text_primary},
        )
        return apply_halo_plot_style(fig, title=title, height=height)

    data = ewma_df.sort("start_time").to_dicts()
    x_values = [d.get("start_time", "") for d in data]
    y_kd = [d.get("kd", 0.0) for d in data]
    y_ewma = [d.get("ewma_kd", 0.0) for d in data]
    _add_ewma_traces(
        fig,
        x_values,
        y_kd=y_kd,
        y_ewma=y_ewma,
        regression_data=regression_data,
        lang=lang,
    )
    fig.add_hline(
        y=1.0,
        line_dash="dash",
        line_color=PERFORMANCE_COLORS["baseline"],
        annotation_text=viz_t("label_kd_ref", lang),
        annotation_position="right",
    )
    y_range = iqr_yrange(y_kd + y_ewma, allow_negative=False)
    layout: dict = {
        "yaxis_title": viz_t("axis_kd_ratio", lang),
        "xaxis_title": "Match",
        "hovermode": "x unified",
        "showlegend": True,
        "legend": {
            "orientation": "h",
            "yanchor": "bottom",
            "y": 1.02,
            "xanchor": "center",
            "x": 0.5,
        },
    }
    if y_range:
        layout["yaxis_range"] = list(y_range)
    fig.update_layout(**layout)
    _add_outcome_markers(fig, x_values, outcome_values, lang=lang)
    return apply_halo_plot_style(fig, title=title, height=height)
