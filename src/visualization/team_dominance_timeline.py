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
from dataclasses import dataclass, field
from typing import Any

import plotly.graph_objects as go
from plotly.subplots import make_subplots

# ─────────────────────────────────────────────────────────────────────────────
# Constantes visuelles (palette HALO_COLORS)
# ─────────────────────────────────────────────────────────────────────────────

# Couleurs mises à jour pour respecter la palette Okabe-Ito (accessibilité daltonisme).
# Anciens code hexadécimaux (deuteranopie/protanopie-incompatibles) :
#   _MY_TEAM_COLOR: #3DFFB5 (vert néon Halo) → #0072B2 (bleu Okabe-Ito)
#   _ENEMY_COLOR:   #FF4D6D (rose-rouge)      → #D55E00 (vermillon Okabe-Ito)
_MY_TEAM_COLOR = "#0072B2"   # Bleu Okabe-Ito — mon équipe
_ENEMY_COLOR = "#D55E00"     # Vermillon Okabe-Ito — équipe adverse
_MY_TEAM_RGBA = "rgba(0, 114, 178, 0.80)"
_ENEMY_RGBA = "rgba(213, 94, 0, 0.80)"
_NEUTRAL_RGBA = "rgba(255, 255, 255, 0.10)"
_BG_TRANSPARENT = "rgba(0,0,0,0)"

_STREAK_MARKER_SIZE = 12
_NORMAL_MARKER_SIZE = 7
_STREAK_LINE_WIDTH = 2


# ─────────────────────────────────────────────────────────────────────────────
# Structures de données
# ─────────────────────────────────────────────────────────────────────────────


@dataclass
class DominanceBucket:
    """Stats d'une tranche de temps du match."""

    t_start_s: float
    t_end_s: float
    my_kills: int = 0
    enemy_kills: int = 0

    @property
    def t_center_s(self) -> float:
        return (self.t_start_s + self.t_end_s) / 2.0

    @property
    def total_kills(self) -> int:
        return self.my_kills + self.enemy_kills

    @property
    def my_share(self) -> float:
        """% de kills pour mon équipe (0–100). 50 si aucun kill."""
        if self.total_kills == 0:
            return 50.0
        return 100.0 * self.my_kills / self.total_kills

    @property
    def enemy_share(self) -> float:
        return 100.0 - self.my_share


@dataclass
class KillStreak:
    """Série de kills individuelle détectée dans un match."""

    xuid: str
    gamertag: str
    team_id: int | None
    kill_times_ms: list[int] = field(default_factory=list)

    @property
    def kills_count(self) -> int:
        return len(self.kill_times_ms)

    @property
    def t_start_s(self) -> float:
        return self.kill_times_ms[0] / 1000.0

    @property
    def t_end_s(self) -> float:
        return self.kill_times_ms[-1] / 1000.0

    @property
    def t_center_s(self) -> float:
        return (self.kill_times_ms[0] + self.kill_times_ms[-1]) / 2000.0


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


def detect_streaks(
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
# Utilitaires
# ─────────────────────────────────────────────────────────────────────────────


def _fmt_s(total_s: float) -> str:
    """Formate des secondes en M:SS."""
    s = int(total_s)
    return f"{s // 60}:{s % 60:02d}"


# ─────────────────────────────────────────────────────────────────────────────
# Visualisation
# ─────────────────────────────────────────────────────────────────────────────


def plot_dominance_chart(
    buckets: list[DominanceBucket],
    streaks: list[KillStreak],
    kill_events: list[dict[str, Any]],
    xuid_to_team: dict[str, int],
    my_team_id: int,
    duration_s: float,
    bucket_s: int = 30,
    height: int = 360,
) -> go.Figure | None:
    """Construit la figure Plotly avec deux panneaux liés par l'axe temps.

    Panneau 1 (haut) — barres de dominance par tranche (tug-of-war) :
      Chaque barre = 1 tranche. Zone verte = % frags de mon équipe,
      zone rouge = % frags adverses. Ligne pointillée à 50 %.

    Panneau 2 (bas) — kill feed individuel :
      Cercles positionnés sur l'axe temps. Deux voies :
        y=1 → kills de mon équipe (vert)
        y=0 → kills adverses (rouge)
      Les séries sont mises en valeur : cercles plus grands, ligne de liaison,
      label flottant avec gamertag et nombre de kills.

    Returns:
        Figure Plotly ou None si les données sont insuffisantes.
    """
    if not buckets or not kill_events:
        return None

    # ── Données panneau 1 : barres stacked ──────────────────────────────────
    t_centers = [b.t_center_s for b in buckets]
    bar_widths = [(b.t_end_s - b.t_start_s) * 0.97 for b in buckets]  # micro-gap visuel
    enemy_ys, my_ys = [], []
    colors_enemy, colors_my = [], []

    for b in buckets:
        if b.total_kills == 0:
            enemy_ys.append(50.0)
            my_ys.append(50.0)
            colors_enemy.append(_NEUTRAL_RGBA)
            colors_my.append(_NEUTRAL_RGBA)
        else:
            enemy_ys.append(b.enemy_share)
            my_ys.append(b.my_share)
            colors_enemy.append(_ENEMY_RGBA)
            colors_my.append(_MY_TEAM_RGBA)

    # ── Données panneau 2 : kill feed ────────────────────────────────────────
    # Timestamps appartenant à une série → points agrandis, pas dans le scatter normal
    streak_times_my: set[int] = set()
    streak_times_enemy: set[int] = set()
    for s in streaks:
        target = streak_times_my if s.team_id == my_team_id else streak_times_enemy
        target.update(s.kill_times_ms)

    my_xs: list[float] = []
    enemy_xs: list[float] = []
    my_tips: list[str] = []
    enemy_tips: list[str] = []

    for e in kill_events:
        t_ms = e.get("time_ms")
        if t_ms is None:
            continue
        xuid = str(e.get("xuid", "")).strip()
        team_id = xuid_to_team.get(xuid)
        t_s = int(t_ms) / 1000.0
        gamertag = str(e.get("gamertag", "") or xuid)
        tip = f"{gamertag} — {_fmt_s(t_s)}"

        if team_id == my_team_id:
            if int(t_ms) not in streak_times_my:
                my_xs.append(t_s)
                my_tips.append(tip)
        else:
            if int(t_ms) not in streak_times_enemy:
                enemy_xs.append(t_s)
                enemy_tips.append(tip)

    # ── Construction figure ──────────────────────────────────────────────────
    fig = make_subplots(
        rows=2,
        cols=1,
        shared_xaxes=True,
        row_heights=[0.62, 0.38],
        vertical_spacing=0.03,
    )

    # ── Panneau 1 : barres empilées ──────────────────────────────────────────
    formatted_times = [_fmt_s(t) for t in t_centers]

    fig.add_trace(
        go.Bar(
            x=t_centers,
            y=enemy_ys,
            width=bar_widths,
            marker_color=colors_enemy,
            marker_line_width=0,
            name="Adversaires",
            customdata=formatted_times,
            hovertemplate="<b>%{customdata}</b><br>Adversaires : %{y:.0f}%<extra></extra>",
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
            name="Mon équipe",
            customdata=formatted_times,
            hovertemplate="<b>%{customdata}</b><br>Mon équipe : %{y:.0f}%<extra></extra>",
            showlegend=False,
        ),
        row=1,
        col=1,
    )

    # Ligne de parité à 50 %
    fig.add_hline(
        y=50,
        line_dash="dot",
        line_color="rgba(255,255,255,0.35)",
        line_width=1,
        row=1,
        col=1,
    )

    # Labels de zone (gauche)
    _x_label = duration_s * 0.015
    fig.add_annotation(
        x=_x_label,
        y=82,
        text="Mon équipe",
        font={"color": _MY_TEAM_COLOR, "size": 10},
        showarrow=False,
        xanchor="left",
        yanchor="middle",
        xref="x",
        yref="y",
    )
    fig.add_annotation(
        x=_x_label,
        y=18,
        text="Adversaires",
        font={"color": _ENEMY_COLOR, "size": 10},
        showarrow=False,
        xanchor="left",
        yanchor="middle",
        xref="x",
        yref="y",
    )

    # ── Panneau 2 : kill feed (points normaux) ───────────────────────────────
    if my_xs:
        fig.add_trace(
            go.Scatter(
                x=my_xs,
                y=[1.0] * len(my_xs),
                mode="markers",
                marker={"color": _MY_TEAM_COLOR, "size": _NORMAL_MARKER_SIZE, "opacity": 0.65},
                text=my_tips,
                hovertemplate="%{text}<extra></extra>",
                showlegend=False,
            ),
            row=2,
            col=1,
        )
    if enemy_xs:
        fig.add_trace(
            go.Scatter(
                x=enemy_xs,
                y=[0.0] * len(enemy_xs),
                mode="markers",
                marker={"color": _ENEMY_COLOR, "size": _NORMAL_MARKER_SIZE, "opacity": 0.65},
                text=enemy_tips,
                hovertemplate="%{text}<extra></extra>",
                showlegend=False,
            ),
            row=2,
            col=1,
        )

    # ── Panneau 2 : séries (ligne + marqueurs larges + label) ────────────────
    for streak in streaks:
        is_mine = streak.team_id == my_team_id
        color = _MY_TEAM_COLOR if is_mine else _ENEMY_COLOR
        y_lane = 1.0 if is_mine else 0.0
        label_y = 1.50 if is_mine else -0.50
        x_pts = [t / 1000.0 for t in streak.kill_times_ms]

        # Ligne + marqueurs agrandis
        fig.add_trace(
            go.Scatter(
                x=x_pts,
                y=[y_lane] * len(x_pts),
                mode="lines+markers",
                line={"color": color, "width": _STREAK_LINE_WIDTH},
                marker={
                    "color": color,
                    "size": _STREAK_MARKER_SIZE,
                    "line": {"color": "rgba(255,255,255,0.6)", "width": 1.5},
                },
                hovertemplate=(
                    f"<b>{streak.gamertag}</b> — série ×{streak.kills_count}"
                    "<br>%{x:.0f}s<extra></extra>"
                ),
                showlegend=False,
            ),
            row=2,
            col=1,
        )

        # Label flottant centré sur la série
        fig.add_annotation(
            x=streak.t_center_s,
            y=label_y,
            text=f"<b>{streak.gamertag}</b> ×{streak.kills_count}",
            font={"color": color, "size": 10},
            showarrow=False,
            xanchor="center",
            yanchor="middle",
            bgcolor="rgba(15, 20, 35, 0.88)",
            borderpad=3,
            xref="x",
            yref="y2",
        )

    # ── Layout global ────────────────────────────────────────────────────────
    tick_step = 60
    tick_vals = list(range(0, int(duration_s) + tick_step, tick_step))
    tick_text = [_fmt_s(v) for v in tick_vals]

    fig.update_layout(
        barmode="stack",
        bargap=0,
        bargroupgap=0,
        height=height,
        paper_bgcolor=_BG_TRANSPARENT,
        plot_bgcolor=_BG_TRANSPARENT,
        margin={"l": 8, "r": 8, "t": 8, "b": 8},
        font={"color": "rgba(245,248,255,0.65)", "size": 11},
        hovermode="closest",
    )

    # Y panneau 1
    fig.update_yaxes(
        range=[0, 100],
        showgrid=False,
        zeroline=False,
        showticklabels=False,
        row=1,
        col=1,
    )
    # X panneau 1 (caché, partagé avec panneau 2)
    fig.update_xaxes(
        showticklabels=False,
        showgrid=False,
        zeroline=False,
        range=[-bucket_s * 0.5, duration_s + bucket_s * 0.5],
        row=1,
        col=1,
    )

    # Y panneau 2
    fig.update_yaxes(
        range=[-0.85, 1.85],
        showgrid=False,
        zeroline=False,
        showticklabels=False,
        row=2,
        col=1,
    )
    # X panneau 2 (affiche les temps)
    fig.update_xaxes(
        tickvals=tick_vals,
        ticktext=tick_text,
        showgrid=False,
        zeroline=False,
        range=[-bucket_s * 0.5, duration_s + bucket_s * 0.5],
        tickfont={"size": 10, "color": "rgba(245,248,255,0.5)"},
        row=2,
        col=1,
    )

    return fig


__all__ = [
    "DominanceBucket",
    "KillStreak",
    "compute_dominance_buckets",
    "detect_streaks",
    "plot_dominance_chart",
]
