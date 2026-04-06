"""Gestion des filtres sidebar pour l'application Streamlit.

Ce module gère :
- Les filtres de période (dates) et de sessions
- Les filtres en cascade (Playlist → Mode → Carte)
- L'état des filtres en session_state
"""

from __future__ import annotations

from datetime import date

import polars as pl
import streamlit as st

from src.app._filters_friends import (
    build_friends_opts_map,
    compute_trio_label,
    get_friends_xuids_for_sessions,
)
from src.app._filters_shared import safe_to_date as _safe_to_date
from src.ui.cache import (
    cached_compute_sessions_db,
)
from src.ui.i18n import t
from src.utils.polars_compat import ensure_polars as _to_polars

# Re-exports pour backward compat
__all__ = ["build_friends_opts_map", "get_friends_xuids_for_sessions"]

# =============================================================================
# Filtres DataFrame
# =============================================================================


def apply_date_filter(df: pl.DataFrame, start_d: date, end_d: date) -> pl.DataFrame:
    """Applique un filtre de dates au DataFrame.

    Args:
        df: DataFrame Polars source.
        start_d: Date de début.
        end_d: Date de fin.

    Returns:
        DataFrame Polars filtré.
    """
    df = _to_polars(df)
    if "date" not in df.columns:
        return df
    start_val = _safe_to_date(start_d)
    end_val = _safe_to_date(end_d)
    return df.filter(
        (pl.col("date").cast(pl.Date) >= start_val) & (pl.col("date").cast(pl.Date) <= end_val)
    )


def apply_checkbox_filters(
    df: pl.DataFrame,
    playlists_selected: list[str] | None,
    modes_selected: list[str] | None,
    maps_selected: list[str] | None,
) -> pl.DataFrame:
    """Applique les filtres checkbox (playlists, modes, cartes).

    Args:
        df: DataFrame Polars source avec colonnes UI.
        playlists_selected: Playlists sélectionnées ou None pour tout.
        modes_selected: Modes sélectionnés ou None pour tout.
        maps_selected: Cartes sélectionnées ou None pour tout.

    Returns:
        DataFrame Polars filtré.
    """
    df = _to_polars(df)
    if playlists_selected:
        df = df.filter(pl.col("playlist_ui").fill_null("").is_in(playlists_selected))
    if modes_selected:
        df = df.filter(pl.col("mode_ui").fill_null("").is_in(modes_selected))
    if maps_selected:
        df = df.filter(pl.col("map_ui").fill_null("").is_in(maps_selected))
    return df


# =============================================================================
# Rendu des filtres sidebar
# =============================================================================


def render_date_filters(
    dmin: date,
    dmax: date,
) -> tuple[date, date]:
    """Rend les filtres de date et retourne la sélection.

    Args:
        dmin: Date minimum disponible.
        dmax: Date maximum disponible.

    Returns:
        Tuple (start_date, end_date) sélectionnées.
    """
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
            t("flt_date_start"),
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
            t("flt_date_end"),
            min_value=start_limit_date,
            max_value=end_default_date,
            format="DD/MM/YYYY",
            key="end_date_cal",
        )
    return start_date, end_date


GAP_MINUTES_FIXED = 120  # Figé (sessions stockées en base)


def render_session_filters(  # noqa: C901
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    aliases_key: int | None,
    base_for_filters: pl.DataFrame,
) -> tuple[int, list[str] | None]:
    """Rend les filtres de session et retourne la sélection (gap fixé à 120 min)."""
    gap_minutes = GAP_MINUTES_FIXED

    friends_tuple = get_friends_xuids_for_sessions(db_path, xuid.strip(), db_key, aliases_key)
    base_s_ui = cached_compute_sessions_db(
        db_path,
        xuid.strip(),
        db_key,
        True,  # Inclure Firefight (filtrage via checkboxes)
        gap_minutes,
        friends_xuids=friends_tuple,
    )
    # Tri par date du dernier match (robuste au type session_id et à la logique 4h Cas A/B)
    if (
        not base_s_ui.is_empty()
        and "start_time" in base_s_ui.columns
        and "session_label" in base_s_ui.columns
    ):
        agg = base_s_ui.group_by(["session_id", "session_label"]).agg(pl.col("start_time").max())
        agg = agg.sort("start_time", descending=True)
        options_ui = agg["session_label"].to_list()
    else:
        options_ui = []
    st.session_state["_latest_session_label"] = options_ui[0] if options_ui else None

    def _set_session_selection(label: str) -> None:
        st.session_state.picked_session_label = label
        if label == t("flt_session_all"):
            st.session_state.picked_sessions = []
        elif label in options_ui:
            st.session_state.picked_sessions = [label]

    if "picked_session_label" not in st.session_state:
        _set_session_selection(options_ui[0] if options_ui else t("flt_session_all"))
    if "picked_sessions" not in st.session_state:
        st.session_state.picked_sessions = options_ui[:1] if options_ui else []

    cols = st.columns(2)
    if cols[0].button(t("flt_session_last"), width="stretch"):
        _set_session_selection(options_ui[0] if options_ui else t("flt_session_all"))
        st.session_state["min_matches_maps"] = 1
        st.session_state["_min_matches_maps_auto"] = True
        st.session_state["min_matches_maps_friends"] = 1
        st.session_state["_min_matches_maps_friends_auto"] = True
    if cols[1].button(t("flt_session_prev"), width="stretch"):
        current = st.session_state.get("picked_session_label", t("flt_session_all"))
        if not options_ui:
            _set_session_selection(t("flt_session_all"))
        elif current == t("flt_session_all") or current not in options_ui:
            _set_session_selection(options_ui[0])
        else:
            idx = options_ui.index(current)
            next_idx = min(idx + 1, len(options_ui) - 1)
            _set_session_selection(options_ui[next_idx])

    # Bouton Trio
    trio_label = compute_trio_label(
        db_path, xuid, db_key, aliases_key, _to_polars(base_for_filters), base_s_ui
    )
    st.session_state["_trio_latest_session_label"] = trio_label
    disabled_trio = not isinstance(trio_label, str) or not trio_label

    def _apply_trio_filter(tl: str | None = trio_label) -> None:
        st.session_state["_pending_filter_mode"] = "Sessions"
        st.session_state["_pending_picked_session_label"] = tl
        st.session_state["_pending_picked_sessions"] = [tl]
        st.session_state["min_matches_maps"] = 1
        st.session_state["_min_matches_maps_auto"] = True
        st.session_state["min_matches_maps_friends"] = 1
        st.session_state["_min_matches_maps_friends_auto"] = True

    st.button(
        t("flt_session_trio"),
        width="stretch",
        disabled=disabled_trio,
        on_click=_apply_trio_filter,
    )
    if not disabled_trio:
        st.caption(t("flt_trio_caption", label=trio_label))

    picked_one = st.selectbox(
        t("flt_session_select"),
        options=[t("flt_session_all")] + options_ui,
        key="picked_session_label",
    )
    picked_session_labels = None if picked_one == t("flt_session_all") else [picked_one]

    return gap_minutes, picked_session_labels


# =============================================================================
# État des filtres
# =============================================================================


def consume_pending_filter_state() -> None:
    """Consomme l'état en attente pour les filtres (changements demandés).

    Applique les changements de mode/session stockés en session_state
    par d'autres composants (ex: boutons trio).
    """
    pending_mode = st.session_state.pop("_pending_filter_mode", None)
    if pending_mode in ("Période", "Sessions"):
        st.session_state["filter_mode"] = pending_mode

    pending_label = st.session_state.pop("_pending_picked_session_label", None)
    if isinstance(pending_label, str) and pending_label:
        st.session_state["picked_session_label"] = pending_label
    pending_sessions = st.session_state.pop("_pending_picked_sessions", None)
    if isinstance(pending_sessions, list):
        st.session_state["picked_sessions"] = pending_sessions


def reset_auto_min_matches(filter_mode: str) -> None:
    """Réinitialise les valeurs auto de min_matches si on revient en mode Période.

    Args:
        filter_mode: Mode de filtre actuel ("Période" ou "Sessions").
    """
    if filter_mode == "Période" and bool(st.session_state.get("_min_matches_maps_auto")):
        st.session_state["min_matches_maps"] = 5
        st.session_state["_min_matches_maps_auto"] = False

    if filter_mode == "Période" and bool(st.session_state.get("_min_matches_maps_friends_auto")):
        st.session_state["min_matches_maps_friends"] = 5
        st.session_state["_min_matches_maps_friends_auto"] = False
