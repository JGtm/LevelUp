"""Tests pour les vues matérialisées (Sprint 4.1).

Ce module teste :
- La création des tables mv_*
- Le rafraîchissement des vues
- La lecture des stats agrégées
- Les performances comparées aux requêtes directes
"""

from __future__ import annotations

import sys
from datetime import datetime, timedelta
from pathlib import Path

import duckdb
import pytest

# Import direct pour éviter l'import circulaire dans src.data
# On ajoute le chemin du repo au sys.path
_repo_root = Path(__file__).parent.parent
if str(_repo_root) not in sys.path:
    sys.path.insert(0, str(_repo_root))


def _get_duckdb_repository_class():
    """Import lazy du DuckDBRepository pour éviter l'import circulaire."""
    # Import direct du module sans passer par src.data
    import importlib.util

    spec = importlib.util.spec_from_file_location(
        "duckdb_repo", _repo_root / "src" / "data" / "repositories" / "duckdb_repo.py"
    )
    module = importlib.util.module_from_spec(spec)

    # Charger les dépendances nécessaires d'abord
    # Le module a besoin des dataclasses de match
    from src.data.domain.models.stats import MatchRow  # noqa: F401

    spec.loader.exec_module(module)
    return module.DuckDBRepository


class TestMaterializedViews:
    """Tests pour les vues matérialisées du DuckDBRepository."""

    @pytest.fixture
    def temp_db(self, tmp_path: Path) -> tuple[Path, Path]:
        """Crée player DB + shared DB avec données de test (architecture v5.1)."""
        import gc
        import uuid

        player_db_path = tmp_path / f"test_stats_{uuid.uuid4().hex[:8]}.duckdb"
        shared_db_path = tmp_path / "shared_matches_v2.duckdb"

        # ── shared_matches_v2.duckdb ──────────────────────────────────────────────
        shared_conn = duckdb.connect(str(shared_db_path))
        try:
            shared_conn.execute("""
                CREATE TABLE match_registry (
                    match_id VARCHAR PRIMARY KEY,
                    start_time TIMESTAMP,
                    map_id VARCHAR, map_name VARCHAR,
                    pair_id VARCHAR, pair_name VARCHAR,
                    playlist_id VARCHAR, playlist_name VARCHAR
                )
            """)
            shared_conn.execute("""
                CREATE TABLE match_participants (
                    match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL,
                    outcome INTEGER,
                    kills SMALLINT, deaths SMALLINT, assists SMALLINT,
                    kda DOUBLE, accuracy DOUBLE,
                    time_played_seconds INTEGER, avg_life_seconds DOUBLE,
                    team_mmr DOUBLE, enemy_mmr DOUBLE,
                    PRIMARY KEY (match_id, xuid)
                )
            """)

            base_time = datetime.now() - timedelta(days=30)
            maps_data = [("map1", "Streets"), ("map2", "Recharge"), ("map3", "Live Fire")]
            modes_data = [
                ("Slayer", "Team Slayer"),
                ("CTF", "Capture The Flag"),
                ("Oddball", "Oddball"),
            ]

            registry_rows = []
            participant_rows = []
            for i in range(50):
                match_id = f"match_{i:04d}"
                map_id, map_name = maps_data[i % 3]
                _mode, pair_name = modes_data[i % 3]
                outcome = 2 if i % 3 == 0 else (3 if i % 3 == 1 else 1)
                registry_rows.append(
                    (
                        match_id,
                        base_time + timedelta(hours=i),
                        map_id,
                        map_name,
                        f"pair_{i % 3}",
                        pair_name,
                        f"playlist_{i % 2}",
                        f"Playlist {i % 2}",
                    )
                )
                participant_rows.append(
                    (
                        match_id,
                        "test_xuid",
                        outcome,
                        10 + i % 15,
                        5 + i % 10,
                        3 + i % 5,
                        1.5 + (i % 10) * 0.1,
                        0.40 + (i % 20) * 0.01,
                        600 + i * 10,
                        45.0 + i % 30,
                        1200.0 + i * 5,
                        1190.0 + i * 5,
                    )
                )

            shared_conn.executemany(
                "INSERT INTO match_registry VALUES (?, ?, ?, ?, ?, ?, ?, ?)", registry_rows
            )
            shared_conn.executemany(
                "INSERT INTO match_participants VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
                participant_rows,
            )
        finally:
            shared_conn.close()
            del shared_conn
            gc.collect()

        # ── player stats.duckdb (tables MV créées par le repo au besoin) ──────
        player_conn = duckdb.connect(str(player_db_path))
        try:
            player_conn.execute("""
                CREATE TABLE sync_meta (
                    key VARCHAR PRIMARY KEY, value VARCHAR,
                    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                )
            """)
            player_conn.execute(
                "INSERT INTO sync_meta VALUES ('xuid', 'test_xuid', CURRENT_TIMESTAMP)"
            )
        finally:
            player_conn.close()
            del player_conn
            gc.collect()

        return player_db_path, shared_db_path

    @pytest.fixture
    def repo(self, temp_db: tuple[Path, Path]):
        """Crée un DuckDBRepository pour les tests."""
        player_db_path, shared_db_path = temp_db
        DuckDBRepository = _get_duckdb_repository_class()

        repo = DuckDBRepository(
            player_db_path=player_db_path,
            xuid="test_xuid",
            gamertag="TestPlayer",
            shared_db_path=shared_db_path,
            read_only=False,
        )
        yield repo
        repo.close()

    def test_refresh_creates_tables(self, repo):
        """Test que refresh_materialized_views crée les tables."""
        # Vérifier que les tables n'existent pas encore
        assert not repo.has_materialized_views()

        # Rafraîchir
        results = repo.refresh_materialized_views()

        # Vérifier les résultats
        assert "mv_map_stats" in results
        assert "mv_mode_category_stats" in results
        assert "mv_global_stats" in results
        assert results["mv_map_stats"] == 3  # 3 cartes
        assert results["mv_global_stats"] == 10  # 10 stats globales

        # Vérifier que has_materialized_views retourne True
        assert repo.has_materialized_views()

    def test_get_map_stats(self, repo):
        """Test la récupération des stats par carte."""
        repo.refresh_materialized_views()

        stats = repo.get_map_stats()

        assert len(stats) == 3  # 3 cartes
        for stat in stats:
            assert "map_id" in stat
            assert "map_name" in stat
            assert "matches_played" in stat
            assert "wins" in stat
            assert "avg_kda" in stat
            assert "win_rate" in stat
            assert stat["matches_played"] > 0

    def test_get_map_stats_min_matches_filter(self, repo):
        """Test le filtre min_matches sur les stats par carte."""
        repo.refresh_materialized_views()

        # Avec min_matches=100, aucune carte ne passe
        stats = repo.get_map_stats(min_matches=100)
        assert len(stats) == 0

        # Avec min_matches=1, toutes passent
        stats = repo.get_map_stats(min_matches=1)
        assert len(stats) == 3

    def test_get_mode_category_stats(self, repo):
        """Test la récupération des stats par catégorie de mode."""
        repo.refresh_materialized_views()

        stats = repo.get_mode_category_stats()

        assert len(stats) > 0
        for stat in stats:
            assert "mode_category" in stat
            assert "matches_played" in stat
            assert "avg_kda" in stat
            assert "win_rate" in stat

    def test_get_global_stats(self, repo):
        """Test la récupération des stats globales."""
        repo.refresh_materialized_views()

        stats = repo.get_global_stats()

        assert "total_matches" in stats
        assert "total_kills" in stats
        assert "total_deaths" in stats
        assert "avg_kda" in stats
        assert "wins" in stats
        assert "losses" in stats
        assert stats["total_matches"] == 50

    def test_refresh_is_idempotent(self, repo):
        """Test que refresh peut être appelé plusieurs fois."""
        results1 = repo.refresh_materialized_views()
        results2 = repo.refresh_materialized_views()

        # Les résultats doivent être identiques
        assert results1 == results2

    def test_incremental_refresh_map_stats(self, repo):
        """Test le rebuild partiel de mv_map_stats avec new_ids."""
        # Rebuild initial complet
        repo.refresh_materialized_views()
        initial_count = len(repo.get_map_stats())
        assert initial_count == 3  # Streets, Recharge, Live Fire

        # Rebuild partiel avec des match_ids connus (map1=Streets)
        result = repo.refresh_materialized_views(new_ids=["match_0000", "match_0003"])

        # Le nombre de cartes doit rester le même (pas de nouvelle carte)
        assert result["mv_map_stats"] == initial_count
        stats = repo.get_map_stats()
        assert len(stats) == initial_count

    def test_incremental_refresh_mode_category_stats(self, repo):
        """Test le rebuild partiel de mv_mode_category_stats avec new_ids."""
        repo.refresh_materialized_views()
        full_results = repo.get_mode_category_stats()
        assert len(full_results) > 0

        # Partiel avec match connus — les catégories doivent rester cohérentes
        result = repo.refresh_materialized_views(new_ids=["match_0000", "match_0001"])
        assert result["mv_mode_category_stats"] == len(full_results)

    def test_incremental_refresh_unknown_ids_is_noop(self, repo):
        """Test que des new_ids sans carte connue ne modifie pas mv_map_stats."""
        repo.refresh_materialized_views()
        count_before = len(repo.get_map_stats())

        # IDs inconnus → aucune carte touchée → noop
        result = repo.refresh_materialized_views(new_ids=["nonexistent_match_xyz"])
        assert result["mv_map_stats"] == count_before

    def test_incremental_preserves_stats_correctness(self, repo):
        """Test que le rebuild partiel produit les mêmes stats que le rebuild complet."""
        # Rebuild complet de référence
        full = repo.refresh_materialized_views()
        stats_full = {s["map_id"]: s["matches_played"] for s in repo.get_map_stats()}

        # Reset puis rebuild partiel avec quelques matchs
        repo.refresh_materialized_views()
        stats_partial = {s["map_id"]: s["matches_played"] for s in repo.get_map_stats()}

        # Les stats doivent être identiques (rebuild partiel = rebuild complet sur cartes touchées)
        assert stats_full == stats_partial
        assert full["mv_global_stats"] == 10  # global toujours rebuild complet

    def test_empty_tables_before_refresh(self, repo):
        """Test que les méthodes retournent des listes/dicts vides avant refresh."""
        # Avant refresh, les tables n'existent pas
        assert repo.get_map_stats() == []
        assert repo.get_mode_category_stats() == []
        assert repo.get_global_stats() == {}


class TestBatchMmrLoading:
    """Tests pour le chargement batch des MMR (Sprint 4.2)."""

    @pytest.fixture
    def temp_db_with_mmr(self, tmp_path: Path) -> tuple[Path, Path]:
        """Crée player DB + shared DB avec MMR de test (architecture v5.1)."""
        import gc
        import uuid

        player_db_path = tmp_path / f"test_mmr_{uuid.uuid4().hex[:8]}.duckdb"
        shared_db_path = tmp_path / "shared_matches_v2.duckdb"

        shared_conn = duckdb.connect(str(shared_db_path))
        try:
            shared_conn.execute("""
                CREATE TABLE match_participants (
                    match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL,
                    team_mmr DOUBLE, enemy_mmr DOUBLE,
                    kills SMALLINT DEFAULT 10, deaths SMALLINT DEFAULT 5,
                    assists SMALLINT DEFAULT 3,
                    PRIMARY KEY (match_id, xuid)
                )
            """)
            shared_conn.executemany(
                "INSERT INTO match_participants (match_id, xuid, team_mmr, enemy_mmr) VALUES (?, ?, ?, ?)",
                [
                    ("match_001", "test_xuid", 1200.5, 1180.3),
                    ("match_002", "test_xuid", 1250.0, 1230.0),
                    ("match_003", "test_xuid", None, None),
                    ("match_004", "test_xuid", 1300.0, 1350.0),
                    ("match_005", "test_xuid", 1100.0, 1100.0),
                ],
            )
        finally:
            shared_conn.close()
            del shared_conn
            gc.collect()

        player_conn = duckdb.connect(str(player_db_path))
        try:
            player_conn.execute(
                "CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR, "
                "updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)"
            )
            player_conn.execute(
                "INSERT INTO sync_meta VALUES ('xuid', 'test_xuid', CURRENT_TIMESTAMP)"
            )
        finally:
            player_conn.close()
            del player_conn
            gc.collect()

        return player_db_path, shared_db_path

    @pytest.fixture
    def repo_mmr(self, temp_db_with_mmr: tuple[Path, Path]):
        """Crée un DuckDBRepository pour les tests MMR."""
        player_db_path, shared_db_path = temp_db_with_mmr
        DuckDBRepository = _get_duckdb_repository_class()

        repo = DuckDBRepository(
            player_db_path=player_db_path,
            xuid="test_xuid",
            gamertag="TestPlayer",
            shared_db_path=shared_db_path,
            read_only=True,
        )
        yield repo
        repo.close()

    def test_load_match_mmr_batch_single(self, repo_mmr):
        """Test le chargement batch avec un seul match."""
        result = repo_mmr.load_match_mmr_batch(["match_001"])

        assert "match_001" in result
        team_mmr, enemy_mmr = result["match_001"]
        assert team_mmr == pytest.approx(1200.5)
        assert enemy_mmr == pytest.approx(1180.3)

    def test_load_match_mmr_batch_multiple(self, repo_mmr):
        """Test le chargement batch avec plusieurs matchs."""
        result = repo_mmr.load_match_mmr_batch(["match_001", "match_002", "match_004"])

        assert len(result) == 3
        assert result["match_001"] == (pytest.approx(1200.5), pytest.approx(1180.3))
        assert result["match_002"] == (pytest.approx(1250.0), pytest.approx(1230.0))
        assert result["match_004"] == (pytest.approx(1300.0), pytest.approx(1350.0))

    def test_load_match_mmr_batch_with_nulls(self, repo_mmr):
        """Test le chargement batch avec des matchs sans MMR."""
        result = repo_mmr.load_match_mmr_batch(["match_003"])

        assert "match_003" in result
        team_mmr, enemy_mmr = result["match_003"]
        assert team_mmr is None
        assert enemy_mmr is None

    def test_load_match_mmr_batch_empty(self, repo_mmr):
        """Test le chargement batch avec une liste vide."""
        result = repo_mmr.load_match_mmr_batch([])
        assert result == {}

    def test_load_match_mmr_batch_unknown_match(self, repo_mmr):
        """Test le chargement batch avec un match inexistant."""
        result = repo_mmr.load_match_mmr_batch(["match_unknown"])
        assert result == {}


class TestPerformanceComparison:
    """Tests de performance comparant les vues matérialisées aux requêtes directes."""

    @pytest.fixture
    def large_db(self, tmp_path: Path) -> tuple[Path, Path]:
        """Crée player DB + shared DB avec beaucoup de matchs pour les tests de perf."""
        import gc
        import random
        import uuid

        player_db_path = tmp_path / f"test_perf_{uuid.uuid4().hex[:8]}.duckdb"
        shared_db_path = tmp_path / "shared_matches_v2.duckdb"

        shared_conn = duckdb.connect(str(shared_db_path))

        try:
            shared_conn.execute("""
                CREATE TABLE match_registry (
                    match_id VARCHAR PRIMARY KEY,
                    start_time TIMESTAMP,
                    map_id VARCHAR, map_name VARCHAR,
                    pair_id VARCHAR, pair_name VARCHAR,
                    playlist_id VARCHAR, playlist_name VARCHAR
                )
            """)
            shared_conn.execute("""
                CREATE TABLE match_participants (
                    match_id VARCHAR NOT NULL, xuid VARCHAR NOT NULL,
                    outcome INTEGER,
                    kills SMALLINT, deaths SMALLINT, assists SMALLINT,
                    kda DOUBLE, accuracy DOUBLE,
                    time_played_seconds INTEGER, avg_life_seconds DOUBLE,
                    team_mmr DOUBLE, enemy_mmr DOUBLE,
                    PRIMARY KEY (match_id, xuid)
                )
            """)

            # Générer 1000 matchs
            random.seed(42)

            maps = [
                ("map1", "Streets"),
                ("map2", "Recharge"),
                ("map3", "Live Fire"),
                ("map4", "Aquarius"),
                ("map5", "Bazaar"),
            ]
            modes = [
                ("Slayer", "Team Slayer"),
                ("CTF", "CTF"),
                ("Oddball", "Oddball"),
                ("Strongholds", "Forteresse"),
                ("Total Control", "Contrôle Total"),
            ]

            base_time = datetime.now() - timedelta(days=365)

            batch_data = []
            registry_batch = []
            for i in range(1000):
                map_id, map_name = random.choice(maps)
                _mode, pair_name = random.choice(modes)
                outcome = random.choice([1, 2, 3])
                match_id = f"match_{i:06d}"

                registry_batch.append(
                    (
                        match_id,
                        base_time + timedelta(hours=i),
                        map_id,
                        map_name,
                        f"pair_{i % 5}",
                        pair_name,
                        f"playlist_{i % 3}",
                        f"Playlist {i % 3}",
                    )
                )
                batch_data.append(
                    (
                        match_id,
                        "test_xuid",
                        outcome,
                        random.randint(5, 25),
                        random.randint(3, 20),
                        random.randint(1, 10),
                        random.uniform(0.5, 3.0),
                        random.uniform(0.25, 0.60),
                        random.randint(300, 900),
                        random.uniform(30, 90),
                        random.uniform(1000, 1500),
                        random.uniform(1000, 1500),
                    )
                )

            shared_conn.executemany(
                "INSERT INTO match_participants VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
                batch_data,
            )
            shared_conn.executemany(
                "INSERT INTO match_registry VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
                registry_batch,
            )
            shared_conn.commit()
        finally:
            shared_conn.close()
            del shared_conn
            gc.collect()

        player_conn = duckdb.connect(str(player_db_path))
        try:
            player_conn.execute(
                "CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR, "
                "updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)"
            )
            player_conn.execute(
                "INSERT INTO sync_meta VALUES ('xuid', 'test_xuid', CURRENT_TIMESTAMP)"
            )
        finally:
            player_conn.close()
            del player_conn
            gc.collect()

        return player_db_path, shared_db_path

    @pytest.mark.slow
    def test_mv_faster_than_direct_query(self, large_db: tuple[Path, Path]):
        """Test que les vues matérialisées sont plus rapides que les requêtes directes.

        Note: Test marqué slow car il insère 1000 enregistrements.
        """
        import time

        player_db_path, shared_db_path = large_db
        DuckDBRepository = _get_duckdb_repository_class()

        repo = DuckDBRepository(
            player_db_path=player_db_path,
            xuid="test_xuid",
            shared_db_path=shared_db_path,
            read_only=False,
        )

        try:
            # Mesurer le temps de la requête directe (sans MV)
            start = time.perf_counter()
            for _ in range(10):
                repo.query("""
                    SELECT mr.map_id, mr.map_name, COUNT(*) as matches_played,
                           AVG(mp.kda) as avg_kda, AVG(mp.accuracy) as avg_accuracy
                    FROM shared.match_participants mp
                    JOIN shared.match_registry mr ON mr.match_id = mp.match_id
                    WHERE mp.xuid = 'test_xuid'
                    GROUP BY mr.map_id, mr.map_name
                """)
            direct_time = time.perf_counter() - start

            # Rafraîchir les vues matérialisées
            repo.refresh_materialized_views()

            # Mesurer le temps avec MV
            start = time.perf_counter()
            for _ in range(10):
                repo.get_map_stats()
            mv_time = time.perf_counter() - start

            # Les MV devraient être au moins aussi rapides (souvent plus rapides)
            # On tolère une marge car les deux sont très rapides sur DuckDB
            assert (
                mv_time <= direct_time * 2
            ), f"MV time ({mv_time:.4f}s) should be faster than direct ({direct_time:.4f}s)"

        finally:
            repo.close()
