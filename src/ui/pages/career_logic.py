"""Page Carrière — Logique de calcul et projections."""

from __future__ import annotations

import logging
from datetime import datetime, timedelta

from src.ui.components.career_progress_circle import XP_HERO_TOTAL
from src.ui.i18n import t

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
# Date d'introduction du système de rangs (Career Rank, mise à jour CU32 — 20 juin 2023)
# Avant cette date il n'y avait pas d'XP : tout le monde démarrait à 0.
CAREER_XP_LAUNCH_DATE: datetime = datetime(2023, 6, 20)


# ── Estimation pré-sync ─────────────────────────────────────────────────────


def _compute_estimated_xp_curve(
    history: list[dict],
    pre_sync_match_dates: list[datetime],
) -> list[tuple[datetime, int]]:
    """Estime la courbe XP pour les matchs antérieurs au premier sync.

    Logique : l'XP total au 1er snapshot est réparti uniformément sur tous les
    matchs pré-sync éligibles (après le 20 juin 2023, date de lancement des
    rangs de carrière). On remonte dans le temps depuis le 1er snapshot en
    soustrayant cet XP moyen à chaque match éligible.
    Les matchs antérieurs au lancement sont exclus : il n'existait pas d'XP.

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

    # Exclure les matchs antérieurs au lancement du système de rangs (20/06/2023).
    # Utiliser .date() pour rester compatible avec les datetimes tz-aware ou naives.
    launch = CAREER_XP_LAUNCH_DATE.date()
    eligible_dates = [d for d in pre_sync_match_dates if d.date() >= launch]

    if not eligible_dates:
        return []

    # Moyenne basée sur l'XP total au 1er snapshot réparti sur les matchs éligibles.
    # (utiliser la moyenne post-sync introduirait un biais si le rythme change)
    n_pre = len(eligible_dates)
    avg_xp_per_match = first_xp / n_pre

    # Remonter dans le temps depuis le 1er snapshot
    curve: list[tuple[datetime, int]] = []
    current_xp = float(first_xp)

    # Parcourir les matchs éligibles du plus récent au plus ancien
    for match_date in reversed(eligible_dates):
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


def _compute_fallback_xp_per_day(xp_total: int, first_date: datetime) -> float:
    """Rythme XP moyen global quand le delta inter-snapshots est nul.

    Utilise le rapport XP total / jours depuis la première date connue
    (typiquement le début de la courbe estimée pré-sync) pour débloquer
    les projections quand le joueur a trop peu de snapshots.

    Returns:
        XP par jour actif moyen, ou 0.0 si impossible à calculer.
    """
    try:
        tz = getattr(first_date, "tzinfo", None)
        now = datetime.now(tz=tz) if tz else datetime.now()
    except Exception:
        now = datetime.now()
    days = (now - first_date).total_seconds() / 86400.0
    if days <= 0 or xp_total <= 0:
        return 0.0
    return xp_total / days


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
