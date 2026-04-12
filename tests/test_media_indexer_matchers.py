"""Tests de non-regression pour l'association media -> match."""

from __future__ import annotations

from datetime import datetime, timezone
from pathlib import Path

import duckdb

from src.data.media_indexer_matchers import _associate_single_media, _load_matches_by_xuid


def _player_db_path(tmp_path: Path) -> Path:
    db_path = tmp_path / "data" / "players" / "JGtm" / "stats.duckdb"
    db_path.parent.mkdir(parents=True, exist_ok=True)
    return db_path


def _shared_db_path(tmp_path: Path) -> Path:
    shared_path = tmp_path / "data" / "warehouse" / "shared_matches_v2.duckdb"
    shared_path.parent.mkdir(parents=True, exist_ok=True)
    return shared_path


def _create_player_db(db_path: Path) -> None:
    with duckdb.connect(str(db_path)) as conn:
        conn.execute(
            """
            CREATE TABLE media_match_associations (
                media_path VARCHAR,
                match_id VARCHAR,
                xuid VARCHAR,
                match_start_time TIMESTAMP,
                map_id VARCHAR,
                map_name VARCHAR,
                association_confidence DOUBLE,
                PRIMARY KEY (media_path, match_id, xuid)
            )
            """
        )


def _create_shared_db(shared_path: Path) -> None:
    with duckdb.connect(str(shared_path)) as conn:
        conn.execute(
            """
            CREATE TABLE match_registry (
                match_id VARCHAR PRIMARY KEY,
                start_time TIMESTAMP,
                duration_seconds INTEGER,
                map_id VARCHAR,
                map_name VARCHAR
            )
            """
        )
        conn.execute(
            """
            CREATE TABLE match_participants (
                match_id VARCHAR,
                xuid VARCHAR
            )
            """
        )
        conn.executemany(
            "INSERT INTO match_registry VALUES (?, ?, ?, ?, ?)",
            [
                ("winter-match", datetime(2026, 1, 25, 16, 9, 57), 489, "map-w", "Catalyst"),
                (
                    "summer-match",
                    datetime(2025, 8, 14, 13, 42, 8, 730000),
                    489,
                    "map-s",
                    "Oasis Heavies",
                ),
            ],
        )
        conn.executemany(
            "INSERT INTO match_participants VALUES (?, ?)",
            [("winter-match", "2533274823110022"), ("summer-match", "2533274823110022")],
        )


def test_load_matches_by_xuid_uses_sql_epoch_without_dst_shift(tmp_path: Path) -> None:
    """Les epochs retournes doivent correspondre au calcul SQL, hiver comme ete."""
    player_db = _player_db_path(tmp_path)
    shared_db = _shared_db_path(tmp_path)
    _create_player_db(player_db)
    _create_shared_db(shared_db)

    matches_by_xuid = _load_matches_by_xuid(player_db, [(player_db, "2533274823110022")])

    rows = matches_by_xuid["2533274823110022"]
    epochs_by_match = {row[0]: row[2] for row in rows}

    assert (
        epochs_by_match["winter-match"]
        == datetime(2026, 1, 25, 16, 9, 57, tzinfo=timezone.utc).timestamp()
    )
    assert (
        epochs_by_match["summer-match"]
        == datetime(2025, 8, 14, 13, 42, 8, 730000, tzinfo=timezone.utc).timestamp()
    )


def test_associate_single_media_matches_real_window_in_winter(tmp_path: Path) -> None:
    """Un clip d'hiver pris pendant le match doit etre associe sans decalage d'une heure."""
    player_db = _player_db_path(tmp_path)
    shared_db = _shared_db_path(tmp_path)
    _create_player_db(player_db)
    _create_shared_db(shared_db)

    matches_by_xuid = _load_matches_by_xuid(player_db, [(player_db, "2533274823110022")])
    media_epoch = datetime(2026, 1, 25, 16, 17, 54, 935995, tzinfo=timezone.utc).timestamp()

    with duckdb.connect(str(player_db), read_only=False) as conn:
        _associate_single_media(
            conn,
            media_path="winter-media.mp4",
            mtime_epoch=media_epoch,
            matches_by_xuid=matches_by_xuid,
            tol_seconds=3 * 60,
        )
        inserted = conn.execute(
            "SELECT match_id, xuid, map_name FROM media_match_associations WHERE media_path = ?",
            ["winter-media.mp4"],
        ).fetchone()

    assert inserted == ("winter-match", "2533274823110022", "Catalyst")


def test_associate_single_media_matches_real_window_in_summer(tmp_path: Path) -> None:
    """Un clip d'ete pris pendant le match doit etre associe sans decalage de deux heures."""
    player_db = _player_db_path(tmp_path)
    shared_db = _shared_db_path(tmp_path)
    _create_player_db(player_db)
    _create_shared_db(shared_db)

    matches_by_xuid = _load_matches_by_xuid(player_db, [(player_db, "2533274823110022")])
    media_epoch = datetime(2025, 8, 14, 13, 47, 0, tzinfo=timezone.utc).timestamp()

    with duckdb.connect(str(player_db), read_only=False) as conn:
        _associate_single_media(
            conn,
            media_path="summer-media.mp4",
            mtime_epoch=media_epoch,
            matches_by_xuid=matches_by_xuid,
            tol_seconds=3 * 60,
        )
        inserted = conn.execute(
            "SELECT match_id, xuid, map_name FROM media_match_associations WHERE media_path = ?",
            ["summer-media.mp4"],
        ).fetchone()

    assert inserted == ("summer-match", "2533274823110022", "Oasis Heavies")
