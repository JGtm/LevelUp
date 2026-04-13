"""Schémas Pydantic pour les endpoints Setup / Auth — V7.

Couvre :
- Device Code Flow Microsoft (DeviceFlow*)
- Création de profil joueur (CreatePlayerProfile*)
"""

from __future__ import annotations

from pydantic import BaseModel, Field

from apps.api.app.schemas.common import ApiErrorSchema, PlayerSummary

# ---------------------------------------------------------------------------
# Device Code Flow
# ---------------------------------------------------------------------------


class DeviceFlowStartResponse(BaseModel):
    """Réponse à POST /auth/device-flow/start."""

    attempt_id: str
    user_code: str
    verification_uri: str
    verification_uri_complete: str | None = None
    # Durée de validité en secondes depuis l'émission (two names for compatibility)
    expires_in: int = 900
    expires_in_seconds: int = 900
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
