"""Tests Phase F — Fraîcheur indicateur "Mis à jour il y a XXX" (multi-joueurs)."""

from __future__ import annotations

import duckdb

SCHEMA_SYNC_META = """
CREATE TABLE IF NOT EXISTS sync_meta (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at TIMESTAMP DEFAULT current_timestamp
)
"""

SCHEMA_MATCH_PARTICIPANTS = """
CREATE TABLE IF NOT EXISTS match_participants (
    match_id TEXT,
    xuid TEXT
)
"""


def _make_player_db(tmp_path, gamertag: str, last_sync_at: str | None = None):
    """Crée une player DB minimale avec sync_meta optionnelle."""
    db_path = tmp_path / f"{gamertag}.duckdb"
    conn = duckdb.connect(str(db_path))
    conn.execute(SCHEMA_SYNC_META)
    if last_sync_at:
        conn.execute(
            "INSERT OR REPLACE INTO sync_meta (key, value) VALUES ('last_sync_at', ?)",
            [last_sync_at],
        )
    conn.close()
    return db_path


def test_get_sync_metadata_reads_from_player_db(tmp_path):
    """get_sync_metadata() lit depuis sync_meta WHERE key='last_sync_at' (player DB)."""
    from src.data.repositories._diagnostic_repo import DiagnosticMixin

    expected_ts = "2026-01-01T12:00:00Z"
    db_path = _make_player_db(tmp_path, "TestPlayer", last_sync_at=expected_ts)

    conn = duckdb.connect(str(db_path))

    class _FakeRepo(DiagnosticMixin):
        _xuid = "xuid-test"
        _gamertag = "TestPlayer"
        _attached_dbs: set = set()
        _player_db_path = db_path
        _shared_db_path = db_path
        _metadata_db_path = db_path

        def _get_connection(self):
            return conn

        def get_match_count(self):
            return 0

        @property
        def has_shared(self):
            return False

    repo = _FakeRepo()
    meta = repo.get_sync_metadata()
    conn.close()

    assert meta["last_sync_at"] == expected_ts, (
        f"Attendu {expected_ts!r}, obtenu {meta['last_sync_at']!r}"
    )


def test_get_sync_metadata_no_entry_returns_none(tmp_path):
    """Player DB sans entrée last_sync_at → last_sync_at=None, pas d'exception."""
    from src.data.repositories._diagnostic_repo import DiagnosticMixin

    db_path = _make_player_db(tmp_path, "EmptyPlayer", last_sync_at=None)
    conn = duckdb.connect(str(db_path))

    class _FakeRepo(DiagnosticMixin):
        _xuid = "xuid-empty"
        _gamertag = "EmptyPlayer"
        _attached_dbs: set = set()
        _player_db_path = db_path
        _shared_db_path = db_path
        _metadata_db_path = db_path

        def _get_connection(self):
            return conn

        def get_match_count(self):
            return 0

        @property
        def has_shared(self):
            return False

    repo = _FakeRepo()
    meta = repo.get_sync_metadata()
    conn.close()

    assert meta["last_sync_at"] is None
    assert meta["storage_type"] == "duckdb"


def test_get_sync_metadata_ignores_meta_db(tmp_path):
    """get_sync_metadata() ne lit plus depuis meta.sync_meta (ancienne logique)."""
    from src.data.repositories._diagnostic_repo import DiagnosticMixin

    db_path = _make_player_db(tmp_path, "TestPlayer2", last_sync_at="2026-03-01T08:00:00Z")
    conn = duckdb.connect(str(db_path))

    class _FakeRepo(DiagnosticMixin):
        _xuid = "xuid-test2"
        _gamertag = "TestPlayer2"
        _attached_dbs: set = {"meta"}  # simule meta attaché — ne doit plus être utilisé
        _player_db_path = db_path
        _shared_db_path = db_path
        _metadata_db_path = db_path

        def _get_connection(self):
            return conn

        def get_match_count(self):
            return 0

        @property
        def has_shared(self):
            return False

    repo = _FakeRepo()
    meta = repo.get_sync_metadata()
    conn.close()

    # Doit lire depuis la player DB locale, pas meta
    assert meta["last_sync_at"] == "2026-03-01T08:00:00Z"


def test_get_sync_metadata_broken_table_returns_none(tmp_path):
    """Si sync_meta n'existe pas, last_sync_at=None (pas d'exception propagée)."""
    from src.data.repositories._diagnostic_repo import DiagnosticMixin

    db_path = tmp_path / "broken.duckdb"
    conn = duckdb.connect(str(db_path))
    # Ne pas créer sync_meta

    class _FakeRepo(DiagnosticMixin):
        _xuid = "xuid-broken"
        _gamertag = "Broken"
        _attached_dbs: set = set()
        _player_db_path = db_path
        _shared_db_path = db_path
        _metadata_db_path = db_path

        def _get_connection(self):
            return conn

        def get_match_count(self):
            return 0

        @property
        def has_shared(self):
            return False

    repo = _FakeRepo()
    meta = repo.get_sync_metadata()
    conn.close()

    assert meta["last_sync_at"] is None
