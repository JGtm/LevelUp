"""Router bootstrap — endpoints Slice 0a.

Routes :
- `GET  /api/v1/bootstrap`        → `BootstrapResponse`
- `GET  /api/v1/players`          → `PlayersListResponse`
- `POST /api/v1/session/context`  → `SessionContextResponse`
"""

from __future__ import annotations

from fastapi import APIRouter, Depends, Request, Response

from apps.api.app.core.errors import ApiError
from apps.api.app.deps.auth import SessionData, get_or_create_session
from apps.api.app.deps.players import get_available_players
from apps.api.app.schemas.bootstrap import (
    BootstrapResponse,
    PlayersListResponse,
    SessionContextRequest,
    SessionContextResponse,
)
from apps.api.app.services.bootstrap_service import (
    build_bootstrap_response,
    build_players_list_response,
    build_session_context_response,
)

router = APIRouter(tags=["bootstrap"])


@router.get("/bootstrap", response_model=BootstrapResponse, summary="Bootstrap du shell React")
async def bootstrap(
    request: Request,
    response: Response,
    session_tuple: tuple[SessionData, bool] = Depends(get_or_create_session),
) -> BootstrapResponse:
    """Point d'entrée unique du shell React.

    Retourne l'état complet de l'application :
    configuration du setup, état auth, joueur courant, capacités, feature flags.

    Le frontend appelle cet endpoint au démarrage et après chaque action
    susceptible de changer l'état global (auth, changement de joueur, etc.).
    """
    session, _ = session_tuple
    return build_bootstrap_response(session)


@router.get("/players", response_model=PlayersListResponse, summary="Liste des joueurs disponibles")
async def players_list(
    response: Response,
    session_tuple: tuple[SessionData, bool] = Depends(get_or_create_session),
) -> PlayersListResponse:
    """Retourne la liste de tous les joueurs configurés.

    En DEMO_MODE, retourne le joueur de démo pointant sur les fixtures.
    """
    return build_players_list_response()


@router.post(
    "/session/context",
    response_model=SessionContextResponse,
    summary="Mise à jour du contexte de session",
)
async def update_session_context(
    body: SessionContextRequest,
    request: Request,
    response: Response,
    session_tuple: tuple[SessionData, bool] = Depends(get_or_create_session),
) -> SessionContextResponse:
    """Met à jour le joueur courant, la locale ou les hints dans la session web.

    N'expose jamais le contenu de session — retourne uniquement le contexte
    nécessaire au shell React (joueur courant, locale, capacités).
    """
    from apps.api.app.deps.auth import _get_store  # import local évite cycle

    session, _ = session_tuple
    store = _get_store()

    if body.player_slug is not None:
        # Valider que le slug existe bien
        available = get_available_players()
        if not any(p.player_slug == body.player_slug for p in available):
            raise ApiError.not_found("Joueur", body.player_slug)
        session.current_player_slug = body.player_slug

    if body.locale is not None:
        if body.locale not in ("fr", "en"):
            raise ApiError.bad_request("Locale invalide — valeurs acceptées : fr, en")
        session.locale = body.locale

    if body.hints_visible is not None:
        session.hints_visible = body.hints_visible

    store.save(session)
    return build_session_context_response(session)
