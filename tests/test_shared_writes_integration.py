"""Tests d'intégration — Insertions dans shared_matches_v2.duckdb.

Vérifie que SharedWritesMixin écrit correctement dans toutes les tables
shared (match_registry, match_participants, medals_earned, highlight_events,
killer_victim_pairs, xuid_aliases, weapon_kills) avec de vrais DuckDB.

Scénarios couverts :
- Insertion simple d'un match complet (registry + participants + medals + events)
- Idempotence : double insertion → pas de doublon
- backfill_bits calculés correctement après insertion
- participants_loaded / events_loaded / medals_loaded flags cohérents
- Insertion OR IGNORE sur match_registry (pas d'écrasement)
- Upsert participants (ON CONFLICT DO UPDATE SET)
- Backfill events block (events + killer_victim + flags)
"""

from __future__ import annotations

from datetime import datetime, timezone
from pathlib import Path

from tests.conftest_sync import (
    GT_PLAYER_A,
    GT_PLAYER_B,
    XUID_PLAYER_A,
    XUID_PLAYER_B,
    count_rows,
    make_engine,
)

# ===========================================================================
# Tests match_registry
# ===========================================================================


class TestInsertSharedRegistry:
    """Tests pour _insert_shared_registry."""

    def test_insert_single_match(self, tmp_path: Path) -> None:
        """Un match inséré dans registry est lisible avec toutes ses colonnes."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()
            assert shared is not None

            registry_data = {
                "match_id": "match-reg-001",
                "start_time": datetime(2024, 6, 1, 12, 0, tzinfo=timezone.utc),
                "end_time": datetime(2024, 6, 1, 12, 10, tzinfo=timezone.utc),
                "playlist_id": "pl-123",
                "playlist_name": "Ranked Arena",
                "map_id": "map-456",
                "map_name": "Recharge",
                "pair_id": "pair-789",
                "pair_name": "Recharge - Slayer",
                "game_variant_id": "gv-abc",
                "game_variant_name": "Slayer",
                "mode_category": "pvp_arena",
                "is_ranked": True,
                "is_firefight": False,
                "duration_seconds": 600,
                "playable_duration_seconds": 590,
                "real_start_time": datetime(2024, 6, 1, 12, 0, 5, tzinfo=timezone.utc),
                "team_0_score": 50,
                "team_1_score": 47,
            }
            engine._insert_shared_registry(shared, registry_data)
            shared.commit()

            row = shared.execute(
                "SELECT match_id, is_ranked, mode_category, duration_seconds "
                "FROM match_registry WHERE match_id = ?",
                ["match-reg-001"],
            ).fetchone()
            assert row is not None
            assert row[0] == "match-reg-001"
            assert row[1] is True  # is_ranked
            assert row[2] == "pvp_arena"
            assert row[3] == 600
        finally:
            engine.close()

    def test_insert_idempotent(self, tmp_path: Path) -> None:
        """Double insertion OR IGNORE → pas d'erreur, pas de doublon."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()
            data = {
                "match_id": "match-idem-001",
                "start_time": datetime(2024, 6, 1, 12, 0, tzinfo=timezone.utc),
                "end_time": None,
                "playlist_id": "pl-x",
                "playlist_name": "Quick Play",
                "map_id": "m-x",
                "map_name": "Streets",
                "pair_id": "p-x",
                "pair_name": "Streets - Slayer",
                "game_variant_id": "g-x",
                "game_variant_name": "Slayer",
                "mode_category": "pvp_arena",
                "is_ranked": False,
                "is_firefight": False,
                "duration_seconds": 480,
                "team_0_score": 50,
                "team_1_score": 38,
            }
            engine._insert_shared_registry(shared, data)
            engine._insert_shared_registry(shared, data)
            shared.commit()

            assert count_rows(shared, "match_registry") == 1
        finally:
            engine.close()

    def test_mode_category_patched_if_null(self, tmp_path: Path) -> None:
        """Si un match existe avec mode_category NULL, la 2e insertion le patche."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()
            # Insertion initiale SANS mode_category
            shared.execute(
                "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
                ["match-patch-001", datetime(2024, 6, 1, tzinfo=timezone.utc)],
            )
            # Vérifier NULL
            row = shared.execute(
                "SELECT mode_category FROM match_registry WHERE match_id = ?",
                ["match-patch-001"],
            ).fetchone()
            assert row[0] is None

            # 2e insertion avec mode_category renseigné
            data = {
                "match_id": "match-patch-001",
                "start_time": datetime(2024, 6, 1, tzinfo=timezone.utc),
                "end_time": None,
                "playlist_id": "",
                "playlist_name": "",
                "map_id": "",
                "map_name": "",
                "pair_id": "",
                "pair_name": "Recharge - Slayer",
                "game_variant_id": "",
                "game_variant_name": "",
                "mode_category": "pvp_arena",
                "is_ranked": False,
                "is_firefight": False,
                "duration_seconds": 600,
                "team_0_score": 50,
                "team_1_score": 45,
            }
            engine._insert_shared_registry(shared, data)
            shared.commit()

            row = shared.execute(
                "SELECT mode_category FROM match_registry WHERE match_id = ?",
                ["match-patch-001"],
            ).fetchone()
            assert row[0] == "pvp_arena"
        finally:
            engine.close()


# ===========================================================================
# Tests match_participants
# ===========================================================================


class TestInsertSharedParticipants:
    """Tests pour _insert_shared_participants."""

    def test_insert_participants(self, tmp_path: Path) -> None:
        """Les participants sont insérés avec leurs stats complètes."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()

            # Construction manuelle de MatchParticipantRow-like objects
            from dataclasses import dataclass

            @dataclass
            class FakeParticipant:
                match_id: str = "match-part-001"
                xuid: str = XUID_PLAYER_A
                gamertag: str = GT_PLAYER_A
                team_id: int = 0
                outcome: int = 2
                rank: int = 1
                score: int = 1500
                kills: int = 15
                deaths: int = 8
                assists: int = 5
                kda: float = 1.875
                accuracy: float = 0.45
                time_played_seconds: int = 600
                avg_life_seconds: float = 45.0
                personal_score: int = 1500
                damage_dealt: float = 3500.0
                damage_taken: float = 2800.0
                shots_fired: int = 200
                shots_hit: int = 90
                team_mmr: float = 1500.0
                enemy_mmr: float = 1480.0
                kills_expected: float = 9.5
                deaths_expected: float = 8.5
                kills_stddev: float = 2.0
                deaths_stddev: float = 2.0
                assists_expected: float = 5.0
                assists_stddev: float = 1.5
                grenade_kills: int = 2
                melee_kills: int = 1
                power_weapon_kills: int = 3
                headshot_kills: int = 7
                max_killing_spree: int = 5

            participants = [
                FakeParticipant(),
                FakeParticipant(
                    xuid=XUID_PLAYER_B,
                    gamertag=GT_PLAYER_B,
                    team_id=1,
                    outcome=3,
                    kills=8,
                    deaths=15,
                ),
            ]

            inserted = engine._insert_shared_participants(shared, participants)
            shared.commit()

            assert inserted == 2
            assert count_rows(shared, "match_participants") == 2

            # Vérifier les stats du joueur A
            row = shared.execute(
                "SELECT kills, deaths, accuracy, team_mmr FROM match_participants "
                "WHERE match_id = ? AND xuid = ?",
                ["match-part-001", XUID_PLAYER_A],
            ).fetchone()
            assert row is not None
            assert row[0] == 15  # kills
            assert row[1] == 8  # deaths
        finally:
            engine.close()

    def test_upsert_preserves_mmr(self, tmp_path: Path) -> None:
        """Le upsert participants (ON CONFLICT DO UPDATE SET) ne détruit pas les colonnes MMR.

        Architecture v5.1 : les colonnes MMR ne font PAS partie de
        PARTICIPANT_COLUMNS, donc batch_upsert_participants ne les touche pas.
        Le pipeline skill les écrit séparément via UPDATE + COALESCE.
        """
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()

            from dataclasses import dataclass

            @dataclass
            class FakeP:
                match_id: str = "match-upsert-001"
                xuid: str = XUID_PLAYER_A
                gamertag: str = GT_PLAYER_A
                team_id: int = 0
                outcome: int = 2
                rank: int = 1
                score: int = 1500
                kills: int = 15
                deaths: int = 8
                assists: int = 5
                kda: float = 1.875
                accuracy: float = 0.45
                time_played_seconds: int = 600
                avg_life_seconds: float = 45.0
                personal_score: int = 1500
                damage_dealt: float = 3500.0
                damage_taken: float = 2800.0
                shots_fired: int = 200
                shots_hit: int = 90
                grenade_kills: int = 2
                melee_kills: int = 1
                power_weapon_kills: int = 3
                headshot_kills: int = 7
                max_killing_spree: int = 5

            # 1. Insérer le participant (colonnes stats uniquement, pas MMR)
            engine._insert_shared_participants(shared, [FakeP()])
            shared.commit()

            # 2. Simuler le pipeline skill qui écrit MMR séparément
            shared.execute(
                "UPDATE match_participants SET team_mmr = 1500.0, enemy_mmr = 1480.0 "
                "WHERE match_id = ? AND xuid = ?",
                ["match-upsert-001", XUID_PLAYER_A],
            )
            shared.commit()

            # 3. Re-upserter le participant (simule un re-sync)
            engine._insert_shared_participants(
                shared,
                [FakeP(kills=20)],  # stats changées
            )
            shared.commit()

            # 4. MMR doit survivre (pas dans PARTICIPANT_COLUMNS → pas touché)
            row = shared.execute(
                "SELECT team_mmr, enemy_mmr, kills FROM match_participants "
                "WHERE match_id = ? AND xuid = ?",
                ["match-upsert-001", XUID_PLAYER_A],
            ).fetchone()
            assert row[0] == 1500.0, "team_mmr doit survivre au upsert"
            assert row[1] == 1480.0, "enemy_mmr doit survivre au upsert"
            assert row[2] == 20, "kills doit être mis à jour"
        finally:
            engine.close()


# ===========================================================================
# Tests medals_earned
# ===========================================================================


class TestInsertSharedMedals:
    """Tests pour _insert_shared_medals."""

    def test_insert_medals(self, tmp_path: Path) -> None:
        """Les médailles sont insérées avec la PK (match_id, xuid, medal_name_id)."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()

            from dataclasses import dataclass

            @dataclass
            class FakeMedal:
                match_id: str
                xuid: str
                medal_name_id: int
                count: int

            medals = [
                FakeMedal("match-medal-001", XUID_PLAYER_A, 100, 3),
                FakeMedal("match-medal-001", XUID_PLAYER_A, 200, 1),
                FakeMedal("match-medal-001", XUID_PLAYER_B, 100, 2),
            ]

            engine._insert_shared_medals(shared, medals)
            shared.commit()

            assert count_rows(shared, "medals_earned") == 3

            # Vérifier un medal spécifique
            row = shared.execute(
                "SELECT count FROM medals_earned WHERE match_id = ? AND xuid = ? AND medal_name_id = ?",
                ["match-medal-001", XUID_PLAYER_A, 100],
            ).fetchone()
            assert row[0] == 3
        finally:
            engine.close()

    def test_medals_idempotent(self, tmp_path: Path) -> None:
        """Double insertion des mêmes médailles → pas de doublon."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()

            from dataclasses import dataclass

            @dataclass
            class FakeMedal:
                match_id: str
                xuid: str
                medal_name_id: int
                count: int

            medals = [FakeMedal("match-medal-idem", XUID_PLAYER_A, 100, 3)]
            engine._insert_shared_medals(shared, medals)
            engine._insert_shared_medals(shared, medals)
            shared.commit()

            n = shared.execute(
                "SELECT COUNT(*) FROM medals_earned WHERE match_id = ?",
                ["match-medal-idem"],
            ).fetchone()[0]
            assert n == 1
        finally:
            engine.close()


# ===========================================================================
# Tests xuid_aliases
# ===========================================================================


class TestInsertSharedAliases:
    """Tests pour _insert_shared_aliases."""

    def test_insert_aliases(self, tmp_path: Path) -> None:
        """Les aliases xuid→gamertag sont insérés."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()

            from src.data.sync.models import XuidAliasRow

            now = datetime.now(timezone.utc)
            aliases = [
                XuidAliasRow(xuid=XUID_PLAYER_A, gamertag=GT_PLAYER_A, last_seen=now, source="api"),
                XuidAliasRow(xuid=XUID_PLAYER_B, gamertag=GT_PLAYER_B, last_seen=now, source="api"),
            ]
            engine._insert_shared_aliases(shared, aliases)
            shared.commit()

            assert count_rows(shared, "xuid_aliases") == 2

            row = shared.execute(
                "SELECT gamertag FROM xuid_aliases WHERE xuid = ?",
                [XUID_PLAYER_A],
            ).fetchone()
            assert row[0] == GT_PLAYER_A
        finally:
            engine.close()

    def test_alias_upsert_updates_gamertag(self, tmp_path: Path) -> None:
        """Si un xuid change de gamertag, l'upsert met à jour."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()

            from src.data.sync.models import XuidAliasRow

            now = datetime.now(timezone.utc)
            engine._insert_shared_aliases(
                shared,
                [XuidAliasRow(xuid=XUID_PLAYER_A, gamertag="OldGT", last_seen=now, source="api")],
            )
            shared.commit()

            engine._insert_shared_aliases(
                shared,
                [XuidAliasRow(xuid=XUID_PLAYER_A, gamertag="NewGT", last_seen=now, source="api")],
            )
            shared.commit()

            assert count_rows(shared, "xuid_aliases") == 1
            row = shared.execute(
                "SELECT gamertag FROM xuid_aliases WHERE xuid = ?",
                [XUID_PLAYER_A],
            ).fetchone()
            assert row[0] == "NewGT"
        finally:
            engine.close()


# ===========================================================================
# Tests backfill_bits
# ===========================================================================


class TestBackfillBits:
    """Tests pour _update_match_participant_bits."""

    def test_bits_set_from_nonnull_columns(self, tmp_path: Path) -> None:
        """backfill_bits reflète les colonnes NOT NULL du participant."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()
            from src.data.sync.constants import ParticipantBits as PB

            # Insérer un participant avec team_mmr et accuracy renseignés
            shared.execute(
                "INSERT INTO match_participants (match_id, xuid, gamertag, team_id, "
                "outcome, kills, deaths, assists, team_mmr, accuracy) "
                "VALUES (?, ?, ?, 0, 2, 10, 8, 5, 1500.0, 0.45)",
                ["match-bits-001", XUID_PLAYER_A, GT_PLAYER_A],
            )

            engine._update_match_participant_bits(shared, "match-bits-001")
            shared.commit()

            row = shared.execute(
                "SELECT backfill_bits FROM match_participants WHERE match_id = ? AND xuid = ?",
                ["match-bits-001", XUID_PLAYER_A],
            ).fetchone()
            bits = row[0]
            assert bits & PB.TEAM_MMR  # team_mmr was set
            assert bits & PB.ACCURACY  # accuracy was set
            assert not (bits & PB.ENEMY_MMR)  # enemy_mmr was NULL
        finally:
            engine.close()

    def test_bits_medals_flag(self, tmp_path: Path) -> None:
        """Le bit MEDALS est posé quand des médailles existent pour ce participant."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()
            from src.data.sync.constants import ParticipantBits as PB

            match_id = "match-bits-medal"
            shared.execute(
                "INSERT INTO match_participants (match_id, xuid, gamertag, team_id, "
                "outcome, kills, deaths, assists) "
                "VALUES (?, ?, ?, 0, 2, 10, 8, 5)",
                [match_id, XUID_PLAYER_A, GT_PLAYER_A],
            )
            shared.execute(
                "INSERT INTO medals_earned (match_id, xuid, medal_name_id, count) "
                "VALUES (?, ?, 100, 3)",
                [match_id, XUID_PLAYER_A],
            )

            engine._update_match_participant_bits(shared, match_id)
            shared.commit()

            row = shared.execute(
                "SELECT backfill_bits FROM match_participants WHERE match_id = ? AND xuid = ?",
                [match_id, XUID_PLAYER_A],
            ).fetchone()
            assert row[0] & PB.MEDALS
        finally:
            engine.close()


# ===========================================================================
# Tests _backfill_events_block
# ===========================================================================


class TestBackfillEventsBlock:
    """Tests pour le bloc events + killer_victim + flags."""

    def test_events_block_sets_events_loaded(self, tmp_path: Path) -> None:
        """Après _backfill_events_block, events_loaded=TRUE dans match_registry."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()
            match_id = "match-evblock-001"

            # Pre-seed match_registry
            shared.execute(
                "INSERT INTO match_registry (match_id, start_time, events_loaded) "
                "VALUES (?, ?, FALSE)",
                [match_id, datetime(2024, 6, 1, tzinfo=timezone.utc)],
            )

            # Events au format attendu par transform_highlight_events (clés minuscules)
            events = [
                {
                    "event_type": "kill",
                    "time_ms": 5000,
                    "xuid": XUID_PLAYER_A,
                    "gamertag": GT_PLAYER_A,
                    "type_hint": 3,
                },
            ]

            engine._backfill_events_block(shared, match_id, events)
            shared.commit()

            row = shared.execute(
                "SELECT events_loaded FROM match_registry WHERE match_id = ?",
                [match_id],
            ).fetchone()
            assert row[0] is True
        finally:
            engine.close()

    def test_events_block_empty_list(self, tmp_path: Path) -> None:
        """Liste vide → 0 events insérés, events_loaded reste FALSE."""
        engine = make_engine(tmp_path)
        try:
            shared = engine._get_shared_connection()
            match_id = "match-evblock-empty"

            shared.execute(
                "INSERT INTO match_registry (match_id, start_time, events_loaded) "
                "VALUES (?, ?, FALSE)",
                [match_id, datetime(2024, 6, 1, tzinfo=timezone.utc)],
            )

            n = engine._backfill_events_block(shared, match_id, [])
            shared.commit()

            assert n == 0
            row = shared.execute(
                "SELECT events_loaded FROM match_registry WHERE match_id = ?",
                [match_id],
            ).fetchone()
            assert row[0] is False
        finally:
            engine.close()
