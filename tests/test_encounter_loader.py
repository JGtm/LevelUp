"""Tests pour src/data/repositories/_encounter_loader.py.

Vérifie que :
- Les participants orphelins (absents de match_registry) ne sont PAS comptés.
- Les rencontres normales (participant + entrée registry) sont correctement agrégées.
- Les comptages ally/enemy et kills/deaths sont exacts.
- Le retour est vide pour des cibles inconnues ou une liste vide.
"""

from __future__ import annotations

from datetime import datetime
from pathlib import Path

import duckdb
import pytest

from src.data.repositories._encounter_loader import load_encounter_stats

# ---------------------------------------------------------------------------
# Schéma minimal — reproduit uniquement les tables utilisées par le SQL de
# _encounter_loader._build_encounter_sql
# ---------------------------------------------------------------------------

_SCHEMA = """
CREATE TABLE match_registry (
    match_id VARCHAR PRIMARY KEY,
    start_time TIMESTAMP NOT NULL
);
CREATE TABLE match_participants (
    match_id VARCHAR NOT NULL,
    xuid VARCHAR NOT NULL,
    gamertag VARCHAR,
    team_id INTEGER,
    outcome INTEGER,
    PRIMARY KEY (match_id, xuid)
);
CREATE TABLE highlight_events (
    id INTEGER PRIMARY KEY,
    match_id VARCHAR,
    event_type VARCHAR,
    xuid VARCHAR,
    gamertag VARCHAR
);
CREATE TABLE xuid_aliases (
    xuid VARCHAR PRIMARY KEY,
    gamertag VARCHAR NOT NULL
);
CREATE TABLE killer_victim_pairs (
    match_id VARCHAR NOT NULL,
    killer_xuid VARCHAR NOT NULL,
    victim_xuid VARCHAR NOT NULL,
    kill_count INTEGER DEFAULT 1
);
"""

_ME = "xuid-me-0001"
_TARGET = "xuid-target-0002"
_START = datetime(2025, 1, 15, 20, 0, 0)


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


def _create_shared_db(path: Path) -> None:
    """Crée un shared_matches_v2.duckdb minimal dans *path*."""
    path.parent.mkdir(parents=True, exist_ok=True)
    with duckdb.connect(str(path)) as conn:
        for stmt in _SCHEMA.strip().split(";"):
            s = stmt.strip()
            if s:
                conn.execute(s)


def _player_db_path(tmp_path: Path) -> Path:
    """Chemin fictif du stats.duckdb joueur, cohérent avec _get_shared_db_path."""
    # _get_shared_db_path remonte 3 niveaux puis descend warehouse/shared_matches_v2.duckdb
    return tmp_path / "players" / "testplayer" / "stats.duckdb"


def _shared_db_path(tmp_path: Path) -> Path:
    return tmp_path / "warehouse" / "shared_matches_v2.duckdb"


@pytest.fixture()
def shared_db(tmp_path: Path) -> Path:
    """Crée et retourne le chemin du shared_matches_v2.duckdb de test."""
    path = _shared_db_path(tmp_path)
    _create_shared_db(path)
    return path


@pytest.fixture()
def player_db(tmp_path: Path) -> str:
    """Retourne le chemin (str) fictif du stats.duckdb joueur."""
    return str(_player_db_path(tmp_path))


# ---------------------------------------------------------------------------
# Helpers d'insertion
# ---------------------------------------------------------------------------


def _insert_registry(conn: duckdb.DuckDBPyConnection, match_id: str) -> None:
    conn.execute(
        "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
        (match_id, _START),
    )


def _insert_participant(  # noqa: PLR0913
    conn: duckdb.DuckDBPyConnection,
    match_id: str,
    xuid: str,
    team_id: int = 0,
    outcome: int = 2,
    gamertag: str = "Player",
) -> None:
    conn.execute(
        "INSERT INTO match_participants (match_id, xuid, gamertag, team_id, outcome) "
        "VALUES (?, ?, ?, ?, ?)",
        (match_id, xuid, gamertag, team_id, outcome),
    )


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


class TestLoadEncounterStatsEmptyInput:
    """Cas avec entrées vides ou absentes."""

    def test_empty_target_list_returns_empty_df(self, shared_db: Path, player_db: str) -> None:
        result = load_encounter_stats(_ME, [], player_db)
        assert result.is_empty()

    def test_empty_self_xuid_returns_empty_df(self, shared_db: Path, player_db: str) -> None:
        result = load_encounter_stats("", [_TARGET], player_db)
        assert result.is_empty()

    def test_unknown_target_returns_empty_df(self, shared_db: Path, player_db: str) -> None:
        with duckdb.connect(str(shared_db)) as conn:
            _insert_registry(conn, "match-001")
            _insert_participant(conn, "match-001", _ME, team_id=0)
            # _TARGET n'est PAS inséré

        result = load_encounter_stats(_ME, [_TARGET], player_db)
        assert result.is_empty()


class TestOrphanedParticipantNotCounted:
    """Vérifie que les participants sans entrée match_registry sont exclus (INNER JOIN)."""

    def test_orphan_participant_not_in_result(self, shared_db: Path, player_db: str) -> None:
        """Un participant dans match_participants sans match_registry ne doit pas être compté."""
        with duckdb.connect(str(shared_db)) as conn:
            # match sans entrée registry (orphelin)
            _insert_participant(conn, "orphan-match", _ME, team_id=0, outcome=2)
            _insert_participant(conn, "orphan-match", _TARGET, team_id=1, outcome=3)
            # PAS d'INSERT dans match_registry

        result = load_encounter_stats(_ME, [_TARGET], player_db)
        assert result.is_empty(), (
            f"Un participant orphelin ne devrait pas être compté, "
            f"mais on a obtenu {len(result)} ligne(s)"
        )

    def test_mixed_valid_and_orphan_counts_only_valid(
        self, shared_db: Path, player_db: str
    ) -> None:
        """Parmi 2 matchs (1 avec registry, 1 sans), seul le valide est comptabilisé."""
        with duckdb.connect(str(shared_db)) as conn:
            # Match valide (registry + participants)
            _insert_registry(conn, "match-valid")
            _insert_participant(conn, "match-valid", _ME, team_id=0, outcome=2)
            _insert_participant(conn, "match-valid", _TARGET, team_id=1, outcome=3)

            # Match orphelin (participants sans registry)
            _insert_participant(conn, "match-orphan", _ME, team_id=0, outcome=2)
            _insert_participant(conn, "match-orphan", _TARGET, team_id=1, outcome=3)

        result = load_encounter_stats(_ME, [_TARGET], player_db)
        assert len(result) == 1
        assert result["total_encounters"][0] == 1, (
            "Seul le match avec entrée registry doit être comptabilisé"
        )


class TestEncounterCounting:
    """Vérification du comptage et de la classification ally/enemy."""

    def test_single_encounter_enemy(self, shared_db: Path, player_db: str) -> None:
        with duckdb.connect(str(shared_db)) as conn:
            _insert_registry(conn, "match-001")
            _insert_participant(conn, "match-001", _ME, team_id=0, outcome=2)
            _insert_participant(conn, "match-001", _TARGET, team_id=1, outcome=3)

        result = load_encounter_stats(_ME, [_TARGET], player_db)
        assert len(result) == 1
        row = result.row(0, named=True)
        assert row["total_encounters"] == 1
        assert row["ally_count"] == 0
        assert row["enemy_count"] == 1

    def test_single_encounter_ally(self, shared_db: Path, player_db: str) -> None:
        with duckdb.connect(str(shared_db)) as conn:
            _insert_registry(conn, "match-001")
            _insert_participant(conn, "match-001", _ME, team_id=0, outcome=2)
            # Même équipe que _ME
            _insert_participant(conn, "match-001", _TARGET, team_id=0, outcome=2)

        result = load_encounter_stats(_ME, [_TARGET], player_db)
        assert len(result) == 1
        row = result.row(0, named=True)
        assert row["total_encounters"] == 1
        assert row["ally_count"] == 1
        assert row["enemy_count"] == 0

    def test_multiple_matches_accumulate_correctly(self, shared_db: Path, player_db: str) -> None:
        with duckdb.connect(str(shared_db)) as conn:
            for i in range(1, 4):
                mid = f"match-00{i}"
                _insert_registry(conn, mid)
                _insert_participant(conn, mid, _ME, team_id=0, outcome=2)
                # Alternance ally/enemy
                team = 0 if i % 2 == 0 else 1
                _insert_participant(conn, mid, _TARGET, team_id=team, outcome=3)

        result = load_encounter_stats(_ME, [_TARGET], player_db)
        assert len(result) == 1
        row = result.row(0, named=True)
        assert row["total_encounters"] == 3
        # match 1 → enemy (team 1), match 2 → ally (team 0), match 3 → enemy (team 1)
        assert row["ally_count"] == 1
        assert row["enemy_count"] == 2


class TestKillsDeaths:
    """Vérification des kills/deaths via killer_victim_pairs."""

    def test_kills_and_deaths_counted(self, shared_db: Path, player_db: str) -> None:
        with duckdb.connect(str(shared_db)) as conn:
            _insert_registry(conn, "match-001")
            _insert_participant(conn, "match-001", _ME, team_id=0, outcome=2)
            _insert_participant(conn, "match-001", _TARGET, team_id=1, outcome=3)

            conn.execute(
                "INSERT INTO killer_victim_pairs (match_id, killer_xuid, victim_xuid, kill_count) "
                "VALUES (?, ?, ?, ?)",
                ("match-001", _ME, _TARGET, 3),
            )
            conn.execute(
                "INSERT INTO killer_victim_pairs (match_id, killer_xuid, victim_xuid, kill_count) "
                "VALUES (?, ?, ?, ?)",
                ("match-001", _TARGET, _ME, 1),
            )

        result = load_encounter_stats(_ME, [_TARGET], player_db)
        row = result.row(0, named=True)
        assert row["kills_dealt"] == 3
        assert row["deaths_suffered"] == 1

    def test_orphan_match_kills_not_counted(self, shared_db: Path, player_db: str) -> None:
        """Les kills d'un match orphelin ne doivent pas être comptés (INNER JOIN my_matches)."""
        with duckdb.connect(str(shared_db)) as conn:
            # Match orphelin (pas dans registry)
            _insert_participant(conn, "orphan", _ME, team_id=0)
            _insert_participant(conn, "orphan", _TARGET, team_id=1)
            conn.execute(
                "INSERT INTO killer_victim_pairs (match_id, killer_xuid, victim_xuid, kill_count) "
                "VALUES ('orphan', ?, ?, 5)",
                (_ME, _TARGET),
            )

        result = load_encounter_stats(_ME, [_TARGET], player_db)
        # Aucun match valide → résultat vide
        assert result.is_empty()


class TestResultColumns:
    """Vérifie que toutes les colonnes attendues sont présentes."""

    def test_expected_columns_present(self, shared_db: Path, player_db: str) -> None:
        with duckdb.connect(str(shared_db)) as conn:
            _insert_registry(conn, "match-001")
            _insert_participant(conn, "match-001", _ME, team_id=0)
            _insert_participant(conn, "match-001", _TARGET, team_id=1, gamertag="EnemyPlayer")

        result = load_encounter_stats(_ME, [_TARGET], player_db)
        expected = {
            "xuid",
            "gamertag",
            "total_encounters",
            "ally_count",
            "enemy_count",
            "winrate_as_ally",
            "winrate_vs_enemy",
            "kills_dealt",
            "deaths_suffered",
            "last_seen",
        }
        assert set(result.columns) == expected
