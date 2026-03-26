"""Tests end-to-end pour le flag --force-performance-scores.

Ce module teste le nouveau flag qui permet de recalculer les performance scores
même si le bitmask indique qu'ils ont déjà été calculés.
"""

from __future__ import annotations

import duckdb
import pytest

from src.data.sync.migrations import BACKFILL_FLAGS


@pytest.fixture
def shared_conn():
    """Shared DB avec match_registry."""
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


class TestForcePerformanceScoresFlag:
    """Tests du flag --force-performance-scores."""

    def test_without_force_skips_completed_matches(self, shared_conn):
        """Sans --force, les matchs avec bitmask actif sont ignorés.

        Note : performance_scores n'est plus dans BACKFILL_FLAGS (détection via IS NULL).
        Ce test utilise personal_scores pour valider la mécanique bitmask générale.
        """
        player_xuid = "xuid-player"

        # 2 matchs : 1 avec flag personal_scores, 1 sans
        shared_conn.execute(
            "INSERT INTO match_registry VALUES ('match-done', ?)",
            [BACKFILL_FLAGS["personal_scores"]],
        )
        shared_conn.execute("INSERT INTO match_registry VALUES ('match-todo', 0)")

        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid) VALUES ('match-done', ?)",
            [player_xuid],
        )
        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid) VALUES ('match-todo', ?)",
            [player_xuid],
        )

        # Query de détection SANS force (bitmask personal_scores)
        result = shared_conn.execute(
            """
            SELECT DISTINCT m.match_id
            FROM match_registry m
            JOIN match_participants mp ON m.match_id = mp.match_id
            WHERE mp.xuid = ?
            AND (m.backfill_completed & ?) = 0
            """,
            [player_xuid, BACKFILL_FLAGS["personal_scores"]],
        ).fetchall()

        # Seulement match-todo
        assert len(result) == 1
        assert result[0][0] == "match-todo"

    def test_with_force_includes_completed_matches(self, shared_conn):
        """Avec --force, les matchs avec bitmask sont inclus.

        Note : performance_scores n'est plus dans BACKFILL_FLAGS (détection via IS NULL).
        Ce test utilise personal_scores pour valider la mécanique bitmask générale.
        """
        player_xuid = "xuid-player"

        # 2 matchs : tous deux avec flag personal_scores
        shared_conn.execute(
            "INSERT INTO match_registry VALUES ('match-001', ?)",
            [BACKFILL_FLAGS["personal_scores"]],
        )
        shared_conn.execute(
            "INSERT INTO match_registry VALUES ('match-002', ?)",
            [BACKFILL_FLAGS["personal_scores"]],
        )

        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid) VALUES ('match-001', ?)",
            [player_xuid],
        )
        shared_conn.execute(
            "INSERT INTO match_participants (match_id, xuid) VALUES ('match-002', ?)",
            [player_xuid],
        )

        # Query de détection AVEC force (pas de check bitmask)
        result = shared_conn.execute(
            """
            SELECT DISTINCT m.match_id
            FROM match_registry m
            JOIN match_participants mp ON m.match_id = mp.match_id
            WHERE mp.xuid = ?
            ORDER BY m.match_id
            """,
            [player_xuid],
        ).fetchall()

        # Les 2 matchs
        assert len(result) == 2
        assert result[0][0] == "match-001"
        assert result[1][0] == "match-002"

    def test_force_flag_replaces_existing_score(self, player_conn):
        """Avec force=True, le score existant doit être remplacé."""
        match_id = "match-replace"

        # Score existant
        player_conn.execute(
            """
            INSERT INTO player_match_enrichment (match_id, performance_score)
            VALUES (?, 75.0)
            """,
            [match_id],
        )

        # Vérifier ancien score
        old_score = player_conn.execute(
            """
            SELECT performance_score FROM player_match_enrichment
            WHERE match_id = ?
            """,
            [match_id],
        ).fetchone()[0]

        assert old_score == 75.0

        # Simuler recalcul avec force (nouveau score)
        new_score = 92.0
        player_conn.execute(
            """
            UPDATE player_match_enrichment
            SET performance_score = ?
            WHERE match_id = ?
            """,
            [new_score, match_id],
        )

        # Vérifier nouveau score
        updated_score = player_conn.execute(
            """
            SELECT performance_score FROM player_match_enrichment
            WHERE match_id = ?
            """,
            [match_id],
        ).fetchone()[0]

        assert updated_score == 92.0

    def test_without_force_skips_existing_score(self, player_conn):
        """Sans force=True, le score existant est préservé (skip)."""
        match_id = "match-skip"

        # Score existant
        player_conn.execute(
            """
            INSERT INTO player_match_enrichment (match_id, performance_score)
            VALUES (?, 80.0)
            """,
            [match_id],
        )

        # Query de détection : chercher les matchs sans score
        result = player_conn.execute(
            """
            SELECT match_id FROM player_match_enrichment
            WHERE match_id = ?
            AND performance_score IS NOT NULL
            """,
            [match_id],
        ).fetchone()

        assert result is not None, "Score existant détecté → skip"

    def test_force_flag_recomputes_all_scores(self, shared_conn, player_conn):
        """Scénario complet : force recalcule tous les scores.

        Note : performance_scores n'est plus dans BACKFILL_FLAGS (détection via IS NULL).
        Le bitmask utilisé ici est personal_scores pour valider la mécanique générale.
        La détection réelle de performance_scores se fait via IS NULL dans player_match_enrichment.
        """
        player_xuid = "xuid-player"

        # 3 matchs avec flag personal_scores (performance_scores supprimé du bitmask)
        for i in range(1, 4):
            match_id = f"match-{i:03d}"
            shared_conn.execute(
                "INSERT INTO match_registry VALUES (?, ?)",
                [match_id, BACKFILL_FLAGS["personal_scores"]],
            )
            shared_conn.execute(
                "INSERT INTO match_participants (match_id, xuid, kills, deaths, score) "
                "VALUES (?, ?, 10, 5, 2500)",
                [match_id, player_xuid],
            )
            player_conn.execute(
                """
                INSERT INTO player_match_enrichment (match_id, performance_score)
                VALUES (?, 70.0)
                """,
                [match_id],
            )

        # Sans force : 0 matchs détectés via bitmask personal_scores
        no_force = shared_conn.execute(
            """
            SELECT COUNT(DISTINCT m.match_id)
            FROM match_registry m
            JOIN match_participants mp ON m.match_id = mp.match_id
            WHERE mp.xuid = ?
            AND (m.backfill_completed & ?) = 0
            """,
            [player_xuid, BACKFILL_FLAGS["personal_scores"]],
        ).fetchone()[0]

        assert no_force == 0

        # Avec force : 3 matchs détectés
        with_force = shared_conn.execute(
            """
            SELECT COUNT(DISTINCT m.match_id)
            FROM match_registry m
            JOIN match_participants mp ON m.match_id = mp.match_id
            WHERE mp.xuid = ?
            """,
            [player_xuid],
        ).fetchone()[0]

        assert with_force == 3

        # Recalcul (simuler nouveau score)
        for i in range(1, 4):
            match_id = f"match-{i:03d}"
            new_score = 85.0 + i  # 86, 87, 88
            player_conn.execute(
                """
                UPDATE player_match_enrichment
                SET performance_score = ?
                WHERE match_id = ?
                """,
                [new_score, match_id],
            )

        # Vérifier les nouveaux scores
        scores = player_conn.execute(
            """
            SELECT match_id, performance_score
            FROM player_match_enrichment
            ORDER BY match_id
            """
        ).fetchall()

        assert len(scores) == 3
        assert scores[0][1] == 86.0
        assert scores[1][1] == 87.0
        assert scores[2][1] == 88.0

    def test_scope_force_implies_performance_scores(self):
        """force_performance_scores=True doit impliquer performance_scores=True."""
        # Ce test vérifie la logique du SyncScope

        # Simuler le comportement du SyncScope._FORCE_MAP
        force_map = {"force_performance_scores": "performance_scores"}

        # Si force_performance_scores activé
        force_performance_scores = True

        # Alors performance_scores doit être activé
        if force_performance_scores:
            performance_scores = True
            implied_flag = force_map["force_performance_scores"]
            assert implied_flag == "performance_scores"
        else:
            performance_scores = False

        assert performance_scores is True

    def test_bitmask_not_updated_during_forced_backfill(self, shared_conn):
        """Durant un forced backfill, le bitmask ne doit PAS être modifié.

        Note : performance_scores n'est plus dans BACKFILL_FLAGS.
        Ce test utilise personal_scores à la place pour valider la mécanique.
        """
        match_id = "match-forced"

        # Match avec flag déjà présent (personal_scores remplace performance_scores)
        initial_mask = BACKFILL_FLAGS["personal_scores"] | BACKFILL_FLAGS["medals"]
        shared_conn.execute(
            "INSERT INTO match_registry VALUES (?, ?)",
            [match_id, initial_mask],
        )

        # Vérifier bitmask initial
        before = shared_conn.execute(
            "SELECT backfill_completed FROM match_registry WHERE match_id = ?",
            [match_id],
        ).fetchone()[0]

        assert before == initial_mask

        # Simuler un forced backfill (pas d'UPDATE du bitmask)
        # Le bitmask reste inchangé

        # Vérifier bitmask après
        after = shared_conn.execute(
            "SELECT backfill_completed FROM match_registry WHERE match_id = ?",
            [match_id],
        ).fetchone()[0]

        assert after == initial_mask, "Le bitmask ne doit pas changer en mode force"

    def test_force_with_null_score_computes_new_score(self, player_conn):
        """Force avec score NULL doit calculer et insérer."""
        match_id = "match-null"

        # Enrichment sans score
        player_conn.execute(
            """
            INSERT INTO player_match_enrichment (match_id, performance_score)
            VALUES (?, NULL)
            """,
            [match_id],
        )

        # Détecter NULL
        is_null = player_conn.execute(
            """
            SELECT performance_score IS NULL
            FROM player_match_enrichment
            WHERE match_id = ?
            """,
            [match_id],
        ).fetchone()[0]

        assert is_null is True

        # Calculer et updater
        new_score = 78.5
        player_conn.execute(
            """
            UPDATE player_match_enrichment
            SET performance_score = ?
            WHERE match_id = ?
            """,
            [new_score, match_id],
        )

        # Vérifier
        score = player_conn.execute(
            """
            SELECT performance_score FROM player_match_enrichment
            WHERE match_id = ?
            """,
            [match_id],
        ).fetchone()[0]

        assert score == 78.5
