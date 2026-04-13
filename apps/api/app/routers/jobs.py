"""Router Jobs — Slice 1.

Endpoint :
  GET /api/v1/jobs/{job_id}

Polling du statut des jobs asynchrones (smoke test, reindex médias, etc.).
Retourne 404 si le job est inconnu ou a dépassé la rétention (1 heure).
"""

from __future__ import annotations

import structlog
from fastapi import APIRouter

from apps.api.app.core.errors import ApiError
from apps.api.app.schemas.common import AsyncJobStatus

logger = structlog.get_logger(__name__)

router = APIRouter(tags=["jobs"])


@router.get("/jobs/{job_id}", response_model=AsyncJobStatus)
def get_job_status(job_id: str) -> AsyncJobStatus:
    """Retourne le statut courant d'un job asynchrone.

    Retourne 404 si le ``job_id`` est inconnu ou si le job a dépassé
    la fenêtre de rétention (1 heure après complétion).
    """
    from apps.api.app.services.job_store import JobStore

    store = JobStore.get()
    job = store.get_job(job_id)

    if job is None:
        raise ApiError.not_found("job", job_id)

    return job
