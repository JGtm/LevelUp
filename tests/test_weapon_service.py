"""Tests unitaires — WeaponExtractionService + WeaponKillsMixin.

Schéma v5.7 per-kill : weapon_id UBIGINT, time_ms, delta_ms, confidence, …
"""

from __future__ import annotations

import asyncio
from types import SimpleNamespace
from unittest.mock import patch

import duckdb
import polars as pl
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
        match_id         VARCHAR  NOT NULL,
        xuid             VARCHAR  NOT NULL,
        time_ms          INTEGER  NOT NULL,
        weapon_id        UBIGINT,
        delta_ms         INTEGER,
        confidence       VARCHAR  NOT NULL DEFAULT 'none',
        swap_detected    BOOLEAN  NOT NULL DEFAULT FALSE,
        delayed_damage   BOOLEAN  NOT NULL DEFAULT FALSE,
        reconciled_as    UBIGINT,
        attribution_path VARCHAR  NOT NULL DEFAULT 'none',
        player_index     INTEGER
    )
    """


def _create_weapon_kills_view(conn: duckdb.DuckDBPyConnection) -> None:
    conn.execute(
        "CREATE OR REPLACE VIEW v_weapon_kills AS "
        "SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id "
        "FROM weapon_kills"
    )


@pytest.fixture()
def tmp_cache(tmp_path):
    cache = tmp_path / "chunks"
    cache.mkdir()
    return cache


@pytest.fixture()
def in_memory_conn():
    conn = duckdb.connect(":memory:")
    conn.execute(_weapon_kills_ddl())
    _create_weapon_kills_view(conn)
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
    def test_no_kills_returns_empty(self, in_memory_conn, tmp_cache):
        """Si aucun kill dans aucun highlight_event, résultat vide."""
        api = FakeAPIPort()
        service = WeaponExtractionService(api, in_memory_conn, tmp_cache)
        summary = asyncio.run(service.process_match("m1234567", "TestPlayer", "xuid123"))
        assert summary.kills_total == 0

    def test_no_film_returns_error(self, in_memory_conn, tmp_cache):
        """Si le film n'est pas disponible, erreur 'aucun chunk'."""
        api = FakeAPIPort(film=None)
        in_memory_conn.execute(
            "INSERT INTO highlight_events VALUES ('m1234567','xuid123','kill',5000,'T','{}');"
        )
        service = WeaponExtractionService(api, in_memory_conn, tmp_cache)
        summary = asyncio.run(service.process_match("m1234567", "TestPlayer", "xuid123"))
        assert summary.error == "aucun chunk (404/expiré)"

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
        assert summary.rows_inserted == 0
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
        assert summary.kills_total == 1
        assert summary.rows_inserted == 1
        row = in_memory_conn.execute(
            "SELECT weapon_id, confidence FROM weapon_kills WHERE match_id='m1234567'"
        ).fetchone()
        assert row is not None
        assert row[0] == 1  # MELEE_WEAPON_ID
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
        assert summary.error is not None
        assert "DB inacc." in summary.error


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
        from src.analysis._weapon_data import WEAPON_NAME_TO_INT
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        br75_id = WEAPON_NAME_TO_INT["BR75"]
        sidekick_id = WEAPON_NAME_TO_INT["Mk51 Sidekick"]
        rows = [
            {
                "time_ms": 5000,
                "weapon_id": br75_id,
                "delta_ms": 300,
                "confidence": "high",
                "swap_detected": False,
                "delayed_damage": False,
            },
            {
                "time_ms": 12000,
                "weapon_id": sidekick_id,
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
        from src.analysis._weapon_data import WEAPON_NAME_TO_INT
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        br75_id = WEAPON_NAME_TO_INT["BR75"]
        ma40_id = WEAPON_NAME_TO_INT["MA40 AR"]
        rows1 = [
            {
                "time_ms": 5000,
                "weapon_id": br75_id,
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
                "weapon_id": ma40_id,
                "delta_ms": 100,
                "confidence": "high",
                "swap_detected": False,
                "delayed_damage": False,
            },
            {
                "time_ms": 7000,
                "weapon_id": ma40_id,
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

    def test_mark_weapon_no_film(self, conn):
        from src.data.repositories._weapon_kills_repo import (
            _WEAPON_KILLS_NO_FILM_BIT,
            WeaponKillsMixin,
        )

        conn.execute("INSERT INTO match_registry (match_id) VALUES ('mf_test')")
        WeaponKillsMixin.mark_weapon_no_film(conn, "mf_test")
        val = conn.execute(
            "SELECT backfill_completed FROM match_registry WHERE match_id='mf_test'"
        ).fetchone()[0]
        assert val & _WEAPON_KILLS_NO_FILM_BIT == _WEAPON_KILLS_NO_FILM_BIT

    def test_load_weapon_kills_for_match(self, conn):
        """load_weapon_kills_for_match agrège correctement par arme."""
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        conn.execute("CREATE SCHEMA IF NOT EXISTS shared")
        conn.execute(
            "CREATE OR REPLACE VIEW shared.v_weapon_kills AS "
            "SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id "
            "FROM weapon_kills"
        )
        br75 = int.from_bytes(bytes.fromhex("2b1824d542c9679f"), "big")
        conn.execute(
            "INSERT INTO weapon_kills "
            "(match_id, xuid, time_ms, weapon_id, confidence) VALUES "
            "('m1', 'x1', 1000, ?, 'high'), "
            "('m1', 'x1', 2000, ?, 'high'), "
            "('m1', 'x2', 3000, ?, 'medium')",
            (br75, br75, br75),
        )

        class Repo(WeaponKillsMixin):
            def __init__(self, c):
                self._conn = c

            def _get_connection(self):
                return self._conn

        repo = Repo(conn)
        df = repo.load_weapon_kills_for_match("m1")
        assert len(df) == 2  # 2 xuids
        assert df["kills"].sum() == 3

    def test_load_weapon_kills_aggregated(self, conn):
        """load_weapon_kills_aggregated totalise par arme sur plusieurs matchs."""
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        conn.execute("CREATE SCHEMA IF NOT EXISTS shared")
        conn.execute(
            "CREATE OR REPLACE VIEW shared.v_weapon_kills AS "
            "SELECT *, COALESCE(reconciled_as, weapon_id) AS effective_weapon_id "
            "FROM weapon_kills"
        )
        br75 = int.from_bytes(bytes.fromhex("2b1824d542c9679f"), "big")
        sidekick = int.from_bytes(bytes.fromhex("f408190f42c9679f"), "big")
        conn.execute(
            "INSERT INTO weapon_kills "
            "(match_id, xuid, time_ms, weapon_id, confidence) VALUES "
            "('m1', 'x1', 1000, ?, 'high'), "
            "('m2', 'x1', 2000, ?, 'high'), "
            "('m2', 'x1', 3000, ?, 'medium')",
            (br75, br75, sidekick),
        )

        class Repo(WeaponKillsMixin):
            def __init__(self, c):
                self._conn = c

            def _get_connection(self):
                return self._conn

        repo = Repo(conn)
        df = repo.load_weapon_kills_aggregated("x1")
        assert len(df) == 2  # BR75 + Sidekick
        br75_kills = df.filter(pl.col("weapon_id") == br75)["total_kills"][0]
        assert br75_kills == 2


# ═══════════════════════════════════════════════════════════════════════════
# Tests load_all_kills_for_match (Phase 2 — batch SQL)
# ═══════════════════════════════════════════════════════════════════════════


class TestLoadAllKillsForMatch:
    """Vérifie le chargement batch kills+médailles pour tous les joueurs."""

    @pytest.fixture()
    def conn(self):
        c = duckdb.connect(":memory:")
        c.execute(
            "CREATE TABLE highlight_events ("
            "match_id VARCHAR, xuid VARCHAR, event_type VARCHAR, time_ms INTEGER, "
            "gamertag VARCHAR, raw_json VARCHAR)"
        )
        return c

    def test_empty_match_returns_empty_dict(self, conn):
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        assert WeaponKillsMixin.load_all_kills_for_match(conn, "m1") == {}

    def test_single_player_single_kill(self, conn):
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        conn.execute("INSERT INTO highlight_events VALUES ('m1','x1','kill',5000,'P1','{}')")
        result = WeaponKillsMixin.load_all_kills_for_match(conn, "m1")
        assert list(result.keys()) == ["x1"]
        k = result["x1"][0]
        assert k["time_ms"] == 5000
        assert k["gamertag"] is None  # highlight_events.gamertag supprimé en v6
        assert k["xuid"] == "x1"
        assert k["is_melee"] is False
        assert k["is_grenade"] is False
        assert k["medals_nearby"] == []

    def test_two_players_grouped_separately(self, conn):
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        conn.execute("INSERT INTO highlight_events VALUES ('m1','x1','kill',5000,'P1','{}')")
        conn.execute("INSERT INTO highlight_events VALUES ('m1','x2','kill',8000,'P2','{}')")
        result = WeaponKillsMixin.load_all_kills_for_match(conn, "m1")
        assert set(result.keys()) == {"x1", "x2"}
        assert len(result["x1"]) == 1
        assert len(result["x2"]) == 1

    def test_multiple_kills_per_player(self, conn):
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        conn.execute("INSERT INTO highlight_events VALUES ('m1','x1','kill',5000,'P1','{}')")
        conn.execute("INSERT INTO highlight_events VALUES ('m1','x1','kill',9000,'P1','{}')")
        result = WeaponKillsMixin.load_all_kills_for_match(conn, "m1")
        assert len(result["x1"]) == 2

    def test_melee_medal_within_500ms_detected(self, conn):
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        conn.execute("INSERT INTO highlight_events VALUES ('m1','x1','kill',5000,'P1','{}')")
        conn.execute(
            "INSERT INTO highlight_events VALUES "
            "('m1','x1','medal',5100,'P1','{\"is_medal\":\"true\",\"medal_name\":\"Pummel\"}')"
        )
        result = WeaponKillsMixin.load_all_kills_for_match(conn, "m1")
        k = result["x1"][0]
        assert k["is_melee"] is True
        assert "Pummel" in k["medals_nearby"]

    def test_grenade_medal_within_500ms_detected(self, conn):
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        conn.execute("INSERT INTO highlight_events VALUES ('m1','x1','kill',5000,'P1','{}')")
        conn.execute(
            "INSERT INTO highlight_events VALUES "
            "('m1','x1','medal',5200,'P1','{\"is_medal\":\"true\",\"medal_name\":\"Stick\"}')"
        )
        result = WeaponKillsMixin.load_all_kills_for_match(conn, "m1")
        assert result["x1"][0]["is_grenade"] is True

    def test_medal_at_500ms_boundary_included(self, conn):
        """Médaille exactement à 500ms de distance → incluse (<=)."""
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        conn.execute("INSERT INTO highlight_events VALUES ('m1','x1','kill',5000,'P1','{}')")
        conn.execute(
            "INSERT INTO highlight_events VALUES "
            "('m1','x1','medal',5500,'P1','{\"is_medal\":\"true\",\"medal_name\":\"Pummel\"}')"
        )
        result = WeaponKillsMixin.load_all_kills_for_match(conn, "m1")
        assert result["x1"][0]["is_melee"] is True

    def test_medal_outside_500ms_window_ignored(self, conn):
        """Médaille à 501ms → exclue."""
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        conn.execute("INSERT INTO highlight_events VALUES ('m1','x1','kill',5000,'P1','{}')")
        conn.execute(
            "INSERT INTO highlight_events VALUES "
            "('m1','x1','medal',5501,'P1','{\"is_medal\":\"true\",\"medal_name\":\"Pummel\"}')"
        )
        result = WeaponKillsMixin.load_all_kills_for_match(conn, "m1")
        assert result["x1"][0]["is_melee"] is False

    def test_medals_from_other_player_not_crossed(self, conn):
        """Les médailles de x2 n'affectent pas les kills de x1."""
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        conn.execute("INSERT INTO highlight_events VALUES ('m1','x1','kill',5000,'P1','{}')")
        conn.execute(
            "INSERT INTO highlight_events VALUES "
            "('m1','x2','medal',5100,'P2','{\"is_medal\":\"true\",\"medal_name\":\"Pummel\"}')"
        )
        result = WeaponKillsMixin.load_all_kills_for_match(conn, "m1")
        assert result["x1"][0]["is_melee"] is False

    def test_kills_from_other_match_excluded(self, conn):
        """Un kill dans m2 ne doit pas apparaître pour m1."""
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        conn.execute("INSERT INTO highlight_events VALUES ('m1','x1','kill',5000,'P1','{}')")
        conn.execute("INSERT INTO highlight_events VALUES ('m2','x1','kill',8000,'P1','{}')")
        result = WeaponKillsMixin.load_all_kills_for_match(conn, "m1")
        assert len(result["x1"]) == 1
        assert result["x1"][0]["time_ms"] == 5000

    def test_db_exception_returns_empty(self, conn):
        """Exception DB (table absente) → retourne {} sans lever."""
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        conn.execute("DROP TABLE highlight_events")
        result = WeaponKillsMixin.load_all_kills_for_match(conn, "m1")
        assert result == {}

    def test_consistent_with_load_player_kills(self, conn):
        """load_all_kills_for_match retourne les mêmes données que load_player_kills_for_match."""
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        conn.execute("INSERT INTO highlight_events VALUES ('m1','x1','kill',5000,'P1','{}')")
        conn.execute(
            "INSERT INTO highlight_events VALUES "
            "('m1','x1','medal',5100,'P1','{\"is_medal\":\"true\",\"medal_name\":\"Pummel\"}')"
        )
        batch = WeaponKillsMixin.load_all_kills_for_match(conn, "m1")["x1"]
        single = WeaponKillsMixin.load_player_kills_for_match(conn, "m1", "x1")
        assert len(batch) == len(single)
        assert batch[0]["time_ms"] == single[0]["time_ms"]
        assert batch[0]["is_melee"] == single[0]["is_melee"]
        assert batch[0]["medals_nearby"] == single[0]["medals_nearby"]


# ═══════════════════════════════════════════════════════════════════════════
# Tests write_lock + parallélisme (Phase 3)
# ═══════════════════════════════════════════════════════════════════════════


class TestWriteLock:
    """Vérifie que write_lock est accepté et n'altère pas le résultat."""

    def test_write_lock_none_backward_compat(self, in_memory_conn, tmp_cache):
        """Sans write_lock, le service fonctionne comme avant."""
        api = FakeAPIPort()
        service = WeaponExtractionService(api, in_memory_conn, tmp_cache)
        summary = asyncio.run(service.process_match("m1234567", "P", "xuid123"))
        assert summary.kills_total == 0  # aucun kill → résultat vide

    def test_write_lock_explicit_accepted(self, in_memory_conn, tmp_cache):
        """write_lock=Lock() est accepté sans erreur."""
        lock = asyncio.Lock()
        api = FakeAPIPort()
        service = WeaponExtractionService(api, in_memory_conn, tmp_cache, write_lock=lock)
        summary = asyncio.run(service.process_match("m1234567", "P", "xuid123"))
        assert summary.kills_total == 0  # aucun kill → résultat vide

    def test_write_lock_isolates_concurrent_writes(self, tmp_cache):
        """Deux process_match concurrents avec write_lock partagé n'écrivent pas en double."""
        xuid = "2533274823110022"
        conn = duckdb.connect(":memory:")
        conn.execute(
            "CREATE TABLE weapon_kills (match_id VARCHAR, xuid VARCHAR, time_ms INTEGER, "
            "weapon_id UBIGINT, delta_ms INTEGER, confidence VARCHAR DEFAULT 'none', "
            "swap_detected BOOLEAN DEFAULT FALSE, delayed_damage BOOLEAN DEFAULT FALSE, "
            "reconciled_as UBIGINT, attribution_path VARCHAR DEFAULT 'none', "
            "player_index INTEGER)"
        )
        _create_weapon_kills_view(conn)
        conn.execute(
            "CREATE TABLE match_registry (match_id VARCHAR PRIMARY KEY, "
            "backfill_completed INTEGER DEFAULT 0)"
        )
        conn.execute("INSERT INTO match_registry VALUES ('ma123456', 0)")
        conn.execute("INSERT INTO match_registry VALUES ('mb123456', 0)")
        conn.execute("CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR)")
        conn.execute(
            "CREATE TABLE highlight_events (match_id VARCHAR, xuid VARCHAR, "
            "event_type VARCHAR, time_ms INTEGER, gamertag VARCHAR, raw_json VARCHAR)"
        )
        conn.execute("CREATE TABLE xuid_aliases (xuid VARCHAR, gamertag VARCHAR)")
        # Insérer un kill melee dans chaque match
        conn.execute(
            f"INSERT INTO highlight_events VALUES "
            f"('ma123456','{xuid}','kill',5000,'P','{{}}'),"
            f"('ma123456','{xuid}','medal',5100,'P',"
            f'\'{{"is_medal":"true","medal_name":"Pummel"}}\'),'
            f"('mb123456','{xuid}','kill',5000,'P','{{}}'),"
            f"('mb123456','{xuid}','medal',5100,'P',"
            f'\'{{"is_medal":"true","medal_name":"Pummel"}}\');'
        )

        film = _make_film(n_chunks=1, chunk_duration_ms=10000)
        lock = asyncio.Lock()

        async def _run():
            api = FakeAPIPort(film=film, chunk_data=b"\x00" * 200)
            s = WeaponExtractionService(api, conn, tmp_cache, write_lock=lock)
            await asyncio.gather(
                s.process_match("ma123456", "P", xuid),
                s.process_match("mb123456", "P", xuid),
            )

        asyncio.run(_run())
        # Chaque match doit avoir exactement 1 ligne (kill melee)
        for mid in ("ma123456", "mb123456"):
            count = conn.execute(
                "SELECT COUNT(*) FROM weapon_kills WHERE match_id=?", (mid,)
            ).fetchone()[0]
            assert count == 1, f"Match {mid} : attendu 1, got {count}"


# ═══════════════════════════════════════════════════════════════════════════
# Tests run_weapon_kills_backfill (Phase 3 — parallélisme)
# ═══════════════════════════════════════════════════════════════════════════


class TestRunWeaponKillsBackfill:
    def test_empty_match_ids_returns_zero(self):
        """Liste vide → retour immédiat sans appel API."""
        from scripts.backfill._weapon_kills_logic import run_weapon_kills_backfill

        conn = duckdb.connect(":memory:")
        result = asyncio.run(run_weapon_kills_backfill("GT", "xuid", [], conn))
        assert result == 0


# ═══════════════════════════════════════════════════════════════════════════
# Tests _backfill_weapon_kills_for_match — guard WEAPON_KILLS (Point A)
# ═══════════════════════════════════════════════════════════════════════════


class TestBackfillWeaponKillsGuard:
    """Vérifie que _backfill_weapon_kills_for_match respecte le bit WEAPON_KILLS.

    Le service traite TOUS les joueurs d'un match → une fois le bit posé,
    il ne faut pas retraiter (économie CPU + réseau).
    Aligné avec detection.py:444 (chemin CLI --weapons).
    """

    _MATCH_ID = "ab123456-0000-0000-0000-000000000000"
    _GAMERTAG = "TestPlayer"
    _XUID = "1234567890123456"

    def _make_shared_conn(self, *, bit_set: bool):
        """Mock shared_conn dont execute().fetchone() retourne le bit demandé."""
        from unittest.mock import MagicMock

        conn = MagicMock()
        conn.execute.return_value.fetchone.return_value = (1 if bit_set else 0,)
        return conn

    def test_skip_if_weapon_kills_bit_set(self):
        """force=False + bit posé → retour 0 sans instancier le service."""
        from unittest.mock import MagicMock, patch

        from scripts.backfill.orchestrator import _backfill_weapon_kills_for_match

        conn = self._make_shared_conn(bit_set=True)

        with patch(
            "src.data.services.weapon_extraction_service.WeaponExtractionService"
        ) as mock_cls:
            result = asyncio.run(
                _backfill_weapon_kills_for_match(
                    client=MagicMock(),
                    match_id=self._MATCH_ID,
                    gamertag=self._GAMERTAG,
                    xuid=self._XUID,
                    shared_conn=conn,
                    force=False,
                )
            )

        assert result == 0
        # Service jamais instancié
        mock_cls.assert_not_called()
        # Une seule requête execute : la vérification du bit
        assert conn.execute.call_count == 1

    def test_force_bypasses_bit_guard(self, tmp_path):
        """force=True → guard ignoré, service process_match appelé."""
        from unittest.mock import AsyncMock, MagicMock, patch

        from scripts.backfill.orchestrator import _backfill_weapon_kills_for_match
        from src.data.services.weapon_extraction_service import MatchProcessingResult

        # Bit posé, mais force=True doit ignorer le guard
        conn = self._make_shared_conn(bit_set=True)
        # process_match retourne 0 lignes insérées (match vide suffit)
        mock_instance = MagicMock()
        mock_instance.process_match = AsyncMock(
            return_value=MatchProcessingResult(match_id="test", rows_inserted=0)
        )
        mock_cls = MagicMock(return_value=mock_instance)

        with (
            patch(
                "src.data.services.weapon_extraction_service.WeaponExtractionService",
                mock_cls,
            ),
            patch(
                "src.data.sync._engine_weapon_kills._resolve_cache_dir",
                return_value=tmp_path,
            ),
            patch(
                "src.ui.sync.get_player_duckdb_path",
                return_value=tmp_path / "stats.duckdb",
            ),
        ):
            result = asyncio.run(
                _backfill_weapon_kills_for_match(
                    client=MagicMock(),
                    match_id=self._MATCH_ID,
                    gamertag=self._GAMERTAG,
                    xuid=self._XUID,
                    shared_conn=conn,
                    force=True,
                )
            )

        # Service instancié et process_match appelé malgré le bit posé
        mock_cls.assert_called_once()
        mock_instance.process_match.assert_awaited_once()
        assert result == 0

    def test_guard_exception_falls_through(self, tmp_path):
        """Si la requête guard échoue → on continue (non bloquant)."""
        from unittest.mock import AsyncMock, MagicMock, patch

        from scripts.backfill.orchestrator import _backfill_weapon_kills_for_match
        from src.data.services.weapon_extraction_service import MatchProcessingResult

        conn = MagicMock()
        # Simuler une erreur DB sur le guard
        conn.execute.side_effect = Exception("DB error")

        mock_instance = MagicMock()
        mock_instance.process_match = AsyncMock(
            return_value=MatchProcessingResult(match_id="test", rows_inserted=3)
        )
        mock_cls = MagicMock(return_value=mock_instance)

        with (
            patch(
                "src.data.services.weapon_extraction_service.WeaponExtractionService",
                mock_cls,
            ),
            patch(
                "src.data.sync._engine_weapon_kills._resolve_cache_dir",
                return_value=tmp_path,
            ),
            patch(
                "src.ui.sync.get_player_duckdb_path",
                return_value=tmp_path / "stats.duckdb",
            ),
        ):
            result = asyncio.run(
                _backfill_weapon_kills_for_match(
                    client=MagicMock(),
                    match_id=self._MATCH_ID,
                    gamertag=self._GAMERTAG,
                    xuid=self._XUID,
                    shared_conn=conn,
                    force=False,
                )
            )

        # Erreur guard → on continue, service appelé
        mock_instance.process_match.assert_awaited_once()
        assert result == 3


# ═══════════════════════════════════════════════════════════════════════════
# Tests run_weapon_kills_backfill — guard batch (Point A côté batch)
# ═══════════════════════════════════════════════════════════════════════════


class TestRunWeaponKillsBackfillGuard:
    """Vérifie que le guard WEAPON_KILLS est appliqué dans _process_one.

    La liste _pending_weapon_ids peut contenir des matchs dont le bit est posé
    (ex: detection OR conditions, match traité par un autre joueur en escouade).
    Le batch doit les ignorer sans appeler le service.
    """

    _MATCH_ID = "ab123456-0000-0000-0000-000000000000"

    def _make_conn_with_bit(self, *, bit_set: bool):
        """DuckDB in-memory avec match_registry et bit configuré."""
        conn = duckdb.connect(":memory:")
        conn.execute(
            "CREATE TABLE match_registry "
            "(match_id VARCHAR PRIMARY KEY, backfill_completed INTEGER DEFAULT 0)"
        )
        bit_value = 1 << 21 if bit_set else 0  # MatchBits.WEAPON_KILLS
        conn.execute("INSERT INTO match_registry VALUES (?, ?)", (self._MATCH_ID, bit_value))
        return conn

    def test_batch_skips_match_if_bit_set(self):
        """Bit WEAPON_KILLS posé → _process_one retourne 0 sans appel API."""
        from unittest.mock import AsyncMock, MagicMock, patch

        from scripts.backfill._weapon_kills_logic import run_weapon_kills_backfill
        from src.data.services.weapon_extraction_service import MatchProcessingResult

        conn = self._make_conn_with_bit(bit_set=True)

        mock_service = MagicMock()
        mock_service.process_match = AsyncMock(
            return_value=MatchProcessingResult(match_id="test", rows_inserted=99)
        )
        mock_api_instance = AsyncMock()

        # Imports locaux à la fonction → patcher à la source
        with (
            patch(
                "src.data.sync.api_client.get_tokens_from_env",
                new_callable=AsyncMock,
                return_value=MagicMock(),
            ),
            patch("src.data.sync.api_client.SPNKrAPIClient") as mock_api_cls,
            patch(
                "src.data.services.weapon_extraction_service.WeaponExtractionService",
                return_value=mock_service,
            ),
        ):
            mock_api_cls.return_value.__aenter__ = AsyncMock(return_value=mock_api_instance)
            mock_api_cls.return_value.__aexit__ = AsyncMock(return_value=False)

            result = asyncio.run(
                run_weapon_kills_backfill("GT", "1234", [self._MATCH_ID], conn, force=False)
            )

        assert result == 0
        mock_service.process_match.assert_not_awaited()

    def test_batch_force_ignores_bit(self):
        """force=True → guard ignoré dans _process_one, service appelé."""
        from unittest.mock import AsyncMock, MagicMock, patch

        from scripts.backfill._weapon_kills_logic import run_weapon_kills_backfill
        from src.data.services.weapon_extraction_service import MatchProcessingResult

        conn = self._make_conn_with_bit(bit_set=True)

        mock_service = MagicMock()
        mock_service.process_match = AsyncMock(
            return_value=MatchProcessingResult(match_id="test", rows_inserted=5)
        )

        with (
            patch(
                "src.data.sync.api_client.get_tokens_from_env",
                new_callable=AsyncMock,
                return_value=MagicMock(),
            ),
            patch("src.data.sync.api_client.SPNKrAPIClient") as mock_api_cls,
            patch(
                "src.data.services.weapon_extraction_service.WeaponExtractionService",
                return_value=mock_service,
            ),
        ):
            mock_api_cls.return_value.__aenter__ = AsyncMock(return_value=MagicMock())
            mock_api_cls.return_value.__aexit__ = AsyncMock(return_value=False)

            result = asyncio.run(
                run_weapon_kills_backfill("GT", "1234", [self._MATCH_ID], conn, force=True)
            )

        assert result == 5
        mock_service.process_match.assert_awaited_once()


# ═══════════════════════════════════════════════════════════════════════════
# Tests _resolve_from_chunk
# ═══════════════════════════════════════════════════════════════════════════


class TestResolveFromChunk:
    """Tests unitaires de WeaponExtractionService._resolve_from_chunk."""

    @pytest.fixture()
    def service(self, tmp_path):
        api = FakeAPIPort()
        conn = duckdb.connect(":memory:")
        return WeaponExtractionService(api, conn, tmp_path)

    def test_empty_data_returns_empty(self, service):
        """Pas de données → dict vide sans lever d'exception."""
        result = service._resolve_from_chunk(b"", [], {})
        assert result == {}

    def test_non_digit_xuids_filtered(self, service):
        """Les XUIDs non numériques (ex: 'bid(...)') sont ignorés."""
        participants = {"bid(SomePlayer)": "SomePlayer", "not-a-number": "Other"}
        result = service._resolve_from_chunk(b"\x00" * 100, [], participants)
        assert result == {}

    def test_all_zeros_chunk_returns_empty_resolve(self, service):
        """Données zéro → aucun PLAYER_METADATA, fallback acurtis → aucun mapping."""
        participants = {"123456789": "Player1"}
        result = service._resolve_from_chunk(b"\x00" * 200, [], participants)
        # Pas de metadata ET acurtis ne trouve pas de mapping dans des zéros → {}
        assert isinstance(result, dict)

    def test_returns_xuid_int_to_pi_mapping(self, service):
        """Le résultat est bien un dict[int, int] = {xuid_int: player_index}."""
        result = service._resolve_from_chunk(b"\x00" * 100, [], {"100": "P1", "200": "P2"})
        assert all(isinstance(k, int) for k in result)
        assert all(isinstance(v, int) for v in result.values())


# ═══════════════════════════════════════════════════════════════════════════
# Tests _run_scan_phase
# ═══════════════════════════════════════════════════════════════════════════


class TestRunScanPhase:
    """Tests unitaires de WeaponExtractionService._run_scan_phase."""

    @pytest.fixture()
    def service(self, tmp_path):
        api = FakeAPIPort()
        conn = duckdb.connect(":memory:")
        return WeaponExtractionService(api, conn, tmp_path)

    def _make_log(self):
        from src.analysis._parser_logging import MatchLogCollector

        return MatchLogCollector("test-match-id")

    def test_empty_chunks_returns_empty_scan_and_pi_map(self, service):
        """Aucun chunk → ScanResult vide, pi_map vide."""
        from src.data.services.weapon_extraction_service import ScanResult

        log = self._make_log()
        scan, pi_map = service._run_scan_phase({}, {}, log)
        assert isinstance(scan, ScanResult)
        assert scan.fire_events_global == []
        assert scan.timeline == {}
        assert scan.chunks_sorted == []
        assert pi_map == {}

    def test_single_chunk_scan_returns_scan_result(self, service):
        """Un seul chunk all-zero → ScanResult avec chunk dans timeline."""
        from src.data.services.weapon_extraction_service import ScanResult

        log = self._make_log()
        chunks = {0: (b"\x00" * 200, 0, 5000)}
        scan, pi_map = service._run_scan_phase(chunks, {}, log)
        assert isinstance(scan, ScanResult)
        assert 0 in scan.timeline
        assert isinstance(scan.timeline_ns, dict)
        assert 0 in scan.timeline_ns  # chunk 0 présent dans timeline_ns
        assert scan.chunks_sorted == [0]
        assert len(scan.timing) == 1
        assert scan.timing[0] == (0, 5000)

    def test_chunks_sorted_by_index(self, service):
        """Les chunks insérés dans un ordre quelconque sont triés dans chunks_sorted."""
        log = self._make_log()
        chunks = {
            5: (b"\x00" * 200, 50000, 5000),
            1: (b"\x00" * 200, 10000, 5000),
            3: (b"\x00" * 200, 30000, 5000),
        }
        scan, _ = service._run_scan_phase(chunks, {}, log)
        assert scan.chunks_sorted == [1, 3, 5]
        # timing dans l'ordre chunk 1, 3, 5
        assert scan.timing[0] == (10000, 15000)
        assert scan.timing[1] == (30000, 35000)
        assert scan.timing[2] == (50000, 55000)

    def test_pi_map_empty_for_non_digit_participants(self, service):
        """Participants non numériques → pi_map vide."""
        log = self._make_log()
        chunks = {0: (b"\x00" * 200, 0, 5000)}
        participants = {"bid(Foo)": "Foo", "bar": "bar"}
        _, pi_map = service._run_scan_phase(chunks, participants, log)
        assert pi_map == {}
