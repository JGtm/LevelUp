"""Store de jobs asynchrones en mémoire — Slice 1.

Gère le cycle de vie des jobs longs (smoke test, reindex médias, etc.)
avec un store process-level thread-safe.

Cycle de vie d'un job :
    queued → running → succeeded | failed | cancelled

Rétention : les jobs terminaux sont conservés pendant ``JOB_RETENTION_S`` secondes
puis purgés au prochain ``create_job()`` (purge lazy).

Usage :
    from apps.api.app.services.job_store import JobStore

    store = JobStore.get()
    job = store.create("setup_smoke_test")
    store.update(job.job_id, status="running", progress_pct=10)
    job = store.get_job(job.job_id)
"""

from __future__ import annotations

import threading
import time
import uuid
from datetime import datetime, timezone

from apps.api.app.schemas.common import ApiErrorSchema, AsyncJobStatus

# Rétention des jobs terminaux (secondes)
JOB_RETENTION_S = 3600  # 1 heure


class JobStore:
    """Singleton process-level pour le suivi des jobs asynchrones."""

    _instance: JobStore | None = None
    _lock: threading.Lock = threading.Lock()

    def __init__(self) -> None:
        self._jobs: dict[str, _JobEntry] = {}
        self._jobs_lock = threading.Lock()

    @classmethod
    def get(cls) -> JobStore:
        """Retourne le singleton (thread-safe, lazy init)."""
        if cls._instance is None:
            with cls._lock:
                if cls._instance is None:
                    cls._instance = cls()
        return cls._instance

    # ------------------------------------------------------------------
    # CRUD
    # ------------------------------------------------------------------

    def create(self, job_type: str) -> AsyncJobStatus:
        """Crée un nouveau job en statut ``queued``, retourne son statut initial."""
        self._purge_expired()
        job_id = str(uuid.uuid4())
        now = datetime.now(timezone.utc)
        entry = _JobEntry(
            job_id=job_id,
            job_type=job_type,
            status="queued",
            created_at=now,
        )
        with self._jobs_lock:
            self._jobs[job_id] = entry
        return entry.to_status()

    def get_job(self, job_id: str) -> AsyncJobStatus | None:
        """Retourne le statut du job ou None si inconnu/expiré."""
        with self._jobs_lock:
            entry = self._jobs.get(job_id)
        if entry is None:
            return None
        if entry.is_expired():
            with self._jobs_lock:
                self._jobs.pop(job_id, None)
            return None
        return entry.to_status()

    def update(  # noqa: PLR0913
        self,
        job_id: str,
        *,
        status: str | None = None,
        progress_pct: int | None = None,
        current_step: str | None = None,
        result: dict | None = None,
        error: ApiErrorSchema | None = None,
    ) -> bool:
        """Met à jour un job existant. Retourne False si job inconnu."""
        with self._jobs_lock:
            entry = self._jobs.get(job_id)
            if entry is None:
                return False
            if status is not None:
                entry.status = status
                if status in ("succeeded", "failed", "cancelled"):
                    entry.finished_at = datetime.now(timezone.utc)
                elif status == "running" and entry.started_at is None:
                    entry.started_at = datetime.now(timezone.utc)
            if progress_pct is not None:
                entry.progress_pct = progress_pct
            if current_step is not None:
                entry.current_step = current_step
            if result is not None:
                entry.result = result
            if error is not None:
                entry.error = error
        return True

    # ------------------------------------------------------------------
    # Purge
    # ------------------------------------------------------------------

    def _purge_expired(self) -> int:
        """Supprime les jobs terminaux expirés. Retourne le nombre supprimé."""
        with self._jobs_lock:
            expired = [jid for jid, e in self._jobs.items() if e.is_expired()]
            for jid in expired:
                del self._jobs[jid]
        return len(expired)


class _JobEntry:
    """Entrée interne d'un job (mutable, protégée par le lock du store)."""

    def __init__(
        self,
        job_id: str,
        job_type: str,
        status: str,
        created_at: datetime,
    ) -> None:
        self.job_id = job_id
        self.job_type = job_type
        self.status = status
        self.progress_pct: int | None = None
        self.current_step: str | None = None
        self.started_at: datetime | None = None
        self.finished_at: datetime | None = None
        self.result: dict | None = None
        self.error: ApiErrorSchema | None = None
        self.created_at = created_at

    def is_expired(self) -> bool:
        """Retourne True si le job est terminal ET a dépassé la rétention."""
        if self.status not in ("succeeded", "failed", "cancelled"):
            return False
        if self.finished_at is None:
            return False
        age = time.time() - self.finished_at.timestamp()
        return age > JOB_RETENTION_S

    def to_status(self) -> AsyncJobStatus:
        """Convertit en schéma AsyncJobStatus sérialisable."""
        return AsyncJobStatus(
            job_id=self.job_id,
            job_type=self.job_type,
            status=self.status,
            progress_pct=self.progress_pct,
            current_step=self.current_step,
            started_at=self.started_at,
            finished_at=self.finished_at,
            result=self.result,
            error=self.error,
        )
