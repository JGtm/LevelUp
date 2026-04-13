"""Router de synchronisation — Sprint 3.

Endpoints :
  POST /api/v1/sync/initial
"""

from __future__ import annotations

import structlog
from fastapi import APIRouter

from apps.api.app.core.errors import ApiError
from apps.api.app.schemas.common import AsyncJobStatus
from apps.api.app.schemas.sync import InitialSyncStartRequest

logger = structlog.get_logger(__name__)

router = APIRouter(tags=["sync"])


@router.post("/sync/initial", response_model=AsyncJobStatus, status_code=202)
def start_initial_sync(body: InitialSyncStartRequest) -> AsyncJobStatus:
    """Lance la synchronisation initiale des données Halo pour un joueur.

    Crée un job asynchrone ``initial_sync`` et retourne immédiatement.
    Utiliser ``GET /api/v1/jobs/{job_id}`` pour suivre la progression.

    Guard : retourne 403 si ``can_start_initial_sync=false`` dans app_settings.json.
    """
    from apps.api.app.services.bootstrap_service import _build_capabilities, _load_app_settings

    capabilities = _build_capabilities(_load_app_settings())
    if not capabilities.can_start_initial_sync:
        raise ApiError(
            403,
            "initial_sync_disabled",
            "La synchronisation initiale est désactivée sur cette instance.",
            retryable=False,
        )

    from apps.api.app.services.sync_service import start_initial_sync as _start

    job = _start(body)
    logger.info(
        "initial_sync_started",
        job_id=job.job_id,
        player_slug=body.player_slug,
        max_matches=body.max_matches,
    )
    return job
