"""Logique async backfill weapon_kills — délégué depuis strategies.py."""

from __future__ import annotations

import asyncio
import logging
import time
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)

_PROJECT_ROOT = Path(__file__).resolve().parents[2]
_DEFAULT_CACHE_DIR = _PROJECT_ROOT / "data" / "cache" / "film_chunks"

# Nombre de matchs traités en parallèle (limité pour ne pas saturer l'API SPNKr)
_MATCH_CONCURRENCY = 4

# Intervalle d'affichage progression (toutes les N secondes)
_PROGRESS_INTERVAL_SECS = 10.0


def _log_weapon_progress(progress: dict, total: int) -> None:
    """Affiche la progression du backfill weapon_kills toutes les _PROGRESS_INTERVAL_SECS secondes."""
    done = progress["done"]
    is_final = done == total
    now = time.monotonic()
    if is_final or (now - progress["last_log"]) >= _PROGRESS_INTERVAL_SECS:
        pct = done * 100 // total if total else 100
        logger.info(
            "⚔️  Weapon kills : %d/%d matchs (%d%%) — %d lignes, %d skips, %d erreurs",
            done,
            total,
            pct,
            progress["rows"],
            progress["skipped"],
            progress["errors"],
        )
        progress["last_log"] = now


async def collect_weapon_match_ids_all_players(
    players: list,
    shared_conn: Any,
    *,
    force: bool = False,
    limit_per_player: int = 5000,
) -> list[str]:
    """Collecte l'union dédupliquée des match_ids weapons pour tous les joueurs.

    Avec la v2 du parser (scan_fire_events_all + correlate_all_players), un
    match est traité pour **tous les joueurs en une seule passe**. Appeler
    run_weapon_kills_backfill par joueur causerait N re-téléchargements inutiles
    des mêmes films pour les matchs partagés (escouade).

    Cette fonction collecte les match_ids manquants de chaque joueur et les
    déduplique avant de passer l'union à run_weapon_kills_backfill.

    Args:
        players: Liste de DuckDBPlayerInfo (champ xuid optionnel).
        shared_conn: Connexion ouverte vers shared_matches_v2.duckdb.
        force: Si True, inclure aussi les matchs déjà traités.
        limit_per_player: Limite de matchs par joueur (garde-fou).

    Returns:
        Liste de match_ids uniques, sans ordre garanti.
    """
    from src.data.repositories._weapon_kills_repo import WeaponKillsMixin

    all_ids: set[str] = set()
    n_players = 0

    for player_info in players:
        xuid = getattr(player_info, "xuid", None)
        if not xuid:
            xuid = WeaponKillsMixin.get_xuid_by_gamertag(shared_conn, player_info.gamertag)
        if not xuid:
            logger.warning(
                "collect_weapon_match_ids : xuid introuvable pour %s — ignoré",
                player_info.gamertag,
            )
            continue
        ids = WeaponKillsMixin.get_matches_missing_weapons(
            shared_conn, xuid, limit=limit_per_player, force=force
        )
        all_ids.update(ids)
        n_players += 1

    logger.info(
        "collect_weapon_match_ids : %d match_ids uniques issus de %d joueur(s)",
        len(all_ids),
        n_players,
    )
    return list(all_ids)


async def run_weapon_kills_backfill(
    gamertag: str,
    xuid: str,
    match_ids: list[str],
    shared_conn: Any,
    *,
    force: bool = False,
    cache_dir: Path | None = None,
) -> int:
    """Télécharge les films et upserte weapon_kills pour une liste de matchs.

    Traite jusqu'à _MATCH_CONCURRENCY matchs en parallèle. Un asyncio.Lock
    sérialise les écritures DuckDB (shared_conn n'est pas concurrent-safe).

    Args:
        gamertag: Gamertag du joueur (nécessaire pour charger kills POV).
        xuid: XUID du joueur.
        match_ids: Liste de match_id à traiter.
        shared_conn: Connexion ouverte vers shared_matches_v2.duckdb.
        force: Si True, re-traite même si MatchBits.WEAPON_KILLS déjà posé.
        cache_dir: Répertoire cache chunks (défaut : data/investigation/chunks).

    Returns:
        Nombre total de lignes weapon_kills insérées.
    """
    from src.data.services.weapon_extraction_service import WeaponExtractionService
    from src.data.sync.api_client import SPNKrAPIClient, get_tokens_from_env
    from src.data.sync.constants import MatchBits

    if not match_ids:
        return 0

    cache = cache_dir or _DEFAULT_CACHE_DIR
    tokens = await get_tokens_from_env()
    if tokens is None:
        logger.error("run_weapon_kills_backfill : tokens API introuvables (.env.local)")
        return 0

    write_lock = asyncio.Lock()
    match_sem = asyncio.Semaphore(_MATCH_CONCURRENCY)

    async with SPNKrAPIClient(tokens=tokens) as api:
        service = WeaponExtractionService(api, shared_conn, cache, write_lock=write_lock)
        total_matches = len(match_ids)
        logger.info("⚔️  Weapon kills : démarrage — %d matchs à traiter", total_matches)
        progress = {"done": 0, "rows": 0, "skipped": 0, "errors": 0, "last_log": time.monotonic()}

        async def _process_one(match_id: str) -> int:
            async with match_sem:
                # Guard : skip si bit WEAPON_KILLS déjà posé (sauf force=True).
                # Nécessaire car _pending_weapon_ids peut contenir des matchs qui
                # avaient besoin d'autres données (OR detection) mais dont les
                # weapon kills sont déjà traités (par un autre joueur en escouade).
                if not force:
                    try:
                        row = shared_conn.execute(
                            "SELECT COALESCE(backfill_completed, 0) & ? "
                            "FROM match_registry WHERE match_id = ?",
                            (MatchBits.WEAPON_KILLS, match_id),
                        ).fetchone()
                        if row and row[0]:
                            logger.debug("weapon_kills batch %s : bit posé, skip", match_id[:8])
                            progress["done"] += 1
                            progress["skipped"] += 1
                            _log_weapon_progress(progress, total_matches)
                            return 0
                    except Exception as exc:
                        logger.debug("weapon_kills guard batch %s : %s", match_id[:8], exc)
                try:
                    summary = await service.process_match(match_id, gamertag, xuid)
                    rows = summary.rows_inserted
                    progress["done"] += 1
                    progress["rows"] += rows
                    if rows > 0:
                        logger.debug(
                            "weapon_kills backfill %s : %d lignes (%d/%d kills)",
                            match_id[:8],
                            rows,
                            summary.kills_attributed,
                            summary.kills_total,
                        )
                    elif summary.error:
                        logger.debug("weapon_kills %s : %s", match_id[:8], summary.error)
                    else:
                        logger.debug(
                            "weapon_kills %s : 0 lignes (match sans kills attribuables)",
                            match_id[:8],
                        )
                    _log_weapon_progress(progress, total_matches)
                    return rows
                except Exception as exc:
                    progress["done"] += 1
                    progress["errors"] += 1
                    _log_weapon_progress(progress, total_matches)
                    logger.warning(
                        "weapon_kills backfill %s : erreur non fatale : %s", match_id[:8], exc
                    )
                    return 0

        results = await asyncio.gather(*[_process_one(mid) for mid in match_ids])
        total = sum(r for r in results if isinstance(r, int))

    return total
