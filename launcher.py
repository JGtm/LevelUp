"""Lanceur LevelUp.

Architecture v5 DuckDB unifiée avec stockage partagé (shared_matches).

Usage
-----
Mode interactif (recommandé):
  python launcher.py

Commandes CLI:
  python launcher.py run              # Dashboard seul
  python launcher.py sync             # Sync tous les joueurs
  python launcher.py sync --run       # Sync + lance le dashboard

Configuration:
  - Données joueurs: data/players/{gamertag}/stats.duckdb
  - Métadonnées: data/warehouse/metadata.duckdb
"""

from __future__ import annotations

import argparse
import asyncio
import contextlib
import io
import logging
import os
import signal
import socket
import subprocess
import sys
import threading
import time
import webbrowser
from dataclasses import dataclass
from pathlib import Path

logger = logging.getLogger(__name__)

# Forcer l'encodage UTF-8 sur Windows pour les emojis
if sys.platform == "win32":
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding="utf-8", errors="replace")
    sys.stderr = io.TextIOWrapper(sys.stderr.buffer, encoding="utf-8", errors="replace")

# Ajouter src au path pour les imports
sys.path.insert(0, str(Path(__file__).resolve().parent))

# =============================================================================
# Configuration
# =============================================================================

REPO_ROOT = Path(__file__).resolve().parent
DEFAULT_STREAMLIT_APP = REPO_ROOT / "streamlit_app.py"

# Architecture v5 - Chemins DuckDB (centralisés dans src/utils/paths)
from src.utils.paths import (
    METADATA_DB_FILENAME,
    PLAYER_DB_FILENAME,
    PLAYERS_DIR,
    WAREHOUSE_DIR,
    get_metadata_db_path,
    get_pve_db_path,
    get_shared_matches_path,
)

# =============================================================================
# i18n — Détection de la langue système et traductions
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

    import locale as _locale

    candidates: list[str] = []
    try:
        lc = _locale.getlocale()[0]
        if lc:
            candidates.append(lc.lower())
    except Exception:
        pass
    try:
        lc = _locale.getdefaultlocale()[0]  # deprecated but broader support
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


try:
    from src.utils.launcher_i18n import t as _t
except ImportError:

    def _t(key: str, lang: str = "fr", **kwargs: object) -> str:  # type: ignore[misc]
        """Fallback no-op quand le module i18n n'est pas encore disponible."""
        return key


_LANG: str = _detect_lang()

# =============================================================================
# Gestion propre du Ctrl+C
# =============================================================================

_shutdown_event = threading.Event()
_active_process: subprocess.Popen | None = None
_shutdown_lock = threading.Lock()
_ctrl_c_count = 0


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

    # Sur Windows, utiliser taskkill pour tuer l'arbre de processus
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


def _signal_handler(signum: int, frame) -> None:
    """Handler pour Ctrl+C."""
    global _ctrl_c_count

    with _shutdown_lock:
        _ctrl_c_count += 1
        count = _ctrl_c_count

        if count == 1:
            _shutdown_event.set()
            print(_t("shutdown_in_progress", _LANG), flush=True)
            _kill_active_process()
        elif count >= 2:
            print(_t("shutdown_forced", _LANG), flush=True)
            _kill_active_process()
            os._exit(1)


def _install_signal_handler() -> None:
    """Installe le handler de signal."""
    signal.signal(signal.SIGINT, _signal_handler)
    if hasattr(signal, "SIGBREAK"):
        signal.signal(signal.SIGBREAK, _signal_handler)


def _check_shutdown() -> bool:
    """Vérifie si un arrêt a été demandé."""
    return _shutdown_event.is_set()


def _flush_stdin() -> None:
    """Vide le buffer d'entrée console avant une saisie interactive (Windows).

    Pendant l'installation initiale (pip, winget…) des frappes clavier
    peuvent s'accumuler dans le buffer du terminal Windows. Sans ce drain,
    la première ``input()`` les consomme instantanément, ce qui produit une
    réponse invalide et un code de sortie 2 inattendu.
    """
    if sys.platform != "win32":
        return
    try:
        import msvcrt  # noqa: PLC0415

        while msvcrt.kbhit():
            msvcrt.getwch()
    except Exception:
        pass


# =============================================================================
# Helpers Python / venv
# =============================================================================


def _preferred_python_executable() -> Path | None:
    """Trouve le python du venv local."""
    candidates = [
        REPO_ROOT / ".venv_windows" / "Scripts" / "python.exe",  # Windows (prioritaire)
        REPO_ROOT / ".venv" / "Scripts" / "python.exe",  # Windows
        REPO_ROOT / ".venv_windows" / "bin" / "python",  # Linux/macOS (prioritaire)
        REPO_ROOT / ".venv" / "bin" / "python",  # Linux/macOS
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
    os.execv(str(preferred_r), [str(preferred_r), str(Path(__file__).resolve()), *argv])


def _require_module(name: str, *, install_hint: str) -> None:
    """Vérifie qu'un module est disponible."""
    try:
        __import__(name)
    except Exception as e:
        print(_t("dep_missing", _LANG, name=name))
        print(_t("dep_detail", _LANG), e)
        print(_t("dep_install_hint", _LANG))
        print(f"  {install_hint}")
        raise SystemExit(2) from e


def _pick_free_port() -> int:
    """Trouve un port libre."""
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(("127.0.0.1", 0))
        return int(s.getsockname()[1])


# =============================================================================
# Helpers DuckDB (Architecture v5)
# =============================================================================


def _import_duckdb():
    """Importe duckdb de manière lazy."""
    try:
        import duckdb

        return duckdb
    except ImportError as err:
        print(_t("duckdb_not_installed", _LANG))
        print("   pip install duckdb")
        raise SystemExit(2) from err


@dataclass
class PlayerInfo:
    """Informations sur un joueur (architecture v5)."""

    gamertag: str
    db_path: Path
    total_matches: int
    xuid: str | None = None


def _list_players() -> list[PlayerInfo]:
    """Liste les joueurs depuis data/players/*/stats.duckdb."""
    players = []

    if not PLAYERS_DIR.exists():
        return players

    duckdb = _import_duckdb()

    for player_dir in sorted(PLAYERS_DIR.iterdir()):
        if not player_dir.is_dir():
            continue

        db_path = player_dir / PLAYER_DB_FILENAME
        if not db_path.exists():
            continue

        gamertag = player_dir.name
        total_matches = 0
        xuid = None
        db_readable = True

        try:
            con = duckdb.connect(str(db_path), read_only=True)
            try:
                # Architecture v5 : utilise player_match_enrichment si disponible
                # (plus fiable que player_match_stats qui peut contenir des stats agrégées)
                try:
                    result = con.execute("SELECT COUNT(*) FROM player_match_enrichment").fetchone()
                    total_matches = result[0] if result else 0
                except Exception:
                    # Fallback v4 : chercher match_stats
                    try:
                        result = con.execute("SELECT COUNT(*) FROM match_stats").fetchone()
                        total_matches = result[0] if result else 0
                    except Exception:
                        # Dernier fallback : player_match_stats (v5 avec sync récent)
                        try:
                            result = con.execute(
                                "SELECT COUNT(*) FROM player_match_stats"
                            ).fetchone()
                            total_matches = result[0] if result else 0
                        except Exception:
                            pass

                # Récupérer le XUID depuis sync_meta si disponible
                try:
                    result = con.execute(
                        "SELECT value FROM sync_meta WHERE key = 'xuid'"
                    ).fetchone()
                    xuid = result[0] if result else None
                except Exception:
                    pass
            finally:
                con.close()
        except Exception:
            db_readable = False

        players.append(
            PlayerInfo(
                gamertag=gamertag,
                db_path=db_path,
                total_matches=total_matches,
                xuid=xuid,
            )
        )
        if not db_readable:
            print(_t("db_unreadable", _LANG, gamertag=gamertag))

    # Trier par nombre de matchs décroissant
    players.sort(key=lambda p: p.total_matches, reverse=True)
    return players


def _get_player_db_path(gamertag: str) -> Path:
    """Retourne le chemin vers stats.duckdb d'un joueur."""
    return PLAYERS_DIR / gamertag / PLAYER_DB_FILENAME


def _player_db_exists(gamertag: str) -> bool:
    """Vérifie si la DB d'un joueur existe."""
    return _get_player_db_path(gamertag).exists()


def _count_matches_duckdb(db_path: Path) -> int:
    """Compte les matchs dans une DB DuckDB."""
    if not db_path.exists():
        return 0
    try:
        duckdb = _import_duckdb()
        con = duckdb.connect(str(db_path), read_only=True)
        try:
            result = con.execute("SELECT COUNT(*) FROM match_stats").fetchone()
            count = result[0] if result else 0
            return count
        finally:
            con.close()
    except Exception:
        return 0


def _display_path(p: Path) -> str:
    """Affiche un chemin relatif au repo."""
    try:
        return str(p.resolve().relative_to(REPO_ROOT))
    except Exception:
        return str(p)


def _metadata_db_exists() -> bool:
    """Vérifie si metadata.duckdb existe."""
    return (WAREHOUSE_DIR / METADATA_DB_FILENAME).exists()


# =============================================================================
# Synchronisation DuckDB (Architecture v5)
# =============================================================================


def _classify_sync_error(err: str, gamertag: str) -> str:
    """Retourne un message d'erreur de sync actionnable selon le type d'échec."""
    s = err.lower()
    if "invalid_grant" in s or "aadsts" in s or ("expir" in s and ("token" in s or "refresh" in s)):
        return _t("sync_error_token", _LANG, gamertag=gamertag)
    if any(
        kw in s
        for kw in (
            "could not set lock",
            "could not open lock",
            "locked by",
            "being used by another process",
        )
    ):
        return _t("sync_error_locked", _LANG)
    if any(
        kw in s
        for kw in (
            "cannot connect",
            "clientconnectorerror",
            "getaddrinfo",
            "name resolution",
            "timed out",
            "network unreachable",
            "connection refused",
        )
    ):
        return _t("sync_error_network", _LANG)
    # Erreur inconnue — afficher le message complet pour faciliter le diagnostic
    return _t("sync_error_unknown", _LANG, gamertag=gamertag, err=err)


async def _sync_player_duckdb_async(
    gamertag: str, *, delta: bool = True, max_matches: int = 100
) -> tuple[int, int]:
    """Synchronise un joueur via DuckDBSyncEngine (async).

    Returns:
        Tuple (matchs_avant, matchs_après)
    """
    try:
        from src.data.sync.engine import DuckDBSyncEngine
        from src.data.sync.models import SyncOptions
    except ImportError as e:
        print(f"  ⚠ Import error: {e}")
        return (0, 0)

    db_path = _get_player_db_path(gamertag)

    # Créer le dossier si nécessaire
    db_path.parent.mkdir(parents=True, exist_ok=True)

    # Compter les matchs avant
    matches_before = _count_matches_duckdb(db_path)

    # Récupérer les tokens via la couche auth unifiée (Device Code Flow si nécessaire)
    try:
        from src.auth.provider import get_halo_tokens

        tokens = await get_halo_tokens(db_path)
    except Exception as e:
        print(_classify_sync_error(str(e), gamertag))
        return (matches_before, matches_before)

    # Créer le moteur de sync
    try:
        engine = DuckDBSyncEngine(
            player_db_path=db_path,
            xuid="",  # Sera résolu par l'engine via gamertag
            gamertag=gamertag,
            tokens=tokens,
        )

        # Exécuter la sync
        options = SyncOptions(
            max_matches=max_matches,
            with_skill=True,
            with_aliases=True,
        )

        # Lancer la sync
        if delta:
            result = await engine.sync_delta()
        else:
            result = await engine.sync_full(options)

        if result.error:
            print(_classify_sync_error(str(result.error), gamertag))

    except Exception as e:
        print(_classify_sync_error(str(e), gamertag))
        return (matches_before, matches_before)

    # Compter les matchs après
    matches_after = _count_matches_duckdb(db_path)

    return (matches_before, matches_after)


def _sync_player_duckdb(
    gamertag: str, *, delta: bool = True, max_matches: int = 100
) -> tuple[int, int]:
    """Synchronise un joueur via DuckDBSyncEngine (wrapper sync).

    Returns:
        Tuple (matchs_avant, matchs_après)
    """
    return asyncio.run(_sync_player_duckdb_async(gamertag, delta=delta, max_matches=max_matches))


def _fetch_profile_assets(gamertag: str) -> None:
    """Récupère les assets profil du joueur."""
    try:
        from src.ui.profile_api import (
            fetch_appearance_via_spnkr,
            fetch_xuid_via_spnkr,
            save_cached_appearance,
            save_cached_xuid,
        )
    except ImportError:
        return

    print(_t("fetch_profile_assets", _LANG))

    player_str = str(gamertag).strip()
    xuid = None

    if player_str.isdigit():
        xuid = player_str
    else:
        try:
            xuid, _ = fetch_xuid_via_spnkr(gamertag=player_str)
            if xuid:
                save_cached_xuid(player_str, xuid)
        except Exception:
            pass

    if not xuid:
        return

    try:
        appearance = fetch_appearance_via_spnkr(xuid=xuid)
        if appearance:
            save_cached_appearance(xuid, appearance)
    except Exception:
        pass


# =============================================================================
# Commande: setup (remplace setup.bat / setup_env.ps1)
# =============================================================================


def _find_system_python() -> str | None:
    """Trouve un Python 3.10-3.13 sur le système."""
    import shutil

    # 1. Essayer le Python Launcher (py) — Windows uniquement
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

    # 2. Essayer les binaires versionnés (Homebrew macOS, apt Linux)
    for minor in (12, 13, 11, 10):
        exe = shutil.which(f"python3.{minor}")
        if exe:
            return exe

    # 3. Essayer python3 / python générique dans le PATH
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
        print(_t("winget_windows_only", _LANG))
        print(_t("python_manual_install", _LANG))
        return False

    import shutil

    if not shutil.which("winget"):
        print(_t("winget_unavailable", _LANG))
        print(_t("python_manual_install", _LANG))
        return False

    print(_t("winget_installing", _LANG))
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
        print(_t("winget_error", _LANG, err=e))
        return False


def _cmd_setup(args: argparse.Namespace) -> int:
    """Commande: configure l'environnement (venv + dépendances)."""
    update_mode = getattr(args, "update", False)

    print("=" * 60)
    print(_t("setup_title_update" if update_mode else "setup_title", _LANG))
    print("=" * 60)

    venv_python = REPO_ROOT / ".venv" / "Scripts" / "python.exe"
    if sys.platform != "win32":
        venv_python = REPO_ROOT / ".venv" / "bin" / "python"

    created_venv = False

    # ── 1. Vérifier/créer le venv ──
    if venv_python.exists() and not update_mode:
        print(_t("setup_venv_exists", _LANG))
        py = str(venv_python)
    else:
        print(_t("setup_step1_searching", _LANG))

        py = _find_system_python()
        if not py:
            print(_t("setup_python_not_found", _LANG))
            if _install_python_via_winget():
                py = _find_system_python()

        if not py:
            print(_t("setup_python_impossible", _LANG))
            print(_t("setup_python_install_url", _LANG))
            return 1

        print(_t("setup_python_found", _LANG, py=py))

        if not venv_python.exists():
            print(_t("setup_step2_creating_venv", _LANG))
            result = subprocess.run([py, "-m", "venv", str(REPO_ROOT / ".venv")])
            if result.returncode != 0:
                print(_t("setup_venv_create_failed", _LANG))
                return 1
            print(_t("setup_venv_created", _LANG))
            created_venv = True
        else:
            print(_t("setup_step2_venv_exists", _LANG))

        py = str(venv_python)

    # ── 2. Installer/mettre à jour les dépendances ──
    step = "3/3" if created_venv else "2/2"
    print(_t("setup_step_deps", _LANG, step=step))

    subprocess.run([py, "-m", "pip", "install", "--upgrade", "pip", "-q"])
    pip_cmd = [py, "-m", "pip", "install", "-e", ".[spnkr]", "-q"]
    if update_mode:
        pip_cmd.insert(-1, "--upgrade")

    result = subprocess.run(pip_cmd)
    if result.returncode != 0:
        print(_t("setup_deps_failed", _LANG))
        print(_t("setup_deps_causes", _LANG))
        print(_t("setup_deps_no_internet", _LANG))
        print(_t("setup_deps_readonly", _LANG))
        print(_t("setup_deps_diskspace", _LANG))
        return 1
    print(_t("setup_deps_ok", _LANG))

    # ── 3. Vérification rapide ──
    check = subprocess.run(
        [py, "-c", "import streamlit; import duckdb; import polars; print('OK')"],
        capture_output=True,
        text=True,
    )
    if check.returncode != 0:
        print(_t("setup_critical_missing", _LANG))
        return 1

    print("\n" + "=" * 60)
    print(_t("setup_done", _LANG))
    print("=" * 60)
    print(_t("setup_python_path", _LANG, py=py))
    print(_t("setup_useful_cmds", _LANG))
    print(_t("setup_cmd_run", _LANG))
    print(_t("setup_cmd_doctor", _LANG))
    return 0


# =============================================================================
# Commande: doctor (absorbe check_env.py)
# =============================================================================


def _cmd_doctor(args: argparse.Namespace) -> int:
    """Commande: vérifie la santé de l'environnement."""
    from importlib import metadata as importlib_metadata

    print("=" * 60)
    print("🩺 LEVELUP — DOCTOR")
    print("=" * 60)

    import platform as pf

    print(f"\n  OS:     {pf.system()} {pf.release()}")
    print(f"  Python: {sys.version.split()[0]}")
    print(f"  Exe:    {sys.executable}")
    print(f"  Venv:   {'oui' if sys.prefix != getattr(sys, 'base_prefix', sys.prefix) else 'non'}")

    errors: list[str] = []
    warnings: list[str] = []

    # Vérifier qu'on est dans le bon venv
    expected_venv = (REPO_ROOT / ".venv").resolve()
    if expected_venv.exists():
        _venv_py = _preferred_python_executable()
        if _venv_py is not None:
            exe_r = Path(sys.executable).resolve()
            if exe_r != _venv_py.resolve():
                errors.append(
                    _t("doctor_wrong_interpreter", _LANG, exe=exe_r, expected=_venv_py.resolve())
                )
    else:
        errors.append(_t("doctor_no_venv", _LANG))

    # Vérifier les versions des packages critiques
    expected_packages = {
        "duckdb": "1.4.4",
        "polars": "1.38.1",
        "pyarrow": "23.0.0",
        "streamlit": None,
        "spnkr": None,
    }
    for pkg, expected_ver in expected_packages.items():
        try:
            actual = importlib_metadata.version(pkg)
            status = "✓"
            if expected_ver and actual != expected_ver:
                status = "⚠"
                warnings.append(
                    _t(
                        "doctor_pkg_version_mismatch",
                        _LANG,
                        pkg=pkg,
                        actual=actual,
                        expected=expected_ver,
                    )
                )
            print(f"  {status} {pkg}=={actual}")
        except importlib_metadata.PackageNotFoundError:
            print(f"  ✗ {pkg} — MANQUANT")
            errors.append(_t("doctor_pkg_missing", _LANG, pkg=pkg))

    # Vérifier les données
    print()
    players = _list_players()
    if players:
        total = sum(p.total_matches for p in players)
        print(_t("doctor_players_info", _LANG, count=len(players), total=total))
    else:
        warnings.append(_t("doctor_no_players", _LANG))

    meta_path = WAREHOUSE_DIR / METADATA_DB_FILENAME
    if meta_path.exists():
        print(f"  ✓ metadata.duckdb ({meta_path.stat().st_size / 1024 / 1024:.1f} MB)")
    else:
        warnings.append(_t("doctor_no_metadata", _LANG))

    # Résultats
    if warnings:
        print(_t("doctor_warnings", _LANG))
        for w in warnings:
            print(f"  - {w}")

    if errors:
        print(_t("doctor_errors", _LANG))
        for e in errors:
            print(f"  - {e}")
        print(_t("doctor_fix_hint", _LANG))
        return 1

    print(_t("doctor_ok", _LANG))
    return 0


# =============================================================================
# Migrations de schéma automatiques
# =============================================================================


def _run_migrations() -> None:
    """Applique les migrations de schéma pendantes sur toutes les DB.

    Exécuté avant le lancement de Streamlit. Non-bloquant en cas d'erreur.
    """
    from src.data.migration.runner import apply_pending_migrations

    players = _list_players()
    shared_path = get_shared_matches_path()
    pve_path = get_pve_db_path()

    # Initialiser shared_matches.duckdb si absent (premier lancement avec joueurs)
    if players and not shared_path.exists():
        _ensure_warehouse_dbs()

    if not players and not shared_path.exists():
        return

    print(_t("migrations_checking", _LANG), flush=True)

    total_schemas = 0
    total_backfills = 0
    errors: list[str] = []

    # Migrations shared (une seule fois)
    try:
        meta_path = get_metadata_db_path()
        report = apply_pending_migrations(
            shared_db_path=shared_path if shared_path.exists() else None,
            pve_db_path=pve_path if pve_path.exists() else None,
            metadata_db_path=meta_path if meta_path.exists() else None,
        )
        total_schemas += report.schemas_applied
        total_backfills += report.backfills_applied
        if report.errors:
            errors.extend(report.errors)
    except Exception as e:
        errors.append(f"shared: {e}")

    # Migrations player (pour chaque joueur)
    for player in players:
        db_path = PLAYERS_DIR / player.gamertag / PLAYER_DB_FILENAME
        if not db_path.exists():
            continue
        try:
            report = apply_pending_migrations(player_db_path=db_path)
            total_schemas += report.schemas_applied
            total_backfills += report.backfills_applied
            if report.errors:
                errors.extend(report.errors)
        except Exception as e:
            errors.append(f"{player.gamertag}: {e}")

    if total_schemas == 0 and total_backfills == 0:
        print(_t("migrations_up_to_date", _LANG), flush=True)
    else:
        if total_schemas:
            print(_t("migrations_schemas_applied", _LANG, n=total_schemas), flush=True)
        if total_backfills:
            print(_t("migrations_backfills_applied", _LANG, n=total_backfills), flush=True)

    if errors:
        print(_t("migrations_non_blocking_errors", _LANG, n=len(errors)), flush=True)
        for err in errors[:5]:
            print(f"     - {err}", flush=True)


# =============================================================================
# Commandes principales
# =============================================================================


def _launch_streamlit(
    *, db_path: Path | None = None, port: int | None = None, no_browser: bool = False
) -> int:
    """Lance le dashboard Streamlit.

    Note: Dans l'architecture v5, db_path n'est plus nécessaire.
    Le dashboard détecte automatiquement les joueurs depuis data/players/.
    """
    if not DEFAULT_STREAMLIT_APP.exists():
        raise SystemExit(f"Introuvable: {DEFAULT_STREAMLIT_APP}")

    _pip_hint = ".venv\\Scripts\\python.exe" if sys.platform == "win32" else ".venv/bin/python"
    _require_module("streamlit", install_hint=f"{_pip_hint} -m pip install -e .[spnkr]")

    chosen_port = int(port) if port else _pick_free_port()
    url = f"http://localhost:{chosen_port}"

    cmd = [
        sys.executable,
        "-m",
        "streamlit",
        "run",
        str(DEFAULT_STREAMLIT_APP),
        "--server.address",
        "localhost",
        "--server.port",
        str(chosen_port),
        "--server.headless",
        "true",
    ]

    # Délai avant ouverture du navigateur pour laisser Streamlit démarrer
    STREAMLIT_STARTUP_DELAY_SECONDS = 1.2

    # Appliquer les migrations de schéma avant le lancement
    _run_migrations()

    print(_t("launching_dashboard", _LANG), flush=True)
    print(_t("launching_url", _LANG, url=url), flush=True)
    print(_t("launching_arch", _LANG), flush=True)
    print(_t("launching_data", _LANG, path=_display_path(PLAYERS_DIR)), flush=True)

    global _active_process
    # Ne pas hériter stdin pour éviter que le sous-processus bloque (ex. Cursor/IDE)
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


def _cmd_run(args: argparse.Namespace) -> int:
    """Commande: lance le dashboard."""
    # Vérifier qu'il y a des données
    players = _list_players()

    if not players:
        print(_t("run_no_data", _LANG))
        print()
        if sys.stdin.isatty():
            try:
                go = input(_t("run_configure_prompt", _LANG)).strip().lower()
            except (EOFError, KeyboardInterrupt):
                return 2
            if go in ("", "o", "oui", "y", "yes"):
                rc = _onboard_first_player()
                if rc == 0:
                    players = _list_players()
                    if players:
                        return _launch_streamlit(
                            db_path=None, port=args.port, no_browser=args.no_browser
                        )
        else:
            print(_t("run_no_tty_hint", _LANG))
        return 2

    # Afficher les infos
    total_matches = sum(p.total_matches for p in players)
    print(_t("run_stats", _LANG, count=len(players), total=total_matches), flush=True)
    for p in players:
        print(_t("run_player_row", _LANG, gamertag=p.gamertag, matches=p.total_matches), flush=True)

    return _launch_streamlit(db_path=None, port=args.port, no_browser=args.no_browser)


def _cmd_sync(args: argparse.Namespace) -> int:
    """Commande: sync tous les joueurs (architecture v5 DuckDB)."""

    # Lister les joueurs existants
    players = _list_players()

    if not players:
        print(_t("sync_no_players", _LANG))
        print(_t("sync_no_players_hint1", _LANG))
        print("   python launcher.py add-player")
        print(_t("sync_no_players_hint3", _LANG))
        print("   python scripts/sync.py --delta --gamertag <gamertag>")
        return 2

    print("=" * 60)
    print(_t("sync_title", _LANG))
    print("=" * 60)
    print(_t("sync_players_detected", _LANG, count=len(players)))
    for p in players:
        print(_t("sync_player_row", _LANG, gamertag=p.gamertag, matches=p.total_matches))

    print(_t("sync_in_progress", _LANG))

    delta_mode = not getattr(args, "full", False)
    max_matches = int(getattr(args, "max_matches", 100))

    total_new = 0
    failures = 0

    for player in players:
        if _check_shutdown():
            return 0

        print(f"\n[{player.gamertag}]")
        print(_t("sync_mode_delta" if delta_mode else "sync_mode_full", _LANG))

        try:
            before, after = _sync_player_duckdb(
                player.gamertag,
                delta=delta_mode,
                max_matches=max_matches,
            )

            new_matches = after - before
            total_new += new_matches

            if new_matches > 0:
                print(_t("sync_new_matches", _LANG, n=new_matches))
            else:
                print(_t("sync_up_to_date", _LANG, n=after))

            # Fetch assets profil
            _fetch_profile_assets(player.gamertag)

        except Exception as e:
            print(_classify_sync_error(str(e), player.gamertag))
            failures += 1

    if _check_shutdown():
        return 0

    print("\n" + "=" * 60)
    print(_t("sync_done", _LANG))
    print("=" * 60)

    # Afficher le résumé
    players_after = _list_players()
    total_matches = sum(p.total_matches for p in players_after)
    print(_t("sync_summary_players", _LANG, n=len(players_after)))
    print(_t("sync_summary_total", _LANG, n=total_matches))
    if total_new > 0:
        print(_t("sync_summary_new", _LANG, n=total_new))
    if failures > 0:
        print(_t("sync_summary_failures", _LANG, n=failures))

    # Lancer le dashboard si demandé
    if getattr(args, "run", False):
        return _launch_streamlit(db_path=None, port=None, no_browser=False)

    return 0


def _cmd_info(args: argparse.Namespace) -> int:
    """Commande: affiche les infos sur les données."""
    players = _list_players()

    if not players:
        print(_t("sync_no_players", _LANG))
        return 2

    print("=" * 60)
    print(_t("info_title", _LANG))
    print("=" * 60)

    total_matches = sum(p.total_matches for p in players)

    print(_t("info_dir", _LANG, path=_display_path(PLAYERS_DIR)))
    print(_t("info_players", _LANG, n=len(players)))
    print(_t("info_total_matches", _LANG, n=total_matches))

    print(_t("info_players_detail", _LANG))
    for p in players:
        size_mb = p.db_path.stat().st_size / (1024 * 1024) if p.db_path.exists() else 0
        print(
            _t("info_player_row", _LANG, gamertag=p.gamertag, matches=p.total_matches, size=size_mb)
        )

    # Vérifier metadata.duckdb
    metadata_path = WAREHOUSE_DIR / METADATA_DB_FILENAME
    if metadata_path.exists():
        size_mb = metadata_path.stat().st_size / (1024 * 1024)
        print(_t("info_metadata", _LANG, path=_display_path(metadata_path), size=size_mb))
    else:
        print(_t("info_no_metadata", _LANG, path=_display_path(metadata_path)))

    return 0


# =============================================================================
# Onboarding — Premier joueur
# =============================================================================


def _ensure_warehouse_dbs() -> None:
    """Crée data/warehouse/ et initialise shared_matches.duckdb et metadata.duckdb si absents (idempotent)."""
    WAREHOUSE_DIR.mkdir(parents=True, exist_ok=True)

    shared_path = get_shared_matches_path()
    if not shared_path.exists():
        try:
            from src.data.sync._engine_connections import _bootstrap_shared_matches_db

            _bootstrap_shared_matches_db(shared_path)
            print(_t("warehouse_shared_init", _LANG), flush=True)
        except Exception as exc:
            print(_t("warehouse_shared_init_fail", _LANG, err=exc), flush=True)

    meta_path = WAREHOUSE_DIR / "metadata.duckdb"
    if not meta_path.exists():
        try:
            import duckdb as _duckdb

            _conn = _duckdb.connect(str(meta_path))
            _conn.close()
            print(_t("warehouse_meta_init", _LANG), flush=True)
        except Exception as exc:
            print(_t("warehouse_meta_init_fail", _LANG, err=exc), flush=True)


def _load_dotenv_for_launcher() -> None:
    """Charge .env.local puis .env dans os.environ (ne surcharge pas les vars existantes)."""
    for name in (".env.local", ".env"):
        env_file = REPO_ROOT / name
        if not env_file.exists():
            continue
        try:
            content = env_file.read_text(encoding="utf-8")
        except Exception:
            continue
        for raw_line in content.splitlines():
            line = raw_line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, _, value = line.partition("=")
            key = key.strip()
            value = value.strip().strip('"').strip("'")
            if key and key not in os.environ:
                os.environ[key] = value


def _env_check_for_player(gamertag: str) -> dict[str, object]:
    """Vérifie la présence d'un token valide pour un gamertag.

    Avec la nouvelle couche auth (MSAL cache DuckDB), le client_id est
    intégré dans LevelUp — plus besoin de SPNKR_AZURE_CLIENT_ID.
    La vérification porte maintenant sur le cache MSAL (stats.duckdb/sync_meta)
    ou les variables d'env legacy.

    Returns:
        Dictionnaire avec les clés ``has_auth`` (bool) et ``player_token_key`` (str).
    """
    _load_dotenv_for_launcher()
    gt_norm = gamertag.upper().replace(" ", "_").replace("-", "_")
    token_key = f"SPNKR_OAUTH_REFRESH_TOKEN_{gt_norm}"

    # Vérifie en priorité le cache MSAL dans la DB joueur
    has_msal_cache = False
    try:
        from src.auth._msal import acquire_token_silent, build_msal_app, load_msal_cache

        db_path = _get_player_db_path(gamertag)
        if db_path.exists():
            _cache = load_msal_cache(db_path)
            _app = build_msal_app(_cache)
            has_msal_cache = bool(acquire_token_silent(_app))
    except Exception:
        pass

    # Fallback : variables d'env legacy
    has_env_token = bool(os.environ.get(token_key) or os.environ.get("SPNKR_OAUTH_REFRESH_TOKEN"))

    return {
        "client_id": True,  # Toujours disponible (app Azure intégrée)
        "player_token": has_msal_cache or has_env_token,
        "player_token_key": token_key,
    }


# =============================================================================
# Wizards de configuration interactive (zéro CLI)
# =============================================================================

_BOOTSTRAP_AUTH_DB = WAREHOUSE_DIR / "bootstrap_auth.duckdb"
# DB temporaire utilisée lors du premier lancement pour stocker le cache MSAL
# avant que le gamertag soit connu. Supprimée après transfert vers stats.duckdb.


def _store_xuid_in_player_db(db_path: Path, xuid: str, gamertag: str) -> None:
    """Persiste xuid et gamertag dans sync_meta de la DB joueur.

    Appelé juste après _transfer_msal_cache() pour que le moteur de sync
    trouve le XUID dès sa première instanciation (évite le warning
    «XUID non résolu» au premier lancement).
    """
    from src.utils.db import duckdb_read_write  # noqa: PLC0415

    ddl = (
        "CREATE TABLE IF NOT EXISTS sync_meta ("
        "key VARCHAR PRIMARY KEY, value VARCHAR, "
        "updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)"
    )
    with duckdb_read_write(db_path) as conn:
        conn.execute(ddl)
        for k, v in (("xuid", xuid), ("gamertag", gamertag)):
            if v:
                conn.execute(
                    "INSERT OR REPLACE INTO sync_meta (key, value, updated_at)"
                    " VALUES (?, ?, CURRENT_TIMESTAMP)",
                    (k, v),
                )


def _transfer_msal_cache(source: Path, target: Path) -> None:
    """Transfère le cache MSAL depuis la DB source vers la DB target.

    Lit la valeur JSON sérialisée dans ``sync_meta`` de source et l'écrit
    dans ``sync_meta`` de target (créée si absente).
    """
    from src.auth._constants import MSAL_CACHE_DB_KEY
    from src.utils.db import duckdb_read_only, duckdb_read_write

    logger.debug("_transfer_msal_cache: %s → %s", source.name, target.name)
    with duckdb_read_only(source) as conn:
        row = conn.execute(
            "SELECT value FROM sync_meta WHERE key = ?", (MSAL_CACHE_DB_KEY,)
        ).fetchone()
    if not row or not row[0]:
        logger.warning(
            "_transfer_msal_cache: aucun cache MSAL dans %s — transfert annulé", source.name
        )
        return
    serialized = row[0]
    ddl = (
        "CREATE TABLE IF NOT EXISTS sync_meta ("
        "key VARCHAR PRIMARY KEY, value VARCHAR, "
        "updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)"
    )
    with duckdb_read_write(target) as conn:
        conn.execute(ddl)
        conn.execute(
            "INSERT OR REPLACE INTO sync_meta (key, value, updated_at)"
            " VALUES (?, ?, CURRENT_TIMESTAMP)",
            (MSAL_CACHE_DB_KEY, serialized),
        )
    logger.info("_transfer_msal_cache: cache MSAL transféré vers %s", target.name)


def _print_device_code(user_code: str, verification_url: str, expires_in: int) -> None:
    """Affiche le code Device Code Flow de façon bien visible (encadré)."""
    # Utilise les caractères simples compatibles cmd.exe (pas les doubles ╔╗║╠╣╚╝)
    import subprocess  # noqa: PLC0415

    print(flush=True)
    print("  +----------------------------------------------------------+", flush=True)
    print(_t("dcf_box_title_line", _LANG), flush=True)
    print("  +----------------------------------------------------------+", flush=True)
    print(f"  |  URL  : {verification_url:<49}|", flush=True)
    print(f"  |  Code : {user_code:<49}|", flush=True)
    _n_min = expires_in // 60
    _exp_label = _t("dcf_box_expires_label", _LANG)
    print(f"  |  {_exp_label}: {_n_min} min{' ' * (44 - len(str(_n_min)))}|", flush=True)
    print("  +----------------------------------------------------------+", flush=True)
    print(flush=True)
    # Copier le code dans le presse-papiers Windows si possible
    with contextlib.suppress(Exception):
        subprocess.run(["clip"], input=user_code.encode(), check=False)  # noqa: S603, S607
        print(_t("dcf_clipboard", _LANG), flush=True)
    print(flush=True)


def _wizard_oauth_token(gamertag: str, client_id: str = "") -> bool:  # noqa: ARG001
    """Wizard interactif : connexion Xbox via MSAL Device Code Flow.

    Démarre le Device Code Flow via l'app Azure LevelUp intégrée (client_id
    hardcodé — plus besoin de le passer en paramètre).
    Le cache MSAL (contenant le refresh_token tourné) est persisté directement
    dans stats.duckdb du joueur — plus de sauvegarde dans .env.local.

    Args:
        gamertag: Gamertag Xbox du joueur à authentifier.
        client_id: Ignoré (app Azure intégrée dans src/auth/_constants.py).

    Returns:
        True si connexion réussie, False sinon.
    """
    import asyncio

    print()
    print("  ┌────────────────────────────────────────────────────────┐")
    print(f"  │       {_t('wizard_title_text', _LANG):<49}│")
    print("  └────────────────────────────────────────────────────────┘")
    print()

    try:
        from src.auth.provider import DeviceCodePending, start_device_flow
    except ImportError:
        print(_t("wizard_module_missing", _LANG))
        return False

    db_path = _get_player_db_path(gamertag)
    db_path.parent.mkdir(parents=True, exist_ok=True)

    print(_t("wizard_dcf_init", _LANG))
    try:
        pending: DeviceCodePending = start_device_flow(db_path)
    except Exception as exc:  # noqa: BLE001
        code = getattr(exc, "code", "unknown")
        detail = getattr(exc, "detail", str(exc))
        print(_t("wizard_dcf_error", _LANG, code=code, detail=detail))
        return False

    _print_device_code(pending.user_code, pending.verification_url, pending.expires_in)
    with contextlib.suppress(Exception):
        webbrowser.open(pending.verification_url)
    # Rappel après ouverture du navigateur (la fenêtre peut être passée en arrière-plan)
    print(_t("wizard_dcf_reminder", _LANG, code=pending.user_code))
    print(_t("wizard_dcf_waiting", _LANG))
    print()

    async def _wait() -> tuple[str, str]:
        from src.auth.provider import complete_device_flow

        return await complete_device_flow(db_path, pending)

    try:
        resolved_gamertag, xuid = asyncio.run(_wait())
        print(_t("wizard_dcf_connected", _LANG, gamertag=resolved_gamertag, xuid=xuid))
        print(_t("wizard_dcf_token_saved", _LANG))
        return True
    except Exception as exc:  # noqa: BLE001
        code = getattr(exc, "code", "unknown")
        detail = getattr(exc, "detail", str(exc))
        print(_t("wizard_dcf_failed", _LANG, code=code, detail=detail))
        return False


def _onboard_first_player() -> int:  # noqa: PLR0912, PLR0915
    """Guide interactif pour configurer et synchroniser un premier joueur.

    Wizard zéro-CLI : Device Code Flow d'abord, gamertag résolu depuis l'API
    Halo après authentification — aucune saisie manuelle requise.

    Returns:
        0 si la synchronisation a réussi, 2 sinon.
    """
    print()
    print("  ┌────────────────────────────────────────────────────────┐")
    print(f"  │       {_t('onboard_title_text', _LANG):<49}│")
    print("  └────────────────────────────────────────────────────────┘")
    print()

    if not sys.stdin.isatty():
        print(_t("onboard_non_tty", _LANG))
        print(_t("onboard_non_tty_hint", _LANG))
        return 2

    _load_dotenv_for_launcher()

    # ── Étape 1 : Connexion Xbox — Device Code Flow (gamertag résolu après auth) ──
    try:
        from src.auth.provider import DeviceCodePending, complete_device_flow, start_device_flow
    except ImportError:
        print(_t("wizard_module_missing", _LANG))
        return 2

    _BOOTSTRAP_AUTH_DB.parent.mkdir(parents=True, exist_ok=True)
    print(_t("wizard_dcf_init", _LANG))
    try:
        pending: DeviceCodePending = start_device_flow(_BOOTSTRAP_AUTH_DB)
    except Exception as exc:  # noqa: BLE001
        print(
            _t(
                "wizard_dcf_error",
                _LANG,
                code=getattr(exc, "code", "unknown"),
                detail=getattr(exc, "detail", exc),
            )
        )
        return 2

    _print_device_code(pending.user_code, pending.verification_url, pending.expires_in)
    with contextlib.suppress(Exception):
        webbrowser.open(pending.verification_url)
    # Rappel après ouverture du navigateur (la fenêtre peut être passée en arrière-plan)
    print(_t("wizard_dcf_reminder", _LANG, code=pending.user_code))
    print(_t("wizard_dcf_waiting", _LANG))
    print()

    try:
        gamertag, xuid = asyncio.run(complete_device_flow(_BOOTSTRAP_AUTH_DB, pending))
        print(_t("wizard_dcf_connected", _LANG, gamertag=gamertag, xuid=xuid))
        print()
    except Exception as exc:  # noqa: BLE001
        print(
            _t(
                "wizard_dcf_failed",
                _LANG,
                code=getattr(exc, "code", "unknown"),
                detail=getattr(exc, "detail", exc),
            )
        )
        _BOOTSTRAP_AUTH_DB.unlink(missing_ok=True)
        return 2

    # Transférer le cache MSAL dans stats.duckdb du joueur, puis nettoyer
    real_db_path = _get_player_db_path(gamertag)
    real_db_path.parent.mkdir(parents=True, exist_ok=True)
    try:
        _transfer_msal_cache(_BOOTSTRAP_AUTH_DB, real_db_path)
    except Exception as exc:
        print(_t("onboard_msal_transfer_fail", _LANG, err=exc))
    finally:
        _BOOTSTRAP_AUTH_DB.unlink(missing_ok=True)

    # Persister le xuid résolu par MSAL dès maintenant pour éviter le warning
    # «XUID non résolu» lors du premier appel à DuckDBSyncEngine.
    if xuid:
        try:
            _store_xuid_in_player_db(real_db_path, xuid, gamertag)
        except Exception as exc:
            logger.debug("_store_xuid_in_player_db: %s", exc)

    # ── Étape 2 : Initialisation des bases de données ─────────────────────────
    print()
    _ensure_warehouse_dbs()
    _run_migrations()

    # ── Étape 3 : Synchronisation ─────────────────────────────────────────────
    print()
    print(_t("onboard_sync_how", _LANG))
    print()
    print(_t("onboard_sync_choice1", _LANG))
    print(_t("onboard_sync_choice2", _LANG))
    print()

    try:
        sync_choice = input(_t("onboard_sync_prompt", _LANG)).strip() or "1"
    except (EOFError, KeyboardInterrupt):
        sync_choice = "1"

    if sync_choice == "2":
        # Sync complet direct
        print()
        print(_t("onboard_sync_full_starting", _LANG, gamertag=gamertag))
        print()
        try:
            before, after = _sync_player_duckdb(gamertag, delta=False, max_matches=200)
        except Exception as e:
            print(_classify_sync_error(str(e), gamertag))
            return 2
        new_matches = after - before
        if new_matches > 0:
            print(_t("onboard_sync_ok", _LANG, n=new_matches, gamertag=gamertag))
        elif after > 0:
            print(_t("onboard_sync_already", _LANG, n=after, gamertag=gamertag))
        else:
            print(_t("onboard_sync_no_matches", _LANG))
            return 2
        return 0

    # ── Test 10 matchs (avec retry) ──────────────────────────────────────────
    new_matches = 0
    after = 0
    while True:
        print()
        print(_t("onboard_test_starting", _LANG, gamertag=gamertag))
        print()
        test_error: str | None = None
        before = after = 0
        try:
            before, after = _sync_player_duckdb(gamertag, delta=False, max_matches=10)
        except Exception as e:
            test_error = _classify_sync_error(f"{type(e).__name__}: {e}", gamertag)

        new_matches = after - before

        if test_error or (new_matches == 0 and after == 0):
            if test_error:
                print(_t("onboard_test_failed", _LANG, err=test_error))
            else:
                print(_t("onboard_sync_no_matches", _LANG))
            print()
            print(_t("onboard_test_what_now", _LANG))
            print()
            print(_t("onboard_test_retry", _LANG))
            print(_t("onboard_test_launch_anyway", _LANG))
            print(_t("onboard_quit", _LANG))
            print()
            try:
                fail_choice = input(_t("onboard_test_prompt", _LANG)).strip().lower() or "2"
            except (EOFError, KeyboardInterrupt):
                fail_choice = "q"

            if fail_choice == "1":
                continue  # relance le test
            if fail_choice == "q":
                return 2
            return 0  # lancer quand même

        # Test concluant
        if new_matches > 0:
            print(_t("onboard_test_ok", _LANG, n=new_matches))
        else:
            print(_t("onboard_test_existing", _LANG, n=after))
        break

    # ── Proposition de poursuivre ─────────────────────────────────────────────
    print()
    print(_t("onboard_more_matches", _LANG))
    print()
    print(_t("onboard_more_continue", _LANG))
    print(_t("onboard_more_launch", _LANG))
    print()

    try:
        continue_choice = input(_t("onboard_more_prompt", _LANG)).strip() or "1"
    except (EOFError, KeyboardInterrupt):
        continue_choice = "2"

    if continue_choice != "1":
        return 0

    # ── Batches de 200 ────────────────────────────────────────────────────────
    total_new = new_matches
    batch_num = 1
    while True:
        print()
        print(_t("onboard_batch_starting", _LANG, n=batch_num))
        print()
        try:
            before_b, after_b = _sync_player_duckdb(gamertag, delta=False, max_matches=200)
        except Exception as e:
            print(_classify_sync_error(str(e), gamertag))
            break

        gained = after_b - before_b
        total_new += gained
        print(_t("onboard_batch_ok", _LANG, gained=gained, total=after_b))

        if gained == 0:
            print(_t("onboard_batch_done", _LANG))
            break

        print()
        print(_t("onboard_batch_continue", _LANG))
        print()
        print(_t("onboard_batch_yes", _LANG))
        print(_t("onboard_batch_no", _LANG))
        print()
        try:
            again = input(_t("onboard_more_prompt", _LANG)).strip() or "1"
        except (EOFError, KeyboardInterrupt):
            again = "2"
        if again != "1":
            break
        batch_num += 1

    print(_t("onboard_sync_total", _LANG, n=total_new))
    return 0


def _cmd_add_player(args: argparse.Namespace) -> int:
    """Commande: ajoute/synchronise un joueur par son gamertag."""
    gamertag = getattr(args, "gamertag", None)

    if not gamertag:
        return _onboard_first_player()

    print(_t("add_player_sync_starting", _LANG, gamertag=gamertag))

    _load_dotenv_for_launcher()
    env_info = _env_check_for_player(gamertag)

    if not env_info["player_token"]:
        print(_t("add_player_no_token", _LANG, gamertag=gamertag))
        if sys.stdin.isatty():
            ok = _wizard_oauth_token(gamertag)
            if not ok:
                return 2
        else:
            print(_t("add_player_no_tty_hint", _LANG, gamertag=gamertag))
            return 2

    full_sync = getattr(args, "full", False)
    max_matches = int(getattr(args, "max_matches", 200))

    _ensure_warehouse_dbs()

    try:
        before, after = _sync_player_duckdb(gamertag, delta=not full_sync, max_matches=max_matches)
    except Exception as e:
        print(_classify_sync_error(str(e), gamertag))
        return 2

    new_matches = after - before
    if new_matches > 0:
        print(_t("add_player_new_matches", _LANG, n=new_matches, gamertag=gamertag))
    else:
        print(_t("add_player_up_to_date", _LANG, n=after, gamertag=gamertag))
    return 0


def _cmd_reauth(args: argparse.Namespace) -> int:
    """Commande: renouvelle uniquement le token OAuth d'un joueur.

    Réutilise le client_id déjà configuré dans .env.local et relance
    uniquement le Device Code Flow MSAL pour obtenir un nouveau refresh_token.
    Utile quand le token a expiré ou a été révoqué sans avoir à recréer l'app.
    """
    gamertag = args.gamertag.strip()
    _load_dotenv_for_launcher()

    print()
    print(_t("reauth_starting", _LANG, gamertag=gamertag))
    print()

    ok = _wizard_oauth_token(gamertag)
    if not ok:
        print(_t("reauth_failed", _LANG))
        return 2

    print(_t("reauth_ok", _LANG, gamertag=gamertag))
    return 0


# =============================================================================
# Détection d'état au démarrage + menu de récupération
# =============================================================================


@dataclass
class _ConfigState:
    """État global de la configuration détecté au démarrage."""

    players: list[PlayerInfo]
    players_missing_token: list[str]  # gamertags sans token OAuth valide

    @property
    def is_first_launch(self) -> bool:
        """Aucun joueur configuré — premier démarrage."""
        return not self.players

    @property
    def is_ready(self) -> bool:
        """Tout est en ordre : peut lancer Streamlit directement."""
        return bool(self.players) and not self.players_missing_token

    @property
    def is_partial(self) -> bool:
        """Configuration incomplète : au moins un élément manquant."""
        return bool(self.players) and not self.is_ready


def _detect_config_state() -> _ConfigState:
    """Évalue l'état global de la configuration au démarrage.

    Lit .env.local, liste les joueurs et vérifie la présence de chaque
    token OAuth par gamertag.  Aucune connexion réseau n'est effectuée.
    """
    players = _list_players()
    _load_dotenv_for_launcher()
    players_missing_token: list[str] = []
    for p in players:
        gt_norm = p.gamertag.upper().replace(" ", "_").replace("-", "_")
        token_key = f"SPNKR_OAUTH_REFRESH_TOKEN_{gt_norm}"
        has_token = bool(os.environ.get(token_key) or os.environ.get("SPNKR_OAUTH_REFRESH_TOKEN"))
        if not has_token:
            players_missing_token.append(p.gamertag)
    return _ConfigState(
        players=players,
        players_missing_token=players_missing_token,
    )


def _recovery_menu(state: _ConfigState) -> int:  # noqa: PLR0912
    """Menu de récupération affiché quand la configuration est incomplète.

    S'adapte à ce qui manque : app Azure, token OAuth, ou les deux.
    Permet de reprendre ou de changer de méthode sans ligne de commande.
    Après correction, relance le flux interactif normal.
    """
    print()
    print("  ┌────────────────────────────────────────────────────────┐")
    print(f"  │       {_t('recovery_title_text', _LANG):<49}│")
    print("  └────────────────────────────────────────────────────────┘")
    print()

    if state.players_missing_token:
        missing = ", ".join(state.players_missing_token)
        print(_t("recovery_missing_token", _LANG, missing=missing))
    print()

    if not sys.stdin.isatty():
        print(_t("onboard_non_tty", _LANG))
        print(_t("recovery_non_tty_hint1", _LANG))
        print(_t("recovery_non_tty_hint2", _LANG))
        return 2

    # ── Construire les options selon l'état ───────────────────────────────────
    # Format : (action_key, label_affiché)
    options: list[tuple[str, str]] = []

    for gt in state.players_missing_token:
        options.append(
            (
                f"reauth:{gt}",
                _t("recovery_option_reauth", _LANG, gt=gt),
            )
        )
    options.append(("launch", _t("recovery_option_launch", _LANG)))

    # Quitter toujours en dernier
    options.append(("quit", _t("recovery_option_quit", _LANG)))

    for i, (_, label) in enumerate(options[:-1], 1):
        print(f"  {i}) {label}")
    print()
    print(f"  Q) {options[-1][1]}")
    print()

    keys_str = "/".join(str(i) for i in range(1, len(options))) + "/Q"
    _flush_stdin()
    try:
        choice = input(_t("recovery_prompt", _LANG, keys=keys_str)).strip().lower()
    except (EOFError, KeyboardInterrupt):
        return 2

    if choice in {"q", "quit", "exit"}:
        return 0

    try:
        idx = int(choice) - 1
    except ValueError:
        print(_t("recovery_invalid_choice", _LANG))
        return 2

    if not (0 <= idx < len(options) - 1):
        print(_t("recovery_invalid_choice", _LANG))
        return 2

    action, _ = options[idx]

    if action.startswith("reauth:"):
        gt = action.split(":", 1)[1]
        print(_t("recovery_renewing", _LANG, gt=gt))
        ok = _wizard_oauth_token(gt)
        if not ok:
            print(_t("reauth_failed", _LANG))
            return 2
        return _interactive()

    if action == "launch":
        return _launch_streamlit(db_path=None, port=None, no_browser=False)

    print(_t("recovery_invalid_choice", _LANG))
    return 2


def _interactive() -> int:
    """Menu interactif principal.

    Branche selon l'état de la configuration :
    - Premier lancement (aucun joueur) → wizard de configuration
    - Config incomplète → menu de récupération contextuel
    - Tout OK → lancement direct de Streamlit
    """
    print("=" * 60)
    print(_t("interactive_title", _LANG))
    print(_t("interactive_arch", _LANG))
    print("=" * 60)

    state = _detect_config_state()

    # ── Premier lancement ──────────────────────────────────────────────────────
    if state.is_first_launch:
        print(_t("interactive_state_header", _LANG))
        print(_t("interactive_no_player", _LANG))
        print("\n" + "-" * 60)
        print(_t("interactive_choose_action", _LANG))
        print(_t("interactive_add_player_option", _LANG))
        print(_t("interactive_add_player_desc", _LANG))
        print()
        print(_t("interactive_quit_option", _LANG))
        print()

        if not sys.stdin.isatty():
            print(_t("interactive_non_tty", _LANG))
            return 2

        _flush_stdin()
        try:
            choice = input(_t("interactive_choice_prompt", _LANG)).strip().lower()
        except (EOFError, KeyboardInterrupt):
            return 2

        if choice in {"q", "quit", "exit"}:
            return 0

        if choice == "1":
            rc = _onboard_first_player()
            if rc != 0:
                return rc
            state = _detect_config_state()
            if not state.players:
                return 2
            print()
            try:
                go = input(_t("interactive_launch_prompt", _LANG)).strip().lower()
            except (EOFError, KeyboardInterrupt):
                return 0
            if go not in ("n", "non", "no"):
                return _launch_streamlit(db_path=None, port=None, no_browser=False)
            return 0

        print(_t("interactive_invalid_choice", _LANG))
        return 2

    # ── Afficher l'état des joueurs ────────────────────────────────────────────
    print(_t("interactive_state_header", _LANG))
    total_matches = sum(p.total_matches for p in state.players)
    print(_t("interactive_storage", _LANG, path=_display_path(PLAYERS_DIR)))
    print(_t("interactive_players_count", _LANG, n=len(state.players)))
    for p in state.players:
        print(_t("interactive_player_row", _LANG, gamertag=p.gamertag, matches=p.total_matches))
    print(_t("interactive_total_matches", _LANG, n=total_matches))
    if _metadata_db_exists():
        print(_t("interactive_metadata_ok", _LANG))
    else:
        print(_t("interactive_metadata_missing", _LANG))

    # ── Config incomplète → menu de récupération ───────────────────────────────
    if state.is_partial:
        return _recovery_menu(state)

    # ── Tout OK → lancement direct ─────────────────────────────────────────────
    print(_t("interactive_all_ok", _LANG))
    return _launch_streamlit(db_path=None, port=None, no_browser=False)


def _build_parser() -> argparse.ArgumentParser:
    """Construit le parser CLI."""
    ap = argparse.ArgumentParser(
        prog="levelup",
        description="LevelUp - Dashboard Halo Infinite (Architecture DuckDB v5)",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Exemples:
  python launcher.py                           # Mode interactif
  python launcher.py run                       # Dashboard seul
  python launcher.py sync                      # Sync tous les joueurs
  python launcher.py sync --run                # Sync + dashboard
  python launcher.py add-player                # Ajouter un joueur (guidé)
  python launcher.py add-player --gamertag JGtm  # Ajouter un joueur spécifique
  python launcher.py setup                     # Installer venv + dépendances
  python launcher.py setup --update            # Mettre à jour les dépendances
  python launcher.py doctor                    # Vérifier l'environnement
  python launcher.py info                      # Affiche les infos

Architecture v5:
  - Données joueurs: data/players/{gamertag}/stats.duckdb
  - Matchs partagés: data/warehouse/shared_matches.duckdb
  - Métadonnées: data/warehouse/metadata.duckdb
""",
    )

    sub = ap.add_subparsers(dest="cmd")

    # run
    p_run = sub.add_parser("run", help="Lance le dashboard")
    p_run.add_argument("--port", type=int, default=8501, help="Port (défaut : 8501)")
    p_run.add_argument("--no-browser", action="store_true", help="Ne pas ouvrir le navigateur")
    p_run.set_defaults(func=_cmd_run)

    # sync
    p_sync = sub.add_parser("sync", help="Synchronise les données de tous les joueurs")
    p_sync.add_argument("--run", action="store_true", help="Lance le dashboard après la sync")
    p_sync.add_argument("--full", action="store_true", help="Sync complète (pas de delta)")
    p_sync.add_argument(
        "--max-matches", type=int, default=100, help="Max matchs par joueur (défaut: 100)"
    )
    p_sync.set_defaults(func=_cmd_sync)

    # info
    p_info = sub.add_parser("info", help="Affiche les informations sur les données")
    p_info.set_defaults(func=_cmd_info)

    # setup
    p_setup = sub.add_parser("setup", help="Configure l'environnement (venv + dépendances)")
    p_setup.add_argument(
        "--update", action="store_true", help="Met à jour les dépendances existantes"
    )
    p_setup.set_defaults(func=_cmd_setup)

    # doctor
    p_doctor = sub.add_parser("doctor", help="Vérifie la santé de l'environnement")
    p_doctor.set_defaults(func=_cmd_doctor)

    # add-player
    p_add = sub.add_parser("add-player", help="Ajoute/synchronise un joueur par son gamertag")
    p_add.add_argument("--gamertag", type=str, default=None, help="Gamertag Xbox du joueur")
    p_add.add_argument("--full", action="store_true", help="Sync complète (pas de delta)")
    p_add.add_argument("--max-matches", type=int, default=200, help="Max matchs (défaut: 200)")
    p_add.set_defaults(func=_cmd_add_player)

    # reauth
    p_reauth = sub.add_parser(
        "reauth",
        help="Renouvelle uniquement le token OAuth d'un joueur (sans recréer l'app Azure)",
    )
    p_reauth.add_argument("--gamertag", type=str, required=True, help="Gamertag Xbox du joueur")
    p_reauth.set_defaults(func=_cmd_reauth)

    return ap


# =============================================================================
# Point d'entrée
# =============================================================================


def main(argv: list[str] | None = None) -> int:
    """Point d'entrée principal."""
    _install_signal_handler()

    argv = list(sys.argv[1:] if argv is None else argv)
    _maybe_reexec_into_venv(argv)

    try:
        if not argv:
            return _interactive()

        ap = _build_parser()
        args = ap.parse_args(argv)

        if not getattr(args, "cmd", None):
            ap.print_help()
            return 2

        return int(args.func(args))

    except KeyboardInterrupt:
        if not _shutdown_event.is_set():
            print("\n⏹ Arrêt en cours...", flush=True)
        return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except KeyboardInterrupt:
        sys.exit(0)
