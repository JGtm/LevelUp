#!/usr/bin/env python3
"""Migration one-shot : modes_fr.json + modes_en.json → metadata.duckdb.

Crée 4 tables dans metadata.duckdb :
- mode_prefix_names  (14 entrées × 2 langues = 28 lignes)
- mode_name_tr       (31 entrées × 2 langues = 62 lignes)
- mode_pair_overrides (11 entrées × 2 langues = 22 lignes)
- mode_lang_settings (2 lignes)

Usage :
    python scripts/migrate_modes_json_to_db.py
"""
from __future__ import annotations

import json
from pathlib import Path

import duckdb

ROOT = Path(__file__).resolve().parent.parent
LANGS: dict[str, Path] = {
    "fr": ROOT / "static" / "i18n" / "modes_fr.json",
    "en": ROOT / "static" / "i18n" / "modes_en.json",
}

_DDL_STATEMENTS = [
    """CREATE TABLE IF NOT EXISTS mode_prefix_names (
        prefix_en  VARCHAR NOT NULL,
        lang       VARCHAR NOT NULL,
        name       VARCHAR NOT NULL,
        PRIMARY KEY (prefix_en, lang)
    )""",
    """CREATE TABLE IF NOT EXISTS mode_name_tr (
        mode_en    VARCHAR NOT NULL,
        lang       VARCHAR NOT NULL,
        name       VARCHAR NOT NULL,
        PRIMARY KEY (mode_en, lang)
    )""",
    """CREATE TABLE IF NOT EXISTS mode_pair_overrides (
        pattern    VARCHAR NOT NULL,
        lang       VARCHAR NOT NULL,
        name       VARCHAR NOT NULL,
        PRIMARY KEY (pattern, lang)
    )""",
    """CREATE TABLE IF NOT EXISTS mode_lang_settings (
        lang       VARCHAR NOT NULL PRIMARY KEY,
        separator  VARCHAR NOT NULL DEFAULT ' : '
    )""",
]


def migrate(db_path: Path) -> None:
    """Migre les JSON modes vers metadata.duckdb (idempotent)."""
    with duckdb.connect(str(db_path)) as conn:
        for stmt in _DDL_STATEMENTS:
            conn.execute(stmt)

        for lang, path in LANGS.items():
            if not path.exists():
                print(f"  [SKIP] {path} introuvable")
                continue
            data = json.loads(path.read_text(encoding="utf-8"))

            sep = data.get("_separator", ": ")
            conn.execute(
                "INSERT OR REPLACE INTO mode_lang_settings VALUES (?, ?)",
                [lang, sep],
            )

            for prefix_en, name in data.get("_prefixes", {}).items():
                conn.execute(
                    "INSERT OR REPLACE INTO mode_prefix_names VALUES (?, ?, ?)",
                    [prefix_en, lang, name],
                )

            for mode_en, name in data.items():
                if not mode_en.startswith("_"):
                    conn.execute(
                        "INSERT OR REPLACE INTO mode_name_tr VALUES (?, ?, ?)",
                        [mode_en, lang, name],
                    )

            for pattern, name in data.get("_pairs", {}).items():
                # Normaliser la clé : strip " on <carte>" pour uniformiser
                key = pattern.split(" on ", 1)[0].strip()
                conn.execute(
                    "INSERT OR REPLACE INTO mode_pair_overrides VALUES (?, ?, ?)",
                    [key, lang, name],
                )

    print(f"Migration OK → {db_path}")
    _verify(db_path)


def _verify(db_path: Path) -> None:
    """Vérifie les comptages post-migration."""
    tables = [
        "mode_prefix_names",
        "mode_name_tr",
        "mode_pair_overrides",
        "mode_lang_settings",
    ]
    with duckdb.connect(str(db_path), read_only=True) as conn:
        for table in tables:
            count = conn.execute(f"SELECT COUNT(*) FROM {table}").fetchone()[0]  # noqa: S608
            print(f"  {table}: {count} lignes")


if __name__ == "__main__":
    db_path = ROOT / "data" / "warehouse" / "metadata.duckdb"
    if not db_path.exists():
        print(f"ERREUR: {db_path} n'existe pas")
        raise SystemExit(1)
    migrate(db_path)
