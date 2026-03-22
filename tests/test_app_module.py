"""Tests pour le module src.app (Phase 1 refactoring)."""

from __future__ import annotations

from src.app.state import AppState


class TestAppState:
    """Tests pour AppState."""

    def test_default_values(self):
        """Test valeurs par défaut."""
        state = AppState()
        assert state.db_path == ""
        assert state.xuid_input == ""
        assert state.filter_playlists == []
        assert state.current_page == "Accueil"
