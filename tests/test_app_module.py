"""Tests pour le module src.app (Phase 1 refactoring)."""

from __future__ import annotations

from src.app.state import (
    AppState,
    PlayerIdentity,
    get_db_cache_key,
)


class TestPlayerIdentity:
    """Tests pour PlayerIdentity."""

    def test_display_name_with_gamertag(self):
        """Test avec gamertag."""
        identity = PlayerIdentity(
            xuid_or_gamertag="Spartan117",
            xuid_fallback="1234567890",
            waypoint_player="Spartan117",
        )
        assert identity.display_name == "Spartan117"

    def test_display_name_with_xuid_only(self):
        """Test avec XUID uniquement."""
        identity = PlayerIdentity(
            xuid_or_gamertag="",
            xuid_fallback="1234567890",
            waypoint_player="",
        )
        assert identity.display_name == "1234567890"

    def test_display_name_empty(self):
        """Test sans identité."""
        identity = PlayerIdentity()
        assert identity.display_name == "Joueur"

    def test_xuid_from_gamertag(self):
        """Test extraction XUID quand gamertag fourni."""
        identity = PlayerIdentity(
            xuid_or_gamertag="Spartan117",
            xuid_fallback="1234567890",
        )
        assert identity.xuid == "1234567890"

    def test_xuid_from_numeric(self):
        """Test extraction XUID quand xuid_or_gamertag est numérique."""
        identity = PlayerIdentity(
            xuid_or_gamertag="1234567890",
            xuid_fallback="",
        )
        assert identity.xuid == "1234567890"


class TestAppState:
    """Tests pour AppState."""

    def test_clear_filters(self):
        """Test réinitialisation des filtres."""
        state = AppState(
            filter_playlists=["Quick Play"],
            filter_modes=["Slayer"],
            filter_maps=["Aquarius"],
        )
        state.clear_filters()
        assert state.filter_playlists == []
        assert state.filter_modes == []
        assert state.filter_maps == []


class TestGetDbCacheKey:
    """Tests pour get_db_cache_key."""

    def test_nonexistent_file(self):
        """Test avec fichier inexistant."""
        result = get_db_cache_key("/nonexistent/path.db")
        assert result is None

    def test_existing_file(self, tmp_path):
        """Test avec fichier existant."""
        db_file = tmp_path / "test.db"
        db_file.write_text("test content")

        result = get_db_cache_key(str(db_file))
        assert result is not None
        assert isinstance(result, tuple)
        # Depuis v5 : (mtime_ns_player, size_player, mtime_ns_shared, size_shared)
        assert len(result) == 4
        assert isinstance(result[0], int)  # mtime_ns_player
        assert isinstance(result[1], int)  # size_player
