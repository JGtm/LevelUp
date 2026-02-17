"""Rendu des filtres sidebar extraits de main() pour simplification.

Ce module gère:
- Le rendu complet de la section filtres dans la sidebar
- La logique de sélection Période / Sessions
- Les filtres cascade Playlists -> Modes -> Cartes
"""

from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass
from datetime import date

import polars as pl
import streamlit as st

from src.app.filters import get_friends_xuids_for_sessions
from src.ui import translate_pair_name, translate_playlist_name
from src.ui.cache import (
    cached_compute_sessions_db,
    cached_same_team_match_ids_with_friend,
)
from src.ui.components import (
    get_firefight_playlists,
    render_checkbox_filter,
    render_hierarchical_checkbox_filter,
)
from src.ui.filter_state import (
    _get_player_key,
    apply_filter_preferences,
    load_filter_preferences,
    save_filter_preferences,
)
from src.ui.vectorize_helpers import build_mapping

GAP_MINUTES_FIXED = 120  # Figé (sessions stockées en base, cf. SESSIONS_STOCKAGE_PLAN.md)


def _to_polars(df: object) -> pl.DataFrame:
    """Convertit un DataFrame Pandas en Polars si nécessaire (pont de sécurité)."""
    if isinstance(df, pl.DataFrame):
        return df
    try:
        return pl.from_pandas(df)  # type: ignore[arg-type]
    except Exception:
        return pl.DataFrame()


def _safe_to_date(val: object) -> date:
    """Convertit une valeur en date Python, date.today() si invalide."""
    if isinstance(val, date):
        return val
    try:
        from dateutil.parser import parse as _parse_dt

        return _parse_dt(str(val)).date()
    except (ValueError, TypeError, ImportError):
        return date.today()


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


def render_filters_sidebar(
    df: pl.DataFrame,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    aliases_key: int | None,
    date_range_fn: Callable[[pl.DataFrame], tuple[date, date]],
    clean_asset_label_fn: Callable[[str], str],
    normalize_mode_label_fn: Callable[[str], str],
    normalize_map_label_fn: Callable[[str], str],
    build_friends_opts_map_fn: Callable,
) -> FilterState:
    """Rend la section complète des filtres dans la sidebar.

    Returns:
        FilterState avec tous les paramètres de filtrage sélectionnés.
    """
    df = _to_polars(df)

    st.header("Filtres")

    base_for_filters = df.clone()
    dmin, dmax = date_range_fn(base_for_filters)

    # Charger les filtres sauvegardés au premier rendu pour ce joueur/DB spécifique
    # Le flag est scopé par joueur/DB pour permettre le rechargement lors du changement de joueur
    player_key = _get_player_key(xuid, db_path)
    filters_loaded_key = f"_filters_loaded_{player_key}"

    if filters_loaded_key not in st.session_state:
        try:
            prefs = load_filter_preferences(xuid, db_path)
            if prefs is not None:
                apply_filter_preferences(xuid, db_path, preferences=prefs)
            else:
                # Aucun filtre en mémoire → charger par défaut la dernière session du joueur
                _apply_default_last_session(db_path, xuid, db_key, aliases_key)
            st.session_state[filters_loaded_key] = True
        except Exception:
            # Ne pas bloquer si le chargement échoue
            st.session_state[filters_loaded_key] = True

    # Consommation des états pending
    pending_mode = st.session_state.pop("_pending_filter_mode", None)
    if pending_mode in ("Période", "Sessions"):
        st.session_state["filter_mode"] = pending_mode

    pending_label = st.session_state.pop("_pending_picked_session_label", None)
    if isinstance(pending_label, str) and pending_label:
        st.session_state["picked_session_label"] = pending_label
    pending_sessions = st.session_state.pop("_pending_picked_sessions", None)
    if isinstance(pending_sessions, list):
        st.session_state["picked_sessions"] = pending_sessions

    # Sélecteur de mode
    if "filter_mode" not in st.session_state:
        st.session_state["filter_mode"] = "Période"
    filter_mode = st.radio(
        "Sélection",
        options=["Période", "Sessions"],
        horizontal=True,
        key="filter_mode",
    )

    # UX: reset min_matches_maps en mode Période
    if filter_mode == "Période" and bool(st.session_state.get("_min_matches_maps_auto")):
        st.session_state["min_matches_maps"] = 5
        st.session_state["_min_matches_maps_auto"] = False
    if filter_mode == "Période" and bool(st.session_state.get("_min_matches_maps_friends_auto")):
        st.session_state["min_matches_maps_friends"] = 5
        st.session_state["_min_matches_maps_friends_auto"] = False

    # Valeurs par défaut
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
        )

    # Filtres cascade
    playlists_selected, modes_selected, maps_selected = _render_cascade_filters(
        base_for_filters=base_for_filters,
        filter_mode=filter_mode,
        start_d=start_d,
        end_d=end_d,
        picked_session_labels=picked_session_labels,
        base_s_ui=base_s_ui,
        clean_asset_label_fn=clean_asset_label_fn,
        normalize_mode_label_fn=normalize_mode_label_fn,
        normalize_map_label_fn=normalize_map_label_fn,
    )

    # Sauvegarder automatiquement les filtres si le joueur n'a pas changé depuis le dernier rendu
    # Cela permet de persister les modifications de filtres sans intervention manuelle
    last_saved_key = f"_last_saved_player_{player_key}"
    if last_saved_key not in st.session_state or st.session_state[last_saved_key] == player_key:
        # Sauvegarder les filtres actuels (sans bloquer si la sauvegarde échoue)
        try:
            save_filter_preferences(xuid, db_path)
            st.session_state[last_saved_key] = player_key
        except Exception:
            # Ne pas bloquer l'application si la sauvegarde échoue
            pass

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
    )


def _apply_default_last_session(
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    aliases_key: int | None = None,
) -> None:
    """Applique par défaut la dernière session du joueur quand aucun filtre n'est en mémoire.

    Utilisé au premier chargement ou changement de joueur/db.
    """
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
            "Début",
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
            "Fin",
            min_value=start_limit_date,
            max_value=end_default_date,
            format="DD/MM/YYYY",
            key="end_date_cal",
        )
    if start_date > end_date:
        st.warning("La date de début est après la date de fin.")
    return start_date, end_date


def _render_session_filter(
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    aliases_key: int | None,
    base_for_filters: pl.DataFrame,
    build_friends_opts_map_fn: Callable,
) -> tuple[int, list[str] | None, pl.DataFrame, tuple[str, ...] | None]:
    """Rend les contrôles en mode Sessions (gap fixé à 120 min, stockage en base)."""
    gap_minutes = GAP_MINUTES_FIXED

    friends_tuple = get_friends_xuids_for_sessions(db_path, xuid.strip(), db_key, aliases_key)
    base_s_ui = _to_polars(
        cached_compute_sessions_db(
            db_path,
            xuid.strip(),
            db_key,
            True,
            gap_minutes,
            friends_xuids=friends_tuple,
        )
    )
    options_ui = _session_labels_ordered_by_last_match(base_s_ui)
    st.session_state["_latest_session_label"] = options_ui[0] if options_ui else None

    def _set_session_selection(label: str) -> None:
        st.session_state.picked_session_label = label
        if label == "(toutes)":
            st.session_state.picked_sessions = []
        elif label in options_ui:
            st.session_state.picked_sessions = [label]

    if "picked_session_label" not in st.session_state:
        _set_session_selection(options_ui[0] if options_ui else "(toutes)")
    if "picked_sessions" not in st.session_state:
        st.session_state.picked_sessions = options_ui[:1] if options_ui else []

    # Boutons de navigation
    cols = st.columns(2)
    if cols[0].button("Dernière session", width="stretch"):
        _set_session_selection(options_ui[0] if options_ui else "(toutes)")
        st.session_state["min_matches_maps"] = 1
        st.session_state["_min_matches_maps_auto"] = True
        st.session_state["min_matches_maps_friends"] = 1
        st.session_state["_min_matches_maps_friends_auto"] = True
    if cols[1].button("Session précédente", width="stretch"):
        current = st.session_state.get("picked_session_label", "(toutes)")
        if not options_ui:
            _set_session_selection("(toutes)")
        elif current == "(toutes)" or current not in options_ui:
            _set_session_selection(options_ui[0])
        else:
            idx = options_ui.index(current)
            next_idx = min(idx + 1, len(options_ui) - 1)
            _set_session_selection(options_ui[next_idx])

    # Trio
    trio_label = _compute_trio_label(
        db_path,
        xuid,
        db_key,
        aliases_key,
        base_for_filters,
        base_s_ui,
        options_ui,
        build_friends_opts_map_fn,
    )
    _render_trio_button(trio_label)

    # Sélecteur de session
    picked_one = st.selectbox(
        "Session", options=["(toutes)"] + options_ui, key="picked_session_label"
    )
    picked_session_labels = None if picked_one == "(toutes)" else [picked_one]

    return gap_minutes, picked_session_labels, base_s_ui, friends_tuple


@st.cache_data(show_spinner=False, ttl=120)
def _cached_get_trio_match_ids(
    db_path: str,
    xuid: str,
    f1_xuid: str,
    f2_xuid: str,
    db_key: tuple[int, int] | None,
) -> tuple[str, ...]:
    """Récupère les match IDs où les 3 joueurs sont dans la même équipe (cachée).

    Cette fonction est coûteuse car elle fait 2 requêtes SQL.
    Le cache TTL de 120s évite les recalculs fréquents.
    """
    try:
        ids_m = set(
            cached_same_team_match_ids_with_friend(db_path, xuid.strip(), f1_xuid, db_key=db_key)
        )
        ids_c = set(
            cached_same_team_match_ids_with_friend(db_path, xuid.strip(), f2_xuid, db_key=db_key)
        )
        trio_ids = ids_m & ids_c
        return tuple(sorted(trio_ids))
    except Exception:
        return ()


def _compute_trio_label(
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    aliases_key: int | None,
    base_for_filters: pl.DataFrame,
    base_s_ui: pl.DataFrame,
    options_ui: list[str],
    build_friends_opts_map_fn: Callable,
) -> str | None:
    """Calcule le label de la dernière session en trio.

    Optimisé: utilise un cache TTL pour éviter les recalculs coûteux
    des requêtes SQL à chaque rendu.
    """
    try:
        base_for_filters = _to_polars(base_for_filters)
        base_s_ui = _to_polars(base_s_ui)
        # Récupérer les amis sélectionnés (déjà caché via @st.cache_data)
        friends_opts_map, friends_default_labels = build_friends_opts_map_fn(
            db_path, xuid.strip(), db_key, aliases_key
        )
        picked_friend_labels = st.session_state.get("friends_picked_labels")
        if not isinstance(picked_friend_labels, list) or not picked_friend_labels:
            picked_friend_labels = friends_default_labels
        picked_xuids = [
            friends_opts_map[lbl] for lbl in picked_friend_labels if lbl in friends_opts_map
        ]
        if len(picked_xuids) < 2:
            return None

        f1_xuid, f2_xuid = picked_xuids[0], picked_xuids[1]

        # Utiliser la fonction cachée avec TTL pour les requêtes coûteuses
        trio_ids_tuple = _cached_get_trio_match_ids(db_path, xuid.strip(), f1_xuid, f2_xuid, db_key)
        if not trio_ids_tuple:
            return None

        trio_ids = set(trio_ids_tuple)

        # Filtrer par les matchs disponibles dans base_for_filters
        base_match_ids = set(base_for_filters["match_id"].cast(pl.Utf8).to_list())
        trio_ids = trio_ids & base_match_ids

        if not trio_ids:
            return None

        # Trouver la dernière session trio (par max(start_time), pas session_id)
        trio_rows = base_s_ui.filter(pl.col("match_id").cast(pl.Utf8).is_in(list(trio_ids)))
        if trio_rows.is_empty() or "start_time" not in trio_rows.columns:
            return None

        agg = (
            trio_rows.group_by(["session_id", "session_label"])
            .agg(pl.col("start_time").max())
            .sort("start_time", descending=True)
        )
        if not agg.is_empty():
            return agg["session_label"][0]
    except Exception:
        pass
    return None


def _render_trio_button(trio_label: str | None) -> None:
    """Rend le bouton Dernière session en trio."""
    st.session_state["_trio_latest_session_label"] = trio_label
    disabled_trio = not isinstance(trio_label, str) or not trio_label
    if st.button("Dernière session en trio", width="stretch", disabled=disabled_trio):
        st.session_state["_pending_filter_mode"] = "Sessions"
        st.session_state["_pending_picked_session_label"] = trio_label
        st.session_state["_pending_picked_sessions"] = [trio_label]
        st.session_state["min_matches_maps"] = 1
        st.session_state["_min_matches_maps_auto"] = True
        st.session_state["min_matches_maps_friends"] = 1
        st.session_state["_min_matches_maps_friends_auto"] = True
        st.rerun()
    if not disabled_trio:
        st.caption(f"Trio : {trio_label}")


def _render_cascade_filters(
    base_for_filters: pl.DataFrame,
    filter_mode: str,
    start_d: date,
    end_d: date,
    picked_session_labels: list[str] | None,
    base_s_ui: pl.DataFrame | None,
    clean_asset_label_fn: Callable[[str], str],
    normalize_mode_label_fn: Callable[[str], str],
    normalize_map_label_fn: Callable[[str], str],
) -> tuple[list[str], list[str], list[str]]:
    """Rend les filtres cascade Playlists -> Modes -> Cartes."""
    dropdown_base = _to_polars(base_for_filters)

    if filter_mode == "Période":
        start_val = _safe_to_date(start_d)
        end_val = _safe_to_date(end_d)
        if "date" in dropdown_base.columns:
            dropdown_base = dropdown_base.filter(
                (pl.col("date").cast(pl.Date) >= start_val)
                & (pl.col("date").cast(pl.Date) <= end_val)
            )
    else:
        # En mode Sessions, base_s_ui n'a que session_id/session_label (pas playlist_name, etc.)
        # On restreint base_for_filters aux match_id des sessions sélectionnées.
        if base_s_ui is not None:
            s_ui = _to_polars(base_s_ui)
            if picked_session_labels:
                session_match_ids = s_ui.filter(
                    pl.col("session_label").is_in(picked_session_labels)
                )["match_id"]
            else:
                session_match_ids = s_ui["match_id"]
            allowed_ids = set(session_match_ids.cast(pl.Utf8).to_list())
            dropdown_base = dropdown_base.filter(
                pl.col("match_id").cast(pl.Utf8).is_in(list(allowed_ids))
            )

    # Vectorisation: build_mapping + replace_strict au lieu de map_elements
    _pl_map = build_mapping(
        dropdown_base["playlist_name"],
        lambda x: translate_playlist_name(clean_asset_label_fn(x)),
    )
    _mode_map = build_mapping(dropdown_base["pair_name"], normalize_mode_label_fn)
    _map_map = build_mapping(dropdown_base["map_name"], normalize_map_label_fn)
    dropdown_base = dropdown_base.with_columns(
        pl.col("playlist_name")
        .cast(pl.Utf8)
        .replace_strict(_pl_map, default=None, return_dtype=pl.Utf8)
        .alias("playlist_ui"),
        pl.col("pair_name")
        .cast(pl.Utf8)
        .replace_strict(_mode_map, default=None, return_dtype=pl.Utf8)
        .alias("mode_ui"),
        pl.col("map_name")
        .cast(pl.Utf8)
        .replace_strict(_map_map, default=None, return_dtype=pl.Utf8)
        .alias("map_ui"),
    )

    # --- Playlists ---
    playlist_values = sorted(
        {
            str(x).strip()
            for x in dropdown_base["playlist_ui"].drop_nulls().to_list()
            if str(x).strip()
        }
    )
    preferred_order = ["Partie rapide", "Arène classée", "Assassin classé"]
    playlist_values = [p for p in preferred_order if p in playlist_values] + [
        p for p in playlist_values if p not in preferred_order
    ]

    firefight_playlists = get_firefight_playlists(playlist_values)
    playlists_selected = render_checkbox_filter(
        label="Playlists",
        options=playlist_values,
        session_key="filter_playlists",
        default_unchecked=firefight_playlists,
        expanded=False,
    )

    # Scope après filtre playlist
    scope1 = dropdown_base
    if playlists_selected and len(playlists_selected) < len(playlist_values):
        scope1 = scope1.filter(pl.col("playlist_ui").fill_null("").is_in(playlists_selected))

    # --- Modes ---
    mode_values = sorted(
        {str(x).strip() for x in scope1["mode_ui"].drop_nulls().to_list() if str(x).strip()}
    )
    modes_selected = render_hierarchical_checkbox_filter(
        label="Modes",
        options=mode_values,
        session_key="filter_modes",
        expanded=False,
    )

    # Scope après filtre mode
    scope2 = scope1
    if modes_selected and len(modes_selected) < len(mode_values):
        scope2 = scope2.filter(pl.col("mode_ui").fill_null("").is_in(modes_selected))

    # --- Cartes ---
    map_values = sorted(
        {str(x).strip() for x in scope2["map_ui"].drop_nulls().to_list() if str(x).strip()}
    )
    maps_selected = render_checkbox_filter(
        label="Cartes",
        options=map_values,
        session_key="filter_maps",
        expanded=False,
    )

    return playlists_selected, modes_selected, maps_selected


def apply_filters(
    dff: pl.DataFrame,
    filter_state: FilterState | dict | None,
    db_path: str | None = None,
    xuid: str | None = None,
    db_key: tuple[int, int] | None = None,
    clean_asset_label_fn: Callable[[str], str] | None = None,
    normalize_mode_label_fn: Callable[[str], str] | None = None,
    normalize_map_label_fn: Callable[[str], str] | None = None,
) -> pl.DataFrame:
    """Applique tous les filtres au DataFrame.

    Args:
        dff: DataFrame Polars de base.
        filter_state: État des filtres depuis render_filters_sidebar.

    Returns:
        DataFrame Polars filtré.
    """
    from src.ui.perf import perf_section

    def _identity(s: str) -> str:
        return s

    dff = _to_polars(dff)

    # Compat tests/migration : si filter_state n'est pas un FilterState,
    # on ne filtre pas.
    if not isinstance(filter_state, FilterState):
        return dff.clone()

    if clean_asset_label_fn is None:
        clean_asset_label_fn = _identity
    if normalize_mode_label_fn is None:
        normalize_mode_label_fn = _identity
    if normalize_map_label_fn is None:
        normalize_map_label_fn = _identity
    if db_path is None:
        db_path = ""
    if xuid is None:
        xuid = ""

    with perf_section("filters/apply"):
        if filter_state.filter_mode == "Sessions" and db_path and xuid:
            base_s = _to_polars(
                cached_compute_sessions_db(
                    db_path,
                    xuid.strip(),
                    db_key,
                    True,
                    filter_state.gap_minutes,
                    friends_xuids=filter_state.friends_tuple,
                )
            )
            # base_s n'a que match_id, session_id, session_label (pas playlist_name, etc.)
            # On filtre dff par les match_id des sessions sélectionnées au lieu de remplacer dff.
            if filter_state.picked_session_labels:
                session_subset = base_s.filter(
                    pl.col("session_label").is_in(filter_state.picked_session_labels)
                )
            else:
                session_subset = base_s
            session_match_ids = set(session_subset["match_id"].cast(pl.Utf8).to_list())
            dff = dff.filter(pl.col("match_id").cast(pl.Utf8).is_in(list(session_match_ids)))
        else:
            dff = dff.clone()

        # Colonnes dérivées (nécessitent playlist_name, pair_name, map_name)
        # Vectorisation: build_mapping + replace_strict au lieu de map_elements
        derived_exprs: list[pl.Expr] = []
        if "playlist_name" in dff.columns:
            if "playlist_fr" not in dff.columns:
                _pfr_map = build_mapping(dff["playlist_name"], translate_playlist_name)
                derived_exprs.append(
                    pl.col("playlist_name")
                    .cast(pl.Utf8)
                    .replace_strict(_pfr_map, default=None, return_dtype=pl.Utf8)
                    .alias("playlist_fr")
                )
            if "playlist_ui" not in dff.columns:
                _pui_map = build_mapping(
                    dff["playlist_name"],
                    lambda x: translate_playlist_name(clean_asset_label_fn(x)),
                )
                derived_exprs.append(
                    pl.col("playlist_name")
                    .cast(pl.Utf8)
                    .replace_strict(_pui_map, default=None, return_dtype=pl.Utf8)
                    .alias("playlist_ui")
                )
        if "pair_name" in dff.columns:
            if "pair_fr" not in dff.columns:
                _pair_map = build_mapping(dff["pair_name"], translate_pair_name)
                derived_exprs.append(
                    pl.col("pair_name")
                    .cast(pl.Utf8)
                    .replace_strict(_pair_map, default=None, return_dtype=pl.Utf8)
                    .alias("pair_fr")
                )
            if "mode_ui" not in dff.columns:
                # Optimisation: si normalize_mode_label_fn est _identity, utiliser cast
                if normalize_mode_label_fn is _identity:
                    derived_exprs.append(pl.col("pair_name").cast(pl.Utf8).alias("mode_ui"))
                else:
                    _mui_map = build_mapping(dff["pair_name"], normalize_mode_label_fn)
                    derived_exprs.append(
                        pl.col("pair_name")
                        .cast(pl.Utf8)
                        .replace_strict(_mui_map, default=None, return_dtype=pl.Utf8)
                        .alias("mode_ui")
                    )
        if "map_name" in dff.columns and "map_ui" not in dff.columns:
            # Optimisation: si normalize_map_label_fn est _identity, utiliser cast
            if normalize_map_label_fn is _identity:
                derived_exprs.append(pl.col("map_name").cast(pl.Utf8).alias("map_ui"))
            else:
                _mapui_map = build_mapping(dff["map_name"], normalize_map_label_fn)
                derived_exprs.append(
                    pl.col("map_name")
                    .cast(pl.Utf8)
                    .replace_strict(_mapui_map, default=None, return_dtype=pl.Utf8)
                    .alias("map_ui")
                )
        if derived_exprs:
            dff = dff.with_columns(derived_exprs)
        # Ajouter les colonnes vides manquantes
        for col_name in ("playlist_fr", "playlist_ui", "pair_fr", "mode_ui", "map_ui"):
            if col_name not in dff.columns:
                dff = dff.with_columns(pl.lit("").alias(col_name))

    # Debug: Afficher l'état des filtres avant application
    show_debug = st.session_state.get("_show_debug_info", False)
    if show_debug:
        st.write(
            f"🔍 **Debug filtres** - Avant application des filtres checkboxes: {len(dff)} matchs"
        )
        st.write(
            f"- Playlists sélectionnées: {filter_state.playlists_selected if filter_state.playlists_selected else 'Toutes'}"
        )
        st.write(
            f"- Modes sélectionnés: {filter_state.modes_selected if filter_state.modes_selected else 'Tous'}"
        )
        st.write(
            f"- Cartes sélectionnées: {filter_state.maps_selected if filter_state.maps_selected else 'Toutes'}"
        )
        if "start_time" in dff.columns:
            recent = dff.sort("start_time", descending=True).head(5)
            st.write("**5 matchs les plus récents avant filtres checkboxes:**")
            for row in recent.iter_rows(named=True):
                map_ui = row.get("map_ui") or normalize_map_label_fn(row.get("map_name", ""))
                playlist_ui = row.get("playlist_ui") or row.get("playlist_name", "")
                mode_ui = row.get("mode_ui") or normalize_mode_label_fn(row.get("pair_name", ""))
                st.write(
                    f"- {row.get('start_time')} | Map: {map_ui} | Playlist: {playlist_ui} | Mode: {mode_ui}"
                )

    # Application des filtres checkboxes
    if filter_state.playlists_selected:
        before = len(dff)
        dff = dff.filter(pl.col("playlist_ui").fill_null("").is_in(filter_state.playlists_selected))
        if show_debug:
            st.write(f"🔍 Après filtre playlists: {before} → {len(dff)} matchs")
            null_count = dff["playlist_ui"].is_null().sum()
            if null_count > 0:
                st.warning(f"⚠️ {null_count} matchs avec playlist_ui=NULL exclus par le filtre")
    if filter_state.modes_selected:
        before = len(dff)
        dff = dff.filter(pl.col("mode_ui").fill_null("").is_in(filter_state.modes_selected))
        if show_debug:
            st.write(f"🔍 Après filtre modes: {before} → {len(dff)} matchs")
            null_count = dff["mode_ui"].is_null().sum()
            if null_count > 0:
                st.warning(f"⚠️ {null_count} matchs avec mode_ui=NULL exclus par le filtre")
    if filter_state.maps_selected:
        before = len(dff)
        dff = dff.filter(pl.col("map_ui").fill_null("").is_in(filter_state.maps_selected))
        if show_debug:
            st.write(f"🔍 Après filtre cartes: {before} → {len(dff)} matchs")
            null_count = dff["map_ui"].is_null().sum()
            if null_count > 0:
                st.warning(f"⚠️ {null_count} matchs avec map_ui=NULL exclus par le filtre")

    if filter_state.filter_mode == "Période":
        before = len(dff)
        start_val = _safe_to_date(filter_state.start_d)
        end_val = _safe_to_date(filter_state.end_d)
        if "date" in dff.columns:
            dff = dff.filter(
                (pl.col("date").cast(pl.Date) >= start_val)
                & (pl.col("date").cast(pl.Date) <= end_val)
            )
        if show_debug:
            st.write(
                f"🔍 Après filtre période ({filter_state.start_d} à {filter_state.end_d}): {before} → {len(dff)} matchs"
            )

    return dff
