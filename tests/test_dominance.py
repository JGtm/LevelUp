"""Tests unitaires pour la détection de dominance (médaille Steaktacular)."""

from __future__ import annotations

import duckdb
import pytest

from src.analysis._medal_verdicts import (
    MEDAL_STEAKTACULAR_ID,
    DominanceFlag,
)


class TestDominanceFlag:
    """Tests de l'enum DominanceFlag."""

    def test_values(self) -> None:
        assert DominanceFlag.NONE == 0
        assert DominanceFlag.DOMINATION == 1
        assert DominanceFlag.HUMILIATION == 2

    def test_is_int(self) -> None:
        assert isinstance(DominanceFlag.NONE, int)
        assert isinstance(DominanceFlag.DOMINATION, int)

    def test_medal_id_is_int(self) -> None:
        assert isinstance(MEDAL_STEAKTACULAR_ID, int)
        assert MEDAL_STEAKTACULAR_ID == 1169390319


class TestDominanceBackfill:
    """Tests du calcul de dominance via DuckDB en mémoire."""

    @pytest.fixture()
    def shared_conn(self) -> duckdb.DuckDBPyConnection:
        conn = duckdb.connect(":memory:")
        conn.execute("""
            CREATE TABLE match_registry (
                match_id VARCHAR PRIMARY KEY,
                medals_loaded BOOLEAN DEFAULT TRUE
            )
        """)
        conn.execute("""
            CREATE TABLE match_participants (
                match_id VARCHAR,
                xuid VARCHAR,
                team_id INTEGER
            )
        """)
        conn.execute("""
            CREATE TABLE medals_earned (
                match_id VARCHAR,
                xuid VARCHAR,
                medal_name_id BIGINT
            )
        """)
        return conn

    @pytest.fixture()
    def player_conn(self) -> duckdb.DuckDBPyConnection:
        conn = duckdb.connect(":memory:")
        conn.execute("""
            CREATE TABLE player_match_enrichment (
                match_id VARCHAR PRIMARY KEY,
                performance_score DOUBLE,
                had_bot_teammate BOOLEAN,
                dominance_flag TINYINT,
                updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            )
        """)
        return conn

    def _seed_match(  # noqa: PLR0913
        self,
        shared: duckdb.DuckDBPyConnection,
        match_id: str,
        player_xuid: str,
        player_team: int,
        steak_xuid: str | None = None,
        steak_team: int | None = None,
    ) -> None:
        shared.execute("INSERT INTO match_registry VALUES (?, TRUE)", [match_id])
        shared.execute(
            "INSERT INTO match_participants VALUES (?, ?, ?)",
            [match_id, player_xuid, player_team],
        )
        if steak_xuid and steak_team is not None:
            shared.execute(
                "INSERT INTO match_participants VALUES (?, ?, ?)",
                [match_id, steak_xuid, steak_team],
            )
            shared.execute(
                "INSERT INTO medals_earned VALUES (?, ?, ?)",
                [match_id, steak_xuid, MEDAL_STEAKTACULAR_ID],
            )

    def test_no_steaktacular(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        player_conn: duckdb.DuckDBPyConnection,
    ) -> None:
        """Match sans Steaktacular → flag=0."""
        from src.data.dominance_backfill import compute_dominance_for_player

        self._seed_match(shared_conn, "m1", "xuid1", 0)
        result = compute_dominance_for_player(player_conn, shared_conn, "xuid1")
        assert result["processed"] == 1
        assert result["domination"] == 0
        assert result["humiliation"] == 0
        row = player_conn.execute(
            "SELECT dominance_flag FROM player_match_enrichment WHERE match_id = 'm1'"
        ).fetchone()
        assert row is not None
        assert row[0] == DominanceFlag.NONE

    def test_domination(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        player_conn: duckdb.DuckDBPyConnection,
    ) -> None:
        """Notre équipe a le Steaktacular → DOMINATION."""
        from src.data.dominance_backfill import compute_dominance_for_player

        self._seed_match(
            shared_conn,
            "m2",
            "xuid1",
            0,
            steak_xuid="xuid_ally",
            steak_team=0,
        )
        result = compute_dominance_for_player(player_conn, shared_conn, "xuid1")
        assert result["domination"] == 1
        assert result["humiliation"] == 0
        row = player_conn.execute(
            "SELECT dominance_flag FROM player_match_enrichment WHERE match_id = 'm2'"
        ).fetchone()
        assert row is not None
        assert row[0] == DominanceFlag.DOMINATION

    def test_humiliation(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        player_conn: duckdb.DuckDBPyConnection,
    ) -> None:
        """L'équipe ennemie a le Steaktacular → HUMILIATION."""
        from src.data.dominance_backfill import compute_dominance_for_player

        self._seed_match(
            shared_conn,
            "m3",
            "xuid1",
            0,
            steak_xuid="xuid_enemy",
            steak_team=1,
        )
        result = compute_dominance_for_player(player_conn, shared_conn, "xuid1")
        assert result["domination"] == 0
        assert result["humiliation"] == 1
        row = player_conn.execute(
            "SELECT dominance_flag FROM player_match_enrichment WHERE match_id = 'm3'"
        ).fetchone()
        assert row is not None
        assert row[0] == DominanceFlag.HUMILIATION

    def test_idempotent_without_force(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        player_conn: duckdb.DuckDBPyConnection,
    ) -> None:
        """Les matchs déjà flaggués ne sont pas recalculés sans force."""
        from src.data.dominance_backfill import compute_dominance_for_player

        self._seed_match(shared_conn, "m4", "xuid1", 0)
        compute_dominance_for_player(player_conn, shared_conn, "xuid1")
        # 2e appel — ne doit rien recalculer (flag déjà posé)
        result = compute_dominance_for_player(player_conn, shared_conn, "xuid1")
        assert result["processed"] == 1  # UPDATE exécuté mais WHERE dominance_flag IS NULL

    def test_force_recalculates(
        self,
        shared_conn: duckdb.DuckDBPyConnection,
        player_conn: duckdb.DuckDBPyConnection,
    ) -> None:
        """Avec force=True, les matchs déjà flaggués sont recalculés."""
        from src.data.dominance_backfill import compute_dominance_for_player

        self._seed_match(shared_conn, "m5", "xuid1", 0)
        compute_dominance_for_player(player_conn, shared_conn, "xuid1")
        result = compute_dominance_for_player(player_conn, shared_conn, "xuid1", force=True)
        assert result["processed"] == 1
