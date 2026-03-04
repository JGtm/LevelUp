"""Constantes des clés Streamlit session_state partagées entre modules.

Centralise les clés *publiques* (utilisées dans plusieurs modules).
Les clés internes à un seul module (ex: ``_filter_mode_shadow`` dans
``filters_render.py``) restent des strings locales dans leur module.

Usage::

    from src.app.session_keys import SK

    if SK.DB_PATH not in st.session_state:
        st.session_state[SK.DB_PATH] = ""

"""

from __future__ import annotations


class SK:
    """Clés session_state Streamlit (namespace court, usage : ``SK.DB_PATH``)."""

    # ------------------------------------------------------------------
    # Identité / connexion
    # ------------------------------------------------------------------
    DB_PATH: str = "db_path"
    XUID_INPUT: str = "xuid_input"
    WAYPOINT_PLAYER: str = "waypoint_player"
    LANG: str = "lang"
    APP_SETTINGS: str = "app_settings"

    # ------------------------------------------------------------------
    # Navigation
    # ------------------------------------------------------------------
    CURRENT_PAGE: str = "current_page"
    PENDING_PAGE: str = "_pending_page"
    PENDING_MATCH_ID: str = "_pending_match_id"
    MATCH_ID_INPUT: str = "match_id_input"

    # ------------------------------------------------------------------
    # Cycle de vie application
    # ------------------------------------------------------------------
    CACHE_BUSTER: str = "_cache_buster"
    STARTUP_WARNINGS: str = "_startup_cfg_warnings"
    STARTUP_ERRORS: str = "_startup_cfg_errors"
    SECRETS_LOADED: str = "_secrets_loaded"
    TAILSCALE_STARTED: str = "_tailscale_started"
    CONSUMED_QUERY_PARAMS: str = "_consumed_query_params"

    # ------------------------------------------------------------------
    # Filtres (inter-modules : filters_render ↔ filter_state ↔ pages)
    # ------------------------------------------------------------------
    FILTER_MODE: str = "filter_mode"
    START_DATE: str = "start_date_cal"
    END_DATE: str = "end_date_cal"
    PICKED_SESSIONS: str = "picked_sessions"
    GAP_MINUTES: str = "gap_minutes"
    MIN_MATCHES_MAPS: str = "min_matches_maps"
    MIN_MATCHES_MAPS_FRIENDS: str = "min_matches_maps_friends"
