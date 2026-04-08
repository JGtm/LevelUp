"""Tests pour la page Séries temporelles (Sprint 7bis).

Couvre :
- render_timeseries_page avec MockStreamlit (flux de contrôle)
- Sous-sections individuelles (_render_kda_section, etc.)
- Edge cases : DataFrame vide, colonnes manquantes
"""

from __future__ import annotations

from datetime import datetime, timedelta
from unittest.mock import MagicMock, patch

import numpy as np
import polars as pl


def _make_timeseries_df(n: int = 30) -> pl.DataFrame:
    """Crée un DataFrame synthétique pour timeseries."""
    np.random.seed(42)
    start = datetime(2025, 1, 1)

    return pl.DataFrame(
        {
            "match_id": [f"match_{i}" for i in range(n)],
            "start_time": [start + timedelta(hours=i) for i in range(n)],
            "kills": np.random.randint(5, 25, n).tolist(),
            "deaths": np.random.randint(3, 15, n).tolist(),
            "assists": np.random.randint(2, 12, n).tolist(),
            "accuracy": np.random.uniform(30, 60, n).tolist(),
            "ratio": np.random.uniform(0.5, 2.5, n).tolist(),
            "kda": np.random.uniform(-5, 10, n).tolist(),
            "outcome": np.random.choice([1, 2, 3, 4], n).tolist(),
            "map_name": np.random.choice(["Recharge", "Streets"], n).tolist(),
            "time_played_seconds": np.random.randint(300, 900, n).tolist(),
            "kills_per_min": np.random.uniform(0.3, 1.5, n).tolist(),
            "deaths_per_min": np.random.uniform(0.2, 1.0, n).tolist(),
            "assists_per_min": np.random.uniform(0.1, 0.8, n).tolist(),
            "headshot_kills": np.random.randint(1, 10, n).tolist(),
            "max_killing_spree": np.random.randint(0, 8, n).tolist(),
            "average_life_seconds": np.random.uniform(20, 60, n).tolist(),
            "personal_score": np.random.randint(800, 3000, n).tolist(),
            "performance_score": np.random.uniform(20, 90, n).tolist(),
            "rank": np.random.randint(1, 9, n).tolist(),
        }
    )


class TestRenderTimeseriesPage:
    """Tests du point d'entrée render_timeseries_page."""

    def test_empty_df_shows_warning(self, mock_st) -> None:
        from src.ui.pages import timeseries as mod

        ms = mock_st(mod)
        empty_df = pl.DataFrame(schema={"match_id": pl.Utf8, "kda": pl.Float64})
        mod.render_timeseries_page(empty_df)
        ms.calls["warning"].assert_called_once()

    def test_normal_flow(self, mock_st) -> None:
        """Parcours nominal : les sections KDA, distributions, corrélations sont rendues."""
        from src.ui.pages import timeseries as mod

        ms = mock_st(mod)
        ms.set_columns_dynamic()
        dff = _make_timeseries_df(30)

        # Mock les fonctions de plot pour simplifier
        with (
            patch.object(mod, "plot_timeseries", return_value=MagicMock()),
            patch.object(mod, "plot_kda_distribution", return_value=MagicMock()),
            patch.object(mod, "render_distributions", return_value=None),
            patch.object(mod, "render_correlations", return_value=None),
            patch.object(mod, "plot_first_event_distribution", return_value=MagicMock()),
            patch.object(mod, "plot_performance_timeseries", return_value=MagicMock()),
            patch.object(mod, "plot_assists_timeseries", return_value=MagicMock()),
            patch.object(mod, "plot_per_minute_timeseries", return_value=MagicMock()),
            patch.object(mod, "plot_average_life", return_value=MagicMock()),
            patch.object(mod, "plot_spree_headshots_accuracy", return_value=MagicMock()),
            patch.object(mod, "plot_rank_score", return_value=MagicMock()),
            patch.object(mod, "TimeseriesService") as mock_svc,
        ):
            # Mock enrich_performance_score
            mock_svc.enrich_performance_score.return_value = dff

            # Mock compute_cumulative_metrics
            cumul = MagicMock()
            cumul.cumul_net = [0.0]
            cumul.cumul_kd = [1.0]
            cumul.rolling_kd = [1.0]
            cumul.time_played_seconds = [300]
            cumul.has_enough_for_trend = False
            cumul.pl_df = dff
            mock_svc.compute_cumulative_metrics.return_value = cumul

            # Mock compute_score_per_minute
            spm = MagicMock()
            spm.has_data = False
            mock_svc.compute_score_per_minute.return_value = spm

            # Mock compute_rolling_win_rate
            wr = MagicMock()
            wr.has_data = False
            wr.missing_column = False
            wr.not_enough_matches = True
            mock_svc.compute_rolling_win_rate.return_value = wr

            # Mock load_first_event_times
            first_ev = MagicMock()
            first_ev.available = False
            mock_svc.load_first_event_times.return_value = first_ev

            # Mock load_perfect_kills
            pk = MagicMock()
            pk.counts = {}
            mock_svc.load_perfect_kills.return_value = pk

            mod.render_timeseries_page(dff)

        # Vérifie qu'au moins des plotly_chart ont été faits
        assert ms.calls["plotly_chart"].call_count >= 4


class TestRenderKdaSection:
    """Tests pour _render_kda_section."""

    def test_basic(self, mock_st) -> None:
        from src.ui.pages import timeseries as mod

        ms = mock_st(mod)
        ms.set_columns_dynamic()
        dff = _make_timeseries_df(10)

        with (
            patch.object(mod, "plot_timeseries", return_value=MagicMock()),
            patch.object(mod, "plot_kda_distribution", return_value=MagicMock()),
        ):
            mod._render_kda_section(dff)

        assert ms.calls["plotly_chart"].call_count >= 1
        ms.calls["subheader"].assert_called()

    def test_empty_kda(self, mock_st) -> None:
        from src.ui.pages import timeseries as mod

        ms = mock_st(mod)
        ms.set_columns_dynamic()
        # DataFrame sans colonne kda
        dff = pl.DataFrame({"match_id": ["m1"], "kills": [5]})

        with patch.object(mod, "plot_timeseries", return_value=MagicMock()):
            mod._render_kda_section(dff)

        ms.calls["info"].assert_called()


class TestRenderDistributions:
    """Tests pour render_distributions."""

    def test_renders_histograms(self, mock_st) -> None:
        from src.ui.pages import _timeseries_distributions as mod

        ms = mock_st(mod)
        ms.set_columns_dynamic()
        dff = _make_timeseries_df(20)

        with (
            patch.object(mod, "plot_histogram", return_value=MagicMock()),
            patch.object(mod, "TimeseriesService") as mock_svc,
        ):
            spm = MagicMock()
            spm.has_data = False
            mock_svc.compute_score_per_minute.return_value = spm
            wr = MagicMock()
            wr.has_data = False
            wr.missing_column = False
            wr.not_enough_matches = True
            mock_svc.compute_rolling_win_rate.return_value = wr

            mod.render_distributions(dff)

        ms.calls["subheader"].assert_called()


class TestRenderSingleHistogram:
    """Tests pour _render_single_histogram."""

    def test_missing_column(self, mock_st) -> None:
        from src.ui.pages import _timeseries_distributions as mod

        ms = mock_st(mod)
        dff = pl.DataFrame({"match_id": ["m1"]})
        mod._render_single_histogram(dff, "nonexistent", "Title", "X", "#000")
        ms.calls["info"].assert_called()

    def test_not_enough_data(self, mock_st) -> None:
        from src.ui.pages import _timeseries_distributions as mod

        ms = mock_st(mod)
        dff = pl.DataFrame({"kills": [5, 6]})
        mod._render_single_histogram(dff, "kills", "Title", "X", "#000", min_data=10)
        ms.calls["info"].assert_called()

    def test_enough_data(self, mock_st) -> None:
        from src.ui.pages import _timeseries_distributions as mod

        ms = mock_st(mod)
        dff = pl.DataFrame({"kills": list(range(20))})

        with patch.object(mod, "plot_histogram", return_value=MagicMock()):
            mod._render_single_histogram(dff, "kills", "Title", "X", "#000")

        ms.calls["plotly_chart"].assert_called()


class TestRenderScatter:
    """Tests pour _render_scatter."""

    def test_missing_columns(self, mock_st) -> None:
        from src.ui.pages import _timeseries_distributions as mod

        ms = mock_st(mod)
        dff = pl.DataFrame({"match_id": ["m1"]})
        mod._render_scatter(dff, "kills", "deaths", "outcome", "T", "X", "Y")
        ms.calls["info"].assert_called()

    def test_not_enough_data(self, mock_st) -> None:
        from src.ui.pages import _timeseries_distributions as mod

        ms = mock_st(mod)
        dff = pl.DataFrame({"kills": [5], "deaths": [3], "outcome": [2]})
        mod._render_scatter(dff, "kills", "deaths", "outcome", "T", "X", "Y")
        ms.calls["info"].assert_called()

    def test_enough_data(self, mock_st) -> None:
        from src.ui.pages import _timeseries_distributions as mod

        ms = mock_st(mod)
        np.random.seed(42)
        dff = pl.DataFrame(
            {
                "kills": np.random.randint(5, 25, 20).tolist(),
                "deaths": np.random.randint(3, 15, 20).tolist(),
                "outcome": np.random.choice([1, 2, 3], 20).tolist(),
            }
        )

        with patch.object(mod, "plot_correlation_scatter", return_value=MagicMock()):
            mod._render_scatter(dff, "kills", "deaths", "outcome", "T", "X", "Y")

        ms.calls["plotly_chart"].assert_called()


# =============================================================================
# _render_cumulative_performance
# =============================================================================


class TestRenderCumulativePerformance:
    """Tests pour _render_cumulative_performance (Progression tab)."""

    def _mock_cumul(self, *, has_trend: bool = False) -> MagicMock:
        c = MagicMock()
        c.has_enough_for_trend = has_trend
        return c

    def test_none_cumul_shows_info(self, mock_st) -> None:
        """Si compute_cumulative_metrics retourne None → st.info affiché."""
        from src.ui.pages import timeseries as mod

        ms = mock_st(mod)
        dff = _make_timeseries_df(5)

        with patch.object(mod, "TimeseriesService") as mock_svc:
            mock_svc.compute_cumulative_metrics.return_value = None
            mod._render_cumulative_performance(dff, lang="fr")

        ms.calls["info"].assert_called_once()

    def test_basic_flow_renders_charts(self, mock_st) -> None:
        """Flux nominal : 3 plotly_chart au minimum."""
        from src.ui.pages import timeseries as mod

        ms = mock_st(mod)
        ms.set_columns_dynamic()
        dff = _make_timeseries_df(30)

        with (
            patch.object(mod, "TimeseriesService") as mock_svc,
            patch.object(mod, "plot_net_score_per_hour", return_value=MagicMock()),
            patch.object(mod, "plot_cumulative_kd_with_ci", return_value=MagicMock()),
            patch.object(mod, "plot_ewma_kd", return_value=MagicMock()),
            patch.object(mod, "plot_regression_trend", return_value=MagicMock()),
            patch.object(mod, "safe_chart_render"),
        ):
            mock_svc.compute_cumulative_metrics.return_value = self._mock_cumul(has_trend=False)
            mock_svc.compute_rolling_net_score_per_hour.return_value = dff
            mock_svc.compute_cumulative_kd_with_ci.return_value = dff
            mock_svc.compute_ewma_kd.return_value = dff
            mock_svc.compute_linear_regression_kd.return_value = {
                "slope": 0.0,
                "r_squared": 0.0,
                "is_significant": False,
                "trend": "stable",
                "y_hat": [],
                "x_labels": [],
            }

            mod._render_cumulative_performance(dff, lang="fr")

        assert ms.calls["plotly_chart"].call_count >= 2

    def test_nph_none_shows_info(self, mock_st) -> None:
        """Quand nph_data est None → st.info 'ts_nph_unavailable'."""
        from src.ui.pages import timeseries as mod

        ms = mock_st(mod)
        ms.set_columns_dynamic()
        dff = _make_timeseries_df(20)

        with (
            patch.object(mod, "TimeseriesService") as mock_svc,
            patch.object(mod, "plot_cumulative_kd_with_ci", return_value=MagicMock()),
            patch.object(mod, "plot_ewma_kd", return_value=MagicMock()),
            patch.object(mod, "safe_chart_render"),
        ):
            mock_svc.compute_cumulative_metrics.return_value = self._mock_cumul(has_trend=False)
            mock_svc.compute_rolling_net_score_per_hour.return_value = None  # ← absent
            mock_svc.compute_cumulative_kd_with_ci.return_value = dff
            mock_svc.compute_ewma_kd.return_value = dff
            mock_svc.compute_linear_regression_kd.return_value = {
                "slope": 0.0,
                "r_squared": 0.0,
                "is_significant": False,
                "trend": "stable",
                "y_hat": [],
                "x_labels": [],
            }

            mod._render_cumulative_performance(dff, lang="fr")

        # st.info doit être appelé (NPH unavailable)
        ms.calls["info"].assert_called()

    def test_trend_shown_when_enough_matches(self, mock_st) -> None:
        """Quand has_enough_for_trend=True → plot_regression_trend appelé."""
        from src.ui.pages import timeseries as mod

        ms = mock_st(mod)
        ms.set_columns_dynamic()
        dff = _make_timeseries_df(30)

        with (
            patch.object(mod, "TimeseriesService") as mock_svc,
            patch.object(mod, "plot_net_score_per_hour", return_value=MagicMock()),
            patch.object(mod, "plot_cumulative_kd_with_ci", return_value=MagicMock()),
            patch.object(mod, "plot_ewma_kd", return_value=MagicMock()),
            patch.object(mod, "plot_regression_trend", return_value=MagicMock()) as mock_reg,
            patch.object(mod, "safe_chart_render"),
        ):
            mock_svc.compute_cumulative_metrics.return_value = self._mock_cumul(has_trend=True)
            mock_svc.compute_rolling_net_score_per_hour.return_value = dff
            mock_svc.compute_cumulative_kd_with_ci.return_value = dff
            mock_svc.compute_ewma_kd.return_value = dff
            mock_svc.compute_linear_regression_kd.return_value = {
                "slope": 0.1,
                "r_squared": 0.6,
                "is_significant": True,
                "trend": "improving",
                "y_hat": [1.0] * 30,
                "x_labels": [f"m{i}" for i in range(30)],
            }

            mod._render_cumulative_performance(dff, lang="fr")

        mock_reg.assert_called_once()

    def test_trend_info_when_not_enough(self, mock_st) -> None:
        """Quand has_enough_for_trend=False → st.info 'ts_trend_min_matches'."""
        from src.ui.pages import timeseries as mod

        ms = mock_st(mod)
        ms.set_columns_dynamic()
        dff = _make_timeseries_df(5)

        with (
            patch.object(mod, "TimeseriesService") as mock_svc,
            patch.object(mod, "plot_cumulative_kd_with_ci", return_value=MagicMock()),
            patch.object(mod, "plot_ewma_kd", return_value=MagicMock()),
            patch.object(mod, "safe_chart_render"),
        ):
            mock_svc.compute_cumulative_metrics.return_value = self._mock_cumul(has_trend=False)
            mock_svc.compute_rolling_net_score_per_hour.return_value = None
            mock_svc.compute_cumulative_kd_with_ci.return_value = dff
            mock_svc.compute_ewma_kd.return_value = dff
            mock_svc.compute_linear_regression_kd.return_value = {
                "slope": 0.0,
                "r_squared": 0.0,
                "is_significant": False,
                "trend": "stable",
                "y_hat": [],
                "x_labels": [],
            }

            mod._render_cumulative_performance(dff, lang="fr")

        # st.info doit être appelé (ts_trend_min_matches + ts_nph_unavailable)
        assert ms.calls["info"].call_count >= 1


# =============================================================================
# Tests de la fusion Victoires/Défaites → Séries (v6.3)
# =============================================================================


class TestMergedPageSignature:
    """Vérifie que render_timeseries_page accepte les nouveaux paramètres de win_loss."""

    def test_accepts_base_and_session_params(self, mock_st) -> None:
        """render_timeseries_page ne lève pas si base/picked_session_labels/db_key sont passés."""
        from src.ui.pages import timeseries as mod

        ms = mock_st(mod)
        ms.set_columns_dynamic()
        dff = _make_timeseries_df(10)
        base_df = _make_timeseries_df(20)

        with patch.object(mod, "TimeseriesService") as mock_svc:
            mock_svc.enrich_performance_score.return_value = dff
            cumul = MagicMock()
            cumul.has_enough_for_trend = False
            mock_svc.compute_cumulative_metrics.return_value = cumul
            mock_svc.compute_rolling_net_score_per_hour.return_value = None
            mock_svc.compute_cumulative_kd_with_ci.return_value = None
            mock_svc.compute_ewma_kd.return_value = dff
            mock_svc.compute_linear_regression_kd.return_value = {
                "slope": 0.0,
                "r_squared": 0.0,
                "is_significant": False,
                "trend": "stable",
                "y_hat": [],
                "x_labels": [],
            }
            first_ev = MagicMock()
            first_ev.available = False
            mock_svc.load_first_event_times.return_value = first_ev
            pk = MagicMock()
            pk.counts = {}
            mock_svc.load_perfect_kills.return_value = pk

            # Ne doit pas lever d'exception
            mod.render_timeseries_page(
                dff,
                df_full=base_df,
                base=base_df,
                picked_session_labels=["Session 1"],
                db_path=None,
                xuid=None,
                db_key=None,
            )

    def test_base_defaults_to_dff_when_none(self, mock_st) -> None:
        """Quand base=None, base_df est initialisé depuis dff (pas d'erreur)."""
        from src.ui.pages import timeseries as mod

        mock_st(mod)
        dff = pl.DataFrame(schema={"match_id": pl.Utf8})

        # DataFrame vide → warning immédiat, pas de crash sur base=None
        mod.render_timeseries_page(dff, base=None)
        # Vérifie que la page n'explose pas si base n'est pas fourni


class TestWinLossIntegration:
    """Vérifie que les fonctions win_loss s'importent bien et sont accessibles depuis timeseries."""

    def test_win_loss_functions_importable_from_timeseries(self) -> None:
        """Les helpers win_loss réexportés dans timeseries.py sont bien présents."""
        from src.ui.pages import timeseries as mod

        for name in [
            "_render_outcomes_over_time",
            "_render_streak_section",
            "_render_map_mode_breakdown",
            "_render_winrate_perf_vs_history",
            "_render_wl_heatmap_section",
            "_render_top_by_week",
            "_render_personal_score_section",
        ]:
            assert hasattr(mod, name), f"{name} manquant dans timeseries.py"
            assert callable(getattr(mod, name))

    def test_win_loss_module_still_importable(self) -> None:
        """win_loss.py reste importable (utilisé comme bibliothèque)."""
        import importlib

        mod = importlib.import_module("src.ui.pages.win_loss")
        assert mod is not None
        assert callable(mod.render_win_loss_page)


class TestPageRouterNoWinLoss:
    """Vérifie que win_loss a été retiré de la navigation."""

    def test_win_loss_not_in_page_keys(self) -> None:
        from src.app.page_router import PAGE_KEYS

        assert "win_loss" not in PAGE_KEYS, "win_loss ne doit plus figurer dans PAGE_KEYS"

    def test_legacy_redirect_points_to_timeseries(self) -> None:
        """L'ancienne URL Victoires/Défaites redirige vers timeseries."""
        from src.app.page_router import _LEGACY_NAME_TO_SLUG

        assert _LEGACY_NAME_TO_SLUG.get("Victoires/Défaites") == "timeseries"
