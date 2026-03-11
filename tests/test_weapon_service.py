"""Tests unitaires — WeaponExtractionService + WeaponKillsMixin.

Schéma v5.7 per-kill : weapon_id UBIGINT, time_ms, delta_ms, confidence, …
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
        weapon_id      UBIGINT,
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
        assert summary["error"] == "aucun chunk téléchargé (404 ou expiré)"

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
        from src.analysis._weapon_data import WEAPON_NAME_TO_INT

        br75_id = WEAPON_NAME_TO_INT["BR75"]
        return [
            {
                "weapon_id": br75_id,
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

    def test_step4b_handles_grenade_deficit_before_step4a(self, conn):
        """Avec grenade_kills=2 API et 0 détectés, step4b injecte 2 GRENADE avant step4a.

        Avant step4b, on aurait demoted 2 kills en MEDIUM. Maintenant, step4b
        les reclassifie correctement en GRENADE — résultat plus précis.
        """
        from src.analysis._weapon_data import EXCLUDED_WEAPON_IDS, GRENADE_WEAPON_ID

        conn.execute("INSERT INTO match_participants VALUES ('m1','x1',2,0)")
        rows = self._weapon_rows(5, "high", base_delta=100)
        result = WeaponExtractionService._reconcile_api_aggregates(rows, conn, "m1", "x1")
        grenade_count = sum(1 for r in result if r["weapon_id"] == GRENADE_WEAPON_ID)
        weapon_high = sum(
            1
            for r in result
            if r["confidence"] == "high"
            and r["weapon_id"] is not None
            and r["weapon_id"] not in EXCLUDED_WEAPON_IDS
        )
        assert grenade_count == 2  # step4b a injecté exactement 2 GRENADE
        assert weapon_high == 3  # api_weapon_kills = max(5-2-0,0) = 3

    def test_step4b_handles_grenade_largest_delta_first(self, conn):
        """Step4b prend les kills avec le plus grand delta_ms en premier (moins certains)."""
        from src.analysis._weapon_data import GRENADE_WEAPON_ID, WEAPON_NAME_TO_INT

        br75_id = WEAPON_NAME_TO_INT["BR75"]
        conn.execute("INSERT INTO match_participants VALUES ('m1','x1',1,0)")
        rows = [
            {
                "weapon_id": br75_id,
                "confidence": "high",
                "delta_ms": 100,
                "swap_detected": False,
                "delayed_damage": False,
            },
            {
                "weapon_id": br75_id,
                "confidence": "high",
                "delta_ms": 200,
                "swap_detected": False,
                "delayed_damage": False,
            },
            {
                "weapon_id": br75_id,
                "confidence": "high",
                "delta_ms": 999,
                "swap_detected": False,
                "delayed_damage": False,
            },
        ]
        # grenade_kills=1 → step4b injecte 1 GRENADE depuis le kill avec delta=999
        result = WeaponExtractionService._reconcile_api_aggregates(rows, conn, "m1", "x1")
        assert result[2]["weapon_id"] == GRENADE_WEAPON_ID  # delta=999, le plus incertain
        assert result[0]["weapon_id"] == br75_id
        assert result[1]["weapon_id"] == br75_id

    def test_step4c_promotes_medium_to_high(self, conn):
        """Moins de HIGH que api_weapon_kills → promouvoir les MEDIUM (petit delta en premier)."""
        conn.execute("INSERT INTO match_participants VALUES ('m1','x1',0,0)")
        # 3 HIGH + 2 MEDIUM → api_weapon=max(5-0-0,0)=5 → promote 2 MEDIUM
        rows = self._weapon_rows(3, "high", base_delta=100) + self._weapon_rows(
            2, "medium", base_delta=700
        )
        result = WeaponExtractionService._reconcile_api_aggregates(rows, conn, "m1", "x1")
        from src.analysis._weapon_data import EXCLUDED_WEAPON_IDS

        high_count = sum(
            1
            for r in result
            if r.get("confidence") == "high"
            and r.get("weapon_id") is not None
            and r.get("weapon_id") not in EXCLUDED_WEAPON_IDS
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
        from src.analysis._weapon_data import GRENADE_WEAPON_ID, MELEE_WEAPON_ID, WEAPON_NAME_TO_INT

        br75_id = WEAPON_NAME_TO_INT["BR75"]
        conn.execute("INSERT INTO match_participants VALUES ('m1','x1',0,0)")
        rows = [
            {
                "weapon_id": MELEE_WEAPON_ID,
                "confidence": "none",
                "delta_ms": None,
                "swap_detected": False,
                "delayed_damage": False,
            },
            {
                "weapon_id": GRENADE_WEAPON_ID,
                "confidence": "none",
                "delta_ms": None,
                "swap_detected": False,
                "delayed_damage": False,
            },
            {
                "weapon_id": br75_id,
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
        assert k["gamertag"] == "P1"
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
        assert "error" in summary  # aucun kill → erreur attendue

    def test_write_lock_explicit_accepted(self, in_memory_conn, tmp_cache):
        """write_lock=Lock() est accepté sans erreur."""
        lock = asyncio.Lock()
        api = FakeAPIPort()
        service = WeaponExtractionService(api, in_memory_conn, tmp_cache, write_lock=lock)
        summary = asyncio.run(service.process_match("m1234567", "P", "xuid123"))
        assert "error" in summary  # aucun kill → erreur attendue

    def test_write_lock_isolates_concurrent_writes(self, tmp_cache):
        """Deux process_match concurrents avec write_lock partagé n'écrivent pas en double."""
        xuid = "2533274823110022"
        conn = duckdb.connect(":memory:")
        conn.execute(
            "CREATE TABLE weapon_kills (match_id VARCHAR, xuid VARCHAR, time_ms INTEGER, "
            "weapon_id UBIGINT, delta_ms INTEGER, confidence VARCHAR DEFAULT 'none', "
            "swap_detected BOOLEAN DEFAULT FALSE, delayed_damage BOOLEAN DEFAULT FALSE)"
        )
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

        # Bit posé, mais force=True doit ignorer le guard
        conn = self._make_shared_conn(bit_set=True)
        # process_match retourne 0 lignes insérées (match vide suffit)
        mock_instance = MagicMock()
        mock_instance.process_match = AsyncMock(return_value={"rows_inserted": 0})
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

        conn = MagicMock()
        # Simuler une erreur DB sur le guard
        conn.execute.side_effect = Exception("DB error")

        mock_instance = MagicMock()
        mock_instance.process_match = AsyncMock(return_value={"rows_inserted": 3})
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

        conn = self._make_conn_with_bit(bit_set=True)

        mock_service = MagicMock()
        mock_service.process_match = AsyncMock(return_value={"rows_inserted": 99})
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

        conn = self._make_conn_with_bit(bit_set=True)

        mock_service = MagicMock()
        mock_service.process_match = AsyncMock(return_value={"rows_inserted": 5})

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
# Tests directs _step4a_demote / _step4c_promote
# ═══════════════════════════════════════════════════════════════════════════


class TestStep4aStep4cHelpers:
    """Tests unitaires des helpers _step4a_demote et _step4c_promote."""

    @pytest.fixture(autouse=True)
    def _imports(self):
        from src.analysis._weapon_data import EXCLUDED_WEAPON_IDS, WEAPON_NAME_TO_INT
        from src.data.services.weapon_extraction_service import _step4a_demote, _step4c_promote

        self.step4a = _step4a_demote
        self.step4c = _step4c_promote
        self.EXCLUDED = EXCLUDED_WEAPON_IDS
        self.BR75 = WEAPON_NAME_TO_INT["BR75"]
        self.MA40 = WEAPON_NAME_TO_INT["MA40 AR"]

    def _weapon_high_rows(self, n, base_delta=100):
        return [
            {
                "weapon_id": self.BR75,
                "confidence": "high",
                "delta_ms": base_delta * (i + 1),
                "swap_detected": False,
            }
            for i in range(n)
        ]

    def test_step4a_noop_when_exact(self):
        rows = self._weapon_high_rows(3)
        fired = self.step4a(rows, 3, "mX", "xX")
        assert fired is False
        assert all(r["confidence"] == "high" for r in rows)

    def test_step4a_demotes_excess_and_returns_true(self):
        rows = self._weapon_high_rows(5, base_delta=100)  # delta: 100,200,300,400,500
        fired = self.step4a(rows, 3, "mX", "xX")
        assert fired is True
        high_c = sum(1 for r in rows if r["confidence"] == "high")
        medium_c = sum(1 for r in rows if r["confidence"] == "medium")
        assert high_c == 3
        assert medium_c == 2

    def test_step4a_demotes_largest_delta_first(self):
        rows = [
            {"weapon_id": self.BR75, "confidence": "high", "delta_ms": 100, "swap_detected": False},
            {"weapon_id": self.BR75, "confidence": "high", "delta_ms": 999, "swap_detected": False},
            {"weapon_id": self.BR75, "confidence": "high", "delta_ms": 50, "swap_detected": False},
        ]
        self.step4a(rows, 2, "mX", "xX")
        assert rows[1]["confidence"] == "medium"  # delta=999 dégradé en premier
        assert rows[0]["confidence"] == "high"
        assert rows[2]["confidence"] == "high"

    def test_step4c_noop_when_exact(self):
        rows = [
            {"weapon_id": self.BR75, "confidence": "high", "delta_ms": 100, "swap_detected": False},
            {"weapon_id": self.MA40, "confidence": "high", "delta_ms": 200, "swap_detected": False},
        ]
        self.step4c(rows, 2, self.EXCLUDED, "mX", "xX")
        assert all(r["confidence"] == "high" for r in rows)

    def test_step4c_promotes_medium_smallest_delta_first(self):
        rows = [
            {"weapon_id": self.BR75, "confidence": "high", "delta_ms": 100, "swap_detected": False},
            {
                "weapon_id": self.MA40,
                "confidence": "medium",
                "delta_ms": 700,
                "swap_detected": False,
            },
            {
                "weapon_id": self.BR75,
                "confidence": "medium",
                "delta_ms": 300,
                "swap_detected": False,
            },
        ]
        self.step4c(rows, 3, self.EXCLUDED, "mX", "xX")
        high_ids = [r["weapon_id"] for r in rows if r["confidence"] == "high"]
        assert self.BR75 in high_ids  # delta=300, promu en premier
        assert len(high_ids) == 3


# ═══════════════════════════════════════════════════════════════════════════
# Tests _inject_missing_sentinels (Step 4b — fix Chocoboflor)
# ═══════════════════════════════════════════════════════════════════════════


class TestInjectMissingSentinels:
    """Vérifie la reclassification melee/grenade quand les médailles sont absentes."""

    @pytest.fixture(autouse=True)
    def _imports(self):
        from src.analysis._weapon_data import (
            EXCLUDED_WEAPON_IDS,
            GRENADE_WEAPON_ID,
            MELEE_WEAPON_ID,
            WEAPON_NAME_TO_INT,
        )
        from src.data.services.weapon_extraction_service import _inject_missing_sentinels

        self.fn = _inject_missing_sentinels
        self.MELEE = MELEE_WEAPON_ID
        self.GRENADE = GRENADE_WEAPON_ID
        self.EXCLUDED = EXCLUDED_WEAPON_IDS
        self.BR75 = WEAPON_NAME_TO_INT["BR75"]
        self.SIDEKICK = WEAPON_NAME_TO_INT["Mk51 Sidekick"]
        self.MA40 = WEAPON_NAME_TO_INT["MA40 AR"]

    def _row(self, weapon_id, conf="high", delta=200, swap=False):
        return {
            "weapon_id": weapon_id,
            "confidence": conf,
            "delta_ms": delta,
            "swap_detected": swap,
            "delayed_damage": False,
        }

    def test_no_deficit_is_noop(self):
        """Aucun déficit → kill_rows non modifié."""
        rows = [self._row(self.MELEE), self._row(self.BR75)]
        snapshot = [dict(r) for r in rows]
        self.fn(rows, 1, 0, self.MELEE, self.GRENADE, self.EXCLUDED, "mABC", "xABC")
        assert rows == snapshot

    def test_melee_deficit_reclassifies_least_certain(self):
        """2 melee_kills API, 0 détectés → 2 kills weapon reclassifiés en MELEE."""
        rows = [
            self._row(self.SIDEKICK, conf="high", delta=500),
            self._row(self.MA40, conf="high", delta=200),
            self._row(self.BR75, conf="high", delta=100),
        ]
        self.fn(rows, 2, 0, self.MELEE, self.GRENADE, self.EXCLUDED, "m1234567", "x1234567")
        melee_count = sum(1 for r in rows if r["weapon_id"] == self.MELEE)
        assert melee_count == 2
        # Les 2 kills avec les plus grands delta sont reclassifiés (500, 200) → MELEE
        melee_rows = [r for r in rows if r["weapon_id"] == self.MELEE]
        assert all(r["confidence"] == "high" for r in melee_rows)
        assert all(r["swap_detected"] is False for r in melee_rows)

    def test_grenade_deficit_reclassifies_least_certain(self):
        """1 grenade_kill API, 0 détectés → 1 kill reclassifié en GRENADE."""
        rows = [
            self._row(self.SIDEKICK, conf="high", delta=800),
            self._row(self.MA40, conf="high", delta=150),
        ]
        self.fn(rows, 0, 1, self.MELEE, self.GRENADE, self.EXCLUDED, "m1234567", "x1234567")
        grenade_count = sum(1 for r in rows if r["weapon_id"] == self.GRENADE)
        assert grenade_count == 1
        # Le plus grand delta (800) reclassifié
        assert rows[0]["weapon_id"] == self.GRENADE

    def test_chocoboflor_scenario(self):
        """Reproduce le bug Chocoboflor : 12 kills weapon, api_melee=2, api_grenade=1."""
        # Tous les kills étaient weapon HIGH, sans aucune sentinelle détectée
        rows = [
            self._row(self.SIDEKICK, conf="high", delta=d)
            for d in [100, 150, 200, 250, 300, 350, 400, 450, 500, 550, 600, 650]
        ]
        self.fn(rows, 2, 1, self.MELEE, self.GRENADE, self.EXCLUDED, "20fd2c23", "25354691")
        melee_count = sum(1 for r in rows if r["weapon_id"] == self.MELEE)
        grenade_count = sum(1 for r in rows if r["weapon_id"] == self.GRENADE)
        weapon_count = sum(1 for r in rows if r["weapon_id"] == self.SIDEKICK)
        assert melee_count == 2
        assert grenade_count == 1
        assert weapon_count == 9

    def test_priority_low_before_none_before_medium_before_high(self):
        """Ordre de reclassification : low < none < medium < high."""
        rows = [
            self._row(self.BR75, conf="high", delta=100),
            self._row(self.MA40, conf="medium", delta=100),
            self._row(self.SIDEKICK, conf="none", delta=100),
            self._row(self.BR75, conf="low", delta=100),
        ]
        # 1 melee déficit → doit prendre le "low" en premier
        self.fn(rows, 1, 0, self.MELEE, self.GRENADE, self.EXCLUDED, "mX", "xX")
        assert rows[3]["weapon_id"] == self.MELEE  # la ligne "low"
        assert rows[0]["weapon_id"] == self.BR75  # "high" → non touché
        assert rows[1]["weapon_id"] == self.MA40  # "medium" → non touché
        assert rows[2]["weapon_id"] == self.SIDEKICK  # "none" → non touché (low pris en premier)

    def test_high_swap_treated_before_high_no_swap(self):
        """high+swap est reclassifié avant high sans swap (rank équivalent à medium)."""
        rows = [
            self._row(self.BR75, conf="high", delta=100, swap=False),
            self._row(self.MA40, conf="high", delta=100, swap=True),
        ]
        # 1 melee déficit → doit prendre le high+swap en premier
        self.fn(rows, 1, 0, self.MELEE, self.GRENADE, self.EXCLUDED, "mX", "xX")
        assert rows[1]["weapon_id"] == self.MELEE  # high+swap reclassifié
        assert rows[0]["weapon_id"] == self.BR75  # high non touché

    def test_step4b_via_reconcile_flow(self):
        """Intégration : _reconcile_api_aggregates appelle correctement step4b."""
        import duckdb

        conn = duckdb.connect(":memory:")
        conn.execute(
            "CREATE TABLE match_participants "
            "(match_id VARCHAR, xuid VARCHAR, grenade_kills INTEGER, melee_kills INTEGER)"
        )
        conn.execute("INSERT INTO match_participants VALUES ('m1','x1',1,2)")
        # 5 weapon kills tous HIGH, aucune sentinelle détectée
        rows = [
            self._row(wid, conf="high", delta=d)
            for wid, d in [
                (self.SIDEKICK, 673),
                (self.SIDEKICK, 557),
                (self.MA40, 300),
                (self.MA40, 200),
                (self.SIDEKICK, 288),
            ]
        ]
        result = WeaponExtractionService._reconcile_api_aggregates(rows, conn, "m1", "x1")
        melee_count = sum(1 for r in result if r["weapon_id"] == self.MELEE)
        grenade_count = sum(1 for r in result if r["weapon_id"] == self.GRENADE)
        weapon_count = sum(1 for r in result if r["weapon_id"] not in self.EXCLUDED)
        assert melee_count == 2
        assert grenade_count == 1
        assert weapon_count == 2  # api_weapon_kills = max(5-1-2,0) = 2
