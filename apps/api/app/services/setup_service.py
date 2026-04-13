"""Service métier du wizard de setup — Slice 1.

Gère :
- L'évaluation de l'état de configuration (SetupStatusResponse)
- Le Device Code Flow Microsoft (initiation + polling)
- La création de profil joueur (wrapping de setup_wizard_logic)
- Le démarrage du smoke test (job asynchrone)

Le Device Code Flow utilise un cache MSAL éphémère (en mémoire) pendant le
premier setup, avant que le profil joueur ne soit créé. Après création du
profil, le cache est persisté dans stats.duckdb via save_msal_cache_if_changed.

En DEMO_MODE :
- get_setup_status() → needs_setup=False, next_blocking_step="done"
- start_device_flow() → ApiError 422 (opération non supportée en démo)
- create_player_profile() → crée un profil fictif
- start_smoke_test() → crée un job synthétique qui passe immédiatement
"""

from __future__ import annotations

import logging
import threading
import uuid
from dataclasses import dataclass, field
from datetime import datetime, timezone
from pathlib import Path

from apps.api.app.schemas.common import ApiErrorSchema, AsyncJobStatus, PlayerSummary
from apps.api.app.schemas.setup import (
    CreatePlayerProfileRequest,
    CreatePlayerProfileResponse,
    DeviceFlowStartResponse,
    DeviceFlowStatusResponse,
    SetupAuthInfo,
    SetupPlayerInfo,
    SetupStatusResponse,
    SmokeTestStartRequest,
)
from apps.api.app.services.job_store import JobStore

logger = logging.getLogger(__name__)

# ---------------------------------------------------------------------------
# Store d'attempts Device Code Flow (process-level, thread-safe)
# ---------------------------------------------------------------------------

_ATTEMPTS_LOCK = threading.Lock()
_device_flow_attempts: dict[str, _DeviceFlowAttempt] = {}


@dataclass
class _DeviceFlowAttempt:
    """Contexte interne d'un attempt Device Code Flow."""

    attempt_id: str
    status: str  # "pending" | "authorized" | "provisioned" | "failed" | "expired"
    user_code: str
    verification_uri: str
    expires_in_seconds: int
    started_at: datetime
    gamertag: str | None = None
    xuid: str | None = None
    error_detail: str | None = None
    # Objets MSAL opaques (non sérialisés)
    _app: object = field(default=None, repr=False)
    _flow: dict = field(default_factory=dict, repr=False)
    _cache: object = field(default=None, repr=False)


# ---------------------------------------------------------------------------
# get_setup_status
# ---------------------------------------------------------------------------


def get_setup_status() -> SetupStatusResponse:
    """Évalue l'état complet de la configuration de l'application.

    Délègue à src.utils.auth.get_auth_status() + comptage des players.
    """
    from src.utils.auth import get_auth_status

    auth_status = get_auth_status()
    player_count, default_slug = _count_players_with_default()

    preferred = _get_preferred_auth_method()

    auth_info = SetupAuthInfo(
        has_client_id=auth_status.has_client_id,
        has_refresh_token=auth_status.has_refresh_token,
        has_msal_cache=_has_any_msal_cache(),
        preferred_method=preferred,
    )
    player_info = SetupPlayerInfo(
        has_any_profile=player_count > 0,
        default_player_slug=default_slug,
    )

    needs_setup = not (auth_status.has_refresh_token and player_count > 0)
    next_step = _compute_next_step(auth_status.has_refresh_token, player_count > 0)

    return SetupStatusResponse(
        needs_setup=needs_setup,
        auth=auth_info,
        player=player_info,
        next_blocking_step=next_step,
    )


def get_setup_status_demo() -> SetupStatusResponse:
    """Version DEMO_MODE — retourne un état configuré sans accéder au disque."""
    return SetupStatusResponse(
        needs_setup=False,
        auth=SetupAuthInfo(
            has_client_id=True,
            has_refresh_token=True,
            has_msal_cache=False,
            preferred_method="refresh_token",
        ),
        player=SetupPlayerInfo(has_any_profile=True, default_player_slug="demo"),
        next_blocking_step="done",
    )


# ---------------------------------------------------------------------------
# Device Code Flow
# ---------------------------------------------------------------------------


def start_device_flow() -> DeviceFlowStartResponse:
    """Initie un Device Code Flow Microsoft.

    Crée un attempt en mémoire, lance un thread background qui attend la
    complétion, et retourne immédiatement avec le user_code à afficher.

    Raises:
        ApiError: Si MSAL n'est pas disponible ou si le flow échoue à l'init.
    """
    from apps.api.app.core.errors import ApiError

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
        _app=app,
        _flow=info._flow,
        _cache=cache,
    )

    with _ATTEMPTS_LOCK:
        _device_flow_attempts[attempt_id] = attempt

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
        expires_in_seconds=info.expires_in,
        poll_interval_seconds=5,
    )


def get_device_flow_status(attempt_id: str) -> DeviceFlowStatusResponse:
    """Retourne le statut courant d'un attempt Device Code Flow.

    Raises:
        ApiError: 404 si l'attempt est inconnu ou expiré.
    """
    from apps.api.app.core.errors import ApiError

    with _ATTEMPTS_LOCK:
        attempt = _device_flow_attempts.get(attempt_id)

    if attempt is None:
        raise ApiError.not_found("attempt", attempt_id)

    error_schema = None
    if attempt.error_detail:
        error_schema = ApiErrorSchema(
            code="device_flow_error",
            message=attempt.error_detail,
            retryable=False,
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

    try:
        from src.auth._halo_exchange import exchange_access_token_for_halo, resolve_player_identity
        from src.auth._msal import (
            acquire_token_by_device_flow,
        )

        # Appel bloquant — attend que l'utilisateur complète le flow
        access_token = acquire_token_by_device_flow(attempt._app, attempt._flow)

        # Échange access_token → tokens Halo
        import aiohttp

        async def _exchange() -> tuple[str, str]:
            async with aiohttp.ClientSession(timeout=aiohttp.ClientTimeout(total=45)) as session:
                return await exchange_access_token_for_halo(session, access_token)

        loop = asyncio.new_event_loop()
        try:
            spartan, clearance = loop.run_until_complete(_exchange())
            gamertag, xuid = loop.run_until_complete(resolve_player_identity(spartan, clearance))
        finally:
            loop.close()

        with _ATTEMPTS_LOCK:
            attempt.status = "provisioned"
            attempt.gamertag = gamertag
            attempt.xuid = xuid

        logger.info("device_flow_completed: gamertag=%s xuid=%s", gamertag, xuid)

    except Exception as exc:
        logger.exception("device_flow_bg_error: %s", exc)
        with _ATTEMPTS_LOCK:
            if _device_flow_attempts.get(attempt_id):
                _device_flow_attempts[attempt_id].status = "failed"
                _device_flow_attempts[attempt_id].error_detail = str(exc)


# ---------------------------------------------------------------------------
# Création de profil joueur
# ---------------------------------------------------------------------------


def create_player_profile(req: CreatePlayerProfileRequest) -> CreatePlayerProfileResponse:
    """Crée un profil joueur dans db_profiles.json et le dossier data/players/.

    Délègue à src.ui.pages.setup_wizard_logic.create_player_profile().

    Raises:
        ApiError: 400 si le gamertag est invalide.
    """
    from apps.api.app.core.errors import ApiError
    from src.ui.pages.setup_wizard_logic import create_player_profile as _create
    from src.ui.pages.setup_wizard_logic import validate_gamertag

    errors = validate_gamertag(req.gamertag)
    if errors:
        raise ApiError.bad_request(" ; ".join(errors), code="invalid_gamertag")

    try:
        player_key = _create(req.gamertag, xuid=req.xuid or "")
    except Exception as exc:
        logger.exception("create_player_profile: erreur")
        raise ApiError.internal(f"Impossible de créer le profil joueur : {exc}") from exc

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


# ---------------------------------------------------------------------------
# Smoke test
# ---------------------------------------------------------------------------


def start_smoke_test(req: SmokeTestStartRequest) -> AsyncJobStatus:
    """Lance le smoke test post-installation en tant que job asynchrone.

    Retourne immédiatement un AsyncJobStatus en statut ``queued``.
    Le job passe ensuite en ``running`` puis ``succeeded``/``failed``
    au fil de l'exécution dans un thread background.

    Args:
        req: Paramètres du smoke test (player_slug, max_matches, run_backfill).
    """
    store = JobStore.get()
    job = store.create("setup_smoke_test")

    t = threading.Thread(
        target=_run_smoke_test_bg,
        args=(job.job_id, req),
        daemon=True,
        name=f"smoke-test-{job.job_id[:8]}",
    )
    t.start()

    return job


def _run_smoke_test_bg(job_id: str, req: SmokeTestStartRequest) -> None:
    """Thread background pour l'exécution du smoke test."""
    store = JobStore.get()
    store.update(job_id, status="running", progress_pct=0, current_step="Démarrage")

    try:
        from apps.api.app.deps.players import load_db_profiles
        from src.ui.pages.setup_smoke_test_logic import (
            run_backfill_smoke_test,
            run_sync_smoke_test,
            verify_data_integrity,
        )

        # Résoudre le db_path depuis player_slug
        profiles_raw = load_db_profiles()
        db_path: str | None = None
        gamertag: str | None = None

        if isinstance(profiles_raw, list):
            for p in profiles_raw:
                if p.get("player_slug") == req.player_slug or p.get("gamertag") == req.player_slug:
                    db_path = p.get("db_path")
                    gamertag = p.get("gamertag", req.player_slug)
                    break
        else:
            prof = profiles_raw.get(req.player_slug, {})
            db_path = prof.get("db_path")
            gamertag = prof.get("waypoint_player", req.player_slug)

        if not db_path or not gamertag:
            store.update(
                job_id,
                status="failed",
                current_step="Erreur",
                error=ApiErrorSchema(
                    code="player_not_found",
                    message=f"Joueur '{req.player_slug}' introuvable dans db_profiles.json",
                    retryable=False,
                ),
            )
            return

        # Étape 1 : Sync
        store.update(job_id, progress_pct=10, current_step="Synchronisation des matchs")
        sync_ok, sync_msg = run_sync_smoke_test(
            gamertag=gamertag,
            db_path=db_path,
            max_matches=req.max_matches,
        )

        if not sync_ok:
            store.update(
                job_id,
                status="failed",
                progress_pct=20,
                current_step="Échec de la sync",
                error=ApiErrorSchema(code="sync_failed", message=sync_msg, retryable=True),
            )
            return

        # Étape 2 : Backfill (optionnel)
        if req.run_backfill:
            store.update(job_id, progress_pct=40, current_step="Backfill des enrichissements")
            _backfill_ok, _backfill_msg, _stats = run_backfill_smoke_test(gamertag)

        # Étape 3 : Vérification
        store.update(job_id, progress_pct=80, current_step="Vérification de l'intégrité")
        result = verify_data_integrity(gamertag=gamertag, db_path=db_path)

        final_status = "succeeded" if result.all_ok else "failed"
        store.update(
            job_id,
            status=final_status,
            progress_pct=100,
            current_step="Terminé",
            result={
                "all_ok": result.all_ok,
                "passed": result.passed,
                "total": result.total,
                "warnings": result.warnings,
                "sync_message": result.sync_message,
                "matches_synced": result.matches_synced,
            },
        )

    except Exception as exc:
        logger.exception("smoke_test_bg_error: %s", exc)
        store.update(
            job_id,
            status="failed",
            current_step="Erreur inattendue",
            error=ApiErrorSchema(
                code="internal_error",
                message=str(exc),
                retryable=True,
            ),
        )


# ---------------------------------------------------------------------------
# Helpers privés
# ---------------------------------------------------------------------------


def _count_players_with_default() -> tuple[int, str | None]:
    """Compte les joueurs existants et retourne (count, default_slug)."""
    from src.utils.paths import PLAYERS_DIR

    if not PLAYERS_DIR.exists():
        return 0, None

    slugs: list[str] = []
    for d in PLAYERS_DIR.iterdir():
        if d.is_dir() and (d / "stats.duckdb").exists():
            slugs.append(d.name)

    return len(slugs), slugs[0] if slugs else None


def _get_preferred_auth_method() -> str:
    """Retourne la méthode d'auth préférée selon app_settings.json."""
    try:
        from src.ui.settings import load_settings

        s = load_settings()
        return "device_code" if s.auth_method == "msal" else "refresh_token"
    except Exception:
        return "refresh_token"


def _has_any_msal_cache() -> bool:
    """Vérifie si un cache MSAL existe dans au moins une stats.duckdb joueur."""
    from src.utils.paths import PLAYERS_DIR

    if not PLAYERS_DIR.exists():
        return False

    try:
        from src.utils.db import duckdb_read_only

        for d in PLAYERS_DIR.iterdir():
            db = d / "stats.duckdb"
            if not db.exists():
                continue
            with duckdb_read_only(db) as conn:
                t = conn.execute(
                    "SELECT 1 FROM information_schema.tables WHERE table_name='sync_meta' LIMIT 1"
                ).fetchone()
                if not t:
                    continue
                row = conn.execute(
                    "SELECT value FROM sync_meta WHERE key='msal_token_cache' LIMIT 1"
                ).fetchone()
                if row and row[0]:
                    return True
    except Exception:
        pass

    return False


def _compute_next_step(has_refresh_token: bool, has_players: bool) -> str:
    """Calcule la prochaine étape bloquante du wizard."""
    if not has_refresh_token:
        return "auth"
    if not has_players:
        return "player"
    return "done"
