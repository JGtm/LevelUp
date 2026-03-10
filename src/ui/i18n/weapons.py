"""Traductions des armes Halo Infinite (weapon_id → nom localisé).

Les données sont stockées dans ``static/i18n/weapons_{lang}.json``.
Chaque entrée mappe un ``weapon_id`` (entier sérialisé en clé string)
vers ``{"name": "...", "faction": "..."}``.
"""

from __future__ import annotations

import json
import logging
from functools import lru_cache
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)

_I18N_DIR = Path(__file__).resolve().parent.parent.parent.parent / "static" / "i18n"


@lru_cache(maxsize=4)
def _load_weapons_json(lang: str = "fr") -> dict[str, dict[str, str]]:
    """Charge et met en cache le fichier ``weapons_{lang}.json``."""
    path = _I18N_DIR / f"weapons_{lang}.json"
    if not path.exists():
        logger.warning("Fichier armes introuvable : %s", path)
        return {}
    try:
        with open(path, encoding="utf-8") as fh:
            data: dict[str, Any] = json.load(fh)
        return data
    except Exception as exc:
        logger.error("Erreur chargement armes %s : %s", path, exc)
        return {}


def get_weapon_label(weapon_id: int, lang: str = "fr") -> str:
    """Retourne le nom localisé d'une arme, ou ``weapon_{id}``."""
    data = _load_weapons_json(lang)
    entry = data.get(str(weapon_id))
    if entry and "name" in entry:
        return entry["name"]
    return f"weapon_{weapon_id}"


def get_weapon_faction(weapon_id: int, lang: str = "fr") -> str:
    """Retourne la faction d'une arme, ou ``Unknown``."""
    data = _load_weapons_json(lang)
    entry = data.get(str(weapon_id))
    if entry and "faction" in entry:
        return entry["faction"]
    return "Unknown"


def get_all_weapon_ids(lang: str = "fr") -> list[int]:
    """Retourne la liste de tous les weapon_id connus."""
    data = _load_weapons_json(lang)
    result: list[int] = []
    for key in data:
        try:
            result.append(int(key))
        except ValueError:
            continue
    return result


def get_weapon_ids_by_faction(faction: str, lang: str = "fr") -> list[int]:
    """Retourne les weapon_id d'une faction donnée (ex : 'UNSC')."""
    data = _load_weapons_json(lang)
    result: list[int] = []
    for key, entry in data.items():
        if entry.get("faction") == faction:
            try:
                result.append(int(key))
            except ValueError:
                continue
    return result
