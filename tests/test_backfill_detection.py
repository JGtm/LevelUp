"""Tests pour scripts/backfill/detection.py — détection des matchs avec données manquantes.

Teste find_matches_missing_data avec différents flags via shared DB (V5).
"""

from __future__ import annotations

import duckdb
import pytest

from scripts.backfill.detection import (
    _done_guard,
    _has_backfill_completed_column,
    find_matches_missing_data,
)

# ── Helpers ──────────────────────────────────────────────────────────────────

# m2 est "complet" : on met tous les bits pour l'exclure de la détection
_M2_BACKFILL_BITS = (
    1 | 2 | 4 | 8 | 16 | 32 | 64 | 128 | 256 | 512 | 1024 | 2048 | 4096 | 8192 | 16384
)


@pytest.fixture
def shared_conn():
    """Connexion DuckDB :memory: simulant shared_matches.duckdb (V5)."""
    c = duckdb.connect(":memory:")
    c.execute("""
        CREATE TABLE match_registry (
            match_id VARCHAR PRIMARY KEY,
            start_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            end_time TIMESTAMP,
            playlist_name VARCHAR,
            playlist_id VARCHAR,
            map_name VARCHAR,
            map_id VARCHAR,
            pair_name VARCHAR,
            pair_id VARCHAR,
            game_variant_name VARCHAR,
            game_variant_id VARCHAR,
            backfill_completed INTEGER DEFAULT 0
        )
    """)
    c.execute("""
        CREATE TABLE match_participants (
            match_id VARCHAR,
            xuid VARCHAR,
            rank INTEGER,
            score INTEGER,
            kills INTEGER,
            deaths INTEGER,
            assists INTEGER,
            accuracy DOUBLE,
            shots_fired INTEGER,
            shots_hit INTEGER,
            damage_dealt DOUBLE,
            damage_taken DOUBLE,
            avg_life_seconds DOUBLE,
            team_mmr DOUBLE,
            PRIMARY KEY (match_id, xuid)
        )
    """)
    c.execute("""
        CREATE TABLE medals_earned (
            match_id VARCHAR,
            medal_name_id VARCHAR,
            count INTEGER,
            xuid VARCHAR
        )
    """)
    c.execute("""
        CREATE TABLE highlight_events (
            match_id VARCHAR,
            event_type VARCHAR,
            time_ms INTEGER,
            xuid VARCHAR,
            gamertag VARCHAR
        )
    """)
    # Insert test matches in match_registry
    # m1, m3 : données vides / NULLs — m2 : complet avec backfill bits
    c.execute(
        """
        INSERT INTO match_registry (match_id, playlist_name, playlist_id,
            map_name, map_id, pair_name, pair_id,
            game_variant_name, game_variant_id, backfill_completed)
        VALUES
            ('m1', NULL, 'p1', NULL, 'map1', NULL, 'pair1', NULL, 'gv1', 0),
            ('m2', 'Ranked', 'p2', 'Recharge', 'map2', 'Slayer', 'pair2', 'Slayer', 'gv2', ?),
            ('m3', NULL, 'p3', NULL, 'map3', NULL, 'pair3', NULL, 'gv3', 0)
    """,
        [_M2_BACKFILL_BITS],
    )
    # match_participants : m1 + m3 NULLs, m2 complet
    c.execute("""
        INSERT INTO match_participants (match_id, xuid, accuracy, shots_fired, shots_hit,
            kills, deaths, assists, rank, score, damage_dealt, damage_taken, team_mmr)
        VALUES
            ('m1', '1234567890123456', NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL),
            ('m2', '1234567890123456', 0.5, 100, 50, 10, 5, 3, 1, 100, 1000.0, 800.0, 1500.0),
            ('m3', '1234567890123456', NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL)
    """)
    # m2 has medals, events
    c.execute("INSERT INTO medals_earned VALUES ('m2', 'double_kill', 1, '1234567890123456')")
    c.execute("INSERT INTO highlight_events VALUES ('m2', 'Kill', 0, '1234567890123456', 'GT')")
    yield c
    c.close()


@pytest.fixture
def player_conn():
    """Connexion DuckDB :memory: simulant la DB joueur."""
    c = duckdb.connect(":memory:")
    c.execute("""
        CREATE TABLE player_match_enrichment (
            match_id VARCHAR PRIMARY KEY,
            performance_score FLOAT
        )
    """)
    c.execute("""
        CREATE TABLE personal_score_awards (
            match_id VARCHAR,
            xuid VARCHAR,
            award_name VARCHAR
        )
    """)
    yield c
    c.close()


@pytest.fixture
def shared_conn_no_bf_col():
    """Connexion DuckDB :memory: SANS colonne backfill_completed."""
    c = duckdb.connect(":memory:")
    c.execute("""
        CREATE TABLE match_registry (
            match_id VARCHAR PRIMARY KEY,
            start_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    c.execute("""
        CREATE TABLE match_participants (
            match_id VARCHAR,
            xuid VARCHAR,
            accuracy DOUBLE,
            PRIMARY KEY (match_id, xuid)
        )
    """)
    c.execute("""
        CREATE TABLE medals_earned (
            match_id VARCHAR,
            medal_name_id VARCHAR,
            count INTEGER,
            xuid VARCHAR
        )
    """)
    c.execute("INSERT INTO match_registry (match_id) VALUES ('m1')")
    c.execute("INSERT INTO match_participants (match_id, xuid) VALUES ('m1', '1234567890123456')")
    yield c
    c.close()


# ── Tests _has_backfill_completed_column ─────────────────────────────────────


class TestHasBackfillCompletedColumn:
    def test_with_column(self, shared_conn):
        assert _has_backfill_completed_column(shared_conn) is True

    def test_without_column(self, shared_conn_no_bf_col):
        assert _has_backfill_completed_column(shared_conn_no_bf_col) is False


# ── Tests _done_guard ────────────────────────────────────────────────────────


class TestDoneGuard:
    def test_returns_empty_when_no_column(self):
        result = _done_guard("medals", False)
        assert result == ""

    def test_returns_clause_when_column_exists(self):
        result = _done_guard("medals", True)
        assert "backfill_completed" in result
        assert "& " in result
        assert "= 0" in result

    def test_unknown_flag_returns_empty(self):
        result = _done_guard("unknown_flag_xyz", True)
        assert result == ""


# ── Tests find_matches_missing_data — medals ─────────────────────────────────


class TestFindMatchesMissingMedals:
    def test_finds_matches_without_medals(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn, "1234567890123456", shared_conn=shared_conn, medals=True
        )
        # m1 and m3 have no medals
        assert "m1" in result
        assert "m3" in result
        assert "m2" not in result

    def test_force_medals_includes_all(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn, "1234567890123456", shared_conn=shared_conn, medals=True, force_medals=True
        )
        # force includes m1, m3 which have no medals (m2 garde le guard)
        assert "m1" in result
        assert "m3" in result

    def test_no_flags_returns_empty(self, shared_conn, player_conn):
        result = find_matches_missing_data(player_conn, "1234567890123456", shared_conn=shared_conn)
        assert result == []


# ── Tests find_matches_missing_data — events ─────────────────────────────────


class TestFindMatchesMissingEvents:
    def test_finds_matches_without_events(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn, "1234567890123456", shared_conn=shared_conn, events=True
        )
        assert "m1" in result
        assert "m3" in result
        assert "m2" not in result


# ── Tests find_matches_missing_data — skill ──────────────────────────────────


class TestFindMatchesMissingSkill:
    def test_finds_matches_without_skill(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn, "1234567890123456", shared_conn=shared_conn, skill=True
        )
        assert "m1" in result
        assert "m3" in result
        assert "m2" not in result


# ── Tests find_matches_missing_data — accuracy ──────────────────────────────


class TestFindMatchesMissingAccuracy:
    def test_finds_matches_with_null_accuracy(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn, "1234567890123456", shared_conn=shared_conn, accuracy=True
        )
        assert "m1" in result
        assert "m3" in result
        assert "m2" not in result

    def test_force_accuracy_includes_all(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn,
            "1234567890123456",
            shared_conn=shared_conn,
            accuracy=True,
            force_accuracy=True,
        )
        # force = 1=1 → tous les matchs (sauf m2 exclu par bitmask)
        assert "m1" in result
        assert "m3" in result


# ── Tests find_matches_missing_data — shots ──────────────────────────────────


class TestFindMatchesMissingShots:
    def test_finds_matches_with_null_shots(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn, "1234567890123456", shared_conn=shared_conn, shots=True
        )
        assert "m1" in result
        assert "m3" in result
        assert "m2" not in result

    def test_force_shots_includes_all(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn, "1234567890123456", shared_conn=shared_conn, shots=True, force_shots=True
        )
        # force = 1=1, mais m2 n'a pas le bit shots forcé → 3 résultats
        assert "m1" in result
        assert "m3" in result


# ── Tests find_matches_missing_data — personal_scores ────────────────────────


class TestFindMatchesMissingPersonalScores:
    def test_finds_matches_without_scores(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn, "1234567890123456", shared_conn=shared_conn, personal_scores=True
        )
        # V5 : condition "1=1" + bitmask guard → m1 et m3 (pas de bit)
        assert "m1" in result
        assert "m3" in result
        assert "m2" not in result


# ── Tests find_matches_missing_data — performance_scores ─────────────────────


class TestFindMatchesMissingPerformance:
    def test_finds_matches_with_null_performance(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn, "1234567890123456", shared_conn=shared_conn, performance_scores=True
        )
        # V5 : condition "1=1" + bitmask guard → m1 et m3 (pas de bit)
        assert "m1" in result
        assert "m3" in result
        assert "m2" not in result


# ── Tests find_matches_missing_data — enemy_mmr ─────────────────────────────


class TestFindMatchesMissingEnemyMmr:
    def test_finds_matches_with_null_enemy_mmr(self, shared_conn, player_conn):
        # V5 : enemy_mmr vérifie match_participants.team_mmr IS NULL
        # m1, m3 ont team_mmr NULL → détectés
        result = find_matches_missing_data(
            player_conn, "1234567890123456", shared_conn=shared_conn, enemy_mmr=True
        )
        assert "m1" in result
        assert "m3" in result
        assert "m2" not in result

    def test_force_enemy_mmr(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn,
            "1234567890123456",
            shared_conn=shared_conn,
            enemy_mmr=True,
            force_enemy_mmr=True,
        )
        # force_enemy_mmr utilise (mp.team_mmr IS NULL) → m1, m3
        assert "m1" in result
        assert "m3" in result
        assert "m2" not in result


# ── Tests find_matches_missing_data — assets ─────────────────────────────────


class TestFindMatchesMissingAssets:
    def test_finds_matches_with_null_asset_names(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn, "1234567890123456", shared_conn=shared_conn, assets=True
        )
        # m1 and m3 have NULL names with non-null IDs in match_registry
        assert "m1" in result
        assert "m3" in result
        # m2 has all names filled
        assert "m2" not in result

    def test_force_assets(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn, "1234567890123456", shared_conn=shared_conn, assets=True, force_assets=True
        )
        assert "m1" in result
        assert "m3" in result


# ── Tests find_matches_missing_data — participants ───────────────────────────


class TestFindMatchesMissingParticipants:
    def test_finds_matches_without_participants(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn, "1234567890123456", shared_conn=shared_conn, participants=True
        )
        # V5 : condition "1=1" + bitmask → m1 et m3 (pas de bit)
        assert "m1" in result
        assert "m3" in result
        assert "m2" not in result

    def test_force_participants(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn,
            "1234567890123456",
            shared_conn=shared_conn,
            participants=True,
            force_participants=True,
        )
        # force = 1=1, sans guard → tous les matchs
        assert len(result) == 3


# ── Tests find_matches_missing_data — detection_mode ─────────────────────────


class TestDetectionMode:
    def test_or_mode_default(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn,
            "1234567890123456",
            shared_conn=shared_conn,
            medals=True,
            accuracy=True,
            detection_mode="or",
        )
        # OR: matches missing medals OR accuracy
        assert "m1" in result
        assert "m3" in result

    def test_and_mode(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn,
            "1234567890123456",
            shared_conn=shared_conn,
            medals=True,
            accuracy=True,
            detection_mode="and",
        )
        # AND: matches missing medals AND accuracy
        assert "m1" in result
        assert "m3" in result
        # m2 has both medals and accuracy → excluded
        assert "m2" not in result


# ── Tests find_matches_missing_data — max_matches ────────────────────────────


class TestMaxMatches:
    def test_limits_results(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn, "1234567890123456", shared_conn=shared_conn, medals=True, max_matches=1
        )
        assert len(result) == 1


# ── Tests find_matches_missing_data — force_aliases ──────────────────────────


class TestForceAliases:
    def test_force_aliases_returns_all(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn, "1234567890123456", shared_conn=shared_conn, force_aliases=True
        )
        assert len(result) == 3


# ── Tests backfill_completed bitmask filtering ───────────────────────────────


class TestBitmaskFiltering:
    def test_already_backfilled_excluded(self, shared_conn, player_conn):
        """Les matchs avec le bit medals activé ne sont pas re-détectés."""
        from src.data.sync.migrations import BACKFILL_FLAGS

        medal_bit = BACKFILL_FLAGS.get("medals", 0)
        if medal_bit:
            shared_conn.execute(
                "UPDATE match_registry SET backfill_completed = ? WHERE match_id = 'm1'",
                [medal_bit],
            )
            result = find_matches_missing_data(
                player_conn, "1234567890123456", shared_conn=shared_conn, medals=True
            )
            # m1 should be excluded now
            assert "m1" not in result
            # m3 still missing
            assert "m3" in result


# ── Tests sans colonne backfill_completed ────────────────────────────────────


class TestWithoutBackfillColumn:
    def test_medals_works_without_bf_column(self, shared_conn_no_bf_col):
        """Détection fonctionne même sans colonne backfill_completed."""
        shared_conn_no_bf_col.execute("""
            CREATE TABLE IF NOT EXISTS highlight_events (
                match_id VARCHAR, event_type VARCHAR, time_ms INTEGER,
                xuid VARCHAR, gamertag VARCHAR
            )
        """)
        player_conn = duckdb.connect(":memory:")
        try:
            result = find_matches_missing_data(
                player_conn, "1234567890123456", shared_conn=shared_conn_no_bf_col, medals=True
            )
            # m1 has no medals
            assert "m1" in result
        finally:
            player_conn.close()


# ── Tests participants column conditions ─────────────────────────────────────


class TestParticipantsColumnConditions:
    def test_participants_kda(self, shared_conn, player_conn):
        # m1 already has NULL kills → détecté
        result = find_matches_missing_data(
            player_conn, "1234567890123456", shared_conn=shared_conn, participants_kda=True
        )
        assert "m1" in result
        assert "m3" in result
        assert "m2" not in result

    def test_participants_shots(self, shared_conn, player_conn):
        # m1 already has NULL shots_fired → détecté
        result = find_matches_missing_data(
            player_conn, "1234567890123456", shared_conn=shared_conn, participants_shots=True
        )
        assert "m1" in result
        assert "m3" in result
        assert "m2" not in result

    def test_force_participants_shots(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn,
            "1234567890123456",
            shared_conn=shared_conn,
            participants_shots=True,
            force_participants_shots=True,
        )
        # force = 1=1 → all matches
        assert len(result) == 3

    def test_participants_damage(self, shared_conn, player_conn):
        # m1 already has NULL damage_dealt → détecté
        result = find_matches_missing_data(
            player_conn, "1234567890123456", shared_conn=shared_conn, participants_damage=True
        )
        assert "m1" in result
        assert "m3" in result
        assert "m2" not in result

    def test_force_participants_damage(self, shared_conn, player_conn):
        result = find_matches_missing_data(
            player_conn,
            "1234567890123456",
            shared_conn=shared_conn,
            participants_damage=True,
            force_participants_damage=True,
        )
        assert len(result) == 3

    def test_participants_scores(self, shared_conn, player_conn):
        # m1 already has NULL rank → détecté
        result = find_matches_missing_data(
            player_conn, "1234567890123456", shared_conn=shared_conn, participants_scores=True
        )
        assert "m1" in result
        assert "m3" in result
        assert "m2" not in result


# ── Tests pour détection dans shared DB (v5) — données participants ──────────


@pytest.fixture
def shared_conn_participants():
    """Shared DB dédiée aux tests participants (données différentes du fixture principal)."""
    c = duckdb.connect(":memory:")

    c.execute("""
        CREATE TABLE match_registry (
            match_id VARCHAR PRIMARY KEY,
            end_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            backfill_completed INTEGER DEFAULT 0
        )
    """)
    c.execute("""
        CREATE TABLE match_participants (
            match_id VARCHAR,
            xuid VARCHAR,
            gamertag VARCHAR,
            rank INTEGER,
            score INTEGER,
            kills INTEGER,
            deaths INTEGER,
            assists INTEGER,
            shots_fired INTEGER,
            shots_hit INTEGER,
            damage_dealt DOUBLE,
            damage_taken DOUBLE,
            avg_life_seconds DOUBLE
        )
    """)

    c.execute("""
        INSERT INTO match_registry (match_id, backfill_completed)
        VALUES ('m1', 0), ('m2', 0), ('m3', 0)
    """)
    # m1: NULL avg_life, shots/damage renseignés
    c.execute("""
        INSERT INTO match_participants
        (match_id, xuid, gamertag, kills, deaths, assists,
         shots_fired, shots_hit, damage_dealt, damage_taken, avg_life_seconds)
        VALUES ('m1', 'player123', 'TestPlayer', 10, 5, 3,
                200, 80, 1500.0, 1200.0, NULL)
    """)
    # m2: tout renseigné
    c.execute("""
        INSERT INTO match_participants
        (match_id, xuid, gamertag, kills, deaths, assists,
         shots_fired, shots_hit, damage_dealt, damage_taken, avg_life_seconds)
        VALUES ('m2', 'player123', 'TestPlayer', 15, 8, 5,
                300, 120, 2000.0, 1800.0, 45.2)
    """)
    # m3: NULL shots, avg_life renseigné
    c.execute("""
        INSERT INTO match_participants
        (match_id, xuid, gamertag, kills, deaths, assists, shots_fired, shots_hit, avg_life_seconds)
        VALUES ('m3', 'player123', 'TestPlayer', 12, 6, 4, NULL, NULL, 28.5)
    """)
    c.commit()
    yield c
    c.close()


class TestSharedDBDetection:
    """Tests pour _find_matches_in_shared_db (v5)."""

    def test_participants_avg_life_null(self, shared_conn_participants):
        from scripts.backfill.detection import _find_matches_in_shared_db

        result = _find_matches_in_shared_db(
            shared_conn=shared_conn_participants,
            xuid="player123",
            max_matches=None,
            participants_scores=False,
            participants_kda=False,
            participants_shots=False,
            participants_damage=False,
            participants_avg_life=True,
            force_participants_shots=False,
            force_participants_damage=False,
            force_participants_avg_life=False,
        )
        assert "m1" in result
        assert "m2" not in result
        assert "m3" not in result

    def test_participants_shots_null(self, shared_conn_participants):
        from scripts.backfill.detection import _find_matches_in_shared_db

        result = _find_matches_in_shared_db(
            shared_conn=shared_conn_participants,
            xuid="player123",
            max_matches=None,
            participants_scores=False,
            participants_kda=False,
            participants_shots=True,
            participants_damage=False,
            participants_avg_life=False,
            force_participants_shots=False,
            force_participants_damage=False,
            force_participants_avg_life=False,
        )
        assert "m3" in result
        assert "m1" not in result
        assert "m2" not in result

    def test_force_participants_avg_life(self, shared_conn_participants):
        from scripts.backfill.detection import _find_matches_in_shared_db

        result = _find_matches_in_shared_db(
            shared_conn=shared_conn_participants,
            xuid="player123",
            max_matches=None,
            participants_scores=False,
            participants_kda=False,
            participants_shots=False,
            participants_damage=False,
            participants_avg_life=True,
            force_participants_shots=False,
            force_participants_damage=False,
            force_participants_avg_life=True,
        )
        assert len(result) == 3

    def test_max_matches_limit(self, shared_conn_participants):
        from scripts.backfill.detection import _find_matches_in_shared_db

        result = _find_matches_in_shared_db(
            shared_conn=shared_conn_participants,
            xuid="player123",
            max_matches=1,
            participants_scores=False,
            participants_kda=False,
            participants_shots=False,
            participants_damage=False,
            participants_avg_life=True,
            force_participants_shots=False,
            force_participants_damage=False,
            force_participants_avg_life=True,
        )
        assert len(result) == 1

    def test_multiple_conditions_or(self, shared_conn_participants):
        from scripts.backfill.detection import _find_matches_in_shared_db

        result = _find_matches_in_shared_db(
            shared_conn=shared_conn_participants,
            xuid="player123",
            max_matches=None,
            participants_scores=False,
            participants_kda=False,
            participants_shots=True,
            participants_damage=False,
            participants_avg_life=True,
            force_participants_shots=False,
            force_participants_damage=False,
            force_participants_avg_life=False,
        )
        assert "m1" in result
        assert "m3" in result
        assert "m2" not in result


class TestSharedConnIntegration:
    """Tests d'intégration find_matches_missing_data avec shared_conn."""

    def test_participants_only_uses_shared_conn(self, shared_conn_participants):
        """Quand shared_conn fourni + participants-only → utilise shared DB."""
        local_conn = duckdb.connect(":memory:")
        try:
            result = find_matches_missing_data(
                conn=local_conn,
                xuid="player123",
                shared_conn=shared_conn_participants,
                participants_avg_life=True,
            )
            assert "m1" in result
        finally:
            local_conn.close()

    def test_mixed_flags_includes_shared_participants(self, shared_conn_participants):
        """Flags mixtes : les participants sont détectés via shared DB."""
        local_conn = duckdb.connect(":memory:")
        try:
            result = find_matches_missing_data(
                conn=local_conn,
                xuid="player123",
                shared_conn=shared_conn_participants,
                participants_avg_life=True,
            )
            assert "m1" in result
        finally:
            local_conn.close()
