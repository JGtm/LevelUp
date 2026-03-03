"""Mixin — insertions dans shared_matches.duckdb.

Regroupe les méthodes d'écriture dans la base partagée :
match_registry, match_participants, highlight_events, medals_earned, xuid_aliases.
"""

from __future__ import annotations

import logging
from datetime import datetime, timezone
from typing import TYPE_CHECKING, Any

import duckdb

from src.data.sync.batch_insert import ALIAS_COLUMNS, PARTICIPANT_COLUMNS, batch_upsert_rows

if TYPE_CHECKING:
    pass

logger = logging.getLogger(__name__)


class SharedWritesMixin:
    """Méthodes d'écriture dans shared_matches.duckdb."""

    def _insert_shared_registry(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        data: dict[str, Any],
    ) -> None:
        """Insère un match dans match_registry (shared).

        Args:
            shared_conn: Connexion vers shared_matches.duckdb.
            data: Dict retourné par extract_match_registry_data().
        """
        shared_conn.execute(
            """INSERT OR IGNORE INTO match_registry (
                match_id, start_time, end_time,
                playlist_id, playlist_name,
                map_id, map_name,
                pair_id, pair_name,
                game_variant_id, game_variant_name,
                mode_category, is_ranked, is_firefight,
                duration_seconds,
                team_0_score, team_1_score,
                sync_spnkr_version,
                created_at, updated_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)""",
            (
                data["match_id"],
                data["start_time"],
                data["end_time"],
                data["playlist_id"],
                data["playlist_name"],
                data["map_id"],
                data["map_name"],
                data["pair_id"],
                data["pair_name"],
                data["game_variant_id"],
                data["game_variant_name"],
                data["mode_category"],
                data["is_ranked"],
                data["is_firefight"],
                data["duration_seconds"],
                data["team_0_score"],
                data["team_1_score"],
                self._spnkr_version,  # type: ignore[attr-defined]
            ),
        )
        # Si le match existait déjà avec mode_category NULL (anciens matchs insérés
        # avant le calcul de _determine_mode_category), on le patche maintenant.
        # pair_name est aussi mis à jour si absent (UUID brut → nom résolu).
        if data.get("mode_category"):
            shared_conn.execute(
                """UPDATE match_registry
                   SET mode_category  = ?,
                       pair_name      = COALESCE(pair_name, ?),
                       updated_at     = CURRENT_TIMESTAMP
                   WHERE match_id = ? AND mode_category IS NULL""",
                (
                    data["mode_category"],
                    data["pair_name"],
                    data["match_id"],
                ),
            )

    def _insert_shared_participants(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        participants: list,
    ) -> None:
        """Insère les participants dans shared.match_participants.

        Architecture v5.1 :
            Utilise batch_upsert_participants() (ON CONFLICT DO UPDATE SET)
            au lieu de l'ancien batch_upsert_rows() (INSERT OR REPLACE) qui
            détruisait les colonnes MMR déjà peuplées par le pipeline skill.

        Args:
            shared_conn: Connexion vers shared_matches.duckdb.
            participants: Liste de MatchParticipantRow.
        """
        if not participants:
            return
        from src.data.sync.batch_insert import batch_upsert_participants

        batch_upsert_participants(shared_conn, participants, PARTICIPANT_COLUMNS)

    def _update_match_participant_bits(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        match_id: str,
    ) -> None:
        """Met à jour ``backfill_bits`` pour tous les participants d'un match (vectorisé).

        Lit l'état réel des colonnes NOT NULL dans match_participants et pose
        les bits correspondants via une requête SQL unique (sans boucle Python).
        Appelé après chaque insertion/mise à jour de participants pendant le sync.

        Args:
            shared_conn: Connexion vers shared_matches.duckdb.
            match_id: ID du match à mettre à jour.
        """
        from src.data.sync.constants import ParticipantBits as PB

        try:
            shared_conn.execute(
                """UPDATE match_participants SET backfill_bits = COALESCE(backfill_bits, 0) | (
                    CASE WHEN team_mmr IS NOT NULL THEN ? ELSE 0 END |
                    CASE WHEN enemy_mmr IS NOT NULL THEN ? ELSE 0 END |
                    CASE WHEN kills_expected IS NOT NULL THEN ? ELSE 0 END |
                    CASE WHEN deaths_expected IS NOT NULL THEN ? ELSE 0 END |
                    CASE WHEN assists_expected IS NOT NULL THEN ? ELSE 0 END |
                    CASE WHEN accuracy IS NOT NULL THEN ? ELSE 0 END |
                    CASE WHEN shots_fired IS NOT NULL THEN ? ELSE 0 END |
                    CASE WHEN damage_dealt IS NOT NULL THEN ? ELSE 0 END |
                    CASE WHEN avg_life_seconds IS NOT NULL THEN ? ELSE 0 END |
                    CASE WHEN grenade_kills IS NOT NULL THEN ? ELSE 0 END |
                    CASE WHEN melee_kills IS NOT NULL THEN ? ELSE 0 END |
                    CASE WHEN power_weapon_kills IS NOT NULL THEN ? ELSE 0 END |
                    CASE WHEN personal_score IS NOT NULL THEN ? ELSE 0 END |
                    CASE WHEN headshot_kills IS NOT NULL THEN ? ELSE 0 END |
                    CASE WHEN max_killing_spree IS NOT NULL THEN ? ELSE 0 END |
                    CASE WHEN kda IS NOT NULL THEN ? ELSE 0 END |
                    CASE WHEN time_played_seconds IS NOT NULL THEN ? ELSE 0 END
                )
                WHERE match_id = ?""",
                [
                    PB.TEAM_MMR,
                    PB.ENEMY_MMR,
                    PB.KILLS_EXP,
                    PB.DEATHS_EXP,
                    PB.ASSISTS_EXP,
                    PB.ACCURACY,
                    PB.SHOTS,
                    PB.DAMAGE,
                    PB.AVG_LIFE,
                    PB.GRENADE_KILLS,
                    PB.MELEE_KILLS,
                    PB.POWER_WEAPON,
                    PB.PERSONAL_SCORE,
                    PB.HEADSHOT_KILLS,
                    PB.MAX_SPREE,
                    PB.KDA,
                    PB.TIME_PLAYED,
                    match_id,
                ],
            )
            # Médailles : table séparée, requête dédiée
            shared_conn.execute(
                """UPDATE match_participants
                   SET backfill_bits = COALESCE(backfill_bits, 0) | ?
                   WHERE match_id = ?
                   AND EXISTS (
                       SELECT 1 FROM medals_earned me
                       WHERE me.match_id = match_participants.match_id
                         AND me.xuid = match_participants.xuid
                   )""",
                [PB.MEDALS, match_id],
            )
        except Exception as e:
            logger.debug(f"Mise à jour backfill_bits match {match_id}: {e}")

    def _insert_shared_events(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        event_rows: list,
    ) -> None:
        """Insère les highlight events dans shared.highlight_events.

        Args:
            shared_conn: Connexion vers shared_matches.duckdb.
            event_rows: Liste de HighlightEventRow.
        """
        if not event_rows:
            return
        from src.data.sync.batch_insert import HIGHLIGHT_EVENT_COLUMNS, batch_insert_rows

        batch_insert_rows(shared_conn, "highlight_events", event_rows, HIGHLIGHT_EVENT_COLUMNS)

    def _insert_shared_medals(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        medals: list,
    ) -> None:
        """Insère les médailles de TOUS les joueurs dans shared.medals_earned.

        Args:
            shared_conn: Connexion vers shared_matches.duckdb.
            medals: Liste de SharedMedalEarnedRow.
        """
        if not medals:
            return
        columns = ["match_id", "xuid", "medal_name_id", "count"]

        batch_upsert_rows(shared_conn, "medals_earned", medals, columns)

    def _insert_shared_aliases(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        alias_rows: list,
    ) -> None:
        """Insère les aliases xuid→gamertag dans shared.xuid_aliases.

        Args:
            shared_conn: Connexion vers shared_matches.duckdb.
            alias_rows: Liste de XuidAliasRow.
        """
        if not alias_rows:
            return

        now = datetime.now(timezone.utc)
        alias_dicts = [
            {
                "xuid": row.xuid,
                "gamertag": row.gamertag,
                "last_seen": row.last_seen,
                "source": row.source,
                "updated_at": now,
            }
            for row in alias_rows
        ]
        batch_upsert_rows(shared_conn, "xuid_aliases", alias_dicts, ALIAS_COLUMNS)
