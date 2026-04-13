"""Router Historique des parties (Slice 3).

Endpoints :
  POST /api/v1/players/{player_slug}/pages/match-history/query
  POST /api/v1/players/{player_slug}/pages/match-history/export
"""

from __future__ import annotations

import structlog
from fastapi import APIRouter, Depends

from apps.api.app.deps.players import PlayerContext, resolve_player
from apps.api.app.schemas.match_history import (
    FileTokenResponse,
    MatchHistoryExportRequest,
    MatchHistoryPageResponse,
    MatchHistoryQueryRequest,
)

logger = structlog.get_logger(__name__)

router = APIRouter(
    prefix="/players/{player_slug}",
    tags=["match-history"],
)


@router.post("/pages/match-history/query", response_model=MatchHistoryPageResponse)
def query_match_history(
    body: MatchHistoryQueryRequest,
    player: PlayerContext = Depends(resolve_player),
) -> MatchHistoryPageResponse:
    """Retourne la page paginée de l'historique des parties pour un joueur."""
    from apps.api.app.services.match_history_service import get_match_history_page

    return get_match_history_page(player, body)


@router.post("/pages/match-history/export", response_model=FileTokenResponse)
def export_match_history(
    body: MatchHistoryExportRequest,
    player: PlayerContext = Depends(resolve_player),
) -> FileTokenResponse:
    """Génère un jeton d'export CSV pour l'historique filtré des parties."""
    from apps.api.app.services.match_history_service import get_match_history_export

    return get_match_history_export(player, body)
