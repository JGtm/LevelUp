"""Router filtres — POST /api/v1/players/{player_slug}/filters/resolve."""

from __future__ import annotations

import logging

from fastapi import APIRouter, Depends

from apps.api.app.deps.players import PlayerContext, resolve_player
from apps.api.app.schemas.filters import FilterContextInput, FilterContextResolved
from apps.api.app.services.filter_service import resolve_filters

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/players/{player_slug}", tags=["filters"])


@router.post(
    "/filters/resolve",
    response_model=FilterContextResolved,
    summary="Résoudre le contexte de filtres",
    description=(
        "Normalise ``FilterContextInput``, calcule les options disponibles "
        "(playlists/modes/cartes/sessions) et retourne le contexte résolu. "
        "Remplace les shadow keys Streamlit et ``GAP_MINUTES_FIXED``."
    ),
)
def filters_resolve(
    ctx: FilterContextInput,
    player: PlayerContext = Depends(resolve_player),
) -> FilterContextResolved:
    """Résout les filtres pour un joueur donné."""
    logger.debug(
        "filters/resolve player=%s filter_mode=%s",
        player.player_slug,
        ctx.filter_mode,
    )
    return resolve_filters(player, ctx)
