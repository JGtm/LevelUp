"""Composant graphe radar pour comparer des métriques entre joueurs.

Utilise Plotly pour générer des graphiques radar interactifs.
Les radars de participation sont dans ``_radar_participation.py``,
les radars coéquipiers dans ``_radar_teammates.py``.
"""

from __future__ import annotations

from typing import Any

import plotly.graph_objects as go

from src.ui.i18n import t

# ─── Réexports depuis sous-modules ────────────────────────────────────────
from ._radar_participation import (  # noqa: F401
    create_participation_profile_radar,
    create_participation_radar,
)
from ._radar_teammates import (  # noqa: F401
    create_session_trend_radar,
    create_teammate_synergy_radar,
)


def create_radar_chart(  # noqa: PLR0913
    data: list[dict[str, Any]],
    *,
    title: str | None = None,
    show_legend: bool = True,
    fill_opacity: float = 0.25,
    line_width: float = 2,
    height: int = 400,
) -> go.Figure:
    """Crée un graphe radar comparant des métriques entre plusieurs entités.

    Args:
        data: Liste de dicts avec format:
            [
                {
                    "name": "Joueur 1",
                    "values": [val1, val2, val3, ...],
                    "color": "#FF6B6B",  # optionnel
                },
                ...
            ]
        title: Titre du graphe (optionnel).
        show_legend: Afficher la légende.
        fill_opacity: Opacité du remplissage (0-1).
        line_width: Épaisseur des lignes.
        height: Hauteur du graphe en pixels.

    Returns:
        Figure Plotly.
    """
    fig = go.Figure()

    for item in data:
        name = item.get("name", "")
        values = item.get("values", [])
        color = item.get("color")

        # Fermer le polygone
        values_closed = list(values) + [values[0]] if values else []

        trace_kwargs: dict[str, Any] = {
            "r": values_closed,
            "name": name,
            "fill": "toself",
            "fillcolor": color if color else None,
            "opacity": fill_opacity if color else 1.0,
            "line": {"width": line_width},
        }
        if color:
            trace_kwargs["line"]["color"] = color
            trace_kwargs["fillcolor"] = color

        fig.add_trace(go.Scatterpolar(**trace_kwargs))

    fig.update_layout(
        polar={
            "radialaxis": {
                "visible": True,
                "showticklabels": True,
                "tickfont": {"size": 10},
            },
        },
        showlegend=show_legend,
        title=title,
        height=height,
        margin={"l": 60, "r": 60, "t": 60 if title else 30, "b": 40},
    )

    return fig


def create_stats_per_minute_radar(
    players: list[dict[str, Any]],
    *,
    title: str | None = None,
    categories: list[str] | None = None,
    height: int = 350,
) -> go.Figure:
    """Crée un graphe radar pour les stats par minute (frags/morts/assists).

    Args:
        players: Liste de dicts avec clés kills_per_min, deaths_per_min,
                 assists_per_min, color (optionnel).
        title: Titre du graphe.
        categories: Labels des axes (par défaut: Frags/min, Morts/min, Assists/min).
        height: Hauteur du graphe.

    Returns:
        Figure Plotly.
    """
    if categories is None:
        categories = [t("radar_kpm"), t("radar_dpm"), t("radar_apm")]
    if title is None:
        title = t("radar_stats_per_min")

    # Gestion du cas vide
    if not players:
        fig = go.Figure()
        fig.update_layout(title={"text": title, "x": 0.5, "xanchor": "center"}, height=height)
        return fig

    # Seuils de référence FIXES pour une échelle absolue
    ref_kills_per_min = 1.2
    ref_deaths_per_min = 1.0
    ref_assists_per_min = 0.6

    fig = go.Figure()

    for player in players:
        name = player.get("name", "")
        color = player.get("color")

        orig_kills = player.get("kills_per_min") or 0
        orig_deaths = player.get("deaths_per_min") or 0
        orig_assists = player.get("assists_per_min") or 0

        kills = min(orig_kills / ref_kills_per_min, 1.0)
        deaths = min(orig_deaths / ref_deaths_per_min, 1.0)
        assists = min(orig_assists / ref_assists_per_min, 1.0)

        values = [kills, deaths, assists, kills]  # Fermer le polygone
        theta = categories + [categories[0]]

        customdata = [
            [orig_kills],
            [orig_deaths],
            [orig_assists],
            [orig_kills],
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
                hovertemplate="%{theta}: %{customdata[0]:.2f}<extra>%{fullData.name}</extra>",
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
        showlegend=True,
        title={"text": title, "x": 0.5, "xanchor": "center"},
        height=height,
        margin={"l": 60, "r": 60, "t": 50, "b": 40},
    )

    return fig


def create_performance_radar(
    players: list[dict[str, Any]],
    *,
    title: str | None = None,
    height: int = 400,
) -> go.Figure:
    """Crée un graphe radar pour le profil de performance (objectif/frags/morts/assists).

    Args:
        players: Liste de dicts avec clés objective_score, kills, deaths, assists, color.
        title: Titre du graphe.
        height: Hauteur du graphe.

    Returns:
        Figure Plotly.
    """
    categories = [
        t("radar_objectives"),
        t("radar_kills_label"),
        t("radar_survival"),
        t("radar_assists_label"),
    ]

    if not players:
        fig = go.Figure()
        fig.update_layout(
            title={"text": title or t("radar_perf_profile"), "x": 0.5, "xanchor": "center"},
            height=height,
        )
        return fig

    max_obj = max((p.get("objective_score") or 0) for p in players) or 1
    max_kills = max((p.get("kills") or 0) for p in players) or 1
    max_deaths = max((p.get("deaths") or 1) for p in players) or 1
    max_assists = max((p.get("assists") or 0) for p in players) or 1

    fig = go.Figure()

    for player in players:
        name = player.get("name", "")
        color = player.get("color")

        obj = (player.get("objective_score") or 0) / max_obj
        kills_norm = (player.get("kills") or 0) / max_kills
        deaths_raw = player.get("deaths") or 0
        survival = 1 - (deaths_raw / max_deaths) if max_deaths > 0 else 0
        assists_norm = (player.get("assists") or 0) / max_assists

        values = [obj, kills_norm, survival, assists_norm, obj]
        theta = categories + [categories[0]]

        orig_obj = player.get("objective_score") or 0
        orig_kills = player.get("kills") or 0
        orig_deaths = deaths_raw
        orig_assists = player.get("assists") or 0

        customdata = [
            [f"{orig_obj:.1f}"],
            [f"{orig_kills}"],
            [f"{orig_deaths} {t('radar_hover_deaths')}"],
            [f"{orig_assists}"],
            [f"{orig_obj:.1f}"],
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
        showlegend=True,
        title={"text": title or t("radar_perf_profile"), "x": 0.5, "xanchor": "center"},
        height=height,
        margin={"l": 60, "r": 60, "t": 50, "b": 40},
    )

    return fig


__all__ = [
    "create_radar_chart",
    "create_stats_per_minute_radar",
    "create_performance_radar",
    # Sprint 8.2: Radars de participation
    "create_participation_radar",
    "create_participation_profile_radar",
    "create_teammate_synergy_radar",
    "create_session_trend_radar",
]
