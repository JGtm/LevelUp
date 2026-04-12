"""Utilitaires internes pour le catalogue metadata battle pass."""

from __future__ import annotations

import hashlib
import json
from typing import Any

from src.data.challenges import normalize_challenge_lang
from src.data.sync._asset_langs import DEFAULT_LANG


def extract_localized_texts(data: Any) -> dict[str, str]:
    """Extrait les traductions localisées d'un payload GameCMS."""
    if isinstance(data, str):
        text = data.strip()
        return {DEFAULT_LANG: text} if text else {}
    if not isinstance(data, dict):
        return {}
    results: dict[str, str] = {}
    translations = data.get("translations") or {}
    if isinstance(translations, dict):
        for raw_lang, raw_value in translations.items():
            if not isinstance(raw_value, str) or not raw_value.strip():
                continue
            results[normalize_challenge_lang(str(raw_lang))] = raw_value.strip()
    fallback = data.get("value")
    if isinstance(fallback, str) and fallback.strip():
        results.setdefault(DEFAULT_LANG, fallback.strip())
    return results


def extract_display_path(payload: dict[str, Any], common: dict[str, Any]) -> str | None:
    """Résout le chemin d'image à afficher pour un item battle pass."""
    for candidate in (
        common.get("DisplayPath"),
        payload.get("DisplayPath"),
        payload.get("ImagePath"),
    ):
        path = extract_media_path(candidate)
        if path:
            return path
    return None


def extract_media_path(node: Any) -> str | None:
    """Extrait un chemin média GameCMS depuis plusieurs formes de payload."""
    if not isinstance(node, dict):
        return None
    media = node.get("Media") or {}
    media_url = media.get("MediaUrl") or {}
    path = media_url.get("Path")
    if isinstance(path, str) and path.strip():
        return path.strip()
    folder = node.get("FolderPath")
    filename = node.get("FileName")
    if isinstance(folder, str) and folder and isinstance(filename, str) and filename:
        return f"{folder.rstrip('/')}/{filename.lstrip('/')}"
    return None


def build_content_hash(payload: dict[str, Any]) -> str:
    """Construit un hash stable à partir d'un payload GameCMS."""
    canonical = json.dumps(payload, sort_keys=True, separators=(",", ":"), ensure_ascii=True)
    return hashlib.sha1(canonical.encode("utf-8")).hexdigest()


def dump_payload(payload: dict[str, Any]) -> str:
    """Sérialise un payload GameCMS de manière canonique."""
    return json.dumps(payload, sort_keys=True, separators=(",", ":"), ensure_ascii=True)


def loads_payload(raw_payload_json: str | None) -> dict[str, Any]:
    """Recharge un payload sérialisé depuis DuckDB."""
    if not raw_payload_json:
        return {}
    try:
        payload = json.loads(raw_payload_json)
    except (TypeError, ValueError):
        return {}
    return payload if isinstance(payload, dict) else {}


def coerce_int(value: Any) -> int | None:
    """Convertit une valeur CMS en entier si possible."""
    if isinstance(value, bool):
        return int(value)
    if isinstance(value, int):
        return value
    if isinstance(value, float) and value.is_integer():
        return int(value)
    try:
        text = str(value).strip()
    except Exception:
        return None
    if not text:
        return None
    try:
        return int(text)
    except ValueError:
        return None


def coerce_str(value: Any) -> str | None:
    """Convertit une valeur CMS en chaîne trimée si possible."""
    if not isinstance(value, str):
        return None
    text = value.strip()
    return text or None
