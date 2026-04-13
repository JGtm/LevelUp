"""Tests unitaires — endpoints Synthèse (Slice 7).

Couvre :
- POST /players/{slug}/pages/synthesis (200, schéma, KPIs solo/squad, comparaison, périodes)
- POST /players/unknown/pages/synthesis (404)
"""

from __future__ import annotations

from unittest.mock import patch

import pytest
from httpx import ASGITransport, AsyncClient

from apps.api.app.schemas.synthesis import (
    ComparisonMetricItem,
    SynthesisKPIs,
    SynthesisPageResponse,
)

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_kpis(matches: int = 60, wins: int = 33) -> SynthesisKPIs:
    return SynthesisKPIs(
        match_count=matches,
        wins=wins,
        kd_ratio=1.35,
        win_rate=round(wins / matches, 4),
        accuracy=47.8,
        kills_per_min=0.82,
        avg_life_seconds=42.3,
        performance_score=78.5,
    )


def _make_comparison_metrics() -> list[ComparisonMetricItem]:
    return [
        ComparisonMetricItem(
            label="K/D", solo_value=1.35, squad_value=1.65, solo_text="1.35", squad_text="1.65"
        ),
        ComparisonMetricItem(
            label="Win Rate", solo_value=55.0, squad_value=62.5, solo_text="55.0", squad_text="62.5"
        ),
    ]


def _make_synthesis_response(period: str = "all") -> SynthesisPageResponse:
    return SynthesisPageResponse(
        period=period,
        total_matches=120,
        solo_kpis=_make_kpis(60, 33),
        squad_kpis=_make_kpis(60, 37),
        comparison_metrics=_make_comparison_metrics(),
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


# ===========================================================================
# POST /players/{slug}/pages/synthesis
# ===========================================================================


@pytest.mark.anyio
async def test_synthesis_page_returns_200(client: AsyncClient) -> None:
    """En DEMO_MODE, l'endpoint synthesis retourne 200."""
    mock_resp = _make_synthesis_response()
    with patch(
        "apps.api.app.services.synthesis_service.get_synthesis_page", return_value=mock_resp
    ):
        resp = await client.post("/api/v1/players/demo/pages/synthesis", json={})
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_synthesis_schema_complete(client: AsyncClient) -> None:
    """La réponse contient tous les champs de SynthesisPageResponse."""
    mock_resp = _make_synthesis_response()
    with patch(
        "apps.api.app.services.synthesis_service.get_synthesis_page", return_value=mock_resp
    ):
        resp = await client.post("/api/v1/players/demo/pages/synthesis", json={})

    assert resp.status_code == 200
    data = resp.json()
    for key in ("period", "total_matches", "solo_kpis", "squad_kpis", "comparison_metrics"):
        assert key in data, f"Clé manquante : {key}"


@pytest.mark.anyio
async def test_synthesis_solo_kpis(client: AsyncClient) -> None:
    """Les KPIs solo contiennent kd_ratio et win_rate."""
    mock_resp = _make_synthesis_response()
    with patch(
        "apps.api.app.services.synthesis_service.get_synthesis_page", return_value=mock_resp
    ):
        resp = await client.post("/api/v1/players/demo/pages/synthesis", json={})

    kpis = resp.json()["solo_kpis"]
    assert kpis is not None
    assert kpis["kd_ratio"] == pytest.approx(1.35)
    assert kpis["match_count"] == 60


@pytest.mark.anyio
async def test_synthesis_squad_kpis(client: AsyncClient) -> None:
    """Les KPIs squad sont différents des KPIs solo."""
    mock_resp = _make_synthesis_response()
    with patch(
        "apps.api.app.services.synthesis_service.get_synthesis_page", return_value=mock_resp
    ):
        resp = await client.post("/api/v1/players/demo/pages/synthesis", json={})

    data = resp.json()
    assert data["squad_kpis"]["win_rate"] != data["solo_kpis"]["win_rate"]


@pytest.mark.anyio
async def test_synthesis_comparison_metrics(client: AsyncClient) -> None:
    """Les métriques comparatives contiennent label, solo_text et squad_text."""
    mock_resp = _make_synthesis_response()
    with patch(
        "apps.api.app.services.synthesis_service.get_synthesis_page", return_value=mock_resp
    ):
        resp = await client.post("/api/v1/players/demo/pages/synthesis", json={})

    metrics = resp.json()["comparison_metrics"]
    assert len(metrics) >= 1
    for m in metrics:
        assert "label" in m
        assert "solo_text" in m
        assert "squad_text" in m


@pytest.mark.anyio
async def test_synthesis_period_forwarded(client: AsyncClient) -> None:
    """La période passée dans le body est bien incluse dans la réponse."""
    mock_resp = _make_synthesis_response(period="1y")
    with patch(
        "apps.api.app.services.synthesis_service.get_synthesis_page", return_value=mock_resp
    ):
        resp = await client.post("/api/v1/players/demo/pages/synthesis", json={"period": "1y"})

    assert resp.json()["period"] == "1y"


@pytest.mark.anyio
async def test_synthesis_total_matches(client: AsyncClient) -> None:
    """Le total_matches correspond à la valeur renvoyée."""
    mock_resp = _make_synthesis_response()
    with patch(
        "apps.api.app.services.synthesis_service.get_synthesis_page", return_value=mock_resp
    ):
        resp = await client.post("/api/v1/players/demo/pages/synthesis", json={})
    assert resp.json()["total_matches"] == 120


@pytest.mark.anyio
async def test_synthesis_slug_not_found(client: AsyncClient) -> None:
    """Slug inconnu → 404."""
    resp = await client.post("/api/v1/players/unknown-xyz/pages/synthesis", json={})
    assert resp.status_code == 404
