"""Helpers internes pour team_dominance_timeline.

Contient les constantes visuelles, structures de données et fonctions utilitaires
pour la construction du graphique de dominance d'équipe.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

import plotly.graph_objects as go

from src.ui.i18n.viz import viz_t
from src.visualization._plot_options import DEFAULT_THEME

# ─────────────────────────────────────────────────────────────────────────────
# Constantes visuelles (palette HALO_COLORS)
# ─────────────────────────────────────────────────────────────────────────────

# Couleurs mises à jour pour respecter la palette Okabe-Ito (accessibilité daltonisme).
# Anciens code hexadécimaux (deuteranopie/protanopie-incompatibles) :
#   _MY_TEAM_COLOR: #3DFFB5 (vert néon Halo) → #0072B2 (bleu Okabe-Ito)
#   _ENEMY_COLOR:   #FF4D6D (rose-rouge)      → #D55E00 (vermillon Okabe-Ito)
MY_TEAM_COLOR = "#0072B2"  # Bleu Okabe-Ito — mon équipe
ENEMY_COLOR = "#D55E00"  # Vermillon Okabe-Ito — équipe adverse
MY_TEAM_RGBA = "rgba(0, 114, 178, 0.80)"
ENEMY_RGBA = "rgba(213, 94, 0, 0.80)"
NEUTRAL_RGBA = "rgba(255, 255, 255, 0.10)"
BG_TRANSPARENT = "rgba(0,0,0,0)"

STREAK_MARKER_SIZE = 12
NORMAL_MARKER_SIZE = 7
STREAK_LINE_WIDTH = 2

LEAD_MY_COLOR = "#009E73"  # Vert Okabe-Ito — avantage cumulé mon équipe
LEAD_MY_BG = DEFAULT_THEME.zone_win_bg
LEAD_ENEMY_BG = DEFAULT_THEME.zone_loss_bg


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
# Utilitaires
# ─────────────────────────────────────────────────────────────────────────────


def fmt_s(total_s: float) -> str:
    """Formate des secondes en M:SS."""
    s = int(total_s)
    return f"{s // 60}:{s % 60:02d}"


# ─────────────────────────────────────────────────────────────────────────────
# Helpers plot_dominance_chart
# ─────────────────────────────────────────────────────────────────────────────


def prepare_bar_data(
    buckets: list[DominanceBucket],
) -> tuple[list[float], list[float], list[float], list[float], list[str], list[str]]:
    """Prépare les données des barres empilées du panneau 1."""
    t_centers = [b.t_center_s for b in buckets]
    bar_widths = [(b.t_end_s - b.t_start_s) * 0.97 for b in buckets]
    enemy_ys: list[float] = []
    my_ys: list[float] = []
    colors_enemy: list[str] = []
    colors_my: list[str] = []
    for b in buckets:
        if b.total_kills == 0:
            enemy_ys.append(50.0)
            my_ys.append(50.0)
            colors_enemy.append(NEUTRAL_RGBA)
            colors_my.append(NEUTRAL_RGBA)
        else:
            enemy_ys.append(b.enemy_share)
            my_ys.append(b.my_share)
            colors_enemy.append(ENEMY_RGBA)
            colors_my.append(MY_TEAM_RGBA)
    return t_centers, bar_widths, enemy_ys, my_ys, colors_enemy, colors_my


def prepare_kill_feed(
    kill_events: list[dict[str, Any]],
    streaks: list[KillStreak],
    xuid_to_team: dict[str, int],
    my_team_id: int,
) -> tuple[list[float], list[float], list[str], list[str]]:
    """Prépare les points kill feed (hors séries) pour le panneau 2."""
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
        tip = f"{gamertag} — {fmt_s(t_s)}"
        if team_id == my_team_id:
            if int(t_ms) not in streak_times_my:
                my_xs.append(t_s)
                my_tips.append(tip)
        elif int(t_ms) not in streak_times_enemy:
            enemy_xs.append(t_s)
            enemy_tips.append(tip)
    return my_xs, enemy_xs, my_tips, enemy_tips


def add_cumul_annotations(
    fig: go.Figure,
    t_centers: list[float],
    cumul_my_list: list[int],
    cumul_enemy_list: list[int],
) -> None:
    """Ajoute les annotations de frags cumulés avec highlight conditionnel."""
    for i, t in enumerate(t_centers):
        my_val = cumul_my_list[i]
        en_val = cumul_enemy_list[i]
        my_lead = my_val > en_val
        enemy_lead = en_val > my_val
        fig.add_annotation(
            x=t,
            y=110.0,
            text=f"<b>{my_val}</b>",
            font={
                "color": LEAD_MY_COLOR if my_lead else MY_TEAM_COLOR,
                "size": 12,
                "family": "Arial Black, Arial, sans-serif",
            },
            showarrow=False,
            xanchor="center",
            yanchor="middle",
            bgcolor=LEAD_MY_BG if my_lead else "rgba(0,0,0,0)",
            bordercolor=LEAD_MY_COLOR if my_lead else "rgba(0,0,0,0)",
            borderwidth=2 if my_lead else 0,
            borderpad=3,
            xref="x",
            yref="y",
        )
        fig.add_annotation(
            x=t,
            y=-10.0,
            text=f"<b>{en_val}</b>",
            font={
                "color": ENEMY_COLOR,
                "size": 12,
                "family": "Arial Black, Arial, sans-serif",
            },
            showarrow=False,
            xanchor="center",
            yanchor="middle",
            bgcolor=LEAD_ENEMY_BG if enemy_lead else "rgba(0,0,0,0)",
            bordercolor=ENEMY_COLOR if enemy_lead else "rgba(0,0,0,0)",
            borderwidth=2 if enemy_lead else 0,
            borderpad=3,
            xref="x",
            yref="y",
        )


def add_streaks(
    fig: go.Figure,
    streaks: list[KillStreak],
    my_team_id: int,
    lang: str,
) -> None:
    """Ajoute les visualisations de séries (lignes + marqueurs + labels)."""
    for streak in streaks:
        is_mine = streak.team_id == my_team_id
        color = MY_TEAM_COLOR if is_mine else ENEMY_COLOR
        if is_mine:
            y_lane, label_y, target_row, yref_annot = 143.0, 160.0, 1, "y"
        else:
            y_lane, label_y, target_row, yref_annot = 0.65, 1.25, 2, "y2"

        x_pts = [t / 1000.0 for t in streak.kill_times_ms]
        formatted_pts = [fmt_s(t / 1000.0) for t in streak.kill_times_ms]
        fig.add_trace(
            go.Scatter(
                x=x_pts,
                y=[y_lane] * len(x_pts),
                mode="lines+markers",
                line={"color": color, "width": STREAK_LINE_WIDTH},
                marker={
                    "color": color,
                    "size": STREAK_MARKER_SIZE,
                    "line": {"color": "rgba(255,255,255,0.6)", "width": 1.5},
                },
                customdata=formatted_pts,
                hovertemplate=(
                    f"<b>{streak.gamertag}</b> — {viz_t('label_streak', lang)} ×{streak.kills_count}"
                    "<br>%{customdata}<extra></extra>"
                ),
                showlegend=False,
            ),
            row=target_row,
            col=1,
        )
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
            yref=yref_annot,
        )


def configure_axes(
    fig: go.Figure,
    duration_s: float,
    bucket_s: int,
    height: int,
) -> None:
    """Configure le layout global et les axes des deux panneaux."""
    tick_step = 60
    tick_vals = list(range(0, int(duration_s) + tick_step, tick_step))
    tick_text = [fmt_s(v) for v in tick_vals]

    fig.update_layout(
        barmode="stack",
        bargap=0,
        bargroupgap=0,
        height=height,
        paper_bgcolor=BG_TRANSPARENT,
        plot_bgcolor=BG_TRANSPARENT,
        margin={"l": 8, "r": 8, "t": 8, "b": 8},
        font={"color": "rgba(245,248,255,0.65)", "size": 11},
        hovermode="closest",
    )
    x_range = [-bucket_s * 0.5, duration_s + bucket_s * 0.5]
    fig.update_yaxes(
        range=[-18, 170],
        showgrid=False,
        zeroline=False,
        showticklabels=False,
        row=1,
        col=1,
    )
    fig.update_xaxes(
        showticklabels=False,
        showgrid=False,
        zeroline=False,
        range=x_range,
        row=1,
        col=1,
    )
    fig.update_yaxes(
        range=[-0.5, 1.7],
        showgrid=False,
        zeroline=False,
        showticklabels=False,
        row=2,
        col=1,
    )
    fig.update_xaxes(
        tickvals=tick_vals,
        ticktext=tick_text,
        showgrid=False,
        zeroline=False,
        range=x_range,
        tickfont={"size": 13, "color": "rgba(245,248,255,0.75)"},
        row=2,
        col=1,
    )
