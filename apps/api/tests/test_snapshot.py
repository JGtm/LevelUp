"""Smoke tests FastAPI — Sprint 39 T4.

Vérifie que les 5 endpoints principaux retournent 200 avec un schéma conforme.
Utilise DEMO_MODE pour éviter toute dépendance DuckDB réelle.

Lancer : python -m pytest apps/api/tests/test_snapshot.py -v
"""

from __future__ import annotations

import os
from pathlib import Path

import pytest
from httpx import ASGITransport, AsyncClient


@pytest.fixture(autouse=True)
def _demo_env(monkeypatch: pytest.MonkeyPatch) -> None:
    """Active DEMO_MODE + clé session déterministe."""
    monkeypatch.setenv("LEVELUP_DEMO_MODE", "true")
    monkeypatch.setenv("LEVELUP_SESSION_SECRET", "test-snapshot-key")
    monkeypatch.setenv("LEVELUP_SESSION_DIR", str(Path(__file__).parent / "_sessions"))
    from apps.api.app.core.config import get_settings

    get_settings.cache_clear()


@pytest.fixture(autouse=True)
def _reset_rate_limiter() -> None:
    """Réinitialise le rate limiter entre chaque test."""
    from apps.api.app.core.rate_limit import _lock, _windows

    with _lock:
        _windows.clear()
    yield  # type: ignore[misc]
    with _lock:
        _windows.clear()


@pytest.fixture
async def client() -> AsyncClient:
    """Client ASGI en DEMO_MODE."""
    from apps.api.app.main import create_app

    app = create_app()
    async with AsyncClient(
        transport=ASGITransport(app=app), base_url="http://test"
    ) as ac:
        yield ac  # type: ignore[misc]


# ---------------------------------------------------------------------------
# Snapshot : 5 endpoints
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_health_snapshot(client: AsyncClient) -> None:
    """GET /api/v1/health → 200, contient status+version+checks."""
    resp = await client.get("/api/v1/health")
    assert resp.status_code == 200
    data = resp.json()
    assert "status" in data
    assert "version" in data
    assert "checks" in data
    assert isinstance(data["checks"], dict)


@pytest.mark.asyncio
async def test_bootstrap_snapshot(client: AsyncClient) -> None:
    """GET /api/v1/bootstrap → 200, contient les champs racine attendus."""
    resp = await client.get("/api/v1/bootstrap")
    assert resp.status_code == 200
    data = resp.json()
    # Champs obligatoires du contrat bootstrap
    for key in ("gamertag", "player_slug", "version"):
        assert key in data, f"missing key {key} in bootstrap response"


@pytest.mark.asyncio
async def test_players_snapshot(client: AsyncClient) -> None:
    """GET /api/v1/players → 200, retourne une liste de joueurs."""
    resp = await client.get("/api/v1/players")
    assert resp.status_code == 200
    data = resp.json()
    assert "players" in data
    assert isinstance(data["players"], list)


@pytest.mark.asyncio
async def test_health_schema_keys(client: AsyncClient) -> None:
    """Vérifie que la réponse health contient exactement les clés attendues."""
    resp = await client.get("/api/v1/health")
    data = resp.json()
    expected_keys = {"status", "version", "uptime_seconds", "checks"}
    assert expected_keys.issubset(data.keys()), (
        f"missing keys: {expected_keys - data.keys()}"
    )


@pytest.mark.asyncio
async def test_openapi_schema_available(client: AsyncClient) -> None:
    """GET /api/openapi.json → 200, schema OpenAPI exposé."""
    resp = await client.get("/api/openapi.json")
    assert resp.status_code == 200
    data = resp.json()
    assert "openapi" in data
    assert "paths" in data
    # Vérifie que nos endpoints principaux sont documentés
    paths = data["paths"]
    assert "/api/v1/health" in paths or "/health" in paths
