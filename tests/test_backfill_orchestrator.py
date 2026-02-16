"""Tests pour scripts/backfill/orchestrator.py — fonctions helpers pures et DuckDB.

Couvre : _empty_result, _update_accuracy_shots, _backfill_local_only, _resolve_xuid_fallback.
"""

from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime, timezone
from unittest.mock import patch

import duckdb
import pytest

# ─────────────────────────────────────────────────────────────────────────────
# Fixtures
# ─────────────────────────────────────────────────────────────────────────────


@pytest.fixture()
def conn():
    """Connexion DuckDB in-memory (player DB V5)."""
    c = duckdb.connect(":memory:")
    c.execute("""
        CREATE TABLE player_match_enrichment (
            match_id VARCHAR NOT NULL PRIMARY KEY,
            performance_score FLOAT,
            session_id VARCHAR,
            session_label VARCHAR,
            is_with_friends BOOLEAN,
            teammates_signature VARCHAR,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    yield c
    c.close()


@dataclass
class FakeMatchRow:
    """Simule un MatchStatsRow pour _update_accuracy_shots."""

    accuracy: float | None = None
    shots_fired: int | None = None
    shots_hit: int | None = None


# ─────────────────────────────────────────────────────────────────────────────
# Tests _empty_result
# ─────────────────────────────────────────────────────────────────────────────


class TestEmptyResult:
    def test_has_expected_keys(self):
        from scripts.backfill.orchestrator import _empty_result

        result = _empty_result()
        assert isinstance(result, dict)
        expected_keys = [
            "matches_checked",
            "matches_missing_data",
            "medals_inserted",
            "events_inserted",
            "skill_inserted",
            "personal_scores_inserted",
            "performance_scores_inserted",
            "aliases_inserted",
            "accuracy_updated",
            "shots_updated",
            "enemy_mmr_updated",
            "assets_updated",
            "participants_inserted",
            "participants_scores_updated",
            "participants_kda_updated",
            "participants_shots_updated",
            "participants_damage_updated",
            "participants_avg_life_updated",
            "killer_victim_pairs_inserted",
            "end_time_updated",
            "sessions_updated",
            "citations_computed",
            "participants_enriched",
        ]
        for key in expected_keys:
            assert key in result
            assert result[key] == 0

    def test_returns_new_instance(self):
        from scripts.backfill.orchestrator import _empty_result

        r1 = _empty_result()
        r2 = _empty_result()
        r1["medals_inserted"] = 99
        assert r2["medals_inserted"] == 0


# ─────────────────────────────────────────────────────────────────────────────
# Tests _update_accuracy_shots
# ─────────────────────────────────────────────────────────────────────────────


class TestUpdateAccuracyShots:
    """Tests _update_accuracy_shots — écrit dans match_participants (V5)."""

    @pytest.fixture()
    def shared_conn(self):
        """Connexion DuckDB in-memory avec match_participants (V5)."""
        c = duckdb.connect(":memory:")
        c.execute("""
            CREATE TABLE match_participants (
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
                accuracy FLOAT,
                shots_fired INTEGER,
                shots_hit INTEGER,
                PRIMARY KEY (match_id, xuid)
            )
        """)
        yield c
        c.close()

    def test_update_accuracy_when_null(self, shared_conn):
        from scripts.backfill.orchestrator import _update_accuracy_shots

        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid, accuracy) VALUES (?, ?, NULL)",
            ["m1", "xuid1"],
        )
        row = FakeMatchRow(accuracy=0.55)
        a, s = _update_accuracy_shots(
            shared_conn,
            row,
            "m1",
            "xuid1",
            accuracy=True,
            shots=False,
            force_accuracy=False,
            force_shots=False,
        )
        assert a == 1
        assert s == 0
        result = shared_conn.execute(
            "SELECT accuracy FROM match_participants WHERE match_id='m1' AND xuid='xuid1'"
        ).fetchone()[0]
        assert result == pytest.approx(0.55)

    def test_skip_accuracy_when_already_set(self, shared_conn):
        from scripts.backfill.orchestrator import _update_accuracy_shots

        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid, accuracy) VALUES (?, ?, ?)",
            ["m1", "xuid1", 0.40],
        )
        row = FakeMatchRow(accuracy=0.55)
        a, _ = _update_accuracy_shots(
            shared_conn,
            row,
            "m1",
            "xuid1",
            accuracy=True,
            shots=False,
            force_accuracy=False,
            force_shots=False,
        )
        assert a == 0  # not updated

    def test_force_accuracy(self, shared_conn):
        from scripts.backfill.orchestrator import _update_accuracy_shots

        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid, accuracy) VALUES (?, ?, ?)",
            ["m1", "xuid1", 0.40],
        )
        row = FakeMatchRow(accuracy=0.55)
        a, _ = _update_accuracy_shots(
            shared_conn,
            row,
            "m1",
            "xuid1",
            accuracy=True,
            shots=False,
            force_accuracy=True,
            force_shots=False,
        )
        assert a == 1

    def test_update_shots_when_null(self, shared_conn):
        from scripts.backfill.orchestrator import _update_accuracy_shots

        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid, shots_fired, shots_hit) "
            "VALUES (?, ?, NULL, NULL)",
            ["m1", "xuid1"],
        )
        row = FakeMatchRow(shots_fired=200, shots_hit=110)
        _, s = _update_accuracy_shots(
            shared_conn,
            row,
            "m1",
            "xuid1",
            accuracy=False,
            shots=True,
            force_accuracy=False,
            force_shots=False,
        )
        assert s == 1

    def test_force_shots(self, shared_conn):
        from scripts.backfill.orchestrator import _update_accuracy_shots

        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid, shots_fired, shots_hit) "
            "VALUES (?, ?, ?, ?)",
            ["m1", "xuid1", 100, 50],
        )
        row = FakeMatchRow(shots_fired=200, shots_hit=110)
        _, s = _update_accuracy_shots(
            shared_conn,
            row,
            "m1",
            "xuid1",
            accuracy=False,
            shots=True,
            force_accuracy=False,
            force_shots=True,
        )
        assert s == 1

    def test_no_accuracy_value(self, shared_conn):
        from scripts.backfill.orchestrator import _update_accuracy_shots

        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid) VALUES (?, ?)",
            ["m1", "xuid1"],
        )
        row = FakeMatchRow(accuracy=None)
        a, _ = _update_accuracy_shots(
            shared_conn,
            row,
            "m1",
            "xuid1",
            accuracy=True,
            shots=False,
            force_accuracy=False,
            force_shots=False,
        )
        assert a == 0


# ─────────────────────────────────────────────────────────────────────────────
# Tests _backfill_local_only
# ─────────────────────────────────────────────────────────────────────────────


class TestBackfillLocalOnly:
    def test_no_options(self, conn):
        from pathlib import Path

        from scripts.backfill.orchestrator import _backfill_local_only

        result = _backfill_local_only(
            conn,
            Path("/fake"),
            "xuid1",
            killer_victim=False,
            end_time=False,
            sessions=False,
            citations=False,
        )
        assert isinstance(result, dict)
        assert result["killer_victim_pairs_inserted"] == 0

    def test_end_time_only(self, conn):
        from pathlib import Path

        from scripts.backfill.orchestrator import _backfill_local_only

        # Créer une shared_conn in-memory avec match_registry
        shared = duckdb.connect(":memory:")
        shared.execute("""
            CREATE TABLE match_registry (
                match_id VARCHAR NOT NULL PRIMARY KEY,
                start_time TIMESTAMP,
                end_time TIMESTAMP,
                time_played_seconds INTEGER,
                backfill_completed INTEGER DEFAULT 0
            )
        """)
        t = datetime(2024, 1, 15, 10, 0, 0, tzinfo=timezone.utc)
        shared.execute(
            "INSERT INTO match_registry (match_id, start_time, time_played_seconds, end_time) "
            "VALUES (?, ?, ?, NULL)",
            ["m1", t, 600],
        )
        with patch("scripts.backfill.orchestrator._get_shared_connection", return_value=shared):
            result = _backfill_local_only(
                conn,
                Path("/fake"),
                "xuid1",
                killer_victim=False,
                end_time=True,
                sessions=False,
                citations=False,
            )
        assert result["end_time_updated"] == 1
        shared.close()

    def test_killer_victim(self, conn):
        from pathlib import Path

        from scripts.backfill.orchestrator import _backfill_local_only

        # No events → 0 pairs
        # Mock _get_shared_connection pour éviter d'accéder à la vraie shared DB
        with patch("scripts.backfill.orchestrator._get_shared_connection", return_value=None):
            result = _backfill_local_only(
                conn,
                Path("/fake"),
                "xuid1",
                killer_victim=True,
                end_time=False,
                sessions=False,
                citations=False,
            )
        assert result["killer_victim_pairs_inserted"] == 0


# ─────────────────────────────────────────────────────────────────────────────
# Tests _resolve_xuid_fallback
# ─────────────────────────────────────────────────────────────────────────────


class TestResolveXuidFallback:
    def test_from_highlight_events(self, tmp_path):
        from scripts.backfill.orchestrator import _resolve_xuid_fallback

        db_path = tmp_path / "test.duckdb"
        c = duckdb.connect(str(db_path))
        c.execute("""
            CREATE TABLE highlight_events (
                match_id VARCHAR,
                event_type VARCHAR,
                time_ms INTEGER,
                xuid VARCHAR,
                gamertag VARCHAR
            )
        """)
        c.execute(
            "INSERT INTO highlight_events (match_id, event_type, xuid, gamertag) "
            "VALUES (?, ?, ?, ?)",
            ["m1", "Kill", "1234567890", "TestPlayer"],
        )
        c.close()

        with patch(
            "src.data.repositories.factory.load_db_profiles", side_effect=Exception("no profiles")
        ):
            result = _resolve_xuid_fallback(db_path, "TestPlayer")
        assert result == "1234567890"

    def test_not_found(self, tmp_path):
        from scripts.backfill.orchestrator import _resolve_xuid_fallback

        db_path = tmp_path / "test.duckdb"
        c = duckdb.connect(str(db_path))
        c.execute("""
            CREATE TABLE highlight_events (
                match_id VARCHAR,
                event_type VARCHAR,
                time_ms INTEGER,
                xuid VARCHAR,
                gamertag VARCHAR
            )
        """)
        c.close()

        with patch("src.data.repositories.factory.load_db_profiles", side_effect=Exception("no")):
            result = _resolve_xuid_fallback(db_path, "UnknownPlayer")
        assert result is None

    def test_db_without_events_table(self, tmp_path):
        from scripts.backfill.orchestrator import _resolve_xuid_fallback

        db_path = tmp_path / "test.duckdb"
        c = duckdb.connect(str(db_path))
        c.close()

        with patch("src.data.repositories.factory.load_db_profiles", side_effect=Exception("no")):
            result = _resolve_xuid_fallback(db_path, "TestPlayer")
        assert result is None

    def test_from_db_profiles(self, tmp_path):
        from scripts.backfill.orchestrator import _resolve_xuid_fallback

        db_path = tmp_path / "test.duckdb"
        c = duckdb.connect(str(db_path))
        c.close()

        profiles = {
            "profiles": {
                "TestPlayer": {
                    "xuid": "9876543210",
                    "waypoint_player": "TestPlayer",
                }
            }
        }
        with patch("src.data.repositories.factory.load_db_profiles", return_value=profiles):
            result = _resolve_xuid_fallback(db_path, "TestPlayer")
        assert result == "9876543210"
