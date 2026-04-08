"""Tests de non-régression pour la remédiation post-v6.2.1.

Couvre :
- P0.2 : Guard process-level sur le watcher média Linux
- P0.3 : Guard migrations success-based (retry après échec)
- P1.1 : Ordre metadata → shared dans le runner de migrations
- P1.2 : HealthCheckResult.recompute_status() après auto-repair
- O3   : Logging des succès après retry intermédiaire
"""

from __future__ import annotations

import threading
from pathlib import Path
from unittest.mock import MagicMock, patch

# ─────────────────────────────────────────────────────────────────────────────
# P0.2 — Guard watcher Linux
# ─────────────────────────────────────────────────────────────────────────────


class TestMediaGuardProcessLevel:
    """Vérifie qu'un seul Observer/thread est créé même avec plusieurs appels."""

    def _make_settings(self) -> MagicMock:
        s = MagicMock()
        s.media_enabled = True
        s.media_captures_base_dir = "/tmp/fake_captures"
        s.media_videos_dir = ""
        s.media_screens_dir = ""
        s.media_watcher_enabled = True
        s.media_indexing_interval_hours = 0
        s.media_tolerance_minutes = 5
        return s

    def test_single_watcher_on_linux(self, tmp_path: Path) -> None:
        """Appeler background_media_indexing 2× sur Linux → 1 seul start_media_watcher."""
        import src.app.media_background as mod

        # Reset du guard process-level
        mod._PERIODIC_STARTED = False

        settings = self._make_settings()
        settings.media_captures_base_dir = str(tmp_path)
        (tmp_path / "SomePlayer").mkdir()

        with (
            patch.object(mod, "platform") as mock_platform,
            patch("src.app.media_watcher.start_media_watcher") as mock_watcher,
        ):
            mock_platform.system.return_value = "Linux"

            mod.background_media_indexing(settings, str(tmp_path / "test.duckdb"))
            mod.background_media_indexing(settings, str(tmp_path / "test.duckdb"))

            assert mock_watcher.call_count == 1

        # Cleanup
        mod._PERIODIC_STARTED = False

    def test_single_thread_on_windows(self, tmp_path: Path) -> None:
        """Appeler background_media_indexing 2× sur Windows → 1 seul thread."""
        import src.app.media_background as mod

        mod._PERIODIC_STARTED = False

        settings = self._make_settings()
        settings.media_captures_base_dir = str(tmp_path)
        (tmp_path / "SomePlayer").mkdir()

        threads_created: list[threading.Thread] = []
        original_init = threading.Thread.__init__

        def tracking_init(self_thread, *args, **kwargs):
            original_init(self_thread, *args, **kwargs)
            if getattr(self_thread, "name", "") == "media-indexer":
                threads_created.append(self_thread)

        with (
            patch.object(mod, "platform") as mock_platform,
            patch.object(threading.Thread, "__init__", tracking_init),
            patch.object(threading.Thread, "start"),
        ):
            mock_platform.system.return_value = "Windows"

            mod.background_media_indexing(settings, str(tmp_path / "test.duckdb"))
            mod.background_media_indexing(settings, str(tmp_path / "test.duckdb"))

            assert len(threads_created) == 1

        mod._PERIODIC_STARTED = False


# ─────────────────────────────────────────────────────────────────────────────
# P0.3 — Guard migrations success-based
# ─────────────────────────────────────────────────────────────────────────────


class TestSharedMigrationsRetry:
    """Vérifie que l'échec d'une migration critique permet le retry."""

    def test_failure_does_not_mark_done(self, tmp_path: Path) -> None:
        """Si ensure_resolution_views échoue, db_key ne doit PAS être dans le set."""
        from src.data.sync._engine_connections import (
            _SHARED_MIGRATIONS_DONE,
            ConnectionMixin,
        )

        _SHARED_MIGRATIONS_DONE.clear()

        mock_conn = MagicMock()
        mock_conn.execute.return_value.fetchone.return_value = ("test_db",)
        fake_path = tmp_path / "test.duckdb"
        fake_path.touch()

        with patch(
            "src.data.sync.migrations.ensure_resolution_views",
            side_effect=RuntimeError("metadata non attachée"),
        ):
            ConnectionMixin._run_shared_migrations(mock_conn, fake_path)

        resolved_key = str(fake_path.resolve())
        assert resolved_key not in _SHARED_MIGRATIONS_DONE

        _SHARED_MIGRATIONS_DONE.clear()

    def test_success_marks_done(self, tmp_path: Path) -> None:
        """Si toutes les migrations réussissent, db_key doit être dans le set."""
        from src.data.sync._engine_connections import (
            _SHARED_MIGRATIONS_DONE,
            ConnectionMixin,
        )

        _SHARED_MIGRATIONS_DONE.clear()

        mock_conn = MagicMock()
        mock_conn.execute.return_value.fetchone.return_value = ("test_db",)
        fake_path = tmp_path / "test.duckdb"
        fake_path.touch()

        with patch("src.data.sync.migrations.ensure_resolution_views"):
            ConnectionMixin._run_shared_migrations(mock_conn, fake_path)

        resolved_key = str(fake_path.resolve())
        assert resolved_key in _SHARED_MIGRATIONS_DONE

        _SHARED_MIGRATIONS_DONE.clear()

    def test_retry_after_failure(self, tmp_path: Path) -> None:
        """Après un échec, un second appel doit retenter les migrations."""
        from src.data.sync._engine_connections import (
            _SHARED_MIGRATIONS_DONE,
            ConnectionMixin,
        )

        _SHARED_MIGRATIONS_DONE.clear()

        mock_conn = MagicMock()
        mock_conn.execute.return_value.fetchone.return_value = ("test_db",)
        fake_path = tmp_path / "test.duckdb"
        fake_path.touch()

        call_count = 0

        def counting_views(conn):
            nonlocal call_count
            call_count += 1
            if call_count == 1:
                raise RuntimeError("transitoire")

        with patch(
            "src.data.sync.migrations.ensure_resolution_views",
            side_effect=counting_views,
        ):
            ConnectionMixin._run_shared_migrations(mock_conn, fake_path)
            ConnectionMixin._run_shared_migrations(mock_conn, fake_path)

        assert call_count == 2
        resolved_key = str(fake_path.resolve())
        assert resolved_key in _SHARED_MIGRATIONS_DONE

        _SHARED_MIGRATIONS_DONE.clear()


# ─────────────────────────────────────────────────────────────────────────────
# P1.2 — HealthCheckResult recompute_status
# ─────────────────────────────────────────────────────────────────────────────


class TestHealthCheckRecompute:
    """Vérifie que recompute_status reflète les mutations post-add."""

    def test_repaired_promotes_to_warning(self) -> None:
        """Un check 'repaired' doit donner un statut global 'warning'."""
        from src.utils.healthcheck_db import CheckDetail, HealthCheckResult

        result = HealthCheckResult(db_name="test")
        result.add(CheckDetail(category="view", name="v_test", status="repaired"))
        assert result.status == "warning"

    def test_recompute_after_mutation(self) -> None:
        """Mutation d'un check broken→repaired + recompute → 'warning' (pas 'error')."""
        from src.utils.healthcheck_db import CheckDetail, HealthCheckResult

        result = HealthCheckResult(db_name="test")
        check = CheckDetail(category="view", name="v_test", status="broken")
        result.add(check)
        assert result.status == "error"

        # Simuler ce que fait _try_repair_views
        check.status = "repaired"
        check.message = "Vue recréée avec succès"

        result.recompute_status()
        assert result.status == "warning"

    def test_recompute_all_ok(self) -> None:
        """Si tous les checks sont 'ok', recompute donne 'ok'."""
        from src.utils.healthcheck_db import CheckDetail, HealthCheckResult

        result = HealthCheckResult(db_name="test")
        result.add(CheckDetail(category="table", name="t1", status="ok"))
        result.add(CheckDetail(category="table", name="t2", status="ok"))

        result.recompute_status()
        assert result.status == "ok"

    def test_error_takes_precedence(self) -> None:
        """Un check 'error' l'emporte sur 'repaired'."""
        from src.utils.healthcheck_db import CheckDetail, HealthCheckResult

        result = HealthCheckResult(db_name="test")
        result.add(CheckDetail(category="table", name="t1", status="repaired"))
        result.add(CheckDetail(category="view", name="v1", status="error"))

        result.recompute_status()
        assert result.status == "error"

    def test_missing_after_repaired_stays_warning(self) -> None:
        """Un check 'missing' après un 'repaired' garde le statut 'warning'."""
        from src.utils.healthcheck_db import CheckDetail, HealthCheckResult

        result = HealthCheckResult(db_name="test")
        result.add(CheckDetail(category="table", name="t1", status="repaired"))
        assert result.status == "warning"
        result.add(CheckDetail(category="view", name="v1", status="missing"))
        assert result.status == "warning"


# ─────────────────────────────────────────────────────────────────────────────
# P1.1 — Runner : metadata avant shared
# ─────────────────────────────────────────────────────────────────────────────


class TestMigrationRunnerOrder:
    """Vérifie que le runner traite metadata avant shared."""

    def test_metadata_processed_before_shared(self, tmp_path: Path) -> None:
        """apply_pending_migrations doit appeler _run_for_db(metadata) avant _run_for_db(shared)."""
        from src.data.migration.runner import MigrationReport, apply_pending_migrations

        call_order: list[str] = []

        def fake_run_for_db(path, target, **kwargs):
            call_order.append(target)
            return MigrationReport(errors=[])

        meta_path = tmp_path / "metadata.duckdb"
        shared_path = tmp_path / "shared.duckdb"
        meta_path.touch()
        shared_path.touch()

        with (
            patch("src.data.migration.runner._run_for_db", side_effect=fake_run_for_db),
            patch("src.data.migration.runner._load_migration_steps"),
        ):
            apply_pending_migrations(
                shared_db_path=shared_path,
                metadata_db_path=meta_path,
            )

        assert "metadata" in call_order
        assert "shared" in call_order
        meta_idx = call_order.index("metadata")
        shared_idx = call_order.index("shared")
        assert meta_idx < shared_idx, f"metadata at {meta_idx}, shared at {shared_idx}"


# ─────────────────────────────────────────────────────────────────────────────
# O3 — Retry logging : succès après tentative intermédiaire
# ─────────────────────────────────────────────────────────────────────────────


class TestRetryLogging:
    """Vérifie que _index_with_retry log un succès après une tentative échouée."""

    def test_logs_success_after_retry(self, tmp_path: Path, caplog) -> None:
        """Si la 1ère tentative échoue et la 2ème réussit, un log info est émis."""
        import logging

        import src.app.media_background as mod

        call_count = 0

        def flaky_index(db_file, gamertag, captures_dir, tolerance):
            nonlocal call_count
            call_count += 1
            if call_count == 1:
                raise OSError("Permission denied")

        with (
            patch.object(mod, "_index_media_for_player", side_effect=flaky_index),
            patch.object(mod.time, "sleep"),
            caplog.at_level(logging.INFO, logger="src.app.media_background"),
        ):
            mod._index_with_retry(
                db_file=tmp_path / "test.duckdb",
                gamertag="TestPlayer",
                captures_dir=tmp_path,
                tolerance=5,
            )

        assert call_count == 2
        assert any("réussie après" in rec.message for rec in caplog.records)


# ─────────────────────────────────────────────────────────────────────────────
# P0.3 — Best-effort : échec non bloquant si critique réussit
# ─────────────────────────────────────────────────────────────────────────────


class TestBestEffortMigrations:
    """Vérifie qu'un échec best-effort ne bloque pas le marquage."""

    def test_best_effort_failure_still_marks_done(self, tmp_path: Path) -> None:
        """Si une migration best-effort échoue mais ensure_resolution_views réussit, db est marquée."""
        from src.data.sync._engine_connections import (
            _SHARED_MIGRATIONS_DONE,
            ConnectionMixin,
        )

        _SHARED_MIGRATIONS_DONE.clear()

        mock_conn = MagicMock()
        mock_conn.execute.return_value.fetchone.return_value = ("test_db",)
        fake_path = tmp_path / "test.duckdb"
        fake_path.touch()

        original_getattr = getattr

        def failing_best_effort(module, name):
            fn = original_getattr(module, name)
            if name in (
                "ensure_match_participants_columns",
                "ensure_performance_indexes",
            ):

                def _raise(conn):
                    raise RuntimeError(f"{name} failed")

                return _raise
            return fn

        with (
            patch("src.data.sync.migrations.ensure_resolution_views"),
            patch("builtins.__import__") as mock_import,
        ):
            mock_mod = MagicMock()
            mock_mod.__name__ = "src.data.sync.migrations"
            mock_mod.ensure_match_participants_columns.side_effect = RuntimeError("fail")
            mock_mod.ensure_performance_indexes.side_effect = RuntimeError("fail")
            mock_mod.ensure_match_registry_spnkr_version = MagicMock()
            mock_mod.ensure_weapon_kills_table = MagicMock()
            mock_import.return_value = mock_mod

            ConnectionMixin._run_shared_migrations(mock_conn, fake_path)

        resolved_key = str(fake_path.resolve())
        assert resolved_key in _SHARED_MIGRATIONS_DONE

        _SHARED_MIGRATIONS_DONE.clear()
