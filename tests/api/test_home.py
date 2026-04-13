"""Tests unitaires — endpoints Accueil Mission Control (Slice 5).

Couvre :
- GET /players/{slug}/pages/home (200, schéma, hero, highlights, matchs, sessions, médias)
- GET /players/{slug}/battlepass (200, available=False graceful)
- GET /players/{slug}/challenges (200, available=False graceful)
- GET /players/unknown/pages/home (404)
"""

from __future__ import annotations

from datetime import datetime, timezone
from unittest.mock import patch

import pytest
from httpx import ASGITransport, AsyncClient

from apps.api.app.schemas.home import (
    BattlePassResponse,
    ChallengesResponse,
    HeroKPIs,
    HeroTrend,
    HighlightItem,
    HomeHeroCard,
    HomePageResponse,
    RecentMatchItem,
    RecentMediaItem,
    SessionSummaryItem,
)

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

_NOW = datetime(2025, 3, 15, 20, 0, tzinfo=timezone.utc)


def _make_hero() -> HomeHeroCard:
    return HomeHeroCard(
        player_name="DemoPlayer",
        kpis=HeroKPIs(
            win_rate=0.55,
            global_ratio=1.42,
            avg_accuracy=48.3,
            total_matches=120,
            wins=66,
            losses=54,
        ),
        trend=HeroTrend(ratio_delta=0.12, accuracy_delta=1.5, win_rate_delta=0.05),
    )


def _make_home_response() -> HomePageResponse:
    return HomePageResponse(
        hero=_make_hero(),
        highlights=[
            HighlightItem(title="Pic KD récent", value="KD 2.10", detail="Recharge · Slayer"),
            HighlightItem(title="Tendance", value="KD +0.12", detail="WR +5%"),
        ],
        recent_matches=[
            RecentMatchItem(
                match_id="match-001",
                title="Victoire · Recharge",
                detail="Slayer · KD 2.10 · 52%",
                started_at=_NOW,
                outcome_label="Victoire",
                outcome_tone="win",
            ),
        ],
        recent_media=[
            RecentMediaItem(basename="clip_001.mp4", match_id="match-001", match_start_time=_NOW),
        ],
        solo_session=SessionSummaryItem(
            session_label="SoloMorning",
            match_count=5,
            win_rate=0.6,
            global_ratio=1.3,
            started_at=_NOW,
        ),
        squad_session=SessionSummaryItem(
            session_label="SquadNight",
            match_count=8,
            win_rate=0.75,
            global_ratio=1.8,
            started_at=_NOW,
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


# ===========================================================================
# GET /players/{slug}/pages/home
# ===========================================================================


@pytest.mark.anyio
async def test_home_page_returns_200(client: AsyncClient) -> None:
    """En DEMO_MODE, l'endpoint home retourne 200."""
    mock_resp = _make_home_response()
    with patch("apps.api.app.services.home_service.get_home_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/home")
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_home_page_schema_complete(client: AsyncClient) -> None:
    """La réponse contient tous les champs de HomePageResponse."""
    mock_resp = _make_home_response()
    with patch("apps.api.app.services.home_service.get_home_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/home")

    assert resp.status_code == 200
    data = resp.json()
    for key in (
        "hero",
        "highlights",
        "recent_matches",
        "recent_media",
        "solo_session",
        "squad_session",
    ):
        assert key in data, f"Clé manquante : {key}"


@pytest.mark.anyio
async def test_home_hero_fields(client: AsyncClient) -> None:
    """Le bloc hero contient player_name et kpis."""
    mock_resp = _make_home_response()
    with patch("apps.api.app.services.home_service.get_home_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/home")

    data = resp.json()
    hero = data["hero"]
    assert hero["player_name"] == "DemoPlayer"
    assert "kpis" in hero
    assert hero["kpis"]["total_matches"] == 120
    assert hero["kpis"]["win_rate"] == pytest.approx(0.55)


@pytest.mark.anyio
async def test_home_hero_trend(client: AsyncClient) -> None:
    """Le trend est sérialisé correctement."""
    mock_resp = _make_home_response()
    with patch("apps.api.app.services.home_service.get_home_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/home")

    trend = resp.json()["hero"]["trend"]
    assert trend is not None
    assert trend["ratio_delta"] == pytest.approx(0.12)


@pytest.mark.anyio
async def test_home_highlights(client: AsyncClient) -> None:
    """Les highlights contiennent title et value."""
    mock_resp = _make_home_response()
    with patch("apps.api.app.services.home_service.get_home_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/home")

    highlights = resp.json()["highlights"]
    assert len(highlights) >= 1
    for h in highlights:
        assert "title" in h
        assert "value" in h


@pytest.mark.anyio
async def test_home_recent_matches(client: AsyncClient) -> None:
    """Les matchs récents contiennent outcome_label et match_id."""
    mock_resp = _make_home_response()
    with patch("apps.api.app.services.home_service.get_home_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/home")

    matches = resp.json()["recent_matches"]
    assert len(matches) >= 1
    assert matches[0]["match_id"] == "match-001"
    assert matches[0]["outcome_tone"] == "win"


@pytest.mark.anyio
async def test_home_sessions(client: AsyncClient) -> None:
    """Les blocs solo_session et squad_session sont présents."""
    mock_resp = _make_home_response()
    with patch("apps.api.app.services.home_service.get_home_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/home")

    data = resp.json()
    assert data["solo_session"]["session_label"] == "SoloMorning"
    assert data["squad_session"]["match_count"] == 8


@pytest.mark.anyio
async def test_home_recent_media(client: AsyncClient) -> None:
    """Les médias récents contiennent un basename."""
    mock_resp = _make_home_response()
    with patch("apps.api.app.services.home_service.get_home_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/home")

    media = resp.json()["recent_media"]
    assert len(media) >= 1
    assert media[0]["basename"] == "clip_001.mp4"


@pytest.mark.anyio
async def test_home_page_slug_not_found(client: AsyncClient) -> None:
    """Slug inconnu → 404."""
    resp = await client.get("/api/v1/players/unknown-xyz/pages/home")
    assert resp.status_code == 404


# ===========================================================================
# GET /players/{slug}/battlepass
# ===========================================================================


@pytest.mark.anyio
async def test_battlepass_returns_200(client: AsyncClient) -> None:
    """L'endpoint battlepass retourne 200 (graceful fallback)."""
    mock_resp = BattlePassResponse(available=False, error_hint="live_unavailable")
    with patch("apps.api.app.services.home_service.get_battlepass", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/battlepass")
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_battlepass_graceful_unavailable(client: AsyncClient) -> None:
    """Sans session live, available=False et error_hint renseigné."""
    mock_resp = BattlePassResponse(available=False, error_hint="live_unavailable")
    with patch("apps.api.app.services.home_service.get_battlepass", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/battlepass")

    data = resp.json()
    assert data["available"] is False
    assert data["error_hint"] == "live_unavailable"


# ===========================================================================
# GET /players/{slug}/challenges
# ===========================================================================


@pytest.mark.anyio
async def test_challenges_returns_200(client: AsyncClient) -> None:
    """L'endpoint challenges retourne 200 (graceful fallback)."""
    mock_resp = ChallengesResponse(available=False, error_hint="live_unavailable")
    with patch("apps.api.app.services.home_service.get_challenges", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/challenges")
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_challenges_graceful_unavailable(client: AsyncClient) -> None:
    """Sans session live, available=False et error_hint renseigné."""
    mock_resp = ChallengesResponse(available=False, error_hint="live_unavailable")
    with patch("apps.api.app.services.home_service.get_challenges", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/challenges")

    data = resp.json()
    assert data["available"] is False
