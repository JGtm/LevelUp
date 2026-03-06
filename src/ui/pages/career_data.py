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
                results.append({"gamertag": gamertag, "history": hist})

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


# ── Chargement top encounters / antagonistes ────────────────────────────────

_TOP_ENCOUNTERED_SQL = """
WITH my_matches AS (
    SELECT match_id, team_id, outcome
    FROM match_participants
    WHERE xuid = ?
),
encounters AS (
    SELECT
        p.xuid,
        MAX(COALESCE(a.gamertag, p.gamertag)) AS gamertag,
        COUNT(*)                                AS total_encounters,
        SUM(CASE WHEN p.team_id = m.team_id   THEN 1 ELSE 0 END) AS ally_count,
        SUM(CASE WHEN p.team_id != m.team_id  THEN 1 ELSE 0 END) AS enemy_count,
        AVG(CASE
            WHEN p.team_id = m.team_id AND m.outcome = 2         THEN 1.0
            WHEN p.team_id = m.team_id AND m.outcome IN (3, 4)   THEN 0.0
            ELSE NULL
        END) AS winrate_as_ally,
        AVG(CASE
            WHEN p.team_id != m.team_id AND m.outcome = 2        THEN 1.0
            WHEN p.team_id != m.team_id AND m.outcome IN (3, 4)  THEN 0.0
            ELSE NULL
        END) AS winrate_vs_enemy,
        MAX(r.start_time) AS last_seen
    FROM match_participants p
    INNER JOIN my_matches m  ON m.match_id = p.match_id
    LEFT JOIN  xuid_aliases a ON a.xuid = p.xuid
    LEFT JOIN  match_registry r ON r.match_id = p.match_id
    WHERE p.xuid != ?
    GROUP BY p.xuid
),
kvp_agg AS (
    SELECT
        CASE WHEN k.killer_xuid = ? THEN k.victim_xuid
             ELSE k.killer_xuid END AS opp,
        SUM(CASE WHEN k.killer_xuid = ? THEN k.kill_count ELSE 0 END) AS kills_dealt,
        SUM(CASE WHEN k.victim_xuid = ? THEN k.kill_count ELSE 0 END) AS deaths_suffered
    FROM killer_victim_pairs k
    WHERE k.killer_xuid = ? OR k.victim_xuid = ?
    GROUP BY 1
)
SELECT
    e.xuid, e.gamertag, e.total_encounters, e.ally_count, e.enemy_count,
    e.winrate_as_ally, e.winrate_vs_enemy,
    COALESCE(kvp.kills_dealt, 0)     AS kills_dealt,
    COALESCE(kvp.deaths_suffered, 0) AS deaths_suffered,
    e.last_seen
FROM encounters e
LEFT JOIN kvp_agg kvp ON kvp.opp = e.xuid
ORDER BY e.total_encounters DESC
LIMIT ?
"""


def _load_top_encountered(
    xuid: str,
    limit: int = 10,
    exclude_xuids: set[str] | None = None,
) -> list[dict]:
    """Charge les joueurs les plus croisés depuis shared_matches.duckdb."""
    try:
        from src.utils.db import duckdb_read_only

        shared_path = get_shared_matches_path()
        if not shared_path.exists():
            return []

        extra = len(exclude_xuids) if exclude_xuids else 0
        sql_limit = limit + extra

        with duckdb_read_only(shared_path) as conn:
            rows = conn.execute(
                _TOP_ENCOUNTERED_SQL,
                [xuid, xuid, xuid, xuid, xuid, xuid, xuid, sql_limit],
            ).fetchall()

        cols = (
            "xuid",
            "gamertag",
            "total_encounters",
            "ally_count",
            "enemy_count",
            "winrate_as_ally",
            "winrate_vs_enemy",
            "kills_dealt",
            "deaths_suffered",
            "last_seen",
        )
        results = [dict(zip(cols, r, strict=False)) for r in rows]
        if exclude_xuids:
            lower_excl = {e.casefold() for e in exclude_xuids}
            results = [
                r
                for r in results
                if r["xuid"] not in exclude_xuids
                and (r.get("gamertag") or "").casefold() not in lower_excl
            ]
        return results[:limit]
    except Exception as e:
        logger.debug("Impossible de charger top_encountered: %s", e)
        return []


def _load_top_nemeses(
    xuid: str,
    limit: int = 10,
    exclude_xuids: set[str] | None = None,
) -> list[dict]:
    """Charge les adversaires qui nous tuent le plus (némésis)."""
    return _load_antagonists_from_shared(
        xuid,
        mode="nemesis",
        limit=limit,
        exclude_xuids=exclude_xuids,
    )


def _load_top_victims(
    xuid: str,
    limit: int = 10,
    exclude_xuids: set[str] | None = None,
) -> list[dict]:
    """Charge les adversaires qu'on tue le plus (souffre-douleurs)."""
    return _load_antagonists_from_shared(
        xuid,
        mode="victim",
        limit=limit,
        exclude_xuids=exclude_xuids,
    )


_ANTAGONISTS_SQL = """
WITH kills_dealt AS (
    SELECT victim_xuid AS opponent_xuid, SUM(kill_count) AS times_killed
    FROM killer_victim_pairs WHERE killer_xuid = ? GROUP BY victim_xuid
),
kills_suffered AS (
    SELECT killer_xuid AS opponent_xuid, SUM(kill_count) AS times_killed_by
    FROM killer_victim_pairs WHERE victim_xuid = ? GROUP BY killer_xuid
),
matches_vs AS (
    SELECT DISTINCT kvp.match_id,
           CASE WHEN kvp.killer_xuid = ? THEN kvp.victim_xuid
                ELSE kvp.killer_xuid END AS opponent_xuid
    FROM killer_victim_pairs kvp
    WHERE kvp.killer_xuid = ? OR kvp.victim_xuid = ?
),
match_counts AS (
    SELECT opponent_xuid, COUNT(*) AS matches_against
    FROM matches_vs GROUP BY opponent_xuid
),
combined AS (
    SELECT COALESCE(kd.opponent_xuid, ks.opponent_xuid) AS opponent_xuid,
           COALESCE(kd.times_killed, 0) AS times_killed,
           COALESCE(ks.times_killed_by, 0) AS times_killed_by
    FROM kills_dealt kd
    FULL OUTER JOIN kills_suffered ks ON kd.opponent_xuid = ks.opponent_xuid
)
SELECT c.opponent_xuid, COALESCE(xa.gamertag, '') AS opponent_gamertag,
       c.times_killed, c.times_killed_by,
       COALESCE(mc.matches_against, 0) AS matches_against,
       c.times_killed - c.times_killed_by AS net_kills
FROM combined c
LEFT JOIN xuid_aliases xa ON xa.xuid = c.opponent_xuid
LEFT JOIN match_counts mc ON mc.opponent_xuid = c.opponent_xuid
WHERE c.opponent_xuid != ?
ORDER BY {order_col} DESC
LIMIT ?
"""


def _load_antagonists_from_shared(
    xuid: str,
    *,
    mode: str,
    limit: int = 10,
    exclude_xuids: set[str] | None = None,
) -> list[dict]:
    """Charge némésis ou souffre-douleurs depuis shared killer_victim_pairs."""
    try:
        from src.utils.db import duckdb_read_only

        shared_path = get_shared_matches_path()
        if not shared_path.exists():
            return []

        extra = len(exclude_xuids) if exclude_xuids else 0
        order_col = "times_killed_by" if mode == "nemesis" else "times_killed"
        sql = _ANTAGONISTS_SQL.format(order_col=order_col)

        with duckdb_read_only(shared_path) as conn:
            rows = conn.execute(
                sql,
                [xuid, xuid, xuid, xuid, xuid, xuid, limit + extra],
            ).fetchall()

        cols = (
            "opponent_xuid",
            "opponent_gamertag",
            "times_killed",
            "times_killed_by",
            "matches_against",
            "net_kills",
        )
        results = [dict(zip(cols, r, strict=False)) for r in rows]
        if exclude_xuids:
            lower_excl = {e.casefold() for e in exclude_xuids}
            results = [
                r
                for r in results
                if r["opponent_xuid"] not in exclude_xuids
                and (r.get("opponent_gamertag") or "").casefold() not in lower_excl
            ]
        return results[:limit]
    except Exception as e:
        logger.debug("Impossible de charger antagonistes depuis shared: %s", e)
        return []
