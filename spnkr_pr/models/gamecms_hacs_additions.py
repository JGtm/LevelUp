"""Additions to spnkr/models/gamecms_hacs.py — inventory item metadata.

InventoryItem is the common wrapper returned by:
  GET gamecms-hacs.svc.halowaypoint.com/hi/Waypoint/file/{inventory_item_path}

The outer structure is consistent across item types (armor, weapon, vehicle…).
The inner ``customization_data`` varies by item type and is left as a raw dict
to avoid premature modelling of fields we haven't fully observed yet.
"""

from __future__ import annotations

from typing import Any

from pydantic import Field
from spnkr.models.base import PascalCaseModel


class ItemDisplayPath(PascalCaseModel, frozen=True):
    """Path information for the item's display/preview image.

    Attributes:
        media_url: Relative path passable to ``GameCmsHacsService.get_image()``.
    """

    media_url: str = Field(alias="MediaUrl")


class ItemDisplayMedia(PascalCaseModel, frozen=True):
    """Wrapper around the display path of an inventory item."""

    display_path: ItemDisplayPath = Field(alias="DisplayPath")


class ItemCommonData(PascalCaseModel, frozen=True):
    """Fields present on every inventory item regardless of type.

    Attributes:
        id: Unique item identifier.
        item_type: Item category string, e.g. "ArmorCore", "ArmorCoating",
            "WeaponCoating", "VehicleCore", …
        display_path: Contains the relative path to the item's preview image.
        title: Human-readable item name.
        description: Human-readable item description (may be empty).
        quality: Rarity tier, e.g. "Common", "Rare", "Epic", "Legendary".
        season_unicode_icon: Season icon path (None for non-seasonal items).
    """

    id: str = Field(alias="Id")
    item_type: str = Field(alias="ItemType")
    display_path: ItemDisplayMedia = Field(alias="DisplayPath")
    title: str = Field(alias="Title")
    description: str = Field(alias="Description", default="")
    quality: str = Field(alias="Quality", default="")
    season_unicode_icon: str | None = Field(alias="SeasonUnicodeIcon", default=None)


class InventoryItem(PascalCaseModel, frozen=True):
    """Metadata for any inventory item fetched from gamecms_hacs.

    Only ``common_data`` is modelled; the rest of the payload varies too much
    per item type to model safely without a full corpus of responses.

    Usage example::

        item_resp = await client.gamecms_hacs.get_item_metadata(
            "Inventory/Armor/Cores/013-001-olympus-c13d0f2f.json"
        )
        item = await item_resp.parse()
        image_path = item.common_data.display_path.display_path.media_url
        image_resp = await client.gamecms_hacs.get_image(image_path)
        png_bytes = await image_resp.read()

    Attributes:
        common_data: Fields shared across all item types (id, title, image path…).
        extra: Remaining fields from the response, preserved as a raw dict.
    """

    common_data: ItemCommonData = Field(alias="CommonData")
    extra: dict[str, Any] = Field(default_factory=dict, alias="Extra", exclude=True)

    model_config = {  # type: ignore[assignment]
        "alias_generator": None,  # fields use explicit aliases above
        "populate_by_name": True,
        "extra": "allow",
    }
