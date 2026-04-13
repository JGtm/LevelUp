"""Configuration pytest pour tests/api/."""

from __future__ import annotations

import pytest


@pytest.fixture(autouse=True)
def _reset_rate_limiter() -> None:
    """Réinitialise les fenêtres glissantes du rate limiter entre chaque test.

    Le store du rate limiter est process-level ; sans reset, les entrées
    s'accumulent entre tests et déclenchent des faux 429.
    """
    from apps.api.app.core.rate_limit import _lock, _windows

    with _lock:
        _windows.clear()
    yield
    with _lock:
        _windows.clear()
