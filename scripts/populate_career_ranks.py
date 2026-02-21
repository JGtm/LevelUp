#!/usr/bin/env python3
"""Peuple la table career_ranks dans metadata.duckdb depuis le JSON.

Ce script est idempotent : il crée la table si absente et insère/met à jour
les 272 rangs Career Halo Infinite.

Usage :
    python scripts/populate_career_ranks.py
    python scripts/populate_career_ranks.py --reset   # DROP + recréation
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

import duckdb

# ── Chemins ─────────────────────────────────────────────────────────────────
METADATA_DB = Path("data/warehouse/metadata.duckdb")
CAREER_RANKS_JSON = Path("data/cache/career_ranks_metadata.json")

# ── DDL ─────────────────────────────────────────────────────────────────────
CREATE_TABLE_DDL = """
CREATE TABLE IF NOT EXISTS career_ranks (
    rank_id               INTEGER PRIMARY KEY,
    title_en              VARCHAR NOT NULL,
    subtitle_en           VARCHAR,
    tier                  VARCHAR,
    tier_type             VARCHAR,
    grade                 INTEGER,
    xp_required           INTEGER NOT NULL,
    icon_path             VARCHAR,
    large_icon_path       VARCHAR,
    adornment_icon_path   VARCHAR
);
"""


def populate_career_ranks(
    conn: duckdb.DuckDBPyConnection, json_path: Path
) -> int:
    """Peuple career_ranks depuis le JSON (migration initiale).

    Returns:
        Nombre de rangs insérés/mis à jour.
    """
    conn.execute(CREATE_TABLE_DDL)

    with open(json_path, encoding="utf-8") as f:
        data = json.load(f)

    ranks_data = data.get("Ranks", [])
    if not ranks_data:
        print("⚠️  Aucun rang trouvé dans le JSON")
        return 0

    rows: list[tuple] = []
    for r in ranks_data:
        rank_id = r.get("Rank", 0)

        title_obj = r.get("RankTitle", {})
        title_en = (
            title_obj.get("value", "")
            if isinstance(title_obj, dict)
            else str(title_obj or "")
        )

        subtitle_obj = r.get("RankSubTitle", {})
        subtitle_en = (
            subtitle_obj.get("value", "")
            if isinstance(subtitle_obj, dict)
            else str(subtitle_obj or "")
        )

        tier_obj = r.get("RankTier", {})
        tier = (
            tier_obj.get("value", "")
            if isinstance(tier_obj, dict)
            else str(tier_obj or "")
        )

        tier_type = r.get("TierType", "")
        grade = r.get("RankGrade", 1)
        xp_required = r.get("XpRequiredForRank", 0)
        icon_path = r.get("RankIcon", "")
        large_icon_path = r.get("RankLargeIcon", "")
        adornment_icon_path = r.get("RankAdornmentIcon", "")

        rows.append((
            rank_id,
            title_en,
            subtitle_en or None,
            tier or None,
            tier_type or None,
            grade,
            xp_required,
            icon_path or None,
            large_icon_path or None,
            adornment_icon_path or None,
        ))

    conn.executemany(
        """INSERT INTO career_ranks VALUES (?,?,?,?,?,?,?,?,?,?)
           ON CONFLICT (rank_id) DO UPDATE SET
               title_en = EXCLUDED.title_en,
               subtitle_en = EXCLUDED.subtitle_en,
               tier = EXCLUDED.tier,
               tier_type = EXCLUDED.tier_type,
               grade = EXCLUDED.grade,
               xp_required = EXCLUDED.xp_required,
               icon_path = EXCLUDED.icon_path,
               large_icon_path = EXCLUDED.large_icon_path,
               adornment_icon_path = EXCLUDED.adornment_icon_path
        """,
        rows,
    )
    return len(rows)


def main() -> None:
    """Point d'entrée principal."""
    parser = argparse.ArgumentParser(
        description="Peuple career_ranks dans metadata.duckdb depuis le JSON"
    )
    parser.add_argument(
        "--reset",
        action="store_true",
        help="DROP la table et la recrée from scratch",
    )
    args = parser.parse_args()

    if not METADATA_DB.exists():
        print(f"❌ metadata.duckdb introuvable : {METADATA_DB}")
        sys.exit(1)

    if not CAREER_RANKS_JSON.exists():
        print(f"❌ JSON introuvable : {CAREER_RANKS_JSON}")
        sys.exit(1)

    conn = duckdb.connect(str(METADATA_DB))
    try:
        if args.reset:
            conn.execute("DROP TABLE IF EXISTS career_ranks")
            print("🗑️  Table career_ranks supprimée")

        total = populate_career_ranks(conn, CAREER_RANKS_JSON)

        # Vérification
        count = conn.execute("SELECT COUNT(*) FROM career_ranks").fetchone()[0]
        min_xp = conn.execute("SELECT MIN(xp_required) FROM career_ranks").fetchone()[0]
        max_xp = conn.execute("SELECT MAX(xp_required) FROM career_ranks").fetchone()[0]

        print(f"✅ {total} rangs insérés/mis à jour")
        print(f"   → {count} rangs en base")
        print(f"   → XP range: {min_xp:,} – {max_xp:,}")

        # Échantillon
        print("\n📋 Échantillon (rangs 1, 50, 100, 200, 272) :")
        for row in conn.execute(
            "SELECT rank_id, title_en, subtitle_en, tier_type, xp_required "
            "FROM career_ranks WHERE rank_id IN (1, 50, 100, 200, 272) "
            "ORDER BY rank_id"
        ).fetchall():
            print(f"   #{row[0]:3d} {row[1]:20s} {row[2] or '':10s} {row[3] or '':10s} {row[4]:>10,} XP")

    finally:
        conn.close()


if __name__ == "__main__":
    main()
