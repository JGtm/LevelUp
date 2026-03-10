"""Mixin — extraction weapon_kills depuis les films SPNKr (v5.5).

Intégré dans le pipeline de sync : appelé pour chaque nouveau match
ou match connu quand ``SyncOptions.with_weapons=True``.
"""

from __future__ import annotations

import logging
from pathlib import Path
from typing import TYPE_CHECKING, Any

import duckdb

if TYPE_CHECKING:
    from src.data.sync._protocol import _SyncProtocol

logger = logging.getLogger(__name__)


class WeaponKillsEngineMixin:
    """Extraction weapon_kills intégrée au pipeline de sync."""

    async def _try_extract_weapon_kills(
        self: _SyncProtocol,
        client: Any,
        match_id: str,
        shared_conn: duckdb.DuckDBPyConnection | None,
        *,
        is_firefight: bool = False,
    ) -> int:
        """Extrait et stocke les kills par arme pour un match (non bloquant).

        Télécharge les chunks film SPNKr, scanne les fire events POV,
        corrèle chaque kill à son arme et upserte dans ``weapon_kills``.

        Args:
            client: Client API SPNKr authentifié (implémente HaloAPIPort).
            match_id: ID du match à traiter.
            shared_conn: Connexion shared_matches.duckdb (None = skip).
            is_firefight: Si True, skip (pas de film POV en Firefight).

        Returns:
            Nombre de lignes ``weapon_kills`` insérées (0 si skip ou erreur).
        """
        if is_firefight or shared_conn is None:
            return 0
        try:
            cache_dir = _resolve_cache_dir(self._player_db_path)
            from src.data.services.weapon_extraction_service import WeaponExtractionService

            service = WeaponExtractionService(client, shared_conn, cache_dir)
            summary = await service.process_match(
                match_id,
                self._gamertag,
                self._xuid,
            )
            rows = summary.get("rows_inserted", 0)
            if rows > 0:
                logger.debug(
                    "weapon_kills %s : %d lignes (%d/%d kills attribués)",
                    match_id[:8],
                    rows,
                    summary.get("kills_attributed", 0),
                    summary.get("kills_total", 0),
                )
            elif "error" in summary:
                logger.debug("weapon_kills %s : %s", match_id[:8], summary["error"])
            return rows
        except Exception as exc:
            logger.debug("_try_extract_weapon_kills %s (non bloquant) : %s", match_id[:8], exc)
            return 0


def _resolve_cache_dir(player_db_path: Path) -> Path:
    """Retourne le répertoire cache chunks depuis le chemin player DB.

    ``player_db_path`` = ``data/players/{gt}/stats.duckdb``
    → cache = ``data/investigation/chunks``
    """
    data_dir = player_db_path.parents[2]
    return data_dir / "investigation" / "chunks"
