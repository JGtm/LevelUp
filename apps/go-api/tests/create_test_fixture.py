#!/usr/bin/env python3
"""Crée les fixtures DuckDB légères pour les golden tests CI (Sprint 35).

Ce script génère un jeu de données minimal (~10 matchs fictifs + référentiels)
dans des fichiers DuckDB temporaires, permettant de lancer le serveur Go en CI
sans dépendre d'une vraie base de données joueur.

Usage :
    python apps/go-api/tests/create_test_fixture.py
    python apps/go-api/tests/create_test_fixture.py --out-dir /tmp/test-fixtures

Variables d'environnement :
    LEVELUP_FIXTURES_DIR  : répertoire de sortie (défaut : data/warehouse)
    LEVELUP_REPO_ROOT     : racine du repo (défaut : auto-détecté)
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path

# ---------------------------------------------------------------------------
# Dépendance DuckDB
# ---------------------------------------------------------------------------

try:
    import duckdb
except ImportError:
    print("DuckDB non installé : pip install duckdb", file=sys.stderr)
    sys.exit(1)


# ---------------------------------------------------------------------------
# Constantes
# ---------------------------------------------------------------------------

REPO_ROOT = Path(os.environ.get("LEVELUP_REPO_ROOT", Path(__file__).parents[4]))
DEFAULT_OUT_DIR = REPO_ROOT / "data" / "warehouse"

GAMERTAG = "CITestPlayer"
XUID = "1234567890123456"
MATCH_IDS = [f"ci-match-{i:03d}" for i in range(1, 11)]  # 10 matchs fictifs

# ---------------------------------------------------------------------------
# Helpers DuckDB
# ---------------------------------------------------------------------------


def _conn(path: Path) -> duckdb.DuckDBPyConnection:
    """Ouvre une connexion DuckDB en lecture-écriture (crée le fichier si absent)."""
    path.parent.mkdir(parents=True, exist_ok=True)
    return duckdb.connect(str(path))


# ---------------------------------------------------------------------------
# metadata.duckdb
# ---------------------------------------------------------------------------


def _build_metadata(out_dir: Path) -> Path:
    """Crée metadata.duckdb avec les référentiels minimaux."""
    db_path = out_dir / "metadata.duckdb"
    with _conn(db_path) as con:
        # career_ranks — 3 paliers suffisent
        con.execute("""
            CREATE OR REPLACE TABLE career_ranks (
                rank_id INTEGER PRIMARY KEY,
                rank_name VARCHAR,
                rank_name_fr VARCHAR
            )
        """)
        con.execute("""
            INSERT INTO career_ranks VALUES
                (1, 'Bronze 1', 'Bronze 1'),
                (2, 'Silver 1', 'Argent 1'),
                (3, 'Gold 1', 'Or 1')
        """)

        # mode_name_tr — traductions minimales
        con.execute("""
            CREATE OR REPLACE TABLE mode_name_tr (
                mode_id VARCHAR PRIMARY KEY,
                name_fr VARCHAR,
                name_en VARCHAR
            )
        """)
        con.execute("""
            INSERT INTO mode_name_tr VALUES
                ('slayer', 'Abattage', 'Slayer'),
                ('oddball', 'Balle Folle', 'Oddball'),
                ('ctf',    'CTF',       'Capture the Flag')
        """)

        # weapon_labels — 2 armes fictives
        con.execute("""
            CREATE OR REPLACE TABLE weapon_labels (
                weapon_id UBIGINT PRIMARY KEY,
                label_fr  VARCHAR,
                label_en  VARCHAR
            )
        """)
        con.execute("""
            INSERT INTO weapon_labels VALUES
                (1, 'AR', 'Assault Rifle'),
                (2, 'BR', 'Battle Rifle')
        """)
    return db_path


# ---------------------------------------------------------------------------
# shared_matches_v2.duckdb
# ---------------------------------------------------------------------------


def _build_shared(out_dir: Path) -> Path:
    """Crée shared_matches_v2.duckdb avec 10 matchs fictifs."""
    db_path = out_dir / "shared_matches_v2.duckdb"
    with _conn(db_path) as con:
        # match_registry
        con.execute("""
            CREATE OR REPLACE TABLE match_registry (
                match_id  VARCHAR PRIMARY KEY,
                start_time TIMESTAMP,
                duration_s INTEGER,
                map_id    VARCHAR,
                mode_id   VARCHAR,
                playlist  VARCHAR
            )
        """)
        for i, mid in enumerate(MATCH_IDS):
            con.execute(
                "INSERT INTO match_registry VALUES (?, '2025-01-01'::TIMESTAMP + INTERVAL (? || ' minutes'), ?, ?, ?, ?)",
                [mid, str(i * 10), 600, "bazaar", "slayer", "ranked-arena"],
            )

        # match_participants
        con.execute("""
            CREATE OR REPLACE TABLE match_participants (
                match_id  VARCHAR,
                xuid      VARCHAR,
                gamertag  VARCHAR,
                kills     INTEGER,
                deaths    INTEGER,
                assists   INTEGER,
                score     INTEGER,
                outcome   INTEGER,
                team_id   INTEGER
            )
        """)
        for i, mid in enumerate(MATCH_IDS):
            kills = 10 + i
            deaths = max(1, 8 - i)
            con.execute(
                "INSERT INTO match_participants VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
                [mid, XUID, GAMERTAG, kills, deaths, 2, kills * 100, 2, 0],
            )

        # xuid_aliases
        con.execute("""
            CREATE OR REPLACE TABLE xuid_aliases (
                xuid     VARCHAR PRIMARY KEY,
                gamertag VARCHAR
            )
        """)
        con.execute("INSERT INTO xuid_aliases VALUES (?, ?)", [XUID, GAMERTAG])

        # medals_earned (vide — structure uniquement)
        con.execute("""
            CREATE OR REPLACE TABLE medals_earned (
                match_id   VARCHAR,
                xuid       VARCHAR,
                medal_id   BIGINT,
                count      INTEGER
            )
        """)

        # highlight_events (vide)
        con.execute("""
            CREATE OR REPLACE TABLE highlight_events (
                match_id    VARCHAR,
                xuid        VARCHAR,
                event_type  VARCHAR,
                timestamp_s INTEGER
            )
        """)

        # weapon_kills (vide)
        con.execute("""
            CREATE OR REPLACE TABLE weapon_kills (
                match_id   VARCHAR,
                xuid       VARCHAR,
                weapon_id  UBIGINT,
                kills      INTEGER
            )
        """)

        # v_gamertag_lookup — Vue de résolution xuid↔gamertag
        con.execute("""
            CREATE OR REPLACE VIEW v_gamertag_lookup AS
            SELECT xuid, gamertag FROM xuid_aliases
        """)

        # v_match_full — Vue matchs enrichis (minimale)
        con.execute("""
            CREATE OR REPLACE VIEW v_match_full AS
            SELECT
                mr.match_id,
                mr.start_time,
                mr.duration_s,
                mr.map_id,
                mr.mode_id,
                mr.playlist
            FROM match_registry mr
        """)

    return db_path


# ---------------------------------------------------------------------------
# db_profiles.json
# ---------------------------------------------------------------------------


def _build_db_profiles(out_dir: Path, player_db: Path) -> Path:
    """Crée un db_profiles.json minimal pointant vers la DB joueur de test."""
    profiles_path = REPO_ROOT / "db_profiles.json"
    profiles = {
        "version": "2.1",
        "profiles": {
            GAMERTAG: {
                "db_path": str(player_db),
                "xuid": XUID,
                "waypoint_player": GAMERTAG,
            }
        },
    }
    # Ne pas écraser un db_profiles.json existant réel
    profiles_ci_path = out_dir / "db_profiles_ci.json"
    profiles_ci_path.write_text(json.dumps(profiles, indent=2))
    return profiles_ci_path


# ---------------------------------------------------------------------------
# stats.duckdb (joueur)
# ---------------------------------------------------------------------------


def _build_player_db(out_dir: Path) -> Path:
    """Crée stats.duckdb pour le joueur de test."""
    player_dir = REPO_ROOT / "data" / "players" / GAMERTAG
    player_dir.mkdir(parents=True, exist_ok=True)
    db_path = player_dir / "stats.duckdb"
    with _conn(db_path) as con:
        # player_match_enrichment — seule table match en v6
        con.execute("""
            CREATE OR REPLACE TABLE player_match_enrichment (
                match_id          VARCHAR PRIMARY KEY,
                performance_score DOUBLE,
                session_id        VARCHAR,
                is_with_friends   BOOLEAN
            )
        """)
        for i, mid in enumerate(MATCH_IDS):
            con.execute(
                "INSERT INTO player_match_enrichment VALUES (?, ?, ?, ?)",
                [mid, 0.5 + i * 0.05, f"session-{i // 3}", i % 2 == 0],
            )

        # sync_meta — table vide
        con.execute("""
            CREATE OR REPLACE TABLE sync_meta (
                key   VARCHAR PRIMARY KEY,
                value VARCHAR
            )
        """)
        con.execute("INSERT INTO sync_meta VALUES ('last_sync', '2025-01-01T00:00:00Z')")
    return db_path


# ---------------------------------------------------------------------------
# Entrypoint
# ---------------------------------------------------------------------------


def main(argv: list[str] | None = None) -> int:
    """Point d'entrée principal."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--out-dir",
        default=str(DEFAULT_OUT_DIR),
        help="Répertoire de sortie pour les fichiers DuckDB (défaut : data/warehouse)",
    )
    args = parser.parse_args(argv)
    out_dir = Path(args.out_dir)

    print(f"[fixture] Création des fixtures dans : {out_dir}")

    meta_db = _build_metadata(out_dir)
    print(f"[fixture] ✓ metadata.duckdb -> {meta_db}")

    shared_db = _build_shared(out_dir)
    print(f"[fixture] ✓ shared_matches_v2.duckdb -> {shared_db}")

    player_db = _build_player_db(out_dir)
    print(f"[fixture] ✓ stats.duckdb ({GAMERTAG}) -> {player_db}")

    db_profiles = _build_db_profiles(out_dir, player_db)
    print(f"[fixture] ✓ db_profiles_ci.json -> {db_profiles}")

    print(f"[fixture] Fixture prête — {len(MATCH_IDS)} matchs, joueur={GAMERTAG}, xuid={XUID}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
