"""Gestion centralisée du state de l'application.

Ce module centralise :
- L'initialisation du session_state Streamlit
- La gestion des identités par défaut (XUID, gamertag, waypoint)
- Les clés de cache pour invalidation
"""

from __future__ import annotations

import os
from collections.abc import Mapping
from dataclasses import dataclass, field
from typing import TYPE_CHECKING

import streamlit as st

from src.app.session_keys import SK
from src.config import (
    DEFAULT_PLAYER_GAMERTAG,
    DEFAULT_PLAYER_XUID,
    DEFAULT_WAYPOINT_PLAYER,
)

if TYPE_CHECKING:
    from src.ui.settings import AppSettings


@dataclass
class PlayerIdentity:
    """Identité d'un joueur (XUID, gamertag, waypoint)."""

    xuid_or_gamertag: str = ""
    xuid_fallback: str = ""
    waypoint_player: str = ""

    @property
    def display_name(self) -> str:
        """Retourne le nom d'affichage préféré."""
        return self.xuid_or_gamertag or self.xuid_fallback or "Joueur"

    @property
    def xuid(self) -> str:
        """Retourne le XUID si disponible."""
        if self.xuid_or_gamertag and self.xuid_or_gamertag.isdigit():
            return self.xuid_or_gamertag
        return self.xuid_fallback


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

    @classmethod
    def from_session(cls) -> AppState:
        """Charge l'état depuis session_state."""
        return cls(
            db_path=str(st.session_state.get("db_path", "") or ""),
            xuid_input=str(st.session_state.get("xuid_input", "") or ""),
            waypoint_player=str(st.session_state.get("waypoint_player", "") or ""),
            filter_playlists=list(st.session_state.get("filter_playlists", []) or []),
            filter_modes=list(st.session_state.get("filter_modes", []) or []),
            filter_maps=list(st.session_state.get("filter_maps", []) or []),
            filter_sessions=list(st.session_state.get("filter_sessions", []) or []),
            current_page=str(st.session_state.get("current_page", "Accueil") or "Accueil"),
            pending_page=st.session_state.get("_pending_page"),
            pending_match_id=st.session_state.get("_pending_match_id"),
        )

    def save_to_session(self) -> None:
        """Sauvegarde l'état dans session_state."""
        st.session_state[SK.DB_PATH] = self.db_path
        st.session_state[SK.XUID_INPUT] = self.xuid_input
        st.session_state[SK.WAYPOINT_PLAYER] = self.waypoint_player
        st.session_state["filter_playlists"] = self.filter_playlists
        st.session_state["filter_modes"] = self.filter_modes
        st.session_state["filter_maps"] = self.filter_maps
        st.session_state["filter_sessions"] = self.filter_sessions
        st.session_state[SK.CURRENT_PAGE] = self.current_page
        if self.pending_page:
            st.session_state[SK.PENDING_PAGE] = self.pending_page
        if self.pending_match_id:
            st.session_state[SK.PENDING_MATCH_ID] = self.pending_match_id

    def clear_filters(self) -> None:
        """Réinitialise tous les filtres."""
        self.filter_playlists = []
        self.filter_modes = []
        self.filter_maps = []
        self.filter_sessions = []
        # Nettoie aussi session_state directement
        for key in ["filter_playlists", "filter_modes", "filter_maps", "filter_sessions"]:
            if key in st.session_state:
                del st.session_state[key]


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


def get_default_identity() -> PlayerIdentity:
    """Retourne l'identité par défaut depuis secrets/env/constants.

    Ordre de priorité :
    1. Secrets Streamlit (.streamlit/secrets.toml)
    2. Variables d'environnement
    3. Constantes du module config
    """
    # Secrets Streamlit
    try:
        player = st.secrets.get("player", {})
        if isinstance(player, Mapping):
            gt = str(player.get("gamertag") or "").strip()
            xu = str(player.get("xuid") or "").strip()
            wp = str(player.get("waypoint_player") or "").strip()
        else:
            gt = xu = wp = ""
    except Exception:
        gt = xu = wp = ""

    # Variables d'environnement
    gt = gt or str(os.environ.get("LEVELUP_DEFAULT_GAMERTAG") or "").strip()
    xu = xu or str(os.environ.get("LEVELUP_DEFAULT_XUID") or "").strip()
    wp = wp or str(os.environ.get("LEVELUP_DEFAULT_WAYPOINT_PLAYER") or "").strip() or gt

    # Constantes fallback
    gt = gt or str(DEFAULT_PLAYER_GAMERTAG or "").strip()
    xu = xu or str(DEFAULT_PLAYER_XUID or "").strip()
    wp = wp or str(DEFAULT_WAYPOINT_PLAYER or "").strip() or gt

    return PlayerIdentity(
        xuid_or_gamertag=gt or xu,
        xuid_fallback=xu,
        waypoint_player=wp,
    )


def _infer_gamertag_from_v5_path(db_path: str) -> str:
    """Extrait le gamertag depuis une DB v5 LevelUp.

    Convention : data/players/{gamertag}/stats.duckdb
    Retourne une chaîne vide si la structure ne correspond pas.
    """
    from pathlib import Path

    p = Path(db_path)
    if p.name == "stats.duckdb" and p.parent.parent.name.lower() == "players":
        return p.parent.name
    return ""


def _read_xuid_from_sync_meta(db_path: str) -> str:
    """Lit le XUID depuis sync_meta (stocké par engine après chaque sync).

    C'est la source canonique : engine._update_sync_meta("xuid", xuid)
    est appelé à chaque sync réussi.
    Retourne une chaîne vide si la table n'existe pas ou le XUID absent.
    """
    import os

    if not db_path or not os.path.exists(db_path):
        return ""
    try:
        from src.utils.db import duckdb_read_only

        with duckdb_read_only(db_path) as conn:
            row = conn.execute("SELECT value FROM sync_meta WHERE key = 'xuid' LIMIT 1").fetchone()
            return str(row[0]).strip() if row and row[0] else ""
    except Exception:
        return ""


def init_source_state(default_db: str, settings: AppSettings) -> None:
    """Initialise le session_state avec les valeurs par défaut.

    Args:
        default_db: Chemin par défaut de la DB.
        settings: Paramètres de l'application.
    """
    from src.ui.sync import pick_latest_spnkr_db_if_any

    # DB path
    if "db_path" not in st.session_state:
        chosen = str(default_db or "")

        # Override via env ?
        forced_env_db = str(
            os.environ.get("LEVELUP_DB") or os.environ.get("LEVELUP_DB_PATH") or ""
        ).strip()

        # Auto-sélection SPNKr si préféré
        if (not forced_env_db) and settings.prefer_spnkr_db_if_available:
            spnkr = pick_latest_spnkr_db_if_any()
            if spnkr and os.path.exists(spnkr) and os.path.getsize(spnkr) > 0:
                chosen = spnkr

        st.session_state[SK.DB_PATH] = chosen

    # XUID input — lire depuis sync_meta (source canonique post-sync).
    # Fallback : gamertag depuis le chemin si sync_meta vide (premier lancement).
    if "xuid_input" not in st.session_state:
        db_candidate = str(st.session_state.get("db_path", "") or "")
        xuid = _read_xuid_from_sync_meta(db_candidate)
        if not xuid:
            xuid = _infer_gamertag_from_v5_path(db_candidate)
        st.session_state[SK.XUID_INPUT] = xuid

    # Waypoint player
    if "waypoint_player" not in st.session_state:
        identity = get_default_identity()
        st.session_state[SK.WAYPOINT_PLAYER] = identity.waypoint_player


def get_db_cache_key(db_path: str) -> tuple[int, int, int, int] | None:
    """Retourne une signature stable de la DB pour invalider les caches.

    Délègue à db_cache_key() canonique (cache_loaders.py).
    Surveille player DB + shared_matches.duckdb.

    Args:
        db_path: Chemin vers la base de données joueur.

    Returns:
        Tuple (mtime_ns_player, size_player, mtime_ns_shared, size_shared) ou None.
    """
    from src.ui.cache_loaders import db_cache_key

    return db_cache_key(db_path)


def get_aliases_cache_key() -> int | None:
    """Retourne toujours None depuis v5.2 (plus de fichier xuid_aliases.json).

    Les aliases sont désormais exclusivement dans shared_matches.duckdb.
    L'invalidation de cache se fait via db_cache_key() sur la DB.
    """
    return None


def propagate_env_defaults() -> None:
    """Propage les identités par défaut vers les variables d'environnement.

    Utile pour que les modules downstream puissent résoudre un XUID
    quand la DB SPNKr ne contient pas les gamertags.
    """
    identity = get_default_identity()

    if (
        identity.xuid_or_gamertag
        and not str(identity.xuid_or_gamertag).strip().isdigit()
        and identity.xuid_fallback
    ):
        if not str(os.environ.get("LEVELUP_DEFAULT_GAMERTAG") or "").strip():
            os.environ["LEVELUP_DEFAULT_GAMERTAG"] = str(identity.xuid_or_gamertag).strip()
        if not str(os.environ.get("LEVELUP_DEFAULT_XUID") or "").strip():
            os.environ["LEVELUP_DEFAULT_XUID"] = str(identity.xuid_fallback).strip()

    if (
        identity.waypoint_player
        and not str(os.environ.get("LEVELUP_DEFAULT_WAYPOINT_PLAYER") or "").strip()
    ):
        os.environ["LEVELUP_DEFAULT_WAYPOINT_PLAYER"] = str(identity.waypoint_player).strip()


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
