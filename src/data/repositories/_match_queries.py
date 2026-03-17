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

    def _get_match_source(self, _conn) -> tuple[str, list[str], bool]:
        """Retourne l'expression FROM pour les matchs (v6 shared uniquement).

        La vue mv_player_matches dans shared_matches.duckdb est garantie présente
        en architecture v6. L'utilisation de tables locales match_stats (v4/v3)
        n'est plus supportée depuis v5.1.

        Returns:
            Tuple (from_expression, params, uses_mv).
        """
        if not self._xuid or self._xuid.strip() == "":
            raise RuntimeError(
                "XUID manquant. Impossible de charger les matchs (architecture v6 requiert un XUID)."
            )
        if not self.has_shared:
            raise RuntimeError(
                "shared_matches.duckdb indisponible. "
                "Fermez les scripts en cours puis relancez l'app."
            )
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
        logger.debug("Mode v6 : requête via vue mv_player_matches")
        return source, [self._xuid], True

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
        try:
            result = conn.execute(
                "SELECT COUNT(*) FROM shared.match_participants WHERE xuid = ?",
                [self._xuid],
            ).fetchone()
        except Exception:
            logger.debug("get_match_count: erreur shared.match_participants", exc_info=True)
            return 0
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
                logger.warning(
                    "load_match_mmr_batch: échec requête shared.match_participants", exc_info=True
                )

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
            logger.warning(
                "load_match_skill_data: erreur requête shared.match_participants (match_id=%s)",
                match_id,
                exc_info=True,
            )
            return None

        if not row:
            return None

        team_mmr, enemy_mmr, team_id = row[0], row[1], row[11]

        # Fallback MMR depuis coéquipier si absent pour ce joueur
        if (team_mmr is None or enemy_mmr is None) and team_id is not None:
            logger.debug(
                "load_match_skill_data: MMR NULL pour xuid=%s match_id=%s — tentative fallback coéquipier",
                self._xuid,
                match_id,
            )
            team_mmr, enemy_mmr = self._fallback_mmr_from_teammate(
                conn, match_id, team_id, team_mmr, enemy_mmr
            )

        if team_mmr is None or enemy_mmr is None:
            logger.warning(
                "load_match_skill_data: MMR définitivement absent pour xuid=%s match_id=%s "
                "(team_mmr=%s enemy_mmr=%s) — l'API Halo n'a peut-être pas de données pour ce match",
                self._xuid,
                match_id,
                team_mmr,
                enemy_mmr,
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
                logger.debug(
                    "_fallback_mmr_from_teammate: MMR récupéré depuis coéquipier (match_id=%s team_id=%s)",
                    match_id,
                    team_id,
                )
                return teammate_row[0], teammate_row[1]
        except Exception:
            logger.warning(
                "_fallback_mmr_from_teammate: erreur requête (match_id=%s team_id=%s)",
                match_id,
                team_id,
                exc_info=True,
            )
        return team_mmr, enemy_mmr

    # =========================================================================
    # Enrichissement joueur + détection match abandonné
    # =========================================================================

    def load_player_match_enrichment(self, match_id: str) -> tuple[bool, float | None, int | None]:
        """Charge had_bot_teammate, performance_score et dominance_flag.

        Lit depuis la table ``player_match_enrichment`` de la DB joueur.

        Returns:
            Tuple (had_bot_teammate, performance_score, dominance_flag).
        """
        had_bot = False
        stored_perf: float | None = None
        dominance_flag: int | None = None
        try:
            conn = self._get_connection()
            pme_row = conn.execute(
                "SELECT had_bot_teammate, performance_score, dominance_flag"
                " FROM player_match_enrichment WHERE match_id = ? LIMIT 1",
                [match_id],
            ).fetchone()
            if pme_row:
                had_bot = bool(pme_row[0])
                if pme_row[1] is not None:
                    stored_perf = float(pme_row[1])
                if pme_row[2] is not None:
                    dominance_flag = int(pme_row[2])
        except Exception:
            logger.debug(
                "load_player_match_enrichment: introuvable match=%s", match_id, exc_info=True
            )
        return had_bot, stored_perf, dominance_flag

    def is_abandoned_match(self, match_id: str) -> bool:
        """Retourne True si tous les participants du match ont kills=deaths=score=0.

        Interroge ``shared.match_participants`` pour détecter un match terminé
        sans qu'aucun joueur n'ait enregistré de points (crash serveur, partie
        non disputée).
        """
        if not self._has_shared_table("match_participants"):
            return False
        try:
            conn = self._get_connection()
            result = conn.execute(
                "SELECT COUNT(*) AS n,"
                " COALESCE(SUM(kills), 0) AS k,"
                " COALESCE(SUM(deaths), 0) AS d,"
                " COALESCE(SUM(score), 0) AS s"
                " FROM shared.match_participants WHERE match_id = ?",
                [match_id],
            ).fetchone()
            if result and result[0] > 0:
                abandoned = result[1] == 0 and result[2] == 0 and result[3] == 0
                if abandoned:
                    logger.debug(
                        "is_abandoned_match: match=%s abandonné (%d participants, k=%s d=%s s=%s)",
                        match_id,
                        result[0],
                        result[1],
                        result[2],
                        result[3],
                    )
                return abandoned
        except Exception:
            logger.debug("is_abandoned_match: erreur pour match=%s", match_id, exc_info=True)
        return False
