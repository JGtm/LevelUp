"""Page Dernier match.

Affiche la dernière partie selon la sélection/filtres actuels.
"""

from __future__ import annotations

import logging

import streamlit as st

from src.app._page_context import MatchViewParams
from src.ui.i18n import t
from src.visualization._compat import ensure_polars

logger = logging.getLogger(__name__)


def render_last_match_page(
    dff,
    params: MatchViewParams,
) -> None:
    """Rend la page Dernier match.

    Args:
        dff: DataFrame filtré des matchs.
        params: Paramètres communs (DB, fonctions injectées, settings, etc.).
    """
    st.caption(t("last_match_caption"))

    dff = ensure_polars(dff)
    if dff.is_empty():
        st.info(t("no_data_filter"))
        return

    last_row = dff.sort("start_time").row(-1, named=True)
    last_match_id = str(last_row.get("match_id", "")).strip()
    logger.debug("Dernier match auto: %s", last_match_id)

    params["render_match_view_fn"](
        row=last_row,
        match_id=last_match_id,
        params=params,
    )
