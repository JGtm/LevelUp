"""
Mixin pour l'accès aux paires killer/victim depuis shared_matches.duckdb.

Regroupe les méthodes de chargement et d'analyse des relations
killer/victim depuis la table partagée ``killer_victim_pairs`` :
- load_killer_victim_pairs_as_polars
- get_antagonists_summary_polars
- has_killer_victim_pairs
"""

from __future__ import annotations

import logging

import polars as pl

from src.data.repositories._arrow_bridge import result_to_polars

logger = logging.getLogger(__name__)


class KillerVictimMixin:
    """Mixin fournissant l'accès aux paires killer/victim (shared)."""

    def load_killer_victim_pairs_as_polars(
        self,
        *,
        match_id: str | None = None,
        match_ids: list[str] | None = None,
        limit: int | None = None,
    ):
        """Charge les paires killer→victim en DataFrame Polars.

        Lit depuis shared.v_killer_victim_full (v6, vue garantie présente).

        Args:
            match_id: Filtrer par un match spécifique.
            match_ids: Filtrer par une liste de matchs.
            limit: Limite du nombre de résultats.

        Returns:
            DataFrame Polars avec colonnes:
            - match_id, killer_xuid, killer_gamertag, victim_xuid,
              victim_gamertag, kill_count, time_ms
        """
        conn = self._get_connection()

        # Construire la requête
        where_clauses = []
        params = []

        if match_id:
            where_clauses.append("match_id = ?")
            params.append(match_id)
        elif match_ids:
            placeholders = ", ".join(["?" for _ in match_ids])
            where_clauses.append(f"match_id IN ({placeholders})")
            params.extend(match_ids)

        where_sql = " AND ".join(where_clauses) if where_clauses else "1=1"
        limit_sql = f"LIMIT {int(limit)}" if limit else ""

        # Vue v6 garantie présente
        table_ref = "shared.v_killer_victim_full"

        sql = f"""
            SELECT
                match_id,
                killer_xuid,
                killer_gamertag,
                victim_xuid,
                victim_gamertag,
                kill_count,
                time_ms
            FROM {table_ref}
            WHERE {where_sql}
            ORDER BY match_id, time_ms
            {limit_sql}
        """

        try:
            result = conn.execute(sql, params) if params else conn.execute(sql)
            return result_to_polars(result)
        except Exception as e:
            logger.warning("Erreur chargement killer_victim_pairs: %s", e)
            # Retourner un DataFrame vide avec le bon schéma
            return pl.DataFrame(
                {
                    "match_id": [],
                    "killer_xuid": [],
                    "killer_gamertag": [],
                    "victim_xuid": [],
                    "victim_gamertag": [],
                    "kill_count": [],
                    "time_ms": [],
                }
            )

    def get_antagonists_summary_polars(
        self,
        top_n: int = 20,
    ):
        """Calcule un résumé des antagonistes avec Polars.

        Agrège les paires killer_victim pour obtenir le top némésis/victimes.

        Args:
            top_n: Nombre de résultats par catégorie.

        Returns:
            Dict avec 'nemeses' et 'victims' DataFrames Polars.
        """
        pairs_df = self.load_killer_victim_pairs_as_polars()

        if pairs_df.is_empty():
            return {
                "nemeses": pl.DataFrame(),
                "victims": pl.DataFrame(),
            }

        me_xuid = self._xuid

        # Top némésis (qui m'a le plus tué)
        nemeses = (
            pairs_df.filter(pl.col("victim_xuid") == me_xuid)
            .group_by("killer_xuid", "killer_gamertag")
            .agg(pl.col("kill_count").sum().alias("times_killed_by"))
            .sort("times_killed_by", descending=True)
            .head(top_n)
        )

        # Top victimes (qui j'ai le plus tué)
        victims = (
            pairs_df.filter(pl.col("killer_xuid") == me_xuid)
            .group_by("victim_xuid", "victim_gamertag")
            .agg(pl.col("kill_count").sum().alias("times_killed"))
            .sort("times_killed", descending=True)
            .head(top_n)
        )

        return {
            "nemeses": nemeses,
            "victims": victims,
        }

    def has_killer_victim_pairs(self) -> bool:
        """Vérifie si v_killer_victim_full contient des données.

        Vue v6 garantie présente — aucun fallback nécessaire.

        Returns:
            True si des paires sont disponibles.
        """
        conn = self._get_connection()
        try:
            row = conn.execute("SELECT 1 FROM shared.v_killer_victim_full LIMIT 1").fetchone()
            return row is not None
        except Exception:
            logger.warning("has_killer_victim_pairs: erreur v_killer_victim_full", exc_info=True)
            return False
