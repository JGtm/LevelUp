"""Tests ciblés pour le rendu léger de Match View."""

from __future__ import annotations

from contextlib import nullcontext
from unittest.mock import patch


def _match_view_translation(key: str, **kwargs: str) -> str:
    if key == "mv_match_id_popover":
        return f"ID {kwargs['short_id']}..."
    if key == "mv_match_id_copy_hint":
        return "hint"
    raise KeyError(key)


def _match_view_block_translation(key: str) -> str:
    translations = {
        "mv_performance": "Performance",
        "mv_thumbnail_unavailable": "thumb missing",
    }
    return translations[key]


def _match_view_kpi_translation(key: str) -> str:
    translations = {
        "outcome_domination": "Domination",
    }
    return translations[key]


def test_short_match_id_truncates_long_values() -> None:
    from src.ui.pages.match_view import _short_match_id

    assert _short_match_id("match-id-123456") == "match-id"


def test_short_match_id_keeps_short_values() -> None:
    from src.ui.pages.match_view import _short_match_id

    assert _short_match_id("abc123") == "abc123"


def test_render_match_id_badge_uses_native_popover() -> None:
    with (
        patch("src.ui.pages.match_view.st") as mock_st,
        patch(
            "src.ui.pages.match_view.t",
            side_effect=_match_view_translation,
        ),
    ):
        mock_st.popover.return_value = nullcontext()

        from src.ui.pages import match_view

        match_view._render_match_id_badge("match-id-123456")

    mock_st.popover.assert_called_once_with("ID match-id...")
    mock_st.caption.assert_called_once_with("hint")
    mock_st.code.assert_called_once_with("match-id-123456", language=None)


def test_render_match_id_badge_skips_empty_value() -> None:
    with patch("src.ui.pages.match_view.st") as mock_st:
        from src.ui.pages import match_view

        match_view._render_match_id_badge("")

    mock_st.popover.assert_not_called()


def test_render_map_and_rank_uses_native_streamlit_columns(tmp_path) -> None:
    thumb = tmp_path / "map.png"
    thumb.write_bytes(b"png")

    with (
        patch("src.ui.pages.match_view.st") as mock_st,
        patch(
            "src.ui.pages.match_view.map_thumb_path",
            return_value=thumb,
        ),
        patch(
            "src.ui.pages.match_view._build_match_rank_html",
            return_value="<div>rank</div>",
        ),
        patch(
            "src.ui.pages.match_view.t",
            side_effect=_match_view_block_translation,
        ),
    ):
        mock_st.columns.return_value = (nullcontext(), nullcontext(), nullcontext())

        from src.ui.pages import match_view

        match_view._render_map_and_rank(
            {},
            map_display="Aquarius",
            db_path="player.duckdb",
            match_id="match-001",
            db_key=None,
            had_bot=False,
            outcome_code=2,
            perf_display="82",
            perf_color="#12ab34",
        )

    mock_st.columns.assert_called_once_with([1.8, 0.7, 1.2])
    mock_st.image.assert_called_once_with(str(thumb), width="stretch")
    assert any(
        call.args
        and call.args[0] == "<div>rank</div>"
        and call.kwargs.get("unsafe_allow_html") is True
        for call in mock_st.markdown.call_args_list
    )


def test_render_map_and_rank_without_rank_keeps_native_thumbnail_only(tmp_path) -> None:
    thumb = tmp_path / "map.png"
    thumb.write_bytes(b"png")

    with (
        patch("src.ui.pages.match_view.st") as mock_st,
        patch(
            "src.ui.pages.match_view.map_thumb_path",
            return_value=thumb,
        ),
        patch(
            "src.ui.pages.match_view._build_match_rank_html",
            return_value=None,
        ),
    ):
        from src.ui.pages import match_view

        match_view._render_map_and_rank(
            {},
            map_display="Aquarius",
            db_path="player.duckdb",
            match_id="match-001",
            db_key=None,
            had_bot=False,
            outcome_code=2,
            perf_display="82",
            perf_color="#12ab34",
        )

    mock_st.columns.assert_not_called()
    mock_st.image.assert_called_once_with(str(thumb), width="stretch")
    mock_st.markdown.assert_not_called()


def test_render_map_thumbnail_falls_back_to_info() -> None:
    with (
        patch("src.ui.pages.match_view.st") as mock_st,
        patch(
            "src.ui.pages.match_view.t",
            return_value="thumb missing",
        ),
    ):
        from src.ui.pages import match_view

        match_view._render_map_thumbnail(None)

    mock_st.info.assert_called_once_with("thumb missing")


def test_render_simple_kpi_tile_uses_native_container() -> None:
    with patch("src.ui.pages.match_view.st") as mock_st:
        mock_st.container.return_value = nullcontext()

        from src.ui.pages import match_view

        match_view._render_simple_kpi_tile("Quick Play")

    mock_st.container.assert_called_once_with(border=True)
    mock_st.markdown.assert_called_once()
    assert mock_st.markdown.call_args.kwargs.get("unsafe_allow_html") is True


def test_render_score_kpi_tile_renders_badge_when_present() -> None:
    with (
        patch("src.ui.pages.match_view.st") as mock_st,
        patch(
            "src.ui.pages.match_view.t",
            side_effect=_match_view_kpi_translation,
        ),
    ):
        mock_st.container.return_value = nullcontext()

        from src.ui.pages import match_view

        match_view._render_score_kpi_tile("50 - 42", "#33ff99", 1)

    mock_st.container.assert_called_once_with(border=True)
    rendered = mock_st.markdown.call_args.args[0]
    assert "50 - 42" in rendered
    assert "Domination" in rendered


def test_render_kpi_cards_uses_native_columns() -> None:
    with (
        patch("src.ui.pages.match_view.st") as mock_st,
        patch(
            "src.ui.pages.match_view.get_lang",
            return_value="fr",
        ),
        patch(
            "src.ui.pages.match_view.format_date_fr",
            return_value="12/04/2026",
        ),
        patch(
            "src.ui.pages.match_view._display_map",
            return_value="Aquarius",
        ),
        patch(
            "src.ui.pages.match_view._render_simple_kpi_tile",
        ) as mock_simple,
        patch(
            "src.ui.pages.match_view._render_score_kpi_tile",
        ) as mock_score,
    ):
        mock_st.columns.return_value = (
            nullcontext(),
            nullcontext(),
            nullcontext(),
            nullcontext(),
        )

        from src.ui.pages import match_view

        match_view._render_kpi_cards(
            last_time="2026-04-12T10:00:00Z",
            outcome_code=2,
            outcome_label="Victoire",
            outcome_color="#12ab34",
            score_label="50 - 42",
            had_bot=False,
            dominance_flag=None,
            row={
                "playlist_name_fr": "Quick Play",
                "mode_ui": "Slayer",
            },
        )

    mock_st.columns.assert_called_once_with(4)
    assert [call.args[0] for call in mock_simple.call_args_list] == [
        "12/04/2026",
        "Quick Play",
        "Slayer sur Aquarius",
    ]
    mock_score.assert_called_once_with("50 - 42", "#12ab34", None)
