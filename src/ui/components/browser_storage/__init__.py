"""Persistance des préférences utilisateur dans un fichier JSON côté serveur.

Stocke dans ``data/ui_prefs.json`` un objet JSON plat contenant :

- ``last_gamertag``  : dernier gamertag sélectionné
- ``last_db_path``   : dernier db_path sélectionné (slug, pas chemin complet)
- ``lang``           : langue choisie
- ``show_hints``     : affichage des aides à la lecture ("1" = activé, "0" = masqué)

Usage :

    from src.ui.components.browser_storage import (
        restore_browser_prefs, persist_browser_prefs,
        hints_visible, restore_hints_from_prefs,
    )

    # En haut de main(), avant toute logique DB :
    prefs = restore_browser_prefs()  # None si déjà restauré ce run
    if prefs:
        _apply_browser_prefs(prefs)
        restore_hints_from_prefs(prefs)

    # Lecture dynamique dans n'importe quelle page :
    if hints_visible():
        st.caption("Légende...")

    # Après changement de joueur / langue :
    persist_browser_prefs(last_gamertag="GuiGui", last_db_path="GuiGui", lang="fr")
"""

from __future__ import annotations

import json
import logging
from pathlib import Path

from src.utils.paths import DATA_DIR

logger = logging.getLogger(__name__)

_PREFS_FILE: Path = DATA_DIR / "ui_prefs.json"

# Clé session_state indiquant que le chargement initial est fait ce run
_LOADED_KEY = "_browser_prefs_loaded"


def _read_prefs() -> dict[str, str]:
    """Lit le fichier de préférences. Retourne {} si absent ou invalide."""
    try:
        if _PREFS_FILE.exists():
            data = json.loads(_PREFS_FILE.read_text(encoding="utf-8"))
            if isinstance(data, dict):
                return {str(k): str(v) for k, v in data.items() if v is not None}
    except Exception as exc:
        logger.debug("Lecture ui_prefs.json échouée : %s", exc)
    return {}


def _write_prefs(data: dict[str, str]) -> None:
    """Écrit le fichier de préférences."""
    try:
        _PREFS_FILE.parent.mkdir(parents=True, exist_ok=True)
        _PREFS_FILE.write_text(json.dumps(data, ensure_ascii=False), encoding="utf-8")
    except Exception as exc:
        logger.debug("Écriture ui_prefs.json échouée : %s", exc)


def restore_browser_prefs() -> dict | None:
    """Lit les préférences persistées (une seule fois par session Streamlit).

    Retourne un dict avec les clés présentes, ou ``None`` si déjà restauré
    ce run (pour ne pas déclencher de boucles de re-render).
    """
    import streamlit as st

    if st.session_state.get(_LOADED_KEY):
        return None

    st.session_state[_LOADED_KEY] = True
    prefs = _read_prefs()
    logger.debug("Préférences restaurées : %s", list(prefs.keys()))
    return prefs


def persist_browser_prefs(**fields: str) -> None:
    """Écrit un ou plusieurs champs dans les préférences (merge avec l'existant).

    Args:
        **fields: Paires clé/valeur à persister. Valeurs vides ignorées.

    Exemple :
        persist_browser_prefs(last_gamertag="GuiGui", lang="fr")
    """
    import streamlit as st

    clean = {k: str(v) for k, v in fields.items() if v is not None and str(v).strip()}
    if not clean:
        return

    # Déduplication : éviter les écritures répétées avec les mêmes valeurs au sein d'un run
    write_key = f"_prefs_write_{'_'.join(sorted(clean.keys()))}"
    if st.session_state.get(write_key) == clean:
        return

    existing = _read_prefs()
    existing.update(clean)
    _write_prefs(existing)
    st.session_state[write_key] = clean
    logger.debug("Préférences mises à jour : %s", list(clean.keys()))


def clear_browser_prefs() -> None:
    """Efface toutes les préférences (pour debug / reset)."""
    import streamlit as st

    st.session_state.pop(_LOADED_KEY, None)
    try:
        if _PREFS_FILE.exists():
            _PREFS_FILE.unlink()
    except Exception as exc:
        logger.debug("Suppression ui_prefs.json échouée : %s", exc)


# ---------------------------------------------------------------------------
# Aides à la lecture (hints)
# ---------------------------------------------------------------------------

_HINTS_KEY = "show_hints"
_HINTS_SS_KEY = "_hints_visible"


def hints_visible() -> bool:
    """Retourne True si les aides à la lecture sont activées (défaut : True)."""
    import streamlit as st

    return bool(st.session_state.get(_HINTS_SS_KEY, True))


def restore_hints_from_prefs(prefs: dict) -> None:
    """Restaure la préférence d'affichage des aides depuis ui_prefs.json.

    Appelé une seule fois au démarrage depuis _maybe_apply_browser_prefs().
    Ne fait rien si la clé est absente du fichier (conserve le défaut True).
    """
    import streamlit as st

    raw = prefs.get(_HINTS_KEY)
    if raw is not None:
        st.session_state[_HINTS_SS_KEY] = str(raw) != "0"
        logger.debug("Aides à la lecture restaurées : %s", st.session_state[_HINTS_SS_KEY])
    else:
        logger.debug("Aides à la lecture : clé absente de ui_prefs.json, défaut True conservé")
