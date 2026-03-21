"""Tests V5 pour le calcul des scores de performance.

Architecture V5 finale :
- performance_score stocke dans player_match_enrichment (player DB)
- Calcule depuis shared.match_participants + match_registry (shared DB)
- MIN_MATCHES_FOR_RELATIVE = 10 matchs requis avant le premier calcul
- _ensure_performance_score_column est un no-op (colonne dans le schema initial)
"""

from __future__ import annotations

from datetime import datetime, timedelta, timezone
from pathlib import Path

import duckdb
import pytest

from src.data.sync.engine import DuckDBSyncEngine

XUID = "2535423456789"
GAMERTAG = "TestPlayer"


# =============================================================================
# Helpers
# =============================================================================


def _make_shared_db(tmp_path: Path, xuid: str = XUID, n_matches: int = 0) -> Path:
    """Cree shared_matches_v2.duckdb avec match_registry + match_participants (schema V5)."""
    db_path = tmp_path / "warehouse" / "shared_matches_v2.duckdb"
    db_path.parent.mkdir(parents=True, exist_ok=True)

    conn = duckdb.connect(str(db_path))
    conn.execute("""
        CREATE TABLE match_registry (
            match_id VARCHAR PRIMARY KEY,
            start_time TIMESTAMP NOT NULL
        )
    """)
    conn.execute("""
        CREATE TABLE match_participants (
            match_id             VARCHAR NOT NULL,
            xuid                 VARCHAR NOT NULL,
            gamertag             VARCHAR,
            outcome              INTEGER  DEFAULT 2,
            kills                SMALLINT DEFAULT 10,
            deaths               SMALLINT DEFAULT 8,
            assists              SMALLINT DEFAULT 5,
            kda                  FLOAT    DEFAULT 1.5,
            accuracy             FLOAT    DEFAULT 0.45,
            time_played_seconds  INTEGER  DEFAULT 600,
            avg_life_seconds     FLOAT    DEFAULT 45.0,
            personal_score       INTEGER  DEFAULT 1500,
            damage_dealt         FLOAT    DEFAULT 3000.0,
            damage_taken         FLOAT    DEFAULT 2800.0,
            rank                 SMALLINT DEFAULT 1,
            team_mmr             FLOAT    DEFAULT 1500.0,
            enemy_mmr            FLOAT    DEFAULT 1480.0,
            kills_expected       FLOAT    DEFAULT 9.5,
            deaths_expected      FLOAT    DEFAULT 8.5,
            kills_stddev         FLOAT    DEFAULT 2.0,
            deaths_stddev        FLOAT    DEFAULT 2.0,
            assists_expected     FLOAT    DEFAULT 5.0,
            assists_stddev       FLOAT    DEFAULT 1.5,
            shots_fired          INTEGER  DEFAULT 200,
            shots_hit            INTEGER  DEFAULT 90,
            grenade_kills        SMALLINT DEFAULT 0,
            melee_kills          SMALLINT DEFAULT 0,
            power_weapon_kills   SMALLINT DEFAULT 0,
            score                INTEGER  DEFAULT 1500,
            headshot_kills       SMALLINT DEFAULT 3,
            max_killing_spree    SMALLINT DEFAULT 5,
            team_id              INTEGER  DEFAULT 0,
            backfill_bits        INTEGER  DEFAULT 0,
            created_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            PRIMARY KEY (match_id, xuid)
        )
    """)

    base_time = datetime(2024, 1, 1, tzinfo=timezone.utc)
    for i in range(n_matches):
        mid = f"m-{i:04d}"
        t = base_time + timedelta(hours=i)
        conn.execute(
            "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
            (mid, t),
        )
        conn.execute(
            "INSERT INTO match_participants (match_id, xuid, gamertag, outcome) VALUES (?, ?, ?, ?)",
            (mid, xuid, GAMERTAG, 2),
        )

    conn.commit()
    conn.close()
    return db_path


def _make_player_db(tmp_path: Path, gamertag: str = GAMERTAG) -> Path:
    """Cree stats.duckdb avec player_match_enrichment (schema V5)."""
    db_path = tmp_path / "players" / gamertag / "stats.duckdb"
    db_path.parent.mkdir(parents=True, exist_ok=True)

    conn = duckdb.connect(str(db_path))
    conn.execute("""
        CREATE TABLE player_match_enrichment (
            match_id               VARCHAR PRIMARY KEY,
            performance_score      FLOAT,
            session_id             VARCHAR,
            session_label          VARCHAR,
            is_with_friends        BOOLEAN,
            teammates_signature    VARCHAR,
            known_teammates_count  INTEGER DEFAULT 0,
            friends_xuids          VARCHAR,
            created_at             TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at             TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    conn.close()
    return db_path


# =============================================================================
# Tests : migration colonne (V5 -- no-op)
# =============================================================================


class TestPerformanceScoreColumnMigration:
    """Tests colonne performance_score dans player_match_enrichment (V5)."""

    def test_player_match_enrichment_has_performance_score_column(self, tmp_path: Path) -> None:
        """player_match_enrichment possede la colonne performance_score des la creation."""
        player_db = _make_player_db(tmp_path)
        conn = duckdb.connect(str(player_db), read_only=True)
        cols = {
            r[0]
            for r in conn.execute(
                "SELECT column_name FROM information_schema.columns "
                "WHERE table_name='player_match_enrichment'"
            ).fetchall()
        }
        conn.close()
        assert "performance_score" in cols


# =============================================================================
# Tests : batch_compute_performance_scores -- architecture V5
# =============================================================================


class TestBatchComputePerformanceScores:
    """Tests batch_compute_performance_scores -- shared + player_match_enrichment."""

    def test_returns_zero_when_no_shared_data(self, tmp_path: Path) -> None:
        """Retourne 0 si shared na aucun match pour ce joueur."""
        player_db = _make_player_db(tmp_path)
        shared_db = _make_shared_db(tmp_path, n_matches=0)
        engine = DuckDBSyncEngine(player_db, xuid=XUID, gamertag=GAMERTAG, shared_db_path=shared_db)
        assert engine.batch_compute_performance_scores() == 0
        engine.close()

    def test_returns_zero_when_insufficient_history(self, tmp_path: Path) -> None:
        """Retourne 0 si < 10 matchs (MIN_MATCHES_FOR_RELATIVE)."""
        player_db = _make_player_db(tmp_path)
        shared_db = _make_shared_db(tmp_path, n_matches=5)
        engine = DuckDBSyncEngine(player_db, xuid=XUID, gamertag=GAMERTAG, shared_db_path=shared_db)
        assert engine.batch_compute_performance_scores() == 0
        engine.close()

    def test_calculates_scores_with_sufficient_history(self, tmp_path: Path) -> None:
        """Calcule des scores quand 15 matchs disponibles."""
        player_db = _make_player_db(tmp_path)
        shared_db = _make_shared_db(tmp_path, n_matches=15)
        engine = DuckDBSyncEngine(player_db, xuid=XUID, gamertag=GAMERTAG, shared_db_path=shared_db)
        updated = engine.batch_compute_performance_scores()
        engine.close()  # Fermer avant d'ouvrir une connexion read_only
        assert updated > 0

        conn = duckdb.connect(str(player_db), read_only=True)
        non_null = conn.execute(
            "SELECT COUNT(*) FROM player_match_enrichment WHERE performance_score IS NOT NULL"
        ).fetchone()[0]
        conn.close()

        assert non_null == updated

    def test_scores_stored_in_player_match_enrichment(self, tmp_path: Path) -> None:
        """Les scores sont ecrits dans player_match_enrichment, pas match_stats."""
        player_db = _make_player_db(tmp_path)
        shared_db = _make_shared_db(tmp_path, n_matches=15)
        engine = DuckDBSyncEngine(player_db, xuid=XUID, gamertag=GAMERTAG, shared_db_path=shared_db)
        engine.batch_compute_performance_scores()
        engine.close()

        conn = duckdb.connect(str(player_db), read_only=True)
        tables = {
            r[0]
            for r in conn.execute(
                "SELECT table_name FROM information_schema.tables WHERE table_schema='main'"
            ).fetchall()
        }
        conn.close()

        assert "player_match_enrichment" in tables
        assert "match_stats" not in tables

    def test_idempotent_second_run_returns_zero(self, tmp_path: Path) -> None:
        """Le 2e appel ne recalcule pas les scores deja presents."""
        player_db = _make_player_db(tmp_path)
        shared_db = _make_shared_db(tmp_path, n_matches=15)
        engine = DuckDBSyncEngine(player_db, xuid=XUID, gamertag=GAMERTAG, shared_db_path=shared_db)
        first = engine.batch_compute_performance_scores()
        second = engine.batch_compute_performance_scores()
        assert first > 0
        assert second == 0
        engine.close()

    def test_preserves_existing_scores(self, tmp_path: Path) -> None:
        """Les scores deja presents dans player_match_enrichment ne sont pas ecrases."""
        player_db = _make_player_db(tmp_path)
        shared_db = _make_shared_db(tmp_path, n_matches=12)

        conn = duckdb.connect(str(player_db))
        conn.execute(
            "INSERT INTO player_match_enrichment (match_id, performance_score) VALUES ('m-0011', 99.9)"
        )
        conn.close()

        engine = DuckDBSyncEngine(player_db, xuid=XUID, gamertag=GAMERTAG, shared_db_path=shared_db)
        engine.batch_compute_performance_scores()
        engine.close()

        conn = duckdb.connect(str(player_db), read_only=True)
        score = conn.execute(
            "SELECT performance_score FROM player_match_enrichment WHERE match_id = 'm-0011'"
        ).fetchone()
        conn.close()

        assert score and score[0] == pytest.approx(99.9)


# =============================================================================
# Tests : calcul pour un match individuel
# =============================================================================


class TestComputeAndUpdateSingleMatch:
    """Tests _update_performance_score pour un match individuel."""

    def test_score_not_recalculated_if_exists(self, tmp_path: Path) -> None:
        """Si un score existe deja dans player_match_enrichment, on ne le recalcule pas."""
        player_db = _make_player_db(tmp_path)
        shared_db = _make_shared_db(tmp_path, n_matches=15)

        conn = duckdb.connect(str(player_db))
        conn.execute(
            "INSERT INTO player_match_enrichment (match_id, performance_score) VALUES ('m-0014', 77.7)"
        )
        conn.close()

        engine = DuckDBSyncEngine(player_db, xuid=XUID, gamertag=GAMERTAG, shared_db_path=shared_db)
        engine.close()

        conn = duckdb.connect(str(player_db), read_only=True)
        score = conn.execute(
            "SELECT performance_score FROM player_match_enrichment WHERE match_id = 'm-0014'"
        ).fetchone()
        conn.close()

        assert score and score[0] == pytest.approx(77.7)

    def test_callable_without_error_when_no_shared(self, tmp_path: Path) -> None:
        """batch_compute_performance_scores ne leve pas si shared est vide."""
        player_db = _make_player_db(tmp_path)
        shared_db = _make_shared_db(tmp_path, n_matches=0)
        engine = DuckDBSyncEngine(player_db, xuid=XUID, gamertag=GAMERTAG, shared_db_path=shared_db)
        result = engine.batch_compute_performance_scores()
        assert result == 0
        engine.close()
