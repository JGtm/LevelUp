"""Tests standalone pour transformers metadata.

Objectif: vérifier que les fonctions clés du package `src/data/sync/transformers`
sont bien exposées et se comportent correctement.

Note: `transformers.py` a été converti en package `transformers/` (v5).
On importe désormais directement depuis le package au lieu d'un chargement
par fichier via importlib.
"""

from __future__ import annotations

from src.data.sync.transformers._helpers import _extract_asset_id, _extract_public_name
from src.data.sync.transformers._match import transform_match_stats


def test_extract_public_name_exists() -> None:
    """Test que _extract_public_name est importable et callable."""
    assert callable(_extract_public_name)


def test_extract_asset_id_exists() -> None:
    """Test que _extract_asset_id est importable et callable."""
    assert callable(_extract_asset_id)


def test_transform_match_stats_exists() -> None:
    """Test que transform_match_stats est importable et callable."""
    assert callable(transform_match_stats)


def test_extract_public_name_with_public_name() -> None:
    """Test extraction PublicName quand présent."""
    match_info = {"Playlist": {"AssetId": "playlist-123", "PublicName": "Ranked Slayer"}}
    result = _extract_public_name(match_info, "Playlist")
    assert result == "Ranked Slayer"


def test_extract_public_name_without_public_name() -> None:
    """Test extraction PublicName quand absent."""
    match_info = {"Playlist": {"AssetId": "playlist-123"}}
    result = _extract_public_name(match_info, "Playlist")
    assert result is None


def test_extract_asset_id_with_asset_id() -> None:
    """Test extraction AssetId quand présent."""
    match_info = {"Playlist": {"AssetId": "playlist-123"}}
    result = _extract_asset_id(match_info, "Playlist")
    assert result == "playlist-123"


def test_extract_asset_id_without_asset_id() -> None:
    """Test extraction AssetId quand absent."""
    match_info = {"Playlist": {}}
    result = _extract_asset_id(match_info, "Playlist")
    assert result is None
