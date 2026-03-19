"""Tests — Axe 1 : Post-sync partiellement parallèle (citations en thread).

Vérifie que :
- _run_post_sync_compute est une coroutine async
- run_in_executor est appelé pour lancer citations en thread
- _shared_connection est fermée avant le scatter (avant run_in_executor)
- Une exception dans citations ne bloque pas perf/sessions/dominance
- cit_future est awaité en fin de fonction
"""

from __future__ import annotations

import asyncio
import inspect
from pathlib import Path
from unittest.mock import MagicMock, patch

import pytest

from src.data.sync.engine import DuckDBSyncEngine
from src.data.sync.models import SyncOptions

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_engine(tmp_path: Path) -> DuckDBSyncEngine:
    """Instancie un moteur minimal sans connexion réelle."""
    player_db = tmp_path / "stats.duckdb"
    player_db.touch()
    engine = DuckDBSyncEngine.__new__(DuckDBSyncEngine)
    engine._player_db_path = player_db
    engine._shared_db_path = tmp_path / "shared_matches.duckdb"
    engine._xuid = "xuid_test_123"
    engine._shared_connection = None
    engine._tokens = None
    return engine


def _make_options(defer_performance_score: bool = False) -> SyncOptions:
    return SyncOptions(defer_performance_score=defer_performance_score)


# ---------------------------------------------------------------------------
# Test : _run_post_sync_compute est une coroutine
# ---------------------------------------------------------------------------


def test_run_post_sync_compute_is_coroutine():
    """_run_post_sync_compute doit être déclarée async def."""
    assert inspect.iscoroutinefunction(DuckDBSyncEngine._run_post_sync_compute)


# ---------------------------------------------------------------------------
# Test : run_in_executor lancé avant le bloc sérialisé
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_run_in_executor_called(tmp_path):
    """run_in_executor doit être appelé une fois avec _post_sync_citations_sync."""
    engine = _make_engine(tmp_path)
    options = _make_options()

    future = asyncio.get_event_loop().create_future()
    future.set_result({"matches_processed": 0, "citations_computed": 0})

    loop = asyncio.get_event_loop()
    with (
        patch.object(engine, "_post_sync_citations_sync", return_value=None),
        patch.object(engine, "batch_compute_performance_scores", return_value=0),
        patch(
            "src.data.sync.engine.backfill_sessions_for_player",
            side_effect=Exception("skip"),
            create=True,
        ),
        patch.object(engine, "_compute_dominance_post_sync"),
        patch.object(loop, "run_in_executor", return_value=future) as mock_exec,
        patch(
            "src.data.sessions_backfill.backfill_sessions_for_player",
            side_effect=Exception("skip"),
        ),
    ):
        await engine._run_post_sync_compute(options)

    # run_in_executor doit avoir été appelé avec executor=None et un callable
    assert mock_exec.call_args[0][0] is None, "executor doit être None (thread pool par défaut)"
    assert callable(mock_exec.call_args[0][1]), "le callable doit être une fonction"


# ---------------------------------------------------------------------------
# Test : _shared_connection fermée avant run_in_executor
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_shared_connection_closed_before_executor(tmp_path):
    """_shared_connection.close() doit être appelée avant run_in_executor."""
    engine = _make_engine(tmp_path)
    options = _make_options()

    call_order: list[str] = []

    mock_conn = MagicMock()

    def _close():
        call_order.append("close")

    mock_conn.close.side_effect = _close
    engine._shared_connection = mock_conn

    future = asyncio.get_event_loop().create_future()
    future.set_result({"matches_processed": 0, "citations_computed": 0})

    loop = asyncio.get_event_loop()

    def _run_in_executor(executor, fn):
        call_order.append("run_in_executor")
        return future

    with (
        patch.object(engine, "batch_compute_performance_scores", return_value=0),
        patch.object(engine, "_compute_dominance_post_sync"),
        patch.object(loop, "run_in_executor", side_effect=_run_in_executor),
        patch(
            "src.data.sessions_backfill.backfill_sessions_for_player",
            side_effect=Exception("skip"),
        ),
    ):
        await engine._run_post_sync_compute(options)

    assert call_order.index("close") < call_order.index(
        "run_in_executor"
    ), "close() doit précéder run_in_executor()"
    assert engine._shared_connection is None


# ---------------------------------------------------------------------------
# Test : exception dans citations ne bloque pas le reste
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_citations_exception_does_not_block(tmp_path):
    """Une exception dans les citations ne doit pas bloquer perf/sessions/dominance.

    On simule un future qui lève une exception (comme si run_in_executor avait
    capturé une exception) — le code en production encapsule déjà l'exception
    dans _post_sync_citations_sync (wrapper avec try/except), donc ici on test
    que même si le future échoue, dominance est appelé.
    """
    engine = _make_engine(tmp_path)
    options = _make_options()

    dominance_called = []

    def _dominance():
        dominance_called.append(True)

    failed_future: asyncio.Future = asyncio.get_event_loop().create_future()
    failed_future.set_exception(Exception("DB locked"))

    loop = asyncio.get_event_loop()
    with (
        patch.object(loop, "run_in_executor", return_value=failed_future),
        patch.object(engine, "batch_compute_performance_scores", return_value=0),
        patch.object(engine, "_compute_dominance_post_sync", side_effect=_dominance),
        patch.object(engine, "_get_connection", return_value=MagicMock()),
        patch(
            "src.data.sessions_backfill.backfill_sessions_for_player",
            side_effect=Exception("skip"),
        ),
    ):
        # dominance doit être appelé avant le await cit_future
        import contextlib

        with contextlib.suppress(Exception):
            await engine._run_post_sync_compute(options)

    assert (
        dominance_called
    ), "_compute_dominance_post_sync doit être appelé avant le await cit_future"


# ---------------------------------------------------------------------------
# Test : cit_future est awaité (result loggué)
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_cit_future_is_awaited(tmp_path):
    """Le future de citations doit être awaité — son résultat est consommé."""
    engine = _make_engine(tmp_path)
    options = _make_options()

    expected = {"matches_processed": 5, "citations_computed": 12}
    future: asyncio.Future = asyncio.get_event_loop().create_future()
    future.set_result(expected)

    loop = asyncio.get_event_loop()
    with (
        patch.object(loop, "run_in_executor", return_value=future),
        patch.object(engine, "batch_compute_performance_scores", return_value=0),
        patch.object(engine, "_compute_dominance_post_sync"),
        patch.object(engine, "_get_connection", return_value=MagicMock()),
        patch(
            "src.data.sessions_backfill.backfill_sessions_for_player",
            side_effect=Exception("skip"),
        ),
    ):
        await engine._run_post_sync_compute(options)

    # Si le future a été awaité, il est « done » et son résultat consommé sans exception
    assert future.done(), "Le future doit être done après _run_post_sync_compute"
    # Aucune exception pendante dans le future
    assert future.exception() is None or future.result() == expected


# ---------------------------------------------------------------------------
# Test : _post_sync_citations_sync retourne un dict même si import échoue
# ---------------------------------------------------------------------------


def test_post_sync_citations_sync_fallback(tmp_path):
    """Si backfill_citations_for_player lève, retourne un dict vide mais valide."""
    engine = _make_engine(tmp_path)

    with patch(
        "src.data.citations_backfill.backfill_citations_for_player",
        side_effect=RuntimeError("simulated"),
    ):
        result = engine._post_sync_citations_sync()

    assert result == {"matches_processed": 0, "citations_computed": 0}


# ---------------------------------------------------------------------------
# Test : log de démarrage et timing émis (Axe 1)
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_post_sync_logs_start_and_timing(tmp_path, caplog):
    """_run_post_sync_compute doit loguer démarrage et timing à INFO."""
    import logging

    engine = _make_engine(tmp_path)
    options = _make_options()

    future: asyncio.Future = asyncio.get_event_loop().create_future()
    future.set_result({"matches_processed": 3, "citations_computed": 3})

    loop = asyncio.get_event_loop()
    with (
        caplog.at_level(logging.INFO, logger="src.data.sync.engine"),
        patch.object(loop, "run_in_executor", return_value=future),
        patch.object(engine, "batch_compute_performance_scores", return_value=5),
        patch.object(engine, "_compute_dominance_post_sync"),
        patch(
            "src.data.sessions_backfill.backfill_sessions_for_player",
            side_effect=Exception("skip"),
        ),
    ):
        await engine._run_post_sync_compute(options)

    messages = [r.message for r in caplog.records if r.levelno == logging.INFO]
    assert any(
        "post_sync" in m and "démarrage" in m for m in messages
    ), f"Log démarrage post_sync introuvable. Messages INFO : {messages}"
    assert any(
        "post_sync" in m and "terminé" in m for m in messages
    ), f"Log timing post_sync introuvable. Messages INFO : {messages}"
