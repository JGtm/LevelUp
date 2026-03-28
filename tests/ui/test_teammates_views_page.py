"""Tests pour la page Coéquipiers Vues (Sprint 7bis).

Couvre :
- _merge_trio_dataframes (fonction pure, test unitaire)
- Edge cases : DataFrames vides, f2 optionnel
"""

from __future__ import annotations

from datetime import datetime, timedelta

import numpy as np
import polars as pl

# =============================================================================
# Tests de _merge_trio_dataframes (fonction pure)
# =============================================================================


def _make_player_df(
    player_prefix: str, n: int = 5, *, with_start_time: bool = False
) -> pl.DataFrame:
    """Crée un DataFrame de stats joueur pour les tests trio."""
    np.random.seed(42)
    start = datetime(2025, 1, 1)
    data = {
        "match_id": [f"match_{i}" for i in range(n)],
        "kills": np.random.randint(5, 25, n).tolist(),
        "deaths": np.random.randint(3, 15, n).tolist(),
        "assists": np.random.randint(2, 12, n).tolist(),
        "accuracy": np.random.uniform(30, 60, n).tolist(),
        "ratio": np.random.uniform(0.5, 2.5, n).tolist(),
        "average_life_seconds": np.random.uniform(20, 60, n).tolist(),
    }
    if with_start_time:
        data["start_time"] = [start + timedelta(hours=i) for i in range(n)]
        data["time_played_seconds"] = np.random.randint(300, 900, n).tolist()
    return pl.DataFrame(data)


class TestMergeTrioDataframes:
    """Tests pour _merge_trio_dataframes."""

    def test_basic_merge(self) -> None:
        from src.ui.pages.teammates_views import _merge_trio_dataframes

        me = _make_player_df("me", 5, with_start_time=True)
        f1 = _make_player_df("f1", 5)
        f2 = _make_player_df("f2", 5)

        merged = _merge_trio_dataframes(me, f1, f2)
        assert merged.height == 5  # 5 matchs communs
        assert "f1_kills" in merged.columns
        assert "f2_kills" in merged.columns
        assert "kills" in merged.columns  # kills du joueur principal
        assert "start_time" in merged.columns

    def test_merge_without_f2(self) -> None:
        """Avec f2=None, le merge ne contient que me + f1."""
        from src.ui.pages.teammates_views import _merge_trio_dataframes

        me = _make_player_df("me", 5, with_start_time=True)
        f1 = _make_player_df("f1", 5)

        merged = _merge_trio_dataframes(me, f1, None)
        assert merged.height == 5
        assert "f1_kills" in merged.columns
        assert not any(c.startswith("f2_") for c in merged.columns)

    def test_partial_overlap(self) -> None:
        """Seuls les matchs communs apparaissent dans le merge."""
        from src.ui.pages.teammates_views import _merge_trio_dataframes

        me = _make_player_df("me", 5, with_start_time=True)
        # f1 a seulement les matchs 0 et 1
        f1 = _make_player_df("f1", 2)
        # f2 a seulement les matchs 0, 1, 2
        f2 = _make_player_df("f2", 3)

        merged = _merge_trio_dataframes(me, f1, f2)
        assert merged.height == 2  # intersection = match_0, match_1

    def test_no_overlap(self) -> None:
        from src.ui.pages.teammates_views import _merge_trio_dataframes

        me = _make_player_df("me", 5, with_start_time=True)
        f1 = pl.DataFrame(
            {
                "match_id": ["other_1", "other_2"],
                "kills": [5, 10],
                "deaths": [3, 5],
                "assists": [2, 3],
                "accuracy": [40.0, 50.0],
                "ratio": [1.0, 2.0],
                "average_life_seconds": [30.0, 40.0],
            }
        )
        f2 = _make_player_df("f2", 5)

        merged = _merge_trio_dataframes(me, f1, f2)
        assert merged.height == 0
