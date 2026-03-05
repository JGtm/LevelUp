"""Rendu des filtres sidebar extraits de main() pour simplification.

Ce module gère:
- Le rendu complet de la section filtres dans la sidebar
- La logique de sélection Période / Sessions
- Les filtres cascade Playlists -> Modes -> Cartes
"""

from __future__ import annotations

import contextlib
import logging
from collections.abc import Callable
from dataclasses import dataclass
from datetime import date

import polars as pl
import streamlit as st

logger = logging.getLogger(__name__)

from src.app._filters_cascade import (
    _EXPERIENCE_TYPES_OPTIONS,  # noqa: F401 — re-export pour compatibilité
    _apply_experience_filter,
    _get_experience_type_options,
    _reconcile_filter_options,  # noqa: F401 — re-export pour compatibilité
    _render_cascade_filters,
)
from src.app._filters_period import _render_period_filter
from src.app._filters_session import _apply_default_last_session, _render_session_filter
from src.ui import translate_pair_name, translate_playlist_name
from src.ui.cache import (
    cached_compute_sessions_db,
)
from src.ui.filter_state import (
    _get_player_key,
    apply_filter_preferences,
    load_filter_preferences,
    save_filter_preferences,
)
from src.ui.i18n import get_lang, t
from src.ui.vectorize_helpers import build_mapping
from src.utils.polars_compat import ensure_polars as _to_polars

GAP_MINUTES_FIXED = 120  # Figé (sessions stockées en base, cf. SESSIONS_STOCKAGE_PLAN.md)

# Préfixes des clés widget individuelles créées par render_checkbox_filter /
# render_hierarchical_checkbox_filter pour les filtres cascade.
# Ces clés doivent être nettoyées lors d'un cascade reset pour éviter que
# Streamlit réutilise d'anciennes valeurs (ex: checkbox décochée) et écrase
# la réinitialisation programmée de la sélection.
_CASCADE_WIDGET_KEY_PREFIXES = (
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
    """Réinitialise COMPLÈTEMENT les filtres cascade (playlists/modes/cartes).

    Supprime :
    - Les clés agrégées (filter_playlists, filter_modes, filter_maps)
    - Les clés de mode/exclusions intent-based
    - Les clés widget individuelles des checkboxes (filter_playlists_cb_*, etc.)

    Sans cette dernière étape, Streamlit réutiliserait les anciennes valeurs
    des checkboxes lors du prochain render (le paramètre ``value=`` est ignoré
    quand la clé widget existe déjà dans session_state).
    """
    for _k in (
        "filter_playlists",
        "filter_modes",
        "filter_maps",
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


from src.app._filters_shared import safe_to_date as _safe_to_date


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


def render_filters_sidebar(  # noqa: C901, PLR0912, PLR0913, PLR0915
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

    st.header(t("filter_header"))

    base_for_filters = df.clone()
    dmin, dmax = date_range_fn(base_for_filters)

    # Charger les filtres sauvegardés au premier rendu pour ce joueur/DB spécifique
    # Le flag est scopé par joueur/DB pour permettre le rechargement lors du changement de joueur
    player_key = _get_player_key(xuid, db_path)
    filters_loaded_key = f"_filters_loaded_{player_key}"
    # Clé db_key stockée : détecte les changements de DB (sync, backfill, CLI…)
    # indépendamment de la source de modification.
    filters_db_key_key = f"_filters_db_key_{player_key}"

    # Pré-calcul des options larges (base complète, hors fenêtre temporelle)
    # Nécessaire pour apply_filter_preferences intent-based (v5.2)
    _all_playlists = (
        sorted(
            {
                str(translate_playlist_name(clean_asset_label_fn(x), lang=get_lang())).strip()
                for x in base_for_filters.get_column("playlist_name").drop_nulls().to_list()
                if str(x).strip()
            }
        )
        if "playlist_name" in base_for_filters.columns
        else []
    )
    _all_modes = (
        sorted(
            {
                str(normalize_mode_label_fn(x)).strip()
                for x in base_for_filters.get_column("pair_name").drop_nulls().to_list()
                if str(x).strip()
            }
        )
        if "pair_name" in base_for_filters.columns
        else []
    )
    _all_maps = (
        sorted(
            {
                str(normalize_map_label_fn(x)).strip()
                for x in base_for_filters.get_column("map_name").drop_nulls().to_list()
                if str(x).strip()
            }
        )
        if "map_name" in base_for_filters.columns
        else []
    )
    _all_exp_types = _get_experience_type_options()

    # ── Robustesse Streamlit 1.54 : restauration depuis les clés shadow ──────
    # Avec st.navigation + st.switch_page, Streamlit peut effacer les clés de
    # widgets lors d'un changement de page (radio, selectbox, date_input...).
    # Les clés "_*_shadow" (non liées à un widget) survivent à ces cycles et
    # permettent de restaurer l'état exact de l'utilisateur.
    _SHADOW_RESTORATIONS: list[tuple[str, str]] = [
        ("filter_mode", "_filter_mode_shadow"),
        ("picked_session_label", "_picked_session_label_shadow"),
        ("start_date_cal", "_start_date_cal_shadow"),
        ("end_date_cal", "_end_date_cal_shadow"),
        ("picked_solo_session_label", "_picked_solo_session_label_shadow"),
        ("picked_squad_session_label", "_picked_squad_session_label_shadow"),
    ]
    for _widget_key, _shadow_key in _SHADOW_RESTORATIONS:
        if _widget_key not in st.session_state:
            _shadow_val = st.session_state.get(_shadow_key)
            if _shadow_val is not None:
                st.session_state[_widget_key] = _shadow_val
    # Restaurer picked_sessions depuis shadow (clé non-widget mais peut être lost)
    if "picked_sessions" not in st.session_state:
        _ps_shadow = st.session_state.get("_picked_sessions_shadow")
        if isinstance(_ps_shadow, list):
            st.session_state["picked_sessions"] = _ps_shadow

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
                # Si l'utilisateur suivait la dernière session (tracking), on ré-applique
                # _apply_default_last_session pour pointer vers la vraie dernière session
                # (au cas où de nouvelles sessions ont été créées depuis la sauvegarde).
                # Deux cas :
                #   1. Prefs récentes : picked == latest → l'utilisateur était sur la dernière
                #   2. Prefs anciennes (latest=None) : on assume "suivre la dernière" par défaut
                if (
                    prefs.filter_mode == "Sessions"
                    and prefs.picked_session_label is not None
                    and (
                        prefs.picked_session_label == prefs.latest_session_label
                        or prefs.latest_session_label is None  # rétrocompat : anciennes prefs v5.1
                        # ⚠️ Edge case connu : si l'utilisateur avait épinglé une vieille session
                        # avant la v5.2, elle sera réinitialisée sur la dernière session au
                        # premier démarrage post-upgrade (une seule fois, puis latest est sauvegardé)
                    )
                ):
                    _apply_default_last_session(db_path, xuid, db_key, aliases_key)
            else:
                # Aucun filtre en mémoire → charger par défaut la dernière session du joueur
                _apply_default_last_session(db_path, xuid, db_key, aliases_key)
            st.session_state[filters_loaded_key] = True
            st.session_state[filters_db_key_key] = db_key
        except Exception:
            # Ne pas bloquer si le chargement échoue
            st.session_state[filters_loaded_key] = True
            st.session_state[filters_db_key_key] = db_key
    elif st.session_state.get(filters_db_key_key) != db_key:
        # DB modifiée depuis la dernière init (sync, backfill CLI, changement de profil A→B→A…)
        # → réinitialiser uniquement le pointeur de session, sans recharger les prefs.
        logger.info(
            "DB changée (db_key=%s → %s) pour xuid=%s, réinitialisation session filtre",
            st.session_state.get(filters_db_key_key),
            db_key,
            str(xuid or "")[:8],
        )
        st.session_state[filters_db_key_key] = db_key
        with contextlib.suppress(Exception):
            _apply_default_last_session(db_path, xuid, db_key, aliases_key)

    # Consommation des états pending
    pending_mode = st.session_state.pop("_pending_filter_mode", None)
    if pending_mode in ("Période", "Sessions"):
        st.session_state["filter_mode"] = pending_mode

    pending_label = st.session_state.pop("_pending_picked_session_label", None)
    if isinstance(pending_label, str) and pending_label:
        prev_label = st.session_state.get("picked_session_label", "(toutes)")
        if pending_label != prev_label:
            # Réinitialiser les filtres cascade (playlists/modes/cartes) pour éviter
            # que les filtres de l'ancienne session ne masquent les matchs de la nouvelle.
            _cascade_reset_filters()
        st.session_state["picked_session_label"] = pending_label
    pending_sessions = st.session_state.pop("_pending_picked_sessions", None)
    if isinstance(pending_sessions, list):
        st.session_state["picked_sessions"] = pending_sessions

    # Sélecteur de mode
    if "filter_mode" not in st.session_state:
        st.session_state["filter_mode"] = "Période"
    filter_mode = st.radio(
        t("filter_selection"),
        options=["Période", "Sessions"],
        format_func=lambda x: t("filter_period") if x == "Période" else t("filter_sessions"),
        horizontal=True,
        key="filter_mode",
    )
    # Persister filter_mode dans la clé shadow (Streamlit 1.54+)
    st.session_state["_filter_mode_shadow"] = filter_mode
    logger.info("Mode filtre: %s", filter_mode)

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

    # Filtres cascade (v5.2 : retourne selected + all_options)
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
        clean_asset_label_fn=clean_asset_label_fn,
        normalize_mode_label_fn=normalize_mode_label_fn,
        normalize_map_label_fn=normalize_map_label_fn,
    )

    # Sauvegarder automatiquement les filtres si le joueur n'a pas changé depuis le dernier rendu
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

    # ── Mise à jour des clés shadow en fin de rendu ────────────────────────
    # Capture l'état actuel pour le cycle de navigation suivant.
    if "start_date_cal" in st.session_state:
        st.session_state["_start_date_cal_shadow"] = st.session_state["start_date_cal"]
    if "end_date_cal" in st.session_state:
        st.session_state["_end_date_cal_shadow"] = st.session_state["end_date_cal"]
    if "picked_session_label" in st.session_state:
        st.session_state["_picked_session_label_shadow"] = st.session_state["picked_session_label"]
    if "picked_sessions" in st.session_state:
        st.session_state["_picked_sessions_shadow"] = st.session_state["picked_sessions"]
    if "picked_solo_session_label" in st.session_state:
        st.session_state["_picked_solo_session_label_shadow"] = st.session_state[
            "picked_solo_session_label"
        ]
    if "picked_squad_session_label" in st.session_state:
        st.session_state["_picked_squad_session_label_shadow"] = st.session_state[
            "picked_squad_session_label"
        ]

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


def apply_filters(  # noqa: C901, PLR0912, PLR0913, PLR0915
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

    from src.utils.log_config import log_duration

    # Taille initiale pour diagnostic
    _dff_initial_len = len(dff)

    with perf_section("filters/apply"), log_duration("filters/apply", logger, threshold_ms=50):
        if filter_state.filter_mode == "Sessions":
            # Utiliser base_s_ui depuis FilterState (source de vérité = ce qui était affiché)
            # pour éviter toute désynchronisation avec un recalcul via cached_compute_sessions_db.
            # Fallback sur un recalcul uniquement si base_s_ui n'est pas disponible.
            _bsu = (
                _to_polars(filter_state.base_s_ui) if filter_state.base_s_ui is not None else None
            )
            if _bsu is not None and not _bsu.is_empty():
                base_s = _bsu
            elif db_path and xuid:
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
            else:
                base_s = pl.DataFrame()

            if not base_s.is_empty():
                # base_s n'a que match_id, session_id, session_label
                # On filtre dff par les match_id des sessions sélectionnées.
                if filter_state.picked_session_labels:
                    session_subset = base_s.filter(
                        pl.col("session_label").is_in(filter_state.picked_session_labels)
                    )
                else:
                    session_subset = base_s
                session_match_ids = set(session_subset["match_id"].cast(pl.Utf8).to_list())
                # Garde-fou : si l'intersection est vide (incohérence inattendue),
                # ne pas filtrer plutôt que retourner un dff vide.
                candidate = dff.filter(
                    pl.col("match_id").cast(pl.Utf8).is_in(list(session_match_ids))
                )
                if not candidate.is_empty():
                    dff = candidate
                # else : dff reste inchangé (tous les matchs) — cas anormal
        else:
            dff = dff.clone()

        _dff_after_session = len(dff)

        # Colonnes dérivées (nécessitent playlist_name, pair_name, map_name)
        # Vectorisation: build_mapping + replace_strict au lieu de map_elements
        derived_exprs: list[pl.Expr] = []
        if "playlist_name" in dff.columns:
            if "playlist_fr" not in dff.columns:
                _pfr_map = build_mapping(
                    dff["playlist_name"], lambda x: translate_playlist_name(x, lang=get_lang())
                )
                derived_exprs.append(
                    pl.col("playlist_name")
                    .cast(pl.Utf8)
                    .replace_strict(_pfr_map, default=None, return_dtype=pl.Utf8)
                    .alias("playlist_fr")
                )
            if "playlist_ui" not in dff.columns:
                _pui_map = build_mapping(
                    dff["playlist_name"],
                    lambda x: translate_playlist_name(clean_asset_label_fn(x), lang=get_lang()),
                )
                derived_exprs.append(
                    pl.col("playlist_name")
                    .cast(pl.Utf8)
                    .replace_strict(_pui_map, default=None, return_dtype=pl.Utf8)
                    .alias("playlist_ui")
                )
        if "pair_name" in dff.columns:
            if "pair_fr" not in dff.columns:
                _pair_map = build_mapping(
                    dff["pair_name"], lambda x: translate_pair_name(x, lang=get_lang())
                )
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
    # Filtre type d'expérience (pré-filtre, v5.2)
    _dff_after_experience = len(dff)
    if filter_state.experience_types_selected and len(filter_state.experience_types_selected) < len(
        _get_experience_type_options()
    ):
        _exp_all_pls = (
            sorted(
                {
                    str(x).strip()
                    for x in dff.get_column("playlist_ui").drop_nulls().to_list()
                    if str(x).strip()
                }
            )
            if "playlist_ui" in dff.columns
            else []
        )
        dff = _apply_experience_filter(dff, filter_state.experience_types_selected, _exp_all_pls)
        _dff_after_experience = len(dff)

    # Conversion explicite set→list pour compatibilité Polars is_in()
    _playlists_list = (
        sorted(filter_state.playlists_selected) if filter_state.playlists_selected else []
    )
    _modes_list = sorted(filter_state.modes_selected) if filter_state.modes_selected else []
    _maps_list = sorted(filter_state.maps_selected) if filter_state.maps_selected else []

    _dff_after_playlists = len(dff)
    if _playlists_list:
        before = len(dff)
        _cand_pl = dff.filter(pl.col("playlist_ui").fill_null("").is_in(_playlists_list))
        if not _cand_pl.is_empty():
            dff = _cand_pl
        # Garde-fou : si le filtre playlists viderait dff (valeurs stales/incohérentes), on l'ignore.
        # Cela se produit quand playlist_ui dans dff et dans _render_cascade_filters divergent
        # (ex: get_lang() différent entre les deux contextes, translation race condition).
        _dff_after_playlists = len(dff)
        if show_debug:
            st.write(f"🔍 Après filtre playlists: {before} → {len(dff)} matchs")
    _dff_after_modes = len(dff)
    if _modes_list:
        before = len(dff)
        _cand_mo = dff.filter(pl.col("mode_ui").fill_null("").is_in(_modes_list))
        if not _cand_mo.is_empty():
            dff = _cand_mo
        _dff_after_modes = len(dff)
        if show_debug:
            st.write(f"🔍 Après filtre modes: {before} → {len(dff)} matchs")
    _dff_after_maps = len(dff)
    if _maps_list:
        before = len(dff)
        _cand_ma = dff.filter(pl.col("map_ui").fill_null("").is_in(_maps_list))
        if not _cand_ma.is_empty():
            dff = _cand_ma
        _dff_after_maps = len(dff)
        if show_debug:
            st.write(f"🔍 Après filtre cartes: {before} → {len(dff)} matchs")

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

    # ── Diagnostic automatique : si dff vide alors que l'input avait des données ──
    if dff.is_empty() and _dff_initial_len > 0:
        with st.expander("⚠️ Diagnostic : aucune donnée après filtrage", expanded=True):
            st.write(f"**Filter mode** : `{filter_state.filter_mode}`")
            st.write(f"**Matchs initiaux** : {_dff_initial_len}")
            st.write(f"**Après filtre sessions** : {_dff_after_session}")
            st.write(f"**Après filtre expérience** : {_dff_after_experience}")
            st.write(f"**Après filtre playlists** : {_dff_after_playlists}")
            st.write(f"**Après filtre modes** : {_dff_after_modes}")
            st.write(f"**Après filtre cartes** : {_dff_after_maps}")
            st.write(f"**picked_session_labels** : `{filter_state.picked_session_labels}`")
            if filter_state.base_s_ui is not None:
                _bsu_diag = _to_polars(filter_state.base_s_ui)
                st.write(f"**base_s_ui** : {len(_bsu_diag)} lignes, colonnes={_bsu_diag.columns}")
                if "session_label" in _bsu_diag.columns:
                    _labels_in_bsu = _bsu_diag["session_label"].unique().to_list()[:10]
                    st.write(f"**Labels dans base_s_ui** (10 premiers) : {_labels_in_bsu}")
            else:
                st.write("**base_s_ui** : None")
            st.write(
                f"**Playlists sélectionnées** ({type(filter_state.playlists_selected).__name__}) : {_playlists_list[:5]}"
            )
            st.write(
                f"**Modes sélectionnés** ({type(filter_state.modes_selected).__name__}) : {_modes_list[:5]}"
            )
            st.write(
                f"**Cartes sélectionnées** ({type(filter_state.maps_selected).__name__}) : {_maps_list[:5]}"
            )
            st.write(f"**Experience types** : {filter_state.experience_types_selected}")

    logger.info("Filtres appliqués: %d → %d matchs", _dff_initial_len, len(dff))
    return dff
