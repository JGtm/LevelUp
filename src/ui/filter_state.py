"""Gestion de la persistance des filtres par joueur.

Ce module permet de sauvegarder et charger les filtres activés/désactivés
pour chaque joueur, afin d'améliorer l'UX en conservant les préférences
entre les sessions et les changements de joueur.

Il centralise aussi les clés session_state liées aux filtres pour garantir
un nettoyage exhaustif lors du changement de joueur.

Architecture intent-based (v5.2) :
    Les filtres cascade (playlists/modes/cartes) sont sauvegardés avec leur
    *intention* (include/exclude), pas leurs valeurs brutes. Cela permet de
    gérer automatiquement les nouvelles playlists/modes/cartes sans que
    l'utilisateur ait à les re-cocher manuellement.
"""

from __future__ import annotations

import contextlib
import json
import logging
from datetime import date
from pathlib import Path

import streamlit as st

from src.app.session_keys import SK
from src.ui._filter_state_model import (
    FILTER_DATA_KEYS,
    FILTER_WIDGET_KEY_PREFIXES,
    FilterPreferences,
    get_all_filter_keys_to_clear,
)
from src.ui._filter_state_model import (
    detect_filter_mode as _detect_filter_mode,
)
from src.ui._filter_state_model import (
    get_filters_dir as _get_filters_dir,  # noqa: F401 — backward compat (tests)
)
from src.ui._filter_state_model import (
    get_player_key as _get_player_key,
)

logger = logging.getLogger(__name__)

# Re-exports pour backward compat (tests, imports existants)
__all__ = [
    "FILTER_DATA_KEYS",
    "FILTER_WIDGET_KEY_PREFIXES",
    "FilterPreferences",
    "get_all_filter_keys_to_clear",
]


def _get_filter_file_path(player_key: str) -> Path:
    """Retourne le chemin du fichier de filtres pour un joueur."""
    filters_dir = _get_filters_dir()
    safe_key = player_key.replace("/", "_").replace("\\", "_").replace(":", "_")
    return filters_dir / f"{safe_key}.json"


# ---------------------------------------------------------------------------
# Save / Load / Apply / Clear
# ---------------------------------------------------------------------------


def save_filter_preferences(  # noqa: C901, PLR0912, PLR0913, PLR0915
    xuid: str,
    db_path: str | None = None,
    preferences: FilterPreferences | None = None,
    *,
    all_playlists: list[str] | None = None,
    all_modes: list[str] | None = None,
    all_maps: list[str] | None = None,
    all_experience_types: list[str] | None = None,
) -> None:
    """Sauvegarde les préférences de filtres pour un joueur.

    Si preferences n'est pas fourni, lit depuis session_state.
    Avec les paramètres ``all_*``, utilise l'architecture intent-based :
    détecte automatiquement si le mode est include ou exclude, et stocke
    en conséquence (exclusions en mode exclude, inclusions en mode include).

    Args:
        xuid: XUID ou gamertag du joueur.
        db_path: Chemin vers la base de données (optionnel).
        preferences: Préférences à sauvegarder (optionnel, lit depuis session_state si None).
        all_playlists: Toutes les playlists disponibles (pour détection du mode).
        all_modes: Tous les modes disponibles (pour détection du mode).
        all_maps: Toutes les cartes disponibles (pour détection du mode).
        all_experience_types: Tous les types d'expérience disponibles.
    """
    if preferences is None:
        preferences = FilterPreferences()

        # Mode de filtre
        filter_mode = st.session_state.get("filter_mode")
        if filter_mode in ("Période", "Sessions"):
            preferences.filter_mode = filter_mode

        # Mode Période
        start_date_val = st.session_state.get("start_date_cal")
        if isinstance(start_date_val, date):
            preferences.start_date = start_date_val.isoformat()
        end_date_val = st.session_state.get("end_date_cal")
        if isinstance(end_date_val, date):
            preferences.end_date = end_date_val.isoformat()

        # Mode Sessions
        gap_minutes_val = st.session_state.get("gap_minutes")
        if isinstance(gap_minutes_val, int | float):
            preferences.gap_minutes = int(gap_minutes_val)
        picked_session_label_val = st.session_state.get(SK.PICKED_SESSION_LABEL)
        if isinstance(picked_session_label_val, str):
            preferences.picked_session_label = picked_session_label_val
        # Sauvegarder aussi la "vraie dernière session" pour détecter le tracking
        latest_session_label_val = st.session_state.get("_latest_session_label")
        if isinstance(latest_session_label_val, str):
            preferences.latest_session_label = latest_session_label_val
        # Sessions solo / escouade (v5.3)
        solo_label = st.session_state.get(SK.PICKED_SOLO_SESSION_LABEL)
        if isinstance(solo_label, str):
            preferences.picked_solo_session_label = solo_label
        squad_label = st.session_state.get(SK.PICKED_SQUAD_SESSION_LABEL)
        if isinstance(squad_label, str):
            preferences.picked_squad_session_label = squad_label
        # Amis sélectionnés dans Teammates (v5.3)
        friends_labels = st.session_state.get(SK.TEAMMATES_PICKED_LABELS)
        if isinstance(friends_labels, list):
            preferences.friends_selected_labels = friends_labels

        # Filtres cascade — logique intent-based
        # Mapping session_key → exclusions_key pour mise à jour mid-session
        _EXCLUSIONS_KEY_MAP: dict[str, str] = {
            SK.FILTER_PLAYLISTS: "_playlists_exclusions",
            SK.FILTER_MODES: "_modes_exclusions",
            SK.FILTER_MAPS: "_maps_exclusions",
            SK.FILTER_EXPERIENCE_TYPES: "_experience_types_exclusions",
        }

        def _save_filter(
            ss_key: str,
            mode_ss_key: str,
            all_opts: list[str] | None,
        ) -> tuple[list[str] | None, str | None]:
            """Retourne (stored_list, mode) pour un filtre donné.

            Met aussi à jour la clé d'exclusions dans session_state pour que
            ``_reconcile_filter_options`` distingue les options délibérément
            décochées des options vraiment nouvelles.
            """
            val = st.session_state.get(ss_key)
            if not isinstance(val, set | list):
                return None, None
            current_mode = st.session_state.get(mode_ss_key, "include")
            if all_opts:
                mode = _detect_filter_mode(val, all_opts, current_mode)
                st.session_state[mode_ss_key] = mode
                exclusions = set(all_opts) - set(val)
                stored = sorted(exclusions) if mode == "exclude" else sorted(val)
                # --- FIX: propager les exclusions dans session_state ---
                # Sans cette mise à jour, _reconcile_filter_options considère
                # les items décochés mid-session comme "truly new" et les
                # re-coche automatiquement.
                excl_key = _EXCLUSIONS_KEY_MAP.get(ss_key)
                if excl_key:
                    st.session_state[excl_key] = exclusions
            else:
                mode = "include"
                stored = sorted(val)
            return stored, mode

        pl_list, pl_mode = _save_filter(
            SK.FILTER_PLAYLISTS, "_playlists_filter_mode", all_playlists
        )
        if pl_list is not None:
            preferences.playlists_selected = pl_list
            preferences.playlists_mode = pl_mode

        mo_list, mo_mode = _save_filter(SK.FILTER_MODES, "_modes_filter_mode", all_modes)
        if mo_list is not None:
            preferences.modes_selected = mo_list
            preferences.modes_mode = mo_mode

        ma_list, ma_mode = _save_filter(SK.FILTER_MAPS, "_maps_filter_mode", all_maps)
        if ma_list is not None:
            preferences.maps_selected = ma_list
            preferences.maps_mode = ma_mode

        exp_list, exp_mode = _save_filter(
            SK.FILTER_EXPERIENCE_TYPES, "_experience_types_filter_mode", all_experience_types
        )
        if exp_list is not None:
            preferences.experience_types = exp_list
            preferences.experience_types_mode = exp_mode

    # Sauvegarder dans le fichier
    player_key = _get_player_key(xuid, db_path)
    file_path = _get_filter_file_path(player_key)
    logger.debug("Prefs filtres sauvegardées (xuid=%s...)", str(xuid or "")[:8])

    try:
        with open(file_path, "w", encoding="utf-8") as f:
            json.dump(preferences.to_dict(), f, indent=2, ensure_ascii=False)
    except Exception as e:
        # Ne pas bloquer l'application si la sauvegarde échoue
        st.warning(f"Impossible de sauvegarder les préférences de filtres: {e}")


def load_filter_preferences(
    xuid: str,
    db_path: str | None = None,
) -> FilterPreferences | None:
    """Charge les préférences de filtres pour un joueur.

    Args:
        xuid: XUID ou gamertag du joueur.
        db_path: Chemin vers la base de données (optionnel).

    Returns:
        Préférences chargées ou None si aucune préférence sauvegardée.
    """
    player_key = _get_player_key(xuid, db_path)
    file_path = _get_filter_file_path(player_key)

    if not file_path.exists():
        return None

    try:
        with open(file_path, encoding="utf-8") as f:
            data = json.load(f)
        return FilterPreferences.from_dict(data)
    except Exception:
        # Si le fichier est corrompu, retourner None
        return None


def apply_filter_preferences(  # noqa: C901, PLR0912, PLR0913, PLR0915
    xuid: str,
    db_path: str | None = None,
    preferences: FilterPreferences | None = None,
    *,
    all_playlists: list[str] | None = None,
    all_modes: list[str] | None = None,
    all_maps: list[str] | None = None,
    all_experience_types: list[str] | None = None,
) -> None:
    """Applique les préférences de filtres dans session_state.

    Si preferences n'est pas fourni, charge depuis le fichier.
    Avec les paramètres ``all_*`` (v5.2), interprète les préférences en mode
    intent-based : les exclusions sont reconstituées, les nouvelles options
    sont auto-cochées si le mode est "exclude".

    Args:
        xuid: XUID ou gamertag du joueur.
        db_path: Chemin vers la base de données (optionnel).
        preferences: Préférences à appliquer (optionnel, charge depuis fichier si None).
        all_playlists: Toutes les playlists disponibles (pour reconstruire la sélection).
        all_modes: Tous les modes disponibles.
        all_maps: Toutes les cartes disponibles.
        all_experience_types: Tous les types d'expérience disponibles.
    """
    if preferences is None:
        preferences = load_filter_preferences(xuid, db_path)
        if preferences is None:
            return

    logger.debug("Prefs filtres restaurées (xuid=%s...)", str(xuid or "")[:8])

    # Mode de filtre
    if preferences.filter_mode:
        st.session_state["filter_mode"] = preferences.filter_mode

    # Mode Période
    if preferences.start_date:
        try:
            start_date_obj = date.fromisoformat(preferences.start_date)
            st.session_state["start_date_cal"] = start_date_obj
        except (ValueError, TypeError):
            pass

    if preferences.end_date:
        try:
            end_date_obj = date.fromisoformat(preferences.end_date)
            st.session_state["end_date_cal"] = end_date_obj
        except (ValueError, TypeError):
            pass

    # Mode Sessions
    if preferences.gap_minutes is not None:
        st.session_state["gap_minutes"] = preferences.gap_minutes

    # Sessions solo / escouade (v5.3) — précédence sur l'ancien picked_session_label
    if preferences.picked_solo_session_label:
        st.session_state[SK.PICKED_SOLO_SESSION_LABEL] = preferences.picked_solo_session_label
    if preferences.picked_squad_session_label:
        st.session_state[SK.PICKED_SQUAD_SESSION_LABEL] = preferences.picked_squad_session_label

    # Dériver picked_session_label actif depuis solo/squad, ou fallback legacy
    _active_new = None
    solo_saved = preferences.picked_solo_session_label
    squad_saved = preferences.picked_squad_session_label
    if solo_saved and solo_saved != "(toutes)":
        _active_new = solo_saved
    elif squad_saved and squad_saved != "(toutes)":
        _active_new = squad_saved

    if _active_new:
        st.session_state[SK.PICKED_SESSION_LABEL] = _active_new
        st.session_state["picked_sessions"] = [_active_new]
    elif preferences.picked_session_label:
        st.session_state[SK.PICKED_SESSION_LABEL] = preferences.picked_session_label
        # Mettre à jour picked_sessions aussi
        if preferences.picked_session_label != "(toutes)":
            st.session_state["picked_sessions"] = [preferences.picked_session_label]
        else:
            st.session_state["picked_sessions"] = []

    # Amis sélectionnés dans Teammates (v5.3)
    if preferences.friends_selected_labels:
        st.session_state[SK.TEAMMATES_PICKED_LABELS] = preferences.friends_selected_labels

    # Filtres cascade — logique intent-based
    def _apply_filter(  # noqa: PLR0913
        stored: list[str] | None,
        mode: str | None,
        all_opts: list[str] | None,
        ss_key: str,
        mode_ss_key: str,
        exclusions_ss_key: str,
    ) -> None:
        """Applique un filtre cascade dans session_state."""
        if stored is None:
            return
        effective_mode = mode or "include"  # backward compat : None → include
        exclusions: set[str] = set()

        if effective_mode == "exclude" and all_opts:
            exclusions = set(stored)
            result = set(all_opts) - exclusions
        else:
            result = set(stored)

        st.session_state[ss_key] = result
        st.session_state[mode_ss_key] = effective_mode
        st.session_state[exclusions_ss_key] = exclusions

    _apply_filter(
        preferences.playlists_selected,
        preferences.playlists_mode,
        all_playlists,
        SK.FILTER_PLAYLISTS,
        "_playlists_filter_mode",
        "_playlists_exclusions",
    )
    _apply_filter(
        preferences.modes_selected,
        preferences.modes_mode,
        all_modes,
        SK.FILTER_MODES,
        "_modes_filter_mode",
        "_modes_exclusions",
    )
    _apply_filter(
        preferences.maps_selected,
        preferences.maps_mode,
        all_maps,
        SK.FILTER_MAPS,
        "_maps_filter_mode",
        "_maps_exclusions",
    )
    _apply_filter(
        preferences.experience_types,
        preferences.experience_types_mode,
        all_experience_types,
        SK.FILTER_EXPERIENCE_TYPES,
        "_experience_types_filter_mode",
        "_experience_types_exclusions",
    )
    # Normalisation des labels d'expérience : si des labels stockés ne correspondent
    # pas aux options actuelles (ex : changement de langue FR→EN), on corrige.
    # "PVP non classé" + "PVP classé" (FR) ne matchent pas "Unranked PVP" + "Ranked PVP" (EN)
    # mais "PVE" est commun — sans normalisation, seul PVE resterait coché.
    _EXP_TYPES_TOTAL = 3  # Nombre fixe d'options (pvp_unranked, pvp_ranked, pve)
    if (
        all_experience_types
        and preferences.experience_types is not None
        and SK.FILTER_EXPERIENCE_TYPES in st.session_state
    ):
        stored_set = set(preferences.experience_types)
        current_set = set(all_experience_types)
        has_stale = bool(stored_set - current_set)
        if has_stale:
            valid = stored_set & current_set
            if len(stored_set) >= _EXP_TYPES_TOTAL:
                # Tous les types étaient sélectionnés → restaurer tous les types courants
                st.session_state[SK.FILTER_EXPERIENCE_TYPES] = current_set
            elif valid:
                # Sélection partielle avec certains labels valides
                st.session_state[SK.FILTER_EXPERIENCE_TYPES] = valid
            else:
                # Aucun label valide → sélectionner tout (fallback)
                st.session_state[SK.FILTER_EXPERIENCE_TYPES] = current_set


def clear_filter_preferences(xuid: str, db_path: str | None = None) -> None:
    """Supprime les préférences de filtres sauvegardées pour un joueur.

    Args:
        xuid: XUID ou gamertag du joueur.
        db_path: Chemin vers la base de données (optionnel).
    """
    player_key = _get_player_key(xuid, db_path)
    file_path = _get_filter_file_path(player_key)

    if file_path.exists():
        with contextlib.suppress(Exception):
            file_path.unlink()
