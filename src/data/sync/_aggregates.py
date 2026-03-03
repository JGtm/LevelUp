"""Mixin — rafraîchissement des agrégats et statut de synchronisation.

Vues matérialisées, pré-calculs post-sync et métadonnées sync_meta.
"""

from __future__ import annotations

import asyncio
import contextlib
import logging
from typing import Any

logger = logging.getLogger(__name__)


class AggregatesMixin:
    """Méthodes de rafraîchissement des agrégats post-sync."""

    async def _refresh_aggregates_async(self) -> None:
        """Rafraîchit les agrégats après sync (async wrapper)."""
        # Exécuter dans un thread pour ne pas bloquer l'event loop
        loop = asyncio.get_event_loop()
        await loop.run_in_executor(None, self.refresh_aggregates)

    def refresh_aggregates(self) -> dict[str, int]:
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
            shared_was_open = self._shared_connection is not None  # type: ignore[attr-defined]
            if shared_was_open:
                try:
                    self._shared_connection.close()  # type: ignore[attr-defined]
                    self._shared_connection = None  # type: ignore[attr-defined]
                except Exception:
                    pass

            # Appeler refresh_materialized_views si disponible
            # (implémenté dans DuckDBRepository)
            try:
                from src.data.repositories.duckdb_repo import DuckDBRepository

                repo = DuckDBRepository(
                    self._player_db_path,  # type: ignore[attr-defined]
                    self._xuid,  # type: ignore[attr-defined]
                    read_only=False,
                )
                repo.refresh_materialized_views()
                result["materialized_views"] = 1
                repo.close()
            except Exception as e:
                logger.debug(f"refresh_materialized_views non disponible: {e}")

            # Sprint 8ter.4 : Pré-calcul post-sync des agrégats UI
            try:
                from scripts.post_sync_compute import post_sync_compute

                precomp = post_sync_compute(str(self._player_db_path))  # type: ignore[attr-defined]
                result.update(precomp)
            except Exception as e:
                logger.debug(f"post_sync_compute non disponible: {e}")

            # Rouvrir la connexion shared si elle était ouverte avant
            if shared_was_open:
                self._shared_connection = None  # type: ignore[attr-defined]  # Force réouverture au prochain appel
                with contextlib.suppress(Exception):
                    self._get_shared_connection()  # type: ignore[attr-defined]

        except Exception as e:
            logger.warning(f"Erreur refresh_aggregates: {e}")

        return result

    def get_sync_status(self) -> dict[str, Any]:
        """Retourne l'état de la dernière synchronisation.

        Returns:
            Dict avec last_sync_at, total_matches, etc.
        """
        try:
            # V5 finale : compter depuis shared.match_participants
            shared_conn = self._get_shared_connection()  # type: ignore[attr-defined]
            match_count = 0
            if shared_conn is not None and self._xuid:  # type: ignore[attr-defined]
                with contextlib.suppress(Exception):
                    match_count = shared_conn.execute(
                        "SELECT COUNT(DISTINCT match_id) FROM match_participants WHERE xuid = ?",
                        [self._xuid],  # type: ignore[attr-defined]
                    ).fetchone()[0]

            # Récupérer les métadonnées
            last_sync = self._get_sync_meta("last_sync_at")  # type: ignore[attr-defined]
            last_mode = self._get_sync_meta("last_sync_mode")  # type: ignore[attr-defined]
            last_matches = self._get_sync_meta("last_sync_matches")  # type: ignore[attr-defined]

            return {
                "total_matches": match_count,
                "last_sync_at": last_sync,
                "last_sync_mode": last_mode,
                "last_sync_matches": int(last_matches) if last_matches else 0,
                "gamertag": self._gamertag,  # type: ignore[attr-defined]
                "xuid": self._xuid,  # type: ignore[attr-defined]
            }
        except Exception as e:
            return {"error": str(e)}
