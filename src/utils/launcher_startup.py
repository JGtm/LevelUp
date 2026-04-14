"""Démarrage React/FastAPI pour le lanceur LevelUp.

Contient : état global (processus actif, shutdown), gestionnaire de signal,
``_launch_react`` (point d'entrée principal — FastAPI + Vite).
Extrait de launcher.py (housekeeping Slice 9).
"""

from __future__ import annotations

import contextlib
import subprocess
import sys
import threading
import time
import webbrowser

from src.utils.launcher_env import LANG, _require_module
from src.utils.launcher_i18n import t as _t
from src.utils.launcher_migrations import _run_db_healthcheck, _run_migrations

# =============================================================================
# État global — processus Streamlit + shutdown
# =============================================================================

_shutdown_event = threading.Event()
_active_process: subprocess.Popen | None = None
_active_process_web: subprocess.Popen | None = None
_shutdown_lock = threading.Lock()
_ctrl_c_count = 0

# =============================================================================
# Gestion propre du Ctrl+C
# =============================================================================


def _subprocess_creation_flags() -> int:
    """Retourne les flags pour le sous-processus.

    Note: On n'utilise PAS CREATE_NEW_PROCESS_GROUP pour que Ctrl+C
    soit propagé au processus enfant.
    """
    return 0


def _kill_active_process() -> None:
    """Termine le(s) processus enfant(s) actif(s)."""
    procs = [p for p in (_active_process, _active_process_web) if p is not None]
    for proc in procs:
        if sys.platform == "win32":
            with contextlib.suppress(Exception):
                subprocess.run(
                    ["taskkill", "/F", "/T", "/PID", str(proc.pid)],
                    capture_output=True,
                    timeout=5,
                )
        with contextlib.suppress(Exception):
            proc.terminate()
        with contextlib.suppress(Exception):
            proc.kill()


def _signal_handler(signum: int, frame) -> None:  # noqa: ANN001
    """Handler pour Ctrl+C."""
    global _ctrl_c_count

    with _shutdown_lock:
        _ctrl_c_count += 1
        count = _ctrl_c_count

        if count == 1:
            _shutdown_event.set()
            print(_t("shutdown_in_progress", LANG), flush=True)
            _kill_active_process()
        elif count >= 2:
            print(_t("shutdown_forced", LANG), flush=True)
            _kill_active_process()
            import os  # noqa: PLC0415

            os._exit(1)


def _install_signal_handler() -> None:
    """Installe le handler de signal."""
    import signal  # noqa: PLC0415

    signal.signal(signal.SIGINT, _signal_handler)
    if hasattr(signal, "SIGBREAK"):
        signal.signal(signal.SIGBREAK, _signal_handler)


def _check_shutdown() -> bool:
    """Vérifie si un arrêt a été demandé."""
    return _shutdown_event.is_set()


def _flush_stdin() -> None:
    """Vide le buffer d'entrée console avant une saisie interactive (Windows)."""
    if sys.platform != "win32":
        return
    try:
        import msvcrt  # noqa: PLC0415

        while msvcrt.kbhit():
            msvcrt.getwch()
    except Exception:
        pass


# =============================================================================
# Lancement React + FastAPI (point d'entrée principal depuis Slice 9)
# =============================================================================

_REACT_API_PORT_DEFAULT = 8000
_REACT_WEB_PORT_DEFAULT = 5173
_REACT_STARTUP_DELAY_SECONDS = 3.0


def _launch_react(
    *,
    db_path=None,  # ignoré — non utilisé dans l'architecture React/FastAPI
    port: int | None = None,
    no_browser: bool = False,
) -> int:
    """Lance le dashboard React + API FastAPI.

    Démarre deux sous-processus en parallèle :
    - uvicorn apps.api.app.main:app (port ``port`` ou 8000)
    - npm run dev dans apps/web/ (port 5173)

    Ouvre le navigateur sur http://localhost:5173 après démarrage.
    """
    global _active_process, _active_process_web

    from src.utils.paths import REPO_ROOT  # noqa: PLC0415

    _require_module("uvicorn", install_hint=".venv/Scripts/python.exe -m pip install -e .[api]")

    _run_migrations()
    _run_db_healthcheck()

    api_port = int(port) if port else _REACT_API_PORT_DEFAULT
    web_port = _REACT_WEB_PORT_DEFAULT
    api_url = f"http://localhost:{api_port}"
    web_url = f"http://localhost:{web_port}"

    npm_cmd = "npm.cmd" if sys.platform == "win32" else "npm"

    cmd_api = [
        sys.executable,
        "-m",
        "uvicorn",
        "apps.api.app.main:app",
        "--host",
        "127.0.0.1",
        "--port",
        str(api_port),
        "--reload",
    ]
    cmd_web = [npm_cmd, "run", "dev"]
    web_dir = REPO_ROOT / "apps" / "web"

    print(_t("launching_dashboard", LANG), flush=True)
    print(f"  → API  : {api_url}", flush=True)
    print(f"  → Web  : {web_url}", flush=True)

    proc_api = subprocess.Popen(
        cmd_api,
        cwd=str(REPO_ROOT),
        stdin=subprocess.DEVNULL,
        creationflags=_subprocess_creation_flags(),
    )
    proc_web = subprocess.Popen(
        cmd_web,
        cwd=str(web_dir),
        stdin=subprocess.DEVNULL,
        creationflags=_subprocess_creation_flags(),
    )

    _active_process = proc_api
    _active_process_web = proc_web

    if not no_browser:
        time.sleep(_REACT_STARTUP_DELAY_SECONDS)
        with contextlib.suppress(Exception):
            webbrowser.open(web_url)

    try:
        while True:
            if proc_api.poll() is not None or proc_web.poll() is not None:
                break
            time.sleep(0.5)
        return 0
    except KeyboardInterrupt:
        return 0
    finally:
        _active_process = None
        _active_process_web = None
        for proc in (proc_api, proc_web):
            if proc.poll() is None:
                if sys.platform == "win32":
                    with contextlib.suppress(Exception):
                        subprocess.run(
                            ["taskkill", "/F", "/T", "/PID", str(proc.pid)],
                            capture_output=True,
                            timeout=5,
                        )
                with contextlib.suppress(Exception):
                    proc.terminate()
                    proc.wait(timeout=3)
                with contextlib.suppress(Exception):
                    proc.kill()
