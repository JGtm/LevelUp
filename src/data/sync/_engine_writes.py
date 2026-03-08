"""Mixin — écritures dans la DB joueur (enrichissements personnels).

Regroupe les méthodes d'écriture dans stats.duckdb du joueur :
highlight_events, personal_score_awards, player_match_enrichment.
"""

from __future__ import annotations

import logging
from datetime import datetime, timezone
from typing import TYPE_CHECKING

from src.data.sync.batch_insert import (
    HIGHLIGHT_EVENT_COLUMNS,
    PERSONAL_SCORE_COLUMNS,
    batch_insert_rows,
)
from src.data.sync.models import MatchStatsRow

if TYPE_CHECKING:
    from src.data.sync._protocol import _SyncProtocol

logger = logging.getLogger(__name__)


class EnrichedWritesMixin:
    """Méthodes d'écriture des enrichissements dans la DB joueur."""

    def _insert_event_rows(self: _SyncProtocol, rows: list) -> None:
        """Insère des lignes highlight_events en batch (Sprint 15)."""
        if not rows:
            return
        conn = self._get_connection()
        batch_insert_rows(conn, "highlight_events", rows, HIGHLIGHT_EVENT_COLUMNS)

    def _insert_personal_score_rows(self: _SyncProtocol, rows: list) -> None:
        """Insère des lignes personal_score_awards en batch (Sprint 15)."""
        if not rows:
            return
        conn = self._get_connection()
        now = datetime.now(timezone.utc)
        score_dicts = []
        for row in rows:
            score_dicts.append(
                {
                    "match_id": row.match_id,
                    "xuid": row.xuid,
                    "award_name": row.award_name,
                    "award_category": row.award_category,
                    "award_count": row.award_count,
                    "award_score": row.award_score,
                    "created_at": now,
                }
            )
        batch_insert_rows(conn, "personal_score_awards", score_dicts, PERSONAL_SCORE_COLUMNS)

    def _load_friends_lazy(self: _SyncProtocol) -> frozenset[str]:
        """Charge les XUIDs des amis depuis friends_defaults.json (cache interne)."""
        if self._friends_xuids is None:
            try:
                from src.data.sessions_backfill import get_friends_xuids_for_backfill

                conn = self._get_connection()
                self._friends_xuids = get_friends_xuids_for_backfill(
                    self._player_db_path,
                    self._xuid or "",
                    conn=conn,
                )
            except Exception:
                self._friends_xuids = frozenset()
        return self._friends_xuids

    def _insert_enrichment_row(  # noqa: PLR0912
        self: _SyncProtocol, match_id: str, match_row: MatchStatsRow
    ) -> None:
        """Insère/met à jour une ligne dans player_match_enrichment (V5 finale)."""
        conn = self._get_connection()
        now = datetime.now(timezone.utc)

        teammates_sig = getattr(match_row, "teammates_signature", None)

        try:
            from src.analysis.sessions import _parse_teammates_signature

            friends = self._load_friends_lazy()
            if friends:
                team_set = _parse_teammates_signature(teammates_sig)
                common = team_set & friends
                is_with_friends: bool | None = bool(common)
                known_teammates: int | None = len(common)
                friends_xuids: str | None = ",".join(sorted(common)) if common else None
            else:
                is_with_friends = None
                known_teammates = None
                friends_xuids = None
        except Exception:
            is_with_friends = None
            known_teammates = None
            friends_xuids = None

        conn.execute(
            """INSERT INTO player_match_enrichment
                (match_id, teammates_signature, is_with_friends,
                 known_teammates_count, friends_xuids, created_at, updated_at)
            VALUES (?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT (match_id) DO UPDATE SET
                teammates_signature = COALESCE(EXCLUDED.teammates_signature, player_match_enrichment.teammates_signature),
                is_with_friends = COALESCE(EXCLUDED.is_with_friends, player_match_enrichment.is_with_friends),
                known_teammates_count = COALESCE(EXCLUDED.known_teammates_count, player_match_enrichment.known_teammates_count),
                friends_xuids = COALESCE(EXCLUDED.friends_xuids, player_match_enrichment.friends_xuids),
                updated_at = EXCLUDED.updated_at
            """,
            (match_id, teammates_sig, is_with_friends, known_teammates, friends_xuids, now, now),
        )
