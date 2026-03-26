"""Tests unitaires pour le comportement de _process_matches (boucle de pagination).

Après Phase B : le HEAD check a été déplacé dans _sync_internal (avant le chargement
de existing_ids). _process_matches ne fait plus de HEAD check — il reçoit existing_ids
déjà chargé et itère directement sur l'historique.
"""

from __future__ import annotations

from unittest.mock import MagicMock

import pytest

# =============================================================================
# Helpers
# =============================================================================


def _make_engine_mock(latest_db: str | None = None) -> MagicMock:
    """Crée un mock minimaliste de DuckDBSyncEngine pour les tests de mixin."""
    engine = MagicMock()
    engine._gamertag = "TestPlayer"
    engine._xuid = "xuid_test_123"
    engine._get_latest_match_id_in_db.return_value = latest_db
    # _accumulate_match_result et _maybe_batch_commit utilisés dans la boucle
    # principale — on laisse les MagicMock par défaut (no-op).
    return engine


def _make_client(*, head_match_id: str | None = None, paginated: list | None = None):
    """Crée un mock de client API avec get_match_history configurable."""
    if paginated is None:
        paginated = []

    head_item = None
    if head_match_id is not None:
        head_item = MagicMock()
        head_item.match_id = head_match_id

    async def fake_history(gamertag, match_type=None, start=0, count=25):
        if count == 1:
            return [head_item] if head_item else []
        return paginated

    client = MagicMock()
    client.get_match_history = fake_history
    return client


# =============================================================================
# Classe de tests
# =============================================================================


@pytest.mark.asyncio
class TestEarlyExitDelta:
    """Tests pour la boucle de pagination dans MatchProcessingMixin._process_matches.

    Note Phase B : le HEAD check early-exit est maintenant dans _sync_internal.
    _process_matches reçoit existing_ids déjà chargé et itère directement.
    """

    async def test_early_exit_quand_head_egal_db(self):
        """Si HEAD API == dernier match DB → return immédiat (0 match inseré)."""
        from src.data.sync._match_processing import MatchProcessingMixin
        from src.data.sync.models_sync import SyncOptions, SyncResult

        engine = _make_engine_mock(latest_db="latest-match-abc")
        client = _make_client(head_match_id="latest-match-abc")
        options = SyncOptions(max_matches=200)
        existing_ids = {"latest-match-abc", "older-match-xyz"}

        result = await MatchProcessingMixin._process_matches(
            engine,
            client,
            options,
            existing_ids,
            delta_mode=True,
        )

        assert isinstance(result, SyncResult)
        assert result.matches_inserted == 0
        assert result.matches_skipped == 0

    async def test_early_exit_increments_zero_calls_after_head(self):
        """En cas d'early-exit, get_match_history est appelé exactement 1 fois."""
        from src.data.sync._match_processing import MatchProcessingMixin
        from src.data.sync.models_sync import SyncOptions

        call_count = 0

        async def counting_history(gamertag, match_type=None, start=0, count=25):
            nonlocal call_count
            call_count += 1
            if count == 1:
                item = MagicMock()
                item.match_id = "head-match"
                return [item]
            return []

        engine = _make_engine_mock(latest_db="head-match")
        client = MagicMock()
        client.get_match_history = counting_history
        options = SyncOptions()
        existing_ids = {"head-match"}

        await MatchProcessingMixin._process_matches(
            engine,
            client,
            options,
            existing_ids,
            delta_mode=True,
        )

        # Seul le HEAD check (count=1) doit avoir été appelé
        assert call_count == 1

    async def test_no_early_exit_quand_head_different_de_db(self):
        """_process_matches ne fait pas de HEAD check — pagination directe.

        Phase B : le HEAD check est dans _sync_internal. _process_matches
        reçoit existing_ids et fait la pagination normalement.
        """
        from src.data.sync._match_processing import MatchProcessingMixin
        from src.data.sync.models_sync import SyncOptions

        call_count = 0

        async def counting_history(gamertag, match_type=None, start=0, count=25):
            nonlocal call_count
            call_count += 1
            # count=1 n'est plus appelé depuis _process_matches
            return []

        engine = _make_engine_mock(latest_db="old-match-abc")
        client = MagicMock()
        client.get_match_history = counting_history
        options = SyncOptions()
        existing_ids = {"old-match-abc"}

        await MatchProcessingMixin._process_matches(
            engine,
            client,
            options,
            existing_ids,
            delta_mode=True,
        )

        # _process_matches fait au moins 1 appel de pagination (pas de HEAD check count=1)
        assert call_count >= 1

    async def test_no_early_exit_quand_existing_ids_vide(self):
        """Si existing_ids est vide → pas de HEAD check (condition `and existing_ids`)."""
        from src.data.sync._match_processing import MatchProcessingMixin
        from src.data.sync.models_sync import SyncOptions

        head_check_called = False

        async def tracking_history(gamertag, match_type=None, start=0, count=25):
            nonlocal head_check_called
            if count == 1:
                head_check_called = True
                item = MagicMock()
                item.match_id = "some-match"
                return [item]
            return []

        engine = _make_engine_mock(latest_db="some-match")
        client = MagicMock()
        client.get_match_history = tracking_history
        options = SyncOptions()
        existing_ids: set[str] = set()  # Vide → pas de HEAD check

        await MatchProcessingMixin._process_matches(
            engine,
            client,
            options,
            existing_ids,
            delta_mode=True,
        )

        # Le HEAD check (count=1) ne doit pas avoir été déclenché
        assert not head_check_called

    async def test_no_early_exit_quand_delta_mode_false(self):
        """Si delta_mode=False → pas de HEAD check."""
        from src.data.sync._match_processing import MatchProcessingMixin
        from src.data.sync.models_sync import SyncOptions

        head_check_called = False

        async def tracking_history(gamertag, match_type=None, start=0, count=25):
            nonlocal head_check_called
            if count == 1:
                head_check_called = True
            return []

        engine = _make_engine_mock(latest_db="recent-match")
        client = MagicMock()
        client.get_match_history = tracking_history
        options = SyncOptions()
        existing_ids = {"recent-match"}

        await MatchProcessingMixin._process_matches(
            engine,
            client,
            options,
            existing_ids,
            delta_mode=False,  # Full sync → pas de HEAD check
        )

        assert not head_check_called

    async def test_no_early_exit_quand_latest_db_none(self):
        """_process_matches ne consulte pas latest_db — pagination directe.

        Phase B : latest_db n'est utilisé que dans le HEAD check de _sync_internal.
        _process_matches ignore latest_db et fait la pagination normalement.
        """
        from src.data.sync._match_processing import MatchProcessingMixin
        from src.data.sync.models_sync import SyncOptions

        call_count = 0

        async def counting_history(gamertag, match_type=None, start=0, count=25):
            nonlocal call_count
            call_count += 1
            return []

        engine = _make_engine_mock(latest_db=None)  # non utilisé par _process_matches
        client = MagicMock()
        client.get_match_history = counting_history
        options = SyncOptions()
        existing_ids = {"some-match"}

        await MatchProcessingMixin._process_matches(
            engine,
            client,
            options,
            existing_ids,
            delta_mode=True,
        )

        # Pagination déclenchée (au moins 1 appel), pas de HEAD check count=1
        assert call_count >= 1

    async def test_early_exit_quand_head_vide(self):
        """API retourne [] → boucle de pagination s'arrête immédiatement (0 inserts)."""
        from src.data.sync._match_processing import MatchProcessingMixin
        from src.data.sync.models_sync import SyncOptions

        call_count = 0

        async def empty_history(gamertag, match_type=None, start=0, count=25):
            nonlocal call_count
            call_count += 1
            return []  # Toujours vide

        engine = _make_engine_mock(latest_db="recent-match")
        client = MagicMock()
        client.get_match_history = empty_history
        options = SyncOptions()
        existing_ids = {"recent-match"}

        result = await MatchProcessingMixin._process_matches(
            engine,
            client,
            options,
            existing_ids,
            delta_mode=True,
        )

        # Pagination retourne [] → boucle break dès le premier batch
        assert result.matches_inserted == 0
        # Exactement 1 appel pagination (pas de HEAD check dans _process_matches)
        assert call_count >= 1
