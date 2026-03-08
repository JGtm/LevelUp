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
