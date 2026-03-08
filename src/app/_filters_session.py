"""Filtre Sessions — sélecteur de sessions solo / escouade pour la sidebar.

Ce module expose :
- ``_apply_default_last_session`` : initialise la session par défaut au premier chargement.
- ``_classify_sessions_solo_squad`` : classifie les sessions selon la présence d'amis.
- ``_render_session_filter`` : rend les contrôles complets du mode Sessions.

Importé par ``filters_render.py`` ; ne pas utiliser directement depuis l'UI.

Note : les imports de ``filters_render`` (``GAP_MINUTES_FIXED``,
``_cascade_reset_filters``, ``_session_labels_ordered_by_last_match``) sont
effectués en local dans chaque fonction pour éviter les imports circulaires au
niveau module.
"""

from __future__ import annotations

import logging
from collections.abc import Callable

import polars as pl
import streamlit as st

from src.app.filters import get_friends_xuids_for_sessions
from src.ui.cache import cached_compute_sessions_db
from src.ui.i18n import t
from src.utils.polars_compat import ensure_polars as _to_polars

logger = logging.getLogger(__name__)


def _apply_default_last_session(
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    aliases_key: int | None = None,
) -> None:
    """Applique par défaut la dernière session du joueur quand aucun filtre n'est en mémoire.

    Utilisé au premier chargement ou changement de joueur/db.
    """
    # Import local pour éviter la circularité au niveau module
    from src.app.filters_render import GAP_MINUTES_FIXED, _session_labels_ordered_by_last_match

    gap_default = GAP_MINUTES_FIXED
    friends_tuple = get_friends_xuids_for_sessions(db_path, xuid.strip(), db_key, aliases_key)
    base_s = cached_compute_sessions_db(
        db_path, xuid.strip(), db_key, True, gap_default, friends_xuids=friends_tuple
    )
    options = _session_labels_ordered_by_last_match(base_s)
    last_label = options[0] if options else "(toutes)"
    st.session_state["filter_mode"] = "Sessions"
    st.session_state["gap_minutes"] = gap_default
    st.session_state["picked_session_label"] = last_label
    st.session_state["picked_sessions"] = [last_label] if last_label != "(toutes)" else []
    st.session_state["_latest_session_label"] = last_label if last_label != "(toutes)" else None
    # min_matches pour cohérence avec le bouton "Dernière session"
    st.session_state["min_matches_maps"] = 1
    st.session_state["_min_matches_maps_auto"] = True
    st.session_state["min_matches_maps_friends"] = 1
    st.session_state["_min_matches_maps_friends_auto"] = True


def _classify_sessions_solo_squad(
    base_s: pl.DataFrame,
    friends_xuids: frozenset[str],
) -> tuple[list[str], list[str]]:
    """Classifie les sessions en solo vs escouade.

    Une session est "escouade" si au moins un de ses matchs contient
    un ami parmi friends_xuids (via la colonne teammates_signature).
    Sinon, elle est classée "solo".

    Implémentation vectorisée Polars (str.contains sur chaque XUID ami) —
    O(k * n) avec k = nb amis (≤3) et n = nb matchs, tout en C, sans boucle Python
    sur les lignes.

    Args:
        base_s: DataFrame sessions (doit avoir session_id, session_label,
                start_time et optionnellement teammates_signature).
        friends_xuids: XUIDs des amis sélectionnés dans le multiselect Teammates.

    Returns:
        (solo_labels, squad_labels) ordonnés par date décroissante.
    """
    # Import local pour éviter la circularité au niveau module
    from src.app.filters_render import _session_labels_ordered_by_last_match

    if base_s.is_empty():
        return [], []

    if not friends_xuids or "teammates_signature" not in base_s.columns:
        return _session_labels_ordered_by_last_match(base_s), []

    # ── Détection vectorisée : au moins un ami présent dans la signature ────
    # teammates_signature = XUIDs comma-separated (ex: "1234,5678,9012")
    # XUIDs sont des nombres 16-18 chiffres → pas de faux positifs str.contains
    sig_col = pl.col("teammates_signature").cast(pl.Utf8).fill_null("")
    friend_exprs = [sig_col.str.contains(fxuid, literal=True) for fxuid in friends_xuids]

    has_friend_expr = friend_exprs[0]
    for expr in friend_exprs[1:]:
        has_friend_expr = has_friend_expr | expr

    df_marked = base_s.select(
        ["session_id", "session_label", "start_time", "teammates_signature"]
    ).with_columns(has_friend_expr.alias("_has_friend"))

    # Sessions escouade = celles qui ont au moins un match avec ami
    squad_session_ids: set[str] = set(
        df_marked.filter(pl.col("_has_friend"))["session_id"].drop_nulls().cast(pl.Utf8).to_list()
    )

    if not squad_session_ids:
        return _session_labels_ordered_by_last_match(base_s), []

    # Tri des labels par date décroissante (une itération sur les sessions, pas les matchs)
    agg = (
        base_s.group_by(["session_id", "session_label"])
        .agg(pl.col("start_time").max())
        .sort("start_time", descending=True)
    )
    solo_labels: list[str] = []
    squad_labels: list[str] = []
    for row in agg.iter_rows(named=True):
        sid = str(row.get("session_id") or "")
        lbl = str(row.get("session_label") or "")
        if not lbl:
            continue
        (squad_labels if sid in squad_session_ids else solo_labels).append(lbl)
    return solo_labels, squad_labels


# ---------------------------------------------------------------------------
# Helpers extraits de _render_session_filter (refacto qualité)
# ---------------------------------------------------------------------------


def _select_solo_via_button(label: str) -> None:
    """Écrit des clés pending pour sélection solo — appelé par boutons."""
    st.session_state["_pending_solo_session_label"] = label
    st.session_state["_pending_squad_session_label"] = "(toutes)"


def _select_squad_via_button(label: str) -> None:
    """Écrit des clés pending pour sélection escouade — appelé par boutons."""
    st.session_state["_pending_squad_session_label"] = label
    st.session_state["_pending_solo_session_label"] = "(toutes)"


def _load_session_context(  # noqa: PLR0913
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    aliases_key: int | None,
    build_friends_opts_map_fn: Callable,
    *,
    prefetched_friends: tuple[str, ...] | None = None,
    prefetched_sessions: pl.DataFrame | None = None,
) -> tuple[pl.DataFrame, list[str], list[str], tuple[str, ...], frozenset[str]]:
    """Charge les sessions et classifie solo/escouade.

    Returns:
        (base_s_ui, solo_options, squad_options, friends_tuple, friends_xuids)
    """
    from src.app.filters_render import GAP_MINUTES_FIXED, _session_labels_ordered_by_last_match

    friends_tuple = prefetched_friends or get_friends_xuids_for_sessions(
        db_path, xuid.strip(), db_key, aliases_key
    )
    if prefetched_sessions is not None and not prefetched_sessions.is_empty():
        base_s_raw = _to_polars(prefetched_sessions)
        logger.debug("Sessions: utilisation du cache pré-chargé (%d matchs)", len(base_s_raw))
    else:
        base_s_raw = _to_polars(
            cached_compute_sessions_db(
                db_path,
                xuid.strip(),
                db_key,
                True,
                GAP_MINUTES_FIXED,
                friends_xuids=friends_tuple,
            )
        )

    base_s_ui = (
        base_s_raw.select(["match_id", "start_time", "session_id", "session_label"])
        if not base_s_raw.is_empty()
        else pl.DataFrame(
            schema={
                "match_id": pl.Utf8,
                "start_time": pl.Datetime,
                "session_id": pl.Utf8,
                "session_label": pl.Utf8,
            }
        )
    )

    options_ui = _session_labels_ordered_by_last_match(base_s_ui)
    st.session_state["_latest_session_label"] = options_ui[0] if options_ui else None

    friends_opts_map, friends_default_labels = build_friends_opts_map_fn(
        db_path, xuid.strip(), db_key, aliases_key
    )
    ui_picked_labels: list[str] = (
        st.session_state.get("teammates_picked_labels") or friends_default_labels
    )
    friends_xuids = frozenset(
        friends_opts_map[lbl] for lbl in ui_picked_labels if lbl in friends_opts_map
    )

    solo_options, squad_options = _classify_sessions_solo_squad(base_s_raw, friends_xuids)
    return base_s_ui, solo_options, squad_options, friends_tuple, friends_xuids


def _init_session_state_keys(
    solo_options: list[str],
    squad_options: list[str],
) -> tuple[str, str]:
    """Initialise et réconcilie les clés session_state pour solo/escouade.

    Returns:
        (pre_solo, pre_squad) — captures pré-rendu pour détection de changement.
    """
    if "picked_solo_session_label" not in st.session_state:
        existing = st.session_state.get("picked_session_label", "(toutes)")
        if existing in solo_options:
            st.session_state["picked_solo_session_label"] = existing
            st.session_state.setdefault("picked_squad_session_label", "(toutes)")
        elif existing in squad_options:
            st.session_state["picked_squad_session_label"] = existing
            st.session_state["picked_solo_session_label"] = "(toutes)"
        else:
            first_solo = solo_options[0] if solo_options else "(toutes)"
            st.session_state["picked_solo_session_label"] = first_solo
            st.session_state.setdefault("picked_squad_session_label", "(toutes)")
            if "picked_session_label" not in st.session_state:
                st.session_state["picked_session_label"] = first_solo
                st.session_state["picked_sessions"] = (
                    [first_solo] if first_solo != "(toutes)" else []
                )

    st.session_state.setdefault("picked_squad_session_label", "(toutes)")

    if "picked_sessions" not in st.session_state:
        active = st.session_state.get("picked_session_label", "(toutes)")
        st.session_state["picked_sessions"] = [active] if active != "(toutes)" else []

    # Cohérence : remettre dans la liste si session disparue
    if st.session_state.get("picked_solo_session_label") not in ["(toutes)"] + solo_options:
        st.session_state["picked_solo_session_label"] = (
            solo_options[0] if solo_options else "(toutes)"
        )
    if st.session_state.get("picked_squad_session_label") not in ["(toutes)"] + squad_options:
        st.session_state["picked_squad_session_label"] = "(toutes)"

    # Consommation des clés pending (AVANT tout widget)
    _pending_solo = st.session_state.pop("_pending_solo_session_label", None)
    if _pending_solo is not None:
        st.session_state["picked_solo_session_label"] = _pending_solo
    _pending_squad = st.session_state.pop("_pending_squad_session_label", None)
    if _pending_squad is not None:
        st.session_state["picked_squad_session_label"] = _pending_squad

    return (
        st.session_state.get("picked_solo_session_label", "(toutes)"),
        st.session_state.get("picked_squad_session_label", "(toutes)"),
    )


def _render_solo_section(solo_options: list[str], pre_solo: str) -> None:
    """Rend la sous-section En solo (boutons + selectbox + détection changement)."""
    st.subheader(t("filter_solo_title"))
    st.divider()
    solo_cols = st.columns(2)
    if solo_cols[0].button(t("filter_last_session"), width="stretch", key="btn_solo_last"):
        _select_solo_via_button(solo_options[0] if solo_options else "(toutes)")
        st.session_state["min_matches_maps"] = 1
        st.session_state["_min_matches_maps_auto"] = True
        st.session_state["min_matches_maps_friends"] = 1
        st.session_state["_min_matches_maps_friends_auto"] = True
        st.rerun()
    if solo_cols[1].button(t("filter_prev_session"), width="stretch", key="btn_solo_prev"):
        current = st.session_state.get("picked_solo_session_label", "(toutes)")
        if not solo_options:
            _select_solo_via_button("(toutes)")
        elif current == "(toutes)" or current not in solo_options:
            _select_solo_via_button(solo_options[0])
        else:
            idx = solo_options.index(current)
            _select_solo_via_button(solo_options[min(idx + 1, len(solo_options) - 1)])
        st.rerun()

    st.selectbox(
        t("filter_solo_session_label"),
        options=["(toutes)"] + solo_options,
        format_func=lambda x: t("sel_all_categories") if x == "(toutes)" else x,
        key="picked_solo_session_label",
    )

    # ── Détection changement solo → reset escouade ──────────────────────────
    _post_solo = st.session_state.get("picked_solo_session_label", "(toutes)")
    if _post_solo != pre_solo and _post_solo != "(toutes)":
        st.session_state["picked_squad_session_label"] = "(toutes)"


def _render_squad_section(
    squad_options: list[str],
    no_friends: bool,
    pre_squad: str,
) -> None:
    """Rend la sous-section Mon escouade (boutons + selectbox + détection changement)."""
    st.subheader(t("filter_squad_title"))
    st.divider()
    no_squad_sessions = not squad_options

    if no_friends:
        st.caption(t("filter_squad_no_friends"))

    squad_cols = st.columns(2)
    if squad_cols[0].button(
        t("filter_last_carnage"),
        width="stretch",
        key="btn_squad_last",
        disabled=no_friends or no_squad_sessions,
    ):
        _select_squad_via_button(squad_options[0] if squad_options else "(toutes)")
        st.session_state["min_matches_maps"] = 1
        st.session_state["_min_matches_maps_auto"] = True
        st.session_state["min_matches_maps_friends"] = 1
        st.session_state["_min_matches_maps_friends_auto"] = True
        st.rerun()
    if squad_cols[1].button(
        t("filter_prev_carnage"),
        width="stretch",
        key="btn_squad_prev",
        disabled=no_friends or no_squad_sessions,
    ):
        current = st.session_state.get("picked_squad_session_label", "(toutes)")
        if not squad_options:
            _select_squad_via_button("(toutes)")
        elif current == "(toutes)" or current not in squad_options:
            _select_squad_via_button(squad_options[0])
        else:
            idx = squad_options.index(current)
            _select_squad_via_button(squad_options[min(idx + 1, len(squad_options) - 1)])
        st.rerun()

    st.selectbox(
        t("filter_squad_session_label"),
        options=["(toutes)"] + squad_options,
        format_func=lambda x: t("sel_all_categories") if x == "(toutes)" else x,
        key="picked_squad_session_label",
        disabled=no_friends or no_squad_sessions,
    )

    # ── Détection changement escouade → reset solo via pending ──────────────
    _post_squad = st.session_state.get("picked_squad_session_label", "(toutes)")
    if _post_squad != pre_squad and _post_squad != "(toutes)":
        st.session_state["_pending_solo_session_label"] = "(toutes)"
        st.session_state["_pending_squad_session_label"] = _post_squad
        logger.debug("Reset solo→(toutes) via pending (escouade=%s)", _post_squad)


def _consolidate_active_selection() -> tuple[str, list[str] | None]:
    """Consolide la sélection active solo/escouade et gère le cascade reset.

    Returns:
        (active_label, picked_session_labels)
    """
    from src.app.filters_render import _cascade_reset_filters

    solo_active = st.session_state.get("picked_solo_session_label", "(toutes)")
    squad_active = st.session_state.get("picked_squad_session_label", "(toutes)")
    if solo_active != "(toutes)":
        active_label = solo_active
    elif squad_active != "(toutes)":
        active_label = squad_active
    else:
        active_label = "(toutes)"

    # Sélection active consolidée
    _prev_active = st.session_state.get("_prev_active_session_label", "(toutes)")
    if active_label != _prev_active:
        _cascade_reset_filters()
    st.session_state["_prev_active_session_label"] = active_label

    st.session_state["picked_session_label"] = active_label
    st.session_state["picked_sessions"] = [] if active_label == "(toutes)" else [active_label]
    picked_session_labels = None if active_label == "(toutes)" else [active_label]
    if active_label != _prev_active:
        logger.info("Session sélectionnée: %s", active_label)

    return active_label, picked_session_labels


# ---------------------------------------------------------------------------
# Fonction publique — orchestrateur
# ---------------------------------------------------------------------------


def _render_session_filter(  # noqa: PLR0913
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    aliases_key: int | None,
    base_for_filters: pl.DataFrame,
    build_friends_opts_map_fn: Callable,
    *,
    prefetched_friends: tuple[str, ...] | None = None,
    prefetched_sessions: pl.DataFrame | None = None,
) -> tuple[int, list[str] | None, pl.DataFrame, tuple[str, ...] | None]:
    """Rend les contrôles en mode Sessions : sous-sections En solo / Mon escouade."""
    from src.app.filters_render import GAP_MINUTES_FIXED

    base_s_ui, solo_options, squad_options, friends_tuple, friends_xuids = _load_session_context(
        db_path,
        xuid,
        db_key,
        aliases_key,
        build_friends_opts_map_fn,
        prefetched_friends=prefetched_friends,
        prefetched_sessions=prefetched_sessions,
    )

    pre_solo, pre_squad = _init_session_state_keys(solo_options, squad_options)
    _render_solo_section(solo_options, pre_solo)
    _render_squad_section(squad_options, no_friends=len(friends_xuids) == 0, pre_squad=pre_squad)
    _, picked_session_labels = _consolidate_active_selection()

    return GAP_MINUTES_FIXED, picked_session_labels, base_s_ui, friends_tuple
