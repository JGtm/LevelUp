"""Application FastAPI LevelUp — point d'entrée principal.

Initialise :
- Logging structuré
- Middlewares (CORS, request_id)
- Handlers d'erreurs normalisés
- Router `/api/v1`
- Fichiers statiques React (dist Vite) + SPA fallback
- Purge des sessions expirées au démarrage

DEMO_MODE : activé via `LEVELUP_DEMO_MODE=true` dans l'environnement ou .env.local.
  - Auth bypassée
  - Données pointent sur tests/fixtures/ref_player/
  - Mêmes schémas que le mode normal
"""

from __future__ import annotations

import os
from collections.abc import AsyncGenerator
from contextlib import asynccontextmanager
from pathlib import Path

import structlog
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse
from fastapi.staticfiles import StaticFiles

from apps.api.app.core.config import get_settings
from apps.api.app.core.errors import (
    ApiError,
    RequestIdMiddleware,
    api_error_handler,
    unhandled_error_handler,
)
from apps.api.app.core.logging import configure_logging
from apps.api.app.routers import bootstrap as bootstrap_router
from apps.api.app.routers import career as career_router
from apps.api.app.routers import filters as filters_router
from apps.api.app.routers import health as health_router
from apps.api.app.routers import home as home_router
from apps.api.app.routers import jobs as jobs_router
from apps.api.app.routers import match_history as match_history_router
from apps.api.app.routers import media as media_router
from apps.api.app.routers import session_compare as session_compare_router
from apps.api.app.routers import settings_router
from apps.api.app.routers import setup as setup_router
from apps.api.app.routers import sync as sync_router
from apps.api.app.routers import synthesis as synthesis_router
from apps.api.app.routers import teammates as teammates_router
from apps.api.app.routers import timeseries as timeseries_router
from apps.api.app.routers.explorer import directory_router as explorer_directory_router
from apps.api.app.routers.explorer import player_router as explorer_player_router

logger = structlog.get_logger(__name__)


# ---------------------------------------------------------------------------
# Lifespan (startup / shutdown)
# ---------------------------------------------------------------------------


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncGenerator[None, None]:
    """Hook de démarrage / arrêt de l'application."""
    settings = get_settings()
    configure_logging()

    # Avertissement si clé de session par défaut en production
    _INSECURE_DEFAULT = "CHANGE_ME_IN_PRODUCTION"
    if settings.is_production and settings.session_secret_key == _INSECURE_DEFAULT:
        logger.warning(
            "insecure_session_secret",
            hint="Définir LEVELUP_SESSION_SECRET dans l'environnement de production.",
        )

    # Purge des sessions expirées au démarrage
    from apps.api.app.deps.auth import SessionStore

    store = SessionStore(settings.session_storage_dir, settings.session_ttl_seconds)
    purged = store.purge_expired()
    logger.info(
        "api_startup",
        version=settings.app_version,
        demo_mode=settings.demo_mode,
        sessions_purged=purged,
    )

    yield

    logger.info("api_shutdown")


# ---------------------------------------------------------------------------
# Création de l'app
# ---------------------------------------------------------------------------


def create_app() -> FastAPI:
    """Factory de l'application FastAPI."""
    settings = get_settings()

    app = FastAPI(
        title="LevelUp API",
        description="API FastAPI pour le dashboard de statistiques Halo Infinite LevelUp.",
        version=settings.app_version,
        docs_url="/api/docs",
        redoc_url="/api/redoc",
        openapi_url="/api/openapi.json",
        lifespan=lifespan,
    )

    # --- Middlewares --------------------------------------------------------
    # Note : en production derrière un reverse-proxy, lancer uvicorn avec
    # --proxy-headers --forwarded-allow-ips=<ip_proxy> pour que le middleware
    # ProxyHeadersMiddleware d'uvicorn résout correctement request.client.host
    # depuis X-Forwarded-For.
    app.add_middleware(
        CORSMiddleware,
        allow_origins=settings.cors_origins,
        allow_credentials=True,
        allow_methods=["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"],
        allow_headers=["*"],
        expose_headers=["X-Request-ID"],
    )
    app.add_middleware(RequestIdMiddleware)

    # --- Handlers d'erreurs -------------------------------------------------
    app.add_exception_handler(ApiError, api_error_handler)  # type: ignore[arg-type]
    app.add_exception_handler(Exception, unhandled_error_handler)  # type: ignore[arg-type]

    # --- Routers ------------------------------------------------------------
    v1 = _create_v1_router()
    app.include_router(v1, prefix="/api/v1")

    # --- Fichiers statiques React (production) ------------------------------
    # En développement, Vite tourne sur :5173 — ce bloc est ignoré si le dist
    # est absent (ce qui est toujours le cas sans `npm run build`).
    _web_dist = Path(os.getenv("LEVELUP_WEB_DIST", "apps/web/dist"))
    if _web_dist.exists():
        _assets = _web_dist / "assets"
        if _assets.exists():
            app.mount("/assets", StaticFiles(directory=str(_assets)), name="web-assets")

        @app.get("/{full_path:path}", include_in_schema=False)
        async def _spa_fallback(full_path: str) -> FileResponse:  # noqa: RUF029
            """Retourne index.html pour toutes les routes React (SPA routing)."""
            return FileResponse(str(_web_dist / "index.html"))

    return app


def _create_v1_router():  # type: ignore[return]
    """Assemble le router `/api/v1`."""
    from fastapi import APIRouter

    v1 = APIRouter()
    v1.include_router(health_router.router)
    v1.include_router(bootstrap_router.router)
    v1.include_router(filters_router.router)
    v1.include_router(setup_router.router)
    v1.include_router(sync_router.router)
    v1.include_router(settings_router.router)
    v1.include_router(jobs_router.router)
    v1.include_router(career_router.router)
    v1.include_router(match_history_router.router)
    v1.include_router(explorer_directory_router)
    v1.include_router(explorer_player_router)
    v1.include_router(home_router.router)
    v1.include_router(teammates_router.router)
    v1.include_router(synthesis_router.router)
    v1.include_router(media_router.router)
    v1.include_router(timeseries_router.router)
    v1.include_router(session_compare_router.router)
    return v1


# ---------------------------------------------------------------------------
# Singleton exposé à uvicorn
# ---------------------------------------------------------------------------

app = create_app()
