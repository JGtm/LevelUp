"""Frise chronologique de dominance d'équipe pour un match PvP Halo Infinite.

Deux panneaux liés par l'axe X (temps du match) :
1. Barres de dominance : répartition des frags par tranche de 30s (tug-of-war).
2. Kill feed : événements individuels avec séries annotées.

Usage typique
-------------
>>> buckets = compute_dominance_buckets(events, xuid_to_team, my_team_id, duration_s)
>>> streaks = detect_streaks(events, xuid_to_team, xuid_to_gamertag)
>>> kill_events = [e for e in events if e["event_type"] == "kill"]
>>> fig = plot_dominance_chart(buckets, streaks, kill_events, xuid_to_team, my_team_id, duration_s)
"""

from __future__ import annotations

import math
from typing import Any

import plotly.graph_objects as go
from plotly.subplots import make_subplots

from src.ui.i18n.viz import viz_t
from src.visualization._plot_options import PlotOptions
from src.visualization._team_dominance_helpers import (
    ENEMY_COLOR,
    MY_TEAM_COLOR,
    NORMAL_MARKER_SIZE,
    DominanceBucket,
    KillStreak,
    add_cumul_annotations,
    add_streaks,
    configure_axes,
    fmt_s,
    prepare_bar_data,
    prepare_kill_feed,
)

# ─────────────────────────────────────────────────────────────────────────────
# Calcul : dominance par tranche
# ─────────────────────────────────────────────────────────────────────────────


def compute_dominance_buckets(
    events: list[dict[str, Any]],
    xuid_to_team: dict[str, int],
    my_team_id: int,
    duration_s: float,
    bucket_s: int = 30,
) -> list[DominanceBucket]:
    """Calcule les stats de frags par tranche pour les deux équipes.

    Args:
        events: Liste d'events highlight_events (event_type, time_ms, xuid, …).
        xuid_to_team: Mapping xuid → team_id pour tous les joueurs du match.
        my_team_id: Identifiant de l'équipe du joueur principal.
        duration_s: Durée totale estimée du match en secondes.
        bucket_s: Taille de chaque tranche en secondes (défaut 30).

    Returns:
        Liste de DominanceBucket, une par tranche temporelle.
    """
    if duration_s <= 0 or bucket_s <= 0:
        return []

    n_buckets = math.ceil(duration_s / bucket_s)
    buckets = [
        DominanceBucket(
            t_start_s=i * bucket_s,
            t_end_s=min((i + 1) * bucket_s, duration_s),
        )
        for i in range(n_buckets)
    ]

    for e in events:
        if str(e.get("event_type", "")).lower() != "kill":
            continue
        t_ms = e.get("time_ms")
        if t_ms is None:
            continue
        xuid = str(e.get("xuid", "")).strip()
        if not xuid:
            continue
        team_id = xuid_to_team.get(xuid)
        if team_id is None:
            continue

        idx = min(int(t_ms / 1000 / bucket_s), n_buckets - 1)
        if team_id == my_team_id:
            buckets[idx].my_kills += 1
        else:
            buckets[idx].enemy_kills += 1

    return buckets


# ─────────────────────────────────────────────────────────────────────────────
# Détection : séries individuelles
# ─────────────────────────────────────────────────────────────────────────────


def detect_streaks(  # noqa: C901, PLR0912
    events: list[dict[str, Any]],
    xuid_to_team: dict[str, int],
    xuid_to_gamertag: dict[str, str],
    min_kills: int = 3,
    gap_s: float = 60.0,
) -> list[KillStreak]:
    """Détecte les séries de kills individuelles dans un match.

    Une série est une succession de kills par le même joueur sans qu'il meure
    entre deux kills consécutifs, et sans pause de plus de `gap_s` secondes.

    Args:
        events: Liste d'events highlight_events (kill + death).
        xuid_to_team: Mapping xuid → team_id.
        xuid_to_gamertag: Mapping xuid → gamertag résolu.
        min_kills: Nombre minimum de kills pour constituer une série.
        gap_s: Fenêtre temporelle max entre deux kills consécutifs (secondes).

    Returns:
        Liste de KillStreak triée chronologiquement.
    """
    from collections import defaultdict

    kills_by_player: dict[str, list[int]] = defaultdict(list)
    deaths_by_player: dict[str, list[int]] = defaultdict(list)

    # Collecte des gamertags directement depuis les events (fallback fiable)
    gamertag_from_events: dict[str, str] = {}
    for e in events:
        etype = str(e.get("event_type", "")).lower()
        t_ms = e.get("time_ms")
        xuid = str(e.get("xuid", "")).strip()
        if not xuid or t_ms is None:
            continue
        gt = str(e.get("gamertag", "") or "").strip()
        if gt and not gt.isdigit() and not gt.lower().startswith("xuid("):
            gamertag_from_events.setdefault(xuid, gt)
        if etype == "kill":
            kills_by_player[xuid].append(int(t_ms))
        elif etype == "death":
            deaths_by_player[xuid].append(int(t_ms))

    gap_ms = int(gap_s * 1000)
    streaks: list[KillStreak] = []

    for xuid, kill_times in kills_by_player.items():
        if len(kill_times) < min_kills:
            continue

        kill_times_sorted = sorted(kill_times)
        death_times = sorted(deaths_by_player.get(xuid, []))
        _gt_candidate = xuid_to_gamertag.get(xuid) or gamertag_from_events.get(xuid)
        # Fallback lisible quand le gamertag n'est pas résolu (XUID numérique brut)
        if (
            _gt_candidate
            and not _gt_candidate.isdigit()
            and not _gt_candidate.lower().startswith("xuid(")
        ):
            gamertag = _gt_candidate
        else:
            gamertag = f"#{xuid[-4:]}" if xuid.isdigit() and len(xuid) >= 4 else xuid
        team_id = xuid_to_team.get(xuid)

        current: list[int] = [kill_times_sorted[0]]

        for i in range(1, len(kill_times_sorted)):
            prev_t = kill_times_sorted[i - 1]
            curr_t = kill_times_sorted[i]

            # Mort entre les deux kills → reset
            died_between = any(prev_t < d < curr_t for d in death_times)
            # Trop longue pause → reset
            too_long = (curr_t - prev_t) > gap_ms

            if died_between or too_long:
                if len(current) >= min_kills:
                    streaks.append(
                        KillStreak(
                            xuid=xuid,
                            gamertag=gamertag,
                            team_id=team_id,
                            kill_times_ms=list(current),
                        )
                    )
                current = [curr_t]
            else:
                current.append(curr_t)

        if len(current) >= min_kills:
            streaks.append(
                KillStreak(
                    xuid=xuid,
                    gamertag=gamertag,
                    team_id=team_id,
                    kill_times_ms=list(current),
                )
            )

    return sorted(streaks, key=lambda s: s.kill_times_ms[0])


# ─────────────────────────────────────────────────────────────────────────────
# Visualisation
# ─────────────────────────────────────────────────────────────────────────────


def plot_dominance_chart(  # noqa: PLR0913
    buckets: list[DominanceBucket],
    streaks: list[KillStreak],
    kill_events: list[dict[str, Any]],
    xuid_to_team: dict[str, int],
    my_team_id: int,
    duration_s: float,
    bucket_s: int = 30,
    opts: PlotOptions | None = None,
) -> go.Figure | None:
    """Construit la figure Plotly avec deux panneaux liés par l'axe temps.

    Panneau 1 (haut) — barres de dominance par tranche (tug-of-war).
    Panneau 2 (bas) — kill feed individuel avec séries annotées.
    """
    _opts = opts if opts is not None else PlotOptions()
    lang = _opts.lang
    height = _opts.height_px
    if not buckets or not kill_events:
        return None

    # Préparer les données
    t_centers, bar_widths, enemy_ys, my_ys, colors_enemy, colors_my = prepare_bar_data(buckets)
    my_xs, enemy_xs, my_tips, enemy_tips = prepare_kill_feed(
        kill_events,
        streaks,
        xuid_to_team,
        my_team_id,
    )
    cumul_my, cumul_enemy = 0, 0
    cumul_my_list: list[int] = []
    cumul_enemy_list: list[int] = []
    for b in buckets:
        cumul_my += b.my_kills
        cumul_enemy += b.enemy_kills
        cumul_my_list.append(cumul_my)
        cumul_enemy_list.append(cumul_enemy)

    # Construire la figure (2 panneaux liés par l'axe X)
    fig = make_subplots(
        rows=2,
        cols=1,
        shared_xaxes=True,
        row_heights=[0.68, 0.32],
        vertical_spacing=0.03,
    )
    formatted_times = [fmt_s(t) for t in t_centers]

    # Panneau 1 : barres empilées
    fig.add_trace(
        go.Bar(
            x=t_centers,
            y=enemy_ys,
            width=bar_widths,
            marker_color=colors_enemy,
            marker_line_width=0,
            name=viz_t("trace_opponents", lang),
            customdata=formatted_times,
            hovertemplate="<b>%{customdata}</b><br>"
            + viz_t("trace_opponents", lang)
            + " : %{y:.0f}%<extra></extra>",
            showlegend=False,
        ),
        row=1,
        col=1,
    )
    fig.add_trace(
        go.Bar(
            x=t_centers,
            y=my_ys,
            width=bar_widths,
            marker_color=colors_my,
            marker_line_width=0,
            name=viz_t("trace_my_team", lang),
            customdata=formatted_times,
            hovertemplate="<b>%{customdata}</b><br>"
            + viz_t("trace_my_team", lang)
            + " : %{y:.0f}%<extra></extra>",
            showlegend=False,
        ),
        row=1,
        col=1,
    )

    # Ligne de parité + labels de zone
    fig.add_hline(
        y=50, line_dash="dot", line_color="rgba(255,255,255,0.35)", line_width=1, row=1, col=1
    )
    _x_label = duration_s * 0.015
    for y_pos, text, color in [
        (82, "trace_my_team", MY_TEAM_COLOR),
        (18, "trace_opponents", ENEMY_COLOR),
    ]:
        fig.add_annotation(
            x=_x_label,
            y=y_pos,
            text=viz_t(text, lang),
            font={"color": color, "size": 10},
            showarrow=False,
            xanchor="left",
            yanchor="middle",
            xref="x",
            yref="y",
        )

    # Frags cumulés + kill feed points + séries
    add_cumul_annotations(fig, t_centers, cumul_my_list, cumul_enemy_list)
    if my_xs:
        fig.add_trace(
            go.Scatter(
                x=my_xs,
                y=[143.0] * len(my_xs),
                mode="markers",
                marker={"color": MY_TEAM_COLOR, "size": NORMAL_MARKER_SIZE, "opacity": 0.65},
                text=my_tips,
                hovertemplate="%{text}<extra></extra>",
                showlegend=False,
            ),
            row=1,
            col=1,
        )
    if enemy_xs:
        fig.add_trace(
            go.Scatter(
                x=enemy_xs,
                y=[0.65] * len(enemy_xs),
                mode="markers",
                marker={"color": ENEMY_COLOR, "size": NORMAL_MARKER_SIZE, "opacity": 0.65},
                text=enemy_tips,
                hovertemplate="%{text}<extra></extra>",
                showlegend=False,
            ),
            row=2,
            col=1,
        )
    add_streaks(fig, streaks, my_team_id, lang)
    configure_axes(fig, duration_s, bucket_s, height)
    return fig


__all__ = [
    "DominanceBucket",
    "KillStreak",
    "compute_dominance_buckets",
    "detect_streaks",
    "plot_dominance_chart",
]
