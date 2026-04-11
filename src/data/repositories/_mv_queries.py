"""Mixin – lectures des vues matérialisées player (stats.duckdb).

Extrait de _materialized_views.py (H9 post-v7 housekeeping).
Contient les méthodes de lecture (get_map_stats, get_session_stats, etc.).
Les méthodes de write/refresh restent dans _materialized_views.py.
"""

from __future__ import annotations

import logging

logger = logging.getLogger(__name__)


class MaterializedViewsQueryMixin:
    """Méthodes de lecture des vues matérialisées."""

    def get_map_stats(self, min_matches: int = 1) -> list[dict]:
        """Récupère les stats par carte depuis la vue matérialisée.

        Args:
            min_matches: Nombre minimum de matchs pour inclure une carte.

        Returns:
            Liste de dicts avec les stats par carte.
        """
        conn = self._get_connection()
        try:
            result = conn.execute(
                """
                SELECT
                    map_id, map_name, matches_played, wins, losses, ties,
                    avg_kills, avg_deaths, avg_assists, avg_accuracy,
                    avg_kda, win_rate, updated_at
                FROM mv_map_stats
                WHERE matches_played >= ?
                ORDER BY matches_played DESC
                """,
                [min_matches],
            )
            columns = [desc[0] for desc in result.description]
            return [dict(zip(columns, row, strict=False)) for row in result.fetchall()]
        except Exception:
            return []

    def get_mode_category_stats(self) -> list[dict]:
        """Récupère les stats par catégorie de mode depuis la vue matérialisée.

        Returns:
            Liste de dicts avec les stats par catégorie.
        """
        conn = self._get_connection()
        try:
            result = conn.execute(
                """
                SELECT
                    mode_category, matches_played, avg_kills, avg_deaths,
                    avg_assists, avg_kda, avg_accuracy, win_rate, updated_at
                FROM mv_mode_category_stats
                ORDER BY matches_played DESC
                """
            )
            columns = [desc[0] for desc in result.description]
            return [dict(zip(columns, row, strict=False)) for row in result.fetchall()]
        except Exception:
            return []

    def get_global_stats(self) -> dict[str, float]:
        """Récupère les stats globales depuis la vue matérialisée.

        Returns:
            Dict avec stat_key -> stat_value.
        """
        conn = self._get_connection()
        try:
            result = conn.execute("SELECT stat_key, stat_value FROM mv_global_stats")
            return {row[0]: row[1] for row in result.fetchall()}
        except Exception:
            return {}

    def get_session_stats(self, limit: int = 50) -> list[dict]:
        """Récupère les stats par session depuis la vue matérialisée.

        Args:
            limit: Nombre maximum de sessions à retourner.

        Returns:
            Liste de dicts avec les stats par session.
        """
        conn = self._get_connection()
        try:
            result = conn.execute(
                """
                SELECT
                    session_id, match_count, start_time, end_time,
                    total_kills, total_deaths, total_assists,
                    kd_ratio, win_rate, avg_accuracy, avg_life_seconds, updated_at
                FROM mv_session_stats
                ORDER BY start_time DESC
                LIMIT ?
                """,
                [limit],
            )
            columns = [desc[0] for desc in result.description]
            return [dict(zip(columns, row, strict=False)) for row in result.fetchall()]
        except Exception:
            return []

    def has_materialized_views(self) -> bool:
        """Vérifie si les vues matérialisées sont disponibles et remplies.

        Returns:
            True si au moins mv_global_stats contient des données.
        """
        conn = self._get_connection()
        try:
            count = conn.execute("SELECT COUNT(*) FROM mv_global_stats").fetchone()[0]
            return count > 0
        except Exception:
            return False
