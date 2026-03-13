"""Tests étendus pour src/analysis/stats.py.

Couvre compute_aggregated_stats, extract_mode_category,
compute_mode_category_averages — non couverts par test_analysis.py.
"""

import polars as pl

from src.analysis.stats import (
    compute_aggregated_stats,
    compute_mode_category_averages,
    extract_mode_category,
)
from src.data.domain.models.stats import AggregatedStats


class TestComputeAggregatedStats:
    """Tests pour compute_aggregated_stats."""

    def test_normal_values(self, sample_match_df_polars: pl.DataFrame) -> None:
        result = compute_aggregated_stats(sample_match_df_polars)
        assert isinstance(result, AggregatedStats)
        assert result.total_matches == 20
        assert result.total_kills > 0
        assert result.total_deaths > 0
        assert result.total_assists > 0
        assert result.total_time_seconds > 0

    def test_empty_dataframe(self, empty_df_polars: pl.DataFrame) -> None:
        result = compute_aggregated_stats(empty_df_polars)
        assert result == AggregatedStats()

    def test_single_match(self) -> None:
        df = pl.DataFrame(
            {
                "kills": [10],
                "deaths": [5],
                "assists": [3],
                "time_played_seconds": [600],
            }
        )
        result = compute_aggregated_stats(df)
        assert result.total_kills == 10
        assert result.total_deaths == 5
        assert result.total_assists == 3
        assert result.total_matches == 1
        assert result.total_time_seconds == 600.0

    def test_missing_time_column(self) -> None:
        df = pl.DataFrame({"kills": [10], "deaths": [5], "assists": [3]})
        result = compute_aggregated_stats(df)
        assert result.total_time_seconds == 0.0


class TestExtractModeCategory:
    """Tests pour extract_mode_category."""

    def test_slayer(self) -> None:
        cat = extract_mode_category("Arena:Slayer on Aquarius")
        assert isinstance(cat, str)
        assert cat != ""

    def test_none(self) -> None:
        cat = extract_mode_category(None)
        assert isinstance(cat, str)

    def test_empty(self) -> None:
        cat = extract_mode_category("")
        assert isinstance(cat, str)

    def test_firefight(self) -> None:
        cat = extract_mode_category("Firefight:King of the Hill")
        assert cat.lower() in ("firefight", "pve", "other")

    def test_btb(self) -> None:
        cat = extract_mode_category("BTB:Slayer on Fragmentation")
        assert isinstance(cat, str)


class TestComputeModeCategoryAverages:
    """Tests pour compute_mode_category_averages."""

    def test_empty_dataframe(self) -> None:
        df = pl.DataFrame(
            schema={
                "pair_name": pl.Utf8,
                "kills": pl.Int64,
                "deaths": pl.Int64,
                "assists": pl.Int64,
            }
        )
        result = compute_mode_category_averages(df, "Slayer")
        assert result["match_count"] == 0
        assert result["avg_kills"] is None

    def test_returns_dict_keys(self, sample_match_df_polars: pl.DataFrame) -> None:
        df = sample_match_df_polars.with_columns(
            pl.lit("Arena:Slayer on Recharge").alias("pair_name")
        )
        result = compute_mode_category_averages(df, "Other")
        assert "avg_kills" in result
        assert "avg_deaths" in result
        assert "avg_assists" in result
        assert "avg_ratio" in result
        assert "match_count" in result
