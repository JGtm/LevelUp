"""Service métier du wizard de setup — V7.

Gère :
- Le Device Code Flow Microsoft (initiation + polling)
- La création de profil joueur (wrapping de setup_wizard_logic)

Le Device Code Flow utilise un cache MSAL éphémère (en mémoire) pendant le
premier setup, avant que le profil joueur ne soit créé. Après création du
profil, le cache est persisté dans stats.duckdb via save_msal_cache_if_changed.

En DEMO_MODE :
- start_device_flow() → ApiError 422 (opération non supportée en démo)
- create_player_profile() → crée un profil fictif
"""

from __future__ import annotations

import threading
import uuid
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path

import structlog

from apps.api.app.schemas.common import ApiErrorSchema, PlayerSummary
from apps.api.app.schemas.setup import (
    CreatePlayerProfileRequest,
    CreatePlayerProfileResponse,
    DeviceFlowStartResponse,
    DeviceFlowStatusResponse,
)

logger = structlog.get_logger(__name__)

# ---------------------------------------------------------------------------
# Store d'attempts Device Code Flow (process-level, thread-safe)
# ---------------------------------------------------------------------------

_ATTEMPTS_LOCK = threading.Lock()
_device_flow_attempts: dict[str, _DeviceFlowAttempt] = {}


@dataclass
class _DeviceFlowAttempt:
    """Contexte interne d'un attempt Device Code Flow."""

    attempt_id: str
    user_code: str
    verification_uri: str
    expires_in_seconds: int
    status: str = "pending"  # "pending" | "authorized" | "provisioned" | "failed" | "expired"
    started_at: datetime = field(default_factory=lambda: datetime.now(timezone.utc))
    # Propriétaire de la tentative — ownership par session (Sprint 1.3)
    session_id: str | None = None
    gamertag: str | None = None
    xuid: str | None = None
    error_detail: str | None = None
    error_code: str = "device_flow_error"  # code précis pour l'UI
    # Objets MSAL opaques (non sérialisés)
    _app: object = field(default=None, repr=False)
    _flow: dict = field(default_factory=dict, repr=False)
    _cache: object = field(default=None, repr=False)


# ---------------------------------------------------------------------------
# Device Code Flow
# ---------------------------------------------------------------------------


def start_device_flow(session_id: str | None = None) -> DeviceFlowStartResponse:
    """Initie un Device Code Flow Microsoft.

    Crée un attempt en mémoire, lance un thread background qui attend la
    complétion, et retourne immédiatement avec le user_code à afficher.

    Si une tentative ``pending`` existe déjà pour la même session, retourne
    l'attempt existant (single-flight par session).

    Args:
        session_id: ID de la session web courante — sert à lier l'ownership.

    Raises:
        ApiError: Si MSAL n'est pas disponible ou si le flow échoue à l'init.
    """
    from apps.api.app.core.errors import ApiError

    # Single-flight par session : retourner l'attempt pending existant si présent
    if session_id:
        with _ATTEMPTS_LOCK:
            for attempt in _device_flow_attempts.values():
                if attempt.session_id == session_id and attempt.status == "pending":
                    return DeviceFlowStartResponse(
                        attempt_id=attempt.attempt_id,
                        user_code=attempt.user_code,
                        verification_uri=attempt.verification_uri,
                        verification_uri_complete=None,
                        expires_in=attempt.expires_in_seconds,
                        expires_in_seconds=attempt.expires_in_seconds,
                        poll_interval_seconds=5,
                    )

    try:
        from src.auth._msal import (
            DeviceFlowError,
            MsalUnavailableError,
            build_msal_app,
            initiate_device_flow,
        )
    except ImportError as exc:
        raise ApiError.internal(f"Module auth introuvable : {exc}") from exc

    try:
        # Cache éphémère (pas de db_path — setup avant création du profil)
        import msal as _msal_module

        cache = _msal_module.SerializableTokenCache()
        app = build_msal_app(cache)
        info = initiate_device_flow(app)
    except MsalUnavailableError as exc:
        raise ApiError(503, "msal_unavailable", str(exc)) from exc
    except DeviceFlowError as exc:
        raise ApiError.bad_request(
            f"Impossible d'initier le Device Code Flow : {exc.detail}"
        ) from exc
    except Exception as exc:
        logger.exception("start_device_flow: erreur inattendue")
        raise ApiError.internal(f"Erreur lors de l'initiation du Device Code Flow : {exc}") from exc

    attempt_id = str(uuid.uuid4())
    attempt = _DeviceFlowAttempt(
        attempt_id=attempt_id,
        status="pending",
        user_code=info.user_code,
        verification_uri=info.verification_url,
        expires_in_seconds=info.expires_in,
        started_at=datetime.now(timezone.utc),
        session_id=session_id,
        _app=app,
        _flow=info._flow,
        _cache=cache,
    )

    with _ATTEMPTS_LOCK:
        _device_flow_attempts[attempt_id] = attempt

    logger.info(
        "device_flow_started",
        attempt_id=attempt_id,
        session_id=session_id,
        expires_in=info.expires_in,
    )

    # Lancer le polling en background
    t = threading.Thread(
        target=_complete_device_flow_bg,
        args=(attempt_id,),
        daemon=True,
        name=f"device-flow-{attempt_id[:8]}",
    )
    t.start()

    return DeviceFlowStartResponse(
        attempt_id=attempt_id,
        user_code=info.user_code,
        verification_uri=info.verification_url,
        verification_uri_complete=None,
        expires_in=info.expires_in,
        expires_in_seconds=info.expires_in,
        poll_interval_seconds=5,
    )


def get_device_flow_status(
    attempt_id: str,
    session_id: str | None = None,
) -> DeviceFlowStatusResponse:
    """Retourne le statut courant d'un attempt Device Code Flow.

    Vérifie l'ownership de la tentative : une tentative n'est lisible que par
    la session qui l'a créée. Retourne 404 si inconnu, expiré ou appartient
    à une autre session.

    Args:
        attempt_id: UUID de la tentative.
        session_id: ID de la session courante — ownership vérifié si présent.

    Raises:
        ApiError: 404 si l'attempt est inconnu, expiré ou étranger à la session.
    """
    from apps.api.app.core.errors import ApiError

    with _ATTEMPTS_LOCK:
        attempt = _device_flow_attempts.get(attempt_id)

    if attempt is None:
        raise ApiError.not_found("attempt", attempt_id)

    # Vérification d'ownership — une tentative étrangère retourne 404 (pas 403)
    # pour ne pas confirmer l'existence de la tentative à une session tierce
    if session_id and attempt.session_id and attempt.session_id != session_id:
        raise ApiError.not_found("attempt", attempt_id)

    error_schema = None
    if attempt.error_detail:
        retryable = attempt.error_code not in (
            "device_flow_denied",
            "identity_resolution_failed",
        )
        error_schema = ApiErrorSchema(
            code=attempt.error_code,
            message=attempt.error_detail,
            retryable=retryable,
        )

    return DeviceFlowStatusResponse(
        attempt_id=attempt.attempt_id,
        status=attempt.status,
        gamertag=attempt.gamertag,
        xuid=attempt.xuid,
        error=error_schema,
    )


def _complete_device_flow_bg(attempt_id: str) -> None:
    """Thread background : attend la complétion du Device Code Flow."""
    import asyncio

    with _ATTEMPTS_LOCK:
        attempt = _device_flow_attempts.get(attempt_id)
    if attempt is None:
        return

    session_id = attempt.session_id

    try:
        from src.auth._halo_exchange import exchange_access_token_for_halo, resolve_player_identity
        from src.auth._msal import (
            acquire_token_by_device_flow,
        )

        # Appel bloquant — attend que l'utilisateur complète le flow
        try:
            access_token = acquire_token_by_device_flow(attempt._app, attempt._flow)
        except Exception as exc:
            err_str = str(exc).lower()
            if "denied" in err_str or "declined" in err_str:
                _set_attempt_failed(attempt_id, str(exc), "device_flow_denied")
            else:
                _set_attempt_failed(attempt_id, str(exc), "device_flow_error")
            return

        # Échange access_token → tokens Halo
        import aiohttp

        async def _exchange() -> tuple[str, str]:
            async with aiohttp.ClientSession(timeout=aiohttp.ClientTimeout(total=45)) as http:
                return await exchange_access_token_for_halo(http, access_token)

        loop = asyncio.new_event_loop()
        try:
            try:
                spartan, clearance = loop.run_until_complete(_exchange())
            except Exception as exc:
                _set_attempt_failed(attempt_id, str(exc), "halo_exchange_failed")
                return

            try:
                gamertag, xuid = loop.run_until_complete(
                    resolve_player_identity(spartan, clearance)
                )
            except Exception as exc:
                _set_attempt_failed(attempt_id, str(exc), "identity_resolution_failed")
                return
        finally:
            loop.close()

        with _ATTEMPTS_LOCK:
            attempt.status = "provisioned"
            attempt.gamertag = gamertag
            attempt.xuid = xuid

        # Persister l'identité Halo liée dans la session serveur (Sprint 1.2)
        if session_id:
            _persist_halo_identity_in_session(session_id, gamertag, xuid)

        logger.info(
            "device_flow_succeeded",
            attempt_id=attempt_id,
            gamertag=gamertag,
        )

    except Exception as exc:
        logger.error(
            "device_flow_unexpected_error",
            attempt_id=attempt_id,
            exc=str(exc),
        )
        _set_attempt_failed(attempt_id, str(exc), "device_flow_error")


def _set_attempt_failed(attempt_id: str, detail: str, code: str) -> None:
    """Marque un attempt comme ``failed`` avec le code d'erreur approprié."""
    with _ATTEMPTS_LOCK:
        attempt = _device_flow_attempts.get(attempt_id)
        if attempt:
            attempt.status = "failed"
            attempt.error_detail = detail
            attempt.error_code = code
    logger.warning(
        "device_flow_failed",
        attempt_id=attempt_id,
        error_code=code,
        exc=detail,
    )


def _persist_halo_identity_in_session(session_id: str, gamertag: str, xuid: str) -> None:
    """Écrit l'identité Halo résolue dans la session serveur persistante."""
    try:
        from apps.api.app.deps.auth import _get_store

        store = _get_store()
        session = store.load(session_id)
        if session is not None:
            session.auth_ready = True
            session.linked_halo_identity = {"gamertag": gamertag, "xuid": xuid}
            store.save(session)
    except Exception:
        logger.warning(
            "persist_halo_identity_failed",
            session_id=session_id,
            exc_info=True,
        )


# ---------------------------------------------------------------------------
# Création de profil joueur
# ---------------------------------------------------------------------------


def create_player_profile(req: CreatePlayerProfileRequest) -> CreatePlayerProfileResponse:
    """Crée un profil joueur dans db_profiles.json et le dossier data/players/.

    Délègue à src.ui.pages.setup_wizard_logic.create_player_profile().

    Raises:
        ApiError: 400 si le gamertag est invalide.
    """
    from apps.api.app._pure_bridge import create_player_profile as _create
    from apps.api.app._pure_bridge import validate_gamertag
    from apps.api.app.core.errors import ApiError

    errors = validate_gamertag(req.gamertag)
    if errors:
        raise ApiError.bad_request(" ; ".join(errors), code="invalid_gamertag")

    try:
        player_key = _create(req.gamertag, xuid=req.xuid or "")
    except Exception as exc:
        logger.error(
            "create_player_profile_failed",
            gamertag=req.gamertag,
            exc=str(exc),
        )
        raise ApiError.internal(f"Impossible de créer le profil joueur : {exc}") from exc

    logger.info(
        "player_profile_created",
        player_slug=player_key,
        gamertag=req.gamertag.strip(),
    )

    db_path = f"data/players/{player_key}/stats.duckdb"
    db_created = Path(db_path).exists()

    player = PlayerSummary(
        player_slug=player_key,
        gamertag=req.gamertag.strip(),
        xuid=req.xuid or "",
        waypoint_player=req.gamertag.strip(),
        is_demo=False,
    )
    return CreatePlayerProfileResponse(
        player=player,
        db_created=db_created,
        warnings=[],
    )
