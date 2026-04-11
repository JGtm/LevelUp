"""Tests de non-regression pour le shell V7."""

from __future__ import annotations

import logging
from datetime import date
from unittest.mock import MagicMock


class TestV7SectionMapping:
    """Verifie les mappings de navigation V7."""

    def test_normalize_v7_section_accepts_known_slug(self) -> None:
        """Une section connue doit etre normalisee en minuscule."""
        from src.app.page_router import normalize_v7_section

        assert normalize_v7_section(" Stats ") == "stats"

    def test_normalize_v7_section_defaults_to_home(self) -> None:
        """Une section inconnue doit retomber sur home."""
        from src.app.page_router import normalize_v7_section

        assert normalize_v7_section("unknown") == "home"

    def test_map_legacy_page_to_v7_location_from_slug(self) -> None:
        """Un slug legacy Stats doit conserver sa sous-vue."""
        from src.app.page_router import map_legacy_page_to_v7_location

        assert map_legacy_page_to_v7_location("session_compare") == ("stats", "session_compare")

    def test_map_legacy_page_to_v7_location_from_url_path(self) -> None:
        """Un url_path legacy doit etre mappe vers la bonne section V7."""
        from src.app.page_router import map_legacy_page_to_v7_location

        assert map_legacy_page_to_v7_location("history") == ("stats", "match_history")

    def test_map_legacy_page_to_v7_location_from_legacy_label(self) -> None:
        """Un libelle legacy traduit doit etre compris par la V7."""
        from src.app.page_router import map_legacy_page_to_v7_location

        assert map_legacy_page_to_v7_location("Mes coéquipiers") == ("squad", None)

    def test_map_legacy_page_to_v7_location_unknown_defaults_home(self) -> None:
        """Une page inconnue doit retomber sur home sans sous-vue."""
        from src.app.page_router import map_legacy_page_to_v7_location

        assert map_legacy_page_to_v7_location("does-not-exist") == ("home", None)


class TestV7FilterChipHelpers:
    """Verifie les helpers compacts du bandeau de filtres V7."""

    def test_summarize_values_handles_empty_inputs(self) -> None:
        """Les valeurs vides ne doivent pas produire de texte."""
        from src.ui.layout.filter_chips import _summarize_values

        assert _summarize_values(None) is None
        assert _summarize_values("   ") is None
        assert _summarize_values([]) is None

    def test_summarize_values_limits_long_lists(self) -> None:
        """Les listes longues doivent etre compactees."""
        from src.ui.layout.filter_chips import _summarize_values

        result = _summarize_values(["Arena", "Ranked", "Doubles"])
        assert result == "Arena, Ranked +1"

    def test_format_period_formats_date_range(self) -> None:
        """Une plage de dates doit etre formatee au format court FR."""
        from src.ui.layout.filter_chips import _format_period

        result = _format_period(date(2026, 4, 1), date(2026, 4, 15))
        assert result == "01/04/2026 -> 15/04/2026"

    def test_format_period_supports_open_range(self) -> None:
        """Une borne unique doit etre retournee telle quelle apres trim."""
        from src.ui.layout.filter_chips import _format_period

        assert _format_period(None, "  2026-04  ") == "2026-04"


class TestV7ThemeLoader:
    """Verifie le chargement du CSS V7."""

    def test_load_v7_theme_css_wraps_stylesheet(self) -> None:
        """Le theme V7 doit charger une balise style complete."""
        from src.ui.theme import load_v7_theme_css

        css = load_v7_theme_css()
        assert css.startswith("<style>")
        assert css.endswith("</style>")
        assert "--v7-bg-main" in css


class TestV7LoggingContracts:
    """Verifie la presence des loggers V7 et les logs critiques."""

    def test_header_l1_logger_defined(self) -> None:
        """header_l1 doit exposer un logger module-level."""
        import src.ui.layout.header_l1 as mod

        assert hasattr(mod, "logger")
        assert isinstance(mod.logger, logging.Logger)
        assert mod.logger.name == "src.ui.layout.header_l1"

    def test_header_l2_logger_defined(self) -> None:
        """header_l2 doit exposer un logger module-level."""
        import src.ui.layout.header_l2 as mod

        assert hasattr(mod, "logger")
        assert isinstance(mod.logger, logging.Logger)
        assert mod.logger.name == "src.ui.layout.header_l2"

    def test_v7_sections_logger_defined(self) -> None:
        """v7_sections doit exposer un logger module-level."""
        import src.ui.pages.v7_sections as mod

        assert hasattr(mod, "logger")
        assert isinstance(mod.logger, logging.Logger)
        assert mod.logger.name == "src.ui.pages.v7_sections"

    def test_clear_all_filters_logs_count(self, monkeypatch, caplog) -> None:
        """Le reset global des filtres doit journaliser le nombre de cles effacees."""
        import src.ui.layout.header_l2 as mod

        session_state = {"playlist": ["Arena"], "mode": "Ranked", "keep": "yes"}
        monkeypatch.setattr(mod.st, "session_state", session_state)
        monkeypatch.setattr(
            mod,
            "get_all_filter_keys_to_clear",
            lambda _state: ["playlist", "missing", "mode"],
        )

        with caplog.at_level(logging.INFO, logger="src.ui.layout.header_l2"):
            cleared = mod._clear_all_filters()

        assert cleared == 2
        assert session_state == {"keep": "yes"}
        assert any(
            "Reset filtres V7" in rec.message and "2" in rec.message for rec in caplog.records
        )

    def test_apply_player_change_logs_target_player(self, monkeypatch, caplog) -> None:
        """Le changement de joueur doit produire un log avec l'ancien et le nouveau joueur."""
        import src.ui.layout.header_l1 as mod

        session_state: dict[str, str] = {}
        apply_filter_preferences = MagicMock()
        persist_browser_prefs = MagicMock()
        monkeypatch.setattr(mod.st, "session_state", session_state)
        monkeypatch.setattr(mod, "_reset_player_filters", lambda *_a, **_kw: None)
        monkeypatch.setattr(mod, "apply_filter_preferences", apply_filter_preferences)
        monkeypatch.setattr(mod, "persist_browser_prefs", persist_browser_prefs)
        monkeypatch.setattr(mod, "get_gamertag_from_duckdb_v4_path", lambda _path: "NewGT")

        new_db_path = "data/players/NewGT/stats.duckdb"
        with caplog.at_level(logging.INFO, logger="src.ui.layout.header_l1"):
            db_path, xuid = mod._apply_player_change(
                current_db_path="data/players/OldGT/stats.duckdb",
                current_xuid="OldGT",
                new_db_path=new_db_path,
                new_xuid=None,
            )

        assert db_path == new_db_path
        assert xuid == "NewGT"
        assert session_state[mod.SK.DB_PATH] == new_db_path
        assert session_state[mod.SK.XUID_INPUT] == "NewGT"
        assert session_state[mod.SK.WAYPOINT_PLAYER] == "NewGT"
        apply_filter_preferences.assert_called_once_with("NewGT", new_db_path)
        persist_browser_prefs.assert_called_once_with(last_gamertag="NewGT", last_db_path="NewGT")
        assert any("OldGT" in rec.message and "NewGT" in rec.message for rec in caplog.records)

    def test_render_segmented_view_logs_secondary_navigation(self, monkeypatch, caplog) -> None:
        """La navigation secondaire V7 doit etre journalisee lors d'un changement."""
        import src.ui.pages.v7_sections as mod

        labels = {"timeseries": "Series", "match_history": "Historique"}
        session_state = {"v7_stats_view": "timeseries"}
        monkeypatch.setattr(mod.st, "session_state", session_state)
        monkeypatch.setattr(mod.st, "segmented_control", lambda *_a, **_kw: "Historique")

        with caplog.at_level(logging.INFO, logger="src.ui.pages.v7_sections"):
            selected = mod._render_segmented_view(
                state_key="v7_stats_view",
                options=["timeseries", "match_history"],
                label_fn=labels.__getitem__,
            )

        assert selected == "match_history"
        assert session_state["v7_stats_view"] == "match_history"
        assert any(
            "v7_stats_view" in rec.message
            and "timeseries" in rec.message
            and "match_history" in rec.message
            for rec in caplog.records
        )
