"""Cache lazy des assets battle pass de la home V7."""

from __future__ import annotations

import hashlib
import logging
from pathlib import Path

from src.data.battlepass_asset_refs import (
    load_battlepass_asset_ref,
    persist_battlepass_asset_ref,
    touch_battlepass_asset_ref,
)
from src.utils.paths import DATA_DIR, REPO_ROOT

logger = logging.getLogger(__name__)

_ASSET_CACHE_DIR = DATA_DIR / "cache" / "battlepass_assets"
_STATIC_BATTLEPASS_ASSETS_DIR = REPO_ROOT / "static" / "battlepass-assets"
_STATIC_CURRENCY_ASSETS: dict[str, str] = {
    "xpboost": "xpboost.png",
    "rerollcurrency": "rerollcurrency.png",
}


def read_cached_asset(kind: str, source_path: str) -> bytes | None:
    """Retourne un asset battle pass depuis le cache disque si disponible."""
    stored = load_battlepass_asset_ref(kind, source_path)
    cache_path = stored.cache_path if stored is not None else _asset_cache_path(kind, source_path)
    try:
        if not cache_path.exists():
            return None
        touch_battlepass_asset_ref(kind, source_path)
        return cache_path.read_bytes()
    except OSError:
        return None


def write_cached_asset(kind: str, source_path: str, payload: bytes) -> None:
    """Persiste un asset battle pass dans le cache disque."""
    cache_path = _asset_cache_path(kind, source_path)
    try:
        cache_path.parent.mkdir(parents=True, exist_ok=True)
        cache_path.write_bytes(payload)
        persist_battlepass_asset_ref(
            kind,
            source_path,
            cache_path,
            mime_type="image/png",
            image_source_path=source_path,
            source_origin="cms",
        )
    except OSError:
        logger.debug("battlepass: impossible d'écrire le cache asset %s", source_path)


def read_currency_asset(currency_path: str | None) -> bytes | None:
    """Retourne un visuel statique de monnaie battle pass si disponible."""
    if not currency_path:
        return None

    cached = read_cached_asset("currency", currency_path)
    if cached is not None:
        return cached

    static_path = _static_currency_asset_path(currency_path)
    if static_path is None:
        return None
    try:
        payload = static_path.read_bytes()
    except OSError:
        return None

    persist_battlepass_asset_ref(
        "currency",
        currency_path,
        static_path,
        mime_type="image/png",
        image_source_path=static_path.relative_to(REPO_ROOT).as_posix(),
        source_origin="repo-static",
    )
    return payload


def _asset_cache_path(kind: str, source_path: str) -> Path:
    suffix = Path(source_path).suffix or ".bin"
    digest = hashlib.sha1(source_path.encode("utf-8")).hexdigest()[:24]
    return _ASSET_CACHE_DIR / kind / f"{digest}{suffix}"


def _static_currency_asset_path(currency_path: str) -> Path | None:
    currency_id = Path(currency_path).stem.lower()
    filename = _STATIC_CURRENCY_ASSETS.get(currency_id)
    if filename is None:
        return None
    path = _STATIC_BATTLEPASS_ASSETS_DIR / filename
    return path if path.exists() else None
