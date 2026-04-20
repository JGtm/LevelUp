"""Collecte complète des battle pass Halo Infinite depuis les endpoints live.

Ce module sonde l'endpoint Economy des operations, persiste les définitions
GameCMS dans ``metadata.duckdb``, télécharge les assets associés dans le cache
local, puis écrit un snapshot JSON exploitable hors ligne.
"""

from __future__ import annotations

import asyncio
import hashlib
import json
import logging
import re
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from src.auth.provider import AuthRequiredError, get_halo_tokens_or_raise
from src.data.battlepass import (
    persist_battlepass_item_catalog,
    persist_battlepass_track_definition,
)
from src.data.battlepass_snapshots import persist_battlepass_snapshots
from src.ui.pages.home_mission_control_battlepass_assets import (
    read_cached_asset,
    read_currency_asset,
    write_cached_asset,
)
from src.ui.pages.home_mission_control_battlepass_cms import fetch_progression_file
from src.ui.pages.home_mission_control_battlepass_utils import (
    collect_inventory_item_paths,
    extract_reward_image_path,
)
from src.utils.paths import DATA_DIR

logger = logging.getLogger(__name__)

_ECONOMY_HOST = "https://economy.svc.halowaypoint.com"
_GAMECMS_HOST = "https://gamecms-hacs.svc.halowaypoint.com"
_DEFAULT_OUTPUT_DIR = DATA_DIR / "investigation" / "battlepass"
_FETCH_CONCURRENCY = 12
_REPO_STATIC_CURRENCY_IDS = frozenset({"xpboost", "rerollcurrency"})


@dataclass(frozen=True)
class BattlepassProbeResult:
    """Résumé d'une collecte complète des battle pass."""

    output_dir: Path
    operations_snapshot_path: Path
    manifest_path: Path
    operation_count: int
    snapshot_count: int
    track_count: int
    item_count: int
    track_asset_count: int
    item_asset_count: int
    currency_asset_count: int
    missing_track_assets: tuple[str, ...]
    missing_item_assets: tuple[str, ...]
    repo_missing_currency_assets: tuple[str, ...]
    external_currency_paths: tuple[str, ...]


async def probe_battlepass_catalog(
    player_db_path: str | Path,
    *,
    player_name: str,
    xuid: str,
    lang: str = "fr",
    output_dir: str | Path | None = None,
) -> BattlepassProbeResult:
    """Sonde le catalogue live des battle pass et persiste tous les détails.

    La collecte effectue quatre opérations :
    1. lecture de ``/rewardtracks/operations`` pour lister toutes les operations ;
    2. récupération des reward tracks GameCMS ;
    3. récupération des items référencés par les tracks ;
    4. téléchargement des visuels track/item/currency en cache local.
    """

    db_path = Path(player_db_path)
    target_dir = _resolve_output_dir(output_dir, player_name)
    target_dir.mkdir(parents=True, exist_ok=True)

    tokens = await _resolve_tokens(db_path, player_name)
    operations_payload = await _fetch_operations_payload(tokens, xuid)
    operations_path = target_dir / "operations_snapshot.json"
    _write_json(operations_path, operations_payload)
    snapshot_count = persist_battlepass_snapshots(db_path, xuid, operations_payload)

    track_paths = _extract_track_paths(operations_payload)
    track_payloads = await _fetch_payload_map(track_paths, tokens)
    track_dir = target_dir / "tracks"
    track_dir.mkdir(parents=True, exist_ok=True)
    for track_path, payload in track_payloads.items():
        persist_battlepass_track_definition(track_path, payload)
        _write_json(track_dir / _json_filename_for_path(track_path), payload)

    inventory_paths = _collect_inventory_paths(track_payloads.values())
    item_payloads = await _fetch_payload_map(inventory_paths, tokens)
    persist_battlepass_item_catalog(item_payloads)
    item_dir = target_dir / "items"
    item_dir.mkdir(parents=True, exist_ok=True)
    for item_path, payload in item_payloads.items():
        _write_json(item_dir / _json_filename_for_path(item_path), payload)

    track_assets, missing_track_assets = await _cache_image_paths(
        tokens,
        kind="tracks",
        image_paths=_collect_track_image_paths(track_payloads.values()),
    )
    item_assets, missing_item_assets = await _cache_image_paths(
        tokens,
        kind="rewards",
        image_paths=_collect_item_image_paths(item_payloads.values()),
    )
    currency_assets, repo_missing_currency_assets, external_currency_paths = _cache_currency_assets(
        _collect_currency_paths(track_payloads.values())
    )

    manifest = {
        "captured_at": datetime.now(timezone.utc).isoformat(),
        "player_name": player_name,
        "xuid": xuid,
        "lang": lang,
        "source": {
            "operations_endpoint": f"{_ECONOMY_HOST}/hi/players/xuid({xuid})/rewardtracks/operations",
            "gamecms_host": _GAMECMS_HOST,
        },
        "counts": {
            "operations": len(track_paths),
            "player_snapshots_inserted": snapshot_count,
            "tracks": len(track_payloads),
            "items": len(item_payloads),
            "track_assets": track_assets,
            "item_assets": item_assets,
            "currency_assets": currency_assets,
        },
        "paths": {
            "track_paths": track_paths,
            "inventory_item_paths": inventory_paths,
            "track_image_paths": _collect_track_image_paths(track_payloads.values()),
            "item_image_paths": _collect_item_image_paths(item_payloads.values()),
            "currency_paths": _collect_currency_paths(track_payloads.values()),
        },
        "missing_assets": {
            "tracks": missing_track_assets,
            "items": missing_item_assets,
            "repo_currencies": repo_missing_currency_assets,
        },
        "external_currency_paths": external_currency_paths,
    }
    manifest_path = target_dir / "manifest.json"
    _write_json(manifest_path, manifest)

    return BattlepassProbeResult(
        output_dir=target_dir,
        operations_snapshot_path=operations_path,
        manifest_path=manifest_path,
        operation_count=len(track_paths),
        snapshot_count=snapshot_count,
        track_count=len(track_payloads),
        item_count=len(item_payloads),
        track_asset_count=track_assets,
        item_asset_count=item_assets,
        currency_asset_count=currency_assets,
        missing_track_assets=tuple(missing_track_assets),
        missing_item_assets=tuple(missing_item_assets),
        repo_missing_currency_assets=tuple(repo_missing_currency_assets),
        external_currency_paths=tuple(external_currency_paths),
    )


async def _resolve_tokens(player_db_path: Path, player_name: str) -> Any:
    try:
        return await get_halo_tokens_or_raise(player_db_path)
    except AuthRequiredError:
        pass

    try:
        from src.data.sync._tokens import get_tokens_for_player

        tokens = await get_tokens_for_player(player_name)
    except Exception as exc:
        logger.debug("battlepass_probe: fallback auth échoué pour %s: %s", player_name, exc)
        tokens = None

    if tokens is None:
        raise AuthRequiredError(
            "Aucun token Halo disponible. Rechargez l'authentification du joueur puis relancez la collecte."
        )
    return tokens


async def _fetch_operations_payload(tokens: Any, xuid: str) -> dict[str, Any]:
    from aiohttp import ClientSession, ClientTimeout

    url = f"{_ECONOMY_HOST}/hi/players/xuid({xuid})/rewardtracks/operations"
    headers = {
        "Accept": "application/json",
        "x-343-authorization-spartan": tokens.spartan_token,
        "343-clearance": tokens.clearance_token,
    }
    async with ClientSession(timeout=ClientTimeout(total=30), headers=headers) as session:
        async with session.get(url) as resp:
            if resp.status != 200:
                raise RuntimeError(
                    f"battlepass operations indisponible ({resp.status}) pour {xuid}"
                )
            return await resp.json(content_type=None)


async def _fetch_payload_map(paths: list[str], tokens: Any) -> dict[str, dict[str, Any]]:
    if not paths:
        return {}

    from aiohttp import ClientSession, ClientTimeout

    headers = {
        "Accept": "application/json",
        "x-343-authorization-spartan": tokens.spartan_token,
        "343-clearance": tokens.clearance_token,
    }
    semaphore = asyncio.Semaphore(_FETCH_CONCURRENCY)

    async with ClientSession(timeout=ClientTimeout(total=30), headers=headers) as session:

        async def _fetch(path: str) -> tuple[str, dict[str, Any]] | None:
            async with semaphore:
                payload = await fetch_progression_file(session, _GAMECMS_HOST, path)
                if payload is None:
                    return None
                return path, payload

        results = await asyncio.gather(*[_fetch(path) for path in paths], return_exceptions=True)

    collected: dict[str, dict[str, Any]] = {}
    for result in results:
        if isinstance(result, tuple):
            path, payload = result
            collected[path] = payload
    return collected


async def _cache_image_paths(
    tokens: Any,
    *,
    kind: str,
    image_paths: list[str],
) -> tuple[int, list[str]]:
    if not image_paths:
        return 0, []

    from aiohttp import ClientSession, ClientTimeout

    headers = {
        "Accept": "image/*,application/octet-stream",
        "x-343-authorization-spartan": tokens.spartan_token,
        "343-clearance": tokens.clearance_token,
    }
    semaphore = asyncio.Semaphore(_FETCH_CONCURRENCY)

    async with ClientSession(timeout=ClientTimeout(total=30), headers=headers) as session:

        async def _cache(path: str) -> tuple[bool, str]:
            if read_cached_asset(kind, path) is not None:
                return True, path
            async with semaphore:
                url = f"{_GAMECMS_HOST}/hi/images/file/{path.lstrip('/')}"
                async with session.get(url) as resp:
                    if resp.status != 200:
                        return False, path
                    payload = await resp.read()
                write_cached_asset(kind, path, payload)
                return True, path

        results = await asyncio.gather(*[_cache(path) for path in image_paths])

    cached_count = sum(1 for ok, _ in results if ok)
    missing = [path for ok, path in results if not ok]
    return cached_count, missing


def _cache_currency_assets(currency_paths: list[str]) -> tuple[int, list[str], list[str]]:
    if not currency_paths:
        return 0, [], []
    cached_count = 0
    missing: list[str] = []
    external: list[str] = []
    for currency_path in currency_paths:
        currency_id = Path(currency_path).stem.lower()
        if currency_id not in _REPO_STATIC_CURRENCY_IDS:
            external.append(currency_path)
            continue
        if read_currency_asset(currency_path) is None:
            missing.append(currency_path)
            continue
        cached_count += 1
    return cached_count, missing, external


def _extract_track_paths(operations_payload: dict[str, Any]) -> list[str]:
    tracks = operations_payload.get("OperationRewardTracks") or []
    seen: dict[str, None] = {}
    for entry in tracks:
        if not isinstance(entry, dict):
            continue
        path = entry.get("RewardTrackPath")
        if isinstance(path, str) and path.strip():
            seen.setdefault(path.strip(), None)
    active_path = operations_payload.get("ActiveOperationRewardTrackPath")
    if isinstance(active_path, str) and active_path.strip():
        seen.setdefault(active_path.strip(), None)
    return list(seen)


def _collect_inventory_paths(track_payloads: list[dict[str, Any]]) -> list[str]:
    ranks: list[dict[str, Any] | None] = []
    for payload in track_payloads:
        ranks.extend(payload.get("Ranks") or [])
    return collect_inventory_item_paths(ranks)


def _collect_track_image_paths(track_payloads: list[dict[str, Any]]) -> list[str]:
    seen: dict[str, None] = {}
    for payload in track_payloads:
        for candidate in (
            payload.get("BattlePassImage"),
            payload.get("BackgroundImagePath"),
            payload.get("SummaryImagePath"),
        ):
            if isinstance(candidate, str) and candidate.strip():
                seen.setdefault(candidate.strip(), None)
    return list(seen)


def _collect_item_image_paths(item_payloads: list[dict[str, Any]]) -> list[str]:
    seen: dict[str, None] = {}
    for payload in item_payloads:
        common = payload.get("CommonData") or {}
        image_path = extract_reward_image_path(payload, common)
        if image_path:
            seen.setdefault(image_path, None)
    return list(seen)


def _collect_currency_paths(track_payloads: list[dict[str, Any]]) -> list[str]:
    seen: dict[str, None] = {}
    for payload in track_payloads:
        for rank in payload.get("Ranks") or []:
            if not isinstance(rank, dict):
                continue
            for bucket_name in ("FreeRewards", "PaidRewards"):
                bucket = rank.get(bucket_name) or {}
                for currency in bucket.get("CurrencyRewards") or []:
                    path = currency.get("CurrencyPath")
                    if isinstance(path, str) and path.strip():
                        seen.setdefault(path.strip(), None)
    return list(seen)


def _resolve_output_dir(output_dir: str | Path | None, player_name: str) -> Path:
    if output_dir is not None:
        return Path(output_dir)
    return _DEFAULT_OUTPUT_DIR / _safe_name(player_name)


def _json_filename_for_path(path: str) -> str:
    stem = Path(path).stem or "entry"
    digest = hashlib.sha1(path.encode("utf-8")).hexdigest()[:10]
    return f"{_safe_name(stem)}-{digest}.json"


def _safe_name(value: str) -> str:
    cleaned = re.sub(r"[^A-Za-z0-9._-]+", "_", value.strip())
    return cleaned.strip("._") or "entry"


def _write_json(path: Path, payload: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(payload, indent=2, ensure_ascii=False), encoding="utf-8")
