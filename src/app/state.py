"""Gestion centralisée du state de l'application.

Ce module centralise :
- La classe AppState (état global typé)
- get_page_context() pour les pages UI
- apply_settings_path_overrides() pour les chemins configurés
"""

from __future__ import annotations

import os
from dataclasses import dataclass, field
from typing import TYPE_CHECKING

import streamlit as st

if TYPE_CHECKING:
    from src.ui.settings import AppSettings


@dataclass
class AppState:
    """État global de l'application.

    Centralise l'accès au session_state Streamlit avec typage.
    """

    db_path: str = ""
    xuid_input: str = ""
    waypoint_player: str = ""

    # Filtres actifs
    filter_playlists: list[str] = field(default_factory=list)
    filter_modes: list[str] = field(default_factory=list)
    filter_maps: list[str] = field(default_factory=list)
    filter_sessions: list[str] = field(default_factory=list)

    # Navigation
    current_page: str = "Accueil"
    pending_page: str | None = None
    pending_match_id: str | None = None


def get_page_context() -> tuple[str, str, str]:
    """Retourne le contexte courant (db_path, xuid, waypoint_player) depuis session_state.

    Centralise les accès ``st.session_state.get("db_path")`` etc.
    dispersés dans les pages UI.

    Returns:
        Tuple (db_path, xuid, waypoint_player) — chaînes vides si absent.
    """
    db_path = str(st.session_state.get("db_path", "") or "")
    # Résolution du XUID : player_xuid (résolu) > xuid > xuid_input (brut)
    xuid = str(
        st.session_state.get("player_xuid")
        or st.session_state.get("xuid")
        or st.session_state.get("xuid_input", "")
        or ""
    ).strip()
    wp = str(st.session_state.get("waypoint_player", "") or "")
    return db_path, xuid, wp


def apply_settings_path_overrides(settings: AppSettings) -> None:
    """Applique les overrides de chemins depuis les paramètres.

    Args:
        settings: Paramètres de l'application.
    """
    # Aliases path
    aliases_override = settings.aliases_path.strip()
    if aliases_override:
        os.environ["LEVELUP_ALIASES_PATH"] = aliases_override
    else:
        os.environ.pop("LEVELUP_ALIASES_PATH", None)

    # Profiles path
    profiles_override = settings.profiles_path.strip()
    if profiles_override:
        os.environ["LEVELUP_PROFILES_PATH"] = profiles_override
    else:
        os.environ.pop("LEVELUP_PROFILES_PATH", None)
