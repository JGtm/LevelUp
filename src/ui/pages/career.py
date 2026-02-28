"""Page Carrière — Progression du rang Halo Infinite.

Affiche le rang actuel, une gauge de progression XP, et l'historique
de progression dans le temps.

Inclut :
- Estimation pré-sync (XP moyen/match appliqué rétroactivement)
- Projection vers Héros (rythme actif, hors inactivité)
- Projection optimiste (défis weekly + boost x2)
"""

from __future__ import annotations

import html
import logging
from datetime import datetime, timedelta

import plotly.graph_objects as go
import streamlit as st

from src.config import THEME_COLORS
from src.ui.career_ranks import (
    format_career_rank_label_fr,
    get_rank_icon_path,
)
from src.ui.components.career_progress_circle import (
    RANK_MAX,
    XP_HERO_TOTAL,
    compute_hero_progress,
    create_career_progress_gauge,
    create_hero_progress_gauge,
)
from src.ui.i18n import t
from src.ui.player_assets import ensure_local_image_path
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, PLOTLY_STATIC_CONFIG, fragment_if_available
from src.utils.paths import get_shared_matches_path
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


def _load_career_data(db_path: str, xuid: str) -> dict | None:
    """Charge les dernières données de rang carrière depuis DuckDB.

    Returns:
        Dict avec rank, rank_name, rank_tier, current_xp, etc. ou None.
    """
    try:
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(db_path) as conn:
            result = conn.execute(
                """SELECT rank, rank_name, rank_tier, current_xp,
                          xp_for_next_rank, xp_total, is_max_rank,
                          adornment_path, recorded_at
                   FROM career_progression
                   WHERE xuid = ?
                   ORDER BY recorded_at DESC
                   LIMIT 1""",
                (xuid,),
            ).fetchone()

            if result:
                return {
                    "rank": result[0],
                    "rank_name": result[1],
                    "rank_tier": result[2],
                    "current_xp": result[3],
                    "xp_for_next_rank": result[4],
                    "xp_total": result[5],
                    "is_max_rank": bool(result[6]),
                    "adornment_path": result[7],
                    "recorded_at": result[8],
                }
    except Exception as e:
        logger.debug(f"Impossible de charger career_progression: {e}")

    return None


def _load_career_history(db_path: str, xuid: str, limit: int = 50) -> list[dict]:
    """Charge l'historique de progression depuis DuckDB.

    Returns:
        Liste de dicts ordonnés par date croissante.
    """
    try:
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(db_path) as conn:
            rows = conn.execute(
                """SELECT rank, rank_name, rank_tier, current_xp,
                          xp_for_next_rank, xp_total, is_max_rank,
                          recorded_at
                   FROM career_progression
                   WHERE xuid = ?
                   ORDER BY recorded_at ASC
                   LIMIT ?""",
                (xuid, limit),
            ).fetchall()

            return [
                {
                    "rank": r[0],
                    "rank_name": r[1],
                    "rank_tier": r[2],
                    "current_xp": r[3],
                    "xp_for_next_rank": r[4],
                    "xp_total": r[5],
                    "is_max_rank": bool(r[6]),
                    "recorded_at": r[7],
                }
                for r in rows
            ]
    except Exception as e:
        logger.debug(f"Impossible de charger career_history: {e}")
        return []


# ── Chargement matchs pré-sync ──────────────────────────────────────────────


def _load_pre_sync_match_dates(
    db_path: str,
    xuid: str,
    first_sync_at: datetime,
) -> list[datetime]:
    """Charge les dates de matchs du joueur antérieurs au premier sync.

    Args:
        db_path: Chemin vers stats.duckdb (player DB).
        xuid: XUID du joueur.
        first_sync_at: Date du premier snapshot career_progression.

    Returns:
        Liste de ``start_time`` ordonnées chronologiquement.
    """
    try:
        from src.utils.db import duckdb_read_only

        shared_path = get_shared_matches_path()
        if not shared_path.exists():
            return []

        with duckdb_read_only(shared_path) as conn:
            rows = conn.execute(
                """SELECT mr.start_time
                   FROM match_registry mr
                   JOIN match_participants mp ON mr.match_id = mp.match_id
                   WHERE mp.xuid = ?
                     AND mr.start_time < ?
                   ORDER BY mr.start_time ASC""",
                (xuid, first_sync_at),
            ).fetchall()
            return [r[0] for r in rows if r[0] is not None]
    except Exception as e:
        logger.debug(f"Impossible de charger les matchs pré-sync: {e}")
        return []


def _load_post_sync_match_count(
    xuid: str,
    first_sync_at: datetime,
) -> int:
    """Compte les matchs du joueur postérieurs au premier sync."""
    try:
        from src.utils.db import duckdb_read_only

        shared_path = get_shared_matches_path()
        if not shared_path.exists():
            return 0

        with duckdb_read_only(shared_path) as conn:
            result = conn.execute(
                """SELECT COUNT(*)
                   FROM match_registry mr
                   JOIN match_participants mp ON mr.match_id = mp.match_id
                   WHERE mp.xuid = ?
                     AND mr.start_time >= ?""",
                (xuid, first_sync_at),
            ).fetchone()
            return result[0] if result else 0
    except Exception as e:
        logger.debug(f"Impossible de compter les matchs post-sync: {e}")
        return 0


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


def _load_other_players_histories(current_xuid: str) -> list[dict]:
    """Charge l'historique XP de tous les autres profils disponibles.

    Args:
        current_xuid: XUID du joueur actuellement affiché (exclu).

    Returns:
        Liste de dicts ``{"gamertag": str, "history": list[dict]}``
        pour chaque autre profil ayant des données career_progression.
    """
    try:
        from src.utils.paths import get_player_db_path
        from src.utils.profiles import load_profiles

        profiles = load_profiles()
        results: list[dict] = []

        for gamertag, profile in profiles.items():
            puid = profile.get("xuid", "")
            if puid == current_xuid:
                continue  # ignorer le joueur actuel

            db_path = profile.get("db_path") or str(get_player_db_path(gamertag))
            import os

            if not os.path.exists(db_path):
                continue

            hist = _load_career_history(db_path, puid)
            if len(hist) >= 2:
                results.append({"gamertag": gamertag, "history": hist})

        return results
    except Exception as e:
        logger.debug(f"Impossible de charger les historiques des autres joueurs: {e}")
        return []


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


def _create_xp_history_chart(
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


def _load_lusr_snapshot(db_path: str) -> list[dict]:
    """Charge le dernier rating LUSR/CSR par playlist_group depuis match_skill_rank.

    Le ``rating_delta`` est calculé dynamiquement via une window function ``LAG``
    sur les ``rating_value`` stockés, garantissant la cohérence avec les valeurs
    affichées (évite le delta stocké périmé si le backfill a utilisé un seed
    différent entre deux exécutions).

    Returns:
        Liste de dicts : rating_type, rating_value, tier_label, rating_delta, playlist_group.
        Triée par playlist_group alphabétique.
    """
    try:
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(db_path) as conn:
            # Dernier rating par playlist_group = ligne avec start_time la plus récente.
            # Le delta est recalculé dynamiquement via LAG pour garantir la cohérence
            # avec les rating_value stockés (évite le delta stocké potentiellement
            # périmé en cas de recalcul avec seed différent).
            rows = conn.execute(
                """
                WITH history AS (
                    SELECT
                        msr.match_id, msr.rating_type, msr.rating_value,
                        msr.tier_label, msr.sub_tier, msr.tier, msr.tier_fr,
                        msr.playlist_group,
                        COALESCE(msr.start_time, msr.updated_at) AS sort_time,
                        msr.rating_value - LAG(msr.rating_value) OVER (
                            PARTITION BY msr.playlist_group
                            ORDER BY COALESCE(msr.start_time, msr.updated_at)
                        ) AS rating_delta,
                        ROW_NUMBER() OVER (
                            PARTITION BY msr.playlist_group
                            ORDER BY COALESCE(msr.start_time, msr.updated_at) DESC
                        ) AS rn
                    FROM match_skill_rank msr
                )
                SELECT rating_type, rating_value, tier_label, sub_tier,
                       tier, tier_fr, rating_delta, playlist_group
                FROM history
                WHERE rn = 1
                ORDER BY playlist_group
                """
            ).fetchall()

            if not rows:
                return []
            return [
                {
                    "rating_type": r[0],
                    "rating_value": r[1],
                    "tier_label": r[2],
                    "sub_tier": r[3] or 0,
                    "tier": r[4],
                    "tier_fr": r[5],
                    "rating_delta": r[6],
                    "playlist_group": r[7],
                }
                for r in rows
            ]
    except Exception as e:
        logger.debug(f"Impossible de charger match_skill_rank: {e}")
        return []


def _load_lusr_history(db_path: str, playlist_group: str | None = None) -> list[dict]:
    """Charge l'historique LUSR/CSR pour le graphe d'évolution.

    Args:
        db_path: Chemin vers stats.duckdb.
        playlist_group: Filtrer par groupe (None = tous).

    Returns:
        Liste de dicts avec : match_id, rating_value, rating_deviation,
        rating_type, playlist_group, start_time, tier_label.
    """
    try:
        from src.utils.db import duckdb_read_only

        pg_filter = "AND msr.playlist_group = ?" if playlist_group else ""
        params: list = [playlist_group] if playlist_group else []

        with duckdb_read_only(db_path) as conn:
            rows = conn.execute(
                f"""
                SELECT msr.match_id, msr.rating_value, msr.rating_deviation,
                       msr.rating_type, msr.playlist_group, msr.tier_label,
                       COALESCE(msr.start_time, msr.created_at) AS start_time
                FROM match_skill_rank msr
                WHERE 1=1 {pg_filter}
                ORDER BY COALESCE(msr.start_time, msr.created_at) ASC
                """,
                params,
            ).fetchall()

            return [
                {
                    "match_id": r[0],
                    "rating_value": r[1],
                    "rating_deviation": r[2],
                    "rating_type": r[3],
                    "playlist_group": r[4],
                    "tier_label": r[5],
                    "start_time": r[6],
                }
                for r in rows
            ]
    except Exception as e:
        logger.debug(f"Impossible de charger l'historique LUSR: {e}")
        return []


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


_PG_ICONS: dict[str, str] = {
    "ranked": "🏆",
    "arena": "⚔️",
    "btb": "💥",
    "tactical": "🎯",
    "social": "🎮",
    "fun": "🎉",
}
# Ordre d'affichage des groupes (du plus compétitif au plus détendu)
_PG_ORDER = ["ranked", "arena", "btb", "tactical", "social", "fun"]


def _render_lusr_section(*, db_path: str, xuid: str) -> None:
    """Rend la section LUSR/CSR sur la page Carrière.

    Affiche :
    1. Cartes visuelles par groupe (rang, image, delta)
    2. Graphe d'évolution avec sélecteur de groupe
    """
    from pathlib import Path

    try:
        import polars as pl

        from src.analysis.skill_rating_config import LUSR_SUBTITLE, LUSR_TITLE, get_rank_image_path
        from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG
        from src.visualization.timeseries_combat import plot_lusr_timeseries
    except ImportError:
        logger.debug("Modules LUSR non disponibles pour career.py")
        return

    st.subheader(f"🏅 {LUSR_TITLE} — {LUSR_SUBTITLE}")

    snapshot = _load_lusr_snapshot(db_path)
    if not snapshot:
        st.info(t("career_lusr_no_rating"))
        return

    # ── Cartes visuelles — triées par ordre de compétitivité ──
    snap_by_group = {s["playlist_group"]: s for s in snapshot}
    ordered = [snap_by_group[g] for g in _PG_ORDER if g in snap_by_group]
    # Groupes non listés dans _PG_ORDER (futur) en queue
    ordered += [s for s in snapshot if s["playlist_group"] not in _PG_ORDER]

    # Racine projet depuis l'emplacement du fichier source (parents[3] = LevelUp/)
    project_root = Path(__file__).parents[3]

    # Angles droits sur les bordures des cartes LUSR
    st.markdown(
        "<style>[data-testid='stVerticalBlockBorderWrapper']" "{border-radius:0!important}</style>",
        unsafe_allow_html=True,
    )

    n_per_row = 3
    for row_start in range(0, len(ordered), n_per_row):
        batch = ordered[row_start : row_start + n_per_row]
        cols = st.columns(len(batch))
        for col, snap in zip(cols, batch, strict=False):
            pg = snap["playlist_group"] or "?"
            tier_label = snap["tier_label"] or t("unranked")
            r_value = float(snap["rating_value"] or 0.0)
            delta = snap["rating_delta"]
            r_type = snap["rating_type"] or "LUSR"

            with col, st.container(border=True):
                # ── En-tête du groupe ──
                pg_label = _get_pg_labels().get(pg, pg.capitalize())
                pg_icon = _PG_ICONS.get(pg, "🎮")

                # ── Image du rang encodée en base64 ──
                img_b64 = ""
                img_path_rel = get_rank_image_path(r_value)
                if img_path_rel:
                    img_full = project_root / img_path_rel
                    if img_full.exists():
                        import base64

                        img_b64 = base64.b64encode(img_full.read_bytes()).decode()

                img_html = (
                    f"<img src='data:image/png;base64,{img_b64}' "
                    f"style='width:90px;height:90px;object-fit:contain' />"
                    if img_b64
                    else "<div style='width:90px;height:90px'></div>"
                )

                # ── Badge type ──
                badge_bg = "#00B7EB" if r_type == "LUSR" else "#FFD700"
                badge_fg = "#000"

                # ── Delta ──
                if delta is not None:
                    _d = round(delta)
                    if _d == 0:
                        delta_html = (
                            "<div style='color:#888888;font-size:0.9em;margin-top:4px'>= 0</div>"
                        )
                    else:
                        d_color = "#00C853" if _d > 0 else "#FF5252"
                        d_arrow = "▲" if _d > 0 else "▼"
                        d_sign = "+" if _d > 0 else "-"
                        delta_html = (
                            f"<div style='color:{d_color};font-size:0.9em;margin-top:4px'>"
                            f"{d_arrow} {d_sign}{abs(_d)}</div>"
                        )
                else:
                    delta_html = "<div style='color:#888;font-size:0.9em;margin-top:4px'>—</div>"

                st.markdown(
                    f"""<div style='display:flex;flex-direction:column;
                                    align-items:center;justify-content:center;
                                    text-align:center;padding:4px 0;gap:4px'>
                        <div style='font-size:1.25em;font-weight:700;
                                    letter-spacing:0.02em;line-height:1.2'>
                            {html.escape(pg_icon)} {html.escape(pg_label)}
                        </div>
                        <div style='display:flex;align-items:center;
                                    justify-content:center;height:96px'>
                            {img_html}
                        </div>
                        <div style='font-size:1.05em;font-weight:700'>
                            {html.escape(tier_label)}
                        </div>
                        <div>
                            <span style='background:{badge_bg};color:{badge_fg};
                                         padding:1px 7px;border-radius:10px;
                                         font-size:0.72em;font-weight:600'>
                                {html.escape(r_type)}
                            </span>
                            &nbsp;<span style='font-size:1.1em;font-weight:700'>
                                {r_value:.0f}
                            </span>
                        </div>
                        {delta_html}
                    </div>""",
                    unsafe_allow_html=True,
                )

    # ── Graphe d'évolution avec sélecteur de groupe ──
    history_all = _load_lusr_history(db_path)
    if not history_all:
        return

    df_all = pl.DataFrame(history_all)
    available_groups = sorted(df_all["playlist_group"].drop_nulls().unique().to_list())
    if not available_groups:
        return

    st.markdown(f"#### {t('career_lusr_rating_evolution')}")

    # Sélecteur de groupe : "Tous" + un par groupe disponible
    group_options: dict[str, str | None] = {t("career_lusr_all_groups"): None}
    pg_labels = _get_pg_labels()
    for g in _PG_ORDER:
        if g in available_groups:
            group_options[f"{_PG_ICONS.get(g, '🎮')} {pg_labels.get(g, g.capitalize())}"] = g
    # Groupes hors _PG_ORDER en queue
    for g in available_groups:
        if g not in _PG_ORDER:
            group_options[f"🎮 {g.capitalize()}"] = g

    selected_label = st.selectbox(
        t("career_lusr_group_select"),
        options=list(group_options.keys()),
        key="lusr_group_select",
    )
    selected_group = group_options[selected_label]

    chart_title = f"LUSR / CSR — {selected_label}"
    try:
        fig = plot_lusr_timeseries(
            df_all,
            title=chart_title,
            playlist_group=selected_group,
        )
        st.plotly_chart(
            fig,
            key=f"lusr_chart_{selected_label}",
            width="stretch",
            config=PLOTLY_CLEAN_CONFIG,
        )
    except Exception as e:
        st.warning(t("career_lusr_group_error", error=e))


@fragment_if_available
def render_career_page(
    *,
    db_path: str,
    xuid: str,
    db_key: str | None = None,
) -> None:
    """Rend la page Carrière avec rang actuel, gauge et historique."""
    st.header(t("career_header"))

    # Charger les données
    career_data = _load_career_data(db_path, xuid)

    if career_data is None:
        st.info(t("career_no_data"))
        return

    rank_number = career_data.get("rank", 0)
    rank_name = career_data.get("rank_name", "")
    rank_tier = career_data.get("rank_tier", "")
    current_xp = career_data.get("current_xp", 0) or 0
    xp_for_next = career_data.get("xp_for_next_rank", 1) or 1
    xp_total = career_data.get("xp_total", 0) or 0
    is_max = career_data.get("is_max_rank", False)

    # Calcul progression
    if is_max:
        progress_pct = 100.0
    elif xp_for_next > 0:
        progress_pct = min(100.0, (current_xp / xp_for_next) * 100)
    else:
        progress_pct = 0.0

    # Label FR du rang
    rank_label_fr = format_career_rank_label_fr(tier=rank_tier, title=rank_name, grade=None)
    if not rank_label_fr:
        rank_label_fr = rank_name or t("career_rank_n", n=rank_number)

    # --- Header avec icone + metriques ---
    col_icon, col_info, col_gauge = st.columns([1, 2, 2])

    with col_icon:
        # Priorité: adornment (DB) > icône locale du rang (10C.3.4)
        adornment_db_path = career_data.get("adornment_path") or ""
        displayed_icon = False

        if adornment_db_path:
            # L'adornment_path peut être une URL ou un chemin CMSlocal
            local_adornment = ensure_local_image_path(
                adornment_db_path,
                prefix="adornment",
                download_enabled=True,
                auto_refresh_hours=0,
            )
            if local_adornment:
                st.image(local_adornment, width=140)
                displayed_icon = True

        if not displayed_icon:
            # Fallback: icône locale standard
            icon_path = get_rank_icon_path(rank_number) if rank_number else None
            if icon_path and icon_path.exists():
                st.image(str(icon_path), width=120)
            else:
                st.markdown(f"### {rank_number}")

    with col_info:
        st.subheader(rank_label_fr)

        # Métriques
        m1, m2 = st.columns(2)
        with m1:
            st.metric(t("career_metric_rank"), f"{rank_number} / 272")
            st.metric(t("career_metric_xp_total"), f"{xp_total:,}")
        with m2:
            if is_max:
                st.metric(t("career_rank_max"), t("career_rank_max"))
            else:
                st.metric(t("career_metric_current_xp"), f"{current_xp:,}")
                st.metric(t("career_metric_next_rank_xp"), f"{xp_for_next:,}")

    with col_gauge:
        # Gauge de progression
        try:
            gauge_fig = create_career_progress_gauge(
                current_xp=current_xp,
                xp_for_next_rank=xp_for_next,
                progress_pct=progress_pct,
                rank_name_fr=rank_label_fr,
                is_max_rank=is_max,
            )
            if gauge_fig is not None:
                st.plotly_chart(
                    gauge_fig, key="career_gauge", width="stretch", config=PLOTLY_STATIC_CONFIG
                )
            else:
                st.info(t("career_gauge_generate_error"))
        except Exception as e:
            st.warning(t("career_gauge_error", error=e))

    # --- Progression vers Héros ---
    st.divider()
    st.subheader(t("career_progression_to_hero"))

    hero_data = compute_hero_progress(xp_total=xp_total, rank=rank_number, is_max_rank=is_max)
    hero_pct = hero_data["percentage"]
    xp_remaining = hero_data["xp_remaining"]

    col_hero_metrics, col_hero_gauge = st.columns([3, 2])

    with col_hero_metrics:
        m1, m2, m3, m4 = st.columns(4)
        with m1:
            st.metric(t("career_metric_xp_earned"), f"{xp_total:,}")
        with m2:
            st.metric(t("career_metric_xp_remaining"), f"{xp_remaining:,}")
        with m3:
            st.metric(t("career_metric_xp_required"), f"{XP_HERO_TOTAL:,}")
        with m4:
            st.metric(t("career_metric_rank"), f"{rank_number} / {RANK_MAX}")

    with col_hero_gauge:
        try:
            hero_gauge = create_hero_progress_gauge(
                hero_pct=hero_pct,
                xp_total=xp_total,
                xp_remaining=xp_remaining,
                is_max_rank=is_max,
            )
            st.plotly_chart(
                hero_gauge,
                key="hero_progress_gauge",
                width="stretch",
                config=PLOTLY_STATIC_CONFIG,
            )
        except Exception as e:
            st.warning(t("career_hero_progress_error", error=e))

    # --- Historique de progression ---
    st.divider()

    history = _load_career_history(db_path, xuid)

    if history:
        # ── Estimation pré-sync ──
        estimated_curve: list[tuple[datetime, int]] | None = None
        try:
            first_sync_at = history[0]["recorded_at"]
            if first_sync_at and len(history) >= 2:
                pre_sync_dates = _load_pre_sync_match_dates(db_path, xuid, first_sync_at)
                if pre_sync_dates:
                    estimated_curve = _compute_estimated_xp_curve(history, pre_sync_dates)
        except Exception as e:
            logger.debug(f"Estimation pré-sync échouée: {e}")

        # ── Projections vers Héros ──
        hero_proj: list[tuple[datetime, int]] | None = None
        optimistic_proj: list[tuple[datetime, int]] | None = None
        if not is_max:
            try:
                xp_per_day = _compute_active_xp_per_day(history)
                if xp_per_day > 0:
                    last_date = history[-1]["recorded_at"]
                    hero_proj, optimistic_proj = _compute_hero_projections(
                        xp_total, last_date, xp_per_day
                    )
            except Exception as e:
                logger.debug(f"Projection Héros échouée: {e}")

        try:
            other_players_data = _load_other_players_histories(xuid)
        except Exception as e:
            logger.debug(f"Chargement autres joueurs échoué: {e}")
            other_players_data = []

        try:
            history_fig = _create_xp_history_chart(
                history,
                estimated_curve=estimated_curve,
                hero_projection=hero_proj,
                optimistic_projection=optimistic_proj,
                is_max_rank=is_max,
                other_players=other_players_data or None,
            )
            if history_fig:
                st.plotly_chart(
                    history_fig,
                    key="career_xp_history",
                    width="stretch",
                    config=PLOTLY_CLEAN_CONFIG,
                )
            else:
                st.info(t("career_rank_history_no_data"))
        except Exception as e:
            st.warning(t("career_history_error", error=e))

        # Tableau récapitulatif des derniers snapshots
        with st.expander(t("career_rank_history_title"), expanded=False):
            # Afficher les 10 derniers snapshots (du plus récent au plus ancien)
            recent = list(reversed(history[-10:]))
            for snap in recent:
                snap_label = format_career_rank_label_fr(
                    tier=snap.get("rank_tier", ""),
                    title=snap.get("rank_name", ""),
                    grade=None,
                )
                date_str = str(snap.get("recorded_at", ""))[:19]
                xp_t = snap.get("xp_total", 0) or 0
                st.text(
                    f"{date_str}  |  {t('career_rank_n', n=snap['rank'])}: {snap_label}  |  XP: {xp_t:,}"
                )
    else:
        st.info(t("career_computing"))

    # --- LUSR / CSR — LevelUp Skill Rank ---
    st.divider()
    _render_lusr_section(db_path=db_path, xuid=xuid)
