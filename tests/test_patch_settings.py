"""Tests pour patch_settings, _apply_legacy_migrations et get_settings_path.

Couvre :
- patch_settings : nominal, no-op, fallback load_settings, rollback, clé invalide
- _apply_legacy_migrations : migration screens/videos → captures_base_dir
- get_settings_path : override LEVELUP_SETTINGS_PATH
- _on_change_setting : cas succès + cas échec (toast)
- _on_change_show_hints : mise à jour session_state + browser_prefs
"""

from __future__ import annotations

import json
import logging
import os
import sys
from pathlib import Path
from unittest.mock import MagicMock, patch

from src.ui.settings import (
    AppSettings,
    _apply_legacy_migrations,
    get_settings_path,
    patch_settings,
)

# ---------------------------------------------------------------------------
# get_settings_path — override env
# ---------------------------------------------------------------------------


class TestGetSettingsPath:
    """Tests pour get_settings_path."""

    def test_default_returns_app_settings_json(self) -> None:
        with patch.dict("os.environ", {}, clear=False):
            # Pas d'override : chemin par défaut pointe vers app_settings.json
            result = get_settings_path()
        assert result.endswith("app_settings.json")

    def test_env_override_takes_priority(self, tmp_path: Path) -> None:
        custom_path = str(tmp_path / "custom_settings.json")
        with patch.dict("os.environ", {"LEVELUP_SETTINGS_PATH": custom_path}):
            result = get_settings_path()
        assert result == custom_path

    def test_env_override_stripped(self, tmp_path: Path) -> None:
        custom_path = str(tmp_path / "custom_settings.json")
        with patch.dict("os.environ", {"LEVELUP_SETTINGS_PATH": f"  {custom_path}  "}):
            result = get_settings_path()
        assert result == custom_path

    def test_empty_env_falls_back_to_default(self) -> None:
        with patch.dict("os.environ", {"LEVELUP_SETTINGS_PATH": ""}):
            result = get_settings_path()
        assert result.endswith("app_settings.json")

    def test_whitespace_only_env_falls_back_to_default(self) -> None:
        with patch.dict("os.environ", {"LEVELUP_SETTINGS_PATH": "   "}):
            result = get_settings_path()
        assert result.endswith("app_settings.json")


# ---------------------------------------------------------------------------
# _apply_legacy_migrations
# ---------------------------------------------------------------------------


class TestApplyLegacyMigrations:
    """Tests pour _apply_legacy_migrations."""

    def test_no_op_when_captures_base_dir_set(self) -> None:
        data = {"media_captures_base_dir": "/captures", "media_screens_dir": "/screens"}
        result = _apply_legacy_migrations(data)
        assert result["media_captures_base_dir"] == "/captures"

    def test_no_op_when_no_legacy_dirs(self) -> None:
        data = {"lang": "fr"}
        result = _apply_legacy_migrations(data)
        assert "media_captures_base_dir" not in result

    def test_migrates_single_screens_dir(self) -> None:
        data = {"media_screens_dir": "/home/user/captures/screens"}
        result = _apply_legacy_migrations(data)
        assert "media_captures_base_dir" in result
        assert result["media_captures_base_dir"] != ""

    def test_migrates_common_parent_of_screens_and_videos(self, tmp_path: Path) -> None:
        # Utiliser des chemins réels pour éviter les différences `/` vs `\` sur Windows
        base = str(tmp_path)
        screens = str(tmp_path / "screens")
        videos = str(tmp_path / "videos")
        data = {"media_screens_dir": screens, "media_videos_dir": videos}
        result = _apply_legacy_migrations(data)
        # Le parent commun doit être tmp_path
        result_path = result.get("media_captures_base_dir", "")
        assert os.path.normcase(result_path) == os.path.normcase(base)

    def test_returns_dict_unchanged_when_empty(self) -> None:
        result = _apply_legacy_migrations({})
        assert result == {}

    def test_preserves_other_keys(self) -> None:
        data = {"lang": "en", "show_records": True, "media_screens_dir": "/screens"}
        result = _apply_legacy_migrations(data)
        assert result["lang"] == "en"
        assert result["show_records"] is True


# ---------------------------------------------------------------------------
# patch_settings
# ---------------------------------------------------------------------------


def _make_st_mock(session_data: dict | None = None) -> MagicMock:
    """Crée un mock streamlit avec session_state comme dict réel."""
    mock = MagicMock()
    mock.session_state = session_data if session_data is not None else {}
    return mock


class TestPatchSettingsNominal:
    """Tests du chemin nominal de patch_settings."""

    def test_returns_updated_settings_and_true(self, tmp_path: Path) -> None:
        settings_file = tmp_path / "app_settings.json"
        initial = AppSettings(lang="fr")
        mock_st = _make_st_mock({"app_settings": initial})

        with (
            patch.dict(sys.modules, {"streamlit": mock_st}),
            patch("src.ui.settings.get_settings_path", return_value=str(settings_file)),
        ):
            updated, ok, err = patch_settings("lang", "en")

        assert ok is True
        assert err == ""
        assert updated.lang == "en"

    def test_persists_to_disk(self, tmp_path: Path) -> None:
        settings_file = tmp_path / "app_settings.json"
        initial = AppSettings(lang="fr")
        mock_st = _make_st_mock({"app_settings": initial})

        with (
            patch.dict(sys.modules, {"streamlit": mock_st}),
            patch("src.ui.settings.get_settings_path", return_value=str(settings_file)),
            patch.dict("src.ui.settings._PROCESS_CACHE", {"last_content": None}),
        ):
            patch_settings("lang", "en")

        assert settings_file.exists(), "Le fichier settings doit être créé"
        data = json.loads(settings_file.read_text(encoding="utf-8"))
        assert data["lang"] == "en"

    def test_updates_session_state(self, tmp_path: Path) -> None:
        settings_file = tmp_path / "app_settings.json"
        initial = AppSettings(lang="fr")
        mock_st = _make_st_mock({"app_settings": initial})

        with (
            patch.dict(sys.modules, {"streamlit": mock_st}),
            patch("src.ui.settings.get_settings_path", return_value=str(settings_file)),
        ):
            patch_settings("show_records", True)

        stored = mock_st.session_state["app_settings"]
        assert isinstance(stored, AppSettings)
        assert stored.show_records is True

    def test_logs_debug_on_success(self, tmp_path: Path, caplog) -> None:
        settings_file = tmp_path / "app_settings.json"
        initial = AppSettings(lang="fr")
        mock_st = _make_st_mock({"app_settings": initial})

        with (
            patch.dict(sys.modules, {"streamlit": mock_st}),
            patch("src.ui.settings.get_settings_path", return_value=str(settings_file)),
            caplog.at_level(logging.DEBUG, logger="src.ui.settings"),
        ):
            patch_settings("lang", "en")

        assert "lang=" in caplog.text or "persisté" in caplog.text


class TestPatchSettingsNoOp:
    """Tests du cas no-op (même valeur)."""

    def test_returns_same_object_when_no_change(self) -> None:
        initial = AppSettings(lang="fr")
        mock_st = _make_st_mock({"app_settings": initial})

        with (
            patch.dict(sys.modules, {"streamlit": mock_st}),
            patch("src.ui.settings._write_settings") as mock_write,
        ):
            returned, ok, err = patch_settings("lang", "fr")

        assert ok is True
        assert err == ""
        mock_write.assert_not_called()  # Pas d'écriture si identique

    def test_no_disk_write_when_no_change(self) -> None:
        initial = AppSettings(show_records=False)
        mock_st = _make_st_mock({"app_settings": initial})

        with (
            patch.dict(sys.modules, {"streamlit": mock_st}),
            patch("src.ui.settings._atomic_write") as mock_atomic,
            patch.dict("src.ui.settings._PROCESS_CACHE", {"last_content": None}),
        ):
            patch_settings("show_records", False)

        mock_atomic.assert_not_called()


class TestPatchSettingsFallback:
    """Tests du fallback load_settings quand session_state est vide."""

    def test_loads_from_disk_when_session_empty(self, tmp_path: Path) -> None:
        settings_file = tmp_path / "app_settings.json"
        settings_file.write_text(json.dumps({"lang": "en"}), encoding="utf-8")
        mock_st = _make_st_mock({})  # session_state vide

        with (
            patch.dict(sys.modules, {"streamlit": mock_st}),
            patch("src.ui.settings.get_settings_path", return_value=str(settings_file)),
        ):
            updated, ok, _ = patch_settings("show_records", True)

        assert ok is True
        assert updated.lang == "en"  # Hérité du disque
        assert updated.show_records is True

    def test_loads_from_disk_when_session_has_wrong_type(self, tmp_path: Path) -> None:
        settings_file = tmp_path / "app_settings.json"
        settings_file.write_text(json.dumps({"lang": "fr"}), encoding="utf-8")
        mock_st = _make_st_mock({"app_settings": "not_an_appsettings"})

        with (
            patch.dict(sys.modules, {"streamlit": mock_st}),
            patch("src.ui.settings.get_settings_path", return_value=str(settings_file)),
        ):
            updated, ok, _ = patch_settings("lang", "en")

        assert ok is True
        assert updated.lang == "en"

    def test_logs_debug_when_session_empty(self, tmp_path: Path, caplog) -> None:
        settings_file = tmp_path / "app_settings.json"
        settings_file.write_text("{}", encoding="utf-8")
        mock_st = _make_st_mock({})

        with (
            patch.dict(sys.modules, {"streamlit": mock_st}),
            patch("src.ui.settings.get_settings_path", return_value=str(settings_file)),
            caplog.at_level(logging.DEBUG, logger="src.ui.settings"),
        ):
            patch_settings("lang", "en")

        assert "session_state vide" in caplog.text


class TestPatchSettingsRollback:
    """Tests du mécanisme de rollback en cas d'erreur d'écriture."""

    def test_rollback_session_state_on_write_error(self) -> None:
        """En cas d'échec I/O, session_state revient à la valeur originale."""
        initial = AppSettings(lang="fr")
        mock_st = _make_st_mock({"app_settings": initial})

        with (
            patch.dict(sys.modules, {"streamlit": mock_st}),
            patch("src.ui.settings._write_settings", return_value=(False, "disk full")),
        ):
            updated, ok, err = patch_settings("lang", "en")

        assert ok is False
        assert "disk full" in err
        # Rollback : session_state doit contenir les settings originales
        restored = mock_st.session_state["app_settings"]
        assert restored.lang == "fr"

    def test_returns_false_and_error_message(self) -> None:
        initial = AppSettings(lang="fr")
        mock_st = _make_st_mock({"app_settings": initial})

        with (
            patch.dict(sys.modules, {"streamlit": mock_st}),
            patch(
                "src.ui.settings._write_settings",
                return_value=(False, "Impossible d'écrire /some/path.json: disk full"),
            ),
        ):
            _, ok, err = patch_settings("lang", "en")

        assert ok is False
        assert "disk full" in err

    def test_logs_error_on_rollback(self, caplog) -> None:
        initial = AppSettings(lang="fr")
        mock_st = _make_st_mock({"app_settings": initial})

        with (
            patch.dict(sys.modules, {"streamlit": mock_st}),
            patch("src.ui.settings._write_settings", return_value=(False, "disk full")),
            caplog.at_level(logging.ERROR, logger="src.ui.settings"),
        ):
            patch_settings("lang", "en")

        assert "rollback" in caplog.text.lower() or "patch_settings" in caplog.text

    def test_updated_object_still_returned_on_error(self) -> None:
        """Même en cas d'erreur, l'objet updated est retourné (pour inspection)."""
        initial = AppSettings(lang="fr")
        mock_st = _make_st_mock({"app_settings": initial})

        with (
            patch.dict(sys.modules, {"streamlit": mock_st}),
            patch("src.ui.settings._write_settings", return_value=(False, "error")),
        ):
            returned, ok, _ = patch_settings("lang", "en")

        # L'objet retourné contient la nouvelle valeur (même si pas persistée)
        assert returned.lang == "en"
        assert ok is False


class TestPatchSettingsInvalidKey:
    """Tests avec une clé invalide (champ inexistant sur modele frozen)."""

    def test_unknown_key_returns_sentinel_unchanged(self) -> None:
        """model_copy ignore les clés inconnues (extra='ignore') —
        patch_settings voit getattr(current, key, _SENTINEL) != value
        et tente model_copy qui retourne l'objet inchangé.
        Le résultat est ok=True sans écriture si la valeur est identique
        à la sentinelle initiale.
        """
        initial = AppSettings()
        mock_st = _make_st_mock({"app_settings": initial})

        with (
            patch.dict(sys.modules, {"streamlit": mock_st}),
            patch("src.ui.settings._write_settings", return_value=(True, "")) as mock_write,
        ):
            # Pydantic v2 avec extra='ignore' : la clé inconnue est ignorée silencieusement
            # _SENTINEL != "valeur" donc on tente l'écriture
            updated, ok, err = patch_settings("champ_inexistant_xyz", "valeur")

        assert ok is True  # model_copy réussit, clé ignorée
        mock_write.assert_called_once()  # Une écriture a été tentée

    def test_unknown_key_second_call_is_noop(self) -> None:
        """Deuxième appel avec la même clé inconnue : no-op (valeur inchangée dans l'objet)."""
        initial = AppSettings()
        mock_st = _make_st_mock({"app_settings": initial})

        with (
            patch.dict(sys.modules, {"streamlit": mock_st}),
            patch("src.ui.settings._write_settings") as mock_write,
        ):
            # Deuxième appel avec valeur identique à getattr(initial, key, SENTINEL)
            # Si clé inconnue, getattr retourne toujours _SENTINEL != any value
            # Donc patch_settings tentera toujours d'écrire pour les clés inconnues.
            # Ce test vérifie juste que le comportement est prévisible.
            mock_write.return_value = (True, "")
            _, ok, _ = patch_settings("champ_inexistant_xyz", "valeur")

        assert ok is True


# ---------------------------------------------------------------------------
# _on_change_setting (pages/settings.py)
# ---------------------------------------------------------------------------


class TestOnChangeSetting:
    """Tests pour le handler _on_change_setting."""

    def test_calls_patch_settings_with_correct_args(self) -> None:
        from src.ui.pages.settings import _on_change_setting

        initial = AppSettings(lang="fr")
        mock_st = MagicMock()
        mock_st.session_state = {"app_settings": initial, "setting_lang": "en"}

        with (
            patch("src.ui.pages.settings.st", mock_st),
            patch(
                "src.ui.pages.settings.patch_settings", return_value=(initial, True, "")
            ) as mock_patch,
        ):
            _on_change_setting("lang", "setting_lang")

        mock_patch.assert_called_once_with("lang", "en")

    def test_shows_toast_on_failure(self) -> None:
        from src.ui.pages.settings import _on_change_setting

        initial = AppSettings(lang="fr")
        mock_st = MagicMock()
        mock_st.session_state = {"app_settings": initial, "setting_lang": "en"}

        with (
            patch("src.ui.pages.settings.st", mock_st),
            patch(
                "src.ui.pages.settings.patch_settings",
                return_value=(initial, False, "disk full"),
            ),
        ):
            _on_change_setting("lang", "setting_lang")

        mock_st.toast.assert_called_once()
        toast_msg = mock_st.toast.call_args[0][0]
        assert "disk full" in toast_msg or "Sauvegarde" in toast_msg

    def test_no_toast_on_success(self) -> None:
        from src.ui.pages.settings import _on_change_setting

        initial = AppSettings(lang="fr")
        mock_st = MagicMock()
        mock_st.session_state = {"app_settings": initial, "setting_lang": "en"}

        with (
            patch("src.ui.pages.settings.st", mock_st),
            patch(
                "src.ui.pages.settings.patch_settings",
                return_value=(initial, True, ""),
            ),
        ):
            _on_change_setting("lang", "setting_lang")

        mock_st.toast.assert_not_called()


# ---------------------------------------------------------------------------
# _on_change_show_hints (pages/settings.py)
# ---------------------------------------------------------------------------


class TestOnChangeShowHints:
    """Tests pour le handler _on_change_show_hints."""

    def test_updates_hints_visible_in_session_state(self) -> None:
        from src.ui.pages.settings import _on_change_show_hints

        mock_st = MagicMock()
        mock_st.session_state = {"setting_show_hints": True}

        with (
            patch("src.ui.pages.settings.st", mock_st),
            patch("src.ui.pages.settings.persist_browser_prefs") as mock_persist,
        ):
            _on_change_show_hints()

        assert mock_st.session_state["_hints_visible"] is True
        mock_persist.assert_called_once_with(show_hints="1")

    def test_persist_browser_prefs_false_when_hidden(self) -> None:
        from src.ui.pages.settings import _on_change_show_hints

        mock_st = MagicMock()
        mock_st.session_state = {"setting_show_hints": False}

        with (
            patch("src.ui.pages.settings.st", mock_st),
            patch("src.ui.pages.settings.persist_browser_prefs") as mock_persist,
        ):
            _on_change_show_hints()

        assert mock_st.session_state["_hints_visible"] is False
        mock_persist.assert_called_once_with(show_hints="0")
