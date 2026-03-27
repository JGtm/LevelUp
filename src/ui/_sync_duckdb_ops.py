"""Opérations de synchronisation DuckDB.

Contient les fonctions de sync via DuckDBSyncEngine pour un joueur individuel,
en mode async et sync.
"""

from __future__ import annotations

import logging
import os
import time
from pathlib import Path
from typing import TYPE_CHECKING, Any

if TYPE_CHECKING:
    from src.data.sync import SyncOptions

from src.ui._sync_utils import _shared_path, get_player_duckdb_path
from src.utils.paths import REPO_ROOT

logger = logging.getLogger(__name__)


def _finalize_sync_result(sync_result: Any, db_file: Path) -> tuple[bool, str]:
    """Vérifie le résultat de sync et invalide les caches mtime."""
    if sync_result is not None and sync_result.errors:
        return False, f"Erreur: {'; '.join(sync_result.errors)}"

    # Forcer la mise à jour du mtime pour invalider les caches @st.cache_data
    for path in (str(db_file), str(_shared_path(db_file))):
        try:
            os.utime(path, None)
        except Exception as e:
            logger.debug("event=utime_failed path=%s error=%s", path, e)

    if sync_result is not None:
        return True, sync_result.to_message()
    return True, "Sync terminé."


async def _run_engine_and_cleanup(
    engine: Any,
    *,
    options: Any,
    delta: bool,
) -> tuple[Any, Exception | None]:
    """Exécute engine.sync_delta/full puis ferme l'engine. Retourne (résultat, erreur)."""
    _sync_result = None
    _sync_error: Exception | None = None
    try:
        if delta:
            _sync_result = await engine.sync_delta(options)
        else:
            _sync_result = await engine.sync_full(options)
    except Exception as e:
        _sync_error = e
    finally:
        try:
            engine.close()
            logger.debug("Sync: engine.close() → WAL shared_matches.duckdb checkpointé")
        except Exception:
            pass
    return _sync_result, _sync_error


def _activate_sync_mode() -> None:
    """Active le mode sync : suspend l'ATTACH shared et libère les connexions."""
    try:
        from src.data.repositories.duckdb_repo import begin_sync_mode

        begin_sync_mode()
        logger.debug("Sync: mode sync activé (ATTACH shared suspendu)")
    except Exception:
        pass

    try:
        from src.ui.cache_loaders import get_cached_repository_st

        get_cached_repository_st.clear()
        logger.debug("Sync: cache @st.cache_resource get_cached_repository_st invalidé")
    except Exception:
        pass

    try:
        from src.data.repositories.duckdb_repo import release_all_db_connections

        n_closed = release_all_db_connections()
        if n_closed:
            logger.debug(
                "Sync: %d connexion(s) repo fermée(s) avant ouverture shared R/W", n_closed
            )
    except Exception:
        pass

    # Forcer le GC pour libérer les file handles DuckDB (Windows)
    import gc

    gc.collect()
    time.sleep(0.3)


def _deactivate_sync_mode() -> None:
    """Désactive le mode sync pour rétablir l'ATTACH shared."""
    try:
        from src.data.repositories.duckdb_repo import end_sync_mode

        end_sync_mode()
        logger.debug("Sync: mode sync désactivé (ATTACH shared rétabli)")
    except Exception:
        pass


def _resolve_sync_player_db_path(gamertag: str, repo_root: Path) -> Path:
    """Résout le chemin stats.duckdb d'un joueur (création lazy si nouveau)."""
    player_db_path = get_player_duckdb_path(gamertag, repo_root)
    if player_db_path is None:
        player_db_path = repo_root / "data" / "players" / gamertag / "stats.duckdb"
        logger.info("Nouveau joueur %s, création de la DB: %s", gamertag, player_db_path)
    return player_db_path


def _touch_sync_mtime(player_db_path: Path) -> None:
    """Met à jour le mtime des DBs pour invalider les caches Streamlit."""
    for path in (str(player_db_path), str(_shared_path(player_db_path))):
        try:
            os.utime(path, None)
            logger.debug("_touch_sync_mtime: mtime mis à jour → %s", path)
        except Exception as e:
            logger.debug("event=utime_failed path=%s error=%s", path, e)


async def _run_sync_engine(
    *,
    player_db_path: Path,
    xuid: str,
    gamertag: str,
    delta: bool,
    options,
) -> tuple[bool, str]:
    """Exécute le moteur de sync DuckDB pour un joueur."""
    from src.data.sync import DuckDBSyncEngine

    mode_str = "delta" if delta else "full"
    logger.debug(
        "_run_sync_engine: démarrage gamertag=%s xuid=%s mode=%s db=%s",
        gamertag,
        xuid,
        mode_str,
        player_db_path.name,
    )
    t0 = time.monotonic()
    engine = DuckDBSyncEngine(
        player_db_path=player_db_path,
        xuid=xuid,
        gamertag=gamertag,
    )
    try:
        if delta:
            result = await engine.sync_delta(options)
        else:
            result = await engine.sync_full(options)
        elapsed = time.monotonic() - t0
        logger.info(
            "_run_sync_engine: terminé gamertag=%s succès=%s durée=%.1fs — %s",
            gamertag,
            result.success,
            elapsed,
            result.to_message(),
        )
        return result.success, result.to_message()
    finally:
        try:
            engine.close()
            logger.debug(
                "_run_sync_engine: engine.close() gamertag=%s durée_totale=%.1fs",
                gamertag,
                time.monotonic() - t0,
            )
        except Exception as e:
            logger.debug("event=engine_close_failed gamertag=%s error=%s", gamertag, e)


async def sync_player_duckdb_async(  # noqa: PLR0913
    gamertag: str,
    xuid: str,
    *,
    delta: bool = True,
    match_type: str = "matchmaking",
    max_matches: int = 200,
    with_highlight_events: bool = True,
    with_skill: bool = True,
    with_aliases: bool = True,
    repo_root: Path | None = None,
    _manage_sync_mode: bool = True,
) -> tuple[bool, str]:
    """Synchronise un joueur via le nouveau pipeline DuckDB (async).

    IMPORTANT: Toutes les données sont toujours récupérées (highlights, skill, aliases, médailles).
    Les paramètres sont forcés à True.

    Args:
        gamertag: Gamertag du joueur.
        xuid: XUID du joueur.
        delta: Mode delta (True) ou full (False).
        match_type: Type de matchs.
        max_matches: Nombre max de matchs.
        with_highlight_events: Ignoré (toujours True).
        with_skill: Ignoré (toujours True).
        with_aliases: Ignoré (toujours True).
        repo_root: Racine du repo.
        _manage_sync_mode: Si True (défaut), gère sync_mode et mtime.
            Passer False quand le caller gère déjà (ex. sync_all_players_duckdb).

    Returns:
        Tuple (success, message).
    """
    _ = with_highlight_events, with_skill, with_aliases
    if repo_root is None:
        repo_root = REPO_ROOT

    player_db_path = _resolve_sync_player_db_path(gamertag, repo_root)
    logger.info(
        "sync_player_duckdb_async: démarrage gamertag=%s mode=%s max_matches=%d",
        gamertag,
        "delta" if delta else "full",
        max_matches,
    )

    _already_active = _maybe_activate_sync_mode(_manage_sync_mode)
    try:
        from src.data.sync import SyncOptions

        options = SyncOptions(
            match_type=match_type,
            max_matches=max_matches,
            with_highlight_events=True,
            with_skill=True,
            with_aliases=True,
            with_participants=True,
        )
        ok, msg = await _execute_sync(
            player_db_path=player_db_path,
            xuid=xuid,
            gamertag=gamertag,
            options=options,
            delta=delta,
        )
        if ok:
            logger.info("sync_player_duckdb_async: succès gamertag=%s — %s", gamertag, msg)
        else:
            logger.warning("sync_player_duckdb_async: échec gamertag=%s — %s", gamertag, msg)
        # Phase D : invalider les caches mtime uniquement si la sync a réussi (ok=True).
        # Pas de rafraîchissement si l'échec précède toute écriture.
        if ok and _manage_sync_mode and not _already_active:
            _touch_sync_mtime(player_db_path)
        return ok, msg
    except Exception as e:
        logger.error(
            "sync_player_duckdb_async: exception gamertag=%s — %s", gamertag, e, exc_info=True
        )
        return False, f"Erreur sync DuckDB: {e}"
    finally:
        if _manage_sync_mode and not _already_active:
            _deactivate_sync_mode()


def _maybe_activate_sync_mode(manage_sync_mode: bool) -> bool:
    """Active le mode sync si nécessaire. Retourne True si déjà actif."""
    if not manage_sync_mode:
        return False
    from src.data.repositories.duckdb_repo import _sync_mode as _sm

    already_active = _sm.is_set()
    if not already_active:
        _activate_sync_mode()
    return already_active


async def _execute_sync(
    *,
    player_db_path: Path,
    xuid: str,
    gamertag: str,
    options: SyncOptions,
    delta: bool,
) -> tuple[bool, str]:
    """Lance le moteur de sync et retourne (ok, message)."""
    return await _run_sync_engine(
        player_db_path=player_db_path,
        xuid=xuid,
        gamertag=gamertag,
        delta=delta,
        options=options,
    )


def sync_player_duckdb(  # noqa: PLR0913
    gamertag: str,
    xuid: str,
    *,
    delta: bool = True,
    match_type: str = "matchmaking",
    max_matches: int = 200,
    with_highlight_events: bool = True,
    with_skill: bool = True,
    with_aliases: bool = True,
    repo_root: Path | None = None,
    _manage_sync_mode: bool = True,
) -> tuple[bool, str]:
    """Synchronise un joueur via le nouveau pipeline DuckDB (sync wrapper).

    Wrapper synchrone autour de sync_player_duckdb_async().

    IMPORTANT: Toutes les données sont toujours récupérées (highlights, skill, aliases).

    Args:
        gamertag: Gamertag du joueur.
        xuid: XUID du joueur.
        delta: Mode delta (True) ou full (False).
        match_type: Type de matchs.
        max_matches: Nombre max de matchs.
        with_highlight_events: Ignoré (toujours True).
        with_skill: Ignoré (toujours True).
        with_aliases: Ignoré (toujours True).
        repo_root: Racine du repo.
        _manage_sync_mode: Si True (défaut), gère sync_mode et mtime.

    Returns:
        Tuple (success, message).
    """
    # Forcer la récupération de toutes les données
    with_highlight_events = True
    with_skill = True
    with_aliases = True
    import asyncio

    return asyncio.run(
        sync_player_duckdb_async(
            gamertag=gamertag,
            xuid=xuid,
            delta=delta,
            match_type=match_type,
            max_matches=max_matches,
            with_highlight_events=with_highlight_events,
            with_skill=with_skill,
            with_aliases=with_aliases,
            repo_root=repo_root,
            _manage_sync_mode=_manage_sync_mode,
        )
    )


async def _sync_all_players_loop_async(  # noqa: PLR0913
    *,
    profiles: dict,
    delta: bool,
    match_type: str,
    max_matches: int,
    with_highlight_events: bool,
    with_aliases: bool,
    repo_root: Path,
) -> list[tuple[str, bool, str]]:
    """Boucle async de sync pour tous les joueurs (event loop unique).

    Appelée via un seul asyncio.run() depuis _sync_all_players_loop.
    Utilise await sync_player_duckdb_async() pour éviter N event loops successives.
    """
    results: list[tuple[str, bool, str]] = []
    player_count = len(profiles)
    logger.info(
        "event=async_loop_start players=%d mode=%s",
        player_count,
        "delta" if delta else "full",
    )

    _activate_sync_mode()
    try:
        for gamertag, profile in profiles.items():
            xuid = profile.get("xuid", "")
            player_db_path = repo_root / profile.get("db_path", "")

            if not player_db_path.exists():
                logger.warning("[App Sync] %s: DB introuvable (%s)", gamertag, player_db_path)
                results.append((gamertag, False, f"DB introuvable: {player_db_path}"))
                continue

            ok, msg = await sync_player_duckdb_async(
                gamertag=gamertag,
                xuid=xuid,
                delta=delta,
                match_type=match_type,
                max_matches=max_matches,
                with_highlight_events=with_highlight_events,
                with_skill=True,
                with_aliases=with_aliases,
                repo_root=repo_root,
                _manage_sync_mode=False,
            )

            if ok:
                for path in (str(player_db_path), str(_shared_path(player_db_path))):
                    try:
                        import os

                        os.utime(path, None)
                    except Exception as e:
                        logger.debug("event=utime_failed path=%s error=%s", path, e)

            if ok:
                logger.info("[App Sync] %s: OK — %s", gamertag, msg)
            else:
                logger.error("[App Sync] %s: échec — %s", gamertag, msg)

            results.append((gamertag, ok, msg))
    finally:
        _deactivate_sync_mode()

    ok_count = sum(1 for _, ok, _ in results if ok)
    logger.info(
        "event=async_loop_end players=%d ok=%d failed=%d",
        player_count,
        ok_count,
        player_count - ok_count,
    )
    return results
