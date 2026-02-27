#!/usr/bin/env python3
"""Script de backfill pour remplir les données manquantes.

Ce script identifie les matchs existants qui ont des données manquantes
(medals, highlight_events, skill stats, personal_scores, performance_scores)
et les remplit en re-téléchargeant les données nécessaires depuis l'API SPNKr.

Usage:
    # Backfill toutes les données pour un joueur
    python scripts/backfill_data.py --player SpartanC --all-data

    # Mode strict (pas de re-téléchargement si partiellement rempli)
    python scripts/backfill_data.py --player SpartanC --all-data --detection-mode and

    # Backfill uniquement les médailles
    python scripts/backfill_data.py --player SpartanC --medals

    # Calculer les scores de performance manquants
    python scripts/backfill_data.py --player SpartanC --performance-scores

    # Backfill pour tous les joueurs
    python scripts/backfill_data.py --all --all-data

    # Mode dry-run (liste seulement)
    python scripts/backfill_data.py --player SpartanC --dry-run

    # Limiter le nombre de matchs
    python scripts/backfill_data.py --player SpartanC --max-matches 100

Note: Pour combiner sync + backfill en une seule commande, utilisez :
    python scripts/sync.py --delta --player SpartanC --with-backfill

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
from datetime import datetime, timezone
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
import duckdb  # noqa: E402

from scripts.backfill.cli import create_argument_parser  # noqa: E402
from scripts.backfill.orchestrator import (  # noqa: E402
    backfill_all_players,
    backfill_player_data,
)
from src.data.sync.scope import SyncScope  # noqa: E402

_SHARED_DB = REPO_ROOT / "data" / "warehouse" / "shared_matches.duckdb"


def _open_shared_conn() -> duckdb.DuckDBPyConnection:
    return duckdb.connect(str(_SHARED_DB))


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
    parser.add_argument(
        "--no-discord",
        action="store_true",
        default=False,
        help="Désactive la notification Discord pour cette exécution",
    )
    args = parser.parse_args()

    # --team-scores est global (pas besoin de --player / --all)
    team_scores = getattr(args, "team_scores", False)
    force_team_scores = getattr(args, "force_team_scores", False)
    if team_scores:
        try:
            from scripts.backfill.strategies import backfill_team_scores

            max_matches = getattr(args, "max_matches", None)
            n = asyncio.run(
                backfill_team_scores(
                    _open_shared_conn(),
                    max_matches=max_matches,
                    force=force_team_scores,
                )
            )
            logger.info(f"Team scores : {n} match(s) mis à jour")
        except Exception as e:
            logger.error(f"Erreur --team-scores : {e}")
            import traceback

            traceback.print_exc()
        return 0

    # --mode-category est global (local, sans API)
    mode_category = getattr(args, "mode_category", False)
    force_mode_category = getattr(args, "force_mode_category", False)
    if mode_category or force_mode_category:
        try:
            from scripts.backfill.strategies import backfill_mode_category

            n = backfill_mode_category(_open_shared_conn(), force=force_mode_category)
            logger.info(f"mode_category : {n} match(s) mis à jour")
        except Exception as e:
            logger.error(f"Erreur --mode-category : {e}")
            import traceback

            traceback.print_exc()
        return 0

    # --cleanup-player-dbs est global (local, sans API)
    cleanup_player_dbs = getattr(args, "cleanup_player_dbs", False)
    if cleanup_player_dbs:
        try:
            from scripts.backfill.strategies import cleanup_player_dbs_legacy

            results = cleanup_player_dbs_legacy()
            total = sum(v for v in results.values() if v > 0)
            logger.info(f"Nettoyage DBs joueurs : {total} objet(s) supprimé(s) au total")
            for gt, ops in results.items():
                if ops > 0:
                    logger.info(f"  {gt}: {ops} objet(s)")
        except Exception as e:
            logger.error(f"Erreur --cleanup-player-dbs : {e}")
            import traceback

            traceback.print_exc()
        return 0

    # Validation
    if not args.all and not args.player:
        parser.error("--player ou --all est requis")

    # Construire le scope depuis les arguments CLI
    scope = SyncScope.from_cli_args(args)
    backfill_started_at = datetime.now(timezone.utc)

    try:
        if args.all:
            result = asyncio.run(backfill_all_players(scope=scope))
            _print_summary_all(result, scope)
            # ── Notification Discord (tous joueurs) ────────────────────
            try:
                import json as _json

                from src.ui.multiplayer import list_duckdb_v4_players
                from src.utils.discord_notifier import (
                    DiscordPlayerResult,
                    count_matches_missing_data,
                    fetch_last_match_info,
                    notify_operation_done,
                )

                _profiles_path = REPO_ROOT / "db_profiles.json"
                _xuid_map: dict[str, str] = {}
                if _profiles_path.exists():
                    _pdata = _json.loads(_profiles_path.read_text(encoding="utf-8"))
                    for _k, _v in _pdata.get("profiles", {}).items():
                        if isinstance(_v, dict) and _v.get("xuid"):
                            _xuid_map[_k.lower()] = str(_v["xuid"])

                _totals = result.get("total_results", {})
                _n_players = result.get("players_processed", 0)
                _all_players_list = list_duckdb_v4_players()
                _matches_checked_total = _totals.get("matches_checked", 0)
                _n_ref = max(1, _n_players or len(_all_players_list))
                _discord_players = []
                for _pinfo in _all_players_list:
                    _xuid_bf = _xuid_map.get(_pinfo.gamertag.lower())
                    _missing_bf = count_matches_missing_data(_xuid_bf or "") if _xuid_bf else 0
                    _last_bf = fetch_last_match_info(_xuid_bf or "") if _xuid_bf else None
                    _discord_players.append(
                        DiscordPlayerResult(
                            gamertag=_pinfo.gamertag,
                            xuid=_xuid_bf,
                            matches_synced=_matches_checked_total // _n_ref,
                            missing_data_count=_missing_bf,
                            last_match=_last_bf,
                        )
                    )
                notify_operation_done(
                    operation="backfill",
                    started_at=backfill_started_at,
                    finished_at=datetime.now(timezone.utc),
                    players=_discord_players,
                    success=True,
                    disabled=getattr(args, "no_discord", False),
                )
            except Exception as _de:
                logger.debug(f"[Discord] Notification ignorée : {_de}")
        else:
            result = asyncio.run(backfill_player_data(args.player, scope=scope))
            _print_summary_player(result, scope)
            # ── Notification Discord (joueur unique) ────────────────────
            try:
                import json as _json

                from src.utils.discord_notifier import (
                    DiscordPlayerResult,
                    count_matches_missing_data,
                    fetch_last_match_info,
                    notify_operation_done,
                )

                _profiles_path = REPO_ROOT / "db_profiles.json"
                _xuid_bf = None
                if _profiles_path.exists():
                    _pdata = _json.loads(_profiles_path.read_text(encoding="utf-8"))
                    for _k, _v in _pdata.get("profiles", {}).items():
                        if _k.lower() == args.player.lower() and isinstance(_v, dict):
                            _xuid_bf = str(_v.get("xuid", "") or "") or None
                            break
                _missing_bf = count_matches_missing_data(_xuid_bf or "") if _xuid_bf else 0
                _last_bf = fetch_last_match_info(_xuid_bf or "") if _xuid_bf else None
                notify_operation_done(
                    operation="backfill",
                    started_at=backfill_started_at,
                    finished_at=datetime.now(timezone.utc),
                    players=[
                        DiscordPlayerResult(
                            gamertag=args.player,
                            xuid=_xuid_bf,
                            matches_synced=result.get("matches_checked", 0),
                            missing_data_count=_missing_bf,
                            last_match=_last_bf,
                        )
                    ],
                    success=True,
                    disabled=getattr(args, "no_discord", False),
                )
            except Exception as _de:
                logger.debug(f"[Discord] Notification ignorée : {_de}")

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
    if getattr(scope, "team_scores", False):
        logger.info(f"Team scores mis à jour: {totals.get('team_scores_updated', 0)}")


if __name__ == "__main__":
    sys.exit(main())
