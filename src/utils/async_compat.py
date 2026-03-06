"""Compatibilité async → sync pour Streamlit.

Centralise le pattern ``_run_sync`` copié dans plusieurs modules UI.
"""

from __future__ import annotations

import asyncio
import concurrent.futures
from typing import Any, TypeVar

T = TypeVar("T")


def run_sync(coro: Any, *, timeout: float = 50.0) -> T:  # type: ignore[type-var]
    """Exécute une coroutine de manière synchrone, compatible Streamlit.

    Tente ``asyncio.run()`` d'abord, puis utilise un
    ``ThreadPoolExecutor`` si une event loop est déjà active.

    Args:
        coro: Coroutine à exécuter.
        timeout: Timeout total en secondes pour le fallback threadpool.

    Returns:
        Le résultat de la coroutine.
    """
    try:
        return asyncio.run(coro)
    except RuntimeError as exc:
        msg = str(exc)
        if "asyncio.run() cannot be called" not in msg:
            raise
        with concurrent.futures.ThreadPoolExecutor(max_workers=1) as ex:
            fut = ex.submit(lambda: asyncio.run(coro))
            return fut.result(timeout=timeout)
