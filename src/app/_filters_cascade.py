"""Rendu des filtres cascade Playlists → Modes → Cartes (v5.2/v5.3).

Extrait de filters_render.py pour respecter la limite de taille des modules.
Contient : _get_experience_type_options, _apply_experience_filter,
           _reconcile_filter_options, _render_cascade_filters.
"""

from __future__ import annotations

from collections.abc import Callable
from datetime import date

import polars as pl
import streamlit as st

from src.app._filters_helpers import _safe_to_date, _to_polars
from src.ui import translate_playlist_name
from src.ui.components import (
    get_firefight_playlists,
    render_checkbox_filter,
    render_hierarchical_checkbox_filter,
)
from src.ui.i18n import get_lang, t
from src.ui.vectorize_helpers import build_mapping

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


def _render_cascade_filters(  # noqa: C901, PLR0912, PLR0913, PLR0915
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
