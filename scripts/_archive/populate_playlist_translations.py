#!/usr/bin/env python3
"""Peuple les tables playlist_translations et mode_translations dans metadata.duckdb.

Lit le fichier data/Playlist_modes_translations.json et insère les données
dans deux tables DuckDB. Le script est idempotent (UPSERT).

Usage :
    python scripts/populate_playlist_translations.py
    python scripts/populate_playlist_translations.py --reset   # DROP + recréation
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

import duckdb

# ── Chemins ─────────────────────────────────────────────────────────────────
METADATA_DB = Path("data/warehouse/metadata.duckdb")
JSON_PATH = Path("data/Playlist_modes_translations.json")

# ── DDL ─────────────────────────────────────────────────────────────────────
DDL_PLAYLIST = """
CREATE TABLE IF NOT EXISTS playlist_translations (
    uuid    VARCHAR PRIMARY KEY,
    name_en VARCHAR NOT NULL,
    name_fr VARCHAR NOT NULL
);
"""

DDL_MODE = """
CREATE TABLE IF NOT EXISTS mode_translations (
    name_en  VARCHAR PRIMARY KEY,
    name_fr  VARCHAR NOT NULL,
    category VARCHAR NOT NULL
);
"""

UPSERT_PLAYLIST = """
INSERT INTO playlist_translations VALUES (?, ?, ?)
ON CONFLICT (uuid) DO UPDATE SET
    name_en = EXCLUDED.name_en,
    name_fr = EXCLUDED.name_fr
"""

UPSERT_MODE = """
INSERT INTO mode_translations VALUES (?, ?, ?)
ON CONFLICT (name_en) DO UPDATE SET
    name_fr = EXCLUDED.name_fr,
    category = EXCLUDED.category
"""


def ensure_schema(conn: duckdb.DuckDBPyConnection) -> None:
    """Crée les deux tables si absentes (idempotent)."""
    conn.execute(DDL_PLAYLIST)
    conn.execute(DDL_MODE)


def populate(conn: duckdb.DuckDBPyConnection, data: dict) -> tuple[int, int]:
    """UPSERT depuis le JSON. Retourne (n_playlists, n_modes)."""
    playlists = data.get("playlists", [])
    modes = data.get("modes", [])

    playlist_rows = [
        (p["uuid"], p["en"], p["fr"]) for p in playlists if "uuid" in p and "en" in p and "fr" in p
    ]
    mode_rows = [
        (m["en"], m["fr"], m.get("category", "Other")) for m in modes if "en" in m and "fr" in m
    ]

    if playlist_rows:
        conn.executemany(UPSERT_PLAYLIST, playlist_rows)
    if mode_rows:
        conn.executemany(UPSERT_MODE, mode_rows)

    return len(playlist_rows), len(mode_rows)


def cleanup_obsolete(conn: duckdb.DuckDBPyConnection, data: dict) -> tuple[int, int]:
    """Supprime les lignes absentes du JSON. Retourne (n_pl_del, n_mode_del)."""
    # Playlists
    json_uuids = {p["uuid"] for p in data.get("playlists", []) if "uuid" in p}
    db_uuids = {row[0] for row in conn.execute("SELECT uuid FROM playlist_translations").fetchall()}
    obsolete_pl = db_uuids - json_uuids
    if obsolete_pl:
        placeholders = ", ".join("?" for _ in obsolete_pl)
        conn.execute(
            f"DELETE FROM playlist_translations WHERE uuid IN ({placeholders})",
            list(obsolete_pl),
        )

    # Modes
    json_names = {m["en"] for m in data.get("modes", []) if "en" in m}
    db_names = {row[0] for row in conn.execute("SELECT name_en FROM mode_translations").fetchall()}
    obsolete_modes = db_names - json_names
    if obsolete_modes:
        placeholders = ", ".join("?" for _ in obsolete_modes)
        conn.execute(
            f"DELETE FROM mode_translations WHERE name_en IN ({placeholders})",
            list(obsolete_modes),
        )

    return len(obsolete_pl), len(obsolete_modes)


def main() -> None:
    """Point d'entrée principal."""
    parser = argparse.ArgumentParser(
        description="Peuple playlist_translations et mode_translations dans metadata.duckdb"
    )
    parser.add_argument(
        "--reset",
        action="store_true",
        help="DROP les tables et les recrée from scratch",
    )
    args = parser.parse_args()

    if not METADATA_DB.exists():
        print(f"❌ metadata.duckdb introuvable : {METADATA_DB}")
        sys.exit(1)

    if not JSON_PATH.exists():
        print(f"❌ JSON introuvable : {JSON_PATH}")
        sys.exit(1)

    with open(JSON_PATH, encoding="utf-8") as f:
        data = json.load(f)

    conn = duckdb.connect(str(METADATA_DB))
    try:
        if args.reset:
            conn.execute("DROP TABLE IF EXISTS playlist_translations")
            conn.execute("DROP TABLE IF EXISTS mode_translations")
            print("🗑️  Tables playlist_translations et mode_translations supprimées")

        ensure_schema(conn)

        n_pl, n_modes = populate(conn, data)
        n_pl_del, n_mode_del = cleanup_obsolete(conn, data)

        print(f"✅ {n_pl} playlists insérées/mises à jour")
        print(f"✅ {n_modes} modes insérés/mis à jour")
        if n_pl_del or n_mode_del:
            print(f"   → {n_pl_del} playlists obsolètes supprimées")
            print(f"   → {n_mode_del} modes obsolètes supprimés")

        # Stats de vérification
        total_pl = conn.execute("SELECT COUNT(*) FROM playlist_translations").fetchone()[0]
        total_modes = conn.execute("SELECT COUNT(*) FROM mode_translations").fetchone()[0]
        categories = conn.execute(
            "SELECT category, COUNT(*) FROM mode_translations GROUP BY category ORDER BY category"
        ).fetchall()

        print("\n📋 Résumé :")
        print(f"   playlist_translations : {total_pl} lignes")
        print(f"   mode_translations     : {total_modes} lignes")
        print("\n   Répartition modes par catégorie :")
        for cat, count in categories:
            print(f"     {cat:15s} : {count}")

    finally:
        conn.close()


if __name__ == "__main__":
    main()
