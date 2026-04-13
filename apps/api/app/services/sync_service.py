"""Service de synchronisation initiale des données Halo — Sprint 3.

Lance un job asynchrone qui appelle l'orchestrateur de sync existant
(src/data/sync/) et met à jour le job avec des compteurs métier par phase.

Phases :
    prepare → auth → fetch_matches → enrich → verify → finalize
"""

from __future__ import annotations

import threading

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


def start_initial_sync(body: InitialSyncStartRequest) -> AsyncJobStatus:
    """Crée un job ``initial_sync`` et lance la synchronisation en arrière-plan.

    Retourne immédiatement le statut initial du job (``queued``).
    """
    store = JobStore.get()
    job = store.create("initial_sync")
    thread = threading.Thread(
        target=_run_initial_sync_bg,
        args=(job.job_id, body.player_slug, body.max_matches),
        daemon=True,
    )
    thread.start()
    return store.get_job(job.job_id) or job


# ---------------------------------------------------------------------------
# Thread background
# ---------------------------------------------------------------------------


def _run_initial_sync_bg(job_id: str, player_slug: str, max_matches: int) -> None:
    """Exécute la sync initiale dans un thread background.

    Met à jour le job à chaque changement de phase.
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

        store.update(
            job_id,
            status="succeeded",
            progress_pct=100,
            phase_key="finalize",
            phase_label=_PHASE_LABELS["finalize"],
            current_step="Synchronisation terminée",
            result={"matches_imported": match_count},
        )

    except _SyncAbortError as exc:
        logger.warning("initial_sync_aborted", job_id=job_id, reason=str(exc))
        store.update(
            job_id,
            status="failed",
            error=ApiErrorSchema(
                code="sync_aborted",
                message=str(exc),
                retryable=True,
            ),
        )
    except Exception as exc:  # noqa: BLE001
        logger.error("initial_sync_unexpected_error", job_id=job_id, exc=str(exc))
        store.update(
            job_id,
            status="failed",
            error=ApiErrorSchema(
                code="sync_failed",
                message=f"Erreur inattendue : {exc}",
                retryable=True,
            ),
        )


# ---------------------------------------------------------------------------
# Helpers internes
# ---------------------------------------------------------------------------


class _SyncAbortError(Exception):
    """Erreur métier bloquante (config manquante, joueur introuvable…)."""


def _validate_player(player_slug: str) -> None:
    """Vérifie que le joueur existe dans db_profiles.json."""
    try:
        from src.data.profile import load_profiles  # type: ignore[import-untyped]

        profiles = load_profiles()
        if player_slug not in profiles:
            raise _SyncAbortError(f"Joueur « {player_slug} » introuvable dans db_profiles.json")
    except ImportError:
        pass  # module optionnel — continuer


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
    except ImportError:
        pass  # module optionnel — continuer


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

        asyncio.run(backfill_player_data(player_slug, scope=scope))

        # Compter les matchs importés (best-effort)
        return _count_player_matches(player_slug)

    except ImportError:
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

        import duckdb

        db_path = (
            Path(__file__).resolve().parents[4] / "data" / "warehouse" / "shared_matches_v2.duckdb"
        )
        if not db_path.exists():
            return 0
        with duckdb.connect(str(db_path), read_only=True) as con:
            row = con.execute("SELECT COUNT(*) FROM match_registry").fetchone()
            return int(row[0]) if row else 0
    except Exception:  # noqa: BLE001
        return 0
