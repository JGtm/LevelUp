"""
Mixin pour le chargement des médailles depuis shared_matches.duckdb.

Regroupe les méthodes de requête de médailles extraites de DuckDBRepository :
- load_top_medals
- load_match_medals
- count_medal_by_match
- count_perfect_kills_by_match
"""

from __future__ import annotations

import logging

logger = logging.getLogger(__name__)


class MedalsMixin:
    """Méthodes DuckDBRepository relatives aux médailles."""

    def load_top_medals(
        self,
        match_ids: list[str],
        *,
        top_n: int | None = 25,
    ) -> list[tuple[int, int]]:
        """Charge les médailles les plus fréquentes depuis shared.medals_earned.

        Args:
            match_ids: Liste des IDs de matchs.
            top_n: Nombre max de médailles à retourner.

        Returns:
            Liste de (medal_name_id, total_count) triée par fréquence décroissante.
        """
        if not match_ids:
            return []

        conn = self._get_connection()
        placeholders = ", ".join(["?" for _ in match_ids])
        limit_sql = f"LIMIT {top_n}" if top_n else ""

        try:
            sql = f"""
                SELECT medal_name_id, SUM(count) as total
                FROM shared.medals_earned
                WHERE match_id IN ({placeholders})
                  AND xuid = ?
                GROUP BY medal_name_id
                ORDER BY total DESC
                {limit_sql}
            """
            result = conn.execute(sql, [*match_ids, self._xuid])
            return [(row[0], row[1]) for row in result.fetchall()]
        except Exception as e:
            logger.debug("Erreur load_top_medals shared: %s", e)
            return []

    def load_match_medals(self, match_id: str) -> list[dict[str, int]]:
        """Charge les médailles pour un match spécifique depuis shared.

        Args:
            match_id: ID du match.

        Returns:
            Liste de dicts {name_id, count}.
        """
        conn = self._get_connection()

        try:
            result = conn.execute(
                "SELECT medal_name_id, count FROM shared.medals_earned WHERE match_id = ? AND xuid = ?",
                [match_id, self._xuid],
            )
            return [{"name_id": row[0], "count": row[1]} for row in result.fetchall()]
        except Exception as e:
            logger.debug("Erreur load_match_medals shared: %s", e)
            return []

    def count_medal_by_match(
        self,
        match_ids: list[str],
        medal_name_id: int,
    ) -> dict[str, int]:
        """Compte une médaille spécifique pour chaque match depuis shared.

        Args:
            match_ids: Liste des IDs de matchs.
            medal_name_id: ID de la médaille à compter (ex: 1512363953 pour Perfect).

        Returns:
            Dict {match_id: count} pour les matchs ayant cette médaille.
        """
        if not match_ids:
            return {}

        conn = self._get_connection()
        placeholders = ", ".join(["?" for _ in match_ids])

        try:
            result = conn.execute(
                f"""
                SELECT match_id, count
                FROM shared.medals_earned
                WHERE match_id IN ({placeholders})
                  AND medal_name_id = ?
                  AND xuid = ?
                """,
                [*match_ids, medal_name_id, self._xuid],
            )
            return {str(row[0]): row[1] for row in result.fetchall()}
        except Exception as e:
            logger.debug("Erreur count_medal_by_match shared: %s", e)
            return {}

    def count_perfect_kills_by_match(
        self,
        match_ids: list[str],
    ) -> dict[str, int]:
        """Compte les médailles 'Perfect' (kills parfaits) par match.

        La médaille 'Perfect' (ID 1512363953) est obtenue quand le joueur
        tue un adversaire sans prendre de dégâts.

        Args:
            match_ids: Liste des IDs de matchs.

        Returns:
            Dict {match_id: perfect_count} pour les matchs avec des Perfect.
        """
        return self.count_medal_by_match(match_ids, medal_name_id=1512363953)
