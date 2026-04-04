"""Tests unitaires pour src/visualization/_chart_series.py.

Couvre :
- MatchSeries : construction
- ChartData : propriétés dérivées, downsample, add_record_overlays dispatch
- _add_categorical_record_bars : logique des ghost bars
- HEIGHT_* et MAX_PLOT_POINTS : présence des constantes
"""

from __future__ import annotations

from unittest.mock import MagicMock, patch

import plotly.graph_objects as go
import pytest

from src.visualization._chart_series import (
    HEIGHT_COMPACT,
    HEIGHT_NORMAL,
    HEIGHT_PM,
    MAX_PLOT_POINTS,
    ChartData,
    MatchSeries,
    _add_categorical_record_bars,
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
        with patch(
            "src.visualization._squad_record_shapes.add_record_shapes"
        ) as mock_add:
            cd.add_record_overlays(fig)
        assert mock_add.called

    def test_overlay_mode_calls_add_overlay_record_shapes(self) -> None:
        cd = _make_chart_data(barmode="overlay")
        fig = go.Figure()
        with patch(
            "src.visualization._squad_record_shapes.add_overlay_record_shapes"
        ) as mock_add:
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
