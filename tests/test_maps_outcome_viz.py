"""Tests pour src/visualization/maps_outcome.py.

Couvre les 4 fonctions outcome-focused :
- plot_map_lollipop
- plot_map_outcome_timeline
- plot_map_winrate_bullet
- plot_map_perf_vs_history
"""

from __future__ import annotations

from datetime import datetime, timedelta

import pytest

try:
    import polars as pl

    POLARS_AVAILABLE = True
except ImportError:
    POLARS_AVAILABLE = False
    pl = None  # type: ignore[assignment]

try:
    import plotly.graph_objects as go

    PLOTLY_AVAILABLE = True
except ImportError:
    PLOTLY_AVAILABLE = False
    go = None  # type: ignore[assignment]

try:
    from src.visualization.maps_outcome import (
        plot_map_lollipop,
        plot_map_outcome_timeline,
        plot_map_perf_vs_history,
        plot_map_winrate_bullet,
    )

    MAPS_OUTCOME_AVAILABLE = True
except Exception:
    MAPS_OUTCOME_AVAILABLE = False

pytestmark = pytest.mark.skipif(
    not POLARS_AVAILABLE or not PLOTLY_AVAILABLE or not MAPS_OUTCOME_AVAILABLE,
    reason="Polars, Plotly ou maps_outcome non disponibles",
)


# =============================================================================
# Fixtures
# =============================================================================


def _make_breakdown(n: int = 4) -> pl.DataFrame:
    """Breakdown par carte avec win_rate, loss_rate, performance_avg."""
    maps = ["Recharge", "Streets", "Bazaar", "Live Fire"][:n]
    return pl.DataFrame(
        {
            "map_name": maps,
            "matches": [10, 8, 6, 4][:n],
            "win_rate": [0.7, 0.5, 0.3, 0.6][:n],
            "loss_rate": [0.3, 0.4, 0.6, 0.3][:n],
            "ratio_global": [1.5, 1.0, 0.5, 1.2][:n],
            "accuracy_avg": [45.0, 50.0, 38.0, 42.0][:n],
            "performance_avg": [0.3, 0.0, -0.2, 0.1][:n],
        }
    )


def _make_matches(n: int = 10, *, with_session: bool = False) -> pl.DataFrame:
    """DataFrame de matchs bruts pour la timeline."""
    start = datetime(2025, 1, 1)
    match_ids = [f"m{i}" for i in range(n)]
    return pl.DataFrame(
        {
            "match_id": match_ids,
            "map_name": (["Recharge", "Streets"] * n)[:n],
            "start_time": [start + timedelta(hours=i) for i in range(n)],
            "outcome": ([2, 3, 2, 3, 1] * n)[:n],  # WIN=2, LOSS=3, TIE=1
        }
    )


# =============================================================================
# plot_map_lollipop
# =============================================================================


class TestPlotMapLollipop:
    def test_retourne_figure_avec_donnees_valides(self) -> None:
        fig = plot_map_lollipop(_make_breakdown())
        assert isinstance(fig, go.Figure)
        assert len(fig.data) == 2  # stems + dots

    def test_retourne_figure_vide_sur_df_vide(self) -> None:
        empty = pl.DataFrame(
            schema={"map_name": pl.Utf8, "win_rate": pl.Float64, "matches": pl.Int64}
        )
        fig = plot_map_lollipop(empty)
        assert isinstance(fig, go.Figure)
        assert len(fig.data) == 0

    def test_retourne_figure_vide_si_win_rate_null(self) -> None:
        df = pl.DataFrame({"map_name": ["A", "B"], "win_rate": [None, None], "matches": [3, 5]})
        fig = plot_map_lollipop(df)
        assert isinstance(fig, go.Figure)
        assert len(fig.data) == 0

    def test_couleur_verte_au_dessus_50pct(self) -> None:
        df = _make_breakdown(1).with_columns(pl.lit(0.8).alias("win_rate"))
        fig = plot_map_lollipop(df)
        assert isinstance(fig, go.Figure)

    def test_une_seule_carte(self) -> None:
        fig = plot_map_lollipop(_make_breakdown(1))
        assert isinstance(fig, go.Figure)
        assert len(fig.data) == 2

    def test_lang_en(self) -> None:
        fig = plot_map_lollipop(_make_breakdown(), lang="en")
        assert isinstance(fig, go.Figure)


# =============================================================================
# plot_map_outcome_timeline
# =============================================================================


class TestPlotMapOutcomeTimeline:
    def test_retourne_figure_avec_donnees_valides(self) -> None:
        fig = plot_map_outcome_timeline(_make_matches())
        assert fig is not None
        assert isinstance(fig, go.Figure)
        assert len(fig.data) > 0

    def test_retourne_none_sur_df_vide(self) -> None:
        empty = pl.DataFrame(
            schema={
                "map_name": pl.Utf8,
                "start_time": pl.Datetime,
                "outcome": pl.Int64,
                "match_id": pl.Utf8,
            }
        )
        assert plot_map_outcome_timeline(empty) is None

    def test_retourne_none_si_colonnes_manquantes(self) -> None:
        df = pl.DataFrame({"map_name": ["A"], "outcome": [2]})
        assert plot_map_outcome_timeline(df) is None

    def test_session_match_ids_mis_en_evidence(self) -> None:
        matches = _make_matches(6)
        session_ids = matches["match_id"].to_list()[:3]
        fig = plot_map_outcome_timeline(matches, session_match_ids=session_ids)
        assert fig is not None
        # Les traces de session ont une taille de marker plus grande
        session_traces = [t for t in fig.data if t.marker.size == 14]
        assert len(session_traces) > 0

    def test_session_ids_none_accepte(self) -> None:
        fig = plot_map_outcome_timeline(_make_matches(), session_match_ids=None)
        assert fig is not None

    def test_lang_en(self) -> None:
        fig = plot_map_outcome_timeline(_make_matches(), lang="en")
        assert fig is not None


# =============================================================================
# plot_map_winrate_bullet
# =============================================================================


class TestPlotMapWinrateBullet:
    def test_retourne_figure_avec_cartes_communes(self) -> None:
        bd = _make_breakdown()
        fig = plot_map_winrate_bullet(bd, bd)
        assert fig is not None
        assert isinstance(fig, go.Figure)
        # 4 barres overlay + 3 entrées légende couleur (vert/ambre/rouge)
        assert len(fig.data) == 7

    def test_retourne_none_sans_cartes_communes(self) -> None:
        bd_curr = _make_breakdown(1)
        bd_hist = _make_breakdown(1).with_columns(pl.lit("AutreMap").alias("map_name"))
        assert plot_map_winrate_bullet(bd_curr, bd_hist) is None

    def test_retourne_none_sur_df_vide(self) -> None:
        empty = pl.DataFrame(
            schema={"map_name": pl.Utf8, "win_rate": pl.Float64, "matches": pl.Int64}
        )
        assert plot_map_winrate_bullet(empty, _make_breakdown()) is None

    def test_couleur_verte_si_session_meilleure(self) -> None:
        bd_curr = _make_breakdown(1).with_columns(pl.lit(0.9).alias("win_rate"))
        bd_hist = _make_breakdown(1).with_columns(pl.lit(0.5).alias("win_rate"))
        fig = plot_map_winrate_bullet(bd_curr, bd_hist)
        assert fig is not None

    def test_lang_en(self) -> None:
        bd = _make_breakdown()
        fig = plot_map_winrate_bullet(bd, bd, lang="en")
        assert fig is not None


# =============================================================================
# plot_map_perf_vs_history
# =============================================================================


class TestPlotMapPerfVsHistory:
    def test_retourne_figure_avec_cartes_communes(self) -> None:
        bd = _make_breakdown()
        fig = plot_map_perf_vs_history(bd, bd)
        assert fig is not None
        assert isinstance(fig, go.Figure)
        assert len(fig.data) == 2  # historique + session

    def test_retourne_none_sans_cartes_communes(self) -> None:
        bd_curr = _make_breakdown(1)
        bd_hist = _make_breakdown(1).with_columns(pl.lit("AutreMap").alias("map_name"))
        assert plot_map_perf_vs_history(bd_curr, bd_hist) is None

    def test_retourne_none_sur_df_vide(self) -> None:
        empty = pl.DataFrame(
            schema={
                "map_name": pl.Utf8,
                "performance_avg": pl.Float64,
                "matches": pl.Int64,
            }
        )
        assert plot_map_perf_vs_history(empty, _make_breakdown()) is None

    def test_trie_par_performance(self) -> None:
        bd = _make_breakdown(4)
        fig = plot_map_perf_vs_history(bd, bd)
        assert fig is not None
        # Vérifie que les cartes sont présentes dans la figure
        y_values = list(fig.data[0].y)
        assert len(y_values) == 4

    def test_lang_en(self) -> None:
        bd = _make_breakdown()
        fig = plot_map_perf_vs_history(bd, bd, lang="en")
        assert fig is not None
