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
    """Construit le registre global en fusionnant tous les fichiers de chaînes.

    Supporte les **alias** : si la valeur d'une clé est un ``str`` au lieu
    d'un ``dict``, elle est traitée comme un pointeur vers une autre clé.
    Exemple dans un module STRINGS::

        "ts_kills_label": "col_kills",   # alias → résolu vers common.col_kills

    Les alias sont résolus après la fusion de tous les modules.
    """
    import logging

    from src.ui.i18n import cli, common, pages, setup, viz, widgets

    logger = logging.getLogger(__name__)
    registry: dict[str, dict[str, dict[str, str]]] = {}
    aliases: dict[str, str] = {}  # clé alias → clé cible
    # Ordre de priorité : common > pages > widgets > viz > cli > setup
    _module_names = {
        id(common.STRINGS): "common",
        id(pages.STRINGS): "pages",
        id(widgets.STRINGS): "widgets",
        id(viz.STRINGS): "viz",
        id(cli.STRINGS): "cli",
        id(setup.STRINGS): "setup",
    }
    for module in (common, pages, widgets, viz, cli, setup):
        mod_name = _module_names.get(id(module.STRINGS), "?")
        for key, translations in module.STRINGS.items():
            # Alias : valeur est un str → pointer vers une autre clé
            if isinstance(translations, str):
                if key not in registry:
                    aliases[key] = translations
                continue

            if key in registry:
                # Collision — la première déclaration gagne.
                # Warning si les valeurs diffèrent (bug potentiel).
                existing = registry[key]
                if existing != translations:
                    logger.warning(
                        "i18n: collision de clé '%s' entre modules "
                        "(gardé=%s, ignoré=%s avec valeurs différentes)",
                        key,
                        "earlier",
                        mod_name,
                    )
                continue
            registry[key] = translations

    # Résolution des alias
    for alias_key, target_key in aliases.items():
        if target_key in registry:
            registry[alias_key] = registry[target_key]
        else:
            logger.warning(
                "i18n: alias '%s' → '%s' : clé cible introuvable",
                alias_key,
                target_key,
            )

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
        import contextlib

        with contextlib.suppress(KeyError, ValueError):
            text = text.format(**kwargs)
    return text


def reset_registry() -> None:
    """Vide le cache du registre (utile pour les tests)."""
    global _REGISTRY
    _REGISTRY = None


__all__ = ["t", "get_lang", "set_lang", "reset_registry", "get_outcome_map", "get_weekdays"]

from src.data.domain._refdata_outcomes import get_outcome_map  # noqa: E402 — re-export


def get_weekdays(lang: str | None = None) -> dict[int, str]:
    """Retourne les noms de jours abrégés traduits (1=lundi … 7=dimanche)."""
    return {
        1: t("day_mon", lang=lang),
        2: t("day_tue", lang=lang),
        3: t("day_wed", lang=lang),
        4: t("day_thu", lang=lang),
        5: t("day_fri", lang=lang),
        6: t("day_sat", lang=lang),
        7: t("day_sun", lang=lang),
    }
