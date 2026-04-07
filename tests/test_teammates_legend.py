"""Tests pour le panneau de légende joueurs (page Coéquipiers).

Couvre :
- teammates_legend.render_player_legend_panel — les 3 modes
- teammates_charts._hide_legend — masquage légende Plotly
"""

from __future__ import annotations

from unittest.mock import MagicMock, patch

import plotly.graph_objects as go

# ─────────────────────────────────────────────────────────────────────────────
# Helpers
# ─────────────────────────────────────────────────────────────────────────────

_COLORS = {"Alice": "#E69F00", "Bob": "#56B4E9"}


def _import_legend():
    from src.ui.pages import teammates_legend

    return teammates_legend


# ─────────────────────────────────────────────────────────────────────────────
# _hide_legend (teammates_charts)
# ─────────────────────────────────────────────────────────────────────────────


class TestHideLegend:
    def test_sets_showlegend_false(self) -> None:
        """_hide_legend doit mettre showlegend=False dans le layout."""
        from src.ui.pages.teammates_charts import _hide_legend

        fig = go.Figure()
        fig.update_layout(showlegend=True)
        result = _hide_legend(fig)
        assert result.layout.showlegend is False

    def test_returns_same_figure(self) -> None:
        """_hide_legend doit retourner le même objet Figure (modification in-place via update_layout)."""
        from src.ui.pages.teammates_charts import _hide_legend

        fig = go.Figure()
        result = _hide_legend(fig)
        assert result is fig

    def test_does_not_remove_traces(self) -> None:
        """_hide_legend ne doit pas supprimer les traces existantes."""
        from src.ui.pages.teammates_charts import _hide_legend

        fig = go.Figure(data=[go.Bar(x=[1], y=[1], name="Test")])
        _hide_legend(fig)
        assert len(fig.data) == 1


# ─────────────────────────────────────────────────────────────────────────────
# render_player_legend_panel — mode "hidden"
# ─────────────────────────────────────────────────────────────────────────────


class TestRenderPlayerLegendPanelHidden:
    def test_empty_dict_does_nothing(self) -> None:
        """Un dict vide ne doit pas appeler st.markdown."""
        legend = _import_legend()
        with patch.object(legend.st, "markdown") as mock_md:
            legend.render_player_legend_panel({})
            mock_md.assert_not_called()

    def test_hidden_mode_does_not_call_st(self) -> None:
        """Mode 'hidden' ne doit pas appeler st.markdown."""
        legend = _import_legend()
        original_mode = legend._PANEL_MODE
        try:
            legend._PANEL_MODE = "hidden"
            with patch.object(legend.st, "markdown") as mock_md:
                legend.render_player_legend_panel(_COLORS)
                mock_md.assert_not_called()
        finally:
            legend._PANEL_MODE = original_mode


# ─────────────────────────────────────────────────────────────────────────────
# render_player_legend_panel — mode "fixed"
# ─────────────────────────────────────────────────────────────────────────────


class TestRenderPlayerLegendPanelFixed:
    def test_fixed_mode_calls_st_markdown(self) -> None:
        """Mode 'fixed' doit appeler st.markdown avec unsafe_allow_html=True."""
        legend = _import_legend()
        original_mode = legend._PANEL_MODE
        try:
            legend._PANEL_MODE = "fixed"
            with (
                patch("src.ui.pages.teammates_legend.get_lang", return_value="fr"),
                patch.object(legend.st, "markdown") as mock_md,
            ):
                legend.render_player_legend_panel(_COLORS)
                mock_md.assert_called_once()
                _, kwargs = mock_md.call_args
                assert kwargs.get("unsafe_allow_html") is True
        finally:
            legend._PANEL_MODE = original_mode

    def test_fixed_html_contains_player_names(self) -> None:
        """Le HTML généré doit contenir les noms des joueurs."""
        legend = _import_legend()
        original_mode = legend._PANEL_MODE
        try:
            legend._PANEL_MODE = "fixed"
            captured_html: list[str] = []
            with (
                patch("src.ui.pages.teammates_legend.get_lang", return_value="fr"),
                patch.object(
                    legend.st, "markdown", side_effect=lambda html, **_: captured_html.append(html)
                ),
            ):
                legend.render_player_legend_panel(_COLORS)
            assert len(captured_html) == 1
            html = captured_html[0]
            assert "Alice" in html
            assert "Bob" in html
        finally:
            legend._PANEL_MODE = original_mode

    def test_fixed_html_contains_colors(self) -> None:
        """Le HTML généré doit contenir les couleurs des joueurs."""
        legend = _import_legend()
        original_mode = legend._PANEL_MODE
        try:
            legend._PANEL_MODE = "fixed"
            captured_html: list[str] = []
            with (
                patch("src.ui.pages.teammates_legend.get_lang", return_value="fr"),
                patch.object(
                    legend.st, "markdown", side_effect=lambda html, **_: captured_html.append(html)
                ),
            ):
                legend.render_player_legend_panel(_COLORS)
            html = captured_html[0]
            assert "#E69F00" in html
            assert "#56B4E9" in html
        finally:
            legend._PANEL_MODE = original_mode

    def test_fixed_html_contains_label_fr(self) -> None:
        """Le HTML doit contenir le label 'Joueurs' en français."""
        legend = _import_legend()
        original_mode = legend._PANEL_MODE
        try:
            legend._PANEL_MODE = "fixed"
            captured_html: list[str] = []
            with (
                patch("src.ui.pages.teammates_legend.get_lang", return_value="fr"),
                patch.object(
                    legend.st, "markdown", side_effect=lambda html, **_: captured_html.append(html)
                ),
            ):
                legend.render_player_legend_panel(_COLORS)
            assert "Joueurs" in captured_html[0]
        finally:
            legend._PANEL_MODE = original_mode

    def test_fixed_html_contains_label_en(self) -> None:
        """Le HTML doit contenir le label 'Players' en anglais."""
        legend = _import_legend()
        original_mode = legend._PANEL_MODE
        try:
            legend._PANEL_MODE = "fixed"
            captured_html: list[str] = []
            with (
                patch("src.ui.pages.teammates_legend.get_lang", return_value="en"),
                patch.object(
                    legend.st, "markdown", side_effect=lambda html, **_: captured_html.append(html)
                ),
            ):
                legend.render_player_legend_panel(_COLORS)
            assert "Players" in captured_html[0]
        finally:
            legend._PANEL_MODE = original_mode

    def test_fixed_html_contains_position_fixed(self) -> None:
        """Le CSS doit contenir position:fixed."""
        legend = _import_legend()
        original_mode = legend._PANEL_MODE
        try:
            legend._PANEL_MODE = "fixed"
            captured_html: list[str] = []
            with (
                patch("src.ui.pages.teammates_legend.get_lang", return_value="fr"),
                patch.object(
                    legend.st, "markdown", side_effect=lambda html, **_: captured_html.append(html)
                ),
            ):
                legend.render_player_legend_panel(_COLORS)
            assert "position:fixed" in captured_html[0]
        finally:
            legend._PANEL_MODE = original_mode


# ─────────────────────────────────────────────────────────────────────────────
# render_player_legend_panel — mode "sidebar"
# ─────────────────────────────────────────────────────────────────────────────


class TestRenderPlayerLegendPanelSidebar:
    def test_sidebar_mode_writes_to_sidebar(self) -> None:
        """Mode 'sidebar' doit appeler st.sidebar.markdown."""
        legend = _import_legend()
        original_mode = legend._PANEL_MODE
        try:
            legend._PANEL_MODE = "sidebar"
            mock_sidebar = MagicMock()
            with (
                patch("src.ui.pages.teammates_legend.get_lang", return_value="fr"),
                patch.object(legend.st, "sidebar", mock_sidebar),
            ):
                legend.render_player_legend_panel(_COLORS)
            assert mock_sidebar.markdown.called
        finally:
            legend._PANEL_MODE = original_mode

    def test_sidebar_mode_does_not_call_st_markdown(self) -> None:
        """Mode 'sidebar' ne doit pas appeler st.markdown (le corps principal)."""
        legend = _import_legend()
        original_mode = legend._PANEL_MODE
        try:
            legend._PANEL_MODE = "sidebar"
            mock_main_markdown = MagicMock()
            mock_sidebar = MagicMock()
            with (
                patch("src.ui.pages.teammates_legend.get_lang", return_value="fr"),
                patch.object(legend.st, "markdown", mock_main_markdown),
                patch.object(legend.st, "sidebar", mock_sidebar),
            ):
                legend.render_player_legend_panel(_COLORS)
            mock_main_markdown.assert_not_called()
        finally:
            legend._PANEL_MODE = original_mode
