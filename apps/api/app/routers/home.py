"""Router Accueil — Mission Control (Slice 5).

Endpoints :
  GET /api/v1/players/{player_slug}/pages/home
  GET /api/v1/players/{player_slug}/battlepass
  GET /api/v1/players/{player_slug}/challenges
"""

from __future__ import annotations

import structlog
from fastapi import APIRouter, Depends

from apps.api.app.deps.players import PlayerContext, resolve_player
from apps.api.app.schemas.home import (
    BattlePassResponse,
    ChallengesResponse,
    HomePageResponse,
)

logger = structlog.get_logger(__name__)

router = APIRouter(
    prefix="/players/{player_slug}",
    tags=["home"],
)


@router.get("/pages/home", response_model=HomePageResponse)
def get_home_page(
    player: PlayerContext = Depends(resolve_player),
) -> HomePageResponse:
    """Retourne la page d'accueil Mission Control pour un joueur."""
    from apps.api.app.services.home_service import get_home_page as _svc

    return _svc(player)


@router.get("/battlepass", response_model=BattlePassResponse)
def get_battlepass(
    player: PlayerContext = Depends(resolve_player),
) -> BattlePassResponse:
    """Retourne les informations Battle Pass (best-effort, nécessite une session Halo active)."""
    from apps.api.app.services.home_service import get_battlepass as _svc

    return _svc(player)


@router.get("/challenges", response_model=ChallengesResponse)
def get_challenges(
    player: PlayerContext = Depends(resolve_player),
) -> ChallengesResponse:
    """Retourne les défis actifs (best-effort, nécessite une session Halo active)."""
    from apps.api.app.services.home_service import get_challenges as _svc

    return _svc(player)
