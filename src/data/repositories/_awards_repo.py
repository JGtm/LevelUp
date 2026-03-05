"""Mixin awards et citations pour DuckDBRepository.

Gère les PersonalScoreAwards (Sprint 8.2) et les citations match.
"""

from __future__ import annotations

import logging

logger = logging.getLogger(__name__)


class AwardsMixin:
    """Insertion/chargement des citations et PersonalScoreAwards."""

    def insert_citation(
        self,
        match_id: str,
        citation_name_norm: str,
        value: int,
    ) -> None:
        """Insère ou met à jour une citation pour un match.

        Args:
            match_id: Identifiant du match.
            citation_name_norm: Nom normalisé de la citation.
            value: Valeur de la citation.
        """
        conn = self._get_connection()
        conn.execute(
            "INSERT OR REPLACE INTO match_citations "
            "(match_id, citation_name_norm, value) VALUES (?, ?, ?)",
            [match_id, citation_name_norm, value],
        )

    def load_personal_score_awards_as_polars(  # noqa: PLR0913
        self,
        *,
        match_id: str | None = None,
        match_ids: list[str] | None = None,
        category: str | None = None,
        limit: int | None = None,
    ):
        """Charge les PersonalScoreAwards en DataFrame Polars.

        Sprint 8.2 – Permet de visualiser la participation au match :
        kills, assists, objectifs, véhicules, pénalités.

        Args:
            match_id: Filtrer par un match spécifique.
            match_ids: Filtrer par une liste de matchs.
            category: Filtrer par catégorie (kill, assist, objective, etc.).
            limit: Limite du nombre de résultats.

        Returns:
            DataFrame Polars avec colonnes :
            match_id, xuid, award_name, award_category, award_count, award_score.
        """
        from src.data.repositories._arrow_bridge import result_to_polars

        conn = self._get_connection()

        where_clauses: list[str] = []
        params: list = []

        if match_id:
            where_clauses.append("match_id = ?")
            params.append(match_id)
        elif match_ids:
            placeholders = ", ".join(["?" for _ in match_ids])
            where_clauses.append(f"match_id IN ({placeholders})")
            params.extend(match_ids)

        if category:
            where_clauses.append("award_category = ?")
            params.append(category)

        where_sql = " AND ".join(where_clauses) if where_clauses else "1=1"
        limit_sql = f"LIMIT {int(limit)}" if limit else ""

        sql = f"""
            SELECT
                match_id, xuid, award_name, award_category,
                award_count, award_score, created_at
            FROM personal_score_awards
            WHERE {where_sql}
            ORDER BY award_score DESC
            {limit_sql}
        """

        try:
            result = conn.execute(sql, params) if params else conn.execute(sql)
            return result_to_polars(result)
        except Exception as e:
            logger.warning("Erreur chargement personal_score_awards Polars: %s", e)
            import polars as pl

            return pl.DataFrame()

    def has_personal_score_awards(self) -> bool:
        """Vérifie si des PersonalScoreAwards existent dans la DB."""
        conn = self._get_connection()
        try:
            result = conn.execute("SELECT 1 FROM personal_score_awards LIMIT 1").fetchone()
            return result is not None
        except Exception:
            return False
