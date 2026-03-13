"""Tests pour src/analysis/_composite.py.

Couvre compute_composite_score, _sigmoid_ratio et les composantes.
"""

from src.analysis._composite import (
    _sigmoid_ratio,
    compute_composite_score,
)


class TestSigmoidRatio:
    """Tests pour _sigmoid_ratio."""

    def test_equal_ratio_returns_half(self) -> None:
        assert abs(_sigmoid_ratio(10.0, 10.0) - 0.5) < 1e-6

    def test_high_ratio_above_half(self) -> None:
        result = _sigmoid_ratio(20.0, 10.0)
        assert result > 0.5

    def test_low_ratio_below_half(self) -> None:
        result = _sigmoid_ratio(5.0, 10.0)
        assert result < 0.5

    def test_zero_denominator(self) -> None:
        assert _sigmoid_ratio(10.0, 0.0) == 0.5

    def test_negative_denominator(self) -> None:
        assert _sigmoid_ratio(10.0, -5.0) == 0.5

    def test_bounded_zero_one(self) -> None:
        assert 0.0 <= _sigmoid_ratio(1000.0, 1.0) <= 1.0
        assert 0.0 <= _sigmoid_ratio(1.0, 1000.0) <= 1.0


class TestComputeCompositeScore:
    """Tests pour compute_composite_score."""

    def test_no_data_returns_half(self) -> None:
        result = compute_composite_score({}, None, None, None)
        assert result == 0.5

    def test_win_outcome(self) -> None:
        row = {"outcome": 2, "kills": 10, "deaths": 5, "kills_expected": 8, "deaths_expected": 6}
        result = compute_composite_score(row, None, None, None)
        assert 0.0 <= result <= 1.0
        # Win should push above 0.5
        assert result > 0.5

    def test_loss_outcome(self) -> None:
        row = {"outcome": 3, "kills": 3, "deaths": 10, "kills_expected": 8, "deaths_expected": 6}
        result = compute_composite_score(row, None, None, None)
        assert 0.0 <= result <= 1.0
        # Loss with bad stats should be below 0.5
        assert result < 0.5

    def test_with_accuracy(self) -> None:
        row = {"outcome": 2, "kills": 10, "deaths": 5, "kills_expected": 8, "deaths_expected": 6}
        result = compute_composite_score(
            row, avg_accuracy=40.0, teammate_avg_ke=None, enemy_avg_ke=None
        )
        assert 0.0 <= result <= 1.0

    def test_with_damage_efficiency(self) -> None:
        row = {
            "outcome": 2,
            "kills": 10,
            "deaths": 5,
            "kills_expected": 8,
            "deaths_expected": 6,
            "damage_dealt": 5000,
            "damage_taken": 3000,
        }
        result = compute_composite_score(
            row, avg_accuracy=None, teammate_avg_ke=None, enemy_avg_ke=None, avg_damage_eff=0.6
        )
        assert 0.0 <= result <= 1.0

    def test_bounded(self) -> None:
        row = {
            "outcome": 2,
            "kills": 100,
            "deaths": 0,
            "kills_expected": 10,
            "deaths_expected": 10,
            "accuracy": 95.0,
            "damage_dealt": 99999,
            "damage_taken": 100,
        }
        result = compute_composite_score(row, 50.0, 10.0, 10.0, 0.5)
        assert 0.0 <= result <= 1.0

    def test_custom_weights(self) -> None:
        weights = {
            "kills_vs_expected": 1.0,
            "deaths_vs_expected": 0.0,
            "win_factor": 0.0,
            "damage_efficiency": 0.0,
            "accuracy_delta": 0.0,
        }
        row = {"kills": 20, "deaths": 5, "kills_expected": 10, "deaths_expected": 5}
        result = compute_composite_score(row, None, None, None, weights=weights)
        assert 0.0 <= result <= 1.0
