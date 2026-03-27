"""Tests — Phase H : intégrité transactionnelle (shared DB).

Vérifie :
1. _maybe_batch_commit émet COMMIT + BEGIN TRANSACTION sur la shared connexion.
2. _process_matches ouvre BEGIN TRANSACTION avant le premier match.
3. COMMIT final émis sur shared quand _process_matches se termine normalement.
4. COMMIT final émis même si la boucle sort via delta_stop (return anticipé).
"""

from __future__ import annotations

import pytest

# ─────────────────────────────────────────────────────────────────────────────
# H.1 — _maybe_batch_commit sur shared DB
# ─────────────────────────────────────────────────────────────────────────────


def test_maybe_batch_commit_calls_shared_commit():
    """_maybe_batch_commit doit COMMIT + BEGIN TRANSACTION sur la shared connexion."""
    from unittest.mock import MagicMock

    from src.data.sync._match_processing_helpers import MatchProcessingHelpersMixin

    shared_conn = MagicMock()
    player_conn = MagicMock()

    engine = MagicMock()
    engine._get_connection.return_value = player_conn
    engine._get_shared_connection.return_value = shared_conn

    MatchProcessingHelpersMixin._maybe_batch_commit(engine, n_inserted=50, batch_size=50)

    # Player DB : commit()
    player_conn.commit.assert_called_once()
    # Shared DB : COMMIT puis BEGIN TRANSACTION
    calls = [str(c) for c in shared_conn.execute.call_args_list]
    assert any("COMMIT" in c for c in calls), f"COMMIT absent: {calls}"
    assert any("BEGIN TRANSACTION" in c for c in calls), f"BEGIN TRANSACTION absent: {calls}"


def test_maybe_batch_commit_no_trigger_below_threshold():
    """_maybe_batch_commit ne fait rien si le seuil n'est pas atteint."""
    from unittest.mock import MagicMock

    from src.data.sync._match_processing_helpers import MatchProcessingHelpersMixin

    shared_conn = MagicMock()
    player_conn = MagicMock()

    engine = MagicMock()
    engine._get_connection.return_value = player_conn
    engine._get_shared_connection.return_value = shared_conn

    MatchProcessingHelpersMixin._maybe_batch_commit(engine, n_inserted=30, batch_size=50)

    player_conn.commit.assert_not_called()
    shared_conn.execute.assert_not_called()


def test_maybe_batch_commit_no_shared_conn():
    """_maybe_batch_commit fonctionne si shared conn est None."""
    from unittest.mock import MagicMock

    from src.data.sync._match_processing_helpers import MatchProcessingHelpersMixin

    player_conn = MagicMock()

    engine = MagicMock()
    engine._get_connection.return_value = player_conn
    engine._get_shared_connection.return_value = None

    # Ne doit pas lever d'exception
    MatchProcessingHelpersMixin._maybe_batch_commit(engine, n_inserted=50, batch_size=50)
    player_conn.commit.assert_called_once()


# ─────────────────────────────────────────────────────────────────────────────
# H.2 — BEGIN TRANSACTION émis avant le premier match
# ─────────────────────────────────────────────────────────────────────────────


@pytest.mark.asyncio
async def test_process_matches_opens_shared_transaction(caplog):
    """_process_matches doit émettre BEGIN TRANSACTION sur shared conn avant le traitement."""
    import logging
    from unittest.mock import AsyncMock, MagicMock

    from src.data.sync._match_processing import MatchProcessingMixin

    shared_conn = MagicMock()
    shared_sql: list[str] = []
    shared_conn.execute.side_effect = lambda sql, *_: shared_sql.append(sql)

    engine = MagicMock()
    engine._gamertag = "TestPlayer"
    engine._get_shared_connection.return_value = shared_conn
    engine._get_connection.return_value = MagicMock()

    # Client retourne vide → pas de matchs à traiter → boucle sort immédiatement
    mock_client = MagicMock()
    mock_client.get_match_history = AsyncMock(return_value=[])

    mock_options = MagicMock()
    mock_options.max_matches = 10
    mock_options.match_type = "matchmaking"
    mock_options.parallel_fetch = 1
    mock_options.parallel_matches = 1
    mock_options.batch_commit_size = 50

    with caplog.at_level(logging.DEBUG, logger="src.data.sync._match_processing"):
        await MatchProcessingMixin._process_matches(
            engine,
            mock_client,
            mock_options,
            set(),
            delta_mode=False,
        )

    assert "BEGIN TRANSACTION" in shared_sql, (
        f"BEGIN TRANSACTION non émis sur shared conn: {shared_sql}"
    )


@pytest.mark.asyncio
async def test_process_matches_commits_shared_at_end(caplog):
    """_process_matches doit émettre COMMIT final sur shared conn à la fin."""
    import logging
    from unittest.mock import AsyncMock, MagicMock

    from src.data.sync._match_processing import MatchProcessingMixin

    shared_conn = MagicMock()
    shared_sql: list[str] = []
    shared_conn.execute.side_effect = lambda sql, *_: shared_sql.append(sql)

    engine = MagicMock()
    engine._gamertag = "TestPlayer"
    engine._get_shared_connection.return_value = shared_conn
    engine._get_connection.return_value = MagicMock()

    mock_client = MagicMock()
    mock_client.get_match_history = AsyncMock(return_value=[])

    mock_options = MagicMock()
    mock_options.max_matches = 10
    mock_options.match_type = "matchmaking"
    mock_options.parallel_fetch = 1
    mock_options.parallel_matches = 1
    mock_options.batch_commit_size = 50

    with caplog.at_level(logging.DEBUG, logger="src.data.sync._match_processing"):
        await MatchProcessingMixin._process_matches(
            engine,
            mock_client,
            mock_options,
            set(),
            delta_mode=False,
        )

    assert "COMMIT" in shared_sql, f"COMMIT final non émis: {shared_sql}"
    # ORDER: BEGIN TRANSACTION doit précéder le COMMIT final
    begin_idx = shared_sql.index("BEGIN TRANSACTION")
    commit_idx = len(shared_sql) - 1 - shared_sql[::-1].index("COMMIT")
    assert begin_idx < commit_idx, "BEGIN TRANSACTION doit précéder COMMIT"


@pytest.mark.asyncio
async def test_process_matches_no_shared_conn_no_error():
    """_process_matches ne lève pas d'erreur si shared conn est None."""
    from unittest.mock import AsyncMock, MagicMock

    from src.data.sync._match_processing import MatchProcessingMixin

    engine = MagicMock()
    engine._gamertag = "TestPlayer"
    engine._get_shared_connection.return_value = None
    engine._get_connection.return_value = MagicMock()

    mock_client = MagicMock()
    mock_client.get_match_history = AsyncMock(return_value=[])

    mock_options = MagicMock()
    mock_options.max_matches = 10
    mock_options.match_type = "matchmaking"
    mock_options.parallel_fetch = 1
    mock_options.parallel_matches = 1
    mock_options.batch_commit_size = 50

    # Ne doit pas lever d'exception
    result = await MatchProcessingMixin._process_matches(
        engine,
        mock_client,
        mock_options,
        set(),
        delta_mode=False,
    )
    assert result is not None
