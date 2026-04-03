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
    def _make_fig(self) -> go.Figure:
        return go.Figure()

    def test_no_shape_if_records_empty(self):
        fig = self._make_fig()
        add_record_shapes(fig, xs=[0, 1], records={}, player_names=["Alice"], n_players=1)
        assert len(fig.layout.shapes) == 0

    def test_no_shape_if_record_is_none(self):
        fig = self._make_fig()
        add_record_shapes(
            fig, xs=[0, 1], records={"Alice": None}, player_names=["Alice"], n_players=1
        )
        assert len(fig.layout.shapes) == 0

    def test_one_shape_per_x_per_player(self):
        fig = self._make_fig()
        add_record_shapes(
            fig,
            xs=[0, 1, 2],
            records={"Alice": 15.0},
            player_names=["Alice"],
            n_players=1,
        )
        # 3 xs × 1 player = 3 shapes
        assert len(fig.layout.shapes) == 3

    def test_two_players_correct_shape_count(self):
        fig = self._make_fig()
        add_record_shapes(
            fig,
            xs=[0, 1],
            records={"Alice": 10.0, "Bob": 20.0},
            player_names=["Alice", "Bob"],
            n_players=2,
        )
        # 2 xs × 2 players = 4 shapes
        assert len(fig.layout.shapes) == 4

    def test_shape_y_value_positive(self):
        """Le rectangle va de 0 à la valeur record (non-négatif)."""
        fig = self._make_fig()
        add_record_shapes(
            fig, xs=[0], records={"Alice": 15.0}, player_names=["Alice"], n_players=1
        )
        shape = fig.layout.shapes[0]
        assert shape.y0 == pytest.approx(0.0)
        assert shape.y1 == pytest.approx(15.0)

    def test_shape_y_value_negative_for_deaths(self):
        """Les morts sont dessinées en négatif sur l'axe Y."""
        fig = self._make_fig()
        add_record_shapes(
            fig, xs=[0], records={"Alice": 3.0}, player_names=["Alice"], n_players=1,
            is_negative=True
        )
        shape = fig.layout.shapes[0]
        assert shape.y0 == pytest.approx(-3.0)

    def test_shapes_are_rectangles(self):
        """Les records sont des rectangles (type=rect) de 0 à la valeur record."""
        fig = self._make_fig()
        add_record_shapes(
            fig, xs=[0], records={"Alice": 10.0}, player_names=["Alice"], n_players=1
        )
        shape = fig.layout.shapes[0]
        assert shape.type == "rect"
        assert shape.y0 == pytest.approx(0.0)
        assert shape.y1 == pytest.approx(10.0)

    def test_shape_x_span_width(self):
        """La forme couvre exactement la largeur du baton (2 × half_width)."""
        fig = self._make_fig()
        add_record_shapes(
            fig, xs=[0], records={"Alice": 10.0}, player_names=["Alice"], n_players=1
        )
        shape = fig.layout.shapes[0]
        hw = _bar_half_width(1)
        expected_width = 2 * hw
        actual_width = shape.x1 - shape.x0
        assert actual_width == pytest.approx(expected_width)

    def test_shape_line_color_default(self):
        """Sans colors_by_name, la bordure est blanche (#ffffff)."""
        fig = self._make_fig()
        add_record_shapes(
            fig, xs=[0], records={"Alice": 5.0}, player_names=["Alice"], n_players=1
        )
        shape = fig.layout.shapes[0]
        assert shape.line.color == "#ffffff"

    def test_shape_line_color_from_player(self):
        """Avec colors_by_name, la bordure utilise la couleur du joueur."""
        fig = self._make_fig()
        add_record_shapes(
            fig, xs=[0], records={"Alice": 5.0}, player_names=["Alice"], n_players=1,
            colors_by_name={"Alice": "#56B4E9"},
        )
        shape = fig.layout.shapes[0]
        assert shape.line.color == "#56B4E9"

    def test_shape_layer_above(self):
        fig = self._make_fig()
        add_record_shapes(
            fig, xs=[0], records={"Alice": 5.0}, player_names=["Alice"], n_players=1
        )
        shape = fig.layout.shapes[0]
        assert shape.layer == "above"

    def test_player_not_in_records_skipped(self):
        """Si un joueur est dans player_names mais pas dans records, aucune shape pour lui."""
        fig = self._make_fig()
        add_record_shapes(
            fig,
            xs=[0],
            records={"Alice": 10.0},
            player_names=["Alice", "Bob"],
            n_players=2,
        )
        # Seulement Alice → 1 shape
        assert len(fig.layout.shapes) == 1

    def test_two_players_different_x_offsets(self):
        """Les deux joueurs ont des offsets X différents (barres côte à côte)."""
        fig = self._make_fig()
        add_record_shapes(
            fig,
            xs=[0],
            records={"Alice": 10.0, "Bob": 10.0},
            player_names=["Alice", "Bob"],
            n_players=2,
        )
        shapes = fig.layout.shapes
        assert len(shapes) == 2
        center_0 = (shapes[0].x0 + shapes[0].x1) / 2
        center_1 = (shapes[1].x0 + shapes[1].x1) / 2
        assert center_0 != pytest.approx(center_1)

    def test_empty_xs(self):
        fig = self._make_fig()
        add_record_shapes(
            fig, xs=[], records={"Alice": 10.0}, player_names=["Alice"], n_players=1
        )
        assert len(fig.layout.shapes) == 0


# ---------------------------------------------------------------------------
# add_overlay_record_shapes
# ---------------------------------------------------------------------------


class TestAddOverlayRecordShapes:
    def _make_fig(self) -> go.Figure:
        return go.Figure()

    def test_no_shape_if_record_none(self):
        fig = self._make_fig()
        add_overlay_record_shapes(
            fig, xs=[0], records={"Alice": None}, player_names=["Alice"]
        )
        assert len(fig.layout.shapes) == 0

    def test_one_shape_per_x(self):
        fig = self._make_fig()
        add_overlay_record_shapes(
            fig, xs=[0, 1, 2], records={"Alice": 5.0}, player_names=["Alice"]
        )
        assert len(fig.layout.shapes) == 3

    def test_centered_at_x(self):
        fig = self._make_fig()
        add_overlay_record_shapes(
            fig, xs=[0], records={"Alice": 5.0}, player_names=["Alice"]
        )
        shape = fig.layout.shapes[0]
        center = (shape.x0 + shape.x1) / 2
        assert center == pytest.approx(0.0)

    def test_shape_y_value(self):
        """Le rectangle overlay va de 0 à la valeur record."""
        fig = self._make_fig()
        add_overlay_record_shapes(
            fig, xs=[0], records={"Alice": 7.5}, player_names=["Alice"]
        )
        shape = fig.layout.shapes[0]
        assert shape.y0 == pytest.approx(0.0)
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
        add_overlay_record_shapes(
            fig, xs=[0], records={"Alice": 3.0}, player_names=["Alice"]
        )
        assert fig.layout.shapes[0].line.color == "#ffffff"
