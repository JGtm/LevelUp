"""Graphiques de participation au match basés sur PersonalScores.

Sprint 8.2 - Visualise la contribution au score :
- Kills, Assists, Objectifs, Véhicules, Pénalités
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING

import plotly.graph_objects as go

from src.ui.i18n.data_labels import label as i18n_label
from src.ui.i18n.viz import viz_t
from src.visualization.participation_charts_extra import (  # noqa: F401
    CATEGORY_COLORS,
    aggregate_participation_for_radar,
    compute_participation_percentages,
    create_participation_indicator,
    get_category_labels,
    get_participation_colors,
    plot_participation_sunburst,
)
from src.visualization.theme import apply_halo_plot_style

if TYPE_CHECKING:
    import polars as pl

_log = logging.getLogger(__name__)


# =============================================================================
# Graphique Pie : Répartition du score par catégorie
# =============================================================================


def plot_participation_pie(
    df: pl.DataFrame,
    *,
    title: str | None = None,
    show_values: bool = True,
    lang: str = "fr",
) -> go.Figure:
    """Pie chart de la contribution au score par catégorie.

    Args:
        df: DataFrame avec colonnes award_category, award_score.
        title: Titre du graphique.
        show_values: Afficher les valeurs absolues.

    Returns:
        Figure Plotly.
    """
    import polars as pl

    title = title or viz_t("title_participation_profile", lang)
    cat_labels = get_category_labels(lang)
    # Agréger par catégorie
    agg = (
        df.group_by("award_category")
        .agg(pl.col("award_score").sum().alias("total_score"))
        .sort("total_score", descending=True)
    )

    # Mapper les labels et couleurs (Polars natif)
    agg = agg.with_columns(
        pl.col("award_category")
        .map_elements(
            lambda x: cat_labels.get(x, x.capitalize() if x else viz_t("cat_label_other", lang)),
            return_dtype=pl.Utf8,
        )
        .alias("label"),
        pl.col("award_category")
        .map_elements(
            lambda x: CATEGORY_COLORS.get(x, CATEGORY_COLORS["other"]),
            return_dtype=pl.Utf8,
        )
        .alias("color"),
    )

    # Filtrer les valeurs négatives pour le pie (pénalités)
    penalties = agg.filter(pl.col("total_score") < 0)["total_score"].sum() or 0
    agg_positive = agg.filter(pl.col("total_score") > 0)

    if agg_positive.is_empty():
        _log.debug("plot_participation_pie: aucune catégorie positive")

    fig = go.Figure()

    if not agg_positive.is_empty():
        text_info = "percent+value" if show_values else "percent"

        fig.add_trace(
            go.Pie(
                labels=agg_positive["label"].to_list(),
                values=agg_positive["total_score"].to_list(),
                marker={"colors": agg_positive["color"].to_list()},
                textinfo=text_info,
                texttemplate="%{label}<br>%{value:,.0f} pts<br>(%{percent})"
                if show_values
                else "%{label}<br>%{percent}",
                hole=0.4,  # Donut chart
                hovertemplate="<b>%{label}</b><br>%{value:,.0f} points<br>%{percent}<extra></extra>",
            )
        )

    # Annotation centrale avec le total
    total_positive = agg_positive["total_score"].sum() or 0
    total_net = total_positive + penalties

    fig.update_layout(
        title={"text": title, "x": 0.5},
        showlegend=True,
        legend={"orientation": "h", "yanchor": "bottom", "y": -0.1},
        annotations=[
            {
                "text": f"<b>{int(total_net):,}</b><br>points",
                "x": 0.5,
                "y": 0.5,
                "font_size": 18,
                "showarrow": False,
            }
        ],
    )

    # Annotation pénalités si présentes
    if penalties < 0:
        fig.add_annotation(
            text=viz_t("annot_penalties", lang, pts=f"{int(penalties):,}"),
            xref="paper",
            yref="paper",
            x=0.5,
            y=-0.15,
            showarrow=False,
            font={"size": 12, "color": CATEGORY_COLORS["penalty"]},
        )

    return apply_halo_plot_style(fig)


# =============================================================================
# Graphique Bars : Détail des actions
# =============================================================================


def plot_participation_bars(
    df: pl.DataFrame,
    *,
    title: str | None = None,
    top_n: int = 10,
    orientation: str = "h",
    lang: str = "fr",
) -> go.Figure:
    """Bar chart horizontal des actions par type.

    Args:
        df: DataFrame avec colonnes award_name, award_count, award_score, award_category.
        title: Titre du graphique.
        top_n: Nombre d'actions à afficher.
        orientation: "h" horizontal, "v" vertical.

    Returns:
        Figure Plotly.
    """
    title = title or viz_t("title_action_detail", lang)
    import polars as pl

    # Agréger par action
    agg = (
        df.group_by(["award_name", "award_category"])
        .agg(
            pl.col("award_count").sum().alias("count"),
            pl.col("award_score").sum().alias("score"),
        )
        .sort("score", descending=True)
        .head(top_n)
    )

    # Traduire les award_name techniques en labels localisés (Polars natif)
    agg = agg.with_columns(
        pl.col("award_name")
        .map_elements(lambda x: i18n_label("awards", x, lang=lang) or x, return_dtype=pl.Utf8)
        .alias("award_label"),
        pl.col("award_category")
        .map_elements(
            lambda x: CATEGORY_COLORS.get(x, CATEGORY_COLORS["other"]),
            return_dtype=pl.Utf8,
        )
        .alias("color"),
    )

    if agg.is_empty():
        fig = go.Figure()
        fig.add_annotation(
            text=viz_t("empty_no_data", lang),
            xref="paper",
            yref="paper",
            x=0.5,
            y=0.5,
            showarrow=False,
        )
        return apply_halo_plot_style(fig)

    # Inverser pour afficher du haut vers le bas
    if orientation == "h":
        agg = agg.reverse()

    fig = go.Figure()

    labels = agg["award_label"].to_list()
    scores = agg["score"].to_list()
    counts = agg["count"].to_list()
    colors = agg["color"].to_list()

    if orientation == "h":
        fig.add_trace(
            go.Bar(
                y=labels,
                x=scores,
                orientation="h",
                marker={"color": colors},
                text=[f"{int(s):,} pts ({int(c)}x)" for s, c in zip(scores, counts, strict=False)],
                textposition="outside",
                hovertemplate=viz_t("hover_score_pts_h", lang),
            )
        )
        fig.update_layout(
            xaxis_title=viz_t("axis_points", lang),
            yaxis_title="",
        )
    else:
        fig.add_trace(
            go.Bar(
                x=labels,
                y=scores,
                marker={"color": colors},
                text=[f"{int(s):,}" for s in scores],
                textposition="outside",
                hovertemplate=viz_t("hover_score_pts_v", lang),
            )
        )
        fig.update_layout(
            xaxis_title="",
            yaxis_title=viz_t("axis_points", lang),
            xaxis_tickangle=-45,
        )

    fig.update_layout(
        title={"text": title, "x": 0.5},
        showlegend=False,
        margin={"l": 150} if orientation == "h" else {"b": 100},
    )

    return apply_halo_plot_style(fig)


# =============================================================================
# Graphique Stacked Bars : Participation par match
# =============================================================================


def plot_participation_by_match(
    df: pl.DataFrame,
    *,
    title: str | None = None,
    last_n: int = 20,
    lang: str = "fr",
) -> go.Figure:
    """Stacked bar chart de la participation par match.

    Args:
        df: DataFrame avec colonnes match_id, award_category, award_score.
        title: Titre du graphique.
        last_n: Nombre de matchs à afficher.

    Returns:
        Figure Plotly.
    """
    title = title or viz_t("title_participation_by_match", lang)
    cat_labels = get_category_labels(lang)
    import polars as pl

    # Agréger par match et catégorie
    agg = df.group_by(["match_id", "award_category"]).agg(
        pl.col("award_score").sum().alias("score")
    )

    # Pivoter pour avoir une colonne par catégorie
    pivoted = agg.pivot(
        index="match_id",
        on="award_category",
        values="score",
    ).fill_null(0)

    # Prendre les derniers matchs
    if pivoted.height > last_n:
        pivoted = pivoted.tail(last_n)

    fig = go.Figure()

    # Ordre des catégories pour le stacking
    categories_order = ["kill", "assist", "objective", "vehicle", "other", "penalty"]

    for cat in categories_order:
        if cat in pivoted.columns:
            cat_labels = get_category_labels(lang)
            fig.add_trace(
                go.Bar(
                    name=cat_labels.get(cat, cat.capitalize()),
                    x=pivoted["match_id"].to_list(),
                    y=pivoted[cat].to_list(),
                    marker={"color": CATEGORY_COLORS.get(cat, CATEGORY_COLORS["other"])},
                    hovertemplate=f"<b>{cat_labels.get(cat, cat)}</b><br>"
                    + "%{y:,.0f} pts<extra></extra>",
                )
            )

    fig.update_layout(
        title={"text": title, "x": 0.5},
        barmode="relative",  # Permet les valeurs négatives
        xaxis_title=viz_t("axis_match_number", lang),
        yaxis_title=viz_t("axis_points", lang),
        legend={"orientation": "h", "yanchor": "bottom", "y": 1.02},
        xaxis={"tickangle": -45, "showticklabels": False},  # Masquer les IDs longs
    )

    return apply_halo_plot_style(fig)


# =============================================================================
# Export
# =============================================================================

__all__ = [
    "get_participation_colors",
    "plot_participation_pie",
    "plot_participation_bars",
    "plot_participation_by_match",
    "create_participation_indicator",
    "plot_participation_sunburst",
    # Helpers pour radar
    "aggregate_participation_for_radar",
    "compute_participation_percentages",
]
