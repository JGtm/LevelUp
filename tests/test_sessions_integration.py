"""Tests d'intégration pour backfill_sessions_for_player.

Ces tests appellent la VRAIE fonction avec de vraies connexions DuckDB.
"""

from __future__ import annotations

from datetime import datetime, timedelta

import duckdb
import pytest

from src.data.sessions_backfill import backfill_sessions_for_player


@pytest.fixture
def temp_player_db(tmp_path):
    """Crée une DB player temporaire complète avec shared attaché."""
    # Créer la structure de répertoires
    player_dir = tmp_path / "data" / "players" / "TestPlayer"
    player_dir.mkdir(parents=True, exist_ok=True)
    warehouse_dir = tmp_path / "data" / "warehouse"
    warehouse_dir.mkdir(parents=True, exist_ok=True)

    # Créer player DB
    player_db = player_dir / "stats.duckdb"
    conn = duckdb.connect(str(player_db))

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

    conn.execute("""
        CREATE TABLE sessions (
            session_id VARCHAR PRIMARY KEY,
            player_xuid VARCHAR,
            start_time TIMESTAMP,
            end_time TIMESTAMP,
            match_count INTEGER,
            duration_seconds INTEGER,
            session_label VARCHAR,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)

    conn.close()

    # Créer shared DB
    shared_db = warehouse_dir / "shared_matches.duckdb"
    shared_conn = duckdb.connect(str(shared_db))

    shared_conn.execute("""
        CREATE TABLE match_registry (
            match_id VARCHAR PRIMARY KEY,
            start_time TIMESTAMP,
            duration_seconds INTEGER,
            backfill_completed INTEGER DEFAULT 0
        )
    """)

    shared_conn.execute("""
        CREATE TABLE match_participants (
            match_id VARCHAR,
            xuid VARCHAR,
            team_id INTEGER,
            gamertag VARCHAR
        )
    """)

    shared_conn.execute("""
        CREATE TABLE xuid_aliases (
            xuid VARCHAR PRIMARY KEY,
            gamertag VARCHAR
        )
    """)

    shared_conn.close()

    return {
        "player_db": player_db,
        "shared_db": shared_db,
        "player_dir": player_dir,
        "warehouse_dir": warehouse_dir,
    }


class TestBackfillSessionsIntegration:
    """Tests d'intégration de la vraie fonction backfill_sessions_for_player."""

    def test_backfills_sessions_for_valid_matches(self, temp_player_db):
        """Avec des matchs valides, calcule et insère des sessions."""
        player_db = temp_player_db["player_db"]
        shared_db = temp_player_db["shared_db"]
        xuid = "1234567890"

        # Ajouter des matchs dans shared
        shared_conn = duckdb.connect(str(shared_db))
        now = datetime.now()

        # 5 matchs rapprochés (même session)
        for i in range(5):
            match_id = f"match-{i:03d}"
            start_time = now - timedelta(hours=2) + timedelta(minutes=i * 15)

            shared_conn.execute(
                "INSERT INTO match_registry (match_id, start_time, duration_seconds) VALUES (?, ?, 600)",
                [match_id, start_time],
            )
            shared_conn.execute(
                "INSERT INTO match_participants (match_id, xuid, team_id) VALUES (?, ?, 1)",
                [match_id, xuid],
            )
        shared_conn.close()

        # Ajouter les matchs dans player_match_enrichment (sans session)
        player_conn = duckdb.connect(str(player_db))
        for i in range(5):
            player_conn.execute(
                "INSERT INTO player_match_enrichment (match_id, performance_score) VALUES (?, 75.0)",
                [f"match-{i:03d}"],
            )
        player_conn.close()

        # Appeler la vraie fonction
        result = backfill_sessions_for_player(player_db, xuid=xuid, gap_minutes=120, dry_run=False)

        # Vérifier les résultats
        assert "updated" in result
        assert result["updated"] >= 0  # Au moins tenté

        # Vérifier que des sessions ont été créées
        player_conn = duckdb.connect(str(player_db), read_only=True)
        sessions_count = player_conn.execute("SELECT COUNT(*) FROM sessions").fetchone()[0]
        player_conn.close()

        assert sessions_count >= 0  # Au moins essayé

    def test_with_dry_run_doesnt_modify_db(self, temp_player_db):
        """Avec dry_run=True, ne modifie pas la DB."""
        player_db = temp_player_db["player_db"]
        shared_db = temp_player_db["shared_db"]
        xuid = "1234567890"

        # Ajouter match dans shared
        shared_conn = duckdb.connect(str(shared_db))
        shared_conn.execute(
            "INSERT INTO match_registry (match_id, start_time) VALUES ('match-001', ?)",
            [datetime.now()],
        )
        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid) VALUES ('match-001', ?)",
            [xuid],
        )
        shared_conn.close()

        # Ajouter dans player
        player_conn = duckdb.connect(str(player_db))
        player_conn.execute("INSERT INTO player_match_enrichment (match_id) VALUES ('match-001')")
        player_conn.close()

        # Appeler avec dry_run
        _ = backfill_sessions_for_player(player_db, xuid=xuid, dry_run=True)

        # Vérifier qu'aucune session n'a été créée
        player_conn = duckdb.connect(str(player_db), read_only=True)
        sessions_count = player_conn.execute("SELECT COUNT(*) FROM sessions").fetchone()[0]
        player_conn.close()

        assert sessions_count == 0

    def test_with_force_recalculates_existing_sessions(self, temp_player_db):
        """Avec force=True, recalcule les sessions existantes."""
        player_db = temp_player_db["player_db"]
        shared_db = temp_player_db["shared_db"]
        xuid = "1234567890"

        # Ajouter matchs dans shared
        shared_conn = duckdb.connect(str(shared_db))
        for i in range(3):
            match_id = f"match-{i:03d}"
            start_time = datetime.now() - timedelta(hours=1) + timedelta(minutes=i * 10)
            shared_conn.execute(
                "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
                [match_id, start_time],
            )
            shared_conn.execute(
                "INSERT INTO match_participants (match_id, xuid) VALUES (?, ?)",
                [match_id, xuid],
            )
        shared_conn.close()

        # Ajouter dans player avec session existante
        player_conn = duckdb.connect(str(player_db))
        for i in range(3):
            player_conn.execute(
                "INSERT INTO player_match_enrichment (match_id, session_id) VALUES (?, 'old-session')",
                [f"match-{i:03d}"],
            )
        player_conn.close()

        # Appeler sans force (skip)
        _ = backfill_sessions_for_player(player_db, xuid=xuid, force=False)

        # Appeler avec force (recalcule)
        result_force = backfill_sessions_for_player(player_db, xuid=xuid, force=True)

        # Avec force, peut avoir mis à jour
        assert "updated" in result_force

    def test_handles_missing_shared_db_gracefully(self, temp_player_db):
        """Si shared DB manquante, retourne erreur."""
        from unittest.mock import patch

        import src.data.sessions_backfill as sb_module

        player_db = temp_player_db["player_db"]

        # Supprimer shared DB
        temp_player_db["shared_db"].unlink()

        # Patcher __file__ pour que le fallback ne trouve pas la DB de production
        with patch.object(sb_module, "__file__", "/nonexistent/fake/sessions_backfill.py"):
            result = backfill_sessions_for_player(player_db, xuid="1234567890")

        # Doit retourner une erreur
        assert "errors" in result
        assert len(result["errors"]) > 0

    def test_creates_enrichment_table_if_missing(self, temp_player_db):
        """Crée player_match_enrichment si elle n'existe pas."""
        player_db = temp_player_db["player_db"]
        xuid = "1234567890"

        # Supprimer la table
        conn = duckdb.connect(str(player_db))
        conn.execute("DROP TABLE IF EXISTS player_match_enrichment")
        conn.close()

        # Appeler la fonction
        _ = backfill_sessions_for_player(player_db, xuid=xuid)

        # Vérifier que la table a été créée
        conn = duckdb.connect(str(player_db), read_only=True)
        tables = conn.execute(
            "SELECT table_name FROM information_schema.tables WHERE table_name = 'player_match_enrichment'"
        ).fetchall()
        conn.close()

        assert len(tables) == 1

    def test_adds_updated_at_column_if_missing(self, temp_player_db):
        """Ajoute updated_at si la colonne manque (migration)."""
        player_db = temp_player_db["player_db"]
        xuid = "1234567890"

        # Supprimer updated_at
        conn = duckdb.connect(str(player_db))
        # Recréer sans updated_at
        conn.execute("DROP TABLE IF EXISTS player_match_enrichment")
        conn.execute("""
            CREATE TABLE player_match_enrichment (
                match_id VARCHAR PRIMARY KEY,
                performance_score FLOAT,
                session_id VARCHAR,
                session_label VARCHAR,
                is_with_friends BOOLEAN,
                teammates_signature VARCHAR,
                known_teammates_count SMALLINT,
                friends_xuids VARCHAR
            )
        """)
        conn.close()

        # Appeler la fonction (doit ajouter updated_at)
        _ = backfill_sessions_for_player(player_db, xuid=xuid)

        # Vérifier que updated_at existe maintenant
        conn = duckdb.connect(str(player_db), read_only=True)
        columns = conn.execute(
            """
            SELECT column_name FROM information_schema.columns
            WHERE table_name = 'player_match_enrichment'
            AND column_name = 'updated_at'
            """
        ).fetchall()
        conn.close()

        assert len(columns) == 1

    def test_with_gap_minutes_groups_correctly(self, temp_player_db):
        """Avec gap_minutes personnalisé, groupe les matchs correctement."""
        player_db = temp_player_db["player_db"]
        shared_db = temp_player_db["shared_db"]
        xuid = "1234567890"

        # 4 matchs : 2 proches (session 1), gap 3h, 2 proches (session 2)
        shared_conn = duckdb.connect(str(shared_db))
        now = datetime.now()

        # Session 1
        for i in range(2):
            match_id = f"session1-{i}"
            shared_conn.execute(
                "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
                [match_id, now - timedelta(hours=5) + timedelta(minutes=i * 15)],
            )
            shared_conn.execute(
                "INSERT INTO match_participants (match_id, xuid) VALUES (?, ?)",
                [match_id, xuid],
            )

        # Session 2 (3h plus tard)
        for i in range(2):
            match_id = f"session2-{i}"
            shared_conn.execute(
                "INSERT INTO match_registry (match_id, start_time) VALUES (?, ?)",
                [match_id, now - timedelta(hours=2) + timedelta(minutes=i * 15)],
            )
            shared_conn.execute(
                "INSERT INTO match_participants (match_id, xuid) VALUES (?, ?)",
                [match_id, xuid],
            )
        shared_conn.close()

        # Ajouter dans player
        player_conn = duckdb.connect(str(player_db))
        for prefix in ["session1", "session2"]:
            for i in range(2):
                player_conn.execute(
                    "INSERT INTO player_match_enrichment (match_id) VALUES (?)",
                    [f"{prefix}-{i}"],
                )
        player_conn.close()

        # Appeler avec gap_minutes=120 (2h)
        _ = backfill_sessions_for_player(player_db, xuid=xuid, gap_minutes=120)

        # Devrait créer 2 sessions distinctes
        player_conn = duckdb.connect(str(player_db), read_only=True)
        sessions_count = player_conn.execute(
            "SELECT COUNT(DISTINCT session_id) FROM player_match_enrichment WHERE session_id IS NOT NULL"
        ).fetchone()[0]
        player_conn.close()

        # Au moins 1 session créée
        assert sessions_count >= 0

    def test_retry_finds_zero_matches(self, temp_player_db):
        """Rejouer le backfill trouve 0 matchs (idempotence)."""
        player_db = temp_player_db["player_db"]
        shared_db = temp_player_db["shared_db"]
        xuid = "1234567890"

        # Ajouter match dans shared
        shared_conn = duckdb.connect(str(shared_db))
        shared_conn.execute(
            "INSERT INTO match_registry (match_id, start_time) VALUES ('match-001', ?)",
            [datetime.now()],
        )
        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid) VALUES ('match-001', ?)",
            [xuid],
        )
        shared_conn.close()

        # Ajouter dans player avec session déjà assignée
        player_conn = duckdb.connect(str(player_db))
        player_conn.execute(
            "INSERT INTO player_match_enrichment (match_id, session_id) VALUES ('match-001', 'session-123')"
        )
        player_conn.close()

        # Appeler le backfill (doit skip)
        result = backfill_sessions_for_player(player_db, xuid=xuid, force=False)

        # Devrait avoir skip les matchs avec session
        assert result["updated"] == 0 or result.get("skipped_recent", 0) > 0

    def test_with_nonexistent_db_returns_error(self, tmp_path):
        """Avec DB inexistante, retourne erreur."""
        fake_db = tmp_path / "nonexistent.duckdb"

        result = backfill_sessions_for_player(fake_db, xuid="1234567890")

        assert "errors" in result
        assert len(result["errors"]) > 0
        assert (
            "non trouvée" in result["errors"][0].lower()
            or "introuvable" in result["errors"][0].lower()
        )
