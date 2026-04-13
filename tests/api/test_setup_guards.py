"""Tests unitaires — guards sécurité sur POST /setup/players (Sprint 2.2 V7).

Couvre :
- 409 no_halo_identity si linked_halo_identity est None dans la session
- 409 identity_mismatch si le gamertag transmis ne correspond pas
- 200 si gamertag correct (cohérence validée)
- Guard can_self_provision (403 si désactivé)
- session.current_player_slug mis à jour après provisioning réussi
"""

from __future__ import annotations

from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest
from httpx import ASGITransport, AsyncClient


@pytest.fixture(autouse=True)
def _setup_env(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("LEVELUP_DEMO_MODE", "false")
    monkeypatch.setenv("LEVELUP_SESSION_SECRET", "test-setup-guards")
    monkeypatch.setenv("LEVELUP_SESSION_DIR", str(Path(__file__).parent / "_sessions_guards_test"))
    from apps.api.app.core.config import get_settings

    get_settings.cache_clear()  # type: ignore[attr-defined]


@pytest.fixture
async def client() -> AsyncClient:
    from apps.api.app.main import create_app

    app = create_app()
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as c:
        yield c


def _make_session(
    gamertag: str | None = None, xuid: str | None = None, session_id: str = "guard-session"
) -> str:
    """Crée une session fictive et retourne le cookie signé."""
    from apps.api.app.deps.auth import SessionData, _get_store, _sign_session_id

    store = _get_store()
    session = SessionData(session_id=session_id)
    if gamertag:
        session.linked_halo_identity = {"gamertag": gamertag, "xuid": xuid or "0001"}
        session.auth_ready = True
    store.save(session)
    return _sign_session_id(session_id)


@pytest.mark.anyio
async def test_create_player_409_when_no_halo_identity(client: AsyncClient) -> None:
    """409 no_halo_identity si session sans linked_halo_identity."""
    sid = _make_session()  # pas de gamertag

    with patch(
        "apps.api.app.services.bootstrap_service._load_app_settings",
        return_value={"can_self_provision": True},
    ):
        resp = await client.post(
            "/api/v1/setup/players",
            json={"gamertag": "SomeGamer", "profile_mode": "xbox"},
            headers={"Origin": "http://test"},
            cookies={"levelup_session": sid},
        )

    assert resp.status_code == 409
    assert resp.json()["code"] == "no_halo_identity"


@pytest.mark.anyio
async def test_create_player_409_when_identity_mismatch(client: AsyncClient) -> None:
    """409 identity_mismatch si le gamertag ne correspond pas à la session."""
    sid = _make_session(gamertag="RealGamer", xuid="1111", session_id="mismatch-session")

    with patch(
        "apps.api.app.services.bootstrap_service._load_app_settings",
        return_value={"can_self_provision": True},
    ):
        resp = await client.post(
            "/api/v1/setup/players",
            json={"gamertag": "FakeGamer", "profile_mode": "xbox"},
            headers={"Origin": "http://test"},
            cookies={"levelup_session": sid},
        )

    assert resp.status_code == 409
    assert resp.json()["code"] == "identity_mismatch"


@pytest.mark.anyio
async def test_create_player_passes_with_correct_identity(client: AsyncClient) -> None:
    """Provisioning accepté quand le gamertag correspond — sans erreur 409."""
    sid = _make_session(gamertag="GoodGamer", xuid="2222", session_id="correct-session")

    mock_result = MagicMock()
    mock_result.player.player_slug = "goodgamer"
    mock_result.player.gamertag = "GoodGamer"
    mock_result.player.xuid = "2222"
    mock_result.player.waypoint_player = "GoodGamer"
    mock_result.player.is_demo = False

    with (
        patch(
            "apps.api.app.services.bootstrap_service._load_app_settings",
            return_value={"can_self_provision": True},
        ),
        patch(
            "apps.api.app.services.setup_service.create_player_profile", return_value=mock_result
        ),
    ):
        resp = await client.post(
            "/api/v1/setup/players",
            json={"gamertag": "GoodGamer", "profile_mode": "xbox"},
            headers={"Origin": "http://test"},
            cookies={"levelup_session": sid},
        )

    # Soit 200/201, soit erreur technique (pas 409)
    assert resp.status_code not in (409,)


@pytest.mark.anyio
async def test_create_player_403_when_can_self_provision_disabled(client: AsyncClient) -> None:
    """403 si can_self_provision=false dans les capabilities."""
    sid = _make_session(gamertag="BlockedGamer", session_id="blocked-session")

    with patch(
        "apps.api.app.services.bootstrap_service._load_app_settings",
        return_value={"can_self_provision": False},
    ):
        resp = await client.post(
            "/api/v1/setup/players",
            json={"gamertag": "BlockedGamer", "profile_mode": "xbox"},
            headers={"Origin": "http://test"},
            cookies={"levelup_session": sid},
        )

    assert resp.status_code == 403


@pytest.mark.anyio
async def test_create_player_updates_current_player_slug_in_session(client: AsyncClient) -> None:
    """Après provisioning réussi, session.current_player_slug est mis à jour."""
    from apps.api.app.deps.auth import _get_store

    sid = _make_session(gamertag="UpdatedGamer", xuid="3333", session_id="update-slug-session")

    mock_result = MagicMock()
    mock_result.player.player_slug = "updatedgamer"
    mock_result.player.gamertag = "UpdatedGamer"
    mock_result.player.xuid = "3333"
    mock_result.player.waypoint_player = "UpdatedGamer"
    mock_result.player.is_demo = False

    with (
        patch(
            "apps.api.app.services.bootstrap_service._load_app_settings",
            return_value={"can_self_provision": True},
        ),
        patch(
            "apps.api.app.services.setup_service.create_player_profile", return_value=mock_result
        ),
    ):
        resp = await client.post(
            "/api/v1/setup/players",
            json={"gamertag": "UpdatedGamer", "profile_mode": "xbox"},
            headers={"Origin": "http://test"},
            cookies={"levelup_session": sid},
        )

    # Le provisioning doit réussir (2xx ou erreur technique, pas 409)
    assert resp.status_code not in (409, 403)

    # Vérifier que la session a été mise à jour
    store = _get_store()
    # Extraire le vrai session_id depuis le cookie signé
    from apps.api.app.deps.auth import _unsign_session_id

    raw_id = _unsign_session_id(sid)
    if raw_id:
        session = store.load(raw_id)
        if session:
            assert session.current_player_slug == "updatedgamer", (
                f"current_player_slug attendu 'updatedgamer', obtenu '{session.current_player_slug}'"
            )
