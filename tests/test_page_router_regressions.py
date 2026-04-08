"""Tests de non-régression du routeur de pages Streamlit.

Valident :
- Les clés i18n des nouvelles sections (KPI bloc dans teammates, onglets timeseries)
- La présence du logger dans les modules de page modifiés
- Le logging dans render_teammates_page
"""

from __future__ import annotations


class TestTeammatesI18nKeys:
    """Vérifie que les nouvelles clés i18n pour la page Coéquipiers sont définies."""

    def test_tm_my_stats_section_exists_fr(self) -> None:
        """La clé tm_my_stats_section doit exister en français."""
        from src.ui.i18n.pages.teammates import STRINGS

        assert "tm_my_stats_section" in STRINGS
        assert "fr" in STRINGS["tm_my_stats_section"]
        assert STRINGS["tm_my_stats_section"]["fr"]

    def test_tm_squad_section_exists_fr(self) -> None:
        """La clé tm_squad_section doit exister en français."""
        from src.ui.i18n.pages.teammates import STRINGS

        assert "tm_squad_section" in STRINGS
        assert "fr" in STRINGS["tm_squad_section"]
        assert STRINGS["tm_squad_section"]["fr"]

    def test_tm_my_stats_section_exists_en(self) -> None:
        """La clé tm_my_stats_section doit exister en anglais."""
        from src.ui.i18n.pages.teammates import STRINGS

        assert "en" in STRINGS["tm_my_stats_section"]
        assert STRINGS["tm_my_stats_section"]["en"]

    def test_tm_squad_section_exists_en(self) -> None:
        """La clé tm_squad_section doit exister en anglais."""
        from src.ui.i18n.pages.teammates import STRINGS

        assert "en" in STRINGS["tm_squad_section"]
        assert STRINGS["tm_squad_section"]["en"]


class TestTeammatesLogging:
    """Vérifie la présence et la configuration du logger dans teammates.py."""

    def test_logger_defined(self) -> None:
        """teammates.py doit définir un logger module-level."""
        import logging

        import src.ui.pages.teammates as mod

        assert hasattr(mod, "logger")
        assert isinstance(mod.logger, logging.Logger)

    def test_logger_name(self) -> None:
        """Le logger doit être nommé d'après le module."""
        import src.ui.pages.teammates as mod

        assert mod.logger.name == "src.ui.pages.teammates"


class TestTimeseriesTabLabels:
    """Vérifie que les 5 clés d'onglets timeseries sont bien définies en i18n."""

    TAB_KEYS = [
        "ts_tab_kda",
        "ts_tab_maps",
        "ts_tab_distributions",
        "ts_tab_advanced",
        "ts_tab_progression",
    ]

    def test_all_tab_keys_exist(self) -> None:
        """Toutes les clés d'onglets doivent exister dans les STRINGS timeseries."""
        from src.ui.i18n.pages.timeseries import STRINGS

        for key in self.TAB_KEYS:
            assert key in STRINGS, f"Clé d'onglet manquante : {key}"

    def test_all_tab_keys_have_fr_and_en(self) -> None:
        """Chaque clé d'onglet doit avoir une traduction FR et EN."""
        from src.ui.i18n.pages.timeseries import STRINGS

        for key in self.TAB_KEYS:
            entry = STRINGS[key]
            assert "fr" in entry and entry["fr"], f"Traduction FR manquante pour {key}"
            assert "en" in entry and entry["en"], f"Traduction EN manquante pour {key}"
