"""Tests unitaires — endpoint Comparaison de sessions (Slice 3C).

Couvre :
- POST /players/{slug}/pages/session-compare (200, structure)
- Présence de available_sessions et metrics
- session_a et session_b présents si données disponibles
- Réponse vide valide
"""

from __future__ import annotations

from unittest.mock import patch

import pytest
from httpx import ASGITransport, AsyncClient

from apps.api.app.schemas.timeseries import (
    SessionCompareEntry,
    SessionCompareMetricRow,
    SessionCompareResponse,
)

# ---------------------------------------------------------------------------
# Helpers de construction de réponse mock
# ---------------------------------------------------------------------------


def _make_session_entry(label: str) -> SessionCompareEntry:
    return SessionCompareEntry(
        session_label=label,
        start_time="2025-01-01T18:00:00",
        end_time="2025-01-01T20:00:00",
        total_matches=10,
        wins=6,
        losses=4,
        kda=1.35,
        performance_score=72.5,
        with_friends=True,
        dominant_category="Ranked",
    )


def _make_session_compare_response() -> SessionCompareResponse:
    return SessionCompareResponse(
        session_a=_make_session_entry("2025-01-01 18:00"),
        session_b=_make_session_entry("2025-01-02 18:00"),
        available_sessions=["2025-01-01 18:00", "2025-01-02 18:00"],
        metrics=[
            SessionCompareMetricRow(
                key="kd_ratio",
                label="K/D",
                value_a="1.35",
                value_b="1.20",
                delta="+0.15",
                winner="a",
            )
        ],
    )


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture(autouse=True)
def force_demo_env(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("LEVELUP_DEMO_MODE", "true")


@pytest.fixture
async def client() -> AsyncClient:
    from apps.api.app.core.config import get_settings
    from apps.api.app.main import create_app

    get_settings.cache_clear()  # type: ignore[attr-defined]
    app = create_app()
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as c:
        yield c


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


@pytest.mark.anyio
async def test_session_compare_returns_200(client: AsyncClient) -> None:
    """En DEMO_MODE, POST /pages/session-compare retourne 200."""
    mock_resp = _make_session_compare_response()
    with patch(
        "apps.api.app.services.session_compare_service.get_session_compare",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/session-compare", json={"filters": {}})
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_session_compare_schema_complete(client: AsyncClient) -> None:
    """La réponse contient available_sessions, metrics, session_a, session_b."""
    mock_resp = _make_session_compare_response()
    with patch(
        "apps.api.app.services.session_compare_service.get_session_compare",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/session-compare", json={"filters": {}})
    data = resp.json()
    assert "available_sessions" in data
    assert "metrics" in data
    assert "session_a" in data
    assert "session_b" in data


@pytest.mark.anyio
async def test_session_compare_available_sessions(client: AsyncClient) -> None:
    """available_sessions est une liste."""
    mock_resp = _make_session_compare_response()
    with patch(
        "apps.api.app.services.session_compare_service.get_session_compare",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/session-compare", json={"filters": {}})
    data = resp.json()
    assert isinstance(data["available_sessions"], list)
    assert len(data["available_sessions"]) == 2


@pytest.mark.anyio
async def test_session_compare_metric_fields(client: AsyncClient) -> None:
    """Les champs de SessionCompareMetricRow sont correctement sérialisés."""
    mock_resp = _make_session_compare_response()
    with patch(
        "apps.api.app.services.session_compare_service.get_session_compare",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/session-compare", json={"filters": {}})
    data = resp.json()
    metric = data["metrics"][0]
    assert metric["key"] == "kd_ratio"
    assert metric["winner"] == "a"
    assert metric["delta"] == "+0.15"


@pytest.mark.anyio
async def test_session_compare_empty_response_valid(client: AsyncClient) -> None:
    """Une réponse vide (pas assez de sessions) est valide."""
    empty_resp = SessionCompareResponse(available_sessions=[], metrics=[])
    with patch(
        "apps.api.app.services.session_compare_service.get_session_compare",
        return_value=empty_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/session-compare", json={"filters": {}})
    assert resp.status_code == 200
    data = resp.json()
    assert data["available_sessions"] == []
    assert data["metrics"] == []
    assert data["session_a"] is None
    assert data["session_b"] is None
