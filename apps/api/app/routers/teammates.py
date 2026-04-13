"""Router Coéquipiers (Slice 6).

Endpoints :
  POST /api/v1/players/{player_slug}/pages/teammates
"""

from __future__ import annotations

import structlog
from fastapi import APIRouter, Depends

from apps.api.app.deps.players import PlayerContext, resolve_player
from apps.api.app.schemas.teammates import (
    TeammatesPageResponse,
    TeammatesQueryRequest,
)

logger = structlog.get_logger(__name__)

router = APIRouter(
    prefix="/players/{player_slug}",
    tags=["teammates"],
)


@router.post("/pages/teammates", response_model=TeammatesPageResponse)
def get_teammates_page(
    request: TeammatesQueryRequest,
    player: PlayerContext = Depends(resolve_player),
) -> TeammatesPageResponse:
    """Retourne la page Coéquipiers avec statistiques comparatives pour un joueur."""
    from apps.api.app.services.teammates_service import get_teammates_page as _svc

    return _svc(player, request)
