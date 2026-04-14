"""Service Carrière — extraction et formatage des données.

Toutes les importations src.* sont lazy (dans le corps des fonctions)
pour permettre le mocking en tests.
"""

from __future__ import annotations

import logging
from datetime import datetime
from os.path import exists as path_exists

from apps.api.app._db_helpers import OUTCOME_LABELS, Outcome
from apps.api.app._pure_bridge import (
    compute_career_projections,
    get_rank_info,
    load_career_data,
    load_career_history,
    load_lusr_history,
    load_lusr_snapshot,
    load_top_encountered,
    load_top_matches,
)
from apps.api.app.deps.players import PlayerContext
from apps.api.app.schemas.career import (
    CareerCharts,
    CareerEncounter,
    CareerEncountersResponse,
    CareerHistoryPoint,
    CareerLusrCheckpoint,
    CareerLusrSection,
    CareerPageResponse,
    CareerProjections,
    CareerSummary,
    CareerTopMatch,
    CareerTopMatchesResponse,
    HeroProgress,
)

logger = logging.getLogger(__name__)

XP_HERO_TOTAL: int = 9_319_350
RANK_MAX: int = 272


# ---------------------------------------------------------------------------
# Points d'entrée publics
# ---------------------------------------------------------------------------


def get_career_page(player: PlayerContext, *, exclude_btb: bool = False) -> CareerPageResponse:
    """Construit la réponse complète pour la page Carrière."""
    # career data functions imported at module level from _pure_bridge

    career_data = load_career_data(player.db_path, player.xuid)
    history = load_career_history(player.db_path, player.xuid)
    lusr_snapshot = load_lusr_snapshot(player.db_path, player.xuid)
    lusr_history = load_lusr_history(player.db_path, player.xuid)

    summary = _build_summary(career_data)
    xp_total = (career_data or {}).get("xp_total") or 0
    rank_number = (career_data or {}).get("rank") or 1
    last_date = history[-1]["recorded_at"] if history else None

    hero_progress = _build_hero_progress(xp_total, rank_number)
    projections = _build_projections(history, xp_total, last_date)
    xp_history = _build_xp_history(history)
    lusr = _build_lusr_section(lusr_snapshot, lusr_history)
    top_preview = _build_top_matches_preview(player, exclude_btb=exclude_btb)
    encounters_preview = _build_encounters_preview(player)

    return CareerPageResponse(
        summary=summary,
        hero_progress=hero_progress,
        projections=projections,
        charts=CareerCharts(),
        xp_history=xp_history,
        lusr=lusr,
        top_matches_preview=top_preview,
        encounters_preview=encounters_preview,
    )


def get_top_matches(
    player: PlayerContext, *, exclude_btb: bool = False
) -> CareerTopMatchesResponse:
    """Charge les top 10 meilleurs et pires matchs."""

    best = load_top_matches(
        player.db_path,
        player.xuid,
        best=True,
        exclude_btb=exclude_btb,
        shared_db_path=player.shared_db_path,
    )
    worst = load_top_matches(
        player.db_path,
        player.xuid,
        best=False,
        exclude_btb=exclude_btb,
        shared_db_path=player.shared_db_path,
    )
    items_best = [_match_dict_to_top_match(d, variant="best") for d in best]
    items_worst = [_match_dict_to_top_match(d, variant="worst") for d in worst]
    return CareerTopMatchesResponse(items=items_best + items_worst)


def get_encounters(player: PlayerContext) -> CareerEncountersResponse:
    """Charge les encounters (joueurs les plus croisés)."""
    items = _build_encounters_preview(player)
    return CareerEncountersResponse(items=items)


# ---------------------------------------------------------------------------
# Builders internes
# ---------------------------------------------------------------------------


def _build_summary(career_data: dict | None) -> CareerSummary | None:
    """Construit CareerSummary depuis les données brutes du rang."""
    if not career_data:
        return None

    rank_number: int = career_data.get("rank") or 1
    rank_name_raw: str = career_data.get("rank_name") or ""
    rank_tier: str = career_data.get("rank_tier") or ""
    current_xp: int = career_data.get("current_xp") or 0
    xp_for_next: int = career_data.get("xp_for_next_rank") or 0
    xp_total: int = career_data.get("xp_total") or 0
    is_max_rank: bool = bool(career_data.get("is_max_rank", False))
    recorded_at: datetime | None = career_data.get("recorded_at")

    rank_label = _format_rank_label(rank_number, rank_name_raw, rank_tier)
    progress_pct = _compute_progress_pct(current_xp, xp_for_next, is_max_rank)

    return CareerSummary(
        rank_number=rank_number,
        rank_label=rank_label,
        rank_name_raw=rank_name_raw,
        rank_tier=rank_tier,
        current_xp=current_xp,
        xp_for_next_rank=xp_for_next,
        xp_total=xp_total,
        progress_pct=progress_pct,
        is_max_rank=is_max_rank,
        recorded_at=recorded_at,
    )


def _format_rank_label(rank_number: int, rank_name_raw: str, rank_tier: str) -> str:
    """Formate le libellé FR du rang via career_ranks ou fallback."""
    try:
        info = get_rank_info(rank_number)
        if info:
            return info.full_label_fr
    except Exception:
        pass
    parts = [p for p in (rank_name_raw, rank_tier) if p]
    return " - ".join(parts) if parts else f"Rang {rank_number}"


def _compute_progress_pct(current_xp: int, xp_for_next: int, is_max_rank: bool) -> float:
    """Calcule le % de progression dans le rang courant."""
    if is_max_rank:
        return 100.0
    if xp_for_next <= 0:
        return 0.0
    return round(min(100.0, current_xp / xp_for_next * 100), 2)


def _build_hero_progress(xp_total: int, rank_number: int) -> HeroProgress:
    """Construit la section progression vers le rang Héros."""
    xp_remaining = max(0, XP_HERO_TOTAL - xp_total)
    percentage = round(min(100.0, xp_total / XP_HERO_TOTAL * 100), 2)
    return HeroProgress(
        xp_total_required=XP_HERO_TOTAL,
        xp_remaining=xp_remaining,
        percentage=percentage,
        current_rank=rank_number,
    )


def _build_projections(
    history: list[dict],
    xp_total: int,
    last_date: datetime | None,
) -> CareerProjections:
    """Calcule les projections XP vers le rang Héros."""
    try:
        result = compute_career_projections(history, xp_total, last_date)
        return CareerProjections(
            xp_per_day_active=result.get("xp_per_day_active", 0.0),
            xp_per_day_fallback=result.get("xp_per_day_fallback", 0.0),
            estimated_hero_date=result.get("hero_date"),
            estimated_rank_cap_date=None,
        )
    except Exception:
        logger.warning("Erreur calcul projections", exc_info=True)
        return CareerProjections(
            xp_per_day_active=0.0,
            xp_per_day_fallback=0.0,
            estimated_hero_date=None,
            estimated_rank_cap_date=None,
        )


def _build_xp_history(history: list[dict]) -> list[CareerHistoryPoint]:
    """Convertit l'historique DuckDB en CareerHistoryPoint."""
    points: list[CareerHistoryPoint] = []
    for row in history:
        recorded_at = row.get("recorded_at")
        rank = row.get("rank") or 1
        xp_total = row.get("xp_total") or 0
        if recorded_at:
            points.append(CareerHistoryPoint(recorded_at=recorded_at, rank=rank, xp_total=xp_total))
    return points


def _build_lusr_section(
    lusr_snapshot: list[dict],
    lusr_history: list[dict],
) -> CareerLusrSection | None:
    """Construit la section LUSR/CSR depuis les snapshots et l'historique."""
    if not lusr_snapshot:
        return None

    # Snapshot le plus "actif" (valeur la plus haute)
    current = max(lusr_snapshot, key=lambda r: r.get("rating_value") or 0.0)
    rating_value: float | None = current.get("rating_value")
    tier_label: str | None = current.get("tier_label") or current.get("tier_fr")
    playlist_group: str | None = current.get("playlist_group")

    trend_label = _compute_lusr_trend(current.get("rating_delta"))
    checkpoints = _build_lusr_checkpoints(lusr_history)

    return CareerLusrSection(
        current_rating=rating_value,
        current_tier_label=tier_label,
        current_playlist_group=playlist_group,
        trend_label=trend_label,
        checkpoints=checkpoints,
    )


def _compute_lusr_trend(rating_delta: float | None) -> str | None:
    """Retourne un libellé de tendance LUSR."""
    if rating_delta is None:
        return None
    if rating_delta > 0:
        return f"+{rating_delta:.0f}"
    if rating_delta < 0:
        return f"{rating_delta:.0f}"
    return "stable"


def _build_lusr_checkpoints(lusr_history: list[dict]) -> list[CareerLusrCheckpoint]:
    """Convertit l'historique LUSR en checkpoints."""
    checkpoints: list[CareerLusrCheckpoint] = []
    for row in lusr_history:
        start_time = row.get("start_time")
        rating = row.get("rating_value")
        group = row.get("playlist_group") or ""
        if start_time and rating is not None:
            checkpoints.append(
                CareerLusrCheckpoint(
                    recorded_at=start_time,
                    rating_value=float(rating),
                    playlist_group=group,
                )
            )
    return checkpoints


def _build_top_matches_preview(
    player: PlayerContext,
    *,
    exclude_btb: bool = False,
) -> list[CareerTopMatch]:
    """Charge les top matches (preview 5 meilleurs + 5 pires) si shared disponible."""
    if not path_exists(player.shared_db_path):
        return []
    try:
        matches = load_top_matches(
            player.db_path,
            player.xuid,
            best=True,
            exclude_btb=exclude_btb,
            shared_db_path=player.shared_db_path,
        )
        return [_match_dict_to_top_match(d, variant="best") for d in matches[:5]]
    except Exception:
        logger.warning("Erreur chargement top matches preview", exc_info=True)
        return []


def _build_encounters_preview(player: PlayerContext) -> list[CareerEncounter]:
    """Charge les encounters (preview 5 joueurs les plus croisés) si shared disponible."""
    if not path_exists(player.shared_db_path):
        return []
    try:
        rows = load_top_encountered(player.xuid, player.db_path, limit=5)
        return [_encounter_dict_to_encounter(d) for d in rows]
    except Exception:
        logger.warning("Erreur chargement encounters preview", exc_info=True)
        return []


def _match_dict_to_top_match(d: dict, *, variant: str | None = None) -> CareerTopMatch:
    """Convertit un dict de match brut en CareerTopMatch."""
    outcome_code = d.get("outcome") or 0
    outcome_label = OUTCOME_LABELS.get(int(outcome_code)) if outcome_code else None
    dominance = d.get("dominance_flag") or 0
    badge_type = "dominant" if dominance and outcome_code == Outcome.WIN else None

    my_score = d.get("my_team_score")
    enemy_score = d.get("enemy_team_score")
    score_label: str | None = None
    if my_score is not None and enemy_score is not None:
        score_label = f"{int(my_score)}-{int(enemy_score)}"

    kills = d.get("kills")
    deaths = d.get("deaths")
    assists = d.get("assists")
    kd_ratio: float | None = None
    if kills is not None:
        kd_ratio = kills / max(1, int(deaths or 0))

    return CareerTopMatch(
        match_id=str(d.get("match_id", "")),
        start_time=d.get("start_time"),
        map_ui=d.get("map_name"),
        mode_ui=d.get("game_variant_name"),
        playlist_label=d.get("playlist_name"),
        performance_score=None,
        badge_type=badge_type,
        score_label=score_label,
        outcome_label=outcome_label,
        kills=int(kills) if kills is not None else None,
        deaths=int(deaths) if deaths is not None else None,
        assists=int(assists) if assists is not None else None,
        kd_ratio=round(kd_ratio, 2) if kd_ratio is not None else None,
        variant=variant,
    )


def _encounter_dict_to_encounter(d: dict) -> CareerEncounter:
    """Convertit un dict d'encounter brut en CareerEncounter."""
    total: int = d.get("total_encounters") or 0
    ally: int = d.get("ally_count") or 0
    enemy: int = d.get("enemy_count") or 0
    winrate_ally: float = d.get("winrate_as_ally") or 0.0
    winrate_vs: float = d.get("winrate_vs_enemy") or 0.0

    wins = round(winrate_ally * ally + winrate_vs * enemy)
    losses = max(0, total - wins)

    last_seen = d.get("last_seen")

    return CareerEncounter(
        encounter_key=str(d.get("xuid", "")),
        opponent_gamertag=str(d.get("gamertag") or "Inconnu"),
        count_matches=total,
        wins=wins,
        losses=losses,
        last_seen_at=last_seen if isinstance(last_seen, datetime) else None,
    )
