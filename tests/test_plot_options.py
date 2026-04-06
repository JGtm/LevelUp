"""Tests unitaires pour src/visualization/_plot_options.py.

Couvre :
- KdCiData : construction, invariants
- EwmaData : construction, regression_data optionnel
- PlotOptions : valeurs par défaut, from_session
- ChartTheme : singleton DEFAULT_THEME
"""

from __future__ import annotations

from dataclasses import asdict

from src.visualization._plot_options import (
    DEFAULT_THEME,
    ChartTheme,
    EwmaData,
    KdCiData,
    PlotOptions,
)

# =============================================================================
# KdCiData
# =============================================================================


class TestKdCiData:
    def _make(self) -> KdCiData:
        return KdCiData(
            x=[1, 2, 3],
            y_cumul=[1.0, 1.5, 1.3],
            y_upper=[1.2, 1.8, 1.6],
            y_lower=[0.8, 1.2, 1.0],
            y_match=[1.0, 2.0, 0.5],
        )

    def test_construction(self) -> None:
        data = self._make()
        assert data.x == [1, 2, 3]
        assert len(data.y_cumul) == 3
        assert len(data.y_upper) == 3
        assert len(data.y_lower) == 3
        assert len(data.y_match) == 3

    def test_est_dataclass(self) -> None:
        data = self._make()
        d = asdict(data)
        assert "x" in d
        assert "y_cumul" in d
        assert "y_upper" in d
        assert "y_lower" in d
        assert "y_match" in d

    def test_listes_vides_acceptees(self) -> None:
        data = KdCiData(x=[], y_cumul=[], y_upper=[], y_lower=[], y_match=[])
        assert data.x == []

    def test_egalite(self) -> None:
        d1 = self._make()
        d2 = self._make()
        assert d1 == d2


# =============================================================================
# EwmaData
# =============================================================================


class TestEwmaData:
    def _make(self, with_regression: bool = False) -> EwmaData:
        regression = {"slope": 0.1, "intercept": 1.0} if with_regression else None
        return EwmaData(
            x=[1, 2, 3, 4],
            y_kd=[1.0, 1.2, 0.9, 1.1],
            y_ewma=[1.0, 1.1, 1.0, 1.05],
            regression_data=regression,
        )

    def test_construction_sans_regression(self) -> None:
        data = self._make(with_regression=False)
        assert data.regression_data is None
        assert len(data.x) == 4

    def test_construction_avec_regression(self) -> None:
        data = self._make(with_regression=True)
        assert data.regression_data is not None
        assert "slope" in data.regression_data

    def test_est_dataclass(self) -> None:
        data = self._make()
        d = asdict(data)
        assert "x" in d
        assert "y_kd" in d
        assert "y_ewma" in d
        assert "regression_data" in d

    def test_listes_vides_acceptees(self) -> None:
        data = EwmaData(x=[], y_kd=[], y_ewma=[], regression_data=None)
        assert data.y_ewma == []

    def test_egalite(self) -> None:
        d1 = self._make()
        d2 = self._make()
        assert d1 == d2


# =============================================================================
# PlotOptions
# =============================================================================


class TestPlotOptions:
    def test_valeurs_par_defaut(self) -> None:
        opts = PlotOptions()
        assert opts.lang == "fr"
        assert opts.smooth is False
        assert opts.height_px == 360
        assert opts.show_records is True
        assert opts.is_negative is False

    def test_construction_personnalisee(self) -> None:
        opts = PlotOptions(lang="en", smooth=True, height_px=420)
        assert opts.lang == "en"
        assert opts.smooth is True
        assert opts.height_px == 420

    def test_theme_est_chart_theme(self) -> None:
        opts = PlotOptions()
        assert isinstance(opts.theme, ChartTheme)


# =============================================================================
# ChartTheme / DEFAULT_THEME
# =============================================================================


class TestChartTheme:
    def test_default_theme_est_chart_theme(self) -> None:
        assert isinstance(DEFAULT_THEME, ChartTheme)

    def test_couleurs_sont_des_strings(self) -> None:
        t = DEFAULT_THEME
        assert t.color_kills.startswith("#")
        assert t.color_deaths.startswith("#")
        assert t.color_kd.startswith("#")

    def test_zero_line_values(self) -> None:
        t = DEFAULT_THEME
        assert isinstance(t.zero_line_width, int)
        assert t.zero_line_width > 0
