"""Router de synchronisation — Sprint 3.

Endpoints :
  POST /api/v1/sync/initial
"""

from __future__ import annotations

import structlog
from fastapi import APIRouter, Depends

from apps.api.app.core.csrf import require_same_origin
from apps.api.app.core.errors import ApiError
from apps.api.app.core.rate_limit import check_rate_limit
from apps.api.app.deps.auth import SessionData, get_or_create_session
from apps.api.app.schemas.common import AsyncJobStatus
from apps.api.app.schemas.sync import InitialSyncStartRequest

logger = structlog.get_logger(__name__)

router = APIRouter(tags=["sync"])


@router.post("/sync/initial", response_model=AsyncJobStatus, status_code=202)
def start_initial_sync(
    body: InitialSyncStartRequest,
    session_tuple: tuple[SessionData, bool] = Depends(get_or_create_session),
    _csrf: None = Depends(require_same_origin),
    _rl: None = Depends(check_rate_limit),
) -> AsyncJobStatus:
    """Lance la synchronisation initiale des données Halo pour un joueur.

    Crée un job asynchrone ``initial_sync`` et retourne immédiatement.
    Utiliser ``GET /api/v1/jobs/{job_id}`` pour suivre la progression.

    Guard 403 : ``can_start_initial_sync=false`` dans app_settings.json.
    Guard 409 : un job ``initial_sync`` est déjà actif pour ce joueur (single-flight).
    """
    from apps.api.app.services.bootstrap_service import _build_capabilities, _load_app_settings
    from apps.api.app.services.job_store import JobStore

    capabilities = _build_capabilities(_load_app_settings())
    if not capabilities.can_start_initial_sync:
        raise ApiError(
            403,
            "initial_sync_disabled",
            "La synchronisation initiale est désactivée sur cette instance.",
            retryable=False,
        )

    # Single-flight : rejeter si un job actif existe déjà pour ce joueur
    existing = JobStore.get().find_active_initial_sync(body.player_slug)
    if existing is not None:
        raise ApiError(
            409,
            "sync_already_active",
            "Une synchronisation initiale est déjà en cours pour ce joueur.",
            retryable=False,
            details={"active_job_id": existing.job_id},
        )

    session, _ = session_tuple
    from apps.api.app.services.sync_service import start_initial_sync as _start

    job = _start(body, session_id=session.session_id)
    logger.info(
        "initial_sync_started",
        job_id=job.job_id,
        player_slug=body.player_slug,
        max_matches=body.max_matches,
        session_id=session.session_id,
    )
    return job
