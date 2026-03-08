"""Tests pour src/analysis/cumulative_progression.py.

Couvre :
- compute_ewma_kd_polars : lissage, alpha, cas limites
- compute_cumulative_kd_with_ci : IC 90 %, convergence, zéro mort
- compute_linear_regression_kd : OLS, tendances, win rate, insuffisance
- compute_rolling_net_score_per_hour : colonne absente, fenêtre adaptative
- _ols / _empty_regression : fonctions internes
"""

from __future__ import annotations

from datetime import datetime, timedelta

import polars as pl
import pytest

# =============================================================================
# Fixtures
# =============================================================================


@pytest.fixture
def base_df() -> pl.DataFrame:
    """DataFrame de 10 matchs avec kills/deaths/outcome."""
    start = datetime(2026, 1, 1, 10, 0)
    return pl.DataFrame(
        {
            "match_id": [f"m{i}" for i in range(10)],
            "start_time": [start + timedelta(hours=i) for i in range(10)],
            "kills": [8, 10, 12, 7, 9, 11, 14, 6, 10, 13],
            "deaths": [6, 8, 5, 9, 7, 6, 4, 10, 7, 5],
            "time_played_seconds": [600, 720, 580, 810, 660, 570, 500, 920, 640, 520],
            "outcome": [2, 3, 2, 3, 2, 2, 2, 3, 2, 2],  # 2=WIN, 3=LOSS
        }
    )


@pytest.fixture
def improving_df() -> pl.DataFrame:
    """Session avec K/D nettement croissant."""
    start = datetime(2026, 1, 1, 10, 0)
    return pl.DataFrame(
        {
            "match_id": [f"m{i}" for i in range(15)],
            "start_time": [start + timedelta(hours=i) for i in range(15)],
            "kills": [3, 4, 5, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16],
            "deaths": [10, 10, 9, 9, 8, 8, 7, 7, 6, 5, 5, 4, 4, 3, 3],
            "outcome": [3, 3, 3, 3, 3, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2],
        }
    )


@pytest.fixture
def empty_df() -> pl.DataFrame:
    """DataFrame vide avec le bon schéma."""
    return pl.DataFrame(
        schema={
            "match_id": pl.Utf8,
            "start_time": pl.Utf8,
            "kills": pl.Int64,
            "deaths": pl.Int64,
            "outcome": pl.Int64,
        }
    )


# =============================================================================
# compute_ewma_kd_polars
# =============================================================================


class TestComputeEwmaKd:
    """Tests pour compute_ewma_kd_polars."""

    def test_retourne_colonnes_requises(self, base_df: pl.DataFrame) -> None:
        from src.analysis.cumulative_progression import compute_ewma_kd_polars

        result = compute_ewma_kd_polars(base_df)
        assert "kd" in result.columns
        assert "ewma_kd" in result.columns
        assert "start_time" in result.columns

    def test_preserve_colonne_outcome(self, base_df: pl.DataFrame) -> None:
        from src.analysis.cumulative_progression import compute_ewma_kd_polars

        result = compute_ewma_kd_polars(base_df)
        assert "outcome" in result.columns

    def test_ewma_premier_match_egal_kd(self, base_df: pl.DataFrame) -> None:
        """Pour le premier match, EWMA(adjust=True) = kd."""
        from src.analysis.cumulative_progression import compute_ewma_kd_polars

        result = compute_ewma_kd_polars(base_df)
        first_kd = result["kd"][0]
        first_ewma = result["ewma_kd"][0]
        assert abs(first_ewma - first_kd) < 1e-6

    def test_ewma_lisse_les_valeurs(self, base_df: pl.DataFrame) -> None:
        """EWMA doit être entre min et max des K/D."""
        from src.analysis.cumulative_progression import compute_ewma_kd_polars

        result = compute_ewma_kd_polars(base_df)
        kd_min = result["kd"].min()
        kd_max = result["kd"].max()
        ewma_vals = result["ewma_kd"].to_list()
        for v in ewma_vals:
            assert kd_min - 0.01 <= v <= kd_max + 0.01

    def test_alpha_eleve_plus_reactif(self, base_df: pl.DataFrame) -> None:
        """Alpha élevé produit une série plus variable que alpha faible."""
        from src.analysis.cumulative_progression import compute_ewma_kd_polars

        r_low = compute_ewma_kd_polars(base_df, alpha=0.05)
        r_high = compute_ewma_kd_polars(base_df, alpha=0.5)
        std_low = r_low["ewma_kd"].std()
        std_high = r_high["ewma_kd"].std()
        assert std_high > std_low

    def test_zero_deaths_traite_comme_un(self, empty_df: pl.DataFrame) -> None:
        """Un match avec 0 décompte → deaths remplacé par 1."""
        from src.analysis.cumulative_progression import compute_ewma_kd_polars

        df = pl.DataFrame(
            {
                "start_time": [datetime(2026, 1, 1)],
                "kills": [10],
                "deaths": [0],
            }
        )
        result = compute_ewma_kd_polars(df)
        assert result["kd"][0] == 10.0

    def test_dataframe_vide(self, empty_df: pl.DataFrame) -> None:
        from src.analysis.cumulative_progression import compute_ewma_kd_polars

        result = compute_ewma_kd_polars(empty_df)
        assert result.is_empty()
        assert "ewma_kd" in result.columns

    def test_un_seul_match(self) -> None:
        from src.analysis.cumulative_progression import compute_ewma_kd_polars

        df = pl.DataFrame({"start_time": [datetime(2026, 1, 1)], "kills": [8], "deaths": [4]})
        result = compute_ewma_kd_polars(df)
        assert len(result) == 1
        assert result["kd"][0] == 2.0
        assert result["ewma_kd"][0] == 2.0


# =============================================================================
# compute_cumulative_kd_with_ci
# =============================================================================


class TestComputeCumulativeKdWithCi:
    """Tests pour compute_cumulative_kd_with_ci."""

    def test_colonnes_requises(self, base_df: pl.DataFrame) -> None:
        from src.analysis.cumulative_progression import compute_cumulative_kd_with_ci

        result = compute_cumulative_kd_with_ci(base_df)
        for col in ("kd", "cumulative_kd", "ci_lower", "ci_upper"):
            assert col in result.columns

    def test_ci_entoure_cumul_kd(self, base_df: pl.DataFrame) -> None:
        """ci_lower ≤ cumulative_kd ≤ ci_upper pour chaque ligne."""
        from src.analysis.cumulative_progression import compute_cumulative_kd_with_ci

        result = compute_cumulative_kd_with_ci(base_df)
        for row in result.to_dicts():
            assert row["ci_lower"] <= row["cumulative_kd"] + 1e-9
            assert row["cumulative_kd"] - 1e-9 <= row["ci_upper"]

    def test_ci_lower_jamais_negatif(self, base_df: pl.DataFrame) -> None:
        from src.analysis.cumulative_progression import compute_cumulative_kd_with_ci

        result = compute_cumulative_kd_with_ci(base_df)
        assert result["ci_lower"].min() >= 0.0

    def test_ci_retrecie_avec_plus_de_matchs(self, base_df: pl.DataFrame) -> None:
        """La largeur de l'IC doit tendre à se rétrécir au fil du temps."""
        from src.analysis.cumulative_progression import compute_cumulative_kd_with_ci

        result = compute_cumulative_kd_with_ci(base_df)
        widths = (result["ci_upper"] - result["ci_lower"]).to_list()
        # La largeur au dernier match < la largeur au 2e match (après 1er = 0)
        # On vérifie que la moyenne des 5 derniers < moyenne des 5 premiers après le 1er
        early = sum(widths[1:6]) / 5
        late = sum(widths[-5:]) / 5
        assert late <= early + 0.5  # tolérance pour petits samples

    def test_z_score_95_bande_plus_large(self, base_df: pl.DataFrame) -> None:
        """z=1.96 (95 %) doit produire une bande plus large que z=1.645 (90 %)."""
        from src.analysis.cumulative_progression import compute_cumulative_kd_with_ci

        r90 = compute_cumulative_kd_with_ci(base_df, z=1.645)
        r95 = compute_cumulative_kd_with_ci(base_df, z=1.96)
        w90 = (r90["ci_upper"] - r90["ci_lower"]).mean()
        w95 = (r95["ci_upper"] - r95["ci_lower"]).mean()
        assert w95 > w90

    def test_zero_deaths(self) -> None:
        from src.analysis.cumulative_progression import compute_cumulative_kd_with_ci

        df = pl.DataFrame(
            {
                "start_time": [datetime(2026, 1, i) for i in range(1, 6)],
                "kills": [10, 8, 12, 9, 11],
                "deaths": [0, 0, 0, 0, 0],
            }
        )
        result = compute_cumulative_kd_with_ci(df)
        assert not result.is_empty()
        assert result["cumulative_kd"].min() > 0

    def test_dataframe_vide(self, empty_df: pl.DataFrame) -> None:
        from src.analysis.cumulative_progression import compute_cumulative_kd_with_ci

        result = compute_cumulative_kd_with_ci(empty_df)
        assert result.is_empty()
        assert "cumulative_kd" in result.columns


# =============================================================================
# compute_linear_regression_kd
# =============================================================================


class TestComputeLinearRegressionKd:
    """Tests pour compute_linear_regression_kd."""

    def test_retourne_toutes_les_cles(self, base_df: pl.DataFrame) -> None:
        from src.analysis.cumulative_progression import (
            compute_ewma_kd_polars,
            compute_linear_regression_kd,
        )

        ewma = compute_ewma_kd_polars(base_df)
        result = compute_linear_regression_kd(ewma)
        for key in (
            "slope",
            "intercept",
            "r_squared",
            "y_hat",
            "is_significant",
            "trend",
            "x_labels",
        ):
            assert key in result

    def test_tendance_improving(self, improving_df: pl.DataFrame) -> None:
        """Session nettement croissante → trend == 'improving'."""
        from src.analysis.cumulative_progression import (
            compute_ewma_kd_polars,
            compute_linear_regression_kd,
        )

        ewma = compute_ewma_kd_polars(improving_df)
        result = compute_linear_regression_kd(ewma)
        assert result["slope"] > 0
        if result["is_significant"]:
            assert result["trend"] == "improving"

    def test_r_squared_entre_0_et_1(self, base_df: pl.DataFrame) -> None:
        from src.analysis.cumulative_progression import (
            compute_ewma_kd_polars,
            compute_linear_regression_kd,
        )

        ewma = compute_ewma_kd_polars(base_df)
        result = compute_linear_regression_kd(ewma)
        assert 0.0 <= result["r_squared"] <= 1.0

    def test_y_hat_meme_longueur(self, base_df: pl.DataFrame) -> None:
        from src.analysis.cumulative_progression import (
            compute_ewma_kd_polars,
            compute_linear_regression_kd,
        )

        ewma = compute_ewma_kd_polars(base_df)
        result = compute_linear_regression_kd(ewma)
        assert len(result["y_hat"]) == len(base_df)

    def test_insuffisance_moins_de_3_matchs(self) -> None:
        """Moins de 3 matchs → renvoie regression vide."""
        from src.analysis.cumulative_progression import compute_linear_regression_kd

        df = pl.DataFrame({"start_time": [datetime(2026, 1, 1)], "ewma_kd": [1.5]})
        result = compute_linear_regression_kd(df)
        assert result["slope"] == 0.0
        assert result["is_significant"] is False
        assert result["y_hat"] == []

    def test_presence_outcome_ajoute_win_rate(self, base_df: pl.DataFrame) -> None:
        """Si outcome présent dans le df, win_rate_slope doit être dans le résultat."""
        from src.analysis.cumulative_progression import (
            compute_ewma_kd_polars,
            compute_linear_regression_kd,
        )

        ewma = compute_ewma_kd_polars(base_df)  # outcome est conservé
        result = compute_linear_regression_kd(ewma)
        assert "win_rate_slope" in result
        assert "win_rate_r2" in result

    def test_sans_outcome_pas_de_win_rate(self) -> None:
        from src.analysis.cumulative_progression import compute_linear_regression_kd

        df = pl.DataFrame(
            {
                "start_time": [datetime(2026, 1, i) for i in range(1, 11)],
                "ewma_kd": [1.0, 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, 1.7, 1.8, 1.9],
            }
        )
        result = compute_linear_regression_kd(df)
        assert "win_rate_slope" not in result

    def test_is_significant_seuil_r2(self) -> None:
        """is_significant = True ssi R² ≥ 0.3."""
        from src.analysis.cumulative_progression import compute_linear_regression_kd

        # Série parfaitement linéaire → R² ≈ 1.0
        df = pl.DataFrame(
            {
                "start_time": [datetime(2026, 1, i) for i in range(1, 11)],
                "ewma_kd": [float(i) for i in range(1, 11)],
            }
        )
        result = compute_linear_regression_kd(df)
        assert result["is_significant"] is True
        assert result["r_squared"] > 0.3

    def test_colonne_inexistante(self) -> None:
        from src.analysis.cumulative_progression import compute_linear_regression_kd

        df = pl.DataFrame({"start_time": [datetime(2026, 1, 1)], "kd": [1.5]})
        result = compute_linear_regression_kd(df, kd_col="ewma_kd")
        assert result["slope"] == 0.0
        assert result["is_significant"] is False


# =============================================================================
# compute_rolling_net_score_per_hour
# =============================================================================


class TestComputeRollingNetScorePerHour:
    """Tests pour compute_rolling_net_score_per_hour."""

    def test_colonnes_requises(self, base_df: pl.DataFrame) -> None:
        from src.analysis.cumulative_progression import compute_rolling_net_score_per_hour

        result = compute_rolling_net_score_per_hour(base_df)
        for col in ("start_time", "net_score", "net_per_hour", "rolling_net_per_hour"):
            assert col in result.columns

    def test_net_score_correct(self, base_df: pl.DataFrame) -> None:
        """net_score = kills - deaths."""
        from src.analysis.cumulative_progression import compute_rolling_net_score_per_hour

        result = compute_rolling_net_score_per_hour(base_df)
        # Pour le premier match (trié par start_time) : kills=8, deaths=6 → 2
        net_vals = result.sort("start_time")["net_score"].to_list()
        expected = [
            base_df.sort("start_time")["kills"][i] - base_df.sort("start_time")["deaths"][i]
            for i in range(len(base_df))
        ]
        assert net_vals == expected

    def test_sans_time_played_retourne_vide(self) -> None:
        """Sans time_played_seconds → DataFrame vide retourné."""
        from src.analysis.cumulative_progression import compute_rolling_net_score_per_hour

        df = pl.DataFrame(
            {
                "start_time": [datetime(2026, 1, i) for i in range(1, 6)],
                "kills": [8, 10, 7, 9, 11],
                "deaths": [5, 6, 8, 4, 7],
            }
        )
        result = compute_rolling_net_score_per_hour(df)
        assert result.is_empty()

    def test_fenetre_adaptative_minimum_3(self) -> None:
        """Avec peu de matchs, fenêtre = max(3, 10%)."""
        from src.analysis.cumulative_progression import compute_rolling_net_score_per_hour

        df = pl.DataFrame(
            {
                "start_time": [datetime(2026, 1, i) for i in range(1, 6)],
                "kills": [8, 10, 7, 9, 11],
                "deaths": [5, 6, 8, 4, 7],
                "time_played_seconds": [600, 720, 580, 810, 660],
            }
        )
        # 5 matchs → fenêtre = max(3, 5*10//100) = max(3, 0) = 3
        result = compute_rolling_net_score_per_hour(df)
        assert not result.is_empty()
        # Les 2 premières valeurs rolling sont NaN (fenêtre=3, pas assez)
        rolling = result.sort("start_time")["rolling_net_per_hour"].to_list()
        assert rolling[2] is not None  # la 3e valeur existe

    def test_fenetre_explicite(self, base_df: pl.DataFrame) -> None:
        from src.analysis.cumulative_progression import compute_rolling_net_score_per_hour

        result = compute_rolling_net_score_per_hour(base_df, window_size=2)
        rolling = result.sort("start_time")["rolling_net_per_hour"].to_list()
        # Avec fenêtre=2 : la 2e valeur doit être non-nulle
        assert rolling[1] is not None

    def test_dataframe_vide(self, empty_df: pl.DataFrame) -> None:
        from src.analysis.cumulative_progression import compute_rolling_net_score_per_hour

        result = compute_rolling_net_score_per_hour(empty_df)
        assert result.is_empty()

    def test_net_per_hour_coherent(self) -> None:
        """net_per_hour = net_score / (seconds / 3600)."""
        from src.analysis.cumulative_progression import compute_rolling_net_score_per_hour

        df = pl.DataFrame(
            {
                "start_time": [datetime(2026, 1, 1)],
                "kills": [10],
                "deaths": [4],
                "time_played_seconds": [3600],  # exactement 1 heure
            }
        )
        result = compute_rolling_net_score_per_hour(df)
        assert result["net_per_hour"][0] == pytest.approx(6.0)  # (10-4)/1h


# =============================================================================
# _ols (fonction interne)
# =============================================================================


class TestOls:
    """Tests pour la fonction OLS interne."""

    def test_serie_parfaitement_lineaire(self) -> None:
        from src.analysis.cumulative_progression import _ols

        y = [1.0, 2.0, 3.0, 4.0, 5.0]
        slope, intercept, y_hat, r_sq = _ols(y)
        assert abs(slope - 1.0) < 1e-9
        assert abs(intercept - 1.0) < 1e-9
        assert r_sq == pytest.approx(1.0, abs=1e-6)

    def test_serie_constante(self) -> None:
        """Série constante → pente 0, R² = 0."""
        from src.analysis.cumulative_progression import _ols

        y = [3.0] * 5
        slope, intercept, y_hat, r_sq = _ols(y)
        assert slope == pytest.approx(0.0, abs=1e-9)
        assert r_sq == pytest.approx(0.0, abs=1e-6)

    def test_un_seul_point(self) -> None:
        from src.analysis.cumulative_progression import _ols

        y = [2.5]
        slope, intercept, y_hat, r_sq = _ols(y)
        assert slope == 0.0
        assert r_sq == 0.0

    def test_y_hat_meme_taille(self) -> None:
        from src.analysis.cumulative_progression import _ols

        y = [1.0, 3.0, 2.0, 5.0, 4.0]
        _, _, y_hat, _ = _ols(y)
        assert len(y_hat) == len(y)
