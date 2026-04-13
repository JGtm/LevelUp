"""Router Explorer (Slice 4).

Endpoints :
  GET  /api/v1/directory/gamertags/search                             → GamertagSearchResponse
  POST /api/v1/players/{player_slug}/pages/explorer/matches-query     → ExplorerMatchesQueryResponse
  POST /api/v1/players/{player_slug}/pages/explorer/player-query      → ExplorerPlayerQueryResponse
  GET  /api/v1/players/{player_slug}/matches/{match_id}               → MatchViewResponse [Phase B]
  POST /api/v1/players/{player_slug}/pages/last-match/resolve         → LastMatchResolveResponse [Phase C]
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
from apps.api.app.schemas.match_view import (
    LastMatchResolveRequest,
    LastMatchResolveResponse,
    MatchViewResponse,
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


# ---------------------------------------------------------------------------
# Slice 4 Phase B — Match View
# ---------------------------------------------------------------------------


@player_router.get("/matches/{match_id}", response_model=MatchViewResponse)
def get_match_view(
    match_id: str,
    player: PlayerContext = Depends(resolve_player),
) -> MatchViewResponse:
    """Retourne le détail complet d'un match (4 onglets + header + rank)."""
    from apps.api.app.core.errors import ApiError
    from apps.api.app.services.match_view_service import get_match_view as _get_view

    try:
        return _get_view(player, match_id)
    except Exception as exc:
        logger.error("match_view_error", match_id=match_id, error=str(exc))
        raise ApiError(
            status_code=404,
            code="match_not_found",
            message=f"Match {match_id} introuvable ou inaccessible.",
        ) from exc


# ---------------------------------------------------------------------------
# Slice 4 Phase C — Last Match
# ---------------------------------------------------------------------------


@player_router.post("/pages/last-match/resolve", response_model=LastMatchResolveResponse)
def resolve_last_match(
    body: LastMatchResolveRequest,
    player: PlayerContext = Depends(resolve_player),
) -> LastMatchResolveResponse:
    """Résout le dernier match du scope filtré et retourne la navigation prev/next."""
    from apps.api.app.core.errors import ApiError
    from apps.api.app.services.match_view_service import resolve_last_match as _resolve

    try:
        return _resolve(player, body)
    except ValueError as exc:
        if "no_matches_in_scope" in str(exc):
            raise ApiError(
                status_code=404,
                code="no_matches_in_scope",
                message="Aucun match dans le scope filtré courant.",
            ) from exc
        raise ApiError(
            status_code=400,
            code="last_match_resolve_error",
            message=str(exc),
        ) from exc
