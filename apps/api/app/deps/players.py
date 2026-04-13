"""Dépendance FastAPI pour la résolution du joueur courant.

Résout `player_slug` → chemin vers `stats.duckdb` du joueur,
en validant que ce slug existe bien dans `db_profiles.json`.

En DEMO_MODE, pointe vers les fixtures `tests/fixtures/ref_player/`.
"""

from __future__ import annotations

import json
from dataclasses import dataclass
from pathlib import Path

import structlog
from fastapi import Path as FastAPIPath

from apps.api.app.core.config import get_settings
from apps.api.app.core.errors import ApiError
from apps.api.app.schemas.common import PlayerSummary

logger = structlog.get_logger(__name__)


@dataclass(frozen=True)
class PlayerContext:
    """Contexte joueur résolu — injecté dans les endpoints par dépendance."""

    player_slug: str
    gamertag: str
    xuid: str
    waypoint_player: str
    db_path: str
    shared_db_path: str
    metadata_db_path: str
    is_demo: bool = False


def _repo_root() -> Path:
    settings = get_settings()
    return Path(settings.repo_root)


def load_db_profiles() -> list[dict]:
    """Charge et retourne la liste des profils depuis db_profiles.json."""
    settings = get_settings()
    profiles_path = Path(settings.db_profiles_path)
    if not profiles_path.exists():
        return []
    try:
        data = json.loads(profiles_path.read_text(encoding="utf-8"))
        # Supporte les formats liste et dict {"profiles": [...]}
        if isinstance(data, list):
            return data
        return data.get("profiles", [])
    except Exception:
        logger.warning("db_profiles_load_error", path=str(profiles_path))
        return []


def _slug_from_gamertag(gamertag: str) -> str:
    """Convertit un gamertag en slug URL-safe (lowercase, espaces → tirets)."""
    return gamertag.lower().replace(" ", "-")


def get_available_players() -> list[PlayerSummary]:
    """Retourne la liste des joueurs disponibles sous forme de `PlayerSummary`."""
    settings = get_settings()
    if settings.demo_mode:
        return _demo_players()

    profiles = load_db_profiles()
    players: list[PlayerSummary] = []
    for p in profiles:
        gamertag = p.get("gamertag") or p.get("name", "")
        if not gamertag:
            continue
        slug = _slug_from_gamertag(gamertag)
        players.append(
            PlayerSummary(
                player_slug=slug,
                gamertag=gamertag,
                xuid=p.get("xuid", ""),
                waypoint_player=p.get("waypoint_player", gamertag),
            )
        )
    return players


def _read_demo_xuid(fixtures_dir: Path) -> str:
    """Lit le vrai xuid depuis xuid.txt dans les fixtures, ou retourne le sentinel."""
    xuid_file = fixtures_dir / "xuid.txt"
    if xuid_file.exists():
        return xuid_file.read_text(encoding="utf-8").strip()
    return "0000000000000000"


def _demo_players() -> list[PlayerSummary]:
    """Retourne une liste de joueurs de démo pointant sur les fixtures."""
    from apps.api.app.core.config import get_settings

    settings = get_settings()
    fixtures_dir = Path(settings.demo_fixtures_dir)
    xuid = _read_demo_xuid(fixtures_dir)
    return [
        PlayerSummary(
            player_slug="demo-player",
            gamertag="DemoPlayer",
            xuid=xuid,
            waypoint_player="DemoPlayer",
            is_demo=True,
        )
    ]


def resolve_player(
    player_slug: str = FastAPIPath(..., description="Slug du joueur"),
) -> PlayerContext:
    """Dependency FastAPI : résout `player_slug` en `PlayerContext`.

    Raises `ApiError(404)` si le joueur n'existe pas.
    """
    settings = get_settings()
    repo = _repo_root()

    if settings.demo_mode:
        _valid_demo_slugs = {"demo", "demo-player"}
        if player_slug not in _valid_demo_slugs:
            raise ApiError.not_found("Joueur", player_slug)
        fixtures_dir = Path(settings.demo_fixtures_dir)
        xuid = _read_demo_xuid(fixtures_dir)
        return PlayerContext(
            player_slug=player_slug,
            gamertag="DemoPlayer",
            xuid=xuid,
            waypoint_player="DemoPlayer",
            db_path=str(fixtures_dir / "stats.duckdb"),
            shared_db_path=str(fixtures_dir / "shared_matches_v2.duckdb"),
            metadata_db_path=str(fixtures_dir / "metadata.duckdb"),
            is_demo=True,
        )

    profiles = load_db_profiles()
    for p in profiles:
        gamertag = p.get("gamertag") or p.get("name", "")
        if not gamertag:
            continue
        if _slug_from_gamertag(gamertag) == player_slug:
            db_path = p.get("db_path") or str(repo / "data" / "players" / gamertag / "stats.duckdb")
            return PlayerContext(
                player_slug=player_slug,
                gamertag=gamertag,
                xuid=p.get("xuid", ""),
                waypoint_player=p.get("waypoint_player", gamertag),
                db_path=db_path,
                shared_db_path=str(repo / "data" / "warehouse" / "shared_matches_v2.duckdb"),
                metadata_db_path=str(repo / "data" / "warehouse" / "metadata.duckdb"),
            )

    raise ApiError.not_found("Joueur", player_slug)
