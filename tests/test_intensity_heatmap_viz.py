"""Tests pour src/visualization/match_intensity_heatmap.py."""

from __future__ import annotations

import plotly.graph_objects as go
import polars as pl

from src.analysis.match_intensity import IntensityProfile
from src.visualization._plot_options import PlotOptions
from src.visualization.match_intensity_heatmap import (
    _build_heatmap_trace,
    plot_match_intensity_heatmap,
)


def _make_profile(n_matches: int = 5, n_buckets: int = 10) -> IntensityProfile:
    """Crée un IntensityProfile de test avec n_matches lignes."""
    data = {"match_id": [f"m{i}" for i in range(n_matches)]}
    for j in range(n_buckets):
        data[f"phase_{j}"] = [j + i for i in range(n_matches)]
    return IntensityProfile(df=pl.DataFrame(data), n_buckets=n_buckets)


class TestBuildHeatmapTrace:
    def test_returns_heatmap(self):
        trace = _build_heatmap_trace([[1, 2], [3, 4]], ["a", "b"], ["#1", "#2"], "fr")
        assert isinstance(trace, go.Heatmap)

    def test_colorscale_has_5_stops(self):
        trace = _build_heatmap_trace([[1]], ["a"], ["#1"], "fr")
        assert len(trace.colorscale) == 5


class TestPlotMatchIntensityHeatmap:
    def test_empty_profile_returns_none(self):
        empty = IntensityProfile(
            df=pl.DataFrame(schema={"match_id": pl.Utf8, "phase_0": pl.Int32}),
            n_buckets=1,
        )
        assert plot_match_intensity_heatmap(empty) is None

    def test_single_match_returns_none(self):
        """Moins de 2 matchs → None."""
        profile = _make_profile(n_matches=1)
        assert plot_match_intensity_heatmap(profile) is None

    def test_two_matches_returns_figure(self):
        profile = _make_profile(n_matches=2)
        fig = plot_match_intensity_heatmap(profile)
        assert isinstance(fig, go.Figure)

    def test_five_matches_returns_figure(self):
        profile = _make_profile(n_matches=5)
        fig = plot_match_intensity_heatmap(profile)
        assert isinstance(fig, go.Figure)
        # autorange="reversed" retiré intentionnellement (refonte visuelle v6.5)
        assert fig.layout.yaxis.title.text == ""

    def test_custom_options(self):
        profile = _make_profile(n_matches=3)
        opts = PlotOptions(lang="en")
        fig = plot_match_intensity_heatmap(profile, opts=opts)
        assert isinstance(fig, go.Figure)

    def test_height_scales_with_matches(self):
        small = plot_match_intensity_heatmap(_make_profile(n_matches=3))
        large = plot_match_intensity_heatmap(_make_profile(n_matches=20))
        assert small is not None and large is not None
        assert large.layout.height >= small.layout.height

    def test_height_capped_at_600(self):
        profile = _make_profile(n_matches=50)
        fig = plot_match_intensity_heatmap(profile)
        assert fig is not None
        assert fig.layout.height <= 600

    def test_x_labels_percentage_format(self):
        profile = _make_profile(n_matches=3, n_buckets=10)
        fig = plot_match_intensity_heatmap(profile)
        assert fig is not None
        # Première tranche = "0–10%"
        x_data = fig.data[0].x
        assert x_data[0] == "0–10%"
        assert x_data[-1] == "90–100%"
