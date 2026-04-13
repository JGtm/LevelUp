"""Tests unitaires — Sprint 4 : CSRF + rate limiting + ProxyHeaders config."""

from __future__ import annotations

import pytest
from httpx import ASGITransport, AsyncClient

# ===========================================================================
# Fixtures
# ===========================================================================


@pytest.fixture(autouse=True)
def _demo_env(monkeypatch: pytest.MonkeyPatch) -> None:
    """DEMO_MODE par défaut pour la majorité des tests."""
    monkeypatch.setenv("LEVELUP_DEMO_MODE", "true")


@pytest.fixture
def no_demo_env(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setenv("LEVELUP_DEMO_MODE", "false")


@pytest.fixture
async def client() -> AsyncClient:
    from apps.api.app.core.config import get_settings
    from apps.api.app.main import create_app

    get_settings.cache_clear()  # type: ignore[attr-defined]
    app = create_app()
    async with AsyncClient(transport=ASGITransport(app=app), base_url="http://test") as c:
        yield c


# ===========================================================================
# 4.2 CSRF — require_same_origin
# ===========================================================================


def _make_request(headers: dict | None = None) -> object:
    """Crée un mock minimal de Request avec les headers donnés."""
    from unittest.mock import MagicMock

    req = MagicMock()
    req.headers = headers or {}
    req.url.path = "/api/v1/sync/initial"
    return req


def test_csrf_allows_matching_origin(monkeypatch: pytest.MonkeyPatch) -> None:
    """Une requête avec Origin dans cors_origins n'est pas bloquée."""
    from apps.api.app.core.config import get_settings
    from apps.api.app.core.csrf import require_same_origin

    get_settings.cache_clear()  # type: ignore[attr-defined]
    monkeypatch.setenv("LEVELUP_CORS_ORIGINS", '["http://localhost:5173"]')
    get_settings.cache_clear()  # type: ignore[attr-defined]

    req = _make_request({"origin": "http://localhost:5173"})
    require_same_origin(req)  # type: ignore[arg-type]  # ne doit pas lever

    get_settings.cache_clear()  # type: ignore[attr-defined]


def test_csrf_blocks_foreign_origin(monkeypatch: pytest.MonkeyPatch) -> None:
    """Une origine inconnue lève ApiError 403."""
    from apps.api.app.core.config import get_settings
    from apps.api.app.core.csrf import require_same_origin
    from apps.api.app.core.errors import ApiError

    get_settings.cache_clear()  # type: ignore[attr-defined]
    monkeypatch.setenv("LEVELUP_CORS_ORIGINS", '["http://localhost:5173"]')
    get_settings.cache_clear()  # type: ignore[attr-defined]

    req = _make_request({"origin": "https://evil.example.com"})
    with pytest.raises(ApiError) as exc_info:
        require_same_origin(req)  # type: ignore[arg-type]

    assert exc_info.value.status_code == 403
    assert exc_info.value.code == "csrf_origin_mismatch"

    get_settings.cache_clear()  # type: ignore[attr-defined]


def test_csrf_allows_no_origin(monkeypatch: pytest.MonkeyPatch) -> None:
    """Absence d'Origin (appel CLI/serveur) est autorisée."""
    from apps.api.app.core.config import get_settings
    from apps.api.app.core.csrf import require_same_origin

    get_settings.cache_clear()  # type: ignore[attr-defined]
    req = _make_request({})
    require_same_origin(req)  # type: ignore[arg-type]  # ne doit pas lever
    get_settings.cache_clear()  # type: ignore[attr-defined]


def test_csrf_allows_referer_origin(monkeypatch: pytest.MonkeyPatch) -> None:
    """Referer est utilisé en fallback si Origin est absent."""
    from apps.api.app.core.config import get_settings
    from apps.api.app.core.csrf import require_same_origin

    get_settings.cache_clear()  # type: ignore[attr-defined]
    monkeypatch.setenv("LEVELUP_CORS_ORIGINS", '["http://localhost:5173"]')
    get_settings.cache_clear()  # type: ignore[attr-defined]

    req = _make_request({"referer": "http://localhost:5173/setup"})
    require_same_origin(req)  # type: ignore[arg-type]  # ne doit pas lever

    get_settings.cache_clear()  # type: ignore[attr-defined]


# ===========================================================================
# 4.3 Rate limiting
# ===========================================================================


def test_rate_limit_allows_within_limit() -> None:
    """4 requêtes consécutives d'une même IP réussissent (limite = 5)."""
    from apps.api.app.core import rate_limit

    with rate_limit._lock:
        rate_limit._windows.clear()

    for _ in range(4):
        assert not rate_limit._is_rate_limited("1.2.3.4")


def test_rate_limit_blocks_over_limit() -> None:
    """La 6ème requête d'une même IP est bloquée."""
    from apps.api.app.core import rate_limit

    with rate_limit._lock:
        rate_limit._windows.clear()

    for _ in range(5):
        rate_limit._is_rate_limited("5.6.7.8")

    assert rate_limit._is_rate_limited("5.6.7.8")


def test_rate_limit_independent_ips() -> None:
    """Deux IPs différentes ont des compteurs indépendants."""
    from apps.api.app.core import rate_limit

    with rate_limit._lock:
        rate_limit._windows.clear()

    for _ in range(5):
        rate_limit._is_rate_limited("10.0.0.1")

    assert rate_limit._is_rate_limited("10.0.0.1")
    assert not rate_limit._is_rate_limited("10.0.0.2")


# ===========================================================================
# 4.1 — Config trusted_proxies
# ===========================================================================


def test_trusted_proxy_list_parses_correctly() -> None:
    """trusted_proxy_list retourne la liste parsée depuis la chaîne."""
    import os

    from apps.api.app.core.config import get_settings

    get_settings.cache_clear()  # type: ignore[attr-defined]
    os.environ["LEVELUP_TRUSTED_PROXIES"] = "10.0.0.1, 10.0.0.2"
    get_settings.cache_clear()  # type: ignore[attr-defined]
    settings = get_settings()
    assert settings.trusted_proxy_list == ["10.0.0.1", "10.0.0.2"]
    del os.environ["LEVELUP_TRUSTED_PROXIES"]
    get_settings.cache_clear()  # type: ignore[attr-defined]


def test_is_production_default_false() -> None:
    """is_production retourne False dans le contexte de test (LEVELUP_ENV non défini)."""
    import os

    from apps.api.app.core.config import get_settings

    os.environ.pop("LEVELUP_ENV", None)
    get_settings.cache_clear()  # type: ignore[attr-defined]
    settings = get_settings()
    assert settings.is_production is False
    get_settings.cache_clear()  # type: ignore[attr-defined]
