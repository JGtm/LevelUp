"""
Mixin pour les requêtes de matchs DuckDB.

Regroupe les méthodes de chargement et de requête des matchs
extraites de DuckDBRepository :
- load_matches
- load_matches_in_range
- get_match_count
- load_recent_matches
- load_matches_paginated
- load_match_mmr_batch
- load_match_stats_as_polars
"""

from __future__ import annotations

import logging
from datetime import datetime
from typing import TYPE_CHECKING, Any

import polars as pl

from src.data.domain.models.stats import MatchRow
from src.data.repositories._arrow_bridge import result_to_polars

if TYPE_CHECKING:
    pass

logger = logging.getLogger(__name__)


class MatchQueriesMixin:
    """Mixin fournissant les méthodes de requête de matchs pour DuckDBRepository."""

    # =========================================================================
    # Source de données matchs (v5 shared / v4 local)
    # =========================================================================

    def _get_match_table_name(self, conn) -> str | None:
        """Détecte la table de matchs disponible (avec cache).

        Returns:
            Nom de la table "match_stats" si elle existe localement, None sinon (v5.1+).
        """
        # Cache au niveau instance — évite les requêtes information_schema répétées
        cache_key = "local_table:match_stats"
        if hasattr(self, "_table_cache") and cache_key in self._table_cache:
            return "match_stats" if self._table_cache[cache_key] else None

        # Essayer match_stats (v4) en premier
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

        # Table locale match_stats absente (v5.1+ : tables legacy supprimées)
        if hasattr(self, "_table_cache"):
            self._table_cache[cache_key] = False
        return None

    def _get_match_source(self, conn) -> tuple[str, list[str], bool]:
        """Retourne l'expression FROM pour les matchs (v5 shared ou v4 local).

        En mode v5, utilise la vue mv_player_matches si disponible (v5.1),
        sinon construit une sous-requête combinant shared.match_registry,
        shared.match_participants et un LEFT JOIN vers match_stats local.

        En mode v4/v3, retourne le nom de la table locale directe.

        Returns:
            Tuple (from_expression, params, uses_mv).
            uses_mv est True quand mv_player_matches est utilisée (noms déjà résolus,
            pas besoin de jointures metadata/MMR supplémentaires).
        """
        # Forcer mode local si XUID vide ou None (DBs v3/legacy)
        if not self._xuid or self._xuid.strip() == "":
            match_table = self._get_match_table_name(conn)
            if match_table is None:
                raise RuntimeError(
                    "Table match_stats absente (v5.1+) et XUID vide. "
                    "Impossible de charger les matchs."
                )
            logger.debug(f"Mode v4/v3 (XUID vide) : requête via table locale {match_table}")
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
                    "shared_matches.duckdb indisponible (fichier verrouillé par un autre "
                    "processus ?) et table locale match_stats absente (v5.1+). "
                    "Fermez les scripts en cours (backfill, sync) puis relancez l'app."
                )
            logger.debug(f"Mode v4/v3 : requête via table locale {match_table}")
            if match_table == "match_stats":
                return match_table, [], False
            return f"{match_table} AS match_stats", [], False

        # ── Mode v5.1 : vue mv_player_matches (optimisé) ──
        # 8bis.A5 : Simplification post-étape 8 (tables legacy supprimées)
        # Plus de LEFT JOIN vers match_stats — toutes les données sont dans shared
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

        # ── Erreur v5.1 : vue mv_player_matches requise ──
        # Post-étape 8, les tables legacy sont supprimées — pas de fallback
        raise RuntimeError(
            "Vue mv_player_matches non trouvée dans shared_matches.duckdb. "
            "Exécutez 'python scripts/rebuild_shared_views.py' pour créer les vues."
        )

    # =========================================================================
    # Chargement des matchs
    # =========================================================================

    def load_matches(
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
        """
        Charge tous les matchs (v5 : shared + local, v4 : local uniquement).
        """
        conn = self._get_connection()

        # Source v5 (shared) ou v4 (locale)
        source_sql, source_params, uses_mv = self._get_match_source(conn)
        is_shared = bool(source_params)

        where_clauses = []
        params: list = []

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

        # Construire les clauses LIMIT/OFFSET pour la pagination (Sprint 4.3)
        pagination_sql = ""
        if limit is not None:
            pagination_sql += f" LIMIT {int(limit)}"
        if offset is not None:
            pagination_sql += f" OFFSET {int(offset)}"

        # v5.1 perf — skip jointures redondantes quand mv_player_matches est utilisée
        if uses_mv:
            metadata_joins = ""
            map_name_expr = "match_stats.map_name"
            playlist_name_expr = "match_stats.playlist_name"
            pair_name_expr = "match_stats.pair_name"
            pms_join = ""
            team_mmr_expr = "match_stats.team_mmr"
            enemy_mmr_expr = "match_stats.enemy_mmr"
        else:
            # Résoudre les métadonnées depuis meta.* si disponible
            metadata_joins, map_name_expr, playlist_name_expr, pair_name_expr = (
                self._build_metadata_resolution(conn)
            )
            # Fallback MMR depuis player_match_stats si disponible
            pms_join, team_mmr_expr, enemy_mmr_expr = self._build_mmr_fallback(conn)

        # En mode shared, personal_score est toujours dans la sous-requête
        if is_shared:
            personal_score_select = "match_stats.personal_score"
        else:
            personal_score_select = self._select_optional_column(
                conn,
                table_name="match_stats",
                table_alias="match_stats",
                column_name="personal_score",
            )

        sql = f"""
            SELECT
                match_stats.match_id,
                match_stats.start_time,
                match_stats.map_id,
                {map_name_expr} as map_name,
                match_stats.playlist_id,
                {playlist_name_expr} as playlist_name,
                match_stats.pair_id,
                {pair_name_expr} as pair_name,
                match_stats.game_variant_id,
                match_stats.game_variant_name,
                match_stats.outcome,
                match_stats.team_id,
                match_stats.kda,
                match_stats.max_killing_spree,
                match_stats.headshot_kills,
                match_stats.avg_life_seconds,
                match_stats.time_played_seconds,
                match_stats.kills,
                match_stats.deaths,
                match_stats.assists,
                match_stats.accuracy,
                match_stats.my_team_score,
                match_stats.enemy_team_score,
                {team_mmr_expr} as team_mmr,
                {enemy_mmr_expr} as enemy_mmr,
                {personal_score_select}
            FROM {source_sql}{metadata_joins}{pms_join}
            WHERE {where_sql}
            ORDER BY match_stats.start_time ASC
            {pagination_sql}
        """

        all_params = source_params + params

        # Log de debug pour diagnostiquer les problèmes de requête
        if logger.isEnabledFor(logging.DEBUG):
            logger.debug(f"Requête SQL générée: {sql[:500]}...")
            logger.debug(f"Jointures métadonnées: {metadata_joins}")
            logger.debug(
                f"Expressions: map={map_name_expr}, playlist={playlist_name_expr}, pair={pair_name_expr}"
            )

        try:
            result = conn.execute(sql, all_params) if all_params else conn.execute(sql)
        except Exception as e:
            # Si la requête avec jointures échoue, essayer sans jointures
            logger.warning(
                f"Erreur requête avec jointures métadonnées: {e}. Fallback sans jointures."
            )
            logger.debug(f"Requête SQL qui a échoué: {sql}")
            # Fallback sans jointures métadonnées mais avec jointure MMR si possible
            sql_fallback = f"""
                SELECT
                    match_stats.match_id,
                    match_stats.start_time,
                    match_stats.map_id,
                    match_stats.map_name,
                    match_stats.playlist_id,
                    match_stats.playlist_name,
                    match_stats.pair_id,
                    match_stats.pair_name,
                    match_stats.game_variant_id,
                    match_stats.game_variant_name,
                    match_stats.outcome,
                    match_stats.team_id,
                    match_stats.kda,
                    match_stats.max_killing_spree,
                    match_stats.headshot_kills,
                    match_stats.avg_life_seconds,
                    match_stats.time_played_seconds,
                    match_stats.kills,
                    match_stats.deaths,
                    match_stats.assists,
                    match_stats.accuracy,
                    match_stats.my_team_score,
                    match_stats.enemy_team_score,
                    {team_mmr_expr} as team_mmr,
                    {enemy_mmr_expr} as enemy_mmr,
                    {personal_score_select}
                FROM {source_sql}{pms_join}
                WHERE {where_sql}
                ORDER BY match_stats.start_time ASC
                {pagination_sql}
            """
            result = (
                conn.execute(sql_fallback, all_params) if all_params else conn.execute(sql_fallback)
            )
        rows = result.fetchall()
        columns = [desc[0] for desc in result.description]

        return [
            MatchRow(
                match_id=row[columns.index("match_id")],
                start_time=row[columns.index("start_time")],
                map_id=row[columns.index("map_id")],
                map_name=row[columns.index("map_name")],
                playlist_id=row[columns.index("playlist_id")],
                playlist_name=row[columns.index("playlist_name")],
                map_mode_pair_id=row[columns.index("pair_id")],
                map_mode_pair_name=row[columns.index("pair_name")],
                game_variant_id=row[columns.index("game_variant_id")],
                game_variant_name=row[columns.index("game_variant_name")],
                outcome=row[columns.index("outcome")],
                last_team_id=row[columns.index("team_id")],
                kda=row[columns.index("kda")],
                max_killing_spree=row[columns.index("max_killing_spree")],
                headshot_kills=row[columns.index("headshot_kills")],
                average_life_seconds=row[columns.index("avg_life_seconds")],
                time_played_seconds=row[columns.index("time_played_seconds")],
                kills=row[columns.index("kills")] or 0,
                deaths=row[columns.index("deaths")] or 0,
                assists=row[columns.index("assists")] or 0,
                accuracy=row[columns.index("accuracy")],
                my_team_score=row[columns.index("my_team_score")],
                enemy_team_score=row[columns.index("enemy_team_score")],
                team_mmr=row[columns.index("team_mmr")],
                enemy_mmr=row[columns.index("enemy_mmr")],
                personal_score=row[columns.index("personal_score")]
                if "personal_score" in columns
                else None,
            )
            for row in rows
        ]

    def load_matches_in_range(
        self,
        start_date: datetime,
        end_date: datetime,
    ) -> list[MatchRow]:
        """Charge les matchs dans une plage de dates."""
        conn = self._get_connection()

        # Source v5 (shared) ou v4 (locale)
        source_sql, source_params, uses_mv = self._get_match_source(conn)
        is_shared = bool(source_params)

        # v5.1 perf — skip jointures redondantes quand mv_player_matches est utilisée
        if uses_mv:
            metadata_joins = ""
            map_name_expr = "match_stats.map_name"
            playlist_name_expr = "match_stats.playlist_name"
            pair_name_expr = "match_stats.pair_name"
            pms_join = ""
            team_mmr_expr = "match_stats.team_mmr"
            enemy_mmr_expr = "match_stats.enemy_mmr"
        else:
            # Résoudre les métadonnées depuis meta.* si disponible
            metadata_joins, map_name_expr, playlist_name_expr, pair_name_expr = (
                self._build_metadata_resolution(conn)
            )
            # Fallback MMR depuis player_match_stats si disponible
            pms_join, team_mmr_expr, enemy_mmr_expr = self._build_mmr_fallback(conn)

        if is_shared:
            personal_score_select = "match_stats.personal_score"
        else:
            personal_score_select = self._select_optional_column(
                conn,
                table_name="match_stats",
                table_alias="match_stats",
                column_name="personal_score",
            )

        sql = f"""
            SELECT
                match_stats.match_id,
                match_stats.start_time,
                match_stats.map_id,
                {map_name_expr} as map_name,
                match_stats.playlist_id,
                {playlist_name_expr} as playlist_name,
                match_stats.pair_id,
                {pair_name_expr} as pair_name,
                match_stats.game_variant_id,
                match_stats.game_variant_name,
                match_stats.outcome,
                match_stats.team_id,
                match_stats.kda,
                match_stats.max_killing_spree,
                match_stats.headshot_kills,
                match_stats.avg_life_seconds,
                match_stats.time_played_seconds,
                match_stats.kills,
                match_stats.deaths,
                match_stats.assists,
                match_stats.accuracy,
                match_stats.my_team_score,
                match_stats.enemy_team_score,
                {team_mmr_expr} as team_mmr,
                {enemy_mmr_expr} as enemy_mmr,
                {personal_score_select}
            FROM {source_sql}{metadata_joins}{pms_join}
            WHERE match_stats.start_time >= ? AND match_stats.start_time <= ?
            ORDER BY match_stats.start_time ASC
        """

        all_params = source_params + [start_date, end_date]
        result = conn.execute(sql, all_params)
        rows = result.fetchall()
        columns = [desc[0] for desc in result.description]

        return [
            MatchRow(
                match_id=row[columns.index("match_id")],
                start_time=row[columns.index("start_time")],
                map_id=row[columns.index("map_id")],
                map_name=row[columns.index("map_name")],
                playlist_id=row[columns.index("playlist_id")],
                playlist_name=row[columns.index("playlist_name")],
                map_mode_pair_id=row[columns.index("pair_id")],
                map_mode_pair_name=row[columns.index("pair_name")],
                game_variant_id=row[columns.index("game_variant_id")],
                game_variant_name=row[columns.index("game_variant_name")],
                outcome=row[columns.index("outcome")],
                last_team_id=row[columns.index("team_id")],
                kda=row[columns.index("kda")],
                max_killing_spree=row[columns.index("max_killing_spree")],
                headshot_kills=row[columns.index("headshot_kills")],
                average_life_seconds=row[columns.index("avg_life_seconds")],
                time_played_seconds=row[columns.index("time_played_seconds")],
                kills=row[columns.index("kills")] or 0,
                deaths=row[columns.index("deaths")] or 0,
                assists=row[columns.index("assists")] or 0,
                accuracy=row[columns.index("accuracy")],
                my_team_score=row[columns.index("my_team_score")],
                enemy_team_score=row[columns.index("enemy_team_score")],
                team_mmr=row[columns.index("team_mmr")],
                enemy_mmr=row[columns.index("enemy_mmr")],
                personal_score=row[columns.index("personal_score")]
                if "personal_score" in columns
                else None,
            )
            for row in rows
        ]

    def get_match_count(self) -> int:
        """Retourne le nombre total de matchs.

        En mode v5, compte depuis shared.match_participants pour le xuid.
        En mode v4, compte depuis match_stats locale.
        """
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
        """Charge les N matchs les plus récents.

        Optimisé pour le chargement initial rapide de l'UI.
        Tri par start_time DESC (les plus récents en premier).

        Args:
            limit: Nombre maximum de matchs à retourner.
            include_firefight: Inclure les matchs PvE.

        Returns:
            Liste de MatchRow triée par date décroissante.
        """
        conn = self._get_connection()

        # Source v5 (shared) ou v4 (locale)
        source_sql, source_params, uses_mv = self._get_match_source(conn)
        is_shared = bool(source_params)

        # v5.1 perf — skip jointures redondantes quand mv_player_matches est utilisée
        if uses_mv:
            metadata_joins = ""
            map_name_expr = "match_stats.map_name"
            playlist_name_expr = "match_stats.playlist_name"
            pair_name_expr = "match_stats.pair_name"
            pms_join = ""
            team_mmr_expr = "match_stats.team_mmr"
            enemy_mmr_expr = "match_stats.enemy_mmr"
        else:
            # Résoudre les métadonnées depuis meta.* si disponible
            metadata_joins, map_name_expr, playlist_name_expr, pair_name_expr = (
                self._build_metadata_resolution(conn)
            )
            # Fallback MMR depuis player_match_stats si disponible
            pms_join, team_mmr_expr, enemy_mmr_expr = self._build_mmr_fallback(conn)

        if is_shared:
            personal_score_select = "match_stats.personal_score"
        else:
            personal_score_select = self._select_optional_column(
                conn,
                table_name="match_stats",
                table_alias="match_stats",
                column_name="personal_score",
            )

        where_clauses = []
        if not include_firefight:
            where_clauses.append("match_stats.is_firefight = FALSE")

        where_sql = " AND ".join(where_clauses) if where_clauses else "1=1"

        sql = f"""
            SELECT
                match_stats.match_id,
                match_stats.start_time,
                match_stats.map_id,
                {map_name_expr} as map_name,
                match_stats.playlist_id,
                {playlist_name_expr} as playlist_name,
                match_stats.pair_id,
                {pair_name_expr} as pair_name,
                match_stats.game_variant_id,
                match_stats.game_variant_name,
                match_stats.outcome,
                match_stats.team_id,
                match_stats.kda,
                match_stats.max_killing_spree,
                match_stats.headshot_kills,
                match_stats.avg_life_seconds,
                match_stats.time_played_seconds,
                match_stats.kills,
                match_stats.deaths,
                match_stats.assists,
                match_stats.accuracy,
                match_stats.my_team_score,
                match_stats.enemy_team_score,
                {team_mmr_expr} as team_mmr,
                {enemy_mmr_expr} as enemy_mmr,
                {personal_score_select}
            FROM {source_sql}{metadata_joins}{pms_join}
            WHERE {where_sql}
            ORDER BY match_stats.start_time DESC
            LIMIT {int(limit)}
        """

        result = conn.execute(sql, source_params) if source_params else conn.execute(sql)
        rows = result.fetchall()
        columns = [desc[0] for desc in result.description]

        return [
            MatchRow(
                match_id=row[columns.index("match_id")],
                start_time=row[columns.index("start_time")],
                map_id=row[columns.index("map_id")],
                map_name=row[columns.index("map_name")],
                playlist_id=row[columns.index("playlist_id")],
                playlist_name=row[columns.index("playlist_name")],
                map_mode_pair_id=row[columns.index("pair_id")],
                map_mode_pair_name=row[columns.index("pair_name")],
                game_variant_id=row[columns.index("game_variant_id")],
                game_variant_name=row[columns.index("game_variant_name")],
                outcome=row[columns.index("outcome")],
                last_team_id=row[columns.index("team_id")],
                kda=row[columns.index("kda")],
                max_killing_spree=row[columns.index("max_killing_spree")],
                headshot_kills=row[columns.index("headshot_kills")],
                average_life_seconds=row[columns.index("avg_life_seconds")],
                time_played_seconds=row[columns.index("time_played_seconds")],
                kills=row[columns.index("kills")] or 0,
                deaths=row[columns.index("deaths")] or 0,
                assists=row[columns.index("assists")] or 0,
                accuracy=row[columns.index("accuracy")],
                my_team_score=row[columns.index("my_team_score")],
                enemy_team_score=row[columns.index("enemy_team_score")],
                team_mmr=row[columns.index("team_mmr")],
                enemy_mmr=row[columns.index("enemy_mmr")],
                personal_score=row[columns.index("personal_score")]
                if "personal_score" in columns
                else None,
            )
            for row in rows
        ]

    def load_matches_paginated(
        self,
        page: int = 1,
        page_size: int = 50,
        *,
        order_desc: bool = True,
        include_firefight: bool = True,
    ) -> tuple[list[MatchRow], int]:
        """Charge les matchs avec pagination.

        Args:
            page: Numéro de page (1-indexed).
            page_size: Nombre de matchs par page.
            order_desc: Si True, tri décroissant (récents en premier).
            include_firefight: Inclure les matchs PvE.

        Returns:
            Tuple (matchs, total_pages).
        """
        # Calculer le total de pages
        total_count = self.get_match_count()
        total_pages = (total_count + page_size - 1) // page_size if total_count > 0 else 1

        # Valider la page
        page = max(1, min(page, total_pages))
        offset = (page - 1) * page_size

        conn = self._get_connection()

        # Source v5 (shared) ou v4 (locale)
        source_sql, source_params, uses_mv = self._get_match_source(conn)
        is_shared = bool(source_params)

        # v5.1 perf — skip jointures redondantes quand mv_player_matches est utilisée
        if uses_mv:
            metadata_joins = ""
            map_name_expr = "match_stats.map_name"
            playlist_name_expr = "match_stats.playlist_name"
            pair_name_expr = "match_stats.pair_name"
            pms_join = ""
            team_mmr_expr = "match_stats.team_mmr"
            enemy_mmr_expr = "match_stats.enemy_mmr"
        else:
            # Résoudre les métadonnées depuis meta.* si disponible
            metadata_joins, map_name_expr, playlist_name_expr, pair_name_expr = (
                self._build_metadata_resolution(conn)
            )
            # Fallback MMR depuis player_match_stats si disponible
            pms_join, team_mmr_expr, enemy_mmr_expr = self._build_mmr_fallback(conn)

        if is_shared:
            personal_score_select = "match_stats.personal_score"
        else:
            personal_score_select = self._select_optional_column(
                conn,
                table_name="match_stats",
                table_alias="match_stats",
                column_name="personal_score",
            )

        where_clauses = []
        if not include_firefight:
            where_clauses.append("match_stats.is_firefight = FALSE")

        where_sql = " AND ".join(where_clauses) if where_clauses else "1=1"
        order_dir = "DESC" if order_desc else "ASC"

        sql = f"""
            SELECT
                match_stats.match_id,
                match_stats.start_time,
                match_stats.map_id,
                {map_name_expr} as map_name,
                match_stats.playlist_id,
                {playlist_name_expr} as playlist_name,
                match_stats.pair_id,
                {pair_name_expr} as pair_name,
                match_stats.game_variant_id,
                match_stats.game_variant_name,
                match_stats.outcome,
                match_stats.team_id,
                match_stats.kda,
                match_stats.max_killing_spree,
                match_stats.headshot_kills,
                match_stats.avg_life_seconds,
                match_stats.time_played_seconds,
                match_stats.kills,
                match_stats.deaths,
                match_stats.assists,
                match_stats.accuracy,
                match_stats.my_team_score,
                match_stats.enemy_team_score,
                {team_mmr_expr} as team_mmr,
                {enemy_mmr_expr} as enemy_mmr,
                {personal_score_select}
            FROM {source_sql}{metadata_joins}{pms_join}
            WHERE {where_sql}
            ORDER BY match_stats.start_time {order_dir}
            LIMIT {int(page_size)} OFFSET {int(offset)}
        """

        result = conn.execute(sql, source_params) if source_params else conn.execute(sql)
        rows = result.fetchall()
        columns = [desc[0] for desc in result.description]

        matches = [
            MatchRow(
                match_id=row[columns.index("match_id")],
                start_time=row[columns.index("start_time")],
                map_id=row[columns.index("map_id")],
                map_name=row[columns.index("map_name")],
                playlist_id=row[columns.index("playlist_id")],
                playlist_name=row[columns.index("playlist_name")],
                map_mode_pair_id=row[columns.index("pair_id")],
                map_mode_pair_name=row[columns.index("pair_name")],
                game_variant_id=row[columns.index("game_variant_id")],
                game_variant_name=row[columns.index("game_variant_name")],
                outcome=row[columns.index("outcome")],
                last_team_id=row[columns.index("team_id")],
                kda=row[columns.index("kda")],
                max_killing_spree=row[columns.index("max_killing_spree")],
                headshot_kills=row[columns.index("headshot_kills")],
                average_life_seconds=row[columns.index("avg_life_seconds")],
                time_played_seconds=row[columns.index("time_played_seconds")],
                kills=row[columns.index("kills")] or 0,
                deaths=row[columns.index("deaths")] or 0,
                assists=row[columns.index("assists")] or 0,
                accuracy=row[columns.index("accuracy")],
                my_team_score=row[columns.index("my_team_score")],
                enemy_team_score=row[columns.index("enemy_team_score")],
                team_mmr=row[columns.index("team_mmr")],
                enemy_mmr=row[columns.index("enemy_mmr")],
                personal_score=row[columns.index("personal_score")]
                if "personal_score" in columns
                else None,
            )
            for row in rows
        ]

        return matches, total_pages

    # =========================================================================
    # Chargement batch MMR (Sprint 4.2)
    # =========================================================================

    def load_match_mmr_batch(
        self, match_ids: list[str]
    ) -> dict[str, tuple[float | None, float | None]]:
        """Charge le MMR pour plusieurs matchs depuis shared ou local.

        Priorité : shared.match_participants (v5) → match_stats + player_match_stats (local).

        Args:
            match_ids: Liste des match_id à charger.

        Returns:
            Dict match_id -> (team_mmr, enemy_mmr).
        """
        if not match_ids:
            return {}

        conn = self._get_connection()
        placeholders = ", ".join(["?" for _ in match_ids])

        # V5 : shared.match_participants (colonnes team_mmr/enemy_mmr si disponibles)
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

        # Fallback local : match_stats + player_match_stats (pour MMR historiques)
        try:
            has_pms = self._has_table_cached(conn, "player_match_stats")
            if has_pms:
                result = conn.execute(
                    f"""
                    SELECT
                        ms.match_id,
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
        """Charge team_mmr, enemy_mmr et kills/deaths/assists expected/stddev.

        Pipeline de lecture v5.1 :
            1. shared.match_participants (source principale)
            2. Fallback : player_match_stats locale (données historiques legacy)

        Utilisé par cache_loaders.py pour alimenter l'UI (graphes
        expected vs actual, affichage MMR).

        ⚠️ Limitation API : assists.expected / assists.stddev sont
        toujours NULL (API Halo StatPerformances ne fournit pas Assists).

        Args:
            match_id: ID du match.

        Returns:
            Dict avec team_mmr, enemy_mmr, kills, deaths, assists
            (chacun {count, expected, stddev}). None si non trouvé.
        """
        conn = self._get_connection()

        # V5 : shared.match_participants
        if self._has_shared_table("match_participants"):
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
                if row:
                    team_mmr = row[0]
                    enemy_mmr = row[1]
                    team_id = row[11]

                    # Fallback MMR: si le joueur n'a pas de MMR, chercher depuis un coéquipier
                    # Les MMR d'équipe sont identiques pour tous les joueurs de la même équipe
                    if (team_mmr is None or enemy_mmr is None) and team_id is not None:
                        try:
                            teammate_row = conn.execute(
                                """
                                SELECT team_mmr, enemy_mmr
                                FROM shared.match_participants
                                WHERE match_id = ?
                                  AND team_id = ?
                                  AND team_mmr IS NOT NULL
                                  AND enemy_mmr IS NOT NULL
                                LIMIT 1
                                """,
                                [match_id, team_id],
                            ).fetchone()
                            if teammate_row:
                                team_mmr = teammate_row[0]
                                enemy_mmr = teammate_row[1]
                        except Exception:
                            pass

                    return {
                        "team_id": team_id,
                        "team_mmr": team_mmr,
                        "enemy_mmr": enemy_mmr,
                        "kills": {
                            "count": row[2],
                            "expected": row[3],
                            "stddev": row[4],
                        },
                        "deaths": {
                            "count": row[5],
                            "expected": row[6],
                            "stddev": row[7],
                        },
                        "assists": {
                            "count": row[8],
                            "expected": row[9],
                            "stddev": row[10],
                        },
                    }
            except Exception:
                pass

        # NOTE v5.1 : player_match_stats supprimée, match_participants est la
        # source unique. Pas de fallback legacy.
        return None

    # =========================================================================
    # Chargement Polars zero-copy (Sprint 19 — hot path optimisé)
    # =========================================================================

    def load_matches_as_polars(
        self,
        *,
        include_firefight: bool = True,
        columns: list[str] | None = None,
    ) -> pl.DataFrame:
        """Charge les matchs en DataFrame Polars via Arrow zero-copy.

        Chemin optimisé S19 : DuckDB → Arrow → Polars sans intermédiaire
        MatchRow ni reconstruction Python. ~3× moins de copies mémoire
        que load_matches() + reconstruction DataFrame.

        Args:
            include_firefight: Inclure les matchs PvE.
            columns: Liste de colonnes à projeter (None = toutes).
                     Colonnes disponibles : match_id, start_time, map_id,
                     map_name, playlist_id, playlist_name, pair_id,
                     pair_name, game_variant_id, game_variant_name,
                     outcome, team_id, kda, max_killing_spree,
                     headshot_kills, avg_life_seconds, time_played_seconds,
                     kills, deaths, assists, accuracy, my_team_score,
                     enemy_team_score, team_mmr, enemy_mmr, personal_score.

        Returns:
            DataFrame Polars avec les colonnes demandées.
        """
        conn = self._get_connection()

        # Source v5 (shared) ou v4 (locale)
        source_sql, source_params, uses_mv = self._get_match_source(conn)
        is_shared = bool(source_params)

        where_clauses = []
        if not include_firefight:
            where_clauses.append("match_stats.is_firefight = FALSE")
        where_sql = " AND ".join(where_clauses) if where_clauses else "1=1"

        # v5.1 perf — skip jointures redondantes quand mv_player_matches est utilisée
        if uses_mv:
            metadata_joins = ""
            map_name_expr = "match_stats.map_name"
            playlist_name_expr = "match_stats.playlist_name"
            pair_name_expr = "match_stats.pair_name"
            pms_join = ""
            team_mmr_expr = "match_stats.team_mmr"
            enemy_mmr_expr = "match_stats.enemy_mmr"
        else:
            # Résoudre les métadonnées
            metadata_joins, map_name_expr, playlist_name_expr, pair_name_expr = (
                self._build_metadata_resolution(conn)
            )
            pms_join, team_mmr_expr, enemy_mmr_expr = self._build_mmr_fallback(conn)
        if is_shared:
            personal_score_select = "match_stats.personal_score"
        else:
            personal_score_select = self._select_optional_column(
                conn,
                table_name="match_stats",
                table_alias="match_stats",
                column_name="personal_score",
            )

        # JOIN pour récupérer le rank du joueur principal depuis match_participants
        # rank = classement dans le match (1 = meilleur)
        my_xuid = str(self.xuid or "").strip()
        rank_join = ""
        rank_select = "NULL as rank"
        if my_xuid:
            # En mode v5: utiliser shared.match_participants
            # En mode v4: vérifier si match_participants existe localement
            if is_shared:
                mp_table = "shared.match_participants"
                rank_join = f"""
                    LEFT JOIN {mp_table} mp_rank
                    ON match_stats.match_id = mp_rank.match_id
                    AND mp_rank.xuid = '{my_xuid}'
                """
                rank_select = "mp_rank.rank"
            elif self._has_shared_table("match_participants"):
                # Mode v4 mais avec table match_participants locale
                mp_table = "match_participants"
                rank_join = f"""
                    LEFT JOIN {mp_table} mp_rank
                    ON match_stats.match_id = mp_rank.match_id
                    AND mp_rank.xuid = '{my_xuid}'
                """
                rank_select = "mp_rank.rank"
            # Sinon, rank reste NULL

        # Colonnes complètes avec alias standardisés
        all_select_exprs = f"""
                match_stats.match_id,
                match_stats.start_time,
                match_stats.map_id,
                {map_name_expr} as map_name,
                match_stats.playlist_id,
                {playlist_name_expr} as playlist_name,
                match_stats.pair_id,
                {pair_name_expr} as pair_name,
                match_stats.game_variant_id,
                match_stats.game_variant_name,
                match_stats.outcome,
                match_stats.team_id,
                match_stats.kda,
                match_stats.max_killing_spree,
                match_stats.headshot_kills,
                match_stats.avg_life_seconds,
                match_stats.time_played_seconds,
                COALESCE(match_stats.kills, 0) as kills,
                COALESCE(match_stats.deaths, 0) as deaths,
                COALESCE(match_stats.assists, 0) as assists,
                match_stats.accuracy,
                match_stats.my_team_score,
                match_stats.enemy_team_score,
                {team_mmr_expr} as team_mmr,
                {enemy_mmr_expr} as enemy_mmr,
                {personal_score_select},
                {rank_select}
        """

        sql = f"""
            SELECT {all_select_exprs}
            FROM {source_sql}{metadata_joins}{pms_join}{rank_join}
            WHERE {where_sql}
            ORDER BY match_stats.start_time ASC
        """

        try:
            result = conn.execute(sql, source_params) if source_params else conn.execute(sql)
            df = result_to_polars(result)
        except Exception as e:
            logger.warning(f"Requête avec jointures échouée: {e}. Fallback.")
            sql_fallback = f"""
                SELECT
                    match_stats.match_id,
                    match_stats.start_time,
                    match_stats.map_id,
                    match_stats.map_name,
                    match_stats.playlist_id,
                    match_stats.playlist_name,
                    match_stats.pair_id,
                    match_stats.pair_name,
                    match_stats.game_variant_id,
                    match_stats.game_variant_name,
                    match_stats.outcome,
                    match_stats.team_id,
                    match_stats.kda,
                    match_stats.max_killing_spree,
                    match_stats.headshot_kills,
                    match_stats.avg_life_seconds,
                    match_stats.time_played_seconds,
                    COALESCE(match_stats.kills, 0) as kills,
                    COALESCE(match_stats.deaths, 0) as deaths,
                    COALESCE(match_stats.assists, 0) as assists,
                    match_stats.accuracy,
                    match_stats.my_team_score,
                    match_stats.enemy_team_score,
                    {team_mmr_expr} as team_mmr,
                    {enemy_mmr_expr} as enemy_mmr,
                    {personal_score_select},
                    {rank_select}
                FROM {source_sql}{pms_join}{rank_join}
                WHERE {where_sql}
                ORDER BY match_stats.start_time ASC
            """
            result = (
                conn.execute(sql_fallback, source_params)
                if source_params
                else conn.execute(sql_fallback)
            )
            df = result_to_polars(result)

        if df.is_empty():
            return df

        # Calculer le ratio en Polars (COALESCE kills/deaths déjà fait en SQL)
        df = df.with_columns(
            pl.when(pl.col("deaths") > 0)
            .then(
                (pl.col("kills").cast(pl.Float64) + pl.col("assists").cast(pl.Float64) / 2.0)
                / pl.col("deaths").cast(pl.Float64)
            )
            .otherwise(pl.lit(float("nan")))
            .alias("ratio")
        )

        # Renommer avg_life_seconds → average_life_seconds pour compat avec le code existant
        if "avg_life_seconds" in df.columns:
            df = df.rename({"avg_life_seconds": "average_life_seconds"})

        # Projection de colonnes si demandée (tâche 19.3)
        if columns is not None:
            available = [c for c in columns if c in df.columns]
            df = df.select(available)

        return df

    # =========================================================================
    # Export Polars (legacy — préférer load_matches_as_polars)
    # =========================================================================

    def load_match_stats_as_polars(
        self,
        *,
        match_ids: list[str] | None = None,
        limit: int | None = None,
        include_firefight: bool = True,
    ):
        """Charge les stats de matchs en DataFrame Polars.

        Optimisé pour les analyses avec Polars.

        Args:
            match_ids: Filtrer par une liste de matchs.
            limit: Limite du nombre de résultats.
            include_firefight: Inclure les matchs PvE.

        Returns:
            DataFrame Polars avec les colonnes de match_stats.
        """
        conn = self._get_connection()

        # Source v5 (shared) ou v4 (locale)
        source_sql, source_params, _uses_mv = self._get_match_source(conn)

        where_clauses = []
        params: list = []

        if match_ids:
            placeholders = ", ".join(["?" for _ in match_ids])
            where_clauses.append(f"match_id IN ({placeholders})")
            params.extend(match_ids)

        if not include_firefight:
            where_clauses.append("is_firefight = FALSE")

        where_sql = " AND ".join(where_clauses) if where_clauses else "1=1"
        limit_sql = f"LIMIT {int(limit)}" if limit else ""

        sql = f"""
            SELECT
                match_id,
                start_time,
                map_id,
                map_name,
                playlist_id,
                playlist_name,
                pair_id,
                pair_name,
                game_variant_id,
                game_variant_name,
                outcome,
                team_id,
                kills,
                deaths,
                assists,
                kda,
                accuracy,
                headshot_kills,
                max_killing_spree,
                time_played_seconds,
                avg_life_seconds,
                my_team_score,
                enemy_team_score,
                team_mmr,
                enemy_mmr
            FROM {source_sql}
            WHERE {where_sql}
            ORDER BY start_time ASC
            {limit_sql}
        """

        all_params = source_params + params
        try:
            result = conn.execute(sql, all_params) if all_params else conn.execute(sql)
            return result_to_polars(result)
        except Exception as e:
            logger.warning(f"Erreur chargement match_stats Polars: {e}")
            import polars as pl

            return pl.DataFrame()
