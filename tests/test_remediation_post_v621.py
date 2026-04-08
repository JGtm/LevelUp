"""Tests de non-régression pour la remédiation post-v6.2.1.

Couvre :
- P0.2 : Guard process-level sur le watcher média Linux
- P0.3 : Guard migrations success-based (retry après échec)
- P1.2 : HealthCheckResult.recompute_status() après auto-repair
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
