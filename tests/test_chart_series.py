"""Tests unitaires pour src/visualization/_chart_series.py.

Couvre :
- MatchSeries : construction
- ChartData : propriétés dérivées, downsample, add_record_overlays dispatch
- _add_categorical_record_bars : logique des ghost bars
- HEIGHT_* et MAX_PLOT_POINTS : présence des constantes
- HEIGHT_TIMESERIES / HEIGHT_PROGRESSION / HEIGHT_MINI (Axe F)
- _rolling_mean_list : calcul sans polars
- SingleSeriesChartData : construction + from_series
"""

from __future__ import annotations

from unittest.mock import patch

import plotly.graph_objects as go
import pytest

from src.visualization._chart_series import (
    HEIGHT_COMPACT,
    HEIGHT_MINI,
    HEIGHT_NORMAL,
    HEIGHT_PM,
    HEIGHT_PROGRESSION,
    HEIGHT_TIMESERIES,
    MAX_PLOT_POINTS,
    ChartData,
    MatchSeries,
    SingleSeriesChartData,
    _add_categorical_record_bars,
    _rolling_mean_list,
)

# ─── Helpers ──────────────────────────────────────────────────────────────────


def _make_series(name: str = "Alice", n: int = 5) -> MatchSeries:
    return MatchSeries(
        name=name,
        x=list(range(n)),
        y=[float(i) for i in range(n)],
        color="#56B4E9",
        map_names=["Recharge"] * n,
    )


def _make_chart_data(barmode: str = "group", n: int = 5) -> ChartData:
    return ChartData(
        series=[_make_series("Alice", n), _make_series("Bob", n)],
        x_labels=[f"M{i}" for i in range(n)],
        barmode=barmode,  # type: ignore[arg-type]
        global_records={"Alice": 10.0, "Bob": 8.0},
        per_map_records={"Alice": {"Recharge": 10.0}},
    )


# ─── Constantes ───────────────────────────────────────────────────────────────


class TestConstants:
    def test_height_compact_defined(self) -> None:
        assert isinstance(HEIGHT_COMPACT, int)
        assert HEIGHT_COMPACT > 0

    def test_height_normal_defined(self) -> None:
        assert isinstance(HEIGHT_NORMAL, int)
        assert HEIGHT_NORMAL > 0

    def test_height_pm_defined(self) -> None:
        assert isinstance(HEIGHT_PM, int)
        assert HEIGHT_PM > 0

    def test_max_plot_points_defined(self) -> None:
        assert isinstance(MAX_PLOT_POINTS, int)
        assert MAX_PLOT_POINTS >= 50

    def test_height_timeseries_defined(self) -> None:
        assert isinstance(HEIGHT_TIMESERIES, int)
        assert HEIGHT_TIMESERIES > 0

    def test_height_progression_defined(self) -> None:
        assert isinstance(HEIGHT_PROGRESSION, int)
        assert HEIGHT_PROGRESSION > 0

    def test_height_mini_defined(self) -> None:
        assert isinstance(HEIGHT_MINI, int)
        assert HEIGHT_MINI > 0

    def test_height_ordering(self) -> None:
        """MINI < PROGRESSION < TIMESERIES (tailles croissantes par usage)."""
        assert HEIGHT_MINI < HEIGHT_PROGRESSION
        assert HEIGHT_PROGRESSION <= HEIGHT_TIMESERIES


# ─── _rolling_mean_list ────────────────────────────────────────────────────────


class TestRollingMeanList:
    def test_single_value(self) -> None:
        assert _rolling_mean_list([10.0]) == [10.0]

    def test_simple_3_values(self) -> None:
        # Fenêtre glissante : [10], [10,20], [10,20,15]
        result = _rolling_mean_list([10.0, 20.0, 15.0], window=10)
        assert result[0] == pytest.approx(10.0)
        assert result[1] == pytest.approx(15.0)  # (10+20)/2
        assert result[2] == pytest.approx(15.0)  # (10+20+15)/3

    def test_none_passthrough(self) -> None:
        result = _rolling_mean_list([1.0, None, 3.0], window=10)
        assert result[1] is None
        # Les valeurs None sont ignorées dans le calcul du mean
        assert result[2] == pytest.approx(2.0)  # (1+3)/2

    def test_all_none(self) -> None:
        result = _rolling_mean_list([None, None], window=10)
        assert result == [None, None]

    def test_window_limits(self) -> None:
        vals = [float(i) for i in range(1, 6)]  # [1,2,3,4,5]
        result = _rolling_mean_list(vals, window=3)
        # Fenêtre 3 : [1], [1,2], [1,2,3], [2,3,4], [3,4,5]
        assert result[0] == pytest.approx(1.0)
        assert result[1] == pytest.approx(1.5)
        assert result[2] == pytest.approx(2.0)
        assert result[3] == pytest.approx(3.0)
        assert result[4] == pytest.approx(4.0)

    def test_empty_list(self) -> None:
        assert _rolling_mean_list([]) == []


# ─── SingleSeriesChartData ────────────────────────────────────────────────────


class TestSingleSeriesChartData:
    def test_from_series_basic(self) -> None:
        obj = SingleSeriesChartData.from_series([1, 2, 3], [10.0, 20.0, 15.0])
        assert obj.x == [1, 2, 3]
        assert obj.y == [10.0, 20.0, 15.0]
        assert len(obj.y_smooth) == 3
        assert obj.y_smooth[0] == pytest.approx(10.0)
        assert obj.height == HEIGHT_TIMESERIES

    def test_from_series_height_override(self) -> None:
        obj = SingleSeriesChartData.from_series([1, 2], [5.0, 10.0], height=HEIGHT_MINI)
        assert obj.height == HEIGHT_MINI

    def test_from_series_title(self) -> None:
        obj = SingleSeriesChartData.from_series([1], [3.0], title="KDA")
        assert obj.title == "KDA"

    def test_from_series_empty(self) -> None:
        obj = SingleSeriesChartData.from_series([], [])
        assert obj.x == []
        assert obj.y == []
        assert obj.y_smooth == []

    def test_from_series_with_none(self) -> None:
        obj = SingleSeriesChartData.from_series([1, 2, 3], [1.0, None, 3.0])
        assert obj.y_smooth[1] is None

    def test_default_title_empty(self) -> None:
        obj = SingleSeriesChartData.from_series([1], [1.0])
        assert obj.title == ""

    def test_window_kwarg_forwarded(self) -> None:
        """window=1 → y_smooth == y (chaque valeur est sa propre moyenne)."""
        vals = [10.0, 20.0, 30.0]
        obj = SingleSeriesChartData.from_series([1, 2, 3], vals, window=1)
        assert obj.y_smooth == pytest.approx(vals)


# ─── MatchSeries ──────────────────────────────────────────────────────────────


class TestMatchSeries:
    def test_basic_construction(self) -> None:
        s = _make_series("Alice", 3)
        assert s.name == "Alice"
        assert len(s.x) == 3
        assert len(s.y) == 3
        assert len(s.map_names) == 3

    def test_color_stored(self) -> None:
        s = _make_series()
        assert s.color == "#56B4E9"


# ─── ChartData.player_names / colors_by_name ──────────────────────────────────


class TestChartDataProperties:
    def test_player_names(self) -> None:
        cd = _make_chart_data()
        assert cd.player_names == ["Alice", "Bob"]

    def test_colors_by_name(self) -> None:
        cd = _make_chart_data()
        assert "Alice" in cd.colors_by_name
        assert "Bob" in cd.colors_by_name

    def test_tick_step_small(self) -> None:
        cd = _make_chart_data(n=5)
        assert cd.tick_step == 1

    def test_tick_step_large(self) -> None:
        cd = ChartData(
            series=[_make_series("A", 100)],
            x_labels=[str(i) for i in range(100)],
            barmode="group",
        )
        assert cd.tick_step == 10

    def test_tick_step_empty(self) -> None:
        cd = ChartData(series=[], x_labels=[], barmode="group")
        assert cd.tick_step == 1


# ─── ChartData.downsample ─────────────────────────────────────────────────────


class TestChartDataDownsample:
    def test_no_downsample_when_under_limit(self) -> None:
        cd = _make_chart_data(n=5)
        result = cd.downsample(max_points=200)
        assert result is cd  # même objet, pas de copie

    def test_downsamples_when_over_limit(self) -> None:
        n = 300
        cd = ChartData(
            series=[_make_series("Alice", n)],
            x_labels=[str(i) for i in range(n)],
            barmode="group",
            global_records={"Alice": 5.0},
        )
        result = cd.downsample(max_points=100)
        assert result is not cd
        assert len(result.x_labels) < n
        assert len(result.series[0].x) < n

    def test_downsample_preserves_records(self) -> None:
        n = 300
        cd = ChartData(
            series=[_make_series("Alice", n)],
            x_labels=[str(i) for i in range(n)],
            barmode="group",
            global_records={"Alice": 5.0},
            per_map_records={"Alice": {"Recharge": 5.0}},
        )
        result = cd.downsample(max_points=100)
        assert result.global_records == {"Alice": 5.0}
        assert result.per_map_records == {"Alice": {"Recharge": 5.0}}

    def test_downsample_empty_series(self) -> None:
        cd = ChartData(series=[], x_labels=[], barmode="group")
        result = cd.downsample(max_points=100)
        assert result is cd


# ─── ChartData.add_record_overlays ────────────────────────────────────────────


class TestAddRecordOverlaysDispatch:
    def test_no_op_when_no_records(self) -> None:
        cd = ChartData(series=[], x_labels=[], barmode="group")
        fig = go.Figure()
        cd.add_record_overlays(fig)  # ne doit pas lever d'exception

    def test_group_mode_calls_add_record_shapes(self) -> None:
        cd = _make_chart_data(barmode="group")
        fig = go.Figure()
        with patch("src.visualization._squad_record_shapes.add_record_shapes") as mock_add:
            cd.add_record_overlays(fig)
        assert mock_add.called

    def test_overlay_mode_calls_add_overlay_record_shapes(self) -> None:
        cd = _make_chart_data(barmode="overlay")
        fig = go.Figure()
        with patch("src.visualization._squad_record_shapes.add_overlay_record_shapes") as mock_add:
            cd.add_record_overlays(fig)
        assert mock_add.called

    def test_categorical_mode_calls_categorical_fn(self) -> None:
        """Mode categoriel → _add_categorical_record_bars (sans appel à _squad_record_shapes)."""
        # global_records doit être un tuple (r_kpm, r_dpm, r_apm)
        cd = ChartData(
            series=[_make_series("Alice")],
            x_labels=["Frags/min", "Morts/min", "Assists/min"],
            barmode="categorical",
            global_records={"Alice": (1.5, 0.8, 0.4)},
        )
        fig = go.Figure()
        initial_traces = len(fig.data)
        cd.add_record_overlays(fig)
        # La trace ghost doit avoir été ajoutée
        assert len(fig.data) > initial_traces


# ─── _add_categorical_record_bars ─────────────────────────────────────────────


class TestAddCategoricalRecordBars:
    def test_adds_trace_for_valid_tuple(self) -> None:
        cd = ChartData(
            series=[_make_series("Alice")],
            x_labels=["Frags/min", "Morts/min", "Assists/min"],
            barmode="categorical",
            global_records={"Alice": (1.5, 0.8, 0.4)},
        )
        fig = go.Figure()
        _add_categorical_record_bars(fig, cd)
        assert len(fig.data) == 1
        trace = fig.data[0]
        assert trace.name == "Alice"
        assert trace.y[1] == pytest.approx(-0.8)  # mort → valeur négative

    def test_skips_non_tuple_record(self) -> None:
        cd = ChartData(
            series=[_make_series("Alice")],
            x_labels=["Frags/min", "Morts/min", "Assists/min"],
            barmode="categorical",
            global_records={"Alice": 5.0},  # float, pas tuple → ignoré
        )
        fig = go.Figure()
        _add_categorical_record_bars(fig, cd)
        assert len(fig.data) == 0

    def test_skips_all_none_tuple(self) -> None:
        cd = ChartData(
            series=[_make_series("Alice")],
            x_labels=["A", "B", "C"],
            barmode="categorical",
            global_records={"Alice": (None, None, None)},
        )
        fig = go.Figure()
        _add_categorical_record_bars(fig, cd)
        assert len(fig.data) == 0

    def test_skips_player_not_in_records(self) -> None:
        cd = ChartData(
            series=[_make_series("Alice")],
            x_labels=["A", "B", "C"],
            barmode="categorical",
            global_records={},  # Alice absent
        )
        fig = go.Figure()
        _add_categorical_record_bars(fig, cd)
        assert len(fig.data) == 0

    def test_two_players(self) -> None:
        cd = ChartData(
            series=[_make_series("Alice"), _make_series("Bob")],
            x_labels=["Frags/min", "Morts/min", "Assists/min"],
            barmode="categorical",
            global_records={
                "Alice": (1.5, 0.8, 0.4),
                "Bob": (1.2, 0.6, 0.3),
            },
        )
        fig = go.Figure()
        _add_categorical_record_bars(fig, cd)
        assert len(fig.data) == 2
