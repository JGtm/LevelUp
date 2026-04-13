"""Router Médiathèque (Slice 8).

Endpoints :
  POST /api/v1/players/{player_slug}/pages/media
"""

from __future__ import annotations

import structlog
from fastapi import APIRouter, Depends

from apps.api.app.deps.players import PlayerContext, resolve_player
from apps.api.app.schemas.media import (
    MediaPageResponse,
    MediaQueryRequest,
)

logger = structlog.get_logger(__name__)

router = APIRouter(
    prefix="/players/{player_slug}",
    tags=["media"],
)


@router.post("/pages/media", response_model=MediaPageResponse)
def get_media_page(
    request: MediaQueryRequest,
    player: PlayerContext = Depends(resolve_player),
) -> MediaPageResponse:
    """Retourne la médiathèque paginée pour un joueur."""
    from apps.api.app.services.media_service import get_media_page as _svc

    return _svc(player, request)
