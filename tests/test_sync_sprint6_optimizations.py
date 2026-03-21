"""Tests Sprint 6 — Optimisations API & Sync.

Teste :
- 6.1 Parallélisation appels API (asyncio.gather)
- 6.2 Désactivation perf score pendant sync (defer_performance_score)
- 6.3 Batch compute performance scores post-sync
- 6.4 Batching des insertions DB (batch_commit_size)
- 6.5 Rate limit et parallel_matches augmentés
"""

from __future__ import annotations

import gc
import uuid
from datetime import datetime, timedelta, timezone
from pathlib import Path

import duckdb
import pytest

from src.data.sync.engine import DuckDBSyncEngine
from src.data.sync.models import MatchStatsRow, SyncOptions

# =============================================================================
# Fixtures
# =============================================================================


@pytest.fixture
def temp_duckdb(tmp_path: Path) -> Path:
    """Crée une base DuckDB temporaire avec le schéma V5 (player_match_enrichment)."""
    db_path = tmp_path / f"test_player_{uuid.uuid4().hex[:8]}" / "stats.duckdb"
    db_path.parent.mkdir(parents=True)

    conn = duckdb.connect(str(db_path))
    conn.execute("""
        CREATE TABLE IF NOT EXISTS player_match_enrichment (
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
    conn.execute("""
        CREATE TABLE IF NOT EXISTS sync_meta (
            key VARCHAR PRIMARY KEY,
            value VARCHAR,
            updated_at TIMESTAMP
        )
    """)
    conn.close()
    del conn
    gc.collect()

    return db_path


@pytest.fixture
def temp_shared_db(tmp_path: Path) -> Path:
    """Crée shared_matches_v2.duckdb avec match_registry + match_participants (schéma V5)."""
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
    conn.close()
    del conn
    gc.collect()
    return db_path


def _make_match_row(
    match_id: str,
    start_time: datetime,
    kills: int = 10,
    deaths: int = 8,
    assists: int = 5,
) -> MatchStatsRow:
    """Crée un MatchStatsRow de test."""
    return MatchStatsRow(
        match_id=match_id,
        start_time=start_time,
        kills=kills,
        deaths=deaths,
        assists=assists,
        kda=round((kills + assists / 3) / max(deaths, 1), 2),
        accuracy=0.45,
        time_played_seconds=600,
        avg_life_seconds=45.0,
        playlist_id="playlist-123",
        playlist_name="Ranked Arena",
        map_id="map-456",
        map_name="Recharge",
        outcome=2,
        team_id=0,
    )


# =============================================================================
# 6.1 — Tests parallélisation API
# =============================================================================


class TestAPIParallelization:
    """Tests pour la parallélisation des appels skill + events."""

    def test_sync_options_defaults_sprint6(self):
        """Les valeurs par défaut Sprint 6 sont correctes."""
        opts = SyncOptions()
        assert opts.requests_per_second == 15, "Rate limit devrait être 15 req/s (benchmark Run 3)"
        assert opts.parallel_matches == 10, "parallel_matches devrait être 10 (benchmark Run 3)"
        assert opts.defer_performance_score is True, "defer_perf_score devrait être True"
        assert opts.batch_commit_size == -1, "batch_commit_size devrait être -1 (auto)"


# =============================================================================
# 6.2 — Tests defer_performance_score
# =============================================================================


class TestDeferPerformanceScore:
    """Tests pour la désactivation du calcul perf score pendant sync."""

    def test_defer_performance_score_default_true(self):
        """Par défaut, defer_performance_score est True."""
        opts = SyncOptions()
        assert opts.defer_performance_score is True

    def test_defer_can_be_disabled(self):
        """On peut explicitement désactiver le defer."""
        opts = SyncOptions(defer_performance_score=False)
        assert opts.defer_performance_score is False


# =============================================================================
# 6.3 — Tests batch_compute_performance_scores
# =============================================================================


class TestBatchComputePerformanceScores:
    """Tests batch_compute_performance_scores — architecture V5 (shared_matches)."""

    XUID = "2535423456789"
    GAMERTAG = "TestPlayer"

    def _fill_shared_db(self, db_path: Path, xuid: str, n_matches: int) -> None:
        """Insère n_matches dans la shared DB déjà créée."""
        conn = duckdb.connect(str(db_path))
        base_time = datetime(2024, 1, 1, tzinfo=timezone.utc)
        for i in range(n_matches):
            mid = f"m-{i:04d}"
            t = base_time + timedelta(hours=i)
            conn.execute(
                "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
                (mid, t),
            )
            conn.execute(
                "INSERT INTO match_participants (match_id, xuid, gamertag) VALUES (?, ?, ?)",
                (mid, xuid, self.GAMERTAG),
            )
        conn.commit()
        conn.close()

    def test_batch_compute_no_matches(self, temp_duckdb: Path, temp_shared_db: Path):
        """Avec une shared DB vide, batch_compute retourne 0."""
        engine = DuckDBSyncEngine(
            player_db_path=str(temp_duckdb),
            xuid=self.XUID,
            gamertag=self.GAMERTAG,
            shared_db_path=temp_shared_db,
        )
        result = engine.batch_compute_performance_scores()
        assert result == 0
        engine.close()

    def test_batch_compute_with_sufficient_history(self, temp_duckdb: Path, temp_shared_db: Path):
        """Calcule les scores quand assez de matchs dans shared."""
        self._fill_shared_db(temp_shared_db, self.XUID, n_matches=30)
        engine = DuckDBSyncEngine(
            player_db_path=str(temp_duckdb),
            xuid=self.XUID,
            gamertag=self.GAMERTAG,
            shared_db_path=temp_shared_db,
        )
        updated = engine.batch_compute_performance_scores()
        engine.close()
        assert updated > 0, "Au moins quelques scores doivent etre calcules"

    def test_batch_compute_idempotent(self, temp_duckdb: Path, temp_shared_db: Path):
        """Executer batch_compute 2 fois ne recalcule pas les scores existants."""
        self._fill_shared_db(temp_shared_db, self.XUID, n_matches=30)
        engine = DuckDBSyncEngine(
            player_db_path=str(temp_duckdb),
            xuid=self.XUID,
            gamertag=self.GAMERTAG,
            shared_db_path=temp_shared_db,
        )
        first_run = engine.batch_compute_performance_scores()
        second_run = engine.batch_compute_performance_scores()
        assert first_run > 0
        assert second_run == 0, "Le 2e appel ne doit rien recalculer"
        engine.close()

    def test_batch_compute_all_scores_already_present(
        self, temp_duckdb: Path, temp_shared_db: Path
    ):
        """Si tous les matchs ont deja un score, retourne 0."""
        self._fill_shared_db(temp_shared_db, self.XUID, n_matches=5)
        conn = duckdb.connect(str(temp_duckdb))
        for i in range(5):
            conn.execute(
                "INSERT INTO player_match_enrichment (match_id, performance_score) VALUES (?, ?)",
                (f"m-{i:04d}", 65.0),
            )
        conn.commit()
        conn.close()
        engine = DuckDBSyncEngine(
            player_db_path=str(temp_duckdb),
            xuid=self.XUID,
            gamertag=self.GAMERTAG,
            shared_db_path=temp_shared_db,
        )
        result = engine.batch_compute_performance_scores()
        assert result == 0
        engine.close()


# =============================================================================
# 6.4 — Tests batching DB commits
# =============================================================================


class TestBatchCommitSize:
    """Tests pour le batching des commits DB."""

    def test_batch_commit_size_in_options(self):
        """batch_commit_size est configurable."""
        opts = SyncOptions(batch_commit_size=20)
        assert opts.batch_commit_size == 20

    def test_batch_commit_size_zero_disables(self):
        """batch_commit_size=0 désactive le commit intermédiaire."""
        opts = SyncOptions(batch_commit_size=0)
        assert opts.batch_commit_size == 0


# =============================================================================
# 6.5 — Tests rate limit augmenté
# =============================================================================


class TestRateLimitIncreased:
    """Tests pour l'augmentation du rate limit."""

    def test_default_rate_limit_is_15(self):
        """Le rate limit par défaut est 15 req/s (benchmark Run 3)."""
        opts = SyncOptions()
        assert opts.requests_per_second == 15

    def test_default_parallel_matches_is_10(self):
        """Le nombre de matchs parallèles par défaut est 10 (benchmark Run 3)."""
        opts = SyncOptions()
        assert opts.parallel_matches == 10

    def test_custom_rate_limit(self):
        """On peut personnaliser le rate limit."""
        opts = SyncOptions(requests_per_second=3, parallel_matches=2)
        assert opts.requests_per_second == 3
        assert opts.parallel_matches == 2


# =============================================================================
# Tests d'intégration : engine lifecycle
# =============================================================================


class TestEngineLifecycle:
    """Tests de cycle de vie du moteur avec les optimisations Sprint 6."""

    def test_engine_creates_with_new_options(self, temp_duckdb: Path):
        """Le moteur s'initialise correctement avec les nouvelles options."""
        engine = DuckDBSyncEngine(
            player_db_path=str(temp_duckdb),
            xuid="2535423456789",
            gamertag="TestPlayer",
        )
        # Vérifier que batch_compute_performance_scores est appelable
        assert hasattr(engine, "batch_compute_performance_scores")
        assert callable(engine.batch_compute_performance_scores)
        engine.close()

    def test_batch_compute_perf_scores_returns_int(self, temp_duckdb: Path):
        """batch_compute_performance_scores retourne toujours un int."""
        engine = DuckDBSyncEngine(
            player_db_path=str(temp_duckdb),
            xuid="2535423456789",
            gamertag="TestPlayer",
        )
        result = engine.batch_compute_performance_scores()
        assert isinstance(result, int)
        engine.close()
