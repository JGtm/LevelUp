"""Gestion centralisée des erreurs HTTP — `ApiError` et middlewares associés.

- `ApiError` : exception applicative avec code, message, retryable.
- `api_error_handler` : handler FastAPI qui transforme `ApiError` en JSON.
- `RequestIdMiddleware` : injecte un `X-Request-ID` unique dans chaque requête.
"""

from __future__ import annotations

import uuid

import structlog
from fastapi import Request, status
from fastapi.responses import JSONResponse
from starlette.middleware.base import BaseHTTPMiddleware, RequestResponseEndpoint
from starlette.responses import Response

from apps.api.app.schemas.common import ApiErrorSchema, FieldErrorSchema

logger = structlog.get_logger(__name__)


# ---------------------------------------------------------------------------
# Exception applicative
# ---------------------------------------------------------------------------


class ApiError(Exception):
    """Exception levée par les services pour signaler une erreur métier/HTTP.

    Args:
        status_code: Code HTTP à retourner (ex: 400, 404, 409…).
        code: Code fonctionnel lisible par machine (ex: ``"player_not_found"``).
        message: Message lisible par l'utilisateur.
        retryable: Indique si le front peut relancer automatiquement.
        details: Données supplémentaires libres.
        field_errors: Liste d'erreurs par champ (validation form).
    """

    def __init__(  # noqa: PLR0913
        self,
        status_code: int,
        code: str,
        message: str,
        *,
        retryable: bool = False,
        details: dict | list | None = None,
        field_errors: list[dict] | None = None,
    ) -> None:
        super().__init__(message)
        self.status_code = status_code
        self.code = code
        self.message = message
        self.retryable = retryable
        self.details = details
        self.field_errors = field_errors

    # Helpers communs
    @classmethod
    def not_found(cls, resource: str, identifier: str | None = None) -> ApiError:
        msg = f"{resource} introuvable" + (f" : {identifier}" if identifier else "")
        return cls(status.HTTP_404_NOT_FOUND, "not_found", msg)

    @classmethod
    def conflict(cls, message: str) -> ApiError:
        return cls(status.HTTP_409_CONFLICT, "conflict", message)

    @classmethod
    def bad_request(cls, message: str, code: str = "bad_request") -> ApiError:
        return cls(status.HTTP_400_BAD_REQUEST, code, message)

    @classmethod
    def unauthorized(cls) -> ApiError:
        return cls(status.HTTP_401_UNAUTHORIZED, "unauthorized", "Session expirée ou absente")

    @classmethod
    def internal(cls, message: str = "Erreur interne inattendue") -> ApiError:
        return cls(
            status.HTTP_500_INTERNAL_SERVER_ERROR,
            "internal_error",
            message,
            retryable=True,
        )


# ---------------------------------------------------------------------------
# Handler FastAPI
# ---------------------------------------------------------------------------


async def api_error_handler(request: Request, exc: ApiError) -> JSONResponse:
    """Transforme une `ApiError` en réponse JSON normalisée."""
    field_errors = [FieldErrorSchema(**fe) for fe in exc.field_errors] if exc.field_errors else None
    body = ApiErrorSchema(
        code=exc.code,
        message=exc.message,
        retryable=exc.retryable,
        details=exc.details,
        field_errors=field_errors,
    )
    logger.warning(
        "api_error",
        status_code=exc.status_code,
        code=exc.code,
        message=exc.message,
        path=str(request.url.path),
    )
    return JSONResponse(status_code=exc.status_code, content=body.model_dump())


async def unhandled_error_handler(request: Request, exc: Exception) -> JSONResponse:
    """Capture les exceptions non gérées et retourne un 500 normalisé."""
    logger.exception("unhandled_exception", path=str(request.url.path), exc_info=exc)
    body = ApiErrorSchema(
        code="internal_error",
        message="Une erreur interne inattendue est survenue.",
        retryable=True,
    )
    return JSONResponse(
        status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
        content=body.model_dump(),
    )


# ---------------------------------------------------------------------------
# Middleware request_id
# ---------------------------------------------------------------------------


class RequestIdMiddleware(BaseHTTPMiddleware):
    """Injecte un `X-Request-ID` unique dans chaque requête / réponse.

    L'identifiant est également ajouté au contexte structlog pour corrélation
    de tous les logs de la requête.
    """

    async def dispatch(self, request: Request, call_next: RequestResponseEndpoint) -> Response:
        request_id = request.headers.get("X-Request-ID") or str(uuid.uuid4())
        structlog.contextvars.bind_contextvars(request_id=request_id)
        try:
            response: Response = await call_next(request)
            response.headers["X-Request-ID"] = request_id
            return response
        finally:
            structlog.contextvars.unbind_contextvars("request_id")
