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
            pass

        players.append(
            PlayerInfo(
                gamertag=gamertag,
                db_path=db_path,
                total_matches=total_matches,
                xuid=xuid,
            )
        )

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
        print(f"  ⚠ Erreur tokens: {e}")
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
            print(f"  ⚠ Erreur sync: {result.error}")

    except Exception as e:
        print(f"  ⚠ Erreur sync: {e}")
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

    _require_module(
        "streamlit", install_hint="./.venv/Scripts/python -m pip install -r requirements.txt"
    )

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
        print("\n   Tu dois d'abord synchroniser les données:")
        print("   python launcher.py sync")
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
        print("\n   Pour ajouter un nouveau joueur, utilise:")
        print("   python scripts/sync_player.py --gamertag <gamertag>")
        print("\n   Ou crée manuellement le dossier:")
        print("   mkdir data/players/<gamertag>")
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
            print(f"  ⚠ Erreur: {e}")
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
        return _launch_streamlit(db_path=None, port=8501, no_browser=False)

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
# Mode interactif
# =============================================================================


def _interactive() -> int:  # noqa: C901, PLR0912, PLR0915
    """Menu interactif simplifié."""
    print("=" * 60)
    print("        LevelUp - Dashboard Halo Infinite")
    print("        Architecture DuckDB v5")
    print("=" * 60)

    # Lister les joueurs DuckDB
    players = _list_players()

    # Afficher l'état actuel
    print("\n📊 État actuel:")

    if players:
        total_matches = sum(p.total_matches for p in players)
        print(f"   Stockage: {_display_path(PLAYERS_DIR)}")
        print(f"   Joueurs: {len(players)}")
        for p in players:
            print(f"      - {p.gamertag}: {p.total_matches} matchs")
        print(f"   Total: {total_matches} matchs")

        # Vérifier metadata
        if _metadata_db_exists():
            print("   Métadonnées: ✅")
        else:
            print("   Métadonnées: ⚠ Non trouvées")
    else:
        print("   ❌ Aucun joueur trouvé")
        print("   → Tu dois d'abord synchroniser des données")

    print("\n" + "-" * 60)
    print("Choisis une action:\n")
    print("  1) 🚀 Dashboard                       [recommandé]")
    print("     Lance le dashboard directement")
    print()
    print("  2) 🔄 Sync + Dashboard")
    print("     Synchronise les nouveaux matchs puis lance le dashboard")
    print()
    print("  3) 🔄 Sync seul")
    print("     Synchronise les données sans lancer le dashboard")
    print()
    print("  4) 📊 Infos")
    print("     Affiche les informations détaillées")
    print()
    print("  Q) Quitter")
    print()

    # Si stdin n'est pas un terminal (IDE, Cursor, pipe), ne pas bloquer sur input()
    if not sys.stdin.isatty():
        print("⚠ Terminal non interactif détecté → lancement direct du dashboard (option 1).")
        print("  Pour d'autres actions, utilise la CLI: run / sync / info")
        choice = "1" if players else ""
        if not choice:
            print("  Aucun joueur trouvé. Lance: python launcher.py sync")
            return 2
    else:
        try:
            choice = input("Ton choix (1/2/3/4/Q): ").strip().lower()
        except EOFError:
            print("\n⚠ stdin fermé ou non connecté (terminal/IDE).")
            print("  Utilise les arguments CLI:")
            print("    python launcher.py run    # option 1")
            print("    python launcher.py sync   # option 3")
            print("    python launcher.py info   # option 4")
            return 2

    if choice in {"q", "quit", "exit"}:
        return 0

    if choice == "1":
        if not players:
            print("\n⚠ Aucune donnée joueur trouvée.")
            print("  Lance d'abord une synchronisation (choix 2 ou 3)")
            return 2
        return _launch_streamlit(db_path=None, port=8501, no_browser=False)

    if choice == "2":
        if not players:
            print("\n⚠ Aucun joueur configuré.")
            print("  Crée d'abord un dossier dans data/players/<gamertag>/")
            return 2
        args = argparse.Namespace(max_matches=100, full=False, run=True)
        return _cmd_sync(args)

    if choice == "3":
        if not players:
            print("\n⚠ Aucun joueur configuré.")
            print("  Crée d'abord un dossier dans data/players/<gamertag>/")
            return 2
        args = argparse.Namespace(max_matches=100, full=False, run=False)
        return _cmd_sync(args)

    if choice == "4":
        return _cmd_info(argparse.Namespace())

    print("Choix invalide.")
    return 2


# =============================================================================
# Parser CLI
# =============================================================================


def _build_parser() -> argparse.ArgumentParser:
    """Construit le parser CLI."""
    ap = argparse.ArgumentParser(
        prog="levelup",
        description="LevelUp - Dashboard Halo Infinite (Architecture DuckDB v5)",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Exemples:
  python launcher.py           # Mode interactif
  python launcher.py run       # Dashboard seul
  python launcher.py sync      # Sync tous les joueurs
  python launcher.py sync --run  # Sync + dashboard
  python launcher.py info      # Affiche les infos

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
