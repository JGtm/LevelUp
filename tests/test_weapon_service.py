"""Tests unitaires — WeaponExtractionService + WeaponKillsMixin.

Schéma v5.7 per-kill : weapon_name TEXT, time_ms, delta_ms, confidence, …
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
    def __init__(self, *, film=None, chunk_data=None):
        self._film = film
        self._chunk_data = chunk_data or b"\x00" * 100

    async def get_film_by_match_id(self, match_id: str):
        return self._film

    async def download_film_chunk(self, url: str) -> bytes | None:
        return self._chunk_data


def _make_film(*, n_chunks: int = 2, chunk_duration_ms: int = 5000):
    chunks = [
        SimpleNamespace(
            index=i,
            chunk_type=SimpleNamespace(value=2),
            chunk_start_time_offset_milliseconds=i * chunk_duration_ms,
            duration_milliseconds=chunk_duration_ms,
            file_relative_path=f"/chunk_{i:02d}.bin",
        )
        for i in range(n_chunks)
    ]
    return SimpleNamespace(
        blob_storage_path_prefix="https://blob.test/film/",
        custom_data=SimpleNamespace(chunks=chunks),
    )


def _weapon_kills_ddl() -> str:
    return """
    CREATE TABLE weapon_kills (
        match_id       VARCHAR  NOT NULL,
        xuid           VARCHAR  NOT NULL,
        time_ms        INTEGER  NOT NULL,
        weapon_name    VARCHAR  NOT NULL,
        delta_ms       INTEGER,
        confidence     VARCHAR  NOT NULL DEFAULT 'none',
        swap_detected  BOOLEAN  NOT NULL DEFAULT FALSE,
        delayed_damage BOOLEAN  NOT NULL DEFAULT FALSE
    )
    """


@pytest.fixture()
def tmp_cache(tmp_path):
    cache = tmp_path / "chunks"
    cache.mkdir()
    return cache


@pytest.fixture()
def in_memory_conn():
    conn = duckdb.connect(":memory:")
    conn.execute(_weapon_kills_ddl())
    conn.execute(
        "CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, "
        "backfill_completed INTEGER DEFAULT 0)"
    )
    conn.execute("INSERT INTO match_registry VALUES ('m1234567', 0)")
    conn.execute("CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR)")
    conn.execute(
        "CREATE TABLE highlight_events ("
        "match_id VARCHAR, xuid VARCHAR, event_type VARCHAR, time_ms INTEGER, "
        "gamertag VARCHAR, raw_json VARCHAR)"
    )
    conn.execute("CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR)")
    return conn


# ═══════════════════════════════════════════════════════════════════════════
# Tests WeaponExtractionService
# ═══════════════════════════════════════════════════════════════════════════


class TestWeaponExtractionService:
    def test_no_kills_returns_error(self, in_memory_conn, tmp_cache):
        """Si aucun kill dans aucun highlight_event, erreur renvoyée."""
        api = FakeAPIPort()
        service = WeaponExtractionService(api, in_memory_conn, tmp_cache)
        summary = asyncio.run(service.process_match("m1234567", "TestPlayer", "xuid123"))
        assert "error" in summary
        assert summary["kills_total"] == 0

    def test_no_film_returns_error(self, in_memory_conn, tmp_cache):
        """Si le film n'est pas disponible, erreur 'aucun chunk'."""
        api = FakeAPIPort(film=None)
        in_memory_conn.execute(
            "INSERT INTO highlight_events VALUES ('m1234567','xuid123','kill',5000,'T','{}');"
        )
        service = WeaponExtractionService(api, in_memory_conn, tmp_cache)
        summary = asyncio.run(service.process_match("m1234567", "TestPlayer", "xuid123"))
        assert summary["error"] == "aucun chunk téléchargé"

    def test_dry_run_no_write(self, in_memory_conn, tmp_cache):
        """En dry_run, aucune écriture DB."""
        film = _make_film(n_chunks=1, chunk_duration_ms=10000)
        api = FakeAPIPort(film=film, chunk_data=b"\x00" * 200)
        in_memory_conn.execute(
            "INSERT INTO highlight_events VALUES ('m1234567','xuid123','kill',5000,'T','{}');"
        )
        service = WeaponExtractionService(api, in_memory_conn, tmp_cache)
        summary = asyncio.run(
            service.process_match("m1234567", "TestPlayer", "xuid123", dry_run=True)
        )
        assert summary["rows_inserted"] == 0
        count = in_memory_conn.execute("SELECT COUNT(*) FROM weapon_kills").fetchone()[0]
        assert count == 0

    def test_process_match_writes_rows(self, in_memory_conn, tmp_cache):
        """Un match avec kill melee écrit une ligne per-kill en DB."""
        # xuid doit être numérique (format réel Halo)
        xuid = "2533274823110022"
        film = _make_film(n_chunks=1, chunk_duration_ms=10000)
        api = FakeAPIPort(film=film, chunk_data=b"\x00" * 200)
        in_memory_conn.execute(
            f"INSERT INTO highlight_events VALUES "
            f"('m1234567','{xuid}','kill',5000,'T','{{}}'), "
            f"('m1234567','{xuid}','medal',5100,'T',"
            f'\'{{"is_medal":"true","medal_name":"Pummel"}}\');'
        )
        service = WeaponExtractionService(api, in_memory_conn, tmp_cache)
        summary = asyncio.run(service.process_match("m1234567", "TestPlayer", xuid))
        assert summary["kills_total"] == 1
        assert summary["rows_inserted"] == 1
        row = in_memory_conn.execute(
            "SELECT weapon_name, confidence FROM weapon_kills WHERE match_id='m1234567'"
        ).fetchone()
        assert row is not None
        assert row[0] == "MELEE"
        assert row[1] == "none"

    def test_chunk_caching(self, in_memory_conn, tmp_cache):
        """Les chunks téléchargés sont mis en cache sur disque."""
        film = _make_film(n_chunks=1, chunk_duration_ms=10000)
        api = FakeAPIPort(film=film, chunk_data=b"\xab" * 50)
        in_memory_conn.execute(
            "INSERT INTO highlight_events VALUES ('m1234567','xuid123','kill',5000,'T','{}');"
        )
        service = WeaponExtractionService(api, in_memory_conn, tmp_cache)
        asyncio.run(service.process_match("m1234567", "TestPlayer", "xuid123"))
        cache_files = list(tmp_cache.rglob("*.bin"))
        assert len(cache_files) == 1
        assert cache_files[0].read_bytes() == b"\xab" * 50

    def test_exception_handled_gracefully(self, in_memory_conn, tmp_cache):
        """Une exception ne fait pas crasher."""
        api = FakeAPIPort()
        service = WeaponExtractionService(api, in_memory_conn, tmp_cache)
        with patch.object(service, "_load_participants", side_effect=RuntimeError("DB inacc.")):
            summary = asyncio.run(service.process_match("m1234567", "TestPlayer", "xuid123"))
        assert "error" in summary
        assert "DB inacc." in summary["error"]


# ═══════════════════════════════════════════════════════════════════════════
# Tests WeaponKillsMixin
# ═══════════════════════════════════════════════════════════════════════════


class TestWeaponKillsMixinRepo:
    @pytest.fixture()
    def conn(self):
        conn = duckdb.connect(":memory:")
        conn.execute(_weapon_kills_ddl())
        conn.execute(
            "CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, "
            "backfill_completed INTEGER DEFAULT 0, "
            "start_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP, "
            "is_firefight BOOLEAN DEFAULT FALSE)"
        )
        conn.execute("CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR)")
        conn.execute("CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR)")
        return conn

    def test_insert_and_count(self, conn):
        """insert_weapon_kill_rows crée les lignes per-kill."""
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        rows = [
            {
                "time_ms": 5000,
                "weapon_name": "BR75",
                "delta_ms": 300,
                "confidence": "high",
                "swap_detected": False,
                "delayed_damage": False,
            },
            {
                "time_ms": 12000,
                "weapon_name": "Mk51 Sidekick",
                "delta_ms": 200,
                "confidence": "high",
                "swap_detected": False,
                "delayed_damage": False,
            },
        ]
        n = WeaponKillsMixin.insert_weapon_kill_rows(conn, "m123", "x1", rows)
        assert n == 2
        count = conn.execute("SELECT COUNT(*) FROM weapon_kills WHERE match_id='m123'").fetchone()[
            0
        ]
        assert count == 2

    def test_insert_empty_rows(self, conn):
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        assert WeaponKillsMixin.insert_weapon_kill_rows(conn, "m1", "x1", []) == 0

    def test_insert_replaces_previous(self, conn):
        """Un second insert supprime les lignes précédentes (DELETE+INSERT)."""
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        rows1 = [
            {
                "time_ms": 5000,
                "weapon_name": "BR75",
                "delta_ms": 300,
                "confidence": "high",
                "swap_detected": False,
                "delayed_damage": False,
            }
        ]
        WeaponKillsMixin.insert_weapon_kill_rows(conn, "m1", "x1", rows1)
        rows2 = [
            {
                "time_ms": 6000,
                "weapon_name": "MA40 AR",
                "delta_ms": 100,
                "confidence": "high",
                "swap_detected": False,
                "delayed_damage": False,
            },
            {
                "time_ms": 7000,
                "weapon_name": "MA40 AR",
                "delta_ms": 150,
                "confidence": "high",
                "swap_detected": False,
                "delayed_damage": False,
            },
        ]
        WeaponKillsMixin.insert_weapon_kill_rows(conn, "m1", "x1", rows2)
        count = conn.execute(
            "SELECT COUNT(*) FROM weapon_kills WHERE match_id='m1' AND xuid='x1'"
        ).fetchone()[0]
        assert count == 2  # les lignes précédentes sont remplacées

    def test_mark_backfill_done(self, conn):
        from src.data.repositories._weapon_kills_repo import _WEAPON_KILLS_BIT, WeaponKillsMixin

        conn.execute("INSERT INTO match_registry (match_id) VALUES ('m_test')")
        WeaponKillsMixin.mark_weapon_backfill_done(conn, "m_test")
        val = conn.execute(
            "SELECT backfill_completed FROM match_registry WHERE match_id='m_test'"
        ).fetchone()[0]
        assert val & _WEAPON_KILLS_BIT == _WEAPON_KILLS_BIT

    def test_get_matches_missing_weapons(self, conn):
        from src.data.repositories._weapon_kills_repo import _WEAPON_KILLS_BIT, WeaponKillsMixin

        conn.execute("INSERT INTO match_registry VALUES ('m1', 0, CURRENT_TIMESTAMP, FALSE)")
        conn.execute(
            "INSERT INTO match_registry VALUES ('m2', ?, CURRENT_TIMESTAMP, FALSE)",
            (_WEAPON_KILLS_BIT,),
        )
        conn.execute("INSERT INTO match_participants VALUES ('m1', 'x1')")
        conn.execute("INSERT INTO match_participants VALUES ('m2', 'x1')")

        missing = WeaponKillsMixin.get_matches_missing_weapons(conn, "x1", limit=10)
        assert "m1" in missing
        assert "m2" not in missing

    def test_get_matches_missing_weapons_force(self, conn):
        from src.data.repositories._weapon_kills_repo import _WEAPON_KILLS_BIT, WeaponKillsMixin

        conn.execute(
            "INSERT INTO match_registry VALUES ('m1', ?, CURRENT_TIMESTAMP, FALSE)",
            (_WEAPON_KILLS_BIT,),
        )
        conn.execute("INSERT INTO match_participants VALUES ('m1', 'x1')")
        missing = WeaponKillsMixin.get_matches_missing_weapons(conn, "x1", limit=10, force=True)
        assert "m1" in missing

    def test_get_xuid_by_gamertag(self, conn):
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        conn.execute("INSERT INTO xuid_aliases VALUES ('xuid_abc', 'TestPlayer')")
        assert WeaponKillsMixin.get_xuid_by_gamertag(conn, "testplayer") == "xuid_abc"

    def test_get_xuid_by_gamertag_not_found(self, conn):
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        assert WeaponKillsMixin.get_xuid_by_gamertag(conn, "UnknownPlayer") is None


# ═══════════════════════════════════════════════════════════════════════════
# Tests _reconcile_api_aggregates (FINDINGS §6a Steps 4a/4c)
# ═══════════════════════════════════════════════════════════════════════════


class TestReconcileApiAggregates:
    """Tests unitaires directs de la méthode statique _reconcile_api_aggregates."""

    @pytest.fixture()
    def conn(self):
        c = duckdb.connect(":memory:")
        c.execute(
            "CREATE TABLE match_participants ("
            "match_id VARCHAR, xuid VARCHAR, "
            "grenade_kills INTEGER, melee_kills INTEGER)"
        )
        return c

    @staticmethod
    def _weapon_rows(n: int, conf: str = "high", base_delta: int = 100) -> list[dict]:
        return [
            {
                "weapon_name": "BR75",
                "confidence": conf,
                "delta_ms": base_delta * (i + 1),
                "swap_detected": False,
                "delayed_damage": False,
            }
            for i in range(n)
        ]

    def test_no_participant_row_returns_unchanged(self, conn):
        """Sans ligne dans match_participants, kill_rows retourné inchangé."""
        rows = self._weapon_rows(3)
        result = WeaponExtractionService._reconcile_api_aggregates(rows, conn, "m1", "x1")
        assert result is rows  # même objet
        assert all(r["confidence"] == "high" for r in result)

    def test_db_exception_returns_unchanged(self, conn):
        """Exception DB → retourne kill_rows sans modification."""
        # Supprimer la table pour provoquer une erreur
        conn.execute("DROP TABLE match_participants")
        rows = self._weapon_rows(2)
        result = WeaponExtractionService._reconcile_api_aggregates(rows, conn, "m1", "x1")
        assert all(r["confidence"] == "high" for r in result)

    def test_step4a_demotes_excess_high_kills(self, conn):
        """Plus de HIGH que api_weapon_kills → dégrader les moins certains (grand delta_ms)."""
        # kill_rows = 5 weapon HIGH, grenade_kills=2 → api_weapon=max(5-2-0,0)=3 → demote 2
        conn.execute("INSERT INTO match_participants VALUES ('m1','x1',2,0)")
        rows = self._weapon_rows(5, "high", base_delta=100)
        result = WeaponExtractionService._reconcile_api_aggregates(rows, conn, "m1", "x1")
        high_count = sum(1 for r in result if r["confidence"] == "high")
        medium_count = sum(1 for r in result if r["confidence"] == "medium")
        assert high_count == 3
        assert medium_count == 2

    def test_step4a_demotes_highest_delta_first(self, conn):
        """Step 4a : les kills avec le plus grand delta_ms sont dégradés en premier."""
        conn.execute("INSERT INTO match_participants VALUES ('m1','x1',1,0)")
        rows = [
            {
                "weapon_name": "BR75",
                "confidence": "high",
                "delta_ms": 100,
                "swap_detected": False,
                "delayed_damage": False,
            },
            {
                "weapon_name": "BR75",
                "confidence": "high",
                "delta_ms": 200,
                "swap_detected": False,
                "delayed_damage": False,
            },
            {
                "weapon_name": "BR75",
                "confidence": "high",
                "delta_ms": 999,
                "swap_detected": False,
                "delayed_damage": False,
            },
        ]
        # grenade_kills=1, melee_kills=0 → api_weapon=max(3-1-0,0)=2 → demote 1 (le delta=999)
        result = WeaponExtractionService._reconcile_api_aggregates(rows, conn, "m1", "x1")
        assert result[2]["confidence"] == "medium"  # delta=999, le plus grand
        assert result[0]["confidence"] == "high"
        assert result[1]["confidence"] == "high"

    def test_step4c_promotes_medium_to_high(self, conn):
        """Moins de HIGH que api_weapon_kills → promouvoir les MEDIUM (petit delta en premier)."""
        conn.execute("INSERT INTO match_participants VALUES ('m1','x1',0,0)")
        # 3 HIGH + 2 MEDIUM → api_weapon=max(5-0-0,0)=5 → promote 2 MEDIUM
        rows = self._weapon_rows(3, "high", base_delta=100) + self._weapon_rows(
            2, "medium", base_delta=700
        )
        result = WeaponExtractionService._reconcile_api_aggregates(rows, conn, "m1", "x1")
        high_count = sum(
            1
            for r in result
            if r["confidence"] == "high"
            and r["weapon_name"] not in ("MELEE", "GRENADE", "NON TROUVE", "UNKNOWN")
        )
        assert high_count == 5

    def test_no_change_when_counts_match(self, conn):
        """Exact match api_weapon_kills == len(HIGH) → aucune modification."""
        conn.execute("INSERT INTO match_participants VALUES ('m1','x1',0,0)")
        rows = self._weapon_rows(3, "high", base_delta=100)
        result = WeaponExtractionService._reconcile_api_aggregates(rows, conn, "m1", "x1")
        assert all(r["confidence"] == "high" for r in result)

    def test_excluded_names_not_counted(self, conn):
        """MELEE et GRENADE ne comptent pas dans weapon_high."""
        conn.execute("INSERT INTO match_participants VALUES ('m1','x1',0,0)")
        rows = [
            {
                "weapon_name": "MELEE",
                "confidence": "none",
                "delta_ms": None,
                "swap_detected": False,
                "delayed_damage": False,
            },
            {
                "weapon_name": "GRENADE",
                "confidence": "none",
                "delta_ms": None,
                "swap_detected": False,
                "delayed_damage": False,
            },
            {
                "weapon_name": "BR75",
                "confidence": "high",
                "delta_ms": 200,
                "swap_detected": False,
                "delayed_damage": False,
            },
        ]
        # api_weapon = max(3-0-0,0) = 3; weapon_high = 1 (seulement BR75); 1 < 3 → step4c
        # Pas de MEDIUM à promouvoir → résultat inchangé
        result = WeaponExtractionService._reconcile_api_aggregates(rows, conn, "m1", "x1")
        assert result[2]["confidence"] == "high"
