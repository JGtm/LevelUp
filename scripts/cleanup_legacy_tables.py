#!/usr/bin/env python3
"""Supprime les tables legacy obsolètes des bases joueurs.

V5.1: Les données sont centralisées dans shared_matches.duckdb.
Les tables locales redondantes peuvent être supprimées pour réduire
la taille des DBs joueurs (~30MB → ~4MB par joueur).

Tables supprimées (9):
- match_stats : Remplacée par shared.match_participants
- medals_earned : Remplacée par shared.medals_earned
- highlight_events : Remplacée par shared.highlight_events
- player_stats : Obsolète (agrégats calculés à la volée)
- xuid_aliases : Remplacée par shared.xuid_aliases (13 955 rows centralisées)
- mv_match_stats_with_context : Vue obsolète
- mv_recent_matches : Vue obsolète
- mv_team_stats : Vue obsolète
- mv_opponent_stats : Vue obsolète

Tables conservées (8):
- player_match_enrichment : Enrichissements spécifiques joueur
- teammates_aggregate : Agrégats coéquipiers
- match_citations : Citations calculées
- career_progression : Historique rangs
- sync_meta : Métadonnées de synchronisation
- player_match_stats : Stats du joueur principal (v3/v4)
- toutes les vues mv_* nouvelles

Usage:
    # Dry-run (affiche sans modifier)
    python scripts/cleanup_legacy_tables.py --gamertag MonGamertag --dry-run

    # Cleanup pour un joueur
    python scripts/cleanup_legacy_tables.py --gamertag MonGamertag

    # Cleanup pour tous les joueurs
    python scripts/cleanup_legacy_tables.py --all

    # Avec backup automatique
    python scripts/cleanup_legacy_tables.py --all --backup
"""

from __future__ import annotations

import argparse
import logging
import shutil
import sys
from datetime import datetime
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

try:
    import duckdb
except ImportError:
    print("Error: duckdb not installed")
    sys.exit(1)

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger(__name__)

# Paths
DATA_DIR = Path(__file__).resolve().parent.parent / "data"
PLAYERS_DIR = DATA_DIR / "players"
BACKUP_DIR = Path(__file__).resolve().parent.parent / "backups" / "pre_cleanup"

# Tables à supprimer (redondantes avec shared_matches.duckdb)
TABLES_TO_DROP = [
    "match_stats",
    "medals_earned",
    "highlight_events",
    "player_stats",
    "xuid_aliases",  # Remplacée par shared.xuid_aliases
    "mv_match_stats_with_context",
    "mv_recent_matches",
    "mv_team_stats",
    "mv_opponent_stats",
]


def get_player_db_path(gamertag: str) -> Path:
    """Retourne le chemin vers la DB d'un joueur."""
    return PLAYERS_DIR / gamertag / "stats.duckdb"


def get_table_sizes(db_path: Path) -> dict[str, tuple[int, int]]:
    """Retourne le nombre de rows et taille estimée par table.

    Returns:
        Dict {table_name: (row_count, estimated_size_bytes)}
    """
    if not db_path.exists():
        return {}

    result: dict[str, tuple[int, int]] = {}
    try:
        conn = duckdb.connect(str(db_path), read_only=True)
        tables = conn.execute(
            "SELECT table_name FROM information_schema.tables WHERE table_schema = 'main'"
        ).fetchall()

        for (table_name,) in tables:
            try:
                count = conn.execute(f'SELECT COUNT(*) FROM "{table_name}"').fetchone()[0]
                # Estimation grossière : 100 bytes par row
                result[table_name] = (count, count * 100)
            except Exception:
                result[table_name] = (0, 0)

        conn.close()
    except Exception as e:
        logger.debug(f"Erreur lecture tables: {e}")

    return result


def backup_player_db(gamertag: str, backup_dir: Path | None = None) -> Path | None:
    """Crée une copie de backup de la DB joueur.

    Returns:
        Chemin du backup créé, ou None en cas d'erreur.
    """
    db_path = get_player_db_path(gamertag)
    if not db_path.exists():
        return None

    if backup_dir is None:
        backup_dir = BACKUP_DIR

    backup_dir.mkdir(parents=True, exist_ok=True)
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    backup_path = backup_dir / f"{gamertag}_{timestamp}.duckdb"

    try:
        shutil.copy2(db_path, backup_path)
        logger.info(f"  Backup créé: {backup_path}")
        return backup_path
    except Exception as e:
        logger.error(f"  Erreur backup: {e}")
        return None


def cleanup_player_db(gamertag: str, dry_run: bool = True, do_backup: bool = False) -> dict:
    """Supprime les tables legacy d'une DB joueur.

    Args:
        gamertag: Nom du joueur.
        dry_run: Si True, affiche sans modifier.
        do_backup: Si True, crée un backup avant modification.

    Returns:
        Dict avec statistiques de l'opération.
    """
    result = {
        "gamertag": gamertag,
        "tables_dropped": [],
        "tables_not_found": [],
        "rows_freed": 0,
        "size_before_mb": 0.0,
        "size_after_mb": 0.0,
        "errors": [],
        "backup_path": None,
    }

    db_path = get_player_db_path(gamertag)
    if not db_path.exists():
        result["errors"].append(f"DB non trouvée: {db_path}")
        return result

    # Taille avant
    result["size_before_mb"] = db_path.stat().st_size / (1024 * 1024)

    # Analyse des tables
    table_sizes = get_table_sizes(db_path)

    # Tables à supprimer présentes
    tables_present = [t for t in TABLES_TO_DROP if t in table_sizes]
    tables_missing = [t for t in TABLES_TO_DROP if t not in table_sizes]

    result["tables_not_found"] = tables_missing

    if dry_run:
        for table in tables_present:
            rows, _ = table_sizes.get(table, (0, 0))
            result["tables_dropped"].append(table)
            result["rows_freed"] += rows
            logger.info(f"  [DRY-RUN] Supprimerait: {table} ({rows:,} rows)")

        if tables_missing:
            logger.debug(f"  Tables non trouvées: {', '.join(tables_missing)}")

        return result

    # Backup si demandé
    if do_backup:
        result["backup_path"] = backup_player_db(gamertag)
        if not result["backup_path"]:
            result["errors"].append("Échec backup - opération annulée")
            return result

    # Suppression effective
    try:
        conn = duckdb.connect(str(db_path))

        for table in tables_present:
            rows, _ = table_sizes.get(table, (0, 0))
            try:
                conn.execute(f'DROP TABLE IF EXISTS "{table}"')
                result["tables_dropped"].append(table)
                result["rows_freed"] += rows
                logger.info(f"  ✅ Supprimé: {table} ({rows:,} rows)")
            except Exception as e:
                logger.error(f"  ❌ Erreur {table}: {e}")
                result["errors"].append(f"{table}: {e}")

        # VACUUM pour libérer l'espace (DuckDB le fait automatiquement)
        # Note: DuckDB ne supporte pas VACUUM comme SQLite
        conn.close()

        # Taille après
        result["size_after_mb"] = db_path.stat().st_size / (1024 * 1024)

    except Exception as e:
        result["errors"].append(str(e))
        logger.error(f"  Erreur globale: {e}")

    return result


def find_all_players() -> list[str]:
    """Trouve tous les joueurs avec une DB DuckDB."""
    players = []
    if not PLAYERS_DIR.exists():
        return players

    for player_dir in PLAYERS_DIR.iterdir():
        if player_dir.is_dir() and (player_dir / "stats.duckdb").exists():
            players.append(player_dir.name)

    return sorted(players)


def main():
    parser = argparse.ArgumentParser(
        description="Supprime les tables legacy obsolètes des bases joueurs"
    )
    parser.add_argument("--gamertag", "-g", help="Gamertag du joueur")
    parser.add_argument("--all", "-a", action="store_true", help="Tous les joueurs")
    parser.add_argument("--dry-run", "-n", action="store_true", help="Mode simulation")
    parser.add_argument("--backup", "-b", action="store_true", help="Créer un backup avant")
    parser.add_argument("--list-tables", action="store_true", help="Liste les tables à supprimer")

    args = parser.parse_args()

    # Mode liste des tables
    if args.list_tables:
        print("\nTables qui seront supprimées:")
        print("=" * 40)
        for table in TABLES_TO_DROP:
            print(f"  - {table}")
        print("\nCes tables sont redondantes avec shared_matches.duckdb")
        return

    if not args.gamertag and not args.all:
        parser.error("Spécifiez --gamertag ou --all (ou --list-tables)")

    # Joueurs à traiter
    if args.all:
        players = find_all_players()
        if not players:
            logger.error("Aucun joueur trouvé")
            return
        logger.info(f"Trouvé {len(players)} joueur(s)")
    else:
        players = [args.gamertag]

    # Stats globales
    total_tables_dropped = 0
    total_rows_freed = 0
    total_size_before = 0.0
    total_size_after = 0.0
    errors_count = 0

    prefix = "[DRY-RUN] " if args.dry_run else ""

    # Traiter chaque joueur
    for i, player in enumerate(players, 1):
        logger.info("")
        logger.info("=" * 60)
        logger.info(f"{prefix}[{i}/{len(players)}] {player}")
        logger.info("=" * 60)

        result = cleanup_player_db(
            player,
            dry_run=args.dry_run,
            do_backup=args.backup,
        )

        total_tables_dropped += len(result["tables_dropped"])
        total_rows_freed += result["rows_freed"]
        total_size_before += result["size_before_mb"]
        total_size_after += result["size_after_mb"]
        errors_count += len(result["errors"])

        # Afficher résumé joueur
        if result["tables_dropped"]:
            reduction = 0.0
            if result["size_before_mb"] > 0 and not args.dry_run:
                reduction = (
                    (result["size_before_mb"] - result["size_after_mb"])
                    / result["size_before_mb"]
                    * 100
                )
            if args.dry_run:
                logger.info(f"  {prefix}Tables à supprimer: {len(result['tables_dropped'])}")
            else:
                logger.info(
                    f"  Taille: {result['size_before_mb']:.1f}MB → "
                    f"{result['size_after_mb']:.1f}MB (-{reduction:.0f}%)"
                )

    # Résumé global
    logger.info("")
    logger.info("=" * 60)
    logger.info(f"{prefix}RÉSUMÉ")
    logger.info("=" * 60)
    logger.info(f"{prefix}Joueurs traités: {len(players)}")
    logger.info(f"{prefix}Tables supprimées: {total_tables_dropped}")
    logger.info(f"{prefix}Rows libérées: {total_rows_freed:,}")

    if not args.dry_run and total_size_before > 0:
        reduction = ((total_size_before - total_size_after) / total_size_before) * 100
        logger.info(f"Taille totale: {total_size_before:.1f}MB → {total_size_after:.1f}MB")
        logger.info(f"Réduction: -{reduction:.0f}%")

    if errors_count > 0:
        logger.warning(f"Erreurs: {errors_count}")


if __name__ == "__main__":
    main()
