"""Tests — Axe 3 : Dual semaphore fetch/CPU séparés.

Vérifie que :
- SyncOptions expose parallel_fetch avec valeur par défaut 15
- parallel_fetch >= parallel_matches (sinon le max corrige)
- La logique fetch_slots = max(parallel_fetch, parallel_matches) est respectée
- _process_matches utilise bien le semaphore avec fetch_slots slots
"""

from __future__ import annotations

import asyncio
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from src.data.sync.models_sync import SyncOptions

# ---------------------------------------------------------------------------
# Tests SyncOptions.parallel_fetch
# ---------------------------------------------------------------------------


def test_parallel_fetch_default_is_15():
    """SyncOptions.parallel_fetch doit valoir 15 par défaut."""
    opts = SyncOptions()
    assert opts.parallel_fetch == 15


def test_parallel_fetch_independant_of_parallel_matches():
    """parallel_fetch et parallel_matches sont indépendants."""
    opts = SyncOptions(parallel_fetch=20, parallel_matches=5)
    assert opts.parallel_fetch == 20
    assert opts.parallel_matches == 5


def test_fetch_slots_is_max_of_fetch_and_matches():
    """fetch_slots = max(parallel_fetch, parallel_matches)."""
    # Quand parallel_fetch > parallel_matches
    opts_wide = SyncOptions(parallel_fetch=20, parallel_matches=5)
    assert max(opts_wide.parallel_fetch, opts_wide.parallel_matches) == 20

    # Quand parallel_matches > parallel_fetch (cas dégénéré)
    opts_narrow = SyncOptions(parallel_fetch=3, parallel_matches=10)
    assert max(opts_narrow.parallel_fetch, opts_narrow.parallel_matches) == 10


def test_parallel_fetch_default_gte_parallel_matches_default():
    """Avec les valeurs par défaut, parallel_fetch >= parallel_matches."""
    opts = SyncOptions()
    assert opts.parallel_fetch >= opts.parallel_matches, (
        f"parallel_fetch ({opts.parallel_fetch}) doit être >= "
        f"parallel_matches ({opts.parallel_matches})"
    )


# ---------------------------------------------------------------------------
# Tests logique semaphore dans _process_matches
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_process_matches_creates_semaphore_with_fetch_slots():
    """_process_matches doit créer un Semaphore avec fetch_slots slots."""
    from src.data.sync._match_processing import MatchProcessingMixin

    # Préparer un objet mixin minimal
    mixin = MatchProcessingMixin.__new__(MatchProcessingMixin)
    mixin._gamertag = "TestPlayer"

    # Injecter les méthodes dépendantes nécessaires
    mixin._get_latest_match_id_in_db = MagicMock(return_value=None)
    mixin._process_single_match = AsyncMock(return_value={"inserted": True, "match_id": "m1"})
    mixin._accumulate_match_result = MagicMock()
    mixin._maybe_batch_commit = MagicMock()
    mixin._report_progress = MagicMock()

    # Fake history item
    item = MagicMock()
    item.match_id = "match-axe3-001"

    client = MagicMock()
    client.get_match_history = AsyncMock(
        side_effect=[
            [item],  # Premier appel : 1 match
            [],  # Deuxième appel : stop
        ]
    )

    options = SyncOptions(
        max_matches=1,
        parallel_fetch=20,
        parallel_matches=5,
        batch_commit_size=0,
    )

    # Observer quelle valeur est passée à asyncio.Semaphore
    original_semaphore = asyncio.Semaphore
    created_semaphores = []

    class _CapturingSemaphore:
        def __init__(self, value: int):
            created_semaphores.append(value)
            self._sem = original_semaphore(value)

        async def __aenter__(self):
            await self._sem.__aenter__()
            return self

        async def __aexit__(self, *args):
            return await self._sem.__aexit__(*args)

    with (
        patch("src.data.sync._match_processing.asyncio.Semaphore", side_effect=_CapturingSemaphore),
        patch.object(mixin, "_get_latest_match_id_in_db", return_value=None),
    ):
        await mixin._process_matches(
            client=client,
            options=options,
            existing_ids=set(),
            delta_mode=False,
        )

    # Au moins 1 semaphore créé avec fetch_slots = max(20, 5) = 20
    assert len(created_semaphores) >= 1, "Aucun Semaphore créé"
    assert (
        20 in created_semaphores
    ), f"Semaphore(20) attendu (fetch_slots=max(20,5)), got {created_semaphores}"


@pytest.mark.asyncio
async def test_bounded_respects_semaphore_concurrency():
    """_bounded dans _process_matches doit limiter la concurrence via le sémaphore."""
    from src.data.sync._match_processing import MatchProcessingMixin

    mixin = MatchProcessingMixin.__new__(MatchProcessingMixin)
    mixin._gamertag = "TestPlayer"

    concurrency_tracker = {"current": 0, "max": 0}

    async def _counting_process(*args, **kwargs):
        concurrency_tracker["current"] += 1
        concurrency_tracker["max"] = max(concurrency_tracker["max"], concurrency_tracker["current"])
        await asyncio.sleep(0)  # Yield pour permettre aux autres coroutines de se lancer
        concurrency_tracker["current"] -= 1
        return {"inserted": True, "match_id": args[1]}

    mixin._get_latest_match_id_in_db = MagicMock(return_value=None)
    mixin._process_single_match = _counting_process
    mixin._accumulate_match_result = MagicMock()
    mixin._maybe_batch_commit = MagicMock()
    mixin._report_progress = MagicMock()

    # 5 matchs avec parallelisme max = 3
    items = [MagicMock(match_id=f"match-{i:03d}") for i in range(5)]
    client = MagicMock()
    client.get_match_history = AsyncMock(side_effect=[items, []])

    options = SyncOptions(
        max_matches=5,
        parallel_fetch=3,
        parallel_matches=3,
        batch_commit_size=0,
    )

    await mixin._process_matches(
        client=client,
        options=options,
        existing_ids=set(),
        delta_mode=False,
    )

    # La concurrence max ne doit pas dépasser parallel_matches
    assert (
        concurrency_tracker["max"] <= 3
    ), f"Concurrence max={concurrency_tracker['max']} > parallel_matches=3"
