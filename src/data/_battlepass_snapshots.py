"""Snapshots joueur des battle pass Halo Infinite."""

from __future__ import annotations

import hashlib
import json
import logging
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from src.data.sync.battlepass_migrations import ensure_battlepass_snapshots_table
from src.utils.db import duckdb_read_write

logger = logging.getLogger(__name__)


@dataclass(frozen=True)
class BattlepassProgressSnapshot:
    """État local d'un joueur sur un reward track battle pass."""

    reward_track_path: str
    track_type: str | None
    is_active: bool
    is_owned: bool | None
    current_rank: int | None
    partial_progress: int | None
    previous_rank: int | None
    previous_partial_progress: int | None
    has_reached_max_rank: bool | None
    base_xp: int | None
    boost_xp: int | None
    raw_payload_json: str


def persist_battlepass_snapshots(
    player_db_path: str | Path,
    xuid: str,
    operations_payload: dict[str, Any],
) -> int:
    """Persiste les états battle pass d'un joueur en append-only dédupliqué."""
    tracks = operations_payload.get("OperationRewardTracks")
    if not isinstance(tracks, list) or not tracks:
        return 0

    db_path = Path(player_db_path)
    db_path.parent.mkdir(parents=True, exist_ok=True)
    snapshot_at = datetime.now(timezone.utc)
    active_path = _coerce_str(operations_payload.get("ActiveOperationRewardTrackPath"))
    rows = _build_snapshot_rows(snapshot_at, xuid, active_path, tracks)
    if not rows:
        return 0

    try:
        with duckdb_read_write(db_path) as conn:
            ensure_battlepass_snapshots_table(conn)
            last_hashes = _load_last_snapshot_hashes(conn, xuid)
            to_insert = [row for row in rows if last_hashes.get(row[2]) != row[13]]
            if not to_insert:
                return 0
            conn.executemany(
                """
                INSERT INTO battlepass_snapshots (
                    snapshot_at,
                    xuid,
                    reward_track_path,
                    track_type,
                    is_active,
                    is_owned,
                    current_rank,
                    partial_progress,
                    previous_rank,
                    previous_partial_progress,
                    has_reached_max_rank,
                    base_xp,
                    boost_xp,
                    state_hash,
                    raw_payload_json
                )
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                """,
                to_insert,
            )
            return len(to_insert)
    except Exception as exc:
        logger.debug("battlepass_snapshots: persistance ignorée: %s", exc)
        return 0


def _build_snapshot_rows(
    snapshot_at: datetime,
    xuid: str,
    active_path: str | None,
    tracks: list[dict[str, Any]],
) -> list[tuple[Any, ...]]:
    rows: list[tuple[Any, ...]] = []
    for entry in tracks:
        snapshot = _build_progress_snapshot(entry, active_path)
        if snapshot is None:
            continue
        state_hash = _build_state_hash(snapshot)
        rows.append(
            (
                snapshot_at,
                xuid,
                snapshot.reward_track_path,
                snapshot.track_type,
                snapshot.is_active,
                snapshot.is_owned,
                snapshot.current_rank,
                snapshot.partial_progress,
                snapshot.previous_rank,
                snapshot.previous_partial_progress,
                snapshot.has_reached_max_rank,
                snapshot.base_xp,
                snapshot.boost_xp,
                state_hash,
                snapshot.raw_payload_json,
            )
        )
    return rows


def _build_progress_snapshot(
    entry: dict[str, Any],
    active_path: str | None,
) -> BattlepassProgressSnapshot | None:
    reward_track_path = _coerce_str(entry.get("RewardTrackPath"))
    if not reward_track_path:
        return None
    current_rank, partial_progress, progress_is_owned, has_reached_max_rank = _parse_progress(
        entry.get("CurrentProgress")
    )
    previous_rank, previous_partial_progress, _, _ = _parse_progress(entry.get("PreviousProgress"))
    is_owned = _coerce_bool(entry.get("IsOwned"))
    if is_owned is None:
        is_owned = progress_is_owned

    return BattlepassProgressSnapshot(
        reward_track_path=reward_track_path,
        track_type=_coerce_str(entry.get("TrackType")),
        is_active=reward_track_path == active_path,
        is_owned=is_owned,
        current_rank=current_rank,
        partial_progress=partial_progress,
        previous_rank=previous_rank,
        previous_partial_progress=previous_partial_progress,
        has_reached_max_rank=has_reached_max_rank,
        base_xp=_coerce_int(entry.get("BaseXp")),
        boost_xp=_coerce_int(entry.get("BoostXp")),
        raw_payload_json=json.dumps(entry, sort_keys=True, ensure_ascii=False),
    )


def _parse_progress(progress: Any) -> tuple[int | None, int | None, bool | None, bool | None]:
    if isinstance(progress, dict):
        return (
            _coerce_int(progress.get("Rank")),
            _coerce_int(progress.get("PartialProgress")),
            _coerce_bool(progress.get("IsOwned")),
            _coerce_bool(progress.get("HasReachedMaxRank")),
        )
    scalar = _coerce_int(progress)
    return scalar, None, None, None


def _load_last_snapshot_hashes(conn: Any, xuid: str) -> dict[str, str]:
    rows = conn.execute(
        """
        SELECT reward_track_path, state_hash
        FROM (
            SELECT
                reward_track_path,
                state_hash,
                ROW_NUMBER() OVER (PARTITION BY reward_track_path ORDER BY snapshot_at DESC) AS rn
            FROM battlepass_snapshots
            WHERE xuid = ?
        )
        WHERE rn = 1
        """,
        [xuid],
    ).fetchall()
    return {row[0]: row[1] for row in rows if row[0] and row[1]}


def _build_state_hash(snapshot: BattlepassProgressSnapshot) -> str:
    payload = {
        "reward_track_path": snapshot.reward_track_path,
        "track_type": snapshot.track_type,
        "is_active": snapshot.is_active,
        "is_owned": snapshot.is_owned,
        "current_rank": snapshot.current_rank,
        "partial_progress": snapshot.partial_progress,
        "previous_rank": snapshot.previous_rank,
        "previous_partial_progress": snapshot.previous_partial_progress,
        "has_reached_max_rank": snapshot.has_reached_max_rank,
        "base_xp": snapshot.base_xp,
        "boost_xp": snapshot.boost_xp,
    }
    canonical = json.dumps(payload, sort_keys=True, separators=(",", ":"), default=str)
    return hashlib.sha1(canonical.encode("utf-8")).hexdigest()


def _coerce_int(value: Any) -> int | None:
    if isinstance(value, bool):
        return int(value)
    if isinstance(value, int):
        return value
    if isinstance(value, float) and value.is_integer():
        return int(value)
    try:
        text = str(value).strip()
    except Exception:
        return None
    if not text:
        return None
    try:
        return int(text)
    except ValueError:
        return None


def _coerce_bool(value: Any) -> bool | None:
    if isinstance(value, bool):
        return value
    if isinstance(value, str):
        lowered = value.strip().lower()
        if lowered in {"true", "1", "yes"}:
            return True
        if lowered in {"false", "0", "no"}:
            return False
    return None


def _coerce_str(value: Any) -> str | None:
    if not isinstance(value, str):
        return None
    text = value.strip()
    return text or None
