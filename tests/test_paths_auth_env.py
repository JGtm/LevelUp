"""Tests pour _resolve_data_root, _is_dev_mode, _env_local_path et load_dotenv_if_present."""

from __future__ import annotations

import os
from pathlib import Path
from unittest.mock import patch

import pytest

# ---------------------------------------------------------------------------
# _is_dev_mode
# ---------------------------------------------------------------------------


class TestIsDevMode:
    """Tests pour _is_dev_mode()."""

    def test_dev_mode_with_venv(self, tmp_path: Path) -> None:
        """Détecte le mode dev quand .venv existe."""
        (tmp_path / ".venv").mkdir()
        with patch("src.utils.paths.REPO_ROOT", tmp_path):
            from src.utils.paths import _is_dev_mode

            assert _is_dev_mode() is True

    def test_not_dev_mode_without_venv(self, tmp_path: Path) -> None:
        """Pas de mode dev sans .venv."""
        with patch("src.utils.paths.REPO_ROOT", tmp_path):
            from src.utils.paths import _is_dev_mode

            assert _is_dev_mode() is False


# ---------------------------------------------------------------------------
# _resolve_data_root
# ---------------------------------------------------------------------------


class TestResolveDataRoot:
    """Tests pour _resolve_data_root()."""

    def test_override_via_env_var(self, tmp_path: Path) -> None:
        """LEVELUP_DATA override tout."""
        custom = tmp_path / "custom_data"
        with patch.dict(os.environ, {"LEVELUP_DATA": str(custom)}):
            from src.utils.paths import _resolve_data_root

            result = _resolve_data_root()
            assert result == custom

    def test_dev_mode_uses_repo_data(self, tmp_path: Path) -> None:
        """En mode dev, utilise REPO_ROOT/data."""
        (tmp_path / ".venv").mkdir()
        with (
            patch.dict(os.environ, {}, clear=False),
            patch("src.utils.paths.REPO_ROOT", tmp_path),
        ):
            # Retirer LEVELUP_DATA si présent
            env = {k: v for k, v in os.environ.items() if k != "LEVELUP_DATA"}
            with patch.dict(os.environ, env, clear=True):
                from src.utils.paths import _resolve_data_root

                result = _resolve_data_root()
                assert result == tmp_path / "data"

    def test_portable_mode_windows(self, tmp_path: Path) -> None:
        """Mode portable utilise %APPDATA%/LevelUp."""
        with (
            patch.dict(
                os.environ,
                {"APPDATA": str(tmp_path / "appdata")},
                clear=False,
            ),
            patch.dict(os.environ, {}, clear=False),
            patch("src.utils.paths.REPO_ROOT", tmp_path),
            patch("src.utils.paths._is_dev_mode", return_value=False),
            patch("os.name", "nt"),
        ):
            env = {k: v for k, v in os.environ.items() if k != "LEVELUP_DATA"}
            env["APPDATA"] = str(tmp_path / "appdata")
            with patch.dict(os.environ, env, clear=True):
                from src.utils.paths import _resolve_data_root

                result = _resolve_data_root()
                assert result == tmp_path / "appdata" / "LevelUp"


# ---------------------------------------------------------------------------
# _env_local_path
# ---------------------------------------------------------------------------


class TestEnvLocalPath:
    """Tests pour _env_local_path()."""

    def test_returns_data_dir_when_exists(self, tmp_path: Path) -> None:
        """Retourne DATA_DIR/.env.local si le fichier existe."""
        env_file = tmp_path / "data" / ".env.local"
        env_file.parent.mkdir(parents=True)
        env_file.write_text("KEY=VALUE")
        with (
            patch("src.utils.auth.DATA_DIR", tmp_path / "data"),
            patch("src.utils.auth.REPO_ROOT", tmp_path),
        ):
            from src.utils.auth import _env_local_path

            result = _env_local_path()
            assert result == env_file

    def test_falls_back_to_repo_root(self, tmp_path: Path) -> None:
        """Retombe sur REPO_ROOT/.env.local si DATA_DIR/.env.local n'existe pas."""
        with (
            patch("src.utils.auth.DATA_DIR", tmp_path / "data"),
            patch("src.utils.auth.REPO_ROOT", tmp_path),
        ):
            from src.utils.auth import _env_local_path

            result = _env_local_path()
            assert result == tmp_path / ".env.local"


# ---------------------------------------------------------------------------
# load_dotenv_if_present
# ---------------------------------------------------------------------------


class TestLoadDotenvIfPresent:
    """Tests pour load_dotenv_if_present()."""

    def test_loads_env_file(self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
        """Charge les variables depuis .env.local."""
        env_file = tmp_path / ".env.local"
        env_file.write_text("TEST_DOTENV_KEY=hello_world\n")

        # Nettoyer la variable si elle existe
        monkeypatch.delenv("TEST_DOTENV_KEY", raising=False)

        with (
            patch("src.utils.env.DATA_DIR", tmp_path / "data"),
            patch("src.utils.env.REPO_ROOT", tmp_path),
        ):
            from src.utils.env import load_dotenv_if_present

            load_dotenv_if_present()
            assert os.environ.get("TEST_DOTENV_KEY") == "hello_world"

        # Nettoyage
        monkeypatch.delenv("TEST_DOTENV_KEY", raising=False)

    def test_does_not_overwrite_existing(
        self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Ne remplace pas une variable déjà définie."""
        env_file = tmp_path / ".env.local"
        env_file.write_text("TEST_EXISTING_KEY=new_value\n")

        monkeypatch.setenv("TEST_EXISTING_KEY", "original")

        with (
            patch("src.utils.env.DATA_DIR", tmp_path / "data"),
            patch("src.utils.env.REPO_ROOT", tmp_path),
        ):
            from src.utils.env import load_dotenv_if_present

            load_dotenv_if_present()
            assert os.environ["TEST_EXISTING_KEY"] == "original"

    def test_skips_comments_and_empty(
        self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Ignore les commentaires et lignes vides."""
        env_file = tmp_path / ".env.local"
        env_file.write_text("# comment\n\nVALID_KEY=value\n")

        monkeypatch.delenv("VALID_KEY", raising=False)

        with (
            patch("src.utils.env.DATA_DIR", tmp_path / "data"),
            patch("src.utils.env.REPO_ROOT", tmp_path),
        ):
            from src.utils.env import load_dotenv_if_present

            load_dotenv_if_present()
            assert os.environ.get("VALID_KEY") == "value"

        monkeypatch.delenv("VALID_KEY", raising=False)

    def test_strips_quotes(self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
        """Strip les guillemets autour des valeurs."""
        env_file = tmp_path / ".env.local"
        env_file.write_text('QUOTED_KEY="my value"\n')

        monkeypatch.delenv("QUOTED_KEY", raising=False)

        with (
            patch("src.utils.env.DATA_DIR", tmp_path / "data"),
            patch("src.utils.env.REPO_ROOT", tmp_path),
        ):
            from src.utils.env import load_dotenv_if_present

            load_dotenv_if_present()
            assert os.environ.get("QUOTED_KEY") == "my value"

        monkeypatch.delenv("QUOTED_KEY", raising=False)

    def test_handles_missing_file(self, tmp_path: Path) -> None:
        """Ne crashe pas si le fichier n'existe pas."""
        with (
            patch("src.utils.env.DATA_DIR", tmp_path / "data"),
            patch("src.utils.env.REPO_ROOT", tmp_path),
        ):
            from src.utils.env import load_dotenv_if_present

            load_dotenv_if_present()  # Ne doit pas crasher


# ---------------------------------------------------------------------------
# build_release.py — fonctions utilitaires
# ---------------------------------------------------------------------------

# Le dossier projet "packaging/" entre en conflit avec le paquet stdlib
# "packaging". On importe donc via importlib avec un chemin absolu.

import importlib.util

_BUILD_RELEASE_PATH = Path(__file__).resolve().parent.parent / "packaging" / "build_release.py"


def _import_build_release():
    """Importe packaging/build_release.py sans conflit avec le paquet stdlib."""
    spec = importlib.util.spec_from_file_location("build_release", _BUILD_RELEASE_PATH)
    mod = importlib.util.module_from_spec(spec)  # type: ignore[arg-type]
    spec.loader.exec_module(mod)  # type: ignore[union-attr]
    return mod


@pytest.mark.skipif(not _BUILD_RELEASE_PATH.exists(), reason="build_release.py absent")
class TestBuildRelease:
    """Tests pour packaging/build_release.py."""

    def test_read_version(self) -> None:
        """Lit correctement la version depuis pyproject.toml."""
        mod = _import_build_release()
        version = mod._read_version()
        parts = version.split(".")
        assert len(parts) >= 2
        assert all(p.isdigit() for p in parts)

    def test_copy_project(self, tmp_path: Path) -> None:
        """Copie les fichiers du projet dans le dossier cible."""
        mod = _import_build_release()
        target = tmp_path / "build"
        target.mkdir()

        mod._copy_project(target)

        assert (target / "launcher.py").exists()
        assert (target / "streamlit_app.py").exists()
        assert (target / "src").is_dir()
        assert (target / "data" / "players").is_dir()
        assert (target / "data" / "warehouse").is_dir()

    def test_create_portable_bat(self, tmp_path: Path) -> None:
        """Crée un LevelUp.bat adapté au mode portable."""
        mod = _import_build_release()
        mod._create_portable_bat(tmp_path)

        bat_file = tmp_path / "LevelUp.bat"
        assert bat_file.exists()
        content = bat_file.read_text(encoding="utf-8")
        assert "python\\python.exe" in content
        assert ".deps_installed" in content
        assert "launcher.py run" in content


# ---------------------------------------------------------------------------
# get_shared_matches_path_from_player — fallback v2 / v1
# ---------------------------------------------------------------------------


class TestGetSharedMatchesPathFromPlayer:
    """Tests pour la résolution du chemin shared_matches depuis un path joueur."""

    def test_returns_v2_when_both_exist(self, tmp_path: Path) -> None:
        """shared_matches_v2.duckdb est retourné en priorité si les deux existent."""
        from src.utils.paths import get_shared_matches_path_from_player

        warehouse = tmp_path / "warehouse"
        warehouse.mkdir()
        (warehouse / "shared_matches_v2.duckdb").touch()
        (warehouse / "shared_matches.duckdb").touch()
        player_db = tmp_path / "players" / "GT" / "stats.duckdb"
        player_db.parent.mkdir(parents=True)
        player_db.touch()

        result = get_shared_matches_path_from_player(player_db)
        assert result is not None
        assert result.name == "shared_matches_v2.duckdb"

    def test_falls_back_to_v1_when_v2_absent(self, tmp_path: Path) -> None:
        """Retourne shared_matches.duckdb si v2 est absent."""
        from src.utils.paths import get_shared_matches_path_from_player

        warehouse = tmp_path / "warehouse"
        warehouse.mkdir()
        (warehouse / "shared_matches.duckdb").touch()
        player_db = tmp_path / "players" / "GT" / "stats.duckdb"
        player_db.parent.mkdir(parents=True)
        player_db.touch()

        result = get_shared_matches_path_from_player(player_db)
        assert result is not None
        assert result.name == "shared_matches.duckdb"

    def test_returns_none_when_neither_exists(self, tmp_path: Path) -> None:
        """Retourne None si aucune des deux DBs n'existe."""
        from src.utils.paths import get_shared_matches_path_from_player

        player_db = tmp_path / "players" / "GT" / "stats.duckdb"
        player_db.parent.mkdir(parents=True)
        player_db.touch()

        result = get_shared_matches_path_from_player(player_db)
        assert result is None

    def test_returns_none_when_no_players_in_path(self, tmp_path: Path) -> None:
        """Retourne None si le chemin ne contient pas 'players'."""
        from src.utils.paths import get_shared_matches_path_from_player

        arbitrary_path = tmp_path / "stats.duckdb"
        arbitrary_path.touch()

        result = get_shared_matches_path_from_player(arbitrary_path)
        assert result is None

    def test_accepts_string_path(self, tmp_path: Path) -> None:
        """Accepte un chemin en string (pas seulement Path)."""
        from src.utils.paths import get_shared_matches_path_from_player

        warehouse = tmp_path / "warehouse"
        warehouse.mkdir()
        (warehouse / "shared_matches_v2.duckdb").touch()
        player_db = tmp_path / "players" / "GT" / "stats.duckdb"
        player_db.parent.mkdir(parents=True)
        player_db.touch()

        result = get_shared_matches_path_from_player(str(player_db))
        assert result is not None
        assert result.name == "shared_matches_v2.duckdb"
