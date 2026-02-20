"""Tests pour le backfill des scores de performance (V5).

Ce fichier teste :
- La détection des matchs sans score de performance via shared DB
- Le calcul des scores manquants via backfill (shared → player_match_enrichment)
"""

from __future__ import annotations

from datetime import datetime, timedelta, timezone
from pathlib import Path

import duckdb
import pytest

from scripts.backfill.detection import find_matches_missing_data
from scripts.backfill.strategies import compute_performance_score_for_match

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

_XUID = "2535423456789"
_PERF_BIT = 16  # BACKFILL_FLAGS["performance_scores"]


def _create_shared_conn(
    tmp_path: Path, *, with_backfill_col: bool = True
) -> duckdb.DuckDBPyConnection:
    """Crée shared_matches.duckdb avec match_registry + match_participants.

    25 matchs : match-000 à match-024.
    match-020 à match-024 ont backfill_completed = 16 (performance_scores bit).
    """
    tmp_path.mkdir(parents=True, exist_ok=True)
    db_path = tmp_path / "shared.duckdb"
    conn = duckdb.connect(str(db_path))

    bf_col = ", backfill_completed INTEGER DEFAULT 0" if with_backfill_col else ""
    conn.execute(f"""
        CREATE TABLE match_registry (
            match_id VARCHAR PRIMARY KEY,
            start_time TIMESTAMP,
            end_time TIMESTAMP,
            map_name TEXT,
            playlist_name TEXT,
            game_variant_name TEXT,
            pair_name TEXT
            {bf_col}
        )
    """)
    conn.execute("""
        CREATE TABLE match_participants (
            match_id VARCHAR NOT NULL,
            xuid VARCHAR NOT NULL,
            gamertag TEXT,
            kills SMALLINT DEFAULT 0,
            deaths SMALLINT DEFAULT 0,
            assists SMALLINT DEFAULT 0,
            kda FLOAT,
            accuracy FLOAT,
            time_played_seconds INTEGER DEFAULT 600,
            avg_life_seconds FLOAT DEFAULT 45.0,
            personal_score INTEGER,
            damage_dealt FLOAT,
            rank SMALLINT,
            team_mmr FLOAT,
            shots_fired INTEGER,
            shots_hit INTEGER,
            enemy_mmr FLOAT,
            kills_expected FLOAT,
            deaths_expected FLOAT,
            assists_expected FLOAT,
            PRIMARY KEY (match_id, xuid)
        )
    """)

    base_time = datetime(2024, 1, 1, 12, 0, 0, tzinfo=timezone.utc)

    for i in range(25):
        match_id = f"match-{i:03d}"
        start = base_time + timedelta(hours=i)
        end = start + timedelta(minutes=10)
        bf = _PERF_BIT if i >= 20 else 0

        if with_backfill_col:
            conn.execute(
                "INSERT INTO match_registry "
                "(match_id, start_time, end_time, backfill_completed) VALUES (?, ?, ?, ?)",
                (match_id, start, end, bf),
            )
        else:
            conn.execute(
                "INSERT INTO match_registry (match_id, start_time, end_time) VALUES (?, ?, ?)",
                (match_id, start, end),
            )

        kills = 10 + i if i < 20 else 15
        kda = 1.5 + (i * 0.1) if i < 20 else 3.0
        accuracy = 0.50 if i < 20 else 0.55
        conn.execute(
            "INSERT INTO match_participants "
            "(match_id, xuid, gamertag, kills, deaths, assists, kda, accuracy, "
            "time_played_seconds, avg_life_seconds) "
            "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
            (match_id, _XUID, "TestPlayer", kills, 8, 3, kda, accuracy, 600, 45.0),
        )

    return conn


def _create_player_conn(
    tmp_path: Path, *, with_existing_scores: bool = False
) -> duckdb.DuckDBPyConnection:
    """Crée une DB joueur avec player_match_enrichment."""
    tmp_path.mkdir(parents=True, exist_ok=True)
    db_path = tmp_path / "player.duckdb"
    conn = duckdb.connect(str(db_path))
    conn.execute("""
        CREATE TABLE player_match_enrichment (
            match_id VARCHAR PRIMARY KEY,
            performance_score FLOAT,
            session_id VARCHAR,
            session_label VARCHAR,
            is_with_friends BOOLEAN,
            teammates_signature VARCHAR,
            known_teammates_count SMALLINT,
            friends_xuids VARCHAR,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    if with_existing_scores:
        # Pré-remplir les scores pour match-020 à match-024
        for i in range(20, 25):
            conn.execute(
                "INSERT INTO player_match_enrichment (match_id, performance_score) "
                "VALUES (?, ?)",
                (f"match-{i:03d}", 75.5),
            )
    return conn


# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------


@pytest.fixture
def shared_and_player(
    tmp_path: Path,
) -> tuple[duckdb.DuckDBPyConnection, duckdb.DuckDBPyConnection]:
    """Crée shared_conn + player_conn pour les tests."""
    shared = _create_shared_conn(tmp_path / "shared")
    player = _create_player_conn(tmp_path / "player", with_existing_scores=True)
    yield shared, player
    shared.close()
    player.close()


# ---------------------------------------------------------------------------
# Tests détection
# ---------------------------------------------------------------------------


class TestFindMatchesMissingPerformanceScore:
    """Tests pour la détection des matchs sans score de performance."""

    def test_finds_matches_without_score(
        self, shared_and_player: tuple[duckdb.DuckDBPyConnection, duckdb.DuckDBPyConnection]
    ):
        """Test que find_matches_missing_data trouve les matchs sans score.

        Les matchs 020-024 ont le bit performance_scores (16) positionné
        dans backfill_completed, ils doivent être exclus.
        """
        shared_conn, player_conn = shared_and_player

        match_ids = find_matches_missing_data(
            player_conn,
            _XUID,
            shared_conn=shared_conn,
            performance_scores=True,
        )

        # Devrait trouver les 20 premiers matchs (bit 16 non positionné)
        assert len(match_ids) == 20
        assert "match-000" in match_ids
        assert "match-019" in match_ids
        assert "match-020" not in match_ids

    def test_finds_all_matches_if_backfill_col_missing(self, tmp_path: Path):
        """Test que tous les matchs sont trouvés si backfill_completed n'existe pas."""
        shared_conn = _create_shared_conn(tmp_path / "shared", with_backfill_col=False)
        player_conn = _create_player_conn(tmp_path / "player")

        match_ids = find_matches_missing_data(
            player_conn,
            _XUID,
            shared_conn=shared_conn,
            performance_scores=True,
        )

        shared_conn.close()
        player_conn.close()

        # Sans backfill_completed, pas de bitmask guard → tous les matchs
        assert len(match_ids) == 25


# ---------------------------------------------------------------------------
# Tests calcul
# ---------------------------------------------------------------------------


class TestComputePerformanceScoreForMatch:
    """Tests pour le calcul d'un score de performance via shared DB."""

    def test_computes_score_with_sufficient_history(
        self, shared_and_player: tuple[duckdb.DuckDBPyConnection, duckdb.DuckDBPyConnection]
    ):
        """Test que le score est calculé avec assez d'historique.

        match-010 a 10 matchs d'historique antérieurs (match-000..009),
        ce qui satisfait MIN_MATCHES_FOR_RELATIVE.
        """
        shared_conn, player_conn = shared_and_player

        result = compute_performance_score_for_match(
            player_conn, "match-010", shared_conn=shared_conn, xuid=_XUID
        )

        assert result is True

        # Vérifier que le score a été écrit dans player_match_enrichment
        row = player_conn.execute(
            "SELECT performance_score FROM player_match_enrichment WHERE match_id = ?",
            ("match-010",),
        ).fetchone()
        assert row is not None
        assert row[0] is not None
        assert 0 <= row[0] <= 100

    def test_does_not_recalculate_existing_score(
        self, shared_and_player: tuple[duckdb.DuckDBPyConnection, duckdb.DuckDBPyConnection]
    ):
        """Test que le score existant n'est pas recalculé."""
        shared_conn, player_conn = shared_and_player

        # match-020 a déjà un score dans player_match_enrichment
        result = compute_performance_score_for_match(
            player_conn, "match-020", shared_conn=shared_conn, xuid=_XUID
        )

        assert result is False, "Ne doit pas recalculer un score existant"

    def test_returns_false_with_insufficient_history(self, tmp_path: Path):
        """Test que False est retourné s'il n'y a pas assez d'historique."""
        shared_path = tmp_path / "shared_small.duckdb"
        shared_conn = duckdb.connect(str(shared_path))
        shared_conn.execute("""
            CREATE TABLE match_registry (
                match_id VARCHAR PRIMARY KEY,
                start_time TIMESTAMP,
                end_time TIMESTAMP
            )
        """)
        shared_conn.execute("""
            CREATE TABLE match_participants (
                match_id VARCHAR NOT NULL,
                xuid VARCHAR NOT NULL,
                gamertag TEXT,
                kills SMALLINT, deaths SMALLINT, assists SMALLINT,
                kda FLOAT, accuracy FLOAT,
                time_played_seconds INTEGER, avg_life_seconds FLOAT,
                personal_score INTEGER, damage_dealt FLOAT,
                rank SMALLINT, team_mmr FLOAT,
                enemy_mmr FLOAT,
                kills_expected FLOAT,
                deaths_expected FLOAT,
                assists_expected FLOAT,
                PRIMARY KEY (match_id, xuid)
            )
        """)

        # Seulement 5 matchs (< MIN_MATCHES_FOR_RELATIVE = 10)
        base_time = datetime(2024, 1, 1, 12, 0, 0, tzinfo=timezone.utc)
        for i in range(5):
            mid = f"match-{i:03d}"
            t = base_time + timedelta(hours=i)
            shared_conn.execute(
                "INSERT INTO match_registry VALUES (?, ?, ?)",
                (mid, t, t + timedelta(minutes=10)),
            )
            shared_conn.execute(
                "INSERT INTO match_participants "
                "(match_id, xuid, kills, deaths, assists, kda, accuracy, "
                "time_played_seconds, avg_life_seconds) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
                (mid, _XUID, 10, 8, 3, 1.5, 0.50, 600, 45.0),
            )

        # Match cible
        target_time = base_time + timedelta(hours=18)
        shared_conn.execute(
            "INSERT INTO match_registry VALUES (?, ?, ?)",
            ("target-match", target_time, target_time + timedelta(minutes=10)),
        )
        shared_conn.execute(
            "INSERT INTO match_participants "
            "(match_id, xuid, kills, deaths, assists, kda, accuracy, "
            "time_played_seconds, avg_life_seconds) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
            ("target-match", _XUID, 15, 5, 4, 3.0, 0.55, 600, 50.0),
        )

        player_conn = _create_player_conn(tmp_path / "player_small")

        result = compute_performance_score_for_match(
            player_conn, "target-match", shared_conn=shared_conn, xuid=_XUID
        )

        shared_conn.close()
        player_conn.close()

        assert result is False, "Ne doit pas calculer avec historique insuffisant"

    def test_handles_missing_start_time(
        self, shared_and_player: tuple[duckdb.DuckDBPyConnection, duckdb.DuckDBPyConnection]
    ):
        """Test que le calcul gère correctement les matchs sans start_time."""
        shared_conn, player_conn = shared_and_player

        # Insérer un match sans start_time dans shared
        shared_conn.execute(
            "INSERT INTO match_registry (match_id, start_time, end_time) VALUES (?, ?, ?)",
            ("match-no-time", None, None),
        )
        shared_conn.execute(
            "INSERT INTO match_participants "
            "(match_id, xuid, kills, deaths, assists, kda, accuracy, "
            "time_played_seconds, avg_life_seconds) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
            ("match-no-time", _XUID, 10, 8, 3, 1.5, 0.50, 600, 45.0),
        )

        result = compute_performance_score_for_match(
            player_conn, "match-no-time", shared_conn=shared_conn, xuid=_XUID
        )

        assert result is False
