"""Router Settings — Slice 1.

Endpoints :
  GET   /api/v1/settings
  PATCH /api/v1/settings
  POST  /api/v1/settings/media/reset-index

``discord_webhook_url`` n'est jamais renvoyé côté client —
seul ``discord_webhook_url_present`` (bool) est exposé.
"""

from __future__ import annotations

import structlog
from fastapi import APIRouter

from apps.api.app.core.config import get_settings
from apps.api.app.schemas.common import AsyncJobStatus
from apps.api.app.schemas.settings import (
    MediaResetRequest,
    SettingsResponse,
    UpdateSettingsRequest,
)

logger = structlog.get_logger(__name__)

router = APIRouter(tags=["settings"])


@router.get("/settings", response_model=SettingsResponse)
def get_settings_endpoint() -> SettingsResponse:
    """Retourne les paramètres utilisateur persistés.

    En DEMO_MODE, retourne les valeurs par défaut (sans accéder au disque local).
    """
    cfg = get_settings()

    if cfg.demo_mode:
        from apps.api.app.services.settings_service import _to_response
        from src.ui.settings import AppSettings

        return _to_response(AppSettings())

    from apps.api.app.services.settings_service import load_api_settings

    return load_api_settings()


@router.patch("/settings", response_model=SettingsResponse)
def patch_settings(body: UpdateSettingsRequest) -> SettingsResponse:
    """Met à jour partiellement les paramètres utilisateur.

    Seuls les champs présents dans le corps sont mis à jour.
    Retourne les paramètres complets après persistance.
    Non disponible en DEMO_MODE.
    """
    cfg = get_settings()

    if cfg.demo_mode:
        from apps.api.app.core.errors import ApiError

        raise ApiError(
            422,
            "demo_mode_unsupported",
            "La modification des settings n'est pas disponible en mode démo.",
        )

    from apps.api.app.services.settings_service import update_api_settings

    return update_api_settings(body)


@router.post("/settings/media/reset-index", response_model=AsyncJobStatus, status_code=202)
def reset_media_index(body: MediaResetRequest) -> AsyncJobStatus:
    """Réinitialise l'index des médias (opération destructive).

    ``confirm_destructive`` doit être True pour autoriser l'opération.
    Si ``reindex_after_reset=True``, une réindexation est déclenchée après le reset.
    Retourne un job asynchrone à suivre via ``GET /jobs/{job_id}``.
    """
    from apps.api.app.core.errors import ApiError

    if not body.confirm_destructive:
        raise ApiError.bad_request(
            "confirm_destructive doit être True pour autoriser la réinitialisation de l'index.",
            code="confirmation_required",
        )

    import threading

    from apps.api.app.services.job_store import JobStore
    from apps.api.app.services.settings_service import reset_media_index as _reset

    store = JobStore.get()
    job = store.create("reindex_media")

    def _run() -> None:
        store.update(job.job_id, status="running", progress_pct=0, current_step="Reset index")
        try:
            _reset(reindex_after=body.reindex_after_reset)
            store.update(
                job.job_id,
                status="succeeded",
                progress_pct=100,
                current_step="Terminé",
            )
        except Exception as exc:
            logger.exception("media_reset_error")
            from apps.api.app.schemas.common import ApiErrorSchema

            store.update(
                job.job_id,
                status="failed",
                current_step="Erreur",
                error=ApiErrorSchema(
                    code="internal_error",
                    message=str(exc),
                    retryable=True,
                ),
            )

    threading.Thread(target=_run, daemon=True, name=f"media-reset-{job.job_id[:8]}").start()
    return job
