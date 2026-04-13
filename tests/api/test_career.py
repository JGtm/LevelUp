"""Tests unitaires — endpoints Carrière (Slice 2).

Couvre :
- GET /players/{player_slug}/pages/career (page complète, données manquantes, 404)
- GET /players/{player_slug}/pages/career/top-matches
- GET /players/{player_slug}/pages/career/encounters
"""

from __future__ import annotations

from datetime import datetime, timezone
from unittest.mock import patch

import pytest
from httpx import ASGITransport, AsyncClient

from apps.api.app.schemas.career import (
    CareerCharts,
    CareerEncounter,
    CareerEncountersResponse,
    CareerHistoryPoint,
    CareerLusrSection,
    CareerPageResponse,
    CareerProjections,
    CareerSummary,
    CareerTopMatch,
    CareerTopMatchesResponse,
    HeroProgress,
)

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_summary() -> CareerSummary:
    return CareerSummary(
        rank_number=50,
        rank_label="Lance Corporal - Or II",
        rank_name_raw="Lance Corporal",
        rank_tier="Gold",
        current_xp=1200,
        xp_for_next_rank=5000,
        xp_total=1_200_000,
        progress_pct=24.0,
        is_max_rank=False,
        recorded_at=datetime(2025, 1, 15, 12, 0, tzinfo=timezone.utc),
    )


def _make_hero_progress() -> HeroProgress:
    return HeroProgress(
        xp_total_required=9_319_350,
        xp_remaining=8_119_350,
        percentage=12.88,
        current_rank=50,
    )


def _make_projections() -> CareerProjections:
    return CareerProjections(
        xp_per_day_active=1500.0,
        xp_per_day_fallback=800.0,
        estimated_hero_date=None,
        estimated_rank_cap_date=None,
    )


def _make_top_match() -> CareerTopMatch:
    return CareerTopMatch(
        match_id="abc-123",
        start_time=datetime(2025, 1, 10, 20, 0, tzinfo=timezone.utc),
        map_ui="Recharge",
        mode_ui="Slayer",
        playlist_label="Ranked Arena",
        performance_score=None,
        badge_type="dominant",
        score_label="50-32",
        outcome_label="Victoire",
    )


def _make_encounter() -> CareerEncounter:
    return CareerEncounter(
        encounter_key="xuid-999",
        opponent_gamertag="Rival42",
        count_matches=25,
        wins=12,
        losses=13,
        last_seen_at=datetime(2025, 1, 20, 18, 0, tzinfo=timezone.utc),
    )


def _make_career_page_response(*, no_data: bool = False) -> CareerPageResponse:
    if no_data:
        return CareerPageResponse(
            summary=None,
            hero_progress=None,
            projections=None,
            charts=CareerCharts(),
            xp_history=[],
            lusr=None,
            top_matches_preview=[],
            encounters_preview=[],
        )
    return CareerPageResponse(
        summary=_make_summary(),
        hero_progress=_make_hero_progress(),
        projections=_make_projections(),
        charts=CareerCharts(),
        xp_history=[
            CareerHistoryPoint(
                recorded_at=datetime(2025, 1, 1, 0, 0, tzinfo=timezone.utc),
                rank=48,
                xp_total=1_100_000,
            ),
        ],
        lusr=CareerLusrSection(
            current_rating=1450.0,
            current_tier_label="Onyx",
            current_playlist_group="Ranked Arena",
            trend_label="+25",
            checkpoints=[],
        ),
        top_matches_preview=[_make_top_match()],
        encounters_preview=[_make_encounter()],
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
# GET /players/{slug}/pages/career
# ===========================================================================


@pytest.mark.anyio
async def test_career_page_returns_200(client: AsyncClient) -> None:
    """En DEMO_MODE, l'endpoint career retourne 200."""
    mock_resp = _make_career_page_response()
    with patch("apps.api.app.services.career_service.get_career_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/career")
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_career_page_schema_complete(client: AsyncClient) -> None:
    """La réponse contient tous les champs de CareerPageResponse."""
    mock_resp = _make_career_page_response()
    with patch("apps.api.app.services.career_service.get_career_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/career")

    assert resp.status_code == 200
    data = resp.json()
    for key in ("summary", "hero_progress", "projections", "charts", "xp_history", "lusr"):
        assert key in data, f"Clé manquante : {key}"


@pytest.mark.anyio
async def test_career_summary_fields(client: AsyncClient) -> None:
    """Les champs de CareerSummary sont correctement sérialisés."""
    mock_resp = _make_career_page_response()
    with patch("apps.api.app.services.career_service.get_career_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/career")

    data = resp.json()
    summary = data["summary"]
    assert summary["rank_number"] == 50
    assert summary["rank_label"] == "Lance Corporal - Or II"
    assert summary["current_xp"] == 1200
    assert summary["xp_total"] == 1_200_000
    assert summary["is_max_rank"] is False
    assert summary["progress_pct"] == 24.0


@pytest.mark.anyio
async def test_career_hero_progress_fields(client: AsyncClient) -> None:
    """HeroProgress contient les champs XP_HERO_TOTAL et percentage."""
    mock_resp = _make_career_page_response()
    with patch("apps.api.app.services.career_service.get_career_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/career")

    hero = resp.json()["hero_progress"]
    assert hero["xp_total_required"] == 9_319_350
    assert hero["xp_remaining"] == 8_119_350
    assert hero["current_rank"] == 50


@pytest.mark.anyio
async def test_career_page_no_data_returns_null_summary(client: AsyncClient) -> None:
    """Quand le joueur n'a pas encore de données, summary=null."""
    mock_resp = _make_career_page_response(no_data=True)
    with patch("apps.api.app.services.career_service.get_career_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/career")

    assert resp.status_code == 200
    data = resp.json()
    assert data["summary"] is None
    assert data["hero_progress"] is None
    assert data["xp_history"] == []
    assert data["lusr"] is None


@pytest.mark.anyio
async def test_career_charts_nullable(client: AsyncClient) -> None:
    """Les champs charts sont tous null par défaut."""
    mock_resp = _make_career_page_response()
    with patch("apps.api.app.services.career_service.get_career_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/career")

    charts = resp.json()["charts"]
    assert charts["rank_progress_gauge"] is None
    assert charts["hero_progress_gauge"] is None
    assert charts["xp_history_figure"] is None
    assert charts["lusr_rating_figure"] is None


@pytest.mark.anyio
async def test_career_page_unknown_player_returns_404(client: AsyncClient) -> None:
    """Un slug inconnu retourne 404."""
    resp = await client.get("/api/v1/players/ghost-player-xyz/pages/career")
    assert resp.status_code == 404


# ===========================================================================
# GET /players/{slug}/pages/career/top-matches
# ===========================================================================


@pytest.mark.anyio
async def test_career_top_matches_returns_200(client: AsyncClient) -> None:
    """L'endpoint top-matches retourne 200."""
    mock_resp = CareerTopMatchesResponse(items=[_make_top_match()])
    with patch("apps.api.app.services.career_service.get_top_matches", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/career/top-matches")
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_career_top_matches_schema(client: AsyncClient) -> None:
    """La réponse top-matches contient 'items' avec les champs CareerTopMatch."""
    mock_resp = CareerTopMatchesResponse(items=[_make_top_match()])
    with patch("apps.api.app.services.career_service.get_top_matches", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/career/top-matches")

    data = resp.json()
    assert "items" in data
    assert len(data["items"]) == 1
    item = data["items"][0]
    assert item["match_id"] == "abc-123"
    assert item["map_ui"] == "Recharge"
    assert item["outcome_label"] == "Victoire"
    assert item["badge_type"] == "dominant"


@pytest.mark.anyio
async def test_career_top_matches_empty_list(client: AsyncClient) -> None:
    """L'endpoint retourne items=[] si aucun data."""
    mock_resp = CareerTopMatchesResponse(items=[])
    with patch("apps.api.app.services.career_service.get_top_matches", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/career/top-matches")

    assert resp.status_code == 200
    assert resp.json()["items"] == []


# ===========================================================================
# GET /players/{slug}/pages/career/encounters
# ===========================================================================


@pytest.mark.anyio
async def test_career_encounters_returns_200(client: AsyncClient) -> None:
    """L'endpoint encounters retourne 200."""
    mock_resp = CareerEncountersResponse(items=[_make_encounter()])
    with patch("apps.api.app.services.career_service.get_encounters", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/career/encounters")
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_career_encounters_schema(client: AsyncClient) -> None:
    """La réponse encounters contient 'items' avec les champs CareerEncounter."""
    mock_resp = CareerEncountersResponse(items=[_make_encounter()])
    with patch("apps.api.app.services.career_service.get_encounters", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/career/encounters")

    data = resp.json()
    assert "items" in data
    assert len(data["items"]) == 1
    item = data["items"][0]
    assert item["encounter_key"] == "xuid-999"
    assert item["opponent_gamertag"] == "Rival42"
    assert item["count_matches"] == 25
    assert item["wins"] == 12
    assert item["losses"] == 13


@pytest.mark.anyio
async def test_career_encounters_unknown_player_returns_404(client: AsyncClient) -> None:
    """Un slug inconnu retourne 404 pour encounters aussi."""
    resp = await client.get("/api/v1/players/unknown-xyz/pages/career/encounters")
    assert resp.status_code == 404
