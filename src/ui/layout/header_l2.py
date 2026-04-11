"""Bandeau de contexte léger pour la v7."""

from __future__ import annotations

import logging

import streamlit as st

from src.ui.filter_state import get_all_filter_keys_to_clear
from src.ui.i18n import t
from src.ui.layout.filter_chips import render_filter_chips

logger = logging.getLogger(__name__)


def _clear_all_filters() -> int:
    """Efface tous les filtres persistés dans la session."""
    cleared = 0
    for filter_key in get_all_filter_keys_to_clear(st.session_state):
        if filter_key in st.session_state:
            del st.session_state[filter_key]
            cleared += 1

    logger.info("Reset filtres V7: %s cles effacees", cleared)
    return cleared


def render_header_l2(active_section: str) -> None:
    """Affiche le bandeau de contexte pour Stats et Escouade."""
    if active_section not in {"stats", "squad"}:
        return

    title_key = (
        "v7_section_context_stats" if active_section == "stats" else "v7_section_context_squad"
    )
    left_col, right_col = st.columns([6, 1])
    with left_col:
        st.markdown(
            f"<div class='v7-subshell'><div class='v7-subshell-title'>{t(title_key)}</div></div>",
            unsafe_allow_html=True,
        )
        render_filter_chips()
    with right_col:
        st.markdown("<div class='v7-shell-gap'></div>", unsafe_allow_html=True)
        if st.button(
            t("v7_filters_reset"),
            key=f"v7_reset_filters_{active_section}",
            width="stretch",
        ):
            logger.info("Reset filtres V7 demande depuis la section %s", active_section)
            _clear_all_filters()
            st.rerun()
