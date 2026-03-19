"""Résolution centralisée des labels i18n pour les données de référence.

Domaines supportés : awards, playlists, modes, ranks (fichiers JSON sous
``static/i18n/``). Les citations sont chargées depuis ``citation_mappings``
dans ``metadata.duckdb`` via ``_load_citations_from_db()``.
Les médailles sont chargées depuis ``medal_definitions`` dans
``metadata.duckdb`` via ``medals.py::load_medal_name_maps()``.

Usage::

    from src.ui.i18n.data_labels import label, label_obj

    # Label simple (retourne une chaîne)
    label("playlists", "Quick Play", lang="fr")  # → "Partie rapide"

    # Label objet (citations : name + description + category)
    obj = label_obj("citations", "defenseur du drapeau", lang="en")
    obj["name"]        # → "Flag Defender"
    obj["description"] # → "Protect your team's flag..."
    obj["category"]    # → "Game mode"

    # Fallback : lang demandée → "en" → clé brute
"""

from __future__ import annotations

import json
import logging
from functools import lru_cache
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)

# Racine du repo (deux niveaux au-dessus de src/ui/i18n/)
_REPO_ROOT = Path(__file__).resolve().parents[3]
_I18N_DIR = _REPO_ROOT / "static" / "i18n"

# Domaines supportés et leur structure de fichier
_DOMAIN_FILE_PATTERN = "{domain}_{lang}.json"

# Langues supportées (extensible)
SUPPORTED_LANGS = ("fr", "en")
_FALLBACK_LANG = "en"


@lru_cache(maxsize=32)
def _load_json(path: Path) -> dict[str, Any]:
    """Charge un fichier JSON avec cache LRU."""
    if not path.exists():
        logger.debug("Fichier i18n introuvable : %s", path)
        return {}
    try:
        with open(path, encoding="utf-8") as f:
            data = json.load(f)
        if not isinstance(data, dict):
            logger.warning("Format inattendu dans %s (attendu: dict)", path)
            return {}
        return data
    except Exception:
        logger.exception("Erreur chargement %s", path)
        return {}


def load_domain(domain: str, lang: str) -> dict[str, Any]:
    """Charge toutes les entrées d'un domaine pour une langue donnée.

    Args:
        domain: Nom du domaine (``"citations"``, ``"awards"``, ``"playlists"``,
                ``"modes"``, ``"ranks"``).
        lang: Code langue (``"fr"``, ``"en"``).

    Returns:
        Dict ``{clé: valeur}`` — la valeur est soit une chaîne, soit un dict
        (selon le domaine).
    """
    filename = _DOMAIN_FILE_PATTERN.format(domain=domain, lang=lang)
    return _load_json(_I18N_DIR / filename)


def label(domain: str, key: str, *, lang: str | None = None) -> str:
    """Résout un label simple (chaîne) pour un domaine et une clé.

    Chaîne de fallback : lang demandée → ``"en"`` → clé brute.

    Pour les domaines dont la valeur est un dict (ex: citations),
    retourne le champ ``"name"`` du dict.

    Args:
        domain: Nom du domaine.
        key: Clé de l'entrée (ex: ``"Quick Play"``, ``"defenseur du drapeau"``).
        lang: Code langue. Si ``None``, détecté via ``get_lang()``.

    Returns:
        Label traduit, ou la clé brute si aucune traduction trouvée.
    """
    if lang is None:
        from src.ui.i18n import get_lang

        lang = get_lang()

    # Tentative dans la langue demandée
    data = load_domain(domain, lang)
    val = data.get(key)
    if val is not None:
        if isinstance(val, dict):
            return val.get("name", key)
        return str(val)

    # Fallback vers EN (si différent)
    if lang != _FALLBACK_LANG:
        data_fb = load_domain(domain, _FALLBACK_LANG)
        val_fb = data_fb.get(key)
        if val_fb is not None:
            if isinstance(val_fb, dict):
                return val_fb.get("name", key)
            return str(val_fb)

    # Fallback vers FR (si ni FR ni EN n'ont été testés)
    if lang != "fr" and _FALLBACK_LANG != "fr":
        data_fr = load_domain(domain, "fr")
        val_fr = data_fr.get(key)
        if val_fr is not None:
            if isinstance(val_fr, dict):
                return val_fr.get("name", key)
            return str(val_fr)

    return key


def label_obj(domain: str, key: str, *, lang: str | None = None) -> dict[str, str]:
    """Résout un label structuré (dict) pour un domaine donné.

    Utile pour les domaines riches comme ``"citations"`` qui ont
    ``name``, ``description``, ``category``.

    Args:
        domain: Nom du domaine.
        key: Clé de l'entrée.
        lang: Code langue.

    Returns:
        Dict avec les champs disponibles. Si la valeur est une chaîne simple,
        retourne ``{"name": valeur}``. Si aucune traduction, ``{"name": key}``.
    """
    if lang is None:
        from src.ui.i18n import get_lang

        lang = get_lang()

    data = load_domain(domain, lang)
    val = data.get(key)

    if val is not None:
        if isinstance(val, dict):
            return val
        return {"name": str(val)}

    # Fallback EN
    if lang != _FALLBACK_LANG:
        data_fb = load_domain(domain, _FALLBACK_LANG)
        val_fb = data_fb.get(key)
        if val_fb is not None:
            if isinstance(val_fb, dict):
                return val_fb
            return {"name": str(val_fb)}

    # Fallback FR
    if lang != "fr" and _FALLBACK_LANG != "fr":
        data_fr = load_domain(domain, "fr")
        val_fr = data_fr.get(key)
        if val_fr is not None:
            if isinstance(val_fr, dict):
                return val_fr
            return {"name": str(val_fr)}

    return {"name": key}


def clear_cache() -> None:
    """Vide le cache LRU (utile après mise à jour des fichiers JSON)."""
    _load_json.cache_clear()


def validate_domain(domain: str) -> dict[str, list[str]]:
    """Vérifie la cohérence des fichiers JSON pour un domaine.

    Returns:
        Dict avec les clés ``"missing_in_<lang>"`` listant les clés absentes.
    """
    all_keys: set[str] = set()
    lang_keys: dict[str, set[str]] = {}

    for lang in SUPPORTED_LANGS:
        data = load_domain(domain, lang)
        keys = set(data.keys())
        all_keys |= keys
        lang_keys[lang] = keys

    result: dict[str, list[str]] = {}
    for lang in SUPPORTED_LANGS:
        missing = sorted(all_keys - lang_keys.get(lang, set()))
        if missing:
            result[f"missing_in_{lang}"] = missing

    return result
