"""Tests pour src/visualization/_cadence_histogram.py."""

from __future__ import annotations

import plotly.graph_objects as go

from src.analysis.match_cadence import CadenceBucket
from src.visualization._cadence_histogram import (
    _format_time_label,
    plot_match_cadence_histogram,
)
from src.visualization._plot_options import PlotOptions

# =============================================================================
# _format_time_label
# =============================================================================


class TestFormatTimeLabel:
    def test_zero(self):
        assert _format_time_label(0) == "0:00"

    def test_seconds_only(self):
        assert _format_time_label(45) == "0:45"

    def test_minutes_and_seconds(self):
        assert _format_time_label(90) == "1:30"

    def test_exact_minute(self):
        assert _format_time_label(120) == "2:00"

    def test_single_digit_seconds_padded(self):
        assert _format_time_label(65) == "1:05"

    def test_fractional_truncated(self):
        assert _format_time_label(90.7) == "1:30"


# =============================================================================
# plot_match_cadence_histogram
# =============================================================================


def _make_buckets(
    kills_pairs: list[tuple[int, int]], bucket_s: float = 30.0
) -> list[CadenceBucket]:
    """Helper : crée des CadenceBucket à partir de paires (my_kills, enemy_kills)."""
    return [
        CadenceBucket(
            t_start_s=i * bucket_s,
            t_end_s=(i + 1) * bucket_s,
            my_kills=mk,
            enemy_kills=ek,
        )
        for i, (mk, ek) in enumerate(kills_pairs)
    ]


class TestPlotMatchCadenceHistogram:
    def test_empty_buckets_returns_none(self):
        result = plot_match_cadence_histogram([], duration_s=120.0)
        assert result is None

    def test_insufficient_kills_returns_none(self):
        buckets = _make_buckets([(1, 0), (0, 1)])  # total=2 < 3
        result = plot_match_cadence_histogram(buckets, duration_s=60.0)
        assert result is None

    def test_sufficient_kills_returns_figure(self):
        buckets = _make_buckets([(2, 1), (1, 0)])  # total=4 >= 3
        fig = plot_match_cadence_histogram(buckets, duration_s=60.0)
        assert isinstance(fig, go.Figure)

    def test_three_traces(self):
        """On attend 3 traces : barres mon équipe, barres adverses, moyenne glissante."""
        buckets = _make_buckets([(3, 2), (1, 1), (2, 3)])
        fig = plot_match_cadence_histogram(buckets, duration_s=90.0)
        assert fig is not None
        assert len(fig.data) == 3

    def test_bar_traces_stacked(self):
        buckets = _make_buckets([(2, 1), (1, 2)])
        fig = plot_match_cadence_histogram(buckets, duration_s=60.0)
        assert fig is not None
        assert fig.layout.barmode == "stack"

    def test_custom_plot_options(self):
        buckets = _make_buckets([(3, 2), (1, 1)])
        opts = PlotOptions(lang="en", height_px=500)
        fig = plot_match_cadence_histogram(buckets, duration_s=60.0, opts=opts)
        assert fig is not None
        assert fig.layout.height == 500

    def test_default_height(self):
        buckets = _make_buckets([(3, 2), (1, 1)])
        fig = plot_match_cadence_histogram(buckets, duration_s=60.0)
        assert fig is not None
        assert fig.layout.height == PlotOptions().height_px

    def test_peak_annotation(self):
        """Le pic d'intensité doit avoir une annotation."""
        buckets = _make_buckets([(5, 3), (1, 0), (0, 1)])
        fig = plot_match_cadence_histogram(buckets, duration_s=90.0)
        assert fig is not None
        assert len(fig.layout.annotations) >= 1

    def test_no_peak_when_all_zero_kills(self):
        """Pas d'annotation si aucun kill (mais total < 3 → None)."""
        buckets = _make_buckets([(0, 0), (0, 0)])
        result = plot_match_cadence_histogram(buckets, duration_s=60.0)
        assert result is None

    def test_x_labels_formatted(self):
        """Les labels X doivent être au format mm:ss."""
        buckets = _make_buckets([(2, 1), (1, 1)], bucket_s=30.0)
        fig = plot_match_cadence_histogram(buckets, duration_s=60.0)
        assert fig is not None
        # Premier bucket centré à 15s → "0:15"
        assert fig.data[0].x[0] == "0:15"

    def test_moving_average_trace_is_scatter(self):
        buckets = _make_buckets([(3, 2), (1, 1), (2, 3)])
        fig = plot_match_cadence_histogram(buckets, duration_s=90.0)
        assert fig is not None
        ma_trace = fig.data[2]
        assert isinstance(ma_trace, go.Scatter)
        assert ma_trace.mode == "lines"
