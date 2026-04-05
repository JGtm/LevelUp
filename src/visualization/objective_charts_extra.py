"""Graphiques d'objectifs supplémentaires.

Extrait de objective_charts.py pour respecter la limite de 500L.
Contient la constante OBJECTIVE_COLORS ainsi que les fonctions
plot_assist_breakdown_pie, plot_objective_trend_over_time et
get_objective_chart_colors.
"""

from __future__ import annotations

from typing import Any

import plotly.graph_objects as go
import polars as pl

from src.config import THEME_COLORS
from src.ui.i18n.viz import viz_t
from src.visualization.theme import apply_halo_plot_style

# =============================================================================
# Configuration des couleurs (partagée avec objective_charts.py)
# =============================================================================

# Couleurs mises à jour pour respecter la palette Okabe-Ito (accessibilité daltonisme).
# Anciens code hexadécimaux (deuteranopie/protanopie-incompatibles) :
#   objective: #00ffcc (cyan néon)   → #56B4E9 (bleu ciel)
#   kill:      #ff4444 (rouge néon)  → #D55E00 (vermillon)
#   assist:    #ffaa00 (orange chaud) → #E69F00 (orange Okabe-Ito)
#   mode:      #aa66ff (violet)       → #CC79A7 (rose mauve)
#   highlight: #00ff00 (vert néon)   → #F0E442 (jaune)
#   bar_1:     #00aaff (bleu vif)    → #0072B2 (bleu Okabe-Ito)
#   bar_2:     #ff6666 (rouge clair) → #D55E00 (vermillon)
#   bar_3:     #66ff66 (vert clair)  → #009E73 (vert bleuté)
OBJECTIVE_COLORS = {
    "objective": "#56B4E9",  # Bleu ciel Okabe-Ito pour objectifs
    "kill": "#D55E00",  # Vermillon Okabe-Ito pour kills
    "assist": "#E69F00",  # Orange Okabe-Ito pour assists
    "mode": "#CC79A7",  # Rose mauve Okabe-Ito pour mode
    "other": "#999999",  # Gris neutre
    "highlight": "#F0E442",  # Jaune Okabe-Ito pour highlights
    "bar_1": "#0072B2",  # Bleu Okabe-Ito
    "bar_2": "#D55E00",  # Vermillon Okabe-Ito
    "bar_3": "#009E73",  # Vert bleuté Okabe-Ito
}


def _extract_assist_values(
    assist_breakdown: Any,
) -> tuple[int, int, int, int] | None:
    """Extrait les 4 compteurs d'assistances depuis un objet ou dict."""
    if hasattr(assist_breakdown, "kill_assists"):
        return (
            assist_breakdown.kill_assists,
            assist_breakdown.mark_assists,
            assist_breakdown.emp_assists,
            getattr(assist_breakdown, "other_assists", 0),
        )
    if isinstance(assist_breakdown, dict):
        return (
            assist_breakdown.get("kill_assists", 0),
            assist_breakdown.get("mark_assists", 0),
            assist_breakdown.get("emp_assists", 0),
            assist_breakdown.get("other_assists", 0),
        )
    return None


def plot_assist_breakdown_pie(
    assist_breakdown: Any,  # AssistBreakdownResult
    *,
    height: int = 350,
    lang: str = "fr",
) -> go.Figure:
    """Crée un camembert de la répartition des assistances.

    Args:
        assist_breakdown: AssistBreakdownResult avec les compteurs.
        title: Titre du graphique.
        height: Hauteur en pixels.

    Returns:
        Figure Plotly avec le camembert.
    """
    fig = go.Figure()

    assists = _extract_assist_values(assist_breakdown)
    if assists is None:
        fig.add_annotation(
            text=viz_t("empty_no_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
        )
        return apply_halo_plot_style(fig, height=height)

    kill_assists, mark_assists, emp_assists, other_assists = assists

    labels = [
        viz_t("label_kill_assists", lang),
        viz_t("label_mark_assists", lang),
        viz_t("label_emp_assists", lang),
        viz_t("cat_label_other", lang),
    ]
    values = [kill_assists, mark_assists, emp_assists, other_assists]
    colors = [
        OBJECTIVE_COLORS["kill"],
        OBJECTIVE_COLORS["highlight"],
        OBJECTIVE_COLORS["mode"],
        OBJECTIVE_COLORS["other"],
    ]

    # Filtrer les valeurs nulles
    filtered = [(lbl, v, c) for lbl, v, c in zip(labels, values, colors, strict=False) if v > 0]
    if not filtered:
        fig.add_annotation(
            text=viz_t("empty_no_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
        )
        return apply_halo_plot_style(fig, height=height)

    labels, values, colors = zip(*filtered, strict=False)

    fig.add_trace(
        go.Pie(
            labels=list(labels),
            values=list(values),
            marker_colors=list(colors),
            textinfo="label+percent",
            textposition="outside",
            hole=0.4,
            hovertemplate="<b>%{label}</b><br>%{value} (%{percent})<extra></extra>",
        )
    )

    return apply_halo_plot_style(fig, height=height)


def plot_objective_trend_over_time(
    summary_df: pl.DataFrame,
    *,
    height: int = 400,
    lang: str = "fr",
) -> go.Figure:
    """Crée un graphique de l'évolution du score objectifs dans le temps.

    Args:
        summary_df: DataFrame avec colonnes match_id, start_time, objective_score, etc.
        title: Titre du graphique.
        height: Hauteur en pixels.

    Returns:
        Figure Plotly avec la timeseries.
    """
    fig = go.Figure()

    if summary_df.is_empty():
        fig.add_annotation(
            text=viz_t("empty_no_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
            font={"size": 16, "color": THEME_COLORS.text_primary},
        )
        return apply_halo_plot_style(fig, height=height)

    # Trier par date
    df = summary_df.sort("start_time") if "start_time" in summary_df.columns else summary_df
    data = df.to_dicts()

    x_values = [d.get("start_time", d.get("match_id", str(i))) for i, d in enumerate(data)]

    # Score objectifs
    obj_scores = [d.get("objective_score", 0) for d in data]
    fig.add_trace(
        go.Scatter(
            x=x_values,
            y=obj_scores,
            mode="lines+markers",
            name=viz_t("trace_obj_score", lang),
            line={"color": OBJECTIVE_COLORS["objective"], "width": 2},
            marker={"size": 6},
            hovertemplate=viz_t("hover_obj_score_line", lang),
        )
    )

    # Score total si disponible
    if "total_score" in summary_df.columns:
        total_scores = [d.get("total_score", 0) for d in data]
        fig.add_trace(
            go.Scatter(
                x=x_values,
                y=total_scores,
                mode="lines",
                name=viz_t("trace_total_score", lang),
                line={"color": OBJECTIVE_COLORS["other"], "width": 1, "dash": "dot"},
                hovertemplate=viz_t("hover_obj_total_line", lang),
            )
        )

    fig.update_layout(
        xaxis_title=viz_t("axis_match_number", lang),
        yaxis_title=viz_t("axis_score", lang),
        hovermode="x unified",
        showlegend=True,
        legend={"orientation": "h", "yanchor": "bottom", "y": 1.02, "xanchor": "center", "x": 0.5},
    )

    return apply_halo_plot_style(fig, height=height)


def get_objective_chart_colors() -> dict[str, str]:
    """Retourne le dictionnaire des couleurs pour les graphiques d'objectifs.

    Returns:
        Dict avec les couleurs configurées.
    """
    return OBJECTIVE_COLORS.copy()
