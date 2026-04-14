"""Tests d'intégration pour medal_definitions (nécessite metadata.duckdb réelle)."""

from __future__ import annotations

from pathlib import Path

import duckdb
import pytest

REPO_ROOT = Path(__file__).resolve().parent.parent.parent
METADATA_DB = REPO_ROOT / "data" / "warehouse" / "metadata.duckdb"

pytestmark = pytest.mark.skipif(
    not METADATA_DB.exists(),
    reason="metadata.duckdb absente — exécuter populate_medal_metadata.py d'abord",
)


def _has_medal_definitions() -> bool:
    """Vérifie si la table existe et contient des données."""
    if not METADATA_DB.exists():
        return False
    try:
        conn = duckdb.connect(str(METADATA_DB), read_only=True)
        tables = conn.execute(
            "SELECT table_name FROM information_schema.tables "
            "WHERE table_name = 'medal_definitions'"
        ).fetchall()
        if not tables:
            conn.close()
            return False
        count = conn.execute("SELECT COUNT(*) FROM medal_definitions").fetchone()[0]
        conn.close()
        return count >= 100
    except Exception:
        return False


skip_no_data = pytest.mark.skipif(
    not _has_medal_definitions(),
    reason="medal_definitions non peuplée — exécuter --medal-metadata d'abord",
)


@skip_no_data
def test_medal_definitions_populated():
    """La table medal_definitions contient au moins 100 entrées."""
    conn = duckdb.connect(str(METADATA_DB), read_only=True)
    count = conn.execute("SELECT COUNT(*) FROM medal_definitions").fetchone()[0]
    conn.close()
    assert count >= 100


@skip_no_data
@pytest.mark.skipif(
    not (
        Path(__file__).resolve().parent.parent.parent / "static" / "medals" / "medals_fr.json"
    ).exists(),
    reason="medals_fr.json absent — générer via populate_medal_metadata.py d'abord",
)
def test_all_medals_from_json_in_db():
    """Tous les IDs des JSON sont présents dans la DB."""
    import json

    medals_dir = REPO_ROOT / "static" / "medals"
    with open(medals_dir / "medals_fr.json", encoding="utf-8") as f:
        json_ids = set(json.load(f).keys())

    conn = duckdb.connect(str(METADATA_DB), read_only=True)
    db_ids = {
        str(r[0]) for r in conn.execute("SELECT medal_name_id FROM medal_definitions").fetchall()
    }
    conn.close()

    missing = json_ids - db_ids
    assert len(missing) == 0, f"IDs manquants en DB : {missing}"


@skip_no_data
def test_join_medals_earned_to_definitions():
    """JOIN medals_earned → medal_definitions a un taux de correspondance >= 95%."""
    shared_db = REPO_ROOT / "data" / "warehouse" / "shared_matches_v2.duckdb"
    if not shared_db.exists():
        pytest.skip("shared_matches_v2.duckdb absente")

    conn = duckdb.connect(str(METADATA_DB), read_only=True)
    conn.execute(f"ATTACH '{shared_db}' AS shared (READ_ONLY)")

    result = conn.execute("""
        WITH distinct_medals AS (
            SELECT DISTINCT medal_name_id FROM shared.medals_earned
        )
        SELECT
            COUNT(*) AS total,
            COUNT(md.medal_name_id) AS matched
        FROM distinct_medals dm
        LEFT JOIN medal_definitions md ON dm.medal_name_id = md.medal_name_id
    """).fetchone()
    conn.close()

    total, matched = result
    if total == 0:
        pytest.skip("Aucune médaille dans medals_earned")
    rate = matched / total
    assert rate >= 0.95, f"Taux correspondance {rate:.1%} < 95% ({matched}/{total})"


@skip_no_data
def test_load_medal_name_maps_uses_db(caplog):
    """load_medal_name_maps() charge depuis la DB."""
    import logging

    from src.ui.medals import load_medal_name_maps

    with caplog.at_level(logging.DEBUG, logger="src.ui.medals"):
        load_medal_name_maps.cache_clear()
        fr_map, en_map = load_medal_name_maps()

    assert len(fr_map) >= 100, f"Seulement {len(fr_map)} labels FR depuis DB"
    assert len(en_map) >= 100, f"Seulement {len(en_map)} labels EN depuis DB"
