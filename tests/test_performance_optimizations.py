"""Tests pour les optimisations performance v5.1.

Couvre :
- Vue mv_player_matches (migration + utilisation)
- Cache de schéma (DuckDBRepository)
- Index de performance
- Chemin optimisé _get_match_source via vue
"""

from __future__ import annotations

import time
from pathlib import Path

import duckdb
import pytest

from src.data.repositories.duckdb_repo import DuckDBRepository
from src.data.sync.migrations import (
    ensure_mv_player_matches_view,
    ensure_performance_indexes,
)

# =============================================================================
# Constantes et Helpers
# =============================================================================

PLAYER_XUID = "xuid_test_perf"
MATCH_IDS = ["match_p01", "match_p02", "match_p03"]


def _create_shared_with_data(db_path: Path) -> None:
    """Crée une shared_matches.duckdb avec données de test."""
    db_path.parent.mkdir(parents=True, exist_ok=True)
    conn = duckdb.connect(str(db_path))

    conn.execute("""
        CREATE TABLE match_registry (
            match_id VARCHAR PRIMARY KEY,
            start_time TIMESTAMP NOT NULL,
            end_time TIMESTAMP,
            playlist_id VARCHAR,
            playlist_name VARCHAR,
            map_id VARCHAR,
            map_name VARCHAR,
            pair_id VARCHAR,
            pair_name VARCHAR,
            game_variant_id VARCHAR,
            game_variant_name VARCHAR,
            is_ranked BOOLEAN DEFAULT FALSE,
            is_firefight BOOLEAN DEFAULT FALSE,
            duration_seconds INTEGER,
            team_0_score SMALLINT,
            team_1_score SMALLINT,
            player_count SMALLINT DEFAULT 8,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)

    conn.execute("""
        CREATE TABLE match_participants (
            match_id VARCHAR NOT NULL,
            xuid VARCHAR NOT NULL,
            gamertag VARCHAR,
            team_id INTEGER,
            outcome INTEGER,
            rank SMALLINT,
            score INTEGER,
            kills SMALLINT,
            deaths SMALLINT,
            assists SMALLINT,
            shots_fired INTEGER,
            shots_hit INTEGER,
            damage_dealt FLOAT,
            damage_taken FLOAT,
            avg_life_seconds FLOAT,
            headshot_kills SMALLINT,
            max_killing_spree SMALLINT,
            team_mmr FLOAT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            PRIMARY KEY (match_id, xuid)
        )
    """)

    # Insérer des matchs de test
    for i, mid in enumerate(MATCH_IDS):
        conn.execute(f"""
            INSERT INTO match_registry (match_id, start_time, playlist_name, map_name,
                duration_seconds, team_0_score, team_1_score, is_ranked)
            VALUES ('{mid}', '2025-01-{15 + i} 10:00:00', 'Ranked Arena', 'Aquarius',
                600, 50, 48, TRUE)
        """)
        conn.execute(f"""
            INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome,
                rank, score, kills, deaths, assists, shots_fired, shots_hit,
                avg_life_seconds, headshot_kills, max_killing_spree, team_mmr)
            VALUES ('{mid}', '{PLAYER_XUID}', 'TestPerf', 0, 2,
                1, 2500, 15, 6, 4, 200, 110,
                30.5, 5, 3, 1500.0)
        """)
        # Ajouter un adversaire
        conn.execute(f"""
            INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome,
                rank, score, kills, deaths, assists, shots_fired, shots_hit,
                avg_life_seconds, headshot_kills, max_killing_spree)
            VALUES ('{mid}', 'xuid_enemy', 'EnemyTest', 1, 3,
                2, 1800, 8, 10, 2, 150, 60,
                25.0, 2, 1)
        """)

    conn.close()


def _create_player_db(db_path: Path) -> None:
    """Crée une DB joueur minimale."""
    db_path.parent.mkdir(parents=True, exist_ok=True)
    conn = duckdb.connect(str(db_path))
    conn.execute("""
        CREATE TABLE match_stats (
            match_id VARCHAR PRIMARY KEY,
            start_time TIMESTAMP NOT NULL,
            map_id VARCHAR,
            map_name VARCHAR,
            playlist_id VARCHAR,
            playlist_name VARCHAR,
            pair_id VARCHAR,
            pair_name VARCHAR,
            game_variant_id VARCHAR,
            game_variant_name VARCHAR,
            outcome INTEGER,
            team_id INTEGER,
            kda FLOAT,
            max_killing_spree INTEGER,
            headshot_kills INTEGER,
            avg_life_seconds FLOAT,
            time_played_seconds INTEGER,
            kills INTEGER,
            deaths INTEGER,
            assists INTEGER,
            accuracy FLOAT,
            my_team_score INTEGER,
            enemy_team_score INTEGER,
            team_mmr FLOAT,
            enemy_mmr FLOAT,
            personal_score INTEGER,
            is_firefight BOOLEAN DEFAULT FALSE,
            is_ranked BOOLEAN DEFAULT FALSE
        )
    """)
    # Insérer un match de test avec MMR
    conn.execute("""
        INSERT INTO match_stats VALUES (
            'match_p01', '2025-01-15 10:00:00', 'map1', 'Aquarius', 'pl1',
            'Ranked Arena', 'pair1', 'Slayer', 'gv1', 'Slayer', 2, 0,
            2.5, 5, 3, 30.0, 600, 15, 6, 4, 55.0, 50, 48, 1500.0, 1480.0, 2500, FALSE, TRUE
        )
    """)
    conn.close()


@pytest.fixture
def shared_db(tmp_path: Path) -> Path:
    """Crée une shared_matches.duckdb temporaire avec données."""
    db_path = tmp_path / "data" / "warehouse" / "shared_matches.duckdb"
    _create_shared_with_data(db_path)
    return db_path


@pytest.fixture
def player_db(tmp_path: Path) -> Path:
    """Crée une DB joueur temporaire."""
    db_path = tmp_path / "data" / "players" / "TestPerf" / "stats.duckdb"
    _create_player_db(db_path)
    return db_path


@pytest.fixture
def repo_with_view(player_db: Path, shared_db: Path) -> DuckDBRepository:
    """DuckDBRepository avec vue mv_player_matches créée.

    La vue est créée directement dans le fichier shared_matches.duckdb
    (connexion directe, sans ATTACH). DuckDB résout correctement les
    références internes de la vue quand le fichier est attaché en READ_ONLY.
    """
    conn = duckdb.connect(str(shared_db))
    ensure_mv_player_matches_view(conn)
    conn.close()

    repo = DuckDBRepository(
        player_db_path=player_db,
        xuid=PLAYER_XUID,
        shared_db_path=shared_db,
        gamertag="TestPerf",
        read_only=True,
    )
    return repo


@pytest.fixture
def repo_without_view(player_db: Path, shared_db: Path) -> DuckDBRepository:
    """DuckDBRepository sans vue mv_player_matches (fallback v5.0)."""
    repo = DuckDBRepository(
        player_db_path=player_db,
        xuid=PLAYER_XUID,
        shared_db_path=shared_db,
        gamertag="TestPerf",
        read_only=True,
    )
    return repo


# =============================================================================
# Tests — Vue matérialisée mv_player_matches
# =============================================================================


class TestMvPlayerMatchesView:
    """Tests de la migration CREATE VIEW mv_player_matches."""

    def test_view_creation_idempotent(self, shared_db: Path):
        """La vue peut être créée plusieurs fois sans erreur."""
        conn = duckdb.connect(str(shared_db))
        ensure_mv_player_matches_view(conn)
        ensure_mv_player_matches_view(conn)  # 2e appel
        conn.close()

    def test_view_exists_after_creation(self, shared_db: Path):
        """La vue existe dans duckdb_views() après création."""
        conn = duckdb.connect(str(shared_db))
        ensure_mv_player_matches_view(conn)

        result = conn.execute("""
            SELECT COUNT(*) FROM duckdb_views()
            WHERE view_name = 'mv_player_matches'
        """).fetchone()
        assert result[0] == 1
        conn.close()

    def test_view_returns_expected_columns(self, shared_db: Path):
        """La vue contient toutes les colonnes attendues."""
        conn = duckdb.connect(str(shared_db))
        ensure_mv_player_matches_view(conn)

        # Utiliser DESCRIBE pour lister les colonnes de la vue
        columns = conn.execute("""
            SELECT column_name FROM information_schema.columns
            WHERE table_name = 'mv_player_matches'
            ORDER BY ordinal_position
        """).fetchall()
        col_names = [c[0] for c in columns]

        # Si information_schema.columns ne fonctionne pas avec les vues,
        # utiliser une approche alternative
        if not col_names:
            row = conn.execute("SELECT * FROM mv_player_matches LIMIT 0").description
            col_names = [d[0] for d in row] if row else []

        expected = [
            "match_id", "start_time", "map_id", "map_name",
            "playlist_id", "playlist_name", "pair_id", "pair_name",
            "game_variant_id", "game_variant_name", "xuid", "outcome", "team_id",
            "kda", "max_killing_spree", "headshot_kills", "avg_life_seconds",
            "time_played_seconds", "kills", "deaths", "assists", "accuracy",
            "my_team_score", "enemy_team_score", "team_mmr", "enemy_mmr",
            "personal_score", "is_firefight", "is_ranked",
        ]
        for col in expected:
            assert col in col_names, f"Colonne {col} manquante dans mv_player_matches"

        conn.close()

    def test_view_filters_by_xuid(self, shared_db: Path):
        """La vue retourne les données pour un xuid donné."""
        conn = duckdb.connect(str(shared_db))
        ensure_mv_player_matches_view(conn)

        rows = conn.execute(
            "SELECT * FROM mv_player_matches WHERE xuid = ?",
            [PLAYER_XUID],
        ).fetchall()
        assert len(rows) == len(MATCH_IDS)

        conn.close()

    def test_view_kda_calculation(self, shared_db: Path):
        """Le KDA est correctement calculé dans la vue."""
        conn = duckdb.connect(str(shared_db))
        ensure_mv_player_matches_view(conn)

        row = conn.execute(
            "SELECT kda, kills, deaths, assists FROM mv_player_matches "
            "WHERE xuid = ? LIMIT 1",
            [PLAYER_XUID],
        ).fetchone()

        # kills=15, deaths=6, assists=4
        # kda = (15 + 4/3) / 6 ≈ 2.722
        expected_kda = (15 + 4 / 3.0) / 6
        assert abs(row[0] - expected_kda) < 0.01

        conn.close()

    def test_view_accuracy_calculation(self, shared_db: Path):
        """L'accuracy est correctement calculée dans la vue."""
        conn = duckdb.connect(str(shared_db))
        ensure_mv_player_matches_view(conn)

        row = conn.execute(
            "SELECT accuracy FROM mv_player_matches WHERE xuid = ? LIMIT 1",
            [PLAYER_XUID],
        ).fetchone()

        # shots_fired=200, shots_hit=110 → 55.0%
        assert abs(row[0] - 55.0) < 0.01

        conn.close()

    def test_view_no_match_registry_skips(self, tmp_path: Path):
        """Si match_registry n'existe pas, la vue n'est pas créée."""
        db_path = tmp_path / "empty.duckdb"
        conn = duckdb.connect(str(db_path))
        conn.execute("CREATE TABLE dummy (x INTEGER)")
        ensure_mv_player_matches_view(conn)

        result = conn.execute("""
            SELECT COUNT(*) FROM duckdb_views()
            WHERE view_name = 'mv_player_matches'
        """).fetchone()
        assert result[0] == 0
        conn.close()


# =============================================================================
# Tests — Index de performance
# =============================================================================


class TestPerformanceIndexes:
    """Tests de la migration CREATE INDEX."""

    def test_indexes_creation(self, shared_db: Path):
        """Les index sont créés sans erreur."""
        conn = duckdb.connect(str(shared_db))
        ensure_performance_indexes(conn)
        conn.close()

    def test_indexes_idempotent(self, shared_db: Path):
        """Les index peuvent être créés plusieurs fois."""
        conn = duckdb.connect(str(shared_db))
        ensure_performance_indexes(conn)
        ensure_performance_indexes(conn)  # 2e appel
        conn.close()

    def test_indexes_no_tables_skips(self, tmp_path: Path):
        """Si les tables n'existent pas, pas d'erreur."""
        db_path = tmp_path / "empty_idx.duckdb"
        conn = duckdb.connect(str(db_path))
        conn.execute("CREATE TABLE dummy (x INTEGER)")
        ensure_performance_indexes(conn)
        conn.close()


# =============================================================================
# Tests — Cache de schéma (DuckDBRepository)
# =============================================================================


class TestSchemaCache:
    """Tests du cache de schéma dans DuckDBRepository."""

    def test_has_column_cached(self, repo_with_view: DuckDBRepository):
        """_has_column utilise le cache après le premier appel."""
        conn = repo_with_view._get_connection()

        # Premier appel → requête information_schema
        result1 = repo_with_view._has_column(conn, "match_stats", "match_id")
        assert result1 is True

        # Vérifier que le cache est peuplé
        assert "match_stats.match_id" in repo_with_view._schema_cache

        # Deuxième appel → cache
        result2 = repo_with_view._has_column(conn, "match_stats", "match_id")
        assert result2 is True

    def test_has_column_nonexistent_cached(self, repo_with_view: DuckDBRepository):
        """_has_column cache aussi les résultats négatifs."""
        conn = repo_with_view._get_connection()

        result = repo_with_view._has_column(conn, "match_stats", "nonexistent_col")
        assert result is False
        assert "match_stats.nonexistent_col" in repo_with_view._schema_cache
        assert repo_with_view._schema_cache["match_stats.nonexistent_col"] is False

    def test_has_shared_table_cached(self, repo_with_view: DuckDBRepository):
        """_has_shared_table utilise le cache."""
        # Premier appel
        result1 = repo_with_view._has_shared_table("match_registry")
        assert result1 is True
        assert "shared_table:match_registry" in repo_with_view._table_cache

        # Deuxième appel (cache)
        result2 = repo_with_view._has_shared_table("match_registry")
        assert result2 is True

    def test_has_shared_view_cached(self, repo_with_view: DuckDBRepository):
        """_has_shared_view détecte la vue mv_player_matches."""
        result = repo_with_view._has_shared_view("mv_player_matches")
        assert result is True
        assert "shared_view:mv_player_matches" in repo_with_view._view_cache

    def test_has_shared_view_false_for_nonexistent(self, repo_with_view: DuckDBRepository):
        """_has_shared_view retourne False pour une vue inexistante."""
        result = repo_with_view._has_shared_view("nonexistent_view")
        assert result is False

    def test_cache_cleared_on_close(self, repo_with_view: DuckDBRepository):
        """Les caches sont vidés lors du close()."""
        repo_with_view._has_shared_table("match_registry")
        repo_with_view._has_shared_view("mv_player_matches")
        conn = repo_with_view._get_connection()
        repo_with_view._has_column(conn, "match_stats", "match_id")

        assert len(repo_with_view._schema_cache) > 0
        assert len(repo_with_view._table_cache) > 0
        assert len(repo_with_view._view_cache) > 0

        repo_with_view.close()

        assert len(repo_with_view._schema_cache) == 0
        assert len(repo_with_view._table_cache) == 0
        assert len(repo_with_view._view_cache) == 0

    def test_has_table_cached(self, repo_with_view: DuckDBRepository):
        """_has_table_cached fonctionne pour les tables main."""
        conn = repo_with_view._get_connection()
        result = repo_with_view._has_table_cached(conn, "match_stats")
        assert result is True
        assert "main_table:match_stats" in repo_with_view._table_cache


# =============================================================================
# Tests — Chemin optimisé _get_match_source
# =============================================================================


class TestGetMatchSourceOptimized:
    """Tests du chemin optimisé _get_match_source avec vue."""

    def test_uses_view_when_available(self, repo_with_view: DuckDBRepository):
        """_get_match_source utilise la vue quand elle est disponible."""
        conn = repo_with_view._get_connection()
        source, params = repo_with_view._get_match_source(conn)

        assert "mv_player_matches" in source
        assert len(params) == 1
        assert params[0] == PLAYER_XUID

    def test_fallback_v5_legacy_without_view(self, repo_without_view: DuckDBRepository):
        """_get_match_source fait le fallback v5.0 sans vue."""
        conn = repo_without_view._get_connection()
        source, params = repo_without_view._get_match_source(conn)

        # Devrait utiliser le chemin legacy (shared.match_registry + match_participants)
        assert "shared.match_registry" in source or "mv_player_matches" not in source
        assert len(params) == 1

    def test_load_matches_with_view(self, repo_with_view: DuckDBRepository):
        """load_matches fonctionne avec la vue."""
        matches = repo_with_view.load_matches()
        # Au moins les matchs présents dans shared qui existent aussi en local (ou tous via shared)
        assert len(matches) >= 1

    def test_load_matches_with_view_no_firefight(self, repo_with_view: DuckDBRepository):
        """load_matches filtre firefight correctement via la vue."""
        matches = repo_with_view.load_matches(include_firefight=False)
        for m in matches:
            assert not getattr(m, "is_firefight", False)

    def test_v4_mode_unchanged(self, player_db: Path):
        """Mode v4 (sans shared) n'est pas affecté par les changements."""
        repo = DuckDBRepository(
            player_db_path=player_db,
            xuid=PLAYER_XUID,
            shared_db_path=Path("/nonexistent/shared_matches.duckdb"),
            gamertag="TestPerf",
            read_only=True,
        )
        conn = repo._get_connection()
        source, params = repo._get_match_source(conn)

        assert "match_stats" in source
        assert "mv_player_matches" not in source
        assert params == []
        repo.close()

    def test_empty_xuid_mode_unchanged(self, player_db: Path, shared_db: Path):
        """Avec XUID vide, reste en mode local."""
        repo = DuckDBRepository(
            player_db_path=player_db,
            xuid="",
            shared_db_path=shared_db,
            gamertag="TestPerf",
            read_only=True,
        )
        conn = repo._get_connection()
        source, params = repo._get_match_source(conn)

        assert "match_stats" in source
        assert "mv_player_matches" not in source
        assert params == []
        repo.close()

    def test_match_data_consistency(self, repo_with_view: DuckDBRepository):
        """Les données retournées via la vue sont cohérentes."""
        matches = repo_with_view.load_matches()
        if not matches:
            pytest.skip("Aucun match retourné")

        m = matches[0]
        # Vérifier les champs essentiels
        assert m.match_id is not None
        assert m.kills >= 0
        assert m.deaths >= 0
        assert m.assists >= 0


# =============================================================================
# Tests — Performance (benchmarks légers)
# =============================================================================


class TestPerformanceBenchmarks:
    """Benchmarks légers pour valider les gains de performance."""

    def test_view_query_fast(self, shared_db: Path):
        """La requête via la vue est rapide (<100ms)."""
        conn = duckdb.connect(str(shared_db))
        ensure_mv_player_matches_view(conn)

        # Vérifier que la vue est accessible
        rows = conn.execute(
            "SELECT COUNT(*) FROM mv_player_matches WHERE xuid = ?",
            [PLAYER_XUID],
        ).fetchone()
        assert rows[0] > 0, "La vue devrait contenir des données"

        start = time.perf_counter()
        for _ in range(10):
            conn.execute(
                "SELECT * FROM mv_player_matches WHERE xuid = ?",
                [PLAYER_XUID],
            ).fetchall()
        elapsed = (time.perf_counter() - start) / 10

        assert elapsed < 0.1, f"Requête vue trop lente: {elapsed:.3f}s"
        conn.close()

    def test_schema_cache_faster_than_no_cache(self, repo_with_view: DuckDBRepository):
        """Le cache de schéma est plus rapide que les requêtes directes."""
        conn = repo_with_view._get_connection()

        # Premier appel (pas de cache)
        start = time.perf_counter()
        for _ in range(50):
            # Forcer un miss de cache en nettoyant
            repo_with_view._schema_cache.clear()
            repo_with_view._has_column(conn, "match_stats", "match_id")
        time_no_cache = time.perf_counter() - start

        # Deuxième round (avec cache)
        repo_with_view._schema_cache.clear()
        repo_with_view._has_column(conn, "match_stats", "match_id")  # Peupler le cache
        start = time.perf_counter()
        for _ in range(50):
            repo_with_view._has_column(conn, "match_stats", "match_id")
        time_cached = time.perf_counter() - start

        # Le cache devrait être significativement plus rapide
        assert time_cached < time_no_cache, (
            f"Cache pas plus rapide: {time_cached:.4f}s vs {time_no_cache:.4f}s"
        )
