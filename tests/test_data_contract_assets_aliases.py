"""Contrats data pour assets et aliases (DuckDB)."""

from __future__ import annotations

import re

import pytest

duckdb = pytest.importorskip("duckdb")

from src.ui.aliases import load_aliases_from_db

UUID_LIKE_RE = re.compile(
    r"^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"
)


@pytest.fixture
def assets_aliases_contract_db(tmp_path):
    """Crée une base temporaire avec labels et aliases."""
    db_path = tmp_path / "assets_aliases_contract.duckdb"
    conn = duckdb.connect(str(db_path))
    try:
        conn.execute(
            """
            CREATE TABLE match_stats (
                match_id VARCHAR PRIMARY KEY,
                map_name VARCHAR,
                playlist_name VARCHAR
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

        conn.execute(
            """
            INSERT INTO match_stats (match_id, map_name, playlist_name)
            VALUES
                ('m1', 'Recharge', 'Ranked Arena'),
                ('m2', 'Live Fire', 'Quick Play')
            """
        )
        conn.execute(
            """
            INSERT INTO xuid_aliases (xuid, gamertag)
            VALUES
                ('xuid_1', 'Alpha'),
                ('xuid_2', 'Bravo'),
                ('xuid_3', '')
            """
        )
    finally:
        conn.close()
    return db_path


def test_aliases_table_and_columns_exist(assets_aliases_contract_db) -> None:
    """La table xuid_aliases doit être exploitable."""
    conn = duckdb.connect(str(assets_aliases_contract_db), read_only=True)
    try:
        columns = {
            row[0]
            for row in conn.execute(
                "SELECT column_name FROM information_schema.columns WHERE table_name='xuid_aliases'"
            ).fetchall()
        }
        assert {"xuid", "gamertag"}.issubset(columns)
    finally:
        conn.close()


def test_load_aliases_from_db_reads_non_empty_aliases(tmp_path) -> None:
    """8bis: En v5.1, load_aliases_from_db lit depuis shared_matches.duckdb.

    Ce test crée la structure complète (shared + player) pour valider.
    """
    # Structure v5.1 : créer shared_matches.duckdb avec xuid_aliases
    warehouse = tmp_path / "data" / "warehouse"
    warehouse.mkdir(parents=True, exist_ok=True)
    shared_db = warehouse / "shared_matches.duckdb"

    conn = duckdb.connect(str(shared_db))
    conn.execute("""
        CREATE TABLE xuid_aliases (
            xuid VARCHAR PRIMARY KEY,
            gamertag VARCHAR NOT NULL
        )
    """)
    conn.execute("""
        INSERT INTO xuid_aliases VALUES
        ('xuid_1', 'Alpha'),
        ('xuid_2', 'Bravo')
    """)
    conn.close()

    # Player DB (pour le chemin relatif)
    player_db = tmp_path / "data" / "players" / "TestPlayer" / "stats.duckdb"
    player_db.parent.mkdir(parents=True, exist_ok=True)
    conn = duckdb.connect(str(player_db))
    conn.execute("CREATE TABLE sync_meta (key VARCHAR, value VARCHAR)")
    conn.close()

    aliases = load_aliases_from_db(str(player_db))
    assert aliases["xuid_1"] == "Alpha"
    assert aliases["xuid_2"] == "Bravo"


def test_match_labels_are_not_empty_or_uuid_like(assets_aliases_contract_db) -> None:
    """Les labels map/playlist doivent être lisibles (pas vides, pas UUID bruts)."""
    conn = duckdb.connect(str(assets_aliases_contract_db), read_only=True)
    try:
        rows = conn.execute("SELECT map_name, playlist_name FROM match_stats").fetchall()
    finally:
        conn.close()

    for map_name, playlist_name in rows:
        assert map_name and str(map_name).strip()
        assert playlist_name and str(playlist_name).strip()
        assert UUID_LIKE_RE.match(str(map_name)) is None
        assert UUID_LIKE_RE.match(str(playlist_name)) is None
