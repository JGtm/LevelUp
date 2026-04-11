"""Point d'entree v7 du cockpit analytique."""

from __future__ import annotations

import logging
import sys

import streamlit as st

# Verification : s'assurer que le script est execute via `streamlit run`
if not hasattr(st, "runtime") or not st.runtime.exists():
    print("\n[ERREUR] Ce script doit etre execute via Streamlit\n")
    print("Usage correct:")
    print("  streamlit run streamlit_app_v7.py\n")
    print("ou via le launcher:")
    print("  python launcher.py\n")
    sys.exit(1)

from src.app.page_router import (  # noqa: E402
    consume_pending_match_id,
    map_legacy_page_to_v7_location,
    normalize_v7_section,
)
from src.app.session_keys import SK  # noqa: E402
from src.ui.components.browser_storage import restore_browser_prefs  # noqa: E402
from src.ui.i18n import t  # noqa: E402
from src.ui.layout import render_header_l1, render_header_l2, render_kpi_bar  # noqa: E402
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
)

logger = logging.getLogger("streamlit_app_v7")


def _apply_v7_theme() -> None:
    """Injecte la feuille de style v7 apres le CSS legacy."""
    st.markdown(load_v7_theme_css(), unsafe_allow_html=True)


def _hydrate_v7_navigation() -> None:
    """Traduit les deep links legacy en navigation v7 persistante."""
    pending_page = st.session_state.pop(SK.PENDING_PAGE, None)
    if pending_page:
        section, subview = map_legacy_page_to_v7_location(pending_page)
        logger.info(
            "Hydratation V7 depuis deep link legacy: %s -> section=%s subview=%s",
            pending_page,
            section,
            subview,
        )
        st.session_state[SK.V7_CURRENT_SECTION] = section
        if section == "stats" and subview:
            st.session_state[SK.V7_STATS_VIEW] = subview
        if section == "profile" and subview:
            st.session_state[SK.V7_PROFILE_VIEW] = subview

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
    _parse_query_params()
    _hydrate_v7_navigation()

    db_path = str(st.session_state.get(SK.DB_PATH, "") or "").strip()
    xuid = str(st.session_state.get(SK.XUID_INPUT, "") or "").strip()
    db_path, xuid, active_section = render_header_l1(db_path=db_path, xuid=xuid)
    waypoint_player = str(st.session_state.get(SK.WAYPOINT_PLAYER, "") or "").strip()

    ctx = _load_and_prepare_data(
        db_path,
        xuid,
        default_db,
        settings,
        waypoint_player,
        cfg_warnings,
        cfg_errors,
    )
    if ctx is None:
        return

    render_header_l2(active_section)
    if active_section in {"stats", "squad"}:
        render_kpi_bar(ctx.dff)

    logger.debug("V7 section active: %s", active_section)
    consume_pending_match_id()
    render_v7_section(active_section, ctx)


if __name__ == "__main__":
    main()
