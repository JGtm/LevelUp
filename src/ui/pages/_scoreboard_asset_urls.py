"""Résolution d'URLs d'assets statiques pour le scoreboard inline.

Fournit les index (medal_id → URL, weapon_name → URL) construits une seule fois
au démarrage du processus via ``functools.cache``.
"""

from __future__ import annotations

import functools
import re
import unicodedata
from pathlib import Path

from src.config import get_repo_root


@functools.cache
def _build_medal_icon_url_index() -> dict[int, str]:
    """Construit l'index medal_id -> URL statique Streamlit."""
    icons_dir = Path(get_repo_root()) / "static" / "medals" / "icons"
    index: dict[int, str] = {}
    if not icons_dir.exists():
        return index
    for icon_file in icons_dir.iterdir():
        if not icon_file.is_file() or icon_file.suffix.lower() != ".png":
            continue
        try:
            medal_id = int(icon_file.stem)
        except ValueError:
            continue
        index[medal_id] = f"/app/static/medals/icons/{icon_file.name}"
    return index


def medal_icon_url(medal_id: int) -> str | None:
    """Retourne l'URL statique d'une icône de médaille si disponible."""
    return _build_medal_icon_url_index().get(int(medal_id))


def _normalize_weapon_asset_key(value: str) -> str:
    normalized = unicodedata.normalize("NFKD", value)
    ascii_only = normalized.encode("ascii", "ignore").decode("ascii")
    return re.sub(r"[^a-z0-9]+", "", ascii_only.lower())


@functools.cache
def _build_weapon_asset_url_index() -> dict[str, str]:
    """Construit l'index nom normalisé -> URL statique d'asset d'arme."""
    assets_dir = Path(get_repo_root()) / "static" / "weapons-assets"
    index: dict[str, str] = {}
    if not assets_dir.exists():
        return index
    for asset_file in assets_dir.iterdir():
        if not asset_file.is_file() or asset_file.suffix.lower() != ".png":
            continue
        index[_normalize_weapon_asset_key(asset_file.stem)] = (
            f"/app/static/weapons-assets/{asset_file.name}"
        )
    return index


_WEAPON_ASSET_ALIASES: dict[str, str] = {
    "banditevo": "Bandit",
    "m392bandit": "Bandit",
    "pulsecarbine": "Carabine",
    "carabineaimpulsion": "Carabine",
    "vestigecarbine": "Carabine",
    "carabinevestige": "Carabine",
    "melee": "Melee",
    "corpsacorps": "Melee",
    "grenade": "Grenade",
    "fraggrenade": "Grenade",
    "plasmagrenade": "Grenade",
    "dynamogrenade": "Grenade",
    "br75": "BR75",
    "cindershot": "Cremator",
    "cremateur": "Cremator",
    "cqs48bulldog": "Bulldog",
    "bulldog": "Bulldog",
    "disruptor": "Disruptor",
    "disrupteur": "Disruptor",
    "gravityhammer": "Hammer",
    "marteauantigravite": "Hammer",
    "heatwave": "Heatwave",
    "calcineur": "Heatwave",
    "m41spnkr": "M41",
    "fuelrodspnkr": "M41",
    "ma40ar": "MA40",
    "ma40": "MA40",
    "mangler": "Mangler",
    "dechiqueteur": "Mangler",
    "mutilator": "Mutilator",
    "mutilateur": "Mutilator",
    "mk51sidekick": "Sidekick",
    "mk50sidekick": "Sidekick",
    "sidekick": "Sidekick",
    "mlrs2hydra": "Hydra",
    "hydra": "Hydra",
    "needler": "Needler-1",
    "plasmapistol": "Plasma",
    "pistoletaplasma": "Plasma",
    "ravager": "Ravager",
    "ravageur": "Ravager",
    "s7sniper": "Sniper-S7",
    "sentinelbeam": "Sentinel",
    "rayondesentinelle": "Sentinel",
    "shockrifle": "Shock-rifle",
    "shockrifleranked": "Shock-rifle",
    "fusilelectrique": "Shock-rifle",
    "skewer": "Skewer",
    "empaleur": "Skewer",
    "stalkerrifle": "Stalker",
    "fusiltraqueur": "Stalker",
    "vk78commando": "Commando",
    "commando": "Commando",
    "energysword": "Sword",
    "epeeaenergie": "Sword",
}


def weapon_asset_url(weapon_name: str) -> str | None:
    """Retourne l'URL statique d'un asset d'arme si disponible."""
    normalized_name = _normalize_weapon_asset_key(weapon_name)
    asset_key = _WEAPON_ASSET_ALIASES.get(normalized_name, weapon_name)
    normalized_asset_key = _normalize_weapon_asset_key(asset_key)
    return _build_weapon_asset_url_index().get(normalized_asset_key)


__all__ = ["medal_icon_url", "weapon_asset_url"]
