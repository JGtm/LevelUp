"""Fonctions d'insertion de base pour le backfill DuckDB.

Responsabilités :
- Insertion de médailles, events, skill, personal scores, aliases, participants
- Utilise les insertions batch (Sprint 15) pour de meilleures performances
- Aucun commit : le commit est géré par l'orchestrateur
"""

from __future__ import annotations

import contextlib
import logging
from datetime import datetime, timezone
from typing import Any

from src.data.sync.batch_insert import (
    ALIAS_COLUMNS,
    HIGHLIGHT_EVENT_COLUMNS,
    MEDAL_COLUMNS,
    PARTICIPANT_COLUMNS,
    PERSONAL_SCORE_COLUMNS,
    SHARED_MEDAL_COLUMNS,
    batch_insert_rows,
    batch_upsert_participants,
    batch_upsert_rows,
)

logger = logging.getLogger(__name__)


# ─────────────────────────────────────────────────────────────────────────────
# Insertion helpers
# ─────────────────────────────────────────────────────────────────────────────


def insert_medal_rows(conn: Any, rows: list, xuid: str | None = None) -> int:
    """Insère les médailles dans la table medals_earned (batch Sprint 15).

    Args:
        conn: Connexion DuckDB.
        rows: Liste de MedalRow (avec ou sans xuid).
        xuid: XUID du joueur. Si fourni, sera ajouté à chaque row pour
              insertion dans shared_matches_v2.duckdb.

    Returns:
        Nombre de médailles insérées.
    """
    if not rows:
        return 0

    # Si xuid fourni, convertir en SharedMedalEarnedRow format
    if xuid:
        shared_rows = []
        for r in rows:
            if hasattr(r, "match_id"):
                shared_rows.append(
                    {
                        "match_id": r.match_id,
                        "xuid": xuid,
                        "medal_name_id": r.medal_name_id,
                        "count": r.count,
                    }
                )
            elif isinstance(r, dict):
                shared_rows.append(
                    {
                        "match_id": r.get("match_id"),
                        "xuid": xuid,
                        "medal_name_id": r.get("medal_name_id"),
                        "count": r.get("count"),
                    }
                )
        return batch_upsert_rows(conn, "medals_earned", shared_rows, SHARED_MEDAL_COLUMNS)

    return batch_upsert_rows(conn, "medals_earned", rows, MEDAL_COLUMNS)


def insert_event_rows(conn: Any, rows: list) -> int:
    """Insère les highlight events en batch (Sprint 15).

    Supprime d'abord les events existants pour le match_id afin d'éviter
    les doublons (highlight_events n'a pas de contrainte UNIQUE).

    Args:
        conn: Connexion DuckDB.
        rows: Liste de HighlightEventRow.

    Returns:
        Nombre d'events insérés.
    """
    if not rows:
        return 0

    # Déduplication : supprimer les events existants pour ces match_ids
    match_ids = list(
        {
            getattr(r, "match_id", None) or (r.get("match_id") if isinstance(r, dict) else None)
            for r in rows
        }
    )
    match_ids = [m for m in match_ids if m is not None]
    if match_ids:
        placeholders = ", ".join(["?"] * len(match_ids))
        conn.execute(
            f"DELETE FROM highlight_events WHERE match_id IN ({placeholders})",
            match_ids,
        )

    return batch_insert_rows(conn, "highlight_events", rows, HIGHLIGHT_EVENT_COLUMNS)


def insert_personal_score_rows(conn: Any, rows: list) -> int:
    """Insère les personal score awards en batch (Sprint 15).

    Supprime d'abord les scores existants pour le match_id + xuid
    afin d'éviter les doublons.

    Args:
        conn: Connexion DuckDB.
        rows: Liste de PersonalScoreAwardRow.

    Returns:
        Nombre de scores insérés.
    """
    if not rows:
        return 0

    # Déduplication : supprimer les scores existants pour ces match_id/xuid
    keys = list({(r.match_id, r.xuid) for r in rows if hasattr(r, "match_id")})
    for match_id, xuid in keys:
        conn.execute(
            "DELETE FROM personal_score_awards WHERE match_id = ? AND xuid = ?",
            [match_id, xuid],
        )

    now = datetime.now(timezone.utc)
    dicts = []
    for row in rows:
        dicts.append(
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
    return batch_insert_rows(conn, "personal_score_awards", dicts, PERSONAL_SCORE_COLUMNS)


def insert_alias_rows(conn: Any, rows: list) -> int:
    """Insère les aliases XUID en batch (Sprint 15).

    Args:
        conn: Connexion DuckDB.
        rows: Liste de XuidAliasRow.

    Returns:
        Nombre d'aliases insérés.
    """
    if not rows:
        return 0
    now = datetime.now(timezone.utc)
    dicts = []
    for row in rows:
        dicts.append(
            {
                "xuid": row.xuid,
                "gamertag": row.gamertag,
                "last_seen": row.last_seen.isoformat() if row.last_seen else None,
                "source": row.source,
                "updated_at": now.isoformat(),
            }
        )
    return batch_upsert_rows(conn, "xuid_aliases", dicts, ALIAS_COLUMNS)


def insert_participant_rows(conn: Any, rows: list) -> int:
    """Insère les participants dans match_participants en batch (Sprint 15).

    Crée la table si elle n'existe pas et s'assure que les colonnes
    sont à jour via migrations.

    Args:
        conn: Connexion DuckDB.
        rows: Liste de ParticipantRow.

    Returns:
        Nombre de participants insérés.
    """
    if not rows:
        return 0

    from src.data.sync.migrations import ensure_match_participants_columns

    conn.execute("""
        CREATE TABLE IF NOT EXISTS match_participants (
            match_id VARCHAR NOT NULL,
            xuid VARCHAR NOT NULL,
            team_id INTEGER,
            outcome INTEGER,
            gamertag VARCHAR,
            rank SMALLINT,
            score INTEGER,
            kills SMALLINT,
            deaths SMALLINT,
            assists SMALLINT,
            PRIMARY KEY (match_id, xuid)
        )
    """)
    ensure_match_participants_columns(conn)

    with contextlib.suppress(Exception):
        conn.execute("CREATE INDEX IF NOT EXISTS idx_participants_xuid ON match_participants(xuid)")
        conn.execute(
            "CREATE INDEX IF NOT EXISTS idx_participants_team ON match_participants(match_id, team_id)"
        )

    return batch_upsert_participants(conn, rows, PARTICIPANT_COLUMNS)
