"""Tests pour les commandes launcher.py setup/doctor et utilitaires.

launcher.py remplace sys.stdout/stderr à l'import (UTF-8 Windows),
ce qui casse pytest capture. On l'importe via importlib avec
stdout/stderr sauvegardés, et on patche les fonctions sur le module.
"""

from __future__ import annotations

import argparse
import importlib
import importlib.util
import io
import subprocess
import sys
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

# ---------------------------------------------------------------------------
# Import sécurisé de launcher.py (sans casser pytest capture)
# ---------------------------------------------------------------------------

_LAUNCHER_PATH = Path(__file__).resolve().parent.parent / "launcher.py"


@pytest.fixture(scope="session")
def launcher_mod():
    """Importe launcher.py en protégeant stdout/stderr de pytest."""
    orig_stdout = sys.stdout
    orig_stderr = sys.stderr
    # Fournir de faux streams pour que le code d'init de launcher
    # ne détruise pas les vrais pipes pytest
    sys.stdout = io.TextIOWrapper(io.BytesIO(), encoding="utf-8")
    sys.stderr = io.TextIOWrapper(io.BytesIO(), encoding="utf-8")
    try:
        spec = importlib.util.spec_from_file_location("_test_launcher", _LAUNCHER_PATH)
        mod = importlib.util.module_from_spec(spec)  # type: ignore[arg-type]
        sys.modules["_test_launcher"] = mod
        spec.loader.exec_module(mod)  # type: ignore[union-attr]
    finally:
        sys.stdout = orig_stdout
        sys.stderr = orig_stderr
    return mod


class TestFindSystemPython:
    """Tests pour _find_system_python()."""

    def test_finds_python_via_py_launcher(self, launcher_mod) -> None:  # type: ignore[no-untyped-def]
        """Trouve Python via le py launcher Windows."""
        fake_result = MagicMock(returncode=0, stdout="C:\\Python312\\python.exe\n")
        with (
            patch.object(launcher_mod.sys, "platform", "win32"),
            patch("shutil.which", lambda name: "py" if name == "py" else None),
            patch.object(launcher_mod.subprocess, "run", return_value=fake_result),
        ):
            assert launcher_mod._find_system_python() == "C:\\Python312\\python.exe"

    def test_finds_python_in_path(self, launcher_mod) -> None:  # type: ignore[no-untyped-def]
        """Trouve python3 dans le PATH."""
        fake_version = MagicMock(returncode=0, stdout="12\n")
        with (
            patch.object(launcher_mod.sys, "platform", "linux"),
            patch("shutil.which", lambda name: "/usr/bin/python3" if name == "python3" else None),
            patch.object(launcher_mod.subprocess, "run", return_value=fake_version),
        ):
            assert launcher_mod._find_system_python() == "/usr/bin/python3"

    def test_returns_none_when_nothing_found(self, launcher_mod) -> None:  # type: ignore[no-untyped-def]
        """Retourne None si aucun Python trouvé."""
        with (
            patch.object(launcher_mod.sys, "platform", "linux"),
            patch("shutil.which", return_value=None),
        ):
            assert launcher_mod._find_system_python() is None

    def test_rejects_python_too_old(self, launcher_mod) -> None:  # type: ignore[no-untyped-def]
        """Rejette Python < 3.10."""
        with (
            patch.object(launcher_mod.sys, "platform", "linux"),
            patch("shutil.which", lambda name: "/usr/bin/python3" if name == "python3" else None),
            patch.object(
                launcher_mod.subprocess,
                "run",
                return_value=MagicMock(returncode=0, stdout="9\n"),
            ),
        ):
            assert launcher_mod._find_system_python() is None

    def test_handles_subprocess_timeout(self, launcher_mod) -> None:  # type: ignore[no-untyped-def]
        """Gère un timeout du subprocess."""

        def _timeout(*a: object, **kw: object) -> None:
            raise subprocess.TimeoutExpired("py", 10)

        with (
            patch.object(launcher_mod.sys, "platform", "linux"),
            patch("shutil.which", lambda name: "/usr/bin/python3" if name == "python3" else None),
            patch.object(launcher_mod.subprocess, "run", side_effect=_timeout),
        ):
            assert launcher_mod._find_system_python() is None


# ---------------------------------------------------------------------------
# _install_python_via_winget
# ---------------------------------------------------------------------------


class TestInstallPythonViaWinget:
    """Tests pour _install_python_via_winget()."""

    def test_fails_on_linux(self, launcher_mod) -> None:  # type: ignore[no-untyped-def]
        """Retourne False sur Linux."""
        with patch.object(launcher_mod.sys, "platform", "linux"):
            assert launcher_mod._install_python_via_winget() is False

    def test_fails_without_winget(self, launcher_mod) -> None:  # type: ignore[no-untyped-def]
        """Retourne False si winget n'est pas disponible."""
        with (
            patch.object(launcher_mod.sys, "platform", "win32"),
            patch("shutil.which", return_value=None),
        ):
            assert launcher_mod._install_python_via_winget() is False

    def test_success_with_winget(self, launcher_mod) -> None:  # type: ignore[no-untyped-def]
        """Retourne True si winget installe avec succès."""
        with (
            patch.object(launcher_mod.sys, "platform", "win32"),
            patch("shutil.which", return_value="winget"),
            patch.object(
                launcher_mod.subprocess,
                "run",
                return_value=MagicMock(returncode=0),
            ),
        ):
            assert launcher_mod._install_python_via_winget() is True


# ---------------------------------------------------------------------------
# _cmd_doctor
# ---------------------------------------------------------------------------


class TestCmdDoctor:
    """Tests pour _cmd_doctor()."""

    def test_doctor_runs_without_crash(self, launcher_mod) -> None:  # type: ignore[no-untyped-def]
        """Doctor s'exécute sans crasher."""
        args = argparse.Namespace()
        result = launcher_mod._cmd_doctor(args)
        assert isinstance(result, int)


# ---------------------------------------------------------------------------
# _cmd_setup
# ---------------------------------------------------------------------------


class TestCmdSetup:
    """Tests pour _cmd_setup()."""

    def test_setup_skips_when_venv_exists(
        self,
        launcher_mod,
        tmp_path: Path,  # type: ignore[no-untyped-def]
    ) -> None:
        """En mode non-update, détecte le venv existant."""
        venv_dir = tmp_path / ".venv" / "Scripts"
        venv_dir.mkdir(parents=True)
        fake_python = venv_dir / "python.exe"
        fake_python.write_text("fake")

        with (
            patch.object(launcher_mod, "REPO_ROOT", tmp_path),
            patch.object(launcher_mod.sys, "platform", "win32"),
            patch.object(
                launcher_mod.subprocess,
                "run",
                return_value=MagicMock(returncode=0, stdout="OK\n"),
            ),
        ):
            args = argparse.Namespace(update=False)
            assert launcher_mod._cmd_setup(args) == 0


# ---------------------------------------------------------------------------
# _transfer_msal_cache
# ---------------------------------------------------------------------------


class TestTransferMsalCache:
    """Tests pour _transfer_msal_cache()."""

    def test_copie_le_cache_dans_la_target(self, launcher_mod, tmp_path: Path) -> None:  # type: ignore[no-untyped-def]
        """Le cache MSAL est copié depuis source vers target."""
        import duckdb

        source = tmp_path / "bootstrap_auth.duckdb"
        target = tmp_path / "stats.duckdb"

        # Préparer source avec un cache factice
        with duckdb.connect(str(source)) as conn:
            conn.execute(
                "CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR,"
                " updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)"
            )
            conn.execute(
                "INSERT INTO sync_meta VALUES ('msal_token_cache', '{\"test\": 1}', CURRENT_TIMESTAMP)"
            )

        launcher_mod._transfer_msal_cache(source, target)

        # Vérifier que target contient le cache
        with duckdb.connect(str(target)) as conn:
            row = conn.execute(
                "SELECT value FROM sync_meta WHERE key = 'msal_token_cache'"
            ).fetchone()
        assert row is not None
        assert row[0] == '{"test": 1}'

    def test_ne_plante_pas_si_source_sans_cache(self, launcher_mod, tmp_path: Path) -> None:  # type: ignore[no-untyped-def]
        """Ne lève pas d'exception si la clé msal_token_cache est absente dans source."""
        import duckdb

        source = tmp_path / "bootstrap_empty.duckdb"
        target = tmp_path / "stats.duckdb"

        # Source avec sync_meta vide
        with duckdb.connect(str(source)) as conn:
            conn.execute(
                "CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR,"
                " updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)"
            )

        # Ne doit pas lever d'exception
        launcher_mod._transfer_msal_cache(source, target)

        # Target ne doit pas avoir été créée (rien à transférer)
        assert not target.exists()

    def test_ecrase_cache_existant_dans_target(self, launcher_mod, tmp_path: Path) -> None:  # type: ignore[no-untyped-def]
        """Remplace le cache existant dans target par celui de source."""
        import duckdb

        source = tmp_path / "bootstrap.duckdb"
        target = tmp_path / "stats_existing.duckdb"

        with duckdb.connect(str(source)) as conn:
            conn.execute(
                "CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR,"
                " updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)"
            )
            conn.execute(
                "INSERT INTO sync_meta VALUES ('msal_token_cache', 'NEW_CACHE', CURRENT_TIMESTAMP)"
            )

        # Target avec un ancien cache
        with duckdb.connect(str(target)) as conn:
            conn.execute(
                "CREATE TABLE sync_meta (key VARCHAR PRIMARY KEY, value VARCHAR,"
                " updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)"
            )
            conn.execute(
                "INSERT INTO sync_meta VALUES ('msal_token_cache', 'OLD_CACHE', CURRENT_TIMESTAMP)"
            )

        launcher_mod._transfer_msal_cache(source, target)

        with duckdb.connect(str(target)) as conn:
            row = conn.execute(
                "SELECT value FROM sync_meta WHERE key = 'msal_token_cache'"
            ).fetchone()
        assert row is not None
        assert row[0] == "NEW_CACHE"


# ---------------------------------------------------------------------------
# _onboard_first_player — chemin non-interactif
# ---------------------------------------------------------------------------


class TestOnboardFirstPlayerNonInteractive:
    """Tests pour _onboard_first_player() en mode non-interactif."""

    def test_retourne_2_en_non_interactif(self, launcher_mod) -> None:  # type: ignore[no-untyped-def]
        """Retourne 2 si le terminal n'est pas interactif (isatty=False)."""
        with patch.object(launcher_mod.sys.stdin, "isatty", return_value=False):
            result = launcher_mod._onboard_first_player()
        assert result == 2

    def test_retourne_2_si_start_device_flow_echoue(self, launcher_mod, tmp_path: Path) -> None:  # type: ignore[no-untyped-def]
        """Retourne 2 si start_device_flow lève une exception."""
        fake_bootstrap = tmp_path / "bootstrap_auth.duckdb"
        with (
            patch.object(launcher_mod.sys.stdin, "isatty", return_value=True),
            # _load_dotenv_for_launcher et _BOOTSTRAP_AUTH_DB sont utilisés dans
            # launcher_onboarding._onboard_first_player — patcher à la source du module
            patch("src.utils.launcher_onboarding._load_dotenv_for_launcher"),
            patch("src.utils.launcher_onboarding._BOOTSTRAP_AUTH_DB", fake_bootstrap),
            # start_device_flow est importé localement — patcher à la source
            patch("src.auth.provider.start_device_flow", side_effect=Exception("DeviceFlowError")),
        ):
            result = launcher_mod._onboard_first_player()
        assert result == 2


# ---------------------------------------------------------------------------
# _cmd_add_player — routing gamertag présent / absent
# ---------------------------------------------------------------------------


class TestCmdAddPlayerRouting:
    """Tests de routage _cmd_add_player."""

    def test_sans_gamertag_appelle_onboard(self, launcher_mod) -> None:  # type: ignore[no-untyped-def]
        """Sans --gamertag, délègue à _onboard_first_player."""
        args = argparse.Namespace(gamertag=None)
        # _cmd_add_player est dans launcher_onboarding — patcher _onboard_first_player à la source
        with patch(
            "src.utils.launcher_onboarding._onboard_first_player", return_value=0
        ) as mock_onboard:
            result = launcher_mod._cmd_add_player(args)
        assert result == 0
        mock_onboard.assert_called_once()

    def test_avec_gamertag_et_token_valide_sync_directe(self, launcher_mod) -> None:  # type: ignore[no-untyped-def]
        """Avec --gamertag et token valide, lance directement la sync."""
        args = argparse.Namespace(gamertag="SpartanTest", full=False, max_matches=100)
        env_info = {"player_token": True, "client_id": True, "player_token_key": "KEY"}
        # _cmd_add_player est dans launcher_onboarding — patcher à la source du module appelant
        with (
            patch("src.utils.launcher_onboarding._load_dotenv_for_launcher"),
            patch("src.utils.launcher_players._env_check_for_player", return_value=env_info),
            patch("src.utils.launcher_onboarding._sync_player_duckdb", return_value=(0, 10)),
            patch("src.utils.launcher_onboarding._ensure_warehouse_dbs"),
        ):
            result = launcher_mod._cmd_add_player(args)
        assert result == 0
