"""Tests pour plot_squad_map_heatmap (Feature 1 heatmap escouade).

Couvre les cas nominaux et les edge cases de la heatmap
performance × joueur × carte.
"""

from __future__ import annotations

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
    from src.visualization.friends_impact_heatmap import plot_squad_map_heatmap

    HEATMAP_AVAILABLE = True
except Exception:
    HEATMAP_AVAILABLE = False

pytestmark = pytest.mark.skipif(
    not POLARS_AVAILABLE or not PLOTLY_AVAILABLE or not HEATMAP_AVAILABLE,
    reason="Polars, Plotly ou friends_impact_heatmap non disponibles",
)


# =============================================================================
# Helpers
# =============================================================================


def _make_player_matches(maps: list[str], outcomes: list[int] | None = None) -> pl.DataFrame:
    """Crée un DataFrame de matchs bruts par joueur."""
    n = len(maps)
    outcomes = outcomes or ([2] * n)  # WIN par défaut
    from datetime import datetime, timedelta

    start = datetime(2025, 1, 1)
    return pl.DataFrame(
        {
            "match_id": [f"m{i}" for i in range(n)],
            "map_name": maps,
            "start_time": [start + timedelta(hours=i) for i in range(n)],
            "outcome": outcomes,
            "kills": [10] * n,
            "deaths": [5] * n,
            "assists": [3] * n,
            "accuracy": [50.0] * n,
            "ratio": [1.5] * n,
            "performance_avg": [0.2] * n,
        }
    )


# =============================================================================
# Tests
# =============================================================================


class TestPlotSquadMapHeatmap:
    def test_retourne_figure_avec_deux_joueurs(self) -> None:
        maps = ["Recharge", "Streets", "Bazaar"]
        series = [
            ("Alice", _make_player_matches(maps)),
            ("Bob", _make_player_matches(maps, outcomes=[3, 2, 2])),
        ]
        fig = plot_squad_map_heatmap(series)
        assert fig is not None
        assert isinstance(fig, go.Figure)
        assert len(fig.data) == 1  # un seul Heatmap trace

    def test_retourne_figure_avec_un_seul_joueur(self) -> None:
        """La viz accepte 1 joueur — le guard < 2 est dans le wrapper Streamlit."""
        series = [("Alice", _make_player_matches(["Recharge"]))]
        fig = plot_squad_map_heatmap(series)
        assert fig is not None
        assert isinstance(fig, go.Figure)

    def test_retourne_none_avec_serie_vide(self) -> None:
        assert plot_squad_map_heatmap([]) is None

    def test_retourne_none_si_tous_df_vides(self) -> None:
        empty = pl.DataFrame(schema={"map_name": pl.Utf8, "match_id": pl.Utf8, "outcome": pl.Int64})
        series = [("Alice", empty), ("Bob", empty)]
        assert plot_squad_map_heatmap(series) is None

    def test_axes_joueurs_et_cartes(self) -> None:
        maps = ["Recharge", "Streets"]
        series = [
            ("Alice", _make_player_matches(maps)),
            ("Bob", _make_player_matches(maps)),
        ]
        fig = plot_squad_map_heatmap(series)
        assert fig is not None
        heatmap = fig.data[0]
        # y = joueurs, x = cartes
        assert "Alice" in heatmap.y
        assert "Bob" in heatmap.y

    def test_trois_joueurs(self) -> None:
        maps = ["Recharge", "Streets", "Live Fire"]
        series = [
            ("Alice", _make_player_matches(maps)),
            ("Bob", _make_player_matches(maps)),
            ("Charlie", _make_player_matches(maps)),
        ]
        fig = plot_squad_map_heatmap(series)
        assert fig is not None
        assert len(fig.data[0].y) == 3

    def test_lang_en(self) -> None:
        maps = ["Recharge", "Streets"]
        series = [
            ("Alice", _make_player_matches(maps)),
            ("Bob", _make_player_matches(maps)),
        ]
        fig = plot_squad_map_heatmap(series, lang="en")
        assert fig is not None
