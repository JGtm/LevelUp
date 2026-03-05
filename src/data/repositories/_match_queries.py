"""
Mixin pour les requêtes de matchs DuckDB.

Regroupe les méthodes de chargement et de requête des matchs
extraites de DuckDBRepository. Utilise des helpers DRY pour
éviter la duplication du SELECT 26 colonnes et du constructeur MatchRow.

Sous-modules :
- _match_queries_helpers  : QueryContext, build_match_select, result_to_match_rows
- _match_queries_polars   : load_matches_as_polars, load_match_stats_as_polars
"""

from __future__ import annotations

import logging
from datetime import datetime
from typing import Any

from src.data.domain.models.stats import MatchRow
from src.data.repositories._match_queries_helpers import (
    build_match_select,
    resolve_query_context,
    result_to_match_rows,
)
from src.data.repositories._match_queries_polars import _MatchQueriesPolarsMixin

logger = logging.getLogger(__name__)


class MatchQueriesMixin(_MatchQueriesPolarsMixin):
    """Mixin fournissant les méthodes de requête de matchs pour DuckDBRepository."""

    # =========================================================================
    # Source de données matchs (v5 shared / v4 local)
    # =========================================================================

    def _get_match_table_name(self, conn) -> str | None:
        """Détecte la table de matchs disponible (avec cache).

        Returns:
            Nom de la table "match_stats" si elle existe localement, None sinon (v5.1+).
        """
        cache_key = "local_table:match_stats"
        if hasattr(self, "_table_cache") and cache_key in self._table_cache:
            return "match_stats" if self._table_cache[cache_key] else None

        try:
            result = conn.execute(
                "SELECT COUNT(*) FROM information_schema.tables "
                "WHERE table_schema = 'main' AND table_name = 'match_stats'"
            ).fetchone()
            if result and result[0] > 0:
                if hasattr(self, "_table_cache"):
                    self._table_cache[cache_key] = True
                return "match_stats"
        except Exception:
            pass

        if hasattr(self, "_table_cache"):
            self._table_cache[cache_key] = False
        return None

    def _get_match_source(self, conn) -> tuple[str, list[str], bool]:  # noqa: C901
        """Retourne l'expression FROM pour les matchs (v5 shared ou v4 local).

        Returns:
            Tuple (from_expression, params, uses_mv).
        """
        # Forcer mode local si XUID vide ou None (DBs v3/legacy)
        if not self._xuid or self._xuid.strip() == "":
            match_table = self._get_match_table_name(conn)
            if match_table is None:
                raise RuntimeError(
                    "Table match_stats absente (v5.1+) et XUID vide. "
                    "Impossible de charger les matchs."
                )
            logger.debug("Mode v4/v3 (XUID vide) : requête via table locale %s", match_table)
            if match_table == "match_stats":
                return match_table, [], False
            return f"{match_table} AS match_stats", [], False

        # Forcer mode local si tables shared absentes
        if not (
            self.has_shared
            and self._has_shared_table("match_registry")
            and self._has_shared_table("match_participants")
        ):
            match_table = self._get_match_table_name(conn)
            if match_table is None:
                raise RuntimeError(
                    "shared_matches.duckdb indisponible et table locale match_stats absente. "
                    "Fermez les scripts en cours puis relancez l'app."
                )
            logger.debug("Mode v4/v3 : requête via table locale %s", match_table)
            if match_table == "match_stats":
                return match_table, [], False
            return f"{match_table} AS match_stats", [], False

        # Mode v5.1 : vue mv_player_matches (optimisé)
        if self._has_shared_view("mv_player_matches"):
            source = """(SELECT
            match_id, start_time, map_id, map_name,
            playlist_id, playlist_name, pair_id, pair_name,
            game_variant_id, game_variant_name, outcome, team_id,
            kda, max_killing_spree, headshot_kills, avg_life_seconds,
            time_played_seconds, kills, deaths, assists, accuracy,
            my_team_score, enemy_team_score,
            team_mmr, enemy_mmr,
            personal_score, is_firefight, is_ranked
        FROM shared.mv_player_matches
        WHERE xuid = ?
        ) AS match_stats"""
            logger.debug("Mode v5.1 : requête via vue mv_player_matches")
            return source, [self._xuid], True

        raise RuntimeError(
            "Vue mv_player_matches non trouvée dans shared_matches.duckdb. "
            "Exécutez 'python scripts/rebuild_shared_views.py' pour créer les vues."
        )

    # =========================================================================
    # Chargement des matchs
    # =========================================================================

    def load_matches(  # noqa: PLR0912, PLR0913
        self,
        *,
        playlist_filter: str | None = None,
        map_mode_pair_filter: str | None = None,
        map_filter: str | None = None,
        game_variant_filter: str | None = None,
        include_firefight: bool = True,
        limit: int | None = None,
        offset: int | None = None,
    ) -> list[MatchRow]:
        """Charge tous les matchs (v5 shared ou v4 local)."""
        conn = self._get_connection()
        ctx = resolve_query_context(self, conn)
        select_cols = build_match_select(ctx)

        where_clauses: list[str] = []
        params: list[Any] = []
        if playlist_filter:
            where_clauses.append("playlist_id = ?")
            params.append(playlist_filter)
        if map_mode_pair_filter:
            where_clauses.append("pair_id = ?")
            params.append(map_mode_pair_filter)
        if map_filter:
            where_clauses.append("map_id = ?")
            params.append(map_filter)
        if game_variant_filter:
            where_clauses.append("game_variant_id = ?")
            params.append(game_variant_filter)
        if not include_firefight:
            where_clauses.append("is_firefight = FALSE")
        where_sql = " AND ".join(where_clauses) if where_clauses else "1=1"

        pagination_sql = ""
        if limit is not None:
            pagination_sql += f" LIMIT {int(limit)}"
        if offset is not None:
            pagination_sql += f" OFFSET {int(offset)}"

        all_params = ctx.source_params + params

        sql = f"""
            SELECT {select_cols}
            FROM {ctx.source_sql}{ctx.metadata_joins}{ctx.pms_join}
            WHERE {where_sql}
            ORDER BY match_stats.start_time ASC
            {pagination_sql}
        """

        try:
            result = conn.execute(sql, all_params) if all_params else conn.execute(sql)
        except Exception as e:
            logger.warning("Erreur requête avec jointures: %s. Fallback.", e)
            fallback_cols = build_match_select(ctx, direct_names=True)
            sql_fb = f"""
                SELECT {fallback_cols}
                FROM {ctx.source_sql}{ctx.pms_join}
                WHERE {where_sql}
                ORDER BY match_stats.start_time ASC
                {pagination_sql}
            """
            result = conn.execute(sql_fb, all_params) if all_params else conn.execute(sql_fb)

        return result_to_match_rows(result)

    def load_matches_in_range(
        self,
        start_date: datetime,
        end_date: datetime,
    ) -> list[MatchRow]:
        """Charge les matchs dans une plage de dates."""
        conn = self._get_connection()
        ctx = resolve_query_context(self, conn)
        select_cols = build_match_select(ctx)

        sql = f"""
            SELECT {select_cols}
            FROM {ctx.source_sql}{ctx.metadata_joins}{ctx.pms_join}
            WHERE match_stats.start_time >= ? AND match_stats.start_time <= ?
            ORDER BY match_stats.start_time ASC
        """
        all_params = ctx.source_params + [start_date, end_date]
        result = conn.execute(sql, all_params)
        return result_to_match_rows(result)

    def get_match_count(self) -> int:
        """Retourne le nombre total de matchs."""
        conn = self._get_connection()
        if (
            self.has_shared
            and self._has_shared_table("match_registry")
            and self._has_shared_table("match_participants")
        ):
            result = conn.execute(
                "SELECT COUNT(*) FROM shared.match_participants WHERE xuid = ?",
                [self._xuid],
            ).fetchone()
        else:
            result = conn.execute("SELECT COUNT(*) FROM match_stats").fetchone()
        return result[0] if result else 0

    # =========================================================================
    # Lazy Loading et Pagination (Sprint 4.3)
    # =========================================================================

    def load_recent_matches(
        self,
        limit: int = 50,
        *,
        include_firefight: bool = True,
    ) -> list[MatchRow]:
        """Charge les N matchs les plus récents."""
        conn = self._get_connection()
        ctx = resolve_query_context(self, conn)
        select_cols = build_match_select(ctx)

        where_clauses: list[str] = []
        if not include_firefight:
            where_clauses.append("match_stats.is_firefight = FALSE")
        where_sql = " AND ".join(where_clauses) if where_clauses else "1=1"

        sql = f"""
            SELECT {select_cols}
            FROM {ctx.source_sql}{ctx.metadata_joins}{ctx.pms_join}
            WHERE {where_sql}
            ORDER BY match_stats.start_time DESC
            LIMIT {int(limit)}
        """
        result = conn.execute(sql, ctx.source_params) if ctx.source_params else conn.execute(sql)
        return result_to_match_rows(result)

    def load_matches_paginated(
        self,
        page: int = 1,
        page_size: int = 50,
        *,
        order_desc: bool = True,
        include_firefight: bool = True,
    ) -> tuple[list[MatchRow], int]:
        """Charge les matchs avec pagination.

        Returns:
            Tuple (matchs, total_pages).
        """
        total_count = self.get_match_count()
        total_pages = (total_count + page_size - 1) // page_size if total_count > 0 else 1
        page = max(1, min(page, total_pages))
        offset = (page - 1) * page_size

        conn = self._get_connection()
        ctx = resolve_query_context(self, conn)
        select_cols = build_match_select(ctx)

        where_clauses: list[str] = []
        if not include_firefight:
            where_clauses.append("match_stats.is_firefight = FALSE")
        where_sql = " AND ".join(where_clauses) if where_clauses else "1=1"
        order_dir = "DESC" if order_desc else "ASC"

        sql = f"""
            SELECT {select_cols}
            FROM {ctx.source_sql}{ctx.metadata_joins}{ctx.pms_join}
            WHERE {where_sql}
            ORDER BY match_stats.start_time {order_dir}
            LIMIT {int(page_size)} OFFSET {int(offset)}
        """
        result = conn.execute(sql, ctx.source_params) if ctx.source_params else conn.execute(sql)
        return result_to_match_rows(result), total_pages

    # =========================================================================
    # Chargement batch MMR (Sprint 4.2)
    # =========================================================================

    def load_match_mmr_batch(
        self, match_ids: list[str]
    ) -> dict[str, tuple[float | None, float | None]]:
        """Charge le MMR pour plusieurs matchs depuis shared ou local."""
        if not match_ids:
            return {}

        conn = self._get_connection()
        placeholders = ", ".join(["?" for _ in match_ids])

        if self._has_shared_table("match_participants"):
            try:
                result = conn.execute(
                    f"""
                    SELECT match_id, team_mmr, enemy_mmr
                    FROM shared.match_participants
                    WHERE match_id IN ({placeholders})
                      AND xuid = ?
                    """,
                    [*match_ids, self._xuid],
                )
                shared_result = {row[0]: (row[1], row[2]) for row in result.fetchall()}
                if shared_result:
                    return shared_result
            except Exception:
                pass

        try:
            has_pms = self._has_table_cached(conn, "player_match_stats")
            if has_pms:
                result = conn.execute(
                    f"""
                    SELECT ms.match_id,
                           COALESCE(ms.team_mmr, pms.team_mmr) as team_mmr,
                           COALESCE(ms.enemy_mmr, pms.enemy_mmr) as enemy_mmr
                    FROM match_stats ms
                    LEFT JOIN player_match_stats pms ON ms.match_id = pms.match_id
                    WHERE ms.match_id IN ({placeholders})
                    """,
                    match_ids,
                )
            else:
                result = conn.execute(
                    f"""
                    SELECT match_id, team_mmr, enemy_mmr
                    FROM match_stats
                    WHERE match_id IN ({placeholders})
                    """,
                    match_ids,
                )
            return {row[0]: (row[1], row[2]) for row in result.fetchall()}
        except Exception:
            return {}

    def load_match_skill_data(self, match_id: str) -> dict[str, Any] | None:
        """Charge team_mmr, enemy_mmr et kills/deaths/assists expected/stddev."""
        conn = self._get_connection()

        if not self._has_shared_table("match_participants"):
            return None

        try:
            row = conn.execute(
                """
                SELECT team_mmr, enemy_mmr,
                       kills, kills_expected, kills_stddev,
                       deaths, deaths_expected, deaths_stddev,
                       assists, assists_expected, assists_stddev,
                       team_id
                FROM shared.match_participants
                WHERE match_id = ? AND xuid = ?
                """,
                [match_id, self._xuid],
            ).fetchone()
        except Exception:
            return None

        if not row:
            return None

        team_mmr, enemy_mmr, team_id = row[0], row[1], row[11]

        # Fallback MMR depuis coéquipier si absent pour ce joueur
        if (team_mmr is None or enemy_mmr is None) and team_id is not None:
            team_mmr, enemy_mmr = self._fallback_mmr_from_teammate(
                conn, match_id, team_id, team_mmr, enemy_mmr
            )

        return {
            "team_id": team_id,
            "team_mmr": team_mmr,
            "enemy_mmr": enemy_mmr,
            "kills": {"count": row[2], "expected": row[3], "stddev": row[4]},
            "deaths": {"count": row[5], "expected": row[6], "stddev": row[7]},
            "assists": {"count": row[8], "expected": row[9], "stddev": row[10]},
        }

    @staticmethod
    def _fallback_mmr_from_teammate(
        conn: Any,
        match_id: str,
        team_id: int,
        team_mmr: float | None,
        enemy_mmr: float | None,
    ) -> tuple[float | None, float | None]:
        """Récupère le MMR depuis un coéquipier si absent pour le joueur."""
        try:
            teammate_row = conn.execute(
                """
                SELECT team_mmr, enemy_mmr
                FROM shared.match_participants
                WHERE match_id = ? AND team_id = ?
                  AND team_mmr IS NOT NULL AND enemy_mmr IS NOT NULL
                LIMIT 1
                """,
                [match_id, team_id],
            ).fetchone()
            if teammate_row:
                return teammate_row[0], teammate_row[1]
        except Exception:
            pass
        return team_mmr, enemy_mmr
