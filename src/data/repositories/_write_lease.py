"""Mécanisme de write lease pour la coordination MediaIndexer ↔ DuckDBRepository.

Le MediaIndexer ouvre stats.duckdb en read_write (connexions directes,
hors registre _instances). Pour éviter la race condition
  « Can't open a connection to same database file with a different
    configuration than existing connections »
lors d'un switch de joueur, tout composant qui ouvre une connexion
read_write DOIT acquérir un write lease via db_write_lease(db_path).
_get_connection() attend alors que tous les writers aient terminé.
"""

from __future__ import annotations

import collections
import logging
import threading
import time
from collections.abc import Generator
from contextlib import contextmanager
from pathlib import Path

logger = logging.getLogger(__name__)

_write_leases: dict[str, int] = collections.defaultdict(int)
_write_leases_cond = threading.Condition(threading.Lock())


@contextmanager
def db_write_lease(db_path: str | Path) -> Generator[None, None, None]:
    """Context manager : déclare une connexion read_write active sur *db_path*.

    À utiliser dans MediaIndexer autour de chaque ``duckdb.connect(read_only=False)``
    pour permettre à ``_get_connection()`` d'attendre avant d'ouvrir en read_only.

    Usage::

        from src.data.repositories.duckdb_repo import db_write_lease

        with db_write_lease(self.db_path), duckdb.connect(..., read_only=False) as conn:
            ...
    """
    path = str(Path(db_path).resolve())
    with _write_leases_cond:
        _write_leases[path] += 1
    try:
        yield
    finally:
        with _write_leases_cond:
            _write_leases[path] -= 1
            _write_leases_cond.notify_all()


def wait_for_write_leases_cleared(db_path: str | Path, timeout: float = 5.0) -> None:
    """Attend (max *timeout* s) que les write leases sur *db_path* soient libérées.

    Timeout volontairement court pour éviter un freeze UI prolongé si un writer
    reste bloqué plus longtemps que prévu.
    """
    path = str(Path(db_path).resolve())
    deadline = time.monotonic() + timeout
    with _write_leases_cond:
        while _write_leases.get(path, 0) > 0:
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                logger.warning(
                    "Timeout write lease dépassé pour %s — tentative d'ouverture quand même",
                    Path(db_path).name,
                )
                break
            _write_leases_cond.wait(timeout=min(remaining, 0.5))
