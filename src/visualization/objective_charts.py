"""Graphiques d'analyse des objectifs avec Plotly.

Sprint 7: Visualisations pour la participation aux objectifs et la contribution des joueurs.
"""

from __future__ import annotations

from typing import Any

import plotly.graph_objects as go
import polars as pl

from src.config import THEME_COLORS
from src.ui.i18n.viz import viz_t
from src.visualization.theme import apply_halo_plot_style

# =============================================================================
# Configuration des couleurs
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


# =============================================================================
# Graphique: Score objectifs vs Kills
# =============================================================================


def plot_objective_vs_kills_scatter(
    awards_df: pl.DataFrame,
    match_stats_df: pl.DataFrame,
    *,
    title: str | None = None,
    height: int = 450,
    lang: str = "fr",
) -> go.Figure:
    """Crée un scatter plot comparant score objectifs et kills par match.

    Permet d'identifier les matchs où le joueur a contribué plus aux objectifs
    qu'aux kills (joueur support) ou l'inverse.

    Args:
        awards_df: DataFrame des personal_score_awards.
        match_stats_df: DataFrame des match_stats avec kills.
        title: Titre du graphique.
        height: Hauteur en pixels.

    Returns:
        Figure Plotly avec le scatter plot.
    """
    title = title or viz_t("title_obj_vs_kills", lang)
    fig = go.Figure()

    if awards_df.is_empty() or match_stats_df.is_empty():
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

    # Calculer score objectifs par match
    objective_categories = ["objective", "mode"]
    obj_by_match = (
        awards_df.filter(pl.col("score_category").is_in(objective_categories))
        .group_by("match_id")
        .agg(pl.col("points").sum().alias("objective_score"))
    )

    # Joindre avec match_stats pour avoir les kills
    combined = (
        match_stats_df.select(["match_id", "kills", "map_name", "start_time"])
        .join(obj_by_match, on="match_id", how="left")
        .with_columns([pl.col("objective_score").fill_null(0)])
    )

    if combined.is_empty():
        fig.add_annotation(
            text=viz_t("empty_no_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
        )
        return apply_halo_plot_style(fig, title=title, height=height)

    data = combined.to_dicts()

    # Scatter plot
    fig.add_trace(
        go.Scatter(
            x=[d.get("kills", 0) for d in data],
            y=[d.get("objective_score", 0) for d in data],
            mode="markers",
            marker={
                "size": 12,
                "color": OBJECTIVE_COLORS["objective"],
                "opacity": 0.7,
                "line": {"width": 1, "color": "white"},
            },
            text=[d.get("map_name", "?") for d in data],
            customdata=[[d.get("start_time", ""), d.get("match_id", "")] for d in data],
            hovertemplate=viz_t("hover_obj_kills_score", lang),
            name=viz_t("trace_matches", lang),
        )
    )

    # Ligne de tendance (régression linéaire simple)
    if len(data) > 2:
        kills_list = [d.get("kills", 0) for d in data]
        obj_list = [d.get("objective_score", 0) for d in data]

        # Calcul simple de la régression
        n = len(kills_list)
        sum_x = sum(kills_list)
        sum_y = sum(obj_list)
        sum_xy = sum(k * o for k, o in zip(kills_list, obj_list, strict=False))
        sum_x2 = sum(k**2 for k in kills_list)

        denom = n * sum_x2 - sum_x**2
        if denom != 0:
            slope = (n * sum_xy - sum_x * sum_y) / denom
            intercept = (sum_y - slope * sum_x) / n

            x_range = [min(kills_list), max(kills_list)]
            y_trend = [slope * x + intercept for x in x_range]

            fig.add_trace(
                go.Scatter(
                    x=x_range,
                    y=y_trend,
                    mode="lines",
                    line={"color": OBJECTIVE_COLORS["highlight"], "dash": "dash"},
                    name=viz_t("trace_trend", lang),
                    showlegend=True,
                )
            )

    fig.update_layout(
        xaxis_title=viz_t("axis_kills", lang),
        yaxis_title=viz_t("trace_obj_score", lang),
        hovermode="closest",
    )

    return apply_halo_plot_style(fig, title=title, height=height)


def plot_objective_breakdown_bars(
    awards_df: pl.DataFrame,
    *,
    xuid: str | None = None,
    title: str | None = None,
    height: int = 400,
    lang: str = "fr",
) -> go.Figure:
    """Crée un graphique en barres de la répartition du score par catégorie.

    Args:
        awards_df: DataFrame des personal_score_awards.
        xuid: XUID du joueur à filtrer (optionnel).
        title: Titre du graphique.
        height: Hauteur en pixels.

    Returns:
        Figure Plotly avec les barres.
    """
    title = title or viz_t("title_score_by_category", lang)
    fig = go.Figure()

    if awards_df.is_empty():
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

    df = awards_df
    if xuid is not None and "xuid" in df.columns:
        df = df.filter(pl.col("xuid") == xuid)

    # Agréger par catégorie
    by_category = (
        df.group_by("score_category")
        .agg(
            [
                pl.col("points").sum().alias("total_points"),
                pl.len().alias("count"),
            ]
        )
        .sort("total_points", descending=True)
    )

    if by_category.is_empty():
        fig.add_annotation(
            text=viz_t("empty_no_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
        )
        return apply_halo_plot_style(fig, title=title, height=height)

    data = by_category.to_dicts()

    # Mapping des couleurs par catégorie
    category_colors = {
        "objective": OBJECTIVE_COLORS["objective"],
        "mode": OBJECTIVE_COLORS["mode"],
        "kill": OBJECTIVE_COLORS["kill"],
        "assist": OBJECTIVE_COLORS["assist"],
    }

    categories = [d["score_category"] for d in data]
    points = [d["total_points"] for d in data]
    counts = [d["count"] for d in data]
    colors = [category_colors.get(c, OBJECTIVE_COLORS["other"]) for c in categories]

    fig.add_trace(
        go.Bar(
            x=categories,
            y=points,
            marker_color=colors,
            text=[f"{p:,.0f}" for p in points],
            textposition="outside",
            customdata=counts,
            hovertemplate=(
                "<b>%{x}</b><br>Points: %{y:,.0f}<br>Occurrences: %{customdata}<extra></extra>"
            ),
        )
    )

    fig.update_layout(
        xaxis_title=viz_t("axis_category", lang),
        yaxis_title=viz_t("axis_total_points", lang),
        showlegend=False,
    )

    return apply_halo_plot_style(fig, title=title, height=height)


def plot_top_players_objective_bars(
    rankings: list[Any],  # list[PlayerObjectiveRanking]
    *,
    top_n: int = 10,
    title: str | None = None,
    height: int = 450,
    lang: str = "fr",
) -> go.Figure:
    """Crée un graphique des top joueurs par contribution aux objectifs.

    Args:
        rankings: Liste de PlayerObjectiveRanking.
        top_n: Nombre de joueurs à afficher.
        title: Titre du graphique.
        height: Hauteur en pixels.

    Returns:
        Figure Plotly avec les barres horizontales.
    """
    title = title or viz_t("title_top_players_obj", lang)
    fig = go.Figure()

    if not rankings:
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

    # Limiter au top N
    top_rankings = rankings[:top_n]

    # Extraire les données
    gamertags = [
        getattr(r, "gamertag", r.get("gamertag", "?"))
        if hasattr(r, "gamertag") or isinstance(r, dict)
        else str(r)
        for r in top_rankings
    ]
    scores = [
        getattr(r, "total_objective_score", r.get("total_objective_score", 0))
        if hasattr(r, "total_objective_score") or isinstance(r, dict)
        else 0
        for r in top_rankings
    ]
    matches = [
        getattr(r, "matches_count", r.get("matches_count", 0))
        if hasattr(r, "matches_count") or isinstance(r, dict)
        else 0
        for r in top_rankings
    ]

    # Inverser pour avoir le meilleur en haut
    gamertags = gamertags[::-1]
    scores = scores[::-1]
    matches = matches[::-1]

    fig.add_trace(
        go.Bar(
            y=gamertags,
            x=scores,
            orientation="h",
            marker_color=OBJECTIVE_COLORS["objective"],
            text=[f"{s:,.0f}" for s in scores],
            textposition="outside",
            customdata=matches,
            hovertemplate=viz_t("hover_obj_leaderboard", lang),
        )
    )

    fig.update_layout(
        xaxis_title=viz_t("trace_obj_score", lang),
        yaxis_title="",
        showlegend=False,
    )

    return apply_halo_plot_style(fig, title=title, height=height)


def plot_objective_ratio_gauge(
    ratio: float,
    *,
    title: str | None = None,
    height: int = 250,
    lang: str = "fr",
) -> go.Figure:
    """Crée un indicateur gauge pour le ratio objectifs/total.

    Args:
        ratio: Ratio entre 0 et 1 (objectifs / total).
        title: Titre du graphique.
        height: Hauteur en pixels.

    Returns:
        Figure Plotly avec l'indicateur.
    """
    title = title or viz_t("title_obj_ratio_pct", lang)
    # Convertir en pourcentage
    percentage = ratio * 100

    fig = go.Figure()

    fig.add_trace(
        go.Indicator(
            mode="gauge+number",
            value=percentage,
            number={"suffix": "%", "font": {"size": 32}},
            gauge={
                "axis": {"range": [0, 100], "tickwidth": 1},
                "bar": {"color": OBJECTIVE_COLORS["objective"]},
                "bgcolor": "rgba(0,0,0,0.3)",
                "borderwidth": 2,
                "bordercolor": "gray",
                "steps": [
                    {"range": [0, 33], "color": "rgba(255,68,68,0.3)"},  # Faible
                    {"range": [33, 66], "color": "rgba(255,170,0,0.3)"},  # Moyen
                    {"range": [66, 100], "color": "rgba(0,255,136,0.3)"},  # Élevé
                ],
                "threshold": {
                    "line": {"color": OBJECTIVE_COLORS["highlight"], "width": 4},
                    "thickness": 0.75,
                    "value": percentage,
                },
            },
            title={"text": title, "font": {"size": 16}},
        )
    )

    return apply_halo_plot_style(fig, height=height)


def plot_assist_breakdown_pie(
    assist_breakdown: Any,  # AssistBreakdownResult
    *,
    title: str | None = None,
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
    title = title or viz_t("title_assist_breakdown", lang)
    fig = go.Figure()

    # Extraire les données selon le type
    if hasattr(assist_breakdown, "kill_assists"):
        kill_assists = assist_breakdown.kill_assists
        mark_assists = assist_breakdown.mark_assists
        emp_assists = assist_breakdown.emp_assists
        other_assists = getattr(assist_breakdown, "other_assists", 0)
    elif isinstance(assist_breakdown, dict):
        kill_assists = assist_breakdown.get("kill_assists", 0)
        mark_assists = assist_breakdown.get("mark_assists", 0)
        emp_assists = assist_breakdown.get("emp_assists", 0)
        other_assists = assist_breakdown.get("other_assists", 0)
    else:
        fig.add_annotation(
            text=viz_t("empty_no_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
        )
        return apply_halo_plot_style(fig, title=title, height=height)

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
        return apply_halo_plot_style(fig, title=title, height=height)

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

    return apply_halo_plot_style(fig, title=title, height=height)


def plot_objective_trend_over_time(
    summary_df: pl.DataFrame,
    *,
    title: str | None = None,
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
    title = title or viz_t("title_obj_trend", lang)
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
        return apply_halo_plot_style(fig, title=title, height=height)

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

    return apply_halo_plot_style(fig, title=title, height=height)


# =============================================================================
# Fonctions utilitaires
# =============================================================================


def get_objective_chart_colors() -> dict[str, str]:
    """Retourne le dictionnaire des couleurs pour les graphiques d'objectifs.

    Returns:
        Dict avec les couleurs configurées.
    """
    return OBJECTIVE_COLORS.copy()
