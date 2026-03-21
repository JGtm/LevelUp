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
    """Connexion DuckDB in-memory simulant shared_matches_v2.duckdb (match_registry + match_participants)."""
    c = duckdb.connect(":memory:")
    c.execute("""
        CREATE TABLE match_registry (
            match_id VARCHAR NOT NULL PRIMARY KEY,
            start_time TIMESTAMP,
            end_time TIMESTAMP,
            duration_seconds INTEGER,
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
            "INSERT INTO match_registry (match_id, start_time, duration_seconds, end_time) "
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
            "INSERT INTO match_registry (match_id, start_time, duration_seconds, end_time) "
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
            "INSERT INTO match_registry (match_id, start_time, duration_seconds, end_time) "
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
            "INSERT INTO match_registry (match_id, start_time, duration_seconds, end_time) "
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
            "INSERT INTO highlight_events (match_id, event_type, time_ms, xuid) "
            "VALUES (?, ?, ?, ?)",
            ["m1", "Kill", 5000, "killer1"],
        )
        conn.execute(
            "INSERT INTO highlight_events (match_id, event_type, time_ms, xuid) "
            "VALUES (?, ?, ?, ?)",
            ["m1", "Death", 5000, "victim1"],
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
            "INSERT INTO highlight_events (match_id, event_type, time_ms, xuid) "
            "VALUES (?, ?, ?, ?)",
            ["m1", "Kill", 5000, "k1"],
        )
        conn.execute(
            "INSERT INTO highlight_events (match_id, event_type, time_ms, xuid) "
            "VALUES (?, ?, ?, ?)",
            ["m1", "Death", 5000, "v1"],
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
            "INSERT INTO highlight_events (match_id, event_type, time_ms, xuid) "
            "VALUES (?, ?, ?, ?)",
            ["m1", "Kill", 5000, "k1"],
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


# ─────────────────────────────────────────────────────────────────────────────
# Tests backfill_avenger_medal
# ─────────────────────────────────────────────────────────────────────────────


@pytest.fixture()
def avenger_conn():
    """DB in-memory avec killer_victim_pairs + medals_earned (schéma shared_matches)."""
    c = duckdb.connect(":memory:")
    c.execute("""
        CREATE TABLE killer_victim_pairs (
            match_id VARCHAR NOT NULL,
            killer_xuid VARCHAR NOT NULL,
            victim_xuid VARCHAR NOT NULL,
            time_ms INTEGER,
            kill_count INTEGER DEFAULT 1,
            killer_gamertag VARCHAR,
            victim_gamertag VARCHAR,
            is_validated BOOLEAN DEFAULT FALSE,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    c.execute("""
        CREATE TABLE medals_earned (
            match_id VARCHAR NOT NULL,
            xuid VARCHAR NOT NULL,
            medal_name_id BIGINT NOT NULL,
            count SMALLINT NOT NULL,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            PRIMARY KEY (match_id, xuid, medal_name_id)
        )
    """)
    yield c
    c.close()


def _insert_kill(conn, match_id: str, killer: str, victim: str, time_ms: int) -> None:
    conn.execute(
        "INSERT INTO killer_victim_pairs (match_id, killer_xuid, victim_xuid, time_ms) "
        "VALUES (?, ?, ?, ?)",
        [match_id, killer, victim, time_ms],
    )


class TestBackfillAvengerMedal:
    def test_empty_table_returns_zero(self, avenger_conn):
        """Table vide → 0 médaille insérée."""
        from scripts.backfill.strategies import backfill_avenger_medal

        n = backfill_avenger_medal(avenger_conn)
        assert n == 0
        assert avenger_conn.execute("SELECT COUNT(*) FROM medals_earned").fetchone()[0] == 0

    def test_basic_revenge_kill(self, avenger_conn):
        """Scénario simple : B tue A (t=100), A tue B (t=200) → A obtient Vengeur."""
        from scripts.backfill.strategies import AVENGER_MEDAL_ID, backfill_avenger_medal

        _insert_kill(avenger_conn, "m1", "B", "A", 100)  # B kills A
        _insert_kill(avenger_conn, "m1", "A", "B", 200)  # A kills B → avenger

        n = backfill_avenger_medal(avenger_conn)
        assert n == 1

        row = avenger_conn.execute(
            "SELECT xuid, count FROM medals_earned WHERE match_id='m1' AND medal_name_id=?",
            [AVENGER_MEDAL_ID],
        ).fetchone()
        assert row is not None
        assert row[0] == "A"
        assert row[1] == 1

    def test_no_revenge_different_killer(self, avenger_conn):
        """A est tué par B, puis C tue A, puis A tue B → pas de vengeur (last_killer=C, pas B)."""
        from scripts.backfill.strategies import backfill_avenger_medal

        _insert_kill(avenger_conn, "m1", "B", "A", 100)  # B kills A
        _insert_kill(avenger_conn, "m1", "C", "A", 150)  # C kills A (écrase last_killer)
        _insert_kill(
            avenger_conn, "m1", "A", "B", 200
        )  # A kills B → pas vengeur (dernier tueur = C)

        n = backfill_avenger_medal(avenger_conn)
        assert n == 0

    def test_last_death_determines_killer(self, avenger_conn):
        """Seul le dernier tueur compte : A tué par B (t=100), A tué par C (t=150), A tue C (t=200) → vengeur."""
        from scripts.backfill.strategies import AVENGER_MEDAL_ID, backfill_avenger_medal

        _insert_kill(avenger_conn, "m1", "B", "A", 100)
        _insert_kill(avenger_conn, "m1", "C", "A", 150)  # dernière mort de A → par C
        _insert_kill(avenger_conn, "m1", "A", "C", 200)  # A tue C → vengeur

        n = backfill_avenger_medal(avenger_conn)
        assert n == 1
        row = avenger_conn.execute(
            "SELECT xuid FROM medals_earned WHERE medal_name_id=?",
            [AVENGER_MEDAL_ID],
        ).fetchone()
        assert row[0] == "A"

    def test_multiple_avengers_in_match(self, avenger_conn):
        """Deux vengeurs distincts dans le même match → 1 paire, count=2."""
        from scripts.backfill.strategies import AVENGER_MEDAL_ID, backfill_avenger_medal

        # Séquence 1 : B tue A (t=100), A tue B (t=200) → avenger
        _insert_kill(avenger_conn, "m1", "B", "A", 100)
        _insert_kill(avenger_conn, "m1", "A", "B", 200)
        # Séquence 2 : C tue A (t=300), A tue C (t=400) → avenger
        _insert_kill(avenger_conn, "m1", "C", "A", 300)
        _insert_kill(avenger_conn, "m1", "A", "C", 400)

        n = backfill_avenger_medal(avenger_conn)
        assert n == 1  # 1 paire (match_id, xuid)

        row = avenger_conn.execute(
            "SELECT count FROM medals_earned WHERE match_id='m1' AND xuid='A' AND medal_name_id=?",
            [AVENGER_MEDAL_ID],
        ).fetchone()
        assert row is not None
        assert row[0] == 2

    def test_multiple_players_avenger(self, avenger_conn):
        """A et B obtiennent tous les deux la médaille dans le même match."""
        from scripts.backfill.strategies import AVENGER_MEDAL_ID, backfill_avenger_medal

        _insert_kill(avenger_conn, "m1", "B", "A", 100)
        _insert_kill(avenger_conn, "m1", "A", "B", 200)  # A avenge
        _insert_kill(avenger_conn, "m1", "A", "B", 300)
        _insert_kill(avenger_conn, "m1", "B", "A", 400)  # B avenge

        n = backfill_avenger_medal(avenger_conn)
        assert n == 2  # 2 paires (m1,A) et (m1,B)

        xuids = {
            r[0]
            for r in avenger_conn.execute(
                "SELECT xuid FROM medals_earned WHERE medal_name_id=?",
                [AVENGER_MEDAL_ID],
            ).fetchall()
        }
        assert xuids == {"A", "B"}

    def test_multiple_matches(self, avenger_conn):
        """Vengeur détecté indépendamment dans deux matchs différents."""
        from scripts.backfill.strategies import AVENGER_MEDAL_ID, backfill_avenger_medal

        _insert_kill(avenger_conn, "m1", "B", "A", 100)
        _insert_kill(avenger_conn, "m1", "A", "B", 200)

        _insert_kill(avenger_conn, "m2", "C", "D", 100)
        _insert_kill(avenger_conn, "m2", "D", "C", 200)

        n = backfill_avenger_medal(avenger_conn)
        assert n == 2

        match_ids = {
            r[0]
            for r in avenger_conn.execute(
                "SELECT match_id FROM medals_earned WHERE medal_name_id=?",
                [AVENGER_MEDAL_ID],
            ).fetchall()
        }
        assert match_ids == {"m1", "m2"}

    def test_force_false_does_not_overwrite(self, avenger_conn):
        """Sans force, une médaille déjà présente n'est pas écrasée (INSERT OR IGNORE)."""
        from scripts.backfill.strategies import AVENGER_MEDAL_ID, backfill_avenger_medal

        _insert_kill(avenger_conn, "m1", "B", "A", 100)
        _insert_kill(avenger_conn, "m1", "A", "B", 200)

        # Pré-insérer une valeur différente
        avenger_conn.execute(
            "INSERT INTO medals_earned (match_id, xuid, medal_name_id, count) VALUES (?, ?, ?, ?)",
            ["m1", "A", AVENGER_MEDAL_ID, 99],
        )

        backfill_avenger_medal(avenger_conn, force=False)

        row = avenger_conn.execute(
            "SELECT count FROM medals_earned WHERE match_id='m1' AND xuid='A' AND medal_name_id=?",
            [AVENGER_MEDAL_ID],
        ).fetchone()
        assert row[0] == 99  # valeur inchangée

    def test_force_true_overwrites(self, avenger_conn):
        """Avec force=True, la médaille existante est recalculée et écrasée."""
        from scripts.backfill.strategies import AVENGER_MEDAL_ID, backfill_avenger_medal

        _insert_kill(avenger_conn, "m1", "B", "A", 100)
        _insert_kill(avenger_conn, "m1", "A", "B", 200)

        # Pré-insérer une valeur erronée
        avenger_conn.execute(
            "INSERT INTO medals_earned (match_id, xuid, medal_name_id, count) VALUES (?, ?, ?, ?)",
            ["m1", "A", AVENGER_MEDAL_ID, 99],
        )

        backfill_avenger_medal(avenger_conn, force=True)

        row = avenger_conn.execute(
            "SELECT count FROM medals_earned WHERE match_id='m1' AND xuid='A' AND medal_name_id=?",
            [AVENGER_MEDAL_ID],
        ).fetchone()
        assert row[0] == 1  # recalculé

    def test_no_avenger_without_prior_death(self, avenger_conn):
        """A tue B sans jamais avoir été tué avant → pas de vengeur."""
        from scripts.backfill.strategies import backfill_avenger_medal

        _insert_kill(avenger_conn, "m1", "A", "B", 100)  # A kills B, jamais mort avant

        n = backfill_avenger_medal(avenger_conn)
        assert n == 0

    def test_avenger_id_is_custom_range(self, avenger_conn):
        """AVENGER_MEDAL_ID est hors plage officielle Halo (> 4_285_712_605)."""
        from scripts.backfill.strategies import AVENGER_MEDAL_ID

        assert AVENGER_MEDAL_ID > 4_285_712_605

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
