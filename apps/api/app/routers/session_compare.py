"""Router Comparaison de sessions — Slice 3C.

Routes :
  POST /api/v1/players/{player_slug}/pages/session-compare
"""

from __future__ import annotations

from fastapi import APIRouter, Depends

from apps.api.app.deps.players import PlayerContext, resolve_player
from apps.api.app.schemas.timeseries import SessionCompareRequest, SessionCompareResponse

router = APIRouter(tags=["session-compare"])


@router.post(
    "/players/{player_slug}/pages/session-compare",
    response_model=SessionCompareResponse,
    summary="Comparaison de sessions A/B",
)
def get_session_compare(
    request: SessionCompareRequest,
    player: PlayerContext = Depends(resolve_player),
) -> SessionCompareResponse:
    """Construit la réponse complète pour la page Comparaison de sessions."""
    from apps.api.app.services.session_compare_service import get_session_compare as _svc

    return _svc(player, request)
