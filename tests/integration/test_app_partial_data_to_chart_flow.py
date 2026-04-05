"""Tests d'intégration: flux partiel BDD -> app -> graphes (graceful degradation)."""

from __future__ import annotations

from datetime import datetime, timezone

import pandas as pd
import pytest

duckdb = pytest.importorskip("duckdb")
pl = pytest.importorskip("polars")


@pytest.fixture
def app_partial_flow_db(tmp_path):
    """Crée une structure v6 : player DB partielle + shared DB minimale."""
    now = datetime.now(timezone.utc)

    # ===== Player DB (données partielles intentionnelles) =====
    db_path = tmp_path / "app_partial_flow.duckdb"
    conn = duckdb.connect(str(db_path))
    try:
        conn.execute(
            """
            CREATE TABLE match_stats (
                match_id VARCHAR PRIMARY KEY,
                start_time TIMESTAMP,
                map_id VARCHAR,
                map_name VARCHAR,
                playlist_id VARCHAR,
                playlist_name VARCHAR,
                pair_id VARCHAR,
                pair_name VARCHAR,
                game_variant_id VARCHAR,
                game_variant_name VARCHAR,
                outcome INTEGER,
                team_id INTEGER,
                kills INTEGER,
                deaths INTEGER,
                assists INTEGER,
                kda DOUBLE,
                accuracy DOUBLE,
                headshot_kills INTEGER,
                max_killing_spree INTEGER,
                time_played_seconds INTEGER,
                avg_life_seconds DOUBLE,
                my_team_score INTEGER,
                enemy_team_score INTEGER,
                team_mmr DOUBLE,
                enemy_mmr DOUBLE
            )
            """
        )
        conn.execute(
            """
            CREATE TABLE highlight_events (
                match_id VARCHAR,
                event_type VARCHAR,
                time_ms INTEGER,
                xuid VARCHAR
            )
            """
        )
        conn.execute(
            """
            INSERT INTO match_stats (
                match_id, start_time, map_id, map_name, playlist_id, playlist_name,
                pair_id, pair_name, game_variant_id, game_variant_name, outcome, team_id,
                kills, deaths, assists, kda, accuracy, headshot_kills, max_killing_spree,
                time_played_seconds, avg_life_seconds, my_team_score, enemy_team_score,
                team_mmr, enemy_mmr
            ) VALUES
                ('m1', ?, 'map1', 'Recharge', 'pl1', 'Ranked Arena', 'pair1', 'Slayer', 'gv1', 'Slayer', 2, 1,
                 14, 8, 6, 1.75, 49.2, 5, 4, 680, 41.0, 50, 41, 1502.0, 1475.0),
                ('m2', ?, 'map2', 'Live Fire', 'pl2', 'Quick Play', 'pair2', 'Oddball', 'gv2', 'Oddball', NULL, 1,
                 9, 11, 4, 0.82, NULL, 2, 2, 610, NULL, 38, 45, NULL, NULL)
            """,
            [now, now],
        )
        conn.execute(
            """
            INSERT INTO highlight_events (match_id, event_type, time_ms, xuid)
            VALUES
                ('m1', 'Kill', 1200, 'x_me'),
                ('m2', 'Death', 1800, 'x_me')
            """
        )
    finally:
        conn.close()

    # ===== Shared DB (v6 — requis par DuckDBRepository) =====
    shared_path = tmp_path / "shared_matches_v2.duckdb"
    conn_shared = duckdb.connect(str(shared_path))
    try:
        conn_shared.execute("""
            CREATE TABLE match_registry (
                match_id VARCHAR PRIMARY KEY,
                start_time TIMESTAMP WITH TIME ZONE,
                map_id VARCHAR, map_name VARCHAR,
                playlist_id VARCHAR, playlist_name VARCHAR,
                pair_id VARCHAR, pair_name VARCHAR,
                game_variant_id VARCHAR, game_variant_name VARCHAR,
                team_0_score INTEGER DEFAULT 0, team_1_score INTEGER DEFAULT 0,
                duration_seconds INTEGER DEFAULT 600,
                is_firefight BOOLEAN DEFAULT FALSE, is_ranked BOOLEAN DEFAULT FALSE
            )
        """)
        conn_shared.execute("""
            INSERT INTO match_registry (match_id, start_time, map_id, map_name,
                playlist_id, playlist_name, pair_id, pair_name,
                game_variant_id, game_variant_name, team_0_score, team_1_score) VALUES
                ('m1', '2025-01-01 12:00:00+00', 'map1', 'Recharge',
                 'pl1', 'Ranked Arena', 'pair1', 'Slayer', 'gv1', 'Slayer', 50, 41),
                ('m2', '2025-01-01 12:20:00+00', 'map2', 'Live Fire',
                 'pl2', 'Quick Play', 'pair2', 'Oddball', 'gv2', 'Oddball', 38, 45)
        """)
        conn_shared.execute("""
            CREATE TABLE match_participants (
                match_id VARCHAR, xuid VARCHAR,
                gamertag VARCHAR, team_id INTEGER,
                rank INTEGER, score INTEGER,
                kills INTEGER, deaths INTEGER, assists INTEGER,
                PRIMARY KEY (match_id, xuid)
            )
        """)
        conn_shared.execute("""
            INSERT INTO match_participants (match_id, xuid, gamertag, team_id, rank, score, kills, deaths, assists)
            VALUES
                ('m1', 'x_me', 'Me', 1, 1, 1400, 14, 8, 6),
                ('m2', 'x_me', 'Me', 1, 1, 900, 9, 11, 4)
        """)
        conn_shared.execute("""
            CREATE TABLE xuid_aliases (xuid VARCHAR PRIMARY KEY, gamertag VARCHAR)
        """)
        conn_shared.execute("""
            CREATE VIEW v_gamertag_lookup AS
            SELECT COALESCE(xa.xuid, mp.xuid) AS xuid,
                   COALESCE(xa.gamertag, mp.gamertag) AS gamertag
            FROM xuid_aliases xa
            FULL OUTER JOIN (
                SELECT xuid, MAX(gamertag) AS gamertag
                FROM match_participants WHERE gamertag IS NOT NULL GROUP BY xuid
            ) mp ON xa.xuid = mp.xuid
            WHERE COALESCE(xa.gamertag, mp.gamertag) IS NOT NULL
        """)
        conn_shared.execute("""
            CREATE VIEW mv_player_matches AS
            SELECT r.match_id, r.start_time,
                   r.map_id, r.map_name, r.playlist_id, r.playlist_name,
                   r.pair_id, r.pair_name, r.game_variant_id, r.game_variant_name,
                   p.xuid, 2 AS outcome, 0 AS team_id,
                   0.0 AS kda, 0 AS max_killing_spree, 0 AS headshot_kills,
                   0.0 AS avg_life_seconds, 0.0 AS time_played_seconds,
                   COALESCE(p.kills, 0) AS kills, COALESCE(p.deaths, 0) AS deaths,
                   COALESCE(p.assists, 0) AS assists, NULL AS accuracy,
                   r.team_0_score AS my_team_score, r.team_1_score AS enemy_team_score,
                   NULL AS team_mmr, NULL AS enemy_mmr,
                   COALESCE(p.score, 0) AS personal_score,
                   FALSE AS is_firefight, FALSE AS is_ranked,
                   NULL::VARCHAR AS map_name_fr,
                   NULL::VARCHAR AS playlist_name_fr,
                   NULL::VARCHAR AS pair_name_fr,
                   NULL::VARCHAR AS game_variant_name_fr
            FROM match_registry r
            JOIN match_participants p ON r.match_id = p.match_id
        """)
    finally:
        conn_shared.close()

    return db_path, shared_path


def test_app_partial_data_to_chart_flow_graceful(app_partial_flow_db) -> None:
    """Le flux app doit rester fonctionnel avec des données partielles."""
    from src.analysis.friends_impact import (
        ImpactEventSets,
        build_impact_matrix,
        get_all_impact_events,
    )
    from src.data.repositories.duckdb_repo import DuckDBRepository
    from src.visualization.distributions import plot_win_ratio_heatmap
    from src.visualization.friends_impact_heatmap import plot_friends_impact_heatmap
    from src.visualization.timeseries import plot_timeseries

    player_db, shared_db = app_partial_flow_db
    repo = DuckDBRepository(
        str(player_db), xuid="x_me", shared_db_path=str(shared_db), read_only=True
    )
    try:
        stats_df = repo.load_match_stats_as_polars()
        assert not stats_df.is_empty()
        assert {"match_id", "start_time", "kills", "deaths", "outcome"}.issubset(stats_df.columns)

        stats_pd = pd.DataFrame(stats_df.to_dicts())
        deaths = stats_pd["deaths"].replace(0, 1)
        stats_pd["ratio"] = (stats_pd["kills"] / deaths).astype(float)

        # Graphes principaux : aucun crash même si certaines valeurs sont NULL
        fig_timeseries = plot_timeseries(stats_pd)
        fig_win_ratio = plot_win_ratio_heatmap(stats_pd, min_matches=1)
        assert len(fig_timeseries.data) >= 1
        assert len(fig_win_ratio.data) >= 1

        # Flux impact avec données partielles (depuis player DB)
        conn = duckdb.connect(str(player_db), read_only=True)
        try:
            events_rows = conn.execute(
                "SELECT match_id, xuid, NULL AS gamertag, event_type, time_ms FROM highlight_events"
            ).fetchall()
            match_rows = conn.execute("SELECT match_id, outcome FROM match_stats").fetchall()
        finally:
            conn.close()

        events_df = pl.DataFrame(
            {
                "match_id": [r[0] for r in events_rows],
                "xuid": [r[1] for r in events_rows],
                "gamertag": [r[2] for r in events_rows],
                "event_type": [r[3] for r in events_rows],
                "time_ms": [r[4] for r in events_rows],
            }
        )
        matches_df = pl.DataFrame(
            {
                "match_id": [r[0] for r in match_rows],
                "outcome": [r[1] for r in match_rows],
            }
        )

        (
            first_bloods,
            clutch_finishers,
            last_casualties,
            last_group_kills,
            first_group_deaths,
            scores,
        ) = get_all_impact_events(
            events_df,
            matches_df,
            friend_xuids={"x_me"},
        )

        # Le score peut être vide en données partielles, mais le pipeline ne doit pas casser
        gamertags = sorted(scores.keys()) or ["Me"]
        events = ImpactEventSets(
            first_bloods=first_bloods,
            clutch_finishers=clutch_finishers,
            last_casualties=last_casualties,
            last_group_kills=last_group_kills,
            first_group_deaths=first_group_deaths,
        )
        impact_matrix = build_impact_matrix(
            events,
            match_ids=["m1", "m2"],
            gamertags=gamertags,
        )

        fig_impact = plot_friends_impact_heatmap(impact_matrix, max_matches=20)
        assert fig_impact is not None
        assert len(fig_impact.data) >= 1
    finally:
        repo.close()
