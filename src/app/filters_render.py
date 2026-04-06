"""Rendu des filtres sidebar extraits de main() pour simplification.

Ce module gère:
- Le rendu complet de la section filtres dans la sidebar
- La logique de sélection Période / Sessions
- Les filtres cascade Playlists -> Modes -> Cartes
"""

from __future__ import annotations

import logging
from collections.abc import Callable
from dataclasses import dataclass
from datetime import date

import polars as pl
import streamlit as st

from src.app.helpers import normalize_map_label, normalize_mode_label
from src.app.session_keys import SK

logger = logging.getLogger(__name__)

from src.app._filters_cascade import (
    _EXPERIENCE_TYPES_OPTIONS,  # noqa: F401 — re-export pour compatibilité
    _apply_experience_filter,  # noqa: F401 — re-export pour compatibilité
    _get_experience_type_options,
    _reconcile_filter_options,  # noqa: F401 — re-export pour compatibilité
    _render_cascade_filters,
)
from src.app._filters_period import _render_period_filter
from src.app._filters_session import _apply_default_last_session, _render_session_filter
from src.app._page_context import FilterSidebarCallbacks
from src.ui import translate_playlist_name
from src.ui.cache import cached_compute_sessions_db
from src.ui.filter_state import (
    _get_player_key,
    apply_filter_preferences,
    load_filter_preferences,
    save_filter_preferences,
)
from src.ui.i18n import get_lang, t
from src.utils.polars_compat import ensure_polars as _to_polars

GAP_MINUTES_FIXED = 120  # Figé (sessions stockées en base, cf. SESSIONS_STOCKAGE_PLAN.md)

# Préfixes des clés widget individuelles créées par render_checkbox_filter /
# render_hierarchical_checkbox_filter pour les filtres cascade.
# Ces clés doivent être nettoyées lors d'un cascade reset pour éviter que
# Streamlit réutilise d'anciennes valeurs (ex: checkbox décochée) et écrase
# la réinitialisation programmée de la sélection.
_CASCADE_WIDGET_KEY_PREFIXES = (
    "filter_experience_types_cb_",
    "filter_experience_types_all",
    "filter_experience_types_none",
    "filter_experience_types_confirm",
    "filter_experience_types_cancel",
    "filter_playlists_cb_",
    "filter_playlists_all",
    "filter_playlists_none",
    "filter_playlists_confirm",
    "filter_playlists_cancel",
    "filter_modes_cb_",
    "filter_modes_cat_",
    "filter_modes_mode_",
    "filter_modes_all",
    "filter_modes_none",
    "filter_modes_confirm",
    "filter_modes_cancel",
    "filter_maps_cb_",
    "filter_maps_all",
    "filter_maps_none",
    "filter_maps_confirm",
    "filter_maps_cancel",
)


def _cascade_reset_filters() -> None:
    """Réinitialise COMPLÈTEMENT les filtres cascade (expérience/playlists/modes/cartes).

    Supprime :
    - Les clés agrégées (filter_experience_types, filter_playlists, filter_modes, filter_maps)
    - Les clés de mode/exclusions intent-based
    - Les clés widget individuelles des checkboxes (filter_playlists_cb_*, etc.)

    Sans cette dernière étape, Streamlit réutiliserait les anciennes valeurs
    des checkboxes lors du prochain render (le paramètre ``value=`` est ignoré
    quand la clé widget existe déjà dans session_state).
    """
    for _k in (
        SK.FILTER_EXPERIENCE_TYPES,
        "_experience_types_exclusions",
        "_experience_types_filter_mode",
        SK.FILTER_PLAYLISTS,
        SK.FILTER_MODES,
        SK.FILTER_MAPS,
        "_playlists_exclusions",
        "_modes_exclusions",
        "_maps_exclusions",
        "_playlists_filter_mode",
        "_modes_filter_mode",
        "_maps_filter_mode",
    ):
        st.session_state.pop(_k, None)
    # Nettoyage des clés widget individuelles (checkboxes, boutons associés)
    for wk in list(st.session_state.keys()):
        if any(wk.startswith(p) for p in _CASCADE_WIDGET_KEY_PREFIXES):
            del st.session_state[wk]


from src.app._filters_apply import apply_filters  # noqa: F401


def _session_labels_ordered_by_last_match(base_s: pl.DataFrame) -> list[str]:
    """Retourne les session_label ordonnées par date du dernier match (plus récent en premier).

    Robuste au type de session_id (stocké VARCHAR ou calculé) et à la logique 4h (Cas A/B).
    """
    base_s = _to_polars(base_s)
    if (
        base_s.is_empty()
        or "start_time" not in base_s.columns
        or "session_label" not in base_s.columns
    ):
        return []
    agg = (
        base_s.group_by(["session_id", "session_label"])
        .agg(pl.col("start_time").max())
        .sort("start_time", descending=True)
    )
    return agg["session_label"].to_list()


@dataclass
class FilterState:
    """État des filtres après rendu."""

    filter_mode: str  # "Période" ou "Sessions"
    start_d: date
    end_d: date
    gap_minutes: int
    picked_session_labels: list[str] | None
    playlists_selected: list[str]
    modes_selected: list[str]
    maps_selected: list[str]
    base_s_ui: pl.DataFrame | None  # DataFrame sessions (mode Sessions)
    friends_tuple: tuple[str, ...] | None = None  # Amis pour calcul sessions (mode Sessions)
    experience_types_selected: list[str] | None = None  # Types d'expérience (v5.2)


# ── Sous-fonctions extraites de render_filters_sidebar ───────────────────


def _collect_unique_labels(df: pl.DataFrame, col: str, label_fn: Callable[[str], str]) -> list[str]:
    """Collecte les labels uniques d'une colonne, triés, après transformation."""
    if col not in df.columns:
        return []
    return sorted(
        {
            str(label_fn(x)).strip()
            for x in df.get_column(col).drop_nulls().to_list()
            if str(x).strip()
        }
    )


# Mapping (widget_key → shadow_key) pour la restauration Streamlit 1.54+.
# Avec st.navigation + st.switch_page, Streamlit peut effacer les clés de
# widgets lors d'un changement de page. Les clés shadow survivent à ces cycles.
_SHADOW_RESTORATIONS: list[tuple[str, str]] = [
    ("filter_mode", SK.FILTER_MODE_SHADOW),
    (SK.PICKED_SESSION_LABEL, "_picked_session_label_shadow"),
    ("start_date_cal", "_start_date_cal_shadow"),
    ("end_date_cal", "_end_date_cal_shadow"),
    (SK.PICKED_SOLO_SESSION_LABEL, "_picked_solo_session_label_shadow"),
    (SK.PICKED_SQUAD_SESSION_LABEL, "_picked_squad_session_label_shadow"),
]


def _restore_shadow_keys() -> None:
    """Restaure les clés widget depuis les clés shadow (robustesse Streamlit 1.54+)."""
    for widget_key, shadow_key in _SHADOW_RESTORATIONS:
        if widget_key not in st.session_state:
            shadow_val = st.session_state.get(shadow_key)
            if shadow_val is not None:
                st.session_state[widget_key] = shadow_val
    if "picked_sessions" not in st.session_state:
        ps_shadow = st.session_state.get(SK.PICKED_SESSIONS_SHADOW)
        if isinstance(ps_shadow, list):
            st.session_state["picked_sessions"] = ps_shadow


def _write_shadow_keys() -> None:
    """Capture l'état actuel des widgets dans les clés shadow (fin de rendu)."""
    for widget_key, shadow_key in _SHADOW_RESTORATIONS:
        if widget_key in st.session_state:
            st.session_state[shadow_key] = st.session_state[widget_key]
    if "picked_sessions" in st.session_state:
        st.session_state[SK.PICKED_SESSIONS_SHADOW] = st.session_state["picked_sessions"]


def _init_filter_preferences(  # noqa: PLR0913
    xuid: str,
    db_path: str,
    db_key: tuple[int, int] | None,
    aliases_key: int | None,
    filters_loaded_key: str,
    filters_db_key_key: str,
    all_options: tuple[list[str], list[str], list[str], list[str]],
) -> None:
    """Charge et applique les préférences de filtres au premier rendu, ou réinitialise si la DB a changé."""
    _all_playlists, _all_modes, _all_maps, _all_exp_types = all_options

    if filters_loaded_key not in st.session_state:
        logger.info("Filtres initialisés pour xuid=%s...", str(xuid or "")[:8])
        try:
            prefs = load_filter_preferences(xuid, db_path)
            if prefs is not None:
                apply_filter_preferences(
                    xuid,
                    db_path,
                    preferences=prefs,
                    all_playlists=_all_playlists,
                    all_modes=_all_modes,
                    all_maps=_all_maps,
                    all_experience_types=_all_exp_types,
                )
                # Si l'utilisateur suivait la dernière session (tracking), ré-appliquer
                if (
                    prefs.filter_mode == "Sessions"
                    and prefs.picked_session_label is not None
                    and (
                        prefs.picked_session_label == prefs.latest_session_label
                        or prefs.latest_session_label is None  # rétrocompat : anciennes prefs v5.1
                    )
                ):
                    _apply_default_last_session(db_path, xuid, db_key, aliases_key)
            else:
                _apply_default_last_session(db_path, xuid, db_key, aliases_key)
            st.session_state[filters_loaded_key] = True
            st.session_state[filters_db_key_key] = db_key
        except Exception:
            st.session_state[filters_loaded_key] = True
            st.session_state[filters_db_key_key] = db_key
    elif st.session_state.get(filters_db_key_key) != db_key:
        logger.info(
            "DB changée (db_key=%s → %s) pour xuid=%s, réinitialisation session filtre",
            st.session_state.get(filters_db_key_key),
            db_key,
            str(xuid or "")[:8],
        )
        st.session_state[filters_db_key_key] = db_key
        try:
            _apply_default_last_session(db_path, xuid, db_key, aliases_key)
        except Exception:
            logger.warning(
                "_apply_default_last_session échoué pour xuid=%s (non bloquant)",
                str(xuid or "")[:8],
                exc_info=True,
            )


def _consume_pending_states() -> None:
    """Consomme les clés pending (filter_mode, session_label, sessions)."""
    pending_mode = st.session_state.pop(SK.PENDING_FILTER_MODE, None)
    if pending_mode in ("Période", "Sessions"):
        st.session_state["filter_mode"] = pending_mode

    pending_label = st.session_state.pop("_pending_picked_session_label", None)
    if isinstance(pending_label, str) and pending_label:
        prev_label = st.session_state.get(SK.PICKED_SESSION_LABEL, "(toutes)")
        if pending_label != prev_label:
            _cascade_reset_filters()
        st.session_state[SK.PICKED_SESSION_LABEL] = pending_label

    pending_sessions = st.session_state.pop("_pending_picked_sessions", None)
    if isinstance(pending_sessions, list):
        st.session_state["picked_sessions"] = pending_sessions


def _prefetch_session_data(
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    aliases_key: int | None,
) -> tuple:
    """Pré-charge sessions et amis (warm cache quel que soit le mode filtre)."""
    from src.app.filters import get_friends_xuids_for_sessions

    prefetch_friends = get_friends_xuids_for_sessions(db_path, xuid.strip(), db_key, aliases_key)
    prefetch_sessions = cached_compute_sessions_db(
        db_path,
        xuid.strip(),
        db_key,
        True,
        GAP_MINUTES_FIXED,
        friends_xuids=prefetch_friends,
    )
    logger.debug(
        "Sessions pré-chargées: %d matchs, %d amis",
        len(prefetch_sessions) if prefetch_sessions is not None else 0,
        len(prefetch_friends) if prefetch_friends else 0,
    )
    return prefetch_friends, prefetch_sessions


def _auto_save_preferences(
    xuid: str,
    db_path: str,
    player_key: str,
    cascade_values: tuple[list[str], list[str], list[str]],
) -> None:
    """Sauvegarde automatique des filtres si le joueur n'a pas changé."""
    playlist_values, mode_values, map_values = cascade_values
    last_saved_key = f"_last_saved_player_{player_key}"
    if last_saved_key not in st.session_state or st.session_state[last_saved_key] == player_key:
        try:
            save_filter_preferences(
                xuid,
                db_path,
                all_playlists=playlist_values,
                all_modes=mode_values,
                all_maps=map_values,
                all_experience_types=_get_experience_type_options(),
            )
            st.session_state[last_saved_key] = player_key
        except Exception:
            pass


def _compute_all_filter_options(
    base: pl.DataFrame,
    clean_asset_label_fn: Callable[[str], str],
) -> tuple[list[str], list[str], list[str], list[str]]:
    """Calcule les options larges (base complète, hors fenêtre temporelle)."""
    _lang = get_lang()

    def _playlist_label(x: str) -> str:
        return str(translate_playlist_name(clean_asset_label_fn(x), lang=_lang))

    return (
        _collect_unique_labels(base, "playlist_name", _playlist_label),
        _collect_unique_labels(base, "pair_name", lambda x: normalize_mode_label(x, lang=_lang)),
        # Cohérence avec _add_derived_columns / _vectorize_ui_columns : préférer map_name_fr
        _collect_unique_labels(
            base,
            "map_name_fr" if ("map_name_fr" in base.columns and _lang == "fr") else "map_name",
            normalize_map_label,
        ),
        _get_experience_type_options(),
    )


def _render_filter_mode_selector() -> str:
    """Affiche le radio Période/Sessions et gère les resets UX associés."""
    if "filter_mode" not in st.session_state:
        st.session_state["filter_mode"] = "Période"
    filter_mode = st.radio(
        t("filter_selection"),
        options=["Période", "Sessions"],
        format_func=lambda x: t("filter_period") if x == "Période" else t("filter_sessions"),
        horizontal=True,
        key="filter_mode",
    )
    st.session_state[SK.FILTER_MODE_SHADOW] = filter_mode
    logger.info("Mode filtre: %s", filter_mode)
    for auto_key, val_key in (
        (SK.MIN_MATCHES_MAPS_AUTO, "min_matches_maps"),
        (SK.MIN_MATCHES_MAPS_FRIENDS_AUTO, "min_matches_maps_friends"),
    ):
        if filter_mode == "Période" and bool(st.session_state.get(auto_key)):
            st.session_state[val_key] = 5
            st.session_state[auto_key] = False
    return filter_mode


# ── Fonction principale ──────────────────────────────────────────────────


def render_filters_sidebar(  # noqa: PLR0913
    df: pl.DataFrame,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    aliases_key: int | None,
    callbacks: FilterSidebarCallbacks,
) -> FilterState:
    """Rend la section complète des filtres dans la sidebar."""
    date_range_fn = callbacks["date_range_fn"]
    clean_asset_label_fn = callbacks["clean_asset_label_fn"]
    build_friends_opts_map_fn = callbacks["build_friends_opts_map_fn"]

    df = _to_polars(df)
    st.header(t("filter_header"))
    base_for_filters = df.clone()
    dmin, dmax = date_range_fn(base_for_filters)
    player_key = _get_player_key(xuid, db_path)

    # Options larges (base complète, hors fenêtre temporelle)
    all_options = _compute_all_filter_options(base_for_filters, clean_asset_label_fn)

    _restore_shadow_keys()
    _init_filter_preferences(
        xuid,
        db_path,
        db_key,
        aliases_key,
        f"_filters_loaded_{player_key}",
        f"_filters_db_key_{player_key}",
        all_options,
    )
    _consume_pending_states()
    _prefetch_friends, _prefetch_sessions = _prefetch_session_data(
        db_path,
        xuid,
        db_key,
        aliases_key,
    )

    filter_mode = _render_filter_mode_selector()

    # Rendu conditionnel Période / Sessions
    start_d, end_d = dmin, dmax
    gap_minutes = GAP_MINUTES_FIXED
    picked_session_labels: list[str] | None = None
    base_s_ui: pl.DataFrame | None = None
    friends_tuple: tuple[str, ...] | None = None

    if filter_mode == "Période":
        start_d, end_d = _render_period_filter(dmin, dmax)
    else:
        gap_minutes, picked_session_labels, base_s_ui, friends_tuple = _render_session_filter(
            db_path,
            xuid,
            db_key,
            aliases_key,
            base_for_filters,
            build_friends_opts_map_fn,
            prefetched_friends=_prefetch_friends,
            prefetched_sessions=_prefetch_sessions,
        )

    # Filtres cascade
    (
        playlists_selected,
        modes_selected,
        maps_selected,
        experience_selected,
        playlist_values,
        mode_values,
        map_values,
        _,
    ) = _render_cascade_filters(
        base_for_filters=base_for_filters,
        filter_mode=filter_mode,
        start_d=start_d,
        end_d=end_d,
        picked_session_labels=picked_session_labels,
        base_s_ui=base_s_ui,
    )

    _auto_save_preferences(
        xuid,
        db_path,
        player_key,
        (playlist_values, mode_values, map_values),
    )
    _write_shadow_keys()

    return FilterState(
        filter_mode=filter_mode,
        start_d=start_d,
        end_d=end_d,
        gap_minutes=gap_minutes,
        picked_session_labels=picked_session_labels,
        playlists_selected=playlists_selected,
        modes_selected=modes_selected,
        maps_selected=maps_selected,
        base_s_ui=base_s_ui,
        friends_tuple=friends_tuple,
        experience_types_selected=experience_selected,
    )
