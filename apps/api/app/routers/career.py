"""Router Carrière (Slice 2).

Endpoints :
  GET /api/v1/players/{player_slug}/pages/career
  GET /api/v1/players/{player_slug}/pages/career/top-matches
  GET /api/v1/players/{player_slug}/pages/career/encounters
"""

from __future__ import annotations

import structlog
from fastapi import APIRouter, Depends

from apps.api.app.core.config import get_settings
from apps.api.app.deps.players import PlayerContext, resolve_player
from apps.api.app.schemas.career import (
    CareerEncountersResponse,
    CareerPageResponse,
    CareerTopMatchesResponse,
)

logger = structlog.get_logger(__name__)

router = APIRouter(
    prefix="/players/{player_slug}",
    tags=["career"],
)


@router.get("/pages/career", response_model=CareerPageResponse)
def get_career_page(
    player: PlayerContext = Depends(resolve_player),
) -> CareerPageResponse:
    """Retourne la page Carrière complète pour un joueur."""
    settings = get_settings()
    exclude_btb: bool = getattr(settings, "career_top_exclude_btb", False)

    from apps.api.app.services.career_service import get_career_page as _get_page

    return _get_page(player, exclude_btb=exclude_btb)


@router.get("/pages/career/top-matches", response_model=CareerTopMatchesResponse)
def get_career_top_matches(
    player: PlayerContext = Depends(resolve_player),
) -> CareerTopMatchesResponse:
    """Retourne les top matches (meilleurs + pires) du joueur."""
    settings = get_settings()
    exclude_btb: bool = getattr(settings, "career_top_exclude_btb", False)

    from apps.api.app.services.career_service import get_top_matches as _get_matches

    return _get_matches(player, exclude_btb=exclude_btb)


@router.get("/pages/career/encounters", response_model=CareerEncountersResponse)
def get_career_encounters(
    player: PlayerContext = Depends(resolve_player),
) -> CareerEncountersResponse:
    """Retourne les encounters (adversaires et coéquipiers fréquents) du joueur."""
    from apps.api.app.services.career_service import get_encounters as _get_encounters

    return _get_encounters(player)
