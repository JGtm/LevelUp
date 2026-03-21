"""Migration : calcul initial de ``backfill_bits`` depuis les colonnes existantes.

Ce script calcule et remplit ``match_participants.backfill_bits`` en inspectant
les colonnes NULL/NOT NULL de chaque participant.

Doit être exécuté UNE SEULE FOIS après la migration vers le nouveau système
de bitmask granulaire (v5.2).

Usage :
    python scripts/backfill/migrate_bits.py
    python scripts/backfill/migrate_bits.py --dry-run
    python scripts/backfill/migrate_bits.py --batch-size 5000

Note :
    Avant migration, une sauvegarde est recommandée :
        CREATE TABLE match_participants_backup AS SELECT * FROM match_participants;
"""

from __future__ import annotations

import argparse
import logging
import sys
from pathlib import Path

# Ajouter la racine du projet au PYTHONPATH
_PROJECT_ROOT = Path(__file__).resolve().parents[2]
if str(_PROJECT_ROOT) not in sys.path:
    sys.path.insert(0, str(_PROJECT_ROOT))

import duckdb

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s %(levelname)s %(message)s",
    datefmt="%H:%M:%S",
)
logger = logging.getLogger(__name__)

from src.utils.paths import get_shared_matches_path

SHARED_PATH = get_shared_matches_path()


def _ensure_backfill_bits_column(conn: duckdb.DuckDBPyConnection) -> None:
    """Ajoute backfill_bits si absent."""
    try:
        result = conn.execute(
            "SELECT COUNT(*) FROM information_schema.columns "
            "WHERE table_name = 'match_participants' AND column_name = 'backfill_bits'"
        ).fetchone()
        if result and result[0] == 0:
            conn.execute(
                "ALTER TABLE match_participants ADD COLUMN backfill_bits INTEGER DEFAULT 0"
            )
            logger.info("Colonne backfill_bits ajoutée à match_participants")
        else:
            logger.info("Colonne backfill_bits déjà présente")
    except Exception as e:
        logger.error(f"Impossible d'ajouter backfill_bits: {e}")
        raise


def migrate_backfill_bits(
    conn: duckdb.DuckDBPyConnection,
    *,
    dry_run: bool = False,
    batch_size: int = 10_000,
) -> dict[str, int]:
    """Calcule et remplit ``backfill_bits`` depuis les colonnes existantes.

    Effectue 3 opérations :
    1. Calcul des bits depuis les colonnes NULL/NOT NULL (SQL vectorisé)
    2. Bit MEDALS via sous-requête medals_earned
    3. Bit KILLER_VICTIM via sous-requête killer_victim_pairs

    Args:
        conn: Connexion DuckDB ouverte sur shared_matches.duckdb.
        dry_run: Si True, affiche les stats sans modifier la DB.
        batch_size: Non utilisé (opération en une passe SQL).

    Returns:
        Dict avec les statistiques de migration.
    """
    stats: dict[str, int] = {
        "total_participants": 0,
        "bits_computed": 0,
        "medals_updated": 0,
        "killer_victim_updated": 0,
    }

    # Vérifier le total de participants
    count_row = conn.execute("SELECT COUNT(*) FROM match_participants").fetchone()
    stats["total_participants"] = count_row[0] if count_row else 0
    logger.info(f"Participants à migrer: {stats['total_participants']:,}")

    if dry_run:
        # En mode dry-run : afficher uniquement la distribution actuelle
        _show_current_distribution(conn)
        return stats

    # ── Étape 1 : calcul des bits depuis les colonnes ──────────────────────
    logger.info("Étape 1/3 : calcul des bits depuis les colonnes NULL/NOT NULL...")
    try:
        conn.execute("""
            UPDATE match_participants SET backfill_bits =
                CASE WHEN team_mmr IS NOT NULL THEN 1 ELSE 0 END |
                CASE WHEN enemy_mmr IS NOT NULL THEN 2 ELSE 0 END |
                CASE WHEN kills_expected IS NOT NULL THEN 4 ELSE 0 END |
                CASE WHEN deaths_expected IS NOT NULL THEN 8 ELSE 0 END |
                CASE WHEN assists_expected IS NOT NULL THEN 16 ELSE 0 END |
                CASE WHEN accuracy IS NOT NULL THEN 32 ELSE 0 END |
                CASE WHEN shots_fired IS NOT NULL AND shots_hit IS NOT NULL THEN 64 ELSE 0 END |
                CASE WHEN damage_dealt IS NOT NULL AND damage_taken IS NOT NULL THEN 128 ELSE 0 END |
                CASE WHEN avg_life_seconds IS NOT NULL THEN 256 ELSE 0 END |
                -- bit 9 (512 = MEDALS) géré séparément
                CASE WHEN grenade_kills IS NOT NULL THEN 1024 ELSE 0 END |
                CASE WHEN melee_kills IS NOT NULL THEN 2048 ELSE 0 END |
                CASE WHEN power_weapon_kills IS NOT NULL THEN 4096 ELSE 0 END |
                CASE WHEN personal_score IS NOT NULL THEN 8192 ELSE 0 END |
                CASE WHEN headshot_kills IS NOT NULL THEN 16384 ELSE 0 END |
                CASE WHEN max_killing_spree IS NOT NULL THEN 32768 ELSE 0 END |
                CASE WHEN kda IS NOT NULL THEN 65536 ELSE 0 END |
                CASE WHEN time_played_seconds IS NOT NULL THEN 131072 ELSE 0 END
                -- bit 18 (262144 = KILLER_VICTIM) géré séparément
        """)
        stats["bits_computed"] = stats["total_participants"]
        logger.info(f"  ✅ {stats['bits_computed']:,} participants mis à jour")
    except Exception as e:
        logger.error(f"Étape 1 échouée: {e}")
        raise

    # ── Étape 2 : bit MEDALS depuis medals_earned ──────────────────────────
    logger.info("Étape 2/3 : calcul du bit MEDALS (vérification medals_earned)...")
    try:
        conn.execute("""
            UPDATE match_participants mp
            SET backfill_bits = COALESCE(backfill_bits, 0) | 512
            WHERE EXISTS (
                SELECT 1 FROM medals_earned me
                WHERE me.match_id = mp.match_id AND me.xuid = mp.xuid
            )
        """)
        # DuckDB ne retourne pas le nombre de lignes affectées via execute() directement
        medals_count = conn.execute(
            "SELECT COUNT(*) FROM match_participants WHERE (backfill_bits & 512) = 512"
        ).fetchone()
        stats["medals_updated"] = medals_count[0] if medals_count else 0
        logger.info(f"  ✅ {stats['medals_updated']:,} participants avec médailles marqués")
    except Exception as e:
        logger.warning(f"Étape 2 partiellement échouée (medals_earned peut être vide): {e}")

    # ── Étape 3 : bit KILLER_VICTIM depuis killer_victim_pairs ────────────
    logger.info("Étape 3/3 : calcul du bit KILLER_VICTIM (vérification killer_victim_pairs)...")
    try:
        conn.execute("""
            UPDATE match_participants mp
            SET backfill_bits = COALESCE(backfill_bits, 0) | 262144
            WHERE EXISTS (
                SELECT 1 FROM killer_victim_pairs kvp
                WHERE kvp.match_id = mp.match_id
                  AND (kvp.killer_xuid = mp.xuid OR kvp.victim_xuid = mp.xuid)
            )
        """)
        kv_count = conn.execute(
            "SELECT COUNT(*) FROM match_participants WHERE (backfill_bits & 262144) = 262144"
        ).fetchone()
        stats["killer_victim_updated"] = kv_count[0] if kv_count else 0
        logger.info(
            f"  ✅ {stats['killer_victim_updated']:,} participants avec killer/victim marqués"
        )
    except Exception as e:
        logger.warning(f"Étape 3 partiellement échouée (killer_victim_pairs peut être vide): {e}")

    # ── Index ──────────────────────────────────────────────────────────────
    logger.info("Création de l'index idx_mp_backfill...")
    try:
        conn.execute(
            "CREATE INDEX IF NOT EXISTS idx_mp_backfill ON match_participants(xuid, backfill_bits)"
        )
    except Exception as e:
        logger.debug(f"Index ignoré: {e}")

    conn.commit()

    # ── Rapport final ──────────────────────────────────────────────────────
    _show_current_distribution(conn)

    return stats


def _show_current_distribution(conn: duckdb.DuckDBPyConnection) -> None:
    """Affiche la distribution des bits pour diagnostique."""
    logger.info("\n── Distribution des backfill_bits ─────────────────────────────")
    try:
        bits_labels = [
            (1, "TEAM_MMR"),
            (2, "ENEMY_MMR"),
            (4, "KILLS_EXP"),
            (8, "DEATHS_EXP"),
            (16, "ASSISTS_EXP"),
            (32, "ACCURACY"),
            (64, "SHOTS"),
            (128, "DAMAGE"),
            (256, "AVG_LIFE"),
            (512, "MEDALS"),
            (1024, "GRENADE_KILLS"),
            (2048, "MELEE_KILLS"),
            (4096, "POWER_WEAPON"),
            (8192, "PERSONAL_SCORE"),
            (16384, "HEADSHOT_KILLS"),
            (32768, "MAX_SPREE"),
            (65536, "KDA"),
            (131072, "TIME_PLAYED"),
            (262144, "KILLER_VICTIM"),
        ]

        total = conn.execute("SELECT COUNT(*) FROM match_participants").fetchone()
        total_count = total[0] if total else 0

        for bit_val, bit_name in bits_labels:
            count = conn.execute(
                f"SELECT COUNT(*) FROM match_participants "
                f"WHERE (COALESCE(backfill_bits, 0) & {bit_val}) = {bit_val}"
            ).fetchone()
            n = count[0] if count else 0
            pct = (n / total_count * 100) if total_count > 0 else 0
            logger.info(f"  {bit_name:<20}: {n:>8,} / {total_count:,} ({pct:5.1f}%)")

        null_count = conn.execute(
            "SELECT COUNT(*) FROM match_participants WHERE backfill_bits IS NULL"
        ).fetchone()
        n_null = null_count[0] if null_count else 0
        logger.info(f"  {'(NULL / non migré)':<20}: {n_null:>8,}")
    except Exception as e:
        logger.warning(f"Distribution non disponible: {e}")
    logger.info("─" * 60)


def verify_bitmask_coherence(conn: duckdb.DuckDBPyConnection) -> dict[str, int]:
    """Vérifie la cohérence entre colonnes NOT NULL et bits activés.

    Pour chaque colonne, compte les participants où la colonne est NOT NULL
    mais le bit correspondant n'est PAS activé (données non tracées dans le bitmask).

    Returns:
        Dict {bit_name: count_stale} — nombre de participants avec des bits manquants.
        Un total > 0 signifie que migrate_bits.py doit être relancé (sans --dry-run).
    """
    checks = [
        ("TEAM_MMR", 1, "team_mmr IS NOT NULL"),
        ("ENEMY_MMR", 2, "enemy_mmr IS NOT NULL"),
        ("KILLS_EXP", 4, "kills_expected IS NOT NULL"),
        ("DEATHS_EXP", 8, "deaths_expected IS NOT NULL"),
        ("ASSISTS_EXP", 16, "assists_expected IS NOT NULL"),
        ("ACCURACY", 32, "accuracy IS NOT NULL"),
        ("SHOTS", 64, "shots_fired IS NOT NULL"),
        ("DAMAGE", 128, "damage_dealt IS NOT NULL"),
        ("AVG_LIFE", 256, "avg_life_seconds IS NOT NULL"),
        ("GRENADE_KILLS", 1024, "grenade_kills IS NOT NULL"),
        ("MELEE_KILLS", 2048, "melee_kills IS NOT NULL"),
        ("POWER_WEAPON", 4096, "power_weapon_kills IS NOT NULL"),
        ("PERSONAL_SCORE", 8192, "personal_score IS NOT NULL"),
        ("HEADSHOT_KILLS", 16384, "headshot_kills IS NOT NULL"),
        ("MAX_SPREE", 32768, "max_killing_spree IS NOT NULL"),
        ("KDA", 65536, "kda IS NOT NULL"),
        ("TIME_PLAYED", 131072, "time_played_seconds IS NOT NULL"),
    ]

    result: dict[str, int] = {}
    for bit_name, bit_val, condition in checks:
        try:
            count = conn.execute(
                f"SELECT COUNT(*) FROM match_participants "
                f"WHERE {condition} AND (COALESCE(backfill_bits, 0) & {bit_val}) = 0"
            ).fetchone()
            result[bit_name] = count[0] if count else 0
        except Exception as e:
            logger.debug(f"Vérification {bit_name}: {e}")
            result[bit_name] = -1  # colonne absente ou erreur

    return result


def _print_coherence_report(stale: dict[str, int]) -> bool:
    """Affiche le rapport de cohérence. Retourne True si tout est cohérent."""
    total_stale = sum(v for v in stale.values() if v > 0)
    logger.info("\n── Rapport de cohérence backfill_bits ──────────────────────────")
    for bit_name, count in stale.items():
        if count == -1:
            status = "⚠️  colonne absente"
        elif count == 0:
            status = "✅ cohérent"
        else:
            status = f"❌ {count:,} participants avec bit manquant"
        logger.info(f"  {bit_name:<20}: {status}")
    logger.info("─" * 60)
    if total_stale > 0:
        logger.warning(
            f"  {total_stale:,} incohérences détectées. "
            "Relancer : python scripts/backfill/migrate_bits.py"
        )
        return False
    else:
        logger.info("  ✅ Tous les bits sont cohérents avec les colonnes")
        return True


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Migration backfill_bits — calcul initial depuis les colonnes existantes",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Exemples :
    python scripts/backfill/migrate_bits.py
    python scripts/backfill/migrate_bits.py --dry-run
    python scripts/backfill/migrate_bits.py --verify
    python scripts/backfill/migrate_bits.py --batch-size 5000
        """,
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Affiche les statistiques sans modifier la DB",
    )
    parser.add_argument(
        "--verify",
        action="store_true",
        help=(
            "Vérifie la cohérence entre colonnes NOT NULL et bits activés. "
            "Retourne exit code 1 si des incohérences sont trouvées."
        ),
    )
    parser.add_argument(
        "--batch-size",
        type=int,
        default=10_000,
        help="Taille des batchs (défaut: 10000, non utilisé actuellement)",
    )
    parser.add_argument(
        "--db-path",
        type=str,
        default=None,
        help=f"Chemin vers shared_matches.duckdb (défaut: {SHARED_PATH})",
    )
    args = parser.parse_args()

    db_path = Path(args.db_path) if args.db_path else SHARED_PATH

    if not db_path.exists():
        logger.error(f"DB introuvable: {db_path}")
        sys.exit(1)

    logger.info(f"Connexion à: {db_path}")
    read_only = args.dry_run or args.verify
    conn = duckdb.connect(str(db_path), read_only=read_only)

    try:
        if args.verify:
            stale = verify_bitmask_coherence(conn)
            coherent = _print_coherence_report(stale)
            sys.exit(0 if coherent else 1)

        if not args.dry_run:
            _ensure_backfill_bits_column(conn)

        stats = migrate_backfill_bits(conn, dry_run=args.dry_run, batch_size=args.batch_size)

        if not args.dry_run:
            logger.info("\n── Résumé ──────────────────────────────────────────────────────")
            logger.info(f"  Participants traités : {stats['total_participants']:,}")
            logger.info(f"  Bits calculés        : {stats['bits_computed']:,}")
            logger.info(f"  Avec médailles       : {stats['medals_updated']:,}")
            logger.info(f"  Avec killer/victim   : {stats['killer_victim_updated']:,}")
            logger.info("  ✅ Migration terminée")
        else:
            logger.info("Mode dry-run : aucune modification effectuée")
    finally:
        conn.close()


if __name__ == "__main__":
    main()
