"""Verrou inter-processus pour empêcher les syncs parallèles.

Utilise filelock (déjà installé dans le venv) pour créer un verrou fichier
dans data/.sync.lock. Si le verrou est déjà pris, lève SyncAlreadyRunning
immédiatement (timeout=0 → fail fast).

Usage dans scripts/sync.py et scripts/backfill_data.py :

    from src.utils.sync_lock import SyncLock, SyncAlreadyRunning

    try:
        with SyncLock():
            # sync / backfill ici
    except SyncAlreadyRunning:
        print("Un sync est déjà en cours. Réessaie dans quelques instants.")
        sys.exit(1)
"""

from __future__ import annotations

import logging
from pathlib import Path

logger = logging.getLogger(__name__)

_PROJECT_ROOT = Path(__file__).resolve().parents[2]
_LOCK_FILE = _PROJECT_ROOT / "data" / ".sync.lock"


class SyncAlreadyRunning(RuntimeError):
    """Levée quand un autre processus détient déjà le verrou de sync."""


class SyncLock:
    """Context manager → acquiert le verrou inter-processus data/.sync.lock.

    Args:
        timeout: Secondes d'attente max avant d'abandonner. 0 = fail fast
                 (comportement recommandé pour éviter les deadlocks UI).
        lock_file: Chemin alternatif pour le fichier de verrou (tests).
    """

    def __init__(
        self,
        timeout: float = 0,
        lock_file: Path | None = None,
    ) -> None:
        self._lock_file = lock_file or _LOCK_FILE
        self._timeout = timeout
        self._lock = None

    def __enter__(self) -> SyncLock:
        from filelock import FileLock, Timeout

        # S'assurer que le dossier data/ existe
        self._lock_file.parent.mkdir(parents=True, exist_ok=True)

        self._lock = FileLock(str(self._lock_file), timeout=self._timeout)
        try:
            self._lock.acquire()
        except Timeout as err:
            raise SyncAlreadyRunning(
                "Un sync est déjà en cours. Réessaie dans quelques instants."
            ) from err
        logger.debug("[SyncLock] Verrou acquis : %s", self._lock_file)
        return self

    def __exit__(self, exc_type: object, exc_val: object, exc_tb: object) -> None:
        if self._lock is not None:
            try:
                self._lock.release()
                logger.debug("[SyncLock] Verrou libéré : %s", self._lock_file)
            except Exception:
                pass
