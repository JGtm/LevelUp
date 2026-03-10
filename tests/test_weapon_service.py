"""Tests unitaires — WeaponExtractionService (avec mocks).

Vérifie l'orchestration : téléchargement chunks, corrélation, upsert.
"""

from __future__ import annotations

import asyncio
from types import SimpleNamespace
from unittest.mock import patch

import duckdb
import pytest

from src.data.services.weapon_extraction_service import WeaponExtractionService

# ═══════════════════════════════════════════════════════════════════════════
# Helpers et fixtures
# ═══════════════════════════════════════════════════════════════════════════


class FakeAPIPort:
    """Implémentation minimale de HaloAPIPort pour les tests."""

    def __init__(self, *, film=None, chunk_data=None):
        self._film = film
        self._chunk_data = chunk_data or b"\x00" * 100

    async def __aenter__(self):
        return self

    async def __aexit__(self, *args):
        pass

    async def get_film_by_match_id(self, match_id: str):
        return self._film

    async def download_film_chunk(self, url: str) -> bytes | None:
        return self._chunk_data


def _make_film(*, n_chunks: int = 2, chunk_duration_ms: int = 5000):
    """Construit un objet film simulé (SimpleNamespace)."""
    chunks = []
    for i in range(n_chunks):
        chunks.append(
            SimpleNamespace(
                index=i,
                chunk_type=SimpleNamespace(value=2),  # REPLICATION_DATA
                chunk_start_time_offset_milliseconds=i * chunk_duration_ms,
                duration_milliseconds=chunk_duration_ms,
                file_relative_path=f"/chunk_{i:02d}.bin",
            )
        )
    return SimpleNamespace(
        blob_storage_path_prefix="https://blob.test/film/",
        custom_data=SimpleNamespace(chunks=chunks),
    )


@pytest.fixture()
def tmp_cache(tmp_path):
    """Répertoire cache temporaire."""
    cache = tmp_path / "chunks"
    cache.mkdir()
    return cache


@pytest.fixture()
def in_memory_conn():
    """Connexion DuckDB en mémoire avec la table weapon_kills."""
    conn = duckdb.connect(":memory:")
    conn.execute(
        """
        CREATE TABLE weapon_kills (
            match_id   VARCHAR NOT NULL,
            xuid       VARCHAR NOT NULL,
            weapon_id  INTEGER NOT NULL,
            kills      SMALLINT NOT NULL,
            PRIMARY KEY (match_id, xuid, weapon_id)
        )
        """
    )
    conn.execute(
        """
        CREATE TABLE match_registry (
            match_id VARCHAR PRIMARY KEY,
            backfill_completed INTEGER DEFAULT 0
        )
        """
    )
    conn.execute("INSERT INTO match_registry VALUES ('m1234567', 0)")
    return conn


# ═══════════════════════════════════════════════════════════════════════════
# Tests
# ═══════════════════════════════════════════════════════════════════════════


class TestWeaponExtractionService:
    def test_no_kills_returns_error(self, in_memory_conn, tmp_cache):
        """Si aucun kill POV, on obtient une erreur sans crash."""
        api = FakeAPIPort()
        service = WeaponExtractionService(api, in_memory_conn, tmp_cache)

        with patch.object(
            type(service),
            "process_match",
            wraps=service.process_match,
        ):
            pass

        # Simuler directement sans kills
        with patch(
            "src.data.services.weapon_extraction_service.WeaponKillsMixin"
            ".load_player_kills_for_match",
            return_value=[],
        ):
            summary = asyncio.run(service.process_match("m1234567", "TestPlayer", "xuid123"))

        assert summary["error"] == "aucun kill POV"
        assert summary["kills_total"] == 0

    def test_no_film_returns_error(self, in_memory_conn, tmp_cache):
        """Si le film n'est pas disponible, erreur 'aucun chunk'."""
        api = FakeAPIPort(film=None)  # pas de film
        service = WeaponExtractionService(api, in_memory_conn, tmp_cache)

        kills = [
            {
                "time_ms": 5000,
                "gamertag": "TestPlayer",
                "xuid": "xuid123",
                "medals_nearby": [],
                "is_melee": False,
                "is_grenade": False,
            },
        ]
        with patch(
            "src.data.services.weapon_extraction_service.WeaponKillsMixin"
            ".load_player_kills_for_match",
            return_value=kills,
        ):
            summary = asyncio.run(service.process_match("m1234567", "TestPlayer", "xuid123"))

        assert summary["error"] == "aucun chunk téléchargé"
        assert summary["kills_total"] == 1

    def test_dry_run_no_write(self, in_memory_conn, tmp_cache):
        """En dry_run, aucune écriture DB mais les compteurs sont remplis."""
        film = _make_film(n_chunks=1, chunk_duration_ms=10000)
        api = FakeAPIPort(film=film, chunk_data=b"\x00" * 200)
        service = WeaponExtractionService(api, in_memory_conn, tmp_cache)

        kills = [
            {
                "time_ms": 5000,
                "gamertag": "TestPlayer",
                "xuid": "xuid123",
                "medals_nearby": ["Pummel"],
                "is_melee": True,
                "is_grenade": False,
            },
        ]
        with patch(
            "src.data.services.weapon_extraction_service.WeaponKillsMixin"
            ".load_player_kills_for_match",
            return_value=kills,
        ):
            summary = asyncio.run(
                service.process_match("m1234567", "TestPlayer", "xuid123", dry_run=True)
            )

        assert summary["kills_total"] == 1
        assert summary["rows_inserted"] == 0
        # Vérifie qu'aucune ligne n'a été insérée
        count = in_memory_conn.execute("SELECT COUNT(*) FROM weapon_kills").fetchone()[0]
        assert count == 0

    def test_process_match_upserts(self, in_memory_conn, tmp_cache):
        """Un match avec un kill melee écrit correctement en DB."""
        film = _make_film(n_chunks=1, chunk_duration_ms=10000)
        api = FakeAPIPort(film=film, chunk_data=b"\x00" * 200)
        service = WeaponExtractionService(api, in_memory_conn, tmp_cache)

        kills = [
            {
                "time_ms": 5000,
                "gamertag": "TestPlayer",
                "xuid": "xuid123",
                "medals_nearby": ["Pummel"],
                "is_melee": True,
                "is_grenade": False,
            },
        ]
        with patch(
            "src.data.services.weapon_extraction_service.WeaponKillsMixin"
            ".load_player_kills_for_match",
            return_value=kills,
        ):
            summary = asyncio.run(service.process_match("m1234567", "TestPlayer", "xuid123"))

        assert summary["kills_total"] == 1
        assert summary["kills_attributed"] == 1
        assert summary["rows_inserted"] == 1
        # Vérifie la ligne insérée
        row = in_memory_conn.execute(
            "SELECT weapon_id, kills FROM weapon_kills WHERE match_id = 'm1234567'"
        ).fetchone()
        assert row is not None
        assert row[0] == 1  # MELEE_API_ID
        assert row[1] == 1

    def test_chunk_caching(self, in_memory_conn, tmp_cache):
        """Les chunks téléchargés sont mis en cache sur disque."""
        film = _make_film(n_chunks=1, chunk_duration_ms=10000)
        api = FakeAPIPort(film=film, chunk_data=b"\xab" * 50)
        service = WeaponExtractionService(api, in_memory_conn, tmp_cache)

        kills = [
            {
                "time_ms": 5000,
                "gamertag": "TestPlayer",
                "xuid": "xuid123",
                "medals_nearby": [],
                "is_melee": True,
                "is_grenade": False,
            },
        ]
        with patch(
            "src.data.services.weapon_extraction_service.WeaponKillsMixin"
            ".load_player_kills_for_match",
            return_value=kills,
        ):
            asyncio.run(service.process_match("m1234567", "TestPlayer", "xuid123"))

        # Vérifie que le cache contient le fichier
        cache_files = list(tmp_cache.rglob("*.bin"))
        assert len(cache_files) == 1
        assert cache_files[0].read_bytes() == b"\xab" * 50

    def test_exception_handled_gracefully(self, in_memory_conn, tmp_cache):
        """Une exception pendant le traitement ne fait pas crasher."""
        api = FakeAPIPort()
        service = WeaponExtractionService(api, in_memory_conn, tmp_cache)

        with patch(
            "src.data.services.weapon_extraction_service.WeaponKillsMixin"
            ".load_player_kills_for_match",
            side_effect=RuntimeError("DB inaccessible"),
        ):
            summary = asyncio.run(service.process_match("m1234567", "TestPlayer", "xuid123"))

        assert "error" in summary
        assert "DB inaccessible" in summary["error"]


class TestWeaponKillsMixinRepo:
    """Tests pour les méthodes statiques écrites dans _weapon_kills_repo."""

    @pytest.fixture()
    def conn(self):
        """Connexion DuckDB en mémoire avec les tables nécessaires."""
        conn = duckdb.connect(":memory:")
        conn.execute(
            """
            CREATE TABLE weapon_kills (
                match_id   VARCHAR NOT NULL,
                xuid       VARCHAR NOT NULL,
                weapon_id  INTEGER NOT NULL,
                kills      SMALLINT NOT NULL,
                PRIMARY KEY (match_id, xuid, weapon_id)
            )
            """
        )
        conn.execute(
            """
            CREATE TABLE match_registry (
                match_id VARCHAR PRIMARY KEY,
                backfill_completed INTEGER DEFAULT 0,
                start_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                is_firefight BOOLEAN DEFAULT FALSE
            )
            """
        )
        conn.execute(
            """
            CREATE TABLE match_participants (
                match_id VARCHAR,
                xuid VARCHAR
            )
            """
        )
        conn.execute(
            """
            CREATE TABLE xuid_aliases (
                xuid VARCHAR,
                gamertag VARCHAR
            )
            """
        )
        return conn

    def test_upsert_and_conflict(self, conn):
        """Upsert crée puis met à jour."""
        from collections import Counter

        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        counts = Counter({41533: 5, 1: 2})
        rows = WeaponKillsMixin.upsert_weapon_kills(conn, "m123", "x1", counts)
        assert rows == 2

        # Re-upsert avec de nouvelles valeurs
        counts2 = Counter({41533: 8})
        rows2 = WeaponKillsMixin.upsert_weapon_kills(conn, "m123", "x1", counts2)
        assert rows2 == 1

        # Vérifier update
        val = conn.execute(
            "SELECT kills FROM weapon_kills WHERE match_id='m123' AND weapon_id=41533"
        ).fetchone()[0]
        assert val == 8

    def test_upsert_empty_counter(self, conn):
        from collections import Counter

        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        assert WeaponKillsMixin.upsert_weapon_kills(conn, "m1", "x1", Counter()) == 0

    def test_mark_backfill_done(self, conn):
        from src.data.repositories._weapon_kills_repo import (
            _WEAPON_KILLS_BIT,
            WeaponKillsMixin,
        )

        conn.execute("INSERT INTO match_registry (match_id) VALUES ('m_test')")
        WeaponKillsMixin.mark_weapon_backfill_done(conn, "m_test")
        val = conn.execute(
            "SELECT backfill_completed FROM match_registry WHERE match_id='m_test'"
        ).fetchone()[0]
        assert val & _WEAPON_KILLS_BIT == _WEAPON_KILLS_BIT

    def test_get_matches_missing_weapons(self, conn):
        from src.data.repositories._weapon_kills_repo import (
            _WEAPON_KILLS_BIT,
            WeaponKillsMixin,
        )

        conn.execute("INSERT INTO match_registry VALUES ('m1', 0, CURRENT_TIMESTAMP, FALSE)")
        conn.execute(
            "INSERT INTO match_registry VALUES ('m2', ?, CURRENT_TIMESTAMP, FALSE)",
            (_WEAPON_KILLS_BIT,),
        )
        conn.execute("INSERT INTO match_participants VALUES ('m1', 'x1')")
        conn.execute("INSERT INTO match_participants VALUES ('m2', 'x1')")

        missing = WeaponKillsMixin.get_matches_missing_weapons(conn, "x1", limit=10)
        assert "m1" in missing
        assert "m2" not in missing  # déjà backfillé

    def test_get_matches_missing_weapons_force(self, conn):
        from src.data.repositories._weapon_kills_repo import (
            _WEAPON_KILLS_BIT,
            WeaponKillsMixin,
        )

        conn.execute(
            "INSERT INTO match_registry VALUES ('m1', ?, CURRENT_TIMESTAMP, FALSE)",
            (_WEAPON_KILLS_BIT,),
        )
        conn.execute("INSERT INTO match_participants VALUES ('m1', 'x1')")

        missing = WeaponKillsMixin.get_matches_missing_weapons(conn, "x1", limit=10, force=True)
        assert "m1" in missing  # --force inclut les déjà backfillés

    def test_get_xuid_by_gamertag(self, conn):
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        conn.execute("INSERT INTO xuid_aliases VALUES ('xuid_abc', 'TestPlayer')")
        assert WeaponKillsMixin.get_xuid_by_gamertag(conn, "testplayer") == "xuid_abc"

    def test_get_xuid_by_gamertag_not_found(self, conn):
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        assert WeaponKillsMixin.get_xuid_by_gamertag(conn, "UnknownPlayer") is None
