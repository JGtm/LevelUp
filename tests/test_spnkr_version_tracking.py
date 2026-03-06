"""Tests unitaires pour le tracking de version SPNKr (v5.4).

Couvre :
- Migration : ajout colonne sync_spnkr_version dans match_registry
- Engine : écriture de la version dans match_registry et sync_meta
- Détection : find_matches_with_stale_spnkr()
- CLI : argument --detect-stale-events
- Sidebar : _check_spnkr_version_warning()
- check_env : vérification version SPNKr
"""

from __future__ import annotations

from pathlib import Path
from unittest.mock import patch

import duckdb
import pytest

# =============================================================================
# Fixtures
# =============================================================================


@pytest.fixture
def shared_db(tmp_path: Path) -> duckdb.DuckDBPyConnection:
    """Crée une shared_matches.duckdb temporaire avec le schéma minimal."""
    db_path = tmp_path / "warehouse" / "shared_matches.duckdb"
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
            mode_category VARCHAR,
            is_ranked BOOLEAN DEFAULT FALSE,
            is_firefight BOOLEAN DEFAULT FALSE,
            duration_seconds INTEGER,
            team_0_score SMALLINT,
            team_1_score SMALLINT,
            backfill_completed INTEGER DEFAULT 0,
            participants_loaded BOOLEAN DEFAULT FALSE,
            events_loaded BOOLEAN DEFAULT FALSE,
            medals_loaded BOOLEAN DEFAULT FALSE,
            first_sync_by VARCHAR,
            first_sync_at TIMESTAMP,
            last_updated_at TIMESTAMP,
            player_count SMALLINT DEFAULT 0,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    yield conn
    conn.close()


@pytest.fixture
def shared_db_with_version(shared_db: duckdb.DuckDBPyConnection) -> duckdb.DuckDBPyConnection:
    """shared_db avec la colonne sync_spnkr_version déjà ajoutée."""
    from src.data.sync.migrations import ensure_match_registry_spnkr_version

    ensure_match_registry_spnkr_version(shared_db)
    return shared_db


# =============================================================================
# Tests Migration
# =============================================================================


class TestMigrationSpnkrVersion:
    """Tests pour ensure_match_registry_spnkr_version."""

    def test_adds_column_to_match_registry(self, shared_db: duckdb.DuckDBPyConnection) -> None:
        """La migration ajoute sync_spnkr_version si absente."""
        from src.data.sync.migrations import column_exists, ensure_match_registry_spnkr_version

        assert not column_exists(shared_db, "match_registry", "sync_spnkr_version")
        ensure_match_registry_spnkr_version(shared_db)
        assert column_exists(shared_db, "match_registry", "sync_spnkr_version")

    def test_idempotent(self, shared_db: duckdb.DuckDBPyConnection) -> None:
        """Appeler la migration deux fois ne lève pas d'erreur."""
        from src.data.sync.migrations import ensure_match_registry_spnkr_version

        ensure_match_registry_spnkr_version(shared_db)
        ensure_match_registry_spnkr_version(shared_db)
        # Pas d'exception = OK

    def test_no_op_without_match_registry(self, tmp_path: Path) -> None:
        """Si match_registry n'existe pas, la migration est un no-op."""
        from src.data.sync.migrations import ensure_match_registry_spnkr_version

        db_path = tmp_path / "empty.duckdb"
        conn = duckdb.connect(str(db_path))
        ensure_match_registry_spnkr_version(conn)  # Ne doit pas lever
        conn.close()


# =============================================================================
# Tests Détection stale
# =============================================================================


class TestFindMatchesWithStaleSpnkr:
    """Tests pour find_matches_with_stale_spnkr."""

    def test_no_stale_matches(self, shared_db_with_version: duckdb.DuckDBPyConnection) -> None:
        """Aucun match → résultat vide."""
        from scripts.backfill.detection import find_matches_with_stale_spnkr

        result = find_matches_with_stale_spnkr(shared_db_with_version)
        assert result["stale_versioned"] == []
        assert result["stale_unknown"] == []

    def test_detects_stale_versioned(
        self, shared_db_with_version: duckdb.DuckDBPyConnection
    ) -> None:
        """Détecte un match syncé avec une vieille version de SPNKr."""
        from scripts.backfill.detection import find_matches_with_stale_spnkr

        conn = shared_db_with_version
        conn.execute("""
            INSERT INTO match_registry (match_id, start_time, events_loaded, sync_spnkr_version)
            VALUES ('stale-001', '2026-01-15', TRUE, '0.10.0')
        """)
        conn.execute("""
            INSERT INTO match_registry (match_id, start_time, events_loaded, sync_spnkr_version)
            VALUES ('ok-001', '2026-01-16', TRUE, '0.10.1')
        """)

        result = find_matches_with_stale_spnkr(conn, min_version="0.10.1")
        assert "stale-001" in result["stale_versioned"]
        assert "ok-001" not in result["stale_versioned"]

    def test_detects_stale_unknown(self, shared_db_with_version: duckdb.DuckDBPyConnection) -> None:
        """Détecte un match récent sans version trackée et sans events."""
        from scripts.backfill.detection import find_matches_with_stale_spnkr

        conn = shared_db_with_version
        # Match récent sans events et sans version → stale_unknown
        conn.execute("""
            INSERT INTO match_registry (match_id, start_time, events_loaded, sync_spnkr_version)
            VALUES ('unknown-001', '2026-01-20', FALSE, NULL)
        """)
        # Match ancien sans events → ignoré (avant 2025-12-01)
        conn.execute("""
            INSERT INTO match_registry (match_id, start_time, events_loaded, sync_spnkr_version)
            VALUES ('old-001', '2024-06-01', FALSE, NULL)
        """)

        result = find_matches_with_stale_spnkr(conn, min_version="0.10.1")
        assert "unknown-001" in result["stale_unknown"]
        assert "old-001" not in result["stale_unknown"]

    def test_events_not_loaded_with_version_not_stale(
        self, shared_db_with_version: duckdb.DuckDBPyConnection
    ) -> None:
        """Un match avec events_loaded=FALSE mais version OK n'est pas stale_versioned."""
        from scripts.backfill.detection import find_matches_with_stale_spnkr

        conn = shared_db_with_version
        conn.execute("""
            INSERT INTO match_registry (match_id, start_time, events_loaded, sync_spnkr_version)
            VALUES ('no-events-001', '2026-01-20', FALSE, '0.10.1')
        """)

        result = find_matches_with_stale_spnkr(conn, min_version="0.10.1")
        assert "no-events-001" not in result["stale_versioned"]
        # Pas dans stale_unknown non plus car la version est renseignée
        assert "no-events-001" not in result["stale_unknown"]

    def test_max_matches_limit(self, shared_db_with_version: duckdb.DuckDBPyConnection) -> None:
        """Le paramètre max_matches limite les résultats."""
        from scripts.backfill.detection import find_matches_with_stale_spnkr

        conn = shared_db_with_version
        for i in range(5):
            conn.execute(
                """INSERT INTO match_registry
                   (match_id, start_time, events_loaded, sync_spnkr_version)
                   VALUES (?, ?, TRUE, '0.10.0')""",
                (f"stale-{i:03d}", f"2026-01-{10 + i:02d}"),
            )

        result = find_matches_with_stale_spnkr(conn, min_version="0.10.1", max_matches=3)
        assert len(result["stale_versioned"]) == 3

    def test_works_without_version_column(self, shared_db: duckdb.DuckDBPyConnection) -> None:
        """Fonctionne même si la colonne sync_spnkr_version n'existe pas encore."""
        from scripts.backfill.detection import find_matches_with_stale_spnkr

        conn = shared_db
        conn.execute("""
            INSERT INTO match_registry (match_id, start_time, events_loaded)
            VALUES ('pre-migration-001', '2026-01-20', FALSE)
        """)

        result = find_matches_with_stale_spnkr(conn, min_version="0.10.1")
        # stale_versioned vide car la colonne n'existe pas
        assert result["stale_versioned"] == []
        # Mais stale_unknown peut être rempli
        assert "pre-migration-001" in result["stale_unknown"]


# =============================================================================
# Tests CLI
# =============================================================================


class TestCLIDetectStaleEvents:
    """Tests pour l'argument --detect-stale-events."""

    def test_argument_exists(self) -> None:
        """L'argument --detect-stale-events est reconnu par le parser."""
        from scripts.backfill.cli import create_argument_parser

        parser = create_argument_parser()
        args = parser.parse_args(["--detect-stale-events"])
        assert args.detect_stale_events is True

    def test_argument_default_false(self) -> None:
        """Par défaut, --detect-stale-events est False."""
        from scripts.backfill.cli import create_argument_parser

        parser = create_argument_parser()
        args = parser.parse_args([])
        assert args.detect_stale_events is False


# =============================================================================
# Tests check_env
# =============================================================================


class TestCheckEnvSpnkr:
    """Tests pour la vérification de version SPNKr dans check_env."""

    def test_spnkr_in_expected_versions(self) -> None:
        """SPNKr est listé dans les versions attendues de check_env."""
        from pathlib import Path

        check_env_path = Path(__file__).parent.parent / "scripts" / "check_env.py"
        source = check_env_path.read_text(encoding="utf-8")
        # Vérifier que "spnkr" apparaît dans le dict expected_versions
        assert '"spnkr"' in source or "'spnkr'" in source


# =============================================================================
# Tests sidebar version warning
# =============================================================================


class TestSidebarVersionWarning:
    """Tests pour _check_spnkr_version_warning."""

    def test_no_warning_if_version_ok(self) -> None:
        """Pas de warning si SPNKr >= version minimum."""
        from src.app.sidebar import _check_spnkr_version_warning

        with (
            patch("src.app.sidebar.st") as mock_st,
            patch(
                "importlib.metadata.version",
                return_value="0.10.1",
            ),
        ):
            _check_spnkr_version_warning()
            mock_st.warning.assert_not_called()

    def test_warning_if_version_old(self) -> None:
        """Warning affiché si SPNKr < version minimum."""
        from src.app.sidebar import _check_spnkr_version_warning

        with (
            patch("src.app.sidebar.st") as mock_st,
            patch(
                "importlib.metadata.version",
                return_value="0.10.0",
            ),
        ):
            _check_spnkr_version_warning()
            mock_st.warning.assert_called_once()
            call_args = mock_st.warning.call_args[0][0]
            assert "0.10.0" in call_args
            assert "0.10.1" in call_args

    def test_no_crash_if_spnkr_missing(self) -> None:
        """Pas de crash si SPNKr n'est pas installé."""
        from src.app.sidebar import _check_spnkr_version_warning

        with (
            patch("src.app.sidebar.st") as mock_st,
            patch(
                "importlib.metadata.version",
                side_effect=Exception("not found"),
            ),
        ):
            _check_spnkr_version_warning()  # Ne doit pas lever
            mock_st.warning.assert_not_called()


# =============================================================================
# Tests Engine — version SPNKr
# =============================================================================


class TestEngineSpnkrVersion:
    """Tests pour le tracking de version dans DuckDBSyncEngine."""

    def test_engine_detects_spnkr_version(self, tmp_path: Path) -> None:
        """L'engine détecte la version SPNKr installée."""
        player_db = tmp_path / "players" / "TestGT" / "stats.duckdb"
        player_db.parent.mkdir(parents=True, exist_ok=True)
        shared_db = tmp_path / "warehouse" / "shared_matches.duckdb"
        shared_db.parent.mkdir(parents=True, exist_ok=True)

        with patch("importlib.metadata.version", return_value="0.10.1"):
            engine = DuckDBSyncEngine(
                player_db,
                xuid="1234",
                gamertag="TestGT",
                shared_db_path=shared_db,
            )
            assert engine._spnkr_version == "0.10.1"

    def test_engine_none_if_spnkr_missing(self, tmp_path: Path) -> None:
        """L'engine met None si SPNKr n'est pas installé."""
        player_db = tmp_path / "players" / "TestGT" / "stats.duckdb"
        player_db.parent.mkdir(parents=True, exist_ok=True)

        with patch(
            "importlib.metadata.version",
            side_effect=Exception("not found"),
        ):
            engine = DuckDBSyncEngine(
                player_db,
                xuid="1234",
                gamertag="TestGT",
            )
            assert engine._spnkr_version is None


# Import nécessaire en bas pour éviter les circular imports
from src.data.sync.engine import DuckDBSyncEngine  # noqa: E402
