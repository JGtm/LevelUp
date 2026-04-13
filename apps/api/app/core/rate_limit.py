"""Rate limiting in-memory pour les routes sensibles.

Algorithme : fenêtre glissante par IP (thread-safe, process-level).
Limite par défaut : 5 requêtes par minute par IP.

FastAPI dependency : ``Depends(check_rate_limit)``

En DEMO_MODE : pas de limitation (tests + démo).
"""

from __future__ import annotations

import threading
import time
from collections import deque

import structlog
from fastapi import Request

from apps.api.app.core.errors import ApiError

logger = structlog.get_logger(__name__)

# Paramètres de la fenêtre glissante
WINDOW_SECONDS: int = 60
MAX_REQUESTS: int = 5

# Store process-level thread-safe
_lock = threading.Lock()
_windows: dict[str, deque[float]] = {}


def _get_client_ip(request: Request) -> str:
    """Retourne l'IP cliente, en lisant X-Forwarded-For si disponible."""
    forwarded_for = request.headers.get("x-forwarded-for")
    if forwarded_for:
        return forwarded_for.split(",")[0].strip()
    real_ip = request.headers.get("x-real-ip")
    if real_ip:
        return real_ip.strip()
    if request.client:
        return request.client.host
    return "unknown"


def _is_rate_limited(ip: str) -> bool:
    """Retourne True si l'IP a dépassé la limite dans la fenêtre glissante."""
    now = time.monotonic()
    cutoff = now - WINDOW_SECONDS

    with _lock:
        if ip not in _windows:
            _windows[ip] = deque()

        window = _windows[ip]

        # Supprimer les timestamps hors fenêtre
        while window and window[0] < cutoff:
            window.popleft()

        if len(window) >= MAX_REQUESTS:
            return True

        window.append(now)
        return False


def check_rate_limit(request: Request) -> None:
    """Dépendance FastAPI — vérifie le rate limit par IP.

    Lève ``ApiError(429, "rate_limit_exceeded")`` si la limite est dépassée.
    Sans effet en DEMO_MODE.
    """
    from apps.api.app.core.config import get_settings

    settings = get_settings()
    if settings.demo_mode:
        return

    ip = _get_client_ip(request)
    if _is_rate_limited(ip):
        logger.warning(
            "rate_limit_exceeded",
            ip=ip,
            path=str(request.url.path),
            window_seconds=WINDOW_SECONDS,
            max_requests=MAX_REQUESTS,
        )
        raise ApiError(
            429,
            "rate_limit_exceeded",
            f"Trop de requêtes. Réessayez dans {WINDOW_SECONDS} secondes.",
            retryable=True,
        )
