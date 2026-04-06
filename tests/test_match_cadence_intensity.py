"""Tests pour les modules de cadence et d'intensité de match."""

from __future__ import annotations

import polars as pl
import pytest

from src.analysis.match_cadence import (
    CadenceBucket,
    compute_cadence_buckets,
    compute_cadence_moving_avg,
)
from src.analysis.match_intensity import (
    compute_match_intensity_profiles,
    compute_squad_cadence_profiles,
)


# =============================================================================
# Tests compute_cadence_buckets
# =============================================================================


def _make_kill_event(time_ms: int, xuid: str) -> dict:
    return {"event_type": "kill", "time_ms": time_ms, "xuid": xuid}


class TestComputeCadenceBuckets:
    def test_empty_events(self):
        result = compute_cadence_buckets([], {}, 1, 120.0)
        assert len(result) == 4  # empty buckets still created for the duration
        assert all(b.total == 0 for b in result)

    def test_zero_duration(self):
        result = compute_cadence_buckets([_make_kill_event(1000, "a")], {"a": 1}, 1, 0.0)
        assert result == []

    def test_basic_bucketing(self):
        xuid_to_team = {"a": 1, "b": 2}
        events = [
            _make_kill_event(5_000, "a"),   # 5s → bucket 0 (0-30)
            _make_kill_event(10_000, "b"),  # 10s → bucket 0
            _make_kill_event(35_000, "a"),  # 35s → bucket 1 (30-60)
            _make_kill_event(65_000, "b"),  # 65s → bucket 2 (60-90)
        ]
        buckets = compute_cadence_buckets(events, xuid_to_team, 1, 90.0, bucket_s=30)

        assert len(buckets) == 3
        assert buckets[0].my_kills == 1
        assert buckets[0].enemy_kills == 1
        assert buckets[1].my_kills == 1
        assert buckets[1].enemy_kills == 0
        assert buckets[2].my_kills == 0
        assert buckets[2].enemy_kills == 1

    def test_non_kill_events_ignored(self):
        events = [
            {"event_type": "death", "time_ms": 1000, "xuid": "a"},
            _make_kill_event(2000, "a"),
        ]
        buckets = compute_cadence_buckets(events, {"a": 1}, 1, 30.0, bucket_s=30)
        assert buckets[0].my_kills == 1
        assert buckets[0].total == 1

    def test_unknown_xuid_ignored(self):
        events = [_make_kill_event(1000, "unknown")]
        buckets = compute_cadence_buckets(events, {"a": 1}, 1, 30.0)
        assert buckets[0].total == 0


class TestComputeCadenceMovingAvg:
    def test_empty(self):
        assert compute_cadence_moving_avg([]) == []

    def test_single_bucket(self):
        b = CadenceBucket(0, 30, my_kills=3, enemy_kills=2)
        result = compute_cadence_moving_avg([b], window=3)
        assert len(result) == 1
        assert result[0] == pytest.approx(5.0)

    def test_moving_average(self):
        buckets = [
            CadenceBucket(0, 30, my_kills=2, enemy_kills=0),
            CadenceBucket(30, 60, my_kills=4, enemy_kills=2),
            CadenceBucket(60, 90, my_kills=6, enemy_kills=0),
        ]
        result = compute_cadence_moving_avg(buckets, window=3)
        assert len(result) == 3
        assert result[0] == pytest.approx(2.0)  # [2] / 1
        assert result[1] == pytest.approx(4.0)  # [2, 6] / 2
        assert result[2] == pytest.approx(14 / 3)  # [2, 6, 6] / 3


# =============================================================================
# Tests compute_match_intensity_profiles
# =============================================================================


class TestComputeMatchIntensityProfiles:
    def test_empty_df(self):
        df = pl.DataFrame(schema={"match_id": pl.Utf8, "time_ms": pl.Int64})
        profile = compute_match_intensity_profiles(df, n_buckets=5)
        assert profile.df.is_empty()
        assert profile.n_buckets == 5

    def test_single_match(self):
        df = pl.DataFrame({
            "match_id": ["m1", "m1", "m1", "m1"],
            "time_ms": [0, 2500, 5000, 10000],
        })
        profile = compute_match_intensity_profiles(df, n_buckets=5)
        assert len(profile.df) == 1
        # All phases should have at least some kills
        phase_cols = [f"phase_{i}" for i in range(5)]
        total = sum(profile.df[col][0] for col in phase_cols)
        assert total == 4

    def test_multiple_matches(self):
        df = pl.DataFrame({
            "match_id": ["m1", "m1", "m2", "m2"],
            "time_ms": [1000, 9000, 500, 4500],
        })
        profile = compute_match_intensity_profiles(df, n_buckets=10)
        assert len(profile.df) == 2
        assert profile.n_buckets == 10


# =============================================================================
# Tests compute_squad_cadence_profiles
# =============================================================================


class TestComputeSquadCadenceProfiles:
    def test_empty(self):
        df = pl.DataFrame(schema={"match_id": pl.Utf8, "time_ms": pl.Int64, "xuid": pl.Utf8})
        result = compute_squad_cadence_profiles(df, {"a": "PlayerA"})
        assert result.is_empty()

    def test_single_player(self):
        df = pl.DataFrame({
            "match_id": ["m1", "m1", "m1"],
            "time_ms": [0, 5000, 10000],
            "xuid": ["a", "a", "a"],
        })
        result = compute_squad_cadence_profiles(df, {"a": "PlayerA"}, n_buckets=5)
        assert "phase" in result.columns
        assert "PlayerA" in result.columns
        assert len(result) == 5

    def test_two_players(self):
        df = pl.DataFrame({
            "match_id": ["m1", "m1", "m1", "m1"],
            "time_ms": [0, 5000, 2500, 7500],
            "xuid": ["a", "a", "b", "b"],
        })
        result = compute_squad_cadence_profiles(
            df, {"a": "Alice", "b": "Bob"}, n_buckets=5,
        )
        assert "Alice" in result.columns
        assert "Bob" in result.columns
        assert len(result) == 5
