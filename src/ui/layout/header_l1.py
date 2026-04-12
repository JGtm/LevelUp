"""Shell global L1 pour la v7."""

from __future__ import annotations

import contextlib
import html
import logging
from pathlib import Path
from urllib.parse import quote

import streamlit as st

from src.app.page_router import (
    V7_SECTION_KEYS,
    V7_SECTION_URL_PATHS,
    get_v7_section_label,
    normalize_v7_section,
)
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
)

logger = logging.getLogger(__name__)


def _get_current_player_label(db_path: str, xuid: str) -> str:
    """Retourne le libellé joueur affiché dans le header L1."""
    if _has_multiple_players(db_path):
        gamertag = get_gamertag_from_duckdb_v4_path(db_path)
        if gamertag:
            return gamertag

    if str(xuid or "").strip():
        return display_name_from_xuid(xuid.strip(), db_path=db_path)
    return "-"


def _get_player_widget_width(player_label: str) -> int:
    """Calcule une largeur compacte pour le sélecteur joueur du header."""
    label = str(player_label or "-").strip() or "-"
    return max(104, min(188, 28 + len(label) * 7))


def _get_player_column_ratio(player_width_px: int) -> float:
    """Convertit la largeur souhaitée du sélecteur en poids de colonne Streamlit."""
    return max(0.76, min(1.18, player_width_px / 164.0))


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


def _render_inline_player_selector(db_path: str, xuid: str, *, width_px: int) -> tuple[str, str]:
    """Rend un menu joueur HTML compact pour le header L1."""
    del width_px
    player_markup = _build_player_menu_html(db_path, xuid)
    st.markdown(player_markup, unsafe_allow_html=True)
    return db_path, xuid


def _resolve_player_xuid_for_db(db_path: str) -> str | None:
    """Résout l'XUID d'une DB joueur si disponible."""
    try:
        from src.ui.cache_loaders import _resolve_player_xuid

        resolved = _resolve_player_xuid(db_path)
        if resolved:
            return resolved
    except Exception:
        pass
    return None


def _section_href(section: str) -> str:
    """Retourne l'URL relative d'une section V7."""
    url_path = V7_SECTION_URL_PATHS.get(section, "")
    return "/" if not url_path else f"/{url_path}"


def _player_href(section: str, gamertag: str) -> str:
    """Construit l'URL interne d'un switch joueur pour la section courante."""
    base = _section_href(section)
    return f"{base}?player={quote(gamertag)}"


def _build_tab_links(labels: dict[str, str], nav_sections: list[str], current_section: str) -> str:
    """Construit le markup des tabs L1 en liens HTML internes."""
    links: list[str] = []
    for section in nav_sections:
        class_name = "v7-l1-tab v7-l1-tab--active" if current_section == section else "v7-l1-tab"
        links.append(
            f"<a class='{class_name}' href='{html.escape(_section_href(section), quote=True)}' "
            f"target='_self'>{html.escape(labels[section])}</a>"
        )
    return "<nav class='v7-l1-tabs'>" + "".join(links) + "</nav>"


def _build_player_menu_html(db_path: str, xuid: str) -> str:
    """Construit le dropdown joueur en HTML pur pour éviter le selectbox Streamlit."""
    current_label = _get_current_player_label(db_path, xuid)
    current_section = normalize_v7_section(st.session_state.get(SK.V7_CURRENT_SECTION))
    if not _has_multiple_players(db_path):
        return f"<div class='v7-l1-player-text'>{html.escape(current_label)}</div>"

    current_gamertag = get_gamertag_from_duckdb_v4_path(db_path)
    items: list[str] = []
    for player in list_duckdb_v4_players():
        item_class = "v7-l1-player-option"
        if player.gamertag == current_gamertag:
            item_class = f"{item_class} v7-l1-player-option--active"
        items.append(
            f"<a class='{item_class}' href='{html.escape(_player_href(current_section, player.gamertag), quote=True)}' "
            f"target='_self'>{html.escape(player.gamertag)}</a>"
        )

    return (
        "<details class='v7-l1-player-menu'>"
        "<summary class='v7-l1-player-summary'>"
        f"<span>{html.escape(current_label)}</span>"
        "<span class='v7-l1-player-chevron'>⌄</span>"
        "</summary>"
        f"<div class='v7-l1-player-popover'>{''.join(items)}</div>"
        "</details>"
    )


def _consume_pending_player_switch(db_path: str, xuid: str) -> tuple[str, str]:
    """Applique un switch joueur déclenché via query param `player`."""
    pending_player = str(st.session_state.pop(SK.PENDING_PLAYER, "") or "").strip()
    if not pending_player or not _has_multiple_players(db_path):
        return db_path, xuid

    current_gamertag = str(get_gamertag_from_duckdb_v4_path(db_path) or "").strip()
    if pending_player.casefold() == current_gamertag.casefold():
        return db_path, xuid

    selected_player = next(
        (p for p in list_duckdb_v4_players() if p.gamertag.casefold() == pending_player.casefold()),
        None,
    )
    if selected_player is None:
        return db_path, xuid

    db_path, xuid = _apply_player_change(
        current_db_path=db_path,
        current_xuid=xuid,
        new_db_path=str(selected_player.db_path),
        new_xuid=_resolve_player_xuid_for_db(str(selected_player.db_path)),
    )
    st.rerun()
    return db_path, xuid


def render_header_l1(*, db_path: str, xuid: str) -> tuple[str, str, str]:
    """Affiche le shell global L1 et retourne l'etat de navigation."""
    db_path, xuid = _consume_pending_player_switch(db_path, xuid)
    current_section = normalize_v7_section(st.session_state.get(SK.V7_CURRENT_SECTION))
    labels = {section: get_v7_section_label(section) for section in V7_SECTION_KEYS}
    nav_sections = [section for section in V7_SECTION_KEYS if section != "settings"]
    player_label = _get_current_player_label(db_path, xuid)
    player_width_px = _get_player_widget_width(player_label)
    player_col_ratio = _get_player_column_ratio(player_width_px)
    nav_col_ratio = max(2.75, 3.35 - player_col_ratio * 0.12)

    brand_col, shell_col = st.columns([0.55, 4.45], gap="medium")

    with brand_col:
        st.markdown("<div class='v7-l1-brand'>LevelUp</div>", unsafe_allow_html=True)

    with shell_col:
        st.markdown("<div class='v7-l1-shell-anchor'></div>", unsafe_allow_html=True)
        tabs_col, player_col, sync_col, settings_col = st.columns(
            [nav_col_ratio, player_col_ratio, 0.18, 0.18],
            gap="small",
        )

        with tabs_col:
            st.markdown(
                "<div class='v7-l1-tabs-anchor'>"
                + _build_tab_links(labels, nav_sections, current_section)
                + "</div>",
                unsafe_allow_html=True,
            )

        with player_col:
            db_path, xuid = _render_inline_player_selector(
                db_path,
                xuid,
                width_px=player_width_px,
            )

        with sync_col:
            render_sync_indicator(db_path, xuid=xuid, dot_only=True)

        with settings_col:
            settings_class = (
                "v7-l1-tool-link v7-l1-tool-link--active"
                if current_section == "settings"
                else "v7-l1-tool-link"
            )
            st.markdown(
                (
                    f"<a class='{settings_class}' href='{html.escape(_section_href('settings'), quote=True)}' "
                    f"target='_self' aria-label='{html.escape(t('page_settings'), quote=True)}'>"
                    f"{html.escape(labels['settings'])}</a>"
                ),
                unsafe_allow_html=True,
            )

    return db_path, xuid, current_section
