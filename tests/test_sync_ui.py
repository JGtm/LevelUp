"""Tests pour le module src/ui/sync.py.

Ce fichier teste les fonctionnalités de synchronisation UI :
- Détection DuckDB vs SQLite
- Extraction du gamertag depuis le chemin DuckDB v4
- Fonction sync_player_duckdb (ex-_sync_duckdb_player, supprimé Phase G.3)
- Fonction sync_all_players avec support DuckDB
"""

from __future__ import annotations

from pathlib import Path
from unittest.mock import patch

import pytest

# =============================================================================
# Tests de détection de chemins
# =============================================================================


class TestIsSpnkrDbPath:
    """Tests pour is_spnkr_db_path."""

    def test_sqlite_db_legacy_refused(self):
        """Test que les bases SQLite legacy .db sont refusées."""
        from src.ui.sync import is_spnkr_db_path

        # SQLite .db est maintenant refusé
        assert is_spnkr_db_path("data/spnkr_SpartanC.db") is False
        assert is_spnkr_db_path("spnkr_player.db") is False
        assert is_spnkr_db_path("data/halo_merged.db") is False

    def test_duckdb_v4_stats(self, tmp_path):
        """Test avec une base DuckDB v4 (stats.duckdb)."""
        from src.ui.sync import is_spnkr_db_path

        # Créer la structure de dossiers
        players_dir = tmp_path / "data" / "players" / "SpartanC"
        players_dir.mkdir(parents=True)
        db_path = players_dir / "stats.duckdb"
        db_path.touch()

        assert is_spnkr_db_path(str(db_path)) is True

    def test_non_db_file(self):
        """Test avec un fichier non-DB."""
        from src.ui.sync import is_spnkr_db_path

        assert is_spnkr_db_path("data/config.json") is False
        assert is_spnkr_db_path("data/readme.md") is False
        assert is_spnkr_db_path("data/image.png") is False

    def test_empty_path(self):
        """Test avec chemin vide."""
        from src.ui.sync import is_spnkr_db_path

        assert is_spnkr_db_path("") is False


# =============================================================================
# Tests d'extraction du gamertag depuis le chemin
# =============================================================================


class TestExtractGamertagFromDuckDBPath:
    """Tests pour l'extraction du gamertag depuis le chemin DuckDB v4."""

    def test_extract_gamertag_from_valid_path(self):
        """Test extraction depuis un chemin valide."""
        path = "data/players/SpartanC/stats.duckdb"
        p = Path(path)

        # Le gamertag devrait être le nom du parent directory
        assert p.name == "stats.duckdb"
        assert p.parent.name == "SpartanC"
        assert p.parent.parent.name == "players"

    def test_extract_gamertag_with_spaces(self):
        """Test extraction avec gamertag contenant des caractères spéciaux."""
        path = "data/players/Player With Spaces/stats.duckdb"
        p = Path(path)

        assert p.parent.name == "Player With Spaces"

    def test_invalid_path_structure(self):
        """Test avec une structure de chemin invalide."""
        # Le fichier n'est pas dans data/players/{gamertag}/
        path = "data/other/stats.duckdb"
        p = Path(path)

        # Ce n'est pas une structure valide DuckDB v4
        assert p.parent.parent.name != "players"


# =============================================================================
# Tests de sync_player_duckdb (mocked)
# =============================================================================


class TestSyncDuckDBPlayer:
    """Tests pour sync_player_duckdb (ex-_sync_duckdb_player, supprimé G.3)."""

    def test_sync_missing_tokens(self):
        """Test sync avec tokens manquants — mock sync_player_duckdb."""
        with patch("src.ui.sync.sync_player_duckdb") as mock_sync:
            mock_sync.return_value = (False, "Tokens SPNKr manquants.")
            ok, msg = mock_sync(gamertag="TestPlayer", xuid="", max_matches=10, delta=True)
            assert ok is False
            assert "Tokens" in msg or "manquants" in msg

    def test_sync_success_with_new_matches(self):
        """Test sync réussie avec nouveaux matchs."""
        with patch("src.ui.sync.sync_player_duckdb") as mock_sync:
            mock_sync.return_value = (True, "5 nouveau(x) match(s) synchronisé(s).")
            ok, msg = mock_sync(gamertag="TestPlayer", xuid="", max_matches=100, delta=True)
            assert ok is True
            assert "nouveau" in msg

    def test_sync_already_up_to_date(self):
        """Test sync quand déjà à jour."""
        with patch("src.ui.sync.sync_player_duckdb") as mock_sync:
            mock_sync.return_value = (True, "À jour (150 matchs).")
            ok, msg = mock_sync(gamertag="TestPlayer", xuid="", delta=True)
            assert ok is True
            assert "À jour" in msg


# =============================================================================
# Tests de sync_all_players
# =============================================================================


class TestSyncAllPlayers:
    """Tests pour sync_all_players avec support DuckDB."""

    @pytest.fixture
    def mock_duckdb_db(self, tmp_path):
        """Crée une base DuckDB mock."""
        import uuid

        import duckdb

        players_dir = tmp_path / "data" / "players" / f"MockPlayer_{uuid.uuid4().hex[:8]}"
        players_dir.mkdir(parents=True)
        db_path = players_dir / "stats.duckdb"

        conn = duckdb.connect(str(db_path))
        try:
            conn.execute("""
                CREATE TABLE IF NOT EXISTS match_stats (
                    match_id VARCHAR PRIMARY KEY
                )
            """)
            conn.execute("""
                CREATE TABLE IF NOT EXISTS xuid_aliases (
                    xuid VARCHAR PRIMARY KEY,
                    gamertag VARCHAR,
                    last_seen TIMESTAMP
                )
            """)
            conn.execute("""
                INSERT INTO xuid_aliases VALUES ('999888777666', 'MockPlayer', CURRENT_TIMESTAMP)
            """)
        finally:
            conn.close()

        return str(db_path)

    def test_detects_duckdb_path(self, mock_duckdb_db):
        """Test que sync_all_players détecte correctement un chemin DuckDB."""
        assert mock_duckdb_db.endswith(".duckdb")

        # Vérifier la structure du chemin
        p = Path(mock_duckdb_db)
        assert p.name == "stats.duckdb"
        assert p.parent.parent.name == "players"

    def test_extracts_gamertag_from_duckdb_path(self, mock_duckdb_db):
        """Test que le gamertag est correctement extrait du chemin."""
        p = Path(mock_duckdb_db)
        gamertag = p.parent.name

        # Le nom peut contenir un UUID pour éviter les conflits
        assert gamertag.startswith("MockPlayer")

    def test_extracts_xuid_from_xuid_aliases(self, mock_duckdb_db):
        """Test que le XUID est correctement extrait de xuid_aliases."""
        import duckdb

        conn = duckdb.connect(mock_duckdb_db, read_only=True)
        result = conn.execute(
            "SELECT xuid FROM xuid_aliases ORDER BY last_seen DESC LIMIT 1"
        ).fetchone()
        conn.close()

        assert result is not None
        assert result[0] == "999888777666"

    def test_sync_all_players_uses_duckdb_sync(self, mock_duckdb_db):
        """Test que sync_all_players utilise sync_player_duckdb pour DuckDB (G.3)."""
        from src.ui.sync import sync_all_players

        with patch("src.ui.sync.sync_player_duckdb") as mock_sync:
            mock_sync.return_value = (True, "Sync OK")

            ok, msg = sync_all_players(
                db_path=mock_duckdb_db,
                max_matches=10,
                delta=True,
            )

            # Vérifier que sync_player_duckdb a été appelé
            mock_sync.assert_called_once()
            call_args = mock_sync.call_args
            # Le gamertag peut contenir un UUID pour éviter les conflits
            assert call_args.kwargs["gamertag"].startswith("MockPlayer")


# =============================================================================
# Tests de SyncResult.errors (vs .error)
# =============================================================================


class TestSyncResultErrors:
    """Tests pour vérifier que SyncResult utilise .errors (liste)."""

    def test_sync_result_has_errors_list(self):
        """Vérifie que SyncResult a un attribut errors (liste)."""
        from src.data.sync.models import SyncResult

        result = SyncResult()

        # Doit avoir .errors, pas .error
        assert hasattr(result, "errors")
        assert isinstance(result.errors, list)
        assert not hasattr(result, "error")

    def test_sync_result_with_errors(self):
        """Test SyncResult avec des erreurs."""
        from src.data.sync.models import SyncResult

        result = SyncResult(errors=["Erreur 1", "Erreur 2"])

        assert len(result.errors) == 2
        assert "Erreur 1" in result.errors
        assert result.success is False

    def test_sync_result_without_errors(self):
        """Test SyncResult sans erreurs."""
        from src.data.sync.models import SyncResult

        result = SyncResult(matches_inserted=5)

        assert len(result.errors) == 0
        assert result.success is True


# =============================================================================
# Tests pour refresh_spnkr_db_via_api (SQLite legacy)
# =============================================================================


class TestRefreshSpnkrDbViaApi:
    """Tests pour refresh_spnkr_db_via_api (script SQLite legacy)."""

    def test_missing_script(self, tmp_path):
        """Test quand le script d'import n'existe pas."""
        from src.ui.sync import refresh_spnkr_db_via_api

        # Le script n'existe pas dans tmp_path
        ok, msg = refresh_spnkr_db_via_api(
            db_path=str(tmp_path / "test.db"),
            player="TestPlayer",
            match_type="matchmaking",
            max_matches=10,
            rps=5,
            repo_root=tmp_path,
        )

        assert ok is False
        assert "introuvable" in msg

    def test_empty_player(self, tmp_path):
        """Test avec joueur vide."""
        from src.ui.sync import refresh_spnkr_db_via_api

        # Créer le script mock
        scripts_dir = tmp_path / "scripts"
        scripts_dir.mkdir()
        (scripts_dir / "spnkr_import_db.py").touch()

        ok, msg = refresh_spnkr_db_via_api(
            db_path=str(tmp_path / "test.db"),
            player="",
            match_type="matchmaking",
            max_matches=10,
            rps=5,
            repo_root=tmp_path,
        )

        assert ok is False
        assert "joueur" in msg.lower()


# =============================================================================
# Tests d'intégration légère
# =============================================================================


class TestSyncIntegration:
    """Tests d'intégration pour le module sync."""

    def test_duckdb_path_detection_and_gamertag_extraction(self, tmp_path):
        """Test complet : détection DuckDB + extraction gamertag."""
        import duckdb

        from src.ui.sync import is_spnkr_db_path

        # Créer une structure DuckDB v4 complète
        gamertag = "IntegrationTestPlayer"
        players_dir = tmp_path / "data" / "players" / gamertag
        players_dir.mkdir(parents=True)
        db_path = players_dir / "stats.duckdb"

        # Créer la base avec les tables nécessaires
        conn = duckdb.connect(str(db_path))
        conn.execute("CREATE TABLE match_stats (match_id VARCHAR)")
        conn.execute(
            "CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR, last_seen TIMESTAMP)"
        )
        conn.execute(
            f"INSERT INTO xuid_aliases VALUES ('123456789', '{gamertag}', CURRENT_TIMESTAMP)"
        )
        conn.close()

        # Test 1: is_spnkr_db_path détecte correctement
        assert is_spnkr_db_path(str(db_path)) is True

        # Test 2: Le gamertag peut être extrait du chemin
        p = Path(db_path)
        assert p.parent.name == gamertag
        assert p.parent.parent.name == "players"

        # Test 3: Le XUID peut être extrait de la table
        conn = duckdb.connect(str(db_path), read_only=True)
        result = conn.execute("SELECT xuid FROM xuid_aliases LIMIT 1").fetchone()
        conn.close()
        assert result is not None
        assert result[0] == "123456789"

    def test_sync_all_players_path_parsing(self, tmp_path):
        """Test que sync_all_players parse correctement différents chemins."""
        import duckdb

        # Cas 1: Chemin DuckDB v4 standard
        gamertag = "TestGamer"
        players_dir = tmp_path / "data" / "players" / gamertag
        players_dir.mkdir(parents=True)
        db_path = players_dir / "stats.duckdb"

        conn = duckdb.connect(str(db_path))
        conn.execute("CREATE TABLE match_stats (match_id VARCHAR)")
        conn.execute(
            "CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR, last_seen TIMESTAMP)"
        )
        conn.close()

        # Vérifier que le chemin est bien parsé
        assert str(db_path).endswith(".duckdb")
        p = Path(db_path)
        assert p.name == "stats.duckdb"
        assert p.parent.name == gamertag
        assert p.parent.parent.name == "players"


# =============================================================================
# Tests de régression — sync mode (handle conflict shared_matches_v2.duckdb)
# =============================================================================


def _make_player_db(tmp_path, gamertag: str):
    """Crée une DB joueur minimale pour les tests sync mode."""
    import duckdb

    players_dir = tmp_path / "data" / "players" / gamertag
    players_dir.mkdir(parents=True)
    db_path = players_dir / "stats.duckdb"
    conn = duckdb.connect(str(db_path))
    conn.execute("CREATE TABLE sync_meta (key VARCHAR, value VARCHAR)")
    conn.close()
    return db_path


class TestSyncModeHandleConflict:
    """Régression : sync_player_duckdb_async doit activer le sync mode.

    Bug original : le DuckDBRepository de Streamlit conservait un ATTACH de
    shared_matches_v2.duckdb, causant un « Unique file handle conflict ».

    Architecture implémentée (docs/) :
    - ``_manage_sync_mode=True`` (défaut) : gère activate/deactivate + guard réentrance
    - ``_manage_sync_mode=False`` : le caller gère, pas d'activate/deactivate
    - Guard réentrance : si ``_sync_mode.is_set()`` → _already_active=True → pas de
      double activate/deactivate même si _manage_sync_mode=True
    """

    def test_sync_player_duckdb_async_activates_sync_mode(self, tmp_path):
        """sync_player_duckdb_async active et désactive le sync mode quand _manage_sync_mode=True."""
        import asyncio
        import threading

        _make_player_db(tmp_path, "TestSync")
        calls: list[str] = []
        inactive_event = threading.Event()  # non-set = sync mode inactif

        with (
            patch(
                "src.ui._sync_duckdb_ops._activate_sync_mode",
                side_effect=lambda: calls.append("activate"),
            ),
            patch(
                "src.ui._sync_duckdb_ops._deactivate_sync_mode",
                side_effect=lambda: calls.append("deactivate"),
            ),
            patch("src.data.repositories.duckdb_repo._sync_mode", inactive_event),
            patch("src.data.sync.DuckDBSyncEngine", side_effect=RuntimeError("test stop")),
        ):
            from src.ui._sync_duckdb_ops import sync_player_duckdb_async

            ok, msg = asyncio.run(
                sync_player_duckdb_async(gamertag="TestSync", xuid="123", repo_root=tmp_path)
            )

        assert ok is False
        assert "activate" in calls, "sync mode doit être activé (_manage_sync_mode=True par défaut)"
        assert "deactivate" in calls, "sync mode doit être désactivé dans finally"
        assert calls.index("activate") < calls.index("deactivate"), (
            "activate doit précéder deactivate"
        )

    def test_reentrant_sync_mode_not_deactivated_early(self, tmp_path):
        """Guard réentrance : _sync_mode.is_set() → _already_active=True → ni activate ni deactivate."""
        import asyncio
        import threading

        _make_player_db(tmp_path, "TestReentrant")
        calls: list[str] = []
        active_event = threading.Event()
        active_event.set()  # sync mode déjà actif → _already_active=True

        with (
            patch(
                "src.ui._sync_duckdb_ops._activate_sync_mode",
                side_effect=lambda: calls.append("activate"),
            ),
            patch(
                "src.ui._sync_duckdb_ops._deactivate_sync_mode",
                side_effect=lambda: calls.append("deactivate"),
            ),
            patch("src.data.repositories.duckdb_repo._sync_mode", active_event),
            patch("src.data.sync.DuckDBSyncEngine", side_effect=RuntimeError("test stop")),
        ):
            from src.ui._sync_duckdb_ops import sync_player_duckdb_async

            ok, msg = asyncio.run(
                sync_player_duckdb_async(gamertag="TestReentrant", xuid="456", repo_root=tmp_path)
            )

        assert ok is False
        assert "activate" not in calls, "ne doit PAS réactiver si _sync_mode.is_set()"
        assert "deactivate" not in calls, "ne doit PAS désactiver si _already_active=True"

    def test_manage_sync_mode_false_skips_activate_deactivate(self, tmp_path):
        """_manage_sync_mode=False : ni activate ni deactivate, quelle que soit la situation."""
        import asyncio

        _make_player_db(tmp_path, "TestManageFalse")
        calls: list[str] = []

        with (
            patch(
                "src.ui._sync_duckdb_ops._activate_sync_mode",
                side_effect=lambda: calls.append("activate"),
            ),
            patch(
                "src.ui._sync_duckdb_ops._deactivate_sync_mode",
                side_effect=lambda: calls.append("deactivate"),
            ),
            patch("src.data.sync.DuckDBSyncEngine", side_effect=RuntimeError("test stop")),
        ):
            from src.ui._sync_duckdb_ops import sync_player_duckdb_async

            ok, msg = asyncio.run(
                sync_player_duckdb_async(
                    gamertag="TestManageFalse",
                    xuid="999",
                    repo_root=tmp_path,
                    _manage_sync_mode=False,
                )
            )

        assert ok is False
        assert calls == [], (
            f"_manage_sync_mode=False doit court-circuiter activate/deactivate, got {calls}"
        )

    def test_engine_closed_on_error(self, tmp_path):
        """engine.close() est appelé dans le finally de _run_sync_engine même si sync_delta lève."""
        import asyncio
        import threading

        _make_player_db(tmp_path, "TestClose")
        close_calls: list[str] = []

        class MockEngine:
            async def sync_delta(self, options):
                raise RuntimeError("boom pendant sync")

            def close(self):
                close_calls.append("close")

        # Event pré-set → _already_active=True pour isoler le test du sync_mode
        active_event = threading.Event()
        active_event.set()

        with (
            patch("src.ui._sync_duckdb_ops._activate_sync_mode"),
            patch("src.ui._sync_duckdb_ops._deactivate_sync_mode"),
            patch("src.data.repositories.duckdb_repo._sync_mode", active_event),
            patch("src.data.sync.DuckDBSyncEngine", return_value=MockEngine()),
        ):
            from src.ui._sync_duckdb_ops import sync_player_duckdb_async

            ok, msg = asyncio.run(
                sync_player_duckdb_async(gamertag="TestClose", xuid="789", repo_root=tmp_path)
            )

        assert ok is False
        assert "boom" in msg
        assert "close" in close_calls, (
            "engine.close() doit être appelé dans finally de _run_sync_engine"
        )

    def test_sync_player_duckdb_async_logs_start_and_result(self, tmp_path, caplog):
        """sync_player_duckdb_async émet un log INFO au démarrage (gamertag, mode)."""
        import asyncio
        import logging

        _make_player_db(tmp_path, "TestLog")

        with (
            patch("src.data.sync.DuckDBSyncEngine", side_effect=RuntimeError("stop")),
            caplog.at_level(logging.INFO, logger="src.ui._sync_duckdb_ops"),
        ):
            from src.ui._sync_duckdb_ops import sync_player_duckdb_async

            asyncio.run(sync_player_duckdb_async(gamertag="TestLog", xuid="42", repo_root=tmp_path))

        messages = [r.message for r in caplog.records]
        assert any("TestLog" in m and "démarrage" in m for m in messages), (
            f"Log démarrage attendu, messages={messages}"
        )

    def test_sync_all_players_duckdb_wraps_sync_mode(self, tmp_path):
        """sync_all_players_duckdb active le sync mode UNE FOIS autour de toute la boucle."""
        import json

        import duckdb

        profiles = {
            "version": "2.1",
            "profiles": {
                "Player1": {"db_path": "data/players/Player1/stats.duckdb", "xuid": "111"},
                "Player2": {"db_path": "data/players/Player2/stats.duckdb", "xuid": "222"},
            },
        }
        (tmp_path / "db_profiles.json").write_text(json.dumps(profiles), encoding="utf-8")

        for gt in ("Player1", "Player2"):
            d = tmp_path / "data" / "players" / gt
            d.mkdir(parents=True)
            conn = duckdb.connect(str(d / "stats.duckdb"))
            conn.execute("CREATE TABLE sync_meta (key VARCHAR, value VARCHAR)")
            conn.close()

        calls: list[str] = []

        with (
            patch(
                "src.ui._sync_duckdb_ops._activate_sync_mode",
                side_effect=lambda: calls.append("activate"),
            ),
            patch(
                "src.ui._sync_duckdb_ops._deactivate_sync_mode",
                side_effect=lambda: calls.append("deactivate"),
            ),
            patch(
                "src.ui._sync_duckdb_ops.sync_player_duckdb_async",
                return_value=(True, "OK"),
            ),
        ):
            from src.ui.sync import sync_all_players_duckdb

            ok, msg = sync_all_players_duckdb(repo_root=tmp_path)

        assert ok is True
        assert calls.count("activate") == 1, (
            f"activate appelé {calls.count('activate')} fois (attendu 1)"
        )
        assert calls.count("deactivate") == 1, (
            f"deactivate appelé {calls.count('deactivate')} fois (attendu 1)"
        )
