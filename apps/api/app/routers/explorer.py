"""Router Explorer (Slice 4).

Endpoints :
  GET  /api/v1/directory/gamertags/search                             → GamertagSearchResponse
  POST /api/v1/players/{player_slug}/pages/explorer/matches-query     → ExplorerMatchesQueryResponse
  POST /api/v1/players/{player_slug}/pages/explorer/player-query      → ExplorerPlayerQueryResponse
"""

from __future__ import annotations

import structlog
from fastapi import APIRouter, Depends, Query

from apps.api.app.deps.players import PlayerContext, resolve_player
from apps.api.app.schemas.explorer import (
    ExplorerMatchesQueryRequest,
    ExplorerMatchesQueryResponse,
    ExplorerPlayerQueryRequest,
    ExplorerPlayerQueryResponse,
    GamertagSearchResponse,
)

logger = structlog.get_logger(__name__)

# Router racine pour /directory (pas de préfixe joueur)
directory_router = APIRouter(
    prefix="/directory",
    tags=["explorer"],
)

# Router joueur
player_router = APIRouter(
    prefix="/players/{player_slug}",
    tags=["explorer"],
)


@directory_router.get("/gamertags/search", response_model=GamertagSearchResponse)
def search_gamertags(
    q: str = Query(default="", description="Requête de recherche (min. 2 caractères)"),
    limit: int = Query(default=8, ge=1, le=50, description="Nombre maximal de suggestions"),
) -> GamertagSearchResponse:
    """Recherche floue de gamertags dans la base partagée.

    Accessible sans joueur courant (endpoint global).
    """
    from apps.api.app.core.config import get_settings
    from apps.api.app.services.explorer_service import search_gamertags as _search

    settings = get_settings()

    from pathlib import Path

    if settings.demo_mode:
        shared_db = str(Path(settings.demo_fixtures_dir) / "shared_matches_v2.duckdb")
    else:
        shared_db = str(
            Path(settings.repo_root) / "data" / "warehouse" / "shared_matches_v2.duckdb"
        )

    return _search(q, limit, shared_db)


@player_router.post("/pages/explorer/matches-query", response_model=ExplorerMatchesQueryResponse)
def query_explorer_matches(
    body: ExplorerMatchesQueryRequest,
    player: PlayerContext = Depends(resolve_player),
) -> ExplorerMatchesQueryResponse:
    """Retourne la liste paginée et filtrée des matchs pour la vue Explorer."""
    from apps.api.app.services.explorer_service import get_explorer_matches

    return get_explorer_matches(player, body)


@player_router.post("/pages/explorer/player-query", response_model=ExplorerPlayerQueryResponse)
def query_explorer_player(
    body: ExplorerPlayerQueryRequest,
    player: PlayerContext = Depends(resolve_player),
) -> ExplorerPlayerQueryResponse:
    """Retourne le profil d'encounter et les matchs communs avec un joueur cible."""
    from apps.api.app.services.explorer_service import get_explorer_player

    return get_explorer_player(player, body)
