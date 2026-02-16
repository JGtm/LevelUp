"""Tests pour scripts/backfill/strategies.py (V5).

Couvre : backfill_end_time, backfill_killer_victim_pairs, compute_performance_score_for_match.
Utilise DuckDB :memory: avec schéma V5 (shared_matches).
"""

from __future__ import annotations

from datetime import datetime, timedelta, timezone

import duckdb
import pytest

# ─────────────────────────────────────────────────────────────────────────────
# Fixtures
# ─────────────────────────────────────────────────────────────────────────────


@pytest.fixture()
def shared_conn():
    """Connexion DuckDB in-memory simulant shared_matches.duckdb (match_registry + match_participants)."""
    c = duckdb.connect(":memory:")
    c.execute("""
        CREATE TABLE match_registry (
            match_id VARCHAR NOT NULL PRIMARY KEY,
            start_time TIMESTAMP,
            end_time TIMESTAMP,
            time_played_seconds INTEGER,
            backfill_completed INTEGER DEFAULT 0
        )
    """)
    c.execute("""
        CREATE TABLE match_participants (
            match_id VARCHAR NOT NULL,
            xuid VARCHAR NOT NULL,
            kills SMALLINT DEFAULT 0,
            deaths SMALLINT DEFAULT 0,
            assists SMALLINT DEFAULT 0,
            kda FLOAT,
            accuracy FLOAT,
            avg_life_seconds FLOAT,
            personal_score INTEGER,
            damage_dealt FLOAT,
            rank SMALLINT,
            team_mmr FLOAT,
            time_played_seconds INTEGER,
            PRIMARY KEY (match_id, xuid)
        )
    """)
    yield c
    c.close()


@pytest.fixture()
def player_conn():
    """Connexion DuckDB in-memory simulant la DB joueur (player_match_enrichment)."""
    c = duckdb.connect(":memory:")
    c.execute("""
        CREATE TABLE player_match_enrichment (
            match_id VARCHAR PRIMARY KEY,
            performance_score FLOAT,
            session_id VARCHAR,
            session_label VARCHAR,
            is_with_friends BOOLEAN,
            teammates_signature VARCHAR,
            known_teammates_count SMALLINT,
            friends_xuids VARCHAR,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    yield c
    c.close()


@pytest.fixture()
def conn_with_events():
    """Connexion DuckDB in-memory avec highlight_events + killer_victim_pairs."""
    c = duckdb.connect(":memory:")
    c.execute("""
        CREATE TABLE highlight_events (
            id INTEGER DEFAULT 0,
            match_id VARCHAR NOT NULL,
            event_type VARCHAR,
            time_ms INTEGER,
            xuid VARCHAR,
            gamertag VARCHAR,
            type_hint INTEGER,
            raw_json VARCHAR
        )
    """)
    yield c
    c.close()


# ─────────────────────────────────────────────────────────────────────────────
# Tests backfill_end_time
# ─────────────────────────────────────────────────────────────────────────────


class TestBackfillEndTime:
    def test_updates_null_end_time(self, shared_conn):
        from scripts.backfill.strategies import backfill_end_time

        t = datetime(2024, 1, 15, 10, 0, 0, tzinfo=timezone.utc)
        shared_conn.execute(
            "INSERT INTO match_registry (match_id, start_time, time_played_seconds, end_time) "
            "VALUES (?, ?, ?, NULL)",
            ["m1", t, 600],
        )
        n = backfill_end_time(None, shared_conn=shared_conn)
        assert n == 1
        result = shared_conn.execute(
            "SELECT end_time FROM match_registry WHERE match_id='m1'"
        ).fetchone()
        assert result[0] is not None

    def test_skips_already_set(self, shared_conn):
        from scripts.backfill.strategies import backfill_end_time

        t = datetime(2024, 1, 15, 10, 0, 0, tzinfo=timezone.utc)
        end = t + timedelta(seconds=600)
        shared_conn.execute(
            "INSERT INTO match_registry (match_id, start_time, time_played_seconds, end_time) "
            "VALUES (?, ?, ?, ?)",
            ["m1", t, 600, end],
        )
        n = backfill_end_time(None, shared_conn=shared_conn)
        assert n == 0

    def test_force_recalculates(self, shared_conn):
        from scripts.backfill.strategies import backfill_end_time

        t = datetime(2024, 1, 15, 10, 0, 0, tzinfo=timezone.utc)
        end_old = t + timedelta(seconds=300)  # wrong
        shared_conn.execute(
            "INSERT INTO match_registry (match_id, start_time, time_played_seconds, end_time) "
            "VALUES (?, ?, ?, ?)",
            ["m1", t, 600, end_old],
        )
        n = backfill_end_time(None, force=True, shared_conn=shared_conn)
        assert n == 1

    def test_no_matches(self, shared_conn):
        from scripts.backfill.strategies import backfill_end_time

        assert backfill_end_time(None, shared_conn=shared_conn) == 0

    def test_null_start_time_skipped(self, shared_conn):
        from scripts.backfill.strategies import backfill_end_time

        shared_conn.execute(
            "INSERT INTO match_registry (match_id, start_time, time_played_seconds, end_time) "
            "VALUES (?, NULL, ?, NULL)",
            ["m1", 600],
        )
        n = backfill_end_time(None, shared_conn=shared_conn)
        assert n == 0


# ─────────────────────────────────────────────────────────────────────────────
# Tests backfill_killer_victim_pairs
# ─────────────────────────────────────────────────────────────────────────────


class TestBackfillKillerVictimPairs:
    def test_no_events(self, conn_with_events):
        from scripts.backfill.strategies import backfill_killer_victim_pairs

        n = backfill_killer_victim_pairs(conn_with_events, "xuid1")
        assert n == 0

    def test_creates_table(self, conn_with_events):
        """La table killer_victim_pairs est créée si absente."""
        from scripts.backfill.strategies import backfill_killer_victim_pairs

        conn = conn_with_events
        backfill_killer_victim_pairs(conn, "xuid1")
        # Table should exist now
        result = conn.execute(
            "SELECT COUNT(*) FROM information_schema.tables "
            "WHERE table_name = 'killer_victim_pairs'"
        ).fetchone()[0]
        assert result == 1

    def test_with_kill_death_events(self, conn_with_events):
        """Insert matching kill/death events → paires created."""
        from scripts.backfill.strategies import backfill_killer_victim_pairs

        conn = conn_with_events
        # Insert kill and death events at same time
        conn.execute(
            "INSERT INTO highlight_events (match_id, event_type, time_ms, xuid, gamertag) "
            "VALUES (?, ?, ?, ?, ?)",
            ["m1", "Kill", 5000, "killer1", "KillerGT"],
        )
        conn.execute(
            "INSERT INTO highlight_events (match_id, event_type, time_ms, xuid, gamertag) "
            "VALUES (?, ?, ?, ?, ?)",
            ["m1", "Death", 5000, "victim1", "VictimGT"],
        )
        n = backfill_killer_victim_pairs(conn, "xuid1")
        assert n >= 1

    def test_force_drops_table(self, conn_with_events):
        """Mode force recrée la table."""
        from scripts.backfill.strategies import backfill_killer_victim_pairs

        conn = conn_with_events
        # First call creates table
        backfill_killer_victim_pairs(conn, "xuid1")
        # Force drops and recreates
        backfill_killer_victim_pairs(conn, "xuid1", force=True)
        result = conn.execute(
            "SELECT COUNT(*) FROM information_schema.tables "
            "WHERE table_name = 'killer_victim_pairs'"
        ).fetchone()[0]
        assert result == 1

    def test_incremental_skips_existing(self, conn_with_events):
        """Mode incrémental ne retraite pas les matchs déjà traités."""
        from scripts.backfill.strategies import backfill_killer_victim_pairs

        conn = conn_with_events
        conn.execute(
            "INSERT INTO highlight_events (match_id, event_type, time_ms, xuid, gamertag) "
            "VALUES (?, ?, ?, ?, ?)",
            ["m1", "Kill", 5000, "k1", "K1"],
        )
        conn.execute(
            "INSERT INTO highlight_events (match_id, event_type, time_ms, xuid, gamertag) "
            "VALUES (?, ?, ?, ?, ?)",
            ["m1", "Death", 5000, "v1", "V1"],
        )
        n1 = backfill_killer_victim_pairs(conn, "xuid1")
        assert n1 == 1
        # Second call should find no new matches
        n2 = backfill_killer_victim_pairs(conn, "xuid1")
        assert n2 == 0

    def test_only_kills_no_deaths(self, conn_with_events):
        """Match with only kills (no deaths) → skipped."""
        from scripts.backfill.strategies import backfill_killer_victim_pairs

        conn = conn_with_events
        conn.execute(
            "INSERT INTO highlight_events (match_id, event_type, time_ms, xuid, gamertag) "
            "VALUES (?, ?, ?, ?, ?)",
            ["m1", "Kill", 5000, "k1", "K1"],
        )
        n = backfill_killer_victim_pairs(conn, "xuid1")
        assert n == 0


# ─────────────────────────────────────────────────────────────────────────────
# Tests compute_performance_score_for_match
# ─────────────────────────────────────────────────────────────────────────────


class TestComputePerformanceScoreForMatch:
    def test_score_already_exists(self, shared_conn, player_conn):
        """Retourne False si le score existe déjà dans player_match_enrichment."""
        from scripts.backfill.strategies import compute_performance_score_for_match

        t = datetime(2024, 1, 15, 10, 0, 0, tzinfo=timezone.utc)
        shared_conn.execute(
            "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
            ["m1", t],
        )
        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid, kills, deaths, assists, time_played_seconds) "
            "VALUES (?, ?, ?, ?, ?, ?)",
            ["m1", "xuid1", 10, 5, 3, 600],
        )
        player_conn.execute(
            "INSERT INTO player_match_enrichment (match_id, performance_score) VALUES (?, ?)",
            ["m1", 75.0],
        )
        result = compute_performance_score_for_match(
            player_conn, "m1", shared_conn=shared_conn, xuid="xuid1"
        )
        assert result is False

    def test_not_enough_history(self, shared_conn, player_conn):
        """Retourne False si pas assez de matchs historiques."""
        from scripts.backfill.strategies import compute_performance_score_for_match

        t = datetime(2024, 1, 15, 10, 0, 0, tzinfo=timezone.utc)
        shared_conn.execute(
            "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
            ["m1", t],
        )
        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid, kills, deaths, assists, time_played_seconds) "
            "VALUES (?, ?, ?, ?, ?, ?)",
            ["m1", "xuid1", 10, 5, 3, 600],
        )
        result = compute_performance_score_for_match(
            player_conn, "m1", shared_conn=shared_conn, xuid="xuid1"
        )
        assert result is False

    def test_match_not_found(self, shared_conn, player_conn):
        """Retourne False si le match n'existe pas dans shared."""
        from scripts.backfill.strategies import compute_performance_score_for_match

        result = compute_performance_score_for_match(
            player_conn, "nonexistent", shared_conn=shared_conn, xuid="xuid1"
        )
        assert result is False

    def test_null_start_time(self, shared_conn, player_conn):
        """Retourne False si start_time est NULL dans match_registry."""
        from scripts.backfill.strategies import compute_performance_score_for_match

        shared_conn.execute(
            "INSERT INTO match_registry (match_id, start_time) VALUES (?, NULL)",
            ["m1"],
        )
        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid, kills, deaths, assists, time_played_seconds) "
            "VALUES (?, ?, ?, ?, ?, ?)",
            ["m1", "xuid1", 10, 5, 3, 600],
        )
        result = compute_performance_score_for_match(
            player_conn, "m1", shared_conn=shared_conn, xuid="xuid1"
        )
        assert result is False
