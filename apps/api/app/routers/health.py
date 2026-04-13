"""Router santé — `GET /api/v1/health`.

Retourne l'état de santé de l'API et des ressources critiques (DBs).
"""

from __future__ import annotations

import time
from pathlib import Path

from fastapi import APIRouter
from pydantic import BaseModel

from apps.api.app.core.config import get_settings
from apps.api.app.deps.players import load_db_profiles

router = APIRouter(prefix="/health", tags=["health"])


class HealthResponse(BaseModel):
    """Réponse de health check."""

    status: str  # "ok" | "degraded" | "down"
    version: str
    uptime_seconds: float
    checks: dict[str, str]


_START_TIME = time.monotonic()


@router.get("", response_model=HealthResponse, summary="Health check")
async def health() -> HealthResponse:
    """Vérifie l'état de l'API et ses dépendances critiques."""
    settings = get_settings()
    checks: dict[str, str] = {}

    # Vérification db_profiles.json
    profiles_path = Path(settings.db_profiles_path)
    if settings.demo_mode:
        checks["db_profiles"] = "skipped_demo_mode"
    elif profiles_path.exists():
        profiles = load_db_profiles()
        checks["db_profiles"] = f"ok ({len(profiles)} joueur(s))"
    else:
        checks["db_profiles"] = "missing"

    # Vérification metadata.duckdb
    repo = Path(settings.repo_root)
    meta_db = repo / "data" / "warehouse" / "metadata.duckdb"
    if settings.demo_mode:
        fixtures = Path(settings.demo_fixtures_dir)
        meta_db = fixtures / "metadata.duckdb"
    checks["metadata_db"] = "ok" if meta_db.exists() else "missing"

    # Vérification shared DB
    shared_db = repo / "data" / "warehouse" / "shared_matches_v2.duckdb"
    if settings.demo_mode:
        shared_db = Path(settings.demo_fixtures_dir) / "shared_matches_v2.duckdb"
    checks["shared_db"] = "ok" if shared_db.exists() else "missing"

    overall = "ok" if all("missing" not in v for v in checks.values()) else "degraded"

    return HealthResponse(
        status=overall,
        version=settings.app_version,
        uptime_seconds=round(time.monotonic() - _START_TIME, 2),
        checks=checks,
    )
