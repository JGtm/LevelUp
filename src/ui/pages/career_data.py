"""Page Carrière — Chargement des données depuis DuckDB."""

from __future__ import annotations

import logging
from datetime import datetime

logger = logging.getLogger(__name__)


def _load_career_data(db_path: str, xuid: str) -> dict | None:
    """Charge les dernières données de rang carrière depuis DuckDB.

    Returns:
        Dict avec rank, rank_name, rank_tier, current_xp, etc. ou None.
    """
    logger.debug("Chargement career_data pour xuid=%s", xuid)
    from src.ui._cache_core import get_cached_repository_st

    repo = get_cached_repository_st(db_path, xuid)
    return repo.load_career_data()


def _load_career_history(db_path: str, xuid: str, limit: int = 50) -> list[dict]:
    """Charge l'historique de progression depuis DuckDB.

    Returns:
        Liste de dicts ordonnés par date croissante.
    """
    logger.debug("Chargement career_history pour xuid=%s (limit=%s)", xuid, limit)
    from src.ui._cache_core import get_cached_repository_st

    repo = get_cached_repository_st(db_path, xuid)
    return repo.load_career_history(limit)


# ── Chargement matchs pré-sync ──────────────────────────────────────────────


def _load_pre_sync_match_dates(
    db_path: str,
    xuid: str,
    first_sync_at: datetime,
) -> list[datetime]:
    """Charge les dates de matchs du joueur antérieurs au premier sync."""
    logger.debug("Chargement matchs pré-sync pour xuid=%s avant %s", xuid, first_sync_at)
    from src.ui._cache_core import get_cached_repository_st

    repo = get_cached_repository_st(db_path, xuid)
    return repo.load_pre_sync_match_dates(first_sync_at)


def _load_other_players_histories(current_xuid: str) -> list[dict]:
    """Charge l'historique XP de tous les autres profils disponibles."""
    logger.debug("Chargement historiques carrière des autres joueurs (exclu xuid=%s)", current_xuid)
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
                estimated_curve = None
                hero_proj = None
                optimistic_proj = None
                pre_sync_first_date: datetime | None = None
                try:
                    from src.ui.components.career_progress_circle import (
                        XP_HERO_TOTAL,  # noqa: PLC0415
                    )
                    from src.ui.pages.career_logic import (  # noqa: PLC0415
                        CAREER_XP_LAUNCH_DATE,
                        _compute_active_xp_per_day,
                        _compute_estimated_xp_curve,
                        _compute_fallback_xp_per_day,
                        _compute_hero_projections,
                    )

                    first_sync_at = hist[0]["recorded_at"]
                    pre_sync_dates = _load_pre_sync_match_dates(db_path, puid, first_sync_at)
                    if pre_sync_dates:
                        pre_sync_first_date = pre_sync_dates[0]
                        estimated_curve = _compute_estimated_xp_curve(hist, pre_sync_dates)

                    p_xp_total = hist[-1]["xp_total"] or 0
                    if p_xp_total > 0 and p_xp_total < XP_HERO_TOTAL:
                        xp_per_day = _compute_active_xp_per_day(hist)
                        if xp_per_day <= 0:
                            ref = pre_sync_first_date or CAREER_XP_LAUNCH_DATE
                            xp_per_day = _compute_fallback_xp_per_day(p_xp_total, ref)
                        if xp_per_day > 0:
                            last_date = hist[-1]["recorded_at"]
                            hero_proj, optimistic_proj = _compute_hero_projections(
                                p_xp_total, last_date, xp_per_day
                            )
                except Exception:
                    pass
                results.append(
                    {
                        "gamertag": gamertag,
                        "history": hist,
                        "estimated_curve": estimated_curve,
                        "hero_proj": hero_proj,
                        "optimistic_proj": optimistic_proj,
                    }
                )

        return results
    except Exception as e:
        logger.info("Impossible de charger les historiques des autres joueurs: %s", e)
        return []


def _load_lusr_snapshot(db_path: str, xuid: str) -> list[dict]:
    """Charge le dernier rating LUSR/CSR par playlist_group depuis match_skill_rank."""
    logger.debug("Chargement snapshot LUSR depuis %s", db_path)
    from src.ui._cache_core import get_cached_repository_st

    repo = get_cached_repository_st(db_path, xuid)
    return repo.load_lusr_snapshot()


def _load_lusr_history(db_path: str, xuid: str, playlist_group: str | None = None) -> list[dict]:
    """Charge l'historique LUSR/CSR pour le graphe d'évolution."""
    logger.debug("Chargement historique LUSR depuis %s (groupe=%s)", db_path, playlist_group)
    from src.ui._cache_core import get_cached_repository_st

    repo = get_cached_repository_st(db_path, xuid)
    return repo.load_lusr_history(playlist_group)
