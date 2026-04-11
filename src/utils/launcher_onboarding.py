"""Onboarding, authentification Xbox et commandes add-player/reauth.

Contient : wizards OAuth (Device Code Flow), transfert de cache MSAL,
onboarding premier joueur, et les commandes ``add-player`` et ``reauth``.
Extrait de launcher.py (F8 post-v7 housekeeping).
"""

from __future__ import annotations

import asyncio
import contextlib
import logging
import webbrowser
from pathlib import Path

from src.utils.launcher_env import LANG
from src.utils.launcher_i18n import t as _t
from src.utils.launcher_migrations import _run_migrations
from src.utils.launcher_players import (
    _ensure_warehouse_dbs,
    _get_player_db_path,
    _load_dotenv_for_launcher,
)
from src.utils.launcher_sync import _classify_sync_error, _sync_player_duckdb
from src.utils.paths import WAREHOUSE_DIR

logger = logging.getLogger(__name__)

# DB temporaire pour stocker le cache MSAL avant que le gamertag soit connu
_BOOTSTRAP_AUTH_DB = WAREHOUSE_DIR / "bootstrap_auth.duckdb"

# =============================================================================
# Helpers MSAL / DB joueur
# =============================================================================


def _store_xuid_in_player_db(db_path: Path, xuid: str, gamertag: str) -> None:
    """Persiste xuid et gamertag dans sync_meta de la DB joueur."""
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
    """Transfère le cache MSAL depuis la DB source vers la DB target."""
    from src.auth._constants import MSAL_CACHE_DB_KEY  # noqa: PLC0415
    from src.utils.db import duckdb_read_only, duckdb_read_write  # noqa: PLC0415

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
    import subprocess  # noqa: PLC0415

    print(flush=True)
    print("  +----------------------------------------------------------+", flush=True)
    print(_t("dcf_box_title_line", LANG), flush=True)
    print("  +----------------------------------------------------------+", flush=True)
    print(f"  |  URL  : {verification_url:<49}|", flush=True)
    print(f"  |  Code : {user_code:<49}|", flush=True)
    _n_min = expires_in // 60
    _exp_label = _t("dcf_box_expires_label", LANG)
    print(f"  |  {_exp_label}: {_n_min} min{' ' * (44 - len(str(_n_min)))}|", flush=True)
    print("  +----------------------------------------------------------+", flush=True)
    print(flush=True)
    with contextlib.suppress(Exception):
        subprocess.run(["clip"], input=user_code.encode(), check=False)  # noqa: S603, S607
        print(_t("dcf_clipboard", LANG), flush=True)
    print(flush=True)


# =============================================================================
# Wizard OAuth interactif
# =============================================================================


def _wizard_oauth_token(gamertag: str, client_id: str = "") -> bool:  # noqa: ARG001
    """Wizard interactif : connexion Xbox via MSAL Device Code Flow.

    Returns:
        True si connexion réussie, False sinon.
    """
    print()
    print("  ┌────────────────────────────────────────────────────────┐")
    print(f"  │       {_t('wizard_title_text', LANG):<49}│")
    print("  └────────────────────────────────────────────────────────┘")
    print()

    try:
        from src.auth.provider import DeviceCodePending, start_device_flow  # noqa: PLC0415
    except ImportError:
        print(_t("wizard_module_missing", LANG))
        return False

    db_path = _get_player_db_path(gamertag)
    db_path.parent.mkdir(parents=True, exist_ok=True)

    print(_t("wizard_dcf_init", LANG))
    try:
        pending: DeviceCodePending = start_device_flow(db_path)
    except Exception as exc:  # noqa: BLE001
        code = getattr(exc, "code", "unknown")
        detail = getattr(exc, "detail", str(exc))
        print(_t("wizard_dcf_error", LANG, code=code, detail=detail))
        return False

    _print_device_code(pending.user_code, pending.verification_url, pending.expires_in)
    with contextlib.suppress(Exception):
        webbrowser.open(pending.verification_url)
    print(_t("wizard_dcf_reminder", LANG, code=pending.user_code))
    print(_t("wizard_dcf_waiting", LANG))
    print()

    async def _wait() -> tuple[str, str]:
        from src.auth.provider import complete_device_flow  # noqa: PLC0415

        return await complete_device_flow(db_path, pending)

    try:
        resolved_gamertag, xuid = asyncio.run(_wait())
        print(_t("wizard_dcf_connected", LANG, gamertag=resolved_gamertag, xuid=xuid))
        print(_t("wizard_dcf_token_saved", LANG))
        return True
    except Exception as exc:  # noqa: BLE001
        code = getattr(exc, "code", "unknown")
        detail = getattr(exc, "detail", str(exc))
        print(_t("wizard_dcf_failed", LANG, code=code, detail=detail))
        return False


# =============================================================================
# Onboarding premier joueur
# =============================================================================


def _onboard_first_player() -> int:  # noqa: C901, PLR0912, PLR0915
    """Guide interactif pour configurer et synchroniser un premier joueur.

    Wizard zéro-CLI : Device Code Flow d'abord, gamertag résolu depuis l'API
    Halo après authentification — aucune saisie manuelle requise.

    Returns:
        0 si la synchronisation a réussi, 2 sinon.
    """
    import sys  # noqa: PLC0415

    print()
    print("  ┌────────────────────────────────────────────────────────┐")
    print(f"  │       {_t('onboard_title_text', LANG):<49}│")
    print("  └────────────────────────────────────────────────────────┘")
    print()

    if not sys.stdin.isatty():
        print(_t("onboard_non_tty", LANG))
        print(_t("onboard_non_tty_hint", LANG))
        return 2

    _load_dotenv_for_launcher()

    # ── Étape 1 : Connexion Xbox — Device Code Flow ──
    try:
        from src.auth.provider import (  # noqa: PLC0415
            DeviceCodePending,
            complete_device_flow,
            start_device_flow,
        )
    except ImportError:
        print(_t("wizard_module_missing", LANG))
        return 2

    _BOOTSTRAP_AUTH_DB.parent.mkdir(parents=True, exist_ok=True)
    print(_t("wizard_dcf_init", LANG))
    try:
        pending: DeviceCodePending = start_device_flow(_BOOTSTRAP_AUTH_DB)
    except Exception as exc:  # noqa: BLE001
        print(
            _t(
                "wizard_dcf_error",
                LANG,
                code=getattr(exc, "code", "unknown"),
                detail=getattr(exc, "detail", exc),
            )
        )
        return 2

    _print_device_code(pending.user_code, pending.verification_url, pending.expires_in)
    with contextlib.suppress(Exception):
        webbrowser.open(pending.verification_url)
    print(_t("wizard_dcf_reminder", LANG, code=pending.user_code))
    print(_t("wizard_dcf_waiting", LANG))
    print()

    try:
        gamertag, xuid = asyncio.run(complete_device_flow(_BOOTSTRAP_AUTH_DB, pending))
        print(_t("wizard_dcf_connected", LANG, gamertag=gamertag, xuid=xuid))
        print()
    except Exception as exc:  # noqa: BLE001
        print(
            _t(
                "wizard_dcf_failed",
                LANG,
                code=getattr(exc, "code", "unknown"),
                detail=getattr(exc, "detail", exc),
            )
        )
        _BOOTSTRAP_AUTH_DB.unlink(missing_ok=True)
        return 2

    real_db_path = _get_player_db_path(gamertag)
    real_db_path.parent.mkdir(parents=True, exist_ok=True)
    try:
        _transfer_msal_cache(_BOOTSTRAP_AUTH_DB, real_db_path)
    except Exception as exc:
        print(_t("onboard_msal_transfer_fail", LANG, err=exc))
    finally:
        _BOOTSTRAP_AUTH_DB.unlink(missing_ok=True)

    if xuid:
        try:
            _store_xuid_in_player_db(real_db_path, xuid, gamertag)
        except Exception as exc:
            logger.debug("_store_xuid_in_player_db: %s", exc)

    # ── Étape 2 : Initialisation des bases de données ──
    print()
    _ensure_warehouse_dbs()
    _run_migrations()

    # ── Étape 3 : Synchronisation ──
    print()
    print(_t("onboard_sync_how", LANG))
    print()
    print(_t("onboard_sync_choice1", LANG))
    print(_t("onboard_sync_choice2", LANG))
    print()

    try:
        sync_choice = input(_t("onboard_sync_prompt", LANG)).strip() or "1"
    except (EOFError, KeyboardInterrupt):
        sync_choice = "1"

    if sync_choice == "2":
        print()
        print(_t("onboard_sync_full_starting", LANG, gamertag=gamertag))
        print()
        try:
            before, after = _sync_player_duckdb(gamertag, delta=False, max_matches=200)
        except Exception as e:
            print(_classify_sync_error(str(e), gamertag))
            return 2
        new_matches = after - before
        if new_matches > 0:
            print(_t("onboard_sync_ok", LANG, n=new_matches, gamertag=gamertag))
        elif after > 0:
            print(_t("onboard_sync_already", LANG, n=after, gamertag=gamertag))
        else:
            print(_t("onboard_sync_no_matches", LANG))
            return 2
        _run_migrations()
        return 0

    # ── Test 10 matchs (avec retry) ──
    new_matches = 0
    after = 0
    while True:
        print()
        print(_t("onboard_test_starting", LANG, gamertag=gamertag))
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
                print(_t("onboard_test_failed", LANG, err=test_error))
            else:
                print(_t("onboard_sync_no_matches", LANG))
            print()
            print(_t("onboard_test_what_now", LANG))
            print()
            print(_t("onboard_test_retry", LANG))
            print(_t("onboard_test_launch_anyway", LANG))
            print(_t("onboard_quit", LANG))
            print()
            try:
                fail_choice = input(_t("onboard_test_prompt", LANG)).strip().lower() or "2"
            except (EOFError, KeyboardInterrupt):
                fail_choice = "q"

            if fail_choice == "1":
                continue
            if fail_choice == "q":
                return 2
            return 0

        if new_matches > 0:
            print(_t("onboard_test_ok", LANG, n=new_matches))
        else:
            print(_t("onboard_test_existing", LANG, n=after))
        break

    _run_migrations()

    # ── Proposition de poursuivre ──
    print()
    print(_t("onboard_more_matches", LANG))
    print()
    print(_t("onboard_more_continue", LANG))
    print(_t("onboard_more_launch", LANG))
    print()

    try:
        continue_choice = input(_t("onboard_more_prompt", LANG)).strip() or "1"
    except (EOFError, KeyboardInterrupt):
        continue_choice = "2"

    if continue_choice != "1":
        return 0

    # ── Batches de 200 ──
    total_new = new_matches
    batch_num = 1
    while True:
        print()
        print(_t("onboard_batch_starting", LANG, n=batch_num))
        print()
        try:
            before_b, after_b = _sync_player_duckdb(gamertag, delta=False, max_matches=200)
        except Exception as e:
            print(_classify_sync_error(str(e), gamertag))
            break

        gained = after_b - before_b
        total_new += gained
        print(_t("onboard_batch_ok", LANG, gained=gained, total=after_b))

        if gained == 0:
            print(_t("onboard_batch_done", LANG))
            break

        print()
        print(_t("onboard_batch_continue", LANG))
        print()
        print(_t("onboard_batch_yes", LANG))
        print(_t("onboard_batch_no", LANG))
        print()
        try:
            again = input(_t("onboard_more_prompt", LANG)).strip() or "1"
        except (EOFError, KeyboardInterrupt):
            again = "2"
        if again != "1":
            break
        batch_num += 1

    print(_t("onboard_sync_total", LANG, n=total_new))
    return 0


# =============================================================================
# Commandes: add-player + reauth
# =============================================================================


def _cmd_add_player(args) -> int:  # noqa: ANN001
    """Commande: ajoute/synchronise un joueur par son gamertag."""
    import sys  # noqa: PLC0415

    gamertag = getattr(args, "gamertag", None)

    if not gamertag:
        return _onboard_first_player()

    print(_t("add_player_sync_starting", LANG, gamertag=gamertag))

    _load_dotenv_for_launcher()
    from src.utils.launcher_players import _env_check_for_player  # noqa: PLC0415

    env_info = _env_check_for_player(gamertag)

    if not env_info["player_token"]:
        print(_t("add_player_no_token", LANG, gamertag=gamertag))
        if sys.stdin.isatty():
            ok = _wizard_oauth_token(gamertag)
            if not ok:
                return 2
        else:
            print(_t("add_player_no_tty_hint", LANG, gamertag=gamertag))
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
        print(_t("add_player_new_matches", LANG, n=new_matches, gamertag=gamertag))
    else:
        print(_t("add_player_up_to_date", LANG, n=after, gamertag=gamertag))
    return 0


def _cmd_reauth(args) -> int:  # noqa: ANN001
    """Commande: renouvelle uniquement le token OAuth d'un joueur."""
    gamertag = args.gamertag.strip()
    _load_dotenv_for_launcher()

    print()
    print(_t("reauth_starting", LANG, gamertag=gamertag))
    print()

    ok = _wizard_oauth_token(gamertag)
    if not ok:
        print(_t("reauth_failed", LANG))
        return 2

    print(_t("reauth_ok", LANG, gamertag=gamertag))
    return 0
