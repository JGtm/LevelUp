"""Tests v6.4 — helpers purs de la page Médias (filtres & tri).

Ces tests couvrent :
- _enrich_media_with_match_data : jointure df_full sur match_id
- _apply_media_filters           : filtres kind/nom/carte/mode/outcome/squad + tri

Aucun import Streamlit — ces fonctions sont testables sans UI.
"""

from __future__ import annotations

from datetime import datetime

import polars as pl

from src.ui.pages.media_tab import (
    _SORT_COL_MAP,
    _apply_media_filters,
    _enrich_media_with_match_data,
)

# ── Fixtures ──────────────────────────────────────────────────────────────────


def _make_media_df(rows: list[dict] | None = None) -> pl.DataFrame:
    """DataFrame médias minimal avec les colonnes attendues."""
    default_rows = [
        {
            "file_path": "/a/clip1.mp4",
            "file_name": "clip1.mp4",
            "kind": "video",
            "capture_end_utc": datetime(2026, 3, 1, 12, 0),
            "match_id": "match-001",
            "map_name": "Recharge",
            "section": "mine",
            "thumbnail_path": None,
        },
        {
            "file_path": "/a/screen2.png",
            "file_name": "screen2.png",
            "kind": "image",
            "capture_end_utc": datetime(2026, 3, 2, 10, 0),
            "match_id": "match-002",
            "map_name": "Bazaar",
            "section": "mine",
            "thumbnail_path": None,
        },
        {
            "file_path": "/a/old.png",
            "file_name": "old_capture.png",
            "kind": "image",
            "capture_end_utc": datetime(2025, 12, 31, 23, 0),
            "match_id": None,
            "map_name": None,
            "section": "unassigned",
            "thumbnail_path": None,
        },
    ]
    return pl.DataFrame(rows or default_rows)


def _make_df_full() -> pl.DataFrame:
    """DataFrame matches minimal reproduisant les colonnes enrichissables."""
    return pl.DataFrame(
        {
            "match_id": ["match-001", "match-002"],
            "outcome": [2, 3],  # 2=Victoire, 3=Défaite
            "pair_name": ["Ranked Slayer", "Quick Play CTF"],
            "mode_ui": ["Ranked Slayer", "CTF"],
            "map_ui": ["Recharge", "Bazaar"],
            "is_with_friends": [False, True],
        }
    )


def _base_filters(**overrides) -> dict:
    """Dict de filtres par défaut (aucun filtre actif, tri par date desc)."""
    base = {
        "sections": [],
        "map": None,
        "map_col": "map_ui",
        "mode": None,
        "mode_col": "mode_ui",
        "outcome_codes": [],
        "squad": "Tous",
        "squad_solo": "Solo",
        "squad_squad": "Escouade",
        "kinds": ["image", "video"],
        "name": "",
        "sort_col": "capture_end_utc",
        "sort_desc": True,
        "cols_per_row": 4,
    }
    base.update(overrides)
    return base


# ── _enrich_media_with_match_data ─────────────────────────────────────────────


class TestEnrichMediaWithMatchData:
    def test_returns_unchanged_when_df_full_none(self) -> None:
        media = _make_media_df()
        result = _enrich_media_with_match_data(media, None)
        assert result.columns == media.columns
        assert len(result) == len(media)

    def test_returns_unchanged_when_df_full_empty(self) -> None:
        media = _make_media_df()
        result = _enrich_media_with_match_data(media, pl.DataFrame())
        assert len(result) == len(media)

    def test_returns_unchanged_when_no_match_id_in_media(self) -> None:
        media = pl.DataFrame({"file_path": ["/a/x.mp4"], "kind": ["video"]})
        result = _enrich_media_with_match_data(media, _make_df_full())
        assert "outcome" not in result.columns

    def test_adds_outcome_column(self) -> None:
        media = _make_media_df()
        result = _enrich_media_with_match_data(media, _make_df_full())
        assert "outcome" in result.columns

    def test_adds_mode_and_map_ui_columns(self) -> None:
        media = _make_media_df()
        result = _enrich_media_with_match_data(media, _make_df_full())
        assert "mode_ui" in result.columns
        assert "map_ui" in result.columns

    def test_adds_is_with_friends_column(self) -> None:
        media = _make_media_df()
        result = _enrich_media_with_match_data(media, _make_df_full())
        assert "is_with_friends" in result.columns

    def test_correct_outcome_value_for_match(self) -> None:
        media = _make_media_df()
        result = _enrich_media_with_match_data(media, _make_df_full())
        row = result.filter(pl.col("match_id") == "match-001")
        assert row["outcome"][0] == 2

    def test_null_outcome_for_unassigned_media(self) -> None:
        """Les médias sans match_id doivent avoir outcome NULL après join."""
        media = _make_media_df()
        result = _enrich_media_with_match_data(media, _make_df_full())
        row = result.filter(pl.col("match_id").is_null())
        assert row["outcome"][0] is None

    def test_does_not_overwrite_existing_column(self) -> None:
        """Si map_ui est déjà dans media_df, il ne doit pas être écrasé."""
        media = _make_media_df().with_columns(pl.lit("Existant").alias("map_ui"))
        result = _enrich_media_with_match_data(media, _make_df_full())
        # La colonne existante doit être conservée
        assert result.filter(pl.col("match_id") == "match-001")["map_ui"][0] == "Existant"

    def test_preserves_row_count(self) -> None:
        media = _make_media_df()
        result = _enrich_media_with_match_data(media, _make_df_full())
        assert len(result) == len(media)

    def test_returns_unchanged_if_no_enrichable_columns_in_df_full(self) -> None:
        media = _make_media_df()
        df_full_minimal = pl.DataFrame({"match_id": ["match-001"]})
        result = _enrich_media_with_match_data(media, df_full_minimal)
        assert "outcome" not in result.columns


# ── _apply_media_filters ──────────────────────────────────────────────────────


class TestApplyMediaFilters:
    def test_no_filter_returns_all_rows(self) -> None:
        media = _make_media_df()
        result = _apply_media_filters(media, _base_filters())
        assert len(result) == len(media)

    def test_kind_filter_images_only(self) -> None:
        media = _make_media_df()
        result = _apply_media_filters(media, _base_filters(kinds=["image"]))
        assert all(r == "image" for r in result["kind"].to_list())

    def test_kind_filter_videos_only(self) -> None:
        media = _make_media_df()
        result = _apply_media_filters(media, _base_filters(kinds=["video"]))
        assert all(r == "video" for r in result["kind"].to_list())

    def test_kind_filter_empty_returns_empty(self) -> None:
        media = _make_media_df()
        result = _apply_media_filters(media, _base_filters(kinds=[]))
        # kinds=[] → pas de filtre kind (comportement voulu : retain all)
        assert len(result) == len(media)

    def test_name_filter_case_insensitive(self) -> None:
        media = _make_media_df()
        result = _apply_media_filters(media, _base_filters(name="CLIP"))
        assert len(result) == 1
        assert result["file_name"][0] == "clip1.mp4"

    def test_name_filter_partial_match(self) -> None:
        media = _make_media_df()
        result = _apply_media_filters(media, _base_filters(name="old"))
        assert len(result) == 1

    def test_name_filter_no_match_returns_empty(self) -> None:
        media = _make_media_df()
        result = _apply_media_filters(media, _base_filters(name="zyxwvuts"))
        assert result.is_empty()

    def test_map_filter_exact_match(self) -> None:
        media = _enrich_media_with_match_data(_make_media_df(), _make_df_full())
        result = _apply_media_filters(media, _base_filters(map="Recharge", map_col="map_ui"))
        assert all(r == "Recharge" for r in result["map_ui"].drop_nulls().to_list())
        assert len(result) >= 1

    def test_map_filter_none_skipped(self) -> None:
        media = _enrich_media_with_match_data(_make_media_df(), _make_df_full())
        result = _apply_media_filters(media, _base_filters(map=None))
        assert len(result) == len(media)

    def test_mode_filter(self) -> None:
        media = _enrich_media_with_match_data(_make_media_df(), _make_df_full())
        result = _apply_media_filters(media, _base_filters(mode="CTF", mode_col="mode_ui"))
        assert len(result) == 1
        assert result["mode_ui"][0] == "CTF"

    def test_outcome_filter_victory_only(self) -> None:
        media = _enrich_media_with_match_data(_make_media_df(), _make_df_full())
        result = _apply_media_filters(media, _base_filters(outcome_codes=[2]))
        victories = [r for r in result["outcome"].to_list() if r is not None]
        assert all(v == 2 for v in victories)

    def test_squad_solo_filter(self) -> None:
        media = _enrich_media_with_match_data(_make_media_df(), _make_df_full())
        result = _apply_media_filters(
            media, _base_filters(squad="Solo", squad_solo="Solo", squad_squad="Escouade")
        )
        friends_values = [v for v in result["is_with_friends"].to_list() if v is not None]
        assert all(v is False for v in friends_values)

    def test_squad_squad_filter(self) -> None:
        media = _enrich_media_with_match_data(_make_media_df(), _make_df_full())
        result = _apply_media_filters(
            media, _base_filters(squad="Escouade", squad_solo="Solo", squad_squad="Escouade")
        )
        friends_values = [v for v in result["is_with_friends"].to_list() if v is not None]
        assert all(v is True for v in friends_values)

    def test_apply_match_filters_false_skips_map_mode_outcome_squad(self) -> None:
        """apply_match_filters=False → seuls kind et nom sont appliqués."""
        media = _enrich_media_with_match_data(_make_media_df(), _make_df_full())
        filters = _base_filters(
            map="Recharge",
            map_col="map_ui",
            outcome_codes=[2],
            squad="Solo",
            squad_solo="Solo",
            squad_squad="Escouade",
        )
        result = _apply_media_filters(media, filters, apply_match_filters=False)
        # Tous les médias doivent rester (les filtres match sont ignorés)
        assert len(result) == len(media)

    def test_sort_descending_by_date(self) -> None:
        media = _make_media_df()
        result = _apply_media_filters(
            media, _base_filters(sort_col="capture_end_utc", sort_desc=True)
        )
        dates = result["capture_end_utc"].drop_nulls().to_list()
        assert dates == sorted(dates, reverse=True)

    def test_sort_ascending_by_date(self) -> None:
        media = _make_media_df()
        result = _apply_media_filters(
            media, _base_filters(sort_col="capture_end_utc", sort_desc=False)
        )
        dates = result["capture_end_utc"].drop_nulls().to_list()
        assert dates == sorted(dates)

    def test_sort_falls_back_to_capture_end_utc_if_col_missing(self) -> None:
        """Si sort_col n'existe pas, repli sur capture_end_utc."""
        media = _make_media_df()
        result = _apply_media_filters(media, _base_filters(sort_col="nonexistent_col"))
        # Doit trier quand même sans lever d'exception
        assert len(result) == len(media)

    def test_combined_kind_and_name_filter(self) -> None:
        media = _make_media_df()
        result = _apply_media_filters(media, _base_filters(kinds=["image"], name="screen"))
        assert len(result) == 1
        assert result["file_name"][0] == "screen2.png"


# ── Constantes déclarées ──────────────────────────────────────────────────────


class TestConstants:
    def test_sort_col_map_covers_all_sort_option_keys(self) -> None:
        """Toutes les clés _SORT_OPTIONS_KEYS doivent avoir un mapping."""
        from src.ui.pages.media_tab import _SORT_OPTIONS_KEYS

        for key in _SORT_OPTIONS_KEYS:
            assert key in _SORT_COL_MAP, f"Clé manquante dans _SORT_COL_MAP : {key}"

    def test_sort_col_map_values_are_non_empty_strings(self) -> None:
        for key, val in _SORT_COL_MAP.items():
            assert isinstance(val, str) and val, f"Valeur vide pour {key}"
