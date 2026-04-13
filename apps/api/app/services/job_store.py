"""Store de jobs asynchrones avec persistance JSON — Sprint 3.

Gère le cycle de vie des jobs longs (smoke test, sync initiale, etc.)
avec un store process-level thread-safe et persistance sur disque.

Cycle de vie :
    queued → running → succeeded | failed | cancelled

Persistance : ``data/cache/jobs.json`` — sauvegarde après chaque mutation.
Au démarrage, les jobs ``running`` repris depuis le fichier passent à ``cancelled``
(le process qui les exécutait est mort).

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

import json
import logging
import threading
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path

from apps.api.app.schemas.common import ApiErrorSchema, AsyncJobStatus

logger = logging.getLogger(__name__)

# Rétention des jobs terminaux (secondes)
JOB_RETENTION_S = 3600  # 1 heure


def _default_jobs_file() -> Path:
    """Résout le chemin du fichier de persistance depuis la racine du repo."""
    return Path(__file__).resolve().parents[4] / "data" / "cache" / "jobs.json"


class JobStore:
    """Singleton process-level pour le suivi des jobs asynchrones."""

    _instance: JobStore | None = None
    _lock: threading.Lock = threading.Lock()

    def __init__(self, jobs_file: Path | None = None) -> None:
        self._jobs: dict[str, _JobEntry] = {}
        self._jobs_lock = threading.Lock()
        self._jobs_file: Path = jobs_file or _default_jobs_file()
        self._load()

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
        self._save()
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
        phase_key: str | None = None,
        phase_label: str | None = None,
        matches_done: int | None = None,
        matches_total: int | None = None,
        subtasks_done: int | None = None,
        subtasks_total: int | None = None,
        eta_seconds: int | None = None,
        warnings: list[str] | None = None,
        result: dict | None = None,
        error: ApiErrorSchema | None = None,
    ) -> bool:
        """Met à jour un job existant. Retourne False si job inconnu."""
        with self._jobs_lock:
            entry = self._jobs.get(job_id)
            if entry is None:
                return False
            _apply_status(entry, status)
            _apply_fields(
                entry,
                progress_pct=progress_pct,
                current_step=current_step,
                phase_key=phase_key,
                phase_label=phase_label,
                matches_done=matches_done,
                matches_total=matches_total,
                subtasks_done=subtasks_done,
                subtasks_total=subtasks_total,
                eta_seconds=eta_seconds,
                warnings=warnings,
                result=result,
                error=error,
            )
        self._save()
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
        if expired:
            self._save()
        return len(expired)

    # ------------------------------------------------------------------
    # Persistance JSON
    # ------------------------------------------------------------------

    def _save(self) -> None:
        """Sérialise tous les jobs dans ``_jobs_file``."""
        try:
            self._jobs_file.parent.mkdir(parents=True, exist_ok=True)
            with self._jobs_lock:
                payload = [e.to_dict() for e in self._jobs.values()]
            self._jobs_file.write_text(
                json.dumps(payload, indent=2, default=str),
                encoding="utf-8",
            )
        except Exception:  # noqa: BLE001
            logger.warning("job_store_save_failed", path=str(self._jobs_file))

    def _load(self) -> None:
        """Recharge les jobs depuis ``_jobs_file``.

        Les jobs ``running`` au moment du rechargement sont marqués ``cancelled``
        (le process qui les exécutait est mort).
        """
        if not self._jobs_file.exists():
            return
        try:
            raw = json.loads(self._jobs_file.read_text(encoding="utf-8"))
            cancelled_count = 0
            for d in raw:
                entry = _JobEntry.from_dict(d)
                if entry.status == "running":
                    entry.status = "cancelled"
                    now = datetime.now(timezone.utc)
                    entry.finished_at = entry.finished_at or now
                    cancelled_count += 1
                if not entry.is_expired():
                    self._jobs[entry.job_id] = entry
            if cancelled_count:
                logger.info(
                    "job_store_restart_cancelled",
                    count=cancelled_count,
                    hint="Jobs interrompus au redémarrage du processus.",
                )
        except Exception:  # noqa: BLE001
            logger.warning("job_store_load_failed", path=str(self._jobs_file))


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
        # Champs enrichis (Sprint 3)
        self.phase_key: str | None = None
        self.phase_label: str | None = None
        self.matches_done: int | None = None
        self.matches_total: int | None = None
        self.subtasks_done: int | None = None
        self.subtasks_total: int | None = None
        self.eta_seconds: int | None = None
        self.warnings: list[str] = []

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
            phase_key=self.phase_key,
            phase_label=self.phase_label,
            matches_done=self.matches_done,
            matches_total=self.matches_total,
            subtasks_done=self.subtasks_done,
            subtasks_total=self.subtasks_total,
            eta_seconds=self.eta_seconds,
            warnings=self.warnings,
        )

    def to_dict(self) -> dict:
        """Sérialise vers un dict JSON-compatible."""
        return {
            "job_id": self.job_id,
            "job_type": self.job_type,
            "status": self.status,
            "progress_pct": self.progress_pct,
            "current_step": self.current_step,
            "phase_key": self.phase_key,
            "phase_label": self.phase_label,
            "matches_done": self.matches_done,
            "matches_total": self.matches_total,
            "subtasks_done": self.subtasks_done,
            "subtasks_total": self.subtasks_total,
            "eta_seconds": self.eta_seconds,
            "warnings": self.warnings,
            "started_at": self.started_at.isoformat() if self.started_at else None,
            "finished_at": self.finished_at.isoformat() if self.finished_at else None,
            "result": self.result,
            "error": self.error.model_dump() if self.error else None,
            "created_at": self.created_at.isoformat(),
        }

    @classmethod
    def from_dict(cls, d: dict) -> _JobEntry:
        """Reconstruit depuis un dict JSON chargé depuis le fichier de persistance."""
        entry = cls(
            job_id=d["job_id"],
            job_type=d["job_type"],
            status=d["status"],
            created_at=datetime.fromisoformat(d["created_at"]),
        )
        entry.progress_pct = d.get("progress_pct")
        entry.current_step = d.get("current_step")
        entry.phase_key = d.get("phase_key")
        entry.phase_label = d.get("phase_label")
        entry.matches_done = d.get("matches_done")
        entry.matches_total = d.get("matches_total")
        entry.subtasks_done = d.get("subtasks_done")
        entry.subtasks_total = d.get("subtasks_total")
        entry.eta_seconds = d.get("eta_seconds")
        entry.warnings = d.get("warnings") or []
        entry.result = d.get("result")
        started = d.get("started_at")
        entry.started_at = datetime.fromisoformat(started) if started else None
        finished = d.get("finished_at")
        entry.finished_at = datetime.fromisoformat(finished) if finished else None
        err = d.get("error")
        entry.error = ApiErrorSchema.model_validate(err) if err else None
        return entry


# ---------------------------------------------------------------------------
# Helpers module-level (extraits de JobStore.update pour réduire la complexité)
# ---------------------------------------------------------------------------


def _apply_status(entry: _JobEntry, status: str | None) -> None:
    """Applique le changement de statut et met à jour les timestamps."""
    if status is None:
        return
    entry.status = status
    if status in ("succeeded", "failed", "cancelled"):
        entry.finished_at = datetime.now(timezone.utc)
    elif status == "running" and entry.started_at is None:
        entry.started_at = datetime.now(timezone.utc)


def _apply_fields(  # noqa: PLR0913
    entry: _JobEntry,
    *,
    progress_pct: int | None,
    current_step: str | None,
    phase_key: str | None,
    phase_label: str | None,
    matches_done: int | None,
    matches_total: int | None,
    subtasks_done: int | None,
    subtasks_total: int | None,
    eta_seconds: int | None,
    warnings: list[str] | None,
    result: dict | None,
    error: ApiErrorSchema | None,
) -> None:
    """Applique les mises à jour de champs scalaires d'une entrée de job."""
    updates: dict = {
        "progress_pct": progress_pct,
        "current_step": current_step,
        "phase_key": phase_key,
        "phase_label": phase_label,
        "matches_done": matches_done,
        "matches_total": matches_total,
        "subtasks_done": subtasks_done,
        "subtasks_total": subtasks_total,
        "eta_seconds": eta_seconds,
        "warnings": warnings,
        "result": result,
        "error": error,
    }
    for field, value in updates.items():
        if value is not None:
            setattr(entry, field, value)
