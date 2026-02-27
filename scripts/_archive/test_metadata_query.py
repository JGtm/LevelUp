"""
Script simplifié pour tester la requête SQL de résolution des métadonnées.
"""

import sys
from pathlib import Path

root = Path(__file__).parent.parent
sys.path.insert(0, str(root))

try:
    import duckdb
except ImportError:
    print("❌ Module duckdb non trouvé. Activez l'environnement virtuel.")
    sys.exit(1)


def test_metadata_query():
    """Teste la requête SQL directement."""
    print("🔍 Test de la requête SQL de résolution des métadonnées")
    print("=" * 60)

    db_path = root / "data" / "players" / "SpartanC" / "stats.duckdb"
    meta_path = root / "data" / "warehouse" / "metadata.duckdb"

    if not db_path.exists():
        print(f"❌ Base de données joueur non trouvée: {db_path}")
        return

    if not meta_path.exists():
        print(f"❌ Base de données metadata non trouvée: {meta_path}")
        return

    print(f"✅ DB joueur: {db_path}")
    print(f"✅ DB metadata: {meta_path}")

    conn = duckdb.connect(str(db_path), read_only=True)

    try:
        # Attacher metadata
        print("\n1. Attachement de metadata.duckdb")
        print("-" * 60)
        try:
            conn.execute(f"ATTACH '{meta_path}' AS meta (READ_ONLY)")
            print("✅ metadata.duckdb attaché")
        except Exception as e:
            print(f"❌ Erreur attachement: {e}")
            return

        # Vérifier les tables
        print("\n2. Tables disponibles dans meta")
        print("-" * 60)
        tables = conn.execute(
            "SELECT table_name FROM information_schema.tables "
            "WHERE table_schema = 'meta' ORDER BY table_name"
        ).fetchall()
        meta_tables = [row[0] for row in tables]
        print(f"Tables: {meta_tables}")

        # Exemple de données dans match_stats
        print("\n3. Exemple de données dans match_stats")
        print("-" * 60)
        sample = conn.execute(
            "SELECT map_id, map_name, playlist_id, playlist_name, pair_id, pair_name "
            "FROM match_stats "
            "LIMIT 3"
        ).fetchall()

        for i, row in enumerate(sample, 1):
            print(f"\nMatch {i}:")
            print(f"  map_id={row[0]}, map_name={row[1]}")
            print(f"  playlist_id={row[2]}, playlist_name={row[3]}")
            print(f"  pair_id={row[4]}, pair_name={row[5]}")

        # Test de jointure
        if "maps" in meta_tables:
            print("\n4. Test de jointure LEFT JOIN avec meta.maps")
            print("-" * 60)
            test_query = """
                SELECT
                    match_stats.map_id,
                    match_stats.map_name as map_name_stored,
                    m_meta.public_name as map_name_resolved,
                    COALESCE(m_meta.public_name, match_stats.map_name) as map_name_final
                FROM match_stats
                LEFT JOIN meta.maps m_meta ON match_stats.map_id = m_meta.asset_id
                WHERE match_stats.map_id IS NOT NULL
                LIMIT 5
            """
            try:
                results = conn.execute(test_query).fetchall()
                print("Résultats:")
                for row in results:
                    map_id, stored, resolved, final = row
                    status = "✅" if resolved and resolved != stored else "⚠️"
                    print(f"\n  {status} map_id={map_id}")
                    print(f"     Stored: {stored}")
                    print(f"     Resolved: {resolved}")
                    print(f"     Final (COALESCE): {final}")
            except Exception as e:
                print(f"❌ Erreur jointure: {e}")
                import traceback

                traceback.print_exc()

        # Test avec playlists
        if "playlists" in meta_tables:
            print("\n5. Test de jointure avec meta.playlists")
            print("-" * 60)
            test_query = """
                SELECT
                    match_stats.playlist_id,
                    match_stats.playlist_name as playlist_name_stored,
                    p_meta.public_name as playlist_name_resolved,
                    COALESCE(p_meta.public_name, match_stats.playlist_name) as playlist_name_final
                FROM match_stats
                LEFT JOIN meta.playlists p_meta ON match_stats.playlist_id = p_meta.asset_id
                WHERE match_stats.playlist_id IS NOT NULL
                LIMIT 5
            """
            try:
                results = conn.execute(test_query).fetchall()
                print("Résultats:")
                for row in results:
                    playlist_id, stored, resolved, final = row
                    status = "✅" if resolved and resolved != stored else "⚠️"
                    print(f"\n  {status} playlist_id={playlist_id}")
                    print(f"     Stored: {stored}")
                    print(f"     Resolved: {resolved}")
                    print(f"     Final (COALESCE): {final}")
            except Exception as e:
                print(f"❌ Erreur jointure: {e}")
                import traceback

                traceback.print_exc()

    finally:
        conn.close()


if __name__ == "__main__":
    test_metadata_query()
