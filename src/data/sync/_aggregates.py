"""Mixin — rafraîchissement des agrégats et statut de synchronisation.

Vues matérialisées, pré-calculs post-sync et métadonnées sync_meta.
"""

from __future__ import annotations

import asyncio
import logging
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from src.data.sync._protocol import _SyncProtocol

logger = logging.getLogger(__name__)


class AggregatesMixin:
    """Méthodes de rafraîchissement des agrégats post-sync."""

    async def _refresh_aggregates_async(self, new_ids: list[str] | None = None) -> None:
        """Rafraîchit les agrégats après sync (async wrapper)."""
        # Exécuter dans un thread pour ne pas bloquer l'event loop
        loop = asyncio.get_event_loop()
        await loop.run_in_executor(None, lambda: self.refresh_aggregates(new_ids=new_ids))

    def refresh_aggregates(  # noqa: PLR0912
        self: _SyncProtocol, *, new_ids: list[str] | None = None
    ) -> dict[str, int]:
        """Recalcule les tables d'agrégats après sync.

        Met à jour :
        - Vues matérialisées (mv_*)

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
                from src.data.repositories.duckdb_repo import (
                    DuckDBRepository,
                    begin_sync_mode,
                    end_sync_mode,
                )

                # shared_connection est fermé : désactiver brièvement sync_mode
                # pour permettre au DuckDBRepository d'attacher shared_matches_v2.duckdb
                end_sync_mode()
                try:
                    repo = DuckDBRepository(
                        self._player_db_path,
                        self._xuid,
                        read_only=False,
                    )
                    # Forcer l'initialisation de la connexion ET l'ATTACH de shared
                    # pendant que sync_mode est encore désactivé (connexion lazy).
                    repo._get_connection()
                finally:
                    begin_sync_mode()  # Rétablir après que la connexion est établie

                try:
                    repo.refresh_materialized_views(new_ids=new_ids)
                    result["materialized_views"] = 1
                finally:
                    # CRITIQUE : fermer repo même si refresh_materialized_views lève une
                    # exception, pour libérer le handle sur shared_matches_v2.duckdb et
                    # permettre à _get_shared_connection() de rouvrir la connexion.
                    repo.close()
            except Exception as e:
                logger.debug("refresh_materialized_views non disponible: %s", e)

            # Sprint 8ter.4 : Pré-calcul post-sync des agrégats UI
            try:
                from scripts.post_sync_compute import post_sync_compute

                result.update(post_sync_compute(str(self._player_db_path)))
            except Exception as e:
                logger.debug("post_sync_compute non disponible: %s", e)

            # Rouvrir la connexion shared si elle était ouverte avant
            if shared_was_open:
                self._shared_connection = None  # Force réouverture au prochain appel
                try:
                    self._get_shared_connection()
                except Exception as e:
                    logger.debug(
                        "event=shared_conn_reopen_failed step=refresh_aggregates error=%s", e
                    )

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
                try:
                    match_count = shared_conn.execute(
                        "SELECT COUNT(DISTINCT match_id) FROM match_participants WHERE xuid = ?",
                        [self._xuid],
                    ).fetchone()[0]
                except Exception as e:
                    logger.debug("event=match_count_failed step=get_sync_status error=%s", e)

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
