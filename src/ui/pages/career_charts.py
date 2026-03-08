"""Page Carrière — Fonctions de construction des graphiques XP et projections."""

from __future__ import annotations

import logging
from datetime import datetime

import plotly.graph_objects as go

from src.config import THEME_COLORS
from src.ui.career_ranks import format_career_rank_label_fr
from src.ui.career_ranks import get_rank_info as get_meta_rank_info
from src.ui.components.career_progress_circle import XP_HERO_TOTAL
from src.ui.i18n import t
from src.visualization.theme import apply_halo_plot_style

logger = logging.getLogger(__name__)

# Palette pour les courbes des autres joueurs (distincte des 4 traces existantes)
# Évite : accent (cyan), #CE93D8 (violet estimé), #FFA726 (orange proj), #66BB6A (vert opt)
_OTHER_PLAYERS_COLORS: list[str] = [
    "#EF5350",  # rouge
    "#29B6F6",  # bleu clair
    "#FFCA28",  # ambre
    "#26C6DA",  # cyan
    "#FF7043",  # orange-rouge
    "#AB47BC",  # violet foncé
]


# ── Traces XP principale ────────────────────────────────────────────────────


def _build_main_hover_texts(history: list[dict]) -> list[str]:
    """Construit les textes de survol pour la trace XP principale."""
    hover_texts = []
    for h in history:
        _meta = get_meta_rank_info(h.get("rank", 0))
        if _meta:
            label = _meta.full_label_fr
        else:
            logger.debug(
                "Rang %d introuvable dans metadata.duckdb (hover) — fallback DB",
                h.get("rank", 0),
            )
            name = h.get("rank_name", "")
            tier = h.get("rank_tier", "")
            label = format_career_rank_label_fr(tier=tier, title=name, grade=None)
        hover_texts.append(
            t("career_rank_hover", rank=h["rank"], label=label, xp=f"{h['xp_total']:,}")
        )
    return hover_texts


def _add_main_xp_trace(fig: go.Figure, history: list[dict]) -> None:
    """Ajoute la trace XP réelle du joueur courant."""
    dates = [h["recorded_at"] for h in history]
    xp_totals = [h["xp_total"] or 0 for h in history]
    hover_texts = _build_main_hover_texts(history)
    fig.add_trace(
        go.Scatter(
            x=dates,
            y=xp_totals,
            mode="lines+markers",
            name=t("career_xp_total"),
            line={"color": THEME_COLORS.accent, "width": 2},
            marker={"size": 6, "color": THEME_COLORS.accent},
            hovertext=hover_texts,
            hoverinfo="text",
        )
    )


def _add_estimated_xp_trace(fig: go.Figure, estimated_curve: list | None) -> None:
    """Ajoute la trace XP estimée pré-sync (pointillés)."""
    if not estimated_curve:
        return
    est_dates = [pt[0] for pt in estimated_curve]
    est_xp = [pt[1] for pt in estimated_curve]
    est_hover = [
        t("career_xp_estimated_hover", date=str(pt[0])[:10], xp=f"{pt[1]:,}")
        for pt in estimated_curve
    ]
    fig.add_trace(
        go.Scatter(
            x=est_dates,
            y=est_xp,
            mode="lines",
            name=t("career_xp_estimated"),
            line={"color": "#CE93D8", "width": 2, "dash": "dot"},
            hovertext=est_hover,
            hoverinfo="text",
        )
    )


# ── Traces autres joueurs ────────────────────────────────────────────────────


def _add_single_other_player_proj_traces(
    fig: go.Figure, player: dict, gamertag: str, color: str
) -> None:
    """Ajoute les traces de projection héros et optimiste pour un joueur comparatif."""
    p_hero_proj = player.get("hero_proj")
    p_opt_proj = player.get("optimistic_proj")
    if p_hero_proj:
        fig.add_trace(
            go.Scatter(
                x=[pt[0] for pt in p_hero_proj],
                y=[pt[1] for pt in p_hero_proj],
                mode="lines",
                name=t("career_projection_other_hero", gamertag=gamertag),
                line={"color": color, "width": 1.5, "dash": "dash"},
                hovertext=[
                    t(
                        "career_projection_other_hover",
                        gamertag=gamertag,
                        date=str(pt[0])[:10],
                        xp=f"{pt[1]:,}",
                    )
                    for pt in p_hero_proj
                ],
                hoverinfo="text",
                visible="legendonly",
            )
        )
    if p_opt_proj:
        fig.add_trace(
            go.Scatter(
                x=[pt[0] for pt in p_opt_proj],
                y=[pt[1] for pt in p_opt_proj],
                mode="lines",
                name=t("career_projection_other_optimistic", gamertag=gamertag),
                line={"color": color, "width": 1.5, "dash": "dashdot"},
                hovertext=[
                    t(
                        "career_projection_other_hover",
                        gamertag=gamertag,
                        date=str(pt[0])[:10],
                        xp=f"{pt[1]:,}",
                    )
                    for pt in p_opt_proj
                ],
                hoverinfo="text",
                visible="legendonly",
            )
        )


def _add_single_other_player_traces(fig: go.Figure, player: dict, color: str) -> None:
    """Ajoute les traces réelle, estimée et projections d'un joueur comparatif."""
    p_gamertag = player["gamertag"]
    p_history = player["history"]
    p_hover = [
        t(
            "career_xp_other_player_hover",
            gamertag=p_gamertag,
            date=str(h["recorded_at"])[:10],
            xp=f"{h['xp_total'] or 0:,}",
        )
        for h in p_history
    ]
    fig.add_trace(
        go.Scatter(
            x=[h["recorded_at"] for h in p_history],
            y=[h["xp_total"] or 0 for h in p_history],
            mode="lines+markers",
            name=t("career_xp_other_player", gamertag=p_gamertag),
            line={"color": color, "width": 1.5},
            marker={"size": 5, "color": color},
            hovertext=p_hover,
            hoverinfo="text",
            visible="legendonly",
        )
    )
    p_estimated = player.get("estimated_curve")
    if p_estimated:
        est_hover = [
            t(
                "career_xp_other_estimated_hover",
                gamertag=p_gamertag,
                date=str(pt[0])[:10],
                xp=f"{pt[1]:,}",
            )
            for pt in p_estimated
        ]
        fig.add_trace(
            go.Scatter(
                x=[pt[0] for pt in p_estimated],
                y=[pt[1] for pt in p_estimated],
                mode="lines",
                name=t("career_xp_other_estimated", gamertag=p_gamertag),
                line={"color": color, "width": 1.5, "dash": "dot"},
                hovertext=est_hover,
                hoverinfo="text",
                visible="legendonly",
            )
        )
    _add_single_other_player_proj_traces(fig, player, p_gamertag, color)


def _add_other_player_traces(fig: go.Figure, other_players: list[dict] | None) -> None:
    """Ajoute les traces de tous les joueurs comparatifs."""
    if not other_players:
        return
    for idx, player in enumerate(other_players):
        color = _OTHER_PLAYERS_COLORS[idx % len(_OTHER_PLAYERS_COLORS)]
        _add_single_other_player_traces(fig, player, color)


def _add_xp_traces(
    fig: go.Figure, history: list[dict], estimated_curve: list | None, other_players: list | None
) -> None:
    """Ajoute les traces XP réel, estimé et autres joueurs sur la figure."""
    _add_main_xp_trace(fig, history)
    _add_estimated_xp_trace(fig, estimated_curve)
    _add_other_player_traces(fig, other_players)


# ── Traces de projection ────────────────────────────────────────────────────


def _add_projection_traces(
    fig: go.Figure,
    hero_projection: list | None,
    optimistic_projection: list | None,
    *,
    is_max_rank: bool,
) -> None:
    """Ajoute les traces de projection (héros + optimiste) et la ligne seuil."""
    if hero_projection and not is_max_rank:
        proj_hover = [
            t("career_projection_hero_hover", date=str(pt[0])[:10], xp=f"{pt[1]:,}")
            for pt in hero_projection
        ]
        fig.add_trace(
            go.Scatter(
                x=[pt[0] for pt in hero_projection],
                y=[pt[1] for pt in hero_projection],
                mode="lines",
                name=t("career_projection_hero"),
                line={"color": "#FFA726", "width": 2, "dash": "dash"},
                hovertext=proj_hover,
                hoverinfo="text",
                visible="legendonly",
            )
        )
    if optimistic_projection and not is_max_rank:
        opt_hover = [
            t("career_projection_optimistic_hover", date=str(pt[0])[:10], xp=f"{pt[1]:,}")
            for pt in optimistic_projection
        ]
        fig.add_trace(
            go.Scatter(
                x=[pt[0] for pt in optimistic_projection],
                y=[pt[1] for pt in optimistic_projection],
                mode="lines",
                name=t("career_projection_optimistic"),
                line={"color": "#66BB6A", "width": 2, "dash": "dashdot"},
                hovertext=opt_hover,
                hoverinfo="text",
                visible="legendonly",
            )
        )
    if (hero_projection or optimistic_projection) and not is_max_rank:
        fig.add_hline(
            y=XP_HERO_TOTAL,
            line_dash="dot",
            line_color="rgba(255, 215, 0, 0.3)",
            line_width=1,
            annotation_text=t("career_hero_threshold"),
            annotation_position="top left",
            annotation_font_size=10,
            annotation_font_color="rgba(255, 215, 0, 0.5)",
        )


# ── Graphique XP complet ────────────────────────────────────────────────────


def _create_xp_history_chart(  # noqa: PLR0913
    history: list[dict],
    *,
    estimated_curve: list[tuple[datetime, int]] | None = None,
    hero_projection: list[tuple[datetime, int]] | None = None,
    optimistic_projection: list[tuple[datetime, int]] | None = None,
    is_max_rank: bool = False,
    other_players: list[dict] | None = None,
) -> go.Figure | None:
    """Crée un graphique d'historique XP total dans le temps.

    Traces :
    1. XP réel (accent, lignes + marqueurs)
    2. XP estimé pré-sync (pointillés, couleur atténuée)
    3. Autres joueurs (lignes, couleurs distinctes, masquées par défaut)
    4. Projection → Héros (tirets, orange, masquée par défaut)
    5. Projection optimiste (tirets-points, vert, masquée par défaut)
    + Ligne horizontale au seuil Héros (si projections actives)
    """
    if len(history) < 2:
        return None

    bg_rgb = THEME_COLORS.bg_plot
    bg_color = f"rgb({bg_rgb[0]}, {bg_rgb[1]}, {bg_rgb[2]})"
    fig = go.Figure()

    _add_xp_traces(fig, history, estimated_curve, other_players)
    _add_projection_traces(fig, hero_projection, optimistic_projection, is_max_rank=is_max_rank)

    fig.update_layout(
        title=t("career_xp_progress"),
        xaxis_title=t("col_date"),
        yaxis_title=t("career_xp_total"),
        paper_bgcolor=bg_color,
        plot_bgcolor=bg_color,
        font={"color": "white"},
        height=400,
        xaxis={"gridcolor": "rgba(255,255,255,0.05)"},
        yaxis={"gridcolor": "rgba(255,255,255,0.1)"},
        legend={
            "orientation": "h",
            "yanchor": "top",
            "y": -0.18,
            "xanchor": "center",
            "x": 0.5,
            "font": {"size": 11},
        },
        margin={"t": 40, "b": 80, "l": 60, "r": 20},
    )
    apply_halo_plot_style(fig)
    return fig
