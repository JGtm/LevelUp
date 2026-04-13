"""Tests unitaires — endpoint POST /filters/resolve.

Ces tests couvrent la résolution des filtres en DEMO_MODE (sans DuckDB réel).
Ils vérifient le contrat du schéma, la normalisation de l'input et les cas limites.
"""

from __future__ import annotations

from pathlib import Path
from unittest.mock import patch

import polars as pl
import pytest
from httpx import ASGITransport, AsyncClient

# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------

_EMPTY_DF = pl.DataFrame()

_SAMPLE_DF = pl.DataFrame(
    {
        "match_id": ["m1", "m2", "m3", "m4"],
        "start_time": [
            "2025-01-01 10:00:00",
            "2025-01-02 11:00:00",
            "2025-01-03 12:00:00",
            "2025-01-04 13:00:00",
        ],
        "map_ui": ["Recharge", "Bazaar", "Recharge", "Streets"],
        "mode_ui": ["Slayer", "CTF", "Slayer", "Strongholds"],
        "playlist_ui": ["Ranked Arena", "Social", "Ranked Arena", "Social"],
        "is_firefight": [False, False, False, False],
        "is_ranked": [True, False, True, False],
        "session_id": ["sess1", "sess1", "sess2", "sess2"],
        "session_label": ["2025-01-01 #1", "2025-01-01 #1", "2025-01-03 #2", "2025-01-03 #2"],
        "is_with_friends": [False, False, True, True],
    }
)

_FF_DF = pl.DataFrame(
    {
        "match_id": ["ff1", "ff2"],
        "start_time": ["2025-02-01 10:00:00", "2025-02-02 12:00:00"],
        "map_ui": ["FireMap", "FireMap"],
        "mode_ui": ["Firefight", "Firefight"],
        "playlist_ui": ["Firefight Heroic", "Firefight Heroic"],
        "is_firefight": [True, True],
        "is_ranked": [False, False],
        "session_id": ["sessFF", "sessFF"],
        "session_label": ["2025-02-01 #FF", "2025-02-01 #FF"],
        "is_with_friends": [True, False],
    }
)


@pytest.fixture(autouse=True)
def force_demo_env(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("LEVELUP_DEMO_MODE", "true")
    monkeypatch.setenv("LEVELUP_SESSION_SECRET", "test-secret-key")
    monkeypatch.setenv("LEVELUP_SESSION_DIR", str(Path(__file__).parent / "_sessions_test"))
    from apps.api.app.core.config import get_settings

    get_settings.cache_clear()


@pytest.fixture
async def client() -> AsyncClient:
    from apps.api.app.main import create_app

    app = create_app()
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as ac:
        yield ac


def _mock_loads(df: pl.DataFrame):
    """Patch _load_matches_for_filters pour retourner un DataFrame de test."""
    return patch(
        "apps.api.app.services.filter_service._load_matches_for_filters",
        return_value=df,
    )


# ---------------------------------------------------------------------------
# Tests schéma et base
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_filters_resolve_returns_schema(client: AsyncClient) -> None:
    """Le endpoint retourne un FilterContextResolved valide avec tous les champs."""
    with _mock_loads(_SAMPLE_DF):
        resp = await client.post(
            "/api/v1/players/demo/filters/resolve",
            json={"filter_mode": "period", "period": {}, "cascade": {}},
        )
    assert resp.status_code == 200
    data = resp.json()
    assert "effective" in data
    assert "available_options" in data
    assert "session_options" in data
    assert "counts" in data


@pytest.mark.asyncio
async def test_filters_resolve_counts(client: AsyncClient) -> None:
    """Les comptes before/after sont corrects avec dataset complet et pas de filtre."""
    with _mock_loads(_SAMPLE_DF):
        resp = await client.post(
            "/api/v1/players/demo/filters/resolve",
            json={"filter_mode": "period", "cascade": {}},
        )
    assert resp.status_code == 200
    counts = resp.json()["counts"]
    assert counts["total_matches_before_filters"] == 4
    assert counts["total_matches_after_filters"] == 4


@pytest.mark.asyncio
async def test_filters_resolve_empty_db(client: AsyncClient) -> None:
    """DB vide → comptes à 0, options vides, pas d'erreur."""
    with _mock_loads(_EMPTY_DF):
        resp = await client.post(
            "/api/v1/players/demo/filters/resolve",
            json={"filter_mode": "period", "cascade": {}},
        )
    assert resp.status_code == 200
    data = resp.json()
    assert data["counts"]["total_matches_before_filters"] == 0
    assert data["counts"]["total_matches_after_filters"] == 0
    assert data["available_options"]["playlists"] == []


# ---------------------------------------------------------------------------
# Tests options disponibles
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_filters_resolve_playlists_available(client: AsyncClient) -> None:
    """Les options playlist disponibles sont bien calculées depuis le dataset."""
    with _mock_loads(_SAMPLE_DF):
        resp = await client.post(
            "/api/v1/players/demo/filters/resolve",
            json={"filter_mode": "period", "cascade": {}},
        )
    playlists = [opt["value"] for opt in resp.json()["available_options"]["playlists"]]
    assert "Ranked Arena" in playlists
    assert "Social" in playlists


@pytest.mark.asyncio
async def test_filters_resolve_experience_types_always_present(client: AsyncClient) -> None:
    """Les 3 types d'expérience sont toujours retournés dans les options."""
    with _mock_loads(_SAMPLE_DF):
        resp = await client.post(
            "/api/v1/players/demo/filters/resolve",
            json={"filter_mode": "period", "cascade": {}},
        )
    exp_values = [e["value"] for e in resp.json()["available_options"]["experience_types"]]
    assert "PVP non classé" in exp_values
    assert "PVP classé" in exp_values
    assert "PVE" in exp_values


# ---------------------------------------------------------------------------
# Tests filtres cascade
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_filters_cascade_playlist_reduces_modes(client: AsyncClient) -> None:
    """Filtrer sur 'Ranked Arena' réduit les modes disponibles."""
    with _mock_loads(_SAMPLE_DF):
        resp = await client.post(
            "/api/v1/players/demo/filters/resolve",
            json={
                "filter_mode": "period",
                "cascade": {"playlists": ["Ranked Arena"]},
            },
        )
    data = resp.json()
    assert data["counts"]["total_matches_after_filters"] == 2
    mode_values = [m["value"] for m in data["available_options"]["modes"]]
    assert "Slayer" in mode_values
    # CTF n'est pas dans Ranked Arena
    assert "CTF" not in mode_values


@pytest.mark.asyncio
async def test_filters_cascade_empty_result(client: AsyncClient) -> None:
    """Playlist inconnue → 0 matchs après filtre, not 500."""
    with _mock_loads(_SAMPLE_DF):
        resp = await client.post(
            "/api/v1/players/demo/filters/resolve",
            json={
                "filter_mode": "period",
                "cascade": {"playlists": ["NeverExistsPlaylist"]},
            },
        )
    assert resp.status_code == 200
    assert resp.json()["counts"]["total_matches_after_filters"] == 0


# ---------------------------------------------------------------------------
# Tests sessions
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_filters_sessions_options_populated(client: AsyncClient) -> None:
    """Les options de sessions contiennent les labels attendus."""
    with _mock_loads(_SAMPLE_DF):
        resp = await client.post(
            "/api/v1/players/demo/filters/resolve",
            json={"filter_mode": "sessions", "sessions": {}},
        )
    data = resp.json()
    all_labels = [s["label"] for s in data["session_options"]["all_sessions"]]
    assert "2025-01-01 #1" in all_labels
    assert "2025-01-03 #2" in all_labels


@pytest.mark.asyncio
async def test_filters_sessions_squad_classification(client: AsyncClient) -> None:
    """Les sessions squad (is_with_friends=True) sont correctement classées."""
    with _mock_loads(_SAMPLE_DF):
        resp = await client.post(
            "/api/v1/players/demo/filters/resolve",
            json={"filter_mode": "sessions", "sessions": {}},
        )
    data = resp.json()
    squad_labels = data["session_options"]["squad_labels"]
    # sess2 a is_with_friends=True
    assert "2025-01-03 #2" in squad_labels


@pytest.mark.asyncio
async def test_filters_session_filter_applied(client: AsyncClient) -> None:
    """Sélectionner une session réduit bien le nombre de matchs."""
    with _mock_loads(_SAMPLE_DF):
        resp = await client.post(
            "/api/v1/players/demo/filters/resolve",
            json={
                "filter_mode": "sessions",
                "sessions": {"picked_sessions": ["2025-01-01 #1"]},
            },
        )
    data = resp.json()
    # 2 matchs dans session 2025-01-01 #1
    assert data["counts"]["total_matches_after_filters"] == 2


@pytest.mark.asyncio
async def test_filters_invalid_session_ignored(client: AsyncClient) -> None:
    """Session inconnue → 0 matchs, pas d'erreur."""
    with _mock_loads(_SAMPLE_DF):
        resp = await client.post(
            "/api/v1/players/demo/filters/resolve",
            json={
                "filter_mode": "sessions",
                "sessions": {"picked_sessions": ["NotASession"]},
            },
        )
    assert resp.status_code == 200
    assert resp.json()["counts"]["total_matches_after_filters"] == 0


# ---------------------------------------------------------------------------
# Tests normalisation
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_filters_invalid_player_slug_returns_404(client: AsyncClient) -> None:
    """Joueur inconnu → 404."""
    with _mock_loads(_SAMPLE_DF):
        resp = await client.post(
            "/api/v1/players/nonexistent-player/filters/resolve",
            json={"filter_mode": "period", "cascade": {}},
        )
    assert resp.status_code == 404


@pytest.mark.asyncio
async def test_filters_bad_filter_mode_returns_422(client: AsyncClient) -> None:
    """Mode de filtre invalide → 422 validation error."""
    with _mock_loads(_SAMPLE_DF):
        resp = await client.post(
            "/api/v1/players/demo/filters/resolve",
            json={"filter_mode": "bad_mode"},
        )
    assert resp.status_code == 422


@pytest.mark.asyncio
async def test_filters_date_inversion_normalized(client: AsyncClient) -> None:
    """end_date < start_date est normalisé silencieusement (inversé)."""
    with _mock_loads(_SAMPLE_DF):
        resp = await client.post(
            "/api/v1/players/demo/filters/resolve",
            json={
                "filter_mode": "period",
                "period": {"start_date": "2025-01-10", "end_date": "2025-01-01"},
            },
        )
    assert resp.status_code == 200
    effective = resp.json()["effective"]["period"]
    # start_date doit être <= end_date après normalisation
    from datetime import date as dt

    start = dt.fromisoformat(effective["start_date"])
    end = dt.fromisoformat(effective["end_date"])
    assert start <= end
