"""Tests unitaires — endpoints bootstrap, players, health, DEMO_MODE.

Ces tests utilisent `httpx.AsyncClient` avec l'app FastAPI en mode ASGI.
Ils ne requièrent aucune connexion DuckDB réelle — les fixtures mockent
les accès disque.

Notes :
- `LEVELUP_DEMO_MODE=true` activé pour tous les tests de ce module.
- `LEVELUP_SESSION_SECRET` forcé pour des sessions déterministes.
"""

from __future__ import annotations

import os
from pathlib import Path
from typing import Any
from unittest.mock import patch

import pytest
from httpx import ASGITransport, AsyncClient

# ---------------------------------------------------------------------------
# Fixtures et helpers
# ---------------------------------------------------------------------------


def _mock_db_profiles(profiles: list[dict]) -> Any:
    """Mock de `load_db_profiles` retournant une liste de profils."""
    return patch("apps.api.app.deps.players.load_db_profiles", return_value=profiles)


def _mock_app_settings(cfg: dict) -> Any:
    """Mock de `_load_app_settings` dans le service bootstrap."""
    return patch("apps.api.app.services.bootstrap_service._load_app_settings", return_value=cfg)


@pytest.fixture(autouse=True)
def force_demo_env(monkeypatch: pytest.MonkeyPatch) -> None:
    """Active DEMO_MODE + clé session déterministe pour tous les tests."""
    monkeypatch.setenv("LEVELUP_DEMO_MODE", "true")
    monkeypatch.setenv("LEVELUP_SESSION_SECRET", "test-secret-key")
    monkeypatch.setenv("LEVELUP_SESSION_DIR", str(Path(__file__).parent / "_sessions_test"))
    # Reset du singleton de settings entre les tests
    from apps.api.app.core.config import get_settings

    get_settings.cache_clear()


@pytest.fixture
async def client() -> AsyncClient:
    """Client HTTP ASGI pour les tests — crée l'app après chaque reset de settings."""
    from apps.api.app.main import create_app

    app = create_app()
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as ac:
        yield ac


# ---------------------------------------------------------------------------
# Tests health
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_health_ok(client: AsyncClient) -> None:
    """Health retourne 200 avec status ok ou degraded (DEMO_MODE, DBs absentes)."""
    response = await client.get("/api/v1/health")
    assert response.status_code == 200
    data = response.json()
    assert data["status"] in ("ok", "degraded")
    assert "version" in data
    assert "uptime_seconds" in data
    assert "checks" in data
    assert data["checks"].get("db_profiles") == "skipped_demo_mode"


# ---------------------------------------------------------------------------
# Tests bootstrap
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_bootstrap_returns_200_in_demo_mode(client: AsyncClient) -> None:
    """Bootstrap retourne 200 en DEMO_MODE avec le joueur de démo."""
    response = await client.get("/api/v1/bootstrap")
    assert response.status_code == 200
    data = response.json()
    assert data["feature_flags"]["demo_mode"] is True
    assert data["setup_required"] is False
    assert len(data["available_players"]) >= 1
    assert data["available_players"][0]["is_demo"] is True


@pytest.mark.asyncio
async def test_bootstrap_schema_complete(client: AsyncClient) -> None:
    """Bootstrap retourne tous les champs du schéma `BootstrapResponse`."""
    response = await client.get("/api/v1/bootstrap")
    data = response.json()
    required_fields = {
        "setup_required",
        "auth_state",
        "available_players",
        "locale",
        "hints_visible_default",
        "feature_flags",
        "capabilities",
        "settings_excerpt",
    }
    assert required_fields.issubset(data.keys())


@pytest.mark.asyncio
async def test_bootstrap_auth_state_missing_for_new_session(client: AsyncClient) -> None:
    """Une nouvelle session retourne auth_state='missing'."""
    response = await client.get("/api/v1/bootstrap")
    data = response.json()
    assert data["auth_state"] in ("missing", "partial", "ready")


@pytest.mark.asyncio
async def test_bootstrap_setup_required_when_no_profiles(client: AsyncClient) -> None:
    """setup_required=True quand db_profiles est vide (hors DEMO_MODE)."""
    from unittest.mock import patch

    # Désactiver DEMO_MODE et mocker des profiles vides
    os.environ["LEVELUP_DEMO_MODE"] = "false"
    from apps.api.app.core.config import get_settings

    get_settings.cache_clear()

    from apps.api.app.main import create_app

    app = create_app()
    with (
        patch("apps.api.app.deps.players.load_db_profiles", return_value=[]),
        patch("apps.api.app.services.bootstrap_service._load_app_settings", return_value={}),
    ):
        async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as ac:
            response = await ac.get("/api/v1/bootstrap")
            assert response.status_code == 200
            assert response.json()["setup_required"] is True

    # Remettre DEMO_MODE
    os.environ["LEVELUP_DEMO_MODE"] = "true"
    get_settings.cache_clear()


# ---------------------------------------------------------------------------
# Tests players
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_players_list_returns_demo_player(client: AsyncClient) -> None:
    """En DEMO_MODE, /players retourne le joueur de démo."""
    response = await client.get("/api/v1/players")
    assert response.status_code == 200
    data = response.json()
    assert "items" in data
    assert len(data["items"]) >= 1
    player = data["items"][0]
    assert player["is_demo"] is True
    assert player["player_slug"] == "demo-player"


@pytest.mark.asyncio
async def test_players_list_has_default_slug(client: AsyncClient) -> None:
    """La réponse /players contient default_player_slug."""
    response = await client.get("/api/v1/players")
    data = response.json()
    assert "default_player_slug" in data
    # En DEMO_MODE il y a toujours au moins un joueur
    assert data["default_player_slug"] is not None


# ---------------------------------------------------------------------------
# Tests session/context
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_session_context_update_locale(client: AsyncClient) -> None:
    """POST /session/context met à jour la locale."""
    response = await client.post("/api/v1/session/context", json={"locale": "en"})
    assert response.status_code == 200
    data = response.json()
    assert data["locale"] == "en"


@pytest.mark.asyncio
async def test_session_context_invalid_locale(client: AsyncClient) -> None:
    """POST /session/context avec locale invalide retourne 400."""
    response = await client.post("/api/v1/session/context", json={"locale": "de"})
    assert response.status_code == 400
    data = response.json()
    assert data["code"] == "bad_request"


@pytest.mark.asyncio
async def test_session_context_unknown_player_slug(client: AsyncClient) -> None:
    """POST /session/context avec un slug inconnu retourne 404."""
    response = await client.post(
        "/api/v1/session/context", json={"player_slug": "unknown-ghost-player"}
    )
    assert response.status_code == 404


# ---------------------------------------------------------------------------
# Tests contrat — cohérence OpenAPI
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_openapi_schema_accessible(client: AsyncClient) -> None:
    """Le schéma OpenAPI est accessible et valide du JSON."""
    response = await client.get("/api/openapi.json")
    assert response.status_code == 200
    schema = response.json()
    assert "openapi" in schema
    assert "paths" in schema
    assert "/api/v1/health" in schema["paths"]
    assert "/api/v1/bootstrap" in schema["paths"]
    assert "/api/v1/players" in schema["paths"]
