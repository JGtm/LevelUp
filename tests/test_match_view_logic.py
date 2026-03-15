"""Tests pour src/ui/pages/match_view_logic.py et les méthodes repo correspondantes.

Les fonctions load_enrichment() et detect_abandoned_match() ont été migrées
vers DuckDBRepository (v6) : repo.load_player_match_enrichment() et
repo.is_abandoned_match().  Ces tests valident le comportement des méthodes
repo directement avec de vraies bases DuckDB en mémoire.
"""

from __future__ import annotations

from pathlib import Path

import duckdb
import pytest

from src.data.repositories.duckdb_repo import DuckDBRepository

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _build_player_db(path: Path) -> None:
    """Crée une DB joueur minimale avec player_match_enrichment."""
    path.parent.mkdir(parents=True, exist_ok=True)
    conn = duckdb.connect(str(path))
    conn.execute("""
        CREATE TABLE player_match_enrichment (
            match_id VARCHAR PRIMARY KEY,
            had_bot_teammate BOOLEAN,
            performance_score FLOAT,
            dominance_flag INTEGER
        )
    """)
    conn.close()


def _build_shared_db(path: Path) -> None:
    """Crée une DB shared minimale avec match_participants."""
    path.parent.mkdir(parents=True, exist_ok=True)
    conn = duckdb.connect(str(path))
    conn.execute("""
        CREATE TABLE match_participants (
            match_id VARCHAR,
            xuid VARCHAR,
            kills INTEGER,
            deaths INTEGER,
            score INTEGER
        )
    """)
    conn.close()


def _make_repo(player_path: Path, shared_path: Path) -> DuckDBRepository:
    return DuckDBRepository(
        player_db_path=player_path,
        xuid="xuid_test",
        shared_db_path=shared_path,
        read_only=False,
    )


# ---------------------------------------------------------------------------
# Tests : load_player_match_enrichment
# ---------------------------------------------------------------------------


class TestLoadPlayerMatchEnrichment:
    """Tests pour DuckDBRepository.load_player_match_enrichment()."""

    def test_returns_enrichment_row(self, tmp_path: Path) -> None:
        """Retourne les valeurs stockées pour un match connu."""
        player_db = tmp_path / "stats.duckdb"
        shared_db = tmp_path / "shared_matches.duckdb"
        _build_player_db(player_db)
        _build_shared_db(shared_db)

        conn = duckdb.connect(str(player_db))
        conn.execute(
            "INSERT INTO player_match_enrichment VALUES (?, ?, ?, ?)",
            ["match-001", True, 72.5, 1],
        )
        conn.close()

        repo = _make_repo(player_db, shared_db)
        had_bot, perf, dominance = repo.load_player_match_enrichment("match-001")

        assert had_bot is True
        assert perf == pytest.approx(72.5)
        assert dominance == 1

    def test_returns_defaults_when_match_absent(self, tmp_path: Path) -> None:
        """Retourne (False, None, None) si le match_id est inconnu."""
        player_db = tmp_path / "stats.duckdb"
        shared_db = tmp_path / "shared_matches.duckdb"
        _build_player_db(player_db)
        _build_shared_db(shared_db)

        repo = _make_repo(player_db, shared_db)
        had_bot, perf, dominance = repo.load_player_match_enrichment("match-unknown")

        assert had_bot is False
        assert perf is None
        assert dominance is None

    def test_returns_defaults_when_table_missing(self, tmp_path: Path) -> None:
        """Retourne (False, None, None) si la table n'existe pas encore."""
        player_db = tmp_path / "stats.duckdb"
        shared_db = tmp_path / "shared_matches.duckdb"
        # DB joueur sans la table player_match_enrichment
        player_db.parent.mkdir(parents=True, exist_ok=True)
        duckdb.connect(str(player_db)).close()
        _build_shared_db(shared_db)

        repo = _make_repo(player_db, shared_db)
        had_bot, perf, dominance = repo.load_player_match_enrichment("match-001")

        assert had_bot is False
        assert perf is None
        assert dominance is None

    def test_null_performance_score(self, tmp_path: Path) -> None:
        """Retourne perf=None si performance_score est NULL en DB."""
        player_db = tmp_path / "stats.duckdb"
        shared_db = tmp_path / "shared_matches.duckdb"
        _build_player_db(player_db)
        _build_shared_db(shared_db)

        conn = duckdb.connect(str(player_db))
        conn.execute(
            "INSERT INTO player_match_enrichment VALUES (?, ?, ?, ?)",
            ["match-002", False, None, None],
        )
        conn.close()

        repo = _make_repo(player_db, shared_db)
        had_bot, perf, dominance = repo.load_player_match_enrichment("match-002")

        assert had_bot is False
        assert perf is None
        assert dominance is None

    def test_had_bot_false_explicitly_stored(self, tmp_path: Path) -> None:
        """had_bot=False retourné depuis la DB, pas seulement par défaut."""
        player_db = tmp_path / "stats.duckdb"
        shared_db = tmp_path / "shared_matches.duckdb"
        _build_player_db(player_db)
        _build_shared_db(shared_db)

        conn = duckdb.connect(str(player_db))
        conn.execute(
            "INSERT INTO player_match_enrichment VALUES (?, ?, ?, ?)",
            ["match-003", False, 55.0, 0],
        )
        conn.close()

        repo = _make_repo(player_db, shared_db)
        had_bot, perf, dominance = repo.load_player_match_enrichment("match-003")

        assert had_bot is False
        assert perf == pytest.approx(55.0)
        assert dominance == 0  # flag=0 = ni domination ni humiliation

    def test_dominance_flag_humiliation(self, tmp_path: Path) -> None:
        """dominance_flag=2 (humiliation) est retourné correctement."""
        player_db = tmp_path / "stats.duckdb"
        shared_db = tmp_path / "shared_matches.duckdb"
        _build_player_db(player_db)
        _build_shared_db(shared_db)

        conn = duckdb.connect(str(player_db))
        conn.execute(
            "INSERT INTO player_match_enrichment VALUES (?, ?, ?, ?)",
            ["match-004", False, 18.0, 2],
        )
        conn.close()

        repo = _make_repo(player_db, shared_db)
        _, _, dominance = repo.load_player_match_enrichment("match-004")

        assert dominance == 2


# ---------------------------------------------------------------------------
# Tests : is_abandoned_match
# ---------------------------------------------------------------------------


class TestIsAbandonedMatch:
    """Tests pour DuckDBRepository.is_abandoned_match()."""

    def test_abandoned_when_all_zeros(self, tmp_path: Path) -> None:
        """Retourne True quand tous les participants ont kills=deaths=score=0."""
        player_db = tmp_path / "stats.duckdb"
        shared_db = tmp_path / "shared_matches.duckdb"
        _build_player_db(player_db)
        _build_shared_db(shared_db)

        conn = duckdb.connect(str(shared_db))
        for i in range(8):
            conn.execute(
                "INSERT INTO match_participants VALUES (?, ?, ?, ?, ?)",
                ["match-abc", f"xuid_{i}", 0, 0, 0],
            )
        conn.close()

        repo = _make_repo(player_db, shared_db)
        assert repo.is_abandoned_match("match-abc") is True

    def test_not_abandoned_when_kills_positive(self, tmp_path: Path) -> None:
        """Retourne False quand des kills ont été enregistrés."""
        player_db = tmp_path / "stats.duckdb"
        shared_db = tmp_path / "shared_matches.duckdb"
        _build_player_db(player_db)
        _build_shared_db(shared_db)

        conn = duckdb.connect(str(shared_db))
        conn.execute(
            "INSERT INTO match_participants VALUES (?, ?, ?, ?, ?)",
            ["match-xyz", "xuid_0", 12, 5, 1500],
        )
        conn.close()

        repo = _make_repo(player_db, shared_db)
        assert repo.is_abandoned_match("match-xyz") is False

    def test_returns_false_when_no_participants(self, tmp_path: Path) -> None:
        """Retourne False si aucun participant trouvé (match_id inconnu)."""
        player_db = tmp_path / "stats.duckdb"
        shared_db = tmp_path / "shared_matches.duckdb"
        _build_player_db(player_db)
        _build_shared_db(shared_db)

        repo = _make_repo(player_db, shared_db)
        assert repo.is_abandoned_match("match-unknown") is False

    def test_returns_false_when_shared_unavailable(self, tmp_path: Path) -> None:
        """Retourne False si shared_matches.duckdb n'existe pas."""
        player_db = tmp_path / "stats.duckdb"
        _build_player_db(player_db)
        # shared_db inexistante — pas de fichier créé

        repo = DuckDBRepository(
            player_db_path=player_db,
            xuid="xuid_test",
            shared_db_path=tmp_path / "nonexistent_shared.duckdb",
            read_only=False,
        )
        assert repo.is_abandoned_match("match-abc") is False

    def test_not_abandoned_when_only_score_zero(self, tmp_path: Path) -> None:
        """Retourne False si kills/deaths > 0 même si score=0 (partie comptée)."""
        player_db = tmp_path / "stats.duckdb"
        shared_db = tmp_path / "shared_matches.duckdb"
        _build_player_db(player_db)
        _build_shared_db(shared_db)

        conn = duckdb.connect(str(shared_db))
        conn.execute(
            "INSERT INTO match_participants VALUES (?, ?, ?, ?, ?)",
            ["match-partial", "xuid_0", 8, 3, 0],
        )
        conn.close()

        repo = _make_repo(player_db, shared_db)
        assert repo.is_abandoned_match("match-partial") is False

    def test_abandoned_logged(self, tmp_path: Path, caplog) -> None:
        """Un match abandonné génère un log DEBUG avec le match_id."""
        import logging

        player_db = tmp_path / "stats.duckdb"
        shared_db = tmp_path / "shared_matches.duckdb"
        _build_player_db(player_db)
        _build_shared_db(shared_db)

        conn = duckdb.connect(str(shared_db))
        for i in range(4):
            conn.execute(
                "INSERT INTO match_participants VALUES (?, ?, ?, ?, ?)",
                ["match-log", f"xuid_{i}", 0, 0, 0],
            )
        conn.close()

        repo = _make_repo(player_db, shared_db)
        with caplog.at_level(logging.DEBUG, logger="src.data.repositories._match_queries"):
            result = repo.is_abandoned_match("match-log")

        assert result is True
        assert any("match-log" in record.message for record in caplog.records)
