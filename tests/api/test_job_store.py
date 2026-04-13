"""Tests unitaires — JobStore (Sprint 3.3 + 3.4 V7 onboarding).

Couvre :
- is_expired() inclut "interrupted" dans les terminaux
- _load() change "running" → "interrupted" au rechargement
- metadata serialisé dans to_dict / from_dict
- find_active_initial_sync retourne le bon job (par player_slug)
- find_active_initial_sync retourne None si le job est terminal
"""

from __future__ import annotations

import json
import time
from datetime import datetime, timezone
from pathlib import Path

import pytest

# ---------------------------------------------------------------------------
# Fixture store isolé
# ---------------------------------------------------------------------------


@pytest.fixture
def fresh_store(tmp_path: Path):
    """Retourne un JobStore pointant vers un fichier temporaire vide."""
    from apps.api.app.services.job_store import JobStore

    jobs_file = tmp_path / "jobs_test.json"
    store = JobStore(jobs_file=jobs_file)
    return store


# ---------------------------------------------------------------------------
# Tests is_expired avec "interrupted"
# ---------------------------------------------------------------------------


def test_is_expired_interrupted_is_terminal(fresh_store) -> None:
    """Un job 'interrupted' doit être traité comme terminal pour la rétention."""
    job = fresh_store.create("test_job", metadata={})
    fresh_store.update(
        job.job_id,
        status="interrupted",
    )
    from apps.api.app.services.job_store import JOB_RETENTION_S

    with fresh_store._jobs_lock:
        entry = fresh_store._jobs[job.job_id]

    # Forcer finished_at dans le passé (> JOB_RETENTION_S)
    past = datetime.fromtimestamp(time.time() - JOB_RETENTION_S - 10, tz=timezone.utc)
    entry.finished_at = past

    assert entry.is_expired() is True


def test_is_expired_running_is_not_expired(fresh_store) -> None:
    """Un job 'running' ne doit jamais être considéré expiré."""
    job = fresh_store.create("test_job")
    fresh_store.update(job.job_id, status="running")
    with fresh_store._jobs_lock:
        entry = fresh_store._jobs[job.job_id]
    assert entry.is_expired() is False


# ---------------------------------------------------------------------------
# Tests _load() : running → interrupted au restart
# ---------------------------------------------------------------------------


def test_load_marks_running_as_interrupted(tmp_path: Path) -> None:
    """Un job 'running' dans le fichier JSON doit devenir 'interrupted' au _load()."""
    from apps.api.app.services.job_store import JobStore

    jobs_file = tmp_path / "jobs_restart.json"
    now_iso = datetime.now(timezone.utc).isoformat()

    # Écrire un job running manuellement
    jobs_data = [
        {
            "job_id": "job-running-test",
            "job_type": "initial_sync",
            "status": "running",
            "created_at": now_iso,
            "started_at": now_iso,
            "finished_at": None,
            "progress_pct": 42,
            "current_step": "fetch",
            "metadata": {"player_slug": "testuser"},
            "warnings": [],
        }
    ]
    jobs_file.write_text(json.dumps(jobs_data), encoding="utf-8")

    store = JobStore(jobs_file=jobs_file)
    job = store.get_job("job-running-test")

    assert job is not None
    assert job.status == "interrupted"
    assert any("interrompu" in w.lower() or "red" in w.lower() for w in job.warnings)


# ---------------------------------------------------------------------------
# Tests metadata sérialisation
# ---------------------------------------------------------------------------


def test_metadata_serialized_in_to_dict(fresh_store) -> None:
    """metadata doit apparaître dans to_dict()."""
    job = fresh_store.create("initial_sync", metadata={"player_slug": "myplayer", "env": "test"})
    with fresh_store._jobs_lock:
        entry = fresh_store._jobs[job.job_id]
    d = entry.to_dict()
    assert d["metadata"]["player_slug"] == "myplayer"
    assert d["metadata"]["env"] == "test"


def test_metadata_deserialized_from_dict() -> None:
    """from_dict() doit restaurer metadata."""
    from apps.api.app.services.job_store import _JobEntry

    now_iso = datetime.now(timezone.utc).isoformat()
    d = {
        "job_id": "test-meta",
        "job_type": "initial_sync",
        "status": "queued",
        "created_at": now_iso,
        "metadata": {"player_slug": "restored", "extra": 42},
        "warnings": [],
    }
    entry = _JobEntry.from_dict(d)
    assert entry.metadata["player_slug"] == "restored"
    assert entry.metadata["extra"] == 42


def test_metadata_empty_dict_when_missing_in_json() -> None:
    """from_dict() avec metadata absent → {} (rétrocompatibilité)."""
    from apps.api.app.services.job_store import _JobEntry

    now_iso = datetime.now(timezone.utc).isoformat()
    d = {
        "job_id": "test-no-meta",
        "job_type": "smoke_test",
        "status": "succeeded",
        "created_at": now_iso,
        "warnings": [],
    }
    entry = _JobEntry.from_dict(d)
    assert entry.metadata == {}


# ---------------------------------------------------------------------------
# Tests find_active_initial_sync
# ---------------------------------------------------------------------------


def test_find_active_initial_sync_returns_active_job(fresh_store) -> None:
    """find_active_initial_sync retourne le job actif pour le bon player_slug."""
    job = fresh_store.create("initial_sync", metadata={"player_slug": "alice"})
    fresh_store.update(job.job_id, status="running")

    result = fresh_store.find_active_initial_sync("alice")
    assert result is not None
    assert result.job_id == job.job_id


def test_find_active_initial_sync_returns_none_for_other_player(fresh_store) -> None:
    """find_active_initial_sync ne retourne rien pour un autre joueur."""
    job = fresh_store.create("initial_sync", metadata={"player_slug": "alice"})
    fresh_store.update(job.job_id, status="running")

    result = fresh_store.find_active_initial_sync("bob")
    assert result is None


def test_find_active_initial_sync_returns_none_when_terminal(fresh_store) -> None:
    """find_active_initial_sync retourne None pour un job terminé."""
    job = fresh_store.create("initial_sync", metadata={"player_slug": "alice"})
    fresh_store.update(job.job_id, status="succeeded")

    result = fresh_store.find_active_initial_sync("alice")
    assert result is None


def test_find_active_initial_sync_returns_queued(fresh_store) -> None:
    """find_active_initial_sync inclut les jobs 'queued' (pas encore démarrés)."""
    job = fresh_store.create("initial_sync", metadata={"player_slug": "charlie"})
    # Pas encore démarré (status=queued)

    result = fresh_store.find_active_initial_sync("charlie")
    assert result is not None
    assert result.job_id == job.job_id
