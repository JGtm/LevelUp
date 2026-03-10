"""Mixin — rafraîchissement des agrégats et statut de synchronisation.

Vues matérialisées, pré-calculs post-sync et métadonnées sync_meta.
"""

from __future__ import annotations

import asyncio
import contextlib
import logging
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from src.data.sync._protocol import _SyncProtocol

logger = logging.getLogger(__name__)


class AggregatesMixin:
    """Méthodes de rafraîchissement des agrégats post-sync."""

    async def _refresh_aggregates_async(self) -> None:
        """Rafraîchit les agrégats après sync (async wrapper)."""
        # Exécuter dans un thread pour ne pas bloquer l'event loop
        loop = asyncio.get_event_loop()
        await loop.run_in_executor(None, self.refresh_aggregates)

    def refresh_aggregates(self: _SyncProtocol) -> dict[str, int]:
        """Recalcule les tables d'agrégats après sync.

        Met à jour :
        - Vues matérialisées (mv_*)
        - Tables pré-calculées (precomputed_sessions, precomputed_kda_trend, etc.)

        Returns:
            Dict table_name → rows_affected.
        """
        result: dict[str, int] = {}

        try:
            # Fermer temporairement la connexion shared pour éviter les conflits
            # quand DuckDBRepository tente de l'attacher
            shared_was_open = self._shared_connection is not None
            if shared_was_open:
                try:
                    self._shared_connection.close()  # type: ignore[union-attr]
                    self._shared_connection = None
                except Exception:
                    pass

            # Appeler refresh_materialized_views si disponible
            # (implémenté dans DuckDBRepository)
            try:
                from src.data.repositories.duckdb_repo import DuckDBRepository

                repo = DuckDBRepository(
                    self._player_db_path,
                    self._xuid,
                    read_only=False,
                )
                try:
                    repo.refresh_materialized_views()
                    result["materialized_views"] = 1
                finally:
                    # CRITIQUE : fermer repo même si refresh_materialized_views lève une
                    # exception, pour libérer le handle sur shared_matches.duckdb et
                    # permettre à _get_shared_connection() de rouvrir la connexion.
                    repo.close()
            except Exception as e:
                logger.debug("refresh_materialized_views non disponible: %s", e)

            # Sprint 8ter.4 : Pré-calcul post-sync des agrégats UI
            try:
                from scripts.post_sync_compute import post_sync_compute

                precomp = post_sync_compute(str(self._player_db_path))
                result.update(precomp)
            except Exception as e:
                logger.debug("post_sync_compute non disponible: %s", e)

            # Rouvrir la connexion shared si elle était ouverte avant
            if shared_was_open:
                self._shared_connection = None  # Force réouverture au prochain appel
                with contextlib.suppress(Exception):
                    self._get_shared_connection()

        except Exception as e:
            logger.warning("Erreur refresh_aggregates: %s", e)

        return result

    def get_sync_status(self: _SyncProtocol) -> dict[str, Any]:
        """Retourne l'état de la dernière synchronisation.

        Returns:
            Dict avec last_sync_at, total_matches, etc.
        """
        try:
            # V5 finale : compter depuis shared.match_participants
            shared_conn = self._get_shared_connection()
            match_count = 0
            if shared_conn is not None and self._xuid:
                with contextlib.suppress(Exception):
                    match_count = shared_conn.execute(
                        "SELECT COUNT(DISTINCT match_id) FROM match_participants WHERE xuid = ?",
                        [self._xuid],
                    ).fetchone()[0]

            # Récupérer les métadonnées
            last_sync = self._get_sync_meta("last_sync_at")
            last_mode = self._get_sync_meta("last_sync_mode")
            last_matches = self._get_sync_meta("last_sync_matches")

            return {
                "total_matches": match_count,
                "last_sync_at": last_sync,
                "last_sync_mode": last_mode,
                "last_sync_matches": int(last_matches) if last_matches else 0,
                "gamertag": self._gamertag,
                "xuid": self._xuid,
            }
        except Exception as e:
            return {"error": str(e)}
