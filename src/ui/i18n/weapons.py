"""Helpers d'affichage des armes Halo Infinite.

``get_weapon_label``  — nom localisé (délègue à resolve_weapon_display).
``get_weapon_faction`` — faction via weapons_{lang}.json (données non en DB).
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
    """Retourne le nom localisé d'une arme.

    Délègue à resolve_weapon_display (metadata.duckdb → dicts Python).
    """
    from src.analysis._weapon_data import resolve_weapon_display

    return resolve_weapon_display(weapon_id, lang) or f"weapon_{weapon_id}"


def get_weapon_faction(weapon_id: int, lang: str = "fr") -> str:
    """Retourne la faction d'une arme, ou ``Unknown``.

    Source : weapons_{lang}.json (les IDs sont des clés API courtes,
    différentes des IDs filmshell — lookup sur str(weapon_id)).
    """
    data = _load_weapons_json(lang)
    entry = data.get(str(weapon_id))
    if entry and "faction" in entry:
        return entry["faction"]
    return "Unknown"
