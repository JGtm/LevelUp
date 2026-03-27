"""Tests Phase J — fix R/W→R/O fanout + LEFT JOIN défensif + vérification post-pipeline."""

from __future__ import annotations

import logging

import duckdb
import polars as pl

# ─────────────────────────────────────────────────────────────────────────────
# J.3 — shared_read_only
# ─────────────────────────────────────────────────────────────────────────────


def test_shared_read_only_default_false(tmp_path):
    """DuckDBSyncEngine crée avec shared_read_only=False par défaut."""
    from src.data.sync.engine import DuckDBSyncEngine

    player_db = tmp_path / "stats.duckdb"
    player_db.touch()
    shared_db = tmp_path / "shared.duckdb"
    shared_db.touch()

    engine = DuckDBSyncEngine(
        player_db_path=player_db,
        xuid="xuid-test",
        gamertag="TestGT",
        shared_db_path=shared_db,
    )
    assert engine._shared_read_only is False
    engine._connection = None  # éviter ouverture


def test_shared_read_only_true(tmp_path):
    """DuckDBSyncEngine avec shared_read_only=True → _shared_read_only=True."""
    from src.data.sync.engine import DuckDBSyncEngine

    player_db = tmp_path / "stats.duckdb"
    player_db.touch()
    shared_db = tmp_path / "shared.duckdb"
    shared_db.touch()

    engine = DuckDBSyncEngine(
        player_db_path=player_db,
        xuid="xuid-test",
        gamertag="TestGT",
        shared_db_path=shared_db,
        shared_read_only=True,
    )
    assert engine._shared_read_only is True


def test_get_shared_connection_uses_read_only_flag(tmp_path):
    """_get_shared_connection() ouvre shared en R/O quand shared_read_only=True."""
    import inspect

    from src.data.sync._engine_connections import ConnectionMixin

    source = inspect.getsource(ConnectionMixin._get_shared_connection)
    assert "read_only=read_only" in source, (
        "_get_shared_connection doit propager le flag read_only à duckdb.connect()"
    )


# ─────────────────────────────────────────────────────────────────────────────
# J.6 — _verify_enrichment_completeness
# ─────────────────────────────────────────────────────────────────────────────


SCHEMA_PLAYER_MATCH_ENRICHMENT = """
CREATE TABLE IF NOT EXISTS player_match_enrichment (
    match_id TEXT PRIMARY KEY,
    performance_score FLOAT,
    session_id TEXT
)
"""


def test_verify_enrichment_completeness_missing_perf(tmp_path, caplog):
    """_verify_enrichment_completeness log enrichment_incomplete si perf manquant."""
    from src.data.sync.engine import DuckDBSyncEngine

    player_db = tmp_path / "stats.duckdb"
    conn = duckdb.connect(str(player_db))
    conn.execute(SCHEMA_PLAYER_MATCH_ENRICHMENT)
    conn.execute("INSERT INTO player_match_enrichment VALUES ('match-1', NULL, 'sess-1')")
    conn.close()

    engine = DuckDBSyncEngine.__new__(DuckDBSyncEngine)
    engine._player_db_path = player_db
    engine._connection = duckdb.connect(str(player_db))

    with caplog.at_level(logging.WARNING, logger="src.data.sync.engine"):
        engine._verify_enrichment_completeness(["match-1"])

    engine._connection.close()
    assert any("enrichment_incomplete" in r.message for r in caplog.records), (
        "event=enrichment_incomplete doit être loggé quand performance_score est NULL"
    )


def test_verify_enrichment_completeness_all_ok(tmp_path, caplog):
    """_verify_enrichment_completeness log enrichment_complete quand tout est ok."""
    from src.data.sync.engine import DuckDBSyncEngine

    player_db = tmp_path / "stats.duckdb"
    conn = duckdb.connect(str(player_db))
    conn.execute(SCHEMA_PLAYER_MATCH_ENRICHMENT)
    conn.execute("INSERT INTO player_match_enrichment VALUES ('match-1', 75.5, 'sess-1')")
    conn.close()

    engine = DuckDBSyncEngine.__new__(DuckDBSyncEngine)
    engine._player_db_path = player_db
    engine._connection = duckdb.connect(str(player_db))

    with caplog.at_level(logging.INFO, logger="src.data.sync.engine"):
        engine._verify_enrichment_completeness(["match-1"])

    engine._connection.close()
    assert any("enrichment_complete" in r.message for r in caplog.records), (
        "event=enrichment_complete doit être loggé quand tous les enrichissements sont présents"
    )


def test_verify_enrichment_completeness_empty_ids(tmp_path, caplog):
    """_verify_enrichment_completeness ne fait rien si inserted_ids est vide."""
    from src.data.sync.engine import DuckDBSyncEngine

    engine = DuckDBSyncEngine.__new__(DuckDBSyncEngine)
    with caplog.at_level(logging.INFO, logger="src.data.sync.engine"):
        engine._verify_enrichment_completeness([])

    assert not caplog.records


# ─────────────────────────────────────────────────────────────────────────────
# J.7 — LEFT JOIN défensif dans _join_perf_frames
# ─────────────────────────────────────────────────────────────────────────────


def test_join_perf_frames_left_join_keeps_match_with_missing_coequipier():
    """LEFT JOIN : match conservé si coéquipier n'a pas de performance_score."""
    from src.analysis._performance_squad import _join_perf_frames

    # Joueur principal : match-1 et match-2
    df_main = pl.DataFrame(
        {
            "match_id": ["match-1", "match-2"],
            "start_time": ["2026-01-01T10:00:00", "2026-01-01T11:00:00"],
            "session_id": ["s1", "s1"],
            "performance_score": [80.0, 70.0],
            "outcome": [2, 3],
        }
    )

    # Coéquipier : uniquement match-1 (match-2 manquant)
    df_coequipier = pl.DataFrame(
        {
            "match_id": ["match-1"],
            "performance_score": [60.0],
        }
    )

    result = _join_perf_frames([df_main, df_coequipier])

    # Les deux matchs doivent être présents
    assert len(result) == 2, f"Attendu 2 matchs, obtenu {len(result)}"
    assert "match-1" in result["match_id"].to_list()
    assert "match-2" in result["match_id"].to_list()


def test_join_perf_frames_inner_removed():
    """Vérifier que how='inner' n'est plus présent dans _join_perf_frames."""
    import inspect

    from src.analysis import _performance_squad

    source = inspect.getsource(_performance_squad._join_perf_frames)
    assert 'how="inner"' not in source, "_join_perf_frames ne doit plus utiliser how='inner'"
    assert 'how="left"' in source, "_join_perf_frames doit utiliser how='left'"
