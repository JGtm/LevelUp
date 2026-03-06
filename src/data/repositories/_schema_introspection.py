"""Mixin d'introspection de schéma pour DuckDBRepository.

Regroupe les méthodes de vérification d'existence de tables, vues et colonnes.
Toutes les vérifications utilisent un cache d'instance pour éviter les requêtes
``information_schema`` répétées au sein d'une même session.
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    import duckdb

logger = logging.getLogger(__name__)


class SchemaIntrospectionMixin:
    """Vérification d'existence de tables, vues et colonnes (avec cache)."""

    # ------------------------------------------------------------------
    # Tables shared
    # ------------------------------------------------------------------

    def _has_shared_table(self, table_name: str) -> bool:
        """Vérifie si une table existe dans shared_matches.duckdb (avec cache).

        Args:
            table_name: Nom de la table à vérifier (sans préfixe 'shared.').

        Returns:
            True si la table existe dans le schema shared.
        """
        cache_key = f"shared_table:{table_name}"
        if cache_key in self._table_cache:
            return self._table_cache[cache_key]

        conn = self._get_connection()
        if not self.has_shared:
            self._table_cache[cache_key] = False
            return False
        try:
            result = conn.execute(
                "SELECT table_name FROM information_schema.tables "
                "WHERE table_catalog = 'shared' AND table_name = ?",
                [table_name],
            ).fetchone()
            exists = result is not None
            self._table_cache[cache_key] = exists
            return exists
        except Exception:
            self._table_cache[cache_key] = False
            return False

    def _has_shared_view(self, view_name: str) -> bool:
        """Vérifie si une vue existe dans shared_matches.duckdb (avec cache).

        Utilise ``duckdb_views()`` qui fonctionne bien avec les bases attachées,
        contrairement à ``catalog.information_schema.views``.

        Args:
            view_name: Nom de la vue à vérifier (sans préfixe 'shared.').

        Returns:
            True si la vue existe dans le catalog shared.
        """
        cache_key = f"shared_view:{view_name}"
        if cache_key in self._view_cache:
            return self._view_cache[cache_key]

        conn = self._get_connection()
        if not self.has_shared:
            self._view_cache[cache_key] = False
            return False
        try:
            result = conn.execute(
                "SELECT view_name FROM duckdb_views() "
                "WHERE database_name = 'shared' AND view_name = ?",
                [view_name],
            ).fetchone()
            exists = result is not None
            self._view_cache[cache_key] = exists
            return exists
        except Exception:
            self._view_cache[cache_key] = False
            return False

    # ------------------------------------------------------------------
    # Tables locales (main)
    # ------------------------------------------------------------------

    def _has_table_cached(self, conn: duckdb.DuckDBPyConnection, table_name: str) -> bool:
        """Vérifie si une table existe dans le schema main (avec cache).

        Args:
            conn: Connexion DuckDB.
            table_name: Nom de la table.

        Returns:
            True si la table existe.
        """
        cache_key = f"main_table:{table_name}"
        if cache_key in self._table_cache:
            return self._table_cache[cache_key]

        try:
            result = conn.execute(
                "SELECT COUNT(*) FROM information_schema.tables "
                "WHERE table_schema = 'main' AND table_name = ?",
                [table_name],
            ).fetchone()
            exists = bool(result and result[0] > 0)
            self._table_cache[cache_key] = exists
            return exists
        except Exception:
            self._table_cache[cache_key] = False
            return False

    def _has_table(self, table_name: str) -> bool:
        """Vérifie si une table existe dans la DB (délègue à ``_has_table_cached``)."""
        conn = self._get_connection()
        return self._has_table_cached(conn, table_name)

    def has_table(self, table_name: str) -> bool:
        """Vérifie si une table existe (alias public de ``_has_table``).

        Args:
            table_name: Nom de la table.

        Returns:
            True si la table existe.
        """
        return self._has_table(table_name)

    # ------------------------------------------------------------------
    # Colonnes
    # ------------------------------------------------------------------

    def _has_column(
        self,
        conn: duckdb.DuckDBPyConnection,
        table_name: str,
        column_name: str,
    ) -> bool:
        """Retourne True si une colonne existe dans une table (avec cache).

        Utile pour supporter des schémas historiques (colonnes ajoutées en v4).
        """
        cache_key = f"{table_name}.{column_name}"
        if cache_key in self._schema_cache:
            return self._schema_cache[cache_key]

        try:
            exists = (
                conn.execute(
                    "SELECT COUNT(*) FROM information_schema.columns "
                    "WHERE table_name = ? AND column_name = ?",
                    (table_name, column_name),
                ).fetchone()[0]
                > 0
            )
            self._schema_cache[cache_key] = exists
            return exists
        except Exception:
            self._schema_cache[cache_key] = False
            return False

    def _has_shared_mp_column(self, conn: duckdb.DuckDBPyConnection, column_name: str) -> bool:
        """Vérifie si match_participants (shared) possède une colonne (avec cache).

        Consulte le catalog ``shared`` attaché en priorité, puis fallback
        sur le catalog principal (utile en tests unitaires).
        """
        cache_key = f"shared_mp.{column_name}"
        if cache_key in self._schema_cache:
            return self._schema_cache[cache_key]

        for catalog in ("shared", "main"):
            try:
                result = conn.execute(
                    f"SELECT COUNT(*) FROM {catalog}.information_schema.columns "
                    "WHERE table_name = 'match_participants' AND column_name = ?",
                    (column_name,),
                ).fetchone()
                if result and result[0] > 0:
                    self._schema_cache[cache_key] = True
                    return True
            except Exception:
                continue
        self._schema_cache[cache_key] = False
        return False

    def _select_optional_column(  # noqa: PLR0913
        self,
        conn: duckdb.DuckDBPyConnection,
        *,
        table_name: str,
        table_alias: str,
        column_name: str,
        output_name: str | None = None,
    ) -> str:
        """Construit une expression SELECT tolérante si la colonne manque."""
        out = output_name or column_name
        if self._has_column(conn, table_name, column_name):
            return f"{table_alias}.{column_name} AS {out}"
        return f"NULL AS {out}"
