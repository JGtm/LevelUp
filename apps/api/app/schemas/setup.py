"""Schémas Pydantic pour les endpoints Setup / Auth (Slice 1).

Couvre :
- Machine d'état du wizard d'installation (SetupStatusResponse)
- Device Code Flow Microsoft (DeviceFlow*)
- Création de profil joueur (CreatePlayerProfile*)
- Smoke test post-installation (SmokeTestStartRequest)
"""

from __future__ import annotations

from pydantic import BaseModel, Field

from apps.api.app.schemas.common import ApiErrorSchema, PlayerSummary

# ---------------------------------------------------------------------------
# Setup status
# ---------------------------------------------------------------------------


class SetupAuthInfo(BaseModel):
    """État de l'authentification Halo."""

    has_client_id: bool
    has_refresh_token: bool
    has_msal_cache: bool
    preferred_method: str  # "refresh_token" | "device_code" | "unknown"


class SetupPlayerInfo(BaseModel):
    """État des profils joueurs configurés."""

    has_any_profile: bool
    default_player_slug: str | None = None


class SetupStatusResponse(BaseModel):
    """Réponse de GET /setup/status — machine d'état setup.

    ``next_blocking_step`` contrôle la navigation dans le wizard :
    - ``"auth"``      → pas de refresh_token → afficher le Device Code Flow
    - ``"player"``    → auth OK mais aucun joueur configuré
    - ``"smoke_test"``→ joueur créé mais smoke test jamais lancé (optionnel)
    - ``"done"``      → tout configuré, accès aux routes protégées autorisé
    """

    needs_setup: bool
    auth: SetupAuthInfo
    player: SetupPlayerInfo
    next_blocking_step: str  # "choose_mode" | "auth" | "player" | "smoke_test" | "done"


# ---------------------------------------------------------------------------
# Device Code Flow
# ---------------------------------------------------------------------------


class DeviceFlowStartResponse(BaseModel):
    """Réponse à POST /auth/device-flow/start."""

    attempt_id: str
    user_code: str
    verification_uri: str
    verification_uri_complete: str | None = None
    expires_in_seconds: int
    poll_interval_seconds: int = 5


class DeviceFlowStatusResponse(BaseModel):
    """Réponse à GET /auth/device-flow/{attempt_id}."""

    attempt_id: str
    status: str  # "pending" | "authorized" | "provisioned" | "failed" | "expired"
    gamertag: str | None = None
    xuid: str | None = None
    error: ApiErrorSchema | None = None


# ---------------------------------------------------------------------------
# Création de profil joueur
# ---------------------------------------------------------------------------


class CreatePlayerProfileRequest(BaseModel):
    """Corps de POST /setup/players."""

    gamertag: str = Field(..., min_length=1, max_length=50)
    xuid: str | None = None
    profile_mode: str = "xbox"  # "xbox" | "azure_manual"


class CreatePlayerProfileResponse(BaseModel):
    """Réponse à POST /setup/players."""

    player: PlayerSummary
    db_created: bool
    warnings: list[str] = Field(default_factory=list)


# ---------------------------------------------------------------------------
# Smoke test
# ---------------------------------------------------------------------------


class SmokeTestStartRequest(BaseModel):
    """Corps de POST /setup/smoke-test."""

    player_slug: str
    max_matches: int = Field(default=20, ge=1, le=500)
    run_backfill: bool = True
