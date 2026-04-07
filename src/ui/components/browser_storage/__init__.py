"""Composant Streamlit pour la persistance dans localStorage du navigateur.

Stocke sous la clé ``levelup.prefs`` un objet JSON plat contenant :

- ``last_gamertag``  : dernier gamertag sélectionné
- ``last_db_path``   : dernier db_path sélectionné (slug, pas chemin complet)
- ``lang``           : langue choisie dans ce navigateur

Usage :

    from src.ui.components.browser_storage import restore_browser_prefs, persist_browser_prefs

    # En haut de main(), avant toute logique DB :
    prefs = restore_browser_prefs()  # None si déjà restauré ce run
    if prefs:
        _apply_browser_prefs(prefs)

    # Après changement de joueur / langue :
    persist_browser_prefs(last_gamertag="GuiGui", last_db_path="GuiGui", lang="fr")
"""

from __future__ import annotations

import logging
from pathlib import Path

import streamlit.components.v1 as components

logger = logging.getLogger(__name__)

_FRONTEND_DIR = Path(__file__).parent / "frontend"
_component = components.declare_component(
    "levelup_browser_storage",
    path=str(_FRONTEND_DIR),
)

# Clé session_state indiquant que le chargement initial est fait ce run
_LOADED_KEY = "_browser_prefs_loaded"


def restore_browser_prefs() -> dict | None:
    """Lit les préférences depuis localStorage (une seule fois par session Streamlit).

    Retourne un dict avec les clés présentes, ou ``None`` si déjà restauré
    ce run (pour ne pas déclencher de boucles de re-render).

    Le composant rend une iframe hauteur=0, invisible.
    """
    import streamlit as st

    if st.session_state.get(_LOADED_KEY):
        return None

    result = _component(action="read", data={}, key="_ls_read_init", default=None)
    if result is None:
        # Premier rendu : le composant n'a pas encore répondu — attendre prochain run
        return None

    st.session_state[_LOADED_KEY] = True
    if isinstance(result, dict) and result.get("ok") and result.get("data"):
        logger.debug("localStorage restauré : %s", list(result["data"].keys()))
        return result["data"]
    return {}


def persist_browser_prefs(**fields: str) -> None:
    """Écrit un ou plusieurs champs dans localStorage (merge avec l'existant).

    Args:
        **fields: Paires clé/valeur à persister. Valeurs vides ignorées.

    Exemple :
        persist_browser_prefs(last_gamertag="GuiGui", lang="fr")
    """
    import streamlit as st

    clean = {k: str(v) for k, v in fields.items() if v is not None and str(v).strip()}
    if not clean:
        return

    write_key = f"_ls_write_{'_'.join(sorted(clean.keys()))}"
    if st.session_state.get(write_key) == clean:
        return  # Déjà écrit avec les mêmes valeurs ce run

    _component(action="write", data=clean, key=write_key, default=None)
    st.session_state[write_key] = clean
    logger.debug("localStorage mis à jour : %s", list(clean.keys()))


def clear_browser_prefs() -> None:
    """Efface toutes les préférences localStorage (pour debug / reset)."""
    import streamlit as st

    st.session_state.pop(_LOADED_KEY, None)
    _component(action="clear", data={}, key="_ls_clear", default=None)
