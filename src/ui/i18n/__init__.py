"""Package d'internationalisation (FR/EN) pour LevelUp.

Usage :
    from src.ui.i18n import t, get_lang

    st.warning(t("no_matches"))
    fig = plot_kda(lang=get_lang())
"""
from __future__ import annotations

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    pass

# Modules de chaînes disponibles — importés à la demande pour éviter les
# circular imports. La fonction t() ci-dessous agrège tous les dicts.
_REGISTRY: dict[str, dict[str, dict[str, str]]] | None = None


def _build_registry() -> dict[str, dict[str, dict[str, str]]]:
    """Construit le registre global en fusionnant tous les fichiers de chaînes."""
    from src.ui.i18n import cli, common, pages, viz, widgets

    registry: dict[str, dict[str, dict[str, str]]] = {}
    for module in (common, pages, widgets, viz, cli):
        for key, translations in module.STRINGS.items():
            if key in registry:
                # Collision de clé entre modules — la première déclaration gagne
                continue
            registry[key] = translations
    return registry


def get_lang() -> str:
    """Retourne la langue active depuis st.session_state (défaut : 'fr')."""
    try:
        import streamlit as st

        return st.session_state.get("lang", "fr")
    except Exception:
        # Contexte hors Streamlit (scripts, tests)
        import os

        return os.environ.get("LEVELUP_LANG", "fr")


def set_lang(lang: str) -> None:
    """Définit la langue active dans st.session_state."""
    try:
        import streamlit as st

        st.session_state["lang"] = lang
    except Exception:
        import os

        os.environ["LEVELUP_LANG"] = lang


def t(key: str, lang: str | None = None, **kwargs: object) -> str:
    """Retourne la chaîne traduite pour la clé donnée.

    Args:
        key:    Clé de traduction (ex: ``"no_matches"``).
        lang:   Langue cible. Si None, utilise ``get_lang()``.
        **kwargs: Variables à injerer via str.format() (ex: ``name="foo"``).

    Returns:
        La chaîne traduite, ou la clé entre crochets si introuvable.
    """
    global _REGISTRY
    if _REGISTRY is None:
        _REGISTRY = _build_registry()

    resolved_lang = lang or get_lang()
    translations = _REGISTRY.get(key)
    if translations is None:
        return f"[{key}]"

    text = translations.get(resolved_lang) or translations.get("fr") or f"[{key}]"
    if kwargs:
        try:
            text = text.format(**kwargs)
        except (KeyError, ValueError):
            pass
    return text


def reset_registry() -> None:
    """Vide le cache du registre (utile pour les tests)."""
    global _REGISTRY
    _REGISTRY = None


__all__ = ["t", "get_lang", "set_lang", "reset_registry"]
