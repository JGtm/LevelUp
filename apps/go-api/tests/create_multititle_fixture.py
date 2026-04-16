#!/usr/bin/env python3
"""Crée des fixtures DuckDB multi-titre pour les tests d'isolation (Sprint 44 T11).

Génère un second titre synthétique « halo_mcc » à côté du Halo Infinite par défaut.
Les fixtures sont placées dans l'arbre title-aware :
    data/titles/halo_mcc/warehouse/  — shared + metadata
    data/titles/halo_mcc/players/{gamertag}/  — stats.duckdb

Usage :
    python apps/go-api/tests/create_multititle_fixture.py
    python apps/go-api/tests/create_multititle_fixture.py --out-dir /tmp/test-root
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path

try:
    import duckdb
except ImportError:
    print("DuckDB non installé : pip install duckdb", file=sys.stderr)
    sys.exit(1)

REPO_ROOT = Path(os.environ.get("LEVELUP_REPO_ROOT", Path(__file__).parents[4]))
TITLE_SLUG = "halo_mcc"
GAMERTAG = "MCCTestPlayer"
XUID = "9876543210987654"
MATCH_IDS = [f"mcc-match-{i:03d}" for i in range(1, 6)]  # 5 matchs


def _conn(path: Path) -> duckdb.DuckDBPyConnection:
    path.parent.mkdir(parents=True, exist_ok=True)
    return duckdb.connect(str(path))


def _build_metadata(title_dir: Path) -> Path:
    db_path = title_dir / "warehouse" / "metadata.duckdb"
    with _conn(db_path) as con:
        con.execute("""
            CREATE OR REPLACE TABLE career_ranks (
                rank_id INTEGER PRIMARY KEY,
                rank_name VARCHAR,
                rank_name_fr VARCHAR
            )
        """)
        con.execute("INSERT INTO career_ranks VALUES (1, 'Recruit', 'Recrue')")

        con.execute("""
            CREATE OR REPLACE TABLE mode_name_tr (
                mode_id VARCHAR PRIMARY KEY,
                name_fr VARCHAR,
                name_en VARCHAR
            )
        """)
        con.execute("INSERT INTO mode_name_tr VALUES ('slayer', 'Abattage', 'Slayer')")
    return db_path


def _build_shared(title_dir: Path) -> Path:
    db_path = title_dir / "warehouse" / "shared_matches_v2.duckdb"
    with _conn(db_path) as con:
        con.execute("""
            CREATE OR REPLACE TABLE match_registry (
                match_id VARCHAR PRIMARY KEY,
                start_time TIMESTAMP,
                duration_s INTEGER,
                map_id VARCHAR,
                mode_id VARCHAR,
                playlist VARCHAR
            )
        """)
        for i, mid in enumerate(MATCH_IDS):
            con.execute(
                "INSERT INTO match_registry VALUES (?, '2024-06-01'::TIMESTAMP + INTERVAL (? || ' minutes'), ?, ?, ?, ?)",
                [mid, str(i * 15), 720, "blood_gulch", "slayer", "social"],
            )

        con.execute("""
            CREATE OR REPLACE TABLE match_participants (
                match_id VARCHAR,
                xuid VARCHAR,
                gamertag VARCHAR,
                kills INTEGER,
                deaths INTEGER,
                assists INTEGER,
                score INTEGER,
                outcome INTEGER,
                team_id INTEGER
            )
        """)
        for i, mid in enumerate(MATCH_IDS):
            con.execute(
                "INSERT INTO match_participants VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
                [mid, XUID, GAMERTAG, 15 + i, 5, 3, (15 + i) * 100, 2, 0],
            )

        con.execute("""
            CREATE OR REPLACE TABLE xuid_aliases (
                xuid VARCHAR PRIMARY KEY,
                gamertag VARCHAR
            )
        """)
        con.execute("INSERT INTO xuid_aliases VALUES (?, ?)", [XUID, GAMERTAG])

        con.execute("""
            CREATE OR REPLACE VIEW v_gamertag_lookup AS
            SELECT xuid, gamertag FROM xuid_aliases
        """)
        con.execute("""
            CREATE OR REPLACE VIEW v_match_full AS
            SELECT match_id, start_time, duration_s, map_id, mode_id, playlist
            FROM match_registry
        """)
    return db_path


def _build_player(title_dir: Path) -> Path:
    player_dir = title_dir / "players" / GAMERTAG
    db_path = player_dir / "stats.duckdb"
    with _conn(db_path) as con:
        con.execute("""
            CREATE OR REPLACE TABLE player_match_enrichment (
                match_id VARCHAR PRIMARY KEY,
                performance_score DOUBLE,
                session_id VARCHAR,
                is_with_friends BOOLEAN
            )
        """)
        for i, mid in enumerate(MATCH_IDS):
            con.execute(
                "INSERT INTO player_match_enrichment VALUES (?, ?, ?, ?)",
                [mid, 0.6 + i * 0.03, f"mcc-session-{i // 2}", False],
            )

        con.execute("""
            CREATE OR REPLACE TABLE sync_meta (
                key VARCHAR PRIMARY KEY,
                value VARCHAR
            )
        """)
        con.execute("INSERT INTO sync_meta VALUES ('last_sync', '2024-06-01T00:00:00Z')")
    return db_path


def _build_db_profiles(repo_root: Path, title_dir: Path, player_db: Path) -> Path:
    """Crée un db_profiles.json v3 multi-titre."""
    profiles_path = title_dir / "db_profiles_mcc_ci.json"
    profiles = {
        "version": "3.0",
        "profiles": {
            TITLE_SLUG: {
                GAMERTAG: {
                    "db_path": str(player_db),
                    "xuid": XUID,
                    "waypoint_player": GAMERTAG,
                }
            }
        },
    }
    profiles_path.write_text(json.dumps(profiles, indent=2))
    return profiles_path


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--out-dir",
        default=str(REPO_ROOT),
        help="Racine du repo (défaut : auto-détecté)",
    )
    args = parser.parse_args(argv)
    repo_root = Path(args.out_dir)
    title_dir = repo_root / "data" / "titles" / TITLE_SLUG

    print(f"[multititle] Titre : {TITLE_SLUG}")
    print(f"[multititle] Répertoire : {title_dir}")

    meta = _build_metadata(title_dir)
    print(f"[multititle] ✓ metadata.duckdb -> {meta}")

    shared = _build_shared(title_dir)
    print(f"[multititle] ✓ shared_matches_v2.duckdb -> {shared}")

    player = _build_player(title_dir)
    print(f"[multititle] ✓ stats.duckdb ({GAMERTAG}) -> {player}")

    profiles = _build_db_profiles(repo_root, title_dir, player)
    print(f"[multititle] ✓ db_profiles v3 -> {profiles}")

    print(f"[multititle] Corpus prêt — {len(MATCH_IDS)} matchs, joueur={GAMERTAG}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
