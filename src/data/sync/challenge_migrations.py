"""Migrations dédiées aux défis Halo Infinite."""

from __future__ import annotations

import duckdb

_CHALLENGE_DEFINITIONS_DDL = """
CREATE TABLE IF NOT EXISTS challenge_definitions (
    challenge_path         VARCHAR NOT NULL,
    content_hash           VARCHAR NOT NULL,
    category               VARCHAR,
    difficulty             VARCHAR,
    threshold_for_success  INTEGER,
    reward_xp              INTEGER DEFAULT 0,
    secondary_reward_xp    INTEGER DEFAULT 0,
    first_seen_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_current             BOOLEAN DEFAULT TRUE,
    PRIMARY KEY (challenge_path, content_hash)
);
CREATE INDEX IF NOT EXISTS idx_challenge_definitions_current
    ON challenge_definitions(challenge_path, is_current);
CREATE INDEX IF NOT EXISTS idx_challenge_definitions_category
    ON challenge_definitions(category, difficulty);
"""

_CHALLENGE_TRANSLATIONS_DDL = """
CREATE TABLE IF NOT EXISTS challenge_translations (
    challenge_path  VARCHAR NOT NULL,
    content_hash    VARCHAR NOT NULL,
    lang            VARCHAR NOT NULL,
    title           VARCHAR,
    description     VARCHAR,
    first_seen_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (challenge_path, content_hash, lang)
);
CREATE INDEX IF NOT EXISTS idx_challenge_translations_lookup
    ON challenge_translations(challenge_path, lang);
"""

_CHALLENGE_SNAPSHOTS_DDL = """
CREATE TABLE IF NOT EXISTS challenge_snapshots (
    snapshot_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    xuid              VARCHAR NOT NULL,
    challenge_path    VARCHAR NOT NULL,
    challenge_id      VARCHAR,
    content_hash      VARCHAR,
    status            VARCHAR NOT NULL,
    progress_current  INTEGER,
    progress_target   INTEGER,
    xp_reward         INTEGER DEFAULT 0,
    can_reroll        BOOLEAN,
    expires_at        TIMESTAMP,
    deck_index        INTEGER,
    state_hash        VARCHAR NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_challenge_snapshots_xuid_time
    ON challenge_snapshots(xuid, snapshot_at DESC);
CREATE INDEX IF NOT EXISTS idx_challenge_snapshots_path_time
    ON challenge_snapshots(challenge_path, snapshot_at DESC);
"""


def ensure_challenge_definitions_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée ``challenge_definitions`` dans metadata.duckdb."""
    conn.execute(_CHALLENGE_DEFINITIONS_DDL)


def ensure_challenge_translations_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée ``challenge_translations`` dans metadata.duckdb."""
    conn.execute(_CHALLENGE_TRANSLATIONS_DDL)


def ensure_challenge_snapshots_table(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée ``challenge_snapshots`` dans stats.duckdb."""
    conn.execute(_CHALLENGE_SNAPSHOTS_DDL)
