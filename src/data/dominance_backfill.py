"""Calcul des dominance flags (médaille Steaktacular) pour un joueur.

Interroge ``shared_matches.duckdb`` pour détecter la présence de la médaille
"À table" (Steaktacular) — dans l'équipe du joueur ou l'équipe adverse — et
persiste le résultat dans ``player_match_enrichment.dominance_flag``.

Usage depuis le sync engine ou le backfill CLI :

    >>> from src.data.dominance_backfill import compute_dominance_for_player
    >>> result = compute_dominance_for_player(player_conn, shared_conn, xuid)
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING

from src.analysis._medal_verdicts import (
    MEDAL_STEAKTACULAR_ID,
    DominanceFlag,
)

if TYPE_CHECKING:
    import duckdb

logger = logging.getLogger(__name__)


_NONE = int(DominanceFlag.NONE)
_DOM = int(DominanceFlag.DOMINATION)
_HUM = int(DominanceFlag.HUMILIATION)

_DOMINANCE_SQL = f"""
WITH my_team AS (
    SELECT mp.match_id, mp.team_id
    FROM match_participants mp
    JOIN match_registry mr ON mr.match_id = mp.match_id
    WHERE mp.xuid = ?
      AND COALESCE(mr.medals_loaded, FALSE) = TRUE
),
steak_teams AS (
    SELECT me.match_id, smp.team_id
    FROM medals_earned me
    JOIN match_participants smp
        ON smp.match_id = me.match_id AND smp.xuid = me.xuid
    WHERE me.medal_name_id = ?
    GROUP BY me.match_id, smp.team_id
)
SELECT
    mt.match_id,
    CASE
        WHEN st.team_id IS NULL THEN {_NONE}
        WHEN st.team_id = mt.team_id THEN {_DOM}
        ELSE {_HUM}
    END AS flag
FROM my_team mt
LEFT JOIN steak_teams st ON st.match_id = mt.match_id
"""


def compute_dominance_for_player(
    player_conn: duckdb.DuckDBPyConnection,
    shared_conn: duckdb.DuckDBPyConnection,
    xuid: str,
    *,
    force: bool = False,
) -> dict[str, int]:
    """Calcule et persiste les dominance flags pour un joueur.

    Args:
        player_conn: Connexion RW vers ``stats.duckdb`` du joueur.
        shared_conn: Connexion RO vers ``shared_matches.duckdb``.
        xuid: XUID du joueur.
        force: Recalculer même les matchs déjà flaggués.

    Returns:
        Dict ``{"processed": n, "domination": d, "humiliation": h}``.
    """
    from src.data.sync.migrations import ensure_dominance_flag_column

    ensure_dominance_flag_column(player_conn)

    results = shared_conn.execute(_DOMINANCE_SQL, [xuid, MEDAL_STEAKTACULAR_ID]).fetchall()

    if not results:
        return {"processed": 0, "domination": 0, "humiliation": 0}

    flag_map = {r[0]: r[1] for r in results}
    all_match_ids = list(flag_map.keys())
    ep = ", ".join(["?"] * len(all_match_ids))

    existing_set = {
        r[0]
        for r in player_conn.execute(
            f"SELECT match_id FROM player_match_enrichment WHERE match_id IN ({ep})",
            all_match_ids,
        ).fetchall()
    }

    where = "1=1" if force else "dominance_flag IS NULL"
    n = 0

    for mid in all_match_ids:
        if mid not in existing_set:
            continue
        player_conn.execute(
            "UPDATE player_match_enrichment "
            "SET dominance_flag = ?, updated_at = CURRENT_TIMESTAMP "
            f"WHERE match_id = ? AND ({where})",
            [flag_map[mid], mid],
        )
        n += 1

    new_list = [m for m in all_match_ids if m not in existing_set]
    if new_list:
        player_conn.executemany(
            "INSERT INTO player_match_enrichment " "(match_id, dominance_flag) VALUES (?, ?)",
            [(m, flag_map[m]) for m in new_list],
        )
        n += len(new_list)

    player_conn.commit()

    dom_count = sum(1 for r in results if r[1] == DominanceFlag.DOMINATION)
    hum_count = sum(1 for r in results if r[1] == DominanceFlag.HUMILIATION)

    logger.info(
        "Dominance backfill xuid=%s: %d traités (%d dom, %d hum)",
        xuid,
        n,
        dom_count,
        hum_count,
    )
    return {"processed": n, "domination": dom_count, "humiliation": hum_count}
