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
import io
import os
import sys
from dataclasses import dataclass

# Forcer l'encodage UTF-8 sur Windows pour les emojis
if sys.platform == "win32":
    sys.stdout = io.TextIOWrapper(sys.stdout.buffer, encoding="utf-8", errors="replace")
    sys.stderr = io.TextIOWrapper(sys.stderr.buffer, encoding="utf-8", errors="replace")

# Ajouter src au path pour les imports
sys.path.insert(0, str(__import__("pathlib").Path(__file__).resolve().parent))

# =============================================================================
# Imports depuis les sous-modules extraits (F8)
# =============================================================================

import subprocess  # noqa: F401 — re-export pour compatibilité tests (launcher_mod.subprocess)

from src.utils.launcher_env import (  # noqa: E402
    LANG as _LANG,
)
from src.utils.launcher_env import (
    _cmd_setup,
    _find_system_python,  # noqa: F401 — re-export tests
    _install_python_via_winget,  # noqa: F401 — re-export tests
    _maybe_reexec_into_venv,
)
from src.utils.launcher_i18n import t as _t  # noqa: E402
from src.utils.launcher_onboarding import (  # noqa: E402
    _BOOTSTRAP_AUTH_DB,  # noqa: F401 — re-export tests
    _cmd_add_player,
    _cmd_reauth,
    _onboard_first_player,
    _transfer_msal_cache,  # noqa: F401 — re-export tests
    _wizard_oauth_token,
)
from src.utils.launcher_players import (  # noqa: E402
    PlayerInfo,
    _cmd_doctor,
    _display_path,
    _env_check_for_player,  # noqa: F401 — re-export tests
    _list_players,
    _load_dotenv_for_launcher,
    _metadata_db_exists,
)
from src.utils.launcher_startup import (  # noqa: E402
    _flush_stdin,
    _install_signal_handler,
    _launch_react,
)
from src.utils.launcher_sync import (  # noqa: E402
    _cmd_info,
    _cmd_sync,
    _fetch_profile_assets,  # noqa: F401 — re-export tests
    _sync_player_duckdb,  # noqa: F401 — re-export tests
)
from src.utils.paths import PLAYERS_DIR, REPO_ROOT  # noqa: E402, F401 — REPO_ROOT re-export tests

# =============================================================================
# Détection d'état au démarrage + menu de récupération
# =============================================================================


@dataclass
class _ConfigState:
    """État global de la configuration détecté au démarrage."""

    players: list[PlayerInfo]
    players_missing_token: list[str]

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
    """Évalue l'état global de la configuration au démarrage."""
    players = _list_players()
    _load_dotenv_for_launcher()
    players_missing_token: list[str] = []
    for p in players:
        gt_norm = p.gamertag.upper().replace(" ", "_").replace("-", "_")
        token_key = f"SPNKR_OAUTH_REFRESH_TOKEN_{gt_norm}"
        has_token = bool(os.environ.get(token_key) or os.environ.get("SPNKR_OAUTH_REFRESH_TOKEN"))
        if not has_token:
            players_missing_token.append(p.gamertag)
    return _ConfigState(players=players, players_missing_token=players_missing_token)


def _recovery_menu(state: _ConfigState) -> int:  # noqa: PLR0912
    """Menu de récupération affiché quand la configuration est incomplète."""
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

    options: list[tuple[str, str]] = []

    for gt in state.players_missing_token:
        options.append(
            (
                f"reauth:{gt}",
                _t("recovery_option_reauth", _LANG, gt=gt),
            )
        )
    options.append(("launch", _t("recovery_option_launch", _LANG)))
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
        return _launch_react(db_path=None, port=None, no_browser=False)

    print(_t("recovery_invalid_choice", _LANG))
    return 2


def _interactive() -> int:
    """Menu interactif principal."""
    print("=" * 60)
    print(_t("interactive_title", _LANG))
    print(_t("interactive_arch", _LANG))
    print("=" * 60)

    state = _detect_config_state()

    # ── Premier lancement ──
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
                return _launch_react(db_path=None, port=None, no_browser=False)
            return 0

        print(_t("interactive_invalid_choice", _LANG))
        return 2

    # ── Afficher l'état des joueurs ──
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

    if state.is_partial:
        return _recovery_menu(state)

    print(_t("interactive_all_ok", _LANG))
    return _launch_react(db_path=None, port=None, no_browser=False)


# =============================================================================
# Commande: run
# =============================================================================


def _cmd_run(args: argparse.Namespace) -> int:
    """Commande: lance le dashboard."""
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
                        return _launch_react(
                            db_path=None, port=args.port, no_browser=args.no_browser
                        )
        else:
            print(_t("run_no_tty_hint", _LANG))
        return 2

    total_matches = sum(p.total_matches for p in players)
    print(_t("run_stats", _LANG, count=len(players), total=total_matches), flush=True)
    for p in players:
        print(_t("run_player_row", _LANG, gamertag=p.gamertag, matches=p.total_matches), flush=True)

    return _launch_react(db_path=None, port=args.port, no_browser=args.no_browser)


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
  python launcher.py                           # Mode interactif
  python launcher.py run                       # Dashboard React + API (http://localhost:5173)
  python launcher.py sync                      # Sync tous les joueurs
  python launcher.py sync --run                # Sync + dashboard React
  python launcher.py add-player                # Ajouter un joueur (guidé)
  python launcher.py add-player --gamertag JGtm  # Ajouter un joueur spécifique
  python launcher.py setup                     # Installer venv + dépendances
  python launcher.py setup --update            # Mettre à jour les dépendances
  python launcher.py doctor                    # Vérifier l'environnement
  python launcher.py info                      # Affiche les infos

Architecture v5:
  - Données joueurs: data/players/{gamertag}/stats.duckdb
  - Matchs partagés: data/warehouse/shared_matches_v2.duckdb
  - Métadonnées: data/warehouse/metadata.duckdb
""",
    )

    sub = ap.add_subparsers(dest="cmd")

    p_run = sub.add_parser("run", help="Lance le dashboard")
    p_run.add_argument("--port", type=int, default=8501, help="Port (défaut : 8501)")
    p_run.add_argument("--no-browser", action="store_true", help="Ne pas ouvrir le navigateur")
    p_run.set_defaults(func=_cmd_run)

    p_sync = sub.add_parser("sync", help="Synchronise les données de tous les joueurs")
    p_sync.add_argument("--run", action="store_true", help="Lance le dashboard après la sync")
    p_sync.add_argument("--full", action="store_true", help="Sync complète (pas de delta)")
    p_sync.add_argument(
        "--max-matches", type=int, default=100, help="Max matchs par joueur (défaut: 100)"
    )
    p_sync.set_defaults(func=_cmd_sync)

    p_info = sub.add_parser("info", help="Affiche les informations sur les données")
    p_info.set_defaults(func=_cmd_info)

    p_setup = sub.add_parser("setup", help="Configure l'environnement (venv + dépendances)")
    p_setup.add_argument(
        "--update", action="store_true", help="Met à jour les dépendances existantes"
    )
    p_setup.set_defaults(func=_cmd_setup)

    p_doctor = sub.add_parser("doctor", help="Vérifie la santé de l'environnement")
    p_doctor.set_defaults(func=_cmd_doctor)

    p_add = sub.add_parser("add-player", help="Ajoute/synchronise un joueur par son gamertag")
    p_add.add_argument("--gamertag", type=str, default=None, help="Gamertag Xbox du joueur")
    p_add.add_argument("--full", action="store_true", help="Sync complète (pas de delta)")
    p_add.add_argument("--max-matches", type=int, default=200, help="Max matchs (défaut: 200)")
    p_add.set_defaults(func=_cmd_add_player)

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

    try:
        from src.utils.log_config import setup_script_logging  # noqa: PLC0415

        setup_script_logging(level="WARNING")
    except Exception:
        pass

    try:
        from src.utils.log_config import suppress_asyncio_proactor_connection_reset  # noqa: PLC0415

        suppress_asyncio_proactor_connection_reset()
    except Exception:
        pass

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
        from src.utils.launcher_startup import _shutdown_event  # noqa: PLC0415

        if not _shutdown_event.is_set():
            print("\n⏹ Arrêt en cours...", flush=True)
        return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except KeyboardInterrupt:
        sys.exit(0)
