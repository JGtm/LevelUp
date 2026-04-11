"""Utilitaires bas-niveau partagés entre les modules de migrations DuckDB.

Ce module contient les fonctions utilitaires (get_table_columns, table_exists,
column_exists, _add_column_if_missing, _create_index_safe) qui sont importées
par migrations_player.py, migrations_shared.py et migrations_metadata.py.

Ne pas ajouter de logique métier ici — uniquement des helpers DDL génériques.
"""

from __future__ import annotations

import logging
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    import duckdb

logger = logging.getLogger(__name__)


def get_table_columns(conn: duckdb.DuckDBPyConnection, table_name: str) -> set[str]:
    """Retourne l'ensemble des noms de colonnes d'une table.

    Args:
        conn: Connexion DuckDB.
        table_name: Nom de la table.

    Returns:
        Ensemble des noms de colonnes (vide si la table n'existe pas).
    """
    try:
        cols = conn.execute(
            "SELECT column_name FROM information_schema.columns "
            "WHERE table_schema = 'main' AND table_name = ?",
            [table_name],
        ).fetchall()
        return {r[0] for r in cols} if cols else set()
    except Exception as e:
        logger.debug("Impossible de lire les colonnes de %s: %s", table_name, e)
        return set()


def table_exists(conn: duckdb.DuckDBPyConnection, table_name: str) -> bool:
    """Vérifie si une table existe dans le schéma main.

    Args:
        conn: Connexion DuckDB.
        table_name: Nom de la table.

    Returns:
        True si la table existe.
    """
    try:
        result = conn.execute(
            "SELECT COUNT(*) FROM information_schema.tables "
            "WHERE table_schema = 'main' AND table_name = ?",
            [table_name],
        ).fetchone()
        return bool(result and result[0] > 0)
    except Exception:
        return False


def column_exists(conn: duckdb.DuckDBPyConnection, table_name: str, column_name: str) -> bool:
    """Vérifie si une colonne existe dans une table.

    Args:
        conn: Connexion DuckDB.
        table_name: Nom de la table.
        column_name: Nom de la colonne.

    Returns:
        True si la colonne existe.
    """
    try:
        result = conn.execute(
            "SELECT COUNT(*) FROM information_schema.columns "
            "WHERE table_schema = 'main' AND table_name = ? AND column_name = ?",
            [table_name, column_name],
        ).fetchone()
        return bool(result and result[0] > 0)
    except Exception:
        return False


def _add_column_if_missing(
    conn: duckdb.DuckDBPyConnection,
    table_name: str,
    column_name: str,
    column_type: str,
    existing_cols: set[str] | None = None,
) -> bool:
    """Ajoute une colonne à une table si elle n'existe pas.

    Args:
        conn: Connexion DuckDB.
        table_name: Nom de la table.
        column_name: Nom de la colonne à ajouter.
        column_type: Type SQL de la colonne.
        existing_cols: Colonnes existantes (optionnel, évite un query).

    Returns:
        True si la colonne a été ajoutée, False sinon.
    """
    if existing_cols is not None:
        is_missing = column_name not in existing_cols
    else:
        is_missing = not column_exists(conn, table_name, column_name)

    if is_missing:
        try:
            conn.execute(f"ALTER TABLE {table_name} ADD COLUMN {column_name} {column_type}")
            logger.info("Ajout de la colonne %s à %s", column_name, table_name)
            return True
        except Exception as e:
            logger.warning("Impossible d'ajouter %s à %s: %s", column_name, table_name, e)
    return False


def _create_index_safe(conn: duckdb.DuckDBPyConnection, sql: str, index_name: str) -> None:
    """Exécute une instruction CREATE INDEX de façon sûre (idempotente).

    Args:
        conn: Connexion DuckDB.
        sql: Instruction SQL CREATE INDEX IF NOT EXISTS.
        index_name: Nom de l'index (pour le log).
    """
    try:
        conn.execute(sql)
    except Exception as e:
        err = str(e).lower()
        if "already exists" in err or "read only" in err:
            logger.debug("Index %s ignoré: %s", index_name, e)
        elif "does not exist" in err or "table with name" in err:
            logger.debug("Index %s ignoré (table absente): %s", index_name, e)
        else:
            logger.warning("Index %s non créé: %s", index_name, e)
