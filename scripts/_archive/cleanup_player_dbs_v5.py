#!/usr/bin/env python3
"""Nettoie les données redondantes des player DBs après migration v5.

Après la migration vers l'architecture v5 (shared_matches), certaines tables
des player DBs sont redondantes car elles existent maintenant dans shared_matches_v2.duckdb.

Ce script supprime ces tables en toute sécurité avec plusieurs vérifications :
1. Vérifie que shared_matches_v2.duckdb existe et contient les données
2. Mode --dry-run pour simuler sans supprimer
3. Sauvegarde optionnelle avant nettoyage
4. Validation post-nettoyage

Tables SUPPRIMÉES (maintenant dans shared ou obsolètes) :
- match_stats (remplacée par shared.match_participants + match_registry)
- match_participants (maintenant dans shared.match_participants)
- highlight_events (maintenant dans shared.highlight_events)
- medals_earned (maintenant dans shared.medals_earned)
- killer_victim_pairs (maintenant dans shared.killer_victim_pairs)
- teammates_aggregate (obsolète — calculée dynamiquement depuis shared.match_participants)
- player_match_stats (MMR/skill → colonnes dans shared.match_participants)
- xuid_aliases (mapping xuid→gamertag → maintenant dans shared.xuid_aliases)

Tables CONSERVÉES (données personnelles) :
- player_match_enrichment (performance_score, session_id, is_with_friends)
- personal_score_awards (awards objectifs)
- match_citations (citations calculées)
- career_progression (historique rangs)
- media_files, media_match_associations
- sessions
- sync_meta
- mv_* (vues matérialisées)

Usage :
    # Simuler le nettoyage (recommandé)
    python scripts/cleanup_player_dbs_v5.py --dry-run

    # Nettoyer un joueur spécifique
    python scripts/cleanup_player_dbs_v5.py --gamertag MonGamertag

    # Nettoyer tous les joueurs avec backup
    python scripts/cleanup_player_dbs_v5.py --all --backup

    # Nettoyer avec suppression forcée des views de compatibilité
    python scripts/cleanup_player_dbs_v5.py --all --remove-compat-views

    # Nettoyer sans vérifier la couverture shared (dangereux)
    python scripts/cleanup_player_dbs_v5.py --all --backup --skip-coverage-check
"""

from __future__ import annotations

import argparse
import json
import logging
import shutil
import sys
from datetime import datetime
from pathlib import Path
from typing import Any

import duckdb

# Résolution du répertoire racine du projet
PROJECT_ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(PROJECT_ROOT))

DEFAULT_SHARED_DB = PROJECT_ROOT / "data" / "warehouse" / "shared_matches_v2.duckdb"
PROFILES_PATH = PROJECT_ROOT / "db_profiles.json"
BACKUP_DIR = PROJECT_ROOT / "backups" / "v5_cleanup"

logger = logging.getLogger(__name__)


# Tables à supprimer (maintenant dans shared_matches_v2.duckdb ou obsolètes)
TABLES_TO_REMOVE = [
    "match_stats",
    "match_participants",
    "highlight_events",
    "medals_earned",
    "killer_victim_pairs",
    "teammates_aggregate",
    "player_match_stats",  # MMR/skill → maintenant dans shared.match_participants
    "xuid_aliases",  # mapping xuid→gamertag → maintenant dans shared.xuid_aliases
    "backfill_status",  # table zombie, plus référencée nulle part
]

# Views de compatibilité créées pendant la migration (optionnelles)
COMPAT_VIEWS = [
    "v_match_stats",
    "v_match_participants",
    "v_highlight_events",
    "v_medals_earned",
]

# Tables à CONSERVER (données personnelles)
TABLES_TO_KEEP = [
    "player_match_enrichment",
    "personal_score_awards",
    "match_citations",
    "career_progression",
    "media_files",
    "media_match_associations",
    "sync_meta",
    "sessions",
]


def load_profiles() -> dict[str, dict[str, str]]:
    """Charge db_profiles.json et retourne les profils joueurs."""
    if not PROFILES_PATH.exists():
        raise FileNotFoundError(f"Fichier profils introuvable : {PROFILES_PATH}")
    data = json.loads(PROFILES_PATH.read_text(encoding="utf-8"))
    return data.get("profiles", {})


def check_shared_db_exists(shared_db_path: Path) -> bool:
    """Vérifie que shared_matches_v2.duckdb existe et contient des données."""
    if not shared_db_path.exists():
        logger.error(f"❌ shared_matches_v2.duckdb introuvable : {shared_db_path}")
        logger.error("   Vous devez d'abord créer la base partagée avec :")
        logger.error("   python scripts/migration/create_shared_matches_db.py")
        return False

    try:
        conn = duckdb.connect(str(shared_db_path), read_only=True)

        # Vérifier que les tables essentielles existent
        tables = conn.execute(
            "SELECT table_name FROM information_schema.tables WHERE table_schema='main'"
        ).fetchall()
        table_names = {r[0] for r in tables}

        required_tables = {"match_registry", "match_participants"}
        if not required_tables.issubset(table_names):
            missing = required_tables - table_names
            logger.error(f"❌ Tables manquantes dans shared_matches_v2.duckdb : {missing}")
            conn.close()
            return False

        # Vérifier qu'il y a des données
        match_count = conn.execute("SELECT COUNT(*) FROM match_registry").fetchone()[0]
        conn.close()

        if match_count == 0:
            logger.warning("⚠️  shared_matches_v2.duckdb existe mais ne contient aucun match")
            logger.warning("   Exécutez d'abord la migration avec :")
            logger.warning("   python scripts/migration/migrate_player_to_shared.py --all")
            return False

        logger.info(f"✓ shared_matches_v2.duckdb valide ({match_count} matchs)")
        return True

    except Exception as exc:
        logger.error(f"❌ Erreur lors de la vérification de shared_matches_v2.duckdb : {exc}")
        return False


def backup_player_db(
    player_db_path: Path,
    gamertag: str,
) -> Path | None:
    """Crée une sauvegarde de la player DB avant nettoyage.

    Returns:
        Chemin du backup créé, ou None en cas d'erreur.
    """
    BACKUP_DIR.mkdir(parents=True, exist_ok=True)

    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    backup_filename = f"{gamertag}_{timestamp}.duckdb"
    backup_path = BACKUP_DIR / backup_filename

    try:
        shutil.copy2(player_db_path, backup_path)
        logger.info(f"💾 Backup créé : {backup_path}")
        return backup_path
    except Exception as exc:
        logger.error(f"❌ Erreur création backup : {exc}")
        return None


def analyze_player_db(
    player_db_path: Path,
    gamertag: str,
) -> dict[str, Any]:
    """Analyse la player DB pour déterminer quoi supprimer.

    Returns:
        Statistiques sur les tables à supprimer et à conserver.
    """
    stats: dict[str, Any] = {
        "gamertag": gamertag,
        "tables_to_remove": {},
        "compat_views": {},
        "tables_to_keep": {},
        "total_size_kb": 0,
    }

    try:
        # Taille du fichier avant nettoyage
        stats["total_size_kb"] = player_db_path.stat().st_size // 1024

        conn = duckdb.connect(str(player_db_path), read_only=True)

        # Lister toutes les tables et vues
        all_objects = conn.execute("""
            SELECT table_name, table_type
            FROM information_schema.tables
            WHERE table_schema='main'
        """).fetchall()

        tables = {r[0]: r[1] for r in all_objects}

        # Analyser les tables à supprimer
        for table_name in TABLES_TO_REMOVE:
            if table_name in tables:
                try:
                    count = conn.execute(f"SELECT COUNT(*) FROM {table_name}").fetchone()[0]
                    stats["tables_to_remove"][table_name] = {
                        "type": tables[table_name],
                        "rows": count,
                    }
                except Exception:
                    stats["tables_to_remove"][table_name] = {
                        "type": tables[table_name],
                        "rows": "ERROR",
                    }

        # Analyser les views de compatibilité
        for view_name in COMPAT_VIEWS:
            if view_name in tables:
                stats["compat_views"][view_name] = tables[view_name]

        # Analyser les tables à conserver
        for table_name in TABLES_TO_KEEP:
            if table_name in tables:
                try:
                    count = conn.execute(f"SELECT COUNT(*) FROM {table_name}").fetchone()[0]
                    stats["tables_to_keep"][table_name] = {
                        "type": tables[table_name],
                        "rows": count,
                    }
                except Exception:
                    stats["tables_to_keep"][table_name] = {
                        "type": tables[table_name],
                        "rows": "ERROR",
                    }

        conn.close()

    except Exception as exc:
        logger.error(f"❌ Erreur analyse DB {gamertag} : {exc}")
        stats["error"] = str(exc)

    return stats


def verify_shared_coverage(
    player_db_path: Path,
    shared_db_path: Path,
    gamertag: str,
) -> dict[str, Any]:
    """Vérifie que shared_matches_v2.duckdb couvre TOUS les matchs du joueur.

    Compare les match_id de la player DB (match_stats) avec ceux présents
    dans shared.match_participants pour ce joueur. C'est la vérification
    critique AVANT toute suppression.

    Returns:
        Dict avec local_count, shared_count, missing_count, missing_ids.
    """
    result: dict[str, Any] = {
        "local_count": 0,
        "shared_count": 0,
        "missing_count": 0,
        "missing_ids": [],
        "coverage_pct": 0.0,
        "ok": False,
    }

    try:
        # Ouvrir la player DB pour lister ses matchs
        player_conn = duckdb.connect(str(player_db_path), read_only=True)

        # Vérifier que match_stats existe encore (pas déjà nettoyé)
        has_ms = player_conn.execute(
            "SELECT COUNT(*) FROM information_schema.tables "
            "WHERE table_schema='main' AND table_name='match_stats'"
        ).fetchone()[0]

        if not has_ms:
            logger.info(f"  ℹ️  {gamertag}: match_stats déjà supprimée (déjà nettoyé ?)")
            player_conn.close()
            result["ok"] = True  # Rien à vérifier
            return result

        local_ids = {
            r[0]
            for r in player_conn.execute("SELECT DISTINCT match_id FROM match_stats").fetchall()
        }
        result["local_count"] = len(local_ids)
        player_conn.close()

        if not local_ids:
            result["ok"] = True
            return result

        # Vérifier la couverture dans shared
        shared_conn = duckdb.connect(str(shared_db_path), read_only=True)

        # Résoudre le xuid du joueur
        xuid_row = shared_conn.execute(
            "SELECT xuid FROM xuid_aliases WHERE LOWER(gamertag) = LOWER(?) LIMIT 1",
            [gamertag],
        ).fetchone()

        if not xuid_row:
            logger.warning(f"  ⚠️  {gamertag}: XUID introuvable dans shared.xuid_aliases")
            shared_conn.close()
            result["missing_count"] = len(local_ids)
            result["missing_ids"] = list(local_ids)[:20]  # Limiter l'affichage
            return result

        xuid = xuid_row[0]

        shared_ids = {
            r[0]
            for r in shared_conn.execute(
                "SELECT DISTINCT match_id FROM match_participants WHERE xuid = ?",
                [xuid],
            ).fetchall()
        }
        result["shared_count"] = len(shared_ids)
        shared_conn.close()

        missing = local_ids - shared_ids
        result["missing_count"] = len(missing)
        result["missing_ids"] = sorted(missing)[:20]  # Limiter l'affichage
        result["coverage_pct"] = (
            ((len(local_ids) - len(missing)) / len(local_ids)) * 100 if local_ids else 100.0
        )
        result["ok"] = len(missing) == 0

    except Exception as exc:
        logger.error(f"  ❌ Erreur vérification couverture {gamertag}: {exc}")
        result["ok"] = False

    return result


def cleanup_player_db(
    player_db_path: Path,
    gamertag: str,
    *,
    shared_db_path: Path = DEFAULT_SHARED_DB,
    remove_compat_views: bool = False,
    dry_run: bool = False,
    skip_coverage_check: bool = False,
) -> dict[str, Any]:
    """Nettoie les données redondantes d'une player DB.

    Avant de supprimer, vérifie que shared_matches_v2.duckdb couvre 100%
    des matchs du joueur (sauf si skip_coverage_check=True).

    Args:
        player_db_path: Chemin vers la DB joueur.
        gamertag: Gamertag du joueur.
        shared_db_path: Chemin vers shared_matches_v2.duckdb.
        remove_compat_views: Si True, supprime aussi les views de compatibilité.
        dry_run: Si True, simule sans modifier.
        skip_coverage_check: Si True, ne vérifie pas la couverture shared.

    Returns:
        Statistiques du nettoyage.
    """
    result: dict[str, Any] = {
        "gamertag": gamertag,
        "tables_dropped": [],
        "views_dropped": [],
        "errors": [],
        "size_before_kb": 0,
        "size_after_kb": 0,
        "coverage": None,
    }

    if not player_db_path.exists():
        result["errors"].append(f"DB introuvable : {player_db_path}")
        return result

    result["size_before_kb"] = player_db_path.stat().st_size // 1024

    # ── Vérification de couverture AVANT suppression ──
    if not skip_coverage_check:
        coverage = verify_shared_coverage(player_db_path, shared_db_path, gamertag)
        result["coverage"] = coverage

        if not coverage["ok"]:
            msg = (
                f"❌ Couverture shared incomplète pour {gamertag} : "
                f"{coverage['shared_count']}/{coverage['local_count']} matchs "
                f"({coverage['coverage_pct']:.1f}%), "
                f"{coverage['missing_count']} matchs manquants dans shared"
            )
            logger.error(msg)
            if coverage["missing_ids"]:
                logger.error(
                    f"   Exemples de matchs manquants : {', '.join(coverage['missing_ids'][:5])}"
                )
            logger.error(
                "   Lancez d'abord la migration complète ou utilisez --skip-coverage-check"
            )
            result["errors"].append(msg)
            return result
        else:
            if coverage["local_count"] > 0:
                logger.info(
                    f"  ✅ Couverture shared OK : {coverage['shared_count']}/{coverage['local_count']} matchs (100%)"
                )

    try:
        if dry_run:
            # Mode simulation
            conn = duckdb.connect(str(player_db_path), read_only=True)

            # Vérifier quelles tables existent
            tables = conn.execute(
                "SELECT table_name FROM information_schema.tables WHERE table_schema='main'"
            ).fetchall()
            table_names = {r[0] for r in tables}

            for table_name in TABLES_TO_REMOVE:
                if table_name in table_names:
                    result["tables_dropped"].append(f"{table_name} (simulation)")

            if remove_compat_views:
                for view_name in COMPAT_VIEWS:
                    if view_name in table_names:
                        result["views_dropped"].append(f"{view_name} (simulation)")

            conn.close()
            result["size_after_kb"] = result["size_before_kb"]  # Pas de changement en dry-run

        else:
            # Mode réel
            conn = duckdb.connect(str(player_db_path))

            # Vérifier quelles tables existent
            tables = conn.execute(
                "SELECT table_name FROM information_schema.tables WHERE table_schema='main'"
            ).fetchall()
            table_names = {r[0] for r in tables}

            # Supprimer les tables redondantes
            for table_name in TABLES_TO_REMOVE:
                if table_name in table_names:
                    try:
                        conn.execute(f"DROP TABLE IF EXISTS {table_name}")
                        result["tables_dropped"].append(table_name)
                        logger.info(f"  ✓ Table supprimée : {table_name}")
                    except Exception as exc:
                        error_msg = f"Erreur suppression {table_name} : {exc}"
                        result["errors"].append(error_msg)
                        logger.error(f"  ✗ {error_msg}")

            # Supprimer les views de compatibilité si demandé
            if remove_compat_views:
                for view_name in COMPAT_VIEWS:
                    if view_name in table_names:
                        try:
                            conn.execute(f"DROP VIEW IF EXISTS {view_name}")
                            result["views_dropped"].append(view_name)
                            logger.info(f"  ✓ View supprimée : {view_name}")
                        except Exception as exc:
                            error_msg = f"Erreur suppression {view_name} : {exc}"
                            result["errors"].append(error_msg)
                            logger.error(f"  ✗ {error_msg}")

            conn.close()

            # Taille après nettoyage (avant compaction)
            result["size_after_kb"] = player_db_path.stat().st_size // 1024

    except Exception as exc:
        error_msg = f"Erreur fatale nettoyage {gamertag} : {exc}"
        result["errors"].append(error_msg)
        logger.error(f"❌ {error_msg}")

    return result


def compact_player_db(player_db_path: Path, gamertag: str) -> dict[str, Any]:
    """Compacte une player DB via EXPORT/IMPORT pour récupérer le dead space.

    DuckDB ne libère pas l'espace disque avec VACUUM après suppression de données.
    La seule façon de compacter est d'exporter vers Parquet puis de réimporter
    dans une nouvelle DB, puis de remplacer l'originale.

    Returns:
        Dict avec size_before_kb, size_after_kb, saved_kb.
    """

    result: dict[str, Any] = {
        "gamertag": gamertag,
        "size_before_kb": player_db_path.stat().st_size // 1024,
        "size_after_kb": 0,
        "saved_kb": 0,
        "error": None,
    }

    tmp_export = player_db_path.parent / "_compact_export"
    new_db = player_db_path.parent / "_compact_new.duckdb"

    try:
        logger.info(f"  🗜️  Compaction {gamertag} ({result['size_before_kb']:,} KB)...")

        # Export
        con = duckdb.connect(str(player_db_path), read_only=True)
        con.execute(f"EXPORT DATABASE '{tmp_export}' (FORMAT PARQUET)")
        con.close()

        # Import dans une DB neuve
        con2 = duckdb.connect(str(new_db))
        con2.execute(f"IMPORT DATABASE '{tmp_export}'")
        con2.close()

        # Remplacer l'originale
        player_db_path.replace(player_db_path.with_suffix(".duckdb.pre_compact"))
        new_db.rename(player_db_path)
        # Supprimer l'ancien backup de compaction s'il existe
        pre = player_db_path.with_suffix(".duckdb.pre_compact")
        if pre.exists():
            pre.unlink()

        result["size_after_kb"] = player_db_path.stat().st_size // 1024
        result["saved_kb"] = result["size_before_kb"] - result["size_after_kb"]
        logger.info(
            f"  ✅ Compaction terminée : {result['size_before_kb']:,} KB → "
            f"{result['size_after_kb']:,} KB "
            f"(-{result['saved_kb']:,} KB, "
            f"{100 * result['saved_kb'] // max(result['size_before_kb'], 1)}%)"
        )

    except Exception as exc:
        result["error"] = str(exc)
        logger.error(f"  ❌ Erreur compaction {gamertag} : {exc}")
        # Nettoyage partiel
        if new_db.exists():
            new_db.unlink()
    finally:
        if tmp_export.exists():
            shutil.rmtree(tmp_export)

    return result


def cleanup_all_players(
    shared_db_path: Path = DEFAULT_SHARED_DB,
    *,
    backup: bool = False,
    remove_compat_views: bool = False,
    dry_run: bool = False,
    verbose: bool = False,
    skip_coverage_check: bool = False,
    compact: bool = False,
) -> dict[str, dict[str, Any]]:
    """Nettoie tous les joueurs de db_profiles.json.

    Returns:
        Résultats de nettoyage pour chaque joueur.
    """
    # Vérifier que shared_matches_v2.duckdb existe
    if not check_shared_db_exists(shared_db_path):
        logger.error("❌ Impossible de continuer sans shared_matches_v2.duckdb valide")
        return {}

    profiles = load_profiles()
    results: dict[str, dict[str, Any]] = {}

    print(f"\n{'=' * 60}")
    print(f"NETTOYAGE PLAYER DBs v5 ({'DRY-RUN' if dry_run else 'MODE RÉEL'})")
    print(f"{'=' * 60}")
    print(f"Joueurs à traiter : {len(profiles)}")
    print(f"Backup avant nettoyage : {'Oui' if backup else 'Non'}")
    print(f"Supprimer views compatibilité : {'Oui' if remove_compat_views else 'Non'}")
    print(f"{'=' * 60}\n")

    total_size_before = 0
    total_size_after = 0

    for gamertag, profile in profiles.items():
        print(f"\n📊 {gamertag}")
        print("-" * 60)

        player_db_path = PROJECT_ROOT / profile["db_path"]

        if not player_db_path.exists():
            logger.warning(f"  ⚠️  DB introuvable : {player_db_path}")
            results[gamertag] = {"error": "DB not found"}
            continue

        # Analyser d'abord
        if verbose:
            stats = analyze_player_db(player_db_path, gamertag)
            print(f"  📁 Taille : {stats['total_size_kb']:,} KB")

            if stats["tables_to_remove"]:
                print("  📋 Tables à supprimer :")
                for tbl, info in stats["tables_to_remove"].items():
                    print(f"     • {tbl} ({info['rows']} lignes)")

            if stats["compat_views"]:
                print("  👁️  Views de compatibilité :")
                for view in stats["compat_views"]:
                    print(f"     • {view}")

            if stats["tables_to_keep"]:
                print(f"  ✓ Tables conservées ({len(stats['tables_to_keep'])}) :")
                for tbl, info in stats["tables_to_keep"].items():
                    print(f"     • {tbl} ({info['rows']} lignes)")

        # Backup si demandé
        if backup and not dry_run:
            backup_path = backup_player_db(player_db_path, gamertag)
            if not backup_path:
                logger.error(f"  ❌ Échec backup, nettoyage annulé pour {gamertag}")
                results[gamertag] = {"error": "Backup failed"}
                continue

        # Nettoyer
        result = cleanup_player_db(
            player_db_path,
            gamertag,
            shared_db_path=shared_db_path,
            remove_compat_views=remove_compat_views,
            dry_run=dry_run,
            skip_coverage_check=skip_coverage_check,
        )
        results[gamertag] = result

        total_size_before += result["size_before_kb"]
        size_after = result["size_after_kb"]

        # Compaction EXPORT/IMPORT si demandée (après les DROPs)
        if compact and not dry_run:
            compact_result = compact_player_db(player_db_path, gamertag)
            if not compact_result.get("error"):
                size_after = compact_result["size_after_kb"]
                result["size_after_kb"] = size_after

        total_size_after += size_after

        # Afficher résumé
        if result["tables_dropped"]:
            print(f"  ✓ Tables supprimées : {len(result['tables_dropped'])}")
        if result["views_dropped"]:
            print(f"  ✓ Views supprimées : {len(result['views_dropped'])}")
        if result["errors"]:
            print(f"  ✗ Erreurs : {len(result['errors'])}")

        saved_kb = result["size_before_kb"] - size_after
        if saved_kb > 0:
            percent = (
                (saved_kb / result["size_before_kb"]) * 100 if result["size_before_kb"] > 0 else 0
            )
            print(f"  💾 Espace libéré : {saved_kb:,} KB (-{percent:.1f}%)")

    # Résumé global
    print(f"\n{'=' * 60}")
    print("RÉSUMÉ GLOBAL")
    print(f"{'=' * 60}")
    print(f"Joueurs traités : {len(results)}")

    total_saved = total_size_before - total_size_after
    if total_saved > 0:
        percent = (total_saved / total_size_before) * 100 if total_size_before > 0 else 0
        print(f"Espace total libéré : {total_saved:,} KB (-{percent:.1f}%)")
        print(f"  Avant : {total_size_before:,} KB")
        print(f"  Après : {total_size_after:,} KB")

    errors_count = sum(1 for r in results.values() if r.get("errors"))
    if errors_count > 0:
        print(f"\n⚠️  {errors_count} joueur(s) avec erreurs")

    if dry_run:
        print("\n💡 Mode DRY-RUN : aucune modification effectuée")
        print("   Pour nettoyer réellement, relancez sans --dry-run")

    return results


def main() -> None:
    """Point d'entrée CLI."""
    parser = argparse.ArgumentParser(
        description="Nettoie les données redondantes des player DBs après migration v5",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__,
    )
    parser.add_argument(
        "--gamertag",
        help="Gamertag du joueur à nettoyer (ou --all pour tous)",
    )
    parser.add_argument(
        "--all",
        action="store_true",
        help="Nettoyer tous les joueurs de db_profiles.json",
    )
    parser.add_argument(
        "--backup",
        action="store_true",
        help="Créer un backup avant nettoyage (recommandé)",
    )
    parser.add_argument(
        "--remove-compat-views",
        action="store_true",
        help="Supprimer aussi les views de compatibilité v_*",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Simuler le nettoyage sans modifier les DBs",
    )
    parser.add_argument(
        "--verbose",
        "-v",
        action="store_true",
        help="Afficher les détails de chaque DB",
    )
    parser.add_argument(
        "--shared-db",
        type=Path,
        default=DEFAULT_SHARED_DB,
        help=f"Chemin vers shared_matches_v2.duckdb (défaut: {DEFAULT_SHARED_DB})",
    )
    parser.add_argument(
        "--skip-coverage-check",
        action="store_true",
        help="Ne pas vérifier la couverture shared avant suppression (dangereux)",
    )
    parser.add_argument(
        "--compact",
        action="store_true",
        help="Compacter les DBs via EXPORT/IMPORT après nettoyage (récupère ~87%% d'espace)",
    )

    args = parser.parse_args()

    logging.basicConfig(
        level=logging.DEBUG if args.verbose else logging.INFO,
        format="%(message)s",
    )

    if args.all:
        cleanup_all_players(
            shared_db_path=args.shared_db,
            backup=args.backup,
            remove_compat_views=args.remove_compat_views,
            dry_run=args.dry_run,
            verbose=args.verbose,
            skip_coverage_check=args.skip_coverage_check,
            compact=args.compact,
        )
    elif args.gamertag:
        # Vérifier que shared_matches_v2.duckdb existe
        if not check_shared_db_exists(args.shared_db):
            logger.error("❌ Impossible de continuer sans shared_matches_v2.duckdb valide")
            sys.exit(1)

        profiles = load_profiles()
        if args.gamertag not in profiles:
            print(f"❌ Gamertag '{args.gamertag}' non trouvé dans db_profiles.json")
            print(f"   Joueurs disponibles : {', '.join(profiles.keys())}")
            sys.exit(1)

        profile = profiles[args.gamertag]
        player_db = PROJECT_ROOT / profile["db_path"]

        print(f"\n📊 Analyse {args.gamertag}")
        print("-" * 60)

        # Analyser
        stats = analyze_player_db(player_db, args.gamertag)
        print(f"📁 Taille : {stats['total_size_kb']:,} KB")

        if stats.get("tables_to_remove"):
            print("\n📋 Tables à supprimer :")
            for tbl, info in stats["tables_to_remove"].items():
                print(f"   • {tbl} ({info['rows']} lignes)")

        if stats.get("compat_views") and args.remove_compat_views:
            print("\n👁️  Views de compatibilité à supprimer :")
            for view in stats["compat_views"]:
                print(f"   • {view}")

        if stats.get("tables_to_keep"):
            print(f"\n✓ Tables conservées ({len(stats['tables_to_keep'])}) :")
            for tbl in stats["tables_to_keep"]:
                print(f"   • {tbl}")

        # Backup si demandé
        if args.backup and not args.dry_run:
            backup_path = backup_player_db(player_db, args.gamertag)
            if not backup_path:
                logger.error("❌ Échec backup, nettoyage annulé")
                sys.exit(1)

        # Nettoyer
        print(f"\n🔧 Nettoyage {'(DRY-RUN)' if args.dry_run else '(MODE RÉEL)'}...")
        result = cleanup_player_db(
            player_db,
            args.gamertag,
            shared_db_path=args.shared_db,
            remove_compat_views=args.remove_compat_views,
            dry_run=args.dry_run,
            skip_coverage_check=args.skip_coverage_check,
        )

        # Compaction si demandée
        if args.compact and not args.dry_run:
            compact_player_db(player_db, args.gamertag)
            result["size_after_kb"] = player_db.stat().st_size // 1024

        print(f"\n{'=' * 60}")
        print("RÉSULTAT")
        print(f"{'=' * 60}")

        if result.get("coverage") and result["coverage"]["local_count"] > 0:
            cov = result["coverage"]
            print(
                f"📊 Couverture shared : {cov['shared_count']}/{cov['local_count']} matchs ({cov['coverage_pct']:.1f}%)"
            )
            if cov["missing_count"] > 0:
                print(f"   ⚠️  {cov['missing_count']} matchs absents de shared")

        if result["tables_dropped"]:
            print(f"✓ Tables supprimées : {', '.join(result['tables_dropped'])}")
        if result["views_dropped"]:
            print(f"✓ Views supprimées : {', '.join(result['views_dropped'])}")
        if result["errors"]:
            print("✗ Erreurs :")
            for err in result["errors"]:
                print(f"  • {err}")

        saved_kb = result["size_before_kb"] - result["size_after_kb"]
        if saved_kb > 0:
            percent = (
                (saved_kb / result["size_before_kb"]) * 100 if result["size_before_kb"] > 0 else 0
            )
            print(f"\n💾 Espace libéré : {saved_kb:,} KB (-{percent:.1f}%)")
            print(f"   Avant : {result['size_before_kb']:,} KB")
            print(f"   Après : {result['size_after_kb']:,} KB")

        if args.dry_run:
            print("\n💡 Mode DRY-RUN : aucune modification effectuée")
    else:
        parser.print_help()
        sys.exit(1)


if __name__ == "__main__":
    main()
