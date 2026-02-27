#!/usr/bin/env python3
"""Script de test pour vérifier la récupération des highlight_events lors de la sync.

Usage:
    python scripts/test_highlight_events_sync.py --db-path data/players/SpartanC/stats.duckdb --match-id <match_id>
    python scripts/test_highlight_events_sync.py --db-path data/players/SpartanC/stats.duckdb --last-match
"""

import argparse
import asyncio
import sys
import traceback
from pathlib import Path

# Fix encoding pour Windows
if sys.platform == "win32":
    import io

    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding="utf-8", errors="replace")
    sys.stderr = io.TextIOWrapper(sys.stderr.buffer, encoding="utf-8", errors="replace")

import duckdb


async def test_highlight_events_fetch(match_id: str) -> dict[str, any]:
    """Teste la récupération des highlight_events depuis l'API."""
    results = {
        "match_id": match_id,
        "api_available": False,
        "events_fetched": False,
        "events_count": 0,
        "events_sample": [],
        "errors": [],
    }

    try:
        from src.data.sync.api_client import SPNKrAPIClient, get_tokens_from_env

        # Récupérer les tokens
        tokens = await get_tokens_from_env()
        if not tokens:
            results["errors"].append("Tokens SPNKr non disponibles")
            return results

        results["api_available"] = True

        # Créer le client API
        async with SPNKrAPIClient(tokens=tokens, requests_per_second=5) as client:
            # Récupérer les highlight_events
            highlight_events = await client.get_highlight_events(match_id)

            if highlight_events is None:
                results["errors"].append("get_highlight_events a retourné None")
            elif isinstance(highlight_events, list):
                results["events_fetched"] = True
                results["events_count"] = len(highlight_events)
                # Échantillon des premiers events (convertir en dict pour sérialisation)
                sample = highlight_events[:5] if len(highlight_events) > 0 else []
                results["events_sample"] = [
                    (
                        event.model_dump()
                        if hasattr(event, "model_dump")
                        else (event.dict() if hasattr(event, "dict") else event)
                    )
                    for event in sample
                ]
            else:
                results["errors"].append(f"Type inattendu retourné: {type(highlight_events)}")

    except Exception as e:
        results["errors"].append(f"Erreur lors de la récupération: {e}")
        import traceback

        results["traceback"] = traceback.format_exc()

    return results


def test_highlight_events_transform(events: list, match_id: str) -> dict[str, any]:
    """Teste la transformation des highlight_events."""
    results = {
        "transform_success": False,
        "rows_count": 0,
        "rows_sample": [],
        "errors": [],
    }

    try:
        from src.data.sync.transformers import transform_highlight_events

        rows = transform_highlight_events(events, match_id)
        results["transform_success"] = True
        results["rows_count"] = len(rows)
        results["rows_sample"] = [
            {
                "match_id": r.match_id,
                "event_type": r.event_type,
                "time_ms": r.time_ms,
                "xuid": r.xuid,
                "gamertag": r.gamertag,
            }
            for r in rows[:5]
        ]

    except Exception as e:
        results["errors"].append(f"Erreur lors de la transformation: {e}")
        import traceback

        results["traceback"] = traceback.format_exc()

    return results


def test_highlight_events_insert(db_path: str, match_id: str, rows: list) -> dict[str, any]:
    """Teste l'insertion des highlight_events dans la DB."""
    results = {
        "insert_success": False,
        "rows_inserted": 0,
        "rows_in_db": 0,
        "errors": [],
    }

    try:
        conn = duckdb.connect(str(db_path), read_only=False)

        # Compter les events existants avant
        before_count = conn.execute(
            "SELECT COUNT(*) FROM highlight_events WHERE match_id = ?", [match_id]
        ).fetchone()[0]

        # Insérer les events (simuler le code de _insert_event_rows)

        inserted = 0
        for row in rows:
            try:
                conn.execute(
                    """INSERT INTO highlight_events (
                        match_id, event_type, time_ms, xuid, gamertag, type_hint, raw_json
                    ) VALUES (?, ?, ?, ?, ?, ?, ?)""",
                    (
                        row.match_id,
                        row.event_type,
                        row.time_ms,
                        row.xuid,
                        row.gamertag,
                        row.type_hint,
                        row.raw_json,
                    ),
                )
                inserted += 1
            except Exception as e:
                results["errors"].append(f"Erreur insertion event {row.event_type}: {e}")

        conn.commit()

        # Compter les events après
        after_count = conn.execute(
            "SELECT COUNT(*) FROM highlight_events WHERE match_id = ?", [match_id]
        ).fetchone()[0]

        results["rows_inserted"] = inserted
        results["rows_in_db"] = after_count - before_count
        results["insert_success"] = len(results["errors"]) == 0

        conn.close()

    except Exception as e:
        results["errors"].append(f"Erreur générale insertion: {e}")
        import traceback

        results["traceback"] = traceback.format_exc()

    return results


async def main() -> int:
    """Point d'entrée principal."""
    parser = argparse.ArgumentParser(
        description="Teste la récupération et l'insertion des highlight_events"
    )
    parser.add_argument(
        "--db-path",
        required=True,
        help="Chemin vers la DB DuckDB v4",
    )
    parser.add_argument(
        "--match-id",
        help="ID du match à tester (si non fourni, prend le dernier match)",
    )
    parser.add_argument(
        "--last-match",
        action="store_true",
        help="Tester le dernier match",
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Afficher les résultats en JSON",
    )

    args = parser.parse_args()

    db_path = Path(args.db_path)
    if not db_path.exists():
        print(f"❌ Erreur: La DB {db_path} n'existe pas", file=sys.stderr)
        return 1

    # Récupérer le match_id
    match_id = None if args.last_match or not args.match_id else args.match_id

    if not match_id:
        try:
            conn = duckdb.connect(str(db_path), read_only=True)
            match_row = conn.execute(
                "SELECT match_id FROM match_stats ORDER BY start_time DESC LIMIT 1"
            ).fetchone()
            conn.close()
            if match_row:
                match_id = match_row[0]
                print(f"📋 Match ID détecté: {match_id}\n")
            else:
                print("❌ Aucun match trouvé dans la base", file=sys.stderr)
                return 1
        except Exception as e:
            print(f"❌ Erreur détection match: {e}", file=sys.stderr)
            return 1

    print("=" * 60)
    print(f"🔍 Test highlight_events pour: {match_id}")
    print("=" * 60 + "\n")

    # Étape 1: Récupération depuis l'API
    print("1️⃣  Récupération depuis l'API SPNKr...")
    fetch_results = await test_highlight_events_fetch(match_id)

    if fetch_results["errors"]:
        print("❌ Erreurs lors de la récupération:")
        for error in fetch_results["errors"]:
            print(f"   - {error}")
        if "traceback" in fetch_results:
            print(f"\n{traceback}")
        return 1

    if not fetch_results["api_available"]:
        print("❌ API non disponible")
        return 1

    if not fetch_results["events_fetched"]:
        print(f"⚠️  Aucun événement récupéré (count: {fetch_results['events_count']})")
        print("   Les highlight_events ne sont peut-être pas encore disponibles pour ce match.")
        return 0

    print(f"✅ {fetch_results['events_count']} événements récupérés")
    if fetch_results["events_sample"]:
        print("   Échantillon:")
        for i, event in enumerate(fetch_results["events_sample"][:3], 1):
            if isinstance(event, dict):
                event_type = event.get("event_type", "?")
                time_ms = event.get("time_ms", "?")
            else:
                event_type = getattr(event, "event_type", "?")
                time_ms = getattr(event, "time_ms", "?")
            print(f"     {i}. {event_type} @ {time_ms}ms")
    print()

    # Étape 2: Transformation (utiliser tous les events récupérés)
    print("2️⃣  Transformation des événements...")
    # Récupérer tous les events pour la transformation
    from src.data.sync.api_client import SPNKrAPIClient, get_tokens_from_env

    tokens = await get_tokens_from_env()
    async with SPNKrAPIClient(tokens=tokens, requests_per_second=5) as client:
        all_events = await client.get_highlight_events(match_id)
        events_for_transform = all_events if all_events else []

    transform_results = test_highlight_events_transform(events_for_transform, match_id)

    if transform_results["errors"]:
        print("❌ Erreurs lors de la transformation:")
        for error in transform_results["errors"]:
            print(f"   - {error}")
        return 1

    print(f"✅ {transform_results['rows_count']} lignes transformées")
    if transform_results["rows_sample"]:
        print("   Échantillon:")
        for i, row in enumerate(transform_results["rows_sample"][:3], 1):
            print(f"     {i}. {row['event_type']} @ {row['time_ms']}ms (xuid: {row['xuid']})")
    print()

    # Étape 3: Insertion
    print("3️⃣  Insertion dans la base de données...")
    from src.data.sync.transformers import transform_highlight_events

    all_rows = transform_highlight_events(events_for_transform, match_id)
    insert_results = test_highlight_events_insert(db_path, match_id, all_rows)

    if insert_results["errors"]:
        print("❌ Erreurs lors de l'insertion:")
        for error in insert_results["errors"]:
            print(f"   - {error}")
        return 1

    print(f"✅ {insert_results['rows_inserted']} lignes insérées")
    print(f"   Total dans la DB pour ce match: {insert_results['rows_in_db']}")
    print()

    print("=" * 60)
    print("✅ Test terminé avec succès")
    print("=" * 60 + "\n")

    return 0


if __name__ == "__main__":
    sys.exit(asyncio.run(main()))
