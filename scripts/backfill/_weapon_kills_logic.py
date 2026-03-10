"""Logique async backfill weapon_kills — délégué depuis strategies.py."""

from __future__ import annotations

import asyncio
import logging
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)

_PROJECT_ROOT = Path(__file__).resolve().parents[2]
_DEFAULT_CACHE_DIR = _PROJECT_ROOT / "data" / "investigation" / "chunks"

# Nombre de matchs traités en parallèle (limité pour ne pas saturer l'API SPNKr)
_MATCH_CONCURRENCY = 4


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
                            return 0
                    except Exception as exc:
                        logger.debug("weapon_kills guard batch %s : %s", match_id[:8], exc)
                try:
                    summary = await service.process_match(match_id, gamertag, xuid)
                    rows = summary.get("rows_inserted", 0)
                    if rows > 0:
                        logger.debug(
                            "weapon_kills backfill %s : %d lignes (%d/%d kills)",
                            match_id[:8],
                            rows,
                            summary.get("kills_attributed", 0),
                            summary.get("kills_total", 0),
                        )
                    elif "error" in summary:
                        logger.debug("weapon_kills %s : %s", match_id[:8], summary["error"])
                    else:
                        logger.debug(
                            "weapon_kills %s : 0 lignes (match sans kills attribuables)",
                            match_id[:8],
                        )
                    return rows
                except Exception as exc:
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
