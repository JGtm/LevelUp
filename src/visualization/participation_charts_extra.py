"""Fonctions de visualisation de participation supplémentaires.

Extrait de participation_charts.py pour respecter la limite de 500L.
Contient les constantes partagées, les helpers de catégories ainsi que
les graphiques complexes (indicator, sunburst, radar).
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING

import plotly.express as px
import plotly.graph_objects as go

from src.ui.i18n.data_labels import label as i18n_label
from src.ui.i18n.viz import viz_t
from src.visualization.theme import apply_halo_plot_style

if TYPE_CHECKING:
    import polars as pl

_log = logging.getLogger(__name__)


# =============================================================================
# Constantes de couleurs (partagées avec participation_charts.py)
# =============================================================================

# Couleurs mises à jour pour respecter la palette Okabe-Ito (accessibilité daltonisme).
# Anciens code hexadécimaux (deuteranopie/protanopie-incompatibles) :
#   kill:      #FF6B6B (rouge clair)   → #D55E00 (vermillon)
#   assist:    #4ECDC4 (turquoise)     → #E69F00 (orange Okabe-Ito)
#   objective: #45B7D1 (bleu clair)   → #0072B2 (bleu Okabe-Ito)
#   vehicle:   #96CEB4 (vert pâle)    → #009E73 (vert bleuté)
#   penalty:   #2C3E50 (gris foncé)   → #CC79A7 (rose mauve)
CATEGORY_COLORS: dict[str, str] = {
    "kill": "#D55E00",  # Vermillon Okabe-Ito - kills
    "assist": "#E69F00",  # Orange Okabe-Ito - assists
    "objective": "#0072B2",  # Bleu Okabe-Ito - objectifs
    "vehicle": "#009E73",  # Vert bleuté Okabe-Ito - véhicules
    "penalty": "#CC79A7",  # Rose mauve Okabe-Ito - pénalités
    "other": "#999999",  # Gris neutre
}


def get_category_labels(lang: str = "fr") -> dict[str, str]:
    """Retourne le mapping catégorie → label traduit."""
    return {
        "kill": viz_t("cat_label_kill", lang),
        "assist": viz_t("cat_label_assist", lang),
        "objective": viz_t("cat_label_objective", lang),
        "vehicle": viz_t("cat_label_vehicle", lang),
        "penalty": viz_t("cat_label_penalty", lang),
        "other": viz_t("cat_label_other", lang),
    }


def get_participation_colors() -> dict[str, str]:
    """Retourne le mapping couleur par catégorie."""
    return CATEGORY_COLORS.copy()


# =============================================================================
# Indicateur : Résumé de participation
# =============================================================================


def create_participation_indicator(
    df: pl.DataFrame,
    *,
    title: str | None = None,
    lang: str = "fr",
) -> go.Figure:
    """Indicateur multi-valeurs de participation.

    Affiche kills, assists, objectifs, pénalités en un coup d'œil.

    Args:
        df: DataFrame avec colonnes award_category, award_count, award_score.
        title: Titre.

    Returns:
        Figure Plotly avec indicateurs.
    """
    title = title or viz_t("title_participation", lang)
    import polars as pl

    # Agréger par catégorie
    agg = df.group_by("award_category").agg(
        pl.col("award_count").sum().alias("count"),
        pl.col("award_score").sum().alias("score"),
    )

    # Convertir en dict pour accès facile
    stats = {}
    for row in agg.iter_rows(named=True):
        cat = row["award_category"]
        stats[cat] = {"count": row["count"], "score": row["score"]}

    # Extraire les valeurs
    kills = stats.get("kill", {"count": 0, "score": 0})
    assists = stats.get("assist", {"count": 0, "score": 0})
    objectives = stats.get("objective", {"count": 0, "score": 0})
    penalties = stats.get("penalty", {"count": 0, "score": 0})

    fig = go.Figure()

    # 4 indicateurs sur une ligne
    fig.add_trace(
        go.Indicator(
            mode="number",
            value=kills["count"],
            title={
                "text": f"{viz_t('cat_label_kill', lang)}<br><span style='font-size:0.7em;color:gray'>{kills['score']:,} pts</span>"
            },
            domain={"x": [0, 0.25], "y": [0, 1]},
            number={"font": {"color": CATEGORY_COLORS["kill"], "size": 48}},
        )
    )

    fig.add_trace(
        go.Indicator(
            mode="number",
            value=assists["count"],
            title={
                "text": f"{viz_t('cat_label_assist', lang)}<br><span style='font-size:0.7em;color:gray'>{assists['score']:,} pts</span>"
            },
            domain={"x": [0.25, 0.5], "y": [0, 1]},
            number={"font": {"color": CATEGORY_COLORS["assist"], "size": 48}},
        )
    )

    fig.add_trace(
        go.Indicator(
            mode="number",
            value=objectives["count"],
            title={
                "text": f"{viz_t('cat_label_objective', lang)}<br><span style='font-size:0.7em;color:gray'>{objectives['score']:,} pts</span>"
            },
            domain={"x": [0.5, 0.75], "y": [0, 1]},
            number={"font": {"color": CATEGORY_COLORS["objective"], "size": 48}},
        )
    )

    # Pénalités (avec signe négatif)
    penalty_display = abs(penalties["count"]) if penalties["count"] else 0
    fig.add_trace(
        go.Indicator(
            mode="number",
            value=penalty_display,
            title={
                "text": f"{viz_t('cat_label_penalty', lang)}<br><span style='font-size:0.7em;color:gray'>{penalties['score']:,} pts</span>"
            },
            domain={"x": [0.75, 1], "y": [0, 1]},
            number={"font": {"color": CATEGORY_COLORS["penalty"], "size": 48}},
        )
    )

    fig.update_layout(
        title={"text": title, "x": 0.5},
        height=150,
        margin={"t": 60, "b": 20, "l": 20, "r": 20},
    )

    return fig


# =============================================================================
# Graphique Sunburst : Hiérarchie catégorie → action
# =============================================================================


def plot_participation_sunburst(
    df: pl.DataFrame,
    *,
    title: str | None = None,
    lang: str = "fr",
) -> go.Figure:
    """Sunburst chart hiérarchique catégorie → action.

    Args:
        df: DataFrame avec colonnes award_category, award_name, award_score.
        title: Titre du graphique.

    Returns:
        Figure Plotly.
    """
    title = title or viz_t("title_participation_detail", lang)
    cat_labels = get_category_labels(lang)
    import polars as pl

    # Agréger par catégorie et action
    agg = (
        df.group_by(["award_category", "award_name"])
        .agg(pl.col("award_score").sum().alias("score"))
        .filter(pl.col("score") > 0)  # Exclure les pénalités du sunburst
        .sort("score", descending=True)
    )

    # Préparer les colonnes en Polars
    agg = agg.with_columns(
        pl.col("award_name")
        .map_elements(lambda x: i18n_label("awards", x, lang=lang) or x, return_dtype=pl.Utf8)
        .alias("award_label"),
        pl.col("award_category")
        .map_elements(
            lambda x: cat_labels.get(x, x.capitalize() if x else viz_t("cat_label_other", lang)),
            return_dtype=pl.Utf8,
        )
        .alias("category_label"),
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

    # Mapper les couleurs
    color_map = {}
    for cat, color in CATEGORY_COLORS.items():
        label = cat_labels.get(cat, cat.capitalize())
        color_map[label] = color

    # px.sunburst exige un DataFrame Pandas — conversion à la frontière Plotly
    fig = px.sunburst(
        agg.to_pandas(),
        path=["category_label", "award_label"],
        values="score",
        color="category_label",
        color_discrete_map=color_map,
    )

    fig.update_traces(
        textinfo="label+value",
        hovertemplate="<b>%{label}</b><br>%{value:,.0f} pts<extra></extra>",
    )

    fig.update_layout(
        title={"text": title, "x": 0.5},
    )

    return apply_halo_plot_style(fig)


# =============================================================================
# Helpers pour Radar
# =============================================================================


def aggregate_participation_for_radar(
    df: pl.DataFrame,
    name: str = "Match",
    color: str | None = None,
) -> dict:
    """Agrège les PersonalScores en données pour radar de participation.

    Args:
        df: DataFrame avec colonnes award_category, award_score.
        name: Nom pour le radar (ex: "Match 1", "Session A").
        color: Couleur optionnelle.

    Returns:
        Dict compatible avec create_participation_radar().
    """
    import polars as pl

    if df.is_empty():
        return {
            "name": name,
            "kill_score": 0,
            "assist_score": 0,
            "objective_score": 0,
            "penalty_score": 0,
            "color": color,
        }

    # Agréger par catégorie
    agg = df.group_by("award_category").agg(pl.col("award_score").sum().alias("total"))

    # Convertir en dict
    scores = {row["award_category"]: row["total"] for row in agg.iter_rows(named=True)}

    return {
        "name": name,
        "kill_score": scores.get("kill", 0),
        "assist_score": scores.get("assist", 0),
        "objective_score": scores.get("objective", 0),
        "penalty_score": scores.get("penalty", 0),
        "color": color,
    }


def compute_participation_percentages(
    df: pl.DataFrame,
) -> dict:
    """Calcule les pourcentages de contribution par catégorie.

    Utile pour le radar de complémentarité coéquipiers.

    Args:
        df: DataFrame avec colonnes award_category, award_score.

    Returns:
        Dict avec kills_pct, assists_pct, objectives_pct (sur score positif total).
    """
    import polars as pl

    if df.is_empty():
        return {
            "kills_pct": 0,
            "assists_pct": 0,
            "objectives_pct": 0,
            "vehicles_pct": 0,
        }

    # Filtrer les scores positifs
    positive_df = df.filter(pl.col("award_score") > 0)

    if positive_df.is_empty():
        return {
            "kills_pct": 0,
            "assists_pct": 0,
            "objectives_pct": 0,
            "vehicles_pct": 0,
        }

    total = positive_df["award_score"].sum()
    if total == 0:
        total = 1

    agg = positive_df.group_by("award_category").agg(pl.col("award_score").sum().alias("total"))

    scores = {row["award_category"]: row["total"] for row in agg.iter_rows(named=True)}

    return {
        "kills_pct": (scores.get("kill", 0) / total) * 100,
        "assists_pct": (scores.get("assist", 0) / total) * 100,
        "objectives_pct": (scores.get("objective", 0) / total) * 100,
        "vehicles_pct": (scores.get("vehicle", 0) / total) * 100,
    }
