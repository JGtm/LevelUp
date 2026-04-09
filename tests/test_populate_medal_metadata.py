"""Tests pour le script populate_medal_metadata.py (v6.3).

Le script utilise désormais l'API gamecms (champ translations) plutôt que les JSON statiques.
Les tests vérifient les fonctions de construction + insertion sans appel API réel.
"""

from __future__ import annotations

import duckdb
import pytest

# Payload minimal simulant la réponse API gamecms.get_medal_metadata()
_SAMPLE_MEDALS_RAW = [
    {
        "nameId": 622331684,
        "name": {
            "value": "Double Kill",
            "translations": {
                "fr-FR": "Double frag",
                "de-DE": "Doppelter Abschuss",
            },
        },
        "description": {
            "value": "Kill 2 enemies in quick succession",
            "translations": {
                "fr-FR": "Tuez 2 ennemis d'affilée.",
                "de-DE": "2 Gegner schnell hintereinander eliminieren",
            },
        },
    },
    {
        "nameId": 17866865,
        "name": {"value": "Avenger", "translations": {"fr-FR": "Vengeur"}},
        "description": {"value": "Desc EN", "translations": {"fr-FR": "Desc FR"}},
    },
    # Médaille custom (nameId >= 9_000_000_000) → ne doit pas être dans l'API normalement
    {
        "nameId": 9000000001,
        "name": {"value": "Custom Medal", "translations": {}},
        "description": {"value": "", "translations": {}},
    },
]


@pytest.fixture()
def tmp_conn():
    """Connexion DuckDB en mémoire avec schéma medal_definitions + medal_translations."""
    conn = duckdb.connect(":memory:")
    from src.data.sync.migrations import (
        ensure_medal_definitions_table,
        ensure_medal_translations_table,
    )

    ensure_medal_definitions_table(conn)
    ensure_medal_translations_table(conn)
    yield conn
    conn.close()


# ---------------------------------------------------------------------------
# _build_definitions_rows
# ---------------------------------------------------------------------------


def test_build_definitions_rows_structure():
    """_build_definitions_rows retourne des tuples (id, name_fr, name_en, desc_fr, desc_en, is_custom)."""
    from scripts.populate_medal_metadata import _build_definitions_rows

    rows = _build_definitions_rows(_SAMPLE_MEDALS_RAW)
    assert len(rows) == 3
    first = rows[0]
    assert len(first) == 6
    assert first[0] == 622331684  # medal_name_id
    assert first[1] == "Double frag"  # name_fr
    assert first[2] == "Double Kill"  # name_en
    assert first[5] is False  # is_custom


def test_build_definitions_rows_custom_detection():
    """Les médailles >= 9_000_000_000 sont marquées is_custom=True."""
    from scripts.populate_medal_metadata import _build_definitions_rows

    rows = _build_definitions_rows(_SAMPLE_MEDALS_RAW)
    custom = [r for r in rows if r[5]]
    assert len(custom) == 1
    assert custom[0][0] == 9000000001


# ---------------------------------------------------------------------------
# populate_medal_definitions
# ---------------------------------------------------------------------------


def test_populate_medal_definitions_inserts(tmp_conn):
    """populate_medal_definitions() insère les lignes dans medal_definitions."""
    from scripts.populate_medal_metadata import populate_medal_definitions

    n = populate_medal_definitions(_SAMPLE_MEDALS_RAW, tmp_conn, force=False)
    assert n == 3
    count = tmp_conn.execute("SELECT COUNT(*) FROM medal_definitions").fetchone()[0]
    assert count == 3


def test_populate_medal_definitions_force_overwrites(tmp_conn):
    """--force écrase les entrées existantes."""
    from scripts.populate_medal_metadata import populate_medal_definitions

    populate_medal_definitions(_SAMPLE_MEDALS_RAW, tmp_conn, force=False)
    # Modifier manuellement
    tmp_conn.execute(
        "UPDATE medal_definitions SET name_fr = 'MODIFIE' WHERE medal_name_id = 622331684"
    )

    populate_medal_definitions(_SAMPLE_MEDALS_RAW, tmp_conn, force=True)

    row = tmp_conn.execute(
        "SELECT name_fr FROM medal_definitions WHERE medal_name_id = 622331684"
    ).fetchone()
    assert row[0] == "Double frag"  # restauré depuis le payload


def test_populate_medal_definitions_ignore_without_force(tmp_conn):
    """Sans --force, INSERT OR IGNORE ne touche pas les entrées existantes."""
    from scripts.populate_medal_metadata import populate_medal_definitions

    populate_medal_definitions(_SAMPLE_MEDALS_RAW, tmp_conn, force=False)
    tmp_conn.execute(
        "UPDATE medal_definitions SET name_fr = 'MODIFIE' WHERE medal_name_id = 622331684"
    )

    populate_medal_definitions(_SAMPLE_MEDALS_RAW, tmp_conn, force=False)

    row = tmp_conn.execute(
        "SELECT name_fr FROM medal_definitions WHERE medal_name_id = 622331684"
    ).fetchone()
    assert row[0] == "MODIFIE"  # non écrasé


# ---------------------------------------------------------------------------
# populate_medal_translations
# ---------------------------------------------------------------------------


def test_populate_medal_translations_official_langs(tmp_conn):
    """Les médailles officielles ont des entrées pour en-US + langues translations."""
    from scripts.populate_medal_metadata import populate_medal_translations

    populate_medal_translations(_SAMPLE_MEDALS_RAW, tmp_conn, force=False)

    langs = {
        r[0]
        for r in tmp_conn.execute(
            "SELECT DISTINCT lang FROM medal_translations WHERE medal_name_id = 622331684"
        ).fetchall()
    }
    assert "en-US" in langs
    assert "fr-FR" in langs
    assert "de-DE" in langs


def test_populate_medal_translations_custom_skipped(tmp_conn):
    """Les médailles custom ne sont PAS peuplées via les données API (pas de translations)."""
    from scripts.populate_medal_metadata import populate_medal_translations

    # La custom dans _SAMPLE_MEDALS_RAW a translations={} → seul en-US (si name présent)
    populate_medal_translations(_SAMPLE_MEDALS_RAW, tmp_conn, force=False)

    langs = {
        r[0]
        for r in tmp_conn.execute(
            "SELECT DISTINCT lang FROM medal_translations WHERE medal_name_id = 9000000001"
        ).fetchall()
    }
    # Pas de traductions (translations vide) — custom skippée côté officiel
    assert "fr-FR" not in langs


# ---------------------------------------------------------------------------
# migrate_from_definitions_only
# ---------------------------------------------------------------------------


def test_migrate_from_definitions_only(tmp_conn):
    """migrate_from_definitions_only() peuple medal_translations depuis medal_definitions."""
    from scripts.populate_medal_metadata import (
        migrate_from_definitions_only,
        populate_medal_definitions,
    )

    populate_medal_definitions(_SAMPLE_MEDALS_RAW, tmp_conn, force=False)
    n = migrate_from_definitions_only(tmp_conn, force=False)

    assert n > 0
    en = tmp_conn.execute(
        "SELECT name FROM medal_translations WHERE medal_name_id=622331684 AND lang='en-US'"
    ).fetchone()
    fr = tmp_conn.execute(
        "SELECT name FROM medal_translations WHERE medal_name_id=622331684 AND lang='fr-FR'"
    ).fetchone()
    assert en and en[0] == "Double Kill"
    assert fr and fr[0] == "Double frag"
