"""Radars coéquipiers — Synergie et évolution par session.

Extrait de radar_chart.py pour respecter la limite de 500 lignes.
"""

from __future__ import annotations

from typing import Any

import plotly.graph_objects as go

from src.ui.i18n import t


def _add_synergy_trace(fig: go.Figure, player: dict, categories: list[str]) -> None:
    """Ajoute une trace Scatterpolar normalisée pour un joueur."""
    name = player.get("name", "")
    color = player.get("color")

    raw = {
        "kills_pct": player.get("kills_pct") or 0,
        "assists_pct": player.get("assists_pct") or 0,
        "objectives_pct": player.get("objectives_pct") or 0,
        "kd_ratio": player.get("kd_ratio") or 0,
        "accuracy": player.get("accuracy") or 0,
    }
    values = [
        raw["kills_pct"] / 100,
        raw["assists_pct"] / 100,
        raw["objectives_pct"] / 100,
        min(raw["kd_ratio"] / 3, 1),
        raw["accuracy"] / 100,
    ]
    values.append(values[0])
    theta = categories + [categories[0]]

    customdata = [
        [f"{raw['kills_pct']:.1f}%"],
        [f"{raw['assists_pct']:.1f}%"],
        [f"{raw['objectives_pct']:.1f}%"],
        [f"{raw['kd_ratio']:.2f}"],
        [f"{raw['accuracy']:.1f}%"],
        [f"{raw['kills_pct']:.1f}%"],
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


def create_teammate_synergy_radar(
    me_data: dict[str, Any],
    teammate_data: dict[str, Any],
    *,
    title: str | None = None,
    height: int = 400,
) -> go.Figure:
    """Crée un radar comparant le profil de jeu entre moi et un coéquipier.

    Montre qui apporte quoi à l'équipe (complémentarité).

    Args:
        me_data: Dict avec clés kills_pct, assists_pct, objectives_pct, kd_ratio, accuracy, color.
        teammate_data: Même format pour le coéquipier.
        title: Titre du graphe.
        height: Hauteur du graphe.

    Returns:
        Figure Plotly.
    """
    categories = [
        t("radar_kills_pct"),
        t("radar_assists_pct"),
        t("radar_obj_pct"),
        t("radar_kd"),
        t("col_accuracy"),
    ]

    fig = go.Figure()

    for player in [me_data, teammate_data]:
        _add_synergy_trace(fig, player, categories)

    fig.update_layout(
        polar={
            "radialaxis": {
                "visible": True,
                "range": [0, 1.1],
                "showticklabels": False,
            },
        },
        showlegend=True,
        legend={"orientation": "h", "yanchor": "bottom", "y": -0.15, "x": 0.5, "xanchor": "center"},
        title={"text": title or t("radar_complementarity"), "x": 0.5, "xanchor": "center"},
        height=height,
        margin={"l": 70, "r": 70, "t": 60 if title else 30, "b": 70},
    )

    return fig


def create_session_trend_radar(
    sessions: list[dict[str, Any]],
    *,
    title: str | None = None,
    height: int = 400,
) -> go.Figure:
    """Crée un radar montrant l'évolution du profil entre plusieurs sessions.

    Args:
        sessions: Liste de dicts avec clés kd_ratio, win_rate, accuracy,
                  obj_participation, avg_score, color.
        title: Titre du graphe.
        height: Hauteur du graphe.

    Returns:
        Figure Plotly.
    """
    categories = [
        t("radar_kd"),
        t("radar_win_rate"),
        t("col_accuracy"),
        t("radar_objectives"),
        t("radar_avg_score"),
    ]

    if not sessions:
        fig = go.Figure()
        fig.update_layout(
            title={"text": title or t("radar_session_evolution"), "x": 0.5, "xanchor": "center"},
            height=height,
        )
        return fig

    max_kd = max((s.get("kd_ratio") or 0) for s in sessions) or 1
    max_score = max((s.get("avg_score") or 0) for s in sessions) or 1

    fig = go.Figure()

    for session in sessions:
        name = session.get("name", "")
        color = session.get("color")

        kd_norm = min((session.get("kd_ratio") or 0) / max(max_kd, 2), 1)
        wr_norm = (session.get("win_rate") or 0) / 100
        acc_norm = (session.get("accuracy") or 0) / 100
        obj_norm = (session.get("obj_participation") or 0) / 100
        score_norm = (session.get("avg_score") or 0) / max_score

        values = [kd_norm, wr_norm, acc_norm, obj_norm, score_norm, kd_norm]
        theta = categories + [categories[0]]

        orig_kd = session.get("kd_ratio") or 0
        orig_wr = session.get("win_rate") or 0
        orig_acc = session.get("accuracy") or 0
        orig_obj = session.get("obj_participation") or 0
        orig_score = session.get("avg_score") or 0

        customdata = [
            [f"{orig_kd:.2f}"],
            [f"{orig_wr:.1f}%"],
            [f"{orig_acc:.1f}%"],
            [f"{orig_obj:.1f}%"],
            [f"{int(orig_score):,}"],
            [f"{orig_kd:.2f}"],
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
        title={"text": title or t("radar_session_evolution"), "x": 0.5, "xanchor": "center"},
        height=height,
        margin={"l": 70, "r": 70, "t": 60 if title else 30, "b": 50},
    )

    return fig
