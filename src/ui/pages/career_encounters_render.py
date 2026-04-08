"""Page Carrière — Rendu Streamlit de la section rencontres."""

from __future__ import annotations

from datetime import datetime, timedelta, timezone

import streamlit as st

from src.ui.i18n import get_lang, t
from src.ui.pages.career_encounters_data import (
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


def _compute_encounter_since(period: str) -> datetime | None:
    """Retourne la date de début de période ou None (tout)."""
    now = datetime.now(timezone.utc)
    offsets = {"2y": 730, "1y": 365, "1m": 30, "1w": 7}
    if period in offsets:
        return now - timedelta(days=offsets[period])
    return None


def render_encounters_section(*, db_path: str, xuid: str) -> None:
    """Rend la section top rencontres, némésis et souffre-douleurs."""
    st.subheader(t("career_encounters_header"))

    _PERIOD_KEYS = ["all", "2y", "1y", "1m", "1w"]
    selected_period = st.segmented_control(
        t("encounters_period_label"),
        options=_PERIOD_KEYS,
        format_func=lambda k: t(f"encounters_period_{k}"),
        default="all",
        key="encounters_period",
    )
    since = _compute_encounter_since(selected_period or "all")

    friends_set = _load_friends_from_json(xuid)

    encountered = _load_top_encountered(
        xuid, db_path, limit=10, exclude_xuids=friends_set, since=since
    )
    if encountered:
        st.markdown(
            build_encounters_table_html(encountered, t("career_encounters_header")),
            unsafe_allow_html=True,
        )
        _legend_label = "ℹ️ Légende" if get_lang() == "fr" else "ℹ️ Legend"
        with st.popover(_legend_label):
            st.markdown(build_badge_legend_html(), unsafe_allow_html=True)
    else:
        st.info(t("career_encounters_no_data"))

    nemeses = _load_top_nemeses(xuid, db_path, limit=10, exclude_xuids=friends_set, since=since)
    victims = _load_top_victims(xuid, db_path, limit=10, exclude_xuids=friends_set, since=since)

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
