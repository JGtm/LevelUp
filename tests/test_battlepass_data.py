"""Tests ciblés pour le catalogue metadata battle pass."""

from __future__ import annotations

from pathlib import Path

import duckdb


def _metadata_path(tmp_path: Path) -> Path:
    return tmp_path / "data" / "warehouse" / "metadata.duckdb"


def _track_payload() -> dict:
    return {
        "XpPerRank": 1000,
        "BattlePassImage": "Progression/Tracks/echoes_within.png",
        "BackgroundImagePath": "Progression/Tracks/echoes_within_bg.png",
        "Name": {
            "value": "Echoes Within",
            "translations": {
                "en-US": "Echoes Within",
                "fr-FR": "Échos intérieurs",
            },
        },
        "Ranks": [{"Rank": 0, "FreeRewards": {}, "PaidRewards": {}}],
    }


def _item_payload(*, fr: bool = True) -> dict:
    translations = {"en-US": "Banished Trophy"}
    descriptions = {"en-US": "Claim your spoils."}
    if fr:
        translations["fr-FR"] = "Trophée des Parias"
        descriptions["fr-FR"] = "Réclamez votre butin."
    return {
        "CommonData": {
            "Quality": "Epic",
            "Type": "ArmorCoating",
            "Title": {"value": "Banished Trophy", "translations": translations},
            "Description": {"value": "Claim your spoils.", "translations": descriptions},
            "DisplayPath": {
                "Media": {"MediaUrl": {"Path": "Progression/Items/banished_trophy.png"}}
            },
        }
    }


def test_persist_and_load_battlepass_track_definition(tmp_path: Path, monkeypatch) -> None:
    """Le reward track doit être persisté et relu depuis metadata.duckdb."""
    from src.data.battlepass import (
        load_battlepass_track_definition,
        persist_battlepass_track_definition,
    )

    metadata_path = _metadata_path(tmp_path)
    monkeypatch.setattr("src.data.battlepass.get_metadata_db_path", lambda: metadata_path)

    content_hash = persist_battlepass_track_definition(
        "RewardTracks/Operations/S03BattlePass.json",
        _track_payload(),
    )
    stored = load_battlepass_track_definition(
        "RewardTracks/Operations/S03BattlePass.json",
        lang="fr",
    )

    assert content_hash
    assert stored is not None
    assert stored.content_hash == content_hash
    assert stored.track_name == "Échos intérieurs"
    assert stored.xp_per_rank == 1000
    assert stored.payload["Ranks"][0]["Rank"] == 0

    with duckdb.connect(str(metadata_path)) as conn:
        langs = conn.execute(
            "SELECT lang FROM battlepass_track_translations ORDER BY lang"
        ).fetchall()

    assert langs == [("en-US",), ("fr-FR",)]


def test_load_battlepass_item_metadata_map_falls_back_to_en_us_when_fr_missing(
    tmp_path: Path,
    monkeypatch,
) -> None:
    """La lecture locale des items doit retomber sur l'anglais si le FR manque."""
    from src.data.battlepass import (
        load_battlepass_item_metadata_map,
        persist_battlepass_item_catalog,
    )

    metadata_path = _metadata_path(tmp_path)
    monkeypatch.setattr("src.data.battlepass.get_metadata_db_path", lambda: metadata_path)

    persist_battlepass_item_catalog(
        {"Inventory/Items/banished_trophy.json": _item_payload(fr=False)}
    )
    metadata = load_battlepass_item_metadata_map(
        ["Inventory/Items/banished_trophy.json"],
        lang="fr",
    )

    entry = metadata["Inventory/Items/banished_trophy.json"]
    assert entry.title == "Banished Trophy"
    assert entry.description == "Claim your spoils."
    assert entry.quality == "Epic"
    assert entry.item_type == "ArmorCoating"
    assert entry.display_path == "Progression/Items/banished_trophy.png"


def test_persist_battlepass_item_catalog_stores_available_translations(
    tmp_path: Path,
    monkeypatch,
) -> None:
    """Le catalogue items doit persister toutes les traductions disponibles."""
    from src.data.battlepass import persist_battlepass_item_catalog

    metadata_path = _metadata_path(tmp_path)
    monkeypatch.setattr("src.data.battlepass.get_metadata_db_path", lambda: metadata_path)

    hashes = persist_battlepass_item_catalog(
        {"Inventory/Items/banished_trophy.json": _item_payload(fr=True)}
    )

    assert hashes
    with duckdb.connect(str(metadata_path)) as conn:
        langs = conn.execute(
            "SELECT lang FROM battlepass_item_translations ORDER BY lang"
        ).fetchall()

    assert langs == [("en-US",), ("fr-FR",)]
