"""Tests — propagation de map_id dans le pipeline de rendu d'historique de session.

Régression : _session_compare_history.py perdait map_id lors du `.select(display_cols)`,
empêchant la traduction FR et les thumbnails dans map_name_cell_html.
"""

from __future__ import annotations

from datetime import datetime, timezone

import polars as pl

from src.ui.pages._session_compare_history import _build_history_dataframe

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_session_df(**extra) -> pl.DataFrame:
    """DataFrame minimal d'une session avec map_id présent."""
    base = {
        "match_id": ["m1", "m2"],
        "start_time": [
            datetime(2026, 3, 10, 18, 0, tzinfo=timezone.utc),
            datetime(2026, 3, 10, 19, 0, tzinfo=timezone.utc),
        ],
        "map_id": ["map-guid-aaa", "map-guid-bbb"],
        "map_name": ["Aquarius", "Bazaar"],
        "pair_name": ["Arena:Slayer on Aquarius", "Arena:Slayer on Bazaar"],
        "kills": [10, 8],
        "deaths": [5, 6],
        "assists": [2, 1],
        "outcome": [2, 3],
    }
    base.update(extra)
    return pl.DataFrame(base)


# ---------------------------------------------------------------------------
# Tests _build_history_dataframe — retour de map_ids
# ---------------------------------------------------------------------------


class TestBuildHistoryDataframeMapIds:
    """Vérifie que _build_history_dataframe retourne bien les map_id séparément."""

    def test_returns_three_elements(self) -> None:
        """La fonction doit retourner un tuple à 3 éléments."""
        result = _build_history_dataframe(_make_session_df())
        assert len(result) == 3

    def test_map_ids_is_series_when_map_id_present(self) -> None:
        """map_ids doit être une pl.Series quand map_id est dans le DataFrame source."""
        df_display, perf_scores, map_ids = _build_history_dataframe(_make_session_df())
        assert map_ids is not None
        assert isinstance(map_ids, pl.Series)

    def test_map_ids_values_match_source(self) -> None:
        """Les valeurs de map_ids doivent correspondre au map_id source, dans l'ordre."""
        df = _make_session_df()
        df_display, _, map_ids = _build_history_dataframe(df)
        assert map_ids is not None
        assert map_ids.to_list() == ["map-guid-aaa", "map-guid-bbb"]

    def test_map_id_absent_from_df_display(self) -> None:
        """map_id ne doit PAS apparaître comme colonne dans df_display (pas de fuite)."""
        df_display, _, _ = _build_history_dataframe(_make_session_df())
        assert "map_id" not in df_display.columns

    def test_map_ids_none_when_map_id_missing(self) -> None:
        """map_ids doit être None si la source ne contient pas de colonne map_id."""
        df_no_map_id = _make_session_df().drop("map_id")
        _, _, map_ids = _build_history_dataframe(df_no_map_id)
        assert map_ids is None

    def test_map_ids_length_matches_display_rows(self) -> None:
        """La longueur de map_ids doit être identique au nombre de lignes de df_display."""
        df = _make_session_df()
        df_display, _, map_ids = _build_history_dataframe(df)
        assert map_ids is not None
        assert len(map_ids) == len(df_display)

    def test_perf_scores_still_returned(self) -> None:
        """L'ajout de map_ids ne doit pas casser le retour de perf_scores."""
        _, perf_scores, _ = _build_history_dataframe(_make_session_df())
        # perf_scores peut être None si les colonnes requises sont absentes — pas d'erreur
        assert perf_scores is None or isinstance(perf_scores, pl.Series)
