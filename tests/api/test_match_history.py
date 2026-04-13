"""Tests unitaires — endpoints Historique des parties (Slice 3).

Couvre :
- POST /players/{slug}/pages/match-history/query (200, schema, résumé, pagination, filtres, 404)
- POST /players/{slug}/pages/match-history/export (200, schema, 404)
"""

from __future__ import annotations

from datetime import datetime, timezone
from unittest.mock import patch

import pytest
from httpx import ASGITransport, AsyncClient

from apps.api.app.schemas.common import PaginatedResponse, PaginationMeta
from apps.api.app.schemas.match_history import (
    ExportHint,
    FileTokenResponse,
    MatchHistoryPageResponse,
    MatchHistoryQuerySummary,
    MatchHistoryRow,
)

# ---------------------------------------------------------------------------
# Helpers — factory objects
# ---------------------------------------------------------------------------


def _make_row(match_id: str = "match-001") -> MatchHistoryRow:
    return MatchHistoryRow(
        match_id=match_id,
        start_time=datetime(2025, 1, 10, 20, 0, tzinfo=timezone.utc),
        start_time_label="10/01/2025 20:00",
        outcome_code=2,
        outcome_label="Victoire",
        score_label="50 - 32",
        map_ui="Recharge",
        mode_ui="Slayer",
        playlist_label="Ranked Arena",
        team_mmr=1500.0,
        enemy_mmr=1480.0,
        delta_mmr=20.0,
        win_rate_hist=62.5,
        win_rate_hist_total=40,
        performance_score_relative=74,
        average_life_mmss="1:45",
        match_url="https://www.halowaypoint.com/halo-infinite/players/demo/matches/match-001",
    )


def _make_page_response(
    *,
    rows: list[MatchHistoryRow] | None = None,
    total: int = 1,
    with_export_hint: bool = False,
) -> MatchHistoryPageResponse:
    if rows is None:
        rows = [_make_row()]
    pagination = PaginationMeta(
        total=total,
        page=1,
        page_size=50,
        has_next=False,
        has_prev=False,
    )
    table: PaginatedResponse[MatchHistoryRow] = PaginatedResponse(
        items=rows,
        pagination=pagination,
    )
    return MatchHistoryPageResponse(
        summary=MatchHistoryQuerySummary(
            total_matches_scoped=total,
            total_matches_unfiltered=total,
            period_label=None,
            active_filter_mode="all",
        ),
        table=table,
        export_hint=ExportHint(file_name="levelup_matches.csv", estimated_rows=total)
        if with_export_hint
        else None,
    )


def _make_export_response() -> FileTokenResponse:
    return FileTokenResponse(
        file_token="tok_abc123",
        file_name="levelup_matches.csv",
        content_type="text/csv",
        download_path="/api/v1/downloads/tok_abc123",
        expires_at=datetime(2025, 1, 10, 21, 0, tzinfo=timezone.utc),
        estimated_rows=42,
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
# POST /players/{slug}/pages/match-history/query
# ---------------------------------------------------------------------------


@pytest.mark.anyio
async def test_match_history_query_returns_200(client: AsyncClient) -> None:
    """En DEMO_MODE, l'endpoint match-history/query retourne 200."""
    mock_resp = _make_page_response()
    with patch(
        "apps.api.app.services.match_history_service.get_match_history_page",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/match-history/query", json={})
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_match_history_schema_complete(client: AsyncClient) -> None:
    """La réponse contient summary, table et available_sort_fields."""
    mock_resp = _make_page_response()
    with patch(
        "apps.api.app.services.match_history_service.get_match_history_page",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/match-history/query", json={})

    assert resp.status_code == 200
    data = resp.json()
    assert "summary" in data
    assert "table" in data
    assert "available_sort_fields" in data


@pytest.mark.anyio
async def test_match_history_summary_fields(client: AsyncClient) -> None:
    """Les champs du résumé sont correctement sérialisés."""
    mock_resp = _make_page_response(total=42)
    with patch(
        "apps.api.app.services.match_history_service.get_match_history_page",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/match-history/query", json={})

    summary = resp.json()["summary"]
    assert summary["total_matches_scoped"] == 42
    assert summary["total_matches_unfiltered"] == 42
    assert summary["active_filter_mode"] == "all"
    assert summary["period_label"] is None


@pytest.mark.anyio
async def test_match_history_row_fields(client: AsyncClient) -> None:
    """Les champs d'une MatchHistoryRow sont correctement sérialisés."""
    mock_resp = _make_page_response(rows=[_make_row("match-xyz")])
    with patch(
        "apps.api.app.services.match_history_service.get_match_history_page",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/match-history/query", json={})

    items = resp.json()["table"]["items"]
    assert len(items) == 1
    row = items[0]
    assert row["match_id"] == "match-xyz"
    assert row["outcome_code"] == 2
    assert row["outcome_label"] == "Victoire"
    assert row["map_ui"] == "Recharge"
    assert row["mode_ui"] == "Slayer"
    assert row["performance_score_relative"] == 74
    assert row["win_rate_hist"] == 62.5


@pytest.mark.anyio
async def test_match_history_pagination_meta(client: AsyncClient) -> None:
    """La pagination contient total, page, page_size, has_next, has_prev."""
    mock_resp = _make_page_response(total=5)
    with patch(
        "apps.api.app.services.match_history_service.get_match_history_page",
        return_value=mock_resp,
    ):
        resp = await client.post(
            "/api/v1/players/demo/pages/match-history/query",
            json={"pagination": {"page": 1, "page_size": 50}},
        )

    pagination = resp.json()["table"]["pagination"]
    assert pagination["total"] == 5
    assert pagination["page"] == 1
    assert "has_next" in pagination
    assert "has_prev" in pagination


@pytest.mark.anyio
async def test_match_history_empty_result(client: AsyncClient) -> None:
    """Quand la requête ne retourne aucun match, items=[] et total=0."""
    mock_resp = _make_page_response(rows=[], total=0)
    with patch(
        "apps.api.app.services.match_history_service.get_match_history_page",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/match-history/query", json={})

    data = resp.json()
    assert data["table"]["items"] == []
    assert data["table"]["pagination"]["total"] == 0
    assert data["summary"]["total_matches_scoped"] == 0


@pytest.mark.anyio
async def test_match_history_with_export_hint(client: AsyncClient) -> None:
    """Quand include_export_hint=true, export_hint est présent."""
    mock_resp = _make_page_response(with_export_hint=True)
    with patch(
        "apps.api.app.services.match_history_service.get_match_history_page",
        return_value=mock_resp,
    ):
        resp = await client.post(
            "/api/v1/players/demo/pages/match-history/query",
            json={"include_export_hint": True},
        )

    data = resp.json()
    assert data["export_hint"] is not None
    assert data["export_hint"]["file_name"] == "levelup_matches.csv"
    assert data["export_hint"]["estimated_rows"] > 0


@pytest.mark.anyio
async def test_match_history_export_hint_null_by_default(client: AsyncClient) -> None:
    """Par défaut, export_hint est null."""
    mock_resp = _make_page_response(with_export_hint=False)
    with patch(
        "apps.api.app.services.match_history_service.get_match_history_page",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/match-history/query", json={})

    assert resp.json()["export_hint"] is None


@pytest.mark.anyio
async def test_match_history_available_sort_fields(client: AsyncClient) -> None:
    """available_sort_fields contient les champs de tri attendus."""
    mock_resp = _make_page_response()
    with patch(
        "apps.api.app.services.match_history_service.get_match_history_page",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/match-history/query", json={})

    fields = resp.json()["available_sort_fields"]
    assert "start_time" in fields
    assert "outcome_code" in fields
    assert "performance_score_relative" in fields


@pytest.mark.anyio
async def test_match_history_query_unknown_player_returns_404(client: AsyncClient) -> None:
    """Un slug inconnu retourne 404."""
    resp = await client.post("/api/v1/players/ghost-player-xyz/pages/match-history/query", json={})
    assert resp.status_code == 404


@pytest.mark.anyio
async def test_match_history_with_filter_body(client: AsyncClient) -> None:
    """Un body avec filtres est accepté et transféré au service."""
    mock_resp = _make_page_response(total=10)
    with patch(
        "apps.api.app.services.match_history_service.get_match_history_page",
        return_value=mock_resp,
    ) as mock_svc:
        resp = await client.post(
            "/api/v1/players/demo/pages/match-history/query",
            json={
                "filters": {
                    "filter_mode": "period",
                    "period": {"start_date": "2025-01-01", "end_date": "2025-01-31"},
                },
                "pagination": {"page": 1, "page_size": 25},
            },
        )
    assert resp.status_code == 200
    mock_svc.assert_called_once()
    call_request = mock_svc.call_args[0][1]
    assert call_request.pagination.page_size == 25


# ---------------------------------------------------------------------------
# POST /players/{slug}/pages/match-history/export
# ---------------------------------------------------------------------------


@pytest.mark.anyio
async def test_match_history_export_returns_200(client: AsyncClient) -> None:
    """En DEMO_MODE, l'endpoint match-history/export retourne 200."""
    mock_resp = _make_export_response()
    with patch(
        "apps.api.app.services.match_history_service.get_match_history_export",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/match-history/export", json={})
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_match_history_export_schema(client: AsyncClient) -> None:
    """La réponse d'export contient file_token, file_name, download_path, expires_at."""
    mock_resp = _make_export_response()
    with patch(
        "apps.api.app.services.match_history_service.get_match_history_export",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/match-history/export", json={})

    data = resp.json()
    assert data["file_token"] == "tok_abc123"
    assert data["file_name"] == "levelup_matches.csv"
    assert data["content_type"] == "text/csv"
    assert "download_path" in data
    assert "expires_at" in data
    assert data["estimated_rows"] == 42


@pytest.mark.anyio
async def test_match_history_export_unknown_player_returns_404(client: AsyncClient) -> None:
    """Un slug inconnu retourne 404 pour l'export aussi."""
    resp = await client.post("/api/v1/players/ghost-player-xyz/pages/match-history/export", json={})
    assert resp.status_code == 404
