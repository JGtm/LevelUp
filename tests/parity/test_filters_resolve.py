"""Tests de parité — résolution des filtres.

Ces tests comparent les résultats de ``POST /filters/resolve`` (backend FastAPI)
avec les golden values issues de Streamlit. Ils skipent proprement si le corpus
de référence n'existe pas encore.

Pour les exécuter :
    1. python scripts/create_test_corpus.py --gamertag <Gamertag>
    2. python -m pytest tests/parity/test_filters_resolve.py -v
"""

from __future__ import annotations

import pytest

from apps.api.app.schemas.filters import CascadeInput, FilterContextInput, PeriodInput
from tests.parity.conftest import requires_corpus

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_full_period_ctx() -> FilterContextInput:
    return FilterContextInput(
        filter_mode="period",
        period=PeriodInput(start_date=None, end_date=None),
        cascade=CascadeInput(
            experience_types=[],
            playlists=[],
            modes=[],
            maps=[],
        ),
    )


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


@requires_corpus
def test_filters_resolve_full_period_returns_valid_schema(player_context, golden_match_history):
    """Scope complet : le résolveur retourne le bon type de schéma."""
    from apps.api.app.services.filter_service import resolve_filters

    ctx = _make_full_period_ctx()
    result = resolve_filters(player_context, ctx)

    assert result.counts is not None
    assert result.available_options is not None
    assert result.effective is not None


@requires_corpus
def test_filters_resolve_total_matches_matches_golden(player_context, golden_match_history):
    """Scope complet : le total de matchs corrobore les golden values."""
    from apps.api.app.services.filter_service import resolve_filters

    ctx = _make_full_period_ctx()
    result = resolve_filters(player_context, ctx)

    expected_total: int = golden_match_history.get("total_matches", 0)
    if expected_total == 0:
        pytest.skip("Golden values total_matches non renseigné")

    actual = result.counts.total_matches_before_filters
    # Tolérance ±1 % pour absorber les différences mineures de synchronisation
    tolerance = max(1, int(expected_total * 0.01))
    assert abs(actual - expected_total) <= tolerance, (
        f"total_matches : attendu ~{expected_total}, obtenu {actual}"
    )


@requires_corpus
def test_filters_resolve_options_not_empty(player_context):
    """Scope complet : des options de playlist/map/mode sont disponibles."""
    from apps.api.app.services.filter_service import resolve_filters

    ctx = _make_full_period_ctx()
    result = resolve_filters(player_context, ctx)

    assert len(result.available_options.playlists) > 0, "Aucune playlist disponible"
    assert len(result.available_options.maps) > 0, "Aucune carte disponible"
    assert len(result.available_options.modes) > 0, "Aucun mode disponible"


@requires_corpus
def test_filters_resolve_sessions_not_empty(player_context):
    """Scope complet : des sessions sont détectées."""
    from apps.api.app.services.filter_service import resolve_filters

    ctx = _make_full_period_ctx()
    result = resolve_filters(player_context, ctx)

    assert result.session_options is not None
    assert len(result.session_options.all_sessions) > 0, "Aucune session détectée"


@requires_corpus
def test_filters_resolve_effective_matches_input(player_context):
    """Le contexte effectif retourné est cohérent avec l'input."""
    from apps.api.app.services.filter_service import resolve_filters

    ctx = _make_full_period_ctx()
    result = resolve_filters(player_context, ctx)

    assert result.effective.filter_mode == "period"


@requires_corpus
def test_filters_resolve_cascade_playlist_reduces_count(player_context):
    """Filtrer par une playlist réduit (ou maintient) le nombre de matchs."""
    from apps.api.app.services.filter_service import resolve_filters

    ctx_full = _make_full_period_ctx()
    result_full = resolve_filters(player_context, ctx_full)

    playlists = result_full.available_options.playlists
    if not playlists:
        pytest.skip("Aucune playlist disponible dans le corpus")

    first_playlist = playlists[0].value
    ctx_filtered = FilterContextInput(
        filter_mode="period",
        period=PeriodInput(start_date=None, end_date=None),
        cascade=CascadeInput(
            experience_types=[],
            playlists=[first_playlist],
            modes=[],
            maps=[],
        ),
    )
    result_filtered = resolve_filters(player_context, ctx_filtered)

    assert (
        result_filtered.counts.total_matches_after_filters
        <= result_full.counts.total_matches_after_filters
    ), "Filtrer par playlist a augmenté le nombre de matchs (impossible)"
