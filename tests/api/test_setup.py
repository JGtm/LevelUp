"""Tests unitaires — endpoints Setup / Auth / Jobs (Slice 1).

Couvre :
- GET /setup/status (DEMO_MODE + modes sans credentials)
- POST /auth/device-flow/start (mock MSAL)
- GET /auth/device-flow/{attempt_id} (polling statut, 404 inconnu)
- POST /setup/players (création profil, validation gamertag)
- POST /setup/smoke-test (job asynchrone)
- GET /jobs/{job_id} (polling, 404 expiré)
"""

from __future__ import annotations

import uuid
from unittest.mock import patch

import pytest
from httpx import ASGITransport, AsyncClient

# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture(autouse=True)
def force_demo_env(monkeypatch: pytest.MonkeyPatch) -> None:
    """Active le DEMO_MODE par défaut pour tous les tests sauf override explicite."""
    monkeypatch.setenv("LEVELUP_DEMO_MODE", "true")


@pytest.fixture
def no_demo_env(monkeypatch: pytest.MonkeyPatch) -> None:
    """Désactive le DEMO_MODE pour les tests exigeant le mode normal."""
    monkeypatch.setenv("LEVELUP_DEMO_MODE", "false")


@pytest.fixture
async def client() -> AsyncClient:
    """Client HTTP asynchrone connecté à l'app FastAPI."""
    from apps.api.app.core.config import get_settings
    from apps.api.app.main import create_app

    get_settings.cache_clear()  # type: ignore[attr-defined]
    app = create_app()
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as c:
        yield c


# ===========================================================================
# GET /setup/status
# ===========================================================================


@pytest.mark.anyio
async def test_setup_status_demo_mode_returns_done(client: AsyncClient) -> None:
    """En DEMO_MODE, setup est toujours terminé."""
    resp = await client.get("/api/v1/setup/status")
    assert resp.status_code == 200
    data = resp.json()
    assert data["needs_setup"] is False
    assert data["next_blocking_step"] == "done"
    assert data["auth"]["has_client_id"] is True
    assert data["auth"]["has_refresh_token"] is True
    assert data["player"]["has_any_profile"] is True


@pytest.mark.anyio
async def test_setup_status_no_refresh_token(
    client: AsyncClient,
    no_demo_env: None,
) -> None:
    """Sans refresh_token, next_blocking_step = auth."""
    from apps.api.app.core.config import get_settings
    from apps.api.app.schemas.setup import SetupAuthInfo, SetupPlayerInfo, SetupStatusResponse

    get_settings.cache_clear()  # type: ignore[attr-defined]

    mock_response = SetupStatusResponse(
        needs_setup=True,
        auth=SetupAuthInfo(
            has_client_id=True,
            has_refresh_token=False,
            has_msal_cache=False,
            preferred_method="refresh_token",
        ),
        player=SetupPlayerInfo(has_any_profile=False, default_player_slug=None),
        next_blocking_step="auth",
    )

    with patch(
        "apps.api.app.services.setup_service.get_setup_status",
        return_value=mock_response,
    ):
        resp = await client.get("/api/v1/setup/status")

    assert resp.status_code == 200
    data = resp.json()
    assert data["needs_setup"] is True
    assert data["next_blocking_step"] == "auth"
    assert data["auth"]["has_refresh_token"] is False


@pytest.mark.anyio
async def test_setup_status_has_token_no_player(
    client: AsyncClient,
    no_demo_env: None,
) -> None:
    """Avec refresh_token mais sans joueur, next_blocking_step = player."""
    from apps.api.app.core.config import get_settings
    from apps.api.app.schemas.setup import SetupAuthInfo, SetupPlayerInfo, SetupStatusResponse

    get_settings.cache_clear()  # type: ignore[attr-defined]

    mock_response = SetupStatusResponse(
        needs_setup=True,
        auth=SetupAuthInfo(
            has_client_id=True,
            has_refresh_token=True,
            has_msal_cache=False,
            preferred_method="refresh_token",
        ),
        player=SetupPlayerInfo(has_any_profile=False, default_player_slug=None),
        next_blocking_step="player",
    )

    with patch(
        "apps.api.app.services.setup_service.get_setup_status",
        return_value=mock_response,
    ):
        resp = await client.get("/api/v1/setup/status")

    assert resp.status_code == 200
    data = resp.json()
    assert data["needs_setup"] is True
    assert data["next_blocking_step"] == "player"


@pytest.mark.anyio
async def test_setup_status_fully_configured(
    client: AsyncClient,
    no_demo_env: None,
) -> None:
    """Tout configuré → next_blocking_step = done."""
    from apps.api.app.core.config import get_settings
    from apps.api.app.schemas.setup import SetupAuthInfo, SetupPlayerInfo, SetupStatusResponse

    get_settings.cache_clear()  # type: ignore[attr-defined]

    mock_response = SetupStatusResponse(
        needs_setup=False,
        auth=SetupAuthInfo(
            has_client_id=True,
            has_refresh_token=True,
            has_msal_cache=False,
            preferred_method="refresh_token",
        ),
        player=SetupPlayerInfo(has_any_profile=True, default_player_slug="MyGamertag"),
        next_blocking_step="done",
    )

    with patch(
        "apps.api.app.services.setup_service.get_setup_status",
        return_value=mock_response,
    ):
        resp = await client.get("/api/v1/setup/status")

    assert resp.status_code == 200
    data = resp.json()
    assert data["needs_setup"] is False
    assert data["next_blocking_step"] == "done"
    assert data["player"]["default_player_slug"] == "MyGamertag"


# ===========================================================================
# POST /auth/device-flow/start
# ===========================================================================


@pytest.mark.anyio
async def test_device_flow_start_unavailable_in_demo(client: AsyncClient) -> None:
    """En DEMO_MODE, start device flow retourne 422."""
    resp = await client.post("/api/v1/auth/device-flow/start")
    assert resp.status_code == 422
    assert resp.json()["code"] == "demo_mode_unsupported"


@pytest.mark.anyio
async def test_device_flow_start_returns_user_code(
    client: AsyncClient,
    no_demo_env: None,
) -> None:
    """En mode normal, start device flow retourne attempt_id + user_code."""
    from apps.api.app.core.config import get_settings
    from apps.api.app.schemas.setup import DeviceFlowStartResponse

    get_settings.cache_clear()  # type: ignore[attr-defined]

    mock_response = DeviceFlowStartResponse(
        attempt_id="test-attempt-id-1234",
        user_code="ABCD-1234",
        verification_uri="https://microsoft.com/devicelogin",
        verification_uri_complete=None,
        expires_in_seconds=900,
        poll_interval_seconds=5,
    )

    with patch(
        "apps.api.app.services.setup_service.start_device_flow",
        return_value=mock_response,
    ):
        resp = await client.post("/api/v1/auth/device-flow/start")

    assert resp.status_code == 200
    data = resp.json()
    assert "attempt_id" in data
    assert data["user_code"] == "ABCD-1234"
    assert data["verification_uri"] == "https://microsoft.com/devicelogin"
    assert data["expires_in_seconds"] == 900
    assert data["poll_interval_seconds"] == 5


# ===========================================================================
# GET /auth/device-flow/{attempt_id}
# ===========================================================================


@pytest.mark.anyio
async def test_device_flow_poll_unknown_returns_404(client: AsyncClient) -> None:
    """Un attempt_id inconnu retourne 404."""
    resp = await client.get(f"/api/v1/auth/device-flow/{uuid.uuid4()}")
    assert resp.status_code == 404


@pytest.mark.anyio
async def test_device_flow_poll_pending(client: AsyncClient, no_demo_env: None) -> None:
    """Un attempt injecté manuellement retourne son statut."""
    from datetime import datetime, timezone

    from apps.api.app.core.config import get_settings
    from apps.api.app.services.setup_service import _device_flow_attempts, _DeviceFlowAttempt

    get_settings.cache_clear()  # type: ignore[attr-defined]

    attempt_id = str(uuid.uuid4())
    attempt = _DeviceFlowAttempt(
        attempt_id=attempt_id,
        status="pending",
        user_code="TEST-CODE",
        verification_uri="https://microsoft.com/devicelogin",
        expires_in_seconds=900,
        started_at=datetime.now(timezone.utc),
    )
    _device_flow_attempts[attempt_id] = attempt

    resp = await client.get(f"/api/v1/auth/device-flow/{attempt_id}")
    assert resp.status_code == 200
    data = resp.json()
    assert data["attempt_id"] == attempt_id
    assert data["status"] == "pending"
    assert data["gamertag"] is None

    # Nettoyer
    _device_flow_attempts.pop(attempt_id, None)


@pytest.mark.anyio
async def test_device_flow_poll_provisioned(client: AsyncClient, no_demo_env: None) -> None:
    """Un attempt complété retourne gamertag et xuid."""
    from datetime import datetime, timezone

    from apps.api.app.core.config import get_settings
    from apps.api.app.services.setup_service import _device_flow_attempts, _DeviceFlowAttempt

    get_settings.cache_clear()  # type: ignore[attr-defined]

    attempt_id = str(uuid.uuid4())
    attempt = _DeviceFlowAttempt(
        attempt_id=attempt_id,
        status="provisioned",
        user_code="TEST-CODE",
        verification_uri="https://microsoft.com/devicelogin",
        expires_in_seconds=900,
        started_at=datetime.now(timezone.utc),
        gamertag="TestGamertag",
        xuid="xuid_abc123",
    )
    _device_flow_attempts[attempt_id] = attempt

    resp = await client.get(f"/api/v1/auth/device-flow/{attempt_id}")
    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "provisioned"
    assert data["gamertag"] == "TestGamertag"
    assert data["xuid"] == "xuid_abc123"

    _device_flow_attempts.pop(attempt_id, None)


@pytest.mark.anyio
async def test_device_flow_provisioned_sets_auth_ready(
    client: AsyncClient, no_demo_env: None
) -> None:
    """Quand status=provisioned, la session passe à auth_ready=True."""
    from datetime import datetime, timezone

    from apps.api.app.core.config import get_settings
    from apps.api.app.deps.auth import _get_store
    from apps.api.app.services.setup_service import (
        _device_flow_attempts,
        _DeviceFlowAttempt,
    )

    get_settings.cache_clear()  # type: ignore[attr-defined]

    attempt_id = str(uuid.uuid4())
    attempt = _DeviceFlowAttempt(
        attempt_id=attempt_id,
        status="provisioned",
        user_code="TEST-CODE",
        verification_uri="https://microsoft.com/devicelogin",
        expires_in_seconds=900,
        started_at=datetime.now(timezone.utc),
        gamertag="TestGamertag",
        xuid="xuid_auth_test",
    )
    _device_flow_attempts[attempt_id] = attempt

    # Premier appel : récupère le cookie de session
    resp = await client.get(f"/api/v1/auth/device-flow/{attempt_id}")
    assert resp.status_code == 200

    # Extraire le session_id depuis le cookie et vérifier auth_ready
    cookie = resp.cookies.get("levelup_session")
    if cookie:
        from apps.api.app.deps.auth import _unsign_session_id

        session_id = _unsign_session_id(cookie)
        if session_id:
            store = _get_store()
            session = store.load(session_id)
            assert session is not None
            assert session.auth_ready is True

    _device_flow_attempts.pop(attempt_id, None)


# ===========================================================================
# POST /setup/players
# ===========================================================================


@pytest.mark.anyio
async def test_create_player_valid_gamertag(client: AsyncClient, no_demo_env: None) -> None:
    """Un gamertag valide crée un profil joueur."""
    from apps.api.app.core.config import get_settings
    from apps.api.app.schemas.common import PlayerSummary
    from apps.api.app.schemas.setup import CreatePlayerProfileResponse

    get_settings.cache_clear()  # type: ignore[attr-defined]

    mock_response = CreatePlayerProfileResponse(
        player=PlayerSummary(
            player_slug="TestGamertag",
            gamertag="TestGamertag",
            xuid="",
            waypoint_player="TestGamertag",
        ),
        db_created=False,
        warnings=[],
    )

    with patch(
        "apps.api.app.services.setup_service.create_player_profile",
        return_value=mock_response,
    ):
        resp = await client.post(
            "/api/v1/setup/players",
            json={"gamertag": "TestGamertag"},
        )

    assert resp.status_code == 201
    data = resp.json()
    assert data["player"]["gamertag"] == "TestGamertag"
    assert data["db_created"] is False


@pytest.mark.anyio
async def test_create_player_empty_gamertag_returns_422(
    client: AsyncClient,
    no_demo_env: None,
) -> None:
    """Un gamertag vide échoue à la validation Pydantic (422)."""
    from apps.api.app.core.config import get_settings

    get_settings.cache_clear()  # type: ignore[attr-defined]

    resp = await client.post("/api/v1/setup/players", json={"gamertag": ""})
    assert resp.status_code == 422


@pytest.mark.anyio
async def test_create_player_blocked_when_cant_self_provision(
    client: AsyncClient,
    no_demo_env: None,
) -> None:
    """POST /setup/players → 403 si can_self_provision=false dans app_settings."""
    from apps.api.app.core.config import get_settings

    get_settings.cache_clear()  # type: ignore[attr-defined]

    with patch(
        "apps.api.app.services.bootstrap_service._load_app_settings",
        return_value={"can_self_provision": False},
    ):
        resp = await client.post(
            "/api/v1/setup/players",
            json={"gamertag": "TestGamertag"},
        )

    assert resp.status_code == 403
    assert resp.json()["code"] == "provisioning_disabled"


@pytest.mark.anyio
async def test_create_player_invalid_gamertag_returns_400(
    client: AsyncClient,
    no_demo_env: None,
) -> None:
    """Un gamertag invalide (caractères interdits) retourne 400."""
    from apps.api.app.core.config import get_settings

    get_settings.cache_clear()  # type: ignore[attr-defined]

    # validate_gamertag est importé lazily dans create_player_profile ;
    # on doit patcher à la source.
    with patch(
        "src.ui.pages.setup_wizard_logic.validate_gamertag",
        return_value=["Le gamertag contient des caractères invalides."],
    ):
        resp = await client.post(
            "/api/v1/setup/players",
            json={"gamertag": "invalid@#$gamertag"},
        )

    assert resp.status_code == 400
    assert "gamertag" in resp.json()["code"]


# ===========================================================================
# POST /setup/smoke-test
# ===========================================================================


@pytest.mark.anyio
async def test_smoke_test_demo_mode_returns_succeeded_job(client: AsyncClient) -> None:
    """En DEMO_MODE, le smoke test retourne un job terminé immédiatement."""
    resp = await client.post(
        "/api/v1/setup/smoke-test",
        json={"player_slug": "demo", "max_matches": 10},
    )
    assert resp.status_code == 202
    data = resp.json()
    assert data["status"] == "succeeded"
    assert data["job_type"] == "setup_smoke_test"
    assert data["result"]["all_ok"] is True


@pytest.mark.anyio
async def test_smoke_test_no_demo_creates_queued_job(
    client: AsyncClient,
    no_demo_env: None,
) -> None:
    """En mode normal, le smoke test crée un job en statut queued ou running."""
    from apps.api.app.core.config import get_settings
    from apps.api.app.schemas.common import AsyncJobStatus

    get_settings.cache_clear()  # type: ignore[attr-defined]

    dummy_job = AsyncJobStatus(
        job_id=str(uuid.uuid4()),
        job_type="setup_smoke_test",
        status="queued",
        progress_pct=None,
        current_step=None,
        started_at=None,
        finished_at=None,
        result=None,
        error=None,
    )

    with patch(
        "apps.api.app.routers.setup.start_smoke_test",
        return_value=dummy_job,
    ):
        resp = await client.post(
            "/api/v1/setup/smoke-test",
            json={"player_slug": "MyPlayer", "max_matches": 20},
        )

    assert resp.status_code == 202
    data = resp.json()
    assert data["status"] in ("queued", "running")
    assert "job_id" in data


# ===========================================================================
# GET /jobs/{job_id}
# ===========================================================================


@pytest.mark.anyio
async def test_get_job_unknown_returns_404(client: AsyncClient) -> None:
    """Un job_id inconnu retourne 404."""
    resp = await client.get(f"/api/v1/jobs/{uuid.uuid4()}")
    assert resp.status_code == 404


@pytest.mark.anyio
async def test_get_job_existing_returns_status(client: AsyncClient) -> None:
    """Un job créé dans le store est accessible via GET /jobs/{job_id}."""
    from apps.api.app.services.job_store import JobStore

    store = JobStore.get()
    job = store.create("other")
    store.update(job.job_id, status="running", progress_pct=50, current_step="Étape 1")

    resp = await client.get(f"/api/v1/jobs/{job.job_id}")
    assert resp.status_code == 200
    data = resp.json()
    assert data["job_id"] == job.job_id
    assert data["status"] == "running"
    assert data["progress_pct"] == 50
    assert data["current_step"] == "Étape 1"


@pytest.mark.anyio
async def test_get_job_succeeded_returns_result(client: AsyncClient) -> None:
    """Un job terminé avec result est retourné correctement."""
    from apps.api.app.services.job_store import JobStore

    store = JobStore.get()
    job = store.create("setup_smoke_test")
    store.update(
        job.job_id,
        status="succeeded",
        progress_pct=100,
        current_step="Terminé",
        result={"all_ok": True, "passed": 8, "total": 8},
    )

    resp = await client.get(f"/api/v1/jobs/{job.job_id}")
    assert resp.status_code == 200
    data = resp.json()
    assert data["status"] == "succeeded"
    assert data["result"]["all_ok"] is True
    assert data["result"]["passed"] == 8
