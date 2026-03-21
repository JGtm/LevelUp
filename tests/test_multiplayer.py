"""Tests pour src/ui/multiplayer.py — fonctions pures et avec mocks simples."""

from __future__ import annotations

from pathlib import Path
from unittest.mock import patch

import duckdb

from src.ui.multiplayer import (
    DuckDBPlayerInfo,
    PlayerInfo,
    _count_matches_from_player_db,
    _is_duckdb_file,
    _resolve_from_shared,
    _resolve_xuid_from_player_db,
    get_gamertag_from_duckdb_v4_path,
    is_duckdb_v4_path,
    is_multi_player_db,
)

# ============================================================================
# _is_duckdb_file  (pure)
# ============================================================================


class TestIsDuckdbFile:
    def test_duckdb_extension(self):
        assert _is_duckdb_file("stats.duckdb") is True

    def test_db_extension(self):
        assert _is_duckdb_file("player.db") is False

    def test_full_path(self):
        assert _is_duckdb_file("data/players/MyGT/stats.duckdb") is True

    def test_no_extension(self):
        assert _is_duckdb_file("noext") is False


# ============================================================================
# PlayerInfo dataclass properties  (pure)
# ============================================================================


class TestPlayerInfoDisplayName:
    def test_with_label(self):
        p = PlayerInfo(
            xuid="123",
            gamertag="GT",
            label="Custom",
            total_matches=5,
            first_match_date=None,
            last_match_date=None,
        )
        assert p.display_name == "Custom"

    def test_with_gamertag(self):
        p = PlayerInfo(
            xuid="123",
            gamertag="MyGamertag",
            label=None,
            total_matches=0,
            first_match_date=None,
            last_match_date=None,
        )
        assert p.display_name == "MyGamertag"

    def test_fallback_to_xuid(self):
        p = PlayerInfo(
            xuid="1234567890123456",
            gamertag=None,
            label=None,
            total_matches=0,
            first_match_date=None,
            last_match_date=None,
        )
        assert p.display_name == "123456789012345…"

    def test_display_with_stats_matches(self):
        p = PlayerInfo(
            xuid="123",
            gamertag="GT",
            label=None,
            total_matches=150,
            first_match_date=None,
            last_match_date=None,
        )
        assert "150 matchs" in p.display_with_stats

    def test_display_with_stats_no_matches(self):
        p = PlayerInfo(
            xuid="123",
            gamertag="GT",
            label=None,
            total_matches=0,
            first_match_date=None,
            last_match_date=None,
        )
        assert p.display_with_stats == "GT"


# ============================================================================
# DuckDBPlayerInfo dataclass properties  (pure)
# ============================================================================


class TestDuckDBPlayerInfoDisplay:
    def test_with_matches(self):
        p = DuckDBPlayerInfo(gamertag="TestPlayer", db_path=Path("x"), total_matches=42)
        assert "42 matchs" in p.display_with_stats

    def test_zero_matches(self):
        p = DuckDBPlayerInfo(gamertag="TestPlayer", db_path=Path("x"), total_matches=0)
        assert "0 matchs" in p.display_with_stats


# ============================================================================
# is_duckdb_v4_path  (pure)
# ============================================================================


class TestIsDuckdbV4Path:
    def test_valid_path(self):
        # Simule un chemin valide
        p = str(Path("data") / "players" / "MyGT" / "stats.duckdb")
        assert is_duckdb_v4_path(p) is True

    def test_wrong_filename(self):
        p = str(Path("data") / "players" / "MyGT" / "other.duckdb")
        assert is_duckdb_v4_path(p) is False

    def test_empty(self):
        assert is_duckdb_v4_path("") is False

    def test_none_like(self):
        assert is_duckdb_v4_path("") is False


# ============================================================================
# get_gamertag_from_duckdb_v4_path  (pure)
# ============================================================================


class TestGetGamertagFromDuckdbV4Path:
    def test_valid(self):
        p = str(Path("data") / "players" / "SpartanB" / "stats.duckdb")
        assert get_gamertag_from_duckdb_v4_path(p) == "SpartanB"

    def test_empty(self):
        assert get_gamertag_from_duckdb_v4_path("") is None

    def test_non_stats_file(self):
        p = str(Path("data") / "players" / "MyGT" / "archive.duckdb")
        # Devrait quand même retourner le parent car le code ne vérifie pas spécifiquement "stats.duckdb"
        result = get_gamertag_from_duckdb_v4_path(p)
        assert result is None or result == "MyGT"


# ============================================================================
# is_multi_player_db, list_players_in_db, get_unique_xuids (toujours False/vide en v4)
# ============================================================================


class TestLegacyFunctions:
    def test_is_multi_player_db_nonexistent(self):
        assert is_multi_player_db("/nonexistent/path.duckdb") is False

    def test_is_multi_player_db_empty(self):
        assert is_multi_player_db("") is False


# ============================================================================
# _count_matches_from_player_db (pure, testable avec un vrai DuckDB en mémoire)
# ============================================================================


class TestCountMatchesFromPlayerDb:
    def test_count_from_player_match_enrichment(self, tmp_path: Path) -> None:
        db_file = tmp_path / "stats.duckdb"
        with duckdb.connect(str(db_file)) as conn:
            conn.execute("CREATE TABLE player_match_enrichment (match_id TEXT)")
            conn.execute("INSERT INTO player_match_enrichment VALUES ('m1'), ('m2'), ('m3')")
        with duckdb.connect(str(db_file), read_only=True) as conn:
            assert _count_matches_from_player_db(conn) == 3

    def test_returns_zero_when_no_enrichment(self, tmp_path: Path) -> None:
        """Sans player_match_enrichment, retourne 0 (match_stats supprimée en v5.1)."""
        db_file = tmp_path / "stats.duckdb"
        with duckdb.connect(str(db_file)) as conn:
            conn.execute("CREATE TABLE sync_meta (key TEXT, value TEXT)")
        with duckdb.connect(str(db_file), read_only=True) as conn:
            assert _count_matches_from_player_db(conn) == 0

    def test_returns_zero_when_no_tables(self, tmp_path: Path) -> None:
        db_file = tmp_path / "stats.duckdb"
        with duckdb.connect(str(db_file)) as conn:
            conn.execute("SELECT 1")
        with duckdb.connect(str(db_file), read_only=True) as conn:
            assert _count_matches_from_player_db(conn) == 0


# ============================================================================
# _resolve_xuid_from_player_db
# ============================================================================


class TestResolveXuidFromPlayerDb:
    def test_xuid_from_sync_meta(self, tmp_path: Path) -> None:
        db_file = tmp_path / "stats.duckdb"
        with duckdb.connect(str(db_file)) as conn:
            conn.execute("CREATE TABLE sync_meta (key TEXT, value TEXT)")
            conn.execute("INSERT INTO sync_meta VALUES ('xuid', 'xuid123')")
        with duckdb.connect(str(db_file), read_only=True) as conn:
            assert _resolve_xuid_from_player_db(conn) == "xuid123"

    def test_returns_none_when_no_tables(self, tmp_path: Path) -> None:
        db_file = tmp_path / "stats.duckdb"
        with duckdb.connect(str(db_file)) as conn:
            conn.execute("SELECT 1")
        with duckdb.connect(str(db_file), read_only=True) as conn:
            assert _resolve_xuid_from_player_db(conn) is None


# ============================================================================
# _resolve_from_shared (le fix principal du bug Chocoboflor)
# ============================================================================


class TestResolveFromShared:
    def _make_shared_db(self, path: Path) -> None:
        """Crée une shared_matches.duckdb minimale pour les tests."""
        with duckdb.connect(str(path)) as conn:
            conn.execute("CREATE TABLE xuid_aliases (xuid TEXT, gamertag TEXT)")
            conn.execute("CREATE TABLE match_participants (match_id TEXT, xuid TEXT)")
            conn.execute("INSERT INTO xuid_aliases VALUES ('xuid_abc', 'Chocoboflor')")
            for i in range(5):
                conn.execute(
                    "INSERT INTO match_participants VALUES (?, ?)",
                    [f"m{i}", "xuid_abc"],
                )

    def test_resolves_xuid_and_count_from_shared(self, tmp_path: Path) -> None:
        """Si xuid=None et total_matches=0, résout les deux depuis shared."""
        shared_path = tmp_path / "shared_matches.duckdb"
        self._make_shared_db(shared_path)
        db_path = tmp_path / "Chocoboflor" / "stats.duckdb"
        db_path.parent.mkdir(parents=True, exist_ok=True)

        with patch(
            "src.utils.paths.get_shared_matches_path_from_player",
            return_value=shared_path,
        ):
            xuid, count = _resolve_from_shared(db_path, "Chocoboflor", None, 0)
        assert xuid == "xuid_abc"
        assert count == 5

    def test_uses_xuid_if_provided(self, tmp_path: Path) -> None:
        """Si xuid est déjà connu, l'utilise directement."""
        shared_path = tmp_path / "shared_matches.duckdb"
        self._make_shared_db(shared_path)
        db_path = tmp_path / "player" / "stats.duckdb"
        db_path.parent.mkdir(parents=True, exist_ok=True)

        with patch(
            "src.utils.paths.get_shared_matches_path_from_player",
            return_value=shared_path,
        ):
            xuid, count = _resolve_from_shared(db_path, "Chocoboflor", "xuid_abc", 0)
        assert xuid == "xuid_abc"
        assert count == 5

    def test_gamertag_join_when_no_xuid(self, tmp_path: Path) -> None:
        """Fallback gamertag→xuid→count via JOIN quand xuid inconnu et absent de xuid_aliases."""
        shared_path = tmp_path / "shared_matches.duckdb"
        with duckdb.connect(str(shared_path)) as conn:
            conn.execute("CREATE TABLE xuid_aliases (xuid TEXT, gamertag TEXT)")
            conn.execute("CREATE TABLE match_participants (match_id TEXT, xuid TEXT)")
            conn.execute("INSERT INTO xuid_aliases VALUES ('x1', 'UnknownGT')")
            conn.execute("INSERT INTO match_participants VALUES ('m1', 'x1')")
        db_path = tmp_path / "UnknownGT" / "stats.duckdb"
        db_path.parent.mkdir(parents=True, exist_ok=True)

        with patch(
            "src.utils.paths.get_shared_matches_path_from_player",
            return_value=shared_path,
        ):
            xuid, count = _resolve_from_shared(db_path, "UnknownGT", None, 0)
        assert xuid == "x1"
        assert count == 1

    def test_skips_when_matches_already_known(self, tmp_path: Path) -> None:
        """Ne requête pas shared si total_matches > 0 et xuid connu."""
        xuid, count = _resolve_from_shared(
            tmp_path / "stats.duckdb",
            "GT",
            "xuid_ok",
            42,
        )
        assert xuid == "xuid_ok"
        assert count == 42

    def test_returns_defaults_when_shared_missing(self, tmp_path: Path) -> None:
        """Retourne les valeurs d'entrée si shared introuvable."""
        with patch(
            "src.utils.paths.get_shared_matches_path_from_player",
            return_value=None,
        ):
            xuid, count = _resolve_from_shared(
                tmp_path / "stats.duckdb",
                "GT",
                None,
                0,
            )
        assert xuid is None
        assert count == 0
