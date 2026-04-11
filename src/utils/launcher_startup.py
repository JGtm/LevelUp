"""Démarrage Streamlit et gestion des signaux pour le lanceur LevelUp.

Contient : état global (processus actif, shutdown), gestionnaire de signal,
et ``_launch_streamlit`` qui lance le dashboard.
Extrait de launcher.py (F8 post-v7 housekeeping).
"""

from __future__ import annotations

import contextlib
import subprocess
import sys
import threading
import time
import webbrowser
from pathlib import Path

from src.utils.launcher_env import LANG, _pick_free_port, _require_module
from src.utils.launcher_i18n import t as _t
from src.utils.launcher_migrations import _run_db_healthcheck, _run_migrations

# =============================================================================
# État global — processus Streamlit + shutdown
# =============================================================================

_shutdown_event = threading.Event()
_active_process: subprocess.Popen | None = None
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
    """Termine le processus enfant actif."""
    proc = _active_process
    if proc is None:
        return

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
# Lancement Streamlit
# =============================================================================


def _launch_streamlit(
    *, db_path: Path | None = None, port: int | None = None, no_browser: bool = False
) -> int:
    """Lance le dashboard Streamlit.

    Note: Dans l'architecture v5, db_path n'est plus nécessaire.
    Le dashboard détecte automatiquement les joueurs depuis data/players/.
    """
    from src.utils.paths import REPO_ROOT  # noqa: PLC0415

    v7_app = REPO_ROOT / "streamlit_app_v7.py"
    legacy_app = REPO_ROOT / "streamlit_app.py"
    default_app = v7_app if v7_app.exists() else legacy_app
    if not default_app.exists():
        raise SystemExit(f"Introuvable: {default_app}")

    _pip_hint = ".venv\\Scripts\\python.exe" if sys.platform == "win32" else ".venv/bin/python"
    _require_module("streamlit", install_hint=f"{_pip_hint} -m pip install -e .[spnkr]")

    chosen_port = int(port) if port else _pick_free_port()
    url = f"http://localhost:{chosen_port}"

    cmd = [
        sys.executable,
        "-m",
        "streamlit",
        "run",
        str(default_app),
        "--server.address",
        "localhost",
        "--server.port",
        str(chosen_port),
        "--server.headless",
        "true",
    ]

    STREAMLIT_STARTUP_DELAY_SECONDS = 1.2

    _run_migrations()
    _run_db_healthcheck()

    print(_t("launching_dashboard", LANG), flush=True)
    print(_t("launching_url", LANG, url=url), flush=True)
    print(_t("launching_arch", LANG), flush=True)
    print(f"  → App Streamlit: {default_app.name}", flush=True)

    from src.utils.launcher_players import PLAYERS_DIR, _display_path  # noqa: PLC0415

    print(_t("launching_data", LANG, path=_display_path(PLAYERS_DIR)), flush=True)

    global _active_process
    proc = subprocess.Popen(
        cmd,
        cwd=str(REPO_ROOT),
        stdin=subprocess.DEVNULL,
        creationflags=_subprocess_creation_flags(),
    )
    _active_process = proc

    if not no_browser:
        time.sleep(STREAMLIT_STARTUP_DELAY_SECONDS)
        with contextlib.suppress(Exception):
            webbrowser.open(url)

    try:
        return int(proc.wait())
    except KeyboardInterrupt:
        return 0
    finally:
        _active_process = None
        if proc.poll() is None:
            with contextlib.suppress(Exception):
                proc.terminate()
                proc.wait(timeout=3)
            with contextlib.suppress(Exception):
                proc.kill()
