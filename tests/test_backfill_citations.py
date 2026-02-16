"""Tests pour le backfill des citations (scripts/backfill/strategies.py)."""

from __future__ import annotations

from pathlib import Path

import duckdb
import pytest

from scripts.backfill.strategies import backfill_citations

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _create_metadata_db(path: Path) -> None:
    """Crée une metadata.duckdb avec citation_mappings de test."""
    conn = duckdb.connect(str(path))
    conn.execute("""
        CREATE TABLE citation_mappings (
            citation_name_norm TEXT PRIMARY KEY,
            citation_name_display TEXT NOT NULL,
            mapping_type TEXT NOT NULL,
            medal_id BIGINT,
            medal_ids TEXT,
            stat_name TEXT,
            award_name TEXT,
            award_category TEXT,
            custom_function TEXT,
            confidence TEXT,
            notes TEXT,
            enabled BOOLEAN DEFAULT TRUE,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    conn.executemany(
        "INSERT INTO citation_mappings "
        "(citation_name_norm, citation_name_display, mapping_type, "
        "medal_id, stat_name, award_name, custom_function, confidence, notes) "
        "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
        [
            ("pilote", "Pilote", "medal", 3169118333, None, None, None, "high", "test"),
            ("assistant", "Assistant", "stat", None, "assists", None, None, "high", "test"),
            (
                "defenseur du drapeau",
                "Défenseur du drapeau",
                "award",
                None,
                None,
                "Flag Defense",
                None,
                "high",
                "test",
            ),
        ],
    )
    conn.close()


def _create_shared_db(path: Path) -> None:
    """Crée shared_matches.duckdb avec match_registry, match_participants, medals_earned."""
    conn = duckdb.connect(str(path))
    conn.execute("""
        CREATE TABLE match_registry (
            match_id TEXT PRIMARY KEY,
            start_time TIMESTAMP,
            end_time TIMESTAMP,
            map_name TEXT,
            playlist TEXT,
            game_variant TEXT,
            match_start_date TEXT,
            playlist_name TEXT,
            game_variant_name TEXT,
            pair_name TEXT,
            backfill_completed INTEGER DEFAULT 0
        )
    """)
    conn.execute("""
        CREATE TABLE match_participants (
            match_id TEXT NOT NULL,
            xuid TEXT NOT NULL,
            gamertag TEXT,
            kills SMALLINT DEFAULT 0,
            deaths SMALLINT DEFAULT 0,
            assists SMALLINT DEFAULT 0,
            headshot_kills SMALLINT DEFAULT 0,
            outcome SMALLINT DEFAULT 0,
            kda FLOAT,
            accuracy FLOAT,
            time_played_seconds INTEGER,
            avg_life_seconds FLOAT,
            personal_score INTEGER,
            damage_dealt FLOAT,
            rank SMALLINT,
            team_mmr FLOAT,
            shots_fired INTEGER,
            shots_hit INTEGER,
            PRIMARY KEY (match_id, xuid)
        )
    """)
    conn.execute("""
        CREATE TABLE medals_earned (
            match_id TEXT NOT NULL,
            xuid TEXT NOT NULL,
            medal_name_id BIGINT NOT NULL,
            count INTEGER NOT NULL,
            PRIMARY KEY (match_id, xuid, medal_name_id)
        )
    """)

    # 3 matchs de test
    conn.execute(
        "INSERT INTO match_registry (match_id, start_time, end_time, map_name, playlist, "
        "game_variant, match_start_date, playlist_name, game_variant_name) VALUES "
        "('m1', '2026-01-01 12:00:00', '2026-01-01 12:10:00', 'LiveFire', "
        "'Ranked Slayer', 'Slayer', '2026-01-01', 'Ranked Slayer', 'Slayer'),"
        "('m2', '2026-01-02 14:00:00', '2026-01-02 14:10:00', 'Streets', "
        "'Quick Play', 'CTF', '2026-01-02', 'Quick Play', 'CTF'),"
        "('m3', '2026-01-03 16:00:00', '2026-01-03 16:10:00', 'Recharge', "
        "'Quick Play', 'Oddball', '2026-01-03', 'Quick Play', 'Oddball')"
    )
    conn.execute(
        "INSERT INTO match_participants (match_id, xuid, gamertag, kills, deaths, assists, "
        "headshot_kills, outcome) VALUES "
        "('m1', '12345', 'TestPlayer', 15, 5, 8, 3, 2),"
        "('m2', '12345', 'TestPlayer', 10, 8, 12, 2, 2),"
        "('m3', '12345', 'TestPlayer', 3, 10, 1, 0, 0)"
    )
    conn.execute(
        "INSERT INTO medals_earned VALUES "
        "('m1', '12345', 3169118333, 2), ('m1', '12345', 221693153, 1),"
        "('m2', '12345', 3169118333, 1)"
    )
    conn.close()


def _create_player_db(path: Path) -> None:
    """Crée une DB joueur V5 avec match_citations et personal_score_awards."""
    conn = duckdb.connect(str(path))

    conn.execute("""
        CREATE TABLE match_citations (
            match_id TEXT NOT NULL,
            citation_name_norm TEXT NOT NULL,
            value INTEGER NOT NULL,
            PRIMARY KEY (match_id, citation_name_norm)
        )
    """)
    conn.execute(
        "CREATE INDEX IF NOT EXISTS idx_match_citations_name "
        "ON match_citations(citation_name_norm)"
    )

    conn.execute("""
        CREATE TABLE personal_score_awards (
            match_id TEXT NOT NULL,
            award_name TEXT NOT NULL,
            award_category TEXT,
            award_count INTEGER DEFAULT 0,
            award_score INTEGER DEFAULT 0
        )
    """)
    conn.execute(
        "INSERT INTO personal_score_awards VALUES ('m1', 'Flag Defense', 'objective', 3, 150)"
    )
    conn.close()


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture
def backfill_dir(tmp_path: Path) -> Path:
    """Crée la structure data/warehouse + data/players/TestPlayer/."""
    warehouse = tmp_path / "data" / "warehouse"
    warehouse.mkdir(parents=True)
    player_dir = tmp_path / "data" / "players" / "TestPlayer"
    player_dir.mkdir(parents=True)

    _create_metadata_db(warehouse / "metadata.duckdb")
    _create_shared_db(warehouse / "shared_matches.duckdb")
    _create_player_db(player_dir / "stats.duckdb")

    return tmp_path


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


class TestBackfillCitations:
    """Tests pour la stratégie backfill_citations."""

    def test_backfill_single_match(self, backfill_dir: Path) -> None:
        """Backfill citations insère des données."""
        db_path = backfill_dir / "data" / "players" / "TestPlayer" / "stats.duckdb"
        conn = duckdb.connect(str(db_path))

        n = backfill_citations(conn, db_path, "12345")
        conn.close()

        assert n > 0

        # Vérifier les données insérées
        read = duckdb.connect(str(db_path), read_only=True)
        rows = read.execute("SELECT COUNT(*) FROM match_citations").fetchone()[0]
        read.close()
        assert rows > 0

    def test_backfill_skips_existing(self, backfill_dir: Path) -> None:
        """Sans force, ne recalcule pas les matchs déjà traités."""
        db_path = backfill_dir / "data" / "players" / "TestPlayer" / "stats.duckdb"
        conn = duckdb.connect(str(db_path))

        # 1er backfill
        n1 = backfill_citations(conn, db_path, "12345")
        assert n1 > 0

        # 2ème backfill (devrait skip)
        n2 = backfill_citations(conn, db_path, "12345")
        assert n2 == 0

        conn.close()

    def test_backfill_force_recalculates(self, backfill_dir: Path) -> None:
        """Avec force, recalcule pour tous les matchs."""
        db_path = backfill_dir / "data" / "players" / "TestPlayer" / "stats.duckdb"
        conn = duckdb.connect(str(db_path))

        # 1er backfill
        n1 = backfill_citations(conn, db_path, "12345")
        assert n1 > 0

        # 2ème backfill avec force
        n2 = backfill_citations(conn, db_path, "12345", force=True)
        assert n2 > 0

        conn.close()

    def test_backfill_correct_values(self, backfill_dir: Path) -> None:
        """Vérifie que les valeurs calculées sont correctes."""
        db_path = backfill_dir / "data" / "players" / "TestPlayer" / "stats.duckdb"
        conn = duckdb.connect(str(db_path))

        backfill_citations(conn, db_path, "12345")
        conn.close()

        read = duckdb.connect(str(db_path), read_only=True)

        # m1 a pilote=2 (medal 3169118333), assistant=8, defenseur du drapeau=3
        m1_pilote = read.execute(
            "SELECT value FROM match_citations WHERE match_id = 'm1' AND citation_name_norm = 'pilote'"
        ).fetchone()
        assert m1_pilote is not None
        assert m1_pilote[0] == 2

        m1_assistant = read.execute(
            "SELECT value FROM match_citations WHERE match_id = 'm1' AND citation_name_norm = 'assistant'"
        ).fetchone()
        assert m1_assistant is not None
        assert m1_assistant[0] == 8

        m1_defense = read.execute(
            "SELECT value FROM match_citations WHERE match_id = 'm1' AND citation_name_norm = 'defenseur du drapeau'"
        ).fetchone()
        assert m1_defense is not None
        assert m1_defense[0] == 3

        read.close()
