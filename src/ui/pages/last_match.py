"""Page Dernier match.

Affiche la dernière partie selon la sélection/filtres actuels.
Propose des boutons de navigation pour parcourir les matchs filtrés.
"""

from __future__ import annotations

import logging

import streamlit as st

from src.app._page_context import MatchViewParams
from src.app.session_keys import SK
from src.ui.components.browser_storage import hints_visible
from src.ui.i18n import t
from src.visualization._compat import ensure_polars

logger = logging.getLogger(__name__)


def _resolve_nav_index(
    total: int,
    stored_index: int | None,
    stored_total: int | None,
    session_key: str | None = None,
    stored_session_key: str | None = None,
) -> tuple[int, bool]:
    """Calcule l'index de navigation courant — logique pure, sans Streamlit.

    Args:
        total: Nombre de matchs dans le DataFrame filtré courant.
        stored_index: Index stocké en session_state (None si absent).
        stored_total: Total précédemment stocké (None si absent).
        session_key: Clé de session active (label sélectionné), pour détecter
            les changements de session même quand le total reste inchangé
            (ex : filtre silencieux échoué → dff garde le même total).
        stored_session_key: Valeur précédemment stockée de session_key.

    Returns:
        Tuple (index, reset) où reset=True indique que les filtres ont changé
        et que l'index a été réinitialisé au dernier match.
    """
    session_changed = session_key is not None and session_key != stored_session_key
    if stored_total != total or session_changed:
        logger.debug(
            "Filtres changés : total %s → %d, session %r → %r, reset index",
            stored_total,
            total,
            stored_session_key,
            session_key,
        )
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
    if hints_visible():
        st.caption(t("last_match_caption"))

    dff = ensure_polars(dff)
    if dff.is_empty():
        st.info(t("no_data_filter"))
        return

    sorted_df = dff.sort("start_time")
    total = len(sorted_df)

    # Clé de session = label actif sélectionné dans le filtre sidebar.
    # Permet de détecter un changement de session même si total reste inchangé
    # (ex : filtre silencieusement échoué → dff garde la même taille).
    session_key = str(st.session_state.get("picked_session_label") or "(toutes)")
    _prev_session_key = st.session_state.get(SK.LAST_MATCH_NAV_SESSION_KEY)

    idx, reset = _resolve_nav_index(
        total=total,
        stored_index=st.session_state.get(SK.LAST_MATCH_NAV_INDEX),
        stored_total=st.session_state.get(SK.LAST_MATCH_NAV_TOTAL),
        session_key=session_key,
        stored_session_key=_prev_session_key,
    )
    if reset:
        st.session_state[SK.LAST_MATCH_NAV_INDEX] = idx
        st.session_state[SK.LAST_MATCH_NAV_TOTAL] = total
        st.session_state[SK.LAST_MATCH_NAV_SESSION_KEY] = session_key
        logger.info(
            "Nav reset → match %d/%d | session=%r (précédent: %r)",
            idx + 1,
            total,
            session_key,
            _prev_session_key,
        )

    st.markdown(
        "<style>"
        "div:has(.lm-nav-marker) + div .stButton > button {"
        "  font-size: 11px !important;"
        "  white-space: nowrap !important;"
        "}"
        "</style>"
        '<span class="lm-nav-marker"></span>',
        unsafe_allow_html=True,
    )
    col_prev, _spacer, col_next = st.columns([2, 6, 2])
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

    params.render_match_view_fn(
        row=row,
        match_id=match_id,
        params=params,
    )
