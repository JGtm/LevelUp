"""Tests Phase E — suppress cleanup, bare connect fix, bitmasks.

Couvre E.3 (suppress→try/except), E.4 (bare connect), E.5 (bitmask cleanup).
"""

from __future__ import annotations

import logging

# ─────────────────────────────────────────────────────────────────────────────
# E.5 — Bitmask non-collision
# ─────────────────────────────────────────────────────────────────────────────


def test_backfill_flags_no_collision_with_match_bits():
    """lusr/csr supprimés : collision avec MatchBits.EVENTS/ASSETS éliminée."""
    from src.data.sync.migrations import BACKFILL_FLAGS

    assert "lusr" not in BACKFILL_FLAGS, "collision avec MatchBits.EVENTS (1<<16)"
    assert "csr" not in BACKFILL_FLAGS, "collision avec MatchBits.ASSETS (1<<17)"
    assert "performance_scores" not in BACKFILL_FLAGS, (
        "sémantique incorrecte (granularité joueur×match, pas par match)"
    )


def test_compute_backfill_mask_no_performance_scores():
    """_compute_backfill_mask ne pose plus le bit performance_scores."""
    from src.data.sync._match_processing_helpers import MatchProcessingHelpersMixin
    from src.data.sync.migrations import BACKFILL_FLAGS
    from src.data.sync.models_sync import SyncOptions

    # Créer un helper minimal avec les attributs requis par le protocol
    class _FakeEngine(MatchProcessingHelpersMixin):
        _xuid = "test"
        _metadata_resolver = None

    opts = SyncOptions(with_skill=True, with_participants=True, with_aliases=True, with_assets=True)
    mask = _FakeEngine()._compute_backfill_mask(opts)

    # Le bit performance_scores ne doit plus être dans le masque
    perf_bit = 1 << 4  # ancienne valeur de BACKFILL_FLAGS["performance_scores"]
    assert (mask & perf_bit) == 0, f"performance_scores bit (1<<4) encore présent dans mask={mask}"

    # Les bits attendus doivent toujours être présents
    assert mask & BACKFILL_FLAGS["medals"]
    assert mask & BACKFILL_FLAGS["personal_scores"]
    assert mask & BACKFILL_FLAGS["accuracy"]


# ─────────────────────────────────────────────────────────────────────────────
# E.4 — Bare connect → context manager
# ─────────────────────────────────────────────────────────────────────────────


def test_compute_dominance_uses_context_manager(tmp_path):
    """_compute_dominance_post_sync utilise un context manager DuckDB (pas de fuite)."""
    import inspect

    from src.data.sync import engine as engine_module

    source = inspect.getsource(engine_module.DuckDBSyncEngine._compute_dominance_post_sync)
    # Vérifier que le contexte manager est utilisé et pas un bare connect + try/finally
    assert "with _ddb.connect" in source, (
        "doit utiliser 'with _ddb.connect(...)' comme context manager"
    )
    assert "_sconn.close()" not in source, "ne doit plus appeler _sconn.close() manuellement"


# ─────────────────────────────────────────────────────────────────────────────
# E.3 — suppress(Exception) absent de la chaîne sync
# ─────────────────────────────────────────────────────────────────────────────


def test_no_suppress_exception_in_sync_chain():
    """Zéro contextlib.suppress(Exception) dans la chaîne sync."""
    import re
    from pathlib import Path

    sync_dir = Path(__file__).parent.parent / "src" / "data" / "sync"
    ui_file = Path(__file__).parent.parent / "src" / "ui" / "_sync_duckdb_ops.py"

    pattern = re.compile(r"suppress\(Exception\)")
    violations = []

    for path in list(sync_dir.glob("*.py")) + [ui_file]:
        text = path.read_text(encoding="utf-8")
        for lineno, line in enumerate(text.splitlines(), start=1):
            if pattern.search(line):
                violations.append(f"{path.name}:{lineno}: {line.strip()}")

    assert not violations, "suppress(Exception) trouvé dans la chaîne sync :\n" + "\n".join(
        violations
    )


# ─────────────────────────────────────────────────────────────────────────────
# E.3 — CHECKPOINT failure loggée en warning (pas masquée)
# ─────────────────────────────────────────────────────────────────────────────


def test_checkpoint_failure_logged_as_warning(caplog):
    """Un CHECKPOINT raté est loggé en WARNING, pas masqué."""
    from unittest.mock import MagicMock

    from src.data.sync._engine_connections import ConnectionMixin

    class _FakeEngine(ConnectionMixin):
        _connection = None
        _shared_connection = None
        _pve_connection = None
        _shared_db_path = None
        _player_db_path = None

    engine = _FakeEngine()

    # Simuler une connexion qui lève sur CHECKPOINT
    bad_conn = MagicMock()
    bad_conn.execute.side_effect = Exception("WAL flush error")
    bad_conn.close.return_value = None
    engine._connection = bad_conn

    with caplog.at_level(logging.WARNING, logger="src.data.sync._engine_connections"):
        engine.close()

    assert any("checkpoint_failed" in r.message for r in caplog.records), (
        "event=checkpoint_failed doit être loggé en WARNING quand CHECKPOINT échoue"
    )
