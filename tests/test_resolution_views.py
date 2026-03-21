"""Tests unitaires pour les 3 vues de résolution d'IDs (v6).

Vues testées :
- v_gamertag_lookup  : XUID → gamertag courant (xuid_aliases > match_participants)
- v_match_full       : match_registry + noms résolus depuis metadata.duckdb
- v_killer_victim_full : killer_victim_pairs + gamertags résolus

Chaque test crée ses propres tables en DuckDB in-memory → pas de dépendances externes.
"""

from __future__ import annotations

import duckdb

from src.data.sync.migrations import ensure_resolution_views

# ─────────────────────────────────────────────────────────────────────────────
# Helpers de fixtures
# ─────────────────────────────────────────────────────────────────────────────


def _make_shared_db() -> duckdb.DuckDBPyConnection:
    """Crée une DB in-memory avec le schéma minimal shared_matches."""
    conn = duckdb.connect()
    conn.execute("""
        CREATE TABLE xuid_aliases (
            xuid VARCHAR PRIMARY KEY,
            gamertag VARCHAR NOT NULL
        )
    """)
    conn.execute("""
        CREATE TABLE match_participants (
            match_id VARCHAR NOT NULL,
            xuid VARCHAR,
            gamertag VARCHAR,
            outcome INTEGER,
            kills INTEGER DEFAULT 0,
            deaths INTEGER DEFAULT 0,
            assists INTEGER DEFAULT 0,
            team_id INTEGER DEFAULT 0
        )
    """)
    conn.execute("""
        CREATE TABLE match_registry (
            match_id VARCHAR PRIMARY KEY,
            start_time TIMESTAMP,
            duration_seconds INTEGER,
            map_id VARCHAR,
            playlist_id VARCHAR,
            pair_id VARCHAR,
            game_variant_id VARCHAR,
            map_name VARCHAR,
            playlist_name VARCHAR,
            pair_name VARCHAR,
            game_variant_name VARCHAR,
            team_0_score INTEGER,
            team_1_score INTEGER,
            team_0_ps_score INTEGER,
            team_1_ps_score INTEGER,
            is_firefight BOOLEAN DEFAULT FALSE,
            is_ranked BOOLEAN DEFAULT FALSE,
            events_loaded BOOLEAN DEFAULT FALSE,
            medals_loaded BOOLEAN DEFAULT FALSE,
            participants_loaded BOOLEAN DEFAULT FALSE,
            backfill_completed BOOLEAN DEFAULT FALSE,
            sync_spnkr_version VARCHAR
        )
    """)
    conn.execute("""
        CREATE TABLE killer_victim_pairs (
            match_id VARCHAR NOT NULL,
            killer_xuid VARCHAR NOT NULL,
            killer_gamertag VARCHAR,
            victim_xuid VARCHAR NOT NULL,
            victim_gamertag VARCHAR,
            kill_count INTEGER DEFAULT 1,
            time_ms INTEGER,
            is_validated BOOLEAN DEFAULT FALSE
        )
    """)
    return conn


# ─────────────────────────────────────────────────────────────────────────────
# Tests v_gamertag_lookup
# ─────────────────────────────────────────────────────────────────────────────


def test_v_gamertag_lookup_aliases_prioritaires():
    """xuid_aliases doit l'emporter sur match_participants pour le même XUID."""
    conn = _make_shared_db()
    conn.execute("INSERT INTO xuid_aliases VALUES ('xuid_a', 'Alice')")
    conn.execute(
        "INSERT INTO match_participants(match_id, xuid, gamertag) VALUES ('m1', 'xuid_a', 'alice_old')"
    )
    ensure_resolution_views(conn)

    row = conn.execute("SELECT gamertag FROM v_gamertag_lookup WHERE xuid = 'xuid_a'").fetchone()
    assert row is not None
    assert row[0] == "Alice"


def test_v_gamertag_lookup_fallback_match_participants():
    """Un XUID absent de xuid_aliases doit être résolu via match_participants."""
    conn = _make_shared_db()
    conn.execute(
        "INSERT INTO match_participants(match_id, xuid, gamertag) VALUES ('m1', 'xuid_b', 'Bob')"
    )
    ensure_resolution_views(conn)

    row = conn.execute("SELECT gamertag FROM v_gamertag_lookup WHERE xuid = 'xuid_b'").fetchone()
    assert row is not None
    assert row[0] == "Bob"


def test_v_gamertag_lookup_null_gamertag_exclu():
    """Les lignes avec gamertag NULL ne doivent pas apparaître dans la vue."""
    conn = _make_shared_db()
    conn.execute(
        "INSERT INTO match_participants(match_id, xuid, gamertag) VALUES ('m1', 'xuid_c', NULL)"
    )
    ensure_resolution_views(conn)

    rows = conn.execute("SELECT * FROM v_gamertag_lookup WHERE xuid = 'xuid_c'").fetchall()
    assert rows == []


def test_v_gamertag_lookup_dedup_match_participants():
    """Plusieurs participations d'un même XUID → un seul gamertag en sortie."""
    conn = _make_shared_db()
    conn.execute(
        "INSERT INTO match_participants(match_id, xuid, gamertag) VALUES "
        "('m1', 'xuid_d', 'Dave'), ('m2', 'xuid_d', 'Dave')"
    )
    ensure_resolution_views(conn)

    rows = conn.execute("SELECT gamertag FROM v_gamertag_lookup WHERE xuid = 'xuid_d'").fetchall()
    assert len(rows) == 1
    assert rows[0][0] == "Dave"


# ─────────────────────────────────────────────────────────────────────────────
# Tests v_match_full
# ─────────────────────────────────────────────────────────────────────────────


def test_v_match_full_colonnes_en_non_nulles():
    """Les colonnes EN doivent retourner les valeurs de match_registry (fallback)."""
    conn = _make_shared_db()
    conn.execute("""
        INSERT INTO match_registry(match_id, map_name, playlist_name, pair_name, game_variant_name)
        VALUES ('m1', 'Recharge', 'Ranked Arena', 'Arena | Attrition', 'Arena:Attrition on Recharge')
    """)
    ensure_resolution_views(conn)

    row = conn.execute(
        "SELECT map_name, playlist_name FROM v_match_full WHERE match_id = 'm1'"
    ).fetchone()
    assert row is not None
    assert row[0] == "Recharge"
    assert row[1] == "Ranked Arena"


def test_v_match_full_colonnes_fr_nulles_sans_metadata():
    """Sans metadata.duckdb attaché, les colonnes *_fr doivent être NULL."""
    conn = _make_shared_db()
    conn.execute("""
        INSERT INTO match_registry(match_id, map_name, playlist_name, pair_name, game_variant_name)
        VALUES ('m2', 'Recharge', 'Ranked Arena', 'Arena | Attrition', 'Arena:Attrition on Recharge')
    """)
    ensure_resolution_views(conn)

    row = conn.execute(
        "SELECT map_name_fr, playlist_name_fr, mode_name FROM v_match_full WHERE match_id = 'm2'"
    ).fetchone()
    assert row is not None
    assert row[0] is None  # map_name_fr
    assert row[1] is None  # playlist_name_fr
    assert row[2] is None  # mode_name


def test_v_match_full_avec_metadata_attachee(tmp_path):
    """Avec metadata.duckdb attaché, name_en est résolu depuis les tables meta."""
    # Créer une metadata.duckdb minimale
    meta_path = tmp_path / "metadata.duckdb"
    meta_conn = duckdb.connect(str(meta_path))
    meta_conn.execute("""
        CREATE TABLE maps (
            asset_id VARCHAR PRIMARY KEY,
            name_en VARCHAR,
            name_fr VARCHAR
        )
    """)
    meta_conn.execute("INSERT INTO maps VALUES ('map_001', 'Recharge', 'Recharge')")
    # Tables vides requises par les LEFT JOINs
    meta_conn.execute(
        "CREATE TABLE playlists (asset_id VARCHAR, name_en VARCHAR, name_fr VARCHAR,"
        " playlist_canonical_en VARCHAR, playlist_canonical_fr VARCHAR)"
    )
    meta_conn.execute(
        "CREATE TABLE playlist_map_mode_pairs (asset_id VARCHAR, name_en VARCHAR, name_fr VARCHAR)"
    )
    meta_conn.execute(
        "CREATE TABLE game_variants (asset_id VARCHAR, name_en VARCHAR, name_fr VARCHAR,"
        " mode_name VARCHAR, mode_name_fr VARCHAR)"
    )
    meta_conn.close()

    conn = _make_shared_db()
    conn.execute(
        "INSERT INTO match_registry(match_id, map_id, map_name) VALUES ('m3', 'map_001', 'old_name')"
    )
    conn.execute(f"ATTACH '{meta_path}' AS meta (READ_ONLY)")
    ensure_resolution_views(conn)

    row = conn.execute("SELECT map_name FROM v_match_full WHERE match_id = 'm3'").fetchone()
    assert row is not None
    assert row[0] == "Recharge"  # vient de meta.maps.name_en


def test_v_match_full_idempotente():
    """ensure_resolution_views() appelée 2× ne doit pas lever d'exception."""
    conn = _make_shared_db()
    ensure_resolution_views(conn)
    ensure_resolution_views(conn)  # 2e appel — idempotent


# ─────────────────────────────────────────────────────────────────────────────
# Tests v_killer_victim_full
# ─────────────────────────────────────────────────────────────────────────────


def test_v_killer_victim_full_gamertag_resolu():
    """Les gamertags killer/victim doivent être résolus via v_gamertag_lookup."""
    conn = _make_shared_db()
    conn.execute("INSERT INTO xuid_aliases VALUES ('xuid_k', 'Killer'), ('xuid_v', 'Victim')")
    conn.execute("""
        INSERT INTO killer_victim_pairs(match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag)
        VALUES ('m1', 'xuid_k', 'old_killer', 'xuid_v', 'old_victim')
    """)
    ensure_resolution_views(conn)

    row = conn.execute(
        "SELECT killer_gamertag, victim_gamertag FROM v_killer_victim_full WHERE match_id = 'm1'"
    ).fetchone()
    assert row is not None
    assert row[0] == "Killer"
    assert row[1] == "Victim"


def test_v_killer_victim_full_fallback_snapshot():
    """Si le XUID est absent de v_gamertag_lookup, le snapshot figé est utilisé."""
    conn = _make_shared_db()
    conn.execute("""
        INSERT INTO killer_victim_pairs(match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag)
        VALUES ('m2', 'unknown_k', 'OldKiller', 'unknown_v', 'OldVictim')
    """)
    ensure_resolution_views(conn)

    row = conn.execute(
        "SELECT killer_gamertag, victim_gamertag FROM v_killer_victim_full WHERE match_id = 'm2'"
    ).fetchone()
    assert row is not None
    assert row[0] == "OldKiller"
    assert row[1] == "OldVictim"


def test_v_killer_victim_full_fallback_xuid_brut():
    """Si snapshot aussi NULL, le xuid brut doit apparaître comme gamertag."""
    conn = _make_shared_db()
    conn.execute("""
        INSERT INTO killer_victim_pairs(match_id, killer_xuid, killer_gamertag, victim_xuid, victim_gamertag)
        VALUES ('m3', 'raw_xuid_k', NULL, 'raw_xuid_v', NULL)
    """)
    ensure_resolution_views(conn)

    row = conn.execute(
        "SELECT killer_gamertag, victim_gamertag FROM v_killer_victim_full WHERE match_id = 'm3'"
    ).fetchone()
    assert row is not None
    assert row[0] == "raw_xuid_k"
    assert row[1] == "raw_xuid_v"
