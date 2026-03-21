"""Tests pour src/app/cache_control.py (invalidate_after_sync).

Vérifie :
- S1 : clear_app_caches() est appelé
- S2a : SK.CACHE_BUSTER est incrémenté
- S2b : clés _filters_loaded_* et _filters_db_key_* sont supprimées
- Parité : _resolve_player_xuid renvoie "" pour une DB inconnue (pas d'exception)
"""

from __future__ import annotations

from unittest.mock import MagicMock, patch


def _make_mock_st(store: dict) -> MagicMock:
    """Crée un objet st minimal simulant session_state comme un dict."""
    ss = MagicMock()
    ss.get = lambda k, d=None: store.get(k, d)
    ss.__contains__ = lambda _, k: k in store
    ss.__getitem__ = lambda _, k: store[k]
    ss.__setitem__ = lambda _, k, v: store.__setitem__(k, v)
    ss.__delitem__ = lambda _, k: store.__delitem__(k)
    ss.keys = lambda: list(store.keys())
    mock_st = MagicMock()
    mock_st.session_state = ss
    return mock_st


class TestInvalidateAfterSync:
    """Tests unitaires de invalidate_after_sync."""

    def test_clears_caches(self):
        """clear_app_caches doit être appelé exactement une fois."""
        from src.app.session_keys import SK

        store: dict = {SK.CACHE_BUSTER: 0}
        mock_st = _make_mock_st(store)

        with (
            patch("src.app.cache_control.clear_app_caches") as mock_clear,
            patch("src.app.cache_control.st", mock_st),
        ):
            from src.app.cache_control import invalidate_after_sync

            invalidate_after_sync()

        mock_clear.assert_called_once()

    def test_increments_cache_buster(self):
        """CACHE_BUSTER doit être incrémenté de 1."""
        from src.app.session_keys import SK

        store: dict = {SK.CACHE_BUSTER: 5}
        mock_st = _make_mock_st(store)

        with (
            patch("src.app.cache_control.clear_app_caches"),
            patch("src.app.cache_control.st", mock_st),
        ):
            from src.app.cache_control import invalidate_after_sync

            invalidate_after_sync()

        assert store[SK.CACHE_BUSTER] == 6

    def test_removes_filters_loaded_keys(self):
        """Toutes les clés _filters_loaded_* doivent être supprimées."""
        from src.app.session_keys import SK

        store: dict = {
            SK.CACHE_BUSTER: 0,
            "_filters_loaded_player1": True,
            "_filters_loaded_player2": True,
            "_other_key": "keep",
        }
        mock_st = _make_mock_st(store)

        with (
            patch("src.app.cache_control.clear_app_caches"),
            patch("src.app.cache_control.st", mock_st),
        ):
            from src.app.cache_control import invalidate_after_sync

            invalidate_after_sync()

        assert "_filters_loaded_player1" not in store
        assert "_filters_loaded_player2" not in store
        assert "_other_key" in store  # non touché

    def test_removes_filters_db_key_keys(self):
        """Toutes les clés _filters_db_key_* doivent être supprimées."""
        from src.app.session_keys import SK

        store: dict = {
            SK.CACHE_BUSTER: 0,
            "_filters_db_key_player1": (123, 456, 0, 0),
            "_filters_db_key_player2": (789, 0, 0, 0),
        }
        mock_st = _make_mock_st(store)

        with (
            patch("src.app.cache_control.clear_app_caches"),
            patch("src.app.cache_control.st", mock_st),
        ):
            from src.app.cache_control import invalidate_after_sync

            invalidate_after_sync()

        assert "_filters_db_key_player1" not in store
        assert "_filters_db_key_player2" not in store

    def test_no_crash_when_no_filter_keys(self):
        """Ne doit pas lever d'exception si aucune clé _filters_* n'existe."""
        from src.app.session_keys import SK

        store: dict = {SK.CACHE_BUSTER: 2}
        mock_st = _make_mock_st(store)

        with (
            patch("src.app.cache_control.clear_app_caches"),
            patch("src.app.cache_control.st", mock_st),
        ):
            from src.app.cache_control import invalidate_after_sync

            invalidate_after_sync()  # must not raise

        assert store[SK.CACHE_BUSTER] == 3


class TestResolvePlayerXuid:
    """Parité : _resolve_player_xuid doit renvoyer '' sans crasher."""

    def test_nonexistent_path_returns_empty(self, tmp_path):
        """Une DB inexistante doit retourner '' (pas une exception)."""
        from src.ui._cache_core import _resolve_player_xuid

        result = _resolve_player_xuid(str(tmp_path / "nonexistent.duckdb"))
        assert result == ""

    def test_empty_path_returns_empty(self):
        """Un chemin vide doit retourner '' sans exception."""
        from src.ui._cache_core import _resolve_player_xuid

        result = _resolve_player_xuid("")
        assert result == ""


class TestDbCacheKey:
    """Tests de db_cache_key() — clé de cache 5-tuple avec WAL sentinel."""

    def _make_player_structure(self, tmp_path, gamertag="TestPlayer"):
        """Crée la structure data/players/{gt}/stats.duckdb + data/warehouse/shared_matches_v2.duckdb."""
        import duckdb

        player_dir = tmp_path / "data" / "players" / gamertag
        player_dir.mkdir(parents=True)
        db_path = player_dir / "stats.duckdb"
        conn = duckdb.connect(str(db_path))
        conn.execute("CREATE TABLE t (x INT)")
        conn.close()

        warehouse_dir = tmp_path / "data" / "warehouse"
        warehouse_dir.mkdir(parents=True)
        shared_path = warehouse_dir / "shared_matches_v2.duckdb"
        conn2 = duckdb.connect(str(shared_path))
        conn2.execute("CREATE TABLE t (x INT)")
        conn2.close()

        return db_path, shared_path

    def test_returns_five_tuple(self, tmp_path):
        """db_cache_key doit retourner un tuple de 5 éléments."""
        db_path, _ = self._make_player_structure(tmp_path)

        from src.ui._cache_core import db_cache_key

        key = db_cache_key(str(db_path))
        assert isinstance(key, tuple)
        assert len(key) == 5, f"Attendu 5-tuple, obtenu {len(key)}-tuple"

    def test_mtime_changes_after_write(self, tmp_path):
        """Après une écriture, la clé doit changer (mtime ou size)."""
        import time

        import duckdb

        db_path, _ = self._make_player_structure(tmp_path)

        from src.ui._cache_core import db_cache_key

        key1 = db_cache_key(str(db_path))

        time.sleep(0.05)  # assure un mtime différent
        conn = duckdb.connect(str(db_path))
        conn.execute("INSERT INTO t VALUES (1)")
        conn.close()

        key2 = db_cache_key(str(db_path))
        assert key1 != key2, "La clé doit changer après écriture"

    def test_wal_sentinel_nonzero_when_wal_exists(self, tmp_path):
        """Le wal_sentinel (index 4) est non nul si shared_matches_v2.duckdb.wal existe."""
        db_path, shared_path = self._make_player_structure(tmp_path)

        # Créer le WAL sur shared_matches_v2.duckdb (chemin attendu par db_cache_key)
        wal_path = shared_path.with_suffix(shared_path.suffix + ".wal")
        wal_path.write_bytes(b"fake wal content")

        from src.ui._cache_core import db_cache_key

        key = db_cache_key(str(db_path))
        wal_sentinel = key[4]
        assert (
            wal_sentinel != 0
        ), f"WAL sentinel doit être non nul quand shared .wal existe, obtenu {wal_sentinel}"

    def test_wal_sentinel_zero_without_wal(self, tmp_path):
        """Le wal_sentinel (index 4) est 0 si aucun shared_matches_v2.duckdb.wal n'existe."""
        db_path, shared_path = self._make_player_structure(tmp_path)

        wal_path = shared_path.with_suffix(shared_path.suffix + ".wal")
        if wal_path.exists():
            wal_path.unlink()

        from src.ui._cache_core import db_cache_key

        key = db_cache_key(str(db_path))
        wal_sentinel = key[4]
        assert wal_sentinel == 0, f"WAL sentinel doit être 0 sans .wal, obtenu {wal_sentinel}"

    def test_wal_key_differs_from_no_wal(self, tmp_path):
        """La clé avec WAL doit différer de la clé sans WAL (force cache miss pendant sync)."""
        db_path, shared_path = self._make_player_structure(tmp_path)

        from src.ui._cache_core import db_cache_key

        wal_path = shared_path.with_suffix(shared_path.suffix + ".wal")
        if wal_path.exists():
            wal_path.unlink()
        key_no_wal = db_cache_key(str(db_path))

        wal_path.write_bytes(b"fake wal")
        key_with_wal = db_cache_key(str(db_path))

        assert key_no_wal != key_with_wal, "La présence d'un WAL doit produire une clé différente"

    def test_logs_debug_on_call(self, tmp_path, caplog):
        """db_cache_key émet un log DEBUG avec le nom de fichier et les valeurs."""
        import logging

        db_path, _ = self._make_player_structure(tmp_path)

        with caplog.at_level(logging.DEBUG, logger="src.ui._cache_core"):
            from src.ui._cache_core import db_cache_key

            db_cache_key(str(db_path))

        messages = [r.message for r in caplog.records]
        assert any(
            "db_cache_key" in m and "stats.duckdb" in m for m in messages
        ), f"Log DEBUG db_cache_key attendu, messages={messages}"


class TestInvalidateAfterSyncLogs:
    """invalidate_after_sync doit émettre des logs précis."""

    def test_logs_buster_and_removed_keys(self, caplog):
        """Log INFO doit mentionner le buster et les clés supprimées."""
        import logging

        from src.app.session_keys import SK

        store: dict = {
            SK.CACHE_BUSTER: 3,
            "_filters_loaded_p1": True,
            "_filters_db_key_p1": (1, 2, 3, 4, 0),
        }
        mock_st = _make_mock_st(store)

        with (
            patch("src.app.cache_control.clear_app_caches"),
            patch("src.app.cache_control.st", mock_st),
            caplog.at_level(logging.INFO, logger="src.app.cache_control"),
        ):
            from src.app.cache_control import invalidate_after_sync

            invalidate_after_sync()

        messages = [r.message for r in caplog.records]
        assert any(
            "cache_buster=4" in m for m in messages
        ), f"Log doit contenir cache_buster=4, messages={messages}"
        assert any(
            "_filters_loaded_p1" in m or "_filters_db_key_p1" in m for m in messages
        ), f"Log doit lister les clés supprimées, messages={messages}"
