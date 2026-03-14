"""Page Carrière — Chargement des données rencontres et antagonistes."""

from __future__ import annotations

import logging
from datetime import datetime

from src.utils.paths import get_shared_matches_path

logger = logging.getLogger(__name__)


def _get_kv_source(conn: object) -> str:
    """Retourne la meilleure source killer/victim disponible dans la connexion."""
    try:
        conn.execute("SELECT 1 FROM v_killer_victim_full LIMIT 1")  # type: ignore[union-attr]
        return "v_killer_victim_full"
    except Exception:
        return "killer_victim_pairs"


# ── Chargement top encounters / antagonistes ────────────────────────────────

_TOP_ENCOUNTERED_SQL = """
WITH my_matches AS (
    SELECT mp.match_id, mp.team_id, mp.outcome
    FROM match_participants mp
    JOIN match_registry r2 ON r2.match_id = mp.match_id
    WHERE mp.xuid = ?
      AND (? IS NULL OR r2.start_time >= ?)
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
    FROM {kv_table} k
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
    *,
    since: datetime | None = None,
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
            kv_table = _get_kv_source(conn)
            rows = conn.execute(
                _TOP_ENCOUNTERED_SQL.format(kv_table=kv_table),
                [xuid, since, since, xuid, xuid, xuid, xuid, xuid, xuid, sql_limit],
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
    *,
    since: datetime | None = None,
) -> list[dict]:
    """Charge les adversaires qui nous tuent le plus (némésis)."""
    return _load_antagonists_from_shared(
        xuid,
        mode="nemesis",
        limit=limit,
        exclude_xuids=exclude_xuids,
        since=since,
    )


def _load_top_victims(
    xuid: str,
    limit: int = 10,
    exclude_xuids: set[str] | None = None,
    *,
    since: datetime | None = None,
) -> list[dict]:
    """Charge les adversaires qu'on tue le plus (souffre-douleurs)."""
    return _load_antagonists_from_shared(
        xuid,
        mode="victim",
        limit=limit,
        exclude_xuids=exclude_xuids,
        since=since,
    )


_ANTAGONISTS_SQL = """
WITH period_matches AS (
    SELECT match_id FROM match_registry
    WHERE (? IS NULL OR start_time >= ?)
),
kills_dealt AS (
    SELECT victim_xuid AS opponent_xuid, SUM(kill_count) AS times_killed
    FROM {kv_table}
    WHERE killer_xuid = ? AND match_id IN (SELECT match_id FROM period_matches)
    GROUP BY victim_xuid
),
kills_suffered AS (
    SELECT killer_xuid AS opponent_xuid, SUM(kill_count) AS times_killed_by
    FROM {kv_table}
    WHERE victim_xuid = ? AND match_id IN (SELECT match_id FROM period_matches)
    GROUP BY killer_xuid
),
matches_vs AS (
    SELECT DISTINCT kvp.match_id,
           CASE WHEN kvp.killer_xuid = ? THEN kvp.victim_xuid
                ELSE kvp.killer_xuid END AS opponent_xuid
    FROM {kv_table} kvp
    WHERE (kvp.killer_xuid = ? OR kvp.victim_xuid = ?)
      AND kvp.match_id IN (SELECT match_id FROM period_matches)
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
    since: datetime | None = None,
) -> list[dict]:
    """Charge némésis ou souffre-douleurs depuis shared killer_victim_pairs."""
    try:
        from src.utils.db import duckdb_read_only

        shared_path = get_shared_matches_path()
        if not shared_path.exists():
            return []

        extra = len(exclude_xuids) if exclude_xuids else 0
        order_col = "times_killed_by" if mode == "nemesis" else "times_killed"

        with duckdb_read_only(shared_path) as conn:
            kv_table = _get_kv_source(conn)
            sql = _ANTAGONISTS_SQL.format(order_col=order_col, kv_table=kv_table)
            rows = conn.execute(
                sql,
                [since, since, xuid, xuid, xuid, xuid, xuid, xuid, limit + extra],
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
