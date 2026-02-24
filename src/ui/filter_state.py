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
from dataclasses import asdict, dataclass
from datetime import date
from pathlib import Path
from typing import Any

import streamlit as st

# ---------------------------------------------------------------------------
# Clés session_state liées aux filtres (centralisées pour nettoyage exhaustif)
# ---------------------------------------------------------------------------

FILTER_DATA_KEYS: list[str] = [
    "filter_mode",
    "start_date_cal",
    "end_date_cal",
    "picked_session_label",
    "picked_sessions",
    "filter_playlists",
    "filter_modes",
    "filter_maps",
    "filter_experience_types",
    "gap_minutes",
    "_latest_session_label",
    "_trio_latest_session_label",
    "min_matches_maps",
    "_min_matches_maps_auto",
    "min_matches_maps_friends",
    "_min_matches_maps_friends_auto",
    # Clés intent-based (v5.2) — mode et exclusions pour réconciliation mid-session
    "_playlists_filter_mode",
    "_modes_filter_mode",
    "_maps_filter_mode",
    "_experience_types_filter_mode",
    "_playlists_exclusions",
    "_modes_exclusions",
    "_maps_exclusions",
    "_experience_types_exclusions",
    # Clé shadow non-widget pour résistance au widget-state clearing de Streamlit 1.54+
    # (st.navigation + st.switch_page peut effacer les clés de widgets comme filter_mode)
    "_filter_mode_shadow",
    "_picked_session_label_shadow",
    "_picked_sessions_shadow",
    "_start_date_cal_shadow",
    "_end_date_cal_shadow",
]

FILTER_WIDGET_KEY_PREFIXES: tuple[str, ...] = (
    "filter_playlists_",
    "filter_modes_",
    "filter_maps_",
    "filter_experience_types_",
)


def get_all_filter_keys_to_clear(session_state: dict) -> list[str]:
    """Retourne toutes les clés de filtres à supprimer lors du changement de joueur.

    Inclut les clés de données explicites et toutes les clés de widgets
    dont le nom commence par un préfixe connu.

    Args:
        session_state: Le dictionnaire session_state Streamlit.

    Returns:
        Liste de clés à supprimer.
    """
    keys: list[str] = [k for k in FILTER_DATA_KEYS if k in session_state]
    keys.extend(
        k
        for k in session_state
        if any(k.startswith(prefix) for prefix in FILTER_WIDGET_KEY_PREFIXES)
    )
    return keys


@dataclass
class FilterPreferences:
    """Préférences de filtres pour un joueur.

    Toutes les valeurs sont optionnelles pour permettre une migration progressive.

    Architecture intent-based (v5.2) :
        Les champs ``*_mode`` indiquent comment interpréter les listes ``*_selected``.
        - ``"exclude"`` : la liste contient ce qui est DÉCOCHÉ (exclusions).
          Au chargement, ``all_options - exclusions`` = ce qui doit être coché.
          Les nouvelles options sont auto-cochées car absentes des exclusions.
        - ``"include"`` (ou None/legacy) : la liste contient ce qui est COCHÉ.
          Compatibilité ascendante : les JSON sans ``*_mode`` sont traités comme include.
    """

    # Mode de filtre ("Période" ou "Sessions")
    filter_mode: str | None = None

    # Mode Période
    start_date: str | None = None  # Format ISO: "YYYY-MM-DD"
    end_date: str | None = None  # Format ISO: "YYYY-MM-DD"

    # Mode Sessions
    gap_minutes: int | None = None
    picked_session_label: str | None = None
    # Label de la dernière session au moment de la sauvegarde.
    # Si picked_session_label == latest_session_label, l'utilisateur suivait
    # la session la plus récente → on réinitialise sur la vraie dernière au prochain chargement.
    latest_session_label: str | None = None

    # Filtres cascade (listes de strings)
    # En mode "exclude" : contient ce qui est DÉCOCHÉ
    # En mode "include" (ou None/legacy) : contient ce qui est COCHÉ
    playlists_selected: list[str] | None = None
    modes_selected: list[str] | None = None
    maps_selected: list[str] | None = None

    # Mode d'interprétation des listes ci-dessus (v5.2)
    # "exclude" | "include" | None (None = legacy = traiter comme "include")
    playlists_mode: str | None = None
    modes_mode: str | None = None
    maps_mode: str | None = None

    # Sélecteur type d'expérience (v5.2) — liste statique PVP/PVE
    # En mode "exclude" : contient les types décochés
    # En mode "include" : contient les types cochés
    experience_types: list[str] | None = None
    experience_types_mode: str | None = None

    def to_dict(self) -> dict[str, Any]:
        """Convertit en dictionnaire pour sérialisation JSON."""
        return {k: v for k, v in asdict(self).items() if v is not None}

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> FilterPreferences:
        """Crée depuis un dictionnaire (désérialisation JSON)."""
        return cls(**{k: v for k, v in data.items() if k in cls.__dataclass_fields__})


# ---------------------------------------------------------------------------
# Heuristique de détection du mode include/exclude
# ---------------------------------------------------------------------------


def _detect_filter_mode(
    selected: set[str] | list[str],
    all_options: set[str] | list[str],
    current_mode: str = "include",
) -> str:
    """Détecte si l'utilisateur est en mode inclusion ou exclusion.

    Heuristique :
    - >70% cochés → exclude (on stocke les quelques décochés)
    - <30% cochés → include (on stocke les quelques cochés)
    - entre 30-70% → garde le mode actuel (hystérésis)

    Args:
        selected: Valeurs actuellement cochées.
        all_options: Toutes les valeurs disponibles.
        current_mode: Mode actuel (tie-break zone grise).

    Returns:
        "exclude" ou "include"
    """
    if not all_options:
        return "include"
    ratio = len(set(selected)) / len(set(all_options))
    if ratio > 0.7:
        return "exclude"
    elif ratio < 0.3:
        return "include"
    else:
        return current_mode


# ---------------------------------------------------------------------------
# Helpers d'accès aux fichiers
# ---------------------------------------------------------------------------


def _get_filters_dir() -> Path:
    """Retourne le répertoire pour stocker les filtres."""
    project_root = Path(__file__).parent.parent.parent
    filters_dir = project_root / ".streamlit" / "filter_preferences"
    filters_dir.mkdir(parents=True, exist_ok=True)
    return filters_dir


def _get_player_key(xuid: str, db_path: str | None = None) -> str:
    """Génère une clé unique pour identifier un joueur.

    Pour DuckDB v4, utilise le gamertag depuis le chemin.
    Pour les autres cas, utilise le xuid.

    Args:
        xuid: XUID ou gamertag du joueur.
        db_path: Chemin vers la base de données (optionnel).

    Returns:
        Clé unique pour le joueur.
    """
    # Si c'est un chemin DuckDB v4 (data/players/{gamertag}/stats.duckdb),
    # extraire le gamertag
    if db_path:
        db_path_obj = Path(db_path)
        if "players" in db_path_obj.parts:
            try:
                players_idx = db_path_obj.parts.index("players")
                if players_idx + 1 < len(db_path_obj.parts):
                    gamertag = db_path_obj.parts[players_idx + 1]
                    return f"player_{gamertag}"
            except (ValueError, IndexError):
                pass

    # Sinon, utiliser le xuid
    return f"xuid_{xuid}"


def _get_filter_file_path(player_key: str) -> Path:
    """Retourne le chemin du fichier de filtres pour un joueur."""
    filters_dir = _get_filters_dir()
    # Nettoyer la clé pour éviter les caractères invalides dans les noms de fichiers
    safe_key = player_key.replace("/", "_").replace("\\", "_").replace(":", "_")
    return filters_dir / f"{safe_key}.json"


# ---------------------------------------------------------------------------
# Save / Load / Apply / Clear
# ---------------------------------------------------------------------------


def save_filter_preferences(
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
        if isinstance(gap_minutes_val, (int, float)):  # noqa: UP038
            preferences.gap_minutes = int(gap_minutes_val)
        picked_session_label_val = st.session_state.get("picked_session_label")
        if isinstance(picked_session_label_val, str):
            preferences.picked_session_label = picked_session_label_val
        # Sauvegarder aussi la "vraie dernière session" pour détecter le tracking
        latest_session_label_val = st.session_state.get("_latest_session_label")
        if isinstance(latest_session_label_val, str):
            preferences.latest_session_label = latest_session_label_val

        # Filtres cascade — logique intent-based
        # Mapping session_key → exclusions_key pour mise à jour mid-session
        _EXCLUSIONS_KEY_MAP: dict[str, str] = {
            "filter_playlists": "_playlists_exclusions",
            "filter_modes": "_modes_exclusions",
            "filter_maps": "_maps_exclusions",
            "filter_experience_types": "_experience_types_exclusions",
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

        pl_list, pl_mode = _save_filter("filter_playlists", "_playlists_filter_mode", all_playlists)
        if pl_list is not None:
            preferences.playlists_selected = pl_list
            preferences.playlists_mode = pl_mode

        mo_list, mo_mode = _save_filter("filter_modes", "_modes_filter_mode", all_modes)
        if mo_list is not None:
            preferences.modes_selected = mo_list
            preferences.modes_mode = mo_mode

        ma_list, ma_mode = _save_filter("filter_maps", "_maps_filter_mode", all_maps)
        if ma_list is not None:
            preferences.maps_selected = ma_list
            preferences.maps_mode = ma_mode

        exp_list, exp_mode = _save_filter(
            "filter_experience_types", "_experience_types_filter_mode", all_experience_types
        )
        if exp_list is not None:
            preferences.experience_types = exp_list
            preferences.experience_types_mode = exp_mode

    # Sauvegarder dans le fichier
    player_key = _get_player_key(xuid, db_path)
    file_path = _get_filter_file_path(player_key)

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


def apply_filter_preferences(
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

    if preferences.picked_session_label:
        st.session_state["picked_session_label"] = preferences.picked_session_label
        # Mettre à jour picked_sessions aussi
        if preferences.picked_session_label != "(toutes)":
            st.session_state["picked_sessions"] = [preferences.picked_session_label]
        else:
            st.session_state["picked_sessions"] = []

    # Filtres cascade — logique intent-based
    def _apply_filter(
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
        "filter_playlists",
        "_playlists_filter_mode",
        "_playlists_exclusions",
    )
    _apply_filter(
        preferences.modes_selected,
        preferences.modes_mode,
        all_modes,
        "filter_modes",
        "_modes_filter_mode",
        "_modes_exclusions",
    )
    _apply_filter(
        preferences.maps_selected,
        preferences.maps_mode,
        all_maps,
        "filter_maps",
        "_maps_filter_mode",
        "_maps_exclusions",
    )
    _apply_filter(
        preferences.experience_types,
        preferences.experience_types_mode,
        all_experience_types,
        "filter_experience_types",
        "_experience_types_filter_mode",
        "_experience_types_exclusions",
    )


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
