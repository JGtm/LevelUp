"""Logique async backfill weapon_kills — délégué depuis strategies.py."""

from __future__ import annotations

import asyncio
import logging
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)

_PROJECT_ROOT = Path(__file__).resolve().parents[2]
_DEFAULT_CACHE_DIR = _PROJECT_ROOT / "data" / "cache" / "film_chunks"

# Nombre de matchs traités en parallèle (limité pour ne pas saturer l'API SPNKr)
_MATCH_CONCURRENCY = 4

# Intervalle d'affichage progression (tous les N matchs)
_PROGRESS_INTERVAL = 25


def _log_weapon_progress(progress: dict, total: int) -> None:
    """Affiche la progression du backfill weapon_kills tous les _PROGRESS_INTERVAL matchs."""
    done = progress["done"]
    if done % _PROGRESS_INTERVAL == 0 or done == total:
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
        shared_conn: Connexion ouverte vers shared_matches.duckdb.
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
        progress = {"done": 0, "rows": 0, "skipped": 0, "errors": 0}
        total_matches = len(match_ids)

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

    logger.info(
        "weapon_kills backfill terminé : %d lignes insérées (%d matchs)", total, len(match_ids)
    )
    return total
