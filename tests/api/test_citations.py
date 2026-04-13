"""Tests unitaires — endpoint Citations (Slice 2B).

Couvre :
- POST /players/{slug}/pages/citations (réponse complète, 200)
- Structure de la réponse (commendations, medals, deltas)
- Comportement avec données vides
"""

from __future__ import annotations

from unittest.mock import patch

import pytest
from httpx import ASGITransport, AsyncClient

from apps.api.app.schemas.citations import (
    CitationsDeltas,
    CitationsPageResponse,
    CommendationSummary,
    MedalSummary,
)

# ---------------------------------------------------------------------------
# Helpers de construction de réponse mock
# ---------------------------------------------------------------------------


def _make_citations_response() -> CitationsPageResponse:
    return CitationsPageResponse(
        commendations=[
            CommendationSummary(
                key="killing_spree",
                label="Killing Spree",
                category="kill",
                current_value=42,
                mastery_pct=85.0,
            )
        ],
        medals_summary=[
            MedalSummary(
                medal_name_id=1001,
                name="Killing Spree",
                count_filtered=42,
                count_total=100,
            )
        ],
        deltas=CitationsDeltas(
            filtered_total=100,
            unfiltered_total=200,
            delta_count=100,
        ),
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
async def test_citations_returns_200(client: AsyncClient) -> None:
    """En DEMO_MODE, POST /pages/citations retourne 200."""
    mock_resp = _make_citations_response()
    with patch(
        "apps.api.app.services.citations_service.get_citations_page",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/citations", json={"filters": {}})
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_citations_schema_complete(client: AsyncClient) -> None:
    """La réponse contient commendations, medals, deltas."""
    mock_resp = _make_citations_response()
    with patch(
        "apps.api.app.services.citations_service.get_citations_page",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/citations", json={"filters": {}})
    assert resp.status_code == 200
    data = resp.json()
    assert "commendations" in data
    assert "medals_summary" in data
    assert "deltas" in data


@pytest.mark.anyio
async def test_citations_commendations_fields(client: AsyncClient) -> None:
    """Les champs d'une CommendationSummary sont correctement sérialisés."""
    mock_resp = _make_citations_response()
    with patch(
        "apps.api.app.services.citations_service.get_citations_page",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/citations", json={"filters": {}})
    data = resp.json()
    comm = data["commendations"][0]
    assert comm["key"] == "killing_spree"
    assert comm["current_value"] == 42
    assert comm["mastery_pct"] == 85.0


@pytest.mark.anyio
async def test_citations_empty_response(client: AsyncClient) -> None:
    """Une réponse vide est valide (pas d'erreur)."""
    empty_resp = CitationsPageResponse(
        commendations=[],
        medals_summary=[],
        deltas=CitationsDeltas(filtered_total=0, unfiltered_total=0, delta_count=0),
    )
    with patch(
        "apps.api.app.services.citations_service.get_citations_page",
        return_value=empty_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/citations", json={"filters": {}})
    assert resp.status_code == 200
    data = resp.json()
    assert data["commendations"] == []
    assert data["medals_summary"] == []
