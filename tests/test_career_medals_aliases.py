"""Tests pour src/ui/career_ranks.py, src/ui/medals.py et src/ui/aliases.py."""

from __future__ import annotations

import os
from unittest.mock import patch

# ============================================================================
# career_ranks.py
# ============================================================================
from src.ui.career_ranks import (
    CareerRankInfo,
    format_career_rank_label_fr,
)


class TestFormatCareerRankLabelFr:
    def test_recruit(self):
        assert format_career_rank_label_fr(tier=None, title="Recruit", grade=None) == "Recrue"

    def test_hero(self):
        assert format_career_rank_label_fr(tier=None, title="Hero", grade=None) == "Héros"

    def test_private_silver_2(self):
        result = format_career_rank_label_fr(tier="Silver", title="Private", grade="2")
        assert result == "Soldat - Argent II"

    def test_lt_colonel_gold_1(self):
        result = format_career_rank_label_fr(tier="Gold", title="Lt Colonel", grade="1")
        assert result == "Lieutenant-colonel - Or I"

    def test_brigadier_general_gold_3(self):
        result = format_career_rank_label_fr(tier="Gold", title="Brigadier General", grade="3")
        assert result == "Général de brigade - Or III"

    def test_title_only(self):
        result = format_career_rank_label_fr(tier=None, title="Captain", grade=None)
        assert result == "Capitaine"

    def test_title_and_tier_no_grade(self):
        result = format_career_rank_label_fr(tier="Bronze", title="Sergeant", grade=None)
        assert result == "Sergent - Bronze"

    def test_unknown_title(self):
        result = format_career_rank_label_fr(tier="Diamond", title="Unknown", grade="1")
        assert result == "Unknown - Diamant I"

    def test_empty_title(self):
        assert format_career_rank_label_fr(tier=None, title=None, grade=None) == ""

    def test_all_tiers_translated(self):
        """Vérifie que tous les tiers sont traduits."""
        for en, fr in [
            ("Bronze", "Bronze"),
            ("Silver", "Argent"),
            ("Gold", "Or"),
            ("Platinum", "Platine"),
            ("Diamond", "Diamant"),
            ("Onyx", "Onyx"),
        ]:
            result = format_career_rank_label_fr(tier=en, title="Private", grade="1")
            assert fr in result

    def test_all_titles_translated(self):
        """Vérifie quelques titres principaux."""
        for en, fr in [
            ("Cadet", "Cadet"),
            ("Lieutenant", "Lieutenant"),
            ("Colonel", "Colonel"),
            ("General", "Général"),
        ]:
            result = format_career_rank_label_fr(tier="Bronze", title=en, grade="1")
            assert fr in result


class TestCareerRankInfoProperties:
    def _make_info(self, **kwargs):
        defaults = {
            "rank_number": 1,
            "title": "Private",
            "subtitle": "Silver",
            "tier": "2",
            "xp_required": 1000,
            "icon_path_remote": "path/icon.png",
        }
        defaults.update(kwargs)
        return CareerRankInfo(**defaults)

    def test_full_label(self):
        info = self._make_info()
        label = info.full_label
        assert "Silver" in label
        assert "Private" in label

    def test_full_label_no_subtitle(self):
        info = self._make_info(subtitle=None, tier=None)
        assert info.full_label == "Private"

    def test_full_label_fr(self):
        info = self._make_info()
        label = info.full_label_fr
        assert "Soldat" in label  # Private → Soldat

    def test_display_label(self):
        info = self._make_info()
        assert "Private" in info.display_label

    def test_display_label_fr(self):
        info = self._make_info()
        assert "Soldat" in info.display_label_fr

    def test_recruit(self):
        info = self._make_info(title="Recruit", subtitle=None, tier=None)
        assert info.full_label_fr == "Recrue"


# ============================================================================
# medals.py — fonctions testables
# ============================================================================

from src.ui.medals import get_local_medals_icons_dir, get_medals_cache_dir


class TestMedalsCacheDir:
    def test_default_returns_string(self):
        result = get_medals_cache_dir()
        assert isinstance(result, str)

    def test_override_via_env(self, monkeypatch):
        monkeypatch.setenv("OPENSPARTAN_MEDALS_CACHE", "/custom/path")
        assert get_medals_cache_dir() == "/custom/path"

    def test_local_icons_dir(self):
        result = get_local_medals_icons_dir()
        assert isinstance(result, str)
        assert result.endswith(os.path.join("static", "medals", "icons"))


# ============================================================================
# aliases.py — fonctions testables
# ============================================================================

from src.ui.aliases import (
    _is_duckdb_file as aliases_is_duckdb,
)
from src.ui.aliases import (
    _safe_mtime,
    display_name_from_xuid,
    get_xuid_aliases,
)


class TestAliasesIsDuckdbFile:
    def test_duckdb(self):
        assert aliases_is_duckdb("test.duckdb") is True

    def test_db(self):
        assert aliases_is_duckdb("test.db") is False


class TestSafeMtime:
    def test_nonexistent(self):
        assert _safe_mtime("/nonexistent/file.txt") is None

    def test_existing_file(self, tmp_path):
        f = tmp_path / "test.txt"
        f.write_text("hello")
        result = _safe_mtime(str(f))
        assert isinstance(result, float)


class TestGetXuidAliasesDbOnly:
    """Vérifie que get_xuid_aliases n'utilise plus xuid_aliases.json (v5.2)."""

    def test_uses_db_when_db_path_given(self):
        """Quand db_path est fourni, la DB est consultée."""
        with patch("src.ui.aliases.load_aliases_from_db", return_value={"xuid1": "GT_From_DB"}):
            result = get_xuid_aliases(db_path="some.duckdb")
        assert result.get("xuid1") == "GT_From_DB"

    def test_returns_only_default_when_no_db_path(self, monkeypatch):
        """Sans db_path, seuls XUID_ALIASES_DEFAULT sont retournés."""
        monkeypatch.setattr("src.ui.aliases.XUID_ALIASES_DEFAULT", {})
        result = get_xuid_aliases()
        assert result == {}

    def test_json_file_has_no_effect(self, tmp_path, monkeypatch):
        """Un fichier JSON présent dans le répertoire est complètement ignoré."""
        json_file = tmp_path / "xuid_aliases.json"
        json_file.write_text('{"json_xuid": "JSON_Gamertag"}')
        monkeypatch.setattr("src.ui.aliases.XUID_ALIASES_DEFAULT", {})
        # Pas de db_path → seul DEFAULT est utilisé, pas le JSON
        result = get_xuid_aliases()
        assert "json_xuid" not in result

    def test_db_has_priority_over_default(self, monkeypatch):
        """La DB écrase XUID_ALIASES_DEFAULT si la même clé existe."""
        monkeypatch.setattr("src.ui.aliases.XUID_ALIASES_DEFAULT", {"shared_xuid": "Default_GT"})
        with patch("src.ui.aliases.load_aliases_from_db", return_value={"shared_xuid": "DB_GT"}):
            result = get_xuid_aliases(db_path="some.duckdb")
        assert result["shared_xuid"] == "DB_GT"


class TestDisplayNameFromXuid:
    def test_known_xuid(self):
        """Résolution via get_xuid_aliases (mocké)."""
        with patch("src.ui.aliases.get_xuid_aliases", return_value={"1234567890": "KnownGT"}):
            result = display_name_from_xuid("1234567890")
        assert result == "KnownGT"

    def test_unknown_xuid(self):
        """XUID inconnu → retourne le XUID brut."""
        with patch("src.ui.aliases.get_xuid_aliases", return_value={}):
            result = display_name_from_xuid("9999999999")
        assert result == "9999999999"

    def test_xuid_format_normalization(self):
        """Le format xuid(123) est normalisé avant lookup."""
        with patch("src.ui.aliases.get_xuid_aliases", return_value={"123": "NormGT"}):
            result = display_name_from_xuid("xuid(123)")
        assert result == "NormGT"
