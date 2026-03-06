"""Page Carrière — Rendu Streamlit de la section rencontres."""

from __future__ import annotations

import streamlit as st

from src.ui.i18n import t
from src.ui.pages.career_data import (
    _load_top_encountered,
    _load_top_nemeses,
    _load_top_victims,
)
from src.ui.pages.career_encounters_html import (
    build_antagonist_table_html,
    build_encounters_table_html,
)
from src.ui.pages.match_view_encounters import build_badge_legend_html
from src.ui.pages.match_view_encounters_logic import _load_friends_from_json


def render_encounters_section(*, db_path: str, xuid: str) -> None:
    """Rend la section top rencontres, némésis et souffre-douleurs."""
    st.subheader(t("career_encounters_header"))

    friends_set = _load_friends_from_json(xuid)

    encountered = _load_top_encountered(xuid, limit=10, exclude_xuids=friends_set)
    if encountered:
        st.markdown(
            build_encounters_table_html(encountered, t("career_encounters_header")),
            unsafe_allow_html=True,
        )
        st.markdown(build_badge_legend_html(), unsafe_allow_html=True)
    else:
        st.info(t("career_encounters_no_data"))

    nemeses = _load_top_nemeses(xuid, limit=10, exclude_xuids=friends_set)
    victims = _load_top_victims(xuid, limit=10, exclude_xuids=friends_set)

    if nemeses or victims:
        col_nem, col_vic = st.columns(2)
        with col_nem:
            if nemeses:
                st.markdown(
                    build_antagonist_table_html(
                        nemeses, t("career_nemesis_header"), mode="nemesis"
                    ),
                    unsafe_allow_html=True,
                )
            else:
                st.info(t("career_antagonists_no_data"))
        with col_vic:
            if victims:
                st.markdown(
                    build_antagonist_table_html(victims, t("career_victims_header"), mode="victim"),
                    unsafe_allow_html=True,
                )
            else:
                st.info(t("career_antagonists_no_data"))
    else:
        st.info(t("career_antagonists_no_data"))
