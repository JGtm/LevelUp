"""Tests unitaires — endpoints Médiathèque (Slice 8).

Couvre :
- GET /players/{slug}/pages/media (200, schéma, items, pagination, comptages)
- GET avec filtre kind_filter et section_filter
- Réponse vide graceful
- GET /players/unknown/pages/media (404)
"""

from __future__ import annotations

from datetime import datetime, timezone
from unittest.mock import patch

import pytest
from httpx import ASGITransport, AsyncClient

from apps.api.app.schemas.common import PaginatedResponse, PaginationMeta
from apps.api.app.schemas.media import (
    MediaItemRow,
    MediaPageResponse,
)

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

_NOW = datetime(2025, 3, 15, 20, 0, tzinfo=timezone.utc)


def _make_item(basename: str = "clip_001.mp4", kind: str = "video") -> MediaItemRow:
    return MediaItemRow(
        basename=basename,
        file_path=f"/media/{basename}",
        kind=kind,
        thumbnail_path=f"/thumbnails/{basename}.jpg",
        match_id="match-001",
        capture_end_utc=_NOW,
        match_start_time=_NOW,
        section="mine",
        owner_gamertag="DemoPlayer",
        map_name="Recharge",
    )


def _make_media_response(count: int = 3) -> MediaPageResponse:
    items = [_make_item(f"clip_{i:03d}.mp4") for i in range(count)]
    return MediaPageResponse(
        items=PaginatedResponse(
            items=items,
            pagination=PaginationMeta(
                total=count, page=1, page_size=50, has_next=False, has_prev=False
            ),
        ),
        total_mine=count,
        total_teammates=1,
        total_unassigned=0,
    )


def _make_empty_media_response() -> MediaPageResponse:
    return MediaPageResponse(
        items=PaginatedResponse(
            items=[],
            pagination=PaginationMeta(
                total=0, page=1, page_size=50, has_next=False, has_prev=False
            ),
        ),
        total_mine=0,
        total_teammates=0,
        total_unassigned=0,
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
# GET /players/{slug}/pages/media
# ===========================================================================


@pytest.mark.anyio
async def test_media_page_returns_200(client: AsyncClient) -> None:
    """En DEMO_MODE, l'endpoint media retourne 200."""
    mock_resp = _make_media_response()
    with patch("apps.api.app.services.media_service.get_media_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/media")
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_media_schema_complete(client: AsyncClient) -> None:
    """La réponse contient tous les champs de MediaPageResponse."""
    mock_resp = _make_media_response()
    with patch("apps.api.app.services.media_service.get_media_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/media")

    assert resp.status_code == 200
    data = resp.json()
    for key in ("items", "total_mine", "total_teammates", "total_unassigned"):
        assert key in data, f"Clé manquante : {key}"


@pytest.mark.anyio
async def test_media_items_fields(client: AsyncClient) -> None:
    """Les items MediaItemRow contiennent basename, kind et section."""
    mock_resp = _make_media_response()
    with patch("apps.api.app.services.media_service.get_media_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/media")

    items_data = resp.json()["items"]["items"]
    assert len(items_data) == 3
    for item in items_data:
        assert "basename" in item
        assert "kind" in item
        assert "section" in item
        assert item["section"] == "mine"


@pytest.mark.anyio
async def test_media_pagination_metadata(client: AsyncClient) -> None:
    """Les métadonnées de pagination (total, page, page_size) sont présentes."""
    mock_resp = _make_media_response(count=3)
    with patch("apps.api.app.services.media_service.get_media_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/media")

    items_wrapper = resp.json()["items"]
    assert items_wrapper["pagination"]["total"] == 3
    assert items_wrapper["pagination"]["page"] == 1
    assert items_wrapper["pagination"]["page_size"] == 50


@pytest.mark.anyio
async def test_media_section_counts(client: AsyncClient) -> None:
    """Les comptages par section sont présents et corrects."""
    mock_resp = _make_media_response(count=3)
    with patch("apps.api.app.services.media_service.get_media_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/media")

    data = resp.json()
    assert data["total_mine"] == 3
    assert data["total_teammates"] == 1
    assert data["total_unassigned"] == 0


@pytest.mark.anyio
async def test_media_empty_graceful(client: AsyncClient) -> None:
    """Quand il n'y a pas de médias, la réponse est vide mais valide."""
    mock_resp = _make_empty_media_response()
    with patch("apps.api.app.services.media_service.get_media_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/media")

    assert resp.status_code == 200
    data = resp.json()
    assert data["items"]["pagination"]["total"] == 0
    assert data["items"]["items"] == []


@pytest.mark.anyio
async def test_media_kind_filter_accepted(client: AsyncClient) -> None:
    """Le paramètre kind_filter=video est accepté sans erreur 422."""
    mock_resp = _make_media_response()
    with patch("apps.api.app.services.media_service.get_media_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/media?kind_filter=video")
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_media_section_filter_accepted(client: AsyncClient) -> None:
    """Le paramètre section_filter=mine est accepté sans erreur 422."""
    mock_resp = _make_media_response()
    with patch("apps.api.app.services.media_service.get_media_page", return_value=mock_resp):
        resp = await client.get("/api/v1/players/demo/pages/media?section_filter=mine")
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_media_invalid_kind_rejected(client: AsyncClient) -> None:
    """Un kind_filter invalide retourne 422."""
    resp = await client.get("/api/v1/players/demo/pages/media?kind_filter=audio")
    assert resp.status_code == 422


@pytest.mark.anyio
async def test_media_slug_not_found(client: AsyncClient) -> None:
    """Slug inconnu → 404."""
    resp = await client.get("/api/v1/players/unknown-xyz/pages/media")
    assert resp.status_code == 404
