"""Tests ciblés pour les helpers battle pass de la home V7."""

from __future__ import annotations

import asyncio

from src.data.battlepass import (
    StoredBattlepassItemDefinition,
    StoredBattlepassTrackDefinition,
)
from src.ui.pages.home_mission_control_battlepass import (
    BattlepassRewardPreview,
    BattlepassTierPreview,
    build_currency_reward_label,
    build_tier_preview,
    select_recent_reward_ranks,
)
from src.ui.pages.home_mission_control_battlepass_assets import read_currency_asset
from src.ui.pages.home_mission_control_battlepass_render import _select_tier_window


def test_select_recent_reward_ranks_keeps_current_rank_and_skips_empty() -> None:
    """Les derniers paliers doivent inclure le rang courant et ignorer les paliers vides."""
    ranks = [
        {
            "Rank": 1,
            "FreeRewards": {
                "CurrencyRewards": [
                    {"CurrencyPath": "Currency/Currencies/xpboost.json", "Amount": 1}
                ]
            },
            "PaidRewards": {},
        },
        {
            "Rank": 2,
            "FreeRewards": {"InventoryRewards": []},
            "PaidRewards": {"InventoryRewards": []},
        },
        {
            "Rank": 3,
            "FreeRewards": {},
            "PaidRewards": {
                "InventoryRewards": [{"InventoryItemPath": "inventory/a.json", "Amount": 1}]
            },
        },
        {
            "Rank": 4,
            "FreeRewards": {
                "InventoryRewards": [{"InventoryItemPath": "inventory/b.json", "Amount": 1}]
            },
            "PaidRewards": {},
        },
    ]

    selected = select_recent_reward_ranks(ranks, current_rank=3, limit=3)

    assert [rank["Rank"] for rank in selected] == [1, 3]


def test_build_currency_reward_label_localizes_known_currency_rewards() -> None:
    """Les monnaies connues doivent être rendues avec un libellé exploitable."""
    assert build_currency_reward_label("Currency/Currencies/xpboost.json", 1, "fr") == "Boost d'XP"
    assert (
        build_currency_reward_label("Currency/Currencies/rerollcurrency.json", 2, "en")
        == "Challenge Swap x2"
    )


def test_build_tier_preview_formats_inventory_and_currency_labels() -> None:
    """Le résumé d'un palier doit combiner inventaire et monnaies."""
    rank_payload = {
        "Rank": 51,
        "FreeRewards": {
            "InventoryRewards": [{"InventoryItemPath": "inventory/vehicle.json", "Amount": 1}],
            "CurrencyRewards": [],
        },
        "PaidRewards": {
            "InventoryRewards": [],
            "CurrencyRewards": [{"CurrencyPath": "Currency/Currencies/xpboost.json", "Amount": 1}],
        },
    }
    item_details = {
        "inventory/vehicle.json": {
            "title": "Équipe Cerberus",
            "quality": "Rare",
            "type": "VehicleEmblem",
        }
    }

    preview = build_tier_preview(rank_payload, item_details, "fr")

    assert preview is not None
    assert preview.rank == 51
    assert preview.free_rewards == (
        BattlepassRewardPreview(
            label="Rare · Équipe Cerberus",
            description=None,
            image_bytes=None,
            tile_label="ÉC",
        ),
    )
    assert len(preview.premium_rewards) == 1
    assert preview.premium_rewards[0].label == "Boost d'XP"
    assert preview.premium_rewards[0].description is None
    assert preview.premium_rewards[0].tile_label == "XP"
    assert preview.premium_rewards[0].image_bytes is not None
    assert preview.premium_rewards[0].image_bytes.startswith(b"\x89PNG")


def test_build_tier_preview_can_keep_empty_tier_for_full_navigation() -> None:
    """Le navigateur complet doit pouvoir conserver un palier sans récompense."""
    rank_payload = {
        "Rank": 12,
        "FreeRewards": {"InventoryRewards": [], "CurrencyRewards": []},
        "PaidRewards": {"InventoryRewards": [], "CurrencyRewards": []},
    }

    assert build_tier_preview(rank_payload, {}, "fr") is None

    preview = build_tier_preview(rank_payload, {}, "fr", include_empty=True)

    assert preview is not None
    assert preview.rank == 12
    assert preview.total_rewards == 0
    assert preview.has_rewards is False


def test_select_tier_window_extends_forward_while_budget_allows() -> None:
    """La fenêtre doit inclure précédent/courant/suivant puis étendre vers l'avant."""

    def make_tier(rank: int, reward_count: int) -> BattlepassTierPreview:
        rewards = tuple(
            BattlepassRewardPreview(label=f"R{rank}-{index}") for index in range(reward_count)
        )
        return BattlepassTierPreview(rank=rank, free_rewards=rewards)

    tiers = (
        make_tier(0, 1),
        make_tier(1, 2),
        make_tier(2, 3),
        make_tier(3, 2),
        make_tier(4, 1),
        make_tier(5, 6),
    )

    window = _select_tier_window(tiers, focus_index=2)

    assert [tier.rank for tier in window] == [1, 2, 3, 4]


def test_load_or_fetch_track_metadata_uses_shared_metadata_cache(monkeypatch) -> None:
    """Le reward track ne doit pas être refetché si metadata.duckdb l'a déjà."""
    from src.ui.pages import home_mission_control_battlepass as battlepass

    monkeypatch.setattr(
        battlepass,
        "load_battlepass_track_definition",
        lambda track_path, _lang: StoredBattlepassTrackDefinition(
            reward_track_path=track_path,
            content_hash="hash-track",
            raw_payload_json=(
                '{"Name":{"value":"Echoes Within"},"Ranks":[{"Rank":0}],"XpPerRank":1000}'
            ),
        ),
    )

    async def _fail_fetch(*_args, **_kwargs):
        raise AssertionError("network fetch should not happen")

    monkeypatch.setattr(battlepass, "_fetch_progression_file", _fail_fetch)

    payload = asyncio.run(
        battlepass._load_or_fetch_track_metadata(
            session=object(),
            gamecms_host="https://gamecms-hacs.svc.halowaypoint.com",
            track_path="RewardTracks/Operations/S03BattlePass.json",
            lang="fr",
        )
    )

    assert payload["Name"]["value"] == "Echoes Within"
    assert payload["XpPerRank"] == 1000


def test_fetch_inventory_item_details_uses_shared_metadata_cache(monkeypatch) -> None:
    """Les items déjà présents en metadata ne doivent pas être refetchés."""
    from src.ui.pages import home_mission_control_battlepass as battlepass

    monkeypatch.setattr(
        battlepass,
        "load_battlepass_item_metadata_map",
        lambda item_paths, _lang: {
            item_paths[0]: StoredBattlepassItemDefinition(
                inventory_item_path=item_paths[0],
                content_hash="hash-item",
                quality="Rare",
                item_type="VehicleEmblem",
                display_path="Progression/Items/cerberus.png",
                title="Équipe Cerberus",
                description="Description locale",
            )
        },
    )

    async def _fake_reward_image(*_args, **_kwargs):
        return b"cached-image"

    async def _fail_item_fetch(*_args, **_kwargs):
        raise AssertionError("item fetch should not happen")

    monkeypatch.setattr(battlepass, "_fetch_reward_image_bytes", _fake_reward_image)
    monkeypatch.setattr(battlepass, "_fetch_inventory_item_detail", _fail_item_fetch)

    details = asyncio.run(
        battlepass._fetch_inventory_item_details(
            session=object(),
            gamecms_host="https://gamecms-hacs.svc.halowaypoint.com",
            inventory_paths=["inventory/vehicle.json"],
            lang="fr",
        )
    )

    assert details["inventory/vehicle.json"]["title"] == "Équipe Cerberus"
    assert details["inventory/vehicle.json"]["description"] == "Description locale"
    assert details["inventory/vehicle.json"]["quality"] == "Rare"
    assert details["inventory/vehicle.json"]["type"] == "VehicleEmblem"
    assert details["inventory/vehicle.json"]["image_bytes"] == b"cached-image"


def test_read_currency_asset_uses_repo_static_pngs() -> None:
    """Les monnaies connues doivent charger les PNG statiques embarqués."""
    xpboost = read_currency_asset("Currency/Currencies/xpboost.json")
    reroll = read_currency_asset("Currency/Currencies/rerollcurrency.json")

    assert xpboost is not None
    assert reroll is not None
    assert xpboost.startswith(b"\x89PNG")
    assert reroll.startswith(b"\x89PNG")
