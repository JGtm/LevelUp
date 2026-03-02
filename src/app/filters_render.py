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
    experience_types_selected: list[str] | None = None  # Types d'expérience (v5.2)


# ---------------------------------------------------------------------------
# Sélecteur Type d'expérience — liste statique (v5.2)
# ---------------------------------------------------------------------------

_EXPERIENCE_TYPES_OPTIONS: list[str] = ["PVP non classé", "PVP classé", "PVE"]


def _get_experience_type_options() -> list[str]:
    """Retourne les options de type d'expérience dans la langue active."""
    return [t("exp_pvp_unranked"), t("exp_pvp_ranked"), t("exp_pve")]


def _apply_experience_filter(
    dropdown_base: pl.DataFrame,
    experience_selected: list[str],
    all_playlist_values: list[str],
) -> pl.DataFrame:
    """Pré-filtre dropdown_base selon les types d'expérience sélectionnés.

    Doit être appelé AVANT le calcul de playlist_values, mode_values, map_values.
    Les playlists Firefight sont détectées via get_firefight_playlists().

    Args:
        dropdown_base: DataFrame base avec colonnes playlist_ui (déjà traduit).
        experience_selected: Types cochés parmi _get_experience_type_options().
        all_playlist_values: Toutes les playlist_ui disponibles (pour détecter firefight).

    Returns:
        DataFrame filtré.
    """
    if not experience_selected or len(experience_selected) >= len(_get_experience_type_options()):
        return dropdown_base  # Tout coché → pas de filtre

    firefight_pls = set(get_firefight_playlists(all_playlist_values))
    pve_cond = pl.col("playlist_ui").cast(pl.Utf8).fill_null("").is_in(list(firefight_pls))
    ranked_cond = (
        pl.col("playlist_ui")
        .cast(pl.Utf8)
        .fill_null("")
        .str.to_lowercase()
        .str.contains("classé|ranked")
        & ~pve_cond
    )

    conds: list[pl.Expr] = []
    for exp_type in experience_selected:
        if exp_type == t("exp_pve"):
            conds.append(pve_cond)
        elif exp_type == t("exp_pvp_ranked"):
            conds.append(ranked_cond)
        elif exp_type == t("exp_pvp_unranked"):
            conds.append(~pve_cond & ~ranked_cond)

    if not conds:
        return dropdown_base

    combined = conds[0]
    for c in conds[1:]:
        combined = combined | c
    return dropdown_base.filter(combined)


def _reconcile_filter_options(
    filter_key: str,
    options: list[str],
    mode_key: str,
    exclusions_key: str,
) -> None:
    """Auto-coche les options vraiment nouvelles en mode exclude.

    "Vraiment nouvelle" = dans `options`, absente de session_state[filter_key]
    ET absente des exclusions explicites. Cela garantit que :
    - Les nouvelles playlists/modes/cartes ajoutés par un sync sont auto-cochés ✓
    - Les options que l'utilisateur a délibérément décochées restent décochées ✓
    - En mode include, rien ne change (nouvelles options restent décochées) ✓

    Doit être appelé AVANT render_checkbox_filter.

    Args:
        filter_key: Clé session_state du filtre (ex: "filter_playlists").
        options: Toutes les options disponibles actuellement.
        mode_key: Clé session_state du mode intent (ex: "_playlists_filter_mode").
        exclusions_key: Clé session_state des exclusions (ex: "_playlists_exclusions").
    """
    if filter_key not in st.session_state:
        return  # Pas encore initialisé → render_checkbox_filter s'en charge
    if st.session_state.get(mode_key, "include") != "exclude":
        return
    exclusions: set[str] = st.session_state.get(exclusions_key, set())
    current: set[str] = st.session_state[filter_key]
    truly_new = set(options) - current - exclusions
    if truly_new:
        st.session_state[filter_key] = current | truly_new


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

    st.header(t("filter_header"))

    base_for_filters = df.clone()
    dmin, dmax = date_range_fn(base_for_filters)

    # Charger les filtres sauvegardés au premier rendu pour ce joueur/DB spécifique
    # Le flag est scopé par joueur/DB pour permettre le rechargement lors du changement de joueur
    player_key = _get_player_key(xuid, db_path)
    filters_loaded_key = f"_filters_loaded_{player_key}"

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
        except Exception:
            # Ne pas bloquer si le chargement échoue
            st.session_state[filters_loaded_key] = True

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


def _render_session_filter(
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    aliases_key: int | None,
    base_for_filters: pl.DataFrame,
    build_friends_opts_map_fn: Callable,
) -> tuple[int, list[str] | None, pl.DataFrame, tuple[str, ...] | None]:
    """Rend les contrôles en mode Sessions : sous-sections En solo / Mon escouade.

    - "En solo"      : sessions où aucun ami du multiselect n'était présent.
    - "Mon escouade" : sessions où au moins un ami était présent.

    Les sélections sont mutuellement exclusives : choisir dans une section
    réinitialise l'autre à "(toutes)".
    """
    gap_minutes = GAP_MINUTES_FIXED

    # ── Chargement des sessions (avec teammates_signature pour classification) ──
    friends_tuple = get_friends_xuids_for_sessions(db_path, xuid.strip(), db_key, aliases_key)
    base_s_raw = _to_polars(
        cached_compute_sessions_db(
            db_path,
            xuid.strip(),
            db_key,
            True,
            gap_minutes,
            friends_xuids=friends_tuple,
        )
    )

    # Vue allégée (sans teammates_signature) pour la compatibilité du reste de l'app
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

    # ── Résoudre les amis depuis l'UI (multiselect Teammates persisted) ─────
    friends_opts_map, friends_default_labels = build_friends_opts_map_fn(
        db_path, xuid.strip(), db_key, aliases_key
    )
    ui_picked_labels: list[str] = (
        st.session_state.get("teammates_picked_labels") or friends_default_labels
    )
    friends_xuids_for_classify = frozenset(
        friends_opts_map[lbl] for lbl in ui_picked_labels if lbl in friends_opts_map
    )

    # ── Classification solo / escouade ──────────────────────────────────────
    solo_options, squad_options = _classify_sessions_solo_squad(
        base_s_raw, friends_xuids_for_classify
    )

    # ── Helpers boutons (écrivent UNIQUEMENT des clés _pending_*) ───────────
    # ARCHITECTURE : aucun on_change sur les selectboxes solo/escouade.
    # Raison : les callbacks on_change s'exécutent dans un contexte où
    # Streamlit peut considérer les clés widget du cycle précédent comme
    # « instanciées », provoquant StreamlitAPIException.
    # L'exclusion mutuelle est gérée par :
    #   - Boutons → pending keys + st.rerun()
    #   - Selectbox solo → détection de changement AVANT le rendu du squad
    #   - Selectbox squad → pending key + st.rerun()

    def _select_solo_via_button(label: str) -> None:
        """Écrit des clés pending pour sélection solo — appelé par boutons uniquement."""
        st.session_state["_pending_solo_session_label"] = label
        st.session_state["_pending_squad_session_label"] = "(toutes)"

    def _select_squad_via_button(label: str) -> None:
        """Écrit des clés pending pour sélection escouade — appelé par boutons uniquement."""
        st.session_state["_pending_squad_session_label"] = label
        st.session_state["_pending_solo_session_label"] = "(toutes)"

    # ── Initialisation des clés de sélection ────────────────────────────────
    if "picked_solo_session_label" not in st.session_state:
        # Héritage depuis picked_session_label (rétrocompabilité prefs / apply_default)
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

    # Cohérence : remettre dans la liste si session disparue (ex: changement d'amis)
    if st.session_state.get("picked_solo_session_label") not in ["(toutes)"] + solo_options:
        st.session_state["picked_solo_session_label"] = (
            solo_options[0] if solo_options else "(toutes)"
        )
    if st.session_state.get("picked_squad_session_label") not in ["(toutes)"] + squad_options:
        st.session_state["picked_squad_session_label"] = "(toutes)"

    # ── Consommation des clés pending (AVANT tout widget) ───────────────────
    # Les boutons et le reset croisé squad→solo écrivent dans _pending_*.
    # On les consomme ici pour écrire dans les clés widget en toute sécurité
    # (les selectboxes ne sont pas encore instanciés à ce stade).
    _pending_solo = st.session_state.pop("_pending_solo_session_label", None)
    if _pending_solo is not None:
        st.session_state["picked_solo_session_label"] = _pending_solo
    _pending_squad = st.session_state.pop("_pending_squad_session_label", None)
    if _pending_squad is not None:
        st.session_state["picked_squad_session_label"] = _pending_squad

    # ── Capture pré-rendu pour détection de changement ──────────────────────
    _pre_solo = st.session_state.get("picked_solo_session_label", "(toutes)")
    _pre_squad = st.session_state.get("picked_squad_session_label", "(toutes)")

    # ── Sous-section "En solo" ───────────────────────────────────────────────
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

    # Selectbox solo — PAS de on_change (voir commentaire architecture ci-dessus)
    st.selectbox(
        t("filter_solo_session_label"),
        options=["(toutes)"] + solo_options,
        format_func=lambda x: t("sel_all_categories") if x == "(toutes)" else x,
        key="picked_solo_session_label",
    )

    # ── Détection changement solo → reset escouade ──────────────────────────
    # Sûr : le selectbox squad n'est pas encore instancié, on peut écrire
    # directement dans sa clé widget.
    _post_solo = st.session_state.get("picked_solo_session_label", "(toutes)")
    if _post_solo != _pre_solo and _post_solo != "(toutes)":
        st.session_state["picked_squad_session_label"] = "(toutes)"

    # ── Sous-section "Mon escouade" ──────────────────────────────────────────
    st.subheader(t("filter_squad_title"))
    st.divider()
    no_friends = len(friends_xuids_for_classify) == 0
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

    # Selectbox escouade — PAS de on_change (voir commentaire architecture ci-dessus)
    st.selectbox(
        t("filter_squad_session_label"),
        options=["(toutes)"] + squad_options,
        format_func=lambda x: t("sel_all_categories") if x == "(toutes)" else x,
        key="picked_squad_session_label",
        disabled=no_friends or no_squad_sessions,
    )

    # ── Détection changement escouade → reset solo au prochain cycle ────────
    # Le selectbox solo est DÉJÀ instancié → écriture directe interdite.
    # On passe par pending + rerun pour que la consommation se fasse au
    # début du prochain run (avant l'instanciation des widgets).
    # IMPORTANT : on persiste aussi la valeur escouade dans _pending_squad pour
    # qu'elle survive à la vérification de cohérence du prochain run (ligne 757).
    # Sans ça, si squad_options diffère légèrement entre les runs (ex: amis recomputed),
    # la cohérence peut resetter picked_squad_session_label à "(toutes)" AVANT la
    # consommation du pending solo, effaçant la sélection utilisateur.
    _post_squad = st.session_state.get("picked_squad_session_label", "(toutes)")
    if _post_squad != _pre_squad and _post_squad != "(toutes)":
        st.session_state["_pending_solo_session_label"] = "(toutes)"
        st.session_state["_pending_squad_session_label"] = _post_squad
        st.rerun()

    # ── Sélection active consolidée ──────────────────────────────────────────
    solo_active = st.session_state.get("picked_solo_session_label", "(toutes)")
    squad_active = st.session_state.get("picked_squad_session_label", "(toutes)")
    if solo_active != "(toutes)":
        active_label = solo_active
    elif squad_active != "(toutes)":
        active_label = squad_active
    else:
        active_label = "(toutes)"

    # Cascade reset des filtres ssi la session active effective a changé
    _prev_active = st.session_state.get("_prev_active_session_label", "(toutes)")
    if active_label != _prev_active:
        _cascade_reset_filters()
    st.session_state["_prev_active_session_label"] = active_label

    # Garder picked_session_label cohérent pour les autres parties de l'app
    st.session_state["picked_session_label"] = active_label
    st.session_state["picked_sessions"] = [] if active_label == "(toutes)" else [active_label]
    picked_session_labels = None if active_label == "(toutes)" else [active_label]

    return gap_minutes, picked_session_labels, base_s_ui, friends_tuple


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
) -> tuple[
    list[str],
    list[str],
    list[str],
    list[str],  # selected: playlists, modes, maps, exp
    list[str],
    list[str],
    list[str],
    list[str],  # all: playlist_values, mode_values, map_values, exp_values
]:
    """Rend les filtres Type d'expérience + Playlists + Modes + Cartes (v5.2).

    Changements v5.2 :
    - Ajout sélecteur "Type d'expérience" comme pré-filtre sur dropdown_base
    - Suppression de la cascade scope1/scope2 : modes et cartes calculés depuis
      dropdown_base complet (après fenêtre temporelle), pas depuis la sélection playlist
    - Réconciliation mid-session via _reconcile_filter_options avant chaque render
    - Retour 8-tuple : (selected x4, all_values x4) pour intent-based save
    """
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
        # En mode Sessions, restreindre aux match_id des sessions sélectionnées
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

    # Vectorisation: build_mapping + replace_strict
    # IMPORTANT : utiliser lang=get_lang() pour rester cohérent avec apply_filters.
    # Sans cela, si get_lang()!="fr", les playlist_ui divergent entre les deux fonctions,
    # ce qui fait que le filtre playlists élimine tous les matchs du mode Sessions.
    _lang = get_lang()
    if "playlist_name" in dropdown_base.columns:
        _pl_map = build_mapping(
            dropdown_base["playlist_name"],
            lambda x: translate_playlist_name(clean_asset_label_fn(x), lang=_lang),
        )
        pl_expr = (
            pl.col("playlist_name")
            .cast(pl.Utf8)
            .replace_strict(_pl_map, default=None, return_dtype=pl.Utf8)
            .alias("playlist_ui")
        )
    else:
        pl_expr = pl.lit(None).cast(pl.Utf8).alias("playlist_ui")

    if "pair_name" in dropdown_base.columns:
        _mode_map = build_mapping(dropdown_base["pair_name"], normalize_mode_label_fn)
        mode_expr = (
            pl.col("pair_name")
            .cast(pl.Utf8)
            .replace_strict(_mode_map, default=None, return_dtype=pl.Utf8)
            .alias("mode_ui")
        )
    else:
        mode_expr = pl.lit(None).cast(pl.Utf8).alias("mode_ui")

    if "map_name" in dropdown_base.columns:
        _map_map = build_mapping(dropdown_base["map_name"], normalize_map_label_fn)
        map_expr = (
            pl.col("map_name")
            .cast(pl.Utf8)
            .replace_strict(_map_map, default=None, return_dtype=pl.Utf8)
            .alias("map_ui")
        )
    else:
        map_expr = pl.lit(None).cast(pl.Utf8).alias("map_ui")

    dropdown_base = dropdown_base.with_columns([pl_expr, mode_expr, map_expr])

    # ── Sélecteur Type d'expérience (pré-filtre, v5.2) ──────────────────────
    # Calculer playlist_values pour détecter les firefight AVANT le pré-filtre
    playlist_values_all = sorted(
        {
            str(x).strip()
            for x in dropdown_base["playlist_ui"].drop_nulls().to_list()
            if str(x).strip()
        }
    )
    exp_values = _get_experience_type_options()

    _reconcile_filter_options(
        "filter_experience_types",
        exp_values,
        "_experience_types_filter_mode",
        "_experience_types_exclusions",
    )
    experience_selected = render_checkbox_filter(
        label="Type",
        options=exp_values,
        session_key="filter_experience_types",
        expanded=False,
    )

    # Appliquer le pré-filtre expérience sur dropdown_base
    dropdown_base_filtered = _apply_experience_filter(
        dropdown_base, experience_selected, playlist_values_all
    )

    # ── Options complètes (base pour réconciliation + sauvegarde) ────────────
    playlist_values = sorted(
        {
            str(x).strip()
            for x in dropdown_base_filtered["playlist_ui"].drop_nulls().to_list()
            if str(x).strip()
        }
    )
    preferred_order = [
        t("playlist_quick_play"),
        t("playlist_ranked_arena"),
        t("playlist_ranked_assassin"),
    ]
    playlist_values = [p for p in preferred_order if p in playlist_values] + [
        p for p in playlist_values if p not in preferred_order
    ]

    mode_values = sorted(
        {
            str(x).strip()
            for x in dropdown_base_filtered["mode_ui"].drop_nulls().to_list()
            if str(x).strip()
        }
    )

    map_values = sorted(
        {
            str(x).strip()
            for x in dropdown_base_filtered["map_ui"].drop_nulls().to_list()
            if str(x).strip()
        }
    )

    # Réconciliation mid-session avec options COMPLÈTES (détecte les vraiment nouvelles)
    _reconcile_filter_options(
        "filter_playlists",
        playlist_values,
        "_playlists_filter_mode",
        "_playlists_exclusions",
    )
    _reconcile_filter_options(
        "filter_modes",
        mode_values,
        "_modes_filter_mode",
        "_modes_exclusions",
    )
    _reconcile_filter_options(
        "filter_maps",
        map_values,
        "_maps_filter_mode",
        "_maps_exclusions",
    )

    # ── Filtrage facetté (v5.3) ───────────────────────────────────────────────
    # Chaque dimension n'affiche que les options ayant ≥1 match dans le contexte
    # des sélections actives dans les AUTRES dimensions.
    # None = sentinelle "dimension non encore initialisée" → aucun filtre appliqué.
    _sel_modes = st.session_state.get("filter_modes")
    _sel_maps = st.session_state.get("filter_maps")
    _sel_playlists = st.session_state.get("filter_playlists")

    # Playlists facettées : filtré par modes + cartes actuels
    _base_pl = dropdown_base_filtered
    if _sel_modes is not None and "mode_ui" in _base_pl.columns:
        _base_pl = _base_pl.filter(
            pl.col("mode_ui").is_in(list(_sel_modes)) | pl.col("mode_ui").is_null()
        )
    if _sel_maps is not None and "map_ui" in _base_pl.columns:
        _base_pl = _base_pl.filter(
            pl.col("map_ui").is_in(list(_sel_maps)) | pl.col("map_ui").is_null()
        )
    playlist_values_faceted = sorted(
        {str(x).strip() for x in _base_pl["playlist_ui"].drop_nulls().to_list() if str(x).strip()}
    )
    playlist_values_faceted = [p for p in preferred_order if p in playlist_values_faceted] + [
        p for p in playlist_values_faceted if p not in preferred_order
    ]

    # Modes facettés : filtré par playlists + cartes actuels
    _base_mo = dropdown_base_filtered
    if _sel_playlists is not None and "playlist_ui" in _base_mo.columns:
        _base_mo = _base_mo.filter(
            pl.col("playlist_ui").is_in(list(_sel_playlists)) | pl.col("playlist_ui").is_null()
        )
    if _sel_maps is not None and "map_ui" in _base_mo.columns:
        _base_mo = _base_mo.filter(
            pl.col("map_ui").is_in(list(_sel_maps)) | pl.col("map_ui").is_null()
        )
    mode_values_faceted = sorted(
        {str(x).strip() for x in _base_mo["mode_ui"].drop_nulls().to_list() if str(x).strip()}
    )

    # Cartes facettées : filtré par playlists + modes actuels
    _base_ma = dropdown_base_filtered
    if _sel_playlists is not None and "playlist_ui" in _base_ma.columns:
        _base_ma = _base_ma.filter(
            pl.col("playlist_ui").is_in(list(_sel_playlists)) | pl.col("playlist_ui").is_null()
        )
    if _sel_modes is not None and "mode_ui" in _base_ma.columns:
        _base_ma = _base_ma.filter(
            pl.col("mode_ui").is_in(list(_sel_modes)) | pl.col("mode_ui").is_null()
        )
    map_values_faceted = sorted(
        {str(x).strip() for x in _base_ma["map_ui"].drop_nulls().to_list() if str(x).strip()}
    )

    # ── Playlists — rendu facetté avec Save-Render-Restore ────────────────────
    # render_checkbox_filter fait current & set(options) : sans restauration, les
    # items temporairement cachés seraient perdus de session_state.
    _hidden_playlists = st.session_state.get("filter_playlists", set()) - set(
        playlist_values_faceted
    )
    render_checkbox_filter(
        label=t("filter_playlists"),
        options=playlist_values_faceted,
        session_key="filter_playlists",
        default_unchecked=get_firefight_playlists(playlist_values_faceted),
        expanded=False,
    )
    st.session_state["filter_playlists"] = (
        st.session_state.get("filter_playlists", set()) | _hidden_playlists
    )
    playlists_selected = st.session_state["filter_playlists"]

    # ── Modes — rendu facetté avec Save-Render-Restore ────────────────────────
    _hidden_modes = st.session_state.get("filter_modes", set()) - set(mode_values_faceted)
    render_hierarchical_checkbox_filter(
        label=t("filter_modes"),
        options=mode_values_faceted,
        session_key="filter_modes",
        expanded=False,
    )
    st.session_state["filter_modes"] = st.session_state.get("filter_modes", set()) | _hidden_modes
    modes_selected = st.session_state["filter_modes"]

    # ── Cartes — rendu facetté avec Save-Render-Restore ──────────────────────
    _hidden_maps = st.session_state.get("filter_maps", set()) - set(map_values_faceted)
    render_checkbox_filter(
        label=t("filter_maps"),
        options=map_values_faceted,
        session_key="filter_maps",
        expanded=False,
    )
    st.session_state["filter_maps"] = st.session_state.get("filter_maps", set()) | _hidden_maps
    maps_selected = st.session_state["filter_maps"]

    return (
        playlists_selected,
        modes_selected,
        maps_selected,
        experience_selected,
        playlist_values,  # COMPLET — utilisé pour save_filter_preferences
        mode_values,  # COMPLET
        map_values,  # COMPLET
        exp_values,
    )


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

    # Taille initiale pour diagnostic
    _dff_initial_len = len(dff)

    with perf_section("filters/apply"):
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

    return dff
