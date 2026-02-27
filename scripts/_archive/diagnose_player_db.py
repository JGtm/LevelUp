#!/usr/bin/env python3
"""Diagnostic de la base de données joueur DuckDB.

Vérifie l'état des données :
- Tables présentes
- Statistiques match_stats (accuracy, dates)
- Médailles
- Highlight events
- Rosters (si disponibles)

Usage:
    python scripts/diagnose_player_db.py [db_path]
    python scripts/diagnose_player_db.py data/players/SpartanC/stats.duckdb
"""

import sys
from pathlib import Path

import duckdb

REPO_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(REPO_ROOT))


def diagnose(db_path: str) -> dict:
    """Diagnostique une base de données DuckDB joueur."""
    db_path_obj = Path(db_path)
    if not db_path_obj.exists():
        return {"error": f"Base de données non trouvée: {db_path}"}

    conn = duckdb.connect(str(db_path), read_only=True)
    results = {}

    try:
        # Tables présentes
        tables = conn.execute("""
            SELECT table_name
            FROM information_schema.tables
            WHERE table_schema = 'main'
            ORDER BY table_name
        """).fetchall()
        results["tables"] = [t[0] for t in tables]

        # Stats match_stats
        try:
            match_stats = conn.execute("""
                SELECT
                    COUNT(*) as total,
                    COUNT(accuracy) as with_accuracy,
                    COUNT(CASE WHEN accuracy IS NULL THEN 1 END) as null_accuracy,
                    MAX(start_time) as last_match,
                    MIN(start_time) as first_match,
                    AVG(accuracy) as avg_accuracy,
                    COUNT(CASE WHEN accuracy IS NOT NULL AND accuracy > 0 THEN 1 END) as positive_accuracy
                FROM match_stats
            """).fetchone()
            results["match_stats"] = {
                "total": match_stats[0],
                "with_accuracy": match_stats[1],
                "null_accuracy": match_stats[2],
                "last_match": match_stats[3],
                "first_match": match_stats[4],
                "avg_accuracy": match_stats[5],
                "positive_accuracy": match_stats[6],
            }
        except Exception as e:
            results["match_stats"] = {"error": str(e)}

        # Stats medals_earned
        try:
            medals = conn.execute("""
                SELECT
                    COUNT(*) as total,
                    COUNT(DISTINCT match_id) as distinct_matches,
                    COUNT(DISTINCT medal_name_id) as distinct_medals
                FROM medals_earned
            """).fetchone()
            results["medals"] = {
                "total": medals[0],
                "distinct_matches": medals[1],
                "distinct_medals": medals[2],
            }
        except Exception as e:
            results["medals"] = {"error": str(e)}

        # Stats highlight_events
        try:
            events = conn.execute("""
                SELECT
                    COUNT(*) as total,
                    COUNT(DISTINCT match_id) as distinct_matches,
                    COUNT(DISTINCT event_type) as distinct_types
                FROM highlight_events
            """).fetchone()
            results["highlight_events"] = {
                "total": events[0],
                "distinct_matches": events[1],
                "distinct_types": events[2],
            }
        except Exception as e:
            results["highlight_events"] = {"error": str(e)}

        # Vérifier les colonnes de match_stats
        try:
            columns = conn.execute("""
                SELECT column_name, data_type, is_nullable
                FROM information_schema.columns
                WHERE table_schema = 'main' AND table_name = 'match_stats'
                ORDER BY ordinal_position
            """).fetchall()
            results["match_stats_columns"] = [
                {"name": c[0], "type": c[1], "nullable": c[2]} for c in columns
            ]
        except Exception as e:
            results["match_stats_columns"] = {"error": str(e)}

        # Échantillon de matchs récents avec accuracy
        try:
            sample = conn.execute("""
                SELECT match_id, start_time, accuracy, kills, deaths, assists
                FROM match_stats
                WHERE accuracy IS NOT NULL
                ORDER BY start_time DESC
                LIMIT 5
            """).fetchall()
            results["sample_with_accuracy"] = [
                {
                    "match_id": r[0],
                    "start_time": r[1],
                    "accuracy": r[2],
                    "kills": r[3],
                    "deaths": r[4],
                    "assists": r[5],
                }
                for r in sample
            ]
        except Exception as e:
            results["sample_with_accuracy"] = {"error": str(e)}

        # Échantillon de matchs récents sans accuracy
        try:
            sample_null = conn.execute("""
                SELECT match_id, start_time, accuracy, kills, deaths, assists
                FROM match_stats
                WHERE accuracy IS NULL
                ORDER BY start_time DESC
                LIMIT 5
            """).fetchall()
            results["sample_without_accuracy"] = [
                {
                    "match_id": r[0],
                    "start_time": r[1],
                    "accuracy": r[2],
                    "kills": r[3],
                    "deaths": r[4],
                    "assists": r[5],
                }
                for r in sample_null
            ]
        except Exception as e:
            results["sample_without_accuracy"] = {"error": str(e)}

    finally:
        conn.close()

    return results


def print_report(results: dict, db_path: str) -> None:
    """Affiche un rapport formaté."""
    print("=" * 80)
    print(f"DIAGNOSTIC: {db_path}")
    print("=" * 80)

    if "error" in results:
        print(f"\n❌ ERREUR: {results['error']}")
        return

    # Tables
    print(f"\n📊 TABLES PRÉSENTES ({len(results.get('tables', []))}):")
    for table in results.get("tables", []):
        print(f"  ✓ {table}")

    # Match stats
    ms = results.get("match_stats", {})
    if "error" in ms:
        print(f"\n❌ MATCH_STATS: {ms['error']}")
    else:
        print("\n📈 MATCH_STATS:")
        print(f"  Total matchs: {ms.get('total', 0)}")
        if ms.get("total", 0) > 0:
            acc_pct = (ms.get("with_accuracy", 0) / ms["total"]) * 100
            print(f"  Avec accuracy: {ms.get('with_accuracy', 0)} ({acc_pct:.1f}%)")
            print(f"  Sans accuracy: {ms.get('null_accuracy', 0)}")
            print(f"  Accuracy > 0: {ms.get('positive_accuracy', 0)}")
            print(f"  Dernier match: {ms.get('last_match', 'N/A')}")
            print(f"  Premier match: {ms.get('first_match', 'N/A')}")
            avg_acc = ms.get("avg_accuracy")
            if avg_acc is not None:
                print(f"  Accuracy moyenne: {avg_acc:.2f}%")
            else:
                print("  Accuracy moyenne: NULL")

    # Médailles
    medals = results.get("medals", {})
    if "error" in medals:
        print(f"\n❌ MEDALS_EARNED: {medals['error']}")
    else:
        print("\n🏅 MEDALS_EARNED:")
        print(f"  Total médailles: {medals.get('total', 0)}")
        print(f"  Matchs distincts: {medals.get('distinct_matches', 0)}")
        print(f"  Types de médailles: {medals.get('distinct_medals', 0)}")

    # Highlight events
    events = results.get("highlight_events", {})
    if "error" in events:
        print(f"\n❌ HIGHLIGHT_EVENTS: {events['error']}")
    else:
        print("\n⚡ HIGHLIGHT_EVENTS:")
        print(f"  Total events: {events.get('total', 0)}")
        print(f"  Matchs distincts: {events.get('distinct_matches', 0)}")
        print(f"  Types d'événements: {events.get('distinct_types', 0)}")

    # Échantillons
    sample_acc = results.get("sample_with_accuracy", [])
    if isinstance(sample_acc, list) and sample_acc:
        print(f"\n✅ ÉCHANTILLON AVEC ACCURACY ({len(sample_acc)} matchs):")
        for s in sample_acc[:3]:
            print(
                f"  - {s['match_id'][:8]}... | {s['start_time']} | Acc: {s['accuracy']:.1f}% | K/D/A: {s['kills']}/{s['deaths']}/{s['assists']}"
            )

    sample_null = results.get("sample_without_accuracy", [])
    if isinstance(sample_null, list) and sample_null:
        print(f"\n⚠️  ÉCHANTILLON SANS ACCURACY ({len(sample_null)} matchs):")
        for s in sample_null[:3]:
            print(
                f"  - {s['match_id'][:8]}... | {s['start_time']} | Acc: NULL | K/D/A: {s['kills']}/{s['deaths']}/{s['assists']}"
            )

    print("\n" + "=" * 80)


if __name__ == "__main__":
    db = sys.argv[1] if len(sys.argv) > 1 else "data/players/SpartanC/stats.duckdb"
    results = diagnose(db)
    print_report(results, db)
