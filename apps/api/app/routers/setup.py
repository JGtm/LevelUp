"""Routers Setup + Auth — V7.

Endpoints :
  POST /api/v1/auth/device-flow/start
  GET  /api/v1/auth/device-flow/{attempt_id}
  POST /api/v1/setup/players
"""

from __future__ import annotations

import structlog
from fastapi import APIRouter, Depends, Request, Response

from apps.api.app.core.config import get_settings
from apps.api.app.core.csrf import require_same_origin
from apps.api.app.core.errors import ApiError
from apps.api.app.core.rate_limit import check_rate_limit
from apps.api.app.deps.auth import SessionData, _get_store, get_or_create_session
from apps.api.app.schemas.setup import (
    CreatePlayerProfileRequest,
    CreatePlayerProfileResponse,
    DeviceFlowStartResponse,
    DeviceFlowStatusResponse,
)

logger = structlog.get_logger(__name__)

router = APIRouter(tags=["setup"])


# ---------------------------------------------------------------------------
# Device Code Flow
# ---------------------------------------------------------------------------


@router.post("/auth/device-flow/start", response_model=DeviceFlowStartResponse)
def start_device_flow(
    _csrf: None = Depends(require_same_origin),
    _rl: None = Depends(check_rate_limit),
    session_tuple: tuple[SessionData, bool] = Depends(get_or_create_session),
) -> DeviceFlowStartResponse:
    """Initie un Device Code Flow Microsoft pour l'authentification Halo.

    Non disponible en DEMO_MODE.
    Retourne immédiatement avec ``user_code`` et ``verification_uri`` à afficher.
    Single-flight par session : retourne le flow en cours si une tentative
    ``pending`` existe déjà pour cette session.
    """
    settings = get_settings()

    if settings.demo_mode:
        raise ApiError(
            422,
            "demo_mode_unsupported",
            "Le Device Code Flow n'est pas disponible en mode démo.",
        )

    session, _ = session_tuple
    from apps.api.app.services.setup_service import start_device_flow as _start

    return _start(session_id=session.session_id)


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
    Retourne 404 si l'attempt est inconnu, expiré ou appartient à une autre session.

    Quand ``status == "provisioned"``, met à jour ``session.auth_ready = True``
    pour que le prochain appel à ``GET /bootstrap`` retourne ``auth_state="ready"``.
    L'identité Halo liée est écrite par le thread background via
    ``_persist_halo_identity_in_session`` (pas de double-écriture ici).
    """
    from apps.api.app.services.setup_service import get_device_flow_status as _get_status

    session, _ = session_tuple
    result = _get_status(attempt_id, session_id=session.session_id)

    # auth_ready via poll — double sécurité si le thread bg n'a pas encore écrit
    if result.status in ("authorized", "provisioned") and not session.auth_ready:
        session.auth_ready = True
        if result.gamertag and not session.linked_halo_identity:
            session.linked_halo_identity = {
                "gamertag": result.gamertag,
                "xuid": result.xuid or "",
            }
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
    session_tuple: tuple[SessionData, bool] = Depends(get_or_create_session),
) -> CreatePlayerProfileResponse:
    """Crée un profil joueur dans db_profiles.json.

    Guards :
    - 403 si can_self_provision=false dans app_settings.json
    - 409 si profile_mode="xbox" mais aucune identité Halo liée en session
    - 409 si gamertag/xuid ne correspondent pas à l'identité Halo liée

    Retourne 201 avec le PlayerSummary créé.
    Effets de bord : met à jour ``session.current_player_slug`` et transfère
    le cache MSAL vers la player DB.
    """
    from apps.api.app.services.bootstrap_service import _build_capabilities, _load_app_settings

    capabilities = _build_capabilities(_load_app_settings())
    if not capabilities.can_self_provision:
        logger.warning("provisioning_disabled", session_id=session_tuple[0].session_id)
        raise ApiError(
            403,
            "provisioning_disabled",
            "L'auto-provisioning est désactivé sur cette instance.",
            retryable=False,
        )

    session, _ = session_tuple

    # Guard Sprint 2.2 — identité Halo backend-authoritative en mode xbox
    if body.profile_mode == "xbox":
        if not session.linked_halo_identity:
            logger.warning("provisioning_no_halo_identity", session_id=session.session_id)
            raise ApiError(
                409,
                "no_halo_identity",
                "Vous devez d'abord vous connecter à Xbox via le Device Code Flow.",
                retryable=False,
            )
        linked_gt = session.linked_halo_identity.get("gamertag", "").lower()
        linked_xuid = session.linked_halo_identity.get("xuid", "")
        req_gt = body.gamertag.strip().lower()
        if req_gt != linked_gt:
            logger.warning(
                "provisioning_identity_mismatch",
                session_id=session.session_id,
                requested_gamertag=body.gamertag,
                linked_gamertag=session.linked_halo_identity.get("gamertag", ""),
            )
            raise ApiError(
                409,
                "identity_mismatch",
                "Le gamertag ne correspond pas à votre compte Xbox connecté.",
                retryable=False,
            )
        if body.xuid and linked_xuid and body.xuid != linked_xuid:
            logger.warning(
                "provisioning_identity_mismatch",
                session_id=session.session_id,
                reason="xuid_mismatch",
            )
            raise ApiError(
                409,
                "identity_mismatch",
                "Le XUID ne correspond pas à votre compte Xbox connecté.",
                retryable=False,
            )

    from apps.api.app.services.setup_service import create_player_profile as _create

    result = _create(body)

    # Sprint 2.4 — mettre à jour le joueur courant en session
    session.current_player_slug = result.player.player_slug

    # Sprint 2.3 — transférer le cache MSAL vers la player DB
    _transfer_msal_cache_to_player(
        session_id=session.session_id,
        player_slug=result.player.player_slug,
    )

    _get_store().save(session)

    logger.info(
        "player_provisioned",
        player_slug=result.player.player_slug,
        gamertag=result.player.gamertag,
    )
    return result


def _transfer_msal_cache_to_player(session_id: str, player_slug: str) -> None:
    """Transfère le cache MSAL de l'attempt Device Code Flow vers la player DB.

    Si aucun attempt match à la session, ou si la player DB n'existe pas encore,
    la fonction échoue silencieusement (log seulement).
    """
    try:
        from pathlib import Path

        from apps.api.app.core.config import get_settings
        from apps.api.app.services.setup_service import _ATTEMPTS_LOCK, _device_flow_attempts
        from src.auth._msal import save_msal_cache_if_changed  # type: ignore[import-untyped]

        cache = None
        with _ATTEMPTS_LOCK:
            for attempt in _device_flow_attempts.values():
                if attempt.session_id == session_id and attempt._cache is not None:
                    cache = attempt._cache
                    break

        if cache is None:
            logger.debug("msal_transfer_skipped_no_cache", player_slug=player_slug)
            return

        settings = get_settings()
        player_db = Path(settings.repo_root) / "data" / "players" / player_slug / "stats.duckdb"
        if not player_db.exists():
            logger.debug("msal_transfer_skipped_db_absent", player_slug=player_slug)
            return

        save_msal_cache_if_changed(str(player_db), cache)
        logger.info("msal_cache_transferred", player_slug=player_slug)
    except Exception:
        logger.warning("msal_transfer_failed", player_slug=player_slug, exc_info=True)
