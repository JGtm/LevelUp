"""Tests pour la page Paramètres V3.

Couvre :
- render_settings_page : rendu global, sections, toggles
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
        # V3 utilise 1 expander (section backfill)
        assert ms.calls["expander"].call_count == 1

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
# Tests _render_language_section
# ---------------------------------------------------------------------------
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


# ---------------------------------------------------------------------------
# Tests _render_display_section
# ---------------------------------------------------------------------------


class TestRenderDisplaySection:
    """Tests de la section Affichage."""

    def test_three_toggles_rendered(self, mock_st) -> None:
        """normalize_mode_labels + show_hints + show_records + career_top_exclude_btb + refresh_clears_caches."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        _base_mocks(ms)

        mod._render_display_section(AppSettings())

        assert ms.calls["toggle"].call_count == 5

    def test_career_top_exclude_btb_uses_settings_value(self, mock_st) -> None:
        """Le toggle career_top_exclude_btb lit la valeur dans les settings."""
        from src.ui import AppSettings
        from src.ui.pages import settings as mod

        ms = mock_st(mod)
        _base_mocks(ms)

        mod._render_display_section(AppSettings(career_top_exclude_btb=True))

        toggle_calls = ms.calls["toggle"].call_args_list
        # 4e toggle = career_top_exclude_btb (après normalize_mode_labels, show_hints et show_records)
        career_call = toggle_calls[3]
        assert career_call.kwargs["value"] is True


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
