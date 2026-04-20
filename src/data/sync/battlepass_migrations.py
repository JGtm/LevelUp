"""Migrations dédiées aux snapshots battle pass joueur."""

from __future__ import annotations

import duckdb

_BATTLEPASS_SNAPSHOTS_DDL = """
CREATE TABLE IF NOT EXISTS battlepass_snapshots (
    snapshot_at                TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    xuid                       VARCHAR NOT NULL,
    reward_track_path          VARCHAR NOT NULL,
    track_type                 VARCHAR,
    is_active                  BOOLEAN NOT NULL DEFAULT FALSE,
    is_owned                   BOOLEAN,
    current_rank               INTEGER,
    partial_progress           INTEGER,
    previous_rank              INTEGER,
    previous_partial_progress  INTEGER,
    has_reached_max_rank       BOOLEAN,
    base_xp                    INTEGER,
    boost_xp                   INTEGER,
    state_hash                 VARCHAR NOT NULL,
    raw_payload_json           VARCHAR NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_battlepass_snapshots_xuid_time
    ON battlepass_snapshots(xuid, snapshot_at DESC);
CREATE INDEX IF NOT EXISTS idx_battlepass_snapshots_track_time
    ON battlepass_snapshots(reward_track_path, snapshot_at DESC);
"""


def ensure_battlepass_snapshots_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée ``battlepass_snapshots`` dans stats.duckdb."""
    conn.execute(_BATTLEPASS_SNAPSHOTS_DDL)
