"""Tests unitaires pour src/visualization/_squad_record_shapes.py."""

from __future__ import annotations

import plotly.graph_objects as go
import pytest

from src.visualization._squad_record_shapes import (
    _BAR_FILL,
    _BAR_GAP,
    _bar_center_offset,
    _bar_half_width,
    add_overlay_record_shapes,
    add_record_shapes,
)

# ---------------------------------------------------------------------------
# Helpers internes
# ---------------------------------------------------------------------------


class TestBarHalfWidth:
    def test_single_player(self):
        hw = _bar_half_width(1)
        expected = (1.0 - _BAR_GAP) * _BAR_FILL / 2
        assert hw == pytest.approx(expected)

    def test_two_players(self):
        hw = _bar_half_width(2)
        bar_w = (1.0 - _BAR_GAP) / 2
        assert hw == pytest.approx(bar_w * _BAR_FILL / 2)

    def test_four_players(self):
        hw = _bar_half_width(4)
        bar_w = (1.0 - _BAR_GAP) / 4
        assert hw == pytest.approx(bar_w * _BAR_FILL / 2)


class TestBarCenterOffset:
    def test_single_player_centered(self):
        assert _bar_center_offset(0, 1) == pytest.approx(0.0)

    def test_two_players_symmetric(self):
        off0 = _bar_center_offset(0, 2)
        off1 = _bar_center_offset(1, 2)
        assert off0 == pytest.approx(-off1)

    def test_three_players_middle_is_zero(self):
        assert _bar_center_offset(1, 3) == pytest.approx(0.0)


# ---------------------------------------------------------------------------
# add_record_shapes
# ---------------------------------------------------------------------------


class TestAddRecordShapes:
    """add_record_shapes ajoute des traces go.Bar fantômes hachurées (fig.data)."""

    def _make_fig(self) -> go.Figure:
        return go.Figure()

    def _ghost_traces(self, fig: go.Figure) -> list:
        return [t for t in fig.data if isinstance(t, go.Bar) and t.showlegend is False]

    def test_no_trace_if_records_empty(self):
        fig = self._make_fig()
        add_record_shapes(fig, xs=[0, 1], records={}, player_names=["Alice"], n_players=1)
        assert len(self._ghost_traces(fig)) == 0

    def test_no_trace_if_record_is_none(self):
        fig = self._make_fig()
        add_record_shapes(
            fig, xs=[0, 1], records={"Alice": None}, player_names=["Alice"], n_players=1
        )
        assert len(self._ghost_traces(fig)) == 0

    def test_one_trace_per_player_with_data(self):
        """Une trace fantôme par joueur ayant un record non-None."""
        fig = self._make_fig()
        add_record_shapes(
            fig,
            xs=[0, 1, 2],
            records={"Alice": 15.0},
            player_names=["Alice"],
            n_players=1,
        )
        assert len(self._ghost_traces(fig)) == 1

    def test_two_players_two_traces(self):
        fig = self._make_fig()
        add_record_shapes(
            fig,
            xs=[0, 1],
            records={"Alice": 10.0, "Bob": 20.0},
            player_names=["Alice", "Bob"],
            n_players=2,
        )
        assert len(self._ghost_traces(fig)) == 2

    def test_y_value_positive(self):
        """La trace fantôme a y=[record_val] pour chaque xs."""
        fig = self._make_fig()
        add_record_shapes(
            fig, xs=[0, 1], records={"Alice": 15.0}, player_names=["Alice"], n_players=1
        )
        trace = self._ghost_traces(fig)[0]
        assert list(trace.y) == pytest.approx([15.0, 15.0])

    def test_y_value_negative_for_deaths(self):
        """is_negative=True → valeurs y négatives."""
        fig = self._make_fig()
        add_record_shapes(
            fig,
            xs=[0],
            records={"Alice": 3.0},
            player_names=["Alice"],
            n_players=1,
            is_negative=True,
        )
        trace = self._ghost_traces(fig)[0]
        assert list(trace.y) == pytest.approx([-3.0])

    def test_uses_pattern_shape(self):
        """La trace fantôme utilise un hachurage."""
        fig = self._make_fig()
        add_record_shapes(fig, xs=[0], records={"Alice": 5.0}, player_names=["Alice"], n_players=1)
        trace = self._ghost_traces(fig)[0]
        assert trace.marker.pattern.shape == "/"

    def test_border_color_default(self):
        """Sans colors_by_name, la bordure est blanche (#ffffff)."""
        fig = self._make_fig()
        add_record_shapes(fig, xs=[0], records={"Alice": 5.0}, player_names=["Alice"], n_players=1)
        trace = self._ghost_traces(fig)[0]
        assert trace.marker.line.color == "#ffffff"

    def test_border_color_from_player(self):
        """Avec colors_by_name, la bordure et le hachurage utilisent la couleur du joueur."""
        fig = self._make_fig()
        add_record_shapes(
            fig,
            xs=[0],
            records={"Alice": 5.0},
            player_names=["Alice"],
            n_players=1,
            colors_by_name={"Alice": "#56B4E9"},
        )
        trace = self._ghost_traces(fig)[0]
        assert trace.marker.line.color == "#56B4E9"
        assert trace.marker.pattern.fgcolor == "#56B4E9"

    def test_offsetgroup_equals_player_name(self):
        """offsetgroup=name pour que la barre fantôme s'aligne sur la barre réelle."""
        fig = self._make_fig()
        add_record_shapes(fig, xs=[0], records={"Alice": 5.0}, player_names=["Alice"], n_players=1)
        trace = self._ghost_traces(fig)[0]
        assert trace.offsetgroup == "Alice"

    def test_player_not_in_records_skipped(self):
        """Bob absent de records → aucune trace pour Bob."""
        fig = self._make_fig()
        add_record_shapes(
            fig,
            xs=[0],
            records={"Alice": 10.0},
            player_names=["Alice", "Bob"],
            n_players=2,
        )
        assert len(self._ghost_traces(fig)) == 1
        assert self._ghost_traces(fig)[0].offsetgroup == "Alice"

    def test_empty_xs(self):
        """xs vide → y=[] → aucune trace ajoutée."""
        fig = self._make_fig()
        add_record_shapes(fig, xs=[], records={"Alice": 10.0}, player_names=["Alice"], n_players=1)
        assert len(self._ghost_traces(fig)) == 0


# ---------------------------------------------------------------------------
# add_overlay_record_shapes
# ---------------------------------------------------------------------------


class TestAddOverlayRecordShapes:
    def _make_fig(self) -> go.Figure:
        return go.Figure()

    def test_no_shape_if_record_none(self):
        fig = self._make_fig()
        add_overlay_record_shapes(fig, xs=[0], records={"Alice": None}, player_names=["Alice"])
        assert len(fig.layout.shapes) == 0

    def test_one_shape_per_x(self):
        fig = self._make_fig()
        add_overlay_record_shapes(fig, xs=[0, 1, 2], records={"Alice": 5.0}, player_names=["Alice"])
        assert len(fig.layout.shapes) == 3

    def test_centered_at_x(self):
        fig = self._make_fig()
        add_overlay_record_shapes(fig, xs=[0], records={"Alice": 5.0}, player_names=["Alice"])
        shape = fig.layout.shapes[0]
        center = (shape.x0 + shape.x1) / 2
        assert center == pytest.approx(0.0)

    def test_shape_y_value(self):
        """La ligne overlay est tracée à la hauteur exacte du record."""
        fig = self._make_fig()
        add_overlay_record_shapes(fig, xs=[0], records={"Alice": 7.5}, player_names=["Alice"])
        shape = fig.layout.shapes[0]
        assert shape.type == "line"
        assert shape.y0 == pytest.approx(7.5)
        assert shape.y1 == pytest.approx(7.5)

    def test_two_players_same_x_center(self):
        """En overlay, les deux joueurs ont des shapes centrées au même X."""
        fig = self._make_fig()
        add_overlay_record_shapes(
            fig,
            xs=[0],
            records={"Alice": 5.0, "Bob": 8.0},
            player_names=["Alice", "Bob"],
        )
        shapes = fig.layout.shapes
        assert len(shapes) == 2
        center_alice = (shapes[0].x0 + shapes[0].x1) / 2
        center_bob = (shapes[1].x0 + shapes[1].x1) / 2
        assert center_alice == pytest.approx(center_bob)

    def test_shape_line_color_default(self):
        """Sans colors_by_name, la bordure overlay est blanche (#ffffff)."""
        fig = self._make_fig()
        add_overlay_record_shapes(fig, xs=[0], records={"Alice": 3.0}, player_names=["Alice"])
        assert fig.layout.shapes[0].line.color == "#ffffff"
