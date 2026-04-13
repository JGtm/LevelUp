"""Router Séries temporelles — Slice 3B.

Routes :
  POST /api/v1/players/{player_slug}/pages/timeseries
"""

from __future__ import annotations

from fastapi import APIRouter, Depends

from apps.api.app.deps.players import PlayerContext, resolve_player
from apps.api.app.schemas.timeseries import TimeseriesPageResponse, TimeseriesQueryRequest

router = APIRouter(tags=["timeseries"])


@router.post(
    "/players/{player_slug}/pages/timeseries",
    response_model=TimeseriesPageResponse,
    summary="Page Séries temporelles",
)
def get_timeseries_page(
    request: TimeseriesQueryRequest,
    player: PlayerContext = Depends(resolve_player),
) -> TimeseriesPageResponse:
    """Construit la réponse complète pour la page Séries temporelles."""
    from apps.api.app.services.timeseries_api_service import get_timeseries_page as _svc

    return _svc(player, request)
