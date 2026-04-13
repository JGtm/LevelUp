"""Tests unitaires — endpoints Match View + Last Match (Slices 4B + 4C).

Couvre :
- GET /players/{slug}/matches/{match_id} (200, 404)
- Structure de MatchViewResponse (header + 5 onglets)
- POST /players/{slug}/pages/last-match/resolve (200, 404)
- Structure de LastMatchResolveResponse
"""

from __future__ import annotations

from unittest.mock import patch

import pytest
from httpx import ASGITransport, AsyncClient

from apps.api.app.schemas.match_view import (
    LastMatchResolveResponse,
    MatchCitationsTab,
    MatchCombatTab,
    MatchMediaTab,
    MatchPersonalResult,
    MatchSummaryKpis,
    MatchSummaryTab,
    MatchTeamTab,
    MatchViewHeader,
    MatchViewRank,
    MatchViewResponse,
)

_MATCH_ID = "aaaabbbb-1234-5678-abcd-000000000001"


# ---------------------------------------------------------------------------
# Helpers de construction de réponse mock
# ---------------------------------------------------------------------------


def _make_match_view_response() -> MatchViewResponse:
    return MatchViewResponse(
        header=MatchViewHeader(
            match_id=_MATCH_ID,
            start_time_label="01/01/2025 18:00",
            outcome_label="Victoire",
            outcome_color="#22c55e",
            map_ui="Recharge",
            mode_ui="Slayer",
        ),
        rank=MatchViewRank(),
        summary_tab=MatchSummaryTab(
            kpis=MatchSummaryKpis(kills=12, deaths=5, assists=3, kda=2.4),
            personal_result=MatchPersonalResult(outcome_label="Victoire"),
            medals=[],
            citations=[],
        ),
        combat_tab=MatchCombatTab(weapon_kills=[], highlight_events=[], charts=[]),
        team_tab=MatchTeamTab(roster=[], scoreboard=[], nemesis=[], encounters=[]),
        media_tab=MatchMediaTab(media_items=[]),
        citations_tab=MatchCitationsTab(commendations=[], medals=[]),
    )


def _make_last_match_response() -> LastMatchResolveResponse:
    return LastMatchResolveResponse(
        current_match_id=_MATCH_ID,
        total_matches_in_scope=10,
        current_index=0,
        session_tracking_key=f"session__{_MATCH_ID}",
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
# Tests Match View (4B)
# ---------------------------------------------------------------------------


@pytest.mark.anyio
async def test_match_view_returns_200(client: AsyncClient) -> None:
    """Un match existant retourne HTTP 200."""
    mock_resp = _make_match_view_response()
    with patch(
        "apps.api.app.services.match_view_service.get_match_view",
        return_value=mock_resp,
    ):
        resp = await client.get(f"/api/v1/players/demo/matches/{_MATCH_ID}")
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_match_view_schema_complete(client: AsyncClient) -> None:
    """La réponse contient header + les 5 onglets."""
    mock_resp = _make_match_view_response()
    with patch(
        "apps.api.app.services.match_view_service.get_match_view",
        return_value=mock_resp,
    ):
        resp = await client.get(f"/api/v1/players/demo/matches/{_MATCH_ID}")
    data = resp.json()
    for key in ("header", "summary_tab", "combat_tab", "team_tab", "media_tab", "citations_tab"):
        assert key in data, f"Clé manquante : {key}"


@pytest.mark.anyio
async def test_match_view_header_fields(client: AsyncClient) -> None:
    """Les champs du header sont correctement sérialisés."""
    mock_resp = _make_match_view_response()
    with patch(
        "apps.api.app.services.match_view_service.get_match_view",
        return_value=mock_resp,
    ):
        resp = await client.get(f"/api/v1/players/demo/matches/{_MATCH_ID}")
    data = resp.json()
    header = data["header"]
    assert header["match_id"] == _MATCH_ID
    assert header["outcome_label"] == "Victoire"
    assert header["map_ui"] == "Recharge"


@pytest.mark.anyio
async def test_match_view_returns_404_when_not_found(client: AsyncClient) -> None:
    """L'endpoint retourne 404 si le service lève une erreur NotFound."""
    from apps.api.app.core.errors import ApiError

    with patch(
        "apps.api.app.services.match_view_service.get_match_view",
        side_effect=ApiError(404, "match_not_found", "Match introuvable"),
    ):
        resp = await client.get(f"/api/v1/players/demo/matches/{_MATCH_ID}")
    assert resp.status_code == 404


# ---------------------------------------------------------------------------
# Tests Last Match Resolve (4C)
# ---------------------------------------------------------------------------


@pytest.mark.anyio
async def test_last_match_resolve_returns_200(client: AsyncClient) -> None:
    """POST /pages/last-match/resolve retourne 200 si match trouvé."""
    mock_resp = _make_last_match_response()
    with patch(
        "apps.api.app.services.match_view_service.resolve_last_match",
        return_value=mock_resp,
    ):
        resp = await client.post(
            "/api/v1/players/demo/pages/last-match/resolve", json={"filters": {}}
        )
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_last_match_resolve_fields(client: AsyncClient) -> None:
    """La réponse contient current_match_id, total_matches_in_scope, session_tracking_key."""
    mock_resp = _make_last_match_response()
    with patch(
        "apps.api.app.services.match_view_service.resolve_last_match",
        return_value=mock_resp,
    ):
        resp = await client.post(
            "/api/v1/players/demo/pages/last-match/resolve", json={"filters": {}}
        )
    data = resp.json()
    assert data["current_match_id"] == _MATCH_ID
    assert data["total_matches_in_scope"] == 10
    assert "session_tracking_key" in data


@pytest.mark.anyio
async def test_last_match_resolve_returns_404_when_empty(client: AsyncClient) -> None:
    """L'endpoint retourne 404 si le scope de matchs est vide."""
    from apps.api.app.core.errors import ApiError

    with patch(
        "apps.api.app.services.match_view_service.resolve_last_match",
        side_effect=ApiError(404, "no_match_in_scope", "Aucun match dans le scope"),
    ):
        resp = await client.post(
            "/api/v1/players/demo/pages/last-match/resolve", json={"filters": {}}
        )
    assert resp.status_code == 404
