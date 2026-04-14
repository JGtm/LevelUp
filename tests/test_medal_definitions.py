"""Tests unitaires pour medal_definitions (table + repository + UI fallback)."""

from __future__ import annotations

import duckdb
import polars as pl
import pytest

# ── Fixtures ──────────────────────────────────────────────────────────────────


@pytest.fixture()
def in_memory_metadata_db():
    """DuckDB :memory: avec schéma medal_definitions créé."""
    from src.data.sync.migrations import ensure_medal_definitions_table

    conn = duckdb.connect(":memory:")
    ensure_medal_definitions_table(conn)
    yield conn
    conn.close()


@pytest.fixture()
def tmp_metadata_db(tmp_path):
    """Fichier temporaire metadata.duckdb peuplé."""
    from src.data.sync.migrations import ensure_medal_definitions_table

    db_path = tmp_path / "metadata.duckdb"
    conn = duckdb.connect(str(db_path))
    ensure_medal_definitions_table(conn)
    conn.executemany(
        "INSERT INTO medal_definitions "
        "(medal_name_id, name_fr, name_en, description_fr, description_en, is_custom) "
        "VALUES (?, ?, ?, ?, ?, ?)",
        [
            (17866865, "Affection", "Infected", "Infectez tous.", "Infect all.", False),
            (9000000001, "Vengeur", "Avenger", "Tuez votre tueur.", "Kill your killer.", True),
        ],
    )
    conn.commit()
    conn.close()
    return db_path


# ── Phase 1 : Schéma ─────────────────────────────────────────────────────────


def test_schema_columns(in_memory_metadata_db):
    """La table medal_definitions a les 6 colonnes attendues."""
    conn = in_memory_metadata_db
    cols = conn.execute(
        "SELECT column_name, data_type FROM information_schema.columns "
        "WHERE table_name = 'medal_definitions' ORDER BY ordinal_position"
    ).fetchall()
    names = [c[0] for c in cols]
    assert names == [
        "medal_name_id",
        "name_fr",
        "name_en",
        "description_fr",
        "description_en",
        "is_custom",
    ]
    types = {c[0]: c[1] for c in cols}
    assert types["medal_name_id"] == "BIGINT"
    assert types["is_custom"] == "BOOLEAN"


def test_upsert_idempotent(in_memory_metadata_db):
    """Deux insertions identiques ne doublent pas les lignes."""
    conn = in_memory_metadata_db
    row = (100, "Test FR", "Test EN", "", "", False)
    conn.execute("INSERT OR IGNORE INTO medal_definitions VALUES (?, ?, ?, ?, ?, ?)", row)
    conn.execute("INSERT OR IGNORE INTO medal_definitions VALUES (?, ?, ?, ?, ?, ?)", row)
    count = conn.execute("SELECT COUNT(*) FROM medal_definitions").fetchone()[0]
    assert count == 1


def test_custom_flag_detection(in_memory_metadata_db):
    """Les médailles >= 9_000_000_000 sont marquées custom."""
    conn = in_memory_metadata_db
    conn.execute(
        "INSERT INTO medal_definitions VALUES (?, ?, ?, ?, ?, ?)",
        (9000000001, "Vengeur", "Avenger", "", "", True),
    )
    conn.execute(
        "INSERT INTO medal_definitions VALUES (?, ?, ?, ?, ?, ?)",
        (17866865, "Affection", "Infected", "", "", False),
    )
    custom = conn.execute(
        "SELECT medal_name_id FROM medal_definitions WHERE is_custom = TRUE"
    ).fetchall()
    assert len(custom) == 1
    assert custom[0][0] == 9000000001


# ── Phase 4 : Repository ─────────────────────────────────────────────────────


def test_load_medal_definitions_returns_polars(tmp_metadata_db, tmp_path):
    """load_medal_definitions retourne un DataFrame Polars avec les bonnes colonnes."""
    from src.data.repositories.duckdb_repo import DuckDBRepository

    # Créer une player DB minimale
    player_db = tmp_path / "player.duckdb"
    pconn = duckdb.connect(str(player_db))
    pconn.execute("CREATE TABLE sync_meta (key VARCHAR, value VARCHAR)")
    pconn.execute("INSERT INTO sync_meta VALUES ('xuid', '12345')")
    pconn.close()

    repo = DuckDBRepository(
        player_db_path=player_db,
        xuid="12345",
        metadata_db_path=tmp_metadata_db,
    )
    df = repo.load_medal_definitions()
    assert isinstance(df, pl.DataFrame)
    assert set(df.columns) == {
        "medal_name_id",
        "name_fr",
        "name_en",
        "description_fr",
        "description_en",
        "is_custom",
    }
    assert len(df) == 2


def test_get_medal_label(tmp_metadata_db, tmp_path):
    """get_medal_label retourne le label dans la langue demandée."""
    from src.data.repositories.duckdb_repo import DuckDBRepository

    player_db = tmp_path / "player.duckdb"
    pconn = duckdb.connect(str(player_db))
    pconn.execute("CREATE TABLE sync_meta (key VARCHAR, value VARCHAR)")
    pconn.execute("INSERT INTO sync_meta VALUES ('xuid', '12345')")
    pconn.close()

    repo = DuckDBRepository(
        player_db_path=player_db,
        xuid="12345",
        metadata_db_path=tmp_metadata_db,
    )
    assert repo.get_medal_label(17866865, "fr") == "Affection"
    assert repo.get_medal_label(17866865, "en") == "Infected"
    assert repo.get_medal_label(999999, "fr") is None


def test_load_medal_definitions_no_metadata(tmp_path):
    """Sans metadata DB, load_medal_definitions retourne un DataFrame vide."""
    from src.data.repositories.duckdb_repo import DuckDBRepository

    player_db = tmp_path / "player.duckdb"
    pconn = duckdb.connect(str(player_db))
    pconn.execute("CREATE TABLE sync_meta (key VARCHAR, value VARCHAR)")
    pconn.execute("INSERT INTO sync_meta VALUES ('xuid', '12345')")
    pconn.close()

    repo = DuckDBRepository(player_db_path=player_db, xuid="12345")
    df = repo.load_medal_definitions()
    assert isinstance(df, pl.DataFrame)
    assert len(df) == 0


def test_get_medal_label_no_metadata(tmp_path):
    """Sans metadata DB, get_medal_label retourne None."""
    from src.data.repositories.duckdb_repo import DuckDBRepository

    player_db = tmp_path / "player.duckdb"
    pconn = duckdb.connect(str(player_db))
    pconn.execute("CREATE TABLE sync_meta (key VARCHAR, value VARCHAR)")
    pconn.execute("INSERT INTO sync_meta VALUES ('xuid', '12345')")
    pconn.close()

    repo = DuckDBRepository(player_db_path=player_db, xuid="12345")
    assert repo.get_medal_label(17866865, "fr") is None


# ── Phase 5 : UI — DB-only, pas de fallback JSON ─────────────────────────────


def test_empty_db_returns_empty_maps(tmp_path):
    """DB vide → dicts vides (pas de fallback JSON)."""
    from unittest.mock import patch as _patch

    from src.data.sync.migrations import ensure_medal_definitions_table

    db_path = tmp_path / "metadata.duckdb"
    conn = duckdb.connect(str(db_path))
    ensure_medal_definitions_table(conn)
    conn.close()

    from src.ui.medals import load_medal_name_maps

    with _patch("src.data.medal_definitions.get_metadata_db_path", return_value=db_path):
        load_medal_name_maps.cache_clear()
        fr_map, en_map = load_medal_name_maps()

    assert len(fr_map) == 0
    assert len(en_map) == 0


def test_db_path_used_when_available(tmp_metadata_db):
    """Quand la DB a des entrées, load_medal_name_maps les retourne."""
    from unittest.mock import patch as _patch

    from src.ui.medals import load_medal_name_maps

    with _patch("src.data.medal_definitions.get_metadata_db_path", return_value=tmp_metadata_db):
        load_medal_name_maps.cache_clear()
        fr_map, en_map = load_medal_name_maps()

    assert len(fr_map) >= 2
    assert "17866865" in fr_map
    assert fr_map["17866865"] == "Affection"


def test_missing_db_returns_empty_maps(tmp_path):
    """Si la DB n'existe pas, retourne des dicts vides."""
    from unittest.mock import patch as _patch

    from src.ui.medals import load_medal_name_maps

    absent = tmp_path / "nonexistent.duckdb"
    with _patch("src.data.medal_definitions.get_metadata_db_path", return_value=absent):
        load_medal_name_maps.cache_clear()
        fr_map, en_map = load_medal_name_maps()

    assert fr_map == {}
    assert en_map == {}
