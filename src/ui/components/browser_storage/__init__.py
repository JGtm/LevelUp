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

import ast
import json
import logging
from pathlib import Path

from src.utils.paths import DATA_DIR

logger = logging.getLogger(__name__)

_PREFS_FILE: Path = DATA_DIR / "ui_prefs.json"
_VALID_LANGS = {"fr", "en"}

# Clé session_state indiquant que le chargement initial est fait ce run
_LOADED_KEY = "_browser_prefs_loaded"


def _read_prefs() -> dict[str, str]:
    """Lit uniquement les préférences scalaires. Retourne {} si absent ou invalide."""
    try:
        data = _read_full_prefs()
        return {
            str(k): str(v)
            for k, v in data.items()
            if v is not None and not isinstance(v, (dict, list, tuple, set))
        }
    except Exception as exc:
        logger.debug("Lecture ui_prefs.json échouée : %s", exc)
    return {}


def _write_prefs(data: dict[str, str]) -> None:
    """Écrit les préférences scalaires sans écraser les structures JSON existantes."""
    try:
        full_data = _read_full_prefs()
        full_data.update(data)
        _write_full_prefs(full_data)
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


def resolve_browser_pref_lang(pref_lang: str | None, session_lang: str | None) -> str | None:
    """Retourne la langue à appliquer depuis les prefs, ou None si inutile.

    Cas couvert : si la session n'a pas encore de langue explicite, une préférence
    persistée à ``fr`` doit être appliquée, même si ``fr`` est aussi le défaut UI.
    """
    preferred = str(pref_lang or "").strip().lower()
    if preferred not in _VALID_LANGS:
        return None

    current = str(session_lang or "").strip().lower()
    if current not in _VALID_LANGS:
        return preferred

    if current != preferred:
        return preferred

    return None


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


# ---------------------------------------------------------------------------
# Likes médias
# ---------------------------------------------------------------------------

_LIKES_KEY = "media_likes"
_LIKES_SS_KEY = "_media_likes"


def _read_full_prefs() -> dict:
    """Lit le fichier de préférences en tant qu'objet JSON complet (sans coercion str)."""
    try:
        if _PREFS_FILE.exists():
            data = json.loads(_PREFS_FILE.read_text(encoding="utf-8"))
            if isinstance(data, dict):
                return data
    except Exception as exc:
        logger.debug("Lecture ui_prefs (full) échouée : %s", exc)
    return {}


def _write_full_prefs(data: dict) -> None:
    """Écrit le fichier de préférences (objet JSON complet)."""
    try:
        _PREFS_FILE.parent.mkdir(parents=True, exist_ok=True)
        _PREFS_FILE.write_text(json.dumps(data, ensure_ascii=False), encoding="utf-8")
    except Exception as exc:
        logger.debug("Écriture ui_prefs (full) échouée : %s", exc)


def _normalize_media_likes(values: list[object] | tuple[object, ...] | set[object]) -> set[str]:
    """Nettoie une séquence de likes pour ne garder que des chemins non vides."""
    cleaned: set[str] = set()
    for value in values:
        text = str(value).strip()
        if text:
            cleaned.add(text)
    return cleaned


def _parse_serialized_media_likes(raw: str) -> set[str] | None:
    """Parse une ancienne valeur sérialisée de media_likes si possible."""
    text = raw.strip()
    if not text:
        return set()

    parsed: object | None = None
    try:
        parsed = json.loads(text)
    except json.JSONDecodeError:
        try:
            parsed = ast.literal_eval(text)
        except (ValueError, SyntaxError):
            parsed = None

    if isinstance(parsed, (list, tuple, set)):
        return _normalize_media_likes(parsed)
    return None


def _coerce_media_likes(raw: object) -> tuple[set[str], bool]:
    """Normalise media_likes et indique si le fichier doit être réparé."""
    if raw is None:
        return set(), False
    if isinstance(raw, list):
        return _normalize_media_likes(raw), False
    if isinstance(raw, (tuple, set)):
        return _normalize_media_likes(raw), True
    if isinstance(raw, str):
        parsed = _parse_serialized_media_likes(raw)
        if parsed is not None:
            return parsed, True
        return set(), True
    return set(), True


def load_media_likes() -> set[str]:
    """Retourne l'ensemble des file_path aimés (persisté dans ui_prefs.json)."""
    import streamlit as st

    cached = st.session_state.get(_LIKES_SS_KEY)
    if cached is not None:
        return cached

    data = _read_full_prefs()
    likes, should_repair = _coerce_media_likes(data.get(_LIKES_KEY))
    if should_repair:
        data[_LIKES_KEY] = sorted(likes)
        _write_full_prefs(data)
    st.session_state[_LIKES_SS_KEY] = likes
    return likes


def toggle_media_like(file_path: str) -> bool:
    """Bascule le like pour un file_path. Retourne True si désormais liké.

    Met à jour session_state ET persiste dans ui_prefs.json.
    """
    import streamlit as st

    likes = load_media_likes()
    if file_path in likes:
        likes.discard(file_path)
        liked = False
    else:
        likes.add(file_path)
        liked = True

    st.session_state[_LIKES_SS_KEY] = likes
    data = _read_full_prefs()
    data[_LIKES_KEY] = sorted(likes)
    _write_full_prefs(data)
    return liked
