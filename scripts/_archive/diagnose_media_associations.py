#!/usr/bin/env python3
"""Script de diagnostic pour l'indexation et l'association des médias.

Usage:
    python scripts/diagnose_media_associations.py --db-path data/players/SpartanC/stats.duckdb
"""

from __future__ import annotations

import argparse
import sys
from pathlib import Path

# Ajouter le répertoire racine au path
ROOT_DIR = Path(__file__).parent.parent
sys.path.insert(0, str(ROOT_DIR))

from datetime import datetime

import duckdb

from src.ui.formatting import PARIS_TZ, paris_epoch_seconds


def diagnose_media_associations(db_path: Path, owner_xuid: str) -> None:
    """Diagnostique les problèmes d'association des médias."""
    print("🔍 Diagnostic des associations médias")
    print(f"   DB: {db_path}")
    print(f"   Owner XUID: {owner_xuid}")
    print()

    if not db_path.exists():
        print(f"❌ Erreur: La DB n'existe pas: {db_path}")
        return

    conn = duckdb.connect(str(db_path), read_only=True)
    try:
        # 1. Vérifier si les tables existent
        tables = conn.execute(
            """
            SELECT table_name
            FROM information_schema.tables
            WHERE table_schema = 'main'
            AND table_name IN ('media_files', 'media_match_associations', 'match_stats')
            """
        ).fetchall()
        table_names = {row[0] for row in tables}

        print("📊 Tables disponibles:")
        for table in ["media_files", "media_match_associations", "match_stats"]:
            status = "✅" if table in table_names else "❌"
            print(f"   {status} {table}")
        print()

        if "media_files" not in table_names:
            print("❌ Table media_files n'existe pas - l'indexation n'a pas été lancée")
            return

        if "match_stats" not in table_names:
            print("❌ Table match_stats n'existe pas - pas de matchs disponibles")
            return

        # 2. Compter les médias indexés
        media_count = conn.execute(
            "SELECT COUNT(*) FROM media_files WHERE owner_xuid = ?",
            [owner_xuid],
        ).fetchone()[0]
        print(f"📁 Médias indexés: {media_count}")

        # 3. Compter les associations
        assoc_count = conn.execute(
            """
            SELECT COUNT(*)
            FROM media_match_associations
            WHERE xuid = ?
            """,
            [owner_xuid],
        ).fetchone()[0]
        print(f"🔗 Associations créées: {assoc_count}")

        # 4. Compter les matchs disponibles
        match_count = conn.execute(
            "SELECT COUNT(*) FROM match_stats WHERE start_time IS NOT NULL"
        ).fetchone()[0]
        print(f"🎮 Matchs disponibles: {match_count}")
        print()

        if match_count == 0:
            print("⚠️  Aucun match disponible - impossible d'associer les médias")
            return

        # 5. Analyser quelques médias non associés
        unassociated = conn.execute(
            """
            SELECT
                mf.file_path,
                mf.file_name,
                mf.mtime,
                mf.mtime_paris_epoch,
                mf.kind
            FROM media_files mf
            LEFT JOIN media_match_associations mma
                ON mf.file_path = mma.media_path
                AND mf.owner_xuid = mma.xuid
            WHERE mf.owner_xuid = ?
                AND mma.media_path IS NULL
            ORDER BY mf.mtime_paris_epoch DESC
            LIMIT 5
            """,
            [owner_xuid],
        ).fetchall()

        if unassociated:
            print(f"📋 Exemples de médias non associés ({len(unassociated)} premiers):")
            for file_path, file_name, mtime, mtime_paris_epoch, kind in unassociated:
                dt_mtime = datetime.fromtimestamp(mtime_paris_epoch, tz=PARIS_TZ)
                print(f"\n   📹 {file_name}")
                print(f"      mtime système: {mtime}")
                print(f"      mtime Paris epoch: {mtime_paris_epoch}")
                print(f"      Date/heure Paris: {dt_mtime.strftime('%Y-%m-%d %H:%M:%S %Z')}")

                # Chercher les matchs proches
                matches_nearby = conn.execute(
                    """
                    SELECT
                        match_id,
                        start_time,
                        time_played_seconds
                    FROM match_stats
                    WHERE start_time IS NOT NULL
                    ORDER BY ABS(EXTRACT(EPOCH FROM (start_time - ?::TIMESTAMP)))
                    LIMIT 3
                    """,
                    [datetime.fromtimestamp(mtime_paris_epoch, tz=PARIS_TZ)],
                ).fetchall()

                if matches_nearby:
                    print("      Matchs proches:")
                    for match_id, start_time, duration in matches_nearby:
                        if isinstance(start_time, str):
                            dt_match = datetime.fromisoformat(start_time.replace("Z", "+00:00"))
                        else:
                            dt_match = start_time

                        match_epoch = paris_epoch_seconds(dt_match)
                        if match_epoch:
                            diff_seconds = abs(mtime_paris_epoch - match_epoch)
                            diff_minutes = diff_seconds / 60
                            duration_min = (duration or 720) / 60

                            print(
                                f"        - {match_id[:8]}... "
                                f"({dt_match.strftime('%Y-%m-%d %H:%M:%S') if hasattr(dt_match, 'strftime') else str(dt_match)}) "
                                f"| Écart: {diff_minutes:.1f} min | Durée: {duration_min:.1f} min"
                            )
                else:
                    print("      ⚠️  Aucun match proche trouvé")

        # 6. Vérifier la plage temporelle des matchs
        match_range = conn.execute(
            """
            SELECT
                MIN(start_time) as min_time,
                MAX(start_time) as max_time,
                COUNT(*) as count
            FROM match_stats
            WHERE start_time IS NOT NULL
            """
        ).fetchone()

        if match_range and match_range[0]:
            print("\n📅 Plage temporelle des matchs:")
            print(f"   Début: {match_range[0]}")
            print(f"   Fin: {match_range[1]}")
            print(f"   Total: {match_range[2]} matchs")

        # 7. Vérifier la plage temporelle des médias
        media_range = conn.execute(
            """
            SELECT
                MIN(mtime_paris_epoch) as min_epoch,
                MAX(mtime_paris_epoch) as max_epoch,
                COUNT(*) as count
            FROM media_files
            WHERE owner_xuid = ?
            """,
            [owner_xuid],
        ).fetchone()

        if media_range and media_range[0]:
            min_dt = datetime.fromtimestamp(media_range[0], tz=PARIS_TZ)
            max_dt = datetime.fromtimestamp(media_range[1], tz=PARIS_TZ)
            print("\n📅 Plage temporelle des médias:")
            print(f"   Début: {min_dt.strftime('%Y-%m-%d %H:%M:%S %Z')}")
            print(f"   Fin: {max_dt.strftime('%Y-%m-%d %H:%M:%S %Z')}")
            print(f"   Total: {media_range[2]} médias")

    finally:
        conn.close()


def main() -> int:
    parser = argparse.ArgumentParser(description="Diagnostique les associations médias")
    parser.add_argument(
        "--db-path",
        type=str,
        required=True,
        help="Chemin vers la DB DuckDB du joueur",
    )
    parser.add_argument(
        "--owner-xuid",
        type=str,
        required=True,
        help="XUID du propriétaire des médias",
    )

    args = parser.parse_args()

    db_path = Path(args.db_path)
    if not db_path.exists():
        print(f"❌ Erreur: DB introuvable: {db_path}")
        return 1

    diagnose_media_associations(db_path, args.owner_xuid)
    return 0


if __name__ == "__main__":
    sys.exit(main())
