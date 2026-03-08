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
from src.utils.log_config import setup_script_logging  # noqa: E402

setup_script_logging(sync_log=True)

logger = logging.getLogger(__name__)

_SHARED_DB = REPO_ROOT / "data" / "warehouse" / "shared_matches.duckdb"


def _open_shared_conn() -> duckdb.DuckDBPyConnection:
    return duckdb.connect(str(_SHARED_DB))


def main() -> int:  # noqa: C901, PLR0912, PLR0915
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

    # --detect-stale-events est un diagnostic global (local, sans API)
    detect_stale = getattr(args, "detect_stale_events", False)
    if detect_stale:
        try:
            from scripts.backfill.detection import find_matches_with_stale_spnkr

            shared_conn = _open_shared_conn()
            stale = find_matches_with_stale_spnkr(shared_conn, min_version="0.10.1")
            n_versioned = len(stale["stale_versioned"])
            n_unknown = len(stale["stale_unknown"])

            logger.info("=" * 60)
            logger.info("DIAGNOSTIC : Matchs avec highlight events potentiellement stale")
            logger.info("=" * 60)
            logger.info(f"Matchs syncés avec SPNKr < 0.10.1 (events chargés) : {n_versioned}")
            logger.info(f"Matchs récents sans version trackée + sans events  : {n_unknown}")

            if n_versioned > 0:
                logger.info("\nMatchs avec version obsolète (top 10) :")
                for mid in stale["stale_versioned"][:10]:
                    logger.info(f"  - {mid}")
            if n_unknown > 0:
                logger.info("\nMatchs pré-tracking sans events (top 10) :")
                for mid in stale["stale_unknown"][:10]:
                    logger.info(f"  - {mid}")

            if n_versioned + n_unknown > 0:
                logger.info("\n→ Action recommandée : mettre à jour SPNKr puis lancer :")
                logger.info("  pip install --upgrade spnkr")
                logger.info("  python scripts/backfill_data.py --all --events --force-medals")
            else:
                logger.info("\n✅ Aucun match stale détecté.")
            shared_conn.close()
        except Exception as e:
            logger.error(f"Erreur --detect-stale-events : {e}")
            import traceback

            traceback.print_exc()
        return 0

    # --bot-detection : détecte les coéquipiers bots et met à jour had_bot_teammate
    bot_detection = getattr(args, "bot_detection", False)
    if bot_detection:
        try:
            from src.ui.multiplayer import list_duckdb_v4_players

            _SHARED_DB_BOT = REPO_ROOT / "data" / "warehouse" / "shared_matches.duckdb"
            if not _SHARED_DB_BOT.exists():
                logger.error("shared_matches.duckdb introuvable pour --bot-detection")
                return 1

            shared_conn = duckdb.connect(str(_SHARED_DB_BOT), read_only=True)
            _players_list = list_duckdb_v4_players()
            total_updated = 0

            for _gt_info in _players_list:
                _gt = _gt_info.gamertag
                _db = _gt_info.db_path  # déjà un Path
                if not _db.exists():
                    continue
                # Utiliser le xuid déjà présent dans DuckDBPlayerInfo si disponible
                try:
                    _pconn = duckdb.connect(str(_db), read_only=False)
                    if _gt_info.xuid:
                        _xuid = _gt_info.xuid
                    else:
                        _xuid_row = _pconn.execute(
                            "SELECT value FROM sync_meta WHERE key = 'xuid' LIMIT 1"
                        ).fetchone()
                        if not _xuid_row:
                            _pconn.close()
                            continue
                        _xuid = _xuid_row[0]

                    # Ajouter la colonne si absente
                    from src.data.sync.migrations import ensure_bot_teammate_column

                    ensure_bot_teammate_column(_pconn)

                    # Trouver les matchs où un coéquipier du joueur était un bot
                    # Un bot a xuid LIKE 'bid(%'.
                    # On filtre les bots très brefs (< 60s) = simples déconnexions/reconnexions.
                    # On stocke aussi l'outcome du joueur pour la logique d'indulgence.
                    _bot_matches = shared_conn.execute(
                        """
                        SELECT DISTINCT mp.match_id, mp.outcome
                        FROM match_participants mp
                        JOIN match_participants bot ON mp.match_id = bot.match_id
                            AND mp.team_id = bot.team_id
                            AND bot.xuid LIKE 'bid(%'
                            AND bot.time_played_seconds > 60
                        WHERE mp.xuid = ?
                          AND mp.xuid NOT LIKE 'bid(%'
                        """,
                        [_xuid],
                    ).fetchall()

                    if _bot_matches:
                        _match_ids = [r[0] for r in _bot_matches]
                        placeholders = ", ".join(["?"] * len(_match_ids))

                        # Séparer les match_ids existants (UPDATE) des absents (INSERT)
                        _existing_set = {
                            r[0]
                            for r in _pconn.execute(
                                f"SELECT match_id FROM player_match_enrichment "
                                f"WHERE match_id IN ({placeholders})",
                                _match_ids,
                            ).fetchall()
                        }

                        _n = 0

                        # UPDATE les lignes existantes
                        _existing_list = [m for m in _match_ids if m in _existing_set]
                        if _existing_list:
                            _ep = ", ".join(["?"] * len(_existing_list))
                            _updated = _pconn.execute(
                                f"UPDATE player_match_enrichment "
                                f"SET had_bot_teammate = TRUE, updated_at = CURRENT_TIMESTAMP "
                                f"WHERE match_id IN ({_ep}) "
                                f"  AND (had_bot_teammate IS NULL OR had_bot_teammate = FALSE) "
                                f"RETURNING match_id",
                                _existing_list,
                            ).fetchall()
                            _n += len(_updated)

                        # INSERT les matchs sans ligne dans player_match_enrichment
                        _new_list = [m for m in _match_ids if m not in _existing_set]
                        if _new_list:
                            _pconn.executemany(
                                "INSERT INTO player_match_enrichment (match_id, had_bot_teammate) "
                                "VALUES (?, TRUE)",
                                [(m,) for m in _new_list],
                            )
                            _n += len(_new_list)

                        _pconn.commit()
                        total_updated += _n
                        logger.info(
                            f"[{_gt}] {_n} matchs mis à jour "
                            f"(bots trouvés: {len(_match_ids)}, "
                            f"mis à jour: {len(_existing_list)}, "
                            f"nouveaux: {len(_new_list)})"
                        )
                    _pconn.close()
                except Exception as _e:
                    logger.warning(f"[{_gt}] Erreur bot-detection : {_e}")

            shared_conn.close()
            logger.info(
                f"Bot detection terminée : {total_updated} ligne(s) mise(s) à jour au total"
            )
        except Exception as e:
            logger.error(f"Erreur --bot-detection : {e}")
        return 0

    # --enable-pve-citations : active les citations PVE désactivées dans metadata.duckdb
    enable_pve_citations = getattr(args, "enable_pve_citations", False)
    if enable_pve_citations:
        try:
            _META_DB = REPO_ROOT / "data" / "warehouse" / "metadata.duckdb"
            meta_conn = duckdb.connect(str(_META_DB), read_only=False)
            n_enabled = meta_conn.execute(
                "UPDATE citation_mappings SET enabled = TRUE "
                "WHERE citation_name_norm IN ('brute_slayer', 'skimmer_slayer') "
                "  AND (enabled IS NULL OR enabled = FALSE)"
            ).rowcount
            meta_conn.commit()
            meta_conn.close()
            logger.info(f"Citations PVE activées : {n_enabled} entrée(s) mise(s) à jour")
            logger.info(
                "  → brute_slayer (kills_brute) et skimmer_slayer (kills_skimmer) sont maintenant actives"
            )
            logger.info(
                "  → Lance --all --citations --force-citations pour recalculer les compteurs"
            )
        except Exception as e:
            logger.error(f"Erreur --enable-pve-citations : {e}")
        return 0

    # --xp-total : recalcule xp_total + xp_for_next_rank dans career_progression (tous joueurs)
    xp_total_flag = getattr(args, "xp_total", False)
    if xp_total_flag:
        try:
            from pathlib import Path

            from scripts.backfill.strategies import backfill_career_xp_total
            from src.ui.multiplayer import list_duckdb_v4_players

            players = list_duckdb_v4_players()
            total_updated = 0
            for pinfo in players:
                db = Path(pinfo.db_path)
                if not db.exists():
                    logger.debug(f"[{pinfo.gamertag}] stats.duckdb absent, ignoré")
                    continue
                try:
                    n = backfill_career_xp_total(str(db))
                    if n > 0:
                        logger.info(f"[{pinfo.gamertag}] {n} snapshot(s) xp_total recalculé(s)")
                    else:
                        logger.debug(f"[{pinfo.gamertag}] aucun snapshot career_progression")
                    total_updated += n
                except Exception as _e:
                    logger.warning(f"[{pinfo.gamertag}] Erreur --xp-total : {_e}")
            logger.info(f"xp_total recalculé pour {total_updated} snapshot(s) au total")
        except Exception as e:
            logger.error(f"Erreur --xp-total : {e}")
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
                    _pres = result.get("per_player", {}).get(_pinfo.gamertag, {})
                    _discord_players.append(
                        DiscordPlayerResult(
                            gamertag=_pinfo.gamertag,
                            xuid=_xuid_bf,
                            matches_synced=_pres.get("matches_checked", 0),
                            missing_data_count=_missing_bf,
                            last_match=_last_bf,
                            backfill_counts=_pres,
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
                            backfill_counts=result,
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


def _print_totals(totals: dict, scope: object) -> None:  # noqa: C901, PLR0912
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
    if getattr(scope, "teammates_sig", False):
        logger.info(f"Signatures coéquipiers: {totals.get('teammates_sig_updated', 0)}")
    if getattr(scope, "citations", False):
        logger.info(f"Citations calculées: {totals.get('citations_computed', 0)}")
    if getattr(scope, "participants_enrich", False):
        logger.info(f"Participants enrichis: {totals.get('participants_enriched', 0)}")
    if getattr(scope, "team_scores", False):
        logger.info(f"Team scores mis à jour: {totals.get('team_scores_updated', 0)}")


if __name__ == "__main__":
    from src.utils.sync_lock import SyncAlreadyRunning, SyncLock

    try:
        with SyncLock():
            sys.exit(main())
    except SyncAlreadyRunning as _e:
        logger.error("%s", _e)
        sys.exit(2)
