"""Page Carrière — Chargement des données depuis DuckDB."""

from __future__ import annotations

import logging
from datetime import datetime

from src.utils.paths import get_shared_matches_path

logger = logging.getLogger(__name__)


def _load_career_data(db_path: str, xuid: str) -> dict | None:
    """Charge les dernières données de rang carrière depuis DuckDB.

    Returns:
        Dict avec rank, rank_name, rank_tier, current_xp, etc. ou None.
    """
    logger.debug("Chargement career_data pour xuid=%s", xuid)
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
        logger.debug("Impossible de charger career_progression: %s", e)

    return None


def _load_career_history(db_path: str, xuid: str, limit: int = 50) -> list[dict]:
    """Charge l'historique de progression depuis DuckDB.

    Returns:
        Liste de dicts ordonnés par date croissante.
    """
    logger.debug("Chargement career_history pour xuid=%s (limit=%s)", xuid, limit)
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
        logger.debug("Impossible de charger career_history: %s", e)
        return []


# ── Chargement matchs pré-sync ──────────────────────────────────────────────


def _load_pre_sync_match_dates(
    db_path: str,
    xuid: str,
    first_sync_at: datetime,
) -> list[datetime]:
    """Charge les dates de matchs du joueur antérieurs au premier sync."""
    logger.debug("Chargement matchs pré-sync pour xuid=%s avant %s", xuid, first_sync_at)
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
        logger.debug("Impossible de charger les matchs pré-sync: %s", e)
        return []


def _load_post_sync_match_count(
    xuid: str,
    first_sync_at: datetime,
) -> int:
    """Compte les matchs du joueur postérieurs au premier sync."""
    logger.debug("Comptage matchs post-sync pour xuid=%s après %s", xuid, first_sync_at)
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
        logger.debug("Impossible de compter les matchs post-sync: %s", e)
        return 0


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


def _load_lusr_snapshot(db_path: str) -> list[dict]:
    """Charge le dernier rating LUSR/CSR par playlist_group depuis match_skill_rank."""
    logger.debug("Chargement snapshot LUSR depuis %s", db_path)
    try:
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(db_path) as conn:
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
        logger.debug("Impossible de charger match_skill_rank: %s", e)
        return []


def _load_lusr_history(db_path: str, playlist_group: str | None = None) -> list[dict]:
    """Charge l'historique LUSR/CSR pour le graphe d'évolution."""
    logger.debug("Chargement historique LUSR depuis %s (groupe=%s)", db_path, playlist_group)
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
        logger.debug("Impossible de charger l'historique LUSR: %s", e)
        return []
