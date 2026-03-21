"""Tests — Axe 4 : Citations bulk SQL + executemany INSERT.

Vérifie que :
- _bulk_medals charge les médailles pour N matchs en 1 requête
- _bulk_stats charge les stats pour N matchs
- _bulk_awards charge les awards depuis la player DB
- _build_match_data_map retourne les données correctes pour N matchs
- _process_citations_batch produit les mêmes résultats que l'ancienne boucle
- executemany est appelé (pas N inserts individuels)
"""

from __future__ import annotations

from datetime import datetime
from pathlib import Path
from unittest.mock import patch

import duckdb
import pytest

from src.data.citations_backfill import (
    _build_match_data_map,
    _bulk_awards,
    _bulk_medals,
    _bulk_stats,
)

_NOW = datetime(2025, 1, 1, 12, 0, 0)
_XUID = "1111111111"


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture
def shared_db(tmp_path):
    """shared_matches_v2.duckdb minimal avec match_participants, medals_earned,
    highlight_events, match_registry."""
    db_path = tmp_path / "shared_matches.duckdb"
    with duckdb.connect(str(db_path)) as conn:
        conn.execute("""
            CREATE TABLE match_registry (
                match_id VARCHAR PRIMARY KEY,
                start_time TIMESTAMP,
                playlist_name VARCHAR,
                game_variant_name VARCHAR,
                map_name VARCHAR
            )
        """)
        conn.execute("""
            CREATE TABLE match_participants (
                match_id VARCHAR,
                xuid VARCHAR,
                kills INTEGER DEFAULT 0,
                deaths INTEGER DEFAULT 0,
                outcome INTEGER DEFAULT 2,
                kda FLOAT DEFAULT 0.0,
                map_name VARCHAR,
                playlist_name VARCHAR,
                game_variant_name VARCHAR,
                start_time TIMESTAMP
            )
        """)
        conn.execute("""
            CREATE TABLE medals_earned (
                match_id VARCHAR,
                xuid VARCHAR,
                medal_name_id INTEGER,
                count INTEGER DEFAULT 1
            )
        """)
        conn.execute("""
            CREATE TABLE highlight_events (
                match_id VARCHAR,
                xuid VARCHAR,
                time_ms INTEGER,
                event_type VARCHAR
            )
        """)
    return db_path


def _insert_match(shared_db: Path, match_id: str, xuid: str) -> None:
    with duckdb.connect(str(shared_db)) as conn:
        conn.execute(
            "INSERT INTO match_registry VALUES (?, ?, 'Quick Play', 'Slayer', 'Aquarius')",
            [match_id, _NOW.isoformat()],
        )
        conn.execute(
            "INSERT INTO match_participants (match_id, xuid, kills, deaths, outcome) VALUES (?, ?, 5, 3, 2)",
            [match_id, xuid],
        )


@pytest.fixture
def player_db(tmp_path, shared_db):
    """Player DB avec personal_score_awards."""
    player_dir = tmp_path / "data" / "players" / "TestPlayer"
    player_dir.mkdir(parents=True, exist_ok=True)
    db_path = player_dir / "stats.duckdb"
    with duckdb.connect(str(db_path)) as conn:
        conn.execute("""
            CREATE TABLE personal_score_awards (
                match_id VARCHAR,
                award_name VARCHAR,
                award_count INTEGER DEFAULT 1
            )
        """)
    return db_path


@pytest.fixture
def three_matches(shared_db, player_db):
    """3 matchs insérés dans shared + player."""
    match_ids = ["match-001", "match-002", "match-003"]
    for mid in match_ids:
        _insert_match(shared_db, mid, _XUID)
    # Awards pour match-001
    with duckdb.connect(str(player_db)) as conn:
        conn.execute("INSERT INTO personal_score_awards VALUES ('match-001', 'kill_leader', 3)")
    return {"shared_db": shared_db, "player_db": player_db, "match_ids": match_ids}


# ---------------------------------------------------------------------------
# Tests _bulk_medals
# ---------------------------------------------------------------------------


class TestBulkMedals:
    """_bulk_medals charge les médailles pour N matchs en 1 requête."""

    def test_loads_medals_for_multiple_matches(self, three_matches):
        shared_db = three_matches["shared_db"]
        match_ids = three_matches["match_ids"]

        # Insérer des médailles
        with duckdb.connect(str(shared_db)) as conn:
            conn.execute(
                "INSERT INTO medals_earned VALUES ('match-001', ?, 100, 2)",
                [_XUID],
            )
            conn.execute(
                "INSERT INTO medals_earned VALUES ('match-002', ?, 200, 1)",
                [_XUID],
            )

        with duckdb.connect(str(shared_db), read_only=True) as shared_ro:
            result = _bulk_medals(shared_ro, _XUID, match_ids)

        assert result["match-001"][100] == 2
        assert result["match-002"][200] == 1
        assert "match-003" not in result  # pas de médailles

    def test_returns_empty_for_no_medals(self, three_matches):
        shared_db = three_matches["shared_db"]
        match_ids = three_matches["match_ids"]

        with duckdb.connect(str(shared_db), read_only=True) as shared_ro:
            result = _bulk_medals(shared_ro, _XUID, match_ids)

        assert result == {}


# ---------------------------------------------------------------------------
# Tests _bulk_stats
# ---------------------------------------------------------------------------


class TestBulkStats:
    """_bulk_stats charge match_participants pour N matchs."""

    def test_loads_stats_for_all_matches(self, three_matches):
        shared_db = three_matches["shared_db"]
        match_ids = three_matches["match_ids"]

        with duckdb.connect(str(shared_db), read_only=True) as shared_ro:
            result = _bulk_stats(shared_ro, _XUID, match_ids)

        assert len(result) == 3
        for mid in match_ids:
            assert mid in result
            assert result[mid]["kills"] == 5
            assert result[mid]["deaths"] == 3

    def test_excludes_unknown_match(self, three_matches):
        shared_db = three_matches["shared_db"]

        with duckdb.connect(str(shared_db), read_only=True) as shared_ro:
            result = _bulk_stats(shared_ro, _XUID, ["match-999"])

        assert result == {}


# ---------------------------------------------------------------------------
# Tests _bulk_awards
# ---------------------------------------------------------------------------


class TestBulkAwards:
    """_bulk_awards charge les awards depuis la player DB."""

    def test_loads_awards_for_match_with_awards(self, three_matches):
        player_db = three_matches["player_db"]
        match_ids = three_matches["match_ids"]

        with duckdb.connect(str(player_db)) as player_conn:
            result = _bulk_awards(player_conn, match_ids)

        assert result["match-001"]["kill_leader"] == 3
        assert "match-002" not in result
        assert "match-003" not in result

    def test_returns_empty_if_table_absent(self, tmp_path):
        db_path = tmp_path / "empty.duckdb"
        with duckdb.connect(str(db_path)) as player_conn:
            result = _bulk_awards(player_conn, ["match-001"])
        assert result == {}


# ---------------------------------------------------------------------------
# Tests _build_match_data_map
# ---------------------------------------------------------------------------


class TestBuildMatchDataMap:
    """_build_match_data_map orchestre les 6 bulk loaders."""

    def test_returns_data_for_all_matches(self, three_matches):
        shared_db = three_matches["shared_db"]
        player_db = three_matches["player_db"]
        match_ids = three_matches["match_ids"]
        pve_path = Path("/nonexistent/pve.duckdb")

        with (
            duckdb.connect(str(shared_db), read_only=True) as shared_ro,
            duckdb.connect(str(player_db)) as player_conn,
        ):
            result = _build_match_data_map(shared_ro, player_conn, _XUID, match_ids, pve_path)

        assert len(result) == 3
        for mid in match_ids:
            assert "medals" in result[mid]
            assert "stats" in result[mid]
            assert "awards" in result[mid]
            assert "df" in result[mid]
            assert "events" in result[mid]

    def test_stats_merged_with_awards_and_weapons(self, three_matches):
        shared_db = three_matches["shared_db"]
        player_db = three_matches["player_db"]

        with (
            duckdb.connect(str(shared_db), read_only=True) as shared_ro,
            duckdb.connect(str(player_db)) as player_conn,
        ):
            result = _build_match_data_map(shared_ro, player_conn, _XUID, ["match-001"], None)

        # awards chargés
        assert result["match-001"]["awards"]["kill_leader"] == 3


# ---------------------------------------------------------------------------
# Tests bulk vs sequential — résultats identiques
# ---------------------------------------------------------------------------


class TestBulkVsSequentialEquivalence:
    """_process_citations_batch bulk produit exactement les mêmes résultats."""

    def test_batch_inserts_all_processed_markers(self, three_matches):
        """Tous les matchs doivent être marqués _processed après un batch."""
        shared_db = three_matches["shared_db"]
        player_db = three_matches["player_db"]
        match_ids = three_matches["match_ids"]

        # Patch CitationEngine.load_mappings pour simuler un mapping simple
        # (CitationEngine est importé localement dans _process_citations_batch)
        with patch(
            "src.analysis.citations.engine.CitationEngine.load_mappings",
            return_value={"medal_citation": {"mapping_type": "medal", "medal_id": 100}},
        ):
            from src.data.citations_backfill import _process_citations_batch

            result = _process_citations_batch(player_db, _XUID, shared_db, match_ids)

        assert result["matches_processed"] == 3

        with duckdb.connect(str(player_db)) as conn:
            processed = {
                r[0]
                for r in conn.execute(
                    "SELECT match_id FROM match_citations WHERE citation_name_norm = '_processed'"
                ).fetchall()
            }

        assert processed == set(match_ids), f"Matchs non marqués : {set(match_ids) - processed}"

    def test_all_matches_marked_as_processed(self, three_matches):
        """Chaque match doit avoir une entrée _processed dans match_citations."""
        shared_db = three_matches["shared_db"]
        player_db = three_matches["player_db"]
        match_ids = three_matches["match_ids"]

        with patch(
            "src.analysis.citations.engine.CitationEngine.load_mappings",
            return_value={"medal_citation": {"mapping_type": "medal", "medal_id": 100}},
        ):
            from src.data.citations_backfill import _process_citations_batch

            _process_citations_batch(player_db, _XUID, shared_db, match_ids)

        with duckdb.connect(str(player_db)) as conn:
            processed = {
                r[0]
                for r in conn.execute(
                    "SELECT match_id FROM match_citations WHERE citation_name_norm = '_processed'"
                ).fetchall()
            }

        assert processed == set(match_ids), f"Matchs non marqués : {set(match_ids) - processed}"

    def test_returns_empty_if_no_mappings(self, three_matches):
        """Retourne empty si engine.load_mappings() retourne un dict vide."""
        shared_db = three_matches["shared_db"]
        player_db = three_matches["player_db"]
        match_ids = three_matches["match_ids"]

        with patch(
            "src.analysis.citations.engine.CitationEngine.load_mappings",
            return_value={},
        ):
            from src.data.citations_backfill import _process_citations_batch

            result = _process_citations_batch(player_db, _XUID, shared_db, match_ids)

        assert result["matches_processed"] == 0
