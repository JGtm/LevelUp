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
    V7_CURRENT_SECTION: str = "v7_current_section"
    V7_STATS_VIEW: str = "v7_stats_view"
    V7_PROFILE_VIEW: str = "v7_profile_view"
    PENDING_PAGE: str = "_pending_page"
    PENDING_MATCH_ID: str = "_pending_match_id"
    PENDING_GAMERTAG: str = "_pending_gamertag"
    MATCH_ID_INPUT: str = "match_id_input"
    LAST_MATCH_NAV_INDEX: str = "_last_match_nav_index"
    LAST_MATCH_NAV_TOTAL: str = "_last_match_nav_total"
    LAST_MATCH_NAV_SESSION_KEY: str = "_last_match_nav_session_key"

    # ------------------------------------------------------------------
    # Cycle de vie application
    # ------------------------------------------------------------------
    CACHE_BUSTER: str = "_cache_buster"
    IS_SYNCING: str = "_is_syncing"
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

    # ------------------------------------------------------------------
    # Filtres cascade (multi-select maps/modes/playlists)
    # ------------------------------------------------------------------
    FILTER_MAPS: str = "filter_maps"
    FILTER_MODES: str = "filter_modes"
    FILTER_PLAYLISTS: str = "filter_playlists"
    FILTER_EXPERIENCE_TYPES: str = "filter_experience_types"
    FILTER_MODE_SHADOW: str = "_filter_mode_shadow"
    PENDING_FILTER_MODE: str = "_pending_filter_mode"
    SHOW_DEBUG_INFO: str = "_show_debug_info"

    # ------------------------------------------------------------------
    # Sessions filtrées (solo / escouade)
    # ------------------------------------------------------------------
    PICKED_SESSION_LABEL: str = "picked_session_label"
    PICKED_SOLO_SESSION_LABEL: str = "picked_solo_session_label"
    PICKED_SQUAD_SESSION_LABEL: str = "picked_squad_session_label"
    TEAMMATES_PICKED_LABELS: str = "teammates_picked_labels"
    PICKED_SESSIONS_SHADOW: str = "_picked_sessions_shadow"

    # ------------------------------------------------------------------
    # Seuils automatiques et initialisation session
    # ------------------------------------------------------------------
    MIN_MATCHES_MAPS_AUTO: str = "_min_matches_maps_auto"
    MIN_MATCHES_MAPS_FRIENDS_AUTO: str = "_min_matches_maps_friends_auto"
    APP_SESSION_INIT_DONE: str = "_app_session_init_done"

    # ------------------------------------------------------------------
    # OAuth Xbox — Device Code Flow
    # ------------------------------------------------------------------
    XBOX_OAUTH_RESULT: str = "_xbox_oauth_result"
    DC_FLOW: str = "_dc_flow"
    DC_MSAL_APP: str = "_dc_msal_app"
    DC_RESULT_QUEUE: str = "_dc_result_queue"
    DC_CLIENT_ID: str = "_dc_client_id"
