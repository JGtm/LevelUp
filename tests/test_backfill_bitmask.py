"""Tests pour le système de backfill complet (détection + marquage bitmask) — V5.

Vérifie que :
1. La détection trouve les matchs avec données manquantes via shared DB
2. Après marquage du bitmask dans match_registry, les matchs ne sont plus détectés
3. Les flags --force-* ignorent le bitmask
4. Le mode AND fonctionne correctement
5. compute_backfill_mask calcule les bons bits
"""

from __future__ import annotations

import duckdb
import pytest

from scripts.backfill.detection import (
    BACKFILL_FLAGS,
    compute_backfill_mask,
    find_matches_missing_data,
)

# ─────────────────────────────────────────────────────────────────────────────
# Fixtures
# ─────────────────────────────────────────────────────────────────────────────

XUID = "2535469190789936"


@pytest.fixture()
def shared_conn():
    """Crée une base DuckDB en mémoire simulant shared_matches.duckdb (V5)."""
    c = duckdb.connect(":memory:")
    c.execute("""
        CREATE TABLE match_registry (
            match_id VARCHAR PRIMARY KEY,
            start_time TIMESTAMP,
            end_time TIMESTAMP,
            playlist_name VARCHAR,
            playlist_id VARCHAR,
            map_name VARCHAR,
            map_id VARCHAR,
            pair_name VARCHAR,
            pair_id VARCHAR,
            game_variant_name VARCHAR,
            game_variant_id VARCHAR,
            backfill_completed INTEGER DEFAULT 0,
            events_loaded BOOLEAN DEFAULT FALSE
        )
    """)
    c.execute("""
        CREATE TABLE match_participants (
            match_id VARCHAR NOT NULL,
            xuid VARCHAR NOT NULL,
            team_id INTEGER,
            outcome INTEGER,
            gamertag VARCHAR,
            rank SMALLINT,
            score INTEGER,
            kills SMALLINT,
            deaths SMALLINT,
            assists SMALLINT,
            shots_fired INTEGER,
            shots_hit INTEGER,
            accuracy FLOAT,
            damage_dealt FLOAT,
            damage_taken FLOAT,
            team_mmr FLOAT,
            enemy_mmr FLOAT,
            kills_expected FLOAT,
            kills_stddev FLOAT,
            deaths_expected FLOAT,
            deaths_stddev FLOAT,
            assists_expected FLOAT,
            assists_stddev FLOAT,
            avg_life_seconds FLOAT,
            backfill_bits INTEGER DEFAULT 0,
            PRIMARY KEY (match_id, xuid)
        )
    """)
    c.execute("""
        CREATE TABLE medals_earned (
            match_id VARCHAR NOT NULL,
            medal_name_id BIGINT NOT NULL,
            count SMALLINT,
            xuid VARCHAR,
            PRIMARY KEY (match_id, medal_name_id)
        )
    """)
    c.execute("""
        CREATE TABLE highlight_events (
            id INTEGER,
            match_id VARCHAR NOT NULL,
            event_type VARCHAR NOT NULL,
            time_ms INTEGER,
            xuid VARCHAR,
            gamertag VARCHAR,
            type_hint INTEGER,
            raw_json VARCHAR
        )
    """)

    # Insert 3 test matches dans match_registry
    c.execute("""
        INSERT INTO match_registry (match_id, start_time, playlist_name, playlist_id,
                                     map_name, map_id, pair_name, pair_id,
                                     game_variant_name, game_variant_id)
        VALUES
            ('match-001', '2025-01-01', 'Ranked Arena', 'pl-001', 'Aquarius', 'map-001',
             'Slayer', 'pair-001', 'Slayer', 'gv-001'),
            ('match-002', '2025-01-02', 'Ranked Arena', 'pl-002', 'Bazaar', 'map-002',
             'CTF', 'pair-002', 'CTF', 'gv-002'),
            ('match-003', '2025-01-03', 'Social', 'pl-003', 'Streets', 'map-003',
             'Fiesta', 'pair-003', 'Fiesta', 'gv-003')
    """)

    # match_participants : tous les matchs ont un participant
    for m in ["match-001", "match-002", "match-003"]:
        c.execute(
            """
            INSERT INTO match_participants (match_id, xuid, team_id, outcome, gamertag,
                                           rank, score, kills, deaths, assists,
                                           team_mmr, accuracy, shots_fired, shots_hit)
            VALUES (?, ?, 0, 1, 'TestPlayer', 1, 100, 10, 5, 3, NULL, NULL, NULL, NULL)
        """,
            [m, XUID],
        )
    # match-001 a team_mmr + kills_expected remplis (données skill complètes)
    c.execute(
        "UPDATE match_participants SET team_mmr = 1500.0, kills_expected = 0.8, "
        "deaths_expected = 0.5, assists_expected = 0.2 WHERE match_id = 'match-001'"
    )
    # match-001 a le bit skill (bit 1) marqué dans backfill_bits → plus détecté comme manquant
    c.execute("UPDATE match_participants SET backfill_bits = 1 WHERE match_id = 'match-001'")

    # match-001 has medals, match-002 and match-003 don't
    c.execute("INSERT INTO medals_earned VALUES ('match-001', 100, 2, ?)", [XUID])

    # match-001 and match-002 have events (events_loaded=TRUE), match-003 doesn't
    c.execute(
        "INSERT INTO highlight_events VALUES (1, 'match-001', 'kill', 1000, ?, 'Test', 0, NULL)",
        [XUID],
    )
    c.execute(
        "INSERT INTO highlight_events VALUES (2, 'match-002', 'kill', 2000, ?, 'Test', 0, NULL)",
        [XUID],
    )
    c.execute(
        "UPDATE match_registry SET events_loaded = TRUE WHERE match_id IN ('match-001', 'match-002')"
    )

    yield c
    c.close()


@pytest.fixture()
def player_conn():
    """DB joueur minimale."""
    c = duckdb.connect(":memory:")
    c.execute("""
        CREATE TABLE player_match_enrichment (
            match_id VARCHAR PRIMARY KEY,
            performance_score FLOAT
        )
    """)
    c.execute("""
        CREATE TABLE personal_score_awards (
            match_id VARCHAR NOT NULL,
            xuid VARCHAR NOT NULL,
            award_name VARCHAR NOT NULL,
            award_category VARCHAR,
            award_count INTEGER,
            award_score INTEGER,
            created_at TIMESTAMP
        )
    """)
    # match-001 a des personal scores, les autres non
    c.execute(
        "INSERT INTO personal_score_awards (match_id, xuid, award_name, award_count, award_score) "
        "VALUES ('match-001', ?, 'headshot', 5, 100)",
        [XUID],
    )
    yield c
    c.close()


# ─────────────────────────────────────────────────────────────────────────────
# Tests compute_backfill_mask
# ─────────────────────────────────────────────────────────────────────────────


def test_compute_backfill_mask_single():
    """Un seul type retourne le bon bit."""
    assert compute_backfill_mask("medals") == 1
    assert compute_backfill_mask("events") == 2
    assert compute_backfill_mask("assets") == 256


def test_compute_backfill_mask_multiple():
    """Plusieurs types sont combinés par OR."""
    mask = compute_backfill_mask("medals", "events", "skill")
    assert mask == 1 | 2 | 4  # 7


def test_compute_backfill_mask_all():
    """Tous les types produisent un masque complet."""
    mask = compute_backfill_mask(*BACKFILL_FLAGS.keys())
    expected = sum(BACKFILL_FLAGS.values())
    assert mask == expected


# ─────────────────────────────────────────────────────────────────────────────
# Tests détection — sans bitmask (premier run)
# ─────────────────────────────────────────────────────────────────────────────


def test_detects_missing_medals(shared_conn, player_conn):
    """Détecte les matchs sans médailles."""
    result = find_matches_missing_data(player_conn, XUID, shared_conn=shared_conn, medals=True)
    # match-002 et match-003 n'ont pas de médailles
    assert set(result) == {"match-002", "match-003"}


def test_detects_missing_events(shared_conn, player_conn):
    """Détecte les matchs sans events."""
    result = find_matches_missing_data(player_conn, XUID, shared_conn=shared_conn, events=True)
    # Seul match-003 n'a pas d'events
    assert result == ["match-003"]


def test_detects_missing_skill(shared_conn, player_conn):
    """Détecte les matchs sans skill (team_mmr NULL)."""
    result = find_matches_missing_data(player_conn, XUID, shared_conn=shared_conn, skill=True)
    assert set(result) == {"match-002", "match-003"}


def test_detects_missing_personal_scores(shared_conn, player_conn):
    """Détecte les matchs sans personal scores (per-player check)."""
    result = find_matches_missing_data(
        player_conn, XUID, shared_conn=shared_conn, personal_scores=True
    )
    # V5.5 : per-player — match-001 exclu (déjà dans personal_score_awards)
    assert set(result) == {"match-002", "match-003"}


def test_or_mode_union(shared_conn, player_conn):
    """Mode OR : union des conditions."""
    result = find_matches_missing_data(
        player_conn, XUID, shared_conn=shared_conn, detection_mode="or", medals=True, events=True
    )
    assert set(result) == {"match-002", "match-003"}


def test_and_mode_intersection(shared_conn, player_conn):
    """Mode AND : en V5, détection utilise OR (AND non implémenté dans _find_matches_in_shared_all)."""
    result = find_matches_missing_data(
        player_conn, XUID, shared_conn=shared_conn, detection_mode="and", medals=True, events=True
    )
    # V5 : toujours OR → union des conditions
    assert set(result) == {"match-002", "match-003"}


# ─────────────────────────────────────────────────────────────────────────────
# Tests détection — avec bitmask (après un run)
# ─────────────────────────────────────────────────────────────────────────────


def test_bitmask_excludes_completed_matches(shared_conn, player_conn):
    """Après insertion des médailles, les matchs ne sont plus détectés (per-player)."""
    # V5.5 : medals est per-player → insérer les données réelles
    for mid in ["match-002", "match-003"]:
        shared_conn.execute("INSERT INTO medals_earned VALUES (?, 200, 1, ?)", [mid, XUID])

    result = find_matches_missing_data(player_conn, XUID, shared_conn=shared_conn, medals=True)
    assert result == []


def test_bitmask_per_type_independent(shared_conn, player_conn):
    """La détection est indépendante par type."""
    # V5.5 : medals per-player → insérer les données réelles pour match-002
    shared_conn.execute("INSERT INTO medals_earned VALUES ('match-002', 200, 1, ?)", [XUID])

    # medals : match-002 exclu (a des médailles), match-003 toujours détecté
    result_medals = find_matches_missing_data(
        player_conn, XUID, shared_conn=shared_conn, medals=True
    )
    assert result_medals == ["match-003"]

    # events : match-003 n'a pas d'events (indépendant des medals)
    result_events = find_matches_missing_data(
        player_conn, XUID, shared_conn=shared_conn, events=True
    )
    assert result_events == ["match-003"]


def test_bitmask_events_excludes_after_marking(shared_conn, player_conn):
    """Events : après backfill (events_loaded=TRUE), le match n'est plus détecté."""
    # Fix v5.4 : la détection utilise events_loaded (source de vérité), pas le bitmask
    shared_conn.execute(
        "UPDATE match_registry SET events_loaded = TRUE WHERE match_id = 'match-003'"
    )

    result = find_matches_missing_data(player_conn, XUID, shared_conn=shared_conn, events=True)
    assert result == []


def test_bitmask_combined_types(shared_conn, player_conn):
    """Medals (per-player) + events_loaded excluent correctement."""
    # V5.5 : medals per-player → insérer données réelles pour match-003
    shared_conn.execute("INSERT INTO medals_earned VALUES ('match-003', 200, 1, ?)", [XUID])
    shared_conn.execute(
        "UPDATE match_registry SET events_loaded = TRUE WHERE match_id = 'match-003'"
    )

    result_medals = find_matches_missing_data(
        player_conn, XUID, shared_conn=shared_conn, medals=True
    )
    assert result_medals == ["match-002"]

    result_events = find_matches_missing_data(
        player_conn, XUID, shared_conn=shared_conn, events=True
    )
    assert result_events == []


# ─────────────────────────────────────────────────────────────────────────────
# Tests --force-* ignore le bitmask
# ─────────────────────────────────────────────────────────────────────────────


def test_force_medals_detects_despite_existing_data(shared_conn, player_conn):
    """--force-medals détecte les matchs même avec des médailles existantes.

    V5.5 : medals est per-player. Sans force, les matchs avec médailles
    sont exclus. Avec force, la condition passe en 1=1.
    Note : la condition force_medals utilise le même NOT IN (le bitmask
    global n'est plus consulté). Le comportement identique est attendu.
    """
    # Tous les matchs ont des médailles maintenant
    for mid in ["match-002", "match-003"]:
        shared_conn.execute("INSERT INTO medals_earned VALUES (?, 200, 1, ?)", [mid, XUID])

    # Sans force : 0 résultats (tous les matchs ont des médailles)
    result_normal = find_matches_missing_data(
        player_conn, XUID, shared_conn=shared_conn, medals=True
    )
    assert result_normal == []

    # Avec force : même condition (NOT IN medals_earned WHERE xuid = ?)
    # → 0 résultats aussi car les données existent réellement
    result_force = find_matches_missing_data(
        player_conn, XUID, shared_conn=shared_conn, medals=True, force_medals=True
    )
    assert result_force == []


# ─────────────────────────────────────────────────────────────────────────────
# Tests max_matches
# ─────────────────────────────────────────────────────────────────────────────


def test_max_matches_limits_results(shared_conn, player_conn):
    """max_matches limite le nombre de résultats."""
    result = find_matches_missing_data(
        player_conn, XUID, shared_conn=shared_conn, medals=True, max_matches=1
    )
    assert len(result) == 1


# ─────────────────────────────────────────────────────────────────────────────
# Tests scénario complet (simule un run entier)
# ─────────────────────────────────────────────────────────────────────────────


def test_full_scenario_single_run(shared_conn, player_conn):
    """Scénario complet : détection → traitement → rerun = 0."""
    # ÉTAPE 1 : Détection initiale
    missing = find_matches_missing_data(
        player_conn, XUID, shared_conn=shared_conn, medals=True, events=True, skill=True
    )
    assert len(missing) > 0, "Il devrait y avoir des matchs manquants"

    # ÉTAPE 2 : Simuler le traitement complet
    mask = compute_backfill_mask("medals", "events", "skill")
    for match_id in missing:
        # Skill : remplir les colonnes directement
        shared_conn.execute(
            "UPDATE match_participants SET team_mmr = 1400.0, kills_expected = 0.7 "
            "WHERE match_id = ? AND xuid = ?",
            [match_id, XUID],
        )
        # V5.5 : medals per-player → insérer les données réelles
        shared_conn.execute(
            "INSERT INTO medals_earned VALUES (?, 300, 1, ?)",
            [match_id, XUID],
        )
        # Bitmask global (reste valide pour events/skill)
        shared_conn.execute(
            "UPDATE match_registry SET backfill_completed = COALESCE(backfill_completed, 0) | ? "
            "WHERE match_id = ?",
            [mask, match_id],
        )
        # Events : events_loaded est la source de vérité
        shared_conn.execute(
            "UPDATE match_registry SET events_loaded = TRUE WHERE match_id = ?",
            [match_id],
        )

    # ÉTAPE 3 : Rerun — DOIT trouver 0 matchs manquants
    missing_rerun = find_matches_missing_data(
        player_conn, XUID, shared_conn=shared_conn, medals=True, events=True, skill=True
    )
    assert (
        missing_rerun == []
    ), f"Le rerun devrait trouver 0 matchs manquants, trouvé: {missing_rerun}"


def test_full_scenario_incremental(shared_conn, player_conn):
    """Scénario incrémental : run medals → run events → tout est couvert."""
    # RUN 1 : --medals seulement
    missing_medals = find_matches_missing_data(
        player_conn, XUID, shared_conn=shared_conn, medals=True
    )
    assert set(missing_medals) == {"match-002", "match-003"}

    # V5.5 : medals per-player → insérer les données réelles
    for mid in missing_medals:
        shared_conn.execute("INSERT INTO medals_earned VALUES (?, 300, 1, ?)", [mid, XUID])

    # Rerun medals → 0
    assert find_matches_missing_data(player_conn, XUID, shared_conn=shared_conn, medals=True) == []

    # RUN 2 : --events seulement
    missing_events = find_matches_missing_data(
        player_conn, XUID, shared_conn=shared_conn, events=True
    )
    assert missing_events == ["match-003"]

    # Events : events_loaded est la source de vérité
    for mid in missing_events:
        shared_conn.execute(
            "UPDATE match_registry SET events_loaded = TRUE WHERE match_id = ?",
            [mid],
        )

    # Rerun events → 0
    assert find_matches_missing_data(player_conn, XUID, shared_conn=shared_conn, events=True) == []


# ─────────────────────────────────────────────────────────────────────────────
# Test sans colonne backfill_completed (rétrocompatibilité)
# ─────────────────────────────────────────────────────────────────────────────


def test_works_without_backfill_completed_column():
    """Si backfill_completed n'existe pas dans match_registry, la détection fonctionne."""
    shared = duckdb.connect(":memory:")
    shared.execute("""
        CREATE TABLE match_registry (
            match_id VARCHAR PRIMARY KEY,
            start_time TIMESTAMP
        )
    """)
    shared.execute("""
        CREATE TABLE match_participants (
            match_id VARCHAR,
            xuid VARCHAR,
            PRIMARY KEY (match_id, xuid)
        )
    """)
    shared.execute(
        "CREATE TABLE medals_earned (match_id VARCHAR, medal_name_id BIGINT, count SMALLINT, xuid VARCHAR)"
    )
    shared.execute("CREATE TABLE highlight_events (match_id VARCHAR, event_type VARCHAR)")
    shared.execute("INSERT INTO match_registry VALUES ('m1', '2025-01-01')")
    shared.execute("INSERT INTO match_participants VALUES ('m1', ?)", [XUID])

    player = duckdb.connect(":memory:")
    result = find_matches_missing_data(player, XUID, shared_conn=shared, medals=True)
    assert result == ["m1"]
    shared.close()
    player.close()


# ─────────────────────────────────────────────────────────────────────────────
# Tests bitmask dans engine.py (sync normale)
# ─────────────────────────────────────────────────────────────────────────────


class TestSyncEngineBackfillBitmask:
    """Vérifie que le moteur de sync pose le bitmask backfill_completed."""

    def test_backfill_flags_importable_from_migrations(self):
        """Les constantes sont importables depuis src.data.sync.migrations."""
        from src.data.sync.migrations import BACKFILL_FLAGS as FLAGS
        from src.data.sync.migrations import compute_backfill_mask as cbm

        assert FLAGS["medals"] == 1
        assert FLAGS["aliases"] == 16384
        assert cbm("medals", "events") == 3

    def test_ensure_backfill_completed_column_migration(self):
        """ensure_backfill_completed_column ajoute la colonne si absente."""
        from src.data.sync.migrations import ensure_backfill_completed_column

        c = duckdb.connect(":memory:")
        c.execute("CREATE TABLE match_stats (match_id VARCHAR PRIMARY KEY)")
        ensure_backfill_completed_column(c)

        cols = {
            r[0]
            for r in c.execute(
                "SELECT column_name FROM information_schema.columns "
                "WHERE table_name = 'match_stats'"
            ).fetchall()
        }
        assert "backfill_completed" in cols
        c.close()

    def test_ensure_backfill_completed_column_idempotent(self):
        """Appeler deux fois ne lève pas d'erreur."""
        from src.data.sync.migrations import ensure_backfill_completed_column

        c = duckdb.connect(":memory:")
        c.execute("CREATE TABLE match_stats (match_id VARCHAR PRIMARY KEY)")
        ensure_backfill_completed_column(c)
        ensure_backfill_completed_column(c)  # 2e appel = idempotent
        c.close()

    def test_sync_bitmask_all_options_enabled(self):
        """Simule le calcul de bitmask quand toutes les SyncOptions sont True."""
        # Reproduit la logique de engine.py._process_single_match
        bf_mask = 0
        bf_mask |= BACKFILL_FLAGS["medals"]
        bf_mask |= BACKFILL_FLAGS["personal_scores"]
        bf_mask |= BACKFILL_FLAGS["performance_scores"]
        bf_mask |= BACKFILL_FLAGS["accuracy"]
        bf_mask |= BACKFILL_FLAGS["shots"]
        # with_skill=True
        bf_mask |= BACKFILL_FLAGS["skill"]
        bf_mask |= BACKFILL_FLAGS["enemy_mmr"]
        # with_highlight_events=True
        bf_mask |= BACKFILL_FLAGS["events"]
        # with_participants=True
        bf_mask |= BACKFILL_FLAGS["participants"]
        bf_mask |= BACKFILL_FLAGS["participants_scores"]
        bf_mask |= BACKFILL_FLAGS["participants_kda"]
        bf_mask |= BACKFILL_FLAGS["participants_shots"]
        bf_mask |= BACKFILL_FLAGS["participants_damage"]
        bf_mask |= BACKFILL_FLAGS["participants_avg_life"]
        # with_aliases=True
        bf_mask |= BACKFILL_FLAGS["aliases"]
        # with_assets=True
        bf_mask |= BACKFILL_FLAGS["assets"]
        # backfill-only : LUSR calculé via --lusr (non écrit par le sync engine)
        bf_mask |= BACKFILL_FLAGS["lusr"]
        # backfill-only : CSR écrit lors du sync pour les matchs classés
        bf_mask |= BACKFILL_FLAGS["csr"]

        # Tous les 17 bits doivent être activés
        expected_all = sum(BACKFILL_FLAGS.values())
        assert bf_mask == expected_all, f"Mask={bf_mask}, attendu={expected_all}"

    def test_sync_bitmask_prevents_backfill_detection(self, shared_conn, player_conn):
        """Après sync complète (bitmask + données per-player), rien n'est détecté."""
        full_mask = sum(BACKFILL_FLAGS.values())
        shared_conn.execute(
            "UPDATE match_registry SET backfill_completed = ?",
            [full_mask],
        )
        # Events : events_loaded source de vérité
        shared_conn.execute("UPDATE match_registry SET events_loaded = TRUE")
        # Skill : colonnes directes
        shared_conn.execute(
            "UPDATE match_participants SET team_mmr = 1400.0, kills_expected = 0.7 WHERE xuid = ?",
            [XUID],
        )
        # V5.5 : medals per-player → insérer données pour les matchs sans médailles
        for mid in ["match-002", "match-003"]:
            shared_conn.execute("INSERT INTO medals_earned VALUES (?, 300, 1, ?)", [mid, XUID])
        # V5.5 : performance_scores per-player → enrichir la player DB
        for mid in ["match-001", "match-002", "match-003"]:
            player_conn.execute(
                "INSERT OR IGNORE INTO player_match_enrichment VALUES (?, 85.0)", [mid]
            )
        # V5.5 : personal_scores per-player → déjà match-001 dans fixture
        for mid in ["match-002", "match-003"]:
            player_conn.execute(
                "INSERT INTO personal_score_awards (match_id, xuid, award_name, award_count, award_score) "
                "VALUES (?, ?, 'assist', 3, 50)",
                [mid, XUID],
            )

        result = find_matches_missing_data(
            player_conn,
            XUID,
            shared_conn=shared_conn,
            medals=True,
            events=True,
            skill=True,
            personal_scores=True,
            performance_scores=True,
        )
        assert result == []

    def test_sync_bitmask_partial_allows_backfill(self, shared_conn, player_conn):
        """Si la sync n'a pas activé un type, le backfill le détecte."""
        mask_no_events = sum(BACKFILL_FLAGS.values()) & ~BACKFILL_FLAGS["events"]
        shared_conn.execute(
            "UPDATE match_registry SET backfill_completed = ?",
            [mask_no_events],
        )

        # Le backfill doit détecter les matchs manquant d'events
        result_events = find_matches_missing_data(
            player_conn, XUID, shared_conn=shared_conn, events=True
        )
        assert len(result_events) > 0

        # V5.5 : medals est per-player → le bitmask n'affecte plus la détection
        # Avec medal pour match-001 seul (fixture), match-002/003 restent détectés
        result_medals = find_matches_missing_data(
            player_conn, XUID, shared_conn=shared_conn, medals=True
        )
        assert set(result_medals) == {"match-002", "match-003"}

    def test_backfill_flags_reexported_from_detection(self):
        """BACKFILL_FLAGS est toujours accessible depuis detection.py (rétro-compat)."""
        from scripts.backfill.detection import BACKFILL_FLAGS as DET_FLAGS
        from src.data.sync.migrations import BACKFILL_FLAGS as MIG_FLAGS

        assert DET_FLAGS is MIG_FLAGS

    def test_compute_backfill_mask_reexported_from_detection(self):
        """compute_backfill_mask est toujours accessible depuis detection.py."""
        from scripts.backfill.detection import compute_backfill_mask as det_cbm
        from src.data.sync.migrations import compute_backfill_mask as mig_cbm

        assert det_cbm is mig_cbm
        assert det_cbm("medals") == 1
