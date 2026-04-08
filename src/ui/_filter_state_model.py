"""Modèle et constantes pour la persistance des filtres.

Constants, FilterPreferences dataclass, helpers d'accès fichiers.
Extraits de filter_state.py pour respecter la limite de 500L.
"""

from __future__ import annotations

from dataclasses import asdict, dataclass
from pathlib import Path
from typing import Any

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
    # Clés intent-based (v5.2)
    "_playlists_filter_mode",
    "_modes_filter_mode",
    "_maps_filter_mode",
    "_experience_types_filter_mode",
    "_playlists_exclusions",
    "_modes_exclusions",
    "_maps_exclusions",
    "_experience_types_exclusions",
    # Clé shadow non-widget (Streamlit 1.54+)
    "_filter_mode_shadow",
    "_picked_session_label_shadow",
    "_picked_sessions_shadow",
    "_start_date_cal_shadow",
    "_end_date_cal_shadow",
    # Solo / escouade (v5.3)
    "picked_solo_session_label",
    "picked_squad_session_label",
    "_picked_solo_session_label_shadow",
    "_picked_squad_session_label_shadow",
    # Amis sélectionnés dans l'onglet Teammates (v5.3)
    "teammates_picked_labels",
]

FILTER_WIDGET_KEY_PREFIXES: tuple[str, ...] = (
    "filter_playlists_",
    "filter_modes_",
    "filter_maps_",
    "filter_experience_types_",
)


def get_all_filter_keys_to_clear(session_state: dict) -> list[str]:
    """Retourne toutes les clés de filtres à supprimer lors du changement de joueur."""
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

    Architecture intent-based (v5.2) :
        Les champs ``*_mode`` indiquent comment interpréter les listes ``*_selected``.
        - ``"exclude"`` : la liste contient ce qui est DÉCOCHÉ (exclusions).
        - ``"include"`` (ou None/legacy) : la liste contient ce qui est COCHÉ.
    """

    filter_mode: str | None = None
    start_date: str | None = None
    end_date: str | None = None
    gap_minutes: int | None = None
    picked_session_label: str | None = None
    latest_session_label: str | None = None
    picked_solo_session_label: str | None = None
    picked_squad_session_label: str | None = None
    friends_selected_labels: list[str] | None = None
    playlists_selected: list[str] | None = None
    modes_selected: list[str] | None = None
    maps_selected: list[str] | None = None
    playlists_mode: str | None = None
    modes_mode: str | None = None
    maps_mode: str | None = None
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


def detect_filter_mode(
    selected: set[str] | list[str],
    all_options: set[str] | list[str],
    current_mode: str = "include",
) -> str:
    """Détecte si l'utilisateur est en mode inclusion ou exclusion.

    Heuristique : >70% cochés → exclude, <30% → include, entre → garde mode actuel.
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


def get_filters_dir() -> Path:
    """Retourne le répertoire pour stocker les filtres."""
    project_root = Path(__file__).parent.parent.parent
    filters_dir = project_root / ".streamlit" / "filter_preferences"
    filters_dir.mkdir(parents=True, exist_ok=True)
    return filters_dir


def get_player_key(xuid: str, db_path: str | None = None) -> str:
    """Génère une clé unique pour identifier un joueur."""
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
    return f"xuid_{xuid}"


def get_filter_file_path(player_key: str) -> Path:
    """Retourne le chemin du fichier de filtres pour un joueur."""
    filters_dir = get_filters_dir()
    safe_key = player_key.replace("/", "_").replace("\\", "_").replace(":", "_")
    return filters_dir / f"{safe_key}.json"


def get_player_prefs_path(xuid: str, db_path: str | None = None) -> Path | None:
    """Retourne data/players/{gamertag}/ui_prefs.json si la DB est dans un répertoire joueur.

    Retourne None si le répertoire n'existe pas ou si db_path est absent.
    Ce chemin est dans le volume Docker monté (persistant entre rebuilds).
    """
    if db_path:
        gamertag_dir = Path(db_path).parent
        if gamertag_dir.parent.name == "players" and gamertag_dir.exists():
            return gamertag_dir / "ui_prefs.json"
    return None
