"""Radars de participation — PersonalScores et profil 6 axes.

Extrait de radar_chart.py pour respecter la limite de 500 lignes.
"""

from __future__ import annotations

from typing import Any

import plotly.graph_objects as go

from src.config import THEME_COLORS
from src.ui.i18n import t
from src.visualization.theme import apply_halo_plot_style


def create_participation_radar(
    participation_data: list[dict[str, Any]],
    *,
    height: int = 400,
    show_values: bool = True,
) -> go.Figure:
    """Crée un radar de participation basé sur les PersonalScores.

    Axes : Frags, Assists, Objectifs, Survie (inverse pénalités).

    Args:
        participation_data: Liste de dicts avec format:
            [
                {
                    "name": "Match 1" ou "Joueur",
                    "kill_score": 700,
                    "assist_score": 150,
                    "objective_score": 300,
                    "vehicle_score": 50,
                    "penalty_score": -100,
                    "color": "#FF6B6B",
                },
                ...
            ]
        title: Titre du graphe.
        height: Hauteur du graphe.
        show_values: Afficher les valeurs dans le hover.

    Returns:
        Figure Plotly.
    """
    categories = [
        t("radar_kills_label"),
        t("radar_assists_label"),
        t("radar_objectives"),
        t("radar_survival"),
    ]

    if not participation_data:
        fig = go.Figure()
        fig.update_layout(
            height=height,
        )
        return fig

    # Seuils fixes basés sur les valeurs typiques dans Halo Infinite
    max_kill_score = 2000.0
    max_assist_score = 500.0
    max_objective_score = 1000.0
    max_penalty_score = 500.0

    # Plusieurs matchs : utiliser le max réel pour comparaison relative
    if len(participation_data) > 1:
        max_kill = max(abs(p.get("kill_score") or 0) for p in participation_data) or max_kill_score
        max_assist = (
            max(abs(p.get("assist_score") or 0) for p in participation_data) or max_assist_score
        )
        max_obj = (
            max(abs(p.get("objective_score") or 0) for p in participation_data)
            or max_objective_score
        )
        max_penalty = (
            max(abs(p.get("penalty_score") or 0) for p in participation_data) or max_penalty_score
        )
    else:
        max_kill = max_kill_score
        max_assist = max_assist_score
        max_obj = max_objective_score
        max_penalty = max_penalty_score

    fig = go.Figure()

    for item in participation_data:
        name = item.get("name", "")
        color = item.get("color")

        kill_raw = item.get("kill_score") or 0
        assist_raw = item.get("assist_score") or 0
        obj_raw = item.get("objective_score") or 0
        penalty_raw = item.get("penalty_score") or 0

        kill_norm = min(kill_raw / max_kill if max_kill else 0, 1.0)
        assist_norm = min(assist_raw / max_assist if max_assist else 0, 1.0)
        obj_norm = min(obj_raw / max_obj if max_obj else 0, 1.0)
        survival_norm = max(0.0, 1.0 - (abs(penalty_raw) / max_penalty) if max_penalty else 1.0)

        values = [kill_norm, assist_norm, obj_norm, survival_norm, kill_norm]
        theta = categories + [categories[0]]

        customdata = [
            [f"{int(kill_raw):,} pts"],
            [f"{int(assist_raw):,} pts"],
            [f"{int(obj_raw):,} pts"],
            [f"{int(penalty_raw):,} pts" if penalty_raw else t("radar_hover_no_penalty")],
            [f"{int(kill_raw):,} pts"],
        ]

        fig.add_trace(
            go.Scatterpolar(
                r=values,
                theta=theta,
                name=name,
                fill="toself",
                line={"width": 2, "color": color} if color else {"width": 2},
                fillcolor=color,
                opacity=0.3,
                customdata=customdata,
                hovertemplate="%{theta}: %{customdata[0]}<extra>%{fullData.name}</extra>",
            )
        )

    fig.update_layout(
        polar={
            "radialaxis": {
                "visible": True,
                "range": [0, 1.1],
                "showticklabels": False,
            },
        },
        showlegend=len(participation_data) > 1,
        height=height,
        margin={"l": 70, "r": 70, "t": 30, "b": 50},
    )

    return fig


def create_participation_profile_radar(
    profiles: list[dict[str, Any]],
    *,
    height: int = 400,
    fill_opacity: float = 1.0,
    show_fill: bool = True,
    radial_range: tuple[float, float] = (0, 1.1),
) -> go.Figure:
    """Crée un radar à 6 axes : Objectifs, Combat, Support, Score, Impact, Survie.

    Conçu pour être réutilisable dans Dernier match et Mes coéquipiers.
    Les profils doivent être au format retourné par compute_participation_profile().

    Args:
        profiles: Liste de dicts avec clés *_norm et *_raw pour chaque axe.
        title: Titre du graphe.
        height: Hauteur en pixels.
        fill_opacity: Opacité du remplissage (0-1, défaut 1.0 = opaque).
        show_fill: Si True, remplit la zone ; False = lignes uniquement.
        radial_range: Tuple (min, max) pour l'échelle radiale.

    Returns:
        Figure Plotly.
    """
    categories = [
        t("radar_objectives"),
        t("radar_combat"),
        t("radar_support"),
        t("col_score"),
        t("radar_impact"),
        t("radar_survival"),
    ]

    if not profiles:
        fig = go.Figure()
        fig.update_layout(
            height=height,
        )
        return fig

    fig = go.Figure()

    for item in profiles:
        name = item.get("name", "")
        color = item.get("color")

        obj_n = item.get("objectifs_norm") or 0
        combat_n = item.get("combat_norm") or 0
        support_n = item.get("support_norm") or 0
        score_n = item.get("score_norm") or 0
        impact_n = item.get("impact_norm") or 0
        survie_n = item.get("survie_norm") or 0

        values = [obj_n, combat_n, support_n, score_n, impact_n, survie_n, obj_n]
        theta = categories + [categories[0]]

        obj_r = item.get("objectifs_raw") or 0
        combat_r = item.get("combat_raw") or 0
        support_r = item.get("support_raw") or 0
        score_r = item.get("score_raw") or 0
        impact_r = item.get("impact_raw") or 0
        survie_pct = (item.get("survie_raw") or 0) * 100

        customdata = [
            [f"{int(obj_r):,} pts"],
            [f"{int(combat_r):,} pts"],
            [f"{int(support_r):,} pts"],
            [f"{int(score_r):,} pts"],
            [f"{impact_r:.1f} pts/min"],
            [f"{survie_pct:.0f}% {t('radar_hover_survival_pct')}"],
            [f"{int(obj_r):,} pts"],
        ]

        fill_mode = "toself" if show_fill else "none"
        if show_fill and color:
            fc = color.lstrip("#")
            if len(fc) == 6:
                r_c, g_c, b_c = int(fc[0:2], 16), int(fc[2:4], 16), int(fc[4:6], 16)
                fill_color = f"rgba({r_c},{g_c},{b_c},{fill_opacity})"
            else:
                fill_color = color
        else:
            fill_color = None

        fig.add_trace(
            go.Scatterpolar(
                r=values,
                theta=theta,
                name=name,
                fill=fill_mode,
                line={"width": 3, "color": color} if color else {"width": 3},
                fillcolor=fill_color,
                customdata=customdata,
                hovertemplate="%{theta}: %{customdata[0]}<extra>%{fullData.name}</extra>",
            )
        )

    fig.update_layout(
        polar={
            "radialaxis": {
                "visible": True,
                "range": list(radial_range),
                "showticklabels": True,
                "tickvals": [0.25, 0.5, 0.75, 1.0],
                "ticktext": ["25%", "50%", "75%", "100%"],
                "tickfont": {"color": "rgba(255,255,255,0.85)", "size": 11, "weight": "bold"},
                "gridcolor": "rgba(255,255,255,0.12)",
            },
            "angularaxis": {
                "gridcolor": "rgba(255,255,255,0.12)",
                "linecolor": THEME_COLORS.border,
                "tickfont": {"color": THEME_COLORS.text_primary},
            },
            "bgcolor": THEME_COLORS.bg_plot_rgba(1.0),
        },
        showlegend=len(profiles) > 1,
        legend={
            "orientation": "h",
            "yanchor": "bottom",
            "y": -0.12,
            "x": 0.5,
            "xanchor": "center",
            "font": {"color": THEME_COLORS.text_primary},
        },
        height=height,
        margin={"l": 70, "r": 70, "t": 30, "b": 60},
    )

    return apply_halo_plot_style(fig, height=None)
