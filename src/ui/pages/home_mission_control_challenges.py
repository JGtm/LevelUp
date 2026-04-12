"""Helpers de défis live pour Mission Control V7."""

from __future__ import annotations

import asyncio
import logging
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Any

from src.data.challenges import (
    build_challenge_badge_candidates,
    extract_challenge_xp,
    extract_threshold_for_success,
    load_challenge_metadata_map,
    resolve_localized_value,
)
from src.utils.paths import DATA_DIR

logger = logging.getLogger(__name__)

_BADGE_CACHE_DIR: Path = DATA_DIR / "cache" / "challenge_badges"
_PNG_SIGNATURE = b"\x89PNG\r\n\x1a\n"
_KNOWN_BADGE_STEMS: tuple[str, ...] = (
    "daily-normal",
    "daily-heroic",
    "daily-legendary",
    "weekly-action-normal",
    "weekly-action-heroic",
    "weekly-action-legendary",
    "weekly-gametype-normal",
    "weekly-gametype-heroic",
    "weekly-gametype-legendary",
    "weekly-weapon-normal",
    "weekly-weapon-heroic",
    "weekly-weapon-legendary",
    "capstone-mythic",
)


@dataclass(frozen=True)
class ActiveChallengeEntry:
    """Défi actif joueur issu du deck HaloStats."""

    path: str
    progress: int | None = None


def known_challenge_badge_stems() -> tuple[str, ...]:
    """Retourne les stems Waypoint connus pour les badges de défis."""
    return _KNOWN_BADGE_STEMS


def format_challenge_expiry(iso_date: str | None) -> str | None:
    """Formate une date ISO8601 courte pour la carte home."""
    if not iso_date:
        return None

    try:
        dt = datetime.fromisoformat(iso_date.replace("Z", "+00:00"))
    except ValueError:
        return iso_date
    return dt.strftime("%d/%m %H:%M UTC")


def build_challenge_summary_from_decks(
    data: dict,
) -> tuple[dict[str, Any] | None, list[ActiveChallengeEntry]]:
    """Construit un résumé des défis actifs à partir de /decks."""
    decks = data.get("AssignedDecks")
    if not isinstance(decks, list):
        return None, []
    if not decks:
        return {"total": 0, "completed": 0, "next_expiry": None}, []

    active_decks = [deck for deck in decks if deck.get("ActiveChallenges")]
    if not active_decks:
        return {"total": 0, "completed": 0, "next_expiry": None}, []

    total = 0
    completed = 0
    expiries: list[str] = []
    active_challenges: list[ActiveChallengeEntry] = []
    for deck in active_decks:
        active = deck.get("ActiveChallenges") or []
        done = deck.get("CompletedChallenges") or []
        total += len(active) + len(done)
        completed += len(done)
        expiry = (deck.get("Expiration") or {}).get("ISO8601Date")
        if expiry:
            expiries.append(expiry)
        for challenge in active:
            path = challenge.get("Path")
            if path:
                active_challenges.append(
                    ActiveChallengeEntry(
                        path=path,
                        progress=_coerce_int(challenge.get("Progress")),
                    )
                )

    return (
        {
            "total": total,
            "completed": completed,
            "next_expiry": format_challenge_expiry(min(expiries) if expiries else None),
        },
        _dedupe_active_challenges(active_challenges),
    )


async def enrich_active_challenges(
    session: Any,
    gamecms_host: str,
    active_challenges: list[ActiveChallengeEntry],
    lang: str,
    challenge_paths: list[str] | None = None,
) -> dict[str, Any]:
    """Retourne XP, texte et badge pour le premier défi actif."""
    active_paths = [entry.path for entry in active_challenges]
    paths_to_fetch = challenge_paths or active_paths
    definitions = await fetch_challenge_definitions(
        session,
        gamecms_host,
        paths_to_fetch,
    )
    stored = load_challenge_metadata_map(active_paths, lang)
    xp_available = 0
    for path in active_paths:
        if path in definitions:
            xp_available += extract_challenge_xp(definitions[path])
        elif path in stored:
            xp_available += stored[path].reward_xp

    primary_entry = next(
        (entry for entry in active_challenges if entry.path in definitions or entry.path in stored),
        None,
    )
    if primary_entry is None:
        return {
            "xp_available": xp_available,
            "title": None,
            "description": None,
            "badge_bytes": None,
            "progress_current": None,
            "progress_target": None,
            "definitions": definitions,
        }

    primary = definitions.get(primary_entry.path)
    stored_primary = stored.get(primary_entry.path)
    title = resolve_localized_value(primary.get("Title"), lang) if primary is not None else None
    description = (
        resolve_localized_value(primary.get("Description"), lang) if primary is not None else None
    )
    if not title and stored_primary is not None:
        title = stored_primary.title
    if not description and stored_primary is not None:
        description = stored_primary.description
    badge_stems = build_challenge_badge_candidates(
        primary_entry.path,
        primary.get("Category")
        if primary is not None
        else stored_primary.category
        if stored_primary
        else None,
        primary.get("Difficulty")
        if primary is not None
        else stored_primary.difficulty
        if stored_primary
        else None,
    )
    badge_bytes, _stem = await fetch_challenge_badge_bytes(session, gamecms_host, badge_stems)
    return {
        "xp_available": xp_available,
        "title": title,
        "description": description,
        "badge_bytes": badge_bytes,
        "progress_current": primary_entry.progress,
        "progress_target": (
            extract_threshold_for_success(primary)
            if primary is not None
            else stored_primary.threshold_for_success
            if stored_primary
            else None
        ),
        "definitions": definitions,
    }


async def fetch_challenge_badge_bytes(
    session: Any,
    gamecms_host: str,
    stems: list[str],
) -> tuple[bytes | None, str | None]:
    """Retourne le premier badge Waypoint valide depuis le cache ou le CMS."""
    for stem in stems:
        cached = _read_cached_badge(stem)
        if cached is not None:
            return cached, stem

    for stem in stems:
        badge_url = f"{gamecms_host}/hi/waypoint/file/images/{stem}.png"
        try:
            async with session.get(badge_url) as resp:
                if resp.status != 200:
                    continue
                data = await resp.read()
        except Exception as exc:
            logger.debug("challenge_badge: échec %s: %s", stem, exc)
            continue
        if data.startswith(_PNG_SIGNATURE):
            _write_cached_badge(stem, data)
            return data, stem
    return None, None


async def prewarm_known_challenge_badges(session: Any, gamecms_host: str) -> list[str]:
    """Télécharge les badges connus dans le cache local si disponibles."""
    ready: list[str] = []
    for stem in known_challenge_badge_stems():
        badge_bytes, found_stem = await fetch_challenge_badge_bytes(session, gamecms_host, [stem])
        if badge_bytes is not None and found_stem is not None:
            ready.append(found_stem)
    return ready


async def _fetch_challenge_definitions(
    session: Any,
    gamecms_host: str,
    challenge_paths: list[str],
) -> dict[str, dict]:
    if not challenge_paths:
        return {}

    tasks = [_fetch_one_definition(session, gamecms_host, path) for path in challenge_paths]
    results = await asyncio.gather(*tasks, return_exceptions=True)
    definitions: dict[str, dict] = {}
    for result in results:
        if isinstance(result, tuple):
            path, data = result
            definitions[path] = data
    return definitions


async def fetch_challenge_definitions(
    session: Any,
    gamecms_host: str,
    challenge_paths: list[str],
) -> dict[str, dict]:
    """Récupère les définitions CMS des défis demandés."""
    return await _fetch_challenge_definitions(session, gamecms_host, challenge_paths)


async def _fetch_one_definition(
    session: Any, gamecms_host: str, path: str
) -> tuple[str, dict] | None:
    definition_url = f"{gamecms_host}/hi/Progression/file/{path.lstrip('/')}"
    async with session.get(definition_url) as resp:
        if resp.status != 200:
            return None
        data = await resp.json(content_type=None)
    return path, data


def _badge_cache_path(stem: str) -> Path:
    return _BADGE_CACHE_DIR / f"{stem}.png"


def _dedupe_active_challenges(
    challenges: list[ActiveChallengeEntry],
) -> list[ActiveChallengeEntry]:
    seen: set[str] = set()
    deduped: list[ActiveChallengeEntry] = []
    for entry in challenges:
        if entry.path in seen:
            continue
        seen.add(entry.path)
        deduped.append(entry)
    return deduped


def _coerce_int(value: Any) -> int | None:
    if isinstance(value, int):
        return value
    if isinstance(value, float) and value.is_integer():
        return int(value)
    return None


def _read_cached_badge(stem: str) -> bytes | None:
    path = _badge_cache_path(stem)
    try:
        data = path.read_bytes()
    except FileNotFoundError:
        return None
    except OSError as exc:
        logger.debug("challenge_badge: lecture cache échouée %s: %s", stem, exc)
        return None
    return data if data.startswith(_PNG_SIGNATURE) else None


def _write_cached_badge(stem: str, data: bytes) -> None:
    path = _badge_cache_path(stem)
    try:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(data)
    except OSError as exc:
        logger.debug("challenge_badge: écriture cache échouée %s: %s", stem, exc)
