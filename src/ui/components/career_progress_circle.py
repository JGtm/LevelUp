"""Composant cercle de progression pour le rang carrière Halo Infinite.

Affiche un indicateur gauge Plotly montrant la progression XP
vers le prochain rang, ainsi que la progression globale vers le rang Héros.
"""

from __future__ import annotations

import plotly.graph_objects as go

from src.config import THEME_COLORS
from src.ui.i18n import t

# Constantes de progression vers le rang Héros (rang 272)
XP_HERO_TOTAL: int = 9_319_350  # XP cumulée pour atteindre le rang 272 (officiel)
RANK_MAX: int = 272  # Rang maximum (Héros)


def create_career_progress_gauge(
    current_xp: int,
    xp_for_next_rank: int,
    progress_pct: float,
    rank_name_fr: str,
    *,
    is_max_rank: bool = False,
    height: int = 280,
) -> go.Figure:
    """Crée un indicateur gauge de progression XP vers le prochain rang.

    Args:
        current_xp: XP actuel dans le rang courant.
        xp_for_next_rank: XP requis pour le prochain rang.
        progress_pct: Pourcentage de progression (0-100).
        rank_name_fr: Nom du rang en français.
        is_max_rank: True si le joueur est au rang maximum.
        height: Hauteur de la figure en pixels.

    Returns:
        Figure Plotly avec l'indicateur gauge.
    """
    if is_max_rank:
        progress_pct = 100.0
        subtitle = t("career_max_rank")
    else:
        subtitle = f"{current_xp:,} / {xp_for_next_rank:,} XP"

    # Couleur de la barre selon la progression
    if progress_pct >= 75:
        bar_color = "#00ff88"  # Vert vif
    elif progress_pct >= 50:
        bar_color = THEME_COLORS.accent  # Cyan Halo
    elif progress_pct >= 25:
        bar_color = "#ffaa00"  # Ambre
    else:
        bar_color = "#ff6666"  # Rouge doux

    bg_rgb = THEME_COLORS.bg_plot
    bg_color = f"rgb({bg_rgb[0]}, {bg_rgb[1]}, {bg_rgb[2]})"

    fig = go.Figure(
        go.Indicator(
            mode="gauge+number",
            value=progress_pct,
            number={"suffix": "%", "font": {"size": 36, "color": "white"}},
            title={
                "text": f"<b>{rank_name_fr}</b><br><span style='font-size:12px;color:#aaa'>{subtitle}</span>",
                "font": {"size": 16, "color": "white"},
            },
            gauge={
                "axis": {
                    "range": [0, 100],
                    "tickwidth": 0,
                    "tickcolor": "rgba(0,0,0,0)",
                    "dtick": 25,
                    "tickfont": {"size": 10, "color": "#666"},
                },
                "bar": {"color": bar_color, "thickness": 0.7},
                "bgcolor": "rgba(50, 60, 70, 0.3)",
                "borderwidth": 0,
                "steps": [
                    {"range": [0, 25], "color": "rgba(255, 102, 102, 0.08)"},
                    {"range": [25, 50], "color": "rgba(255, 170, 0, 0.08)"},
                    {"range": [50, 75], "color": "rgba(51, 214, 255, 0.08)"},
                    {"range": [75, 100], "color": "rgba(0, 255, 136, 0.08)"},
                ],
                "threshold": {
                    "line": {"color": "white", "width": 2},
                    "thickness": 0.8,
                    "value": progress_pct,
                },
            },
        )
    )

    fig.update_layout(
        paper_bgcolor=bg_color,
        plot_bgcolor=bg_color,
        font={"color": "white"},
        height=height,
        margin={"t": 80, "b": 20, "l": 30, "r": 30},
    )

    return fig


def compute_hero_progress(xp_total: int, rank: int, is_max_rank: bool) -> dict:
    """Calcule la progression globale vers le rang Héros.

    Args:
        xp_total: XP total cumulé du joueur.
        rank: Numéro du rang actuel (1-272).
        is_max_rank: True si le joueur est au rang maximum.

    Returns:
        Dict avec percentage, xp_remaining, xp_total, xp_hero_total.
    """
    if is_max_rank or rank >= RANK_MAX:
        return {
            "percentage": 100.0,
            "xp_remaining": 0,
            "xp_total": xp_total,
            "xp_hero_total": XP_HERO_TOTAL,
        }

    percentage = min(100.0, (xp_total / XP_HERO_TOTAL) * 100) if XP_HERO_TOTAL > 0 else 0.0
    xp_remaining = max(0, XP_HERO_TOTAL - xp_total)

    return {
        "percentage": round(percentage, 2),
        "xp_remaining": xp_remaining,
        "xp_total": xp_total,
        "xp_hero_total": XP_HERO_TOTAL,
    }


def create_hero_progress_gauge(
    hero_pct: float,
    xp_total: int,
    xp_remaining: int,
    *,
    is_max_rank: bool = False,
    height: int = 280,
) -> go.Figure:
    """Crée un indicateur gauge de progression globale vers le rang Héros.

    Args:
        hero_pct: Pourcentage de progression vers Héros (0-100).
        xp_total: XP total cumulé du joueur.
        xp_remaining: XP restante pour atteindre Héros.
        is_max_rank: True si le joueur est au rang maximum.
        height: Hauteur de la figure en pixels.

    Returns:
        Figure Plotly avec l'indicateur gauge.
    """
    if is_max_rank:
        hero_pct = 100.0
        subtitle = t("career_hero_rank")
    else:
        subtitle = f"{xp_total:,} / {XP_HERO_TOTAL:,} XP"

    # Couleur de la barre selon la progression
    if hero_pct >= 75:
        bar_color = "#00ff88"  # Vert vif
    elif hero_pct >= 50:
        bar_color = THEME_COLORS.accent  # Cyan Halo
    elif hero_pct >= 25:
        bar_color = "#ffaa00"  # Ambre
    else:
        bar_color = "#ff6666"  # Rouge doux

    bg_rgb = THEME_COLORS.bg_plot
    bg_color = f"rgb({bg_rgb[0]}, {bg_rgb[1]}, {bg_rgb[2]})"

    fig = go.Figure(
        go.Indicator(
            mode="gauge+number",
            value=hero_pct,
            number={"suffix": "%", "font": {"size": 36, "color": "white"}},
            title={
                "text": (
                    f"<b>{t('career_progression_to_hero')}</b><br>"
                    f"<span style='font-size:12px;color:#aaa'>{subtitle}</span>"
                ),
                "font": {"size": 16, "color": "white"},
            },
            gauge={
                "axis": {
                    "range": [0, 100],
                    "tickwidth": 0,
                    "tickcolor": "rgba(0,0,0,0)",
                    "dtick": 25,
                    "tickfont": {"size": 10, "color": "#666"},
                },
                "bar": {"color": bar_color, "thickness": 0.7},
                "bgcolor": "rgba(50, 60, 70, 0.3)",
                "borderwidth": 0,
                "steps": [
                    {"range": [0, 25], "color": "rgba(255, 102, 102, 0.08)"},
                    {"range": [25, 50], "color": "rgba(255, 170, 0, 0.08)"},
                    {"range": [50, 75], "color": "rgba(51, 214, 255, 0.08)"},
                    {"range": [75, 100], "color": "rgba(0, 255, 136, 0.08)"},
                ],
                "threshold": {
                    "line": {"color": "white", "width": 2},
                    "thickness": 0.8,
                    "value": hero_pct,
                },
            },
        )
    )

    fig.update_layout(
        paper_bgcolor=bg_color,
        plot_bgcolor=bg_color,
        font={"color": "white"},
        height=height,
        margin={"t": 80, "b": 20, "l": 30, "r": 30},
    )

    return fig
