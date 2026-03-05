"""Utilitaires de connexion DuckDB centralisés.

Fournit un context manager read-only pour éviter les ``duckdb.connect()``
dispersés dans le code UI. Toutes les connexions éphémères doivent passer
par ``duckdb_read_only`` pour garantir la fermeture propre.
"""

from __future__ import annotations

import contextlib
from pathlib import Path
from typing import TYPE_CHECKING

import duckdb

if TYPE_CHECKING:
    from collections.abc import Generator


@contextlib.contextmanager
def duckdb_read_only(db_path: str | Path) -> Generator[duckdb.DuckDBPyConnection, None, None]:
    """Ouvre une connexion DuckDB en lecture seule avec fermeture garantie.

    Usage::

        from src.utils.db import duckdb_read_only

        with duckdb_read_only(db_path) as conn:
            rows = conn.execute("SELECT ...").fetchall()

    Args:
        db_path: Chemin vers le fichier ``.duckdb``.

    Yields:
        Connexion DuckDB read-only.
    """
    conn = duckdb.connect(str(db_path), read_only=True)
    try:
        yield conn
    finally:
        conn.close()


@contextlib.contextmanager
def duckdb_read_write(db_path: str | Path) -> Generator[duckdb.DuckDBPyConnection, None, None]:
    """Ouvre une connexion DuckDB en lecture-écriture avec fermeture garantie.

    Usage::

        from src.utils.db import duckdb_read_write

        with duckdb_read_write(db_path) as conn:
            conn.execute("INSERT INTO ...")

    Args:
        db_path: Chemin vers le fichier ``.duckdb``.

    Yields:
        Connexion DuckDB read-write.
    """
    conn = duckdb.connect(str(db_path))
    try:
        yield conn
    finally:
        conn.close()


@contextlib.contextmanager
def duckdb_flexible(db_path: str | Path) -> Generator[duckdb.DuckDBPyConnection, None, None]:
    """Ouvre une connexion DuckDB, read-only si possible, sinon read-write.

    Utile en contexte Streamlit où un autre thread peut déjà détenir une
    connexion write sur le même fichier.

    Args:
        db_path: Chemin vers le fichier ``.duckdb``.

    Yields:
        Connexion DuckDB (read-only ou read-write).
    """
    conn = None
    for ro in (True, False):
        try:
            conn = duckdb.connect(str(db_path), read_only=ro)
            break
        except Exception:
            if not ro:
                raise
    try:
        yield conn
    finally:
        if conn:
            conn.close()


def is_duckdb_v4_path(db_path: str) -> bool:
    """Détecte si le chemin est une DB joueur DuckDB v4+.

    Args:
        db_path: Chemin vers le fichier DB.

    Returns:
        True si le chemin pointe vers un fichier ``.duckdb``.
    """
    if not db_path:
        return False
    return db_path.endswith(".duckdb") or db_path.endswith("stats.duckdb")


def has_table(conn: duckdb.DuckDBPyConnection, table_name: str, schema: str = "main") -> bool:
    """Vérifie si une table existe dans une connexion DuckDB ouverte.

    Args:
        conn: Connexion DuckDB déjà ouverte.
        table_name: Nom de la table à vérifier.
        schema: Schéma SQL (``main`` par défaut).

    Returns:
        True si la table existe.
    """
    rows = conn.execute(
        "SELECT 1 FROM information_schema.tables WHERE table_schema = ? AND table_name = ?",
        [schema, table_name],
    ).fetchone()
    return rows is not None
