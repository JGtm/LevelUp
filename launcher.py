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
    get_pve_db_path,
    get_shared_matches_path,
)

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
            print("\n⏹ Arrêt en cours (Ctrl+C à nouveau pour forcer)...", flush=True)
            _kill_active_process()
        elif count >= 2:
            print("\n⚠ Arrêt forcé.", flush=True)
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
        print(f"Dépendance manquante: {name}")
        print("Détail:", e)
        print("Installe-la puis relance:")
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
        print("❌ DuckDB non installé. Exécute:")
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
            print(f"  ⚠ {gamertag}/stats.duckdb illisible (fichier corrompu ?)")

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
        return (
            f"  ⚠ Token OAuth expiré pour {gamertag}\n"
            "  → Relance LevelUp puis choisis « Ajouter un joueur » pour renouveler le token."
        )
    if any(
        kw in s
        for kw in (
            "could not set lock",
            "could not open lock",
            "locked by",
            "being used by another process",
        )
    ):
        return (
            "  ⚠ Base de données verrouillée.\n"
            "  → Ferme le dashboard LevelUp (Streamlit) avant de synchroniser."
        )
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
        return (
            "  ⚠ Impossible de joindre les serveurs Halo.\n"
            "  → Vérifie ta connexion internet et réessaie."
        )
    return f"  ⚠ Erreur sync ({gamertag}) : {err}"


async def _sync_player_duckdb_async(
    gamertag: str, *, delta: bool = True, max_matches: int = 100
) -> tuple[int, int]:
    """Synchronise un joueur via DuckDBSyncEngine (async).

    Returns:
        Tuple (matchs_avant, matchs_après)
    """
    try:
        from src.data.sync.api_client import get_tokens_from_env
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

    # Récupérer les tokens (async)
    try:
        tokens = await get_tokens_from_env()
    except SystemExit:
        print("  ⚠ Tokens non configurés (SPNKR_SPARTAN_TOKEN, SPNKR_CLEARANCE_TOKEN)")
        return (matches_before, matches_before)
    except Exception as e:
        print(_classify_sync_error(str(e), gamertag))
        return (matches_before, matches_before)

    if not tokens:
        print("  ⚠ Tokens non configurés")
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

    print("  → Fetch assets profil...")

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
        print("  ⚠ Installation automatique uniquement disponible sur Windows.")
        print("  Installez Python manuellement: https://www.python.org/downloads/")
        return False

    import shutil

    if not shutil.which("winget"):
        print("  ⚠ winget non disponible sur ce système.")
        print("  Installez Python manuellement: https://www.python.org/downloads/")
        return False

    print("  → Installation de Python 3.12 via winget...")
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
        print(f"  ⚠ Erreur winget: {e}")
        return False


def _cmd_setup(args: argparse.Namespace) -> int:
    """Commande: configure l'environnement (venv + dépendances)."""
    update_mode = getattr(args, "update", False)

    print("=" * 60)
    print("⚙️  LEVELUP — SETUP" + (" (mise à jour)" if update_mode else ""))
    print("=" * 60)

    venv_python = REPO_ROOT / ".venv" / "Scripts" / "python.exe"
    if sys.platform != "win32":
        venv_python = REPO_ROOT / ".venv" / "bin" / "python"

    created_venv = False

    # ── 1. Vérifier/créer le venv ──
    if venv_python.exists() and not update_mode:
        print("\n[1/3] Environnement .venv déjà présent ✓")
        py = str(venv_python)
    else:
        print("\n[1/3] Recherche de Python...")

        py = _find_system_python()
        if not py:
            print("  Python 3.10+ non trouvé sur le système.")
            if _install_python_via_winget():
                py = _find_system_python()

        if not py:
            print("\n  ❌ Impossible de trouver Python 3.10+.")
            print("  Installez-le depuis https://www.python.org/downloads/")
            return 1

        print(f"  Python trouvé: {py}")

        if not venv_python.exists():
            print("\n[2/3] Création de l'environnement virtuel...")
            result = subprocess.run([py, "-m", "venv", str(REPO_ROOT / ".venv")])
            if result.returncode != 0:
                print("  ❌ Impossible de créer le venv.")
                return 1
            print("  ✓ .venv créé")
            created_venv = True
        else:
            print("\n[2/3] Environnement .venv existant ✓")

        py = str(venv_python)

    # ── 2. Installer/mettre à jour les dépendances ──
    step = "3/3" if created_venv else "2/2"
    print(f"\n[{step}] Installation des dépendances...")

    subprocess.run([py, "-m", "pip", "install", "--upgrade", "pip", "-q"])
    pip_cmd = [py, "-m", "pip", "install", "-e", ".[spnkr]", "-q"]
    if update_mode:
        pip_cmd.insert(-1, "--upgrade")

    result = subprocess.run(pip_cmd)
    if result.returncode != 0:
        print("  ❌ L'installation des dépendances a échoué.")
        print("  Causes possibles :")
        print("  - Pas de connexion internet")
        print("  - Dossier en lecture seule (déplace LevelUp dans Documents)")
        print("  - Espace disque insuffisant")
        return 1
    print("  ✓ Dépendances installées")

    # ── 3. Vérification rapide ──
    check = subprocess.run(
        [py, "-c", "import streamlit; import duckdb; import polars; print('OK')"],
        capture_output=True,
        text=True,
    )
    if check.returncode != 0:
        print("  ⚠ Packages critiques manquants après installation.")
        return 1

    print("\n" + "=" * 60)
    print("✅ SETUP TERMINÉ")
    print("=" * 60)
    print(f"\n  Python: {py}")
    print("  Commandes utiles:")
    print("    python launcher.py run     # Lancer le dashboard")
    print("    python launcher.py doctor  # Vérifier l'environnement")
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
                errors.append(f"Mauvais interpréteur: {exe_r} (attendu {_venv_py.resolve()})")
    else:
        errors.append("Dossier .venv introuvable — lancez: python launcher.py setup")

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
                warnings.append(f"{pkg}: {actual} (attendu {expected_ver})")
            print(f"  {status} {pkg}=={actual}")
        except importlib_metadata.PackageNotFoundError:
            print(f"  ✗ {pkg} — MANQUANT")
            errors.append(f"Package manquant: {pkg}")

    # Vérifier les données
    print()
    players = _list_players()
    if players:
        total = sum(p.total_matches for p in players)
        print(f"  📊 {len(players)} joueur(s), {total} matchs")
    else:
        warnings.append("Aucune donnée joueur trouvée dans data/players/")

    meta_path = WAREHOUSE_DIR / METADATA_DB_FILENAME
    if meta_path.exists():
        print(f"  ✓ metadata.duckdb ({meta_path.stat().st_size / 1024 / 1024:.1f} MB)")
    else:
        warnings.append("metadata.duckdb manquant")

    # Résultats
    if warnings:
        print("\n⚠ Avertissements:")
        for w in warnings:
            print(f"  - {w}")

    if errors:
        print("\n❌ Erreurs:")
        for e in errors:
            print(f"  - {e}")
        print("\n  → Corrigez avec: python launcher.py setup")
        return 1

    print("\n✅ Environnement OK")
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

    print("\n🔧 Vérification du schéma de données…", flush=True)

    total_schemas = 0
    total_backfills = 0
    errors: list[str] = []

    # Migrations shared (une seule fois)
    try:
        report = apply_pending_migrations(
            shared_db_path=shared_path if shared_path.exists() else None,
            pve_db_path=pve_path if pve_path.exists() else None,
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
        print("   ✓ Schéma à jour", flush=True)
    else:
        if total_schemas:
            print(f"   ✓ {total_schemas} migration(s) de schéma appliquée(s)", flush=True)
        if total_backfills:
            print(f"   ✓ {total_backfills} backfill(s) exécuté(s)", flush=True)

    if errors:
        print(f"   ⚠ {len(errors)} erreur(s) non-bloquante(s):", flush=True)
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

    print("\n\ud83d\ude80 Lancement du dashboard\u2026", flush=True)
    print(f"   URL: {url}", flush=True)
    print("   Architecture: DuckDB v5", flush=True)
    print(f"   Donn\u00e9es: {_display_path(PLAYERS_DIR)}", flush=True)

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
        print("❌ Aucune donnée joueur trouvée")
        print()
        if sys.stdin.isatty():
            try:
                go = input("  Configurer un premier joueur maintenant ? [O/n] : ").strip().lower()
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
            print("   Lance : python launcher.py add-player --gamertag <gamertag>")
        return 2

    # Afficher les infos
    total_matches = sum(p.total_matches for p in players)
    print(
        f"\n\ud83d\udcca Architecture DuckDB v5: {len(players)} joueur(s), {total_matches} matchs",
        flush=True,
    )
    for p in players:
        print(f"   - {p.gamertag}: {p.total_matches} matchs", flush=True)

    return _launch_streamlit(db_path=None, port=args.port, no_browser=args.no_browser)


def _cmd_sync(args: argparse.Namespace) -> int:
    """Commande: sync tous les joueurs (architecture v5 DuckDB)."""

    # Lister les joueurs existants
    players = _list_players()

    if not players:
        print("❌ Aucun joueur trouvé dans data/players/")
        print("\n   Pour synchroniser un premier joueur :")
        print("   python launcher.py add-player")
        print("\n   Ou directement en ligne de commande :")
        print("   python scripts/sync.py --delta --gamertag <gamertag>")
        return 2

    print("=" * 60)
    print("🔄 SYNCHRONISATION (DuckDB v5)")
    print("=" * 60)
    print(f"\n   {len(players)} joueur(s) détecté(s):")
    for p in players:
        print(f"   - {p.gamertag}: {p.total_matches} matchs")

    print("\n📥 Synchronisation en cours...")

    delta_mode = not getattr(args, "full", False)
    max_matches = int(getattr(args, "max_matches", 100))

    total_new = 0
    failures = 0

    for player in players:
        if _check_shutdown():
            return 0

        print(f"\n[{player.gamertag}]")
        print(f"  → Sync {'delta' if delta_mode else 'complète'}...")

        try:
            before, after = _sync_player_duckdb(
                player.gamertag,
                delta=delta_mode,
                max_matches=max_matches,
            )

            new_matches = after - before
            total_new += new_matches

            if new_matches > 0:
                print(f"  ✓ {new_matches} nouveau(x) match(s)")
            else:
                print(f"  ✓ À jour ({after} matchs)")

            # Fetch assets profil
            _fetch_profile_assets(player.gamertag)

        except Exception as e:
            print(_classify_sync_error(str(e), player.gamertag))
            failures += 1

    if _check_shutdown():
        return 0

    print("\n" + "=" * 60)
    print("✅ SYNCHRONISATION TERMINÉE")
    print("=" * 60)

    # Afficher le résumé
    players_after = _list_players()
    total_matches = sum(p.total_matches for p in players_after)
    print(f"\n   Joueurs: {len(players_after)}")
    print(f"   Total matchs: {total_matches}")
    if total_new > 0:
        print(f"   Nouveaux: +{total_new}")
    if failures > 0:
        print(f"   ⚠ Échecs: {failures}")

    # Lancer le dashboard si demandé
    if getattr(args, "run", False):
        return _launch_streamlit(db_path=None, port=None, no_browser=False)

    return 0


def _cmd_info(args: argparse.Namespace) -> int:
    """Commande: affiche les infos sur les données."""
    players = _list_players()

    if not players:
        print("❌ Aucun joueur trouvé dans data/players/")
        return 2

    print("=" * 60)
    print("📊 INFORMATIONS (DuckDB v5)")
    print("=" * 60)

    total_matches = sum(p.total_matches for p in players)

    print(f"\n   Dossier: {_display_path(PLAYERS_DIR)}")
    print(f"   Joueurs: {len(players)}")
    print(f"   Total matchs: {total_matches}")

    print("\n   Détail par joueur:")
    for p in players:
        size_mb = p.db_path.stat().st_size / (1024 * 1024) if p.db_path.exists() else 0
        print(f"   - {p.gamertag}: {p.total_matches} matchs ({size_mb:.1f} MB)")

    # Vérifier metadata.duckdb
    metadata_path = WAREHOUSE_DIR / METADATA_DB_FILENAME
    if metadata_path.exists():
        size_mb = metadata_path.stat().st_size / (1024 * 1024)
        print(f"\n   Métadonnées: {_display_path(metadata_path)} ({size_mb:.1f} MB)")
    else:
        print(f"\n   ⚠ Métadonnées non trouvées: {_display_path(metadata_path)}")

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
            print("  ✓ shared_matches.duckdb initialisé", flush=True)
        except Exception as exc:
            print(f"  ⚠ Impossible d'initialiser shared_matches.duckdb : {exc}", flush=True)

    meta_path = WAREHOUSE_DIR / "metadata.duckdb"
    if not meta_path.exists():
        try:
            import duckdb as _duckdb

            _conn = _duckdb.connect(str(meta_path))
            _conn.close()
            print("  ✓ metadata.duckdb initialisé", flush=True)
        except Exception as exc:
            print(f"  ⚠ Impossible d'initialiser metadata.duckdb : {exc}", flush=True)


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
    """Vérifie la présence des variables d'env requises pour un gamertag.

    Returns:
        Dictionnaire avec les clés « client_id »,
        « player_token » (bool) et « player_token_key » (str).
    """
    _load_dotenv_for_launcher()
    gt_norm = gamertag.upper().replace(" ", "_").replace("-", "_")
    token_key = f"SPNKR_OAUTH_REFRESH_TOKEN_{gt_norm}"
    return {
        "client_id": bool(os.environ.get("SPNKR_AZURE_CLIENT_ID")),
        "player_token": bool(
            os.environ.get(token_key) or os.environ.get("SPNKR_OAUTH_REFRESH_TOKEN")
        ),
        "player_token_key": token_key,
    }


def _print_token_setup_instructions(gamertag: str, token_key: str) -> None:
    """Affiche les instructions pour obtenir et configurer le refresh token OAuth."""
    print()
    print("  ┌────────────────────────────────────────────────────────┐")
    print("  │          Configuration du token OAuth Azure            │")
    print("  └────────────────────────────────────────────────────────┘")
    print()
    print("  Token manquant. Lance la commande suivante pour l'obtenir")
    print("  via Device Code Flow (sans client_secret, sans redirect URI) :")
    print()
    print("       python scripts/spnkr_get_refresh_token.py --device-code")
    print()
    print(f"  Puis vérifie que {token_key} est bien défini dans .env.local")
    print()
    print("  Relance LevelUp une fois le token configuré.")
    print()


# =============================================================================
# Wizards de configuration interactive (zéro CLI)
# =============================================================================


def _upsert_env_key(path: Path, key: str, value: str) -> None:
    """Écrit ou met à jour une clé dans un fichier .env / .env.local."""
    path.parent.mkdir(parents=True, exist_ok=True)
    lines: list[str] = []
    if path.exists():
        with contextlib.suppress(Exception):
            lines = path.read_text(encoding="utf-8").splitlines()
    new_lines: list[str] = []
    replaced = False
    for line in lines:
        if line.strip().startswith(f"{key}="):
            new_lines.append(f"{key}={value}")
            replaced = True
        else:
            new_lines.append(line)
    if not replaced:
        if new_lines and new_lines[-1].strip():
            new_lines.append("")
        new_lines.append(f"{key}={value}")
    path.write_text("\n".join(new_lines) + "\n", encoding="utf-8")


def _extract_code_from_url(value: str) -> str:
    """Extrait le code OAuth depuis une URL de redirection ou retourne la valeur brute."""
    from urllib.parse import parse_qs, urlparse

    v = (value or "").strip()
    if not v or ("code=" not in v and "http" not in v):
        return v
    try:
        qs = parse_qs(urlparse(v).query)
        code = (qs.get("code") or [""])[0]
        if code:
            return str(code).strip()
    except Exception:
        pass
    if "code=" in v:
        return v.split("code=", 1)[1].split("&", 1)[0].strip()
    return v


def _extract_oauth_error_from_url(value: str) -> tuple[str | None, str | None]:
    """Extrait (error, description) depuis une URL OAuth de redirection."""
    from urllib.parse import parse_qs, urlparse

    v = (value or "").strip()
    if "error=" not in v:
        return None, None
    try:
        qs = parse_qs(urlparse(v).query)
        err = (qs.get("error") or [None])[0]
        desc = (qs.get("error_description") or [None])[0]
        if err:
            return str(err), (str(desc) if desc else None)
    except Exception:
        pass
    err = v.split("error=", 1)[1].split("&", 1)[0].strip() or None
    return err, None


async def _async_exchange_oauth_code_v2(
    auth_code: str, client_id: str, client_secret: str, redirect_uri: str
) -> str:
    """Échange un code OAuth contre un refresh_token (endpoint Microsoft v2 consumers)."""
    from aiohttp import ClientSession

    url = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"
    data = {
        "client_id": client_id,
        "client_secret": client_secret,
        "grant_type": "authorization_code",
        "code": auth_code,
        "redirect_uri": redirect_uri,
        "scope": "Xboxlive.signin Xboxlive.offline_access",
    }
    async with ClientSession() as session:
        resp = await session.post(url, data=data)
        payload = await resp.json()
    if resp.status >= 400:
        err = payload.get("error", "unknown")
        desc = str(payload.get("error_description", ""))[:200]
        raise ValueError(f"{err}: {desc}")
    token = payload.get("refresh_token")
    if not isinstance(token, str) or not token.strip():
        raise ValueError("Pas de refresh_token dans la réponse OAuth")
    return token.strip()


def _try_azure_auto_register() -> str | None:
    """Tente d'inscrire automatiquement une app Azure via la CLI az.

    Si az n'est pas trouvé, propose de l'installer (winget/brew/curl selon
    la plateforme) et réessaie après installation.
    Vérifie la présence de az, se connecte si nécessaire, crée l'app
    « LevelUp Halo » (compte Microsoft personnel, public client flow),
    et retourne l'appId (client_id) en cas de succès, None sinon.
    """
    import json as _json
    import shutil

    az_cmd = shutil.which("az")
    if az_cmd is None:
        if not _offer_install_azure_cli():
            return None
        # Recheck after attempted install
        az_cmd = shutil.which("az")
        if az_cmd is None:
            print("  ⚠  Azure CLI introuvable après installation.")
            print("     Fermez et rouvrez votre terminal, puis relancez LevelUp.")
            return None

    print("  🔍 Azure CLI (az) détecté — inscription automatique possible.")
    print()

    # Vérifier si déjà connecté
    check = subprocess.run(
        [az_cmd, "account", "show", "--output", "json"],
        capture_output=True,
        text=True,
        timeout=15,
    )
    if check.returncode != 0:
        print("  → Connexion à votre compte Microsoft (az login)…")
        print("     (Une fenêtre de navigateur va s'ouvrir)")
        print()
        login = subprocess.run(
            [az_cmd, "login", "--allow-no-subscriptions"],
            timeout=180,
        )
        if login.returncode != 0:
            print("  ❌ Connexion az échouée.")
            return None

    # Vérifier si une app LevelUp Halo existe déjà
    existing = subprocess.run(
        [
            az_cmd,
            "ad",
            "app",
            "list",
            "--display-name",
            "LevelUp Halo",
            "--query",
            "[0].appId",
            "--output",
            "tsv",
        ],
        capture_output=True,
        text=True,
        timeout=30,
    )
    if existing.returncode == 0 and existing.stdout.strip():
        existing_id = existing.stdout.strip()
        print(f"  ✅ Application existante trouvée. Client ID : {existing_id}")
        return existing_id

    # Créer l'application
    print("  → Création de l'application Azure « LevelUp Halo »…")
    create = subprocess.run(
        [
            az_cmd,
            "ad",
            "app",
            "create",
            "--display-name",
            "LevelUp Halo",
            "--sign-in-audience",
            "AzureADandPersonalMicrosoftAccount",
            "--output",
            "json",
        ],
        capture_output=True,
        text=True,
        timeout=60,
    )
    if create.returncode != 0:
        print(f"  ❌ Création app échouée : {create.stderr[:300]}")
        return None

    try:
        app_data = _json.loads(create.stdout)
        client_id = str(app_data.get("appId", "")).strip()
        if not client_id:
            return None
    except (_json.JSONDecodeError, KeyError, AttributeError):
        return None

    # Activer le public client flow (Device Code — pas de secret, pas de redirect URI)
    subprocess.run(
        [
            az_cmd,
            "ad",
            "app",
            "update",
            "--id",
            client_id,
            "--set",
            "isFallbackPublicClient=true",
            "--output",
            "none",
        ],
        capture_output=True,
        timeout=30,
    )

    print(f"  ✅ Application créée ! Client ID : {client_id}")
    return client_id


def _offer_install_azure_cli() -> bool:
    """Propose d'installer Azure CLI si absent.

    Adapté à la plateforme :
    - Windows : winget install Microsoft.AzureCLI
    - macOS   : brew install azure-cli  (si brew disponible)
    - Linux   : script officiel curl | bash

    Retourne True si l'installation a été lancée (ou déjà présente),
    False si l'utilisateur a refusé ou si le terminal est non interactif.
    """

    if not sys.stdin.isatty():
        return False

    platform = sys.platform
    print("  💡 Azure CLI (az) n'est pas installé.")
    print("     Avec az, LevelUp peut créer l'application Azure automatiquement")
    print("     sans que vous ayez à visiter le portail Azure.")
    print()

    if platform == "win32":
        _print_az_install_option_windows()
    elif platform == "darwin":
        _print_az_install_option_macos()
    else:
        _print_az_install_option_linux()

    print()
    try:
        choice = input("  Installer Azure CLI maintenant ? [O/n] : ").strip().lower()
    except (EOFError, KeyboardInterrupt):
        return False

    if choice in ("n", "non", "no"):
        return False

    if _run_az_install(platform):
        return True

    # L'installation automatique a échoué → proposer les alternatives
    return _offer_az_manual_fallback()


def _print_az_install_option_windows() -> None:
    """Affiche l'option d'installation az sur Windows."""
    import shutil

    if shutil.which("winget"):
        print("  Installation via winget (recommandé) :")
        print("    winget install --id Microsoft.AzureCLI -e")
    else:
        print("  Téléchargement manuel : https://aka.ms/installazurecli")


def _print_az_install_option_macos() -> None:
    """Affiche l'option d'installation az sur macOS."""
    import shutil

    if shutil.which("brew"):
        print("  Installation via Homebrew :")
        print("    brew install azure-cli")
    else:
        print("  Installation via script officiel :")
        print("    curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash")
        print("  Documentation complète : https://aka.ms/installazurecli")


def _print_az_install_option_linux() -> None:
    """Affiche l'option d'installation az sur Linux."""
    print("  Installation via script officiel Microsoft :")
    print("    curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash")
    print("  Documentation complète : https://aka.ms/installazurecli")


def _run_az_install(platform: str) -> bool:
    """Lance l'installation d'Azure CLI pour la plateforme donnée.

    Retourne True si la commande d'installation a été lancée sans erreur fatale,
    False sinon.
    """
    import shutil

    print()
    if platform == "win32" and shutil.which("winget"):
        print("  → Installation d'Azure CLI via winget…")
        result = subprocess.run(
            [
                "winget",
                "install",
                "--id",
                "Microsoft.AzureCLI",
                "-e",
                "--accept-source-agreements",
                "--accept-package-agreements",
            ],
            timeout=300,
        )
        if result.returncode != 0:
            print("  ❌ Installation via winget échouée.")
            print("     Installez manuellement : https://aka.ms/installazurecli")
            return False
        print("  ✅ Azure CLI installé.")
        print("  ⚠  Redémarrez votre terminal puis relancez LevelUp pour continuer.")
        return True

    if platform == "darwin" and shutil.which("brew"):
        print("  → Installation d'Azure CLI via Homebrew…")
        result = subprocess.run(["brew", "install", "azure-cli"], timeout=300)
        if result.returncode != 0:
            print("  ❌ Installation via brew échouée.")
            print("     Installez manuellement : https://aka.ms/installazurecli")
            return False
        print("  ✅ Azure CLI installé.")
        return True

    if platform not in ("win32", "darwin") and shutil.which("curl"):
        print("  → Installation d'Azure CLI via le script officiel Microsoft…")
        print("     (sudo peut demander votre mot de passe)")
        result = subprocess.run(
            "curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash",
            shell=True,  # noqa: S602
            timeout=300,
        )
        if result.returncode != 0:
            print("  ❌ Installation échouée.")
            print("     Installez manuellement : https://aka.ms/installazurecli")
            return False
        print("  ✅ Azure CLI installé.")
        return True

    print("  ❌ Impossible d'installer automatiquement sur cette plateforme.")
    print("     Installez manuellement : https://aka.ms/installazurecli")
    return False


def _offer_az_manual_fallback() -> bool:
    """Propose des alternatives après un échec d'installation automatique d'AZ CLI.

    Retourne True si az est finalement disponible (install manuelle réussie),
    False pour basculer vers le flux « portail Azure + Device Code Flow ».

    Les deux méthodes aboutissent au même résultat (un client_id + un refresh
    token stockés dans .env.local) mais diffèrent par la façon d'obtenir le
    client_id :
    - az auto  : az crée l'app programmatiquement, pas de visite du portail
    - manuel   : l'utilisateur enregistre l'app sur portal.azure.com (2 min),
                 copie l'Application ID, et le Device Code Flow prend le relai
                 (pas de client_secret, pas de redirect URI — juste cocher
                 « Allow public client flows = Yes » dans Authentication)
    """
    import shutil

    print()
    print("  L'installation automatique n'a pas abouti. Que veux-tu faire ?")
    print()
    print("  1) Installer Azure CLI manuellement, puis réessayer")
    print("     → https://aka.ms/installazurecli")
    print()
    print("  2) Continuer sans Azure CLI  [Device Code Flow — recommandé]")
    print("     → Enregistrer l'app sur portal.azure.com (2 min, gratuit)")
    print("       puis coller le Client ID — pas de secret, pas de redirect URI")
    print("       Le wizard te guide étape par étape.")
    print()
    try:
        alt = input("  Ton choix (1/2) : ").strip()
    except (EOFError, KeyboardInterrupt):
        return False

    if alt == "1":
        print()
        print("  Installe Azure CLI depuis : https://aka.ms/installazurecli")
        with contextlib.suppress(Exception):
            webbrowser.open("https://aka.ms/installazurecli")
        try:
            input("  Appuie sur Entrée une fois l'installation terminée… ")
        except (EOFError, KeyboardInterrupt):
            return False
        if shutil.which("az") is not None:
            print("  ✅ Azure CLI détecté !")
            return True
        print("  ⚠  Azure CLI toujours introuvable.")
        print("     Si tu viens de l'installer, ferme et rouvre ton terminal,")
        print("     puis relance LevelUp pour continuer automatiquement.")
        print("     En attendant, on passe à la méthode manuelle.")

    # Option 2 ou az toujours absent → basculer vers portail + Client ID manuel
    return False


def _wizard_azure_creds(*, no_az: bool = False) -> bool:
    """Wizard interactif : configuration de l'Azure App Registration.

    Deux chemins possibles, tous deux débouchent sur le Device Code Flow :

    Chemin A — az CLI automatique (défaut si az est disponible) :
        az crée l'app programmatiquement → client_id récupéré sans visite du
        portail. Nécessite Azure CLI installé + connexion az login.

    Chemin B — portail Azure manuel (si az absent/refusé ou ``no_az=True``) :
        Ouvre portal.azure.com, l'utilisateur enregistre une app « public
        client » (2 étapes : créer l'app + cocher Allow public client flows),
        copie le Client ID et le colle ici. Pas de client_secret, pas de
        redirect URI requis.

    Dans les deux cas, ``_wizard_oauth_token()`` enchaîne avec le Device Code
    Flow (MSAL) pour obtenir le refresh_token Xbox Live.

    Args:
        no_az: Si True, saute la tentative az et va directement au chemin B.

    Retourne True si configuré avec succès, False sinon.
    """
    print()
    print("  ┌────────────────────────────────────────────────────────┐")
    print("  │     Configuration Azure App Registration               │")
    print("  └────────────────────────────────────────────────────────┘")
    print()

    # Chemin A : inscription automatique via az CLI (sauf si --no-az)
    client_id = None if no_az else _try_azure_auto_register()

    if not client_id:
        # Chemin B : portail Azure + saisie manuelle du Client ID (Device Code Flow)
        # L'app à créer est une « public client app » — configuration minimale :
        #   1) Enregistrer l'app (nom : LevelUp Halo, Personal Microsoft accounts)
        #   2) Authentication → Allow public client flows = Yes → Save
        #   3) Copier l'Application (client) ID
        # Pas de client_secret, pas de redirect URI.
        print("  Pour accéder à l'API Halo, il te faut une Azure App (public client) :")
        print("    1) https://portal.azure.com → App registrations → New registration")
        print("       Nom : LevelUp Halo | Account type : Personal Microsoft accounts")
        print("       Laisser Redirect URI vide → Register")
        print("    2) Copie l'« Application (client) ID »")
        print("    3) Authentication → Advanced settings → Allow public client flows = Yes → Save")
        print()
        print("  💡 Astuce : installez Azure CLI (https://aka.ms/installazurecli) pour")
        print("     que LevelUp crée l'application automatiquement sans visiter le portail.")
        print()
        with contextlib.suppress(Exception):
            webbrowser.open(
                "https://portal.azure.com/#view/Microsoft_AAD_RegisteredApps"
                "/CreateApplicationBlade"
            )
        try:
            client_id = input("  Client ID (Application ID) : ").strip()
            if not client_id:
                return False
        except (EOFError, KeyboardInterrupt):
            return False

    env_local = REPO_ROOT / ".env.local"
    try:
        _upsert_env_key(env_local, "SPNKR_AZURE_CLIENT_ID", client_id)
        os.environ["SPNKR_AZURE_CLIENT_ID"] = client_id
        print("  ✅ Client ID sauvegardé dans .env.local")
        return True
    except Exception as exc:
        print(f"  ❌ Sauvegarde impossible : {exc}")
        return False


def _wizard_oauth_token(gamertag: str, client_id: str) -> str | None:
    """Wizard interactif : obtention du refresh token via MSAL Device Code Flow.

    Affiche le code à entrer sur https://microsoft.com/devicelogin, attend la
    confirmation, sauvegarde le refresh token dans .env.local.
    Retourne le token ou None si échec.
    """
    print()
    print("  ┌────────────────────────────────────────────────────────┐")
    print("  │       Connexion Xbox Live — Device Code Flow           │")
    print("  └────────────────────────────────────────────────────────┘")
    print()

    try:
        from src.utils.msal_device_flow import (
            acquire_token_blocking,
            initiate_device_flow,
        )
    except ImportError:
        print("  ❌ Module msal_device_flow introuvable.")
        return None

    print("  Initialisation du Device Code Flow…")
    try:
        dc_result, app = initiate_device_flow(client_id)
    except Exception as exc:  # noqa: BLE001
        code = getattr(exc, "code", "unknown")
        detail = getattr(exc, "detail", str(exc))
        print(f"  ❌ Erreur : {code} — {detail}")
        return None

    print()
    print(f"  1) Visite : {dc_result.verification_url}")
    print(f"  2) Entre le code : {dc_result.user_code}")
    print(f"     (expire dans {dc_result.expires_in // 60} min)")
    print()
    with contextlib.suppress(Exception):
        webbrowser.open(dc_result.verification_url)
    print("  En attente de votre connexion Xbox… (ne fermez pas cette fenêtre)")
    print()

    try:
        token = acquire_token_blocking(app, dc_result._flow)
    except Exception as exc:  # noqa: BLE001
        code = getattr(exc, "code", "unknown")
        detail = getattr(exc, "detail", str(exc))
        print(f"  ❌ Échec : {code} — {detail}")
        return None

    gt_norm = gamertag.upper().replace(" ", "_").replace("-", "_")
    token_key = f"SPNKR_OAUTH_REFRESH_TOKEN_{gt_norm}"
    env_local = REPO_ROOT / ".env.local"
    try:
        _upsert_env_key(env_local, token_key, token)
        _upsert_env_key(env_local, "SPNKR_OAUTH_REFRESH_TOKEN", token)
        print(f"  ✅ Token sauvegardé dans .env.local ({token_key})")
    except Exception as exc:
        print(f"  ⚠  Sauvegarde impossible ({exc}) — ajoute manuellement dans .env.local :")
        print(f"     {token_key}={token}")
    # Toujours injecter en mémoire pour le sync qui suit (même si l'écriture a échoué)
    os.environ[token_key] = token
    os.environ.setdefault("SPNKR_OAUTH_REFRESH_TOKEN", token)
    return token


def _onboard_first_player(*, no_az: bool = False) -> int:  # noqa: PLR0912, PLR0915
    """Guide interactif pour configurer et synchroniser un premier joueur.

    Wizard zéro-CLI : détecte les credentials manquants et guide l'utilisateur
    pas à pas (Azure App, token OAuth, sync) sans jamais exiger de ligne de commande.

    Args:
        no_az: Si True, contourne la détection Azure CLI et va directement
               au chemin portail + Device Code Flow (--no-az).

    Returns:
        0 si la synchronisation a réussi, 2 sinon.
    """
    print()
    print("  ┌────────────────────────────────────────────────────────┐")
    print("  │       LevelUp — Configuration du premier joueur        │")
    print("  └────────────────────────────────────────────────────────┘")
    print()

    if not sys.stdin.isatty():
        print("  Terminal non interactif — impossible de procéder.")
        print("  Lance : python launcher.py add-player --gamertag <gamertag>")
        return 2

    try:
        gamertag = input("  Ton Gamertag Xbox (ex: JGtm) : ").strip()
    except (EOFError, KeyboardInterrupt):
        return 2

    if not gamertag:
        print("  Gamertag vide — annulé.")
        return 2

    _load_dotenv_for_launcher()

    # ── Étape 1 : Credentials Azure (client_id uniquement — public client) ───
    if not os.environ.get("SPNKR_AZURE_CLIENT_ID"):
        print()
        print("  ❌ Client ID Azure non configuré")
        ok = _wizard_azure_creds(no_az=no_az)
        if not ok:
            print("  → Configure le Client ID Azure et relance LevelUp.")
            return 2
    else:
        print("  ✓ Client ID Azure présent")

    client_id = os.environ.get("SPNKR_AZURE_CLIENT_ID", "")

    # ── Étape 2 : Token OAuth joueur ──────────────────────────────────────────
    gt_norm = gamertag.upper().replace(" ", "_").replace("-", "_")
    token_key = f"SPNKR_OAUTH_REFRESH_TOKEN_{gt_norm}"
    has_token = bool(os.environ.get(token_key) or os.environ.get("SPNKR_OAUTH_REFRESH_TOKEN"))
    if not has_token:
        print(f"  ❌ Token OAuth manquant ({token_key})")
        token = _wizard_oauth_token(gamertag, client_id)
        if not token:
            print("  → Obtiens le token et relance LevelUp.")
            return 2
    else:
        print("  ✓ Token OAuth présent")

    # ── Étape 3 : Initialisation des bases de données ─────────────────────────
    print()
    _ensure_warehouse_dbs()
    _run_migrations()

    # ── Étape 4 : Synchronisation ─────────────────────────────────────────────
    print()
    print(f"  → Synchronisation de « {gamertag} » (premier chargement, quelques minutes)…")
    print()

    try:
        before, after = _sync_player_duckdb(gamertag, delta=False, max_matches=200)
    except Exception as e:
        print(_classify_sync_error(str(e), gamertag))
        return 2

    new_matches = after - before
    if new_matches > 0:
        print(f"\n  ✅ {new_matches} match(s) synchronisé(s) pour {gamertag}")
    elif after > 0:
        print(f"\n  ✅ {after} match(s) déjà présents pour {gamertag}")
    else:
        print("\n  ⚠ Aucun match récupéré. Vérifie ton token ou ta connexion.")
        return 2

    return 0


def _cmd_add_player(args: argparse.Namespace) -> int:
    """Commande: ajoute/synchronise un joueur par son gamertag."""
    gamertag = getattr(args, "gamertag", None)
    no_az = getattr(args, "no_az", False)

    if not gamertag:
        return _onboard_first_player(no_az=no_az)

    print(f"  → Synchronisation de « {gamertag} »…")

    _load_dotenv_for_launcher()
    env_info = _env_check_for_player(gamertag)

    if not env_info["client_id"]:
        print("  ❌ Client ID Azure manquant (SPNKR_AZURE_CLIENT_ID)")
        return 2

    if not env_info["player_token"]:
        print(f"  ❌ Token manquant : {env_info['player_token_key']}")
        if sys.stdin.isatty():
            _print_token_setup_instructions(gamertag, str(env_info["player_token_key"]))
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
        print(f"  ✅ {new_matches} nouveau(x) match(s) pour {gamertag}")
    else:
        print(f"  ✅ À jour ({after} matchs) pour {gamertag}")
    return 0


def _cmd_reauth(args: argparse.Namespace) -> int:
    """Commande: renouvelle uniquement le token OAuth d'un joueur.

    Réutilise le client_id déjà configuré dans .env.local et relance
    uniquement le Device Code Flow MSAL pour obtenir un nouveau refresh_token.
    Utile quand le token a expiré ou a été révoqué sans avoir à recréer l'app.
    """
    gamertag = args.gamertag.strip()
    _load_dotenv_for_launcher()

    client_id = os.environ.get("SPNKR_AZURE_CLIENT_ID", "").strip()
    if not client_id:
        print("  ❌ Client ID Azure manquant (SPNKR_AZURE_CLIENT_ID)")
        print("     Lance d'abord : python launcher.py add-player")
        return 2

    print()
    print(f"  Renouvellement du token OAuth pour « {gamertag} »…")
    print(f"  Client ID : {client_id[:8]}…")
    print()

    token = _wizard_oauth_token(gamertag, client_id)
    if not token:
        print("  ❌ Échec du renouvellement.")
        return 2

    print(f"  ✅ Token renouvelé pour {gamertag}")
    return 0


def _wizard_save_client_id() -> bool:
    """Wizard minimal : demande uniquement le Client ID et le sauvegarde.

    Pour les utilisateurs qui ont déjà créé leur app Azure et connaissent
    leur Client ID.  Le Device Code Flow (MSAL) est ensuite déclenché
    normalement par ``_wizard_oauth_token()``.
    """
    print()
    print("  Colle ton Application (client) ID Azure :")
    print("  (Disponible dans portal.azure.com → App registrations → ton app → Overview)")
    print()
    try:
        client_id = input("  Client ID : ").strip()
    except (EOFError, KeyboardInterrupt):
        return False
    if not client_id:
        print("  Annulé.")
        return False
    env_local = REPO_ROOT / ".env.local"
    try:
        _upsert_env_key(env_local, "SPNKR_AZURE_CLIENT_ID", client_id)
        os.environ["SPNKR_AZURE_CLIENT_ID"] = client_id
        print("  ✅ Client ID sauvegardé.")
        return True
    except Exception as exc:
        print(f"  ❌ Sauvegarde impossible : {exc}")
        return False


# =============================================================================
# Détection d'état au démarrage + menu de récupération
# =============================================================================


@dataclass
class _ConfigState:
    """État global de la configuration détecté au démarrage."""

    players: list[PlayerInfo]
    has_client_id: bool
    players_missing_token: list[str]  # gamertags sans token OAuth valide

    @property
    def is_first_launch(self) -> bool:
        """Aucun joueur configuré — premier démarrage."""
        return not self.players

    @property
    def is_ready(self) -> bool:
        """Tout est en ordre : peut lancer Streamlit directement."""
        return bool(self.players) and self.has_client_id and not self.players_missing_token

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
    has_client_id = bool(os.environ.get("SPNKR_AZURE_CLIENT_ID"))
    players_missing_token: list[str] = []
    for p in players:
        gt_norm = p.gamertag.upper().replace(" ", "_").replace("-", "_")
        token_key = f"SPNKR_OAUTH_REFRESH_TOKEN_{gt_norm}"
        has_token = bool(os.environ.get(token_key) or os.environ.get("SPNKR_OAUTH_REFRESH_TOKEN"))
        if not has_token:
            players_missing_token.append(p.gamertag)
    return _ConfigState(
        players=players,
        has_client_id=has_client_id,
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
    print("  │         ⚠  Configuration incomplète détectée          │")
    print("  └────────────────────────────────────────────────────────┘")
    print()

    if not state.has_client_id:
        print("  ✗ Connexion Azure non configurée (identifiant application manquant)")
    if state.players_missing_token:
        missing = ", ".join(state.players_missing_token)
        print(f"  ✗ Accès Halo expiré ou manquant pour : {missing}")
    print()

    if not sys.stdin.isatty():
        print("  Terminal non interactif — impossible de procéder.")
        print("  → python launcher.py add-player   (reconfigurer)")
        print("  → python launcher.py reauth --gamertag <GT>  (renouveler token)")
        return 2

    # ── Construire les options selon l'état ───────────────────────────────────
    # Format : (action_key, label_affiché)
    options: list[tuple[str, str]] = []

    if not state.has_client_id:
        options.append(("config-az", "🔧 Azure CLI  (détection automatique + Device Code Flow)"))
        options.append(
            (
                "config-noaz",
                "🔑 MSAL Device Code Flow  (portail Azure minimal — recommandé, no secret)",
            )
        )
        options.append(
            (
                "paste-id",
                "📋 Coller un Client ID  (tu as déjà créé ton app Azure, on s'occupe du reste)",
            )
        )
    else:
        for gt in state.players_missing_token:
            options.append(
                (f"reauth:{gt}", f"🔑 MSAL Device Code Flow  (renouveler l'accès Halo pour {gt})")
            )
        options.append(("launch", "🚀 Lancer quand même  (tu synchroniseras plus tard)"))

    # Quitter toujours en dernier
    options.append(("quit", "Quitter"))

    for i, (_, label) in enumerate(options[:-1], 1):
        print(f"  {i}) {label}")
    print()
    print(f"  Q) {options[-1][1]}")
    print()

    keys_str = "/".join(str(i) for i in range(1, len(options))) + "/Q"
    try:
        choice = input(f"Ton choix ({keys_str}): ").strip().lower()
    except (EOFError, KeyboardInterrupt):
        return 2

    if choice in {"q", "quit", "exit"}:
        return 0

    try:
        idx = int(choice) - 1
    except ValueError:
        print("  Choix invalide.")
        return 2

    if not (0 <= idx < len(options) - 1):
        print("  Choix invalide.")
        return 2

    action, _ = options[idx]

    if action == "config-az":
        ok = _wizard_azure_creds(no_az=False)
        return _interactive() if ok else 2

    if action == "config-noaz":
        ok = _wizard_azure_creds(no_az=True)
        return _interactive() if ok else 2

    if action == "paste-id":
        # L'utilisateur a déjà son app Azure — on demande juste le client_id,
        # puis on relance _interactive() qui détectera le token manquant et
        # proposera le Device Code Flow (MSAL) automatiquement.
        ok = _wizard_save_client_id()
        return _interactive() if ok else 2

    if action.startswith("reauth:"):
        gt = action.split(":", 1)[1]
        client_id = os.environ.get("SPNKR_AZURE_CLIENT_ID", "").strip()
        print(f"\n  🔄 Renouvellement du token pour « {gt} »…")
        token = _wizard_oauth_token(gt, client_id)
        if not token:
            print("  ❌ Échec du renouvellement.")
            return 2
        return _interactive()

    if action == "launch":
        return _launch_streamlit(db_path=None, port=None, no_browser=False)

    print("  Choix invalide.")
    return 2


def _interactive() -> int:
    """Menu interactif principal.

    Branche selon l'état de la configuration :
    - Premier lancement (aucun joueur) → wizard de configuration
    - Config incomplète → menu de récupération contextuel
    - Tout OK → lancement direct de Streamlit
    """
    print("=" * 60)
    print("        LevelUp - Dashboard Halo Infinite")
    print("        Architecture DuckDB v5")
    print("=" * 60)

    state = _detect_config_state()

    # ── Premier lancement ──────────────────────────────────────────────────────
    if state.is_first_launch:
        print("\n📊 État actuel:")
        print("   ❌ Aucun joueur trouvé — premier démarrage")
        print("\n" + "-" * 60)
        print("Choisis une action:\n")
        print("  1) ➕ Ajouter un joueur               [premier démarrage]")
        print("     Configure et synchronise ton compte")
        print()
        print("  Q) Quitter")
        print()

        if not sys.stdin.isatty():
            print("⚠ Terminal non interactif → python launcher.py add-player --gamertag <gt>")
            return 2

        try:
            choice = input("Ton choix (1/Q): ").strip().lower()
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
                go = input("  Lancer le dashboard maintenant ? [O/n] : ").strip().lower()
            except (EOFError, KeyboardInterrupt):
                return 0
            if go not in ("n", "non", "no"):
                return _launch_streamlit(db_path=None, port=None, no_browser=False)
            return 0

        print("Choix invalide.")
        return 2

    # ── Afficher l'état des joueurs ────────────────────────────────────────────
    print("\n📊 État actuel:")
    total_matches = sum(p.total_matches for p in state.players)
    print(f"   Stockage: {_display_path(PLAYERS_DIR)}")
    print(f"   Joueurs: {len(state.players)}")
    for p in state.players:
        print(f"      - {p.gamertag}: {p.total_matches} matchs")
    print(f"   Total: {total_matches} matchs")
    if _metadata_db_exists():
        print("   Métadonnées: ✅")
    else:
        print("   Métadonnées: ⚠ Non trouvées")

    # ── Config incomplète → menu de récupération ───────────────────────────────
    if state.is_partial:
        return _recovery_menu(state)

    # ── Tout OK → lancement direct ─────────────────────────────────────────────
    print("\n  ✅ Configuration complète — lancement du dashboard…")
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
    p_add.add_argument(
        "--no-az",
        action="store_true",
        help="Saute la tentative az CLI et va directement au portail Azure + Device Code Flow",
    )
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
