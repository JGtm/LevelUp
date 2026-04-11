"""Requêtes DB — données intra-match pour le score de forme par bucket.

Charge depuis shared_matches_v2 :
- highlight_events  (kills/deaths horodatés par match)
- match_participants (accuracy, damage_dealt, damage_taken par joueur par match)
- match_registry    (duration_seconds par match)
"""

from __future__ import annotations

import logging
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)


def _fetch_events(
    conn: Any,
    match_ids: list[str],
    xuid: str,
    placeholders: str,
) -> dict[str, list[dict[str, Any]]]:
    """Charge les events kill/death horodatés depuis highlight_events."""
    rows_ev = conn.execute(
        f"""
        SELECT match_id, event_type, time_ms
        FROM highlight_events
        WHERE match_id IN ({placeholders})
          AND event_type IN ('kill', 'Kill', 'death', 'Death')
          AND xuid = ?
        ORDER BY match_id, time_ms
        """,
        [*match_ids, xuid],
    ).fetchall()
    result: dict[str, list[dict[str, Any]]] = {}
    for mid, ev_type, t_ms in rows_ev:
        key = str(mid)
        if key not in result:
            result[key] = []
        result[key].append({"event_type": ev_type, "time_ms": t_ms})
    return result


def _fetch_match_meta(
    conn: Any,
    match_ids: list[str],
    xuid: str,
    placeholders: str,
) -> dict[str, dict[str, Any]]:
    """Charge accuracy, damage et duration depuis match_participants + match_registry."""
    rows_mp = conn.execute(
        f"""
        SELECT mp.match_id, mp.accuracy, mp.damage_dealt, mp.damage_taken,
               mr.duration_seconds
        FROM match_participants mp
        JOIN match_registry mr ON mp.match_id = mr.match_id
        WHERE mp.match_id IN ({placeholders})
          AND mp.xuid = ?
        """,
        [*match_ids, xuid],
    ).fetchall()
    return {
        str(mid): {
            "accuracy": float(accuracy) if accuracy is not None else None,
            "damage_dealt": float(dmg_dealt) if dmg_dealt is not None else 0.0,
            "damage_taken": float(dmg_taken) if dmg_taken is not None else 0.0,
            "duration_seconds": float(duration_sec) if duration_sec is not None else 0.0,
        }
        for mid, accuracy, dmg_dealt, dmg_taken, duration_sec in rows_mp
    }


def load_bucket_data(
    player_db_path: str,
    xuid: str,
    match_ids: list[str],
) -> tuple[dict[str, list[dict[str, Any]]], dict[str, dict[str, Any]]]:
    """Charge les événements et métadonnées nécessaires au calcul des buckets.

    Args:
        player_db_path: Chemin vers stats.duckdb du joueur (pour dériver shared).
        xuid: XUID du joueur (pour filtrer highlight_events et match_participants).
        match_ids: Liste des match_id à charger (sélection courante).

    Returns:
        Tuple (events_by_match, match_meta) ou ({}, {}) si aucune donnée / erreur.
    """
    from src.utils.db import duckdb_read_only
    from src.utils.paths import get_shared_matches_path_from_player

    if not match_ids:
        return {}, {}

    player_path = Path(player_db_path)
    if not player_path.exists():
        return {}, {}

    shared_path = get_shared_matches_path_from_player(player_db_path)
    if not shared_path or not shared_path.exists():
        return {}, {}

    placeholders = ", ".join(["?" for _ in match_ids])
    try:
        with duckdb_read_only(str(shared_path)) as conn:
            events_by_match = _fetch_events(conn, match_ids, xuid, placeholders)
            match_meta = _fetch_match_meta(conn, match_ids, xuid, placeholders)
    except Exception as exc:
        logger.debug("load_bucket_data(%s, xuid=%s): %s", player_db_path, xuid, exc)
        return {}, {}

    logger.debug(
        "load_bucket_data: %d matchs avec events, %d avec meta.",
        len(events_by_match),
        len(match_meta),
    )
    return events_by_match, match_meta
