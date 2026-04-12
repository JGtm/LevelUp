"""Façade publique pour le domaine des défis Halo Infinite."""

from __future__ import annotations

from pathlib import Path

from src.data import _challenge_catalog as _catalog
from src.data import _challenge_snapshots as _snapshots

StoredChallengeDefinition = _catalog.StoredChallengeDefinition
build_challenge_badge_candidates = _catalog.build_challenge_badge_candidates
collect_challenge_paths_from_decks = _catalog.collect_challenge_paths_from_decks
extract_challenge_xp = _catalog.extract_challenge_xp
extract_threshold_for_success = _catalog.extract_threshold_for_success
normalize_challenge_lang = _catalog.normalize_challenge_lang
resolve_localized_value = _catalog.resolve_localized_value
get_metadata_db_path = _catalog.get_metadata_db_path


def persist_challenge_catalog(definitions: dict[str, dict]) -> dict[str, str]:
    """Persiste le catalogue des défis via la façade publique."""
    _catalog.get_metadata_db_path = get_metadata_db_path
    return _catalog.persist_challenge_catalog(definitions)


def load_challenge_metadata_map(
    challenge_paths: list[str],
    lang: str = "fr",
) -> dict[str, StoredChallengeDefinition]:
    """Charge les définitions stockées via la façade publique."""
    _catalog.get_metadata_db_path = get_metadata_db_path
    return _catalog.load_challenge_metadata_map(challenge_paths, lang)


def persist_challenge_snapshots(
    player_db_path: str | Path,
    xuid: str,
    decks_data: dict,
    definitions: dict[str, dict] | None = None,
    definition_hashes: dict[str, str] | None = None,
) -> int:
    """Persiste les snapshots joueur via la façade publique."""
    _catalog.get_metadata_db_path = get_metadata_db_path
    return _snapshots.persist_challenge_snapshots(
        player_db_path=player_db_path,
        xuid=xuid,
        decks_data=decks_data,
        definitions=definitions,
        definition_hashes=definition_hashes,
    )


__all__ = [
    "StoredChallengeDefinition",
    "build_challenge_badge_candidates",
    "collect_challenge_paths_from_decks",
    "extract_challenge_xp",
    "extract_threshold_for_success",
    "load_challenge_metadata_map",
    "normalize_challenge_lang",
    "persist_challenge_catalog",
    "persist_challenge_snapshots",
    "resolve_localized_value",
]
