"""Router Synthèse (Slice 7).

Endpoints :
  POST /api/v1/players/{player_slug}/pages/synthesis
"""

from __future__ import annotations

import structlog
from fastapi import APIRouter, Depends

from apps.api.app.deps.players import PlayerContext, resolve_player
from apps.api.app.schemas.synthesis import (
    SynthesisPageResponse,
    SynthesisQueryRequest,
)

logger = structlog.get_logger(__name__)

router = APIRouter(
    prefix="/players/{player_slug}",
    tags=["synthesis"],
)


@router.post("/pages/synthesis", response_model=SynthesisPageResponse)
def get_synthesis_page(
    request: SynthesisQueryRequest,
    player: PlayerContext = Depends(resolve_player),
) -> SynthesisPageResponse:
    """Retourne la page Synthèse solo vs escouade pour un joueur."""
    from apps.api.app.services.synthesis_service import get_synthesis_page as _svc

    return _svc(player, request)
