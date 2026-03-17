"""Mixin de compatibilité legacy pour DuckDBRepository.

Méthodes conservées pour la transition depuis src/db/ — résolution d'XUIDs
et informations de session.
"""

from __future__ import annotations

import logging
from typing import Any

logger = logging.getLogger(__name__)


class LegacyCompatMixin:
    """Méthodes de compatibilité pour la migration depuis src/db/."""

    def list_other_player_xuids(self, limit: int = 500) -> list[str]:
        """Liste les XUIDs des autres joueurs rencontrés.

        V5 : Utilise shared.match_participants si disponible (roster complet).
        Complète avec les sources locales.

        Args:
            limit: Nombre max de XUIDs à retourner.

        Returns:
            Liste de XUIDs uniques (hors le joueur principal).
        """
        conn = self._get_connection()
        xuids: set[str] = set()

        try:
            self._collect_xuids_shared(conn, xuids, limit)
            return list(xuids)[:limit]
        except Exception as e:
            logger.debug("Erreur list_other_player_xuids: %s", e)
            return []

    def _collect_xuids_shared(self, conn, xuids: set[str], limit: int) -> None:
        """Collecte les XUIDs depuis shared.match_participants."""
        if not self.has_shared:
            return
        try:
            rows = conn.execute(
                """
                SELECT DISTINCT p2.xuid
                FROM shared.match_participants p1
                INNER JOIN shared.match_participants p2
                  ON p1.match_id = p2.match_id
                WHERE p1.xuid = ? AND p2.xuid != ?
                LIMIT ?
                """,
                [self._xuid, self._xuid, limit],
            ).fetchall()
            xuids.update(str(row[0]) for row in rows if row[0])
        except Exception:
            pass

    def get_match_session_info(self, match_id: str) -> dict[str, Any] | None:
        """Retourne les infos de session pour un match.

        Cherche dans match_stats (local) puis player_match_enrichment (local).
        session_id est une donnée d'enrichissement joueur qui reste locale.

        Args:
            match_id: ID du match.

        Returns:
            Dict avec session_id ou None.
        """
        if not match_id:
            return None

        conn = self._get_connection()

        for table in ("match_stats", "player_match_enrichment"):
            try:
                row = conn.execute(
                    f"SELECT session_id FROM {table} WHERE match_id = ?",
                    [match_id],
                ).fetchone()
                if row and row[0]:
                    return {"session_id": row[0]}
            except Exception:
                continue

        return None
