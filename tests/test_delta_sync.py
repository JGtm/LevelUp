"""Tests pour les nouvelles fonctionnalités: delta sync, aliases, sync metadata."""

import sqlite3
import tempfile
from datetime import datetime, timezone
from pathlib import Path

from src.analysis.filters import is_allowed_playlist_name
from src.ui.translations import (
    translate_pair_name,
    translate_playlist_name,
)

# =============================================================================
# Tests Translations
# =============================================================================


class TestTranslatePlaylistName:
    """Tests pour translate_playlist_name (passthrough + UUID detection depuis v6.3)."""

    def test_known_playlist_fr(self):
        """Playlists passthrough — traductions FR via asset_translations (DB), pas JSON."""
        assert translate_playlist_name("Quick Play", lang="fr") == "Quick Play"
        assert translate_playlist_name("Ranked Arena", lang="fr") == "Ranked Arena"
        assert translate_playlist_name("Big Team Battle", lang="fr") == "Big Team Battle"

    def test_known_playlist_en(self):
        """Playlists connues restent identiques en EN."""
        assert translate_playlist_name("Quick Play", lang="en") == "Quick Play"
        assert translate_playlist_name("Ranked Arena", lang="en") == "Ranked Arena"

    def test_unknown_playlist(self):
        """Test avec une playlist inconnue - retourne l'original (passthrough)."""
        assert translate_playlist_name("Unknown Playlist") == "Unknown Playlist"

    def test_none_value(self):
        """Test avec None."""
        assert translate_playlist_name(None) is None

    def test_whitespace_handling(self):
        """Test avec espaces autour — strip appliqué puis passthrough."""
        assert translate_playlist_name("  Quick Play  ") == "Quick Play"

    def test_uuid_returns_inconnue(self):
        """UUID brut → label 'Inconnue' + warning loggé."""
        result = translate_playlist_name("a446725e-b281-414c-a21e-1234567890ab")
        assert result == "Inconnue"

    def test_uuid_en_label(self):
        """UUID brut en anglais → 'Unknown'."""
        result = translate_playlist_name("a446725e-b281-414c-a21e-1234567890ab", lang="en")
        assert result == "Unknown"


class TestTranslatePairName:
    """Tests pour translate_pair_name."""

    def test_exact_match(self):
        """Préfixe redondant supprimé (Arena→Assassin, BTB→BTB)."""
        assert translate_pair_name("Arena:CTF on Aquarius") == "Capture du drapeau"
        assert translate_pair_name("BTB:Slayer on Deadlock") == "Assassin"

    def test_generic_fallback(self):
        """Préfixe Arena redondant → mode seul en FR."""
        assert translate_pair_name("Arena:CTF") == "Capture du drapeau"
        assert translate_pair_name("Arena:King of the Hill") == "Roi de la colline"

    def test_case_normalization(self):
        """Test avec normalisation de casse."""
        # Le système normalise la casse mais peut ne pas trouver de match exact
        result = translate_pair_name("arena:ctf on aquarius")
        # Doit soit trouver la traduction, soit retourner une version normalisée
        assert result is not None

    def test_btb_heavies_preserved(self):
        """BTB Heavies qualifier conservé (qualificatif non redondant en contexte BTB)."""
        result = translate_pair_name("BTB Heavies:CTF on Highpower Heavies")
        assert result == "Heavies : Capture du drapeau"

    def test_none_value(self):
        """Test avec None."""
        assert translate_pair_name(None) is None

    def test_empty_string(self):
        """Test avec chaîne vide."""
        assert translate_pair_name("") is None
        assert translate_pair_name("   ") is None

    def test_unknown_mode_arena_fallback(self):
        """Test fallback pour mode Arena inconnu."""
        # Mode Arena avec carte inconnue devrait utiliser le fallback
        result = translate_pair_name("Arena:Slayer on NewMap2025")
        # Devrait retourner "Arène : Assassin" via fallback
        assert "Assassin" in result or "NewMap2025" in result


class TestTranslationCompleteness:
    """Tests de complétude des traductions modes (metadata.duckdb)."""

    def test_mode_pair_overrides_not_empty(self):
        """Les overrides mode_pair_overrides sont présents dans metadata.duckdb.

        Seuil = 8 (au moins quelques entrées pour FR).
        """
        import duckdb

        from src.utils.paths import get_metadata_db_path

        db_path = get_metadata_db_path()
        if not db_path.exists():
            import pytest

            pytest.skip("metadata.duckdb introuvable")
        try:
            with duckdb.connect(str(db_path), read_only=True) as conn:
                count = conn.execute(
                    "SELECT COUNT(*) FROM mode_pair_overrides WHERE lang='fr'"
                ).fetchone()[0]
        except Exception:
            import pytest

            pytest.skip("table mode_pair_overrides absente (migration requise)")
        assert count >= 8


# =============================================================================
# Tests Delta Sync (mock-based)
# =============================================================================


class TestDeltaSyncLogic:
    """Tests pour la logique de synchronisation delta."""

    def test_sync_meta_table_structure(self):
        """Test que SyncMeta peut être créé avec la bonne structure."""
        with tempfile.NamedTemporaryFile(suffix=".db", delete=False) as f:
            db_path = f.name

        try:
            con = sqlite3.connect(db_path)
            cur = con.cursor()
            cur.execute("""
                CREATE TABLE IF NOT EXISTS SyncMeta (
                    key TEXT PRIMARY KEY,
                    value TEXT,
                    updated_at TEXT
                )
            """)
            con.commit()

            # Insérer une valeur
            now = datetime.now(timezone.utc).isoformat()
            cur.execute(
                "INSERT INTO SyncMeta (key, value, updated_at) VALUES (?, ?, ?)",
                ("last_sync", now, now),
            )
            con.commit()

            # Vérifier la lecture
            cur.execute("SELECT value FROM SyncMeta WHERE key = ?", ("last_sync",))
            row = cur.fetchone()
            assert row is not None
            assert row[0] == now

            con.close()
        finally:
            Path(db_path).unlink(missing_ok=True)

    def test_xuid_aliases_table_structure(self):
        """Test que XuidAliases peut être créé avec la bonne structure."""
        with tempfile.NamedTemporaryFile(suffix=".db", delete=False) as f:
            db_path = f.name

        try:
            con = sqlite3.connect(db_path)
            cur = con.cursor()
            cur.execute("""
                CREATE TABLE IF NOT EXISTS XuidAliases (
                    xuid TEXT PRIMARY KEY,
                    gamertag TEXT NOT NULL,
                    source TEXT DEFAULT 'unknown',
                    updated_at TEXT
                )
            """)
            con.commit()

            # Insérer un alias
            cur.execute(
                "INSERT INTO XuidAliases (xuid, gamertag, source, updated_at) VALUES (?, ?, ?, ?)",
                (
                    "xuid:123456",
                    "TestPlayer",
                    "match_roster",
                    datetime.now(timezone.utc).isoformat(),
                ),
            )
            con.commit()

            # Vérifier la lecture
            cur.execute("SELECT gamertag FROM XuidAliases WHERE xuid = ?", ("xuid:123456",))
            row = cur.fetchone()
            assert row is not None
            assert row[0] == "TestPlayer"

            con.close()
        finally:
            Path(db_path).unlink(missing_ok=True)


# =============================================================================
# Tests Highlight Events
# =============================================================================


class TestHighlightEventsExtraction:
    """Tests pour l'extraction des highlight events."""

    def test_gamertag_extraction_from_json(self):
        """Test extraction de gamertag depuis un JSON d'event."""
        event_json = {
            "Events": [
                {"gamertag": "Player1", "type": "kill"},
                {"gamertag": "Player2", "type": "death"},
            ]
        }
        gamertags = {e.get("gamertag") for e in event_json.get("Events", []) if e.get("gamertag")}
        assert gamertags == {"Player1", "Player2"}

    def test_empty_events(self):
        """Test avec events vides."""
        event_json = {"Events": []}
        gamertags = {e.get("gamertag") for e in event_json.get("Events", []) if e.get("gamertag")}
        assert gamertags == set()

    def test_missing_gamertag(self):
        """Test avec events sans gamertag."""
        event_json = {
            "Events": [
                {"type": "kill"},  # pas de gamertag
                {"gamertag": "Player1", "type": "death"},
            ]
        }
        gamertags = {e.get("gamertag") for e in event_json.get("Events", []) if e.get("gamertag")}
        assert gamertags == {"Player1"}


# =============================================================================
# Tests Filter Logic
# =============================================================================


class TestPlaylistFilters:
    """Tests pour les filtres de playlist."""

    def test_btb_allowed(self):
        """Test que Big Team Battle est autorisé."""
        assert is_allowed_playlist_name("Big Team Battle") is True
        assert is_allowed_playlist_name("Big Team Battle: Refresh") is True

    def test_quick_play_allowed(self):
        """Test que Quick Play est autorisé."""
        assert is_allowed_playlist_name("Quick Play") is True

    def test_ranked_allowed(self):
        """Test que Ranked est autorisé."""
        assert is_allowed_playlist_name("Ranked Arena") is True
        assert is_allowed_playlist_name("Ranked Slayer") is True

    def test_none_returns_false(self):
        """Test que None retourne False."""
        assert is_allowed_playlist_name(None) is False
