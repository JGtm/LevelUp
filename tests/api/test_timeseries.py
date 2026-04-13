"""Tests unitaires — endpoint Séries temporelles (Slice 3B).

Couvre :
- POST /players/{slug}/pages/timeseries (200, structure complète)
- Présence des 5 onglets
- KPI cards dans l'onglet résumé
- Réponse vide valide
"""

from __future__ import annotations

from unittest.mock import patch

import pytest
from httpx import ASGITransport, AsyncClient

from apps.api.app.schemas.timeseries import (
    TimeseriesCumulTab,
    TimeseriesDistributionsTab,
    TimeseriesFormTab,
    TimeseriesIntensityTab,
    TimeseriesKpiCard,
    TimeseriesPageResponse,
    TimeseriesSummaryTab,
)

# ---------------------------------------------------------------------------
# Helpers de construction de réponse mock
# ---------------------------------------------------------------------------


def _make_timeseries_response() -> TimeseriesPageResponse:
    return TimeseriesPageResponse(
        total_matches=50,
        summary_tab=TimeseriesSummaryTab(
            kpi_cards=[
                TimeseriesKpiCard(key="total_matches", label="Matchs", value="50"),
                TimeseriesKpiCard(key="win_rate", label="Victoires", value="55.0 %", color="green"),
                TimeseriesKpiCard(key="kda", label="KDA", value="1.42", color="green"),
            ]
        ),
        cumul_tab=TimeseriesCumulTab(),
        form_tab=TimeseriesFormTab(),
        intensity_tab=TimeseriesIntensityTab(),
        distributions_tab=TimeseriesDistributionsTab(correlations=[]),
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
async def test_timeseries_returns_200(client: AsyncClient) -> None:
    """En DEMO_MODE, POST /pages/timeseries retourne 200."""
    mock_resp = _make_timeseries_response()
    with patch(
        "apps.api.app.services.timeseries_api_service.get_timeseries_page",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/timeseries", json={"filters": {}})
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_timeseries_schema_has_5_tabs(client: AsyncClient) -> None:
    """La réponse contient les 5 onglets attendus."""
    mock_resp = _make_timeseries_response()
    with patch(
        "apps.api.app.services.timeseries_api_service.get_timeseries_page",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/timeseries", json={"filters": {}})
    data = resp.json()
    for key in ("summary_tab", "cumul_tab", "form_tab", "intensity_tab", "distributions_tab"):
        assert key in data, f"Onglet manquant : {key}"


@pytest.mark.anyio
async def test_timeseries_total_matches(client: AsyncClient) -> None:
    """total_matches est correctement sérialisé."""
    mock_resp = _make_timeseries_response()
    with patch(
        "apps.api.app.services.timeseries_api_service.get_timeseries_page",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/timeseries", json={"filters": {}})
    data = resp.json()
    assert data["total_matches"] == 50


@pytest.mark.anyio
async def test_timeseries_kpi_cards_present(client: AsyncClient) -> None:
    """L'onglet résumé contient des kpi_cards non vides."""
    mock_resp = _make_timeseries_response()
    with patch(
        "apps.api.app.services.timeseries_api_service.get_timeseries_page",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/timeseries", json={"filters": {}})
    data = resp.json()
    assert len(data["summary_tab"]["kpi_cards"]) == 3
    assert data["summary_tab"]["kpi_cards"][0]["key"] == "total_matches"


@pytest.mark.anyio
async def test_timeseries_empty_response_valid(client: AsyncClient) -> None:
    """Une réponse vide est valide (0 matchs)."""
    from apps.api.app.services.timeseries_api_service import _empty_response

    empty_resp = _empty_response()
    with patch(
        "apps.api.app.services.timeseries_api_service.get_timeseries_page",
        return_value=empty_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/timeseries", json={"filters": {}})
    assert resp.status_code == 200
    data = resp.json()
    assert data["total_matches"] == 0
    assert data["summary_tab"]["kpi_cards"] == []
