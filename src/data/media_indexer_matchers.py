"""Fonctions de matching média↔match, extraites de media_indexer.py."""

from __future__ import annotations

import logging
from pathlib import Path
from typing import Any

import duckdb

from src.data.media_helpers import match_start_to_epoch
from src.utils.paths import get_shared_matches_path_from_player

logger = logging.getLogger(__name__)


def _load_matches_by_xuid(
    db_path: Path,
    player_dbs: list[tuple[Path, str]],
) -> dict[str, list[tuple[Any, ...]]]:
    """Charge les matchs de chaque joueur depuis shared_matches_v2.duckdb."""
    matches_by_xuid: dict[str, list[tuple[Any, ...]]] = {}
    shared_path = get_shared_matches_path_from_player(db_path)
    if shared_path and shared_path.exists():
        try:
            with duckdb.connect(str(shared_path), read_only=True) as c:
                for _db_path_iter, xuid in player_dbs:
                    try:
                        rows = c.execute(
                            """
                            SELECT mp.match_id, mr.start_time,
                                   COALESCE(mr.duration_seconds, 720),
                                   COALESCE(mr.map_id, ''), COALESCE(mr.map_name, '')
                            FROM match_participants mp
                            JOIN match_registry mr ON mp.match_id = mr.match_id
                            WHERE mp.xuid = ? AND mr.start_time IS NOT NULL
                            """,
                            [str(xuid)],
                        ).fetchall()
                        matches_by_xuid[str(xuid)] = rows
                    except Exception:
                        matches_by_xuid[str(xuid)] = []
        except Exception as e:
            logger.warning("associate_with_matches shared_db: %s", e)
    else:
        for _db_path, xuid in player_dbs:
            matches_by_xuid[str(xuid)] = []
    return matches_by_xuid


def _associate_single_media(  # noqa: PLR0913
    conn: duckdb.DuckDBPyConnection,
    media_path: str,
    mtime_epoch: float,
    matches_by_xuid: dict[str, list[tuple[Any, ...]]],
    tol_seconds: int,
) -> None:
    """Associe un seul média aux matchs les plus proches de chaque joueur."""
    for xuid, matches in matches_by_xuid.items():
        candidates: list[tuple[str, Any, Any, str, str, float]] = []
        for row in matches:
            match_id, st, dur = row[0], row[1], row[2]
            map_id = row[3] if len(row) > 3 else ""
            map_name = row[4] if len(row) > 4 else ""
            start_epoch = match_start_to_epoch(st)
            if start_epoch is None:
                continue
            d = float(dur or 0) if dur else 12 * 60
            end_epoch = start_epoch + d
            if start_epoch - tol_seconds <= mtime_epoch <= end_epoch + tol_seconds:
                dist = abs(mtime_epoch - start_epoch)
                candidates.append((xuid, match_id, st, map_id, map_name, dist))

        if not candidates:
            continue
        candidates.sort(key=lambda x: (x[5], x[1]))
        best = candidates[0]
        try:
            conn.execute(
                """
                INSERT INTO media_match_associations
                (media_path, match_id, xuid, match_start_time, map_id, map_name, association_confidence)
                VALUES (?, ?, ?, ?, ?, ?, 1.0)
                ON CONFLICT (media_path, match_id, xuid) DO NOTHING
                """,
                [media_path, best[1], best[0], best[2], best[3], best[4]],
            )
        except Exception as e:
            logger.warning("Association %s/%s: %s", media_path, xuid, e)
