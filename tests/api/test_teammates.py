"""Tests unitaires — endpoints Coéquipiers (Slice 6).

Couvre :
- POST /players/{slug}/pages/teammates (200, schéma, options, teammates, solo_reference)
- POST avec selected_gamertags
- POST /players/unknown/pages/teammates (404)
"""

from __future__ import annotations

from datetime import datetime, timezone
from unittest.mock import patch

import pytest
from httpx import ASGITransport, AsyncClient

from apps.api.app.schemas.teammates import (
    TeammateKPIs,
    TeammateOption,
    TeammateRow,
    TeammatesPageResponse,
)

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

_NOW = datetime(2025, 3, 15, 20, 0, tzinfo=timezone.utc)


def _make_option(gamertag: str = "Buddy42", count: int = 30) -> TeammateOption:
    return TeammateOption(
        gamertag=gamertag,
        xuid="xuid-buddy42",
        encounter_count=count,
        last_seen_at=_NOW,
    )


def _make_kpis(matches: int = 30, wins: int = 18) -> TeammateKPIs:
    return TeammateKPIs(
        match_count=matches,
        wins=wins,
        kd_ratio=1.55,
        win_rate=round(wins / matches, 4),
        accuracy=49.2,
        kills_per_game=8.1,
        assists_per_game=3.4,
    )


def _make_teammate_row() -> TeammateRow:
    return TeammateRow(
        gamertag="Buddy42",
        xuid="xuid-buddy42",
        encounter_count=30,
        last_seen_at=_NOW,
        with_kpis=_make_kpis(30, 18),
        without_kpis=_make_kpis(90, 45),
    )


def _make_teammates_response(with_teammates: bool = True) -> TeammatesPageResponse:
    return TeammatesPageResponse(
        options=[_make_option()],
        teammates=[_make_teammate_row()] if with_teammates else [],
        solo_reference=_make_kpis(50, 25),
        total_matches=120,
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
# POST /players/{slug}/pages/teammates
# ===========================================================================


@pytest.mark.anyio
async def test_teammates_page_returns_200(client: AsyncClient) -> None:
    """En DEMO_MODE, l'endpoint teammates retourne 200."""
    mock_resp = _make_teammates_response()
    with patch(
        "apps.api.app.services.teammates_service.get_teammates_page", return_value=mock_resp
    ):
        resp = await client.post("/api/v1/players/demo/pages/teammates", json={})
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_teammates_schema_complete(client: AsyncClient) -> None:
    """La réponse contient tous les champs de TeammatesPageResponse."""
    mock_resp = _make_teammates_response()
    with patch(
        "apps.api.app.services.teammates_service.get_teammates_page", return_value=mock_resp
    ):
        resp = await client.post("/api/v1/players/demo/pages/teammates", json={})

    assert resp.status_code == 200
    data = resp.json()
    for key in ("options", "teammates", "solo_reference", "total_matches"):
        assert key in data, f"Clé manquante : {key}"


@pytest.mark.anyio
async def test_teammates_options_fields(client: AsyncClient) -> None:
    """Les options coéquipiers contiennent gamertag et encounter_count."""
    mock_resp = _make_teammates_response()
    with patch(
        "apps.api.app.services.teammates_service.get_teammates_page", return_value=mock_resp
    ):
        resp = await client.post("/api/v1/players/demo/pages/teammates", json={})

    options = resp.json()["options"]
    assert len(options) >= 1
    assert options[0]["gamertag"] == "Buddy42"
    assert options[0]["encounter_count"] == 30


@pytest.mark.anyio
async def test_teammates_rows_kpis(client: AsyncClient) -> None:
    """Les lignes TeammateRow contiennent with_kpis et without_kpis."""
    mock_resp = _make_teammates_response()
    with patch(
        "apps.api.app.services.teammates_service.get_teammates_page", return_value=mock_resp
    ):
        resp = await client.post("/api/v1/players/demo/pages/teammates", json={})

    teammates = resp.json()["teammates"]
    assert len(teammates) >= 1
    row = teammates[0]
    assert "with_kpis" in row
    assert "without_kpis" in row
    assert row["with_kpis"]["kd_ratio"] == pytest.approx(1.55)


@pytest.mark.anyio
async def test_teammates_solo_reference(client: AsyncClient) -> None:
    """La référence solo est sérialisée correctement."""
    mock_resp = _make_teammates_response()
    with patch(
        "apps.api.app.services.teammates_service.get_teammates_page", return_value=mock_resp
    ):
        resp = await client.post("/api/v1/players/demo/pages/teammates", json={})

    solo = resp.json()["solo_reference"]
    assert solo is not None
    assert solo["match_count"] == 50
    assert solo["win_rate"] == pytest.approx(0.5)


@pytest.mark.anyio
async def test_teammates_with_selected(client: AsyncClient) -> None:
    """Un filtre selected_gamertags est accepté sans erreur."""
    mock_resp = _make_teammates_response()
    payload = {"selected_gamertags": ["Buddy42"]}
    with patch(
        "apps.api.app.services.teammates_service.get_teammates_page", return_value=mock_resp
    ):
        resp = await client.post("/api/v1/players/demo/pages/teammates", json=payload)
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_teammates_total_matches(client: AsyncClient) -> None:
    """Le champ total_matches correspond à la valeur renvoyée."""
    mock_resp = _make_teammates_response()
    with patch(
        "apps.api.app.services.teammates_service.get_teammates_page", return_value=mock_resp
    ):
        resp = await client.post("/api/v1/players/demo/pages/teammates", json={})
    assert resp.json()["total_matches"] == 120


@pytest.mark.anyio
async def test_teammates_slug_not_found(client: AsyncClient) -> None:
    """Slug inconnu → 404."""
    resp = await client.post("/api/v1/players/unknown-xyz/pages/teammates", json={})
    assert resp.status_code == 404
