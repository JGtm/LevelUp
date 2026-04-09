"""Tests pour l'affichage du total de matchs dans la colonne "Taux historique (%)".

Vérifie que :
1. `_add_win_rate_column` (match_history + teammates_helpers) expose `win_rate_hist_total`.
2. `_render_cell` (match_table_html) produit "47% (N matchs)" dans la cellule.
3. `_win_rate_td` (teammates_helpers) idem.
4. Les cas dégradés (valeur nulle, total absent) restent robustes.
"""

from __future__ import annotations

import polars as pl

# ---------------------------------------------------------------------------
# Helpers communs
# ---------------------------------------------------------------------------

_OUTCOME_WIN = 2
_OUTCOME_LOSS = 3


def _make_matches(map_name: str, outcomes: list[int]) -> pl.DataFrame:
    """DataFrame minimal avec map_name + outcome."""
    return pl.DataFrame(
        {
            "map_name": [map_name] * len(outcomes),
            "outcome": outcomes,
        }
    )


# ---------------------------------------------------------------------------
# 1. _add_win_rate_column – match_history.py
# ---------------------------------------------------------------------------


class TestMatchHistoryWinRateColumn:
    """Tests sur _add_win_rate_column de src/ui/pages/match_history.py."""

    def _fn(self):
        from src.ui.pages.match_history import _add_win_rate_column

        return _add_win_rate_column

    def test_expose_total_column(self):
        """win_rate_hist_total doit être présent après join."""
        add_win_rate = self._fn()
        df = _make_matches("Fragmentation", [_OUTCOME_WIN, _OUTCOME_WIN, _OUTCOME_LOSS])
        result = add_win_rate(df, None)
        assert "win_rate_hist_total" in result.columns

    def test_total_matches_correct_count(self):
        """win_rate_hist_total doit refléter le nombre total de matchs sur cette carte."""
        add_win_rate = self._fn()
        df = _make_matches("Streets", [_OUTCOME_WIN, _OUTCOME_LOSS, _OUTCOME_LOSS])
        result = add_win_rate(df, None)
        totals = result["win_rate_hist_total"].to_list()
        assert all(t == 3 for t in totals)

    def test_total_uses_full_df_not_filtered(self):
        """Quand df_full est fourni, le total est calculé dessus, pas sur dff_table."""
        add_win_rate = self._fn()
        df_full = _make_matches("Aquarius", [_OUTCOME_WIN] * 8 + [_OUTCOME_LOSS] * 2)
        df_filtered = _make_matches("Aquarius", [_OUTCOME_WIN, _OUTCOME_WIN])
        result = add_win_rate(df_filtered, df_full)
        totals = result["win_rate_hist_total"].to_list()
        assert all(t == 10 for t in totals)

    def test_win_rate_hist_value(self):
        """win_rate_hist doit être arrondi au dixième (% sur total)."""
        add_win_rate = self._fn()
        df = _make_matches("Live Fire", [_OUTCOME_WIN, _OUTCOME_WIN, _OUTCOME_LOSS])
        result = add_win_rate(df, None)
        rate = result["win_rate_hist"][0]
        assert abs(rate - 66.7) < 0.2

    def test_missing_outcome_column_returns_null(self):
        """Sans colonne outcome, win_rate_hist et win_rate_hist_total doivent être null."""
        add_win_rate = self._fn()
        df = pl.DataFrame({"map_name": ["Streets", "Streets"]})
        result = add_win_rate(df, None)
        assert "win_rate_hist" in result.columns
        assert result["win_rate_hist"].is_null().all()
        assert "win_rate_hist_total" in result.columns
        assert result["win_rate_hist_total"].is_null().all()


# ---------------------------------------------------------------------------
# 2. _add_win_rate_column – teammates_helpers.py
# ---------------------------------------------------------------------------


class TestTeammatesHelpersWinRateColumn:
    """Tests sur _add_win_rate_column de src/ui/pages/teammates_helpers.py."""

    def _fn(self):
        from src.ui.pages.teammates_helpers import _add_win_rate_column

        return _add_win_rate_column

    def test_expose_total_column(self):
        add_win_rate = self._fn()
        df = _make_matches("Catalyst", [_OUTCOME_WIN, _OUTCOME_WIN])
        result = add_win_rate(df, None)
        assert "win_rate_hist_total" in result.columns

    def test_total_correct(self):
        add_win_rate = self._fn()
        df = _make_matches("Recharge", [_OUTCOME_WIN] * 5 + [_OUTCOME_LOSS] * 5)
        result = add_win_rate(df, None)
        assert result["win_rate_hist_total"][0] == 10

    def test_full_df_overrides_filtered(self):
        add_win_rate = self._fn()
        full = _make_matches("Bazaar", [_OUTCOME_WIN] * 6 + [_OUTCOME_LOSS] * 4)
        filtered = _make_matches("Bazaar", [_OUTCOME_WIN, _OUTCOME_LOSS])
        result = add_win_rate(filtered, full)
        assert result["win_rate_hist_total"][0] == 10

    def test_missing_outcome_returns_null_total(self):
        """Sans colonne outcome, win_rate_hist_total doit aussi être null."""
        add_win_rate = self._fn()
        df = pl.DataFrame({"map_name": ["Recharge"]})
        result = add_win_rate(df, None)
        assert "win_rate_hist_total" in result.columns
        assert result["win_rate_hist_total"].is_null().all()


# ---------------------------------------------------------------------------
# 3. _render_cell – match_table_html.py
# ---------------------------------------------------------------------------


class TestMatchTableHtmlRenderCell:
    """Tests sur le rendu HTML de la cellule win_rate_hist."""

    def _render(self, raw_rate: object, total: object, lang: str = "fr") -> str:
        """Appelle _render_cell via un dict de ligne fictive."""
        from unittest.mock import patch

        from src.ui.pages import match_table_html

        with patch.object(match_table_html, "get_lang", return_value=lang):
            return match_table_html._render_cell(
                {"win_rate_hist": raw_rate, "win_rate_hist_total": total},
                "win_rate_hist",
                outcome_code=2,
            )

    def test_displays_percentage_and_total_fr(self):
        html = self._render(47.3, 52)
        assert "47%" in html
        assert "52" in html
        assert "matchs" in html

    def test_displays_matches_en(self):
        html = self._render(60.0, 10, lang="en")
        assert "60%" in html
        assert "10" in html
        assert "matches" in html

    def test_no_total_shows_only_percentage(self):
        html = self._render(55.5, None)
        assert "56%" in html
        assert "matchs" not in html
        assert "matches" not in html

    def test_null_rate_shows_dash(self):
        html = self._render(None, 10)
        assert ">-<" in html

    def test_html_escaped(self):
        """Le contenu ne doit pas contenir de balises non échappées."""
        html = self._render(47.0, 100)
        # Seule la cellule <td>…</td> elle-même doit contenir les balises
        inner = html[html.index(">") + 1 : html.rindex("<")]
        assert "<" not in inner
        assert ">" not in inner


# ---------------------------------------------------------------------------
# 4. _win_rate_td – teammates_helpers.py
# ---------------------------------------------------------------------------


FAKE_COLORS = {"green": "#00ff00", "red": "#ff0000", "cyan": "#00ffff"}


class TestWinRateTd:
    """Tests sur _win_rate_td de src/ui/pages/teammates_helpers.py."""

    def _td(self, raw: object, total: object = None, lang: str = "fr") -> str:
        from unittest.mock import patch

        from src.ui.pages import teammates_helpers

        with patch.object(teammates_helpers, "get_lang", return_value=lang):
            return teammates_helpers._win_rate_td(raw, FAKE_COLORS, total)

    def test_percentage_with_total_fr(self):
        html = self._td(50.0, 30)
        assert "50%" in html
        assert "30" in html
        assert "matchs" in html

    def test_percentage_with_total_en(self):
        html = self._td(50.0, 30, lang="en")
        assert "matches" in html

    def test_no_total_no_count(self):
        html = self._td(50.0)
        assert "matchs" not in html
        assert "matches" not in html
        assert "50%" in html

    def test_high_rate_green(self):
        html = self._td(60.0, 10)
        assert "#00ff00" in html

    def test_low_rate_red(self):
        html = self._td(40.0, 10)
        assert "#ff0000" in html

    def test_mid_rate_cyan(self):
        html = self._td(50.0, 10)
        assert "#00ffff" in html

    def test_null_returns_dash(self):
        html = self._td(None, 10)
        assert "<td>-</td>" in html

    def test_html_escaped_in_td(self):
        html = self._td(47.0, 99)
        inner = html[html.index(">") + 1 : html.rindex("<")]
        assert "<" not in inner
