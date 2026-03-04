"""Sub-package i18n : chaînes pour les graphiques Plotly.

Fusionne les sous-modules et expose ``viz_t()`` pour résolution
sans dépendance Streamlit.
"""

from __future__ import annotations

import contextlib

from src.ui.i18n.viz import (
    axes,
    hovers,
    labels,
    titles,
    traces,
)

STRINGS: dict[str, dict[str, str] | str] = {}

# Fusion de tous les sous-modules (ordre stable)
for _mod in (traces, hovers, axes, titles, labels):
    STRINGS.update(_mod.STRINGS)


def viz_t(key: str, lang: str = "fr", **kwargs: object) -> str:
    """Helper rapide pour les visualisations, sans dépendance Streamlit.

    Résout d'abord dans ``viz.STRINGS``, puis tombe en fallback sur
    ``common.STRINGS`` pour éviter de dupliquer les labels partagés
    (métriques, colonnes, résultats…).

    Supporte les **alias** : si la valeur dans STRINGS est un ``str``
    au lieu d'un ``dict``, elle est utilisée comme clé à résoudre
    (dans STRINGS d'abord, puis common.STRINGS).

    Args:
        key:  Clé dans STRINGS (ou common.STRINGS en fallback).
        lang: ``"fr"`` ou ``"en"``.
        **kwargs: Variables pour str.format().

    Returns:
        La chaîne traduite ou la clé entre crochets.
    """
    entry = STRINGS.get(key)

    # Résolution d'alias (str → clé cible)
    if isinstance(entry, str):
        target = entry
        entry = STRINGS.get(target)
        if entry is None or isinstance(entry, str):
            from src.ui.i18n.common import STRINGS as COMMON_STRINGS

            entry = COMMON_STRINGS.get(target)

    if entry is None:
        # Fallback vers common.STRINGS (import paresseux pour éviter circular)
        from src.ui.i18n.common import STRINGS as COMMON_STRINGS

        entry = COMMON_STRINGS.get(key)

    if entry is None:
        return f"[{key}]"
    if isinstance(entry, str):
        # Alias non résolu dans common — ne devrait pas arriver
        return entry
    text = entry.get(lang) or entry.get("fr") or f"[{key}]"
    if kwargs:
        with contextlib.suppress(KeyError, ValueError):
            text = text.format(**kwargs)
    return text


__all__ = ["STRINGS", "viz_t"]
