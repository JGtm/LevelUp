"""Tests de propagation du hover map aux tableaux HTML secondaires."""

from __future__ import annotations

from unittest.mock import patch

import polars as pl


def test_career_top_matches_table_uses_map_hover() -> None:
    from src.ui.pages.career_top_matches_render import _build_top_table_html

    rows = [
        {
            "start_time": "2026-03-17T12:00:00",
            "game_variant_name": "Slayer",
            "playlist_name": "Ranked Arena",
            "map_name": "Aquarius",
            "my_team_score": 50,
            "enemy_team_score": 42,
            "kills": 15,
            "deaths": 8,
            "assists": 3,
            "time_played_seconds": 720,
            "dominance_flag": 1,
        }
    ]

    with patch(
        "src.ui.pages.match_table_html.map_thumb_url",
        return_value="/app/static/maps/Aquarius.png",
    ):
        html = _build_top_table_html(rows, best=True)

    assert "os-table-wrap--map-hover" in html
    assert "map-hover" in html
    assert "map-popup" in html


def test_teammates_history_rows_use_map_hover() -> None:
    from src.ui.pages.teammates_helpers import _build_html_rows

    view = pl.DataFrame(
        {
            "match_id": ["m1"],
            "map_name": ["Aquarius"],
            "match_url": [""],
            "outcome_label": ["Victoire"],
        }
    )
    cols = [("Map", "map_name")]
    colors = {"green": "#0f0", "red": "#f00", "violet": "#90f"}

    with patch(
        "src.ui.pages.match_table_html.map_thumb_url",
        return_value="/app/static/maps/Aquarius.png",
    ):
        rows = _build_html_rows(view, cols, colors)

    assert len(rows) == 1
    assert "map-hover" in rows[0]
    assert "map-popup" in rows[0]


def test_session_history_html_uses_map_hover(mock_st) -> None:
    from src.ui.pages import _session_compare_history as mod

    ms = mock_st(mod)
    df_display = pl.DataFrame({mod.t("col_map"): ["Aquarius"]})

    with patch(
        "src.ui.pages.match_table_html.map_thumb_url",
        return_value="/app/static/maps/Aquarius.png",
    ):
        mod._render_history_html(df_display)

    html = ms.calls["markdown"].call_args.args[0]
    assert "os-table-wrap--map-hover" in html
    assert "map-hover" in html
    assert "map-popup" in html
