"""Tests pour src/analysis/_performance_relative.py.

Couvre compute_relative_performance_score et compute_performance_series.
"""

from __future__ import annotations

from datetime import datetime, timedelta

import polars as pl

from src.analysis._performance_relative import (
    compute_performance_series,
    compute_relative_performance_score,
)
from src.analysis.performance_config import MIN_MATCHES_FOR_RELATIVE


def _make_history_df(n: int = 30) -> pl.DataFrame:
    """Crée un historique de matchs suffisant pour le score relatif."""
    start = datetime(2025, 1, 1)
    return pl.DataFrame(
        {
            "match_id": [f"h{i}" for i in range(n)],
            "start_time": [start + timedelta(hours=i) for i in range(n)],
            "kills": [10 + (i % 10) for i in range(n)],
            "deaths": [5 + (i % 5) for i in range(n)],
            "assists": [3 + (i % 4) for i in range(n)],
            "kda": [1.5 + (i % 5) * 0.2 for i in range(n)],
            "accuracy": [40.0 + (i % 10) for i in range(n)],
            "time_played_seconds": [600 + (i % 5) * 60 for i in range(n)],
            "outcome": [2 if i % 3 == 0 else 3 for i in range(n)],
        }
    )


class TestComputeRelativePerformanceScore:
    """Tests pour compute_relative_performance_score."""

    def test_normal_match(self) -> None:
        history = _make_history_df(30)
        row = {
            "kills": 15,
            "deaths": 6,
            "assists": 5,
            "kda": 3.3,
            "accuracy": 48.0,
            "time_played_seconds": 600,
        }
        result = compute_relative_performance_score(row, history)
        assert result is not None
        assert 0.0 <= result <= 100.0

    def test_empty_history_returns_none(self) -> None:
        history = pl.DataFrame(
            schema={
                "kills": pl.Int64,
                "deaths": pl.Int64,
                "assists": pl.Int64,
                "kda": pl.Float64,
                "accuracy": pl.Float64,
                "time_played_seconds": pl.Int64,
            }
        )
        row = {"kills": 10, "deaths": 5, "assists": 3, "time_played_seconds": 600}
        assert compute_relative_performance_score(row, history) is None

    def test_insufficient_history(self) -> None:
        history = _make_history_df(MIN_MATCHES_FOR_RELATIVE - 1)
        row = {"kills": 10, "deaths": 5, "assists": 3, "time_played_seconds": 600}
        assert compute_relative_performance_score(row, history) is None

    def test_high_performance_above_50(self) -> None:
        history = _make_history_df(30)
        row = {
            "kills": 30,
            "deaths": 2,
            "assists": 10,
            "kda": 20.0,
            "accuracy": 65.0,
            "time_played_seconds": 600,
        }
        result = compute_relative_performance_score(row, history)
        if result is not None:
            assert result > 50.0

    def test_low_performance_below_50(self) -> None:
        history = _make_history_df(30)
        row = {
            "kills": 2,
            "deaths": 15,
            "assists": 0,
            "kda": 0.13,
            "accuracy": 20.0,
            "time_played_seconds": 600,
        }
        result = compute_relative_performance_score(row, history)
        if result is not None:
            assert result < 50.0


class TestComputePerformanceSeries:
    """Tests pour compute_performance_series."""

    def test_empty_df(self) -> None:
        df = pl.DataFrame(
            schema={
                "kills": pl.Int64,
                "deaths": pl.Int64,
                "assists": pl.Int64,
                "kda": pl.Float64,
            }
        )
        result = compute_performance_series(df)
        assert len(result) == 0

    def test_returns_series(self) -> None:
        history = _make_history_df(30)
        result = compute_performance_series(history.head(5), history)
        assert isinstance(result, pl.Series)
        assert len(result) == 5

    def test_insufficient_history_fallback(self) -> None:
        df = _make_history_df(3)
        result = compute_performance_series(df)
        assert isinstance(result, pl.Series)
        assert len(result) == len(df)
