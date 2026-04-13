"""Schémas Pydantic — Médias (Slice 8).

Contrats :
  GET  /api/v1/players/{slug}/pages/media  → MediaPageResponse
"""

from __future__ import annotations

from datetime import datetime
from typing import Literal

from pydantic import BaseModel

from apps.api.app.schemas.common import PaginatedResponse, PaginationRequest


class MediaItemRow(BaseModel):
    """Un média indexé (capture ou vidéo)."""

    basename: str
    file_path: str
    kind: str = "screenshot"
    thumbnail_path: str | None = None
    match_id: str | None = None
    capture_end_utc: datetime | None = None
    match_start_time: datetime | None = None
    section: str = "mine"
    owner_gamertag: str | None = None
    map_name: str | None = None


class MediaQueryRequest(BaseModel):
    """Paramètres de filtrage et tri de la galerie médias."""

    sort: str = "date_desc"
    kind_filter: Literal["screenshot", "video", "thumbnail"] | None = None
    section_filter: str | None = None
    pagination: PaginationRequest = PaginationRequest()


class MediaPageResponse(BaseModel):
    """Réponse de la page Médias."""

    items: PaginatedResponse[MediaItemRow]
    total_mine: int = 0
    total_teammates: int = 0
    total_unassigned: int = 0
