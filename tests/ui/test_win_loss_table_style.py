"""Tests unitaires pour les helpers HTML de win_loss_table_style.py."""

from __future__ import annotations

from unittest.mock import patch

# ---------------------------------------------------------------------------
# map_name_cell_html
# ---------------------------------------------------------------------------


class TestMapNameCellHtml:
    def test_with_thumb(self) -> None:
        from src.ui.pages.win_loss_table_style import map_name_cell_html

        with patch(
            "src.ui.pages.match_table_html.map_thumb_url",
            return_value="/app/static/maps/Aquarius.png",
        ):
            html = map_name_cell_html("Aquarius")

        assert "map-hover" in html
        assert "map-popup" in html
        assert "/app/static/maps/Aquarius.png" in html
        assert "Aquarius" in html

    def test_no_thumb(self) -> None:
        from src.ui.pages.win_loss_table_style import map_name_cell_html

        with patch("src.ui.pages.match_table_html.map_thumb_url", return_value=None):
            html = map_name_cell_html("UnknownMap")

        assert "map-hover" not in html
        assert "UnknownMap" in html

    def test_none_input(self) -> None:
        from src.ui.pages.win_loss_table_style import map_name_cell_html

        with patch("src.ui.pages.match_table_html.map_thumb_url", return_value=None):
            html = map_name_cell_html(None)

        assert html == "<td>-</td>"

    def test_html_escape_in_url(self) -> None:
        """Les " dans l'URL doivent être échappés pour ne pas casser l'attribut HTML."""
        from src.ui.pages.win_loss_table_style import map_name_cell_html

        with patch(
            "src.ui.pages.match_table_html.map_thumb_url",
            return_value='/app/static/maps/a"b.png',
        ):
            html = map_name_cell_html("SomeMap")

        assert '"/app/static/maps/a"b.png"' not in html  # URL brute non présente
        assert "SomeMap" in html


# ---------------------------------------------------------------------------
# win_rate_cell_html
# ---------------------------------------------------------------------------


class TestWinRateCellHtml:
    def test_win_dominant_is_green(self) -> None:
        from src.ui.pages.win_loss_table_style import _GREEN, win_rate_cell_html

        html = win_rate_cell_html(0.6, 0.3, "60%")
        assert _GREEN in html
        assert "60%" in html

    def test_loss_dominant_is_red(self) -> None:
        from src.ui.pages.win_loss_table_style import _RED, win_rate_cell_html

        html = win_rate_cell_html(0.3, 0.6, "30%")
        assert _RED in html
        assert "30%" in html

    def test_equal_is_violet(self) -> None:
        from src.ui.pages.win_loss_table_style import win_rate_cell_html

        html = win_rate_cell_html(0.5, 0.5, "50%")
        assert "#8E6CFF" in html
        assert "50%" in html

    def test_none_pct_returns_plain(self) -> None:
        from src.ui.pages.win_loss_table_style import win_rate_cell_html

        html = win_rate_cell_html(None, None, "?")
        assert html == "<td>?</td>"

    def test_one_none_pct_returns_plain(self) -> None:
        from src.ui.pages.win_loss_table_style import win_rate_cell_html

        html = win_rate_cell_html(0.6, None, "60%")
        assert html == "<td>60%</td>"


# ---------------------------------------------------------------------------
# loss_rate_cell_html
# ---------------------------------------------------------------------------


class TestLossRateCellHtml:
    def test_win_dominant_loss_is_red(self) -> None:
        """Si win > loss, la cellule de défaite est rouge (défaite rare = négatif)."""
        from src.ui.pages.win_loss_table_style import _RED, loss_rate_cell_html

        html = loss_rate_cell_html(0.6, 0.3, "30%")
        assert _RED in html

    def test_loss_dominant_loss_is_green(self) -> None:
        """Si loss > win, la cellule de défaite est en vert (lisibilité inversée)."""
        from src.ui.pages.win_loss_table_style import _GREEN, loss_rate_cell_html

        html = loss_rate_cell_html(0.3, 0.6, "60%")
        assert _GREEN in html

    def test_equal_is_violet(self) -> None:
        from src.ui.pages.win_loss_table_style import loss_rate_cell_html

        html = loss_rate_cell_html(0.5, 0.5, "50%")
        assert "#8E6CFF" in html

    def test_none_returns_plain(self) -> None:
        from src.ui.pages.win_loss_table_style import loss_rate_cell_html

        html = loss_rate_cell_html(None, None, "?")
        assert html == "<td>?</td>"


# ---------------------------------------------------------------------------
# ratio_cell_html
# ---------------------------------------------------------------------------


class TestRatioCellHtml:
    def test_above_one_is_green(self) -> None:
        from src.ui.pages.win_loss_table_style import _GREEN, ratio_cell_html

        html = ratio_cell_html(2.0, "2.00")
        assert _GREEN in html
        assert "2.00" in html

    def test_below_one_is_red(self) -> None:
        from src.ui.pages.win_loss_table_style import _RED, ratio_cell_html

        html = ratio_cell_html(0.5, "0.50")
        assert _RED in html
        assert "0.50" in html

    def test_exactly_one_is_violet(self) -> None:
        from src.ui.pages.win_loss_table_style import ratio_cell_html

        html = ratio_cell_html(1.0, "1.00")
        assert "#8E6CFF" in html

    def test_none_returns_plain(self) -> None:
        from src.ui.pages.win_loss_table_style import ratio_cell_html

        html = ratio_cell_html(None, "-")
        assert html == "<td>-</td>"


# ---------------------------------------------------------------------------
# perf_cell_html
# ---------------------------------------------------------------------------


class TestPerfCellHtml:
    def test_valid_score_has_class(self) -> None:
        from src.ui.pages.win_loss_table_style import perf_cell_html

        html = perf_cell_html(85.0)
        assert "class=" in html
        assert "85.0" in html

    def test_nan_returns_dash(self) -> None:
        from src.ui.pages.win_loss_table_style import perf_cell_html

        html = perf_cell_html(float("nan"))
        assert html == "<td>-</td>"

    def test_none_returns_dash(self) -> None:
        from src.ui.pages.win_loss_table_style import perf_cell_html

        html = perf_cell_html(None)
        assert html == "<td>-</td>"

    def test_string_number(self) -> None:
        from src.ui.pages.win_loss_table_style import perf_cell_html

        html = perf_cell_html("72.5")
        assert "72.5" in html
        assert "class=" in html


# ---------------------------------------------------------------------------
# plain_cell_html
# ---------------------------------------------------------------------------


class TestPlainCellHtml:
    def test_simple_string(self) -> None:
        from src.ui.pages.win_loss_table_style import plain_cell_html

        assert plain_cell_html("Slayer") == "<td>Slayer</td>"

    def test_fmt_integer(self) -> None:
        from src.ui.pages.win_loss_table_style import plain_cell_html

        assert plain_cell_html(5.7, fmt="%d") == "<td>5</td>"

    def test_none_returns_dash(self) -> None:
        from src.ui.pages.win_loss_table_style import plain_cell_html

        assert plain_cell_html(None) == "<td>-</td>"

    def test_empty_string_returns_dash(self) -> None:
        from src.ui.pages.win_loss_table_style import plain_cell_html

        assert plain_cell_html("") == "<td>-</td>"

    def test_whitespace_only_returns_dash(self) -> None:
        from src.ui.pages.win_loss_table_style import plain_cell_html

        assert plain_cell_html("   ") == "<td>-</td>"

    def test_html_escape(self) -> None:
        from src.ui.pages.win_loss_table_style import plain_cell_html

        html = plain_cell_html("<script>alert(1)</script>")
        assert "<script>" not in html
        assert "&lt;script&gt;" in html

    def test_fmt_fallback_on_non_numeric(self) -> None:
        """Si la valeur n'est pas numérique, on retombe sur str() sans lever d'exception."""
        from src.ui.pages.win_loss_table_style import plain_cell_html

        html = plain_cell_html("abc", fmt="%d")
        assert "abc" in html
