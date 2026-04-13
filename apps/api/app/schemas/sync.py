"""Schémas Pydantic — endpoints de synchronisation."""

from __future__ import annotations

from pydantic import BaseModel, Field


class InitialSyncStartRequest(BaseModel):
    """Corps de la requête ``POST /api/v1/sync/initial``."""

    player_slug: str = Field(..., min_length=1, max_length=50)
    max_matches: int = Field(default=200, ge=1, le=2000)
