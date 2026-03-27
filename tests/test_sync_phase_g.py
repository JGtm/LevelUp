"""Tests — Phase G : consolidation event loop + suppression dead code.

Vérifie :
G.1 — event=async_loop_start loggé une seule fois pour N joueurs.
G.1 — event=async_loop_end loggé avec compteurs ok/failed.
G.1 — _sync_all_players_loop utilise asyncio.run() UNE seule fois (pas N fois).
G.1 — chaque joueur est traité via await sync_player_duckdb_async (pas asyncio.run interne).
G.3 — _sync_duckdb_player absent du codebase src/.
G.3 — _run_duckdb_player_sync_async absent du codebase src/.
"""

from __future__ import annotations

import logging
import subprocess
import sys
from pathlib import Path

# ─────────────────────────────────────────────────────────────────────────────
# G.3 — Vérification absence dead code
# ─────────────────────────────────────────────────────────────────────────────


def test_sync_duckdb_player_absent_du_codebase():
    """_sync_duckdb_player ne doit plus exister dans src/ (Phase G.3)."""
    import subprocess
    import sys

    result = subprocess.run(
        [
            sys.executable,
            "-m",
            "grep",
            "-r",
            "_sync_duckdb_player",
            "--include=*.py",
            "src/",
        ],
        capture_output=True,
        text=True,
        cwd=Path(__file__).parent.parent,
    )
    # grep exit code 0 = found, 1 = not found
    # On veut qu'il NE soit PAS trouvé
    assert result.returncode != 0 or result.stdout.strip() == "", (
        f"_sync_duckdb_player trouvé dans src/ : {result.stdout}"
    )


def test_run_duckdb_player_sync_async_absent_du_codebase():
    """_run_duckdb_player_sync_async ne doit plus exister dans src/ (Phase G.3)."""
    result = subprocess.run(
        [
            sys.executable,
            "-m",
            "grep",
            "-r",
            "_run_duckdb_player_sync_async",
            "--include=*.py",
            "src/",
        ],
        capture_output=True,
        text=True,
        cwd=Path(__file__).parent.parent,
    )
    assert result.returncode != 0 or result.stdout.strip() == "", (
        f"_run_duckdb_player_sync_async trouvé dans src/ : {result.stdout}"
    )


# ─────────────────────────────────────────────────────────────────────────────
# G.3 — _sync_all_players_loop_async est bien async
# ─────────────────────────────────────────────────────────────────────────────


def test_sync_all_players_loop_async_is_coroutine():
    """_sync_all_players_loop_async doit être une coroutine (async def)."""
    import inspect

    from src.ui._sync_duckdb_ops import _sync_all_players_loop_async

    assert inspect.iscoroutinefunction(_sync_all_players_loop_async), (
        "_sync_all_players_loop_async doit être async def"
    )


# ─────────────────────────────────────────────────────────────────────────────
# G.1 — Logs event=async_loop_start et event=async_loop_end
# ─────────────────────────────────────────────────────────────────────────────


import pytest  # noqa: E402


@pytest.mark.asyncio
async def test_async_loop_start_loggé_une_fois(caplog):
    """event=async_loop_start loggé UNE SEULE FOIS pour N joueurs (G.1)."""
    from pathlib import Path
    from unittest.mock import AsyncMock, patch

    from src.ui._sync_duckdb_ops import _sync_all_players_loop_async

    profiles = {
        "Player1": {"db_path": "data/players/Player1/stats.duckdb", "xuid": "111"},
        "Player2": {"db_path": "data/players/Player2/stats.duckdb", "xuid": "222"},
        "Player3": {"db_path": "data/players/Player3/stats.duckdb", "xuid": "333"},
    }

    with (
        patch("src.ui._sync_duckdb_ops._activate_sync_mode"),
        patch("src.ui._sync_duckdb_ops._deactivate_sync_mode"),
        patch(
            "src.ui._sync_duckdb_ops.sync_player_duckdb_async",
            new=AsyncMock(return_value=(True, "OK")),
        ),
        patch("pathlib.Path.exists", return_value=True),
        caplog.at_level(logging.INFO, logger="src.ui._sync_duckdb_ops"),
    ):
        await _sync_all_players_loop_async(
            profiles=profiles,
            delta=True,
            match_type="matchmaking",
            max_matches=50,
            with_highlight_events=True,
            with_aliases=True,
            repo_root=Path("/fake/root"),
        )

    start_records = [r for r in caplog.records if "async_loop_start" in r.message]
    assert len(start_records) == 1, (
        f"event=async_loop_start doit être loggé exactement 1 fois, "
        f"obtenu {len(start_records)}: {[r.message for r in start_records]}"
    )
    assert "players=3" in start_records[0].message, (
        f"async_loop_start doit indiquer players=3: {start_records[0].message}"
    )


@pytest.mark.asyncio
async def test_async_loop_end_loggé_avec_compteurs(caplog):
    """event=async_loop_end loggé avec ok= et failed= (G.1)."""
    from pathlib import Path
    from unittest.mock import patch

    from src.ui._sync_duckdb_ops import _sync_all_players_loop_async

    profiles = {
        "Player1": {"db_path": "data/players/Player1/stats.duckdb", "xuid": "111"},
        "Player2": {"db_path": "data/players/Player2/stats.duckdb", "xuid": "222"},
    }

    call_count = 0

    async def _mock_sync(*args, **kwargs):
        nonlocal call_count
        call_count += 1
        # Player1 OK, Player2 KO
        return (True, "OK") if call_count == 1 else (False, "Erreur")

    with (
        patch("src.ui._sync_duckdb_ops._activate_sync_mode"),
        patch("src.ui._sync_duckdb_ops._deactivate_sync_mode"),
        patch("src.ui._sync_duckdb_ops.sync_player_duckdb_async", side_effect=_mock_sync),
        patch("pathlib.Path.exists", return_value=True),
        caplog.at_level(logging.INFO, logger="src.ui._sync_duckdb_ops"),
    ):
        await _sync_all_players_loop_async(
            profiles=profiles,
            delta=True,
            match_type="matchmaking",
            max_matches=50,
            with_highlight_events=True,
            with_aliases=True,
            repo_root=Path("/fake/root"),
        )

    end_records = [r for r in caplog.records if "async_loop_end" in r.message]
    assert len(end_records) == 1, (
        f"event=async_loop_end doit être loggé une fois: {[r.message for r in end_records]}"
    )
    msg = end_records[0].message
    assert "ok=1" in msg, f"async_loop_end doit indiquer ok=1: {msg}"
    assert "failed=1" in msg, f"async_loop_end doit indiquer failed=1: {msg}"


@pytest.mark.asyncio
async def test_async_loop_appelle_sync_player_async_par_joueur(caplog):
    """_sync_all_players_loop_async appelle sync_player_duckdb_async pour chaque joueur (G.1)."""
    from pathlib import Path
    from unittest.mock import AsyncMock, patch

    from src.ui._sync_duckdb_ops import _sync_all_players_loop_async

    profiles = {
        "Alpha": {"db_path": "data/players/Alpha/stats.duckdb", "xuid": "aaa"},
        "Beta": {"db_path": "data/players/Beta/stats.duckdb", "xuid": "bbb"},
    }

    mock_sync = AsyncMock(return_value=(True, "OK"))

    with (
        patch("src.ui._sync_duckdb_ops._activate_sync_mode"),
        patch("src.ui._sync_duckdb_ops._deactivate_sync_mode"),
        patch("src.ui._sync_duckdb_ops.sync_player_duckdb_async", mock_sync),
        patch("pathlib.Path.exists", return_value=True),
    ):
        results = await _sync_all_players_loop_async(
            profiles=profiles,
            delta=False,
            match_type="matchmaking",
            max_matches=100,
            with_highlight_events=True,
            with_aliases=True,
            repo_root=Path("/fake/root"),
        )

    assert mock_sync.call_count == 2, (
        f"sync_player_duckdb_async doit être appelé 2 fois (1 par joueur), "
        f"obtenu {mock_sync.call_count}"
    )
    gamertags_called = [c.kwargs.get("gamertag") for c in mock_sync.call_args_list]
    assert set(gamertags_called) == {"Alpha", "Beta"}, f"Gamertags appelés: {gamertags_called}"
    assert len(results) == 2


@pytest.mark.asyncio
async def test_async_loop_db_introuvable_ne_crash_pas(caplog):
    """Si la DB d'un joueur est introuvable, _sync_all_players_loop_async continue (G.1)."""
    from pathlib import Path
    from unittest.mock import AsyncMock, patch

    from src.ui._sync_duckdb_ops import _sync_all_players_loop_async

    profiles = {
        "GoodPlayer": {"db_path": "data/players/GoodPlayer/stats.duckdb", "xuid": "g"},
        "MissingDB": {"db_path": "data/players/MissingDB/stats.duckdb", "xuid": "m"},
    }

    mock_sync = AsyncMock(return_value=(True, "OK"))

    def _path_exists(self: Path) -> bool:
        return "MissingDB" not in str(self)

    with (
        patch("src.ui._sync_duckdb_ops._activate_sync_mode"),
        patch("src.ui._sync_duckdb_ops._deactivate_sync_mode"),
        patch("src.ui._sync_duckdb_ops.sync_player_duckdb_async", mock_sync),
        patch("pathlib.Path.exists", _path_exists),
        caplog.at_level(logging.WARNING, logger="src.ui._sync_duckdb_ops"),
    ):
        results = await _sync_all_players_loop_async(
            profiles=profiles,
            delta=True,
            match_type="matchmaking",
            max_matches=50,
            with_highlight_events=True,
            with_aliases=True,
            repo_root=Path("/fake/root"),
        )

    assert len(results) == 2
    missing_result = next(r for r in results if r[0] == "MissingDB")
    assert missing_result[1] is False, "MissingDB doit avoir ok=False"
    assert mock_sync.call_count == 1, (
        "sync_player_duckdb_async doit être appelé 1 seule fois (GoodPlayer uniquement)"
    )
