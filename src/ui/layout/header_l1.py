"""Shell global L1 pour la v7."""

from __future__ import annotations

import contextlib
import logging
from pathlib import Path

import streamlit as st

from src.app.page_router import V7_SECTION_KEYS, get_v7_section_label, normalize_v7_section
from src.app.session_keys import SK
from src.ui import display_name_from_xuid
from src.ui._sync_indicator import render_sync_indicator
from src.ui.components.browser_storage import persist_browser_prefs
from src.ui.filter_state import (
    _get_player_key,
    apply_filter_preferences,
    get_all_filter_keys_to_clear,
    save_filter_preferences,
)
from src.ui.i18n import t
from src.ui.multiplayer import (
    get_gamertag_from_duckdb_v4_path,
    is_duckdb_v4_path,
    list_duckdb_v4_players,
    render_player_selector_unified,
)

logger = logging.getLogger(__name__)


def _has_multiple_players(db_path: str) -> bool:
    """Retourne True si plusieurs profils joueurs sont disponibles."""
    return is_duckdb_v4_path(db_path) and len(list_duckdb_v4_players()) > 1


def _reset_player_filters(old_xuid: str, old_db_path: str) -> None:
    """Sauvegarde puis réinitialise les filtres lors d'un changement de joueur."""
    with contextlib.suppress(Exception):
        save_filter_preferences(old_xuid, old_db_path)

    cleared_filters = 0
    for filter_key in get_all_filter_keys_to_clear(st.session_state):
        if filter_key in st.session_state:
            del st.session_state[filter_key]
            cleared_filters += 1

    old_player_key = _get_player_key(old_xuid, old_db_path)
    cleared_player_state = 0
    for state_key in (f"_filters_loaded_{old_player_key}", f"_last_saved_player_{old_player_key}"):
        if state_key in st.session_state:
            del st.session_state[state_key]
            cleared_player_state += 1

    logger.info(
        "Reset filtres joueur V7: xuid=%s filtres=%s etat_joueur=%s",
        old_xuid,
        cleared_filters,
        cleared_player_state,
    )


def _apply_player_change(
    *,
    current_db_path: str,
    current_xuid: str,
    new_db_path: str | None,
    new_xuid: str | None,
) -> tuple[str, str]:
    """Applique un changement de joueur et rerun l'application."""
    db_path = current_db_path
    xuid = current_xuid
    _reset_player_filters(current_xuid, current_db_path)

    target_player = new_xuid or current_xuid
    if new_db_path:
        st.session_state[SK.DB_PATH] = new_db_path
        db_path = new_db_path
        gamertag = get_gamertag_from_duckdb_v4_path(new_db_path)
        if gamertag:
            st.session_state[SK.XUID_INPUT] = gamertag
            st.session_state[SK.WAYPOINT_PLAYER] = gamertag
            xuid = gamertag
            target_player = gamertag
            player_slug = Path(new_db_path).parent.name
            persist_browser_prefs(last_gamertag=player_slug, last_db_path=player_slug)

    if new_xuid:
        st.session_state[SK.XUID_INPUT] = new_xuid
        xuid = new_xuid
        target_player = new_xuid

    logger.info(
        "Navigation joueur V7: %s -> %s",
        current_xuid or "<vide>",
        target_player or "<vide>",
    )

    apply_filter_preferences(xuid, db_path)
    return db_path, xuid


def _render_player_block(db_path: str, xuid: str) -> tuple[str, str]:
    """Rend le bloc joueur actif / sélecteur multi-joueurs."""
    st.markdown(f"<div class='v7-brand-title'>{t('col_player')}</div>", unsafe_allow_html=True)

    if _has_multiple_players(db_path):
        new_db_path, new_xuid = render_player_selector_unified(
            db_path,
            xuid,
            key="v7_header_player_selector",
            show_heading=False,
            show_sync_indicator=False,
        )
        if new_db_path or new_xuid:
            db_path, xuid = _apply_player_change(
                current_db_path=db_path,
                current_xuid=xuid,
                new_db_path=new_db_path,
                new_xuid=new_xuid,
            )
            st.rerun()
        return db_path, xuid

    player_name = (
        display_name_from_xuid(xuid.strip(), db_path=db_path) if str(xuid or "").strip() else "-"
    )
    st.markdown(
        f"<div class='v7-active-player'>{player_name}</div>",
        unsafe_allow_html=True,
    )
    return db_path, xuid


def render_header_l1(*, db_path: str, xuid: str) -> tuple[str, str, str]:
    """Affiche le shell global L1 et retourne l'etat de navigation."""
    current_section = normalize_v7_section(st.session_state.get(SK.V7_CURRENT_SECTION))
    initial_section = current_section

    brand_col, nav_col, player_col, sync_col = st.columns([0.9, 2.7, 1.7, 1.1])
    with brand_col:
        st.markdown("<div class='v7-brand-title'>LevelUp</div>", unsafe_allow_html=True)
        if st.button("LevelUp", key="v7_brand_home", width="stretch"):
            current_section = "home"
            st.session_state[SK.V7_CURRENT_SECTION] = current_section
            if initial_section != current_section:
                logger.info(
                    "Navigation V7: section %s -> %s via brand", initial_section, current_section
                )
            pages_dict = st.session_state.get("_v7_pages", {})
            target = pages_dict.get("home")
            if target is not None:
                st.switch_page(target)

    with nav_col:
        labels = {section: get_v7_section_label(section) for section in V7_SECTION_KEYS}
        selected_label = st.segmented_control(
            "Sections",
            options=list(labels.values()),
            default=labels[current_section],
            key="v7_header_nav_widget",
            label_visibility="collapsed",
            width="stretch",
        )
        reverse_map = {label: section for section, label in labels.items()}
        current_section = reverse_map.get(selected_label, current_section)
        st.session_state[SK.V7_CURRENT_SECTION] = current_section
        if current_section != initial_section:
            logger.info("Navigation V7: section %s -> %s", initial_section, current_section)
            pages_dict = st.session_state.get("_v7_pages", {})
            target = pages_dict.get(current_section)
            if target is not None:
                st.switch_page(target)

    with player_col:
        db_path, xuid = _render_player_block(db_path, xuid)

    with sync_col:
        st.markdown(f"<div class='v7-brand-title'>{t('exp_sync')}</div>", unsafe_allow_html=True)
        render_sync_indicator(db_path, xuid=xuid)

    return db_path, xuid, current_section
