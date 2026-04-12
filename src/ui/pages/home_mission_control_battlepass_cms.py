"""Appels GameCMS mutualisés pour le battle pass home V7."""

from __future__ import annotations

import logging
from typing import Any

from src.ui.pages.home_mission_control_battlepass_assets import (
    read_cached_asset,
    write_cached_asset,
)

logger = logging.getLogger(__name__)


async def fetch_progression_file(
    session: Any,
    gamecms_host: str,
    path: str,
) -> dict[str, Any] | None:
    """Charge un fichier de progression GameCMS."""
    url = f"{gamecms_host}/hi/Progression/file/{path.lstrip('/')}"
    async with session.get(url) as resp:
        if resp.status != 200:
            logger.debug("battlepass: progression file indisponible %s (%s)", path, resp.status)
            return None
        return await resp.json(content_type=None)


async def fetch_operation_artwork(
    session: Any,
    gamecms_host: str,
    track_meta: dict[str, Any],
) -> bytes | None:
    """Charge l'illustration principale d'un reward track battle pass."""
    img_path = track_meta.get("BattlePassImage") or track_meta.get("BackgroundImagePath")
    if not img_path:
        return None
    cached = read_cached_asset("tracks", str(img_path))
    if cached is not None:
        return cached
    img_url = f"{gamecms_host}/hi/images/file/{str(img_path).lstrip('/')}"
    async with session.get(img_url) as resp:
        if resp.status != 200:
            return None
        payload = await resp.read()
    write_cached_asset("tracks", str(img_path), payload)
    return payload
