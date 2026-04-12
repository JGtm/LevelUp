"""Façade publique pour le catalogue metadata battle pass."""

from __future__ import annotations

from src.data import _battlepass_catalog as _catalog

StoredBattlepassItemDefinition = _catalog.StoredBattlepassItemDefinition
StoredBattlepassTrackDefinition = _catalog.StoredBattlepassTrackDefinition
get_metadata_db_path = _catalog.get_metadata_db_path


def persist_battlepass_track_definition(track_path: str, payload: dict) -> str | None:
    """Persiste un reward track battle pass via la façade publique."""
    _catalog.get_metadata_db_path = get_metadata_db_path
    return _catalog.persist_battlepass_track_definition(track_path, payload)


def load_battlepass_track_definition(
    track_path: str,
    lang: str = "fr",
) -> StoredBattlepassTrackDefinition | None:
    """Charge un reward track battle pass via la façade publique."""
    _catalog.get_metadata_db_path = get_metadata_db_path
    return _catalog.load_battlepass_track_definition(track_path, lang)


def persist_battlepass_item_catalog(definitions: dict[str, dict]) -> dict[str, str]:
    """Persiste un lot d'items battle pass via la façade publique."""
    _catalog.get_metadata_db_path = get_metadata_db_path
    return _catalog.persist_battlepass_item_catalog(definitions)


def load_battlepass_item_metadata_map(
    item_paths: list[str],
    lang: str = "fr",
) -> dict[str, StoredBattlepassItemDefinition]:
    """Charge un lot d'items battle pass via la façade publique."""
    _catalog.get_metadata_db_path = get_metadata_db_path
    return _catalog.load_battlepass_item_metadata_map(item_paths, lang)


__all__ = [
    "StoredBattlepassItemDefinition",
    "StoredBattlepassTrackDefinition",
    "load_battlepass_item_metadata_map",
    "load_battlepass_track_definition",
    "persist_battlepass_item_catalog",
    "persist_battlepass_track_definition",
]
