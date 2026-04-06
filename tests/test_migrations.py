"""Tests pour src/data/sync/migrations.py — couverture S18.

Couvre :
  - Helpers utilitaires (get_table_columns, table_exists, column_exists)
  - Migrations match_stats, match_participants
  - Migration highlight_events (séquence auto-increment)
  - Migration medals_earned (INT32 → BIGINT)
  - Backfill bitmask
"""

from __future__ import annotations

import duckdb
import pytest

from src.data.sync.migrations import (
    BACKFILL_FLAGS,
    _add_column_if_missing,
    add_spartan_id_to_career_progression,
    column_exists,
    compute_backfill_mask,
    ensure_backfill_completed_column,
    ensure_bot_teammate_column,
    ensure_highlight_events_autoincrement,
    ensure_match_participants_columns,
    ensure_match_stats_columns,
    ensure_medals_earned_bigint,
    ensure_performance_score_column,
    ensure_weapon_kills_table,
    get_table_columns,
    table_exists,
)


@pytest.fixture
def conn():
    """Connexion DuckDB in-memory pour les tests."""
    c = duckdb.connect(":memory:")
    yield c
    c.close()


# ─────────────────────────────────────────────────────────────────────────
# Helpers utilitaires
# ─────────────────────────────────────────────────────────────────────────


class TestHelpers:
    """Tests des fonctions utilitaires de base."""

    def test_get_table_columns_empty_when_no_table(self, conn):
        cols = get_table_columns(conn, "nonexistent_table")
        assert cols == set()

    def test_get_table_columns_returns_column_names(self, conn):
        conn.execute("CREATE TABLE t1 (a INTEGER, b VARCHAR, c FLOAT)")
        cols = get_table_columns(conn, "t1")
        assert cols == {"a", "b", "c"}

    def test_table_exists_false_when_absent(self, conn):
        assert table_exists(conn, "nonexistent") is False

    def test_table_exists_true_when_present(self, conn):
        conn.execute("CREATE TABLE t1 (id INTEGER)")
        assert table_exists(conn, "t1") is True

    def test_column_exists_false_when_absent(self, conn):
        conn.execute("CREATE TABLE t1 (id INTEGER)")
        assert column_exists(conn, "t1", "name") is False

    def test_column_exists_true_when_present(self, conn):
        conn.execute("CREATE TABLE t1 (id INTEGER, name VARCHAR)")
        assert column_exists(conn, "t1", "name") is True

    def test_column_exists_false_when_table_absent(self, conn):
        assert column_exists(conn, "nonexistent", "col") is False

    def test_add_column_if_missing_adds_column(self, conn):
        conn.execute("CREATE TABLE t1 (id INTEGER)")
        result = _add_column_if_missing(conn, "t1", "name", "VARCHAR")
        assert result is True
        assert column_exists(conn, "t1", "name") is True

    def test_add_column_if_missing_idempotent(self, conn):
        conn.execute("CREATE TABLE t1 (id INTEGER, name VARCHAR)")
        result = _add_column_if_missing(conn, "t1", "name", "VARCHAR")
        assert result is False

    def test_add_column_if_missing_uses_existing_cols(self, conn):
        conn.execute("CREATE TABLE t1 (id INTEGER)")
        existing = {"id"}
        result = _add_column_if_missing(conn, "t1", "name", "VARCHAR", existing)
        assert result is True
        assert column_exists(conn, "t1", "name") is True

    def test_add_column_if_missing_skips_with_existing_cols(self, conn):
        conn.execute("CREATE TABLE t1 (id INTEGER, name VARCHAR)")
        existing = {"id", "name"}
        result = _add_column_if_missing(conn, "t1", "name", "VARCHAR", existing)
        assert result is False


# ─────────────────────────────────────────────────────────────────────────
# Migrations match_stats
# ─────────────────────────────────────────────────────────────────────────


class TestMatchStatsMigrations:
    """Tests des migrations match_stats."""

    def test_ensure_match_stats_columns_on_empty_db(self, conn):
        """Ne crashe pas si match_stats n'existe pas."""
        ensure_match_stats_columns(conn)  # no-op, pas de table

    def test_ensure_match_stats_columns_adds_all(self, conn):
        conn.execute("CREATE TABLE match_stats (match_id VARCHAR PRIMARY KEY)")
        ensure_match_stats_columns(conn)
        cols = get_table_columns(conn, "match_stats")
        expected = {
            "accuracy",
            "end_time",
            "session_id",
            "session_label",
            "rank",
            "damage_dealt",
            "personal_score",
            "performance_score",
        }
        assert expected.issubset(cols)

    def test_ensure_match_stats_columns_idempotent(self, conn):
        conn.execute("CREATE TABLE match_stats (match_id VARCHAR PRIMARY KEY)")
        ensure_match_stats_columns(conn)
        ensure_match_stats_columns(conn)  # 2e appel idempotent
        cols = get_table_columns(conn, "match_stats")
        assert "performance_score" in cols

    def test_ensure_performance_score_column_alone(self, conn):
        conn.execute("CREATE TABLE match_stats (match_id VARCHAR PRIMARY KEY)")
        ensure_performance_score_column(conn)
        assert column_exists(conn, "match_stats", "performance_score") is True


# ─────────────────────────────────────────────────────────────────────────
# Migrations match_participants
# ─────────────────────────────────────────────────────────────────────────


class TestMatchParticipantsMigrations:
    def test_ensure_match_participants_on_empty_db(self, conn):
        """Ne crashe pas si match_participants n'existe pas."""
        ensure_match_participants_columns(conn)

    def test_ensure_match_participants_adds_all(self, conn):
        conn.execute("CREATE TABLE match_participants (match_id VARCHAR, xuid VARCHAR)")
        ensure_match_participants_columns(conn)
        cols = get_table_columns(conn, "match_participants")
        expected = {
            "rank",
            "score",
            "kills",
            "deaths",
            "assists",
            "shots_fired",
            "shots_hit",
            "damage_dealt",
            "damage_taken",
        }
        assert expected.issubset(cols)


# ─────────────────────────────────────────────────────────────────────────
# Migration highlight_events (séquence)
# ─────────────────────────────────────────────────────────────────────────


class TestHighlightEventsMigration:
    def _create_legacy_table(self, conn):
        """Crée une table highlight_events legacy (sans nextval)."""
        conn.execute("""
            CREATE TABLE highlight_events (
                id INTEGER PRIMARY KEY,
                match_id VARCHAR NOT NULL,
                event_type VARCHAR NOT NULL,
                time_ms INTEGER,
                xuid VARCHAR,
                gamertag VARCHAR,
                type_hint INTEGER,
                raw_json VARCHAR
            )
        """)

    def test_ensure_highlight_noop_if_no_table(self, conn):
        """Ne crashe pas si la table n'existe pas."""
        ensure_highlight_events_autoincrement(conn)

    def test_ensure_highlight_creates_sequence(self, conn):
        """Migre une table legacy vers nextval."""
        self._create_legacy_table(conn)
        conn.execute("""
            INSERT INTO highlight_events VALUES
            (1, 'match1', 'kill', 1000, 'xuid1', 'player1', 0, '{}'),
            (2, 'match1', 'medal', 2000, 'xuid1', 'player1', 1, '{}')
        """)
        ensure_highlight_events_autoincrement(conn)

        # Vérifier que les données existent toujours
        count = conn.execute("SELECT COUNT(*) FROM highlight_events").fetchone()[0]
        assert count == 2

        # Vérifier que INSERT sans id fonctionne (nextval)
        conn.execute("""
            INSERT INTO highlight_events (match_id, event_type)
            VALUES ('match2', 'kill')
        """)
        new_id = conn.execute("SELECT MAX(id) FROM highlight_events").fetchone()[0]
        assert new_id == 3  # max_id était 2, séquence démarre à 3

    def test_ensure_highlight_idempotent(self, conn):
        """La migration est idempotente."""
        self._create_legacy_table(conn)
        conn.execute("""
            INSERT INTO highlight_events VALUES
            (1, 'match1', 'kill', 1000, 'xuid1', 'player1', 0, '{}')
        """)
        ensure_highlight_events_autoincrement(conn)
        ensure_highlight_events_autoincrement(conn)  # 2e appel
        count = conn.execute("SELECT COUNT(*) FROM highlight_events").fetchone()[0]
        assert count == 1

    def test_ensure_highlight_empty_table(self, conn):
        """Fonctionne sur une table vide."""
        self._create_legacy_table(conn)
        ensure_highlight_events_autoincrement(conn)
        conn.execute("""
            INSERT INTO highlight_events (match_id, event_type)
            VALUES ('match1', 'kill')
        """)
        new_id = conn.execute("SELECT MAX(id) FROM highlight_events").fetchone()[0]
        assert new_id == 1  # start with 0+1=1


# ─────────────────────────────────────────────────────────────────────────
# Migration medals_earned (BIGINT)
# ─────────────────────────────────────────────────────────────────────────


class TestMedalsEarnedMigration:
    def test_ensure_medals_noop_if_no_table(self, conn):
        result = ensure_medals_earned_bigint(conn)
        assert result is False

    def test_ensure_medals_migrates_int_to_bigint(self, conn):
        conn.execute("""
            CREATE TABLE medals_earned (
                match_id VARCHAR,
                medal_name_id INTEGER,
                count SMALLINT,
                PRIMARY KEY (match_id, medal_name_id)
            )
        """)
        conn.execute("INSERT INTO medals_earned VALUES ('m1', 12345, 2)")
        result = ensure_medals_earned_bigint(conn)
        assert result is True

        # Vérifier le type
        col_type = conn.execute(
            "SELECT data_type FROM information_schema.columns "
            "WHERE table_name = 'medals_earned' AND column_name = 'medal_name_id'"
        ).fetchone()
        assert col_type[0] == "BIGINT"

        # Données préservées
        count = conn.execute("SELECT COUNT(*) FROM medals_earned").fetchone()[0]
        assert count == 1

    def test_ensure_medals_already_bigint(self, conn):
        conn.execute("""
            CREATE TABLE medals_earned (
                match_id VARCHAR,
                medal_name_id BIGINT,
                count SMALLINT,
                PRIMARY KEY (match_id, medal_name_id)
            )
        """)
        result = ensure_medals_earned_bigint(conn)
        assert result is False


# ─────────────────────────────────────────────────────────────────────────
# Backfill bitmask
# ─────────────────────────────────────────────────────────────────────────


class TestBackfillBitmask:
    def test_compute_backfill_mask_single(self):
        assert compute_backfill_mask("medals") == 1
        assert compute_backfill_mask("events") == 2
        assert compute_backfill_mask("skill") == 4

    def test_compute_backfill_mask_combined(self):
        assert compute_backfill_mask("medals", "events") == 3
        assert compute_backfill_mask("medals", "events", "skill") == 7

    def test_compute_backfill_mask_unknown_type(self):
        assert compute_backfill_mask("unknown") == 0
        assert compute_backfill_mask("medals", "unknown") == 1

    def test_compute_backfill_mask_all_flags(self):
        mask = compute_backfill_mask(*BACKFILL_FLAGS.keys())
        expected = sum(BACKFILL_FLAGS.values())
        assert mask == expected

    def test_ensure_backfill_completed_column(self, conn):
        conn.execute("CREATE TABLE match_stats (match_id VARCHAR PRIMARY KEY)")
        ensure_backfill_completed_column(conn)
        assert column_exists(conn, "match_stats", "backfill_completed") is True

    def test_ensure_backfill_completed_idempotent(self, conn):
        conn.execute("CREATE TABLE match_stats (match_id VARCHAR PRIMARY KEY)")
        ensure_backfill_completed_column(conn)
        ensure_backfill_completed_column(conn)  # 2e appel
        assert column_exists(conn, "match_stats", "backfill_completed") is True


# ─────────────────────────────────────────────────────────────────────────
# Migration weapon_kills (v5.7)
# ─────────────────────────────────────────────────────────────────────────


class TestWeaponKillsMigration:
    """Tests ensure_weapon_kills_table et ses sous-fonctions de migration."""

    def test_ensure_weapon_kills_creates_table(self, conn):
        """Crée la table weapon_kills sur une DB vide."""
        ensure_weapon_kills_table(conn)
        assert table_exists(conn, "weapon_kills")
        cols = get_table_columns(conn, "weapon_kills")
        expected = {
            "match_id",
            "xuid",
            "time_ms",
            "weapon_id",
            "delta_ms",
            "confidence",
            "swap_detected",
            "delayed_damage",
        }
        assert expected == cols

    def test_ensure_weapon_kills_weapon_id_ubigint(self, conn):
        """weapon_id doit être de type UBIGINT."""
        ensure_weapon_kills_table(conn)
        col_type = conn.execute(
            "SELECT data_type FROM information_schema.columns "
            "WHERE table_name = 'weapon_kills' AND column_name = 'weapon_id'"
        ).fetchone()
        assert col_type[0] == "UBIGINT"

    def test_ensure_weapon_kills_idempotent(self, conn):
        """Double appel ne lève pas d'erreur et préserve le schéma."""
        ensure_weapon_kills_table(conn)
        conn.execute(
            "INSERT INTO weapon_kills (match_id, xuid, time_ms, weapon_id) "
            "VALUES ('m1', 'x1', 1000, 42)"
        )
        ensure_weapon_kills_table(conn)
        count = conn.execute("SELECT COUNT(*) FROM weapon_kills").fetchone()[0]
        assert count == 1

    def test_ensure_weapon_kills_creates_index(self, conn):
        """L'index idx_wk_match_xuid est créé."""
        ensure_weapon_kills_table(conn)
        indexes = conn.execute(
            "SELECT index_name FROM duckdb_indexes() WHERE table_name = 'weapon_kills'"
        ).fetchall()
        index_names = {r[0] for r in indexes}
        assert "idx_wk_match_xuid" in index_names

    def test_migrate_weapon_name_to_id_perkill(self, conn):
        """Legacy per-kill (weapon_name VARCHAR, time_ms) → weapon_id UBIGINT."""
        from src.analysis._weapon_data import WEAPON_NAME_TO_INT

        conn.execute("""
            CREATE TABLE weapon_kills (
                match_id VARCHAR NOT NULL,
                xuid VARCHAR NOT NULL,
                time_ms INTEGER NOT NULL,
                weapon_name VARCHAR,
                delta_ms INTEGER,
                confidence VARCHAR NOT NULL DEFAULT 'none',
                swap_detected BOOLEAN NOT NULL DEFAULT FALSE,
                delayed_damage BOOLEAN NOT NULL DEFAULT FALSE
            )
        """)
        br75_id = WEAPON_NAME_TO_INT.get("BR75")
        conn.execute(
            "INSERT INTO weapon_kills (match_id, xuid, time_ms, weapon_name) "
            "VALUES ('m1', 'x1', 500, 'BR75'), ('m1', 'x1', 800, 'MELEE')"
        )
        ensure_weapon_kills_table(conn)

        cols = get_table_columns(conn, "weapon_kills")
        assert "weapon_id" in cols
        assert "weapon_name" not in cols

        rows = conn.execute("SELECT weapon_id FROM weapon_kills ORDER BY time_ms").fetchall()
        assert rows[0][0] == br75_id
        assert rows[1][0] == 1  # MELEE_WEAPON_ID

    def test_migrate_weapon_name_aggregated_drops(self, conn):
        """Legacy agrégé (weapon_name + kills, pas de time_ms) → DROP+CREATE vide."""
        conn.execute("""
            CREATE TABLE weapon_kills (
                match_id VARCHAR NOT NULL,
                xuid VARCHAR NOT NULL,
                weapon_name VARCHAR,
                kills INTEGER
            )
        """)
        conn.execute("INSERT INTO weapon_kills VALUES ('m1', 'x1', 'BR75', 5)")
        ensure_weapon_kills_table(conn)

        assert table_exists(conn, "weapon_kills")
        cols = get_table_columns(conn, "weapon_kills")
        assert "weapon_id" in cols
        assert "weapon_name" not in cols
        # Ancienne table agrégée → DROP+CREATE → vide
        count = conn.execute("SELECT COUNT(*) FROM weapon_kills").fetchone()[0]
        assert count == 0

    def test_upgrade_bigint_to_ubigint(self, conn):
        """weapon_id BIGINT → UBIGINT, données préservées."""
        conn.execute("""
            CREATE TABLE weapon_kills (
                match_id VARCHAR NOT NULL,
                xuid VARCHAR NOT NULL,
                time_ms INTEGER NOT NULL,
                weapon_id BIGINT,
                delta_ms INTEGER,
                confidence VARCHAR NOT NULL DEFAULT 'none',
                swap_detected BOOLEAN NOT NULL DEFAULT FALSE,
                delayed_damage BOOLEAN NOT NULL DEFAULT FALSE
            )
        """)
        conn.execute(
            "INSERT INTO weapon_kills (match_id, xuid, time_ms, weapon_id) "
            "VALUES ('m1', 'x1', 500, 3107363175969783455)"
        )
        ensure_weapon_kills_table(conn)

        col_type = conn.execute(
            "SELECT data_type FROM information_schema.columns "
            "WHERE table_name = 'weapon_kills' AND column_name = 'weapon_id'"
        ).fetchone()
        assert col_type[0] == "UBIGINT"

        val = conn.execute("SELECT weapon_id FROM weapon_kills").fetchone()[0]
        assert val == 3107363175969783455


# ─────────────────────────────────────────────────────────────────────────
# Migration bot_teammate column (v5.5)
# ─────────────────────────────────────────────────────────────────────────


class TestBotTeammateColumn:
    """Tests ensure_bot_teammate_column."""

    def test_ensure_bot_teammate_noop_no_table(self, conn):
        """Pas de table player_match_enrichment → no-op."""
        ensure_bot_teammate_column(conn)  # ne doit pas lever

    def test_ensure_bot_teammate_adds_column(self, conn):
        """Ajoute had_bot_teammate BOOLEAN si absent."""
        conn.execute(
            "CREATE TABLE player_match_enrichment "
            "(match_id VARCHAR PRIMARY KEY, performance_score FLOAT)"
        )
        ensure_bot_teammate_column(conn)
        assert column_exists(conn, "player_match_enrichment", "had_bot_teammate")
        col_type = conn.execute(
            "SELECT data_type FROM information_schema.columns "
            "WHERE table_name = 'player_match_enrichment' "
            "AND column_name = 'had_bot_teammate'"
        ).fetchone()
        assert col_type[0] == "BOOLEAN"

    def test_ensure_bot_teammate_idempotent(self, conn):
        """Double appel → pas d'erreur, schéma identique."""
        conn.execute(
            "CREATE TABLE player_match_enrichment "
            "(match_id VARCHAR PRIMARY KEY, performance_score FLOAT)"
        )
        ensure_bot_teammate_column(conn)
        ensure_bot_teammate_column(conn)
        cols = get_table_columns(conn, "player_match_enrichment")
        assert "had_bot_teammate" in cols

    def test_ensure_bot_teammate_preserves_data(self, conn):
        """Les données existantes ne sont pas perdues."""
        conn.execute(
            "CREATE TABLE player_match_enrichment "
            "(match_id VARCHAR PRIMARY KEY, performance_score FLOAT)"
        )
        conn.execute("INSERT INTO player_match_enrichment VALUES ('m1', 85.0), ('m2', 72.0)")
        ensure_bot_teammate_column(conn)
        count = conn.execute("SELECT COUNT(*) FROM player_match_enrichment").fetchone()[0]
        assert count == 2
        score = conn.execute(
            "SELECT performance_score FROM player_match_enrichment WHERE match_id = 'm1'"
        ).fetchone()[0]
        assert score == 85.0


# ─────────────────────────────────────────────────────────────────────────
# Migration spartan_id career_progression (v5.x)
# ─────────────────────────────────────────────────────────────────────────


class TestSpartanIdCareerProgression:
    """Tests add_spartan_id_to_career_progression."""

    def test_add_spartan_id_noop_no_table(self, conn):
        """Pas de table career_progression → no-op."""
        add_spartan_id_to_career_progression(conn)

    def test_add_spartan_id_adds_column(self, conn):
        """Ajoute spartan_id VARCHAR si absent."""
        conn.execute("CREATE TABLE career_progression (id INTEGER PRIMARY KEY, rank_name VARCHAR)")
        add_spartan_id_to_career_progression(conn)
        assert column_exists(conn, "career_progression", "spartan_id")
        col_type = conn.execute(
            "SELECT data_type FROM information_schema.columns "
            "WHERE table_name = 'career_progression' "
            "AND column_name = 'spartan_id'"
        ).fetchone()
        assert col_type[0] == "VARCHAR"

    def test_add_spartan_id_idempotent(self, conn):
        """Double appel → schéma identique."""
        conn.execute("CREATE TABLE career_progression (id INTEGER PRIMARY KEY, rank_name VARCHAR)")
        add_spartan_id_to_career_progression(conn)
        add_spartan_id_to_career_progression(conn)
        cols = get_table_columns(conn, "career_progression")
        assert "spartan_id" in cols


# ─────────────────────────────────────────────────────────────────────────
# Highlight events — chemin idempotent séquence existante
# ─────────────────────────────────────────────────────────────────────────


class TestHighlightEventsSequenceIdempotent:
    """Test du chemin où la séquence nextval existe déjà."""

    def test_recreate_already_has_sequence(self, conn):
        """Si nextval existe déjà, les données sont préservées après migration."""
        conn.execute("CREATE SEQUENCE highlight_events_id_seq START WITH 1")
        conn.execute("""
            CREATE TABLE highlight_events (
                id INTEGER PRIMARY KEY DEFAULT nextval('highlight_events_id_seq'),
                match_id VARCHAR NOT NULL,
                event_type VARCHAR NOT NULL,
                time_ms INTEGER,
                xuid VARCHAR,
                gamertag VARCHAR,
                type_hint INTEGER,
                raw_json VARCHAR
            )
        """)
        conn.execute("""
            INSERT INTO highlight_events (match_id, event_type, time_ms, xuid)
            VALUES ('match1', 'kill', 1000, 'xuid1'),
                   ('match1', 'medal', 2000, 'xuid2')
        """)
        original_count = conn.execute("SELECT COUNT(*) FROM highlight_events").fetchone()[0]
        assert original_count == 2

        ensure_highlight_events_autoincrement(conn)

        # Données préservées
        count = conn.execute("SELECT COUNT(*) FROM highlight_events").fetchone()[0]
        assert count == 2

        # Nextval fonctionne toujours
        conn.execute(
            "INSERT INTO highlight_events (match_id, event_type) VALUES ('match2', 'assist')"
        )
        new_id = conn.execute("SELECT MAX(id) FROM highlight_events").fetchone()[0]
        assert new_id == 3


class TestDropLegacyTranslationTables:
    """Tests pour la migration drop_legacy_translation_tables (target_db='metadata')."""

    def test_drops_mode_translations(self) -> None:
        """mode_translations est supprimée si elle existe."""
        with duckdb.connect(":memory:") as conn:
            conn.execute("CREATE TABLE mode_translations (name_en VARCHAR, name_fr VARCHAR)")
            conn.execute("INSERT INTO mode_translations VALUES ('Slayer', 'Assassin')")
            assert conn.execute("SELECT COUNT(*) FROM mode_translations").fetchone()[0] == 1

            from src.data.migration.steps.drop_legacy_translation_tables import (
                _drop_legacy_translation_tables,
            )

            _drop_legacy_translation_tables(conn)
            has = conn.execute(
                "SELECT 1 FROM information_schema.tables WHERE table_name='mode_translations' LIMIT 1"
            ).fetchone()
            assert has is None

    def test_drops_playlist_translations(self) -> None:
        """playlist_translations est supprimée si elle existe."""
        with duckdb.connect(":memory:") as conn:
            conn.execute("CREATE TABLE playlist_translations (uuid VARCHAR, name_fr VARCHAR)")
            conn.execute("INSERT INTO playlist_translations VALUES ('abc', 'Test')")

            from src.data.migration.steps.drop_legacy_translation_tables import (
                _drop_legacy_translation_tables,
            )

            _drop_legacy_translation_tables(conn)
            has = conn.execute(
                "SELECT 1 FROM information_schema.tables WHERE table_name='playlist_translations' LIMIT 1"
            ).fetchone()
            assert has is None

    def test_idempotent_when_tables_absent(self) -> None:
        """La migration ne lève pas d'erreur si les tables sont déjà absentes."""
        with duckdb.connect(":memory:") as conn:
            from src.data.migration.steps.drop_legacy_translation_tables import (
                _drop_legacy_translation_tables,
            )

            # Pas d'exception — DROP TABLE IF EXISTS est idempotent
            _drop_legacy_translation_tables(conn)
            _drop_legacy_translation_tables(conn)

    def test_migration_targets_metadata_db(self) -> None:
        """La migration est enregistrée avec target_db='metadata'."""
        from src.data.migration.registry import MIGRATIONS
        from src.data.migration.runner import _load_migration_steps

        _load_migration_steps()
        mig = next((m for m in MIGRATIONS if m.name == "drop_legacy_translation_tables"), None)
        assert mig is not None, (
            "Migration drop_legacy_translation_tables non trouvée dans MIGRATIONS"
        )
        assert mig.target_db == "metadata"
        assert mig.apply_backfill is None  # pas de backfill

    def test_apply_pending_runs_on_metadata_path(self, tmp_path: pytest.TempPathFactory) -> None:
        """apply_pending_migrations exécute la migration sur metadata_db_path."""
        import duckdb

        from src.data.migration.runner import apply_pending_migrations

        meta_db = tmp_path / "meta_test.duckdb"
        with duckdb.connect(str(meta_db)) as conn:
            conn.execute("CREATE TABLE mode_translations (x INTEGER)")
            conn.execute("CREATE TABLE playlist_translations (x INTEGER)")

        report = apply_pending_migrations(metadata_db_path=meta_db)
        assert report.schemas_applied >= 1
        assert not report.errors

        with duckdb.connect(str(meta_db), read_only=True) as conn:
            for table in ("mode_translations", "playlist_translations"):
                has = conn.execute(
                    f"SELECT 1 FROM information_schema.tables WHERE table_name='{table}' LIMIT 1"  # noqa: S608
                ).fetchone()
                assert has is None, f"{table} aurait dû être supprimée"
