"""Tests pour le chargement des antagonistes depuis shared_matches.duckdb.

Vérifie que _load_top_nemeses, _load_top_victims et _load_top_encountered
requêtent correctement killer_victim_pairs et match_participants.
"""

from __future__ import annotations

from pathlib import Path
from unittest.mock import patch

import duckdb
import pytest


@pytest.fixture()
def shared_db(tmp_path: Path) -> Path:
    """Crée une base shared_matches.duckdb temporaire avec données de test."""
    db_path = tmp_path / "shared_matches.duckdb"
    with duckdb.connect(str(db_path)) as conn:
        conn.execute("""
            CREATE TABLE killer_victim_pairs (
                match_id VARCHAR, killer_xuid VARCHAR, killer_gamertag VARCHAR,
                victim_xuid VARCHAR, victim_gamertag VARCHAR,
                kill_count INTEGER, time_ms INTEGER,
                is_validated BOOLEAN, created_at TIMESTAMP
            )
        """)
        conn.execute("""
            CREATE TABLE match_participants (
                match_id VARCHAR, xuid VARCHAR, gamertag VARCHAR,
                team_id INTEGER, outcome INTEGER,
                kills INTEGER DEFAULT 0, deaths INTEGER DEFAULT 0
            )
        """)
        conn.execute("""
            CREATE TABLE xuid_aliases (
                xuid VARCHAR PRIMARY KEY, gamertag VARCHAR
            )
        """)

        conn.execute("""
            CREATE TABLE match_registry (
                match_id VARCHAR PRIMARY KEY, start_time TIMESTAMP
            )
        """)

        # Joueur principal : XUID_ME
        # Adversaire A : me tue 10 fois, je le tue 3 fois (némésis)
        # Adversaire B : je le tue 8 fois, il me tue 2 fois (souffre-douleur)
        # Adversaire C : me tue 5 fois, je le tue 5 fois (neutre)
        conn.execute("""
            INSERT INTO killer_victim_pairs VALUES
            -- A me tue 10 fois sur 2 matchs
            ('m1', 'XUID_A', 'PlayerA', 'XUID_ME', 'Me', 6, 1000, true, now()),
            ('m2', 'XUID_A', 'PlayerA', 'XUID_ME', 'Me', 4, 2000, true, now()),
            -- Je tue A 3 fois
            ('m1', 'XUID_ME', 'Me', 'XUID_A', 'PlayerA', 2, 3000, true, now()),
            ('m2', 'XUID_ME', 'Me', 'XUID_A', 'PlayerA', 1, 4000, true, now()),
            -- Je tue B 8 fois
            ('m1', 'XUID_ME', 'Me', 'XUID_B', 'PlayerB', 5, 5000, true, now()),
            ('m3', 'XUID_ME', 'Me', 'XUID_B', 'PlayerB', 3, 6000, true, now()),
            -- B me tue 2 fois
            ('m1', 'XUID_B', 'PlayerB', 'XUID_ME', 'Me', 2, 7000, true, now()),
            -- C neutre : 5-5
            ('m1', 'XUID_C', 'PlayerC', 'XUID_ME', 'Me', 3, 8000, true, now()),
            ('m2', 'XUID_C', 'PlayerC', 'XUID_ME', 'Me', 2, 9000, true, now()),
            ('m1', 'XUID_ME', 'Me', 'XUID_C', 'PlayerC', 3, 10000, true, now()),
            ('m2', 'XUID_ME', 'Me', 'XUID_C', 'PlayerC', 2, 11000, true, now())
        """)

        conn.execute("""
            INSERT INTO xuid_aliases VALUES
            ('XUID_ME', 'Me'), ('XUID_A', 'PlayerA'),
            ('XUID_B', 'PlayerB'), ('XUID_C', 'PlayerC')
        """)

        # match_participants pour top encountered
        conn.execute("""
            INSERT INTO match_participants VALUES
            ('m1', 'XUID_ME', 'Me', 1, 2, 10, 5),
            ('m1', 'XUID_A', 'PlayerA', 2, 3, 8, 6),
            ('m1', 'XUID_B', 'PlayerB', 2, 3, 4, 7),
            ('m1', 'XUID_C', 'PlayerC', 2, 3, 3, 3),
            ('m2', 'XUID_ME', 'Me', 1, 2, 5, 4),
            ('m2', 'XUID_A', 'PlayerA', 2, 3, 6, 2),
            ('m2', 'XUID_C', 'PlayerC', 2, 3, 2, 5),
            ('m3', 'XUID_ME', 'Me', 1, 2, 3, 1),
            ('m3', 'XUID_B', 'PlayerB', 2, 3, 1, 3)
        """)

        conn.execute("""
            INSERT INTO match_registry VALUES
            ('m1', '2025-01-01 10:00:00'),
            ('m2', '2025-01-02 10:00:00'),
            ('m3', '2025-01-03 10:00:00')
        """)
    return db_path


def _patch_shared_path(shared_db: Path):
    """Retourne un context manager qui patche get_shared_matches_path."""
    return patch(
        "src.ui.pages.career_encounters_data.get_shared_matches_path",
        return_value=shared_db,
    )


class TestLoadTopNemeses:
    """Tests pour _load_top_nemeses (adversaires qui nous tuent le plus)."""

    def test_returns_sorted_by_killed_by(self, shared_db: Path) -> None:
        from src.ui.pages.career_encounters_data import _load_top_nemeses

        with _patch_shared_path(shared_db):
            result = _load_top_nemeses("XUID_ME", limit=10)

        assert len(result) == 3
        # A me tue 10 fois = top nemesis
        assert result[0]["opponent_gamertag"] == "PlayerA"
        assert result[0]["times_killed_by"] == 10
        assert result[0]["times_killed"] == 3
        assert result[0]["net_kills"] == -7

    def test_limit_respected(self, shared_db: Path) -> None:
        from src.ui.pages.career_encounters_data import _load_top_nemeses

        with _patch_shared_path(shared_db):
            result = _load_top_nemeses("XUID_ME", limit=1)

        assert len(result) == 1
        assert result[0]["opponent_gamertag"] == "PlayerA"

    def test_empty_for_unknown_xuid(self, shared_db: Path) -> None:
        from src.ui.pages.career_encounters_data import _load_top_nemeses

        with _patch_shared_path(shared_db):
            result = _load_top_nemeses("XUID_UNKNOWN", limit=10)

        assert result == []


class TestLoadTopVictims:
    """Tests pour _load_top_victims (adversaires qu'on tue le plus)."""

    def test_returns_sorted_by_killed(self, shared_db: Path) -> None:
        from src.ui.pages.career_encounters_data import _load_top_victims

        with _patch_shared_path(shared_db):
            result = _load_top_victims("XUID_ME", limit=10)

        assert len(result) == 3
        # Je tue B 8 fois = top victim
        assert result[0]["opponent_gamertag"] == "PlayerB"
        assert result[0]["times_killed"] == 8
        assert result[0]["times_killed_by"] == 2
        assert result[0]["net_kills"] == 6

    def test_matches_against_correct(self, shared_db: Path) -> None:
        from src.ui.pages.career_encounters_data import _load_top_victims

        with _patch_shared_path(shared_db):
            result = _load_top_victims("XUID_ME", limit=10)

        # B : matchs m1 et m3 → 2 matchs
        player_b = next(r for r in result if r["opponent_gamertag"] == "PlayerB")
        assert player_b["matches_against"] == 2


class TestLoadTopEncountered:
    """Tests pour _load_top_encountered (joueurs les plus croisés)."""

    def test_returns_sorted_by_encounters(self, shared_db: Path) -> None:
        from src.ui.pages.career_encounters_data import _load_top_encountered

        with _patch_shared_path(shared_db):
            result = _load_top_encountered("XUID_ME", limit=10)

        assert len(result) >= 2
        # A et C apparaissent dans 2 matchs, B dans 2 matchs aussi
        gamertags = [r["gamertag"] for r in result]
        assert "PlayerA" in gamertags
        assert "PlayerB" in gamertags
        assert "PlayerC" in gamertags

        # Vérifier les colonnes enrichies
        player_a = next(r for r in result if r["gamertag"] == "PlayerA")
        assert player_a["ally_count"] >= 0
        assert player_a["enemy_count"] >= 0
        assert "winrate_as_ally" in player_a
        assert "kills_dealt" in player_a
        assert "deaths_suffered" in player_a
        assert "last_seen" in player_a

    def test_excludes_self(self, shared_db: Path) -> None:
        from src.ui.pages.career_encounters_data import _load_top_encountered

        with _patch_shared_path(shared_db):
            result = _load_top_encountered("XUID_ME", limit=10)

        xuids = [r["xuid"] for r in result]
        assert "XUID_ME" not in xuids

    def test_excludes_friends_by_xuid(self, shared_db: Path) -> None:
        from src.ui.pages.career_encounters_data import _load_top_encountered

        with _patch_shared_path(shared_db):
            result = _load_top_encountered(
                "XUID_ME",
                limit=10,
                exclude_xuids={"XUID_A"},
            )

        xuids = [r["xuid"] for r in result]
        assert "XUID_A" not in xuids
        assert "XUID_B" in xuids

    def test_excludes_friends_by_gamertag(self, shared_db: Path) -> None:
        from src.ui.pages.career_encounters_data import _load_top_encountered

        with _patch_shared_path(shared_db):
            result = _load_top_encountered(
                "XUID_ME",
                limit=10,
                exclude_xuids={"PlayerB"},
            )

        gamertags = [r["gamertag"] for r in result]
        assert "PlayerB" not in gamertags
        assert "PlayerA" in gamertags
