"""Tests de parité — page Carrière.

Compare les résultats de ``GET /players/{slug}/pages/career`` avec les
golden values générées par ``create_test_corpus.py`` depuis Streamlit.

Pour les exécuter :
    1. python scripts/create_test_corpus.py --gamertag <Gamertag>
    2. python -m pytest tests/parity/test_career_page.py -v
"""

from __future__ import annotations

import pytest

from tests.parity.conftest import requires_corpus

# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


@requires_corpus
def test_career_page_returns_valid_schema(player_context):
    """La page Carrière retourne un schéma complet et non-nul."""
    from apps.api.app.services.career_service import get_career_page

    result = get_career_page(player_context)

    assert result is not None
    assert result.charts is not None


@requires_corpus
def test_career_summary_rank_matches_golden(player_context, golden_career):
    """Le rang de carrière retourné correspond aux golden values."""
    from apps.api.app.services.career_service import get_career_page

    expected_rank: int | None = golden_career.get("rank_number")
    if expected_rank is None:
        pytest.skip("rank_number non renseigné dans golden_values/career.json")

    result = get_career_page(player_context)
    assert result.summary is not None, "summary manquant dans la réponse"
    assert result.summary.rank_number == expected_rank, (
        f"Rang : attendu {expected_rank}, obtenu {result.summary.rank_number}"
    )


@requires_corpus
def test_career_xp_total_matches_golden(player_context, golden_career):
    """Le XP total retourné correspond aux golden values (±0.1%)."""
    from apps.api.app.services.career_service import get_career_page

    expected_xp: int = golden_career.get("xp_total", 0)
    if expected_xp == 0:
        pytest.skip("xp_total non renseigné dans golden_values/career.json")

    result = get_career_page(player_context)
    assert result.summary is not None

    tolerance = max(1, int(expected_xp * 0.001))
    assert abs(result.summary.xp_total - expected_xp) <= tolerance, (
        f"XP total : attendu ~{expected_xp}, obtenu {result.summary.xp_total}"
    )


@requires_corpus
def test_career_lusr_present_when_golden_indicates(player_context, golden_career):
    """Si les golden values indiquent un LUSR non-nul, la section est présente."""
    from apps.api.app.services.career_service import get_career_page

    expected_lusr = golden_career.get("lusr_rating")
    if expected_lusr is None:
        pytest.skip("lusr_rating non renseigné dans golden_values/career.json")

    result = get_career_page(player_context)
    assert result.lusr is not None, "Section LUSR absente alors que le joueur en a un"
    assert result.lusr.current_rating is not None


@requires_corpus
def test_career_hero_progress_is_valid(player_context, golden_career):
    """La progression Héros est entre 0 et 100%."""
    from apps.api.app.services.career_service import get_career_page

    result = get_career_page(player_context)

    if result.hero_progress is None:
        # Si le joueur est en dessous du rang Héros, c'est normal
        return

    assert 0.0 <= result.hero_progress.percentage <= 100.0, (
        f"Pourcentage Héros hors [0, 100] : {result.hero_progress.percentage}"
    )


@requires_corpus
def test_career_xp_history_ordered_ascending(player_context):
    """L'historique XP est trié par date ascendante."""
    from apps.api.app.services.career_service import get_career_page

    result = get_career_page(player_context)

    dates = [p.recorded_at for p in result.xp_history]
    assert dates == sorted(dates), "L'historique XP n'est pas trié par date"


@requires_corpus
def test_career_top_matches_preview_non_empty(player_context):
    """La preview des top matches contient au moins un match."""
    from apps.api.app.services.career_service import get_career_page

    result = get_career_page(player_context)

    assert len(result.top_matches_preview) > 0, "Aucun top match retourné"


@requires_corpus
def test_career_all_top_matches_have_match_id(player_context):
    """Chaque top match a un match_id non vide."""
    from apps.api.app.services.career_service import get_career_page

    result = get_career_page(player_context)

    for tm in result.top_matches_preview:
        assert tm.match_id, f"match_id manquant dans un top match : {tm}"
