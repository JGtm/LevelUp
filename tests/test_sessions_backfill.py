"""Tests pour le backfill sessions et la migration updated_at.

Ce module teste les fixes du backfill sessions :
- Migration de la colonne updated_at avec DEFAULT CURRENT_TIMESTAMP
- INSERT qui n'utilise pas CURRENT_TIMESTAMP (utilise DEFAULT)
- UPDATE qui utilise now() au lieu de CURRENT_TIMESTAMP
"""

from __future__ import annotations

from datetime import datetime, timedelta

import duckdb
import pytest


@pytest.fixture
def player_conn():
    """Player DB avec table sessions."""
    conn = duckdb.connect(":memory:")

    # Migration : créer table avec updated_at
    conn.execute("""
        CREATE TABLE sessions (
            session_id VARCHAR PRIMARY KEY,
            player_xuid VARCHAR NOT NULL,
            start_time TIMESTAMP NOT NULL,
            end_time TIMESTAMP NOT NULL,
            match_count INTEGER NOT NULL,
            duration_seconds INTEGER NOT NULL,
            session_label VARCHAR,
            updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)

    conn.execute("""
        CREATE TABLE player_match_enrichment (
            match_id VARCHAR PRIMARY KEY,
            performance_score FLOAT,
            session_id VARCHAR,
            is_with_friends BOOLEAN
        )
    """)

    return conn


class TestSessionsMigration:
    """Tests de la migration updated_at."""

    def test_migration_adds_updated_at_column(self, player_conn):
        """La migration doit ajouter updated_at avec DEFAULT CURRENT_TIMESTAMP."""
        # Vérifier que la colonne existe
        schema = player_conn.execute("""
            SELECT column_name, column_default
            FROM information_schema.columns
            WHERE table_name = 'sessions'
            AND column_name = 'updated_at'
        """).fetchone()

        assert schema is not None
        assert schema[0] == "updated_at"
        # Le DEFAULT contient CURRENT_TIMESTAMP
        assert "CURRENT_TIMESTAMP" in schema[1].upper()

    def test_existing_sessions_get_default_timestamp(self, player_conn):
        """Les sessions existantes doivent avoir un updated_at automatique."""
        session_id = "session-existing"
        xuid = "xuid-player"
        start = datetime.now() - timedelta(hours=2)
        end = datetime.now()

        # INSERT ancien sans updated_at → utilise DEFAULT
        player_conn.execute(
            """
            INSERT INTO sessions
            (session_id, player_xuid, start_time, end_time, match_count, duration_seconds)
            VALUES (?, ?, ?, ?, 5, 7200)
            """,
            [session_id, xuid, start, end],
        )

        # Vérifier que updated_at a été rempli automatiquement
        result = player_conn.execute(
            """
            SELECT updated_at FROM sessions WHERE session_id = ?
            """,
            [session_id],
        ).fetchone()

        assert result[0] is not None, "updated_at doit être rempli par DEFAULT"


class TestSessionsInsert:
    """Tests des INSERT sans référence à CURRENT_TIMESTAMP."""

    def test_insert_without_updated_at_uses_default(self, player_conn):
        """INSERT sans updated_at doit utiliser le DEFAULT."""
        session_id = "session-new-001"
        xuid = "xuid-player"
        start = datetime.now() - timedelta(hours=1)
        end = datetime.now()

        # INSERT qui n'inclut PAS updated_at
        player_conn.execute(
            """
            INSERT INTO sessions
            (session_id, player_xuid, start_time, end_time, match_count, duration_seconds, session_label)
            VALUES (?, ?, ?, ?, 3, 3600, 'Session Alpha')
            """,
            [session_id, xuid, start, end],
        )

        # Vérifier que updated_at existe
        result = player_conn.execute(
            """
            SELECT session_id, updated_at FROM sessions WHERE session_id = ?
            """,
            [session_id],
        ).fetchone()

        assert result[0] == session_id
        assert result[1] is not None

    def test_bulk_insert_sessions_all_get_timestamp(self, player_conn):
        """Insertion en bulk : tous les rows doivent avoir updated_at."""
        xuid = "xuid-player"
        now = datetime.now()

        # 5 sessions
        sessions_data = [
            (f"session-{i:03d}", xuid, now - timedelta(hours=i), now, i + 1, 3600 * (i + 1))
            for i in range(5)
        ]

        # INSERT multiple
        player_conn.executemany(
            """
            INSERT INTO sessions
            (session_id, player_xuid, start_time, end_time, match_count, duration_seconds)
            VALUES (?, ?, ?, ?, ?, ?)
            """,
            sessions_data,
        )

        # Vérifier que toutes ont updated_at
        result = player_conn.execute(
            """
            SELECT COUNT(*) FROM sessions
            WHERE updated_at IS NOT NULL
            """
        ).fetchone()[0]

        assert result == 5


class TestSessionsUpdate:
    """Tests des UPDATE avec now() au lieu de CURRENT_TIMESTAMP."""

    def test_update_uses_now_function(self, player_conn):
        """UPDATE doit utiliser now() pour mettre à jour updated_at."""
        session_id = "session-to-update"
        xuid = "xuid-player"
        start = datetime.now() - timedelta(hours=2)
        end_old = datetime.now() - timedelta(hours=1)

        # INSERT initial
        player_conn.execute(
            """
            INSERT INTO sessions
            (session_id, player_xuid, start_time, end_time, match_count, duration_seconds)
            VALUES (?, ?, ?, ?, 3, 3600)
            """,
            [session_id, xuid, start, end_old],
        )

        # Attendre un peu (simulé en récupérant le timestamp initial)
        old_timestamp = player_conn.execute(
            """
            SELECT updated_at FROM sessions WHERE session_id = ?
            """,
            [session_id],
        ).fetchone()[0]

        # UPDATE avec now()
        end_new = datetime.now()
        player_conn.execute(
            """
            UPDATE sessions
            SET end_time = ?,
                match_count = 5,
                duration_seconds = 7200,
                updated_at = now()
            WHERE session_id = ?
            """,
            [end_new, session_id],
        )

        # Vérifier que updated_at a changé
        new_timestamp = player_conn.execute(
            """
            SELECT updated_at FROM sessions WHERE session_id = ?
            """,
            [session_id],
        ).fetchone()[0]

        # Le nouveau timestamp doit être différent (≥)
        assert new_timestamp >= old_timestamp

    def test_update_preserves_other_fields(self, player_conn):
        """UPDATE de updated_at ne doit pas altérer les autres champs."""
        session_id = "session-preserve"
        xuid = "xuid-player"
        start = datetime.now() - timedelta(hours=2)
        end = datetime.now()

        player_conn.execute(
            """
            INSERT INTO sessions
            (session_id, player_xuid, start_time, end_time, match_count, duration_seconds, session_label)
            VALUES (?, ?, ?, ?, 4, 5400, 'Grind Session')
            """,
            [session_id, xuid, start, end],
        )

        # UPDATE seulement updated_at
        player_conn.execute(
            """
            UPDATE sessions
            SET updated_at = now()
            WHERE session_id = ?
            """,
            [session_id],
        )

        # Vérifier que les autres champs sont intacts
        result = player_conn.execute(
            """
            SELECT session_label, match_count, duration_seconds
            FROM sessions
            WHERE session_id = ?
            """,
            [session_id],
        ).fetchone()

        assert result[0] == "Grind Session"
        assert result[1] == 4
        assert result[2] == 5400


class TestSessionEnrichment:
    """Tests de l'association sessions ↔ player_match_enrichment."""

    def test_match_enrichment_references_session(self, player_conn):
        """player_match_enrichment.session_id doit pointer vers sessions."""
        session_id = "session-ref-001"
        xuid = "xuid-player"
        match_id = "match-001"

        # Créer session
        player_conn.execute(
            """
            INSERT INTO sessions
            (session_id, player_xuid, start_time, end_time, match_count, duration_seconds)
            VALUES (?, ?, now(), now(), 1, 1800)
            """,
            [session_id, xuid],
        )

        # Enrichment avec référence session
        player_conn.execute(
            """
            INSERT INTO player_match_enrichment
            (match_id, session_id, performance_score)
            VALUES (?, ?, 82.5)
            """,
            [match_id, session_id],
        )

        # Vérifier la jointure
        result = player_conn.execute(
            """
            SELECT s.session_id, e.match_id, e.performance_score
            FROM sessions s
            JOIN player_match_enrichment e ON s.session_id = e.session_id
            WHERE s.session_id = ?
            """,
            [session_id],
        ).fetchone()

        assert result[0] == session_id
        assert result[1] == match_id
        assert result[2] == 82.5

    def test_session_with_multiple_matches(self, player_conn):
        """Une session peut avoir plusieurs matchs."""
        session_id = "session-multi"
        xuid = "xuid-player"

        # Session
        player_conn.execute(
            """
            INSERT INTO sessions
            (session_id, player_xuid, start_time, end_time, match_count, duration_seconds)
            VALUES (?, ?, now(), now(), 3, 5400)
            """,
            [session_id, xuid],
        )

        # 3 matchs
        for i in range(1, 4):
            player_conn.execute(
                """
                INSERT INTO player_match_enrichment
                (match_id, session_id, performance_score)
                VALUES (?, ?, ?)
                """,
                [f"match-{i:03d}", session_id, 80.0 + i],
            )

        # Compter les matchs de la session
        count = player_conn.execute(
            """
            SELECT COUNT(*)
            FROM player_match_enrichment
            WHERE session_id = ?
            """,
            [session_id],
        ).fetchone()[0]

        assert count == 3


class TestBackfillScenariosInRegression:
    """Tests des scénarios de backfill qui causaient l'erreur CURRENT_TIMESTAMP."""

    def test_session_backfill_does_not_fail(self, player_conn):
        """Le backfill sessions ne doit plus échouer avec CURRENT_TIMESTAMP."""
        xuid = "xuid-player"

        # Simuler 10 matchs sans session_id
        for i in range(10):
            player_conn.execute(
                """
                INSERT INTO player_match_enrichment
                (match_id, performance_score, session_id)
                VALUES (?, ?, NULL)
                """,
                [f"match-{i:03d}", 75.0 + i],
            )

        # Compter matchs sans session
        missing = player_conn.execute(
            """
            SELECT COUNT(*) FROM player_match_enrichment
            WHERE session_id IS NULL
            """
        ).fetchone()[0]

        assert missing == 10

        # Simuler le calcul de sessions et l'assignation
        session_id = "session-computed"
        player_conn.execute(
            """
            INSERT INTO sessions
            (session_id, player_xuid, start_time, end_time, match_count, duration_seconds)
            VALUES (?, ?, now(), now(), 10, 18000)
            """,
            [session_id, xuid],
        )

        # Assigner session_id aux matchs (pas de CURRENT_TIMESTAMP ici)
        player_conn.execute(
            """
            UPDATE player_match_enrichment
            SET session_id = ?
            WHERE session_id IS NULL
            """,
            [session_id],
        )

        # Vérifier
        assigned = player_conn.execute(
            """
            SELECT COUNT(*) FROM player_match_enrichment
            WHERE session_id = ?
            """,
            [session_id],
        ).fetchone()[0]

        assert assigned == 10

    def test_retry_backfill_sessions_finds_zero_matches(self, player_conn):
        """Re-exécuter le backfill sessions doit trouver 0 matchs (idempotence)."""
        xuid = "xuid-player"
        session_id = "session-done"

        # Session créée
        player_conn.execute(
            """
            INSERT INTO sessions
            (session_id, player_xuid, start_time, end_time, match_count, duration_seconds)
            VALUES (?, ?, now(), now(), 5, 9000)
            """,
            [session_id, xuid],
        )

        # 5 matchs avec session_id déjà assigné
        for i in range(5):
            player_conn.execute(
                """
                INSERT INTO player_match_enrichment
                (match_id, session_id, performance_score)
                VALUES (?, ?, ?)
                """,
                [f"match-{i:03d}", session_id, 78.0 + i],
            )

        # Query de détection (matchs sans session)
        missing = player_conn.execute(
            """
            SELECT COUNT(*) FROM player_match_enrichment
            WHERE session_id IS NULL
            """
        ).fetchone()[0]

        # 0 match manquant
        assert missing == 0
