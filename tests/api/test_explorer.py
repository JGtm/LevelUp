"""Tests unitaires — endpoints Explorer (Slice 4).

Couvre :
- GET  /directory/gamertags/search         (200, schema, suggestions, vide)
- POST /players/{slug}/pages/explorer/matches-query (200, schema, summary, rows, pagination, 404)
- POST /players/{slug}/pages/explorer/player-query  (200, schema, target, encounters, 404, gamertag inconnu)
"""

from __future__ import annotations

from datetime import datetime, timezone
from unittest.mock import patch

import pytest
from httpx import ASGITransport, AsyncClient

from apps.api.app.schemas.common import PaginatedResponse, PaginationMeta
from apps.api.app.schemas.explorer import (
    ExplorerEncounterRow,
    ExplorerMatchesQueryResponse,
    ExplorerMatchesQuerySummary,
    ExplorerMatchRow,
    ExplorerPlayerQueryResponse,
    ExplorerPlayerSummary,
    ExplorerPlayerTarget,
    GamertagSearchResponse,
    GamertagSuggestion,
)

# ---------------------------------------------------------------------------
# Factories
# ---------------------------------------------------------------------------


def _make_suggestion(gamertag: str = "TestPlayer", score: float = 0.9) -> GamertagSuggestion:
    return GamertagSuggestion(
        gamertag=gamertag,
        xuid="xuid001",
        score=score,
        exact_match=score == 1.0,
    )


def _make_search_response(q: str = "Test", items: list | None = None) -> GamertagSearchResponse:
    if items is None:
        items = [_make_suggestion()]
    return GamertagSearchResponse(query=q, items=items)


def _make_match_row(match_id: str = "match-001") -> ExplorerMatchRow:
    return ExplorerMatchRow(
        match_id=match_id,
        start_time=datetime(2025, 1, 10, 20, 0, tzinfo=timezone.utc),
        start_time_label="10/01/2025 20:00",
        map_ui="Recharge",
        mode_ui="Slayer",
        playlist_label="Ranked Arena",
        outcome_label="Victoire",
        score_label="50 - 32",
        is_with_friends=True,
        experience_type_label="Classé",
    )


def _make_encounter_row(
    gamertag: str = "Foe42",
    same_team: bool = False,
) -> ExplorerEncounterRow:
    return ExplorerEncounterRow(
        gamertag=gamertag,
        xuid="xuid-foe",
        count_matches=5,
        wins=3,
        losses=2,
        last_seen_at=datetime(2025, 1, 9, tzinfo=timezone.utc),
        same_team=same_team,
    )


def _make_matches_response(
    *,
    rows: list[ExplorerMatchRow] | None = None,
    total: int = 1,
) -> ExplorerMatchesQueryResponse:
    if rows is None:
        rows = [_make_match_row()]
    pagination = PaginationMeta(total=total, page=1, page_size=50, has_next=False, has_prev=False)
    table: PaginatedResponse[ExplorerMatchRow] = PaginatedResponse(
        items=rows, pagination=pagination
    )
    return ExplorerMatchesQueryResponse(
        summary=ExplorerMatchesQuerySummary(total_matches=total, selected_match_id=None),
        table=table,
    )


def _make_player_response(
    *,
    allies: list[ExplorerEncounterRow] | None = None,
    enemies: list[ExplorerEncounterRow] | None = None,
) -> ExplorerPlayerQueryResponse:
    if allies is None:
        allies = [_make_encounter_row("Ally1", same_team=True)]
    if enemies is None:
        enemies = [_make_encounter_row("Foe42", same_team=False)]
    return ExplorerPlayerQueryResponse(
        target=ExplorerPlayerTarget(gamertag="Foe42", xuid="xuid-foe"),
        summary=ExplorerPlayerSummary(
            matches_together=8,
            wins_together=5,
            losses_together=3,
            last_seen_at=datetime(2025, 1, 9, tzinfo=timezone.utc),
        ),
        allies_table=allies,
        enemies_table=enemies,
        common_matches=[_make_match_row()],
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
# GET /directory/gamertags/search
# ---------------------------------------------------------------------------


@pytest.mark.anyio
async def test_gamertag_search_returns_200(client: AsyncClient) -> None:
    """L'endpoint de recherche de gamertags retourne 200."""
    mock_resp = _make_search_response()
    with patch(
        "apps.api.app.services.explorer_service.search_gamertags",
        return_value=mock_resp,
    ):
        resp = await client.get("/api/v1/directory/gamertags/search", params={"q": "Test"})
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_gamertag_search_schema(client: AsyncClient) -> None:
    """La réponse contient query et items."""
    mock_resp = _make_search_response(q="Test", items=[_make_suggestion("TestPlayer")])
    with patch(
        "apps.api.app.services.explorer_service.search_gamertags",
        return_value=mock_resp,
    ):
        resp = await client.get("/api/v1/directory/gamertags/search", params={"q": "Test"})

    data = resp.json()
    assert "query" in data
    assert "items" in data
    assert data["query"] == "Test"


@pytest.mark.anyio
async def test_gamertag_search_suggestion_fields(client: AsyncClient) -> None:
    """Chaque suggestion contient gamertag, xuid, score, exact_match."""
    mock_resp = _make_search_response(items=[_make_suggestion("TestPlayer", score=0.9)])
    with patch(
        "apps.api.app.services.explorer_service.search_gamertags",
        return_value=mock_resp,
    ):
        resp = await client.get("/api/v1/directory/gamertags/search", params={"q": "Test"})

    items = resp.json()["items"]
    assert len(items) == 1
    s = items[0]
    assert s["gamertag"] == "TestPlayer"
    assert "score" in s
    assert "exact_match" in s


@pytest.mark.anyio
async def test_gamertag_search_empty_result(client: AsyncClient) -> None:
    """Une requête sans correspondance retourne items=[]."""
    mock_resp = GamertagSearchResponse(query="zzz", items=[])
    with patch(
        "apps.api.app.services.explorer_service.search_gamertags",
        return_value=mock_resp,
    ):
        resp = await client.get("/api/v1/directory/gamertags/search", params={"q": "zzz"})

    data = resp.json()
    assert data["items"] == []
    assert data["query"] == "zzz"


# ---------------------------------------------------------------------------
# POST /players/{slug}/pages/explorer/matches-query
# ---------------------------------------------------------------------------


@pytest.mark.anyio
async def test_explorer_matches_query_returns_200(client: AsyncClient) -> None:
    """En DEMO_MODE, l'endpoint matches-query retourne 200."""
    mock_resp = _make_matches_response()
    with patch(
        "apps.api.app.services.explorer_service.get_explorer_matches",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/explorer/matches-query", json={})
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_explorer_matches_schema(client: AsyncClient) -> None:
    """La réponse contient summary et table."""
    mock_resp = _make_matches_response()
    with patch(
        "apps.api.app.services.explorer_service.get_explorer_matches",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/explorer/matches-query", json={})

    data = resp.json()
    assert "summary" in data
    assert "table" in data


@pytest.mark.anyio
async def test_explorer_matches_summary_fields(client: AsyncClient) -> None:
    """Le résumé contient total_matches et selected_match_id."""
    mock_resp = _make_matches_response(total=12)
    with patch(
        "apps.api.app.services.explorer_service.get_explorer_matches",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/explorer/matches-query", json={})

    summary = resp.json()["summary"]
    assert summary["total_matches"] == 12
    assert "selected_match_id" in summary


@pytest.mark.anyio
async def test_explorer_matches_row_fields(client: AsyncClient) -> None:
    """Les champs d'une ExplorerMatchRow sont correctement sérialisés."""
    mock_resp = _make_matches_response(rows=[_make_match_row("match-abc")])
    with patch(
        "apps.api.app.services.explorer_service.get_explorer_matches",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/explorer/matches-query", json={})

    items = resp.json()["table"]["items"]
    assert len(items) == 1
    row = items[0]
    assert row["match_id"] == "match-abc"
    assert row["map_ui"] == "Recharge"
    assert row["mode_ui"] == "Slayer"
    assert row["outcome_label"] == "Victoire"
    assert row["is_with_friends"] is True
    assert row["experience_type_label"] == "Classé"


@pytest.mark.anyio
async def test_explorer_matches_empty(client: AsyncClient) -> None:
    """Quand aucun match ne correspond, items=[] et total=0."""
    mock_resp = _make_matches_response(rows=[], total=0)
    with patch(
        "apps.api.app.services.explorer_service.get_explorer_matches",
        return_value=mock_resp,
    ):
        resp = await client.post("/api/v1/players/demo/pages/explorer/matches-query", json={})

    data = resp.json()
    assert data["table"]["items"] == []
    assert data["summary"]["total_matches"] == 0


@pytest.mark.anyio
async def test_explorer_matches_404_unknown_player(client: AsyncClient) -> None:
    """Un joueur inexistant retourne 404."""
    resp = await client.post(
        "/api/v1/players/unknown-player-xyz/pages/explorer/matches-query", json={}
    )
    assert resp.status_code == 404


# ---------------------------------------------------------------------------
# POST /players/{slug}/pages/explorer/player-query
# ---------------------------------------------------------------------------


@pytest.mark.anyio
async def test_explorer_player_query_returns_200(client: AsyncClient) -> None:
    """En DEMO_MODE, l'endpoint player-query retourne 200."""
    mock_resp = _make_player_response()
    with patch(
        "apps.api.app.services.explorer_service.get_explorer_player",
        return_value=mock_resp,
    ):
        resp = await client.post(
            "/api/v1/players/demo/pages/explorer/player-query",
            json={"target_gamertag": "Foe42"},
        )
    assert resp.status_code == 200


@pytest.mark.anyio
async def test_explorer_player_schema(client: AsyncClient) -> None:
    """La réponse contient target, summary, allies_table, enemies_table, common_matches."""
    mock_resp = _make_player_response()
    with patch(
        "apps.api.app.services.explorer_service.get_explorer_player",
        return_value=mock_resp,
    ):
        resp = await client.post(
            "/api/v1/players/demo/pages/explorer/player-query",
            json={"target_gamertag": "Foe42"},
        )

    data = resp.json()
    assert "target" in data
    assert "summary" in data
    assert "allies_table" in data
    assert "enemies_table" in data
    assert "common_matches" in data


@pytest.mark.anyio
async def test_explorer_player_target_fields(client: AsyncClient) -> None:
    """target contient gamertag et xuid."""
    mock_resp = _make_player_response()
    with patch(
        "apps.api.app.services.explorer_service.get_explorer_player",
        return_value=mock_resp,
    ):
        resp = await client.post(
            "/api/v1/players/demo/pages/explorer/player-query",
            json={"target_gamertag": "Foe42"},
        )

    target = resp.json()["target"]
    assert target["gamertag"] == "Foe42"
    assert "xuid" in target


@pytest.mark.anyio
async def test_explorer_player_summary_fields(client: AsyncClient) -> None:
    """summary contient matches_together, wins_together, losses_together, last_seen_at."""
    mock_resp = _make_player_response()
    with patch(
        "apps.api.app.services.explorer_service.get_explorer_player",
        return_value=mock_resp,
    ):
        resp = await client.post(
            "/api/v1/players/demo/pages/explorer/player-query",
            json={"target_gamertag": "Foe42"},
        )

    summary = resp.json()["summary"]
    assert summary["matches_together"] == 8
    assert summary["wins_together"] == 5
    assert summary["losses_together"] == 3
    assert "last_seen_at" in summary


@pytest.mark.anyio
async def test_explorer_player_encounter_fields(client: AsyncClient) -> None:
    """Les lignes enemies_table contiennent les champs d'encounter."""
    mock_resp = _make_player_response()
    with patch(
        "apps.api.app.services.explorer_service.get_explorer_player",
        return_value=mock_resp,
    ):
        resp = await client.post(
            "/api/v1/players/demo/pages/explorer/player-query",
            json={"target_gamertag": "Foe42"},
        )

    enemies = resp.json()["enemies_table"]
    assert len(enemies) >= 1
    row = enemies[0]
    assert "gamertag" in row
    assert "count_matches" in row
    assert "wins" in row
    assert "losses" in row
    assert row["same_team"] is False


@pytest.mark.anyio
async def test_explorer_player_404_unknown_player(client: AsyncClient) -> None:
    """Un joueur courant inexistant retourne 404."""
    resp = await client.post(
        "/api/v1/players/unknownxyz/pages/explorer/player-query",
        json={"target_gamertag": "Anyone"},
    )
    assert resp.status_code == 404
