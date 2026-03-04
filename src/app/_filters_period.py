"""Rendu du filtre Période (sélecteurs de dates).

Extrait de filters_render.py pour respecter la limite de taille des modules.
"""

from __future__ import annotations

from datetime import date

import streamlit as st

from src.app._filters_helpers import _safe_to_date
from src.ui.i18n import t


def _render_period_filter(dmin: date, dmax: date) -> tuple[date, date]:
    """Rend les sélecteurs de dates en mode Période."""
    cols = st.columns(2)
    with cols[0]:
        start_default_date = _safe_to_date(dmin)
        end_limit_date = _safe_to_date(dmax)
        if "start_date_cal" not in st.session_state:
            st.session_state["start_date_cal"] = start_default_date
        else:
            cur = st.session_state["start_date_cal"]
            if not isinstance(cur, date) or cur < start_default_date or cur > end_limit_date:
                st.session_state["start_date_cal"] = start_default_date
        start_date = st.date_input(
            t("filter_start"),
            min_value=start_default_date,
            max_value=end_limit_date,
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
            min_value=start_limit_date,
            max_value=end_default_date,
            format="DD/MM/YYYY",
            key="end_date_cal",
        )
    if start_date > end_date:
        st.warning(t("filter_date_error"))
    return start_date, end_date
