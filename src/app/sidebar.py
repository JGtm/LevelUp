"""Logique et rendu de la sidebar.

Ce module centralise :
- Le rendu de la sidebar (brand, navigation, filtres)
- Le bouton de synchronisation
- Le sélecteur de joueur (multi-joueurs)
"""

from __future__ import annotations

import os
from collections.abc import Callable
from typing import TYPE_CHECKING

import streamlit as st

from src.ui.i18n import set_lang, t
from src.ui.multiplayer import (
    render_player_selector,
)
from src.ui.sync import (
    render_sync_indicator,
    sync_all_players_duckdb,
)

if TYPE_CHECKING:
    from src.ui.settings import AppSettings


def _render_lang_selector(settings: AppSettings) -> bool:
    """Affiche le sélecteur de langue et retourne True si la langue a changé."""
    _LANG_OPTIONS = {"fr": "🇫🇷 Français", "en": "🇬🇧 English"}
    current = st.session_state.get("lang", settings.lang or "fr")
    option_keys = list(_LANG_OPTIONS.keys())
    current_idx = option_keys.index(current) if current in option_keys else 0

    selected_label = st.selectbox(
        t("lang_selector_label"),
        options=list(_LANG_OPTIONS.values()),
        index=current_idx,
        key="_lang_selector_widget",
    )
    selected_lang = next(k for k, v in _LANG_OPTIONS.items() if v == selected_label)

    if selected_lang != current:
        set_lang(selected_lang)
        settings.lang = selected_lang
        from src.ui.settings import save_settings

        save_settings(settings)
        return True
    return False


def render_sidebar(
    *,
    db_path: str,
    xuid: str,
    settings: AppSettings,
    on_player_change: Callable[[str], None] | None = None,
    on_sync_complete: Callable[[], None] | None = None,
) -> str:
    """Rend la sidebar complète.

    Args:
        db_path: Chemin vers la base de données.
        xuid: XUID du joueur courant.
        settings: Paramètres de l'application.
        on_player_change: Callback appelé quand le joueur change.
        on_sync_complete: Callback appelé après une sync réussie.

    Returns:
        Le XUID potentiellement mis à jour.
    """
    with st.sidebar:
        # Brand
        st.header("LevelUp")
        st.divider()

        # Sélecteur de langue (initialise aussi st.session_state["lang"])
        if "lang" not in st.session_state:
            st.session_state["lang"] = settings.lang or "fr"
        lang_changed = _render_lang_selector(settings)
        if lang_changed:
            st.rerun()

        st.divider()

        # Indicateur de dernière synchronisation
        if db_path and os.path.exists(db_path):
            render_sync_indicator(db_path)

        # Sélecteur multi-joueurs (si DB fusionnée)
        new_xuid = render_player_selector_sidebar(
            db_path=db_path,
            xuid=xuid,
            on_change=on_player_change,
        )
        if new_xuid and new_xuid != xuid:
            xuid = new_xuid

        # Bouton Sync
        render_sync_button(
            db_path=db_path,
            settings=settings,
            on_complete=on_sync_complete,
        )

    return xuid


def render_player_selector_sidebar(
    *,
    db_path: str,
    xuid: str,
    on_change: Callable[[str], None] | None = None,
) -> str | None:
    """Rend le sélecteur de joueur dans la sidebar.

    Args:
        db_path: Chemin vers la base de données.
        xuid: XUID du joueur courant.
        on_change: Callback appelé quand le joueur change.

    Returns:
        Le nouveau XUID si changé, None sinon.
    """
    if not (db_path and os.path.exists(db_path)):
        return None

    new_xuid = render_player_selector(db_path, xuid, key="sidebar_player_selector")

    if new_xuid and new_xuid != xuid:
        st.session_state["xuid_input"] = new_xuid

        # Reset des filtres au changement de joueur
        for filter_key in ["filter_playlists", "filter_modes", "filter_maps"]:
            if filter_key in st.session_state:
                del st.session_state[filter_key]

        if on_change:
            on_change(new_xuid)

        return new_xuid

    return None


def render_sync_button(
    *,
    db_path: str,
    settings: AppSettings,
    on_complete: Callable[[], None] | None = None,
) -> bool:
    """Rend le bouton de synchronisation DuckDB v4.

    Synchronise tous les joueurs définis dans db_profiles.json.

    Args:
        db_path: Ignoré (gardé pour compatibilité API).
        settings: Paramètres de l'application.
        on_complete: Callback appelé après une sync réussie.

    Returns:
        True si une sync a été effectuée avec succès.
    """
    from pathlib import Path

    repo_root = Path(__file__).resolve().parent.parent.parent
    db_profiles_path = repo_root / "db_profiles.json"

    if not db_profiles_path.exists():
        return False

    if st.button(
        t("sidebar_sync_btn"),
        key="sidebar_sync_button",
        help=t("sidebar_sync_help"),
        width="stretch",
    ):
        with st.spinner(t("syncing")):
            ok, msg = sync_all_players_duckdb(
                match_type=str(
                    getattr(settings, "spnkr_refresh_match_type", "matchmaking") or "matchmaking"
                ),
                max_matches=int(getattr(settings, "spnkr_refresh_max_matches", 200) or 200),
                with_highlight_events=True,
                with_aliases=True,
                delta=True,
                repo_root=repo_root,
            )

        if ok:
            st.success(msg)

            # Backfill après synchronisation si activé
            backfill_enabled = bool(getattr(settings, "spnkr_refresh_with_backfill", False))
            # Vérifier aussi si au moins une option de backfill est activée individuellement
            has_any_backfill_option = any(
                [
                    bool(getattr(settings, "spnkr_refresh_backfill_medals", False)),
                    bool(getattr(settings, "spnkr_refresh_backfill_events", False)),
                    bool(getattr(settings, "spnkr_refresh_backfill_skill", False)),
                    bool(getattr(settings, "spnkr_refresh_backfill_personal_scores", False)),
                    bool(getattr(settings, "spnkr_refresh_backfill_performance_scores", True)),
                    bool(getattr(settings, "spnkr_refresh_backfill_aliases", False)),
                ]
            )

            if backfill_enabled or has_any_backfill_option:
                import asyncio

                from scripts.backfill_data import backfill_all_players

                with st.spinner(t("sidebar_backfill_running")):
                    backfill_result = asyncio.run(
                        backfill_all_players(
                            dry_run=False,
                            max_matches=None,
                            requests_per_second=5,
                            medals=bool(getattr(settings, "spnkr_refresh_backfill_medals", False)),
                            events=bool(getattr(settings, "spnkr_refresh_backfill_events", False)),
                            skill=bool(getattr(settings, "spnkr_refresh_backfill_skill", False)),
                            personal_scores=bool(
                                getattr(settings, "spnkr_refresh_backfill_personal_scores", False)
                            ),
                            performance_scores=bool(
                                getattr(settings, "spnkr_refresh_backfill_performance_scores", True)
                            ),
                            aliases=bool(
                                getattr(settings, "spnkr_refresh_backfill_aliases", False)
                            ),
                            all_data=False,  # On utilise les options individuelles
                        )
                    )

                    if backfill_result.get("players_processed", 0) > 0:
                        totals = backfill_result.get("total_results", {})
                        backfill_parts = []
                        if totals.get("medals_inserted", 0) > 0:
                            backfill_parts.append(t("backfill_medals", n=totals["medals_inserted"]))
                        if totals.get("events_inserted", 0) > 0:
                            backfill_parts.append(t("backfill_events", n=totals["events_inserted"]))
                        if totals.get("skill_inserted", 0) > 0:
                            backfill_parts.append(t("backfill_skill"))
                        if totals.get("personal_scores_inserted", 0) > 0:
                            backfill_parts.append(
                                t("backfill_personal_scores", n=totals["personal_scores_inserted"])
                            )
                        if totals.get("performance_scores_inserted", 0) > 0:
                            backfill_parts.append(
                                t("backfill_scores", n=totals["performance_scores_inserted"])
                            )
                        if totals.get("aliases_inserted", 0) > 0:
                            backfill_parts.append(
                                t("backfill_aliases", n=totals["aliases_inserted"])
                            )

                        if backfill_parts:
                            st.info(f"Backfill: {', '.join(backfill_parts)}")

            if on_complete:
                on_complete()
            return True
        else:
            st.error(msg)

    return False


def render_navigation_tabs(
    *,
    pages: list[str],
    current_page: str,
    on_change: Callable[[str], None] | None = None,
) -> str:
    """Rend les onglets de navigation.

    Args:
        pages: Liste des noms de pages.
        current_page: Page courante.
        on_change: Callback appelé quand la page change.

    Returns:
        La page sélectionnée.
    """
    # Trouver l'index de la page courante
    try:
        current_index = pages.index(current_page)
    except ValueError:
        current_index = 0

    # Utiliser st.tabs ou st.radio selon le nombre de pages
    if len(pages) <= 8:
        tabs = st.tabs(pages)
        for i, tab in enumerate(tabs):
            with tab:
                if i != current_index and on_change:
                    on_change(pages[i])
        return pages[current_index]
    else:
        selected = st.radio(
            t("sidebar_navigation"),
            options=pages,
            index=current_index,
            horizontal=True,
            label_visibility="collapsed",
        )
        if selected != current_page and on_change:
            on_change(selected)
        return selected


def render_db_info(db_path: str) -> None:
    """Affiche les informations sur la DB dans la sidebar.

    Args:
        db_path: Chemin vers la base de données.
    """
    if not db_path:
        st.caption(t("sidebar_no_db_selected"))
        return

    if not os.path.exists(db_path):
        st.warning(t("sidebar_db_not_found", path=db_path))
        return

    try:
        size_mb = os.path.getsize(db_path) / (1024 * 1024)
        st.caption(f"📁 {os.path.basename(db_path)} ({size_mb:.1f} MB)")
    except Exception:
        st.caption(f"📁 {os.path.basename(db_path)}")


def render_quick_filters(
    *,
    playlists: list[str],
    selected_playlists: list[str],
    on_change: Callable[[list[str]], None] | None = None,
) -> list[str]:
    """Rend les filtres rapides de playlist.

    Args:
        playlists: Liste des playlists disponibles.
        selected_playlists: Playlists actuellement sélectionnées.
        on_change: Callback appelé quand la sélection change.

    Returns:
        Liste des playlists sélectionnées.
    """
    if not playlists:
        return selected_playlists

    with st.expander(t("sidebar_quick_filters"), expanded=False):
        new_selection = st.multiselect(
            t("filter_playlists"),
            options=playlists,
            default=selected_playlists,
            key="quick_filter_playlists",
        )

        if new_selection != selected_playlists and on_change:
            on_change(new_selection)

        return new_selection

    return selected_playlists
