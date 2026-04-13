"""Utilitaires d'environnement pour le lanceur LevelUp.

Contient : détection de langue, gestion venv/Python, installation Python,
et la commande ``setup``.
Extrait de launcher.py (F8 post-v7 housekeeping).
"""

from __future__ import annotations

import os
import socket
import subprocess
import sys
from pathlib import Path

from src.utils.launcher_i18n import t as _t
from src.utils.paths import REPO_ROOT

# =============================================================================
# i18n — Détection de la langue système
# =============================================================================


def _detect_lang() -> str:
    """Détecte la langue système → 'fr' ou 'en' (défaut 'en').

    Ordre de priorité :
    1. Variable d'env LEVELUP_LANG (override explicite)
    2. locale.getlocale() / getdefaultlocale()
    3. Variables d'env LANG / LC_ALL / LC_MESSAGES / LANGUAGE
    4. Registre Windows (Control Panel\\International\\LocaleName)
    """
    forced = os.environ.get("LEVELUP_LANG", "").strip().lower()
    if forced in ("fr", "en"):
        return forced

    import locale as _locale  # noqa: PLC0415

    candidates: list[str] = []
    try:
        lc = _locale.getlocale()[0]
        if lc:
            candidates.append(lc.lower())
    except Exception:
        pass
    for var in ("LANG", "LC_ALL", "LC_MESSAGES", "LANGUAGE"):
        val = os.environ.get(var, "")
        if val and not val.upper().startswith("C"):
            candidates.append(val.lower().split(".")[0].split("@")[0])
    if sys.platform == "win32":
        try:
            import winreg  # noqa: PLC0415

            with winreg.OpenKey(winreg.HKEY_CURRENT_USER, r"Control Panel\International") as _k:
                _locale_name = winreg.QueryValueEx(_k, "LocaleName")[0]
                if _locale_name:
                    candidates.append(_locale_name.lower())
        except Exception:
            pass
    for _c in candidates:
        if _c.startswith("fr"):
            return "fr"
    return "en"


LANG: str = _detect_lang()

# =============================================================================
# Helpers Python / venv
# =============================================================================


def _preferred_python_executable() -> Path | None:
    """Trouve le python du venv local."""
    candidates = [
        REPO_ROOT / ".venv_windows" / "Scripts" / "python.exe",
        REPO_ROOT / ".venv" / "Scripts" / "python.exe",
        REPO_ROOT / ".venv_windows" / "bin" / "python",
        REPO_ROOT / ".venv" / "bin" / "python",
    ]
    for p in candidates:
        if p.exists():
            return p
    return None


def _maybe_reexec_into_venv(argv: list[str]) -> None:
    """Re-exécute dans le venv si nécessaire."""
    if os.environ.get("LEVELUP_LAUNCHER_NO_REEXEC"):
        return

    preferred = _preferred_python_executable()
    if preferred is None:
        return

    try:
        current = Path(sys.executable).resolve()
        preferred_r = preferred.resolve()
    except Exception:
        return

    if current == preferred_r:
        return

    os.environ["LEVELUP_LAUNCHER_NO_REEXEC"] = "1"
    # Utiliser sys.argv[0] (le script lanceur) plutôt que __file__ (ce module utilitaire)
    os.execv(str(preferred_r), [str(preferred_r), str(Path(sys.argv[0]).resolve()), *argv])


def _require_module(name: str, *, install_hint: str) -> None:
    """Vérifie qu'un module est disponible."""
    try:
        __import__(name)
    except Exception as e:
        print(_t("dep_missing", LANG, name=name))
        print(_t("dep_detail", LANG), e)
        print(_t("dep_install_hint", LANG))
        print(f"  {install_hint}")
        raise SystemExit(2) from e


def _import_duckdb():  # noqa: ANN201
    """Importe duckdb de manière lazy."""
    try:
        import duckdb  # noqa: PLC0415

        return duckdb
    except ImportError as err:
        print(_t("duckdb_not_installed", LANG))
        print("   pip install duckdb")
        raise SystemExit(2) from err


def _pick_free_port() -> int:
    """Trouve un port libre."""
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(("127.0.0.1", 0))
        return int(s.getsockname()[1])


# =============================================================================
# Helpers Python — installation système
# =============================================================================


def _find_system_python() -> str | None:  # noqa: C901, PLR0912
    """Trouve un Python 3.10-3.13 sur le système."""
    import shutil  # noqa: PLC0415

    # 1. Python Launcher (py) — Windows uniquement
    if sys.platform == "win32" and shutil.which("py"):
        for minor in (12, 13, 11, 10):
            try:
                result = subprocess.run(
                    ["py", f"-3.{minor}", "-c", "import sys; print(sys.executable)"],
                    capture_output=True,
                    text=True,
                    timeout=10,
                )
                if result.returncode == 0 and result.stdout.strip():
                    return result.stdout.strip()
            except Exception:
                continue

    # 2. Binaires versionnés (Homebrew macOS, apt Linux)
    for minor in (12, 13, 11, 10):
        exe = shutil.which(f"python3.{minor}")
        if exe:
            return exe

    # 3. python3 / python générique dans PATH
    for name in ("python3", "python"):
        exe = shutil.which(name)
        if not exe:
            continue
        try:
            result = subprocess.run(
                [exe, "-c", "import sys; print(sys.version_info.minor)"],
                capture_output=True,
                text=True,
                timeout=10,
            )
            if result.returncode == 0:
                minor = int(result.stdout.strip())
                if 10 <= minor <= 13:
                    return exe
        except Exception:
            continue

    # 4. Chemins standards Windows
    if sys.platform == "win32":
        appdata = os.environ.get("LOCALAPPDATA", "")
        for minor in (12, 13, 11, 10):
            candidate = Path(appdata) / "Programs" / "Python" / f"Python3{minor}" / "python.exe"
            if candidate.exists():
                return str(candidate)

    # 5. Chemins standards macOS (Homebrew Intel + Apple Silicon)
    if sys.platform == "darwin":
        for prefix in ("/opt/homebrew", "/usr/local"):
            for minor in (12, 13, 11, 10):
                candidate = Path(prefix) / "bin" / f"python3.{minor}"
                if candidate.exists():
                    return str(candidate)

    return None


def _install_python_via_winget() -> bool:
    """Installe Python 3.12 via winget (Windows uniquement)."""
    if sys.platform != "win32":
        print(_t("winget_windows_only", LANG))
        print(_t("python_manual_install", LANG))
        return False

    import shutil  # noqa: PLC0415

    if not shutil.which("winget"):
        print(_t("winget_unavailable", LANG))
        print(_t("python_manual_install", LANG))
        return False

    print(_t("winget_installing", LANG))
    try:
        result = subprocess.run(
            [
                "winget",
                "install",
                "--id",
                "Python.Python.3.12",
                "--scope",
                "user",
                "--accept-source-agreements",
                "--accept-package-agreements",
            ],
            timeout=300,
        )
        return result.returncode == 0
    except Exception as e:
        print(_t("winget_error", LANG, err=e))
        return False


# =============================================================================
# Commande: setup (configure venv + dépendances)
# =============================================================================


def _cmd_setup(args) -> int:  # noqa: ANN001
    """Commande: configure l'environnement (venv + dépendances)."""
    update_mode = getattr(args, "update", False)

    print("=" * 60)
    print(_t("setup_title_update" if update_mode else "setup_title", LANG))
    print("=" * 60)

    venv_python = REPO_ROOT / ".venv" / "Scripts" / "python.exe"
    if sys.platform != "win32":
        venv_python = REPO_ROOT / ".venv" / "bin" / "python"

    created_venv = False

    # ── 1. Vérifier/créer le venv ──
    if venv_python.exists() and not update_mode:
        print(_t("setup_venv_exists", LANG))
        py = str(venv_python)
    else:
        print(_t("setup_step1_searching", LANG))
        py = _find_system_python()
        if not py:
            print(_t("setup_python_not_found", LANG))
            if _install_python_via_winget():
                py = _find_system_python()
        if not py:
            print(_t("setup_python_impossible", LANG))
            print(_t("setup_python_install_url", LANG))
            return 1
        print(_t("setup_python_found", LANG, py=py))
        if not venv_python.exists():
            print(_t("setup_step2_creating_venv", LANG))
            result = subprocess.run([py, "-m", "venv", str(REPO_ROOT / ".venv")])
            if result.returncode != 0:
                print(_t("setup_venv_create_failed", LANG))
                return 1
            print(_t("setup_venv_created", LANG))
            created_venv = True
        else:
            print(_t("setup_step2_venv_exists", LANG))
        py = str(venv_python)

    # ── 2. Installer/mettre à jour les dépendances ──
    step = "3/3" if created_venv else "2/2"
    print(_t("setup_step_deps", LANG, step=step))
    subprocess.run([py, "-m", "pip", "install", "--upgrade", "pip", "-q"])
    pip_cmd = [py, "-m", "pip", "install", "-e", ".[spnkr]", "-q"]
    if update_mode:
        pip_cmd.insert(-1, "--upgrade")
    result = subprocess.run(pip_cmd)
    if result.returncode != 0:
        print(_t("setup_deps_failed", LANG))
        print(_t("setup_deps_causes", LANG))
        print(_t("setup_deps_no_internet", LANG))
        print(_t("setup_deps_readonly", LANG))
        print(_t("setup_deps_diskspace", LANG))
        return 1
    print(_t("setup_deps_ok", LANG))

    # ── 3. Vérification rapide ──
    check = subprocess.run(
        [py, "-c", "import uvicorn; import duckdb; import polars; print('OK')"],
        capture_output=True,
        text=True,
    )
    if check.returncode != 0:
        print(_t("setup_critical_missing", LANG))
        return 1

    print("\n" + "=" * 60)
    print(_t("setup_done", LANG))
    print("=" * 60)
    print(_t("setup_python_path", LANG, py=py))
    print(_t("setup_useful_cmds", LANG))
    print(_t("setup_cmd_run", LANG))
    print(_t("setup_cmd_doctor", LANG))
    return 0
