"""Tests de non-regression pour le shell V7."""

from __future__ import annotations

import logging
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

    def test_summarize_collection_formats_set(self) -> None:
        """Un set doit etre trie et tronque correctement."""
        from src.ui.layout.filter_chips import _summarize_collection

        result = _summarize_collection({"Ranked Slayer", "CTF", "Fiesta", "Attrition"})
        assert result is not None
        assert "+1" in result  # 4 items, max=3 -> +1

    def test_summarize_collection_none_on_empty(self) -> None:
        """Un set vide doit retourner None."""
        from src.ui.layout.filter_chips import _summarize_collection

        assert _summarize_collection(set()) is None
        assert _summarize_collection([]) is None


class TestV7ThemeLoader:
    """Verifie le chargement du CSS V7."""

    def test_load_v7_theme_css_wraps_stylesheet(self) -> None:
        """Le theme V7 doit charger une balise style complete."""
        from src.ui.theme import load_v7_theme_css

        css = load_v7_theme_css()
        assert css.startswith("<style>")
        assert css.endswith("</style>")
        assert "--v7-bg-main" in css

    def test_load_v7_theme_css_includes_workspace_surface_overrides(self) -> None:
        """Le theme V7 doit surcharger les primitives de contenu legacy."""
        from src.ui.theme import load_v7_theme_css

        css = load_v7_theme_css()
        assert '[data-testid="stTabs"] [data-baseweb="tab-panel"]' in css
        assert '[data-testid="stVerticalBlockBorderWrapper"]' in css
        assert '[data-testid="stExpander"]' in css

    def test_load_v7_theme_css_includes_widget_toolbar_overrides(self) -> None:
        """Le theme V7 doit aussi couvrir les widgets internes encore visibles."""
        from src.ui.theme import load_v7_theme_css

        css = load_v7_theme_css()
        assert 'div[data-baseweb="tag"]' in css
        assert '[data-testid="stCheckbox"]' in css
        assert '[data-testid="stSlider"]' in css
        assert ".v7-toolbar-divider" in css


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


class TestV7SessionContextBar:
    """Verifie la logique de navigation de session dans la L2 V7."""

    def test_normalize_scope_value_prefers_known_option(self) -> None:
        """Une valeur inconnue doit retomber sur la session la plus recente."""
        from src.ui.layout.header_l2 import _normalize_scope_value

        assert _normalize_scope_value("inconnue", ["Session #3", "Session #2"]) == "Session #3"
        assert _normalize_scope_value("(toutes)", ["Session #3"]) == "(toutes)"

    def test_previous_session_target_steps_back_in_order(self) -> None:
        """Le bouton precedent doit avancer vers une session plus ancienne."""
        from src.ui.layout.header_l2 import _get_previous_session_target

        options = ["Session #5", "Session #4", "Session #3"]
        assert _get_previous_session_target("Session #5", options) == "Session #4"
        assert _get_previous_session_target("Session #3", options) == "Session #3"
        assert _get_previous_session_target("(toutes)", options) == "Session #5"

    def test_apply_session_scope_updates_stats_keys(self, monkeypatch) -> None:
        """Le scope Stats doit poser la session solo active et vider l'escouade."""
        import src.ui.layout.header_l2 as mod

        session_state = {}
        monkeypatch.setattr(mod.st, "session_state", session_state)

        mod._apply_session_scope("stats", "Session #8")

        assert session_state[mod.SK.FILTER_MODE] == "Sessions"
        assert session_state[mod.SK.PICKED_SOLO_SESSION_LABEL] == "Session #8"
        assert session_state[mod.SK.PICKED_SQUAD_SESSION_LABEL] == "(toutes)"
        assert session_state[mod.SK.PICKED_SESSION_LABEL] == "Session #8"
        assert session_state[mod.SK.PICKED_SESSIONS] == ["Session #8"]

    def test_apply_session_scope_updates_squad_keys(self, monkeypatch) -> None:
        """Le scope Escouade doit poser la session escouade active et vider le solo."""
        import src.ui.layout.header_l2 as mod

        session_state = {}
        monkeypatch.setattr(mod.st, "session_state", session_state)

        mod._apply_session_scope("squad", "Carnage #2")

        assert session_state[mod.SK.FILTER_MODE] == "Sessions"
        assert session_state[mod.SK.PICKED_SQUAD_SESSION_LABEL] == "Carnage #2"
        assert session_state[mod.SK.PICKED_SOLO_SESSION_LABEL] == "(toutes)"
        assert session_state[mod.SK.PICKED_SESSION_LABEL] == "Carnage #2"
        assert session_state[mod.SK.PICKED_SESSIONS] == ["Carnage #2"]
