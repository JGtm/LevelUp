"""Page Dernier match.

Affiche la dernière partie selon la sélection/filtres actuels.
Propose des boutons de navigation pour parcourir les matchs filtrés.
"""

from __future__ import annotations

import logging

import streamlit as st

from src.app._page_context import MatchViewParams
from src.app.session_keys import SK
from src.ui.i18n import t
from src.visualization._compat import ensure_polars

logger = logging.getLogger(__name__)


def _resolve_nav_index(
    total: int,
    stored_index: int | None,
    stored_total: int | None,
) -> tuple[int, bool]:
    """Calcule l'index de navigation courant — logique pure, sans Streamlit.

    Args:
        total: Nombre de matchs dans le DataFrame filtré courant.
        stored_index: Index stocké en session_state (None si absent).
        stored_total: Total précédemment stocké (None si absent).

    Returns:
        Tuple (index, reset) où reset=True indique que les filtres ont changé
        et que l'index a été réinitialisé au dernier match.
    """
    if stored_total != total:
        logger.debug("Filtres changés : total %s → %d, reset index", stored_total, total)
        return total - 1, True
    idx = stored_index if stored_index is not None else total - 1
    clamped = max(0, min(idx, total - 1))
    if clamped != idx:
        logger.debug("Index hors bornes %d clamped → %d (total=%d)", idx, clamped, total)
    return clamped, False


def render_last_match_page(
    dff,
    params: MatchViewParams,
) -> None:
    """Rend la page Dernier match avec navigation précédent/suivant.

    Args:
        dff: DataFrame filtré des matchs.
        params: Paramètres communs (DB, fonctions injectées, settings, etc.).
    """
    st.caption(t("last_match_caption"))

    dff = ensure_polars(dff)
    if dff.is_empty():
        st.info(t("no_data_filter"))
        return

    sorted_df = dff.sort("start_time")
    total = len(sorted_df)

    idx, reset = _resolve_nav_index(
        total=total,
        stored_index=st.session_state.get(SK.LAST_MATCH_NAV_INDEX),
        stored_total=st.session_state.get(SK.LAST_MATCH_NAV_TOTAL),
    )
    if reset:
        st.session_state[SK.LAST_MATCH_NAV_INDEX] = idx
        st.session_state[SK.LAST_MATCH_NAV_TOTAL] = total

    col_prev, _spacer, col_next = st.columns([1, 8, 1])
    with col_prev:
        if st.button(t("lm_nav_prev"), disabled=(idx == 0), key="lm_nav_prev", width="stretch"):
            st.session_state[SK.LAST_MATCH_NAV_INDEX] = idx - 1
            st.rerun()
    with col_next:
        if st.button(
            t("lm_nav_next"), disabled=(idx == total - 1), key="lm_nav_next", width="stretch"
        ):
            st.session_state[SK.LAST_MATCH_NAV_INDEX] = idx + 1
            st.rerun()

    row = sorted_df.row(idx, named=True)
    match_id = str(row.get("match_id", "")).strip()
    logger.debug("Match nav [%d/%d]: %s", idx + 1, total, match_id)

    params["render_match_view_fn"](
        row=row,
        match_id=match_id,
        params=params,
    )
