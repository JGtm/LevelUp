"""Tests pour src/analysis/_performance_session.py.

Couvre compute_session_performance_score_v1, v2, et les helpers.
"""

from __future__ import annotations

from datetime import datetime, timedelta

import polars as pl

from src.analysis._performance_session import (
    _compute_accuracy_component,
    _compute_kd_component,
    _compute_kpm_component,
    _compute_life_component,
    _compute_mmr_performance_component,
    _compute_objective_component,
    _compute_win_component,
    _count_wins,
    _mean_numeric,
    _mmr_difficulty_multiplier,
    _saturation_score,
    _sum_int,
    _weighted_score,
    compute_session_performance_score_v1,
    compute_session_performance_score_v2,
)


class TestHelpers:
    """Tests pour les fonctions helper internes."""

    def test_mean_numeric_normal(self) -> None:
        df = pl.DataFrame({"val": [10, 20, 30]})
        assert _mean_numeric(df, "val") == 20.0

    def test_mean_numeric_missing_column(self) -> None:
        df = pl.DataFrame({"other": [1, 2]})
        assert _mean_numeric(df, "val") is None

    def test_mean_numeric_all_null(self) -> None:
        df = pl.DataFrame({"val": [None, None, None]}, schema={"val": pl.Float64})
        assert _mean_numeric(df, "val") is None

    def test_sum_int_normal(self) -> None:
        df = pl.DataFrame({"val": [10, 20, 30]})
        assert _sum_int(df, "val") == 60

    def test_sum_int_missing_column(self) -> None:
        df = pl.DataFrame({"other": [1]})
        assert _sum_int(df, "val") == 0

    def test_count_wins(self) -> None:
        df = pl.DataFrame({"outcome": [2, 3, 2, 1, 4]})
        assert _count_wins(df) == 2

    def test_count_wins_no_outcome_column(self) -> None:
        df = pl.DataFrame({"other": [1, 2]})
        assert _count_wins(df) == 0

    def test_saturation_score_zero(self) -> None:
        assert _saturation_score(0.0, 1.0) == 0.0

    def test_saturation_score_negative_scale(self) -> None:
        assert _saturation_score(5.0, -1.0) == 0.0

    def test_saturation_score_bounded(self) -> None:
        result = _saturation_score(1000.0, 1.0)
        assert 0.0 <= result <= 100.0


def _make_session_df(n: int = 8) -> pl.DataFrame:
    """Crée un DataFrame de session test."""
    start = datetime(2025, 1, 1)
    return pl.DataFrame(
        {
            "match_id": [f"m{i}" for i in range(n)],
            "start_time": [start + timedelta(hours=i) for i in range(n)],
            "kills": [10 + i for i in range(n)],
            "deaths": [5 + i % 3 for i in range(n)],
            "assists": [3 + i % 4 for i in range(n)],
            "outcome": [2, 3, 2, 1, 2, 3, 2, 3][:n],
            "accuracy": [45.0 + i for i in range(n)],
            "kills_per_min": [0.8 + i * 0.05 for i in range(n)],
            "average_life_seconds": [30.0 + i * 2 for i in range(n)],
        }
    )


class TestSessionPerformanceScoreV1:
    """Tests pour compute_session_performance_score_v1."""

    def test_normal_session(self) -> None:
        df = _make_session_df()
        result = compute_session_performance_score_v1(df)
        assert result["score"] is not None
        assert 0 <= result["score"] <= 100
        assert result["matches"] == 8
        assert result["kills"] > 0
        assert result["deaths"] > 0

    def test_empty_session(self) -> None:
        df = pl.DataFrame(
            schema={
                "kills": pl.Int64,
                "deaths": pl.Int64,
                "assists": pl.Int64,
                "outcome": pl.Int64,
            }
        )
        result = compute_session_performance_score_v1(df)
        assert result["score"] is None
        assert result["matches"] == 0

    def test_all_wins_high_score(self) -> None:
        df = pl.DataFrame(
            {
                "kills": [20, 25, 18],
                "deaths": [3, 2, 4],
                "assists": [5, 8, 6],
                "outcome": [2, 2, 2],
                "accuracy": [55.0, 60.0, 52.0],
            }
        )
        result = compute_session_performance_score_v1(df)
        assert result["score"] > 50.0
        assert result["win_rate"] == 100.0

    def test_result_keys(self) -> None:
        df = _make_session_df()
        result = compute_session_performance_score_v1(df)
        expected_keys = {
            "score",
            "kd_ratio",
            "efficiency",
            "win_rate",
            "accuracy",
            "avg_score",
            "avg_life_seconds",
            "matches",
            "kills",
            "deaths",
            "assists",
            "team_mmr_avg",
            "enemy_mmr_avg",
            "delta_mmr_avg",
        }
        assert expected_keys.issubset(result.keys())


class TestSessionPerformanceScoreV2:
    """Tests pour compute_session_performance_score_v2."""

    def test_normal_session(self) -> None:
        df = _make_session_df()
        result = compute_session_performance_score_v2(df)
        assert result["score"] is not None
        assert 0 <= result["score"] <= 100
        assert "components" in result
        assert "confidence" in result
        assert result["version"] == "v2"

    def test_empty_session(self) -> None:
        df = pl.DataFrame(
            schema={
                "kills": pl.Int64,
                "deaths": pl.Int64,
                "assists": pl.Int64,
                "outcome": pl.Int64,
            }
        )
        result = compute_session_performance_score_v2(df)
        assert result["score"] is None
        assert result["confidence"] == 0.0

    def test_confidence_increases_with_matches(self) -> None:
        small = _make_session_df(3)
        large = _make_session_df(8)
        r_small = compute_session_performance_score_v2(small)
        r_large = compute_session_performance_score_v2(large)
        assert r_large["confidence"] >= r_small["confidence"]

    def test_confidence_label(self) -> None:
        df3 = _make_session_df(3)
        df5 = _make_session_df(5)
        assert compute_session_performance_score_v2(df3)["confidence_label"] == "faible"
        assert compute_session_performance_score_v2(df5)["confidence_label"] == "moyenne"


class TestComponents:
    """Tests pour les composantes internes de score (branches non couvertes)."""

    def test_kd_component_both_zero(self) -> None:
        """kills==0 ET deaths==0 → score None."""
        df = pl.DataFrame({"kills": [0], "deaths": [0], "assists": [0]})
        score, meta = _compute_kd_component(df)
        assert score is None
        assert meta["kd_ratio"] is None

    def test_kd_component_no_deaths(self) -> None:
        """deaths==0 mais kills>0 → kd_ratio = float(kills)."""
        df = pl.DataFrame({"kills": [5], "deaths": [0], "assists": [2]})
        score, meta = _compute_kd_component(df)
        assert score is not None
        assert meta["kd_ratio"] == 5.0

    def test_win_component_no_outcome_column(self) -> None:
        """Pas de colonne outcome → None."""
        df = pl.DataFrame({"kills": [5, 10], "deaths": [3, 4]})
        score, meta = _compute_win_component(df)
        assert score is None
        assert meta["win_rate"] is None

    def test_win_component_empty_df_with_outcome(self) -> None:
        """DataFrame vide (avec colonne outcome) → None."""
        df = pl.DataFrame(schema={"kills": pl.Int64, "deaths": pl.Int64, "outcome": pl.Int64})
        score, meta = _compute_win_component(df)
        assert score is None
        assert meta["win_rate"] is None

    def test_objective_component_no_obj_columns(self) -> None:
        """Aucune colonne objectif → None."""
        df = pl.DataFrame({"kills": [5, 10], "deaths": [3, 4]})
        score, meta = _compute_objective_component(df)
        assert score is None
        assert meta["objective_score"] is None
        assert meta["objective_columns"] == []

    def test_mmr_component_mmr_present_no_outcome(self) -> None:
        """MMR présent, pas de colonne outcome → score None, expected_win_rate renseignée."""
        df = pl.DataFrame(
            {
                "team_mmr": [1500.0, 1500.0],
                "enemy_mmr": [1500.0, 1500.0],
                "kills": [5, 8],
                "deaths": [3, 4],
            }
        )
        score, meta = _compute_mmr_performance_component(df)
        assert score is None
        assert meta["expected_win_rate"] is not None
        assert meta["actual_win_rate"] is None

    def test_weighted_score_no_components(self) -> None:
        """Aucune composante dispos → total_weight==0 → None."""
        result = _weighted_score({}, {}, {}, False)
        assert result is None

    def test_accuracy_component_no_column(self) -> None:
        """Pas de colonne accuracy → None."""
        df = pl.DataFrame({"kills": [5], "deaths": [3]})
        score, meta = _compute_accuracy_component(df)
        assert score is None
        assert meta["accuracy"] is None

    def test_kpm_component_no_column(self) -> None:
        """Pas de colonne kills_per_min → None."""
        df = pl.DataFrame({"kills": [5], "deaths": [3]})
        score, meta = _compute_kpm_component(df)
        assert score is None
        assert meta["kills_per_min"] is None

    def test_life_component_no_column(self) -> None:
        """Pas de colonne average_life_seconds → None."""
        df = pl.DataFrame({"kills": [5], "deaths": [3]})
        score, meta = _compute_life_component(df)
        assert score is None
        assert meta["avg_life_seconds"] is None

    def test_objective_component_with_data(self) -> None:
        """Colonnes objectif présentes avec valeurs > 0 → score calculé."""
        df = pl.DataFrame({"flag_captures": [2, 1, 3], "zones_captured": [1, 2, 0]})
        score, meta = _compute_objective_component(df)
        assert score is not None
        assert meta["objective_score"] is not None
        assert len(meta["objective_columns"]) > 0

    def test_mmr_component_with_outcome(self) -> None:
        """MMR présent + outcome → score calculé avec actual_win_rate."""
        df = pl.DataFrame(
            {
                "team_mmr": [1500.0, 1600.0, 1550.0],
                "enemy_mmr": [1450.0, 1400.0, 1500.0],
                "kills": [10, 8, 12],
                "deaths": [5, 6, 4],
                "outcome": [2, 2, 3],  # 2 victoires, 1 défaite
            }
        )
        score, meta = _compute_mmr_performance_component(df)
        assert score is not None
        assert meta["actual_win_rate"] is not None
        assert meta["expected_win_rate"] is not None

    def test_mmr_difficulty_multiplier_with_delta(self) -> None:
        """delta non None : équipe plus forte (delta>0) → malus, plus faible (delta<0) → bonus."""
        # team_mmr - enemy_mmr = 200 : équipe plus forte → score légèrement réduit
        result_stronger = _mmr_difficulty_multiplier(200.0)
        # team_mmr - enemy_mmr = -200 : équipe plus faible → score légèrement augmenté
        result_weaker = _mmr_difficulty_multiplier(-200.0)
        assert result_stronger != 1.0
        assert result_weaker != 1.0
        assert result_stronger < 1.0
        assert result_weaker > 1.0

    def test_objective_component_empty_values(self) -> None:
        """Colonne objectif présente mais toutes nulles → skip, retour None."""
        df = pl.DataFrame(
            {"flag_captures": [None, None, None]}, schema={"flag_captures": pl.Float64}
        )
        score, meta = _compute_objective_component(df)
        assert score is None

    def test_objective_component_zero_values(self) -> None:
        """Colonne objectif avec mean <= 0 (toutes à 0) → skip, retour None."""
        df = pl.DataFrame({"flag_captures": [0, 0, 0]})
        score, meta = _compute_objective_component(df)
        assert score is None
