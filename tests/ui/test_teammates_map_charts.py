"""Tests pour src/ui/pages/teammates_map_charts.py.

Couvre :
- render_map_charts_section (multi-coéquipiers)
- render_squad_heatmap (heatmap escouade)
"""

from __future__ import annotations

from datetime import datetime, timedelta
from unittest.mock import MagicMock, patch

import polars as pl

# =============================================================================
# Helpers
# =============================================================================


def _make_match_df(n: int = 6) -> pl.DataFrame:
    """DataFrame de matchs minimal pour les tests."""
    start = datetime(2025, 1, 1)
    return pl.DataFrame(
        {
            "match_id": [f"m{i}" for i in range(n)],
            "map_name": ["Recharge", "Streets"] * (n // 2),
            "start_time": [start + timedelta(hours=i) for i in range(n)],
            "outcome": [2, 3] * (n // 2),
            "kills": [10] * n,
            "deaths": [5] * n,
            "assists": [3] * n,
            "accuracy": [50.0] * n,
            "ratio": [1.5] * n,
        }
    )


def _make_breakdown(n: int = 2) -> pl.DataFrame:
    """Breakdown par carte simplifié."""
    maps = ["Recharge", "Streets"][:n]
    return pl.DataFrame(
        {
            "map_name": maps,
            "matches": [6, 4][:n],
            "win_rate": [0.6, 0.5][:n],
            "loss_rate": [0.4, 0.5][:n],
            "ratio_global": [1.2, 1.0][:n],
            "accuracy_avg": [48.0, 50.0][:n],
            "performance_avg": [0.2, 0.0][:n],
        }
    )


# =============================================================================
# render_map_charts_section
# =============================================================================


class TestRenderMapChartsSection:
    def test_affiche_info_si_breakdown_vide(self, mock_st) -> None:
        import src.ui.pages.teammates_map_charts as mod

        ms = mock_st(mod)

        mod.render_map_charts_section(
            sub_all=_make_match_df(),
            full_squad_df=_make_match_df(),
            breakdown_all=pl.DataFrame(),
            lang="fr",
        )
        ms.calls["info"].assert_called()

    def test_rend_les_charts_avec_donnees_valides(self, mock_st) -> None:
        import src.ui.pages.teammates_map_charts as mod

        ms = mock_st(mod)
        bd = _make_breakdown()

        mod.render_map_charts_section(
            sub_all=_make_match_df(),
            full_squad_df=_make_match_df(),
            breakdown_all=bd,
            lang="fr",
        )

        ms.calls["plotly_chart"].assert_called()

    def test_rend_bullet_si_full_squad_non_vide(self, mock_st) -> None:
        import src.ui.pages.teammates_map_charts as mod

        ms = mock_st(mod)
        bd = _make_breakdown()
        fig_mock = MagicMock()

        with patch.object(mod, "plot_map_winrate_bullet", return_value=fig_mock) as mock_bullet:
            mod.render_map_charts_section(
                sub_all=_make_match_df(),
                full_squad_df=_make_match_df(),
                breakdown_all=bd,
                lang="fr",
            )

        mock_bullet.assert_called_once()
        ms.calls["plotly_chart"].assert_called()

    def test_skip_bullet_si_full_squad_vide(self, mock_st) -> None:
        import src.ui.pages.teammates_map_charts as mod

        mock_st(mod)
        bd = _make_breakdown()

        with patch.object(mod, "plot_map_winrate_bullet", return_value=None) as mock_bullet:
            mod.render_map_charts_section(
                sub_all=_make_match_df(),
                full_squad_df=pl.DataFrame(),
                breakdown_all=bd,
                lang="fr",
            )

        mock_bullet.assert_not_called()


# =============================================================================
# render_squad_heatmap
# =============================================================================


class TestRenderSquadHeatmap:
    def test_ne_rend_rien_avec_un_seul_joueur(self, mock_st) -> None:
        import src.ui.pages.teammates_map_charts as mod

        ms = mock_st(mod)
        series = [("Alice", _make_match_df())]

        mod.render_squad_heatmap(series, lang="fr")

        ms.calls["plotly_chart"].assert_not_called()
        ms.calls["markdown"].assert_not_called()

    def test_rend_heatmap_avec_deux_joueurs(self, mock_st) -> None:
        import src.ui.pages.teammates_map_charts as mod

        ms = mock_st(mod)
        series = [("Alice", _make_match_df()), ("Bob", _make_match_df())]

        with patch.object(mod, "plot_squad_map_heatmap", return_value=MagicMock()) as mock_hm:
            mod.render_squad_heatmap(series, lang="fr")

        mock_hm.assert_called_once()
        ms.calls["markdown"].assert_called()

    def test_serie_vide_ne_rend_rien(self, mock_st) -> None:
        import src.ui.pages.teammates_map_charts as mod

        ms = mock_st(mod)
        mod.render_squad_heatmap([], lang="fr")
        ms.calls["plotly_chart"].assert_not_called()
