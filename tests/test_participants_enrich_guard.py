"""Tests pour le système guard complet de participants-enrich.

Ce module teste le fix du bug de boucle infinie où participants-enrich
traitait tous les matchs globalement au lieu de filtrer par xuid.
"""

from __future__ import annotations

import duckdb
import pytest

from src.data.sync.migrations import BACKFILL_FLAGS


@pytest.fixture
def shared_conn():
    """Shared DB avec match_registry et match_participants."""
    conn = duckdb.connect(":memory:")

    conn.execute("""
        CREATE TABLE match_registry (
            match_id VARCHAR PRIMARY KEY,
            backfill_completed INTEGER DEFAULT 0
        )
    """)

    conn.execute("""
        CREATE TABLE match_participants (
            match_id VARCHAR,
            xuid VARCHAR,
            kills INTEGER,
            deaths INTEGER,
            avg_life_seconds FLOAT,
            accuracy FLOAT,
            shots_fired INTEGER,
            shots_hit INTEGER,
            damage_dealt INTEGER,
            damage_taken INTEGER,
            kda FLOAT,
            score INTEGER
        )
    """)

    return conn


@pytest.fixture
def player_conn():
    """Player DB avec player_match_enrichment."""
    conn = duckdb.connect(":memory:")

    conn.execute("""
        CREATE TABLE player_match_enrichment (
            match_id VARCHAR PRIMARY KEY,
            performance_score FLOAT,
            session_id VARCHAR,
            is_with_friends BOOLEAN
        )
    """)

    return conn


class TestParticipantsEnrichGuardSystem:
    """Tests du système guard pour éviter la boucle infinie."""

    def test_xuid_filter_only_processes_player_matches(self, shared_conn):
        """Ne traiter QUE les matchs où le joueur a participé."""
        player_xuid = "xuid-player"
        other_xuid = "xuid-other"

        # 3 matchs : 1 avec le joueur, 2 sans
        shared_conn.execute("INSERT INTO match_registry VALUES ('match-001', 0)")
        shared_conn.execute("INSERT INTO match_registry VALUES ('match-002', 0)")
        shared_conn.execute("INSERT INTO match_registry VALUES ('match-003', 0)")

        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid) VALUES ('match-001', ?)",
            [player_xuid],
        )
        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid) VALUES ('match-002', ?)",
            [other_xuid],
        )
        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid) VALUES ('match-003', ?)",
            [other_xuid],
        )

        # Query avec filtre xuid
        result = shared_conn.execute(
            """
            SELECT DISTINCT m.match_id
            FROM match_registry m
            JOIN match_participants mp ON m.match_id = mp.match_id
            WHERE mp.xuid = ?
            AND (m.backfill_completed & ?) = 0
            """,
            [player_xuid, BACKFILL_FLAGS["participants"]],
        ).fetchall()

        # Seulement match-001
        assert len(result) == 1
        assert result[0][0] == "match-001"

    def test_avg_life_null_is_valid_when_deaths_zero(self, shared_conn):
        """avg_life_seconds NULL est valide quand deaths=0 (immortel)."""
        player_xuid = "xuid-player"
        match_id = "match-immortal"

        shared_conn.execute("INSERT INTO match_registry VALUES (?, 0)", [match_id])
        shared_conn.execute(
            """
            INSERT INTO match_participants
            (match_id, xuid, kills, deaths, avg_life_seconds, accuracy)
            VALUES (?, ?, 5, 0, NULL, 0.5)
            """,
            [match_id, player_xuid],
        )

        # Query de détection AVEC guard corrigé (ne doit PAS détecter)
        result = shared_conn.execute(
            """
            SELECT m.match_id
            FROM match_registry m
            JOIN match_participants mp ON m.match_id = mp.match_id
            WHERE mp.xuid = ?
            AND (m.backfill_completed & ?) = 0
            AND (
                mp.accuracy IS NULL
                OR (mp.avg_life_seconds IS NULL AND mp.deaths > 0)
            )
            """,
            [player_xuid, BACKFILL_FLAGS["participants_avg_life"]],
        ).fetchall()

        # Ne doit PAS être détecté
        assert len(result) == 0, "Match avec deaths=0 et avg_life NULL est valide"

    def test_avg_life_null_is_invalid_when_deaths_positive(self, shared_conn):
        """avg_life_seconds NULL est invalide quand deaths>0."""
        player_xuid = "xuid-player"
        match_id = "match-invalid"

        shared_conn.execute("INSERT INTO match_registry VALUES (?, 0)", [match_id])
        shared_conn.execute(
            """
            INSERT INTO match_participants
            (match_id, xuid, kills, deaths, avg_life_seconds, accuracy)
            VALUES (?, ?, 5, 3, NULL, 0.5)
            """,
            [match_id, player_xuid],
        )

        # Query avec guard (DOIT détecter)
        result = shared_conn.execute(
            """
            SELECT m.match_id
            FROM match_registry m
            JOIN match_participants mp ON m.match_id = mp.match_id
            WHERE mp.xuid = ?
            AND (m.backfill_completed & ?) = 0
            AND (
                mp.accuracy IS NULL
                OR (mp.avg_life_seconds IS NULL AND mp.deaths > 0)
            )
            """,
            [player_xuid, BACKFILL_FLAGS["participants_avg_life"]],
        ).fetchall()

        # DOIT être détecté
        assert len(result) == 1
        assert result[0][0] == match_id

    def test_bitmask_prevents_reprocessing(self, shared_conn):
        """Le bitmask doit empêcher le retraitement."""
        player_xuid = "xuid-player"
        match_id = "match-done"

        # Match avec flag participants_avg_life déjà marqué
        mask = BACKFILL_FLAGS["participants_avg_life"]
        shared_conn.execute("INSERT INTO match_registry VALUES (?, ?)", [match_id, mask])
        shared_conn.execute(
            """
            INSERT INTO match_participants
            (match_id, xuid, kills, deaths, avg_life_seconds, accuracy)
            VALUES (?, ?, 5, 3, NULL, NULL)
            """,
            [match_id, player_xuid],
        )

        # Query de détection (ne doit PAS détecter car bitmask actif)
        result = shared_conn.execute(
            """
            SELECT m.match_id
            FROM match_registry m
            JOIN match_participants mp ON m.match_id = mp.match_id
            WHERE mp.xuid = ?
            AND (m.backfill_completed & ?) = 0
            """,
            [player_xuid, BACKFILL_FLAGS["participants_avg_life"]],
        ).fetchall()

        assert len(result) == 0, "Bitmask doit bloquer la détection"

    def test_backfill_marks_bitmask_after_processing(self, shared_conn, player_conn):
        """Après backfill, le bitmask doit être marqué."""
        player_xuid = "xuid-player"
        match_id = "match-to-process"

        # Match sans flag
        shared_conn.execute("INSERT INTO match_registry VALUES (?, 0)", [match_id])
        shared_conn.execute(
            """
            INSERT INTO match_participants
            (match_id, xuid, kills, deaths, avg_life_seconds, accuracy)
            VALUES (?, ?, 5, 3, 45.5, 0.6)
            """,
            [match_id, player_xuid],
        )

        # Simuler le backfill
        player_conn.execute(
            """
            INSERT INTO player_match_enrichment (match_id, performance_score)
            VALUES (?, 85.0)
            """,
            [match_id],
        )

        # Marquer le bitmask (OR pour préserver bits existants)
        mask_to_add = (
            BACKFILL_FLAGS["participants"]
            | BACKFILL_FLAGS["participants_avg_life"]
            | BACKFILL_FLAGS["accuracy"]
        )

        shared_conn.execute(
            """
            UPDATE match_registry
            SET backfill_completed = COALESCE(backfill_completed, 0) | ?
            WHERE match_id = ?
            """,
            [mask_to_add, match_id],
        )

        # Vérifier
        result = shared_conn.execute(
            "SELECT backfill_completed FROM match_registry WHERE match_id = ?",
            [match_id],
        ).fetchone()

        assert result[0] & BACKFILL_FLAGS["participants"] != 0
        assert result[0] & BACKFILL_FLAGS["participants_avg_life"] != 0
        assert result[0] & BACKFILL_FLAGS["accuracy"] != 0

    def test_infinite_loop_scenario_prevented(self, shared_conn, player_conn):
        """Scénario complet : empêcher la boucle infinie."""
        player_xuid = "xuid-player"
        match_id = "match-loop-test"

        # Match avec données légitimes : deaths=0, avg_life NULL
        shared_conn.execute("INSERT INTO match_registry VALUES (?, 0)", [match_id])
        shared_conn.execute(
            """
            INSERT INTO match_participants
            (match_id, xuid, kills, deaths, avg_life_seconds, accuracy)
            VALUES (?, ?, 10, 0, NULL, 0.75)
            """,
            [match_id, player_xuid],
        )

        # Première détection SANS guard (BUG) : détecte à tort
        buggy_result = shared_conn.execute(
            """
            SELECT m.match_id
            FROM match_registry m
            JOIN match_participants mp ON m.match_id = mp.match_id
            WHERE mp.xuid = ?
            AND (m.backfill_completed & ?) = 0
            AND mp.avg_life_seconds IS NULL
            """,
            [player_xuid, BACKFILL_FLAGS["participants_avg_life"]],
        ).fetchall()

        assert len(buggy_result) == 1, "Sans guard, détecte incorrectement"

        # Détection AVEC guard (FIX) : ne détecte PAS
        fixed_result = shared_conn.execute(
            """
            SELECT m.match_id
            FROM match_registry m
            JOIN match_participants mp ON m.match_id = mp.match_id
            WHERE mp.xuid = ?
            AND (m.backfill_completed & ?) = 0
            AND (mp.avg_life_seconds IS NULL AND mp.deaths > 0)
            """,
            [player_xuid, BACKFILL_FLAGS["participants_avg_life"]],
        ).fetchall()

        assert len(fixed_result) == 0, "Avec guard, ne détecte plus"

        # Marquer comme traité pour éviter boucle
        shared_conn.execute(
            """
            UPDATE match_registry
            SET backfill_completed = COALESCE(backfill_completed, 0) | ?
            WHERE match_id = ?
            """,
            [BACKFILL_FLAGS["participants_avg_life"], match_id],
        )

        # Vérifier qu'on ne le redétectera plus
        redetect = shared_conn.execute(
            """
            SELECT m.match_id
            FROM match_registry m
            JOIN match_participants mp ON m.match_id = mp.match_id
            WHERE mp.xuid = ?
            AND (m.backfill_completed & ?) = 0
            """,
            [player_xuid, BACKFILL_FLAGS["participants_avg_life"]],
        ).fetchall()

        assert len(redetect) == 0, "Bitmask empêche redétection"

    def test_multiple_players_same_match_only_count_target(self, shared_conn):
        """Un match avec plusieurs joueurs ne doit compter qu'une fois."""
        player_xuid = "xuid-player"
        teammate_xuid = "xuid-teammate"
        match_id = "match-multi"

        shared_conn.execute("INSERT INTO match_registry VALUES (?, 0)", [match_id])
        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid) VALUES (?, ?)",
            [match_id, player_xuid],
        )
        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid) VALUES (?, ?)",
            [match_id, teammate_xuid],
        )

        # Query avec DISTINCT
        result = shared_conn.execute(
            """
            SELECT DISTINCT m.match_id
            FROM match_registry m
            JOIN match_participants mp ON m.match_id = mp.match_id
            WHERE mp.xuid = ?
            AND (m.backfill_completed & ?) = 0
            """,
            [player_xuid, BACKFILL_FLAGS["participants"]],
        ).fetchall()

        # Un seul match
        assert len(result) == 1
        assert result[0][0] == match_id
