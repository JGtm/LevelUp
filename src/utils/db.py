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
