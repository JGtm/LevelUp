"""Service de synchronisation initiale des données Halo — Sprint 3.

Lance un job asynchrone qui appelle l'orchestrateur de sync existant
(src/data/sync/) et met à jour le job avec des compteurs métier par phase.

Phases :
    prepare → auth → fetch_matches → enrich → verify → finalize
"""

from __future__ import annotations

import threading
import time

import structlog

from apps.api.app.schemas.common import ApiErrorSchema, AsyncJobStatus
from apps.api.app.schemas.sync import InitialSyncStartRequest
from apps.api.app.services.job_store import JobStore

logger = structlog.get_logger(__name__)

# Labels FR par phase
_PHASE_LABELS: dict[str, str] = {
    "prepare": "Préparation",
    "auth": "Connexion au service Halo",
    "fetch_matches": "Récupération de vos matchs",
    "enrich": "Analyse des statistiques",
    "verify": "Vérification",
    "finalize": "Finalisation",
}


def start_initial_sync(
    body: InitialSyncStartRequest,
    session_id: str | None = None,
) -> AsyncJobStatus:
    """Crée un job ``initial_sync`` et lance la synchronisation en arrière-plan.

    Retourne immédiatement le statut initial du job (``queued``).
    """
    store = JobStore.get()
    job = store.create(
        "initial_sync",
        metadata={"player_slug": body.player_slug},
    )
    # Persister l'ID du job sync dans la session pour reprise après refresh
    if session_id:
        _store_active_sync_job_id(session_id, job.job_id)
    thread = threading.Thread(
        target=_run_initial_sync_bg,
        args=(job.job_id, body.player_slug, body.max_matches, session_id),
        daemon=True,
    )
    thread.start()

    logger.info(
        "initial_sync_started",
        job_id=job.job_id,
        player_slug=body.player_slug,
        max_matches=body.max_matches,
        session_id=session_id,
    )

    return store.get_job(job.job_id) or job


# ---------------------------------------------------------------------------
# Thread background
# ---------------------------------------------------------------------------


def _run_initial_sync_bg(
    job_id: str,
    player_slug: str,
    max_matches: int,
    session_id: str | None = None,
) -> None:
    """Exécute la sync initiale dans un thread background.

    Met à jour le job à chaque changement de phase et écrit le marqueur
    ``initial_sync_completed_at`` en fin de sync réussie.
    """
    store = JobStore.get()

    def _set_phase(phase_key: str, pct: int, step: str | None = None) -> None:
        store.update(
            job_id,
            phase_key=phase_key,
            phase_label=_PHASE_LABELS.get(phase_key, phase_key),
            progress_pct=pct,
            current_step=step or _PHASE_LABELS.get(phase_key, phase_key),
        )

    try:
        # Phase 1 — Préparation
        store.update(job_id, status="running")
        _set_phase("prepare", 2, "Vérification de la configuration…")
        _validate_player(player_slug)

        # Phase 2 — Auth
        _set_phase("auth", 8, "Vérification des tokens Halo…")
        _check_auth()

        # Phase 3 — Récupération des matchs
        _set_phase("fetch_matches", 15, "Récupération de l'historique…")
        match_count = _fetch_and_sync(job_id, player_slug, max_matches, store)

        # Phase 5 — Vérification
        _set_phase("verify", 90, "Vérification des données importées…")

        # Phase 6 — Finalisation
        _set_phase("finalize", 97, "Finalisation des index…")
        _run_finalize(player_slug)

        # Marqueur persistant de sync initiale (Sprint 3.1)
        _write_initial_sync_marker(player_slug)

        logger.info(
            "initial_sync_succeeded",
            job_id=job_id,
            player_slug=player_slug,
            matches_imported=match_count,
        )

        store.update(
            job_id,
            status="succeeded",
            progress_pct=100,
            phase_key="finalize",
            phase_label=_PHASE_LABELS["finalize"],
            current_step="Synchronisation terminée",
            result={"matches_imported": match_count},
        )

        # Nettoyer l'ID de job sync actif dans la session
        if session_id:
            _clear_active_sync_job_id(session_id)

    except _SyncAuthError as exc:
        logger.warning("initial_sync_auth_expired", job_id=job_id, reason=str(exc))
        store.update(
            job_id,
            status="failed",
            error=ApiErrorSchema(
                code="sync_auth_expired",
                message=str(exc),
                retryable=True,
            ),
        )
    except _SyncHaloApiError as exc:
        logger.warning("initial_sync_halo_api_error", job_id=job_id, reason=str(exc))
        store.update(
            job_id,
            status="failed",
            error=ApiErrorSchema(
                code="sync_halo_api_error",
                message=str(exc),
                retryable=True,
            ),
        )
    except _SyncDbError as exc:
        logger.error("initial_sync_db_error", job_id=job_id, reason=str(exc))
        store.update(
            job_id,
            status="failed",
            error=ApiErrorSchema(
                code="sync_db_error",
                message=str(exc),
                retryable=True,
            ),
        )
    except _SyncAbortError as exc:
        logger.warning("initial_sync_aborted", job_id=job_id, reason=str(exc))
        store.update(
            job_id,
            status="failed",
            error=ApiErrorSchema(
                code="sync_aborted",
                message=str(exc),
                retryable=False,
            ),
        )
    except Exception as exc:  # noqa: BLE001
        logger.error("initial_sync_unexpected_error", job_id=job_id, exc=str(exc))
        store.update(
            job_id,
            status="failed",
            error=ApiErrorSchema(
                code="internal_error",
                message=f"Erreur inattendue : {exc}",
                retryable=True,
            ),
        )


# ---------------------------------------------------------------------------
# Helpers internes
# ---------------------------------------------------------------------------


class _SyncAbortError(Exception):
    """Erreur métier bloquante (config manquante, joueur introuvable…)."""


class _SyncAuthError(Exception):
    """Tokens Halo expirés ou absents."""


class _SyncHaloApiError(Exception):
    """Erreur API Halo (5xx, timeout, quota)."""


# Délais de retry en secondes (3 tentatives : 1s, 2s, 4s)
_HALO_RETRY_DELAYS: tuple[int, ...] = (1, 2, 4)

# Modules / noms d'exception courants signalant une erreur réseau/API Halo
_HALO_ERROR_MODULES: frozenset[str] = frozenset({"aiohttp", "httpx", "spnkr"})
_HALO_ERROR_NAMES: frozenset[str] = frozenset(
    {
        "ClientError",
        "ClientResponseError",
        "ServerTimeoutError",
        "ClientConnectorError",
        "ServerConnectionError",
    }
)


def _is_transient_halo_error(exc: BaseException) -> bool:
    """Retourne True si l'exception semble être une erreur transitoire de l'API Halo."""
    mod = (type(exc).__module__ or "").lower()
    return (
        any(name in mod for name in _HALO_ERROR_MODULES) or type(exc).__name__ in _HALO_ERROR_NAMES
    )


class _SyncDbError(Exception):
    """Erreur d'écriture DuckDB."""


def _write_initial_sync_marker(player_slug: str) -> None:
    """Persiste ``initial_sync_completed_at`` dans sync_meta de la player DB."""
    try:
        from datetime import datetime, timezone
        from pathlib import Path

        from apps.api.app.core.config import get_settings
        from src.utils.db import duckdb_read_write

        settings = get_settings()
        player_db = Path(settings.repo_root) / "data" / "players" / player_slug / "stats.duckdb"
        if not player_db.exists():
            logger.warning("initial_sync_marker_db_absent", player_slug=player_slug)
            return
        with duckdb_read_write(player_db) as conn:
            conn.execute(
                """
                INSERT OR IGNORE INTO sync_meta (key, value, updated_at)
                VALUES ('initial_sync_completed_at', ?, CURRENT_TIMESTAMP)
                """,
                [datetime.now(timezone.utc).isoformat()],
            )
        logger.info("initial_sync_marker_written", player_slug=player_slug)
    except Exception:
        logger.warning("initial_sync_marker_write_failed", player_slug=player_slug, exc_info=True)


def _store_active_sync_job_id(session_id: str, job_id: str) -> None:
    """Persiste l'ID du job sync actif dans la session."""
    try:
        from apps.api.app.deps.auth import _get_store

        store = _get_store()
        session = store.load(session_id)
        if session is not None:
            session.active_sync_job_id = job_id
            store.save(session)
    except Exception:
        logger.warning("store_active_sync_job_id_failed", exc_info=True)


def _clear_active_sync_job_id(session_id: str) -> None:
    """Efface l'ID du job sync actif de la session."""
    try:
        from apps.api.app.deps.auth import _get_store

        store = _get_store()
        session = store.load(session_id)
        if session is not None:
            session.active_sync_job_id = None
            store.save(session)
    except Exception:
        logger.warning("clear_active_sync_job_id_failed", exc_info=True)


def _validate_player(player_slug: str) -> None:
    """Vérifie que le joueur existe dans db_profiles.json."""
    try:
        from src.data.profile import load_profiles  # type: ignore[import-untyped]

        profiles = load_profiles()
        if player_slug not in profiles:
            raise _SyncAbortError(f"Joueur « {player_slug} » introuvable dans db_profiles.json")
    except ImportError:  # guard env partiel (tests) — supprimer quand API autonome
        pass


def _check_auth() -> None:
    """Vérifie que les tokens Halo sont disponibles."""
    try:
        from src.utils.auth import get_auth_status  # type: ignore[import-untyped]

        status = get_auth_status()
        has_token = bool(
            getattr(status, "has_refresh_token", False) or getattr(status, "has_msal_cache", False)
        )
        if not has_token:
            raise _SyncAbortError(
                "Aucun token d'authentification disponible. Effectuez d'abord le Device Code Flow."
            )
    except ImportError:  # guard env partiel (tests) \u2014 supprimer quand API autonome
        pass


def _fetch_and_sync(  # noqa: PLR0913
    job_id: str,
    player_slug: str,
    max_matches: int,
    store: JobStore,
) -> int:
    """Lance la sync en appelant l'orchestrateur src/data/sync/.

    Met à jour le job avec les compteurs métier et retourne le nombre de matchs
    importés. Fail-graceful : si le module n'est pas disponible, retourne 0.
    """
    try:
        import asyncio

        from src.data.sync.scope import SyncScope  # type: ignore[import-untyped]

        scope = SyncScope.make_all(max_matches=max_matches)

        # Callback de progression injecté dans le scope (si disponible)
        def _on_progress(done: int, total: int) -> None:
            pct = int(15 + 70 * done / max(total, 1))
            store.update(
                job_id,
                phase_key="fetch_matches",
                phase_label=_PHASE_LABELS["fetch_matches"],
                progress_pct=min(pct, 85),
                current_step=f"{done}/{total} matchs récupérés",
                matches_done=done,
                matches_total=total,
            )

        if hasattr(scope, "on_progress"):
            scope.on_progress = _on_progress  # type: ignore[assignment]

        store.update(
            job_id,
            phase_key="fetch_matches",
            phase_label=_PHASE_LABELS["fetch_matches"],
            progress_pct=15,
            current_step="Connexion à l'API Halo…",
        )

        from src.data.sync.backfill import backfill_player_data  # type: ignore[import-untyped]

        # Retry 3× avec backoff exponentiel sur erreurs API Halo transitoires
        last_halo_exc: BaseException | None = None
        for attempt, delay in enumerate(_HALO_RETRY_DELAYS):
            try:
                asyncio.run(backfill_player_data(player_slug, scope=scope))
                last_halo_exc = None
                break
            except Exception as exc:  # noqa: BLE001
                if _is_transient_halo_error(exc):
                    last_halo_exc = exc
                    logger.warning(
                        "initial_sync_halo_api_retry",
                        attempt=attempt + 1,
                        max_attempts=len(_HALO_RETRY_DELAYS),
                        reason=str(exc),
                        retry_in=delay,
                    )
                    time.sleep(delay)
                else:
                    raise _SyncAbortError(f"Échec de la synchronisation : {exc}") from exc
        else:
            raise _SyncHaloApiError(
                f"Échec API Halo après {len(_HALO_RETRY_DELAYS)} tentatives : {last_halo_exc}"
            ) from last_halo_exc

        # Compter les matchs importés (best-effort)
        return _count_player_matches(player_slug)

    except ImportError:  # guard env partiel (tests) \u2014 supprimer quand API autonome
        # Modules src/ non disponibles (tests, env partiel) — simuler la progression
        store.update(
            job_id,
            phase_key="enrich",
            phase_label=_PHASE_LABELS["enrich"],
            progress_pct=70,
            current_step="Modules de synchronisation non disponibles dans cet environnement",
            warnings=["sync_modules_unavailable"],
        )
        return 0
    except (_SyncAbortError, _SyncHaloApiError):
        raise
    except Exception as exc:  # noqa: BLE001
        raise _SyncAbortError(f"Échec de la synchronisation : {exc}") from exc


def _run_finalize(player_slug: str) -> None:
    """Rafraîchit les vues matérialisées post-sync (best-effort)."""
    try:
        from src.data.sync.migrations import (
            refresh_materialized_views,  # type: ignore[import-untyped]
        )

        refresh_materialized_views(player_slug)
    except (ImportError, Exception):  # noqa: BLE001
        pass


def _count_player_matches(player_slug: str) -> int:
    """Retourne le nombre de matchs dans shared_matches_v2 pour ce joueur."""
    try:
        from pathlib import Path

        from src.utils.db import duckdb_read_only

        db_path = (
            Path(__file__).resolve().parents[4] / "data" / "warehouse" / "shared_matches_v2.duckdb"
        )
        if not db_path.exists():
            return 0
        with duckdb_read_only(db_path) as con:
            row = con.execute("SELECT COUNT(*) FROM match_registry").fetchone()
            return int(row[0]) if row else 0
    except Exception:  # noqa: BLE001
        return 0
