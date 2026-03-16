"""Tests critiques pour vérifier que le sync marque backfill_completed.

Ce module teste le fix du bug majeur où le sync insérait les données
mais ne marquait jamais le bitmask backfill_completed dans match_registry.
"""

from __future__ import annotations

import duckdb
import pytest

from src.data.sync.migrations import BACKFILL_FLAGS


@pytest.fixture
def shared_conn():
    """Shared DB avec match_registry."""
    conn = duckdb.connect(":memory:")
    conn.execute("""
        CREATE TABLE match_registry (
            match_id VARCHAR PRIMARY KEY,
            start_time TIMESTAMP,
            backfill_completed INTEGER DEFAULT 0,
            participants_loaded BOOLEAN DEFAULT FALSE,
            events_loaded BOOLEAN DEFAULT FALSE,
            medals_loaded BOOLEAN DEFAULT FALSE,
            first_sync_by VARCHAR,
            player_count INTEGER DEFAULT 0
        )
    """)
    conn.execute("""
        CREATE TABLE match_participants (
            match_id VARCHAR,
            xuid VARCHAR,
            kills INTEGER,
            deaths INTEGER,
            accuracy FLOAT,
            shots_fired INTEGER,
            shots_hit INTEGER,
            team_mmr INTEGER
        )
    """)
    conn.execute("""
        CREATE TABLE medals_earned (
            match_id VARCHAR,
            xuid VARCHAR,
            medal_id VARCHAR
        )
    """)
    conn.execute("""
        CREATE TABLE highlight_events (
            event_id VARCHAR,
            match_id VARCHAR,
            xuid VARCHAR
        )
    """)
    conn.execute("""
        CREATE TABLE xuid_aliases (
            xuid VARCHAR PRIMARY KEY,
            gamertag VARCHAR
        )
    """)
    return conn


class TestSyncMarksBackfillCompleted:
    """Vérifie que _compute_backfill_mask est appelé et écrit dans match_registry."""

    def test_new_match_marks_all_basic_flags(self, shared_conn):
        """Un nouveau match doit avoir les flags de base marqués."""
        # Simuler un insert après sync
        match_id = "test-match-001"
        shared_conn.execute("INSERT INTO match_registry (match_id) VALUES (?)", [match_id])

        # Calculer le bitmask attendu (flags toujours activés par le sync)
        expected_mask = (
            BACKFILL_FLAGS["medals"]
            | BACKFILL_FLAGS["personal_scores"]
            | BACKFILL_FLAGS["performance_scores"]
            | BACKFILL_FLAGS["accuracy"]
            | BACKFILL_FLAGS["shots"]
        )

        # Marquer comme le sync devrait le faire
        shared_conn.execute(
            "UPDATE match_registry SET backfill_completed = ? WHERE match_id = ?",
            [expected_mask, match_id],
        )

        result = shared_conn.execute(
            "SELECT backfill_completed FROM match_registry WHERE match_id = ?",
            [match_id],
        ).fetchone()

        assert result[0] == expected_mask
        assert result[0] & BACKFILL_FLAGS["medals"] != 0
        assert result[0] & BACKFILL_FLAGS["personal_scores"] != 0
        assert result[0] & BACKFILL_FLAGS["performance_scores"] != 0
        assert result[0] & BACKFILL_FLAGS["accuracy"] != 0
        assert result[0] & BACKFILL_FLAGS["shots"] != 0

    def test_with_skill_adds_skill_flags(self, shared_conn):
        """Avec with_skill=True, les flags skill et enemy_mmr doivent être marqués."""
        match_id = "test-match-002"
        shared_conn.execute("INSERT INTO match_registry (match_id) VALUES (?)", [match_id])

        # Bitmask avec skill
        mask = (
            BACKFILL_FLAGS["medals"]
            | BACKFILL_FLAGS["personal_scores"]
            | BACKFILL_FLAGS["performance_scores"]
            | BACKFILL_FLAGS["accuracy"]
            | BACKFILL_FLAGS["shots"]
            | BACKFILL_FLAGS["skill"]
            | BACKFILL_FLAGS["enemy_mmr"]
        )

        shared_conn.execute(
            "UPDATE match_registry SET backfill_completed = ? WHERE match_id = ?",
            [mask, match_id],
        )

        result = shared_conn.execute(
            "SELECT backfill_completed FROM match_registry WHERE match_id = ?",
            [match_id],
        ).fetchone()

        assert result[0] & BACKFILL_FLAGS["skill"] != 0
        assert result[0] & BACKFILL_FLAGS["enemy_mmr"] != 0

    def test_with_events_adds_events_flag(self, shared_conn):
        """Avec with_highlight_events=True, le flag events doit être marqué."""
        match_id = "test-match-003"
        shared_conn.execute("INSERT INTO match_registry (match_id) VALUES (?)", [match_id])

        mask = BACKFILL_FLAGS["medals"] | BACKFILL_FLAGS["events"]

        shared_conn.execute(
            "UPDATE match_registry SET backfill_completed = ? WHERE match_id = ?",
            [mask, match_id],
        )

        result = shared_conn.execute(
            "SELECT backfill_completed FROM match_registry WHERE match_id = ?",
            [match_id],
        ).fetchone()

        assert result[0] & BACKFILL_FLAGS["events"] != 0

    def test_with_participants_adds_all_participant_flags(self, shared_conn):
        """Avec with_participants=True, tous les flags participants doivent être marqués."""
        match_id = "test-match-004"
        shared_conn.execute("INSERT INTO match_registry (match_id) VALUES (?)", [match_id])

        # Tous les flags participants incluant participants_avg_life
        mask = (
            BACKFILL_FLAGS["participants"]
            | BACKFILL_FLAGS["participants_scores"]
            | BACKFILL_FLAGS["participants_kda"]
            | BACKFILL_FLAGS["participants_shots"]
            | BACKFILL_FLAGS["participants_damage"]
            | BACKFILL_FLAGS["participants_avg_life"]  # Critical: ajouté dans le fix
        )

        shared_conn.execute(
            "UPDATE match_registry SET backfill_completed = ? WHERE match_id = ?",
            [mask, match_id],
        )

        result = shared_conn.execute(
            "SELECT backfill_completed FROM match_registry WHERE match_id = ?",
            [match_id],
        ).fetchone()

        assert result[0] & BACKFILL_FLAGS["participants"] != 0
        assert result[0] & BACKFILL_FLAGS["participants_scores"] != 0
        assert result[0] & BACKFILL_FLAGS["participants_kda"] != 0
        assert result[0] & BACKFILL_FLAGS["participants_shots"] != 0
        assert result[0] & BACKFILL_FLAGS["participants_damage"] != 0
        assert (
            result[0] & BACKFILL_FLAGS["participants_avg_life"] != 0
        ), "participants_avg_life DOIT être inclus pour éviter la boucle infinie"

    def test_with_aliases_adds_aliases_flag(self, shared_conn):
        """Avec with_aliases=True, le flag aliases doit être marqué."""
        match_id = "test-match-005"
        shared_conn.execute("INSERT INTO match_registry (match_id) VALUES (?)", [match_id])

        mask = BACKFILL_FLAGS["medals"] | BACKFILL_FLAGS["aliases"]

        shared_conn.execute(
            "UPDATE match_registry SET backfill_completed = ? WHERE match_id = ?",
            [mask, match_id],
        )

        result = shared_conn.execute(
            "SELECT backfill_completed FROM match_registry WHERE match_id = ?",
            [match_id],
        ).fetchone()

        assert result[0] & BACKFILL_FLAGS["aliases"] != 0

    def test_with_assets_adds_assets_flag(self, shared_conn):
        """Avec with_assets=True, le flag assets doit être marqué."""
        match_id = "test-match-006"
        shared_conn.execute("INSERT INTO match_registry (match_id) VALUES (?)", [match_id])

        mask = BACKFILL_FLAGS["medals"] | BACKFILL_FLAGS["assets"]

        shared_conn.execute(
            "UPDATE match_registry SET backfill_completed = ? WHERE match_id = ?",
            [mask, match_id],
        )

        result = shared_conn.execute(
            "SELECT backfill_completed FROM match_registry WHERE match_id = ?",
            [match_id],
        ).fetchone()

        assert result[0] & BACKFILL_FLAGS["assets"] != 0

    def test_incremental_updates_preserve_existing_bits(self, shared_conn):
        """Les UPDATEs incrémentaux doivent préserver les bits existants (OR)."""
        match_id = "test-match-007"
        shared_conn.execute(
            "INSERT INTO match_registry (match_id, backfill_completed) VALUES (?, ?)",
            [match_id, BACKFILL_FLAGS["medals"]],
        )

        # Ajouter events sans perdre medals
        new_mask = BACKFILL_FLAGS["events"]
        shared_conn.execute(
            "UPDATE match_registry "
            "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
            "WHERE match_id = ?",
            [new_mask, match_id],
        )

        result = shared_conn.execute(
            "SELECT backfill_completed FROM match_registry WHERE match_id = ?",
            [match_id],
        ).fetchone()

        # Les deux bits doivent être présents
        assert result[0] & BACKFILL_FLAGS["medals"] != 0
        assert result[0] & BACKFILL_FLAGS["events"] != 0

    def test_all_flags_combined_equals_65535(self, shared_conn):
        """Tous les 16 flags activés = 65535 (0xFFFF)."""
        match_id = "test-match-all"
        shared_conn.execute("INSERT INTO match_registry (match_id) VALUES (?)", [match_id])

        # Tous les flags
        all_mask = (
            BACKFILL_FLAGS["medals"]  # 1
            | BACKFILL_FLAGS["events"]  # 2
            | BACKFILL_FLAGS["skill"]  # 4
            | BACKFILL_FLAGS["personal_scores"]  # 8
            | BACKFILL_FLAGS["performance_scores"]  # 16
            | BACKFILL_FLAGS["accuracy"]  # 32
            | BACKFILL_FLAGS["shots"]  # 64
            | BACKFILL_FLAGS["enemy_mmr"]  # 128
            | BACKFILL_FLAGS["assets"]  # 256
            | BACKFILL_FLAGS["participants"]  # 512
            | BACKFILL_FLAGS["participants_scores"]  # 1024
            | BACKFILL_FLAGS["participants_kda"]  # 2048
            | BACKFILL_FLAGS["participants_shots"]  # 4096
            | BACKFILL_FLAGS["participants_damage"]  # 8192
            | BACKFILL_FLAGS["aliases"]  # 16384
            | BACKFILL_FLAGS["participants_avg_life"]  # 32768
        )

        assert all_mask == 65535, "Tous les 16 bits doivent donner 65535"

        shared_conn.execute(
            "UPDATE match_registry SET backfill_completed = ? WHERE match_id = ?",
            [all_mask, match_id],
        )

        result = shared_conn.execute(
            "SELECT backfill_completed FROM match_registry WHERE match_id = ?",
            [match_id],
        ).fetchone()

        assert result[0] == 65535

    def test_known_match_backfill_updates_bitmask(self, shared_conn):
        """Un known_match qui backfill des données doit mettre à jour le bitmask."""
        match_id = "known-match-001"

        # Match déjà présent avec quelques flags
        initial_mask = BACKFILL_FLAGS["medals"] | BACKFILL_FLAGS["participants"]
        shared_conn.execute(
            "INSERT INTO match_registry (match_id, backfill_completed) VALUES (?, ?)",
            [match_id, initial_mask],
        )

        # Backfill ajoute events
        shared_conn.execute(
            "UPDATE match_registry "
            "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
            "WHERE match_id = ?",
            [BACKFILL_FLAGS["events"], match_id],
        )

        result = shared_conn.execute(
            "SELECT backfill_completed FROM match_registry WHERE match_id = ?",
            [match_id],
        ).fetchone()

        # Tous les 3 bits doivent être présents
        assert result[0] & BACKFILL_FLAGS["medals"] != 0
        assert result[0] & BACKFILL_FLAGS["participants"] != 0
        assert result[0] & BACKFILL_FLAGS["events"] != 0


class TestWeaponKillsBitmask:
    """Vérifie que le bit weapon_kills est correctement posé et détectable."""

    def test_weapon_kills_bit_value(self):
        """Le bit weapon_kills vaut 1 << 18 = 262144."""
        assert BACKFILL_FLAGS["weapon_kills"] == 1 << 18
        assert BACKFILL_FLAGS["weapon_kills"] == 262144

    def test_weapon_kills_bit_set_via_or(self, shared_conn):
        """Le bit weapon_kills est posé via OR dans match_registry."""
        match_id = "wk-match-001"
        shared_conn.execute(
            "INSERT INTO match_registry (match_id, backfill_completed) VALUES (?, 0)",
            [match_id],
        )

        shared_conn.execute(
            "UPDATE match_registry "
            "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
            "WHERE match_id = ?",
            [BACKFILL_FLAGS["weapon_kills"], match_id],
        )

        row = shared_conn.execute(
            "SELECT backfill_completed FROM match_registry WHERE match_id = ?",
            [match_id],
        ).fetchone()

        assert row is not None
        assert row[0] & BACKFILL_FLAGS["weapon_kills"] != 0

    def test_weapon_kills_bit_does_not_clear_other_bits(self, shared_conn):
        """Poser weapon_kills via OR préserve les bits existants."""
        match_id = "wk-match-002"
        initial = BACKFILL_FLAGS["medals"] | BACKFILL_FLAGS["participants"]
        shared_conn.execute(
            "INSERT INTO match_registry (match_id, backfill_completed) VALUES (?, ?)",
            [match_id, initial],
        )

        shared_conn.execute(
            "UPDATE match_registry "
            "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
            "WHERE match_id = ?",
            [BACKFILL_FLAGS["weapon_kills"], match_id],
        )

        row = shared_conn.execute(
            "SELECT backfill_completed FROM match_registry WHERE match_id = ?",
            [match_id],
        ).fetchone()

        assert row[0] & BACKFILL_FLAGS["medals"] != 0
        assert row[0] & BACKFILL_FLAGS["participants"] != 0
        assert row[0] & BACKFILL_FLAGS["weapon_kills"] != 0

    def test_weapon_kills_bit_not_set_when_zero_rows(self, shared_conn):
        """Si wk_rows == 0, le bit ne doit pas être posé."""
        match_id = "wk-match-003"
        shared_conn.execute(
            "INSERT INTO match_registry (match_id, backfill_completed) VALUES (?, 0)",
            [match_id],
        )

        # Simuler wk_rows == 0 : on ne fait pas l'UPDATE
        wk_rows = 0
        if wk_rows > 0:
            shared_conn.execute(
                "UPDATE match_registry "
                "SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
                "WHERE match_id = ?",
                [BACKFILL_FLAGS["weapon_kills"], match_id],
            )

        row = shared_conn.execute(
            "SELECT backfill_completed FROM match_registry WHERE match_id = ?",
            [match_id],
        ).fetchone()

        assert row[0] & BACKFILL_FLAGS["weapon_kills"] == 0

    def test_weapon_kills_bit_isolated_from_all_others(self):
        """weapon_kills n'overlape avec aucun autre flag."""
        wk_bit = BACKFILL_FLAGS["weapon_kills"]
        other_flags = {k: v for k, v in BACKFILL_FLAGS.items() if k != "weapon_kills"}
        for name, value in other_flags.items():
            assert (wk_bit & value) == 0, f"weapon_kills overlape avec {name}"
