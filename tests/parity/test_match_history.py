"""Tests de parité — historique des parties (Match History).

Compare les résultats de ``POST /players/{slug}/pages/match-history`` avec les
golden values issues de Streamlit.

Pour les exécuter :
    1. python scripts/create_test_corpus.py --gamertag <Gamertag>
    2. python -m pytest tests/parity/test_match_history.py -v
"""

from __future__ import annotations

import pytest

from apps.api.app.schemas.filters import CascadeInput, FilterContextInput, PeriodInput
from apps.api.app.schemas.match_history import MatchHistoryQueryRequest
from tests.parity.conftest import requires_corpus


def _full_period_request() -> MatchHistoryQueryRequest:
    return MatchHistoryQueryRequest(
        filters=FilterContextInput(
            filter_mode="period",
            period=PeriodInput(start_date=None, end_date=None),
            cascade=CascadeInput(experience_types=[], playlists=[], modes=[], maps=[]),
        ),
        pagination={"page": 1, "page_size": 50},
        include_export_hint=False,
    )


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


@requires_corpus
def test_match_history_returns_valid_schema(player_context):
    """L'historique retourne un schéma complet."""
    from apps.api.app.services.match_history_service import get_match_history_page

    req = _full_period_request()
    result = get_match_history_page(player_context, req)

    assert result is not None
    assert result.summary is not None
    assert result.table is not None


@requires_corpus
def test_match_history_total_matches_golden(player_context, golden_match_history):
    """Le total de matchs (scope complet) corrobore les golden values."""
    from apps.api.app.services.match_history_service import get_match_history_page

    expected: int = golden_match_history.get("total_matches", 0)
    if expected == 0:
        pytest.skip("total_matches non renseigné dans golden_values/match_history_full.json")

    req = _full_period_request()
    result = get_match_history_page(player_context, req)

    actual = result.summary.total_matches_scoped
    tolerance = max(1, int(expected * 0.01))
    assert abs(actual - expected) <= tolerance, (
        f"total_matches : attendu ~{expected}, obtenu {actual}"
    )


@requires_corpus
def test_match_history_rows_have_required_fields(player_context):
    """Chaque ligne contient les champs obligatoires."""
    from apps.api.app.services.match_history_service import get_match_history_page

    req = _full_period_request()
    result = get_match_history_page(player_context, req)

    for row in result.table.items:
        assert row.match_id, "match_id vide"
        assert row.start_time, "start_time vide"
        assert row.map_ui is not None, "map_ui nul"
        assert row.mode_ui is not None, "mode_ui nul"


@requires_corpus
def test_match_history_pagination_consistent(player_context):
    """La pagination est cohérente (total = items pages × page_size + reste)."""
    from apps.api.app.services.match_history_service import get_match_history_page

    req = _full_period_request()
    result = get_match_history_page(player_context, req)

    pag = result.table.pagination
    assert pag.total >= len(result.table.items), (
        f"total={pag.total} < items retournés={len(result.table.items)}"
    )
    assert pag.page >= 1
    assert pag.page_size >= 1


@requires_corpus
def test_match_history_first_match_id_golden(player_context, golden_match_history):
    """Le premier match (le plus récent) correspond au golden value."""
    from apps.api.app.services.match_history_service import get_match_history_page

    expected_id: str | None = golden_match_history.get("first_match_id")
    if not expected_id or expected_id.startswith("REPLACE"):
        pytest.skip("first_match_id non renseigné dans golden_values/match_history_full.json")

    req = _full_period_request()
    result = get_match_history_page(player_context, req)

    if not result.table.items:
        pytest.skip("Aucune ligne retournée")

    actual_id = result.table.items[0].match_id
    assert actual_id == expected_id, f"Premier match_id : attendu {expected_id}, obtenu {actual_id}"


@requires_corpus
def test_match_history_outcomes_valid_codes(player_context):
    """Les codes de résultat sont dans la plage attendue (1–4)."""
    from apps.api.app.services.match_history_service import get_match_history_page

    req = _full_period_request()
    result = get_match_history_page(player_context, req)

    VALID_CODES = {None, 1, 2, 3, 4}
    for row in result.table.items:
        assert row.outcome_code in VALID_CODES, (
            f"outcome_code invalide : {row.outcome_code} pour match {row.match_id}"
        )
