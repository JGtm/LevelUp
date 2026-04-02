"""Tests unitaires pour src/analysis/squad_records.py."""

from __future__ import annotations

import polars as pl
import pytest

from src.analysis.squad_records import (
    compute_player_record,
    compute_squad_records,
    get_dominant_pair_name,
)


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


def _make_df(rows: list[dict]) -> pl.DataFrame:
    return pl.DataFrame(rows)


# ---------------------------------------------------------------------------
# get_dominant_pair_name
# ---------------------------------------------------------------------------


class TestGetDominantPairName:
    def test_returns_most_frequent(self):
        df1 = _make_df([{"pair_name": "Arena:4v4"}, {"pair_name": "Arena:4v4"}, {"pair_name": "BTB"}])
        df2 = _make_df([{"pair_name": "Arena:4v4"}, {"pair_name": "BTB"}])
        result = get_dominant_pair_name([df1, df2])
        assert result == "Arena:4v4"

    def test_single_df(self):
        df = _make_df([{"pair_name": "BTB"}, {"pair_name": "BTB"}])
        assert get_dominant_pair_name([df]) == "BTB"

    def test_empty_list(self):
        assert get_dominant_pair_name([]) is None

    def test_all_none_pair_name(self):
        df = _make_df([{"pair_name": None}, {"pair_name": None}])
        assert get_dominant_pair_name([df]) is None

    def test_no_pair_name_column(self):
        df = _make_df([{"kills": 10}, {"kills": 20}])
        assert get_dominant_pair_name([df]) is None

    def test_empty_dataframe(self):
        df = pl.DataFrame(schema={"pair_name": pl.String})
        assert get_dominant_pair_name([df]) is None

    def test_mixed_none_and_values(self):
        df = _make_df([{"pair_name": None}, {"pair_name": "Ranked"}, {"pair_name": "Ranked"}])
        assert get_dominant_pair_name([df]) == "Ranked"

    def test_multiple_dfs_aggregated(self):
        df1 = _make_df([{"pair_name": "BTB"}, {"pair_name": "BTB"}, {"pair_name": "BTB"}])
        df2 = _make_df([{"pair_name": "Arena:4v4"}, {"pair_name": "Arena:4v4"}])
        # BTB: 3, Arena:4v4: 2 → BTB wins
        assert get_dominant_pair_name([df1, df2]) == "BTB"


# ---------------------------------------------------------------------------
# compute_player_record
# ---------------------------------------------------------------------------


class TestComputePlayerRecord:
    def test_max_basic(self):
        df = _make_df([
            {"kills": 10, "pair_name": "Arena:4v4"},
            {"kills": 20, "pair_name": "Arena:4v4"},
            {"kills": 15, "pair_name": "Arena:4v4"},
        ])
        assert compute_player_record(df, "kills", "Arena:4v4") == pytest.approx(20.0)

    def test_min_for_negative(self):
        df = _make_df([
            {"deaths": 3, "pair_name": "Arena:4v4"},
            {"deaths": 1, "pair_name": "Arena:4v4"},
            {"deaths": 5, "pair_name": "Arena:4v4"},
        ])
        assert compute_player_record(df, "deaths", "Arena:4v4", is_negative=True) == pytest.approx(1.0)

    def test_pair_name_filter_excludes_other_modes(self):
        df = _make_df([
            {"kills": 30, "pair_name": "BTB"},
            {"kills": 10, "pair_name": "Arena:4v4"},
            {"kills": 12, "pair_name": "Arena:4v4"},
        ])
        # Filtre Arena:4v4 → ne doit pas voir le 30 du BTB
        assert compute_player_record(df, "kills", "Arena:4v4") == pytest.approx(12.0)

    def test_pair_name_none_uses_all_rows(self):
        df = _make_df([
            {"kills": 30, "pair_name": "BTB"},
            {"kills": 10, "pair_name": "Arena:4v4"},
        ])
        assert compute_player_record(df, "kills", None) == pytest.approx(30.0)

    def test_returns_none_if_df_empty(self):
        df = pl.DataFrame(schema={"kills": pl.Int64, "pair_name": pl.String})
        assert compute_player_record(df, "kills", "Arena:4v4") is None

    def test_returns_none_if_metric_absent(self):
        df = _make_df([{"pair_name": "Arena:4v4", "deaths": 5}])
        assert compute_player_record(df, "kills", "Arena:4v4") is None

    def test_returns_none_if_no_row_for_pair_name(self):
        df = _make_df([{"kills": 10, "pair_name": "BTB"}])
        assert compute_player_record(df, "kills", "Arena:4v4") is None

    def test_returns_none_for_all_null_values(self):
        df = _make_df([{"kills": None, "pair_name": "Arena:4v4"}])
        assert compute_player_record(df, "kills", "Arena:4v4") is None

    def test_no_pair_name_column_uses_all_rows(self):
        df = _make_df([{"kills": 5}, {"kills": 15}])
        # Pas de colonne pair_name → filtre ignoré, utilise tout
        assert compute_player_record(df, "kills", "Arena:4v4") == pytest.approx(15.0)

    def test_average_life_seconds_is_max(self):
        """avg_life = durée de vie : plus c'est long, mieux c'est → is_negative=False."""
        df = _make_df([
            {"average_life_seconds": 45.0, "pair_name": "Arena:4v4"},
            {"average_life_seconds": 120.0, "pair_name": "Arena:4v4"},
            {"average_life_seconds": 30.0, "pair_name": "Arena:4v4"},
        ])
        assert compute_player_record(df, "average_life_seconds", "Arena:4v4") == pytest.approx(120.0)

    def test_result_is_float(self):
        df = _make_df([{"kills": 7, "pair_name": "BTB"}])
        result = compute_player_record(df, "kills", "BTB")
        assert isinstance(result, float)


# ---------------------------------------------------------------------------
# compute_squad_records
# ---------------------------------------------------------------------------


class TestComputeSquadRecords:
    def test_basic_two_players(self):
        df_p1 = _make_df([
            {"kills": 10, "deaths": 3, "pair_name": "Arena:4v4"},
            {"kills": 20, "deaths": 5, "pair_name": "Arena:4v4"},
        ])
        df_p2 = _make_df([
            {"kills": 8, "deaths": 2, "pair_name": "Arena:4v4"},
            {"kills": 14, "deaths": 1, "pair_name": "Arena:4v4"},
        ])
        records = compute_squad_records(
            [("Alice", df_p1), ("Bob", df_p2)],
            [("kills", False), ("deaths", True)],
            "Arena:4v4",
        )
        assert records["Alice"]["kills"] == pytest.approx(20.0)
        assert records["Alice"]["deaths"] == pytest.approx(3.0)
        assert records["Bob"]["kills"] == pytest.approx(14.0)
        assert records["Bob"]["deaths"] == pytest.approx(1.0)

    def test_returns_none_for_missing_metric(self):
        df = _make_df([{"kills": 10, "pair_name": "BTB"}])
        records = compute_squad_records([("Player", df)], [("headshot_kills", False)], "BTB")
        assert records["Player"]["headshot_kills"] is None

    def test_empty_players_list(self):
        records = compute_squad_records([], [("kills", False)], "BTB")
        assert records == {}

    def test_dominant_pair_none(self):
        df = _make_df([{"kills": 5, "pair_name": "BTB"}, {"kills": 15, "pair_name": "Arena:4v4"}])
        # dominant_pair=None → pas de filtre → utilise tous les matchs
        records = compute_squad_records([("Player", df)], [("kills", False)], None)
        assert records["Player"]["kills"] == pytest.approx(15.0)

    def test_structure_keys(self):
        df = _make_df([{"kills": 5, "assists": 3, "pair_name": "BTB"}])
        records = compute_squad_records(
            [("X", df)], [("kills", False), ("assists", False)], "BTB"
        )
        assert set(records["X"].keys()) == {"kills", "assists"}
