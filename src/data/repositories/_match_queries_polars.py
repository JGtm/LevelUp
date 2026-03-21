"""Mixin Polars pour les requêtes de matchs DuckDB.

Méthodes de chargement optimisées Arrow → Polars zero-copy,
extraites de MatchQueriesMixin.
"""

from __future__ import annotations

import logging
from typing import Any

import polars as pl

from src.data.repositories._arrow_bridge import result_to_polars
from src.data.repositories._match_queries_helpers import (
    build_match_select,
    resolve_query_context,
)

logger = logging.getLogger(__name__)


class _MatchQueriesPolarsMixin:
    """Mixin Polars pour MatchQueriesMixin.

    Requiert que la classe finale fournisse via MRO :
    _get_connection, _get_match_source, _build_metadata_resolution,
    _build_mmr_fallback, _select_optional_column, _has_shared_table, xuid.
    """

    def load_matches_as_polars(  # noqa: PLR0912, C901
        self,
        *,
        include_firefight: bool = True,
        columns: list[str] | None = None,
        max_matches: int | None = None,
    ) -> pl.DataFrame:
        """Charge les matchs en DataFrame Polars via Arrow zero-copy.

        Chemin optimisé S19 : DuckDB → Arrow → Polars sans intermédiaire
        MatchRow ni reconstruction Python. ~3× moins de copies mémoire.

        Args:
            include_firefight: Inclure les matchs PvE.
            columns: Liste de colonnes à projeter (None = toutes).
            max_matches: Limite SQL sur les N matchs les plus récents.

        Returns:
            DataFrame Polars avec les colonnes demandées.
        """
        conn = self._get_connection()
        ctx = resolve_query_context(self, conn)

        where_clauses: list[str] = []
        if not include_firefight:
            where_clauses.append("match_stats.is_firefight = FALSE")
        where_sql = " AND ".join(where_clauses) if where_clauses else "1=1"

        select_cols = build_match_select(ctx, coalesce_stats=True)

        # JOIN rank du joueur principal depuis match_participants
        rank_join, rank_select = self._build_rank_join(ctx)

        # JOIN performance_score depuis player_match_enrichment (même DB que stats.duckdb)
        # LEFT JOIN → NULL si le match n'est pas encore enrichi (fan-out incomplet OK)
        perf_join = (
            "\n                    LEFT JOIN player_match_enrichment pme"
            "\n                    ON match_stats.match_id = pme.match_id"
        )

        all_select = f"""
                {select_cols},
                {rank_select},
                pme.performance_score
        """

        limit_clause = f"LIMIT {int(max_matches)}" if max_matches else ""

        sql = self._build_polars_query(
            all_select, ctx, rank_join + perf_join, where_sql, limit_clause
        )

        try:
            result = (
                conn.execute(sql, ctx.source_params) if ctx.source_params else conn.execute(sql)
            )
            df = result_to_polars(result)
        except Exception as e:
            logger.warning("Requête avec jointures échouée: %s. Fallback.", e)
            df = self._execute_polars_fallback(
                conn, ctx, rank_join, rank_select, where_sql, limit_clause
            )

        if df.is_empty():
            return df

        return self._finalize_polars_df(df, columns)

    def _build_rank_join(self, ctx: Any) -> tuple[str, str]:
        """Construit le JOIN pour le rank du joueur principal."""
        my_xuid = str(self.xuid or "").strip()
        if not my_xuid:
            return "", "NULL as rank"

        if ctx.is_shared:
            mp_table = "shared.match_participants"
        else:
            return "", "NULL as rank"

        rank_join = f"""
                    LEFT JOIN {mp_table} mp_rank
                    ON match_stats.match_id = mp_rank.match_id
                    AND mp_rank.xuid = '{my_xuid}'
        """
        return rank_join, "mp_rank.rank"

    @staticmethod
    def _build_polars_query(
        select_expr: str,
        ctx: Any,
        rank_join: str,
        where_sql: str,
        limit_clause: str,
    ) -> str:
        """Construit la requête SQL complète pour load_matches_as_polars."""
        from_clause = f"FROM {ctx.source_sql}{ctx.metadata_joins}{ctx.pms_join}{rank_join}"
        if limit_clause:
            return f"""
                SELECT * FROM (
                    SELECT {select_expr}
                    {from_clause}
                    WHERE {where_sql}
                    ORDER BY match_stats.start_time DESC
                    {limit_clause}
                ) _recent
                ORDER BY start_time ASC
            """
        return f"""
                SELECT {select_expr}
                {from_clause}
                WHERE {where_sql}
                ORDER BY match_stats.start_time ASC
        """

    def _execute_polars_fallback(  # noqa: PLR0913
        self,
        conn: Any,
        ctx: Any,
        rank_join: str,
        rank_select: str,
        where_sql: str,
        limit_clause: str,
    ) -> pl.DataFrame:
        """Exécute la requête Polars en fallback (sans joins métadonnées)."""
        select_cols = build_match_select(ctx, coalesce_stats=True, direct_names=True)
        all_select = f"""
                {select_cols},
                {rank_select}
        """

        from_clause = f"FROM {ctx.source_sql}{ctx.pms_join}{rank_join}"
        if limit_clause:
            sql = f"""
                SELECT * FROM (
                    SELECT {all_select}
                    {from_clause}
                    WHERE {where_sql}
                    ORDER BY match_stats.start_time DESC
                    {limit_clause}
                ) _recent
                ORDER BY start_time ASC
            """
        else:
            sql = f"""
                SELECT {all_select}
                {from_clause}
                WHERE {where_sql}
                ORDER BY match_stats.start_time ASC
            """
        result = conn.execute(sql, ctx.source_params) if ctx.source_params else conn.execute(sql)
        return result_to_polars(result)

    @staticmethod
    def _finalize_polars_df(df: pl.DataFrame, columns: list[str] | None) -> pl.DataFrame:
        """Ajoute ratio (alias de kda API), renomme avg_life_seconds, projette les colonnes."""
        # ratio = kda officiel de l'API (source unique pour cohérence inter-panneaux)
        # kda peut être NULL (match sans morts/stats), laissé tel quel — les graphes
        # utilisent drop_nulls() ou fill_null() selon le contexte.
        if "kda" in df.columns:
            df = df.with_columns(pl.col("kda").alias("ratio"))
        else:
            df = df.with_columns(pl.lit(None).cast(pl.Float64).alias("ratio"))

        # Alias pour compat code existant
        if "avg_life_seconds" in df.columns:
            df = df.rename({"avg_life_seconds": "average_life_seconds"})

        # Projection de colonnes (tâche 19.3)
        if columns is not None:
            available = [c for c in columns if c in df.columns]
            df = df.select(available)

        return df

    def load_match_stats_as_polars(
        self,
        *,
        match_ids: list[str] | None = None,
        limit: int | None = None,
        include_firefight: bool = True,
    ) -> pl.DataFrame:
        """Charge les stats de matchs en DataFrame Polars.

        Args:
            match_ids: Filtrer par une liste de matchs.
            limit: Limite du nombre de résultats.
            include_firefight: Inclure les matchs PvE.

        Returns:
            DataFrame Polars avec les colonnes de match_stats.
        """
        conn = self._get_connection()
        source_sql, source_params, _uses_mv = self._get_match_source(conn)

        where_clauses: list[str] = []
        params: list[Any] = []

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
                match_id, start_time, map_id, map_name,
                playlist_id, playlist_name, pair_id, pair_name,
                game_variant_id, game_variant_name, outcome, team_id,
                kills, deaths, assists, kda, accuracy,
                headshot_kills, max_killing_spree,
                time_played_seconds, avg_life_seconds,
                my_team_score, enemy_team_score,
                team_mmr, enemy_mmr
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
            logger.warning("Erreur chargement match_stats Polars: %s", e)
            return pl.DataFrame()
