"""Tests unitaires pour src.app.i18n_columns.add_i18n_display_columns.

Vérifie :
- Priorité aux colonnes map_name_fr/playlist_name_fr sur les colonnes EN
- mode_ui calculé depuis pair_name_fr via _strip_mode_map_suffix (pur, sans DB)
- Idempotence (colonnes existantes non recalculées)
- Comportement en lang="en" (passthrough)
- DataFrame vide retourné intact
"""

from __future__ import annotations

import polars as pl

from src.app.i18n_columns import add_i18n_display_columns


def _base_df(**extra) -> pl.DataFrame:
    data = {
        "match_id": ["m1", "m2"],
        "map_name": ["Cliffhanger", "Aquarius"],
        "pair_name": ["Quick Play:Slayer", "Ranked:Slayer"],
        "playlist_name": ["Quick Play", "Ranked Arena"],
        **extra,
    }
    return pl.DataFrame(data)


class TestAddI18nDisplayColumns:
    def test_adds_map_ui_from_map_name_fr(self) -> None:
        df = _base_df(map_name_fr=["Dévissage", "Aquarius"])
        out = add_i18n_display_columns(df, lang="fr")
        assert "map_ui" in out.columns
        assert out["map_ui"].to_list() == ["Dévissage", "Aquarius"]

    def test_adds_playlist_ui_from_playlist_name_fr(self) -> None:
        df = _base_df(playlist_name_fr=["Jeu rapide", "Arène classée"])
        out = add_i18n_display_columns(df, lang="fr")
        assert "playlist_ui" in out.columns
        assert out["playlist_ui"].to_list() == ["Jeu rapide", "Arène classée"]

    def test_mode_ui_computed_from_pair_name_fr(self) -> None:
        """mode_ui est calculé via _strip_mode_map_suffix : supprime ' on Carte' + Forge/Ranked."""
        df = _base_df(pair_name_fr=["Slayer on Cliffhanger - Forge", "Slayer - Ranked"])
        out = add_i18n_display_columns(df, lang="fr")
        assert "mode_ui" in out.columns
        assert out["mode_ui"].to_list() == ["Slayer", "Slayer"]

    def test_mode_ui_strips_on_suffix_only(self) -> None:
        """Vérifie que seul le ' on MapName' est supprimé, pas les variantes."""
        df = _base_df(pair_name_fr=["Capture the Flag on Bazaar", "Slayer"])
        out = add_i18n_display_columns(df, lang="fr")
        assert out["mode_ui"].to_list() == ["Capture the Flag", "Slayer"]

    def test_mode_ui_fallback_to_pair_name_when_fr_absent(self) -> None:
        """Sans pair_name_fr, mode_ui utilise pair_name (passthrough)."""
        df = _base_df()  # pas de pair_name_fr
        out = add_i18n_display_columns(df, lang="fr")
        assert "mode_ui" in out.columns
        assert out["mode_ui"].to_list() == ["Quick Play:Slayer", "Ranked:Slayer"]

    def test_mode_ui_idempotent(self) -> None:
        """Si mode_ui existe déjà, il n'est pas recalculé."""
        df = _base_df(pair_name_fr=["Slayer on Bazaar", "Slayer"], mode_ui=["EXISTING", "EXISTING"])
        out = add_i18n_display_columns(df, lang="fr")
        assert out["mode_ui"].to_list() == ["EXISTING", "EXISTING"]

    def test_map_ui_fallback_to_en_when_fr_col_absent(self) -> None:
        df = _base_df()  # pas de colonnes _fr
        out = add_i18n_display_columns(df, lang="fr")
        assert out["map_ui"].to_list() == ["Cliffhanger", "Aquarius"]
        assert out["playlist_ui"].to_list() == ["Quick Play", "Ranked Arena"]

    def test_coalesce_null_map_name_fr_with_en(self) -> None:
        """Si map_name_fr est null pour une ligne, fallback sur map_name."""
        df = _base_df(map_name_fr=[None, "Aquarius"])
        out = add_i18n_display_columns(df, lang="fr")
        assert out["map_ui"].to_list() == ["Cliffhanger", "Aquarius"]

    def test_idempotent_map_ui(self) -> None:
        """Si map_ui existe déjà, il n'est pas recalculé."""
        df = _base_df(map_name_fr=["Dévissage", "Aquarius"], map_ui=["EXISTING", "EXISTING"])
        out = add_i18n_display_columns(df, lang="fr")
        assert out["map_ui"].to_list() == ["EXISTING", "EXISTING"]

    def test_idempotent_playlist_ui(self) -> None:
        df = _base_df(playlist_name_fr=["Jeu rapide", "Classé"], playlist_ui=["OLD", "OLD"])
        out = add_i18n_display_columns(df, lang="fr")
        assert out["playlist_ui"].to_list() == ["OLD", "OLD"]

    def test_lang_en_uses_passthrough(self) -> None:
        """En lang='en', les colonnes EN sont utilisées directement."""
        df = _base_df(map_name_fr=["Dévissage", "Aquarius"])
        out = add_i18n_display_columns(df, lang="en")
        assert out["map_ui"].to_list() == ["Cliffhanger", "Aquarius"]

    def test_empty_df_returned_intact(self) -> None:
        df = pl.DataFrame(schema={"map_name": pl.Utf8, "pair_name": pl.Utf8})
        out = add_i18n_display_columns(df, lang="fr")
        assert out.is_empty()
        assert "map_ui" not in out.columns

    def test_missing_source_cols_no_error(self) -> None:
        """Si map_name/playlist_name absents, pas d'erreur."""
        df = pl.DataFrame({"match_id": ["m1"]})
        out = add_i18n_display_columns(df, lang="fr")
        assert "map_ui" not in out.columns
        assert "playlist_ui" not in out.columns
