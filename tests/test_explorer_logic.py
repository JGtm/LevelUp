"""Tests pour les modules Explorer — logique, enrichissement, HTML."""

from __future__ import annotations

from datetime import date, datetime

import polars as pl
import pytest

from src.ui.pages.explorer_logic import (
    classify_experience_type,
    filter_by_date,
    filter_by_squad,
    find_closest_date,
    fuzzy_search_gamertags,
    get_distinct_dates,
    split_by_team,
)

# ---------------------------------------------------------------------------
# fuzzy_search_gamertags
# ---------------------------------------------------------------------------


class TestFuzzySearchGamertags:
    """Tests de la recherche floue par gamertag."""

    def test_exact_match(self) -> None:
        result = fuzzy_search_gamertags("Alpha", ["Alpha", "Beta", "Gamma"])
        assert "Alpha" in result

    def test_prefix_match(self) -> None:
        result = fuzzy_search_gamertags("Alp", ["Alpha", "Beta", "AlphaWolf"])
        assert any("Alp" in r for r in result)

    def test_no_match(self) -> None:
        result = fuzzy_search_gamertags("zzzzzzz", ["Alpha", "Beta"])
        assert result == []

    def test_empty_list(self) -> None:
        result = fuzzy_search_gamertags("Alpha", [])
        assert result == []

    def test_case_insensitive(self) -> None:
        result = fuzzy_search_gamertags("alpha", ["Alpha", "ALPHA", "beta"])
        assert len(result) >= 1


# ---------------------------------------------------------------------------
# classify_experience_type
# ---------------------------------------------------------------------------


class TestClassifyExperienceType:
    """Tests de classification des playlists."""

    def test_ranked(self) -> None:
        assert classify_experience_type("Ranked Arena") == "ranked"

    def test_pve(self) -> None:
        assert classify_experience_type("Firefight Heroic") == "pve"

    def test_unranked(self) -> None:
        assert classify_experience_type("Quick Play") == "unranked"

    def test_none(self) -> None:
        assert classify_experience_type(None) == "unranked"


# ---------------------------------------------------------------------------
# filter_by_date / find_closest_date / get_distinct_dates
# ---------------------------------------------------------------------------


class TestDateFilters:
    """Tests des filtres par date."""

    @pytest.fixture()
    def sample_df(self) -> pl.DataFrame:
        return pl.DataFrame(
            {
                "match_id": ["a", "b", "c"],
                "start_time": [
                    datetime(2025, 1, 10, 14, 0),
                    datetime(2025, 1, 10, 16, 0),
                    datetime(2025, 1, 12, 10, 0),
                ],
                "date": [
                    date(2025, 1, 10),
                    date(2025, 1, 10),
                    date(2025, 1, 12),
                ],
            }
        )

    def test_filter_by_date(self, sample_df: pl.DataFrame) -> None:
        result = filter_by_date(sample_df, date(2025, 1, 10))
        assert len(result) == 2

    def test_filter_by_date_no_match(self, sample_df: pl.DataFrame) -> None:
        result = filter_by_date(sample_df, date(2025, 1, 11))
        assert result.is_empty()

    def test_get_distinct_dates(self, sample_df: pl.DataFrame) -> None:
        dates = get_distinct_dates(sample_df)
        assert dates == [date(2025, 1, 10), date(2025, 1, 12)]

    def test_find_closest_date(self, sample_df: pl.DataFrame) -> None:
        closest = find_closest_date(sample_df, date(2025, 1, 11))
        assert closest in (date(2025, 1, 10), date(2025, 1, 12))


# ---------------------------------------------------------------------------
# filter_by_squad
# ---------------------------------------------------------------------------


class TestFilterBySquad:
    """Tests du filtre escouade/solo."""

    def test_filter_squad(self) -> None:
        df = pl.DataFrame({"match_id": ["a", "b", "c"]})
        friends_map = {"a": True, "b": False, "c": True}
        result = filter_by_squad(df, friends_map, "squad")
        assert len(result) == 2  # a et c

    def test_filter_solo(self) -> None:
        df = pl.DataFrame({"match_id": ["a", "b", "c"]})
        friends_map = {"a": True, "b": False, "c": True}
        result = filter_by_squad(df, friends_map, "solo")
        assert len(result) == 1  # b


# ---------------------------------------------------------------------------
# split_by_team
# ---------------------------------------------------------------------------


class TestSplitByTeam:
    """Tests de la séparation alliés/adversaires."""

    def test_split(self) -> None:
        df = pl.DataFrame(
            {
                "match_id": ["a", "b", "c"],
                "player_team_id": [1, 1, 2],
                "target_team_id": [1, 2, 2],
            }
        )
        allies, enemies = split_by_team(df)
        assert len(allies) == 2
        assert len(enemies) == 1

    def test_empty(self) -> None:
        df = pl.DataFrame(
            {
                "match_id": [],
                "player_team_id": [],
                "target_team_id": [],
            }
        ).cast(
            {
                "match_id": pl.Utf8,
                "player_team_id": pl.Int64,
                "target_team_id": pl.Int64,
            }
        )
        allies, enemies = split_by_team(df)
        assert allies.is_empty()
        assert enemies.is_empty()


# ===========================================================================
# Tests explorer_enrich
# ===========================================================================


class TestEnrichForTable:
    """Tests de l'enrichissement DataFrame pour le tableau HTML."""

    def test_adds_score_column(self) -> None:
        from src.ui.pages.explorer_enrich import _add_score_column

        df = pl.DataFrame(
            {
                "my_team_score": [10, None],
                "enemy_team_score": [5, 3],
            }
        )
        result = _add_score_column(df)
        assert "score" in result.columns
        first = result["score"][0]
        assert "10" in first and "5" in first

    def test_adds_score_missing_columns(self) -> None:
        from src.ui.pages.explorer_enrich import _add_score_column

        df = pl.DataFrame({"match_id": ["a"]})
        result = _add_score_column(df)
        assert "score" in result.columns
        assert result["score"][0] == "-"

    def test_adds_avg_life_column(self) -> None:
        from src.ui.pages.explorer_enrich import _add_avg_life_column

        df = pl.DataFrame({"average_life_seconds": [90.0, 65.0]})
        result = _add_avg_life_column(df)
        assert "average_life_mmss" in result.columns
        assert result["average_life_mmss"][0] == "1:30"
        assert result["average_life_mmss"][1] == "1:05"

    def test_adds_delta_mmr(self) -> None:
        from src.ui.pages.explorer_enrich import _add_delta_mmr

        df = pl.DataFrame({"team_mmr": [1200.0], "enemy_mmr": [1150.0]})
        result = _add_delta_mmr(df)
        assert "delta_mmr" in result.columns
        assert abs(result["delta_mmr"][0] - 50.0) < 0.01

    def test_delta_mmr_no_column(self) -> None:
        from src.ui.pages.explorer_enrich import _add_delta_mmr

        df = pl.DataFrame({"match_id": ["a"]})
        result = _add_delta_mmr(df)
        assert "delta_mmr" in result.columns
        assert result["delta_mmr"][0] is None

    def test_enrich_common_matches_adds_nulls(self) -> None:
        from src.ui.pages.explorer_enrich import enrich_common_matches

        df = pl.DataFrame(
            {
                "match_id": ["a"],
                "start_time": [datetime(2025, 1, 10, 14, 0)],
                "outcome": [2],
            }
        )
        result = enrich_common_matches(df, None)
        # Doit avoir les colonnes null ajoutées
        for col in ("score", "delta_mmr", "performance", "match_url"):
            assert col in result.columns


# ===========================================================================
# Tests explorer_data
# ===========================================================================


class TestExplorerData:
    """Tests de l'accès données Explorer."""

    def test_shared_db_path_derivation(self) -> None:
        from src.ui.pages.explorer_data import _shared_db_path

        path = _shared_db_path("/data/players/TestGT/stats.duckdb")
        assert path.name == "shared_matches.duckdb"
        assert "warehouse" in str(path)

    def test_load_is_with_friends_empty(self) -> None:
        from src.ui.pages.explorer_data import load_is_with_friends

        # Pas de match_ids → dict vide
        assert load_is_with_friends("nonexistent.duckdb", []) == {}

    def test_get_all_gamertags_file_missing(self) -> None:
        from src.ui.pages.explorer_data import get_all_gamertags

        # DB inexistante → liste vide (avec warning)
        result = get_all_gamertags("/nonexistent/path/stats.duckdb")
        assert result == []

    def test_resolve_gamertag_file_missing(self) -> None:
        from src.ui.pages.explorer_data import resolve_gamertag_to_xuid

        result = resolve_gamertag_to_xuid("/nonexistent/path/stats.duckdb", "TestGT")
        assert result is None

    def test_load_common_matches_file_missing(self) -> None:
        from src.ui.pages.explorer_data import load_common_matches

        result = load_common_matches(
            "/nonexistent/path/stats.duckdb",
            "xuid1",
            "xuid2",
        )
        assert isinstance(result, pl.DataFrame)
        assert result.is_empty()


# ===========================================================================
# Tests match_table_html
# ===========================================================================


class TestMatchTableHTML:
    """Tests du rendu de tableau HTML."""

    def test_app_url(self) -> None:
        from src.ui.pages.match_table_html import app_url

        url = app_url("Explorer", match_id="abc123")
        assert "page=Explorer" in url
        assert "match_id=abc123" in url

    def test_app_url_no_empty_params(self) -> None:
        from src.ui.pages.match_table_html import app_url

        url = app_url("Explorer", match_id="")
        assert "match_id" not in url

    def test_gamertag_link(self) -> None:
        from src.ui.pages.match_table_html import gamertag_link

        html = gamertag_link("Test Player")
        assert "Test Player" in html
        assert "gamertag=Test" in html
        assert "<a " in html
        assert "target='_blank'" in html

    def test_gamertag_link_escapes_html(self) -> None:
        from src.ui.pages.match_table_html import gamertag_link

        html = gamertag_link("<script>alert(1)</script>")
        assert "<script>" not in html
        assert "&lt;script&gt;" in html

    def test_fmt_value(self) -> None:
        from src.ui.pages.match_table_html import fmt_value

        assert fmt_value(None) == "-"
        assert fmt_value("") == "-"
        assert fmt_value("test") == "test"
        assert fmt_value(42) == "42"

    def test_fmt_float(self) -> None:
        from src.ui.pages.match_table_html import fmt_float

        assert fmt_float(None) == "-"
        assert fmt_float(1.234) == "1.23"
        assert fmt_float(1.2, decimals=1) == "1.2"

    def test_fmt_mmr_int(self) -> None:
        from src.ui.pages.match_table_html import fmt_mmr_int

        assert fmt_mmr_int(None) == "-"
        assert fmt_mmr_int(1234.7) == "1235"
        assert fmt_mmr_int("not_a_number") == "-"

    def test_outcome_style_win(self) -> None:
        from src.ui.pages.match_table_html import outcome_style

        style = outcome_style(2, "Victoire")
        assert "font-weight" in style

    def test_mmr_gap_style_positive(self) -> None:
        from src.ui.pages.match_table_html import mmr_gap_style

        style = mmr_gap_style(50.0)
        assert "font-weight" in style

    def test_mmr_gap_style_none(self) -> None:
        from src.ui.pages.match_table_html import mmr_gap_style

        assert mmr_gap_style(None) == ""

    def test_render_match_table_html_basic(self) -> None:
        from src.ui.pages.match_table_html import render_match_table_html

        df = pl.DataFrame(
            {
                "match_id": ["abc123"],
                "start_time": [datetime(2025, 1, 10, 14, 0)],
                "start_time_fr": ["10/01/2025 14:00"],
                "map_name": ["Aquarius"],
                "playlist_fr": ["Quick Play"],
                "mode_ui": ["Slayer"],
                "outcome": [2],
                "outcome_label": ["Victoire"],
                "score": ["50 - 40"],
                "performance": [85],
                "team_mmr": [1200.0],
                "enemy_mmr": [1150.0],
                "delta_mmr": [50.0],
                "kda": [2.5],
                "kills": [15],
                "deaths": [6],
                "max_killing_spree": [5],
                "headshot_kills": [8],
                "average_life_mmss": ["1:30"],
                "assists": [3],
                "accuracy": [0.45],
                "ratio": [2.5],
                "match_url": ["https://halowaypoint.com/..."],
            }
        )
        html = render_match_table_html(df, waypoint_player="TestGT")
        assert "<table" in html
        assert "os-table" in html
        assert "Aquarius" in html
        assert "abc123" in html

    def test_render_match_table_html_empty(self) -> None:
        from src.ui.pages.match_table_html import render_match_table_html

        df = pl.DataFrame(
            {
                "match_id": [],
                "start_time": [],
            }
        ).cast({"match_id": pl.Utf8, "start_time": pl.Datetime})
        html = render_match_table_html(df)
        assert "<table" in html
        assert "<tbody></tbody>" in html
