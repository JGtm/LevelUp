#!/usr/bin/env python3
"""Script pour indexer manuellement les médias et les associer aux matchs.

Usage (nouveau - dossier par joueur):
    python scripts/index_media.py --gamertag PlayerA
    python scripts/index_media.py --db-path data/players/PlayerA/stats.duckdb

Usage (legacy - deux dossiers globaux):
    python scripts/index_media.py --db-path ... --videos-dir <path> --screens-dir <path>
"""

from __future__ import annotations

import argparse
import io
import json
import os
import sys
from pathlib import Path

# Encodage UTF-8 pour la console Windows (évite UnicodeEncodeError sur emojis)
if sys.stdout.encoding and sys.stdout.encoding.lower().startswith("cp"):
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding="utf-8", errors="replace")

ROOT_DIR = Path(__file__).parent.parent
sys.path.insert(0, str(ROOT_DIR))

import duckdb

from src.data.media_indexer import MediaIndexer
from src.utils.paths import PLAYER_DB_FILENAME, PLAYERS_DIR


def _reset_associations(db_path: Path) -> None:
    """Vide la table media_match_associations."""
    with duckdb.connect(str(db_path), read_only=False) as conn:
        conn.execute("DELETE FROM media_match_associations")
        conn.commit()


def load_settings() -> dict:
    """Charge les settings depuis app_settings.json."""
    settings_path = ROOT_DIR / "app_settings.json"
    if not settings_path.exists():
        return {}
    try:
        with open(settings_path, encoding="utf-8") as f:
            return json.load(f)
    except Exception:
        return {}


def main() -> int:
    parser = argparse.ArgumentParser(description="Indexe les médias et les associe aux matchs")
    parser.add_argument(
        "--db-path",
        type=str,
        help="Chemin vers la DB DuckDB",
    )
    parser.add_argument(
        "--gamertag",
        type=str,
        help="Gamertag du joueur (utilise base_dir/gamertag pour le scan)",
    )
    parser.add_argument(
        "--videos-dir",
        type=str,
        help="Dossier des vidéos (legacy)",
    )
    parser.add_argument(
        "--screens-dir",
        type=str,
        help="Dossier des captures (legacy)",
    )
    parser.add_argument(
        "--tolerance",
        type=int,
        default=5,
        help="Tolérance en minutes pour l'association (défaut: 5)",
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="Forcer le re-scan de tous les fichiers",
    )
    parser.add_argument(
        "--reset-assoc",
        action="store_true",
        help="Supprimer toutes les associations existantes avant de les recalculer",
    )
    parser.add_argument(
        "--all",
        action="store_true",
        help="Indexer tous les joueurs ayant base_dir/gamertag",
    )

    args = parser.parse_args()
    settings = load_settings()

    base_dir = str(settings.get("media_captures_base_dir") or "").strip()
    videos_dir = args.videos_dir or settings.get("media_videos_dir") or ""
    screens_dir = args.screens_dir or settings.get("media_screens_dir") or ""
    # Migration: dériver base_dir depuis legacy si vide (parent commun)
    if not base_dir and (videos_dir or screens_dir):
        paths = [Path(p) for p in [videos_dir, screens_dir] if p and Path(p).exists()]
        if paths:
            try:
                common = paths[0]
                for p in paths[1:]:
                    common = Path(os.path.commonpath([str(common), str(p)]))
                base_dir = str(common)
            except (ValueError, TypeError):
                pass

    if args.all and base_dir:
        # Indexer tous les joueurs
        base_path = Path(base_dir)
        if not base_path.exists():
            print(f"❌ Erreur: Base introuvable: {base_dir}")
            return 1
        total = 0
        for player_dir in sorted(PLAYERS_DIR.iterdir(), key=lambda p: p.name):
            if not player_dir.is_dir():
                continue
            db_file = player_dir / PLAYER_DB_FILENAME
            if not db_file.exists():
                continue
            gamertag = player_dir.name
            player_captures = base_path / gamertag
            if not player_captures.exists():
                continue
            indexer = MediaIndexer(db_file)
            result = indexer.scan_and_index(
                player_captures_dir=player_captures,
                force_rescan=args.force,
            )
            if args.reset_assoc:
                _reset_associations(db_file)
            n_assoc = indexer.associate_with_matches(tolerance_minutes=args.tolerance)
            n_thumb, _ = indexer.generate_thumbnails_for_new(
                videos_dir=player_captures,
                screens_dir=player_captures,
            )
            print(
                f"✅ {gamertag}: {result.n_new + result.n_updated} médias, {n_assoc} assoc., {n_thumb} thumbs"
            )
            total += 1
        print(f"\n✅ {total} joueur(s) indexé(s)")
        return 0

    # Résoudre db_path
    db_path = args.db_path
    if args.gamertag:
        db_path = str(PLAYERS_DIR / args.gamertag / PLAYER_DB_FILENAME)
    if not db_path:
        db_path = settings.get("db_path")
    if not db_path and PLAYERS_DIR.exists():
        for pd in PLAYERS_DIR.iterdir():
            if pd.is_dir() and (pd / PLAYER_DB_FILENAME).exists():
                db_path = str(pd / PLAYER_DB_FILENAME)
                print(f"📁 DB auto-détectée: {db_path}")
                break

    if not db_path:
        print("❌ Erreur: Impossible de trouver la DB DuckDB")
        return 1

    db_path_obj = Path(db_path)
    if not db_path_obj.exists():
        print(f"❌ Erreur: DB introuvable: {db_path}")
        return 1

    gamertag = args.gamertag or db_path_obj.parent.name
    player_captures = (Path(base_dir) / gamertag) if base_dir else None

    if base_dir and player_captures and player_captures.exists():
        # Nouvelle logique: dossier par joueur
        print(f"🔍 Indexation médias - {gamertag}")
        print(f"   DB: {db_path}")
        print(f"   Dossier: {player_captures}")
        indexer = MediaIndexer(db_path_obj)
        result = indexer.scan_and_index(
            player_captures_dir=player_captures,
            force_rescan=args.force,
        )
        if args.reset_assoc:
            _reset_associations(db_path_obj)
        n_assoc = indexer.associate_with_matches(tolerance_minutes=args.tolerance)
        n_thumb, _ = indexer.generate_thumbnails_for_new(
            videos_dir=player_captures,
            screens_dir=player_captures,
        )
    else:
        # Legacy
        if not videos_dir and not screens_dir:
            print("❌ Erreur: Aucun dossier média configuré")
            return 1
        videos_path = Path(videos_dir) if videos_dir and Path(videos_dir).exists() else None
        screens_path = Path(screens_dir) if screens_dir and Path(screens_dir).exists() else None
        if not videos_path and not screens_path:
            print("❌ Erreur: Aucun dossier média valide trouvé")
            return 1
        indexer = MediaIndexer(db_path_obj)
        result = indexer.scan_and_index(
            videos_dir=videos_path,
            screens_dir=screens_path,
            force_rescan=args.force,
        )
        if args.reset_assoc:
            _reset_associations(db_path_obj)
        n_assoc = indexer.associate_with_matches(tolerance_minutes=args.tolerance)
        n_thumb, _ = indexer.generate_thumbnails_for_new(
            videos_dir=videos_path,
            screens_dir=screens_path,
        )

    print("✅ Scan terminé:")
    print(
        f"   - {result.n_scanned} scannés, {result.n_new} nouveaux, {result.n_updated} mis à jour"
    )
    print(f"   - {n_assoc} association(s), {n_thumb} thumbnail(s)")

    print("\n✅ Indexation terminée!")
    return 0


if __name__ == "__main__":
    sys.exit(main())
