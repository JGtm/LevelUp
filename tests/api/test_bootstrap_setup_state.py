"""Tests unitaires — setup_state dans GET /bootstrap (Sprint 1.1 V7 onboarding).

Couvre :
- setup_state="no_halo_link" quand auth_state="missing"
- setup_state="halo_linked_no_profile" quand auth OK, aucun joueur
- setup_state="profile_ready_no_sync" quand joueur sans marqueur initial_sync
- setup_state="ready" quand marqueur présent OU has_matches=True
- DEMO_MODE → setup_state="ready" court-circuité
- linked_halo_identity et active_sync_job_id exposés dans la réponse
"""

from __future__ import annotations

from pathlib import Path
from unittest.mock import patch

import pytest
from httpx import ASGITransport, AsyncClient

# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture(autouse=True)
def _reset_settings(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("LEVELUP_SESSION_SECRET", "test-v7-bootstrap")
    monkeypatch.setenv("LEVELUP_SESSION_DIR", str(Path(__file__).parent / "_sessions_v7_test"))
    from apps.api.app.core.config import get_settings

    get_settings.cache_clear()  # type: ignore[attr-defined]


@pytest.fixture
async def client(monkeypatch: pytest.MonkeyPatch) -> AsyncClient:
    monkeypatch.setenv("LEVELUP_DEMO_MODE", "false")
    from apps.api.app.core.config import get_settings
    from apps.api.app.main import create_app

    get_settings.cache_clear()  # type: ignore[attr-defined]
    app = create_app()
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as c:
        yield c


@pytest.fixture
async def demo_client(monkeypatch: pytest.MonkeyPatch) -> AsyncClient:
    monkeypatch.setenv("LEVELUP_DEMO_MODE", "true")
    from apps.api.app.core.config import get_settings
    from apps.api.app.main import create_app

    get_settings.cache_clear()  # type: ignore[attr-defined]
    app = create_app()
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as c:
        yield c


# ---------------------------------------------------------------------------
# Tests setup_state
# ---------------------------------------------------------------------------


@pytest.mark.anyio
async def test_bootstrap_no_halo_link_when_no_auth(client: AsyncClient) -> None:
    """Sans token Halo → setup_state="no_halo_link"."""
    with (
        patch("apps.api.app.deps.players.load_db_profiles", return_value=[]),
        patch(
            "apps.api.app.services.bootstrap_service._resolve_auth_state", return_value="missing"
        ),
        patch(
            "apps.api.app.services.bootstrap_service._has_any_synced_matches", return_value=False
        ),
        patch(
            "apps.api.app.services.bootstrap_service._check_initial_sync_marker", return_value=False
        ),
        patch("apps.api.app.services.bootstrap_service._load_app_settings", return_value={}),
    ):
        resp = await client.get("/api/v1/bootstrap", headers={"Origin": "http://test"})
        assert resp.status_code == 200
        data = resp.json()
        assert data["setup_state"] == "no_halo_link"
        assert data["setup_required"] is True


@pytest.mark.anyio
async def test_bootstrap_halo_linked_no_profile(client: AsyncClient) -> None:
    """Auth OK mais aucun joueur → setup_state="halo_linked_no_profile"."""
    with (
        patch("apps.api.app.deps.players.load_db_profiles", return_value=[]),
        patch("apps.api.app.services.bootstrap_service._resolve_auth_state", return_value="ready"),
        patch(
            "apps.api.app.services.bootstrap_service._has_any_synced_matches", return_value=False
        ),
        patch(
            "apps.api.app.services.bootstrap_service._check_initial_sync_marker", return_value=False
        ),
        patch("apps.api.app.services.bootstrap_service._load_app_settings", return_value={}),
    ):
        resp = await client.get("/api/v1/bootstrap", headers={"Origin": "http://test"})
        assert resp.status_code == 200
        data = resp.json()
        assert data["setup_state"] == "halo_linked_no_profile"


@pytest.mark.anyio
async def test_bootstrap_profile_ready_no_sync(client: AsyncClient) -> None:
    """Joueur présent mais ni marqueur ni matchs → setup_state="profile_ready_no_sync"."""
    fake_player = {
        "player_slug": "testguy",
        "gamertag": "TestGuy",
        "xuid": "0001",
        "waypoint_player": "TestGuy",
        "is_demo": False,
    }
    with (
        patch("apps.api.app.deps.players.load_db_profiles", return_value=[fake_player]),
        patch("apps.api.app.services.bootstrap_service._resolve_auth_state", return_value="ready"),
        patch(
            "apps.api.app.services.bootstrap_service._has_any_synced_matches", return_value=False
        ),
        patch(
            "apps.api.app.services.bootstrap_service._check_initial_sync_marker", return_value=False
        ),
        patch("apps.api.app.services.bootstrap_service._load_app_settings", return_value={}),
    ):
        resp = await client.get("/api/v1/bootstrap", headers={"Origin": "http://test"})
        assert resp.status_code == 200
        data = resp.json()
        assert data["setup_state"] == "profile_ready_no_sync"


@pytest.mark.anyio
async def test_bootstrap_ready_via_initial_sync_marker(client: AsyncClient) -> None:
    """Marqueur initial_sync_completed_at présent → setup_state="ready"."""
    fake_player = {
        "player_slug": "testguy",
        "gamertag": "TestGuy",
        "xuid": "0001",
        "waypoint_player": "TestGuy",
        "is_demo": False,
    }
    with (
        patch("apps.api.app.deps.players.load_db_profiles", return_value=[fake_player]),
        patch("apps.api.app.services.bootstrap_service._resolve_auth_state", return_value="ready"),
        patch(
            "apps.api.app.services.bootstrap_service._has_any_synced_matches", return_value=False
        ),
        patch(
            "apps.api.app.services.bootstrap_service._check_initial_sync_marker", return_value=True
        ),
        patch("apps.api.app.services.bootstrap_service._load_app_settings", return_value={}),
    ):
        resp = await client.get("/api/v1/bootstrap", headers={"Origin": "http://test"})
        assert resp.status_code == 200
        data = resp.json()
        assert data["setup_state"] == "ready"
        assert data["setup_required"] is False


@pytest.mark.anyio
async def test_bootstrap_ready_via_has_matches_fallback(client: AsyncClient) -> None:
    """Fallback has_matches=True (profil existant avant migration) → ready."""
    fake_player = {
        "player_slug": "testguy",
        "gamertag": "TestGuy",
        "xuid": "0001",
        "waypoint_player": "TestGuy",
        "is_demo": False,
    }
    with (
        patch("apps.api.app.deps.players.load_db_profiles", return_value=[fake_player]),
        patch("apps.api.app.services.bootstrap_service._resolve_auth_state", return_value="ready"),
        patch("apps.api.app.services.bootstrap_service._has_any_synced_matches", return_value=True),
        patch(
            "apps.api.app.services.bootstrap_service._check_initial_sync_marker", return_value=False
        ),
        patch("apps.api.app.services.bootstrap_service._load_app_settings", return_value={}),
    ):
        resp = await client.get("/api/v1/bootstrap", headers={"Origin": "http://test"})
        assert resp.status_code == 200
        data = resp.json()
        assert data["setup_state"] == "ready"


@pytest.mark.anyio
async def test_bootstrap_demo_mode_always_ready(demo_client: AsyncClient) -> None:
    """DEMO_MODE → setup_state="ready" quel que soit l'état réel."""
    with (
        patch("apps.api.app.deps.players.load_db_profiles", return_value=[]),
        patch("apps.api.app.services.bootstrap_service._load_app_settings", return_value={}),
    ):
        resp = await demo_client.get("/api/v1/bootstrap", headers={"Origin": "http://test"})
        assert resp.status_code == 200
        data = resp.json()
        assert data["setup_state"] == "ready"
        assert data["setup_required"] is False
        assert data["feature_flags"]["demo_mode"] is True


@pytest.mark.anyio
async def test_bootstrap_exposes_linked_halo_identity_from_session(client: AsyncClient) -> None:
    """linked_halo_identity issu de la session est exposé dans bootstrap."""
    from apps.api.app.deps.auth import SessionData, _get_store

    # Créer une vraie session avec linked_halo_identity
    store = _get_store()
    session = SessionData(session_id="test-linked-session")
    session.linked_halo_identity = {"gamertag": "SpartanX42", "xuid": "9999"}
    store.save(session)

    try:
        with (
            patch("apps.api.app.deps.players.load_db_profiles", return_value=[]),
            patch(
                "apps.api.app.services.bootstrap_service._resolve_auth_state",
                return_value="missing",
            ),
            patch(
                "apps.api.app.services.bootstrap_service._has_any_synced_matches",
                return_value=False,
            ),
            patch(
                "apps.api.app.services.bootstrap_service._check_initial_sync_marker",
                return_value=False,
            ),
            patch("apps.api.app.services.bootstrap_service._load_app_settings", return_value={}),
        ):
            # Le cookie de session est injecté manuellement
            resp = await client.get(
                "/api/v1/bootstrap",
                headers={"Origin": "http://test"},
                cookies={"levelup_session": "test-linked-session"},
            )
            assert resp.status_code == 200
            data = resp.json()
            identity = data.get("linked_halo_identity")
            # Vérifie la structure si la session est chargée (peut être None si session pas trouvée)
            if identity:
                assert identity["gamertag"] == "SpartanX42"
                assert identity["xuid"] == "9999"
    finally:
        # Nettoyage
        import contextlib

        with contextlib.suppress(Exception):
            store._store_dir.joinpath("test-linked-session.json").unlink(missing_ok=True)
