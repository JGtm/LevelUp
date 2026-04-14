"""Service bootstrap — assemble le `BootstrapResponse` depuis les sources Python.

Ce service est le seul responsable de lire :
- db_profiles.json (joueurs disponibles)
- app_settings.json (préférences)
- L'état auth du joueur courant (via src/auth/)
- Le flag DEMO_MODE

Il ne doit contenir aucun appel à Streamlit ni aucune import de `src/ui/`.
"""

from __future__ import annotations

import json
import os
from pathlib import Path

import structlog

from apps.api.app.core.config import get_settings
from apps.api.app.deps.auth import SessionData
from apps.api.app.deps.players import get_available_players, load_db_profiles
from apps.api.app.schemas.bootstrap import (
    BootstrapResponse,
    FeatureFlags,
    HaloIdentitySummary,
    PlayersListResponse,
    SessionContextResponse,
    SettingsExcerpt,
)
from apps.api.app.schemas.common import CapabilityMap, PlayerSummary

logger = structlog.get_logger(__name__)


def _load_app_settings() -> dict:
    """Charge app_settings.json. Retourne {} si absent ou illisible."""
    settings = get_settings()
    path = Path(settings.app_settings_path)
    if not path.exists():
        return {}
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        logger.warning("app_settings_load_error", path=str(path))
        return {}


def _resolve_auth_state(session: SessionData) -> str:
    """Déduit l'état auth depuis la session.

    Returns:
        "missing" | "partial" | "ready"
    """
    if session.auth_ready:
        return "ready"
    # Si un joueur courant est défini mais pas auth_ready → partial
    if session.current_player_slug:
        return "partial"
    return "missing"


def _build_capabilities(app_cfg: dict) -> CapabilityMap:
    """Construit la carte de capacités depuis la configuration de l'app."""
    settings = get_settings()
    return CapabilityMap(
        can_read_local_data=True,
        can_run_sync=not settings.demo_mode,
        can_use_live_halo=not settings.demo_mode,
        can_manage_settings=True,
        can_reset_media_index=True,
        can_view_media=bool(app_cfg.get("media_enabled", True)),
        can_self_provision=bool(app_cfg.get("can_self_provision", True)),
        can_start_initial_sync=bool(app_cfg.get("can_start_initial_sync", not settings.demo_mode)),
        can_manage_instance=True,
    )


def _build_settings_excerpt(app_cfg: dict) -> SettingsExcerpt:
    """Extrait les préférences mini nécessaires au bootstrap du shell."""
    return SettingsExcerpt(
        lang=app_cfg.get("lang", os.getenv("LEVELUP_LANG", "fr")),
        user_timezone=app_cfg.get("user_timezone", "Europe/Paris"),
        show_records=bool(app_cfg.get("show_records", True)),
        normalize_mode_labels=bool(app_cfg.get("normalize_mode_labels", True)),
    )


def _setup_required() -> bool:
    """Retourne True si aucun joueur n'est encore configuré (premier lancement)."""
    settings = get_settings()
    if settings.demo_mode:
        return False
    profiles = load_db_profiles()
    return len(profiles) == 0


def _check_initial_sync_marker(available: list[PlayerSummary]) -> bool:
    """Vérifie si le marqueur ``initial_sync_completed_at`` exist dans sync_meta.

    Source de vérité persistante pour distinguer ``profile_ready_no_sync``
    de ``ready``. Cherche sur le premier joueur disponible (MVP single-user).

    Retourne True si le marqueur est trouvé. Fail-open (True) en cas d'erreur.
    """
    if not available:
        return False
    settings = get_settings()
    player_slug = available[0].player_slug
    player_db = Path(settings.repo_root) / "data" / "players" / player_slug / "stats.duckdb"
    if not player_db.exists():
        return False
    try:
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(str(player_db)) as conn:
            row = conn.execute(
                "SELECT value FROM sync_meta WHERE key = 'initial_sync_completed_at' LIMIT 1"
            ).fetchone()
            return bool(row and row[0])
    except Exception:
        logger.debug("initial_sync_marker_check_failed", player_slug=player_slug)
        return False


def _has_any_synced_matches(available: list[PlayerSummary]) -> bool:
    """Vérifie si au moins un joueur a des matchs dans shared_matches_v2.duckdb.

    Utilisé pour distinguer ``profile_ready_no_sync`` de ``ready``.
    Retourne True en cas d'erreur (fail-open — ne bloque pas le démarrage).
    """
    if not available:
        return False
    settings = get_settings()
    shared_path = Path(settings.repo_root) / "data" / "warehouse" / "shared_matches_v2.duckdb"
    if not shared_path.exists():
        return False
    xuids = [p.xuid for p in available if p.xuid]
    if not xuids:
        return False
    try:
        from src.utils.db import duckdb_read_only

        placeholders = ", ".join("?" * len(xuids))
        with duckdb_read_only(str(shared_path)) as conn:
            row = conn.execute(
                f"SELECT COUNT(*) FROM match_participants WHERE xuid IN ({placeholders}) LIMIT 1",
                xuids,
            ).fetchone()
            return bool(row and row[0] > 0)
    except Exception:
        logger.warning("has_any_synced_matches_error", exc_info=True)
        return True  # fail-open : ne pas bloquer le bootstrap


def _compute_setup_state(
    auth_state: str,
    available: list,
    *,
    has_initial_sync_marker: bool = False,
    has_matches: bool = True,
) -> str:
    """Déduit l'état de l'onboarding depuis auth + joueurs + marqueur sync.

    Logique de dérivation :
    - ``no_halo_link``             → aucun token Halo connu
    - ``halo_linked_no_profile``   → auth OK mais aucun joueur local
    - ``profile_ready_no_sync``    → joueur créé mais aucun match synchronisé
    - ``ready``                    → auth OK + joueur(s) avec marqueur ou matchs

    Le marqueur ``initial_sync_completed_at`` est la source de vérité principale.
    ``has_matches`` sert de fallback pour les profils historiques non encore
    migrés (sera supprimé una fois le backfill de migration terminé).
    """
    if auth_state == "missing":
        return "no_halo_link"
    if not available:
        return "halo_linked_no_profile"
    if has_initial_sync_marker or has_matches:
        return "ready"
    return "profile_ready_no_sync"


def build_bootstrap_response(session: SessionData) -> BootstrapResponse:
    """Construit le ``BootstrapResponse`` complet.

    Args:
        session: Session web courante (peut être une session vide toute nouvelle).

    Returns:
        Réponse de bootstrap complète prête à être sérialisée.
    """
    settings = get_settings()
    app_cfg = _load_app_settings()
    available = get_available_players()

    # Mode démo : court-circuit complet de l'onboarding
    if settings.demo_mode:
        return BootstrapResponse(
            setup_required=False,
            auth_state="ready",
            setup_state="ready",
            current_player=available[0] if available else None,
            available_players=available,
            locale=session.locale,
            hints_visible_default=session.hints_visible,
            feature_flags=FeatureFlags(
                v7_enabled=True,
                media_enabled=bool(app_cfg.get("media_enabled", True)),
                demo_mode=True,
                discord_configured=False,
                tailscale_enabled=False,
            ),
            capabilities=_build_capabilities(app_cfg),
            settings_excerpt=_build_settings_excerpt(app_cfg),
            linked_halo_identity=None,
            active_sync_job_id=None,
        )

    # Joueur courant depuis la session, sinon premier joueur disponible
    current: PlayerSummary | None = None
    if session.current_player_slug:
        current = next((p for p in available if p.player_slug == session.current_player_slug), None)
    if current is None and available:
        current = available[0]

    auth_state = _resolve_auth_state(session)
    has_initial_sync_marker = _check_initial_sync_marker(available)
    has_matches = _has_any_synced_matches(available)

    # Identité Halo liée — depuis la session (jamais inventée)
    linked_identity: HaloIdentitySummary | None = None
    if session.linked_halo_identity:
        linked_identity = HaloIdentitySummary(
            gamertag=session.linked_halo_identity.get("gamertag", ""),
            xuid=session.linked_halo_identity.get("xuid", ""),
        )

    # Vérifier que le job sync actif enregistré dans la session est encore valide
    active_sync_job_id: str | None = _resolve_active_sync_job_id(session)

    # setup_state est la source de vérité unique V7 — setup_required en est dérivé
    setup_state = _compute_setup_state(
        auth_state,
        available,
        has_initial_sync_marker=has_initial_sync_marker,
        has_matches=has_matches,
    )

    return BootstrapResponse(
        setup_required=setup_state != "ready",
        auth_state=auth_state,
        setup_state=setup_state,
        current_player=current,
        available_players=available,
        locale=session.locale,
        hints_visible_default=session.hints_visible,
        feature_flags=FeatureFlags(
            v7_enabled=True,
            media_enabled=bool(app_cfg.get("media_enabled", True)),
            demo_mode=settings.demo_mode,
            discord_configured=bool(app_cfg.get("discord_webhook_url")),
            tailscale_enabled=bool(app_cfg.get("tailscale_enabled", False)),
        ),
        capabilities=_build_capabilities(app_cfg),
        settings_excerpt=_build_settings_excerpt(app_cfg),
        linked_halo_identity=linked_identity,
        active_sync_job_id=active_sync_job_id,
    )


def _resolve_active_sync_job_id(session: SessionData) -> str | None:
    """Retourne l'ID du job sync actif de la session s'il est encore en cours."""
    if not session.active_sync_job_id:
        return None
    try:
        from apps.api.app.services.job_store import JobStore

        job = JobStore.get().get_job(session.active_sync_job_id)
        if job is None:
            return None
        if job.status in ("queued", "running"):
            return session.active_sync_job_id
        # Job terminal — plus pertinent à exposer
        return None
    except Exception:
        return None


def build_players_list_response() -> PlayersListResponse:
    """Retourne la liste des joueurs disponibles."""
    players = get_available_players()
    default_slug = players[0].player_slug if players else None
    return PlayersListResponse(items=players, default_player_slug=default_slug)


def build_session_context_response(session: SessionData) -> SessionContextResponse:
    """Construit le `SessionContextResponse` depuis la session courante."""
    app_cfg = _load_app_settings()
    available = get_available_players()

    current: PlayerSummary | None = None
    if session.current_player_slug:
        current = next((p for p in available if p.player_slug == session.current_player_slug), None)

    return SessionContextResponse(
        current_player=current,
        locale=session.locale,
        hints_visible=session.hints_visible,
        capabilities=_build_capabilities(app_cfg),
    )
