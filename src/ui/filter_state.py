"""Gestion de la persistance des filtres par joueur.

Ce module permet de sauvegarder et charger les filtres activés/désactivés
pour chaque joueur, afin d'améliorer l'UX en conservant les préférences
entre les sessions et les changements de joueur.

Il centralise aussi les clés session_state liées aux filtres pour garantir
un nettoyage exhaustif lors du changement de joueur.
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
    "gap_minutes",
    "_latest_session_label",
    "_trio_latest_session_label",
    "min_matches_maps",
    "_min_matches_maps_auto",
    "min_matches_maps_friends",
    "_min_matches_maps_friends_auto",
]

FILTER_WIDGET_KEY_PREFIXES: tuple[str, ...] = (
    "filter_playlists_",
    "filter_modes_",
    "filter_maps_",
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
    
    Les champs *_mode permettent de sauvegarder l'intention de l'utilisateur :
    - "exclude" : tout sauf les éléments listés (ex: "tout sauf Firefight")
    - "include" : uniquement les éléments listés (ex: "seulement Arène classée")
    
    Pour backward compatibility, l'absence de *_mode est interprété comme "include".
    """

    # Mode de filtre ("Période" ou "Sessions")
    filter_mode: str | None = None

    # Mode Période
    start_date: str | None = None  # Format ISO: "YYYY-MM-DD"
    end_date: str | None = None  # Format ISO: "YYYY-MM-DD"

    # Mode Sessions
    gap_minutes: int | None = None
    picked_session_label: str | None = None

    # Filtres cascade (listes de strings)
    playlists_selected: list[str] | None = None
    modes_selected: list[str] | None = None
    maps_selected: list[str] | None = None
    
    # Modes de filtrage (intent-based persistence)
    playlists_mode: str | None = None  # "exclude" ou "include"
    modes_mode: str | None = None      # "exclude" ou "include"
    maps_mode: str | None = None       # "exclude" ou "include"

    def to_dict(self) -> dict[str, Any]:
        """Convertit en dictionnaire pour sérialisation JSON."""
        return {k: v for k, v in asdict(self).items() if v is not None}

    @classmethod
    def from_dict(cls, data: dict[str, Any]) -> FilterPreferences:
        """Crée depuis un dictionnaire (désérialisation JSON)."""
        return cls(**{k: v for k, v in data.items() if k in cls.__dataclass_fields__})


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


def _detect_filter_mode(selected: set | list, all_options: set | list) -> str:
    """Détecte automatiquement le mode de filtrage (exclude vs include).
    
    Utilise un seuil avec zone d'hystérésis :
    - > 70% sélectionné → mode "exclude" (tout sauf X)
    - < 30% sélectionné → mode "include" (seulement Y)
    - Entre 30% et 70% → garde le mode actuel ou par défaut "include"
    
    Args:
        selected: Éléments sélectionnés.
        all_options: Tous les éléments disponibles.
        
    Returns:
        "exclude" ou "include"
    """
    if not all_options:
        return "include"
    
    selected_set = set(selected) if isinstance(selected, list) else selected
    all_set = set(all_options) if isinstance(all_options, list) else all_options
    
    if not selected_set:
        # Rien de sélectionné → mode "include" (par défaut)
        return "include"
    
    ratio = len(selected_set) / len(all_set)
    
    if ratio > 0.7:
        # Plus de 70% sélectionné → intention d'exclusion
        return "exclude"
    elif ratio < 0.3:
        # Moins de 30% sélectionné → intention d'inclusion
        return "include"
    else:
        # Zone grise : garder le comportement par défaut
        return "include"


def save_filter_preferences(
    xuid: str,
    db_path: str | None = None,
    preferences: FilterPreferences | None = None,
    all_playlists: list[str] | None = None,
    all_modes: list[str] | None = None,
    all_maps: list[str] | None = None,
) -> None:
    """Sauvegarde les préférences de filtres pour un joueur.

    Si preferences n'est pas fourni, lit depuis session_state.
    Détecte automatiquement le mode (exclude/include) basé sur le ratio de sélection.

    Args:
        xuid: XUID ou gamertag du joueur.
        db_path: Chemin vers la base de données (optionnel).
        preferences: Préférences à sauvegarder (optionnel, lit depuis session_state si None).
        all_playlists: Toutes les playlists disponibles (pour détection mode).
        all_modes: Tous les modes disponibles (pour détection mode).
        all_maps: Toutes les cartes disponibles (pour détection mode).
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

        # Filtres cascade avec détection automatique du mode
        playlists = st.session_state.get("filter_playlists")
        if isinstance(playlists, (set, list)):  # noqa: UP038
            playlists_set = set(playlists) if isinstance(playlists, list) else playlists
            
            # Détecter le mode si les options sont disponibles
            if all_playlists:
                mode = _detect_filter_mode(playlists_set, all_playlists)
                preferences.playlists_mode = mode
                
                # Sauvegarder selon le mode détecté
                if mode == "exclude":
                    # Mode exclusion : sauvegarder ce qui est EXCLU (non sélectionné)
                    excluded = set(all_playlists) - playlists_set
                    preferences.playlists_selected = sorted(excluded)
                else:
                    # Mode inclusion : sauvegarder ce qui est INCLUS (sélectionné)
                    preferences.playlists_selected = sorted(playlists_set)
            else:
                # Pas d'info sur les options → comportement legacy (include)
                preferences.playlists_selected = sorted(playlists_set)

        modes = st.session_state.get("filter_modes")
        if isinstance(modes, (set, list)):  # noqa: UP038
            modes_set = set(modes) if isinstance(modes, list) else modes
            
            if all_modes:
                mode = _detect_filter_mode(modes_set, all_modes)
                preferences.modes_mode = mode
                
                if mode == "exclude":
                    excluded = set(all_modes) - modes_set
                    preferences.modes_selected = sorted(excluded)
                else:
                    preferences.modes_selected = sorted(modes_set)
            else:
                preferences.modes_selected = sorted(modes_set)

        maps = st.session_state.get("filter_maps")
        if isinstance(maps, (set, list)):  # noqa: UP038
            maps_set = set(maps) if isinstance(maps, list) else maps
            
            if all_maps:
                mode = _detect_filter_mode(maps_set, all_maps)
                preferences.maps_mode = mode
                
                if mode == "exclude":
                    excluded = set(all_maps) - maps_set
                    preferences.maps_selected = sorted(excluded)
                else:
                    preferences.maps_selected = sorted(maps_set)
            else:
                preferences.maps_selected = sorted(maps_set)

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
    all_playlists: list[str] | None = None,
    all_modes: list[str] | None = None,
    all_maps: list[str] | None = None,
) -> None:
    """Applique les préférences de filtres dans session_state.

    Si preferences n'est pas fourni, charge depuis le fichier.
    Applique les préférences selon le mode (exclude/include) sauvegardé.

    Args:
        xuid: XUID ou gamertag du joueur.
        db_path: Chemin vers la base de données (optionnel).
        preferences: Préférences à appliquer (optionnel, charge depuis fichier si None).
        all_playlists: Toutes les playlists disponibles (pour mode exclude).
        all_modes: Tous les modes disponibles (pour mode exclude).
        all_maps: Toutes les cartes disponibles (pour mode exclude).
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

    # Filtres cascade avec gestion du mode exclude/include
    if preferences.playlists_selected is not None:
        saved_items = set(preferences.playlists_selected)
        mode = preferences.playlists_mode or "include"  # Default: include (backward compat)
        
        if mode == "exclude" and all_playlists:
            # Mode exclusion : appliquer tout sauf les éléments sauvegardés
            st.session_state["filter_playlists"] = set(all_playlists) - saved_items
        else:
            # Mode inclusion : appliquer les éléments sauvegardés
            # Si all_playlists fourni, filtrer pour ne garder que les valides
            if all_playlists:
                valid_items = saved_items & set(all_playlists)
                st.session_state["filter_playlists"] = valid_items
            else:
                st.session_state["filter_playlists"] = saved_items

    if preferences.modes_selected is not None:
        saved_items = set(preferences.modes_selected)
        mode = preferences.modes_mode or "include"
        
        if mode == "exclude" and all_modes:
            st.session_state["filter_modes"] = set(all_modes) - saved_items
        else:
            if all_modes:
                valid_items = saved_items & set(all_modes)
                st.session_state["filter_modes"] = valid_items
            else:
                st.session_state["filter_modes"] = saved_items

    if preferences.maps_selected is not None:
        saved_items = set(preferences.maps_selected)
        mode = preferences.maps_mode or "include"
        
        if mode == "exclude" and all_maps:
            st.session_state["filter_maps"] = set(all_maps) - saved_items
        else:
            if all_maps:
                valid_items = saved_items & set(all_maps)
                st.session_state["filter_maps"] = valid_items
            else:
                st.session_state["filter_maps"] = saved_items


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
