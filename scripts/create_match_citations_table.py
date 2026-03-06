#!/usr/bin/env python3
"""
Crée la table match_citations dans chaque DB joueur.

Table sparse stockant les citations calculées par match.
Seules les citations avec value > 0 sont insérées.
"""

import json
import sys
from pathlib import Path

import duckdb


def create_match_citations_table(conn: duckdb.DuckDBPyConnection) -> bool:
    """Crée la table match_citations si elle n'existe pas.

    Args:
        conn: Connexion DuckDB ouverte.

    Returns:
        True si la table a été créée, False si elle existait déjà.
    """
    # Vérifier si la table existe
    exists = conn.execute(
        "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'match_citations'"
    ).fetchone()[0]

    if exists:
        return False

    conn.execute("""
        CREATE TABLE IF NOT EXISTS match_citations (
            match_id TEXT NOT NULL,
            citation_name_norm TEXT NOT NULL,
            value INTEGER NOT NULL,
            PRIMARY KEY (match_id, citation_name_norm)
        )
    """)

    conn.execute("""
        CREATE INDEX IF NOT EXISTS idx_match_citations_name
        ON match_citations(citation_name_norm)
    """)

    return True


def main() -> int:
    """Point d'entrée principal."""
    print("=" * 80)
    print("🏗️  Création table match_citations dans chaque DB joueur")
    print("=" * 80)
    print()

    # Charger les profils
    profiles_path = Path("db_profiles.json")
    if not profiles_path.exists():
        print(f"❌ Fichier non trouvé : {profiles_path}")
        return 1

    with open(profiles_path) as f:
        config = json.load(f)

    profiles = config.get("profiles", {})
    if not profiles:
        print("❌ Aucun profil trouvé dans db_profiles.json")
        return 1

    print(f"📂 {len(profiles)} joueurs trouvés\n")

    created = 0
    skipped = 0
    errors = 0

    for gamertag, profile in profiles.items():
        db_path = Path(profile["db_path"])
        print(f"  🎮 {gamertag} — {db_path}")

        if not db_path.exists():
            print("     ⚠️  DB non trouvée, skip")
            skipped += 1
            continue

        try:
            conn = duckdb.connect(str(db_path))
            was_created = create_match_citations_table(conn)
            conn.close()

            if was_created:
                print("     ✅ Table créée")
                created += 1
            else:
                print("     ℹ️  Table existait déjà")
                skipped += 1
        except Exception as e:
            print(f"     ❌ Erreur : {e}")
            errors += 1

    print()
    print("=" * 80)
    print(f"✅ Terminé — {created} créées, {skipped} existantes, {errors} erreurs")
    print("=" * 80)

    return 1 if errors > 0 else 0


if __name__ == "__main__":
    sys.exit(main())
