"""Schémas Pydantic pour les endpoints Slice 0 — Bootstrap et Players."""

from __future__ import annotations

from pydantic import BaseModel

from apps.api.app.schemas.common import CapabilityMap, PlayerSummary


class FeatureFlags(BaseModel):
    """Flags de fonctionnalités actifs sur ce déploiement."""

    v7_enabled: bool = True
    media_enabled: bool = True
    demo_mode: bool = False
    discord_configured: bool = False
    tailscale_enabled: bool = False


class SettingsExcerpt(BaseModel):
    """Extrait des préférences utilisateur nécessaires au bootstrap du shell."""

    lang: str = "fr"
    user_timezone: str = "Europe/Paris"
    show_records: bool = True
    normalize_mode_labels: bool = True


class HaloIdentitySummary(BaseModel):
    """Identité Halo résolue côté backend — confirmée au provisioning."""

    gamertag: str
    xuid: str


class BootstrapResponse(BaseModel):
    """Réponse de `GET /api/v1/bootstrap` — point d'entrée du shell React.

    Contient tout ce dont le shell a besoin pour s'initialiser :
    état du setup, auth, joueur courant, capacités, drapeaux de fonctionnalités.
    """

    setup_required: bool
    auth_state: str  # "missing" | "partial" | "ready"
    setup_state: (
        str  # "no_halo_link" | "halo_linked_no_profile" | "profile_ready_no_sync" | "ready"
    )
    current_player: PlayerSummary | None = None
    available_players: list[PlayerSummary]
    locale: str = "fr"
    hints_visible_default: bool = True
    feature_flags: FeatureFlags
    capabilities: CapabilityMap
    settings_excerpt: SettingsExcerpt
    # Sprint 1 — identité Halo liée côté serveur (jamais saisie libre côté client)
    linked_halo_identity: HaloIdentitySummary | None = None
    # Sprint 3 — job sync actif (reprise du polling après refresh navigateur)
    active_sync_job_id: str | None = None


class PlayersListResponse(BaseModel):
    """Réponse de `GET /api/v1/players`."""

    items: list[PlayerSummary]
    default_player_slug: str | None = None


class SessionContextRequest(BaseModel):
    """Corps de `POST /api/v1/session/context` pour mettre à jour le joueur courant."""

    player_slug: str | None = None
    locale: str | None = None  # "fr" | "en"
    hints_visible: bool | None = None


class SessionContextResponse(BaseModel):
    """Réponse de `POST /api/v1/session/context`."""

    current_player: PlayerSummary | None = None
    locale: str = "fr"
    hints_visible: bool = True
    capabilities: CapabilityMap
