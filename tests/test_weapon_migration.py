"""Tests migration reconciled_as et vue v_weapon_kills."""

from __future__ import annotations

import duckdb
import pytest


@pytest.fixture
def shared_conn():
    """DB DuckDB in-memory avec schéma weapon_kills initial."""
    conn = duckdb.connect(":memory:")
    conn.execute(
        "CREATE TABLE weapon_kills ("
        "  match_id VARCHAR NOT NULL,"
        "  xuid VARCHAR NOT NULL,"
        "  time_ms INTEGER NOT NULL,"
        "  weapon_id UBIGINT,"
        "  delta_ms INTEGER,"
        "  confidence VARCHAR NOT NULL DEFAULT 'none',"
        "  swap_detected BOOLEAN NOT NULL DEFAULT FALSE,"
        "  delayed_damage BOOLEAN NOT NULL DEFAULT FALSE"
        ")"
    )
    yield conn
    conn.close()


class TestMigrationReconciledAs:
    def test_adds_reconciled_as_column(self, shared_conn):
        from src.data.sync.migrations import ensure_weapon_kills_reconciled_as

        ensure_weapon_kills_reconciled_as(shared_conn)
        cols = {
            r[0]
            for r in shared_conn.execute(
                "SELECT column_name FROM information_schema.columns "
                "WHERE table_name = 'weapon_kills'"
            ).fetchall()
        }
        assert "reconciled_as" in cols

    def test_adds_attribution_path_column(self, shared_conn):
        from src.data.sync.migrations import ensure_weapon_kills_reconciled_as

        ensure_weapon_kills_reconciled_as(shared_conn)
        cols = {
            r[0]
            for r in shared_conn.execute(
                "SELECT column_name FROM information_schema.columns "
                "WHERE table_name = 'weapon_kills'"
            ).fetchall()
        }
        assert "attribution_path" in cols

    def test_adds_player_index_column(self, shared_conn):
        from src.data.sync.migrations import ensure_weapon_kills_reconciled_as

        ensure_weapon_kills_reconciled_as(shared_conn)
        cols = {
            r[0]
            for r in shared_conn.execute(
                "SELECT column_name FROM information_schema.columns "
                "WHERE table_name = 'weapon_kills'"
            ).fetchall()
        }
        assert "player_index" in cols

    def test_creates_v_weapon_kills_view(self, shared_conn):
        from src.data.sync.migrations import ensure_weapon_kills_reconciled_as

        ensure_weapon_kills_reconciled_as(shared_conn)
        views = {
            r[0]
            for r in shared_conn.execute(
                "SELECT table_name FROM information_schema.tables WHERE table_type = 'VIEW'"
            ).fetchall()
        }
        assert "v_weapon_kills" in views

    def test_effective_weapon_id_coalesce(self, shared_conn):
        from src.data.sync.migrations import ensure_weapon_kills_reconciled_as

        ensure_weapon_kills_reconciled_as(shared_conn)
        # Insert row avec weapon_id=100, reconciled_as=200
        shared_conn.execute(
            "INSERT INTO weapon_kills (match_id, xuid, time_ms, weapon_id, "
            "reconciled_as, confidence) VALUES (?, ?, ?, ?, ?, ?)",
            ("m1", "x1", 1000, 100, 200, "low"),
        )
        row = shared_conn.execute("SELECT effective_weapon_id FROM v_weapon_kills").fetchone()
        assert row[0] == 200  # COALESCE(200, 100) = 200

    def test_effective_weapon_id_fallback(self, shared_conn):
        from src.data.sync.migrations import ensure_weapon_kills_reconciled_as

        ensure_weapon_kills_reconciled_as(shared_conn)
        # Insert row avec weapon_id=100, reconciled_as=NULL
        shared_conn.execute(
            "INSERT INTO weapon_kills (match_id, xuid, time_ms, weapon_id, "
            "confidence) VALUES (?, ?, ?, ?, ?)",
            ("m1", "x1", 1000, 100, "high"),
        )
        row = shared_conn.execute("SELECT effective_weapon_id FROM v_weapon_kills").fetchone()
        assert row[0] == 100  # COALESCE(NULL, 100) = 100

    def test_idempotent(self, shared_conn):
        from src.data.sync.migrations import ensure_weapon_kills_reconciled_as

        ensure_weapon_kills_reconciled_as(shared_conn)
        ensure_weapon_kills_reconciled_as(shared_conn)  # 2ème appel → pas d'erreur
        cols = {
            r[0]
            for r in shared_conn.execute(
                "SELECT column_name FROM information_schema.columns "
                "WHERE table_name = 'weapon_kills'"
            ).fetchall()
        }
        assert "reconciled_as" in cols


class TestInsertWeaponKillRowsV2:
    def test_inserts_with_reconciled_as(self, shared_conn):
        from src.analysis._kill_attribution import KillAttribution
        from src.data.sync.migrations import ensure_weapon_kills_reconciled_as

        ensure_weapon_kills_reconciled_as(shared_conn)
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        attrs = [
            KillAttribution(
                match_id="m1",
                xuid="x1",
                time_ms=1000,
                weapon_id=42,
                reconciled_as=99,
                delta_ms=500,
                confidence="high",
                attribution_path="fire_event",
                swap_detected=False,
                delayed_damage=False,
                player_index=1,
                source_chunk_idx=0,
            ),
        ]
        rows = WeaponKillsMixin.insert_weapon_kill_rows_v2(shared_conn, "m1", attrs)
        assert rows == 1
        result = shared_conn.execute(
            "SELECT reconciled_as, attribution_path, player_index "
            "FROM weapon_kills WHERE match_id = 'm1'"
        ).fetchone()
        assert result == (99, "fire_event", 1)

    def test_idempotent_delete_insert(self, shared_conn):
        from src.analysis._kill_attribution import KillAttribution
        from src.data.sync.migrations import ensure_weapon_kills_reconciled_as

        ensure_weapon_kills_reconciled_as(shared_conn)
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        attrs = [
            KillAttribution(
                match_id="m1",
                xuid="x1",
                time_ms=1000,
                weapon_id=42,
                reconciled_as=None,
                delta_ms=500,
                confidence="high",
                attribution_path="fire_event",
                swap_detected=False,
                delayed_damage=False,
                player_index=1,
                source_chunk_idx=0,
            ),
        ]
        WeaponKillsMixin.insert_weapon_kill_rows_v2(shared_conn, "m1", attrs)
        # 2ème appel → DELETE + INSERT, pas de doublon
        WeaponKillsMixin.insert_weapon_kill_rows_v2(shared_conn, "m1", attrs)
        count = shared_conn.execute(
            "SELECT COUNT(*) FROM weapon_kills WHERE match_id = 'm1'"
        ).fetchone()[0]
        assert count == 1

    def test_quality_gate(self, shared_conn):
        from src.analysis._kill_attribution import KillAttribution
        from src.data.sync.migrations import ensure_weapon_kills_reconciled_as

        ensure_weapon_kills_reconciled_as(shared_conn)
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        # Insert 2 bons résultats existants
        good_attrs = [
            KillAttribution(
                match_id="m1",
                xuid=f"x{i}",
                time_ms=1000 + i,
                weapon_id=42,
                reconciled_as=None,
                delta_ms=500,
                confidence="high",
                attribution_path="fire_event",
                swap_detected=False,
                delayed_damage=False,
                player_index=1,
                source_chunk_idx=0,
            )
            for i in range(2)
        ]
        WeaponKillsMixin.insert_weapon_kill_rows_v2(shared_conn, "m1", good_attrs)

        # Tentative d'insérer 1 seul bon → skip (new_good=1 <= existing_good=2)
        worse = [
            KillAttribution(
                match_id="m1",
                xuid="x0",
                time_ms=1000,
                weapon_id=42,
                reconciled_as=None,
                delta_ms=500,
                confidence="high",
                attribution_path="fire_event",
                swap_detected=False,
                delayed_damage=False,
                player_index=1,
                source_chunk_idx=0,
            ),
        ]
        rows = WeaponKillsMixin.insert_weapon_kill_rows_v2(shared_conn, "m1", worse)
        assert rows == 0  # skip

    def test_view_effective_weapon_id(self, shared_conn):
        from src.analysis._kill_attribution import KillAttribution
        from src.data.sync.migrations import ensure_weapon_kills_reconciled_as

        ensure_weapon_kills_reconciled_as(shared_conn)
        from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

        attrs = [
            KillAttribution(
                match_id="m1",
                xuid="x1",
                time_ms=1000,
                weapon_id=100,
                reconciled_as=200,
                delta_ms=500,
                confidence="low",
                attribution_path="formula_a",
                swap_detected=False,
                delayed_damage=False,
                player_index=1,
                source_chunk_idx=0,
            ),
        ]
        WeaponKillsMixin.insert_weapon_kill_rows_v2(shared_conn, "m1", attrs)
        eff = shared_conn.execute(
            "SELECT effective_weapon_id FROM v_weapon_kills WHERE match_id = 'm1'"
        ).fetchone()[0]
        assert eff == 200
