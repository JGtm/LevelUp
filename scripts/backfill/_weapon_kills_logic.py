"""Logique async backfill weapon_kills — délégué depuis strategies.py."""

from __future__ import annotations

import logging
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)

_PROJECT_ROOT = Path(__file__).resolve().parents[2]
_DEFAULT_CACHE_DIR = _PROJECT_ROOT / "data" / "investigation" / "chunks"


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

    cache = cache_dir or _DEFAULT_CACHE_DIR
    tokens = await get_tokens_from_env()
    if tokens is None:
        logger.error("run_weapon_kills_backfill : tokens API introuvables (.env.local)")
        return 0

    total = 0
    async with SPNKrAPIClient(tokens=tokens) as api:
        service = WeaponExtractionService(api, shared_conn, cache)
        for match_id in match_ids:
            try:
                summary = await service.process_match(match_id, gamertag, xuid)
                rows = summary.get("rows_inserted", 0)
                total += rows
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
            except Exception as exc:
                logger.warning(
                    "weapon_kills backfill %s : erreur non fatale : %s", match_id[:8], exc
                )

    logger.info(
        "weapon_kills backfill terminé : %d lignes insérées (%d matchs)", total, len(match_ids)
    )
    return total
