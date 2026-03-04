#!/usr/bin/env python3
"""Script de recalcul des scores de performance v4 pour DuckDB.

Recalcule tous les performance_score dans player_match_enrichment en utilisant la formule v4
(8 métriques : KPM, DPM deaths, APM, KDA, accuracy, PSPM, DPM damage, rank_perf).

Usage:
    # Simulation pour un joueur (affiche les stats sans modifier la DB)
    python scripts/recompute_performance_scores_duckdb.py --player SpartanC --dry-run

    # Recalcul pour un joueur spécifique
    python scripts/recompute_performance_scores_duckdb.py --player SpartanC

    # Recalcul pour tous les joueurs
    python scripts/recompute_performance_scores_duckdb.py --all

    # Forcer le recalcul même pour les matchs qui ont déjà un score
    python scripts/recompute_performance_scores_duckdb.py --player SpartanC --force

    # Spécifier la taille des batches de commit
    python scripts/recompute_performance_scores_duckdb.py --all --batch-size 200
"""

from __future__ import annotations

import argparse
import json
import logging
import sys
from pathlib import Path

import duckdb
import polars as pl

# Ajouter le répertoire racine du projet au path
PROJECT_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(PROJECT_ROOT))

from src.analysis.performance_config import (
    MIN_MATCHES_FOR_RELATIVE,
    PERFORMANCE_SCORE_VERSION,
)
from src.analysis.performance_score import compute_relative_performance_score

logger = logging.getLogger(__name__)

# Colonnes requises pour le calcul v4
HISTORY_COLUMNS = """
    mp.match_id, r.start_time,
    mp.kills, mp.deaths, mp.assists, mp.kda, mp.accuracy,
    mp.time_played_seconds, mp.avg_life_seconds,
    mp.personal_score, mp.damage_dealt,
    mp.rank, mp.team_mmr, mp.enemy_mmr,
    pme.performance_score
"""


def load_player_matches(db_path: Path) -> pl.DataFrame:
    """Charge tous les matchs d'un joueur depuis DuckDB (architecture v5.1).

    Les stats sont dans shared.match_participants, les enrichissements
    (performance_score) dans player_match_enrichment.
    """
    shared_path = db_path.parent.parent.parent / "warehouse" / "shared_matches.duckdb"
    if not shared_path.exists():
        raise FileNotFoundError(f"shared_matches.duckdb introuvable : {shared_path}")

    # Le gamertag est le nom du répertoire parent
    gamertag = db_path.parent.name

    conn = duckdb.connect(str(db_path), read_only=True)
    try:
        conn.execute(f"ATTACH '{shared_path}' AS shared (READ_ONLY)")
        # Résoudre le xuid depuis shared.xuid_aliases (par gamertag, insensible à la casse)
        xuid_row = conn.execute(
            "SELECT xuid FROM shared.xuid_aliases WHERE LOWER(gamertag) = LOWER(?) LIMIT 1",
            [gamertag],
        ).fetchone()
        if not xuid_row:
            # Fallback : chercher parmi les participants pour trouver l'xuid du joueur
            xuid_row = conn.execute(
                """SELECT DISTINCT mp.xuid
                   FROM shared.match_participants mp
                   JOIN player_match_enrichment pme ON mp.match_id = pme.match_id
                   WHERE LOWER(mp.gamertag) = LOWER(?)
                   LIMIT 1""",
                [gamertag],
            ).fetchone()
        if not xuid_row:
            raise ValueError(f"XUID introuvable pour {gamertag}")
        xuid = xuid_row[0]

        df = conn.execute(
            """
            SELECT
                mp.match_id, r.start_time,
                mp.kills, mp.deaths, mp.assists, mp.kda, mp.accuracy,
                mp.time_played_seconds, mp.avg_life_seconds,
                mp.personal_score, mp.damage_dealt,
                mp.rank, mp.team_mmr, mp.enemy_mmr,
                pme.performance_score
            FROM shared.match_participants mp
            JOIN shared.match_registry r ON mp.match_id = r.match_id
            LEFT JOIN player_match_enrichment pme ON mp.match_id = pme.match_id
            WHERE mp.xuid = ?
              AND r.start_time IS NOT NULL
            ORDER BY r.start_time ASC
        """,
            [xuid],
        ).pl()
        return df
    finally:
        conn.close()


def recompute_scores_for_player(
    db_path: Path,
    *,
    dry_run: bool = False,
    force: bool = False,
    batch_size: int = 100,
) -> dict[str, int]:
    """Recalcule les scores de performance v4 pour un joueur.

    Args:
        db_path: Chemin vers stats.duckdb du joueur.
        dry_run: Si True, simule sans écrire.
        force: Si True, recalcule même les scores existants.
        batch_size: Nombre de mises à jour par commit.

    Returns:
        Dict avec les statistiques de traitement.
    """
    stats = {
        "total": 0,
        "computed": 0,
        "skipped": 0,
        "errors": 0,
        "insufficient": 0,
        "sessions_updated": 0,
    }

    # Charger tous les matchs
    df = load_player_matches(db_path)
    if df.is_empty():
        return stats

    stats["total"] = len(df)

    # Ouvrir connexion en écriture si pas dry-run
    conn = None
    if not dry_run:
        conn = duckdb.connect(str(db_path), read_only=False)

    batch_updates: list[tuple[float, str]] = []

    try:
        for idx in range(len(df)):
            row = df.row(idx, named=True)
            match_id = row["match_id"]

            # Skip si score existe déjà et pas force
            if not force and row.get("performance_score") is not None:
                stats["skipped"] += 1
                continue

            # Historique = tous les matchs AVANT celui-ci
            if idx < MIN_MATCHES_FOR_RELATIVE:
                stats["insufficient"] += 1
                continue

            history = df.slice(0, idx)

            # Calculer le score v4
            try:
                score = compute_relative_performance_score(row, history)

                if score is not None:
                    stats["computed"] += 1
                    if not dry_run and conn:
                        batch_updates.append((score, match_id))

                        # Commit par batch
                        if len(batch_updates) >= batch_size:
                            conn.executemany(
                                "UPDATE player_match_enrichment SET performance_score = ? WHERE match_id = ?",
                                batch_updates,
                            )
                            conn.commit()
                            batch_updates = []
                else:
                    stats["insufficient"] += 1
            except Exception as e:
                stats["errors"] += 1
                logger.warning(f"Erreur pour {match_id}: {e}")

        # Commit restant
        if batch_updates and not dry_run and conn:
            conn.executemany(
                "UPDATE player_match_enrichment SET performance_score = ? WHERE match_id = ?",
                batch_updates,
            )
            conn.commit()
    finally:
        if conn:
            conn.close()

    # ── Recalculer sessions.performance_score = AVG(pme.performance_score) ──
    # Fait dans une connexion séparée en écriture car la connexion principale
    # peut être fermée (dry_run = pas de conn).
    if not dry_run:
        conn2 = duckdb.connect(str(db_path), read_only=False)
        try:
            conn2.execute("""
                UPDATE sessions s
                SET performance_score = (
                    SELECT AVG(pme.performance_score)
                    FROM player_match_enrichment pme
                    WHERE pme.session_id = s.session_id
                      AND pme.performance_score IS NOT NULL
                )
                WHERE EXISTS (
                    SELECT 1 FROM player_match_enrichment pme
                    WHERE pme.session_id = s.session_id
                      AND pme.performance_score IS NOT NULL
                )
            """).fetchone()
            conn2.commit()
            n_sessions = conn2.execute(
                "SELECT COUNT(*) FROM sessions WHERE performance_score IS NOT NULL"
            ).fetchone()[0]
            stats["sessions_updated"] = n_sessions
        except Exception as e:
            logger.warning(f"Impossible de mettre à jour sessions.performance_score: {e}")
        finally:
            conn2.close()

    return stats


def find_player_dbs(player: str | None = None) -> list[tuple[str, Path]]:
    """Trouve les DB des joueurs à traiter.

    Args:
        player: Gamertag spécifique ou None pour tous.

    Returns:
        Liste de (gamertag, db_path).
    """
    profiles_path = PROJECT_ROOT / "db_profiles.json"

    if not profiles_path.exists():
        # Fallback : scanner data/players/
        players_dir = PROJECT_ROOT / "data" / "players"
        if not players_dir.exists():
            return []
        results = []
        for p in players_dir.iterdir():
            if p.is_dir():
                db = p / "stats.duckdb"
                if db.exists() and (player is None or p.name == player):
                    results.append((p.name, db))
        return results

    with open(profiles_path) as f:
        profiles = json.load(f)

    results = []
    for gt, info in profiles.get("profiles", {}).items():
        if player is not None and gt != player:
            continue
        db_path = PROJECT_ROOT / info["db_path"]
        if db_path.exists():
            results.append((gt, db_path))

    return results


def main() -> None:
    parser = argparse.ArgumentParser(
        description=f"Recalcul des scores de performance {PERFORMANCE_SCORE_VERSION}"
    )
    parser.add_argument("--player", help="Gamertag spécifique")
    parser.add_argument("--all", action="store_true", help="Tous les joueurs")
    parser.add_argument("--dry-run", action="store_true", help="Simulation (pas d'écriture)")
    parser.add_argument("--force", action="store_true", help="Recalculer même les scores existants")
    parser.add_argument("--batch-size", type=int, default=100, help="Taille des batches de commit")
    args = parser.parse_args()

    if not args.player and not args.all:
        parser.error("Spécifier --player GAMERTAG ou --all")

    logging.basicConfig(level=logging.INFO, format="%(message)s")

    # Trouver les DB à traiter
    player_dbs = find_player_dbs(args.player)

    if not player_dbs:
        print(f"Aucune DB trouvée{f' pour {args.player}' if args.player else ''}")
        sys.exit(1)

    mode = "DRY-RUN" if args.dry_run else "RECALCUL"
    print(f"\n=== {mode} Performance Score {PERFORMANCE_SCORE_VERSION} ===")
    print(f"Joueurs : {len(player_dbs)}")
    if args.force:
        print("Mode : FORCE (recalcul de tous les scores)")
    print()

    total_stats = {
        "total": 0,
        "computed": 0,
        "skipped": 0,
        "errors": 0,
        "insufficient": 0,
        "sessions_updated": 0,
    }

    for gamertag, db_path in player_dbs:
        print(f"  {gamertag} ({db_path.name})... ", end="", flush=True)

        stats = recompute_scores_for_player(
            db_path,
            dry_run=args.dry_run,
            force=args.force,
            batch_size=args.batch_size,
        )

        print(
            f"{stats['total']} matchs, "
            f"{stats['computed']} calculés, "
            f"{stats['skipped']} skippés, "
            f"{stats['insufficient']} insuffisants, "
            f"{stats['errors']} erreurs, "
            f"{stats['sessions_updated']} sessions mises à jour"
        )

        for k in total_stats:
            total_stats[k] += stats[k]

    print("\n=== Résumé ===")
    print(f"Total matchs : {total_stats['total']}")
    print(f"Scores calculés : {total_stats['computed']}")
    print(f"Skippés (déjà présents) : {total_stats['skipped']}")
    print(f"Historique insuffisant : {total_stats['insufficient']}")
    print(f"Erreurs : {total_stats['errors']}")
    print(f"Sessions mises à jour : {total_stats['sessions_updated']}")

    if args.dry_run:
        print("\n(Mode dry-run — aucune modification effectuée)")


if __name__ == "__main__":
    main()
