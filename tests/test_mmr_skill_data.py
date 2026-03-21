"""Tests unitaires pour les chemins MMR critiques.

Couvre :
- ``load_match_skill_data`` : cas nominal, fallback coéquipier, MMR absent total
- ``_fallback_mmr_from_teammate`` : comportement isolé
- ``db_cache_key`` : sentinel WAL pour fermer la race condition write/checkpoint

Ces tests ont été ajoutés après le diagnostic du bug d'affichage MMR NULL
(root cause : race condition WAL + exceptions silencieuses).
"""

from __future__ import annotations

import time
from pathlib import Path

import duckdb
import pytest

from src.data.repositories.duckdb_repo import DuckDBRepository

PLAYER_XUID = "2533274858283686"
TEAMMATE_XUID = "2533274800000001"
MATCH_WITH_MMR = "aaaa0001-0000-0000-0000-000000000001"
MATCH_TEAMMATE_MMR = "aaaa0002-0000-0000-0000-000000000002"
MATCH_NO_MMR = "aaaa0003-0000-0000-0000-000000000003"
MATCH_MISSING = "aaaa0004-0000-0000-0000-000000000004"


# =============================================================================
# Helpers
# =============================================================================


def _create_minimal_player_db(db_path: Path) -> None:
    """DB joueur minimale (sync_meta uniquement — v5.1)."""
    db_path.parent.mkdir(parents=True, exist_ok=True)
    with duckdb.connect(str(db_path)) as conn:
        conn.execute(
            "CREATE TABLE IF NOT EXISTS sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR)"
        )
        conn.execute(
            "INSERT INTO sync_meta VALUES ('xuid', ?)",
            [PLAYER_XUID],
        )


def _create_shared_db_with_mmr(db_path: Path) -> None:
    """shared_matches avec match_participants incluant MMR — 3 cas de figure."""
    db_path.parent.mkdir(parents=True, exist_ok=True)
    with duckdb.connect(str(db_path)) as conn:
        conn.execute("""
            CREATE TABLE match_participants (
                match_id     VARCHAR NOT NULL,
                xuid         VARCHAR NOT NULL,
                gamertag     VARCHAR,
                team_id      INTEGER,
                outcome      INTEGER,
                rank         SMALLINT,
                score        INTEGER,
                kills        SMALLINT,
                deaths       SMALLINT,
                assists      SMALLINT,
                team_mmr     FLOAT,
                enemy_mmr    FLOAT,
                kills_expected     FLOAT,
                kills_stddev       FLOAT,
                deaths_expected    FLOAT,
                deaths_stddev      FLOAT,
                assists_expected   FLOAT,
                assists_stddev     FLOAT,
                PRIMARY KEY (match_id, xuid)
            )
        """)
        # MATCH_WITH_MMR : joueur a team_mmr + enemy_mmr
        conn.execute(
            "INSERT INTO match_participants VALUES (?, ?, 'PlayerMain', 0, 2, 1, 2500, "
            "15, 6, 4, 1220.0, 1091.0, 0.72, 0.15, 0.35, 0.10, 0.08, 0.05)",
            [MATCH_WITH_MMR, PLAYER_XUID],
        )
        # MATCH_TEAMMATE_MMR : joueur a MMR NULL, coéquipier a le MMR
        conn.execute(
            "INSERT INTO match_participants VALUES (?, ?, 'PlayerMain', 0, 2, 1, 2100, "
            "12, 7, 3, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL)",
            [MATCH_TEAMMATE_MMR, PLAYER_XUID],
        )
        conn.execute(
            "INSERT INTO match_participants VALUES (?, ?, 'Teammate', 0, 2, 2, 1900, "
            "10, 8, 5, 1350.0, 1280.0, NULL, NULL, NULL, NULL, NULL, NULL)",
            [MATCH_TEAMMATE_MMR, TEAMMATE_XUID],
        )
        # MATCH_NO_MMR : tout le monde a NULL
        conn.execute(
            "INSERT INTO match_participants VALUES (?, ?, 'PlayerMain', 0, 2, 1, 1800, "
            "10, 9, 2, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL)",
            [MATCH_NO_MMR, PLAYER_XUID],
        )
        conn.execute(
            "INSERT INTO match_participants VALUES (?, ?, 'Teammate', 0, 2, 2, 1600, "
            "8, 10, 1, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL)",
            [MATCH_NO_MMR, TEAMMATE_XUID],
        )


@pytest.fixture()
def shared_db_path(tmp_path: Path) -> Path:
    p = tmp_path / "data" / "warehouse" / "shared_matches_v2.duckdb"
    _create_shared_db_with_mmr(p)
    return p


@pytest.fixture()
def player_db_path(tmp_path: Path) -> Path:
    p = tmp_path / "data" / "players" / "TestPlayer" / "stats.duckdb"
    _create_minimal_player_db(p)
    return p


@pytest.fixture()
def repo(player_db_path: Path, shared_db_path: Path) -> DuckDBRepository:
    return DuckDBRepository(
        player_db_path=player_db_path,
        xuid=PLAYER_XUID,
        shared_db_path=shared_db_path,
        gamertag="TestPlayer",
        read_only=True,
    )


# =============================================================================
# Tests load_match_skill_data
# =============================================================================


class TestLoadMatchSkillData:
    """Tests de load_match_skill_data avec la table shared.match_participants."""

    def test_returns_mmr_when_present(self, repo: DuckDBRepository) -> None:
        """Cas nominal : le joueur a team_mmr + enemy_mmr dans match_participants."""
        result = repo.load_match_skill_data(MATCH_WITH_MMR)

        assert result is not None
        assert result["team_mmr"] == pytest.approx(1220.0, abs=0.1)
        assert result["enemy_mmr"] == pytest.approx(1091.0, abs=0.1)
        assert result["kills"]["count"] == 15
        assert result["deaths"]["count"] == 6

    def test_returns_expected_kills_when_present(self, repo: DuckDBRepository) -> None:
        """Les kills_expected/stddev sont bien chargés quand disponibles."""
        result = repo.load_match_skill_data(MATCH_WITH_MMR)

        assert result is not None
        assert result["kills"]["expected"] == pytest.approx(0.72, abs=0.01)
        assert result["kills"]["stddev"] == pytest.approx(0.15, abs=0.01)

    def test_fallback_to_teammate_mmr(self, repo: DuckDBRepository) -> None:
        """Quand le joueur a NULL MMR, le fallback coéquipier retourne le MMR."""
        result = repo.load_match_skill_data(MATCH_TEAMMATE_MMR)

        assert result is not None
        # Le fallback doit avoir récupéré le MMR du coéquipier
        assert result["team_mmr"] == pytest.approx(1350.0, abs=0.1)
        assert result["enemy_mmr"] == pytest.approx(1280.0, abs=0.1)

    def test_returns_none_mmr_when_all_null(self, repo: DuckDBRepository) -> None:
        """Quand TOUS les joueurs de l'équipe ont NULL MMR, retourne None dans le dict."""
        result = repo.load_match_skill_data(MATCH_NO_MMR)

        assert result is not None, "load_match_skill_data doit retourner un dict même si MMR absent"
        assert result["team_mmr"] is None
        assert result["enemy_mmr"] is None

    def test_returns_none_when_match_not_found(self, repo: DuckDBRepository) -> None:
        """Retourne None si le match_id n'existe pas dans match_participants."""
        result = repo.load_match_skill_data(MATCH_MISSING)

        assert result is None

    def test_returns_none_when_no_shared_db(self, player_db_path: Path, tmp_path: Path) -> None:
        """Retourne None si shared_matches n'est pas disponible."""
        absent_shared = tmp_path / "data" / "warehouse" / "absent.duckdb"
        repo_no_shared = DuckDBRepository(
            player_db_path=player_db_path,
            xuid=PLAYER_XUID,
            shared_db_path=absent_shared,
            gamertag="TestPlayer",
            read_only=True,
        )
        result = repo_no_shared.load_match_skill_data(MATCH_WITH_MMR)
        assert result is None

    def test_team_id_present_in_result(self, repo: DuckDBRepository) -> None:
        """team_id est inclus dans le dict retourné."""
        result = repo.load_match_skill_data(MATCH_WITH_MMR)

        assert result is not None
        assert "team_id" in result
        assert result["team_id"] == 0


# =============================================================================
# Tests db_cache_key (sentinel WAL)
# =============================================================================


class TestDbCacheKeyWal:
    """Tests de la sentinel WAL dans db_cache_key pour invalider le cache Streamlit
    dès qu'un sync commence à écrire (avant checkpoint DuckDB).
    """

    def test_key_is_5_tuple(self, player_db_path: Path, shared_db_path: Path) -> None:
        """db_cache_key doit retourner un tuple de 5 entiers."""
        from src.ui._cache_core import db_cache_key

        key = db_cache_key(str(player_db_path))

        assert key is not None
        assert len(key) == 5
        assert all(isinstance(v, int) for v in key)

    def test_no_wal_has_zero_sentinel(self, player_db_path: Path, shared_db_path: Path) -> None:
        """Sans fichier WAL, le 5e composant (wal_sentinel) vaut 0."""
        from src.ui._cache_core import db_cache_key

        wal = shared_db_path.with_suffix(shared_db_path.suffix + ".wal")
        assert not wal.exists(), "Le WAL ne doit pas exister avant ce test"

        key = db_cache_key(str(player_db_path))

        assert key is not None
        assert key[4] == 0, f"wal_sentinel attendu=0, obtenu={key[4]}"

    def test_wal_creates_nonzero_sentinel(self, player_db_path: Path, shared_db_path: Path) -> None:
        """Dès que le WAL existe, le 5e composant est non-nul → cache miss forcé."""
        from src.ui._cache_core import db_cache_key

        wal = shared_db_path.with_suffix(shared_db_path.suffix + ".wal")

        key_before = db_cache_key(str(player_db_path))
        assert key_before is not None
        assert key_before[4] == 0

        # Simule la création d'un WAL par le process de sync
        wal.write_bytes(b"WAL_STUB")

        key_after = db_cache_key(str(player_db_path))
        assert key_after is not None
        assert key_after[4] != 0, "wal_sentinel doit être non-nul quand le WAL existe"
        assert key_before != key_after, "La clé doit changer quand le WAL est créé"

        wal.unlink()

    def test_wal_removal_resets_sentinel(self, player_db_path: Path, shared_db_path: Path) -> None:
        """Après checkpoint (suppression WAL), wal_sentinel revient à 0."""
        from src.ui._cache_core import db_cache_key

        wal = shared_db_path.with_suffix(shared_db_path.suffix + ".wal")
        wal.write_bytes(b"WAL_STUB")

        key_with_wal = db_cache_key(str(player_db_path))
        assert key_with_wal is not None
        assert key_with_wal[4] != 0

        wal.unlink()

        key_after_checkpoint = db_cache_key(str(player_db_path))
        assert key_after_checkpoint is not None
        assert key_after_checkpoint[4] == 0

    def test_key_changes_on_shared_mtime(self, player_db_path: Path, shared_db_path: Path) -> None:
        """La clé change quand shared_matches_v2.duckdb est modifié (checkpoint)."""
        from src.ui._cache_core import db_cache_key

        key_before = db_cache_key(str(player_db_path))

        # Touche le fichier shared pour simuler un checkpoint
        time.sleep(0.01)
        shared_db_path.touch()

        key_after = db_cache_key(str(player_db_path))

        assert key_before != key_after

    def test_key_none_for_missing_player_db(self) -> None:
        """Retourne None si la DB joueur n'existe pas."""
        from src.ui._cache_core import db_cache_key

        result = db_cache_key("/nonexistent/path/stats.duckdb")
        assert result is None
