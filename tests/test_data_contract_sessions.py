"""Contrats data pour sessions/navigation (DuckDB)."""

from __future__ import annotations

from datetime import datetime, timedelta, timezone
from unittest.mock import patch

import pytest

duckdb = pytest.importorskip("duckdb")

from src.ui.cache import cached_compute_sessions_db


@pytest.fixture
def sessions_contract_db(tmp_path):
    """Crée une base temporaire avec colonnes de sessions dans match_stats."""
    db_path = tmp_path / "sessions_contract.duckdb"
    conn = duckdb.connect(str(db_path))
    try:
        conn.execute(
            """
            CREATE TABLE match_stats (
                match_id VARCHAR PRIMARY KEY,
                start_time TIMESTAMP,
                teammates_signature VARCHAR,
                session_id VARCHAR,
                session_label VARCHAR,
                is_firefight BOOLEAN
            )
            """
        )

        t0 = datetime.now(timezone.utc) - timedelta(days=3)
        rows = [
            ("m1", t0, "a;b;c", "s1", "Session 1", False),
            ("m2", t0 + timedelta(minutes=18), "a;b;c", "s1", "Session 1", False),
            ("m3", t0 + timedelta(hours=6), "a;d;e", "s2", "Session 2", False),
        ]
        conn.executemany(
            """
            INSERT INTO match_stats (
                match_id, start_time, teammates_signature, session_id, session_label, is_firefight
            ) VALUES (?, ?, ?, ?, ?, ?)
            """,
            rows,
        )
    finally:
        conn.close()

    return db_path


def test_session_columns_exist_in_match_stats(sessions_contract_db) -> None:
    """Les colonnes session_id/session_label doivent exister dans match_stats."""
    conn = duckdb.connect(str(sessions_contract_db), read_only=True)
    try:
        columns = {
            row[0]
            for row in conn.execute(
                "SELECT column_name FROM information_schema.columns WHERE table_name='match_stats'"
            ).fetchall()
        }
        assert {"session_id", "session_label", "start_time", "match_id"}.issubset(columns)
    finally:
        conn.close()


def test_sessions_columns_are_non_empty_for_loaded_rows(sessions_contract_db) -> None:
    """Les lignes de match_stats exploitables ne doivent pas avoir de session vide."""
    conn = duckdb.connect(str(sessions_contract_db), read_only=True)
    try:
        invalid = conn.execute(
            """
            SELECT COUNT(*)
            FROM match_stats
            WHERE start_time IS NOT NULL
              AND (session_id IS NULL OR TRIM(session_id) = ''
                   OR session_label IS NULL OR TRIM(session_label) = '')
            """
        ).fetchone()[0]
        assert invalid == 0
    finally:
        conn.close()


def test_cached_compute_sessions_db_returns_expected_contract(tmp_path) -> None:
    """Le calcul de sessions doit fournir les colonnes attendues et des labels cohérents (v5)."""
    # Crée la structure v5 : shared_matches.duckdb + player stats.duckdb
    shared_path = tmp_path / "shared_matches.duckdb"
    t0 = datetime.now(timezone.utc) - timedelta(days=3)
    shared_conn = duckdb.connect(str(shared_path))
    try:
        shared_conn.execute("""
            CREATE TABLE match_registry (
                match_id VARCHAR PRIMARY KEY,
                start_time TIMESTAMPTZ,
                is_firefight BOOLEAN DEFAULT FALSE
            )
        """)
        shared_conn.execute("""
            CREATE TABLE match_participants (
                match_id VARCHAR NOT NULL,
                xuid VARCHAR NOT NULL,
                PRIMARY KEY (match_id, xuid)
            )
        """)
        for mid, ts in [
            ("m1", t0),
            ("m2", t0 + timedelta(minutes=18)),
            ("m3", t0 + timedelta(hours=6)),
        ]:
            shared_conn.execute("INSERT INTO match_registry VALUES (?, ?, FALSE)", [mid, ts])
            shared_conn.execute("INSERT INTO match_participants VALUES (?, ?)", [mid, "x_me"])
    finally:
        shared_conn.close()

    db_path = tmp_path / "stats.duckdb"
    player_conn = duckdb.connect(str(db_path))
    try:
        player_conn.execute("""
            CREATE TABLE player_match_enrichment (
                match_id VARCHAR PRIMARY KEY,
                teammates_signature VARCHAR,
                session_id VARCHAR,
                session_label VARCHAR
            )
        """)
        for mid, sig, sid, slabel in [
            ("m1", "a;b;c", "s1", "Session 1"),
            ("m2", "a;b;c", "s1", "Session 1"),
            ("m3", "a;d;e", "s2", "Session 2"),
        ]:
            player_conn.execute(
                "INSERT INTO player_match_enrichment VALUES (?, ?, ?, ?)",
                [mid, sig, sid, slabel],
            )
    finally:
        player_conn.close()

    with patch(
        "src.utils.paths.get_shared_matches_path_from_player",
        return_value=shared_path,
    ):
        result = cached_compute_sessions_db(
            db_path=str(db_path),
            xuid="x_me",
            db_key=None,
            include_firefight=True,
            gap_minutes=120,
            friends_xuids=None,
        )

    assert not result.is_empty()
    assert ["match_id", "start_time", "session_id", "session_label"] == result.columns
    assert set(result["session_id"].cast(str).to_list()) == {"s1", "s2"}
    assert set(result["session_label"].cast(str).to_list()) == {"Session 1", "Session 2"}
