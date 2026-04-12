"""Helpers de pass de combat live pour Mission Control V7."""

from __future__ import annotations

import asyncio
import logging
from dataclasses import dataclass
from typing import Any

from src.data.battlepass import (
    StoredBattlepassItemDefinition,
    load_battlepass_item_metadata_map,
    load_battlepass_track_definition,
    persist_battlepass_item_catalog,
    persist_battlepass_track_definition,
)
from src.data.challenges import resolve_localized_value
from src.ui.pages.home_mission_control_battlepass_assets import (
    read_cached_asset,
    read_currency_asset,
    write_cached_asset,
)
from src.ui.pages.home_mission_control_battlepass_cms import (
    fetch_operation_artwork as _fetch_operation_artwork,
)
from src.ui.pages.home_mission_control_battlepass_cms import (
    fetch_progression_file as _fetch_progression_file,
)
from src.ui.pages.home_mission_control_battlepass_utils import (
    append_amount as _append_amount,
)
from src.ui.pages.home_mission_control_battlepass_utils import (
    build_currency_reward_label as _build_currency_reward_label,
)
from src.ui.pages.home_mission_control_battlepass_utils import (
    build_currency_tile_label as _build_currency_tile_label,
)
from src.ui.pages.home_mission_control_battlepass_utils import (
    build_reward_tile_label as _build_reward_tile_label,
)
from src.ui.pages.home_mission_control_battlepass_utils import (
    coerce_int as _coerce_int,
)
from src.ui.pages.home_mission_control_battlepass_utils import (
    collect_inventory_item_paths as _collect_inventory_item_paths,
)
from src.ui.pages.home_mission_control_battlepass_utils import (
    extract_item_type as _extract_item_type,
)
from src.ui.pages.home_mission_control_battlepass_utils import (
    extract_quality as _extract_quality,
)
from src.ui.pages.home_mission_control_battlepass_utils import (
    extract_reward_image_path as _extract_reward_image_path,
)
from src.ui.pages.home_mission_control_battlepass_utils import (
    format_inventory_item_label as _format_inventory_item_label,
)
from src.ui.pages.home_mission_control_battlepass_utils import (
    path_basename as _path_basename,
)
from src.ui.pages.home_mission_control_battlepass_utils import (
    rank_has_rewards as _rank_has_rewards,
)

logger = logging.getLogger(__name__)

_ECONOMY_HOST = "https://economy.svc.halowaypoint.com"
_GAMECMS_HOST = "https://gamecms-hacs.svc.halowaypoint.com"
_RECENT_UNLOCK_LIMIT = 3
_DETAIL_FETCH_CONCURRENCY = 12


@dataclass(frozen=True)
class BattlepassRewardPreview:
    """Résumé compact d'une récompense de pass."""

    label: str
    description: str | None = None
    image_bytes: bytes | None = None
    tile_label: str | None = None


@dataclass(frozen=True)
class BattlepassTierPreview:
    """Résumé compact d'un palier du pass."""

    rank: int
    free_rewards: tuple[BattlepassRewardPreview, ...] = ()
    premium_rewards: tuple[BattlepassRewardPreview, ...] = ()

    @property
    def total_rewards(self) -> int:
        """Retourne le volume total de récompenses du palier."""
        return len(self.free_rewards) + len(self.premium_rewards)

    @property
    def has_rewards(self) -> bool:
        """Indique si le palier contient au moins une récompense."""
        return self.total_rewards > 0


@dataclass(frozen=True)
class HomeBattlepassInfo:
    """Info d'opération active pour l'accueil."""

    track_name: str
    op_rank: int = 0
    is_owned: bool = False
    career_rank: int = 0
    career_rank_label: str | None = None
    track_image_bytes: bytes | None = None
    tiers: tuple[BattlepassTierPreview, ...] = ()
    recent_unlocks: tuple[BattlepassTierPreview, ...] = ()
    next_unlock: BattlepassTierPreview | None = None
    partial_progress: int | None = None
    xp_per_rank: int | None = None
    has_reached_max_rank: bool = False


async def fetch_battlepass_info(
    session: Any,
    xuid_clean: str,
    lang: str,
    career_rank: int,
) -> HomeBattlepassInfo | None:
    """Récupère le pass actif sélectionné par le joueur et ses paliers clés."""
    ops_url = f"{_ECONOMY_HOST}/hi/players/xuid({xuid_clean})/rewardtracks/operations"
    async with session.get(ops_url) as resp:
        if resp.status != 200:
            logger.debug("battlepass: rewardtracks/operations indisponible (%s)", resp.status)
            return None
        ops_data = await resp.json(content_type=None)

    active_path = ops_data.get("ActiveOperationRewardTrackPath")
    if not active_path:
        return None

    active_op = next(
        (
            op
            for op in ops_data.get("OperationRewardTracks", [])
            if op.get("RewardTrackPath") == active_path
        ),
        {},
    )
    progress = active_op.get("CurrentProgress") or {}
    current_rank = _coerce_int(progress.get("Rank")) or 0

    track_meta = await _load_or_fetch_track_metadata(session, _GAMECMS_HOST, active_path, lang)
    track_name = resolve_localized_value((track_meta or {}).get("Name"), lang) or _path_basename(
        active_path
    )
    if track_meta is None:
        return HomeBattlepassInfo(
            track_name=track_name,
            op_rank=current_rank,
            is_owned=bool(active_op.get("IsOwned") or progress.get("IsOwned")),
            career_rank=career_rank,
            partial_progress=_coerce_int(progress.get("PartialProgress")),
            has_reached_max_rank=bool(progress.get("HasReachedMaxRank")),
        )

    ranks = sorted(
        track_meta.get("Ranks") or [],
        key=lambda rank: _coerce_int(rank.get("Rank")) or 0,
    )
    selected_ranks = _select_recent_reward_ranks(ranks, current_rank, limit=_RECENT_UNLOCK_LIMIT)
    next_rank = _select_next_reward_rank(ranks, current_rank)
    inventory_paths = _collect_inventory_item_paths(ranks)
    item_details = await _fetch_inventory_item_details(
        session,
        _GAMECMS_HOST,
        inventory_paths,
        lang,
    )
    tiers = _build_all_tier_previews(ranks, item_details, lang)
    recent_unlocks = _build_tier_previews(selected_ranks, item_details, lang)
    next_unlock = _build_tier_preview(next_rank, item_details, lang) if next_rank else None

    return HomeBattlepassInfo(
        track_name=track_name,
        op_rank=current_rank,
        is_owned=bool(active_op.get("IsOwned") or progress.get("IsOwned")),
        career_rank=career_rank,
        track_image_bytes=await _fetch_operation_artwork(session, _GAMECMS_HOST, track_meta),
        tiers=tiers,
        recent_unlocks=recent_unlocks,
        next_unlock=next_unlock,
        partial_progress=_coerce_int(progress.get("PartialProgress")),
        xp_per_rank=_coerce_int(track_meta.get("XpPerRank")),
        has_reached_max_rank=bool(progress.get("HasReachedMaxRank")),
    )


def select_recent_reward_ranks(
    ranks: list[dict[str, Any]],
    current_rank: int,
    *,
    limit: int = _RECENT_UNLOCK_LIMIT,
) -> list[dict[str, Any]]:
    """Expose la sélection des derniers paliers débloqués utiles."""
    return _select_recent_reward_ranks(ranks, current_rank, limit=limit)


def build_tier_preview(
    rank_payload: dict[str, Any],
    item_details: dict[str, dict[str, str | None]],
    lang: str,
    *,
    include_empty: bool = False,
) -> BattlepassTierPreview | None:
    """Expose la construction d'un résumé de palier pour les tests."""
    return _build_tier_preview(rank_payload, item_details, lang, allow_empty=include_empty)


def build_currency_reward_label(currency_path: str | None, amount: Any, lang: str) -> str | None:
    """Expose le libellé d'une récompense monnaie pour les tests."""
    return _build_currency_reward_label(currency_path, amount, lang)


async def _load_or_fetch_track_metadata(
    session: Any,
    gamecms_host: str,
    track_path: str,
    lang: str,
) -> dict[str, Any] | None:
    stored = load_battlepass_track_definition(track_path, lang)
    if stored is not None:
        return stored.payload
    payload = await _fetch_progression_file(session, gamecms_host, track_path)
    if payload is not None:
        persist_battlepass_track_definition(track_path, payload)
    return payload


def _select_recent_reward_ranks(
    ranks: list[dict[str, Any]],
    current_rank: int,
    *,
    limit: int,
) -> list[dict[str, Any]]:
    selected = [
        rank
        for rank in ranks
        if (_coerce_int(rank.get("Rank")) or 0) <= current_rank and _rank_has_rewards(rank)
    ]
    selected.sort(key=lambda rank: _coerce_int(rank.get("Rank")) or 0)
    return selected[-limit:]


def _select_next_reward_rank(
    ranks: list[dict[str, Any]],
    current_rank: int,
) -> dict[str, Any] | None:
    for rank in ranks:
        rank_value = _coerce_int(rank.get("Rank")) or 0
        if rank_value > current_rank and _rank_has_rewards(rank):
            return rank
    return None


def _build_tier_previews(
    rank_payloads: list[dict[str, Any]],
    item_details: dict[str, dict[str, str | None]],
    lang: str,
) -> tuple[BattlepassTierPreview, ...]:
    previews: list[BattlepassTierPreview] = []
    for rank_payload in rank_payloads:
        preview = _build_tier_preview(rank_payload, item_details, lang)
        if preview is not None:
            previews.append(preview)
    return tuple(previews)


def _build_all_tier_previews(
    rank_payloads: list[dict[str, Any]],
    item_details: dict[str, dict[str, Any]],
    lang: str,
) -> tuple[BattlepassTierPreview, ...]:
    previews: list[BattlepassTierPreview] = []
    for rank_payload in rank_payloads:
        preview = _build_tier_preview(rank_payload, item_details, lang, allow_empty=True)
        if preview is not None:
            previews.append(preview)
    return tuple(previews)


def _build_tier_preview(
    rank_payload: dict[str, Any],
    item_details: dict[str, dict[str, Any]],
    lang: str,
    *,
    allow_empty: bool = False,
) -> BattlepassTierPreview | None:
    free_rewards = _build_bucket_labels(rank_payload.get("FreeRewards"), item_details, lang)
    premium_rewards = _build_bucket_labels(rank_payload.get("PaidRewards"), item_details, lang)
    if not allow_empty and not free_rewards and not premium_rewards:
        return None
    return BattlepassTierPreview(
        rank=_coerce_int(rank_payload.get("Rank")) or 0,
        free_rewards=tuple(free_rewards),
        premium_rewards=tuple(premium_rewards),
    )


def _build_bucket_labels(
    reward_bucket: dict[str, Any] | None,
    item_details: dict[str, dict[str, Any]],
    lang: str,
) -> list[BattlepassRewardPreview]:
    if not isinstance(reward_bucket, dict):
        return []

    labels: list[BattlepassRewardPreview] = []
    for item in reward_bucket.get("InventoryRewards") or []:
        path = item.get("InventoryItemPath")
        detail = item_details.get(str(path or ""), {})
        label = _format_inventory_item_label(
            title=detail.get("title"),
            quality=detail.get("quality"),
            reward_type=detail.get("type"),
            lang=lang,
        )
        if label:
            labels.append(
                BattlepassRewardPreview(
                    label=_append_amount(label, item.get("Amount")),
                    description=detail.get("description"),
                    image_bytes=detail.get("image_bytes"),
                    tile_label=_build_reward_tile_label(detail.get("title"), detail.get("type")),
                )
            )

    for currency in reward_bucket.get("CurrencyRewards") or []:
        currency_path = currency.get("CurrencyPath")
        label = _build_currency_reward_label(
            currency_path,
            currency.get("Amount"),
            lang,
        )
        if label:
            labels.append(
                BattlepassRewardPreview(
                    label=label,
                    image_bytes=read_currency_asset(str(currency_path) if currency_path else None),
                    tile_label=_build_currency_tile_label(currency_path, lang),
                )
            )
    return labels


async def _fetch_inventory_item_details(
    session: Any,
    gamecms_host: str,
    inventory_paths: list[str],
    lang: str,
) -> dict[str, dict[str, Any]]:
    if not inventory_paths:
        return {}
    cached_metadata = load_battlepass_item_metadata_map(inventory_paths, lang)
    details = await _build_cached_inventory_item_details(session, gamecms_host, cached_metadata)
    missing_paths = [path for path in inventory_paths if path not in cached_metadata]
    if not missing_paths:
        return details

    fetched_details, fetched_payloads = await _fetch_missing_inventory_item_details(
        session,
        gamecms_host,
        missing_paths,
        lang,
    )
    if fetched_payloads:
        persist_battlepass_item_catalog(fetched_payloads)
    details.update(fetched_details)
    return details


async def _build_cached_inventory_item_details(
    session: Any,
    gamecms_host: str,
    cached_metadata: dict[str, StoredBattlepassItemDefinition],
) -> dict[str, dict[str, Any]]:
    if not cached_metadata:
        return {}
    results = await asyncio.gather(
        *[
            _build_cached_inventory_item_detail(session, gamecms_host, item_path, stored)
            for item_path, stored in cached_metadata.items()
        ],
        return_exceptions=True,
    )
    details: dict[str, dict[str, Any]] = {}
    for result in results:
        if isinstance(result, tuple):
            path, detail = result
            details[path] = detail
    return details


async def _build_cached_inventory_item_detail(
    session: Any,
    gamecms_host: str,
    item_path: str,
    stored: StoredBattlepassItemDefinition,
) -> tuple[str, dict[str, Any]]:
    return item_path, {
        "title": stored.title,
        "description": stored.description,
        "quality": stored.quality,
        "type": stored.item_type,
        "image_bytes": await _fetch_reward_image_bytes(session, gamecms_host, stored.display_path),
    }


async def _fetch_missing_inventory_item_details(
    session: Any,
    gamecms_host: str,
    inventory_paths: list[str],
    lang: str,
) -> tuple[dict[str, dict[str, Any]], dict[str, dict[str, Any]]]:
    semaphore = asyncio.Semaphore(_DETAIL_FETCH_CONCURRENCY)

    async def _fetch_with_limit(path: str) -> tuple[str, dict[str, Any], dict[str, Any]] | None:
        async with semaphore:
            return await _fetch_inventory_item_detail(session, gamecms_host, path, lang)

    results = await asyncio.gather(
        *[_fetch_with_limit(path) for path in inventory_paths],
        return_exceptions=True,
    )
    details: dict[str, dict[str, Any]] = {}
    payloads: dict[str, dict[str, Any]] = {}
    for result in results:
        if isinstance(result, tuple):
            path, detail, payload = result
            details[path] = detail
            payloads[path] = payload
    return details, payloads


async def _fetch_inventory_item_detail(
    session: Any,
    gamecms_host: str,
    inventory_path: str,
    lang: str,
) -> tuple[str, dict[str, Any], dict[str, Any]] | None:
    payload = await _fetch_progression_file(session, gamecms_host, inventory_path)
    if payload is None:
        return None
    detail = await _build_inventory_item_detail(session, gamecms_host, payload, lang)
    return inventory_path, detail, payload


async def _build_inventory_item_detail(
    session: Any,
    gamecms_host: str,
    payload: dict[str, Any],
    lang: str,
) -> dict[str, Any]:
    common = payload.get("CommonData") or {}
    image_path = _extract_reward_image_path(payload, common)
    return {
        "title": resolve_localized_value(common.get("Title"), lang),
        "description": resolve_localized_value(common.get("Description"), lang),
        "quality": _extract_quality(common),
        "type": _extract_item_type(common),
        "image_bytes": await _fetch_reward_image_bytes(session, gamecms_host, image_path),
    }


async def _fetch_reward_image_bytes(
    session: Any,
    gamecms_host: str,
    image_path: str | None,
) -> bytes | None:
    if not image_path:
        return None
    cached = read_cached_asset("rewards", image_path)
    if cached is not None:
        return cached
    image_url = f"{gamecms_host}/hi/images/file/{image_path.lstrip('/')}"
    async with session.get(image_url) as resp:
        if resp.status != 200:
            return None
        payload = await resp.read()
    write_cached_asset("rewards", image_path, payload)
    return payload
