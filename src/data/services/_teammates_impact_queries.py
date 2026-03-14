"""Helpers privés de chargement des données d'impact (extracted de teammates_service)."""

from __future__ import annotations

import polars as pl


def _query_impact_events(conn: object, match_ids: list[str], placeholders: str) -> list:
    """Récupère les événements highlight depuis shared.highlight_events."""
    query = f"""
        SELECT
            match_id,
            CASE WHEN event_type IN ('kill', 'Kill') THEN killer_xuid::TEXT
                 WHEN event_type IN ('death', 'Death') THEN victim_xuid::TEXT
                 ELSE COALESCE(killer_xuid, victim_xuid)::TEXT
            END AS xuid,
            CASE WHEN event_type IN ('kill', 'Kill') THEN COALESCE(killer_gamertag, killer_xuid::TEXT)
                 WHEN event_type IN ('death', 'Death') THEN COALESCE(victim_gamertag, victim_xuid::TEXT)
                 ELSE COALESCE(killer_gamertag, victim_gamertag, 'Unknown')
            END AS gamertag,
            event_type,
            COALESCE(time_ms, 0) AS time_ms
        FROM shared.highlight_events
        WHERE match_id IN ({placeholders})
    """
    return conn.execute(query, match_ids).fetchall()  # type: ignore[union-attr]


def _query_match_outcomes(conn: object, match_ids: list[str], xuid: str, placeholders: str) -> list:
    """Récupère les outcomes depuis shared.match_participants."""
    result = conn.execute(  # type: ignore[union-attr]
        f"SELECT match_id, outcome FROM shared.match_participants"
        f" WHERE match_id IN ({placeholders}) AND xuid = ?",
        [*match_ids, xuid.strip()],
    ).fetchall()
    return result or []


def _collect_impact_data(
    conn: object,
    match_ids: list[str],
    xuid: str,
    friend_xuids: set[str],
    placeholders: str,
) -> tuple | None:
    """Construit DataFrames events/matches et calcule les événements d'impact.

    Returns:
        Tuple (first_bloods, clutch_finishers, last_casualties,
        last_group_kills, first_group_deaths, scores) ou None si aucun événement.
    """
    from src.analysis.friends_impact import get_all_impact_events

    events_result = _query_impact_events(conn, match_ids, placeholders)
    if not events_result:
        return None

    events_df = pl.DataFrame(
        {
            "match_id": [str(r[0]) for r in events_result],
            "xuid": [str(r[1]) for r in events_result],
            "gamertag": [r[2] or "Unknown" for r in events_result],
            "event_type": [r[3] for r in events_result],
            "time_ms": [int(r[4] or 0) for r in events_result],
        }
    )
    matches_result = _query_match_outcomes(conn, match_ids, xuid, placeholders)
    matches_df = pl.DataFrame(
        {
            "match_id": [str(r[0]) for r in matches_result],
            "outcome": [int(r[1] or 0) for r in matches_result],
        }
    )
    return get_all_impact_events(events_df, matches_df, friend_xuids=friend_xuids)
