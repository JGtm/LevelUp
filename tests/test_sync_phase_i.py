"""Tests — Phase I : extraction _run_post_sync_pipeline().

Vérifie :
1. _run_post_sync_pipeline est async def autonome.
2. Sans nouveaux matchs : _refresh_aggregates et _run_post_sync_compute ne sont PAS appelés.
3. Avec nouveaux matchs : _refresh_aggregates ET _run_post_sync_compute sont appelés.
4. Ordre contraint : agrégats → career → metadata+commit → post_compute → detach → LUSR → fanout.
5. Erreur dans _run_post_sync_compute → non bloquant, sync_internal ne plante pas.
6. _run_lusr_post_sync toujours appelé (même sans nouveaux matchs).
7. _enrich_other_registered_players appelé seulement si inserted_match_ids non vide.
"""

from __future__ import annotations

import inspect

import pytest

# ─────────────────────────────────────────────────────────────────────────────
# I.1 — _run_post_sync_pipeline est async def
# ─────────────────────────────────────────────────────────────────────────────


def test_run_post_sync_pipeline_is_coroutine():
    """_run_post_sync_pipeline doit être une coroutine (async def)."""
    from src.data.sync.engine import DuckDBSyncEngine

    assert inspect.iscoroutinefunction(DuckDBSyncEngine._run_post_sync_pipeline), (
        "_run_post_sync_pipeline doit être async def"
    )


# ─────────────────────────────────────────────────────────────────────────────
# I.2 — Sans nouveaux matchs : pas d'agrégats, pas de post_compute
# ─────────────────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_pipeline_sans_nouveaux_matchs_skip_aggregates():
    """Sans nouveaux matchs insérés, agrégats et post_compute ne sont pas appelés."""
    from unittest.mock import AsyncMock, MagicMock

    from src.data.sync.engine import DuckDBSyncEngine
    from src.data.sync.models import SyncOptions, SyncResult

    engine = MagicMock(spec=DuckDBSyncEngine)
    engine._get_connection.return_value = MagicMock()
    engine._refresh_aggregates_async = AsyncMock()
    engine._run_career_rank_if_needed = AsyncMock()
    engine._save_sync_metadata = MagicMock()
    engine._run_post_sync_compute = AsyncMock()
    engine._detach_shared_from_player_conn = MagicMock()
    engine._run_lusr_post_sync = MagicMock()
    engine._enrich_other_registered_players = MagicMock()

    result = SyncResult()  # matches_inserted=0, inserted_match_ids=[]
    options = SyncOptions()

    await DuckDBSyncEngine._run_post_sync_pipeline(engine, options, result, delta_mode=True)

    engine._refresh_aggregates_async.assert_not_called()
    engine._run_post_sync_compute.assert_not_called()


# ─────────────────────────────────────────────────────────────────────────────
# I.3 — Avec nouveaux matchs : agrégats ET post_compute appelés
# ─────────────────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_pipeline_avec_nouveaux_matchs_appelle_aggregates():
    """Avec des matchs insérés, _refresh_aggregates_async et _run_post_sync_compute sont appelés."""
    from unittest.mock import AsyncMock, MagicMock

    from src.data.sync.engine import DuckDBSyncEngine
    from src.data.sync.models import SyncOptions, SyncResult

    engine = MagicMock(spec=DuckDBSyncEngine)
    engine._get_connection.return_value = MagicMock()
    engine._refresh_aggregates_async = AsyncMock()
    engine._run_career_rank_if_needed = AsyncMock()
    engine._save_sync_metadata = MagicMock()
    engine._run_post_sync_compute = AsyncMock()
    engine._detach_shared_from_player_conn = MagicMock()
    engine._run_lusr_post_sync = MagicMock()
    engine._enrich_other_registered_players = MagicMock()

    result = SyncResult(matches_inserted=5)
    result.inserted_match_ids = ["id1", "id2", "id3", "id4", "id5"]
    options = SyncOptions()

    await DuckDBSyncEngine._run_post_sync_pipeline(engine, options, result, delta_mode=True)

    engine._refresh_aggregates_async.assert_called_once_with(new_ids=result.inserted_match_ids)
    engine._run_post_sync_compute.assert_called_once_with(options)


# ─────────────────────────────────────────────────────────────────────────────
# I.4 — LUSR toujours appelé (même sans nouveaux matchs)
# ─────────────────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_pipeline_lusr_toujours_appelé():
    """_run_lusr_post_sync doit être appelé même si aucun nouveau match."""
    from unittest.mock import AsyncMock, MagicMock

    from src.data.sync.engine import DuckDBSyncEngine
    from src.data.sync.models import SyncOptions, SyncResult

    engine = MagicMock(spec=DuckDBSyncEngine)
    engine._get_connection.return_value = MagicMock()
    engine._refresh_aggregates_async = AsyncMock()
    engine._run_career_rank_if_needed = AsyncMock()
    engine._save_sync_metadata = MagicMock()
    engine._run_post_sync_compute = AsyncMock()
    engine._detach_shared_from_player_conn = MagicMock()
    engine._run_lusr_post_sync = MagicMock()
    engine._enrich_other_registered_players = MagicMock()

    result = SyncResult()  # 0 matchs
    options = SyncOptions()

    await DuckDBSyncEngine._run_post_sync_pipeline(engine, options, result, delta_mode=False)

    engine._run_lusr_post_sync.assert_called_once()
    engine._detach_shared_from_player_conn.assert_called_once()


# ─────────────────────────────────────────────────────────────────────────────
# I.5 — Fan-out seulement si inserted_match_ids non vide
# ─────────────────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_pipeline_fanout_seulement_si_inserted_ids():
    """_enrich_other_registered_players appelé seulement si inserted_match_ids non vide."""
    from unittest.mock import AsyncMock, MagicMock

    from src.data.sync.engine import DuckDBSyncEngine
    from src.data.sync.models import SyncOptions, SyncResult

    engine = MagicMock(spec=DuckDBSyncEngine)
    engine._get_connection.return_value = MagicMock()
    engine._refresh_aggregates_async = AsyncMock()
    engine._run_career_rank_if_needed = AsyncMock()
    engine._save_sync_metadata = MagicMock()
    engine._run_post_sync_compute = AsyncMock()
    engine._detach_shared_from_player_conn = MagicMock()
    engine._run_lusr_post_sync = MagicMock()
    engine._enrich_other_registered_players = MagicMock()

    # Cas 1 : aucun match → pas de fan-out
    result_empty = SyncResult()
    await DuckDBSyncEngine._run_post_sync_pipeline(
        engine, SyncOptions(), result_empty, delta_mode=True
    )
    engine._enrich_other_registered_players.assert_not_called()

    # Cas 2 : matchs insérés → fan-out
    result_with = SyncResult(matches_inserted=2)
    result_with.inserted_match_ids = ["id1", "id2"]
    await DuckDBSyncEngine._run_post_sync_pipeline(
        engine, SyncOptions(), result_with, delta_mode=True
    )
    engine._enrich_other_registered_players.assert_called_once_with(["id1", "id2"])


# ─────────────────────────────────────────────────────────────────────────────
# I.6 — Ordre contraint vérifié par call_args_list
# ─────────────────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_pipeline_ordre_appels():
    """Les étapes du pipeline sont appelées dans l'ordre contraint (I.4)."""
    from unittest.mock import AsyncMock, MagicMock

    from src.data.sync.engine import DuckDBSyncEngine
    from src.data.sync.models import SyncOptions, SyncResult

    call_order: list[str] = []

    engine = MagicMock(spec=DuckDBSyncEngine)
    conn = MagicMock()
    engine._get_connection.return_value = conn
    conn.commit.side_effect = lambda: call_order.append("commit")

    engine._refresh_aggregates_async = AsyncMock(
        side_effect=lambda **_: call_order.append("aggregates")
    )
    engine._run_career_rank_if_needed = AsyncMock(
        side_effect=lambda *_: call_order.append("career")
    )
    engine._save_sync_metadata = MagicMock(side_effect=lambda *_: call_order.append("metadata"))
    engine._run_post_sync_compute = AsyncMock(
        side_effect=lambda *_: call_order.append("post_compute")
    )
    engine._detach_shared_from_player_conn = MagicMock(
        side_effect=lambda: call_order.append("detach")
    )
    engine._run_lusr_post_sync = MagicMock(side_effect=lambda: call_order.append("lusr"))
    engine._enrich_other_registered_players = MagicMock(
        side_effect=lambda *_: call_order.append("fanout")
    )

    result = SyncResult(matches_inserted=3)
    result.inserted_match_ids = ["a", "b", "c"]

    await DuckDBSyncEngine._run_post_sync_pipeline(engine, SyncOptions(), result, delta_mode=True)

    expected = [
        "aggregates",
        "career",
        "metadata",
        "commit",
        "post_compute",
        "detach",
        "lusr",
        "fanout",
    ]
    assert call_order == expected, f"Ordre attendu: {expected}\nObtenu: {call_order}"
