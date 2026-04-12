"""Utilitaires de formatage et d'extraction pour le battle pass home V7."""

from __future__ import annotations

from typing import Any

_CURRENCY_LABELS: dict[str, dict[str, str]] = {
    "xpboost": {"fr": "Boost d'XP", "en": "XP Boost"},
    "rerollcurrency": {"fr": "Échange de défi", "en": "Challenge Swap"},
}
_CURRENCY_TILE_LABELS: dict[str, dict[str, str]] = {
    "xpboost": {"fr": "XP", "en": "XP"},
    "rerollcurrency": {"fr": "Défi", "en": "Swap"},
}
_QUALITY_LABELS: dict[str, dict[str, str]] = {
    "common": {"fr": "Commun", "en": "Common"},
    "rare": {"fr": "Rare", "en": "Rare"},
    "epic": {"fr": "Épique", "en": "Epic"},
    "legendary": {"fr": "Légendaire", "en": "Legendary"},
}


def coerce_int(value: Any) -> int | None:
    """Convertit une valeur en entier si possible."""
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


def path_basename(path: str) -> str:
    """Retourne le nom de base d'un chemin CMS sans extension json."""
    return path.split("/")[-1].replace(".json", "")


def append_amount(label: str, amount: Any) -> str:
    """Ajoute un multiplicateur si la récompense est donnée en plusieurs exemplaires."""
    amount_value = coerce_int(amount)
    if amount_value is None or amount_value <= 1:
        return label
    return f"{label} x{amount_value}"


def build_currency_reward_label(currency_path: str | None, amount: Any, lang: str) -> str | None:
    """Construit le libellé d'une reward currency battle pass."""
    currency_id = path_basename(currency_path or "").lower()
    if not currency_id:
        return None
    label = _CURRENCY_LABELS.get(currency_id, {}).get(lang) or currency_id.replace("_", " ")
    return append_amount(label, amount)


def format_inventory_item_label(
    *,
    title: str | None,
    quality: str | None,
    reward_type: str | None,
    lang: str,
) -> str | None:
    """Construit un libellé compact pour une reward d'inventaire."""
    quality_label = _localize_quality(quality, lang)
    base_label = title or reward_type
    if quality_label and base_label:
        return f"{quality_label} · {base_label}"
    return base_label or quality_label


def build_reward_tile_label(title: str | None, reward_type: str | None) -> str | None:
    """Construit un monogramme court pour une tuile reward."""
    text = title or reward_type
    if not text:
        return None
    tokens = [chunk for chunk in str(text).replace("/", " ").split() if chunk]
    if not tokens:
        return None
    if len(tokens) == 1:
        return tokens[0][:4].upper()
    return "".join(token[0] for token in tokens[:2]).upper()


def build_currency_tile_label(currency_path: str | None, lang: str) -> str:
    """Construit le label court d'une monnaie battle pass."""
    currency_id = path_basename(currency_path or "").lower()
    return _CURRENCY_TILE_LABELS.get(currency_id, {}).get(lang) or currency_id[:4].upper()


def collect_inventory_item_paths(rank_payloads: list[dict[str, Any] | None]) -> list[str]:
    """Collecte les chemins d'items d'inventaire présents dans un reward track."""
    seen: dict[str, None] = {}
    for rank_payload in rank_payloads:
        if not isinstance(rank_payload, dict):
            continue
        for bucket_name in ("FreeRewards", "PaidRewards"):
            bucket = rank_payload.get(bucket_name) or {}
            for item in bucket.get("InventoryRewards") or []:
                path = item.get("InventoryItemPath")
                if isinstance(path, str) and path:
                    seen.setdefault(path, None)
    return list(seen)


def extract_reward_image_path(payload: dict[str, Any], common: dict[str, Any]) -> str | None:
    """Résout le chemin d'image principal d'un payload item battle pass."""
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
    """Extrait un chemin image depuis un noeud GameCMS."""
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


def extract_quality(common: dict[str, Any]) -> str | None:
    """Extrait la qualité d'un item GameCMS."""
    quality = common.get("Quality") or common.get("Rarity")
    return str(quality).strip() if isinstance(quality, str) and quality.strip() else None


def extract_item_type(common: dict[str, Any]) -> str | None:
    """Extrait le type d'un item GameCMS."""
    item_type = common.get("Type")
    return str(item_type).strip() if isinstance(item_type, str) and item_type.strip() else None


def rank_has_rewards(rank_payload: dict[str, Any]) -> bool:
    """Indique si un palier contient au moins une reward utile."""
    for bucket_name in ("FreeRewards", "PaidRewards"):
        bucket = rank_payload.get(bucket_name) or {}
        if (bucket.get("InventoryRewards") or []) or (bucket.get("CurrencyRewards") or []):
            return True
    return False


def _localize_quality(quality: str | None, lang: str) -> str | None:
    if not quality:
        return None
    return _QUALITY_LABELS.get(str(quality).strip().lower(), {}).get(lang) or str(quality)
