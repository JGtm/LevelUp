"""Snapshots joueur des défis Halo Infinite."""

from __future__ import annotations

import hashlib
import json
import logging
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Any

from src.data._challenge_catalog import (
    StoredChallengeDefinition,
    collect_challenge_paths_from_decks,
    extract_challenge_xp,
    extract_threshold_for_success,
    load_challenge_metadata_map,
)
from src.data.sync.challenge_migrations import ensure_challenge_snapshots_table
from src.utils.db import duckdb_read_write

logger = logging.getLogger(__name__)


@dataclass(frozen=True)
class SnapshotBuildContext:
    """Contexte partagé pour construire les snapshots d'un joueur."""

    snapshot_at: datetime
    xuid: str
    definitions: dict[str, dict]
    definition_hashes: dict[str, str]
    stored: dict[str, StoredChallengeDefinition]


@dataclass(frozen=True)
class DeckSnapshotContext:
    """Contexte dérivé d'un deck actif/completed/upcoming."""

    deck_index: int
    expires_at: datetime | None
    expiry_for_hash: str | None


def persist_challenge_snapshots(
    player_db_path: str | Path,
    xuid: str,
    decks_data: dict,
    definitions: dict[str, dict] | None = None,
    definition_hashes: dict[str, str] | None = None,
) -> int:
    """Persiste les états de défis joueur en append-only dédupliqué."""
    challenge_paths = collect_challenge_paths_from_decks(decks_data)
    if not challenge_paths:
        return 0

    db_path = Path(player_db_path)
    db_path.parent.mkdir(parents=True, exist_ok=True)
    definitions = definitions or {}
    definition_hashes = dict(definition_hashes or {})
    stored = load_challenge_metadata_map(challenge_paths, "en-US")
    rows = _build_snapshot_rows(
        xuid=xuid,
        decks_data=decks_data,
        definitions=definitions,
        definition_hashes=definition_hashes,
        stored=stored,
    )
    if not rows:
        return 0

    try:
        with duckdb_read_write(db_path) as conn:
            ensure_challenge_snapshots_table(conn)
            last_hashes = _load_last_snapshot_hashes(conn, xuid)
            to_insert = [row for row in rows if last_hashes.get(row[2]) != row[-1]]
            if not to_insert:
                return 0
            conn.executemany(
                """
                INSERT INTO challenge_snapshots (
                    snapshot_at,
                    xuid,
                    challenge_path,
                    challenge_id,
                    content_hash,
                    status,
                    progress_current,
                    progress_target,
                    xp_reward,
                    can_reroll,
                    expires_at,
                    deck_index,
                    state_hash
                )
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                """,
                to_insert,
            )
            return len(to_insert)
    except Exception as exc:
        logger.debug("challenge_snapshots: persistance joueur ignorée: %s", exc)
        return 0


def _build_snapshot_rows(
    *,
    xuid: str,
    decks_data: dict,
    definitions: dict[str, dict],
    definition_hashes: dict[str, str],
    stored: dict[str, StoredChallengeDefinition],
) -> list[tuple[Any, ...]]:
    decks = decks_data.get("AssignedDecks")
    if not isinstance(decks, list):
        return []

    build_context = SnapshotBuildContext(
        snapshot_at=datetime.utcnow(),
        xuid=xuid,
        definitions=definitions,
        definition_hashes=definition_hashes,
        stored=stored,
    )
    rows: list[tuple[Any, ...]] = []
    for deck_index, deck in enumerate(decks):
        deck_context = DeckSnapshotContext(
            deck_index=deck_index,
            expires_at=_parse_iso_timestamp((deck.get("Expiration") or {}).get("ISO8601Date")),
            expiry_for_hash=(deck.get("Expiration") or {}).get("ISO8601Date"),
        )
        rows.extend(_build_rows_for_deck(build_context, deck_context, deck))
    return rows


def _build_rows_for_deck(
    build_context: SnapshotBuildContext,
    deck_context: DeckSnapshotContext,
    deck: dict,
) -> list[tuple[Any, ...]]:
    rows: list[tuple[Any, ...]] = []
    for status, key in (
        ("active", "ActiveChallenges"),
        ("completed", "CompletedChallenges"),
        ("upcoming", "UpcomingChallenges"),
    ):
        for challenge in deck.get(key) or []:
            row = _build_snapshot_row(build_context, deck_context, challenge, status)
            if row is not None:
                rows.append(row)
    return rows


def _build_snapshot_row(
    build_context: SnapshotBuildContext,
    deck_context: DeckSnapshotContext,
    challenge: dict,
    status: str,
) -> tuple[Any, ...] | None:
    challenge_path = challenge.get("Path")
    if not isinstance(challenge_path, str) or not challenge_path.strip():
        return None

    challenge_id = challenge.get("Id")
    definition = build_context.definitions.get(challenge_path)
    stored_definition = build_context.stored.get(challenge_path)
    content_hash = build_context.definition_hashes.get(challenge_path)
    if content_hash is None and stored_definition is not None:
        content_hash = stored_definition.content_hash

    progress_current = _coerce_int(challenge.get("Progress"))
    progress_target = _resolve_progress_target(definition, stored_definition)
    xp_reward = _resolve_xp_reward(definition, stored_definition)
    can_reroll = challenge.get("CanReroll")
    if not isinstance(can_reroll, bool):
        can_reroll = None

    state_hash = _build_state_hash(
        challenge_path=challenge_path,
        challenge_id=challenge_id,
        content_hash=content_hash,
        status=status,
        progress_current=progress_current,
        progress_target=progress_target,
        xp_reward=xp_reward,
        can_reroll=can_reroll,
        expires_at=deck_context.expiry_for_hash,
    )
    return (
        build_context.snapshot_at,
        build_context.xuid,
        challenge_path,
        challenge_id,
        content_hash,
        status,
        progress_current,
        progress_target,
        xp_reward,
        can_reroll,
        deck_context.expires_at,
        deck_context.deck_index,
        state_hash,
    )


def _resolve_progress_target(
    definition: dict[str, Any] | None,
    stored_definition: StoredChallengeDefinition | None,
) -> int | None:
    if definition is not None:
        return extract_threshold_for_success(definition)
    return stored_definition.threshold_for_success if stored_definition else None


def _resolve_xp_reward(
    definition: dict[str, Any] | None,
    stored_definition: StoredChallengeDefinition | None,
) -> int:
    if definition is not None:
        return extract_challenge_xp(definition)
    return stored_definition.reward_xp if stored_definition else 0


def _load_last_snapshot_hashes(conn: Any, xuid: str) -> dict[str, str]:
    rows = conn.execute(
        """
        SELECT challenge_path, state_hash
        FROM (
            SELECT
                challenge_path,
                state_hash,
                ROW_NUMBER() OVER (PARTITION BY challenge_path ORDER BY snapshot_at DESC) AS rn
            FROM challenge_snapshots
            WHERE xuid = ?
        )
        WHERE rn = 1
        """,
        [xuid],
    ).fetchall()
    return {row[0]: row[1] for row in rows if row[0] and row[1]}


def _build_state_hash(**payload: Any) -> str:
    canonical = json.dumps(payload, sort_keys=True, separators=(",", ":"), default=str)
    return hashlib.sha1(canonical.encode("utf-8")).hexdigest()


def _parse_iso_timestamp(value: Any) -> datetime | None:
    if not isinstance(value, str) or not value.strip():
        return None
    try:
        return datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        return None


def _coerce_int(value: Any) -> int | None:
    if isinstance(value, int):
        return value
    if isinstance(value, float) and value.is_integer():
        return int(value)
    return None
