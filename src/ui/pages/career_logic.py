"""Page Carrière — Logique de calcul et projections."""

from __future__ import annotations

import logging
from datetime import datetime, timedelta

import plotly.graph_objects as go

from src.config import THEME_COLORS
from src.ui.career_ranks import format_career_rank_label_fr
from src.ui.components.career_progress_circle import XP_HERO_TOTAL
from src.ui.i18n import t
from src.visualization.theme import apply_halo_plot_style

logger = logging.getLogger(__name__)

# ── Constantes estimation / projection ──────────────────────────────────────
# XP des 10 défis hebdomadaires (source : Halopedia, post-CU32)
# 4 Normal × 50 + 3 Heroic × 100 + 3 Legendary × 150 = 950 XP/semaine
WEEKLY_CHALLENGE_XP: int = 950
# XP du défi quotidien (1 défi/jour × 500 XP, source : Halopedia)
DAILY_CHALLENGE_XP: int = 500
# Multiplicateur boost XP (consommable Double XP)
XP_BOOST_MULTIPLIER: float = 2.0
# Seuil d'inactivité en jours — les gaps plus longs sont exclus du rythme
INACTIVITY_GAP_DAYS: int = 14

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


# ── Estimation pré-sync ─────────────────────────────────────────────────────


def _compute_estimated_xp_curve(
    history: list[dict],
    pre_sync_match_dates: list[datetime],
) -> list[tuple[datetime, int]]:
    """Estime la courbe XP pour les matchs antérieurs au premier sync.

    Logique : l'XP total au 1er snapshot est réparti uniformément sur tous les
    matchs pré-sync (average = first_xp / n_pre_sync). On remonte dans le temps
    depuis le 1er snapshot en soustrayant cet XP moyen à chaque match.
    Cela garantit que la courbe part de ~0 au match le plus ancien.

    Returns:
        Liste de (date, xp_estimé) en ordre chronologique, se terminant
        au 1er point réel (inclus pour raccord visuel).
    """
    if not pre_sync_match_dates or not history:
        return []

    first_xp = history[0]["xp_total"] or 0
    first_sync_at = history[0]["recorded_at"]

    if first_xp <= 0:
        return []

    # Moyenne basée sur l'XP total au 1er snapshot réparti sur tous les matchs
    # pré-sync : garantit que la courbe part de ~0 au match le plus ancien.
    # (utiliser la moyenne post-sync introduirait un biais si le rythme change)
    n_pre = len(pre_sync_match_dates)
    avg_xp_per_match = first_xp / n_pre

    # Remonter dans le temps depuis le 1er snapshot
    curve: list[tuple[datetime, int]] = []
    current_xp = float(first_xp)

    # Parcourir les matchs pré-sync du plus récent au plus ancien
    for match_date in reversed(pre_sync_match_dates):
        current_xp -= avg_xp_per_match
        if current_xp < 0:
            current_xp = 0
        curve.append((match_date, int(current_xp)))

    # Remettre en ordre chronologique
    curve.reverse()

    # Ajouter le point de raccord (1er snapshot réel)
    curve.append((first_sync_at, first_xp))

    return curve


# ── Projection vers Héros ───────────────────────────────────────────────────


def _compute_active_xp_per_day(history: list[dict]) -> float:
    """Calcule le rythme d'XP par jour actif (hors gaps d'inactivité).

    Les périodes sans activité de plus de ``INACTIVITY_GAP_DAYS`` jours
    sont exclues du calcul pour ne pas sous-estimer le rythme réel.

    Returns:
        XP par jour actif, ou 0 si impossible à calculer.
    """
    if len(history) < 2:
        return 0.0

    total_active_days = 0.0
    first_xp = history[0]["xp_total"] or 0
    last_xp = history[-1]["xp_total"] or 0
    xp_delta = last_xp - first_xp
    if xp_delta <= 0:
        return 0.0

    for i in range(1, len(history)):
        prev_date = history[i - 1]["recorded_at"]
        curr_date = history[i]["recorded_at"]
        if not prev_date or not curr_date:
            continue
        gap = (curr_date - prev_date).total_seconds() / 86400.0
        # Exclure les gaps d'inactivité > seuil
        if gap <= INACTIVITY_GAP_DAYS:
            total_active_days += gap
        else:
            # On considère qu'il y a eu INACTIVITY_GAP_DAYS/2 jours actifs
            # dans le gap pour rester indulgent
            total_active_days += INACTIVITY_GAP_DAYS / 2

    if total_active_days <= 0:
        return 0.0

    return xp_delta / total_active_days


def _compute_hero_projections(
    xp_total: int,
    last_date: datetime,
    xp_per_active_day: float,
) -> tuple[list[tuple[datetime, int]], list[tuple[datetime, int]]]:
    """Calcule les courbes de projection vers le rang Héros.

    Args:
        xp_total: XP actuel du joueur.
        last_date: Dernière date connue.
        xp_per_active_day: Rythme d'XP par jour actif.

    Returns:
        Tuple (projection_normale, projection_optimiste).
        Chaque projection est une liste de (date, xp) hebdomadaire.
    """
    if xp_total >= XP_HERO_TOTAL or xp_per_active_day <= 0:
        return [], []

    xp_remaining = XP_HERO_TOTAL - xp_total

    # ── Projection normale : rythme réel ──
    normal_xp_per_day = xp_per_active_day
    normal_days = xp_remaining / normal_xp_per_day
    # Cap à 10 ans pour éviter des courbes absurdes
    normal_days = min(normal_days, 365 * 10)

    # ── Projection optimiste : (rythme réel + challenges/jour) × boost ──
    # challenges/jour = défis hebdo répartis sur 7 jours + 1 défi quotidien
    challenge_xp_per_day = WEEKLY_CHALLENGE_XP / 7.0 + DAILY_CHALLENGE_XP
    optimistic_xp_per_day = (xp_per_active_day + challenge_xp_per_day) * XP_BOOST_MULTIPLIER
    optimistic_days = xp_remaining / optimistic_xp_per_day
    optimistic_days = min(optimistic_days, 365 * 10)

    def _build_curve(days_total: float, xp_day: float) -> list[tuple[datetime, int]]:
        """Génère des points hebdomadaires du départ jusqu'à Hero."""
        points: list[tuple[datetime, int]] = []
        # Point de départ
        points.append((last_date, xp_total))

        weeks = int(days_total / 7) + 1
        for w in range(1, weeks + 1):
            day_offset = w * 7
            d = last_date + timedelta(days=day_offset)
            xp = int(min(xp_total + xp_day * day_offset, XP_HERO_TOTAL))
            points.append((d, xp))
            if xp >= XP_HERO_TOTAL:
                break

        # Si Hero n'a pas été atteint (rythme trop faible, cap 10 ans),
        # on ajoute le point d'arrivée seulement s'il est strictement
        # postérieur au dernier point (évite l'inversion chronologique
        # quand days_total < weeks*7)
        if points[-1][1] < XP_HERO_TOTAL:
            arrival = last_date + timedelta(days=days_total)
            final_xp = int(xp_total + xp_day * days_total)
            if arrival > points[-1][0]:
                points.append((arrival, min(final_xp, XP_HERO_TOTAL)))
            else:
                # Remplacer le dernier point par le point d'arrivée exact
                points[-1] = (arrival, min(final_xp, XP_HERO_TOTAL))

        return points

    normal_curve = _build_curve(normal_days, normal_xp_per_day)
    optimistic_curve = _build_curve(optimistic_days, optimistic_xp_per_day)

    return normal_curve, optimistic_curve


# ── Graphique XP enrichi ────────────────────────────────────────────────────


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

    dates = [h["recorded_at"] for h in history]
    xp_totals = [h["xp_total"] or 0 for h in history]

    # Texte au survol avec le rang
    hover_texts = []
    for h in history:
        name = h.get("rank_name", "")
        tier = h.get("rank_tier", "")
        label = format_career_rank_label_fr(tier=tier, title=name, grade=None)
        hover_texts.append(
            t("career_rank_hover", rank=h["rank"], label=label, xp=f"{h['xp_total']:,}")
        )

    bg_rgb = THEME_COLORS.bg_plot
    bg_color = f"rgb({bg_rgb[0]}, {bg_rgb[1]}, {bg_rgb[2]})"

    fig = go.Figure()

    # ── Trace 1 : XP réel ──
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

    # ── Trace 2 : XP estimé pré-sync ──
    if estimated_curve:
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

    # ── Traces autres joueurs (masquées par défaut) ──
    if other_players:
        for idx, player in enumerate(other_players):
            color = _OTHER_PLAYERS_COLORS[idx % len(_OTHER_PLAYERS_COLORS)]
            p_gamertag = player["gamertag"]
            p_history = player["history"]
            p_dates = [h["recorded_at"] for h in p_history]
            p_xp = [h["xp_total"] or 0 for h in p_history]
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
                    x=p_dates,
                    y=p_xp,
                    mode="lines+markers",
                    name=t("career_xp_other_player", gamertag=p_gamertag),
                    line={"color": color, "width": 1.5},
                    marker={"size": 5, "color": color},
                    hovertext=p_hover,
                    hoverinfo="text",
                    visible="legendonly",
                )
            )

    # ── Trace 3 : Projection → Héros (masquée par défaut) ──
    if hero_projection and not is_max_rank:
        proj_dates = [pt[0] for pt in hero_projection]
        proj_xp = [pt[1] for pt in hero_projection]

        proj_hover = [
            t("career_projection_hero_hover", date=str(pt[0])[:10], xp=f"{pt[1]:,}")
            for pt in hero_projection
        ]

        fig.add_trace(
            go.Scatter(
                x=proj_dates,
                y=proj_xp,
                mode="lines",
                name=t("career_projection_hero"),
                line={"color": "#FFA726", "width": 2, "dash": "dash"},
                hovertext=proj_hover,
                hoverinfo="text",
                visible="legendonly",
            )
        )

    # ── Trace 4 : Projection optimiste (masquée par défaut) ──
    if optimistic_projection and not is_max_rank:
        opt_dates = [pt[0] for pt in optimistic_projection]
        opt_xp = [pt[1] for pt in optimistic_projection]

        opt_hover = [
            t("career_projection_optimistic_hover", date=str(pt[0])[:10], xp=f"{pt[1]:,}")
            for pt in optimistic_projection
        ]

        fig.add_trace(
            go.Scatter(
                x=opt_dates,
                y=opt_xp,
                mode="lines",
                name=t("career_projection_optimistic"),
                line={"color": "#66BB6A", "width": 2, "dash": "dashdot"},
                hovertext=opt_hover,
                hoverinfo="text",
                visible="legendonly",
            )
        )

    # ── Ligne horizontale seuil Héros ──
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


def _get_pg_labels() -> dict[str, str]:
    """Retourne les labels traduits des groupes de playlist."""
    return {
        "ranked": t("career_ranked"),
        "arena": "Arena",
        "btb": "Big Team Battle",
        "tactical": t("career_tactical"),
        "social": "Social",
        "fun": "Fun",
    }


# ── Builders HTML — Top rencontres / antagonistes ────────────────────────────


def _kd_style(kills: int, deaths: int) -> str:
    """Retourne un style CSS selon le ratio K/D."""
    if deaths == 0:
        return "color:#33ffbf;font-weight:700;" if kills > 0 else ""
    ratio = kills / deaths
    if ratio >= 1.5:
        return "color:#33ffbf;font-weight:700;"
    if ratio <= 0.5:
        return "color:#ff9e6b;font-weight:700;"
    return ""


def build_encounters_table_html(rows: list[dict], title: str) -> str:
    """Construit un tableau HTML top joueurs croisés."""
    from src.ui.pages.match_table_html import gamertag_link

    header = (
        "<thead><tr>"
        f"<th class='os-sb-th' style='text-align:left'>#</th>"
        f"<th class='os-sb-th' style='text-align:left'>{t('col_player')}</th>"
        f"<th class='os-sb-th'>{t('col_encounters')}</th>"
        "</tr></thead>"
    )
    body_rows = []
    for i, r in enumerate(rows, 1):
        gt = gamertag_link(r["gamertag"]) if r.get("gamertag") else r.get("xuid", "—")[:12]
        n = r.get("total_encounters", 0)
        body_rows.append(
            f"<tr class='os-sb-row'><td class='os-sb-td'>{i}</td>"
            f"<td class='os-sb-td'>{gt}</td>"
            f"<td class='os-sb-td' style='text-align:center'>{n}</td></tr>"
        )
    tbody = "<tbody>" + "".join(body_rows) + "</tbody>"
    return (
        f"<div class='os-table-wrap os-sb-wrap'>"
        f"<table class='os-table os-scoreboard'>"
        f"<thead><tr><th class='os-sb-team' colspan='3'>{title}</th></tr></thead>"
        f"{header}{tbody}</table></div>"
    )


def build_antagonist_table_html(
    rows: list[dict],
    title: str,
    *,
    mode: str,
) -> str:
    """Construit un tableau HTML top némésis ou souffre-douleurs."""
    from src.ui.pages.match_table_html import gamertag_link

    col_main = t("col_times_killed_by") if mode == "nemesis" else t("col_times_killed")
    col_sec = t("col_times_killed") if mode == "nemesis" else t("col_times_killed_by")
    header = (
        "<thead><tr>"
        f"<th class='os-sb-th' style='text-align:left'>#</th>"
        f"<th class='os-sb-th' style='text-align:left'>{t('col_player')}</th>"
        f"<th class='os-sb-th'>{col_main}</th>"
        f"<th class='os-sb-th'>{col_sec}</th>"
        f"<th class='os-sb-th'>{t('col_net_kills')}</th>"
        f"<th class='os-sb-th'>{t('col_matches_against')}</th>"
        "</tr></thead>"
    )
    body_rows = []
    for i, r in enumerate(rows, 1):
        opp_gt = r.get("opponent_gamertag") or ""
        gt = gamertag_link(opp_gt) if opp_gt else "—"
        killed = r.get("times_killed", 0)
        killed_by = r.get("times_killed_by", 0)
        net = r.get("net_kills", killed - killed_by)
        matches = r.get("matches_against", 0)
        main_val = killed_by if mode == "nemesis" else killed
        sec_val = killed if mode == "nemesis" else killed_by
        net_style = _kd_style(killed, killed_by)
        net_sign = "+" if net > 0 else ""
        body_rows.append(
            f"<tr class='os-sb-row'><td class='os-sb-td'>{i}</td>"
            f"<td class='os-sb-td'>{gt}</td>"
            f"<td class='os-sb-td' style='text-align:center;font-weight:700'>{main_val}</td>"
            f"<td class='os-sb-td' style='text-align:center'>{sec_val}</td>"
            f"<td class='os-sb-td' style='text-align:center;{net_style}'>{net_sign}{net}</td>"
            f"<td class='os-sb-td' style='text-align:center;color:#aaa'>{matches}</td></tr>"
        )
    tbody = "<tbody>" + "".join(body_rows) + "</tbody>"
    return (
        f"<div class='os-table-wrap os-sb-wrap'>"
        f"<table class='os-table os-scoreboard'>"
        f"<thead><tr><th class='os-sb-team' colspan='6'>{title}</th></tr></thead>"
        f"{header}{tbody}</table></div>"
    )
