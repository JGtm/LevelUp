"""Tests unitaires — score de performance d'escouade."""

from __future__ import annotations

import pytest

from src.analysis._performance_squad import compute_squad_performance_score
from src.analysis.performance_config import SQUAD_GRADE_THRESHOLDS, resolve_squad_grade

# =============================================================================
# compute_squad_performance_score
# =============================================================================


class TestComputeSquadPerformanceScore:
    def test_empty_list_returns_none(self) -> None:
        result = compute_squad_performance_score([])
        assert result["score"] is None
        assert result["grade"] == "N/A"

    def test_all_none_scores_returns_none(self) -> None:
        result = compute_squad_performance_score([{"score": None}, {"score": None}])
        assert result["score"] is None

    def test_single_player(self) -> None:
        result = compute_squad_performance_score([{"score": 70.0, "kd_ratio": 1.5, "kills": 10}])
        assert result["score"] is not None
        assert 0 <= result["score"] <= 100

    def test_score_is_average_of_individuals(self) -> None:
        scores = [{"score": 60.0}, {"score": 80.0}]
        result = compute_squad_performance_score(scores)
        # Base = 70, bonus éventuels ajoutés → score >= 70
        assert result["score"] is not None
        assert result["score"] >= 70.0

    def test_score_capped_at_100(self) -> None:
        scores = [
            {"score": 99.0, "win_rate": 90.0, "kd_ratio": 3.0, "kills": 8},
            {"score": 99.0, "win_rate": 90.0, "kd_ratio": 3.0, "kills": 9},
        ]
        result = compute_squad_performance_score(scores)
        assert result["score"] <= 100.0

    def test_score_not_negative(self) -> None:
        result = compute_squad_performance_score([{"score": 0.0}])
        assert result["score"] >= 0.0

    def test_grade_key_present(self) -> None:
        result = compute_squad_performance_score([{"score": 55.0}])
        assert "grade" in result
        assert result["grade"] != "N/A"

    def test_components_contain_base_avg(self) -> None:
        result = compute_squad_performance_score([{"score": 50.0}, {"score": 70.0}])
        assert "base_avg" in result["components"]
        assert result["components"]["base_avg"] == pytest.approx(60.0, abs=0.1)

    def test_bonus_winrate_applied(self) -> None:
        """Win rate > 60 % → +5 points de bonus."""
        base = [{"score": 50.0, "win_rate": 80.0}]
        result_with = compute_squad_performance_score(base)
        base_no_win = [{"score": 50.0}]
        result_without = compute_squad_performance_score(base_no_win)
        assert result_with["score"] > result_without["score"]  # type: ignore[operator]

    def test_bonus_cohesion_applied(self) -> None:
        """Tous les joueurs ont K/D > 1 → +5 points."""
        high_kd = [{"score": 60.0, "kd_ratio": 2.0}, {"score": 60.0, "kd_ratio": 1.5}]
        low_kd = [{"score": 60.0, "kd_ratio": 0.5}, {"score": 60.0, "kd_ratio": 2.0}]
        result_high = compute_squad_performance_score(high_kd)
        result_low = compute_squad_performance_score(low_kd)
        assert result_high["score"] > result_low["score"]  # type: ignore[operator]


# =============================================================================
# resolve_squad_grade
# =============================================================================


class TestResolveSquadGrade:
    def test_none_score_returns_na(self) -> None:
        assert resolve_squad_grade(None) == "N/A"

    @pytest.mark.parametrize(
        "score,expected_fragment",
        [
            (95.0, "LEGENDAIRE"),
            (80.0, "CARNAGE"),
            (65.0, "SOLIDE"),
            (50.0, "MOYEN"),
            (35.0, "DIFFICILE"),
            (10.0, "CATASTROPHIQUE"),
        ],
    )
    def test_grade_thresholds(self, score: float, expected_fragment: str) -> None:
        grade = resolve_squad_grade(score, lang="fr")
        assert expected_fragment in grade.upper()

    def test_thresholds_are_sorted_descending(self) -> None:
        """SQUAD_GRADE_THRESHOLDS doit être en ordre décroissant de seuil."""
        thresholds = [t for t, _ in SQUAD_GRADE_THRESHOLDS]
        assert thresholds == sorted(thresholds, reverse=True)
