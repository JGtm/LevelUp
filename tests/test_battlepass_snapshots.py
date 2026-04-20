"""Tests ciblés pour l'historique battle pass joueur."""

from __future__ import annotations

from pathlib import Path

from src.data.battlepass_snapshots import persist_battlepass_snapshots
from src.utils.db import duckdb_read_only


def _player_db_path(tmp_path: Path) -> Path:
    return tmp_path / "data" / "players" / "Sample" / "stats.duckdb"


def _operations_payload() -> dict:
    return {
        "ActiveOperationRewardTrackPath": "RewardTracks/Operations/S13Op01.json",
        "OperationRewardTracks": [
            {
                "RewardTrackPath": "RewardTracks/Operations/S13Op01.json",
                "TrackType": "Operation",
                "CurrentProgress": {
                    "Rank": 20,
                    "PartialProgress": 250,
                    "IsOwned": True,
                    "HasReachedMaxRank": False,
                },
                "PreviousProgress": None,
                "IsOwned": True,
                "BaseXp": 1000,
                "BoostXp": 250,
            },
            {
                "RewardTrackPath": "RewardTracks/Operations/S12Op03.json",
                "TrackType": "Operation",
                "CurrentProgress": {
                    "Rank": 0,
                    "PartialProgress": 0,
                    "IsOwned": False,
                    "HasReachedMaxRank": False,
                },
                "PreviousProgress": None,
                "IsOwned": False,
                "BaseXp": None,
                "BoostXp": None,
            },
        ],
    }


def test_persist_battlepass_snapshots_inserts_and_deduplicates(tmp_path: Path) -> None:
    """Les snapshots battle pass joueur doivent être append-only mais dédupliqués."""
    player_db = _player_db_path(tmp_path)

    inserted = persist_battlepass_snapshots(player_db, "2535469190789936", _operations_payload())
    inserted_again = persist_battlepass_snapshots(player_db, "2535469190789936", _operations_payload())

    assert inserted == 2
    assert inserted_again == 0

    with duckdb_read_only(player_db) as conn:
        rows = conn.execute(
            """
            SELECT reward_track_path, is_active, current_rank, partial_progress, is_owned
            FROM battlepass_snapshots
            ORDER BY reward_track_path
            """
        ).fetchall()

    assert rows == [
        ("RewardTracks/Operations/S12Op03.json", False, 0, 0, False),
        ("RewardTracks/Operations/S13Op01.json", True, 20, 250, True),
    ]
