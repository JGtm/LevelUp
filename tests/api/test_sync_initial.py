"""Tests unitaires — POST /sync/initial (Sprint 3.5 V7 onboarding).

Couvre :
- 202 avec job_id quand aucun job actif pour le joueur
- 409 sync_already_active si un job est déjà en cours pour ce joueur
- 403 si can_start_initial_sync=false
- active_sync_job_id stocké dans la session après démarrage
- job failed + error.code="sync_halo_api_error" + retryable=true
- job failed + error.code="sync_auth_expired"
- active_sync_job_id présent dans bootstrap pendant sync, absent après fin
"""

from __future__ import annotations

from pathlib import Path
from unittest.mock import patch

import pytest
from httpx import ASGITransport, AsyncClient


@pytest.fixture(autouse=True)
def _setup_env(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("LEVELUP_DEMO_MODE", "false")
    monkeypatch.setenv("LEVELUP_SESSION_SECRET", "test-sync-initial")
    monkeypatch.setenv("LEVELUP_SESSION_DIR", str(Path(__file__).parent / "_sessions_sync_test"))
    from apps.api.app.core.config import get_settings

    get_settings.cache_clear()  # type: ignore[attr-defined]


@pytest.fixture
def reset_job_store(tmp_path: pytest.TempPathFactory):
    """Vide le JobStore entre tests en utilisant un fichier temporaire isolé."""
    from apps.api.app.services.job_store import JobStore

    tmp_file = tmp_path / "jobs_test.json"

    with JobStore._lock:
        JobStore._instance = None
        # Injecter un store avec fichier temporaire vide
        JobStore._instance = JobStore(jobs_file=tmp_file)

    yield

    with JobStore._lock:
        if JobStore._instance:
            with JobStore._instance._jobs_lock:
                JobStore._instance._jobs.clear()
        JobStore._instance = None


@pytest.fixture
async def client() -> AsyncClient:
    from apps.api.app.main import create_app

    app = create_app()
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as c:
        yield c


@pytest.mark.anyio
async def test_sync_initial_202_starts_job(client: AsyncClient, reset_job_store: None) -> None:
    """POST /sync/initial retourne 202 et un job_id."""
    with (
        patch(
            "apps.api.app.services.bootstrap_service._load_app_settings",
            return_value={"can_start_initial_sync": True},
        ),
        patch("apps.api.app.services.sync_service._check_auth"),
        patch("apps.api.app.services.sync_service._fetch_and_sync", return_value=10),
        patch("apps.api.app.services.sync_service._run_finalize"),
        patch("apps.api.app.services.sync_service._write_initial_sync_marker"),
    ):
        resp = await client.post(
            "/api/v1/sync/initial",
            json={"player_slug": "testguy", "max_matches": 50},
            headers={"Origin": "http://test"},
        )

    assert resp.status_code == 202
    data = resp.json()
    assert "job_id" in data
    assert data["job_type"] == "initial_sync"
    assert data["status"] in ("queued", "running")


@pytest.mark.anyio
async def test_sync_initial_409_when_job_already_active(
    client: AsyncClient, reset_job_store: None
) -> None:
    """409 sync_already_active si un job initial_sync est déjà actif pour ce joueur."""
    from apps.api.app.services.job_store import JobStore

    # Injecter un job actif dans le store
    existing = JobStore.get().create("initial_sync", metadata={"player_slug": "testguy"})
    JobStore.get().update(existing.job_id, status="running")

    with patch(
        "apps.api.app.services.bootstrap_service._load_app_settings",
        return_value={"can_start_initial_sync": True},
    ):
        resp = await client.post(
            "/api/v1/sync/initial",
            json={"player_slug": "testguy", "max_matches": 50},
            headers={"Origin": "http://test"},
        )

    assert resp.status_code == 409
    data = resp.json()
    assert data["code"] == "sync_already_active"
    assert "active_job_id" in data.get("details", {})


@pytest.mark.anyio
async def test_sync_initial_403_when_disabled(client: AsyncClient, reset_job_store: None) -> None:
    """403 si can_start_initial_sync=false dans les capabilities."""
    with patch(
        "apps.api.app.services.bootstrap_service._load_app_settings",
        return_value={"can_start_initial_sync": False},
    ):
        resp = await client.post(
            "/api/v1/sync/initial",
            json={"player_slug": "testguy", "max_matches": 50},
            headers={"Origin": "http://test"},
        )

    assert resp.status_code == 403
    assert resp.json()["code"] == "initial_sync_disabled"


@pytest.mark.anyio
async def test_sync_job_fails_with_halo_api_error_code(reset_job_store: None) -> None:
    """Erreur API Halo → job failed + error.code='sync_halo_api_error' + retryable=True."""
    from apps.api.app.services.job_store import JobStore
    from apps.api.app.services.sync_service import _run_initial_sync_bg, _SyncHaloApiError

    store = JobStore.get()
    job = store.create("initial_sync", metadata={"player_slug": "testguy"})

    with (
        patch("apps.api.app.services.sync_service._validate_player"),
        patch("apps.api.app.services.sync_service._check_auth"),
        patch(
            "apps.api.app.services.sync_service._fetch_and_sync",
            side_effect=_SyncHaloApiError("Halo cloud timeout"),
        ),
    ):
        _run_initial_sync_bg(job.job_id, "testguy", 50, session_id=None)

    result = store.get_job(job.job_id)
    assert result is not None
    assert result.status == "failed"
    assert result.error is not None
    assert result.error.code == "sync_halo_api_error"
    assert result.error.retryable is True


@pytest.mark.anyio
async def test_sync_job_fails_with_auth_expired_error_code(reset_job_store: None) -> None:
    """Erreur auth expirée → job failed + error.code='sync_auth_expired'."""
    from apps.api.app.services.job_store import JobStore
    from apps.api.app.services.sync_service import _run_initial_sync_bg, _SyncAuthError

    store = JobStore.get()
    job = store.create("initial_sync", metadata={"player_slug": "testguy"})

    with (
        patch("apps.api.app.services.sync_service._validate_player"),
        patch(
            "apps.api.app.services.sync_service._check_auth",
            side_effect=_SyncAuthError("Token Halo expiré"),
        ),
    ):
        _run_initial_sync_bg(job.job_id, "testguy", 50, session_id=None)

    result = store.get_job(job.job_id)
    assert result is not None
    assert result.status == "failed"
    assert result.error is not None
    assert result.error.code == "sync_auth_expired"
    assert result.error.retryable is True


@pytest.mark.anyio
async def test_active_sync_job_id_in_bootstrap_during_sync(reset_job_store: None) -> None:
    """active_sync_job_id présent dans bootstrap pendant sync, absent après fin."""
    from httpx import ASGITransport, AsyncClient

    from apps.api.app.deps.auth import SessionData, _get_store, _sign_session_id
    from apps.api.app.main import create_app
    from apps.api.app.services.job_store import JobStore

    # Créer une session avec un job sync actif
    session_store = _get_store()
    session = SessionData(session_id="sync-active-session")
    job = JobStore.get().create("initial_sync", metadata={"player_slug": "testguy"})
    JobStore.get().update(job.job_id, status="running")
    session.active_sync_job_id = job.job_id
    session_store.save(session)
    signed = _sign_session_id("sync-active-session")

    try:
        app = create_app()
        async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as c:
            with (
                patch("apps.api.app.deps.players.load_db_profiles", return_value=[]),
                patch(
                    "apps.api.app.services.bootstrap_service._resolve_auth_state",
                    return_value="ready",
                ),
                patch(
                    "apps.api.app.services.bootstrap_service._has_any_synced_matches",
                    return_value=False,
                ),
                patch(
                    "apps.api.app.services.bootstrap_service._check_initial_sync_marker",
                    return_value=False,
                ),
                patch(
                    "apps.api.app.services.bootstrap_service._load_app_settings", return_value={}
                ),
            ):
                resp = await c.get(
                    "/api/v1/bootstrap",
                    headers={"Origin": "http://test"},
                    cookies={"levelup_session": signed},
                )
                assert resp.status_code == 200
                data = resp.json()
                assert data.get("active_sync_job_id") == job.job_id

            # Marquer le job comme terminé
            JobStore.get().update(job.job_id, status="succeeded")

            with (
                patch("apps.api.app.deps.players.load_db_profiles", return_value=[]),
                patch(
                    "apps.api.app.services.bootstrap_service._resolve_auth_state",
                    return_value="ready",
                ),
                patch(
                    "apps.api.app.services.bootstrap_service._has_any_synced_matches",
                    return_value=False,
                ),
                patch(
                    "apps.api.app.services.bootstrap_service._check_initial_sync_marker",
                    return_value=False,
                ),
                patch(
                    "apps.api.app.services.bootstrap_service._load_app_settings", return_value={}
                ),
            ):
                resp2 = await c.get(
                    "/api/v1/bootstrap",
                    headers={"Origin": "http://test"},
                    cookies={"levelup_session": signed},
                )
                assert resp2.status_code == 200
                assert resp2.json().get("active_sync_job_id") is None
    finally:
        import contextlib

        with contextlib.suppress(Exception):
            session_store._store_dir.joinpath("sync-active-session.json").unlink(missing_ok=True)
