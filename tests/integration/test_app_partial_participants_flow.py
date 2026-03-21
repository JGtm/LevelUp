"""INT-003: intégration participants partiels (graceful degradation)."""

from __future__ import annotations

from datetime import datetime, timezone

import pytest

duckdb = pytest.importorskip("duckdb")


@pytest.fixture
def participants_partial_dbs(tmp_path):
    """Crée une structure v5 avec shared_matches.duckdb + player stats.duckdb."""
    # Créer la shared DB avec match_participants
    shared_path = tmp_path / "data" / "warehouse" / "shared_matches.duckdb"
    shared_path.parent.mkdir(parents=True, exist_ok=True)
    conn_shared = duckdb.connect(str(shared_path))
    try:
        conn_shared.execute(
            """
            CREATE TABLE match_participants (
                match_id VARCHAR,
                xuid VARCHAR,
                gamertag VARCHAR,
                team_id INTEGER,
                rank INTEGER,
                score INTEGER,
                kills INTEGER,
                deaths INTEGER,
                assists INTEGER,
                PRIMARY KEY (match_id, xuid)
            )
            """
        )

        conn_shared.execute(
            """
            INSERT INTO match_participants (match_id, xuid, gamertag, team_id, rank, score, kills, deaths, assists)
            VALUES
                ('m1', 'x_me', 'Me', 1, 1, 1900, 12, 8, 6),
                ('m1', 'x_friend', 'Friend', 1, 2, 1700, 10, 9, 7),
                ('m1', 'x_opp', 'Enemy', 2, 3, NULL, NULL, NULL, NULL)
            """
        )

        conn_shared.execute(
            """
            CREATE TABLE xuid_aliases (
                xuid VARCHAR PRIMARY KEY,
                gamertag VARCHAR NOT NULL
            )
            """
        )
        conn_shared.execute(
            """
            INSERT INTO xuid_aliases (xuid, gamertag) VALUES
                ('x_me', 'Me'), ('x_friend', 'Friend'), ('x_opp', 'Enemy')
            """
        )
        conn_shared.execute(
            """
            CREATE VIEW v_gamertag_lookup AS
            SELECT COALESCE(xa.xuid, mp.xuid) AS xuid,
                   COALESCE(xa.gamertag, mp.gamertag) AS gamertag
            FROM xuid_aliases xa
            FULL OUTER JOIN (
                SELECT xuid, MAX(gamertag) AS gamertag
                FROM match_participants WHERE gamertag IS NOT NULL GROUP BY xuid
            ) mp ON xa.xuid = mp.xuid
            WHERE COALESCE(xa.gamertag, mp.gamertag) IS NOT NULL
            """
        )
    finally:
        conn_shared.close()

    # Créer la player DB avec match_stats et mv_player_matches
    player_path = tmp_path / "data" / "players" / "TestPlayer" / "stats.duckdb"
    player_path.parent.mkdir(parents=True, exist_ok=True)
    conn_player = duckdb.connect(str(player_path))
    try:
        conn_player.execute(
            """
            CREATE TABLE match_stats (
                match_id VARCHAR PRIMARY KEY,
                start_time TIMESTAMP
            )
            """
        )
        conn_player.execute(
            "INSERT INTO match_stats (match_id, start_time) VALUES ('m1', ?)",
            [datetime.now(timezone.utc)],
        )

        # Vue mv_player_matches requise par v5.1
        conn_player.execute(
            """
            CREATE VIEW IF NOT EXISTS mv_player_matches AS
            SELECT * FROM match_stats
            """
        )
    finally:
        conn_player.close()

    return player_path, shared_path


def test_partial_participants_are_handled_without_crash(participants_partial_dbs) -> None:
    """load_match_players_stats doit tolérer l'absence de colonnes rank/score et NULL KDA."""
    from src.data.repositories.duckdb_repo import DuckDBRepository

    player_path, shared_path = participants_partial_dbs
    repo = DuckDBRepository(
        player_db_path=str(player_path),
        xuid="x_me",
        shared_db_path=str(shared_path),
        read_only=True,
    )
    try:
        rows = repo.load_match_players_stats("m1")

        assert len(rows) == 3
        assert all("xuid" in row and "gamertag" in row for row in rows)

        # Colonnes manquantes => fallback propre (rank indexé, score None)
        assert all("rank" in row for row in rows)
        assert all("score" in row for row in rows)

        # KDA partiel => valeurs nulles transformées sans crash
        enemy = next(row for row in rows if row["xuid"] == "x_opp")
        assert enemy["kills"] == 0
        assert enemy["deaths"] == 0
        assert enemy["assists"] == 0
    finally:
        repo.close()
