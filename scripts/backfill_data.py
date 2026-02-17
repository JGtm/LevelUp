#!/usr/bin/env python3
"""Script de backfill pour remplir les données manquantes.

Ce script identifie les matchs existants qui ont des données manquantes
(medals, highlight_events, skill stats, personal_scores, performance_scores)
et les remplit en re-téléchargeant les données nécessaires depuis l'API SPNKr.

Usage:
    # Backfill toutes les données pour un joueur
    python scripts/backfill_data.py --player JGtm --all-data

    # Mode strict (pas de re-téléchargement si partiellement rempli)
    python scripts/backfill_data.py --player JGtm --all-data --detection-mode and

    # Backfill uniquement les médailles
    python scripts/backfill_data.py --player JGtm --medals

    # Calculer les scores de performance manquants
    python scripts/backfill_data.py --player JGtm --performance-scores

    # Backfill pour tous les joueurs
    python scripts/backfill_data.py --all --all-data

    # Mode dry-run (liste seulement)
    python scripts/backfill_data.py --player JGtm --dry-run

    # Limiter le nombre de matchs
    python scripts/backfill_data.py --player JGtm --max-matches 100

Note: Pour combiner sync + backfill en une seule commande, utilisez :
    python scripts/sync.py --delta --player JGtm --with-backfill

Architecture (Sprint 10B) :
    scripts/backfill/
    ├── __init__.py
    ├── core.py          — Fonctions d'insertion de base
    ├── detection.py     — Détection des matchs manquants (AND/OR configurable)
    ├── strategies.py    — Stratégies spécifiques (killer/victim, end_time, perf_score)
    ├── orchestrator.py  — Orchestration du backfill
    └── cli.py           — Parsing des arguments CLI
"""

from __future__ import annotations

import asyncio
import logging
import sys
from pathlib import Path

# Ajouter le répertoire parent au path pour les imports
REPO_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(REPO_ROOT))

# ─────────────────────────────────────────────────────────────────────────────
# Rétro-compatibilité : les imports existants restent fonctionnels
#   from scripts.backfill_data import backfill_player_data
#   from scripts.backfill_data import backfill_all_players
#   from scripts.backfill_data import _find_matches_missing_data
#   etc.
# ─────────────────────────────────────────────────────────────────────────────
from scripts.backfill.cli import create_argument_parser  # noqa: E402
from scripts.backfill.orchestrator import (  # noqa: E402
    backfill_all_players,
    backfill_player_data,
)
from src.data.sync.scope import SyncScope  # noqa: E402

# Configuration du logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger(__name__)


def main() -> int:
    """Point d'entrée principal."""
    parser = create_argument_parser()
    args = parser.parse_args()

    # Validation
    if not args.all and not args.player:
        parser.error("--player ou --all est requis")

    # Construire le scope depuis les arguments CLI
    scope = SyncScope.from_cli_args(args)

    try:
        if args.all:
            result = asyncio.run(backfill_all_players(scope=scope))
            _print_summary_all(result, scope)
        else:
            result = asyncio.run(backfill_player_data(args.player, scope=scope))
            _print_summary_player(result, scope)

        return 0

    except KeyboardInterrupt:
        logger.info("\nInterrompu par l'utilisateur")
        return 1
    except Exception as e:
        logger.error(f"Erreur fatale: {e}")
        import traceback

        traceback.print_exc()
        return 1


def _print_summary_all(result: dict, scope: object) -> None:
    """Affiche le résumé global pour tous les joueurs."""
    logger.info("\n" + "=" * 60)
    logger.info("=== RÉSUMÉ GLOBAL ===")
    logger.info("=" * 60)
    logger.info(f"Joueurs traités: {result['players_processed']}")
    totals = result["total_results"]
    _print_totals(totals, scope)


def _print_summary_player(result: dict, scope: object) -> None:
    """Affiche le résumé pour un joueur."""
    logger.info("\n=== Résumé ===")
    _print_totals(result, scope)


def _print_totals(totals: dict, scope: object) -> None:
    """Affiche les totaux du backfill."""
    logger.info(f"Matchs vérifiés: {totals.get('matches_checked', 0)}")
    logger.info(f"Matchs avec données manquantes: {totals.get('matches_missing_data', 0)}")
    logger.info(f"Médailles insérées: {totals.get('medals_inserted', 0)}")
    logger.info(f"Events insérés: {totals.get('events_inserted', 0)}")
    logger.info(f"Skill inséré: {totals.get('skill_inserted', 0)}")
    logger.info(f"Personal scores insérés: {totals.get('personal_scores_inserted', 0)}")
    logger.info(f"Scores de performance calculés: {totals.get('performance_scores_inserted', 0)}")
    logger.info(f"Aliases insérés: {totals.get('aliases_inserted', 0)}")

    if getattr(scope, "accuracy", False):
        logger.info(f"Accuracy mis à jour: {totals.get('accuracy_updated', 0)}")
    if getattr(scope, "shots", False):
        logger.info(f"Shots mis à jour: {totals.get('shots_updated', 0)}")
    if getattr(scope, "enemy_mmr", False):
        logger.info(f"Enemy MMR mis à jour: {totals.get('enemy_mmr_updated', 0)}")
    if getattr(scope, "assets", False):
        logger.info(f"Noms assets mis à jour: {totals.get('assets_updated', 0)}")
    if getattr(scope, "participants", False):
        logger.info(f"Participants insérés: {totals.get('participants_inserted', 0)}")
    if getattr(scope, "participants_scores", False):
        logger.info(f"Scores/rang participants: {totals.get('participants_scores_updated', 0)}")
    if getattr(scope, "participants_kda", False):
        logger.info(f"K/D/A participants: {totals.get('participants_kda_updated', 0)}")
    if getattr(scope, "participants_shots", False):
        logger.info(f"Shots participants: {totals.get('participants_shots_updated', 0)}")
    if getattr(scope, "participants_damage", False):
        logger.info(f"Damage participants: {totals.get('participants_damage_updated', 0)}")
    if getattr(scope, "participants_avg_life", False):
        logger.info(f"Durée de vie participants: {totals.get('participants_avg_life_updated', 0)}")
    if getattr(scope, "killer_victim", False):
        logger.info(f"Paires killer/victim: {totals.get('killer_victim_pairs_inserted', 0)}")
    if getattr(scope, "end_time", False):
        logger.info(f"End time mis à jour: {totals.get('end_time_updated', 0)}")
    if getattr(scope, "sessions", False):
        logger.info(f"Sessions mises à jour: {totals.get('sessions_updated', 0)}")
    if getattr(scope, "citations", False):
        logger.info(f"Citations calculées: {totals.get('citations_computed', 0)}")
    if getattr(scope, "participants_enrich", False):
        logger.info(f"Participants enrichis: {totals.get('participants_enriched', 0)}")


if __name__ == "__main__":
    sys.exit(main())
