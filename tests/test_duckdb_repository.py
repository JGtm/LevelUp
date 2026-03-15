"""
Tests pour le DuckDBRepository.

Ce module teste le nouveau repository DuckDB natif (architecture v4).
"""

from __future__ import annotations

from pathlib import Path

import duckdb
import pytest

# Skip si DuckDB n'est pas disponible
pytest.importorskip("duckdb")


class TestReleaseDbConnections:
    """Tests pour release_db_connections et le retry _get_connection."""

    def test_release_closes_matching_connection(self, tmp_path: Path) -> None:
        """release_db_connections ferme la connexion du repo ciblé."""
        from src.data.repositories.duckdb_repo import (
            DuckDBRepository,
            release_db_connections,
        )

        db_file = tmp_path / "player.duckdb"
        duckdb.connect(str(db_file)).close()  # créer le fichier

        repo = DuckDBRepository(str(db_file), xuid="x1")
        conn = repo._get_connection()
        assert conn is not None

        closed = release_db_connections(db_file)
        assert closed == 1
        assert repo._connection is None
        assert len(repo._attached_dbs) == 0

    def test_release_does_not_close_other_db(self, tmp_path: Path) -> None:
        """release_db_connections n'affecte pas les connexions vers d'autres fichiers."""
        from src.data.repositories.duckdb_repo import (
            DuckDBRepository,
            release_db_connections,
        )

        db_a = tmp_path / "a.duckdb"
        db_b = tmp_path / "b.duckdb"
        duckdb.connect(str(db_a)).close()
        duckdb.connect(str(db_b)).close()

        repo_a = DuckDBRepository(str(db_a), xuid="x1")
        repo_b = DuckDBRepository(str(db_b), xuid="x2")
        repo_a._get_connection()
        repo_b._get_connection()

        closed = release_db_connections(db_a)
        assert closed == 1
        assert repo_a._connection is None
        # repo_b doit rester ouvert
        assert repo_b._connection is not None

    def test_repo_reconnects_after_release(self, tmp_path: Path) -> None:
        """Le repo se reconnecte automatiquement après un release."""
        from src.data.repositories.duckdb_repo import (
            DuckDBRepository,
            release_db_connections,
        )

        db_file = tmp_path / "player.duckdb"
        duckdb.connect(str(db_file)).close()

        repo = DuckDBRepository(str(db_file), xuid="x1")
        repo._get_connection()

        release_db_connections(db_file)
        assert repo._connection is None

        # Reconnexion automatique
        new_conn = repo._get_connection()
        assert new_conn is not None

    def test_media_indexer_after_release(self, tmp_path: Path) -> None:
        """Après release, un MediaIndexer peut ouvrir la même DB en écriture."""
        from src.data.repositories.duckdb_repo import (
            DuckDBRepository,
            release_db_connections,
        )

        db_file = tmp_path / "player.duckdb"
        duckdb.connect(str(db_file)).close()

        # Ouvrir en read_only (comme le fait le cache Streamlit)
        repo = DuckDBRepository(str(db_file), xuid="x1", read_only=True)
        repo._get_connection()

        # Sans release, ouvrir en read_write échoue
        with pytest.raises(Exception, match="(?i)different configuration"):
            duckdb.connect(str(db_file), read_only=False)

        # Après release, ça fonctionne
        release_db_connections(db_file)
        write_conn = duckdb.connect(str(db_file), read_only=False)
        write_conn.execute("CREATE TABLE IF NOT EXISTS test_t (id INT)")
        write_conn.close()


class TestWriteLease:
    """Tests pour le mécanisme db_write_lease / wait_for_write_leases_cleared.

    Reproduit la race condition « MediaIndexer (read_write) vs switch joueur
    (read_only) » qui causait le stuck au chargement.
    """

    def test_write_lease_blocks_get_connection(self, tmp_path: Path) -> None:
        """_get_connection() attend que le write lease soit libéré."""
        import threading
        import time

        from src.data.repositories._write_lease import db_write_lease
        from src.data.repositories.duckdb_repo import DuckDBRepository

        db_file = tmp_path / "player.duckdb"
        duckdb.connect(str(db_file)).close()

        results: list[str] = []

        def writer():
            with db_write_lease(db_file), duckdb.connect(str(db_file), read_only=False) as conn:
                conn.execute("CREATE TABLE IF NOT EXISTS t (x INT)")
                conn.commit()
                results.append("write_start")
                time.sleep(0.2)  # Simule le travail du MediaIndexer
                results.append("write_end")

        t = threading.Thread(target=writer)
        t.start()
        time.sleep(0.05)  # Laisser le writer s'installer

        # _get_connection() doit attendre que le writer ait fini
        repo = DuckDBRepository(str(db_file), xuid="x1", read_only=True)
        conn = repo._get_connection()
        results.append("read_opened")
        conn  # noqa: B018 — juste pour vérifier qu'on a bien une connexion

        t.join(timeout=2.0)

        # L'ordre doit être : write_start → write_end → read_opened
        assert results == ["write_start", "write_end", "read_opened"], results

    def test_write_lease_released_on_exception(self, tmp_path: Path) -> None:
        """Le write lease est libéré même si une exception survient dans le bloc."""
        from src.data.repositories._write_lease import (
            _write_leases,
            db_write_lease,
        )

        db_file = tmp_path / "player.duckdb"
        path_key = str(db_file.resolve())

        with pytest.raises(RuntimeError, match="test"), db_write_lease(db_file):
            assert _write_leases[path_key] == 1
            raise RuntimeError("test")

        assert _write_leases[path_key] == 0

    def test_wait_returns_immediately_when_no_lease(self, tmp_path: Path) -> None:
        """wait_for_write_leases_cleared retourne immédiatement si pas de lease actif."""
        import time

        from src.data.repositories._write_lease import wait_for_write_leases_cleared

        db_file = tmp_path / "player.duckdb"
        start = time.monotonic()
        wait_for_write_leases_cleared(db_file)
        elapsed = time.monotonic() - start
        assert elapsed < 0.1, f"Devrait être quasi-instantané, got {elapsed:.3f}s"

    def test_wait_timeout_is_bounded(self, tmp_path: Path) -> None:
        """L'attente est bornée par timeout même si un write lease reste actif."""
        import threading
        import time

        from src.data.repositories._write_lease import (
            db_write_lease,
            wait_for_write_leases_cleared,
        )

        db_file = tmp_path / "player.duckdb"
        duckdb.connect(str(db_file)).close()

        def writer_holds_lease() -> None:
            with db_write_lease(db_file):
                time.sleep(0.3)

        t = threading.Thread(target=writer_holds_lease)
        t.start()
        time.sleep(0.05)

        start = time.monotonic()
        wait_for_write_leases_cleared(db_file, timeout=0.1)
        elapsed = time.monotonic() - start
        t.join(timeout=1.0)

        assert elapsed < 0.2, f"L'attente devrait être bornée, got {elapsed:.3f}s"

    def test_no_conflict_with_write_lease_pattern(self, tmp_path: Path) -> None:
        """Reproduit exactement le bug du switch joueur : plus de conflit avec le lease."""
        import threading
        import time

        from src.data.repositories._write_lease import db_write_lease
        from src.data.repositories.duckdb_repo import DuckDBRepository, release_db_connections

        db_file = tmp_path / "player.duckdb"
        duckdb.connect(str(db_file)).close()

        errors: list[Exception] = []

        def simulate_media_indexer():
            """Simule le MediaIndexer ouvrant la DB en écriture."""
            with db_write_lease(db_file):
                release_db_connections(db_file)
                with duckdb.connect(str(db_file), read_only=False) as conn:
                    conn.execute("CREATE TABLE IF NOT EXISTS media_files (id INT)")
                    conn.commit()
                    time.sleep(0.15)

        def simulate_player_switch():
            """Simule le switch de joueur qui tente d'ouvrir en read_only."""
            try:
                repo = DuckDBRepository(str(db_file), xuid="x1", read_only=True)
                repo._get_connection()
            except Exception as e:
                errors.append(e)

        t_indexer = threading.Thread(target=simulate_media_indexer)
        t_indexer.start()
        time.sleep(0.05)  # S'assurer que le writer est déjà en cours

        t_switch = threading.Thread(target=simulate_player_switch)
        t_switch.start()

        t_indexer.join(timeout=2.0)
        t_switch.join(timeout=2.0)

        assert not errors, f"Le switch joueur a échoué : {errors}"


class TestDuckDBRepositoryImport:
    """Tests d'import et de structure."""

    def test_import_duckdb_repository(self):
        """Vérifie que DuckDBRepository peut être importé."""
        from src.data.repositories.duckdb_repo import DuckDBRepository

        assert DuckDBRepository is not None

    def test_import_from_factory(self):
        """Vérifie que le mode DUCKDB est disponible dans le factory."""
        from src.data.repositories.factory import RepositoryMode

        assert hasattr(RepositoryMode, "DUCKDB")
        assert RepositoryMode.DUCKDB.value == "duckdb"

    def test_import_get_repository_from_profile(self):
        """Vérifie que get_repository_from_profile est disponible."""
        from src.data.repositories.factory import get_repository_from_profile

        assert callable(get_repository_from_profile)

    def test_import_load_db_profiles(self):
        """Vérifie que load_db_profiles est disponible."""
        from src.data.repositories.factory import load_db_profiles

        assert callable(load_db_profiles)


class TestDuckDBRepositoryInit:
    """Tests d'initialisation du repository."""

    def test_init_with_nonexistent_db(self):
        """Vérifie que l'init ne fait pas d'erreur avec une DB inexistante."""
        from src.data.repositories.duckdb_repo import DuckDBRepository

        repo = DuckDBRepository(
            player_db_path="nonexistent.duckdb",
            xuid="123456789",
        )

        assert repo.xuid == "123456789"
        assert repo.db_path == "nonexistent.duckdb"

    def test_init_with_gamertag(self):
        """Vérifie que le gamertag est stocké."""
        from src.data.repositories.duckdb_repo import DuckDBRepository

        repo = DuckDBRepository(
            player_db_path="test.duckdb",
            xuid="123456789",
            gamertag="TestPlayer",
        )

        assert repo._gamertag == "TestPlayer"

    def test_connection_error_on_missing_db(self):
        """Vérifie l'erreur si la DB n'existe pas lors de la connexion."""
        from src.data.repositories.duckdb_repo import DuckDBRepository

        repo = DuckDBRepository(
            player_db_path="nonexistent.duckdb",
            xuid="123456789",
        )

        with pytest.raises(FileNotFoundError):
            repo._get_connection()


class TestRepositoryModeSelection:
    """Tests de sélection du mode de repository."""

    def test_mode_duckdb_from_string(self):
        """Vérifie la conversion string -> RepositoryMode."""
        from src.data.repositories.factory import RepositoryMode

        mode = RepositoryMode("duckdb")
        assert mode == RepositoryMode.DUCKDB

    def test_get_repository_with_duckdb_mode(self):
        """Vérifie que get_repository crée un DuckDBRepository."""
        from src.data.repositories.duckdb_repo import DuckDBRepository
        from src.data.repositories.factory import RepositoryMode, get_repository

        repo = get_repository(
            "test.duckdb",
            "123456789",
            mode=RepositoryMode.DUCKDB,
        )

        assert isinstance(repo, DuckDBRepository)

    def test_get_repository_with_duckdb_string(self):
        """Vérifie que get_repository fonctionne avec 'duckdb' en string."""
        from src.data.repositories.duckdb_repo import DuckDBRepository
        from src.data.repositories.factory import get_repository

        repo = get_repository(
            "test.duckdb",
            "123456789",
            mode="duckdb",
        )

        assert isinstance(repo, DuckDBRepository)


class TestLoadDbProfiles:
    """Tests pour load_db_profiles."""

    def test_load_existing_profiles(self):
        """Vérifie le chargement de db_profiles.json."""
        from src.data.repositories.factory import load_db_profiles

        profiles = load_db_profiles()

        # Doit contenir version et profiles
        assert "version" in profiles
        assert "profiles" in profiles

    def test_profiles_version_2(self):
        """Vérifie que la version est >= 2.0 pour DuckDB."""
        from src.data.repositories.factory import load_db_profiles

        profiles = load_db_profiles()
        version = profiles.get("version", "1.0")

        assert version >= "2.0", "db_profiles.json devrait être en version 2.0+"

    def test_profiles_contain_duckdb_paths(self):
        """Vérifie que les profils pointent vers des fichiers .duckdb."""
        from src.data.repositories.factory import load_db_profiles

        profiles = load_db_profiles()

        for gamertag, profile in profiles.get("profiles", {}).items():
            db_path = profile.get("db_path", "")
            assert db_path.endswith(".duckdb"), f"{gamertag} devrait avoir un chemin .duckdb"


def _has_player_match_enrichment_table() -> bool:
    """Vérifie que la DB de test contient la table player_match_enrichment."""
    db_path = Path("data/players/JGtm/stats.duckdb")
    if not db_path.exists():
        return False
    try:
        import duckdb as _ddb

        c = _ddb.connect(str(db_path), read_only=True)
        r = c.execute(
            "SELECT COUNT(*) FROM information_schema.tables "
            "WHERE table_schema='main' AND table_name='player_match_enrichment'"
        ).fetchone()
        c.close()
        return bool(r and r[0] > 0)
    except Exception:
        return False


@pytest.mark.skipif(
    not _has_player_match_enrichment_table(),
    reason="Base de données de test non disponible ou vide (player_match_enrichment manquante)",
)
class TestDuckDBRepositoryWithRealData:
    """Tests avec vraies données (si disponibles)."""

    @pytest.fixture
    def repo(self):
        """Crée un repository avec des vraies données."""
        from src.data.repositories.duckdb_repo import DuckDBRepository

        repo = DuckDBRepository(
            player_db_path="data/players/JGtm/stats.duckdb",
            xuid="2533274823110022",
            gamertag="JGtm",
        )
        yield repo
        repo.close()

    def test_load_matches_returns_list(self, repo):
        """Vérifie que load_matches retourne une liste."""
        matches = repo.load_matches()
        assert isinstance(matches, list)

    def test_load_matches_returns_match_rows(self, repo):
        """Vérifie que les matchs sont des MatchRow."""
        from src.data.domain.models.stats import MatchRow

        matches = repo.load_matches()
        if matches:
            assert isinstance(matches[0], MatchRow)

    def test_get_match_count(self, repo):
        """Vérifie que get_match_count retourne un entier."""
        count = repo.get_match_count()
        assert isinstance(count, int)
        assert count >= 0

    def test_is_hybrid_available(self, repo):
        """Vérifie que is_hybrid_available fonctionne (architecture hybride supprimée en v4)."""
        # Note: L'architecture hybride a été supprimée en v4.
        # is_hybrid_available retourne maintenant False par défaut.
        result = repo.is_hybrid_available()
        assert isinstance(result, bool)

    def test_get_storage_info(self, repo):
        """Vérifie les infos de stockage."""
        info = repo.get_storage_info()

        assert info["type"] == "duckdb"
        assert "tables" in info
        assert info["file_size_mb"] >= 0

    def test_get_sync_metadata(self, repo):
        """Vérifie les métadonnées de sync."""
        meta = repo.get_sync_metadata()

        assert meta["storage_type"] == "duckdb"
        assert meta["player_xuid"] == "2533274823110022"
        assert "total_matches" in meta


@pytest.mark.skipif(
    not _has_player_match_enrichment_table(),
    reason="Base de données de test non disponible ou vide (player_match_enrichment manquante)",
)
class TestDuckDBRepositoryQueries:
    """Tests des requêtes avancées."""

    @pytest.fixture
    def repo(self):
        """Crée un repository avec des vraies données."""
        from src.data.repositories.duckdb_repo import DuckDBRepository

        repo = DuckDBRepository(
            player_db_path="data/players/JGtm/stats.duckdb",
            xuid="2533274823110022",
        )
        yield repo
        repo.close()

    def test_query_raw_sql(self, repo):
        """Vérifie que query() fonctionne."""
        results = repo.query("SELECT COUNT(*) as cnt FROM player_match_enrichment")

        assert len(results) == 1
        assert "cnt" in results[0]

    def test_query_with_params(self, repo):
        """Vérifie query() avec paramètres."""
        results = repo.query(
            "SELECT COUNT(*) as cnt FROM player_match_enrichment WHERE match_id IS NOT NULL",
            None,
        )

        assert len(results) == 1

    def test_query_df_returns_polars(self, repo):
        """Vérifie que query_df retourne un DataFrame Polars."""
        import polars as pl

        df = repo.query_df(
            "SELECT match_id, performance_score FROM player_match_enrichment LIMIT 5"
        )

        assert isinstance(df, pl.DataFrame)
        assert "match_id" in df.columns
        assert "performance_score" in df.columns

    def test_load_top_teammates(self, repo):
        """Vérifie list_top_teammates."""
        teammates = repo.list_top_teammates(limit=5)

        assert isinstance(teammates, list)
        # Peut être vide si pas de données
