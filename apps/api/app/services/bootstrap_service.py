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
        can_start_initial_sync=not settings.demo_mode,
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


def _compute_setup_state(auth_state: str, available: list) -> str:
    """Déduit l'état de l'onboarding depuis auth + joueurs disponibles.

    Returns:
        "no_halo_link"           → aucun token Halo connu
        "halo_linked_no_profile" → auth OK mais aucun joueur local
        "ready"                  → auth OK + joueur(s) créé(s)
    """
    if auth_state == "missing":
        return "no_halo_link"
    if not available:
        return "halo_linked_no_profile"
    return "ready"


def build_bootstrap_response(session: SessionData) -> BootstrapResponse:
    """Construit le `BootstrapResponse` complet.

    Args:
        session: Session web courante (peut être une session vide toute nouvelle).

    Returns:
        Réponse de bootstrap complète prête à être sérialisée.
    """
    settings = get_settings()
    app_cfg = _load_app_settings()
    available = get_available_players()

    # Joueur courant depuis la session, sinon premier joueur disponible
    current: PlayerSummary | None = None
    if session.current_player_slug:
        current = next((p for p in available if p.player_slug == session.current_player_slug), None)
    if current is None and available:
        current = available[0]

    auth_state = _resolve_auth_state(session)
    return BootstrapResponse(
        setup_required=_setup_required(),
        auth_state=auth_state,
        setup_state=_compute_setup_state(auth_state, available),
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
    )


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
