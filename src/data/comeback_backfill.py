"""Calcul des badges narrative (Remontada / Débandade / Contre-Remontada) pour un joueur.

Interroge ``shared_matches_v2.duckdb`` pour reconstruire une timeline approximative
à partir des kill-events de ``highlight_events``, puis persiste le résultat dans
``player_match_enrichment.dominance_flag``.

Seuls les matchs avec ``dominance_flag = 0`` (NONE) sont traités, sauf si
``force=True`` (qui re-traite aussi les valeurs 3-5 mais jamais 1 ou 2).

Usage :
    >>> from src.data.comeback_backfill import compute_comeback_badges_for_player
    >>> result = compute_comeback_badges_for_player(player_conn, shared_conn, xuid)
"""

from __future__ import annotations

import logging

import duckdb
import polars as pl

from src.analysis._medal_verdicts import DominanceFlag
from src.analysis.comeback_analysis import MatchMeta, detect_comeback_badge

logger = logging.getLogger(__name__)

_COMEBACK_FLAGS = {
    int(DominanceFlag.REMONTADA),
    int(DominanceFlag.DEBANDADE),
    int(DominanceFlag.CONTRE_REMONTADA),
}


def _load_candidate_matches(
    player_conn: duckdb.DuckDBPyConnection,
    shared_conn: duckdb.DuckDBPyConnection,
    xuid: str,
    *,
    force: bool,
) -> list[dict]:
    """Retourne les matchs candidats avec leurs métadonnées."""
    from src.data.sync.migrations import ensure_dominance_flag_column

    ensure_dominance_flag_column(player_conn)

    where_flag = (
        "dominance_flag IN (0, 3, 4, 5) OR dominance_flag IS NULL"
        if force
        else "dominance_flag = 0 OR dominance_flag IS NULL"
    )

    rows = player_conn.execute(
        f"""
        SELECT pme.match_id, pme.dominance_flag
        FROM player_match_enrichment pme
        WHERE {where_flag}
        """
    ).fetchall()

    if not rows:
        return []

    match_ids = [r[0] for r in rows]
    ep = ", ".join(["?"] * len(match_ids))

    # Récupérer les match_ids ayant events_loaded + win_score + variant depuis shared
    meta = shared_conn.execute(
        f"""
        SELECT match_id,
               GREATEST(CAST(team_0_score AS INT), CAST(team_1_score AS INT)) AS win_score,
               game_variant_name
        FROM match_registry
        WHERE match_id IN ({ep})
          AND COALESCE(events_loaded, FALSE) = TRUE
        """,
        match_ids,
    ).fetchall()

    return [{"match_id": r[0], "win_score": r[1], "game_variant_name": r[2]} for r in meta]


def _load_match_events(
    shared_conn: duckdb.DuckDBPyConnection,
    match_id: str,
) -> pl.DataFrame:
    """Charge les kill-events d'un match depuis shared."""
    rows = shared_conn.execute(
        "SELECT event_type, time_ms, xuid FROM highlight_events "
        "WHERE match_id = ? AND event_type = 'kill'",
        [match_id],
    ).fetchall()
    if not rows:
        return pl.DataFrame({"event_type": [], "time_ms": [], "xuid": []})
    return pl.DataFrame(
        {
            "event_type": [r[0] for r in rows],
            "time_ms": [r[1] for r in rows],
            "xuid": [r[2] for r in rows],
        }
    )


def _load_match_participants(
    shared_conn: duckdb.DuckDBPyConnection,
    match_id: str,
) -> pl.DataFrame:
    """Charge participants (xuid, team_id, outcome) d'un match depuis shared."""
    rows = shared_conn.execute(
        "SELECT xuid, team_id, outcome FROM match_participants WHERE match_id = ?",
        [match_id],
    ).fetchall()
    if not rows:
        return pl.DataFrame({"xuid": [], "team_id": [], "outcome": []})
    return pl.DataFrame(
        {
            "xuid": [r[0] for r in rows],
            "team_id": [r[1] for r in rows],
            "outcome": [r[2] for r in rows],
        }
    )


def _reset_objective_mode_flags(
    player_conn: duckdb.DuckDBPyConnection,
    shared_conn: duckdb.DuckDBPyConnection,
    xuid: str,
) -> int:
    """Remet à 0 les badges comeback sur les matchs non-Slayer.

    Couvre les cas où un badge a été posé avant l'introduction du filtre
    sur game_variant_name, ou sur des matchs sans events_loaded (hors boucle).
    """
    rows = shared_conn.execute(
        "SELECT match_id FROM match_registry WHERE LOWER(game_variant_name) NOT LIKE '%slayer%'"
    ).fetchall()
    if not rows:
        return 0
    ids = [r[0] for r in rows]
    ep = ", ".join(["?"] * len(ids))
    result = player_conn.execute(
        f"UPDATE player_match_enrichment "
        f"SET dominance_flag = 0, updated_at = CURRENT_TIMESTAMP "
        f"WHERE dominance_flag IN (3, 4, 5) AND match_id IN ({ep})",
        ids,
    )
    count = result.rowcount if result else 0
    if count > 0:
        logger.info("reset_objective_flags xuid=%s: %d flag(s) remis à 0", xuid, count)
    return count


def compute_comeback_badges_for_player(
    player_conn: duckdb.DuckDBPyConnection,
    shared_conn: duckdb.DuckDBPyConnection,
    xuid: str,
    *,
    force: bool = False,
) -> dict[str, int]:
    """Calcule et persiste les badges narrative pour un joueur.

    Args:
        player_conn: Connexion RW vers ``stats.duckdb`` du joueur.
        shared_conn: Connexion RO vers ``shared_matches_v2.duckdb``.
        xuid: XUID du joueur.
        force: Re-traiter les matchs déjà badgés (3-5). Ne jamais écraser 1 ou 2.

    Returns:
        Dict ``{"processed": n, "remontada": r, "debandade": d, "contre_remontada": c}``.
    """
    _reset_objective_mode_flags(player_conn, shared_conn, xuid)
    candidates = _load_candidate_matches(player_conn, shared_conn, xuid, force=force)
    if not candidates:
        logger.info("Comeback backfill xuid=%s: aucun match candidat.", xuid)
        return {"processed": 0, "remontada": 0, "debandade": 0, "contre_remontada": 0}

    counts = {
        int(DominanceFlag.REMONTADA): 0,
        int(DominanceFlag.DEBANDADE): 0,
        int(DominanceFlag.CONTRE_REMONTADA): 0,
    }
    n = 0

    for item in candidates:
        mid = item["match_id"]

        events = _load_match_events(shared_conn, mid)
        participants = _load_match_participants(shared_conn, mid)

        my_rows = participants.filter(pl.col("xuid") == xuid)
        if my_rows.is_empty():
            continue
        outcome = my_rows["outcome"][0]

        flag = detect_comeback_badge(
            events,
            participants,
            xuid,
            outcome,
            meta=MatchMeta(
                win_score=item.get("win_score"),
                game_variant_name=item.get("game_variant_name"),
            ),
        )

        player_conn.execute(
            "UPDATE player_match_enrichment "
            "SET dominance_flag = ?, updated_at = CURRENT_TIMESTAMP "
            "WHERE match_id = ?",
            [flag, mid],
        )
        if flag in counts:
            counts[flag] += 1
        n += 1

    player_conn.commit()

    logger.info(
        "Comeback backfill xuid=%s: %d traités (remontada=%d, débandade=%d, contre=%d)",
        xuid,
        n,
        counts[int(DominanceFlag.REMONTADA)],
        counts[int(DominanceFlag.DEBANDADE)],
        counts[int(DominanceFlag.CONTRE_REMONTADA)],
    )
    return {
        "processed": n,
        "remontada": counts[int(DominanceFlag.REMONTADA)],
        "debandade": counts[int(DominanceFlag.DEBANDADE)],
        "contre_remontada": counts[int(DominanceFlag.CONTRE_REMONTADA)],
    }
