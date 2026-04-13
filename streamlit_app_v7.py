# ARCHIVED — Interface Streamlit v7. Décommissioné dans Slice 9.
# Le front actif est React/Vite sur :5173 (API FastAPI sur :8000).
# Ce fichier est conservé pour référence historique uniquement.
"""Point d'entree v7 du cockpit analytique."""

from __future__ import annotations

import logging
import sys
from typing import Any

import streamlit as st

# Verification : s'assurer que le script est execute via `streamlit run`
if not hasattr(st, "runtime") or not st.runtime.exists():
    print("\n[ERREUR] Ce script doit etre execute via Streamlit\n")
    print("Usage correct:")
    print("  streamlit run streamlit_app_v7.py\n")
    print("ou via le launcher:")
    print("  python launcher.py\n")
    sys.exit(1)

from collections.abc import Callable  # noqa: E402

from src.app.page_router import (  # noqa: E402
    V7_SECTION_KEYS,
    V7_SECTION_URL_PATHS,
    consume_pending_match_id,
    get_v7_section_i18n_key,
    map_legacy_page_to_v7_location,
    normalize_v7_section,
)
from src.app.session_keys import SK  # noqa: E402
from src.ui.components.browser_storage import restore_browser_prefs  # noqa: E402
from src.ui.i18n import t  # noqa: E402
from src.ui.layout import render_header_l1, render_header_l2, render_kpi_bar  # noqa: E402
from src.ui.pages.home_mission_control_api import (  # noqa: E402
    prefetch_home_progressions,
)
from src.ui.pages.setup_wizard import render_setup_wizard_page  # noqa: E402
from src.ui.pages.setup_wizard_logic import get_setup_status  # noqa: E402
from src.ui.pages.v7_sections import render_v7_section  # noqa: E402
from src.ui.theme import load_v7_theme_css  # noqa: E402
from src.utils.demo import is_demo_mode  # noqa: E402
from streamlit_app import (  # noqa: E402
    _initialize_app,
    _load_and_prepare_data,
    _maybe_apply_browser_prefs,
    _parse_query_params,
    _start_background_services,
    build_friends_opts_map,
    clean_asset_label,
    date_range,
)

logger = logging.getLogger("streamlit_app_v7")


def _make_v7_filter_callbacks() -> Any:
    """Construit le FilterSidebarCallbacks pour le mode V7 (sans sidebar)."""
    from src.app._page_context import FilterSidebarCallbacks

    return FilterSidebarCallbacks(
        date_range_fn=date_range,
        clean_asset_label_fn=clean_asset_label,
        build_friends_opts_map_fn=build_friends_opts_map,
    )


def _apply_v7_theme() -> None:
    """Injecte la feuille de style v7 apres le CSS legacy."""
    st.markdown(load_v7_theme_css(), unsafe_allow_html=True)


def _make_section_callable(section: str) -> Callable[[], None]:
    """Retourne le callable de rendu pour une section v7 (lu depuis session_state)."""

    def _page() -> None:
        ctx = st.session_state.get("_v7_ctx")
        if ctx is None:
            st.info(t("v7_loading") if "v7_loading" in dir(t) else "Chargement en cours...")
            return
        if section in {"stats", "squad"}:
            render_header_l2(section, ctx)
            render_kpi_bar(ctx.dff)
        consume_pending_match_id()
        render_v7_section(section, ctx)

    _page.__name__ = f"v7_{section}"
    return _page


def _build_v7_pages() -> dict[str, st.Page]:
    """Construit le mapping section \u2192 st.Page pour la navigation URL v7."""
    return {
        section: st.Page(
            _make_section_callable(section),
            title=t(get_v7_section_i18n_key(section)),
            url_path=V7_SECTION_URL_PATHS[section],
            default=(section == "home"),
        )
        for section in V7_SECTION_KEYS
    }


def _pg_to_section(pg: st.Page, pages_dict: dict[str, st.Page]) -> str:
    """Retourne la section v7 correspondant à la page active."""
    for section, page in pages_dict.items():
        if page == pg:
            return section
    return "home"


def _hydrate_v7_navigation(pages_dict: dict[str, st.Page]) -> None:
    """Traduit les deep links legacy en navigation v7 (peut appeler st.switch_page)."""
    pending_page = st.session_state.pop(SK.PENDING_PAGE, None)
    if pending_page:
        section, subview = map_legacy_page_to_v7_location(pending_page)
        logger.info(
            "Hydratation V7 depuis deep link legacy: %s -> section=%s subview=%s",
            pending_page,
            section,
            subview,
        )
        if section == "stats" and subview:
            st.session_state[SK.V7_STATS_VIEW] = subview
        if section == "profile" and subview:
            st.session_state[SK.V7_PROFILE_VIEW] = subview
        target = pages_dict.get(section)
        if target is not None:
            st.switch_page(target)  # stoppe l'exécution, URL mise à jour

    st.session_state[SK.V7_CURRENT_SECTION] = normalize_v7_section(
        st.session_state.get(SK.V7_CURRENT_SECTION)
    )
    logger.debug("Section V7 hydratee: %s", st.session_state[SK.V7_CURRENT_SECTION])


def main() -> None:
    """Point d'entree principal de la v7."""
    browser_prefs = restore_browser_prefs()
    if browser_prefs is not None:
        st.session_state["_browser_prefs_restored"] = browser_prefs
        _maybe_apply_browser_prefs(browser_prefs)

    settings, default_db, cfg_warnings, cfg_errors = _initialize_app()
    _apply_v7_theme()

    if is_demo_mode():
        st.info(t("demo_banner"), icon="ℹ️")

    setup_status = get_setup_status()
    if setup_status.needs_setup and not is_demo_mode():
        render_setup_wizard_page()
        return

    _start_background_services(settings, default_db)

    # === Routing URL via st.navigation — doit être appelé avant les widgets ===
    pages_dict = _build_v7_pages()
    st.session_state["_v7_pages"] = pages_dict
    pg = st.navigation(list(pages_dict.values()), position="hidden")

    # Synchroniser la section courante depuis l'URL (source de vérité)
    active_section = _pg_to_section(pg, pages_dict)
    st.session_state[SK.V7_CURRENT_SECTION] = active_section

    _parse_query_params()
    _hydrate_v7_navigation(pages_dict)  # peut appeler st.switch_page → stoppe le run

    db_path = str(st.session_state.get(SK.DB_PATH, "") or "").strip()
    xuid = str(st.session_state.get(SK.XUID_INPUT, "") or "").strip()
    db_path, xuid, _ = render_header_l1(db_path=db_path, xuid=xuid)  # peut appeler st.switch_page
    waypoint_player = str(st.session_state.get(SK.WAYPOINT_PLAYER, "") or "").strip()

    # Pré-fetch battlepass/défis en arrière-plan dès que le joueur est connu
    if db_path and xuid:
        from pathlib import Path as _Path

        _gamertag = _Path(db_path).parent.name or None
        prefetch_home_progressions(db_path, xuid, _gamertag)

    v7_callbacks = _make_v7_filter_callbacks()
    st.session_state["_v7_filter_callbacks_ref"] = v7_callbacks

    ctx = _load_and_prepare_data(
        db_path,
        xuid,
        default_db,
        settings,
        waypoint_player,
        cfg_warnings,
        cfg_errors,
        v7_filter_callbacks=v7_callbacks,
    )
    if ctx is None:
        return

    st.session_state["_v7_ctx"] = ctx
    logger.debug("V7 section active: %s (url=%s)", active_section, pg.url_path)
    pg.run()  # L2 + KPI + contenu spécifique à la section (défini dans le callable)


if __name__ == "__main__":
    main()
