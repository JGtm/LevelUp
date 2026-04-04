"""Tests pour la dérivation i18n des colonnes map_ui / mode_ui.

Couvre :
- resolve_map_display_names : fallback + traduction (sans DB réelle via mock)
- _add_derived_columns : présence et valeur de map_ui et mode_ui
- compute_map_breakdown : utilise map_ui comme clé de groupe si présent
"""

from __future__ import annotations

from unittest.mock import patch

import polars as pl
import pytest

from src.analysis.maps import compute_map_breakdown
from src.ui.translations import resolve_map_display_names


# ---------------------------------------------------------------------------
# resolve_map_display_names — sans base de données réelle
# ---------------------------------------------------------------------------


class TestResolveMapDisplayNames:
    """Tests unitaires pour resolve_map_display_names."""

    def test_empty_input_returns_empty(self) -> None:
        result = resolve_map_display_names({}, "fr")
        assert result == {}

    def test_no_db_returns_fallbacks(self, tmp_path) -> None:
        """Sans metadata.duckdb, retourne les valeurs fallback EN."""
        fallbacks = {"uuid-1": "Aquarius", "uuid-2": "Catalyst"}
        with patch("src.utils.paths.get_metadata_db_path", return_value=tmp_path / "absent.duckdb"):
            result = resolve_map_display_names(fallbacks, "fr")
        assert result == fallbacks

    def test_db_without_asset_translations_returns_fallbacks(self, tmp_path) -> None:
        """Si asset_translations absente, retourne les valeurs fallback."""
        import duckdb

        db = tmp_path / "metadata.duckdb"
        conn = duckdb.connect(str(db))
        conn.execute("CREATE TABLE other_table (id VARCHAR)")
        conn.close()

        fallbacks = {"uuid-1": "Aquarius"}
        with patch("src.utils.paths.get_metadata_db_path", return_value=db):
            result = resolve_map_display_names(fallbacks, "fr")
        assert result == fallbacks

    def test_db_with_translation_returns_translated(self, tmp_path) -> None:
        """Retourne le nom traduit quand asset_translations est peuplé."""
        import duckdb

        db = tmp_path / "metadata.duckdb"
        conn = duckdb.connect(str(db))
        conn.execute(
            "CREATE TABLE asset_translations "
            "(asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR)"
        )
        conn.execute(
            "INSERT INTO asset_translations VALUES "
            "('uuid-1', 'map', 'fr-FR', 'Aquarius'), "
            "('uuid-2', 'map', 'fr-FR', 'Catalyst')"
        )
        conn.close()

        fallbacks = {"uuid-1": "Aquarius_EN", "uuid-2": "Catalyst_EN"}
        with patch("src.utils.paths.get_metadata_db_path", return_value=db):
            result = resolve_map_display_names(fallbacks, "fr")

        assert result["uuid-1"] == "Aquarius"
        assert result["uuid-2"] == "Catalyst"

    def test_partial_translation_keeps_fallback_for_missing(self, tmp_path) -> None:
        """Les map_id sans traduction gardent leur valeur fallback."""
        import duckdb

        db = tmp_path / "metadata.duckdb"
        conn = duckdb.connect(str(db))
        conn.execute(
            "CREATE TABLE asset_translations "
            "(asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR)"
        )
        conn.execute(
            "INSERT INTO asset_translations VALUES "
            "('uuid-1', 'map', 'fr-FR', 'Aquarius')"
        )
        conn.close()

        fallbacks = {"uuid-1": "Aquarius_EN", "uuid-2": "Unknown_EN"}
        with patch("src.utils.paths.get_metadata_db_path", return_value=db):
            result = resolve_map_display_names(fallbacks, "fr")

        assert result["uuid-1"] == "Aquarius"
        assert result["uuid-2"] == "Unknown_EN"  # fallback conservé

    def test_en_us_fallback_when_lang_not_found(self, tmp_path) -> None:
        """Si lang=fr non disponible, tente en-US comme fallback."""
        import duckdb

        db = tmp_path / "metadata.duckdb"
        conn = duckdb.connect(str(db))
        conn.execute(
            "CREATE TABLE asset_translations "
            "(asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR)"
        )
        conn.execute(
            "INSERT INTO asset_translations VALUES "
            "('uuid-1', 'map', 'en-US', 'Aquarius EN')"
        )
        conn.close()

        fallbacks = {"uuid-1": "fallback_raw"}
        with patch("src.utils.paths.get_metadata_db_path", return_value=db):
            result = resolve_map_display_names(fallbacks, "de")

        assert result["uuid-1"] == "Aquarius EN"

    def test_exception_in_db_returns_fallbacks(self, tmp_path) -> None:
        """En cas d'erreur DB, retourne silencieusement les valeurs fallback."""
        import duckdb

        # DB corrompue / inaccessible → simule via patch de duckdb_read_only
        fallbacks = {"uuid-1": "Aquarius"}
        with patch(
            "src.utils.db.duckdb_read_only",
            side_effect=RuntimeError("connexion échouée"),
        ):
            result = resolve_map_display_names(fallbacks, "fr")
        assert result == fallbacks


# ---------------------------------------------------------------------------
# _add_derived_columns — map_ui / mode_ui sans DB (fallback path)
# ---------------------------------------------------------------------------


class TestAddDerivedColumnsI18n:
    """Tests pour les colonnes map_ui et mode_ui produites par _add_derived_columns."""

    def _identity(self, s: str) -> str:
        return s

    def _make_df(self, with_map_id: bool = True) -> pl.DataFrame:
        data: dict = {
            "match_id": ["m1", "m2", "m3"],
            "pair_name": ["Arena:Slayer", "BTB:Slayer", "Arena:CTF"],
            "playlist_name": ["Ranked Arena", "BTB", "Ranked Arena"],
            "map_name": ["Aquarius", "Behemoth", "Catalyst"],
        }
        if with_map_id:
            data["map_id"] = ["uuid-aqua", "uuid-behe", "uuid-cata"]
        return pl.DataFrame(data)

    def test_mode_ui_produced_without_db(self) -> None:
        """mode_ui doit être produit même sans DB (translate_pair_name en mémoire)."""
        from src.app._filters_apply import _add_derived_columns

        df = self._make_df(with_map_id=False)
        result = _add_derived_columns(df, self._identity)
        assert "mode_ui" in result.columns
        modes = result["mode_ui"].to_list()
        # translate_pair_name("Arena:Slayer") → "Assassin" en fr
        assert all(m is not None for m in modes)

    def test_map_ui_produced_when_map_id_absent(self) -> None:
        """Sans map_id, map_ui = copie de map_name."""
        from src.app._filters_apply import _add_derived_columns

        df = self._make_df(with_map_id=False)
        result = _add_derived_columns(df, self._identity)
        assert "map_ui" in result.columns
        assert result["map_ui"].to_list() == result["map_name"].to_list()

    def test_map_ui_uses_fallback_without_db(self, tmp_path) -> None:
        """Avec map_id mais sans DB, map_ui = map_name (fallback EN)."""
        from src.app._filters_apply import _add_derived_columns

        df = self._make_df(with_map_id=True)
        with patch(
            "src.utils.paths.get_metadata_db_path",
            return_value=tmp_path / "absent.duckdb",
        ):
            result = _add_derived_columns(df, self._identity)

        assert "map_ui" in result.columns
        # Sans DB, map_ui = fallback = map_name
        assert result["map_ui"].to_list() == result["map_name"].to_list()

    def test_map_ui_translated_when_db_available(self, tmp_path) -> None:
        """Avec map_id et DB peuplée, map_ui contient les noms traduits."""
        import duckdb

        from src.app._filters_apply import _add_derived_columns

        db = tmp_path / "metadata.duckdb"
        conn = duckdb.connect(str(db))
        conn.execute(
            "CREATE TABLE asset_translations "
            "(asset_id VARCHAR, asset_type VARCHAR, lang VARCHAR, name VARCHAR)"
        )
        conn.execute(
            "INSERT INTO asset_translations VALUES "
            "('uuid-aqua', 'map', 'fr-FR', 'Aquarius FR'), "
            "('uuid-behe', 'map', 'fr-FR', 'Béhémoth FR'), "
            "('uuid-cata', 'map', 'fr-FR', 'Catalyste FR')"
        )
        conn.close()

        df = self._make_df(with_map_id=True)
        with (
            patch("src.utils.paths.get_metadata_db_path", return_value=db),
            patch("src.ui.i18n.get_lang", return_value="fr"),
            patch("src.app._filters_apply.get_lang", return_value="fr"),
        ):
            result = _add_derived_columns(df, self._identity)

        assert "map_ui" in result.columns
        map_uis = result["map_ui"].to_list()
        assert "Aquarius FR" in map_uis
        assert "Béhémoth FR" in map_uis
        assert "Catalyste FR" in map_uis

    def test_existing_map_ui_not_overwritten(self) -> None:
        """Si map_ui est déjà présent dans le DataFrame, il n'est pas recalculé."""
        from src.app._filters_apply import _add_derived_columns

        df = self._make_df(with_map_id=True).with_columns(
            pl.lit("custom_map_ui").alias("map_ui")
        )
        result = _add_derived_columns(df, self._identity)
        assert all(v == "custom_map_ui" for v in result["map_ui"].to_list())

    def test_existing_mode_ui_not_overwritten(self) -> None:
        """Si mode_ui est déjà présent dans le DataFrame, il n'est pas recalculé."""
        from src.app._filters_apply import _add_derived_columns

        df = self._make_df().with_columns(pl.lit("custom_mode").alias("mode_ui"))
        result = _add_derived_columns(df, self._identity)
        assert all(v == "custom_mode" for v in result["mode_ui"].to_list())


# ---------------------------------------------------------------------------
# compute_map_breakdown — utilisation de map_ui comme clé de groupe
# ---------------------------------------------------------------------------


class TestComputeMapBreakdownI18n:
    """Tests pour la prise en compte de map_ui dans compute_map_breakdown."""

    def _make_breakdown_df(self, with_map_ui: bool = False) -> pl.DataFrame:
        from datetime import datetime

        data: dict = {
            "match_id": ["m1", "m2", "m3", "m4"],
            "map_name": ["Aquarius EN", "Aquarius EN", "Behemoth EN", "Behemoth EN"],
            "outcome": [2, 3, 2, 2],
            "kills": [10, 8, 12, 9],
            "deaths": [5, 7, 4, 6],
            "start_time": [datetime(2026, 1, i + 1) for i in range(4)],
        }
        if with_map_ui:
            data["map_ui"] = ["Aquarius FR", "Aquarius FR", "Béhémoth FR", "Béhémoth FR"]
        return pl.DataFrame(data)

    def test_groups_by_map_name_when_no_map_ui(self) -> None:
        """Sans map_ui, retourne un DataFrame vide (map_ui est requis depuis Phase 4)."""
        df = self._make_breakdown_df(with_map_ui=False)
        result = compute_map_breakdown(df)
        assert result.is_empty()

    def test_groups_by_map_ui_when_present(self) -> None:
        """Avec map_ui, le résultat porte les noms traduits dans la colonne map_name."""
        df = self._make_breakdown_df(with_map_ui=True)
        result = compute_map_breakdown(df)
        map_names = set(result["map_name"].to_list())
        assert "Aquarius FR" in map_names
        assert "Béhémoth FR" in map_names
        # Les noms EN ne doivent plus apparaître
        assert "Aquarius EN" not in map_names

    def test_match_counts_correct_with_map_ui(self) -> None:
        """Les compteurs de matchs restent corrects quand on groupe par map_ui."""
        df = self._make_breakdown_df(with_map_ui=True)
        result = compute_map_breakdown(df)
        row_aqua = result.filter(pl.col("map_name") == "Aquarius FR")
        row_behe = result.filter(pl.col("map_name") == "Béhémoth FR")
        assert row_aqua["matches"].item() == 2
        assert row_behe["matches"].item() == 2

    def test_win_rate_correct_with_map_ui(self) -> None:
        """Les win rates sont corrects même avec map_ui."""
        df = self._make_breakdown_df(with_map_ui=True)
        result = compute_map_breakdown(df)
        row_behe = result.filter(pl.col("map_name") == "Béhémoth FR")
        # Behemoth : 2 victoires (outcome=2) sur 2 matchs → win_rate = 1.0
        assert row_behe["win_rate"].item() == pytest.approx(1.0)
