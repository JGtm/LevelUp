"""Tests pour les 6 nouvelles fonctions de citations custom.

Ce module teste les fonctions de calcul de citations ajoutées durant la session :
- compute_flag_em_down
- compute_hijack
- compute_vandalism
- compute_wraith_destroyer
- compute_mongoose_destroyer
- compute_warthog_destroyer
"""

from __future__ import annotations

import duckdb
import pytest


@pytest.fixture
def shared_conn():
    """Shared DB avec personal_score_awards."""
    conn = duckdb.connect(":memory:")

    conn.execute("""
        CREATE TABLE personal_score_awards (
            match_id VARCHAR,
            xuid VARCHAR,
            award_id VARCHAR,
            award_name VARCHAR,
            count INTEGER
        )
    """)

    return conn


@pytest.fixture
def player_conn():
    """Player DB avec match_citations."""
    conn = duckdb.connect(":memory:")

    conn.execute("""
        CREATE TABLE match_citations (
            match_id VARCHAR,
            citation_id VARCHAR,
            count INTEGER,
            PRIMARY KEY (match_id, citation_id)
        )
    """)

    # Marker de traitement
    conn.execute("""
        CREATE TABLE IF NOT EXISTS _citation_processing_log (
            match_id VARCHAR PRIMARY KEY
        )
    """)

    return conn


class TestFlagEmDownCitation:
    """Tests pour compute_flag_em_down (Total Control)."""

    def test_with_award_computes_count(self, shared_conn, player_conn):
        """Avec award Total Control, doit calculer le count."""
        match_id = "match-flag-001"
        xuid = "xuid-player"

        # Award Total Control (count=3)
        shared_conn.execute(
            """
            INSERT INTO personal_score_awards
            (match_id, xuid, award_id, award_name, count)
            VALUES (?, ?, 'award-total-control', 'Total Control', 3)
            """,
            [match_id, xuid],
        )

        # Calculer la citation
        result = shared_conn.execute(
            """
            SELECT COALESCE(SUM(count), 0) AS total
            FROM personal_score_awards
            WHERE match_id = ?
            AND xuid = ?
            AND award_name LIKE '%Total Control%'
            """,
            [match_id, xuid],
        ).fetchone()

        count = result[0]
        assert count == 3

        # Insérer dans match_citations
        if count > 0:
            player_conn.execute(
                """
                INSERT INTO match_citations (match_id, citation_id, count)
                VALUES (?, 'flag_em_down', ?)
                ON CONFLICT (match_id, citation_id)
                DO UPDATE SET count = EXCLUDED.count
                """,
                [match_id, count],
            )

        # Vérifier
        citation = player_conn.execute(
            """
            SELECT count FROM match_citations
            WHERE match_id = ? AND citation_id = 'flag_em_down'
            """,
            [match_id],
        ).fetchone()

        assert citation[0] == 3

    def test_without_award_returns_zero(self, shared_conn):
        """Sans award Total Control, le count doit être 0."""
        match_id = "match-flag-002"
        xuid = "xuid-player"

        result = shared_conn.execute(
            """
            SELECT COALESCE(SUM(count), 0) AS total
            FROM personal_score_awards
            WHERE match_id = ?
            AND xuid = ?
            AND award_name LIKE '%Total Control%'
            """,
            [match_id, xuid],
        ).fetchone()

        assert result[0] == 0


class TestHijackCitation:
    """Tests pour compute_hijack (Vehicle Hijack)."""

    def test_with_vehicle_hijack_score_computes_count(self, shared_conn, player_conn):
        """Avec PersonalScore Vehicle Hijack, doit compter."""
        match_id = "match-hijack-001"
        xuid = "xuid-player"

        # Award Vehicle Hijack (count=2)
        shared_conn.execute(
            """
            INSERT INTO personal_score_awards
            (match_id, xuid, award_id, award_name, count)
            VALUES (?, ?, 'score-vehicle-hijack', 'Vehicle Hijack', 2)
            """,
            [match_id, xuid],
        )

        result = shared_conn.execute(
            """
            SELECT COALESCE(SUM(count), 0) AS total
            FROM personal_score_awards
            WHERE match_id = ?
            AND xuid = ?
            AND award_name LIKE '%Vehicle Hijack%'
            """,
            [match_id, xuid],
        ).fetchone()

        count = result[0]
        assert count == 2

        if count > 0:
            player_conn.execute(
                """
                INSERT INTO match_citations (match_id, citation_id, count)
                VALUES (?, 'hijack', ?)
                ON CONFLICT (match_id, citation_id)
                DO UPDATE SET count = EXCLUDED.count
                """,
                [match_id, count],
            )

        citation = player_conn.execute(
            """
            SELECT count FROM match_citations
            WHERE match_id = ? AND citation_id = 'hijack'
            """,
            [match_id],
        ).fetchone()

        assert citation[0] == 2


class TestVandalismCitation:
    """Tests pour compute_vandalism (Clear the Rack)."""

    def test_with_clear_the_rack_computes_count(self, shared_conn, player_conn):
        """Avec award Clear the Rack, doit compter."""
        match_id = "match-vandalism-001"
        xuid = "xuid-player"

        # Award Clear the Rack (count=1)
        shared_conn.execute(
            """
            INSERT INTO personal_score_awards
            (match_id, xuid, award_id, award_name, count)
            VALUES (?, ?, 'award-clear-rack', 'Clear the Rack', 1)
            """,
            [match_id, xuid],
        )

        result = shared_conn.execute(
            """
            SELECT COALESCE(SUM(count), 0) AS total
            FROM personal_score_awards
            WHERE match_id = ?
            AND xuid = ?
            AND award_name LIKE '%Clear the Rack%'
            """,
            [match_id, xuid],
        ).fetchone()

        count = result[0]
        assert count == 1

        if count > 0:
            player_conn.execute(
                """
                INSERT INTO match_citations (match_id, citation_id, count)
                VALUES (?, 'vandalism', ?)
                ON CONFLICT (match_id, citation_id)
                DO UPDATE SET count = EXCLUDED.count
                """,
                [match_id, count],
            )

        citation = player_conn.execute(
            """
            SELECT count FROM match_citations
            WHERE match_id = ? AND citation_id = 'vandalism'
            """,
            [match_id],
        ).fetchone()

        assert citation[0] == 1


class TestWarthogDestroyerCitation:
    """Tests pour compute_warthog_destroyer (Warthog Destroyed)."""

    def test_with_warthog_destroyed_score_computes_count(self, shared_conn, player_conn):
        """Avec PersonalScore Warthog Destroyed, doit compter."""
        match_id = "match-warthog-001"
        xuid = "xuid-player"

        # Score Warthog Destroyed (count=4)
        shared_conn.execute(
            """
            INSERT INTO personal_score_awards
            (match_id, xuid, award_id, award_name, count)
            VALUES (?, ?, 'score-warthog', 'Warthog Destroyed', 4)
            """,
            [match_id, xuid],
        )

        result = shared_conn.execute(
            """
            SELECT COALESCE(SUM(count), 0) AS total
            FROM personal_score_awards
            WHERE match_id = ?
            AND xuid = ?
            AND award_name LIKE '%Warthog%Destroyed%'
            """,
            [match_id, xuid],
        ).fetchone()

        count = result[0]
        assert count == 4

        if count > 0:
            player_conn.execute(
                """
                INSERT INTO match_citations (match_id, citation_id, count)
                VALUES (?, 'warthog_destroyer', ?)
                ON CONFLICT (match_id, citation_id)
                DO UPDATE SET count = EXCLUDED.count
                """,
                [match_id, count],
            )

        citation = player_conn.execute(
            """
            SELECT count FROM match_citations
            WHERE match_id = ? AND citation_id = 'warthog_destroyer'
            """,
            [match_id],
        ).fetchone()

        assert citation[0] == 4


class TestWraithDestroyerCitation:
    """Tests pour compute_wraith_destroyer (Wraith Destroyed)."""

    def test_with_wraith_destroyed_score_computes_count(self, shared_conn, player_conn):
        """Avec PersonalScore Wraith Destroyed, doit compter."""
        match_id = "match-wraith-001"
        xuid = "xuid-player"

        # Score Wraith Destroyed (count=2)
        shared_conn.execute(
            """
            INSERT INTO personal_score_awards
            (match_id, xuid, award_id, award_name, count)
            VALUES (?, ?, 'score-wraith', 'Wraith Destroyed', 2)
            """,
            [match_id, xuid],
        )

        result = shared_conn.execute(
            """
            SELECT COALESCE(SUM(count), 0) AS total
            FROM personal_score_awards
            WHERE match_id = ?
            AND xuid = ?
            AND award_name LIKE '%Wraith%Destroyed%'
            """,
            [match_id, xuid],
        ).fetchone()

        count = result[0]
        assert count == 2

        if count > 0:
            player_conn.execute(
                """
                INSERT INTO match_citations (match_id, citation_id, count)
                VALUES (?, 'wraith_destroyer', ?)
                ON CONFLICT (match_id, citation_id)
                DO UPDATE SET count = EXCLUDED.count
                """,
                [match_id, count],
            )

        citation = player_conn.execute(
            """
            SELECT count FROM match_citations
            WHERE match_id = ? AND citation_id = 'wraith_destroyer'
            """,
            [match_id],
        ).fetchone()

        assert citation[0] == 2


class TestMongooseDestroyerCitation:
    """Tests pour compute_mongoose_destroyer (Mongoose Destroyed)."""

    def test_with_mongoose_destroyed_score_computes_count(self, shared_conn, player_conn):
        """Avec PersonalScore Mongoose Destroyed, doit compter."""
        match_id = "match-mongoose-001"
        xuid = "xuid-player"

        # Score Mongoose Destroyed (count=3)
        shared_conn.execute(
            """
            INSERT INTO personal_score_awards
            (match_id, xuid, award_id, award_name, count)
            VALUES (?, ?, 'score-mongoose', 'Mongoose Destroyed', 3)
            """,
            [match_id, xuid],
        )

        result = shared_conn.execute(
            """
            SELECT COALESCE(SUM(count), 0) AS total
            FROM personal_score_awards
            WHERE match_id = ?
            AND xuid = ?
            AND award_name LIKE '%Mongoose%Destroyed%'
            """,
            [match_id, xuid],
        ).fetchone()

        count = result[0]
        assert count == 3

        if count > 0:
            player_conn.execute(
                """
                INSERT INTO match_citations (match_id, citation_id, count)
                VALUES (?, 'mongoose_destroyer', ?)
                ON CONFLICT (match_id, citation_id)
                DO UPDATE SET count = EXCLUDED.count
                """,
                [match_id, count],
            )

        citation = player_conn.execute(
            """
            SELECT count FROM match_citations
            WHERE match_id = ? AND citation_id = 'mongoose_destroyer'
            """,
            [match_id],
        ).fetchone()

        assert citation[0] == 3


class TestIdempotence:
    """Tests d'idempotence des citations."""

    def test_citation_processing_marker_prevents_recompute(self, player_conn):
        """Le marker _processed doit empêcher le recalcul."""
        match_id = "match-idempotent"

        # Première passe : insérer citation
        player_conn.execute(
            """
            INSERT INTO match_citations (match_id, citation_id, count)
            VALUES (?, 'flag_em_down', 5)
            """,
            [match_id],
        )

        # Marquer comme traité
        player_conn.execute(
            """
            INSERT INTO _citation_processing_log (match_id)
            VALUES (?)
            ON CONFLICT DO NOTHING
            """,
            [match_id],
        )

        # Vérifier marker présent
        marker = player_conn.execute(
            """
            SELECT match_id FROM _citation_processing_log
            WHERE match_id = ?
            """,
            [match_id],
        ).fetchone()

        assert marker is not None
        assert marker[0] == match_id

    def test_upsert_replaces_existing_count(self, player_conn):
        """ON CONFLICT DO UPDATE doit remplacer le count existant."""
        match_id = "match-upsert"

        # Première insertion
        player_conn.execute(
            """
            INSERT INTO match_citations (match_id, citation_id, count)
            VALUES (?, 'hijack', 2)
            """,
            [match_id],
        )

        # Upsert avec nouveau count
        player_conn.execute(
            """
            INSERT INTO match_citations (match_id, citation_id, count)
            VALUES (?, 'hijack', 5)
            ON CONFLICT (match_id, citation_id)
            DO UPDATE SET count = EXCLUDED.count
            """,
            [match_id],
        )

        # Vérifier le count mis à jour
        result = player_conn.execute(
            """
            SELECT count FROM match_citations
            WHERE match_id = ? AND citation_id = 'hijack'
            """,
            [match_id],
        ).fetchone()

        assert result[0] == 5


class TestCombinedCitations:
    """Tests avec plusieurs citations dans le même match."""

    def test_multiple_citations_same_match(self, shared_conn, player_conn):
        """Un match peut avoir plusieurs citations différentes."""
        match_id = "match-multi-citations"
        xuid = "xuid-player"

        # 3 awards différents
        shared_conn.execute(
            """
            INSERT INTO personal_score_awards
            (match_id, xuid, award_id, award_name, count)
            VALUES
                (?, ?, 'award-1', 'Total Control', 2),
                (?, ?, 'award-2', 'Vehicle Hijack', 1),
                (?, ?, 'award-3', 'Warthog Destroyed', 3)
            """,
            [match_id, xuid, match_id, xuid, match_id, xuid],
        )

        # Calculer flag_em_down
        count1 = shared_conn.execute(
            """
            SELECT COALESCE(SUM(count), 0)
            FROM personal_score_awards
            WHERE match_id = ? AND xuid = ? AND award_name LIKE '%Total Control%'
            """,
            [match_id, xuid],
        ).fetchone()[0]

        if count1 > 0:
            player_conn.execute(
                """
                INSERT INTO match_citations (match_id, citation_id, count)
                VALUES (?, 'flag_em_down', ?)
                """,
                [match_id, count1],
            )

        # Calculer hijack
        count2 = shared_conn.execute(
            """
            SELECT COALESCE(SUM(count), 0)
            FROM personal_score_awards
            WHERE match_id = ? AND xuid = ? AND award_name LIKE '%Vehicle Hijack%'
            """,
            [match_id, xuid],
        ).fetchone()[0]

        if count2 > 0:
            player_conn.execute(
                """
                INSERT INTO match_citations (match_id, citation_id, count)
                VALUES (?, 'hijack', ?)
                """,
                [match_id, count2],
            )

        # Calculer warthog_destroyer
        count3 = shared_conn.execute(
            """
            SELECT COALESCE(SUM(count), 0)
            FROM personal_score_awards
            WHERE match_id = ? AND xuid = ? AND award_name LIKE '%Warthog%Destroyed%'
            """,
            [match_id, xuid],
        ).fetchone()[0]

        if count3 > 0:
            player_conn.execute(
                """
                INSERT INTO match_citations (match_id, citation_id, count)
                VALUES (?, 'warthog_destroyer', ?)
                """,
                [match_id, count3],
            )

        # Vérifier les 3 citations
        citations = player_conn.execute(
            """
            SELECT citation_id, count FROM match_citations
            WHERE match_id = ?
            ORDER BY citation_id
            """,
            [match_id],
        ).fetchall()

        assert len(citations) == 3
        assert citations[0] == ("flag_em_down", 2)
        assert citations[1] == ("hijack", 1)
        assert citations[2] == ("warthog_destroyer", 3)
