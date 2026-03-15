"""Tests pour RosterLoaderMixin.load_friend_match_details().

Valide le comportement du chargement des matchs partagés entre deux joueurs
via de vraies bases DuckDB en mémoire.
"""

from __future__ import annotations

from datetime import datetime, timezone
from pathlib import Path

import duckdb
import polars as pl

from src.data.repositories.duckdb_repo import DuckDBRepository

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

_SELF_XUID = "0001"
_FRIEND_XUID = "0002"
_MATCH_A = "match-aaa"
_MATCH_B = "match-bbb"


def _build_shared_db(path: Path) -> None:
    """Crée une shared DB avec match_registry et match_participants."""
    path.parent.mkdir(parents=True, exist_ok=True)
    conn = duckdb.connect(str(path))
    conn.execute("""
        CREATE TABLE match_registry (
            match_id VARCHAR PRIMARY KEY,
            start_time TIMESTAMP,
            playlist_name VARCHAR,
            pair_name VARCHAR
        )
    """)
    conn.execute("""
        CREATE TABLE match_participants (
            match_id VARCHAR,
            xuid VARCHAR,
            team_id INTEGER,
            outcome INTEGER,
            kills INTEGER,
            deaths INTEGER,
            score INTEGER
        )
    """)
    conn.close()


def _build_player_db(path: Path) -> None:
    """Crée une DB joueur minimale (sync_meta uniquement)."""
    path.parent.mkdir(parents=True, exist_ok=True)
    conn = duckdb.connect(str(path))
    conn.execute("""
        CREATE TABLE sync_meta (
            key VARCHAR PRIMARY KEY,
            value VARCHAR,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    conn.execute(
        "INSERT INTO sync_meta VALUES (?, ?, CURRENT_TIMESTAMP)",
        ["xuid", _SELF_XUID],
    )
    conn.close()


def _make_repo(player_path: Path, shared_path: Path) -> DuckDBRepository:
    return DuckDBRepository(
        player_db_path=player_path,
        xuid=_SELF_XUID,
        shared_db_path=shared_path,
        read_only=False,
    )


def _insert_match(
    conn: duckdb.DuckDBPyConnection,
    match_id: str,
    self_team: int = 0,
    friend_team: int = 1,
    start_time: datetime | None = None,
) -> None:
    """Insère un match dans registry + participants pour les deux joueurs."""
    ts = start_time or datetime(2024, 1, 1, 12, 0, 0, tzinfo=timezone.utc)
    conn.execute(
        "INSERT INTO match_registry VALUES (?, ?, ?, ?)",
        [match_id, ts, "Ranked", "RSSLAYER"],
    )
    conn.execute(
        "INSERT INTO match_participants VALUES (?, ?, ?, ?, ?, ?, ?)",
        [match_id, _SELF_XUID, self_team, 2, 10, 5, 1000],
    )
    conn.execute(
        "INSERT INTO match_participants VALUES (?, ?, ?, ?, ?, ?, ?)",
        [match_id, _FRIEND_XUID, friend_team, 2, 8, 3, 800],
    )


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


class TestLoadFriendMatchDetails:
    """Tests pour DuckDBRepository.load_friend_match_details()."""

    def test_returns_match_row(self, tmp_path: Path) -> None:
        """Retourne un DataFrame avec les colonnes attendues pour un match connu."""
        player_db = tmp_path / "stats.duckdb"
        shared_db = tmp_path / "shared.duckdb"
        _build_player_db(player_db)
        _build_shared_db(shared_db)

        conn = duckdb.connect(str(shared_db))
        _insert_match(conn, _MATCH_A, self_team=0, friend_team=1)
        conn.close()

        repo = _make_repo(player_db, shared_db)
        dfr = repo.load_friend_match_details(_FRIEND_XUID, [_MATCH_A])

        assert not dfr.is_empty()
        assert dfr["match_id"][0] == _MATCH_A
        assert dfr["playlist_name"][0] == "Ranked"
        assert dfr["my_team_id"][0] == 0
        assert dfr["friend_team_id"][0] == 1
        assert dfr["same_team"][0] is False

    def test_same_team_true_when_same_id(self, tmp_path: Path) -> None:
        """same_team=True quand les deux joueurs sont dans la même équipe."""
        player_db = tmp_path / "stats.duckdb"
        shared_db = tmp_path / "shared.duckdb"
        _build_player_db(player_db)
        _build_shared_db(shared_db)

        conn = duckdb.connect(str(shared_db))
        _insert_match(conn, _MATCH_A, self_team=0, friend_team=0)
        conn.close()

        repo = _make_repo(player_db, shared_db)
        dfr = repo.load_friend_match_details(_FRIEND_XUID, [_MATCH_A])

        assert dfr["same_team"][0] is True

    def test_returns_multiple_matches_ordered_by_start_time(self, tmp_path: Path) -> None:
        """Retourne plusieurs matchs triés par start_time ASC."""
        player_db = tmp_path / "stats.duckdb"
        shared_db = tmp_path / "shared.duckdb"
        _build_player_db(player_db)
        _build_shared_db(shared_db)

        conn = duckdb.connect(str(shared_db))
        _insert_match(conn, _MATCH_B, start_time=datetime(2024, 1, 2, tzinfo=timezone.utc))
        _insert_match(conn, _MATCH_A, start_time=datetime(2024, 1, 1, tzinfo=timezone.utc))
        conn.close()

        repo = _make_repo(player_db, shared_db)
        dfr = repo.load_friend_match_details(_FRIEND_XUID, [_MATCH_A, _MATCH_B])

        assert dfr.shape[0] == 2
        assert dfr["match_id"][0] == _MATCH_A  # plus ancien en premier

    def test_empty_when_no_match_ids(self, tmp_path: Path) -> None:
        """Retourne un DataFrame vide (avec schéma) si match_ids est vide."""
        player_db = tmp_path / "stats.duckdb"
        shared_db = tmp_path / "shared.duckdb"
        _build_player_db(player_db)
        _build_shared_db(shared_db)

        repo = _make_repo(player_db, shared_db)
        dfr = repo.load_friend_match_details(_FRIEND_XUID, [])

        assert dfr.is_empty()
        assert "match_id" in dfr.columns
        assert "same_team" in dfr.columns

    def test_empty_schema_when_shared_unavailable(self, tmp_path: Path) -> None:
        """Retourne un DataFrame vide si la shared DB est absente ou en erreur."""
        player_db = tmp_path / "stats.duckdb"
        shared_db = tmp_path / "shared_inexistant.duckdb"  # n'existe pas
        _build_player_db(player_db)

        repo = _make_repo(player_db, shared_db)
        dfr = repo.load_friend_match_details(_FRIEND_XUID, [_MATCH_A])

        assert isinstance(dfr, pl.DataFrame)
        assert dfr.is_empty()

    def test_filters_only_requested_match_ids(self, tmp_path: Path) -> None:
        """Retourne uniquement les matchs demandés, pas tous les matchs partagés."""
        player_db = tmp_path / "stats.duckdb"
        shared_db = tmp_path / "shared.duckdb"
        _build_player_db(player_db)
        _build_shared_db(shared_db)

        conn = duckdb.connect(str(shared_db))
        _insert_match(conn, _MATCH_A)
        _insert_match(conn, _MATCH_B)
        conn.close()

        repo = _make_repo(player_db, shared_db)
        # Demande seulement _MATCH_A
        dfr = repo.load_friend_match_details(_FRIEND_XUID, [_MATCH_A])

        assert dfr.shape[0] == 1
        assert dfr["match_id"][0] == _MATCH_A
