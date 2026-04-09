"""Tests de robustesse pour save_settings / load_settings / _atomic_write.

Couvre :
- _try_parse_settings : fichier valide, JSON corrompu, non-dict, introuvable
- _atomic_write : création atomique, pas de fichier .tmp résiduel, backup .bak créé
- save_settings : écriture atomique, .bak créé, retour (True, "")
- load_settings — cascade de récupération :
    ① Cas nominal (fichier valide)
    ② Première utilisation (fichier absent)
    ③ Principal corrompu + backup valide → récupération + restauration principal
    ④ Principal corrompu + backup absent  → defaults
    ⑤ Principal corrompu + backup corrompu → defaults
- career_top_matches_render : session_state prioritaire sur load_settings()
"""

from __future__ import annotations

import json
import logging
from pathlib import Path
from unittest.mock import MagicMock, patch

from src.ui.settings import (
    AppSettings,
    _atomic_write,
    _settings_bak_path,
    _try_parse_settings,
    load_settings,
    save_settings,
)

# ---------------------------------------------------------------------------
# _try_parse_settings
# ---------------------------------------------------------------------------


class TestTryParseSettings:
    """Tests unitaires pour _try_parse_settings."""

    def test_valid_file_returns_settings(self, tmp_path: Path) -> None:
        f = tmp_path / "s.json"
        f.write_text(json.dumps({"lang": "en", "show_records": True}), encoding="utf-8")
        result = _try_parse_settings(str(f), "test")
        assert isinstance(result, AppSettings)
        assert result.lang == "en"
        assert result.show_records is True

    def test_file_not_found_returns_none(self, tmp_path: Path) -> None:
        result = _try_parse_settings(str(tmp_path / "absent.json"), "test")
        assert result is None

    def test_corrupted_json_returns_none(self, tmp_path: Path, caplog) -> None:
        f = tmp_path / "bad.json"
        f.write_text("{not valid json!!!", encoding="utf-8")
        with caplog.at_level(logging.WARNING, logger="src.ui.settings"):
            result = _try_parse_settings(str(f), "principal")
        assert result is None
        assert "JSON invalide" in caplog.text

    def test_non_dict_json_returns_none(self, tmp_path: Path, caplog) -> None:
        f = tmp_path / "list.json"
        f.write_text("[1, 2, 3]", encoding="utf-8")
        with caplog.at_level(logging.WARNING, logger="src.ui.settings"):
            result = _try_parse_settings(str(f), "principal")
        assert result is None
        assert "contenu inattendu" in caplog.text

    def test_unknown_fields_ignored(self, tmp_path: Path) -> None:
        """Pydantic extra='ignore' : les champs inconnus ne lèvent pas d'erreur."""
        f = tmp_path / "extra.json"
        f.write_text(
            json.dumps({"lang": "fr", "_comment": "ignored", "unknown_key": 42}), encoding="utf-8"
        )
        result = _try_parse_settings(str(f), "test")
        assert result is not None
        assert result.lang == "fr"

    def test_empty_object_returns_defaults(self, tmp_path: Path) -> None:
        f = tmp_path / "empty.json"
        f.write_text("{}", encoding="utf-8")
        result = _try_parse_settings(str(f), "test")
        assert isinstance(result, AppSettings)
        assert result.lang == "fr"  # défaut


# ---------------------------------------------------------------------------
# _atomic_write
# ---------------------------------------------------------------------------


class TestAtomicWrite:
    """Tests pour l'écriture atomique."""

    def test_creates_file_with_correct_content(self, tmp_path: Path) -> None:
        target = tmp_path / "out.json"
        _atomic_write(str(target), '{"lang": "en"}')
        assert target.exists()
        assert json.loads(target.read_text(encoding="utf-8")) == {"lang": "en"}

    def test_no_tmp_file_left_on_success(self, tmp_path: Path) -> None:
        target = tmp_path / "out.json"
        _atomic_write(str(target), "{}")
        tmp_files = list(tmp_path.glob(".settings_*.tmp"))
        assert tmp_files == [], f"Fichier .tmp résiduel trouvé : {tmp_files}"

    def test_creates_bak_file(self, tmp_path: Path) -> None:
        target = tmp_path / "out.json"
        bak = Path(_settings_bak_path(str(target)))
        _atomic_write(str(target), '{"lang": "fr"}')
        assert bak.exists()
        assert json.loads(bak.read_text(encoding="utf-8")) == {"lang": "fr"}

    def test_bak_content_matches_main(self, tmp_path: Path) -> None:
        target = tmp_path / "out.json"
        content = json.dumps({"show_records": True, "lang": "en"})
        _atomic_write(str(target), content)
        bak = Path(_settings_bak_path(str(target)))
        assert target.read_text(encoding="utf-8") == bak.read_text(encoding="utf-8")

    def test_creates_parent_dirs(self, tmp_path: Path) -> None:
        target = tmp_path / "subdir" / "nested" / "out.json"
        _atomic_write(str(target), "{}")
        assert target.exists()

    def test_overwrites_existing_file_atomically(self, tmp_path: Path) -> None:
        target = tmp_path / "out.json"
        target.write_text('{"lang": "fr"}', encoding="utf-8")
        _atomic_write(str(target), '{"lang": "en"}')
        assert json.loads(target.read_text(encoding="utf-8"))["lang"] == "en"


# ---------------------------------------------------------------------------
# save_settings
# ---------------------------------------------------------------------------


class TestSaveSettings:
    """Tests pour save_settings."""

    def test_returns_true_on_success(self, tmp_path: Path) -> None:
        settings_file = tmp_path / "app_settings.json"
        with patch("src.ui.settings.get_settings_path", return_value=str(settings_file)):
            ok, err = save_settings(AppSettings())
        assert ok is True
        assert err == ""

    def test_file_is_valid_json(self, tmp_path: Path) -> None:
        settings_file = tmp_path / "app_settings.json"
        with patch("src.ui.settings.get_settings_path", return_value=str(settings_file)):
            save_settings(AppSettings(lang="en", show_records=True))
        data = json.loads(settings_file.read_text(encoding="utf-8"))
        assert data["lang"] == "en"
        assert data["show_records"] is True

    def test_bak_file_created(self, tmp_path: Path) -> None:
        settings_file = tmp_path / "app_settings.json"
        bak_file = Path(_settings_bak_path(str(settings_file)))
        with patch("src.ui.settings.get_settings_path", return_value=str(settings_file)):
            save_settings(AppSettings())
        assert bak_file.exists()

    def test_bak_matches_main(self, tmp_path: Path) -> None:
        settings_file = tmp_path / "app_settings.json"
        bak_file = Path(_settings_bak_path(str(settings_file)))
        with patch("src.ui.settings.get_settings_path", return_value=str(settings_file)):
            save_settings(AppSettings(lang="en"))
        assert settings_file.read_text(encoding="utf-8") == bak_file.read_text(encoding="utf-8")

    def test_no_tmp_residual(self, tmp_path: Path) -> None:
        settings_file = tmp_path / "app_settings.json"
        with patch("src.ui.settings.get_settings_path", return_value=str(settings_file)):
            save_settings(AppSettings())
        tmp_files = list(tmp_path.glob(".settings_*.tmp"))
        assert tmp_files == []

    def test_returns_false_on_write_error(self, tmp_path: Path) -> None:
        settings_file = tmp_path / "app_settings.json"
        with (
            patch("src.ui.settings.get_settings_path", return_value=str(settings_file)),
            patch("src.ui.settings._atomic_write", side_effect=OSError("disk full")),
        ):
            ok, err = save_settings(AppSettings())
        assert ok is False
        assert "disk full" in err

    def test_roundtrip_preserves_all_fields(self, tmp_path: Path) -> None:
        """Sauvegarde puis rechargement : toutes les valeurs non-défaut sont préservées."""
        settings_file = tmp_path / "app_settings.json"
        original = AppSettings(
            lang="en",
            show_records=True,
            career_top_exclude_btb=True,
            discord_notifications_enabled=True,
            spnkr_refresh_max_matches=500,
        )
        with patch("src.ui.settings.get_settings_path", return_value=str(settings_file)):
            save_settings(original)
            loaded = load_settings()

        assert loaded.lang == "en"
        assert loaded.show_records is True
        assert loaded.career_top_exclude_btb is True
        assert loaded.discord_notifications_enabled is True
        assert loaded.spnkr_refresh_max_matches == 500


# ---------------------------------------------------------------------------
# load_settings — cascade de récupération
# ---------------------------------------------------------------------------


class TestLoadSettingsCascade:
    """Tests de la cascade de récupération de load_settings."""

    def test_nominal_loads_from_principal(self, tmp_path: Path) -> None:
        """① Cas nominal : fichier valide → settings correctes."""
        settings_file = tmp_path / "app_settings.json"
        settings_file.write_text(json.dumps({"lang": "en"}), encoding="utf-8")
        with patch("src.ui.settings.get_settings_path", return_value=str(settings_file)):
            result = load_settings()
        assert result.lang == "en"

    def test_first_use_creates_empty_file(self, tmp_path: Path) -> None:
        """② Première utilisation : fichier absent → crée {} et retourne defaults."""
        settings_file = tmp_path / "app_settings.json"
        assert not settings_file.exists()
        with patch("src.ui.settings.get_settings_path", return_value=str(settings_file)):
            result = load_settings()
        assert isinstance(result, AppSettings)
        assert result.lang == "fr"  # défaut
        assert settings_file.exists()

    def test_corrupted_principal_recovers_from_backup(self, tmp_path: Path, caplog) -> None:
        """③ Principal corrompu + backup valide → récupère depuis backup."""
        settings_file = tmp_path / "app_settings.json"
        bak_file = Path(_settings_bak_path(str(settings_file)))
        settings_file.write_text("{CORRUPT!!!", encoding="utf-8")
        bak_file.write_text(json.dumps({"lang": "en", "show_records": True}), encoding="utf-8")

        with (
            patch("src.ui.settings.get_settings_path", return_value=str(settings_file)),
            caplog.at_level(logging.WARNING, logger="src.ui.settings"),
        ):
            result = load_settings()

        assert result.lang == "en"
        assert result.show_records is True
        assert "récupérées depuis le backup" in caplog.text

    def test_corrupted_principal_restores_from_backup(self, tmp_path: Path) -> None:
        """③ Après récupération depuis backup, le fichier principal est restauré."""
        settings_file = tmp_path / "app_settings.json"
        bak_file = Path(_settings_bak_path(str(settings_file)))
        settings_file.write_text("{CORRUPT!!!", encoding="utf-8")
        bak_file.write_text(json.dumps({"lang": "en"}), encoding="utf-8")

        with patch("src.ui.settings.get_settings_path", return_value=str(settings_file)):
            load_settings()

        # Le principal doit être lisible après restauration
        assert json.loads(settings_file.read_text(encoding="utf-8"))["lang"] == "en"

    def test_corrupted_principal_no_backup_returns_defaults(self, tmp_path: Path, caplog) -> None:
        """④ Principal corrompu + pas de backup → defaults + log ERROR."""
        settings_file = tmp_path / "app_settings.json"
        settings_file.write_text("{CORRUPT!!!", encoding="utf-8")
        # Pas de .bak

        with (
            patch("src.ui.settings.get_settings_path", return_value=str(settings_file)),
            caplog.at_level(logging.WARNING, logger="src.ui.settings"),
        ):
            result = load_settings()

        assert isinstance(result, AppSettings)
        assert result.lang == "fr"  # défaut
        assert "Pas de backup disponible" in caplog.text

    def test_both_corrupted_returns_defaults(self, tmp_path: Path, caplog) -> None:
        """⑤ Principal ET backup corrompus → defaults + logs ERROR."""
        settings_file = tmp_path / "app_settings.json"
        bak_file = Path(_settings_bak_path(str(settings_file)))
        settings_file.write_text("{CORRUPT!!!", encoding="utf-8")
        bak_file.write_text("{ALSO CORRUPT!!!", encoding="utf-8")

        with (
            patch("src.ui.settings.get_settings_path", return_value=str(settings_file)),
            caplog.at_level(logging.ERROR, logger="src.ui.settings"),
        ):
            result = load_settings()

        assert isinstance(result, AppSettings)
        assert result.lang == "fr"  # défaut
        assert "defaults appliqués" in caplog.text

    def test_corrupted_does_not_overwrite_recoverable_file(self, tmp_path: Path) -> None:
        """⑤ En cas de double échec, on n'écrit PAS de defaults sur le fichier corrompu."""
        settings_file = tmp_path / "app_settings.json"
        settings_file.write_text("{CORRUPT!!!", encoding="utf-8")

        with patch("src.ui.settings.get_settings_path", return_value=str(settings_file)):
            load_settings()

        # Le fichier corrompu doit être intact (récupérable par DBA)
        assert settings_file.read_text(encoding="utf-8") == "{CORRUPT!!!"

    def test_nominal_logs_info(self, tmp_path: Path, caplog) -> None:
        """Le chemin nominal log un INFO avec le fichier source."""
        settings_file = tmp_path / "app_settings.json"
        settings_file.write_text(json.dumps({"lang": "fr"}), encoding="utf-8")

        with (
            patch("src.ui.settings.get_settings_path", return_value=str(settings_file)),
            caplog.at_level(logging.INFO, logger="src.ui.settings"),
        ):
            load_settings()

        assert "Settings chargées depuis" in caplog.text


# ---------------------------------------------------------------------------
# career_top_matches_render — session_state prioritaire
# ---------------------------------------------------------------------------


class TestCareerTopMatchesSessionState:
    """Vérifie que render_top_matches_section lit session_state avant le disque."""

    def test_uses_session_state_when_available(self) -> None:
        """Si session_state['app_settings'] est un AppSettings valide, load_settings n'est pas appelé."""
        from src.ui.settings import AppSettings

        mock_settings = AppSettings(career_top_exclude_btb=True)
        mock_st = MagicMock()
        mock_st.session_state = {"app_settings": mock_settings}

        with (
            patch("src.ui.pages.career_top_matches_render.st", mock_st),
            patch("src.ui.pages.career_top_matches_render.load_settings") as mock_load,
            patch("src.ui.pages.career_top_matches_render.load_top_best_matches", return_value=[]),
            patch("src.ui.pages.career_top_matches_render.load_top_worst_matches", return_value=[]),
            patch("src.ui.pages.career_top_matches_render.t", return_value=""),
            patch("src.ui.pages.career_top_matches_render.logger"),
        ):
            mock_st.info = MagicMock()
            from src.ui.pages.career_top_matches_render import render_top_matches_section

            render_top_matches_section(db_path="/fake", xuid="xuid123")

        mock_load.assert_not_called()

    def test_falls_back_to_load_settings_when_session_empty(self) -> None:
        """Si session_state ne contient pas AppSettings, load_settings est appelé."""
        mock_st = MagicMock()
        mock_st.session_state = {}  # Pas d'app_settings

        fallback = AppSettings(career_top_exclude_btb=False)

        with (
            patch("src.ui.pages.career_top_matches_render.st", mock_st),
            patch(
                "src.ui.pages.career_top_matches_render.load_settings", return_value=fallback
            ) as mock_load,
            patch("src.ui.pages.career_top_matches_render.load_top_best_matches", return_value=[]),
            patch("src.ui.pages.career_top_matches_render.load_top_worst_matches", return_value=[]),
            patch("src.ui.pages.career_top_matches_render.t", return_value=""),
            patch("src.ui.pages.career_top_matches_render.logger"),
        ):
            mock_st.info = MagicMock()
            from src.ui.pages.career_top_matches_render import render_top_matches_section

            render_top_matches_section(db_path="/fake", xuid="xuid123")

        mock_load.assert_called_once()
