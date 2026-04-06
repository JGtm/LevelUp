"""Tests pour la page Paramètres V2.

Couvre :
- render_settings_page : rendu global, sections, toggles
- _get_preserved_settings : champs sync préservés, lang/discord déplacés vers l'UI
- _build_settings_from_ui : lang, career_top_exclude_btb, discord sauvegardés
- _render_language_section : sélecteurs lang + timezone
- _render_backfill_section : correction bug disabled sur backfill_events
- _render_display_section : career_top_exclude_btb exposé
- _render_discord_section : URL et lang désactivés quand toggle off
"""

from __future__ import annotations

from unittest.mock import MagicMock, patch


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _base_mocks(ms, *, toggle=False, checkbox=False, selectbox="fr", slider=3):
    """Configure les mocks courants pour les tests settings."""
    ms.calls["toggle"].return_value = toggle
    ms.calls["checkbox"].return_value = checkbox
    ms.calls["selectbox"].return_value = selectbox
    ms.calls["slider"].return_value = slider
    ms.calls["text_input"].return_value = ""
    ms.calls["button"].return_value = False
    ms.set_columns_dynamic()


def _full_build_kwargs(**overrides):
    """Retourne un dict complet de kwargs pour _build_settings_from_ui."""
    from src.ui import AppSettings

    defaults = dict(
        settings=AppSettings(),
        lang="fr",
        user_timezone="Europe/Paris",
        backfill_enabled=False,
        backfill_medals=False,
        backfill_events=False,
        backfill_skill=False,
        backfill_personal_scores=False,
        backfill_performance_scores=True,
        backfill_aliases=False,
        backfill_lusr=True,
        backfill_weapons=False,
        refresh_clears_caches=False,
        normalize_mode_labels=True,
        career_top_exclude_btb=False,
        media_captures_base_dir="",
        media_tolerance_minutes=3,
        discord_notifications_enabled=False,
        discord_webhook_url="",
        discord_lang="fr",
    )
    defaults.update(overrides)
    return defaults


# ---------------------------------------------------------------------------
# Tests render_settings_page
# ---------------------------------------------------------------------------


class TestRenderSettingsPage:
    """Tests de la render function render_settings_page."""

    def _setup_mocks(self, ms):
        _base_mocks(ms)
        for c in ms.columns:
            c.button = MagicMock(return_value=False)

    def test_returns_settings(self, mock_st) -> None:
        """La function retourne les settings passés (sans clic Enregistrer)."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        self._setup_mocks(ms)

        with patch("src.ui.path_picker.directory_input", return_value=""):
            result = mod.render_settings_page(AppSettings())

        assert result is not None

    def test_sections_rendered(self, mock_st) -> None:
        """Vérifie que les 5 sections fixes + titre = au moins 6 appels st.subheader."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        self._setup_mocks(ms)

        with patch("src.ui.path_picker.directory_input", return_value=""):
            mod.render_settings_page(AppSettings())

        # Titre + Langue & Région + Backfill + Affichage + Médias + Discord
        assert ms.calls["subheader"].call_count >= 6
        # V2 n'utilise aucun expander
        assert ms.calls["expander"].call_count == 0

    def test_toggles_rendered(self, mock_st) -> None:
        """Vérifie que les 5 toggles sont rendus."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        self._setup_mocks(ms)

        with patch("src.ui.path_picker.directory_input", return_value=""):
            mod.render_settings_page(AppSettings())

        # backfill_enabled + normalize_mode_labels + career_top_exclude_btb
        # + refresh_clears_caches + discord_enabled = 5
        assert ms.calls["toggle"].call_count >= 5

    def test_custom_settings_values(self, mock_st) -> None:
        """Teste avec des settings non-default sans erreur."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        self._setup_mocks(ms)
        ms.calls["toggle"].return_value = True

        settings = AppSettings(
            lang="en",
            media_tolerance_minutes=10,
            spnkr_refresh_with_backfill=True,
            career_top_exclude_btb=True,
            discord_notifications_enabled=True,
        )

        with patch("src.ui.path_picker.directory_input", return_value=""):
            result = mod.render_settings_page(settings)

        assert result is not None


# ---------------------------------------------------------------------------
# Tests _get_preserved_settings
# ---------------------------------------------------------------------------


class TestGetPreservedSettings:
    """Vérifie que _get_preserved_settings préserve les bons champs."""

    def test_preserves_spnkr_max_matches(self) -> None:
        """spnkr_refresh_max_matches doit être préservé depuis les settings."""
        from src.ui import AppSettings
        from src.ui.pages.settings import _get_preserved_settings

        s = AppSettings(spnkr_refresh_max_matches=750)
        preserved = _get_preserved_settings(s)
        assert preserved["spnkr_refresh_max_matches"] == 750

    def test_preserves_spnkr_rps(self) -> None:
        """spnkr_refresh_rps doit être préservé depuis les settings."""
        from src.ui import AppSettings
        from src.ui.pages.settings import _get_preserved_settings

        s = AppSettings(spnkr_refresh_rps=8)
        preserved = _get_preserved_settings(s)
        assert preserved["spnkr_refresh_rps"] == 8

    def test_preserves_sync_booleans(self) -> None:
        """Les flags sync (on_start, on_manual_refresh, highlight_events) sont préservés."""
        from src.ui import AppSettings
        from src.ui.pages.settings import _get_preserved_settings

        s = AppSettings(
            spnkr_refresh_on_start=False,
            spnkr_refresh_on_manual_refresh=False,
            spnkr_refresh_with_highlight_events=True,
        )
        preserved = _get_preserved_settings(s)
        assert preserved["spnkr_refresh_on_start"] is False
        assert preserved["spnkr_refresh_on_manual_refresh"] is False
        assert preserved["spnkr_refresh_with_highlight_events"] is True

    def test_lang_not_in_preserved(self) -> None:
        """lang est désormais dans l'UI, pas dans les champs préservés."""
        from src.ui import AppSettings
        from src.ui.pages.settings import _get_preserved_settings

        preserved = _get_preserved_settings(AppSettings(lang="en"))
        assert "lang" not in preserved

    def test_discord_fields_not_in_preserved(self) -> None:
        """discord_* sont désormais dans l'UI, pas dans les champs préservés."""
        from src.ui import AppSettings
        from src.ui.pages.settings import _get_preserved_settings

        preserved = _get_preserved_settings(AppSettings())
        assert "discord_notifications_enabled" not in preserved
        assert "discord_webhook_url" not in preserved
        assert "discord_lang" not in preserved

    def test_enable_duckdb_analytics_preserved(self) -> None:
        """enable_duckdb_analytics doit être préservé (n'était pas hardcodé à True)."""
        from src.ui import AppSettings
        from src.ui.pages.settings import _get_preserved_settings

        s = AppSettings(enable_duckdb_analytics=False)
        preserved = _get_preserved_settings(s)
        assert preserved["enable_duckdb_analytics"] is False


# ---------------------------------------------------------------------------
# Tests _build_settings_from_ui
# ---------------------------------------------------------------------------


class TestBuildSettingsFromUI:
    """Vérifie que _build_settings_from_ui construit un AppSettings correct."""

    def test_lang_saved(self) -> None:
        """lang est correctement transmis dans l'AppSettings résultant."""
        from src.ui.pages.settings import _build_settings_from_ui

        result = _build_settings_from_ui(**_full_build_kwargs(lang="en"))
        assert result.lang == "en"

    def test_career_top_exclude_btb_saved(self) -> None:
        """career_top_exclude_btb est correctement transmis."""
        from src.ui.pages.settings import _build_settings_from_ui

        result = _build_settings_from_ui(**_full_build_kwargs(career_top_exclude_btb=True))
        assert result.career_top_exclude_btb is True

    def test_discord_fields_saved(self) -> None:
        """discord_notifications_enabled, webhook_url et discord_lang sont sauvegardés."""
        from src.ui.pages.settings import _build_settings_from_ui

        result = _build_settings_from_ui(
            **_full_build_kwargs(
                discord_notifications_enabled=True,
                discord_webhook_url="https://discord.com/api/webhooks/test",
                discord_lang="en",
            )
        )
        assert result.discord_notifications_enabled is True
        assert result.discord_webhook_url == "https://discord.com/api/webhooks/test"
        assert result.discord_lang == "en"

    def test_sync_params_preserved_from_settings(self) -> None:
        """max_matches et rps sont préservés depuis les settings, pas réinitialisés."""
        from src.ui import AppSettings
        from src.ui.pages.settings import _build_settings_from_ui

        s = AppSettings(spnkr_refresh_max_matches=999, spnkr_refresh_rps=12)
        result = _build_settings_from_ui(**_full_build_kwargs(settings=s))
        assert result.spnkr_refresh_max_matches == 999
        assert result.spnkr_refresh_rps == 12

    def test_backfill_flags_saved(self) -> None:
        """Tous les flags backfill sont correctement transmis."""
        from src.ui.pages.settings import _build_settings_from_ui

        result = _build_settings_from_ui(
            **_full_build_kwargs(
                backfill_enabled=True,
                backfill_medals=True,
                backfill_events=True,
                backfill_skill=True,
                backfill_weapons=True,
            )
        )
        assert result.spnkr_refresh_with_backfill is True
        assert result.spnkr_refresh_backfill_medals is True
        assert result.spnkr_refresh_backfill_events is True
        assert result.spnkr_refresh_backfill_skill is True
        assert result.spnkr_refresh_backfill_weapons is True


# ---------------------------------------------------------------------------
# Tests _render_language_section
# ---------------------------------------------------------------------------


class TestRenderLanguageSection:
    """Tests de la section Langue & Région."""

    def test_selectbox_called_twice(self, mock_st) -> None:
        """Deux selectbox doivent être rendus : langue et timezone."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        _base_mocks(ms)

        mod._render_language_section(AppSettings())

        assert ms.calls["selectbox"].call_count == 2

    def test_lang_selectbox_options(self, mock_st) -> None:
        """Le premier selectbox est bien pour la langue (options fr/en)."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        _base_mocks(ms)

        mod._render_language_section(AppSettings(lang="en"))

        lang_call = ms.calls["selectbox"].call_args_list[0]
        assert lang_call.kwargs["options"] == ["fr", "en"]
        # Settings lang="en" → index 1
        assert lang_call.kwargs["index"] == 1

    def test_returns_lang_and_timezone(self, mock_st) -> None:
        """La fonction retourne un tuple (str, str)."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        ms.calls["selectbox"].return_value = "fr"

        lang, tz = mod._render_language_section(AppSettings())

        assert isinstance(lang, str)
        assert isinstance(tz, str)


# ---------------------------------------------------------------------------
# Tests _render_backfill_section
# ---------------------------------------------------------------------------


class TestRenderBackfillSection:
    """Tests de la section Backfill — dont la correction du bug disabled."""

    def test_backfill_events_has_disabled_param(self, mock_st) -> None:
        """backfill_events doit avoir le paramètre disabled (bug corrigé)."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        _base_mocks(ms, toggle=False)

        mod._render_backfill_section(AppSettings())

        calls_with_disabled = [
            c for c in ms.calls["checkbox"].call_args_list if "disabled" in c.kwargs
        ]
        # 7 checkboxes sur 8 ont disabled (sauf performance_scores)
        assert len(calls_with_disabled) == 7

    def test_all_checkboxes_disabled_when_toggle_off(self, mock_st) -> None:
        """Quand backfill_enabled=False, tous les checkboxes avec disabled l'ont à True."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        _base_mocks(ms, toggle=False)

        mod._render_backfill_section(AppSettings(spnkr_refresh_with_backfill=False))

        calls_with_disabled = [
            c for c in ms.calls["checkbox"].call_args_list if "disabled" in c.kwargs
        ]
        assert all(c.kwargs["disabled"] is True for c in calls_with_disabled)

    def test_checkboxes_enabled_when_toggle_on(self, mock_st) -> None:
        """Quand backfill_enabled=True, les checkboxes avec disabled l'ont à False."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        _base_mocks(ms, toggle=True)

        mod._render_backfill_section(AppSettings(spnkr_refresh_with_backfill=True))

        calls_with_disabled = [
            c for c in ms.calls["checkbox"].call_args_list if "disabled" in c.kwargs
        ]
        assert all(c.kwargs["disabled"] is False for c in calls_with_disabled)

    def test_returns_all_nine_flags(self, mock_st) -> None:
        """La fonction retourne un dict avec les 9 clés attendues."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        _base_mocks(ms)

        result = mod._render_backfill_section(AppSettings())

        expected_keys = {
            "backfill_enabled",
            "backfill_medals",
            "backfill_events",
            "backfill_skill",
            "backfill_personal_scores",
            "backfill_performance_scores",
            "backfill_aliases",
            "backfill_lusr",
            "backfill_weapons",
        }
        assert set(result.keys()) == expected_keys
        assert all(isinstance(v, bool) for v in result.values())


# ---------------------------------------------------------------------------
# Tests _render_display_section
# ---------------------------------------------------------------------------


class TestRenderDisplaySection:
    """Tests de la section Affichage."""

    def test_three_toggles_rendered(self, mock_st) -> None:
        """normalize_mode_labels + career_top_exclude_btb + refresh_clears_caches."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        _base_mocks(ms)

        mod._render_display_section(AppSettings())

        assert ms.calls["toggle"].call_count == 3

    def test_career_top_exclude_btb_uses_settings_value(self, mock_st) -> None:
        """Le toggle career_top_exclude_btb lit la valeur dans les settings."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        _base_mocks(ms)

        mod._render_display_section(AppSettings(career_top_exclude_btb=True))

        toggle_calls = ms.calls["toggle"].call_args_list
        # 2e toggle = career_top_exclude_btb
        career_call = toggle_calls[1]
        assert career_call.kwargs["value"] is True

    def test_returns_three_bools(self, mock_st) -> None:
        """La fonction retourne un tuple (bool, bool, bool)."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        _base_mocks(ms, toggle=True)

        result = mod._render_display_section(AppSettings())

        assert len(result) == 3
        assert all(isinstance(v, bool) for v in result)


# ---------------------------------------------------------------------------
# Tests _render_discord_section
# ---------------------------------------------------------------------------


class TestRenderDiscordSection:
    """Tests de la section Notifications Discord."""

    def test_url_disabled_when_toggle_off(self, mock_st) -> None:
        """Le champ URL webhook doit être disabled quand discord_enabled=False."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        _base_mocks(ms, toggle=False)

        mod._render_discord_section(AppSettings(discord_notifications_enabled=False))

        url_call = ms.calls["text_input"].call_args_list[0]
        assert url_call.kwargs.get("disabled") is True

    def test_lang_selectbox_disabled_when_toggle_off(self, mock_st) -> None:
        """Le selectbox langue Discord doit être disabled quand discord_enabled=False."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        _base_mocks(ms, toggle=False)

        mod._render_discord_section(AppSettings(discord_notifications_enabled=False))

        selectbox_call = ms.calls["selectbox"].call_args_list[0]
        assert selectbox_call.kwargs.get("disabled") is True

    def test_url_enabled_when_toggle_on(self, mock_st) -> None:
        """Le champ URL webhook doit être enabled quand discord_enabled=True."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        _base_mocks(ms, toggle=True)

        mod._render_discord_section(AppSettings(discord_notifications_enabled=True))

        url_call = ms.calls["text_input"].call_args_list[0]
        assert url_call.kwargs.get("disabled") is False

    def test_returns_three_values(self, mock_st) -> None:
        """La fonction retourne (bool, str, str)."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        _base_mocks(ms)
        ms.calls["toggle"].return_value = False
        ms.calls["text_input"].return_value = ""
        ms.calls["selectbox"].return_value = "fr"

        enabled, url, lang = mod._render_discord_section(AppSettings())

        assert isinstance(enabled, bool)
        assert isinstance(url, str)
        assert isinstance(lang, str)

    def test_existing_webhook_url_prefilled(self, mock_st) -> None:
        """L'URL webhook existante est utilisée comme valeur par défaut."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        _base_mocks(ms)
        ms.calls["text_input"].return_value = "https://discord.com/api/webhooks/existing"

        mod._render_discord_section(
            AppSettings(discord_webhook_url="https://discord.com/api/webhooks/existing")
        )

        url_call = ms.calls["text_input"].call_args_list[0]
        assert "https://discord.com/api/webhooks/existing" in url_call.kwargs["value"]
