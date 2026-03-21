"""Tests pour le script populate_medal_metadata.py."""

from __future__ import annotations

import duckdb
import pytest


@pytest.fixture()
def tmp_metadata_env(tmp_path, monkeypatch):
    """Prépare un environnement avec metadata.duckdb temporaire."""
    db_path = tmp_path / "data" / "warehouse" / "metadata.duckdb"
    db_path.parent.mkdir(parents=True, exist_ok=True)

    import scripts.populate_medal_metadata as mod

    monkeypatch.setattr(mod, "METADATA_DB", db_path)
    return db_path


def test_build_medal_rows():
    """build_medal_rows retourne au moins 167 médailles."""
    from scripts.populate_medal_metadata import build_medal_rows

    rows = build_medal_rows()
    assert len(rows) >= 167
    # Vérifier la structure
    first = rows[0]
    assert len(first) == 6  # medal_name_id, name_fr, name_en, desc_fr, desc_en, is_custom
    assert isinstance(first[0], int)
    assert isinstance(first[5], bool)


def test_custom_detection():
    """Les médailles >= 9_000_000_000 sont marquées custom."""
    from scripts.populate_medal_metadata import build_medal_rows

    rows = build_medal_rows()
    custom = [r for r in rows if r[5]]
    non_custom = [r for r in rows if not r[5]]
    for r in custom:
        assert r[0] >= 9_000_000_000
    for r in non_custom:
        assert r[0] < 9_000_000_000


def test_dry_run_makes_no_writes(tmp_metadata_env):
    """--dry-run ne crée pas de données dans la table."""
    from scripts.populate_medal_metadata import populate_medal_definitions

    n = populate_medal_definitions(dry_run=True)
    assert n == 0
    # La DB n'existe pas encore (dry-run ne connecte pas)
    if tmp_metadata_env.exists():
        conn = duckdb.connect(str(tmp_metadata_env), read_only=True)
        try:
            tables = conn.execute(
                "SELECT table_name FROM information_schema.tables "
                "WHERE table_name = 'medal_definitions'"
            ).fetchall()
            if tables:
                count = conn.execute("SELECT COUNT(*) FROM medal_definitions").fetchone()[0]
                assert count == 0
        finally:
            conn.close()


def test_all_json_entries_inserted(tmp_metadata_env):
    """Toutes les entrées JSON sont insérées."""
    from scripts.populate_medal_metadata import populate_medal_definitions

    n = populate_medal_definitions(dry_run=False, force=False)
    assert n >= 167

    conn = duckdb.connect(str(tmp_metadata_env), read_only=True)
    count = conn.execute("SELECT COUNT(*) FROM medal_definitions").fetchone()[0]
    conn.close()
    assert count >= 167


def test_force_flag_overwrites_existing(tmp_metadata_env):
    """--force écrase les entrées existantes."""
    from scripts.populate_medal_metadata import populate_medal_definitions

    # Première insertion
    populate_medal_definitions(dry_run=False, force=False)

    # Modifier une entrée manuellement
    conn = duckdb.connect(str(tmp_metadata_env), read_only=False)
    conn.execute("UPDATE medal_definitions SET name_fr = 'MODIFIE' WHERE medal_name_id = 17866865")
    conn.commit()
    conn.close()

    # Force re-insertion → doit écraser
    populate_medal_definitions(dry_run=False, force=True)

    conn = duckdb.connect(str(tmp_metadata_env), read_only=True)
    row = conn.execute(
        "SELECT name_fr FROM medal_definitions WHERE medal_name_id = 17866865"
    ).fetchone()
    conn.close()
    assert row[0] != "MODIFIE"
    assert row[0] == "Affection"


def test_missing_en_label_logged_as_warning(tmp_path, monkeypatch, caplog):
    """Un ID présent en FR mais absent de EN génère un WARNING."""
    import scripts.populate_medal_metadata as mod

    # Créer des JSON avec asymétrie
    medals_dir = tmp_path / "medals"
    medals_dir.mkdir()
    import json

    (medals_dir / "medals_fr.json").write_text(
        json.dumps({"100": "Test FR", "200": "Autre FR"}), encoding="utf-8"
    )
    (medals_dir / "medals_en.json").write_text(json.dumps({"100": "Test EN"}), encoding="utf-8")
    (medals_dir / "medals_descriptions_fr.json").write_text("{}", encoding="utf-8")
    (medals_dir / "medals_descriptions_en.json").write_text("{}", encoding="utf-8")

    monkeypatch.setattr(mod, "MEDALS_DIR", medals_dir)

    with caplog.at_level("WARNING", logger="levelup.medals"):
        rows = mod.build_medal_rows()

    assert any("200" in r.message and "FR" in r.message for r in caplog.records)
    assert len(rows) == 2
