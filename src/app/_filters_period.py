"""Filtre Période — sélecteurs de date pour la sidebar.

Ce module expose uniquement :
- ``_render_period_filter`` : rend les deux date_input (début / fin) en mode Période.

Importé par ``filters_render.py`` ; ne pas utiliser directement depuis l'UI.
"""

from __future__ import annotations

from datetime import date

import streamlit as st

from src.app._filters_shared import safe_to_date as _safe_to_date
from src.ui.i18n import t


def _render_period_filter(dmin: date, dmax: date) -> tuple[date, date]:
    """Rend les sélecteurs de dates en mode Période."""
    cols = st.columns(2)
    with cols[0]:
        start_default_date = _safe_to_date(dmin)
        end_limit_date = _safe_to_date(dmax)
        nav_min = date(start_default_date.year, 1, 1)
        nav_max = date(end_limit_date.year, 12, 31)
        if "start_date_cal" not in st.session_state:
            st.session_state["start_date_cal"] = start_default_date
        else:
            cur = st.session_state["start_date_cal"]
            if not isinstance(cur, date) or cur < start_default_date or cur > end_limit_date:
                st.session_state["start_date_cal"] = start_default_date
        start_date = st.date_input(
            t("filter_start"),
            min_value=nav_min,
            max_value=nav_max,
            format="DD/MM/YYYY",
            key="start_date_cal",
        )
    with cols[1]:
        end_default_date = _safe_to_date(dmax)
        start_limit_date = _safe_to_date(dmin)
        if "end_date_cal" not in st.session_state:
            st.session_state["end_date_cal"] = end_default_date
        else:
            cur = st.session_state["end_date_cal"]
            if not isinstance(cur, date) or cur < start_limit_date or cur > end_default_date:
                st.session_state["end_date_cal"] = end_default_date
        end_date = st.date_input(
            t("filter_end"),
            min_value=nav_min,
            max_value=nav_max,
            format="DD/MM/YYYY",
            key="end_date_cal",
        )
    if start_date > end_date:
        st.warning(t("filter_date_error"))
    return start_date, end_date
