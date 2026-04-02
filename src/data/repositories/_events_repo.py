"""
Mixin pour le chargement des highlight events depuis shared_matches.duckdb.

Regroupe les méthodes d'événements extraites de DuckDBRepository :
- load_first_event_times
- get_first_kill_death_times
- load_highlight_events
"""

from __future__ import annotations

import logging
from typing import Any

logger = logging.getLogger(__name__)


class EventsMixin:
    """Méthodes DuckDBRepository relatives aux highlight events."""

    def load_first_event_times(
        self,
        match_ids: list[str],
        event_type: str = "Kill",
    ) -> dict[str, int | None]:
        """Charge le timestamp du premier événement par match.

        V5 : Utilise shared.highlight_events si disponible.

        Args:
            match_ids: Liste des IDs de matchs.
            event_type: Type d'événement ("Kill" ou "Death"). Accepte toute casse.

        Returns:
            Dict {match_id: time_ms} pour le premier événement de chaque match.
        """
        if not match_ids:
            return {}

        conn = self._get_connection()
        # Préparer les variantes de casse pour utiliser l'index sans LOWER()
        event_variants = list({event_type, event_type.lower(), event_type.capitalize()})
        event_placeholders = ", ".join(["?" for _ in event_variants])
        placeholders = ", ".join(["?" for _ in match_ids])

        # Lecture depuis shared.highlight_events (xuid = le joueur de l'event)
        # Note v5.1 : xuid est le killer pour 'kill', la victime pour 'death'
        # countdown_ms = compte à rebours pré-match en ms (0 si données absentes).
        # Soustrait de MIN(time_ms) pour obtenir le temps depuis le vrai début
        # du gameplay, pas depuis le début de la timeline Halo.
        try:
            result = conn.execute(
                f"""
                SELECT
                    e.match_id,
                    GREATEST(
                        MIN(e.time_ms)
                        - GREATEST(
                            (COALESCE(ANY_VALUE(r.duration_seconds), 0)
                             - COALESCE(ANY_VALUE(r.playable_duration_seconds),
                                        ANY_VALUE(r.duration_seconds), 0))
                            * 1000,
                            0
                        ),
                        0
                    ) AS first_time
                FROM shared.highlight_events e
                JOIN shared.match_registry r ON r.match_id = e.match_id
                WHERE e.match_id IN ({placeholders})
                  AND e.event_type IN ({event_placeholders})
                  AND e.xuid = ?
                GROUP BY e.match_id
                """,
                [*match_ids, *event_variants, self._xuid],
            )
            return {row[0]: row[1] for row in result.fetchall()}
        except Exception as e:
            logger.debug("Erreur load_first_event_times shared: %s", e)
            return {}

    def get_first_kill_death_times(
        self,
        match_ids: list[str],
    ) -> tuple[dict[str, int | None], dict[str, int | None]]:
        """Charge les timestamps du premier kill et première mort par match.

        Args:
            match_ids: Liste des IDs de matchs.

        Returns:
            Tuple (first_kills, first_deaths) où chaque dict est {match_id: time_ms}.
        """
        first_kills = self.load_first_event_times(match_ids, event_type="Kill")
        first_deaths = self.load_first_event_times(match_ids, event_type="Death")
        return first_kills, first_deaths

    def load_highlight_events(self, match_id: str) -> list[dict[str, Any]]:
        """Charge les highlight events pour un match depuis shared.

        Args:
            match_id: ID du match.

        Returns:
            Liste de dicts avec: event_type, time_ms, xuid, gamertag, type_hint.
        """
        if not match_id:
            return []

        conn = self._get_connection()

        try:
            result = conn.execute(
                """
                SELECT he.event_type, he.time_ms, he.xuid,
                       vg.gamertag,
                       he.type_hint
                FROM shared.highlight_events he
                LEFT JOIN shared.v_gamertag_lookup vg ON vg.xuid = he.xuid
                WHERE he.match_id = ?
                ORDER BY he.time_ms ASC NULLS LAST
                """,
                [match_id],
            )
            columns = ["event_type", "time_ms", "xuid", "gamertag", "type_hint"]
            return [dict(zip(columns, row, strict=False)) for row in result.fetchall()]
        except Exception as e:
            logger.debug("Erreur load_highlight_events shared: %s", e)
            return []
