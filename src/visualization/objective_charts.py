"""Graphiques d'analyse des objectifs avec Plotly.

Sprint 7: Visualisations pour la participation aux objectifs et la contribution des joueurs.
"""

from __future__ import annotations

from typing import Any

import plotly.graph_objects as go
import polars as pl

from src.config import THEME_COLORS
from src.ui.i18n.viz import viz_t
from src.visualization.objective_charts_extra import (  # noqa: F401
    OBJECTIVE_COLORS,
    get_objective_chart_colors,
    plot_assist_breakdown_pie,
    plot_objective_trend_over_time,
)
from src.visualization.theme import apply_halo_plot_style


def _get_ranking_attr(r: Any, attr: str, default: Any = 0) -> Any:
    """Extrait un attribut d'un ranking (objet ou dict)."""
    if hasattr(r, attr) or isinstance(r, dict):
        return getattr(r, attr, r.get(attr, default)) if hasattr(r, attr) else r.get(attr, default)
    return default


def _extract_ranking_data(
    rankings: list[Any],
) -> tuple[list[str], list[float], list[int]]:
    """Extrait gamertags, scores et matches depuis une liste de rankings."""
    gamertags = [str(_get_ranking_attr(r, "gamertag", "?")) for r in rankings]
    scores = [_get_ranking_attr(r, "total_objective_score", 0) for r in rankings]
    matches = [_get_ranking_attr(r, "matches_count", 0) for r in rankings]
    return gamertags, scores, matches


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

    _obj_map_cols = ["match_id", "kills", "map_name", "start_time"]
    if "map_ui" in match_stats_df.columns:
        _obj_map_cols.append("map_ui")
    # Joindre avec match_stats pour avoir les kills
    combined = (
        match_stats_df.select(_obj_map_cols)
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
            text=[d.get("map_ui") or d.get("map_name", "?") for d in data],
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

    gamertags, scores, matches = _extract_ranking_data(top_rankings)

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
