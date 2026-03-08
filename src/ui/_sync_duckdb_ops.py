"""Opérations de synchronisation DuckDB.

Contient les fonctions de sync via DuckDBSyncEngine pour un joueur individuel,
en mode async et sync.
"""

from __future__ import annotations

import contextlib
import logging
import os
import time
from pathlib import Path

from src.ui._sync_utils import _shared_path, get_player_duckdb_path
from src.utils.paths import REPO_ROOT

logger = logging.getLogger(__name__)


def _sync_duckdb_player(  # noqa: C901, PLR0915
    *,
    db_path: str,
    gamertag: str,
    max_matches: int = 100,
    delta: bool = True,
    timeout_seconds: int = 120,
) -> tuple[bool, str]:
    """Synchronise un joueur DuckDB v4 via DuckDBSyncEngine.

    Args:
        db_path: Chemin vers le fichier stats.duckdb.
        gamertag: Gamertag du joueur.
        max_matches: Nombre max de matchs à synchroniser.
        delta: Mode delta (arrêt au premier match connu).
        timeout_seconds: Timeout en secondes.

    Returns:
        Tuple (succès, message).
    """
    import asyncio

    async def _sync_async() -> tuple[bool, str]:  # noqa: C901, PLR0912, PLR0915
        try:
            from src.data.sync.api_client import get_tokens_from_env
            from src.data.sync.engine import DuckDBSyncEngine
            from src.data.sync.models import SyncOptions
        except ImportError as e:
            return False, f"Module sync non disponible: {e}"

        db_file = Path(db_path)

        # Résoudre le XUID du joueur (obligatoire pour transformer les stats)
        from src.ui.cache_loaders import _resolve_player_xuid

        resolved_xuid = _resolve_player_xuid(str(db_file))

        # Récupérer les tokens
        try:
            tokens = await get_tokens_from_env()
        except SystemExit:
            return False, "Tokens SPNKr non configurés."
        except Exception as e:
            return False, f"Erreur tokens: {e}"

        if not tokens:
            return False, "Tokens SPNKr manquants."

        # Préparer l'ouverture exclusive de shared_matches.duckdb en R/W.
        _activate_sync_mode()

        # Créer le moteur de sync
        _sync_error: Exception | None = None
        _sync_result = None
        _engine_ref = None
        try:
            engine = DuckDBSyncEngine(
                player_db_path=db_file,
                xuid=resolved_xuid,
                gamertag=gamertag,
                tokens=tokens,
            )
            _engine_ref = engine

            options = SyncOptions(
                max_matches=max_matches,
                with_highlight_events=True,
                with_skill=True,
                with_aliases=True,
                with_participants=True,
            )

            if delta:
                _sync_result = await engine.sync_delta(options)
            else:
                _sync_result = await engine.sync_full(options)

        except Exception as e:
            _sync_error = e
        finally:
            # Fermer le moteur AVANT end_sync_mode() pour forcer le checkpoint WAL
            if _engine_ref is not None:
                try:
                    _engine_ref.close()
                    logger.debug("Sync: engine.close() → WAL shared_matches.duckdb checkpointé")
                except Exception:
                    pass
                _engine_ref = None
            _deactivate_sync_mode()

        if _sync_error is not None:
            return False, f"Erreur sync: {_sync_error}"

        sync_result = _sync_result
        if sync_result is not None and sync_result.errors:
            return False, f"Erreur: {'; '.join(sync_result.errors)}"

        # Forcer la mise à jour du mtime pour invalider les caches @st.cache_data
        for path in (str(db_file), str(_shared_path(db_file))):
            with contextlib.suppress(Exception):
                os.utime(path, None)

        if sync_result is not None:
            return True, sync_result.to_message()
        return True, "Sync terminé."

    try:
        from src.utils.sync_lock import SyncAlreadyRunning, SyncLock

        with SyncLock(timeout=0):
            return asyncio.run(asyncio.wait_for(_sync_async(), timeout=timeout_seconds))
    except SyncAlreadyRunning:
        return (
            False,
            "Un sync est déjà en cours (CLI ou autre onglet). Réessaie dans quelques instants.",
        )
    except asyncio.TimeoutError:
        return False, f"Timeout après {timeout_seconds}s."
    except Exception as e:
        return False, f"Erreur: {e}"


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
        with contextlib.suppress(Exception):
            os.utime(path, None)
            logger.debug("_touch_sync_mtime: mtime mis à jour → %s", path)


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
        with contextlib.suppress(Exception):
            engine.close()
            logger.debug(
                "_run_sync_engine: engine.close() gamertag=%s durée_totale=%.1fs",
                gamertag,
                time.monotonic() - t0,
            )


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
    # Forcer la récupération de toutes les données (arguments legacy ignorés)
    _ = with_highlight_events, with_skill, with_aliases
    if repo_root is None:
        repo_root = REPO_ROOT

    player_db_path = _resolve_sync_player_db_path(gamertag, repo_root)

    logger.info(
        "sync_player_duckdb_async: démarrage gamertag=%s mode=%s max_matches=%d manage_sync=%s",
        gamertag,
        "delta" if delta else "full",
        max_matches,
        _manage_sync_mode,
    )

    # Guard réentrance : ne pas ré-activer si déjà en mode sync (ex. sync_all_players)
    _already_active = False
    if _manage_sync_mode:
        from src.data.repositories.duckdb_repo import _sync_mode as _sm

        _already_active = _sm.is_set()
        if not _already_active:
            _activate_sync_mode()

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
        ok, msg = await _run_sync_engine(
            player_db_path=player_db_path,
            xuid=xuid,
            gamertag=gamertag,
            delta=delta,
            options=options,
        )

        if ok:
            logger.info("sync_player_duckdb_async: succès gamertag=%s — %s", gamertag, msg)
        else:
            logger.warning("sync_player_duckdb_async: échec gamertag=%s — %s", gamertag, msg)

        if _manage_sync_mode and not _already_active:
            _touch_sync_mtime(player_db_path)

        return ok, msg

    except Exception as e:
        logger.error(
            "sync_player_duckdb_async: exception gamertag=%s — %s",
            gamertag,
            e,
            exc_info=True,
        )
        return False, f"Erreur sync DuckDB: {e}"
    finally:
        if _manage_sync_mode and not _already_active:
            _deactivate_sync_mode()


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
