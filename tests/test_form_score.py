"""Tests pour le score de forme (compute_form_score_history, plot_form_score_history,
helpers UI, étiquettes clés et requête DB).
"""

from __future__ import annotations

from datetime import datetime, timezone
from unittest.mock import patch

import polars as pl
import pytest

from src.analysis._performance_form import _WINDOW_LONG, _WINDOW_SHORT, compute_form_score_history

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_history(n: int, perf_start: float = 50.0, perf_step: float = 0.0) -> pl.DataFrame:
    """Crée un DataFrame d'historique avec n matchs consécutifs."""
    return pl.DataFrame(
        {
            "match_id": [str(i) for i in range(n)],
            "start_time": [
                datetime(2024, 1, 1, tzinfo=timezone.utc).replace(day=1 + i % 28, month=1 + i // 28)
                for i in range(n)
            ],
            "performance_score": [perf_start + i * perf_step for i in range(n)],
        }
    )


# ---------------------------------------------------------------------------
# compute_form_score_history
# ---------------------------------------------------------------------------


class TestComputeFormScoreHistory:
    def test_empty_df_returns_unchanged(self):
        df = pl.DataFrame(
            {"match_id": [], "start_time": [], "performance_score": []},
            schema={
                "match_id": pl.Utf8,
                "start_time": pl.Datetime,
                "performance_score": pl.Float64,
            },
        )
        result = compute_form_score_history(df)
        assert result.is_empty()
        assert "form_score" not in result.columns

    def test_missing_performance_score_returns_unchanged(self):
        df = pl.DataFrame(
            {
                "match_id": ["a"],
                "start_time": [datetime(2024, 1, 1, tzinfo=timezone.utc)],
            }
        )
        result = compute_form_score_history(df)
        assert "form_score" not in result.columns
        assert list(result.columns) == ["match_id", "start_time"]

    def test_missing_start_time_returns_unchanged(self):
        df = pl.DataFrame({"match_id": ["a"], "performance_score": [50.0]})
        result = compute_form_score_history(df)
        assert "form_score" not in result.columns

    def test_output_columns_present(self):
        df = _make_history(5, perf_start=50.0)
        result = compute_form_score_history(df)
        assert "avg_14" in result.columns
        assert "avg_90" in result.columns
        assert "form_score" in result.columns

    def test_sorted_by_start_time(self):
        """Le résultat doit être trié par start_time, même si l'entrée ne l'est pas."""
        df = _make_history(5).sort("start_time", descending=True)
        result = compute_form_score_history(df)
        times = result["start_time"].to_list()
        assert times == sorted(times)

    def test_single_match_form_score_zero(self):
        """Avec 1 seul match, avg_14 == avg_90 == perf → form_score == 0."""
        df = _make_history(1, perf_start=60.0)
        result = compute_form_score_history(df)
        assert result["form_score"][0] == pytest.approx(0.0)

    def test_constant_perf_gives_zero_form(self):
        """Performance constante → avg_14 == avg_90 → form_score == 0 partout."""
        df = _make_history(30, perf_start=50.0, perf_step=0.0)
        result = compute_form_score_history(df)
        assert result["form_score"].abs().max() == pytest.approx(0.0, abs=1e-6)

    def test_improving_perf_gives_positive_form_late(self):
        """Perf croissante : récente (avg_14) > long terme (avg_90) → form > 0 à la fin."""
        df = _make_history(_WINDOW_LONG + 10, perf_start=0.0, perf_step=1.0)
        result = compute_form_score_history(df)
        last_form = result["form_score"][-1]
        assert last_form > 0, f"Attendu form > 0 pour perf croissante, obtenu {last_form}"

    def test_declining_perf_gives_negative_form_late(self):
        """Perf décroissante → avg_14 < avg_90 → form < 0 à la fin."""
        df = _make_history(_WINDOW_LONG + 10, perf_start=100.0, perf_step=-1.0)
        result = compute_form_score_history(df)
        last_form = result["form_score"][-1]
        assert last_form < 0, f"Attendu form < 0 pour perf décroissante, obtenu {last_form}"

    def test_avg_14_window_boundary(self):
        """avg_14 au match n°1 (index 0) vaut juste perf[0] (min_samples=1)."""
        df = _make_history(20, perf_start=42.0)
        result = compute_form_score_history(df)
        assert result["avg_14"][0] == pytest.approx(42.0)

    def test_avg_14_uses_short_window(self):
        """Après _WINDOW_SHORT matchs, avg_14 est la moyenne des _WINDOW_SHORT derniers."""
        perf_vals = list(range(_WINDOW_SHORT))
        df = pl.DataFrame(
            {
                "match_id": [str(i) for i in range(_WINDOW_SHORT)],
                "start_time": [
                    datetime(2024, 1, i + 1, tzinfo=timezone.utc) for i in range(_WINDOW_SHORT)
                ],
                "performance_score": [float(v) for v in perf_vals],
            }
        )
        result = compute_form_score_history(df)
        expected_avg_14 = sum(perf_vals) / _WINDOW_SHORT
        assert result["avg_14"][-1] == pytest.approx(expected_avg_14)

    def test_form_score_equals_avg14_minus_avg90(self):
        """form_score == avg_14 - avg_90 exactement."""
        df = _make_history(50, perf_start=30.0, perf_step=0.5)
        result = compute_form_score_history(df)
        diff = (result["avg_14"] - result["avg_90"] - result["form_score"]).abs().max()
        assert diff == pytest.approx(0.0, abs=1e-9)


# ---------------------------------------------------------------------------
# _key_label_indices
# ---------------------------------------------------------------------------


class TestKeyLabelIndices:
    def test_empty_list_returns_empty_set(self):
        from src.visualization._form_score import _key_label_indices

        assert _key_label_indices([]) == set()

    def test_all_none_returns_empty_set(self):
        from src.visualization._form_score import _key_label_indices

        assert _key_label_indices([None, None, None]) == set()

    def test_single_value_returns_index_zero(self):
        from src.visualization._form_score import _key_label_indices

        result = _key_label_indices([5.0])
        assert result == {0}

    def test_two_values_returns_both_indices(self):
        from src.visualization._form_score import _key_label_indices

        result = _key_label_indices([1.0, 2.0])
        # first=0, last=1, min=0, max=1 → {0, 1}
        assert result == {0, 1}

    def test_marks_first_last_min_max(self):
        from src.visualization._form_score import _key_label_indices

        # [10, 1, 5, 8, 15] → first=0, last=4, min=1(idx1), max=15(idx4)
        result = _key_label_indices([10.0, 1.0, 5.0, 8.0, 15.0])
        assert 0 in result  # first
        assert 4 in result  # last + max
        assert 1 in result  # min

    def test_labels_only_on_key_indices(self):
        """Seuls les indices clés ont une étiquette non vide dans le graphe."""
        from src.visualization._form_score import _key_label_indices, plot_form_score_history

        df = compute_form_score_history(_make_history(10, perf_start=40.0, perf_step=2.0))
        fig = plot_form_score_history({"": df}, highlight_match_ids=set(), lang="fr")
        assert fig is not None
        # Le trace principal est lines+text
        line_trace = next(t for t in fig.data if getattr(t, "mode", "") == "lines+text")
        labels = list(line_trace.text)
        non_empty = [lbl for lbl in labels if lbl]
        key_count = len(_key_label_indices(list(df["form_score"].to_list())))
        assert len(non_empty) == key_count


# ---------------------------------------------------------------------------
# render_form_score_section — filtrage df_full→dff
# ---------------------------------------------------------------------------


class TestRenderFormScoreSectionFiltering:
    """Tests de la logique de filtrage sans Streamlit (fonctions pures)."""

    def test_display_filtered_to_dff_ids(self):
        """Le graphe n'affiche que les matchs présents dans dff, pas tout df_full."""
        full = compute_form_score_history(_make_history(90, perf_start=40.0, perf_step=0.5))
        # dff = seulement les 10 derniers matchs
        dff_ids = set(full["match_id"].to_list()[-10:])
        display = full.filter(pl.col("match_id").is_in(dff_ids))
        assert len(display) == 10
        # form_score du display calculé sur l'historique complet → valeurs non-nulles et non-zéro
        assert display["form_score"].drop_nulls().len() == 10

    def test_form_score_on_full_history_is_non_trivial(self):
        """Calculé sur 90 matchs, le form_score final n'est pas zéro (perf croissante)."""
        full = compute_form_score_history(_make_history(90, perf_start=30.0, perf_step=1.0))
        assert abs(full["form_score"][-1]) > 0.1

    def test_form_score_on_short_series_is_near_zero(self):
        """Sur 5 matchs seulement, avg_14 ≈ avg_90 → form ≈ 0 (problème original)."""
        short = compute_form_score_history(_make_history(5, perf_start=60.0, perf_step=1.0))
        # Avec seulement 5 matchs les deux fenêtres convergent → form proche de 0
        assert abs(short["form_score"][-1]) < 1.0


# ---------------------------------------------------------------------------
# render_squad_form_score_section — fallback series
# ---------------------------------------------------------------------------


class TestSquadFormScoreFallback:
    def test_fallback_uses_series_data_when_db_empty(self):
        """Si load_full_performance_history retourne vide, on utilise les données series."""
        from unittest.mock import patch

        import polars as pl

        # Préparer une série avec performance_score + start_time
        df_series = _make_history(20, perf_start=50.0, perf_step=0.5)
        series = [("TestPlayer", df_series)]
        sub_all = df_series  # tous les matchs = sélection courante

        with (
            patch(
                "src.ui.pages.teammates_map_charts.load_full_performance_history",
                return_value=pl.DataFrame(),
            ),
            patch(
                "src.ui.pages.teammates_map_charts.plot_form_score_history",
            ) as mock_plot,
            patch("src.ui.pages.teammates_map_charts.st"),
        ):
            mock_plot.return_value = None
            from src.ui.pages.teammates_map_charts import render_squad_form_score_section

            render_squad_form_score_section(series, "/fake/db_path/stats.duckdb", sub_all, None)

        # plot_form_score_history doit avoir été appelé (données fallback trouvées)
        mock_plot.assert_called_once()
        called_series = mock_plot.call_args[0][0]
        assert "TestPlayer" in called_series
        assert "form_score" in called_series["TestPlayer"].columns

    def test_empty_series_returns_early(self):
        """series=[] → retour immédiat sans appel DB."""
        from unittest.mock import patch

        with patch("src.ui.pages.teammates_map_charts.load_full_performance_history") as mock_load:
            from src.ui.pages.teammates_map_charts import render_squad_form_score_section

            render_squad_form_score_section([], "/fake/db_path/stats.duckdb", pl.DataFrame(), None)
        mock_load.assert_not_called()

    def test_display_filtered_to_sub_all(self):
        """Seuls les matchs de sub_all apparaissent dans l'appel à plot_form_score_history."""
        from unittest.mock import patch

        full_hist = _make_history(80, perf_start=40.0, perf_step=0.5)
        # sub_all = seulement les 10 derniers matchs
        sub_ids = set(full_hist["match_id"].to_list()[-10:])
        sub_all = full_hist.filter(pl.col("match_id").is_in(sub_ids))
        series = [("Player1", full_hist)]

        with (
            patch(
                "src.ui.pages.teammates_map_charts.load_full_performance_history",
                return_value=full_hist.select(["match_id", "start_time", "performance_score"]),
            ),
            patch(
                "src.ui.pages.teammates_map_charts.plot_form_score_history",
            ) as mock_plot,
            patch("src.ui.pages.teammates_map_charts.st"),
        ):
            mock_plot.return_value = None
            from src.ui.pages.teammates_map_charts import render_squad_form_score_section

            render_squad_form_score_section(series, "/fake/db_path/stats.duckdb", sub_all, None)

        mock_plot.assert_called_once()
        called_series = mock_plot.call_args[0][0]
        assert len(called_series["Player1"]) == 10


# ---------------------------------------------------------------------------
# load_full_performance_history
# ---------------------------------------------------------------------------


class TestLoadFullPerformanceHistory:
    def test_nonexistent_path_returns_empty(self):
        from src.data.services._form_score_queries import load_full_performance_history

        result = load_full_performance_history("/nonexistent/path/stats.duckdb")
        assert isinstance(result, pl.DataFrame)
        assert result.is_empty()

    def test_missing_shared_db_returns_empty(self, tmp_path):
        """Si la DB partagée est absente, retour vide sans exception."""
        from src.data.services._form_score_queries import load_full_performance_history

        player_db = tmp_path / "stats.duckdb"
        player_db.touch()

        with patch(
            "src.utils.paths.get_shared_matches_path_from_player",
            return_value=None,
        ):
            result = load_full_performance_history(str(player_db))
        assert result.is_empty()


# ---------------------------------------------------------------------------
# plot_form_score_history
# ---------------------------------------------------------------------------


class TestPlotFormScoreHistory:
    def _series_with_form(self, n: int = 20, name: str = "") -> dict[str, pl.DataFrame]:
        df = _make_history(n, perf_start=40.0, perf_step=1.0)
        return {name: compute_form_score_history(df)}

    def test_empty_series_returns_none(self):
        from src.visualization._form_score import plot_form_score_history

        result = plot_form_score_history({}, highlight_match_ids=set(), lang="fr")
        assert result is None

    def test_all_empty_dfs_returns_none(self):
        from src.visualization._form_score import plot_form_score_history

        empty = pl.DataFrame(
            schema={"match_id": pl.Utf8, "start_time": pl.Datetime, "form_score": pl.Float64}
        )
        result = plot_form_score_history({"": empty}, highlight_match_ids=set(), lang="fr")
        assert result is None

    def test_single_player_returns_figure(self):
        import plotly.graph_objects as go

        from src.visualization._form_score import plot_form_score_history

        series = self._series_with_form(30)
        fig = plot_form_score_history(series, highlight_match_ids=set(), lang="fr")
        assert isinstance(fig, go.Figure)

    def test_single_player_no_legend(self):
        """Joueur unique → pas de légende."""
        from src.visualization._form_score import plot_form_score_history

        series = self._series_with_form(30)
        fig = plot_form_score_history(series, highlight_match_ids=set(), lang="fr")
        assert fig is not None
        assert fig.layout.showlegend is False

    def test_multi_player_shows_legend(self):
        """Multi-joueurs → légende visible en bas."""
        from src.visualization._form_score import plot_form_score_history

        s1 = compute_form_score_history(_make_history(20, perf_start=40.0))
        s2 = compute_form_score_history(_make_history(20, perf_start=60.0))
        fig = plot_form_score_history(
            {"Alice": s1, "Bob": s2}, highlight_match_ids=set(), lang="fr"
        )
        assert fig is not None
        assert fig.layout.showlegend is True
        assert fig.layout.legend.yanchor == "top"
        assert fig.layout.legend.y < 0

    def test_highlight_ids_add_scatter_trace(self):
        """Des match_ids en highlight doivent produire un trace supplémentaire (points)."""
        from src.visualization._form_score import plot_form_score_history

        df = compute_form_score_history(_make_history(20, perf_start=50.0))
        highlight_ids = set(df["match_id"].to_list()[:5])
        fig = plot_form_score_history({"": df}, highlight_match_ids=highlight_ids, lang="fr")
        assert fig is not None
        # Au moins un trace de type scatter pour les points highlights
        scatter_traces = [t for t in fig.data if t.mode == "markers"]
        assert len(scatter_traces) >= 1

    def test_zero_hline_present(self):
        """Une ligne horizontale à y=0 doit être présente."""
        from src.visualization._form_score import plot_form_score_history

        series = self._series_with_form(20)
        fig = plot_form_score_history(series, highlight_match_ids=set(), lang="fr")
        assert fig is not None
        hlines = [s for s in fig.layout.shapes if s.y0 == 0 and s.y1 == 0]
        assert len(hlines) >= 1

    def test_custom_height_applied(self):
        from src.visualization._form_score import plot_form_score_history

        series = self._series_with_form(20)
        fig = plot_form_score_history(series, highlight_match_ids=set(), lang="fr", height=400)
        assert fig is not None
        assert fig.layout.height == 400


# ---------------------------------------------------------------------------
# _current_form_value / _baseline_form_value
# ---------------------------------------------------------------------------


class TestFormValueHelpers:
    def _history_with_form(self, n: int = 30) -> pl.DataFrame:
        return compute_form_score_history(_make_history(n, perf_start=50.0, perf_step=0.5))

    # _current_form_value
    def test_current_empty_df_returns_none(self):
        from src.ui.pages._timeseries_form import _current_form_value

        result = _current_form_value(pl.DataFrame(), highlight_ids={"x"})
        assert result is None

    def test_current_no_highlight_ids_returns_none(self):
        from src.ui.pages._timeseries_form import _current_form_value

        df = self._history_with_form()
        result = _current_form_value(df, highlight_ids=set())
        assert result is None

    def test_current_missing_form_score_column_returns_none(self):
        from src.ui.pages._timeseries_form import _current_form_value

        df = _make_history(10)
        result = _current_form_value(df, highlight_ids={"0", "1"})
        assert result is None

    def test_current_highlight_not_in_df_returns_none(self):
        from src.ui.pages._timeseries_form import _current_form_value

        df = self._history_with_form()
        result = _current_form_value(df, highlight_ids={"nonexistent_id"})
        assert result is None

    def test_current_valid_returns_float(self):
        from src.ui.pages._timeseries_form import _current_form_value

        df = self._history_with_form(30)
        ids = set(df["match_id"].to_list()[:5])
        result = _current_form_value(df, highlight_ids=ids)
        assert isinstance(result, float)

    def test_current_is_mean_of_selected(self):
        from src.ui.pages._timeseries_form import _current_form_value

        df = self._history_with_form(30)
        ids = set(df["match_id"].to_list()[:5])
        expected = float(df.filter(pl.col("match_id").is_in(ids))["form_score"].drop_nulls().mean())
        result = _current_form_value(df, highlight_ids=ids)
        assert result == pytest.approx(expected)

    # _baseline_form_value
    def test_baseline_empty_df_returns_none(self):
        from src.ui.pages._timeseries_form import _baseline_form_value

        result = _baseline_form_value(pl.DataFrame(), highlight_ids={"x"})
        assert result is None

    def test_baseline_no_highlight_ids_returns_none(self):
        from src.ui.pages._timeseries_form import _baseline_form_value

        df = self._history_with_form()
        result = _baseline_form_value(df, highlight_ids=set())
        assert result is None

    def test_baseline_all_matches_selected_returns_none(self):
        """Si tous les matchs sont dans la sélection, pas de 'before' → None."""
        from src.ui.pages._timeseries_form import _baseline_form_value

        df = self._history_with_form(5)
        ids = set(df["match_id"].to_list())
        result = _baseline_form_value(df, highlight_ids=ids)
        assert result is None

    def test_baseline_valid_returns_float(self):
        from src.ui.pages._timeseries_form import _baseline_form_value

        df = self._history_with_form(30)
        # Sélectionner les 5 derniers matchs, la baseline = avant ceux-là
        last_ids = set(df["match_id"].to_list()[-5:])
        result = _baseline_form_value(df, highlight_ids=last_ids)
        assert isinstance(result, float)

    def test_baseline_uses_tail_14(self):
        """La baseline utilise les 14 dernières valeurs avant la sélection."""
        from src.ui.pages._timeseries_form import _baseline_form_value

        df = self._history_with_form(40)
        last_ids = set(df["match_id"].to_list()[-5:])
        before = df.filter(~pl.col("match_id").is_in(last_ids))
        expected = float(before["form_score"].drop_nulls().tail(14).mean())
        result = _baseline_form_value(df, highlight_ids=last_ids)
        assert result == pytest.approx(expected)
