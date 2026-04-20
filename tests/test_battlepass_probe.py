"""Tests unitaires ciblés pour la collecte battle pass complète."""

from __future__ import annotations

from src.data.battlepass_probe import (
    _collect_currency_paths,
    _collect_inventory_paths,
    _collect_item_image_paths,
    _collect_track_image_paths,
    _cache_currency_assets,
    _extract_track_paths,
)


def test_extract_track_paths_includes_active_and_deduplicates() -> None:
    payload = {
        "ActiveOperationRewardTrackPath": "RewardTracks/Operations/Season6.json",
        "OperationRewardTracks": [
            {"RewardTrackPath": "RewardTracks/Operations/Season6.json"},
            {"RewardTrackPath": "RewardTracks/Operations/Season5.json"},
            {"RewardTrackPath": "RewardTracks/Operations/Season5.json"},
        ],
    }

    assert _extract_track_paths(payload) == [
        "RewardTracks/Operations/Season6.json",
        "RewardTracks/Operations/Season5.json",
    ]


def test_collect_inventory_and_currency_paths_from_tracks() -> None:
    track_payloads = [
        {
            "Ranks": [
                {
                    "FreeRewards": {
                        "InventoryRewards": [
                            {"InventoryItemPath": "Inventory/Items/a.json"},
                        ],
                        "CurrencyRewards": [
                            {"CurrencyPath": "Currency/xpboost.json"},
                        ],
                    },
                    "PaidRewards": {
                        "InventoryRewards": [
                            {"InventoryItemPath": "Inventory/Items/b.json"},
                            {"InventoryItemPath": "Inventory/Items/a.json"},
                        ],
                    },
                }
            ]
        }
    ]

    assert _collect_inventory_paths(track_payloads) == [
        "Inventory/Items/a.json",
        "Inventory/Items/b.json",
    ]
    assert _collect_currency_paths(track_payloads) == ["Currency/xpboost.json"]


def test_collect_track_and_item_image_paths() -> None:
    track_payloads = [
        {
            "BattlePassImage": "Progression/Tracks/season6.png",
            "BackgroundImagePath": "Progression/Tracks/season6-bg.png",
            "SummaryImagePath": "Progression/Tracks/season6-summary.png",
        }
    ]
    item_payloads = [
        {
            "CommonData": {
                "DisplayPath": {
                    "Media": {
                        "MediaUrl": {"Path": "Progression/Items/reward-a.png"}
                    }
                }
            }
        }
    ]

    assert _collect_track_image_paths(track_payloads) == [
        "Progression/Tracks/season6.png",
        "Progression/Tracks/season6-bg.png",
        "Progression/Tracks/season6-summary.png",
    ]
    assert _collect_item_image_paths(item_payloads) == ["Progression/Items/reward-a.png"]


def test_cache_currency_assets_splits_repo_static_and_external() -> None:
    cached_count, missing, external = _cache_currency_assets(
        [
            "Currency/Currencies/xpboost.json",
            "Currency/Currencies/rerollcurrency.json",
            "Currency/Currencies/softcurrency.json",
        ]
    )

    assert cached_count == 2
    assert missing == []
    assert external == ["Currency/Currencies/softcurrency.json"]