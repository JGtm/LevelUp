"""Mixin — synchronisation du rang carrière.

Endpoints economy player-gated (nécessite un token par joueur).
"""

from __future__ import annotations

import logging
from datetime import datetime, timezone
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from src.data.sync._protocol import _SyncProtocol

from src.data.sync.api_client import (
    get_player_token_env_key,
    get_tokens_for_player,
)
from src.data.sync.api_factory import create_api_client
from src.data.sync.models import CareerRankData

logger = logging.getLogger(__name__)


class CareerMixin:
    """Méthodes de synchronisation du rang carrière."""

    async def sync_career_rank(self: _SyncProtocol) -> CareerRankData | None:
        """Synchronise la progression du rang carrière.

        Utilise le token propre au joueur (``SPNKR_OAUTH_REFRESH_TOKEN_<GT>``
        dans ``.env.local``) pour les endpoints economy player-gated.
        Si aucun token joueur n'est défini, la sync est sautée proprement.

        Returns:
            CareerRankData ou None si token absent ou erreur.
        """
        try:
            # Priorité : token propre au joueur (endpoint economy player-gated)
            player_tokens = await get_tokens_for_player(self._gamertag)
            if player_tokens is None:
                logger.warning(
                    "event=player_gated_skip gamertag=%s step=career_rank"
                    " reason=no_player_token hint=set %s in .env.local",
                    self._gamertag,
                    get_player_token_env_key(self._gamertag),
                )
                return None

            async with create_api_client(tokens=player_tokens) as client:
                career_data = await client.get_career_rank_progression(self._xuid)

                if career_data is None:
                    logger.info("Career rank non disponible pour %s", self._gamertag)
                    return None

                # Sauvegarder en BDD
                self._save_career_rank(career_data)

                logger.info(
                    "Career rank sync: %s → Rang %d (%s)",
                    self._gamertag,
                    career_data.current_rank,
                    career_data.current_rank_name,
                )

                return career_data

        except Exception as e:
            logger.error("Erreur sync_career_rank: %s", e)
            return None

    def _save_career_rank(self: _SyncProtocol, data: CareerRankData) -> None:
        """Sauvegarde un snapshot de la progression de rang."""
        conn = self._get_connection()
        now = datetime.now(timezone.utc)

        conn.execute(
            """INSERT INTO career_progression (
                xuid, rank, rank_name, rank_tier,
                current_xp, xp_for_next_rank, xp_total,
                is_max_rank, adornment_path, spartan_id, recorded_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)""",
            (
                data.xuid,
                data.current_rank,
                data.current_rank_name,
                data.current_rank_tier,
                data.current_xp,
                data.xp_for_next_rank,
                data.xp_total,
                data.is_max_rank,
                data.adornment_path,
                data.spartan_id,
                now,
            ),
        )
        conn.commit()

        # Mettre à jour sync_meta
        self._update_sync_meta("last_career_sync_at", now.isoformat())
        self._update_sync_meta("current_rank", str(data.current_rank))

    def get_career_rank_history(self: _SyncProtocol, limit: int = 50) -> list[dict[str, Any]]:
        """Récupère l'historique de progression de rang.

        Args:
            limit: Nombre maximum d'entrées.

        Returns:
            Liste des snapshots de progression.
        """
        try:
            conn = self._get_connection()
            result = conn.execute(
                """SELECT rank, rank_name, rank_tier, current_xp,
                          xp_for_next_rank, xp_total, is_max_rank,
                          adornment_path, spartan_id, recorded_at
                   FROM career_progression
                   WHERE xuid = ?
                   ORDER BY recorded_at DESC
                   LIMIT ?""",
                (self._xuid, limit),
            ).fetchall()

            return [
                {
                    "rank": r[0],
                    "rank_name": r[1],
                    "rank_tier": r[2],
                    "current_xp": r[3],
                    "xp_for_next_rank": r[4],
                    "xp_total": r[5],
                    "is_max_rank": r[6],
                    "adornment_path": r[7],
                    "spartan_id": r[8],
                    "recorded_at": r[9],
                }
                for r in result
            ]

        except Exception as e:
            logger.warning("Erreur get_career_rank_history: %s", e)
            return []

    def get_latest_career_rank(self) -> dict[str, Any] | None:
        """Récupère le dernier rang carrière enregistré.

        Returns:
            Dict avec les infos du rang ou None.
        """
        history = self.get_career_rank_history(limit=1)
        return history[0] if history else None
