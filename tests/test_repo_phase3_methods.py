"""Tests de couverture pour les méthodes repo ajoutées en Phase 3 v6.

Couvre :
- CareerMixin.load_friends_xuids_csv          (_career_repo.py)
- CareerMixin.load_skill_ratings_batch        (_career_repo.py)
- MediaLibraryMixin.load_match_registry_raw   (_media_repo.py)
- MediaLibraryMixin.load_media_files_raw      (_media_repo.py)
"""

from __future__ import annotations

from pathlib import Path

import duckdb
import pytest

# ---------------------------------------------------------------------------
# Fixtures communes
# ---------------------------------------------------------------------------


def _make_repo(player_db: Path, shared_db: Path | None, xuid: str):
    """Construit un DuckDBRepository pointant vers les DBs de test."""
    from src.data.repositories.duckdb_repo import DuckDBRepository

    kwargs = dict(  # noqa: C408 — muté après (shared_db conditionnel)
        player_db_path=player_db,
        xuid=xuid,
        metadata_db_path=player_db.parent / "metadata.duckdb",  # inexistant = pas attaché
        read_only=True,
    )
    if shared_db is not None:
        kwargs["shared_db_path"] = shared_db
    else:
        kwargs["shared_db_path"] = player_db.parent / "shared_matches_v2.duckdb"  # inexistant
    return DuckDBRepository(**kwargs)


@pytest.fixture()
def player_db_with_enrichment(tmp_path: Path) -> Path:
    """Player DB avec player_match_enrichment et match_skill_rank."""
    db_path = tmp_path / "stats.duckdb"
    with duckdb.connect(str(db_path)) as conn:
        conn.execute("""
            CREATE TABLE player_match_enrichment (
                match_id VARCHAR PRIMARY KEY,
                friends_xuids VARCHAR,
                is_with_friends BOOLEAN DEFAULT FALSE,
                dominance_flag TINYINT DEFAULT 0
            )
        """)
        conn.execute("""
            CREATE TABLE match_skill_rank (
                match_id VARCHAR PRIMARY KEY,
                rating_value DOUBLE,
                rating_type VARCHAR,
                playlist_group VARCHAR
            )
        """)
        conn.execute("""
            INSERT INTO player_match_enrichment VALUES
            ('m1', 'XUID_A,XUID_B', true, 0),
            ('m2', NULL, false, 0),
            ('m3', '', false, 1)
        """)
        conn.execute("""
            INSERT INTO match_skill_rank VALUES
            ('m1', 1450.5, 'LUSR', 'ranked_arena'),
            ('m2', 1500.0, 'LUSR', 'ranked_arena'),
            ('m3', 1200.0, 'CSR', 'ranked_btb')
        """)
    return db_path


@pytest.fixture()
def player_db_with_media(tmp_path: Path) -> Path:
    """Player DB avec media_files et media_match_associations."""
    db_path = tmp_path / "stats.duckdb"
    with duckdb.connect(str(db_path)) as conn:
        conn.execute("""
            CREATE TABLE media_files (
                file_path VARCHAR PRIMARY KEY,
                mtime DOUBLE,
                mtime_paris_epoch BIGINT,
                file_ext VARCHAR,
                kind VARCHAR,
                file_name VARCHAR,
                thumbnail_path VARCHAR
            )
        """)
        conn.execute("""
            CREATE TABLE media_match_associations (
                media_path VARCHAR,
                match_id VARCHAR,
                match_start_time TIMESTAMP,
                association_confidence DOUBLE,
                xuid VARCHAR
            )
        """)
        conn.execute("""
            INSERT INTO media_files VALUES
            ('/videos/clip1.mp4', 1000.0, 1000, 'mp4', 'video', 'clip1.mp4', '/thumbs/clip1.jpg'),
            ('/videos/clip2.mp4', 2000.0, 2000, 'mp4', 'video', 'clip2.mp4', '/thumbs/clip2.jpg'),
            ('/videos/clip3.mp4', 3000.0, 3000, 'mp4', 'video', 'clip3.mp4', NULL)
        """)
        conn.execute("""
            INSERT INTO media_match_associations VALUES
            ('/videos/clip1.mp4', 'm1', '2025-01-01', 0.95, 'XUID_ME'),
            ('/videos/clip2.mp4', 'm2', '2025-01-02', 0.80, 'XUID_OTHER'),
            ('/videos/clip3.mp4', 'm3', '2025-01-03', 0.70, 'XUID_ME')
        """)
    return db_path


@pytest.fixture()
def shared_db_with_registry(tmp_path: Path) -> Path:
    """Shared DB avec match_registry."""
    db_path = tmp_path / "shared_matches_v2.duckdb"
    with duckdb.connect(str(db_path)) as conn:
        conn.execute("""
            CREATE TABLE match_registry (
                match_id VARCHAR PRIMARY KEY,
                start_time TIMESTAMP,
                duration_seconds INTEGER
            )
        """)
        conn.execute("""
            INSERT INTO match_registry VALUES
            ('m1', '2025-01-01 10:00:00', 600),
            ('m2', '2025-01-02 11:00:00', NULL),
            ('m3', '2025-01-03 12:00:00', 450)
        """)
        # match sans start_time → exclu par la requête
        conn.execute("""
            INSERT INTO match_registry VALUES ('m4', NULL, 300)
        """)
    return db_path


# ---------------------------------------------------------------------------
# Tests : load_friends_xuids_csv
# ---------------------------------------------------------------------------


class TestLoadFriendsXuidsCsv:
    """Tests pour CareerMixin.load_friends_xuids_csv."""

    def test_returns_csv_when_present(
        self, player_db_with_enrichment: Path, tmp_path: Path
    ) -> None:
        repo = _make_repo(player_db_with_enrichment, None, "XUID_ME")
        result = repo.load_friends_xuids_csv("m1")
        assert result == "XUID_A,XUID_B"

    def test_returns_none_when_null(self, player_db_with_enrichment: Path, tmp_path: Path) -> None:
        repo = _make_repo(player_db_with_enrichment, None, "XUID_ME")
        result = repo.load_friends_xuids_csv("m2")
        assert result is None

    def test_returns_none_when_empty_string(
        self, player_db_with_enrichment: Path, tmp_path: Path
    ) -> None:
        """Chaîne vide traitée comme None (falsy)."""
        repo = _make_repo(player_db_with_enrichment, None, "XUID_ME")
        result = repo.load_friends_xuids_csv("m3")
        assert result is None

    def test_returns_none_when_match_not_found(
        self, player_db_with_enrichment: Path, tmp_path: Path
    ) -> None:
        repo = _make_repo(player_db_with_enrichment, None, "XUID_ME")
        result = repo.load_friends_xuids_csv("unknown_match")
        assert result is None

    def test_returns_none_when_table_absent(self, tmp_path: Path) -> None:
        """DB sans player_match_enrichment → None sans crash."""
        db_path = tmp_path / "empty_stats.duckdb"
        with duckdb.connect(str(db_path)):
            pass
        repo = _make_repo(db_path, None, "XUID_ME")
        result = repo.load_friends_xuids_csv("m1")
        assert result is None


# ---------------------------------------------------------------------------
# Tests : load_skill_ratings_batch
# ---------------------------------------------------------------------------


class TestLoadSkillRatingsBatch:
    """Tests pour CareerMixin.load_skill_ratings_batch."""

    def test_returns_correct_mapping(self, player_db_with_enrichment: Path) -> None:
        repo = _make_repo(player_db_with_enrichment, None, "XUID_ME")
        result = repo.load_skill_ratings_batch(["m1", "m2"])
        assert "m1" in result
        assert "m2" in result
        val1, typ1 = result["m1"]
        assert val1 == pytest.approx(1450.5)
        assert typ1 == "LUSR"
        val2, typ2 = result["m2"]
        assert val2 == pytest.approx(1500.0)
        assert typ2 == "LUSR"

    def test_returns_empty_for_empty_input(self, player_db_with_enrichment: Path) -> None:
        repo = _make_repo(player_db_with_enrichment, None, "XUID_ME")
        result = repo.load_skill_ratings_batch([])
        assert result == {}

    def test_partial_result_when_some_missing(self, player_db_with_enrichment: Path) -> None:
        repo = _make_repo(player_db_with_enrichment, None, "XUID_ME")
        result = repo.load_skill_ratings_batch(["m1", "nonexistent"])
        assert "m1" in result
        assert "nonexistent" not in result

    def test_returns_empty_when_table_absent(self, tmp_path: Path) -> None:
        """DB sans match_skill_rank → {} sans crash."""
        db_path = tmp_path / "empty_stats.duckdb"
        with duckdb.connect(str(db_path)):
            pass
        repo = _make_repo(db_path, None, "XUID_ME")
        result = repo.load_skill_ratings_batch(["m1"])
        assert result == {}


# ---------------------------------------------------------------------------
# Tests : load_match_registry_raw
# ---------------------------------------------------------------------------


class TestLoadMatchRegistryRaw:
    """Tests pour MediaLibraryMixin.load_match_registry_raw."""

    def test_returns_tuples(
        self, player_db_with_enrichment: Path, shared_db_with_registry: Path
    ) -> None:
        repo = _make_repo(player_db_with_enrichment, shared_db_with_registry, "XUID_ME")
        result = repo.load_match_registry_raw()
        assert isinstance(result, list)
        assert len(result) == 3  # m4 exclu (start_time IS NULL)
        match_ids = {r[0] for r in result}
        assert "m1" in match_ids
        assert "m4" not in match_ids  # filtré par WHERE start_time IS NOT NULL

    def test_columns_correct(
        self, player_db_with_enrichment: Path, shared_db_with_registry: Path
    ) -> None:
        repo = _make_repo(player_db_with_enrichment, shared_db_with_registry, "XUID_ME")
        result = repo.load_match_registry_raw()
        # Chaque tuple : (match_id, start_time, duration_seconds)
        assert len(result[0]) == 3

    def test_returns_empty_when_shared_absent(self, tmp_path: Path) -> None:
        """Sans shared DB → [] sans crash."""
        db_path = tmp_path / "stats.duckdb"
        with duckdb.connect(str(db_path)):
            pass
        repo = _make_repo(db_path, None, "XUID_ME")
        result = repo.load_match_registry_raw()
        assert result == []


# ---------------------------------------------------------------------------
# Tests : load_media_files_raw
# ---------------------------------------------------------------------------


class TestLoadMediaFilesRaw:
    """Tests pour MediaLibraryMixin.load_media_files_raw."""

    def test_returns_none_when_table_absent(self, tmp_path: Path) -> None:
        """DB sans media_files → None (table absente, non erreur)."""
        db_path = tmp_path / "stats.duckdb"
        with duckdb.connect(str(db_path)):
            pass
        repo = _make_repo(db_path, None, "XUID_ME")
        result = repo.load_media_files_raw("XUID_ME", "MyGamertag")
        assert result is None

    def test_returns_list_when_table_exists(self, player_db_with_media: Path) -> None:
        repo = _make_repo(player_db_with_media, None, "XUID_ME")
        result = repo.load_media_files_raw("XUID_ME", None)
        assert isinstance(result, list)
        assert len(result) == 2  # clip1 et clip3 pour XUID_ME
        paths = {r[0] for r in result}
        assert "/videos/clip1.mp4" in paths
        assert "/videos/clip3.mp4" in paths
        assert "/videos/clip2.mp4" not in paths  # XUID_OTHER

    def test_filter_by_gamertag(self, player_db_with_media: Path) -> None:
        """Gamertag utilisé si xuid absent."""
        repo = _make_repo(player_db_with_media, None, "XUID_ME")
        result = repo.load_media_files_raw(None, "XUID_OTHER")
        assert result is not None
        paths = {r[0] for r in result}
        assert "/videos/clip2.mp4" in paths

    def test_returns_all_when_no_uid(self, player_db_with_media: Path) -> None:
        """Sans xuid ni gamertag → tous les fichiers (pas de filtre)."""
        repo = _make_repo(player_db_with_media, None, "XUID_ME")
        result = repo.load_media_files_raw(None, None)
        assert result is not None
        assert len(result) == 3  # tous

    def test_tuple_has_correct_columns(self, player_db_with_media: Path) -> None:
        """Chaque tuple a 11 colonnes."""
        repo = _make_repo(player_db_with_media, None, "XUID_ME")
        result = repo.load_media_files_raw(None, None)
        assert result is not None
        assert len(result[0]) == 11
