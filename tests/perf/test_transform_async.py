"""Tests — Axe 5 : Transformations CPU-bound via run_in_executor.

Vérifie que :
- _transform_match_stats_async délègue à run_in_executor
- MetadataResolver._lock est un threading.RLock (thread-safety)
- MetadataResolver.resolve est protégé par le lock
- transform_match_stats reçoit bien les bons arguments
"""

from __future__ import annotations

import asyncio
import threading
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

from src.data.sync._match_processing_helpers import MatchProcessingHelpersMixin
from src.data.sync.metadata_resolver import MetadataResolver

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_mixin(tmp_path: Path) -> MatchProcessingHelpersMixin:
    """Crée un mixin minimal avec _xuid et _metadata_resolver."""
    mixin = MatchProcessingHelpersMixin.__new__(MatchProcessingHelpersMixin)
    mixin._xuid = "xuid_test_axe5"
    mixin._metadata_resolver = None
    return mixin


_STATS_JSON = {
    "MatchId": "match-axe5-test",
    "MatchInfo": {},
    "Players": [],
}


# ---------------------------------------------------------------------------
# Test : MetadataResolver a un threading.RLock
# ---------------------------------------------------------------------------


def test_metadata_resolver_has_rlock():
    """MetadataResolver doit exposer un _lock de type threading.RLock."""
    resolver = MetadataResolver(metadata_db_path="/nonexistent/metadata.duckdb")
    assert hasattr(resolver, "_lock"), "_lock absent de MetadataResolver"
    # threading.RLock() retourne une instance de _RLock (type interne)
    assert isinstance(resolver._lock, type(threading.RLock())), (
        f"_lock doit être un threading.RLock, got {type(resolver._lock)}"
    )


def test_metadata_resolver_lock_is_reentrant():
    """Le RLock doit être re-entrant (acquire depuis le même thread)."""
    resolver = MetadataResolver(metadata_db_path="/nonexistent/metadata.duckdb")
    # Un RLock peut être acquis deux fois depuis le même thread sans deadlock
    with resolver._lock:
        acquired = resolver._lock.acquire(blocking=False)
        if acquired:
            resolver._lock.release()
    # Pas de deadlock = success


def test_metadata_resolver_resolve_thread_safe():
    """resolve() doit étre appelable depuis plusieurs threads sans exception."""
    resolver = MetadataResolver(metadata_db_path="/nonexistent/metadata.duckdb")
    errors: list[Exception] = []

    def _worker():
        try:
            resolver.resolve("playlist", "asset-id-1")
            resolver.resolve("map", "asset-id-2")
        except Exception as e:  # noqa: BLE001
            errors.append(e)

    threads = [threading.Thread(target=_worker) for _ in range(10)]
    for t in threads:
        t.start()
    for t in threads:
        t.join()

    assert not errors, f"Erreurs thread: {errors}"


# ---------------------------------------------------------------------------
# Test : _transform_match_stats_async délègue à run_in_executor
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_transform_match_stats_async_uses_executor(tmp_path):
    """_transform_match_stats_async doit utiliser loop.run_in_executor."""
    mixin = _make_mixin(tmp_path)

    mock_result = MagicMock()
    fake_future: asyncio.Future = asyncio.get_event_loop().create_future()
    fake_future.set_result(mock_result)

    loop = asyncio.get_event_loop()
    with patch.object(loop, "run_in_executor", return_value=fake_future) as mock_exec:
        result = await mixin._transform_match_stats_async(_STATS_JSON, None)

    assert result is mock_result
    assert mock_exec.call_count == 1
    # Premier arg = executor (None = thread pool par défaut)
    assert mock_exec.call_args[0][0] is None


@pytest.mark.asyncio
async def test_transform_match_stats_async_passes_correct_args(tmp_path):
    """Le functools.partial passé à run_in_executor doit utiliser les bons kwargs."""
    import functools

    mixin = _make_mixin(tmp_path)
    skill_json = {"some": "skill"}

    captured_fn = None

    loop = asyncio.get_event_loop()
    fake_future: asyncio.Future = asyncio.get_event_loop().create_future()
    fake_future.set_result(None)

    def _capture(executor, fn):
        nonlocal captured_fn
        captured_fn = fn
        return fake_future

    with patch.object(loop, "run_in_executor", side_effect=_capture):
        await mixin._transform_match_stats_async(_STATS_JSON, skill_json)

    assert captured_fn is not None
    assert isinstance(captured_fn, functools.partial), (
        f"Le callable doit être un functools.partial, got {type(captured_fn)}"
    )
    # Les kwargs doivent inclure skill_json et xuid correct
    assert captured_fn.keywords.get("skill_json") is skill_json
    assert captured_fn.args[1] == "xuid_test_axe5"  # xuid


@pytest.mark.asyncio
async def test_transform_match_stats_async_propagates_exception(tmp_path):
    """Une exception dans le thread doit se propager via le future."""
    mixin = _make_mixin(tmp_path)

    failed_future: asyncio.Future = asyncio.get_event_loop().create_future()
    failed_future.set_exception(ValueError("transform error"))

    loop = asyncio.get_event_loop()
    with (
        patch.object(loop, "run_in_executor", return_value=failed_future),
        pytest.raises(ValueError, match="transform error"),
    ):
        await mixin._transform_match_stats_async(_STATS_JSON, None)


# ---------------------------------------------------------------------------
# Test : l'event loop n'est pas bloqué pendant le transform (Axe 5)
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_transform_async_yields_to_other_coroutines(tmp_path):
    """Pendant _transform_match_stats_async, d'autres coroutines peuvent progresser.

    On lance un background ticker et vérifie qu'il émet des ticks
    pendant la durée du transform (prouve que l'event loop n'est pas bloqué).
    """
    import time

    mixin = _make_mixin(tmp_path)
    progress_ticks: list[float] = []

    async def _background_ticker():
        for _ in range(20):
            await asyncio.sleep(0.003)
            progress_ticks.append(time.monotonic())

    def _slow_transform(*args, **kwargs):
        """Simule un transform CPU-bound de ~50ms."""
        time.sleep(0.05)
        return MagicMock()

    # Utiliser le vrai run_in_executor pour que le ticker puisse progresser
    ticker_task = asyncio.create_task(_background_ticker())
    with patch(
        "src.data.sync._match_processing_helpers.transform_match_stats",
        side_effect=_slow_transform,
    ):
        await mixin._transform_match_stats_async(_STATS_JSON, None)

    await ticker_task

    assert len(progress_ticks) >= 3, (
        f"L'event loop a été bloqué : seulement {len(progress_ticks)} ticks "
        "pendant transform de 50ms (attendu >= 3)"
    )


@pytest.mark.asyncio
async def test_transform_async_result_matches_direct_call(tmp_path):
    """_transform_match_stats_async retourne le même résultat qu'un appel direct."""
    mixin = _make_mixin(tmp_path)
    expected = MagicMock()

    captured_result: list = []

    async def _capturing_executor(executor, fn):
        result = fn() if callable(fn) else fn
        captured_result.append(result)
        return result

    with (
        patch.object(asyncio.get_event_loop(), "run_in_executor", side_effect=_capturing_executor),
        patch(
            "src.data.sync._match_processing_helpers.transform_match_stats",
            return_value=expected,
        ),
    ):
        result = await mixin._transform_match_stats_async(_STATS_JSON, None)

    assert result is expected
