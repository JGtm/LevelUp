"""Traductions FR/EN pour launcher.py.

Usage:
    from src.utils.launcher_i18n import t as _t
    print(_t("setup_done", lang))
    print(_t("sync_new_matches", lang, n=5))

Les chaînes sont stockées dans ``launcher_i18n.json`` (même dossier).
"""

from __future__ import annotations

import json
from pathlib import Path

_JSON_PATH = Path(__file__).with_suffix(".json")

with _JSON_PATH.open(encoding="utf-8") as _f:
    STRINGS: dict[str, dict[str, str]] = json.load(_f)


def t(key: str, lang: str = "fr", **kwargs: object) -> str:
    """Retourne la chaîne traduite pour la clé et la langue données.

    Args:
        key: Clé de traduction.
        lang: Code langue ('fr' ou 'en'). Par défaut 'fr'.
        **kwargs: Variables à interpoler dans la chaîne via .format().

    Returns:
        Chaîne traduite, ou la clé elle-même si introuvable.
    """
    entry = STRINGS.get(key)
    if entry is None:
        return key
    text = entry.get(lang) or entry.get("fr") or key
    return text.format(**kwargs) if kwargs else text
