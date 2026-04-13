"""Routers Setup + Auth — Slice 1.

Endpoints :
  GET  /api/v1/setup/status
  POST /api/v1/auth/device-flow/start
  GET  /api/v1/auth/device-flow/{attempt_id}
  POST /api/v1/setup/players
  POST /api/v1/setup/smoke-test
"""

from __future__ import annotations

import structlog
from fastapi import APIRouter, Depends, Request, Response

from apps.api.app.core.config import get_settings
from apps.api.app.core.csrf import require_same_origin
from apps.api.app.core.errors import ApiError
from apps.api.app.core.rate_limit import check_rate_limit
from apps.api.app.deps.auth import SessionData, _get_store, get_or_create_session
from apps.api.app.schemas.common import AsyncJobStatus
from apps.api.app.schemas.setup import (
    CreatePlayerProfileRequest,
    CreatePlayerProfileResponse,
    DeviceFlowStartResponse,
    DeviceFlowStatusResponse,
    SetupStatusResponse,
    SmokeTestStartRequest,
)

logger = structlog.get_logger(__name__)

router = APIRouter(tags=["setup"])


# ---------------------------------------------------------------------------
# Setup status
# ---------------------------------------------------------------------------


@router.get("/setup/status", response_model=SetupStatusResponse)
def get_setup_status() -> SetupStatusResponse:
    """Retourne l'état courant de la configuration de l'application.

    La clé ``next_blocking_step`` pilote la navigation du wizard côté React.
    En DEMO_MODE, retourne toujours ``next_blocking_step="done"``.
    """
    settings = get_settings()

    if settings.demo_mode:
        from apps.api.app.services.setup_service import get_setup_status_demo

        return get_setup_status_demo()

    from apps.api.app.services.setup_service import get_setup_status as _get_status

    return _get_status()


# ---------------------------------------------------------------------------
# Device Code Flow
# ---------------------------------------------------------------------------


@router.post("/auth/device-flow/start", response_model=DeviceFlowStartResponse)
def start_device_flow(
    _csrf: None = Depends(require_same_origin),
    _rl: None = Depends(check_rate_limit),
) -> DeviceFlowStartResponse:
    """Initie un Device Code Flow Microsoft pour l'authentification Halo.

    Non disponible en DEMO_MODE.
    Retourne immédiatement avec ``user_code`` et ``verification_uri`` à afficher.
    """
    settings = get_settings()

    if settings.demo_mode:
        raise ApiError(
            422,
            "demo_mode_unsupported",
            "Le Device Code Flow n'est pas disponible en mode démo.",
        )

    from apps.api.app.services.setup_service import start_device_flow as _start

    return _start()


@router.get(
    "/auth/device-flow/{attempt_id}",
    response_model=DeviceFlowStatusResponse,
)
def get_device_flow_status(
    attempt_id: str,
    request: Request,
    response: Response,
    session_tuple: tuple[SessionData, bool] = Depends(get_or_create_session),
) -> DeviceFlowStatusResponse:
    """Retourne le statut d'un Device Code Flow en cours.

    Polling attendu toutes les ``poll_interval_seconds`` secondes (5s par défaut).
    Retourne 404 si l'attempt est inconnu ou a expiré.

    Quand ``status == "provisioned"``, met à jour ``session.auth_ready = True``
    pour que le prochain appel à ``GET /bootstrap`` retourne ``auth_state="ready"``.
    """
    from apps.api.app.services.setup_service import get_device_flow_status as _get_status

    result = _get_status(attempt_id)

    session, _ = session_tuple
    if result.status in ("authorized", "provisioned") and not session.auth_ready:
        session.auth_ready = True
        _get_store().save(session)

    return result


# ---------------------------------------------------------------------------
# Création de profil joueur
# ---------------------------------------------------------------------------


@router.post("/setup/players", response_model=CreatePlayerProfileResponse, status_code=201)
def create_player_profile(
    body: CreatePlayerProfileRequest,
    _csrf: None = Depends(require_same_origin),
    _rl: None = Depends(check_rate_limit),
) -> CreatePlayerProfileResponse:
    """Crée un profil joueur dans db_profiles.json.

    Validations :
    - gamertag requis, 1-50 caractères, alphanum+tirets+espaces
    - Si xuid fourni, il est enregistré directement (évite une résolution API)
    - Retourne 403 si can_self_provision=false dans app_settings.json

    Retourne 201 avec le PlayerSummary créé.
    """
    from apps.api.app.services.bootstrap_service import _build_capabilities, _load_app_settings

    capabilities = _build_capabilities(_load_app_settings())
    if not capabilities.can_self_provision:
        raise ApiError(
            403,
            "provisioning_disabled",
            "L'auto-provisioning est désactivé sur cette instance.",
            retryable=False,
        )

    from apps.api.app.services.setup_service import create_player_profile as _create

    return _create(body)


# ---------------------------------------------------------------------------
# Smoke test
# ---------------------------------------------------------------------------


@router.post("/setup/smoke-test", response_model=AsyncJobStatus, status_code=202)
def start_smoke_test(body: SmokeTestStartRequest) -> AsyncJobStatus:
    """Lance le smoke test post-installation en tant que job asynchrone.

    Retourne immédiatement un ``AsyncJobStatus`` en statut ``queued``.
    Utiliser ``GET /jobs/{job_id}`` pour suivre la progression.
    """
    settings = get_settings()

    if settings.demo_mode:
        # Mode démo : job synthétique qui passe instantanément
        from apps.api.app.services.job_store import JobStore

        store = JobStore.get()
        job = store.create("setup_smoke_test")
        store.update(
            job.job_id,
            status="succeeded",
            progress_pct=100,
            current_step="Terminé (démo)",
            result={"all_ok": True, "passed": 5, "total": 5, "warnings": 0},
        )
        return store.get_job(job.job_id) or job

    from apps.api.app.services.setup_service import start_smoke_test as _start

    return _start(body)
