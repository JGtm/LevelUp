"""Synchronisation des données et commandes CLI sync/info pour le lanceur LevelUp.

Contient : classification d'erreur sync, sync asynchrone/synchrone d'un joueur,
fetch des assets profil, et les commandes ``sync`` et ``info``.
Extrait de launcher.py (F8 post-v7 housekeeping).
"""

from __future__ import annotations

import asyncio

from src.utils.launcher_env import LANG
from src.utils.launcher_i18n import t as _t
from src.utils.launcher_players import (
    _count_matches_duckdb,
    _display_path,
    _get_player_db_path,
    _list_players,
)
from src.utils.launcher_startup import _check_shutdown, _launch_streamlit
from src.utils.paths import METADATA_DB_FILENAME, PLAYERS_DIR, WAREHOUSE_DIR


def _classify_sync_error(err: str, gamertag: str) -> str:
    """Retourne un message d'erreur de sync actionnable selon le type d'échec."""
    s = err.lower()
    if "invalid_grant" in s or "aadsts" in s or ("expir" in s and ("token" in s or "refresh" in s)):
        return _t("sync_error_token", LANG, gamertag=gamertag)
    if any(
        kw in s
        for kw in (
            "could not set lock",
            "could not open lock",
            "locked by",
            "being used by another process",
        )
    ):
        return _t("sync_error_locked", LANG)
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
        return _t("sync_error_network", LANG)
    return _t("sync_error_unknown", LANG, gamertag=gamertag, err=err)


async def _sync_player_duckdb_async(
    gamertag: str, *, delta: bool = True, max_matches: int = 100
) -> tuple[int, int]:
    """Synchronise un joueur via DuckDBSyncEngine (async).

    Returns:
        Tuple (matchs_avant, matchs_après)
    """
    try:
        from src.data.sync.engine import DuckDBSyncEngine  # noqa: PLC0415
        from src.data.sync.models import SyncOptions  # noqa: PLC0415
    except ImportError as e:
        print(f"  ⚠ Import error: {e}")
        return (0, 0)

    db_path = _get_player_db_path(gamertag)
    db_path.parent.mkdir(parents=True, exist_ok=True)
    matches_before = _count_matches_duckdb(db_path)

    try:
        from src.auth.provider import get_halo_tokens  # noqa: PLC0415

        tokens = await get_halo_tokens(db_path)
    except Exception as e:
        print(_classify_sync_error(str(e), gamertag))
        return (matches_before, matches_before)

    try:
        engine = DuckDBSyncEngine(
            player_db_path=db_path,
            xuid="",
            gamertag=gamertag,
            tokens=tokens,
        )
        options = SyncOptions(
            max_matches=max_matches,
            with_skill=True,
            with_aliases=True,
        )
        if delta:
            result = await engine.sync_delta()
        else:
            result = await engine.sync_full(options)

        if result.errors:
            print(_classify_sync_error(result.errors[0], gamertag))
    except Exception as e:
        print(_classify_sync_error(str(e), gamertag))
        return (matches_before, matches_before)

    matches_after = matches_before + result.matches_inserted
    return (matches_before, matches_after)


def _sync_player_duckdb(
    gamertag: str, *, delta: bool = True, max_matches: int = 100
) -> tuple[int, int]:
    """Synchronise un joueur via DuckDBSyncEngine (wrapper sync)."""
    return asyncio.run(_sync_player_duckdb_async(gamertag, delta=delta, max_matches=max_matches))


def _fetch_profile_assets(gamertag: str) -> None:
    """Récupère les assets profil du joueur."""
    try:
        from src.ui.profile_api import (  # noqa: PLC0415
            fetch_appearance_via_spnkr,
            fetch_xuid_via_spnkr,
            save_cached_appearance,
            save_cached_xuid,
        )
    except ImportError:
        return

    print(_t("fetch_profile_assets", LANG))

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
# Commandes: sync + info
# =============================================================================


def _cmd_sync(args) -> int:  # noqa: ANN001
    """Commande: sync tous les joueurs (architecture v5 DuckDB)."""
    players = _list_players()

    if not players:
        print(_t("sync_no_players", LANG))
        print(_t("sync_no_players_hint1", LANG))
        print("   python launcher.py add-player")
        print(_t("sync_no_players_hint3", LANG))
        print("   python scripts/sync.py --delta --gamertag <gamertag>")
        return 2

    print("=" * 60)
    print(_t("sync_title", LANG))
    print("=" * 60)
    print(_t("sync_players_detected", LANG, count=len(players)))
    for p in players:
        print(_t("sync_player_row", LANG, gamertag=p.gamertag, matches=p.total_matches))

    print(_t("sync_in_progress", LANG))

    delta_mode = not getattr(args, "full", False)
    max_matches = int(getattr(args, "max_matches", 100))

    total_new = 0
    failures = 0

    for player in players:
        if _check_shutdown():
            return 0

        print(f"\n[{player.gamertag}]")
        print(_t("sync_mode_delta" if delta_mode else "sync_mode_full", LANG))

        try:
            before, after = _sync_player_duckdb(
                player.gamertag,
                delta=delta_mode,
                max_matches=max_matches,
            )
            new_matches = after - before
            total_new += new_matches
            if new_matches > 0:
                print(_t("sync_new_matches", LANG, n=new_matches))
            else:
                print(_t("sync_up_to_date", LANG, n=after))
            _fetch_profile_assets(player.gamertag)
        except Exception as e:
            print(_classify_sync_error(str(e), player.gamertag))
            failures += 1

    if _check_shutdown():
        return 0

    print("\n" + "=" * 60)
    print(_t("sync_done", LANG))
    print("=" * 60)

    players_after = _list_players()
    total_matches = sum(p.total_matches for p in players_after)
    print(_t("sync_summary_players", LANG, n=len(players_after)))
    print(_t("sync_summary_total", LANG, n=total_matches))
    if total_new > 0:
        print(_t("sync_summary_new", LANG, n=total_new))
    if failures > 0:
        print(_t("sync_summary_failures", LANG, n=failures))

    if getattr(args, "run", False):
        return _launch_streamlit(db_path=None, port=None, no_browser=False)

    return 0


def _cmd_info(args) -> int:  # noqa: ANN001, ARG001
    """Commande: affiche les infos sur les données."""
    players = _list_players()

    if not players:
        print(_t("sync_no_players", LANG))
        return 2

    print("=" * 60)
    print(_t("info_title", LANG))
    print("=" * 60)

    total_matches = sum(p.total_matches for p in players)
    print(_t("info_dir", LANG, path=_display_path(PLAYERS_DIR)))
    print(_t("info_players", LANG, n=len(players)))
    print(_t("info_total_matches", LANG, n=total_matches))

    print(_t("info_players_detail", LANG))
    for p in players:
        size_mb = p.db_path.stat().st_size / (1024 * 1024) if p.db_path.exists() else 0
        print(
            _t("info_player_row", LANG, gamertag=p.gamertag, matches=p.total_matches, size=size_mb)
        )

    metadata_path = WAREHOUSE_DIR / METADATA_DB_FILENAME
    if metadata_path.exists():
        size_mb = metadata_path.stat().st_size / (1024 * 1024)
        print(_t("info_metadata", LANG, path=_display_path(metadata_path), size=size_mb))
    else:
        print(_t("info_no_metadata", LANG, path=_display_path(metadata_path)))

    return 0
