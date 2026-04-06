"""Tests pour src/visualization/squad_cadence_chart.py."""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl

from src.visualization._plot_options import PlotOptions
from src.visualization.squad_cadence_chart import plot_squad_cadence_profiles


def _make_profiles(n_buckets: int = 10, players: list[str] | None = None) -> pl.DataFrame:
    """Crée un DataFrame de profils de test."""
    players = players or ["Alice", "Bob"]
    data: dict[str, list] = {"phase": list(range(n_buckets))}
    for i, name in enumerate(players):
        data[name] = [float(j + i) for j in range(n_buckets)]
    return pl.DataFrame(data)


class TestPlotSquadCadenceProfiles:
    def test_empty_df_returns_none(self):
        assert plot_squad_cadence_profiles(pl.DataFrame(), []) is None

    def test_empty_names_returns_none(self):
        df = _make_profiles()
        assert plot_squad_cadence_profiles(df, []) is None

    def test_two_players_returns_figure(self):
        df = _make_profiles(players=["Alice", "Bob"])
        fig = plot_squad_cadence_profiles(df, ["Alice", "Bob"])
        assert isinstance(fig, go.Figure)

    def test_trace_count_matches_players(self):
        df = _make_profiles(players=["A", "B", "C"])
        fig = plot_squad_cadence_profiles(df, ["A", "B", "C"])
        assert fig is not None
        assert len(fig.data) == 3

    def test_missing_player_name_skipped(self):
        """Un joueur non présent dans les colonnes est ignoré."""
        df = _make_profiles(players=["Alice"])
        fig = plot_squad_cadence_profiles(df, ["Alice", "Unknown"])
        assert fig is not None
        assert len(fig.data) == 1

    def test_custom_height(self):
        df = _make_profiles()
        opts = PlotOptions(height_px=500)
        fig = plot_squad_cadence_profiles(df, ["Alice", "Bob"], opts=opts)
        assert fig is not None
        assert fig.layout.height == 500

    def test_custom_lang_en(self):
        df = _make_profiles()
        opts = PlotOptions(lang="en")
        fig = plot_squad_cadence_profiles(df, ["Alice", "Bob"], opts=opts)
        assert fig is not None

    def test_x_labels_percentage(self):
        df = _make_profiles(n_buckets=10)
        fig = plot_squad_cadence_profiles(df, ["Alice", "Bob"])
        assert fig is not None
        x_data = fig.data[0].x
        assert x_data[0] == "0–10%"
        assert x_data[-1] == "90–100%"

    def test_legend_horizontal(self):
        df = _make_profiles()
        fig = plot_squad_cadence_profiles(df, ["Alice", "Bob"])
        assert fig is not None
        assert fig.layout.legend.orientation == "h"

    def test_bar_traces(self):
        df = _make_profiles(players=["Alice"])
        fig = plot_squad_cadence_profiles(df, ["Alice"])
        assert fig is not None
        assert isinstance(fig.data[0], go.Bar)

    def test_barmode_group(self):
        df = _make_profiles(players=["Alice", "Bob"])
        fig = plot_squad_cadence_profiles(df, ["Alice", "Bob"])
        assert fig is not None
        assert fig.layout.barmode == "group"

    def test_color_map_applied(self):
        """Les couleurs du color_map doivent être utilisées."""
        df = _make_profiles(players=["Alice", "Bob"])
        color_map = {"Alice": "#FF0000", "Bob": "#00FF00"}
        fig = plot_squad_cadence_profiles(df, ["Alice", "Bob"], color_map=color_map)
        assert fig is not None
        assert fig.data[0].marker.color == "#FF0000"
        assert fig.data[1].marker.color == "#00FF00"

    def test_color_map_none_uses_defaults(self):
        """Sans color_map, la palette interne est utilisée (pas de crash)."""
        df = _make_profiles(players=["Alice"])
        fig = plot_squad_cadence_profiles(df, ["Alice"], color_map=None)
        assert fig is not None
        assert fig.data[0].marker.color == "#0072B2"
