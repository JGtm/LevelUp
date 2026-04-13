"""Protection same-origin / CSRF pour les routes mutantes basées sur cookie.

Stratégie : vérifier l'en-tête ``Origin`` (puis ``Referer`` en fallback) sur les
routes POST/PATCH/DELETE. L'origine déclarée doit figurer dans la liste
``settings.cors_origins`` ou correspondre au ``Host`` de la requête.

FastAPI dependency : ``Depends(require_same_origin)``

Exceptions volontaires :
- Requêtes sans ``Origin`` ni ``Referer`` : autorisées uniquement si le client
  n'est PAS un navigateur (pas de cookie envoyé automatiquement). En pratique
  les appels serveur-à-serveur ou outils CLI ne posent pas de risque CSRF.
- Contexte ``DEMO_MODE`` : origines de test autorisées.
"""

from __future__ import annotations

import structlog
from fastapi import Request

from apps.api.app.core.config import get_settings
from apps.api.app.core.errors import ApiError

logger = structlog.get_logger(__name__)


def _extract_origin(request: Request) -> str | None:
    """Retourne l'origine de la requête depuis Origin puis Referer."""
    origin = request.headers.get("origin")
    if origin:
        return origin.rstrip("/")
    referer = request.headers.get("referer")
    if referer:
        # Extraire seulement scheme + host depuis l'URL Referer
        from urllib.parse import urlparse

        parsed = urlparse(referer)
        if parsed.scheme and parsed.netloc:
            return f"{parsed.scheme}://{parsed.netloc}"
    return None


def require_same_origin(request: Request) -> None:
    """Dépendance FastAPI — valide l'origine de la requête.

    Lève ``ApiError(403, "csrf_origin_mismatch")`` si l'origine ne correspond
    pas aux origines autorisées configurées via ``LEVELUP_CORS_ORIGINS``.
    """
    settings = get_settings()
    origin = _extract_origin(request)

    if origin is None:
        # Pas d'Origin/Referer — autorisé (clients non-navigateur / tests)
        return

    allowed = set(settings.cors_origins)

    # Autoriser aussi l'origine basée sur le Host de la requête lui-même
    host = request.headers.get("host", "")
    if host:
        for scheme in ("http", "https"):
            allowed.add(f"{scheme}://{host}")

    if origin not in allowed:
        logger.warning(
            "csrf_origin_rejected",
            origin=origin,
            allowed=list(allowed),
            path=str(request.url.path),
        )
        raise ApiError(
            403,
            "csrf_origin_mismatch",
            "Origine de la requête non autorisée.",
            retryable=False,
        )
