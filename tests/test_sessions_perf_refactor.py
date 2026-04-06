"""Tests pour le refactor sessions-perf (P1a, P1b, P2, P4).

Couvre :
- ensure_pme_session_index         (P1b)
- ensure_mv_session_stats_varchar  (P1a migration)
- _add_friends_columns             (P2 helper Polars)
- _upsert_session_rows             (P2 bulk upsert)
- _partial_refresh_session_stats   (P4 incrémental)
- _refresh_session_stats           (P4 full/delta dispatch)
"""

from __future__ import annotations

import contextlib
from datetime import datetime, timezone

import duckdb
import polars as pl
import pytest

from src.data.sessions_backfill import _add_friends_columns, _upsert_session_rows
from src.data.sync.migrations import ensure_mv_session_stats_varchar, ensure_pme_session_index

# ─────────────────────────────────────────────────────────────────────────────
# Fixtures
# ─────────────────────────────────────────────────────────────────────────────


@pytest.fixture
def pme_conn():
    """Connexion DuckDB en mémoire avec player_match_enrichment."""
    conn = duckdb.connect(":memory:")
    conn.execute("""
        CREATE TABLE player_match_enrichment (
            match_id               VARCHAR PRIMARY KEY,
            performance_score      FLOAT,
            session_id             VARCHAR,
            session_label          VARCHAR,
            is_with_friends        BOOLEAN,
            teammates_signature    VARCHAR,
            known_teammates_count  SMALLINT,
            friends_xuids          VARCHAR,
            created_at             TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            updated_at             TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    """)
    return conn


@pytest.fixture
def mv_conn():
    """Connexion DuckDB en mémoire avec mv_session_stats (type INTEGER — état avant migration)."""
    conn = duckdb.connect(":memory:")
    conn.execute("""
        CREATE TABLE mv_session_stats (
            session_id   INTEGER PRIMARY KEY,
            match_count  INTEGER,
            start_time   TIMESTAMP,
            end_time     TIMESTAMP,
            total_kills  INTEGER,
            total_deaths INTEGER,
            total_assists INTEGER,
            kd_ratio     DOUBLE,
            win_rate     DOUBLE,
            avg_accuracy DOUBLE,
            avg_life_seconds DOUBLE,
            updated_at   TIMESTAMP
        )
    """)
    return conn


def _make_df_sessions(rows: list[dict]) -> pl.DataFrame:
    """Construit un DataFrame sessions minimal pour _upsert_session_rows."""
    ts = datetime(2026, 4, 1, 20, 0, 0, tzinfo=timezone.utc).timestamp()
    base = {
        "match_id": "m1",
        "session_id": "0",
        "session_label": "01/04/2026 20:00–21:00 (1)",
        "teammates_signature": None,
        "_ts": ts,
    }
    records = []
    for r in rows:
        rec = {**base, **r}
        records.append(rec)

    return pl.DataFrame(
        {
            "match_id": [r["match_id"] for r in records],
            "session_id": [r["session_id"] for r in records],
            "session_label": [r["session_label"] for r in records],
            "teammates_signature": [r.get("teammates_signature") for r in records],
            "_ts": [r["_ts"] for r in records],
        }
    )


# ─────────────────────────────────────────────────────────────────────────────
# P1b — ensure_pme_session_index
# ─────────────────────────────────────────────────────────────────────────────


class TestEnsurePmeSessionIndex:
    def test_creates_index_when_table_exists(self, pme_conn):
        """Doit créer idx_pme_session si la table existe."""
        ensure_pme_session_index(pme_conn)
        indexes = pme_conn.execute(
            "SELECT index_name FROM duckdb_indexes() WHERE table_name = 'player_match_enrichment'"
        ).fetchall()
        index_names = [r[0] for r in indexes]
        assert "idx_pme_session" in index_names

    def test_idempotent_double_call(self, pme_conn):
        """Double appel — pas d'erreur, index toujours présent."""
        ensure_pme_session_index(pme_conn)
        ensure_pme_session_index(pme_conn)  # ne doit pas lever
        indexes = pme_conn.execute(
            "SELECT index_name FROM duckdb_indexes() WHERE table_name = 'player_match_enrichment'"
        ).fetchall()
        assert any(r[0] == "idx_pme_session" for r in indexes)

    def test_noop_when_table_absent(self):
        """Ne doit pas lever si player_match_enrichment est absente."""
        conn = duckdb.connect(":memory:")
        ensure_pme_session_index(conn)  # ne doit pas lever


# ─────────────────────────────────────────────────────────────────────────────
# P1a — ensure_mv_session_stats_varchar
# ─────────────────────────────────────────────────────────────────────────────


class TestEnsureMvSessionStatsVarchar:
    def test_migrates_integer_to_varchar(self, mv_conn):
        """Doit recréer la table avec session_id VARCHAR si c'était INTEGER."""
        ensure_mv_session_stats_varchar(mv_conn)
        row = mv_conn.execute(
            "SELECT data_type FROM information_schema.columns "
            "WHERE table_name = 'mv_session_stats' AND column_name = 'session_id'"
        ).fetchone()
        assert row is not None
        assert row[0].upper() in ("VARCHAR", "TEXT")

    def test_noop_when_already_varchar(self):
        """Doit être idempotente si session_id est déjà VARCHAR."""
        conn = duckdb.connect(":memory:")
        conn.execute("""
            CREATE TABLE mv_session_stats (
                session_id VARCHAR PRIMARY KEY,
                match_count INTEGER,
                start_time TIMESTAMP,
                end_time TIMESTAMP,
                total_kills INTEGER,
                total_deaths INTEGER,
                total_assists INTEGER,
                kd_ratio DOUBLE,
                win_rate DOUBLE,
                avg_accuracy DOUBLE,
                avg_life_seconds DOUBLE,
                updated_at TIMESTAMP
            )
        """)
        conn.execute(
            "INSERT INTO mv_session_stats VALUES ('0', 3, now(), now(), 10, 5, 2, 2.0, 0.67, 80.0, 30.0, now())"
        )
        ensure_mv_session_stats_varchar(conn)
        # Données préservées
        count = conn.execute("SELECT COUNT(*) FROM mv_session_stats").fetchone()[0]
        assert count == 1

    def test_noop_when_table_absent(self):
        """Ne doit pas lever si mv_session_stats est absente."""
        conn = duckdb.connect(":memory:")
        ensure_mv_session_stats_varchar(conn)  # ne doit pas lever


# ─────────────────────────────────────────────────────────────────────────────
# P2 — _add_friends_columns
# ─────────────────────────────────────────────────────────────────────────────


class TestAddFriendsColumns:
    def test_with_empty_friends_set_returns_nulls(self):
        """Sans amis configurés, les 3 colonnes doivent être NULL."""
        df = pl.DataFrame({"teammates_signature": ["xuid1,xuid2", None]})
        result = _add_friends_columns(df, frozenset())
        assert result["_iwf"].is_null().all()
        assert result["_ktc"].is_null().all()
        assert result["_fx"].is_null().all()

    def test_detects_friends_in_signature(self):
        """Doit détecter les amis présents dans la signature."""
        df = pl.DataFrame({"teammates_signature": ["xuid1,xuid2,xuid3"]})
        result = _add_friends_columns(df, frozenset({"xuid1", "xuid2"}))
        assert result["_iwf"][0] is True
        assert result["_ktc"][0] == 2
        assert result["_fx"][0] == "xuid1,xuid2"

    def test_no_friends_in_signature(self):
        """Aucun ami dans la signature → _iwf=False, _ktc=0, _fx=None."""
        df = pl.DataFrame({"teammates_signature": ["xuid9,xuid8"]})
        result = _add_friends_columns(df, frozenset({"xuid1", "xuid2"}))
        assert result["_iwf"][0] is False
        assert result["_ktc"][0] == 0
        assert result["_fx"][0] is None

    def test_null_signature_with_friends_set(self):
        """Signature NULL avec amis configurés → _iwf=None (Polars skip_nulls=True)."""
        df = pl.DataFrame({"teammates_signature": [None]})
        result = _add_friends_columns(df, frozenset({"xuid1"}))
        # map_elements ignore les null par défaut : valeur résultante = None
        assert result["_iwf"][0] is None
        assert result["_ktc"][0] is None

    def test_fx_sorted(self):
        """_fx doit être trié alphabétiquement."""
        df = pl.DataFrame({"teammates_signature": ["xuid3,xuid1,xuid2"]})
        result = _add_friends_columns(df, frozenset({"xuid3", "xuid1"}))
        assert result["_fx"][0] == "xuid1,xuid3"


# ─────────────────────────────────────────────────────────────────────────────
# P2 — _upsert_session_rows
# ─────────────────────────────────────────────────────────────────────────────


class TestUpsertSessionRows:
    def test_inserts_new_rows(self, pme_conn):
        """Doit insérer les lignes dans player_match_enrichment."""
        df = _make_df_sessions([{"match_id": "m1"}, {"match_id": "m2"}])
        updated, skipped, errors = _upsert_session_rows(
            pme_conn, df, frozenset(), include_recent=True, threshold=0.0
        )
        assert errors == []
        assert updated == 2
        assert skipped == 0
        count = pme_conn.execute("SELECT COUNT(*) FROM player_match_enrichment").fetchone()[0]
        assert count == 2

    def test_updates_existing_rows_on_conflict(self, pme_conn):
        """ON CONFLICT doit mettre à jour session_id si le match_id existe déjà."""
        pme_conn.execute(
            "INSERT INTO player_match_enrichment (match_id, session_id) VALUES ('m1', '0')"
        )
        df = _make_df_sessions([{"match_id": "m1", "session_id": "1"}])
        _upsert_session_rows(pme_conn, df, frozenset(), include_recent=True, threshold=0.0)
        row = pme_conn.execute(
            "SELECT session_id FROM player_match_enrichment WHERE match_id = 'm1'"
        ).fetchone()
        assert row[0] == "1"

    def test_skips_recent_when_include_recent_false(self, pme_conn):
        """Les matchs au-delà du threshold doivent être ignorés."""
        import time

        future_ts = time.time() + 3600
        df = _make_df_sessions([{"match_id": "m1", "_ts": future_ts}])
        updated, skipped, errors = _upsert_session_rows(
            pme_conn, df, frozenset(), include_recent=False, threshold=time.time()
        )
        assert updated == 0
        assert skipped == 1
        assert errors == []

    def test_missing_ts_column_returns_error(self, pme_conn):
        """Sans colonne _ts, doit retourner une erreur descriptive."""
        df = pl.DataFrame(
            {
                "match_id": ["m1"],
                "session_id": ["0"],
                "session_label": ["x"],
                "teammates_signature": [None],
            }
        )
        updated, skipped, errors = _upsert_session_rows(
            pme_conn, df, frozenset(), include_recent=True, threshold=0.0
        )
        assert updated == 0
        assert len(errors) == 1
        assert "_ts" in errors[0]

    def test_empty_df_after_filter_returns_zero(self, pme_conn):
        """DataFrame vide après filtrage → (0, N_skipped, [])."""
        import time

        ts_past = time.time() - 9999
        df = _make_df_sessions([{"match_id": "m1", "_ts": None}])
        # Forcer _ts = None explicitement
        df = df.with_columns(pl.lit(None).cast(pl.Float64).alias("_ts"))
        updated, skipped, errors = _upsert_session_rows(
            pme_conn, df, frozenset(), include_recent=True, threshold=ts_past
        )
        assert updated == 0
        assert errors == []

    def test_populates_friends_columns(self, pme_conn):
        """Avec un friends_set, is_with_friends/known_teammates_count/friends_xuids doivent être peuplés."""
        df = _make_df_sessions([{"match_id": "m1", "teammates_signature": "xuid1,xuid2"}])
        _upsert_session_rows(pme_conn, df, frozenset({"xuid1"}), include_recent=True, threshold=0.0)
        row = pme_conn.execute(
            "SELECT is_with_friends, known_teammates_count, friends_xuids "
            "FROM player_match_enrichment WHERE match_id = 'm1'"
        ).fetchone()
        assert row[0] is True
        assert row[1] == 1
        assert row[2] == "xuid1"


# ─────────────────────────────────────────────────────────────────────────────
# P4 — _partial_refresh_session_stats & _refresh_session_stats
# ─────────────────────────────────────────────────────────────────────────────


def _make_mv_conn_with_data() -> duckdb.DuckDBPyConnection:
    """Crée une connexion avec mv_session_stats peuplée (5 sessions)."""
    conn = duckdb.connect(":memory:")
    conn.execute("""
        CREATE TABLE mv_session_stats (
            session_id VARCHAR PRIMARY KEY,
            match_count INTEGER,
            start_time TIMESTAMP,
            end_time TIMESTAMP,
            total_kills INTEGER,
            total_deaths INTEGER,
            total_assists INTEGER,
            kd_ratio DOUBLE,
            win_rate DOUBLE,
            avg_accuracy DOUBLE,
            avg_life_seconds DOUBLE,
            updated_at TIMESTAMP
        )
    """)
    for i in range(5):
        conn.execute(
            "INSERT INTO mv_session_stats VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, now())",
            [
                str(i),
                3,
                f"2026-04-0{i + 1} 20:00:00",
                f"2026-04-0{i + 1} 22:00:00",
                10,
                5,
                2,
                2.0,
                0.67,
                80.0,
                30.0,
            ],
        )
    return conn


class TestPartialRefreshSessionStats:
    def test_removes_3_most_recent(self):
        """Doit supprimer exactement les 3 sessions récentes (par start_time DESC)."""
        from src.data.repositories._materialized_views import MaterializedViewsMixin

        conn = _make_mv_conn_with_data()

        # On veut tester _partial_refresh_session_stats isolément.
        # On crée un mock minimal du mixin avec les attributs nécessaires.
        class FakeMixin(MaterializedViewsMixin):
            _xuid = "test_xuid"

            def _get_connection(self):
                return conn

        mixin = object.__new__(FakeMixin)
        mixin._xuid = "test_xuid"

        # Avant : 5 sessions
        assert conn.execute("SELECT COUNT(*) FROM mv_session_stats").fetchone()[0] == 5

        # _partial_refresh_session_stats tente de réinsérer depuis shared.match_participants
        # absent ici → l'INSERT échouera silencieusement, mais le DELETE aura eu lieu.
        with contextlib.suppress(Exception):
            mixin._partial_refresh_session_stats(conn)

        # Les 3 sessions les plus récentes (ids 4, 3, 2) ont été supprimées
        remaining = conn.execute(
            "SELECT session_id FROM mv_session_stats ORDER BY start_time"
        ).fetchall()
        remaining_ids = {r[0] for r in remaining}
        # sessions 0 et 1 doivent rester
        assert "0" in remaining_ids
        assert "1" in remaining_ids
        assert len(remaining_ids) == 2

    def test_noop_when_table_empty(self):
        """Ne doit pas lever si mv_session_stats est vide."""
        from src.data.repositories._materialized_views import MaterializedViewsMixin

        conn = duckdb.connect(":memory:")
        conn.execute("""
            CREATE TABLE mv_session_stats (
                session_id VARCHAR PRIMARY KEY, match_count INTEGER,
                start_time TIMESTAMP, end_time TIMESTAMP,
                total_kills INTEGER, total_deaths INTEGER, total_assists INTEGER,
                kd_ratio DOUBLE, win_rate DOUBLE, avg_accuracy DOUBLE,
                avg_life_seconds DOUBLE, updated_at TIMESTAMP
            )
        """)

        class FakeMixin(MaterializedViewsMixin):
            _xuid = "x"

            def _get_connection(self):
                return conn

        mixin = object.__new__(FakeMixin)
        mixin._xuid = "x"
        mixin._partial_refresh_session_stats(conn)  # ne doit pas lever


class TestRefreshSessionStatsDispatch:
    def test_full_rebuild_when_new_ids_is_none(self):
        """new_ids=None → chemin full rebuild (DELETE + INSERT)."""
        from src.data.repositories._materialized_views import MaterializedViewsMixin

        conn = _make_mv_conn_with_data()
        # Ajouter player_match_enrichment minimal (sans données)
        conn.execute("""
            CREATE TABLE player_match_enrichment (
                match_id VARCHAR PRIMARY KEY, session_id VARCHAR, session_label VARCHAR
            )
        """)

        class FakeMixin(MaterializedViewsMixin):
            _xuid = "x"

            def _get_connection(self):
                return conn

        mixin = object.__new__(FakeMixin)
        mixin._xuid = "x"

        # full rebuild : DELETE + INSERT depuis shared (absent → 0 lignes, mais table vidée)
        result = mixin._refresh_session_stats(conn, new_ids=None)
        assert result == 0  # shared absent → 0 lignes insérées
        # Mais la table a été vidée
        assert conn.execute("SELECT COUNT(*) FROM mv_session_stats").fetchone()[0] == 0

    def test_incremental_when_new_ids_provided(self):
        """new_ids fourni → chemin incrémental (3 dernières sessions)."""
        from src.data.repositories._materialized_views import MaterializedViewsMixin

        conn = _make_mv_conn_with_data()
        conn.execute("""
            CREATE TABLE player_match_enrichment (
                match_id VARCHAR PRIMARY KEY, session_id VARCHAR, session_label VARCHAR
            )
        """)

        class FakeMixin(MaterializedViewsMixin):
            _xuid = "x"

            def _get_connection(self):
                return conn

        mixin = object.__new__(FakeMixin)
        mixin._xuid = "x"

        before = conn.execute("SELECT COUNT(*) FROM mv_session_stats").fetchone()[0]
        assert before == 5

        mixin._refresh_session_stats(conn, new_ids=["m1"])
        after = conn.execute("SELECT COUNT(*) FROM mv_session_stats").fetchone()[0]
        # 3 sessions supprimées (pas réinsérées car shared absent) → 2 restantes
        assert after == 2

    def test_returns_zero_when_no_pme_table(self):
        """Doit retourner 0 si player_match_enrichment est absente."""
        from src.data.repositories._materialized_views import MaterializedViewsMixin

        conn = duckdb.connect(":memory:")
        conn.execute("""
            CREATE TABLE mv_session_stats (
                session_id VARCHAR PRIMARY KEY, match_count INTEGER,
                start_time TIMESTAMP, end_time TIMESTAMP,
                total_kills INTEGER, total_deaths INTEGER, total_assists INTEGER,
                kd_ratio DOUBLE, win_rate DOUBLE, avg_accuracy DOUBLE,
                avg_life_seconds DOUBLE, updated_at TIMESTAMP
            )
        """)

        class FakeMixin(MaterializedViewsMixin):
            _xuid = "x"

            def _get_connection(self):
                return conn

        mixin = object.__new__(FakeMixin)
        mixin._xuid = "x"

        # player_match_enrichment absente → has_sessions=False → return 0
        result = mixin._refresh_session_stats(conn, new_ids=None)
        assert result == 0
