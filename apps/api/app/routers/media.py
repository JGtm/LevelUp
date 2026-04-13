"""Router Médiathèque (Slice 8).

Endpoints :
  GET /api/v1/players/{player_slug}/pages/media
"""

from __future__ import annotations

import structlog
from fastapi import APIRouter, Depends, Query

from apps.api.app.deps.players import PlayerContext, resolve_player
from apps.api.app.schemas.common import PaginationRequest
from apps.api.app.schemas.media import (
    MediaPageResponse,
    MediaQueryRequest,
)

logger = structlog.get_logger(__name__)

router = APIRouter(
    prefix="/players/{player_slug}",
    tags=["media"],
)


@router.get("/pages/media", response_model=MediaPageResponse)
def get_media_page(  # noqa: PLR0913
    player: PlayerContext = Depends(resolve_player),
    sort: str = Query(default="date_desc", pattern="^(date_desc|date_asc)$"),
    kind_filter: str | None = Query(default=None, pattern="^(screenshot|video)$"),
    section_filter: str | None = Query(default=None, pattern="^(mine|teammate|unassigned)$"),
    page: int = Query(default=1, ge=1),
    page_size: int = Query(default=50, ge=1, le=200),
) -> MediaPageResponse:
    """Retourne la médiathèque paginée pour un joueur."""
    from apps.api.app.services.media_service import get_media_page as _svc

    request = MediaQueryRequest(
        sort=sort,
        kind_filter=kind_filter,
        section_filter=section_filter,
        pagination=PaginationRequest(page=page, page_size=page_size),
    )
    return _svc(player, request)
