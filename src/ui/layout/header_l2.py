"""Bandeau de contexte interactif pour la v7."""

from __future__ import annotations

import logging
from html import escape
from typing import Any

import streamlit as st

from src.app._filters_session import _classify_sessions_solo_squad
from src.app.filters import get_friends_xuids_for_sessions
from src.app.session_keys import SK
from src.ui.cache import cached_compute_sessions_db
from src.ui.filter_state import get_all_filter_keys_to_clear
from src.ui.i18n import t
from src.ui.layout.filter_chips import render_filter_chips
from src.visualization._compat import ensure_polars

logger = logging.getLogger(__name__)


def _render_v7_filter_expander(ctx: Any) -> None:
    """Panneau de filtres inline V7 : st.expander sous la barre de contrôle.

    N'instancie pas de widget key='filter_mode' pour éviter le conflit avec
    le segmented_control du L2 qui écrit dans SK.FILTER_MODE via callback.
    Rend uniquement les filtres cascade (Expérience / Playlists / Modes / Cartes)
    et, si le mode est Période, les sélecteurs de dates.
    """
    from datetime import date as _date

    from src.app._filters_cascade import _render_cascade_filters
    from src.app._filters_period import _render_period_filter
    from src.app.helpers import date_range

    filter_mode = str(st.session_state.get(SK.FILTER_MODE) or "Période").strip()
    base_df = ensure_polars(ctx.base)
    if base_df.is_empty():
        return
    try:
        dmin, dmax = date_range(base_df)
    except Exception:
        return
    if dmin is None or dmax is None:
        return

    start_d = st.session_state.get("start_date_cal") or dmin
    end_d = st.session_state.get("end_date_cal") or dmax
    if not isinstance(start_d, _date):
        start_d = dmin
    if not isinstance(end_d, _date):
        end_d = dmax

    picked = st.session_state.get(SK.PICKED_SESSIONS)
    picked_labels = picked if isinstance(picked, list) and picked else None

    with st.expander(t("v7_filters_button"), expanded=False):
        if filter_mode == "Période":
            _render_period_filter(dmin, dmax)
        _render_cascade_filters(
            base_for_filters=base_df,
            filter_mode=filter_mode,
            start_d=start_d,
            end_d=end_d,
            picked_session_labels=picked_labels,
            base_s_ui=ctx.base_s_ui,
        )


def _clear_all_filters() -> int:
    """Efface tous les filtres persistés dans la session."""
    cleared = 0
    for filter_key in get_all_filter_keys_to_clear(st.session_state):
        if filter_key in st.session_state:
            del st.session_state[filter_key]
            cleared += 1

    logger.info("Reset filtres V7: %s cles effacees", cleared)
    return cleared


def _build_context_caption() -> str:
    """Construit le sous-texte de contexte affiché dans la bande L2."""
    filter_mode = str(st.session_state.get(SK.FILTER_MODE) or "Période").strip()
    session_label = str(st.session_state.get(SK.PICKED_SESSION_LABEL) or "").strip()
    if filter_mode == "Sessions" and session_label and session_label != "(toutes)":
        return t("v7_context_session_active", session=session_label)
    if filter_mode == "Sessions":
        return t("v7_context_session_mode")
    return t("v7_context_period_mode")


def _get_scope_state_key(active_section: str) -> str:
    """Retourne la clé session_state du scope visible pour la section."""
    return (
        SK.PICKED_SQUAD_SESSION_LABEL if active_section == "squad" else SK.PICKED_SOLO_SESSION_LABEL
    )


def _get_scope_label_key(active_section: str) -> str:
    """Retourne la clé i18n du libellé de scope."""
    return (
        "filter_squad_session_label" if active_section == "squad" else "filter_solo_session_label"
    )


def _get_last_button_label(active_section: str) -> str:
    """Retourne le libellé du bouton de dernière session."""
    return "filter_last_carnage" if active_section == "squad" else "filter_last_session"


def _get_previous_button_label(active_section: str) -> str:
    """Retourne le libellé du bouton de session précédente."""
    return "filter_prev_carnage" if active_section == "squad" else "filter_prev_session"


def _normalize_scope_value(current_value: str | None, options: list[str]) -> str:
    """Normalise la valeur de scope affichée dans la barre V7."""
    candidate = str(current_value or "").strip()
    if candidate == "(toutes)":
        return candidate
    if candidate in options:
        return candidate
    return options[0] if options else "(toutes)"


def _get_previous_session_target(current_value: str | None, options: list[str]) -> str:
    """Retourne la cible du bouton précédent pour une liste ordonnée récente→ancienne."""
    if not options:
        return "(toutes)"
    normalized = _normalize_scope_value(current_value, options)
    if normalized == "(toutes)" or normalized not in options:
        return options[0]
    index = options.index(normalized)
    return options[min(index + 1, len(options) - 1)]


def _apply_session_scope(active_section: str, session_label: str) -> None:
    """Applique une sélection de session depuis la barre V7."""
    state_key = _get_scope_state_key(active_section)
    other_key = (
        SK.PICKED_SOLO_SESSION_LABEL if active_section == "squad" else SK.PICKED_SQUAD_SESSION_LABEL
    )
    normalized = str(session_label or "(toutes)").strip() or "(toutes)"
    st.session_state[SK.FILTER_MODE] = "Sessions"
    st.session_state[state_key] = normalized
    st.session_state[other_key] = "(toutes)"
    st.session_state[SK.PICKED_SESSION_LABEL] = normalized
    st.session_state[SK.PICKED_SESSIONS] = [] if normalized == "(toutes)" else [normalized]
    logger.info("Scope V7 %s: session active -> %s", active_section, normalized)


def _on_v7_filter_mode_change(section: str, opts: list[str], scope: str) -> None:
    """Callback on_change du segmented_control de mode filtre V7."""
    widget_key = f"v7_filter_mode_widget_{section}"
    new_mode = st.session_state.get(widget_key)
    if new_mode is None:
        return
    old_mode = str(st.session_state.get(SK.FILTER_MODE) or "Période").strip()
    st.session_state[SK.FILTER_MODE] = new_mode
    logger.info("Mode filtre V7: %s -> %s", old_mode, new_mode)
    if new_mode == "Sessions" and opts:
        _apply_session_scope(section, scope)


def _on_v7_scope_select(section: str) -> None:
    """Callback on_change du sélecteur de session V7."""
    select_key = f"v7_scope_widget_{section}"
    new_scope = st.session_state.get(select_key)
    if new_scope is not None:
        _apply_session_scope(section, str(new_scope))


def _on_v7_scope_button(section: str, target: str) -> None:
    """Callback on_click des boutons de navigation de session V7."""
    _apply_session_scope(section, target)


def _load_scope_options(ctx: Any, active_section: str) -> list[str]:
    """Charge les options de sessions pour la section courante."""
    friends_tuple = get_friends_xuids_for_sessions(
        ctx.db_path,
        ctx.xuid.strip(),
        ctx.db_key,
        ctx.aliases_key,
    )
    base_sessions = cached_compute_sessions_db(
        ctx.db_path,
        ctx.xuid.strip(),
        ctx.db_key,
        True,
        ctx.gap_minutes,
        friends_xuids=friends_tuple,
    )
    solo_options, squad_options = _classify_sessions_solo_squad(
        ensure_polars(base_sessions),
        frozenset(friends_tuple),
    )
    return squad_options if active_section == "squad" else solo_options


def _render_context_controls(active_section: str, ctx: Any) -> None:
    """Rend la barre de contrôle visible du bandeau V7."""
    current_mode = str(st.session_state.get(SK.FILTER_MODE) or "Période").strip()
    options = _load_scope_options(ctx, active_section)
    current_scope = _normalize_scope_value(
        st.session_state.get(_get_scope_state_key(active_section)),
        options,
    )

    st.markdown(
        f"<div class='v7-context-toolbar-label'>{escape(t('exp_filters'))}</div>",
        unsafe_allow_html=True,
    )
    mode_col, scope_col, previous_col, last_col = st.columns([1.4, 2.6, 1.2, 1.2])
    with mode_col:
        st.segmented_control(
            t("filter_selection"),
            options=["Période", "Sessions"],
            default=current_mode if current_mode in {"Période", "Sessions"} else "Période",
            format_func=lambda value: (
                t("filter_period") if value == "Période" else t("filter_sessions")
            ),
            key=f"v7_filter_mode_widget_{active_section}",
            width="stretch",
            on_change=_on_v7_filter_mode_change,
            args=(active_section, options, current_scope),
        )

    with scope_col:
        if current_mode == "Sessions":
            select_options = ["(toutes)", *options]
            st.selectbox(
                t(_get_scope_label_key(active_section)),
                options=select_options,
                index=select_options.index(current_scope) if current_scope in select_options else 0,
                key=f"v7_scope_widget_{active_section}",
                on_change=_on_v7_scope_select,
                args=(active_section,),
            )
        else:
            st.markdown(
                f"<div class='v7-inline-note v7-context-inline-note'>{escape(_build_context_caption())}</div>",
                unsafe_allow_html=True,
            )

    previous_target = _get_previous_session_target(current_scope, options)
    previous_disabled = current_mode != "Sessions" or not options or current_scope == options[-1]
    with previous_col:
        st.markdown("<div class='v7-shell-gap'></div>", unsafe_allow_html=True)
        st.button(
            t(_get_previous_button_label(active_section)),
            key=f"v7_previous_scope_{active_section}",
            width="stretch",
            disabled=previous_disabled,
            on_click=_on_v7_scope_button,
            args=(active_section, previous_target),
        )

    with last_col:
        st.markdown("<div class='v7-shell-gap'></div>", unsafe_allow_html=True)
        st.button(
            t(_get_last_button_label(active_section)),
            key=f"v7_last_scope_{active_section}",
            width="stretch",
            disabled=current_mode != "Sessions" or not options,
            on_click=_on_v7_scope_button,
            args=(active_section, options[0] if options else "(toutes)"),
        )


def render_header_l2(active_section: str, ctx: Any) -> None:
    """Affiche le bandeau de contexte pour Stats et Escouade."""
    if active_section not in {"stats", "squad"}:
        return

    title_key = (
        "v7_section_context_stats" if active_section == "stats" else "v7_section_context_squad"
    )
    left_col, right_col = st.columns([6, 1])
    with left_col:
        st.markdown(
            "".join(
                [
                    "<div class='v7-subshell'>",
                    f"<div class='v7-subshell-title'>{escape(t(title_key))}</div>",
                    f"<div class='v7-subshell-caption'>{escape(_build_context_caption())}</div>",
                    "</div>",
                ]
            ),
            unsafe_allow_html=True,
        )
        _render_context_controls(active_section, ctx)
        _render_v7_filter_expander(ctx)
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
