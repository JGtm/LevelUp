"""Rendu des filtres cascade Playlists → Modes → Cartes (v5.2/v5.3).

Extrait de filters_render.py pour respecter la limite de taille des modules.
Contient : _get_experience_type_options, _apply_experience_filter,
           _reconcile_filter_options, _render_cascade_filters.
"""

from __future__ import annotations

from datetime import date

import polars as pl
import streamlit as st

from src.app.i18n_columns import add_i18n_display_columns
from src.app.session_keys import SK
from src.ui.components import (
    get_firefight_playlists,
    render_checkbox_filter,
    render_hierarchical_checkbox_filter,
)
from src.ui.i18n import get_lang, t
from src.utils.polars_compat import ensure_polars as _to_polars

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


# ---------------------------------------------------------------------------
# Helpers extraits de _render_cascade_filters (refacto qualité)
# ---------------------------------------------------------------------------


def _unique_sorted_values(df: pl.DataFrame, col: str) -> list[str]:
    """Extrait les valeurs uniques non-nulles triées d'une colonne."""
    return sorted({str(x).strip() for x in df[col].drop_nulls().to_list() if str(x).strip()})


def _apply_preferred_order(values: list[str], preferred: list[str]) -> list[str]:
    """Réordonne values en mettant les éléments preferred en tête."""
    return [p for p in preferred if p in values] + [p for p in values if p not in preferred]


def _apply_temporal_filter(  # noqa: PLR0913
    dropdown_base: pl.DataFrame,
    filter_mode: str,
    start_d: date,
    end_d: date,
    picked_session_labels: list[str] | None,
    base_s_ui: pl.DataFrame | None,
) -> pl.DataFrame:
    """Filtre temporel sur dropdown_base (période ou sessions)."""
    from src.app._filters_shared import safe_to_date as _safe_to_date

    if filter_mode == "Période":
        start_val = _safe_to_date(start_d)
        end_val = _safe_to_date(end_d)
        if "date" in dropdown_base.columns:
            dropdown_base = dropdown_base.filter(
                (pl.col("date").cast(pl.Date) >= start_val)
                & (pl.col("date").cast(pl.Date) <= end_val)
            )
    elif base_s_ui is not None:
        s_ui = _to_polars(base_s_ui)
        if picked_session_labels:
            session_match_ids = s_ui.filter(pl.col("session_label").is_in(picked_session_labels))[
                "match_id"
            ]
        else:
            session_match_ids = s_ui["match_id"]
        allowed_ids = set(session_match_ids.cast(pl.Utf8).to_list())
        dropdown_base = dropdown_base.filter(
            pl.col("match_id").cast(pl.Utf8).is_in(list(allowed_ids))
        )
    return dropdown_base


def _compute_faceted_options(
    dropdown_base_filtered: pl.DataFrame,
    preferred_order: list[str],
) -> tuple[list[str], list[str], list[str]]:
    """Calcule les options facettées par dimension croisée (v5.3).

    Chaque dimension n'affiche que les options ayant ≥1 match dans le contexte
    des sélections actives dans les AUTRES dimensions.
    """
    _sel_modes = st.session_state.get(SK.FILTER_MODES)
    _sel_maps = st.session_state.get(SK.FILTER_MAPS)
    _sel_playlists = st.session_state.get(SK.FILTER_PLAYLISTS)

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
    playlist_faceted = _apply_preferred_order(
        _unique_sorted_values(_base_pl, "playlist_ui"), preferred_order
    )

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
    mode_faceted = _unique_sorted_values(_base_mo, "mode_ui")

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
    map_faceted = _unique_sorted_values(_base_ma, "map_ui")

    return playlist_faceted, mode_faceted, map_faceted


def _render_dimension_with_restore(
    label: str,
    options_faceted: list[str],
    session_key: str,
    *,
    use_hierarchical: bool = False,
    default_unchecked: list[str] | None = None,
) -> set[str]:
    """Rend un filtre checkbox facetté avec le pattern Save-Render-Restore.

    Le pattern préserve les items temporairement cachés par le facettage :
    render_checkbox_filter fait current & set(options), donc sans restauration
    les items hors-facette seraient perdus de session_state.
    """
    hidden = st.session_state.get(session_key, set()) - set(options_faceted)
    if use_hierarchical:
        render_hierarchical_checkbox_filter(
            label=label,
            options=options_faceted,
            session_key=session_key,
            expanded=False,
        )
    else:
        render_checkbox_filter(
            label=label,
            options=options_faceted,
            session_key=session_key,
            expanded=False,
            **({"default_unchecked": default_unchecked} if default_unchecked else {}),
        )
    st.session_state[session_key] = st.session_state.get(session_key, set()) | hidden
    return st.session_state[session_key]


# ---------------------------------------------------------------------------
# Fonction principale (orchestrateur)
# ---------------------------------------------------------------------------


_CascadeResult = tuple[
    list[str],
    list[str],
    list[str],
    list[str],  # selected: playlists, modes, maps, exp
    list[str],
    list[str],
    list[str],
    list[str],  # all: playlist_values, mode_values, map_values, exp_values
]


def _render_cascade_filters(  # noqa: PLR0913
    base_for_filters: pl.DataFrame,
    filter_mode: str,
    start_d: date,
    end_d: date,
    picked_session_labels: list[str] | None,
    base_s_ui: pl.DataFrame | None,
) -> _CascadeResult:
    """Rend les filtres Type d'expérience + Playlists + Modes + Cartes (v5.2)."""
    dropdown_base = _to_polars(base_for_filters)
    dropdown_base = _apply_temporal_filter(
        dropdown_base,
        filter_mode,
        start_d,
        end_d,
        picked_session_labels,
        base_s_ui,
    )
    dropdown_base = add_i18n_display_columns(dropdown_base, get_lang())

    # ── Sélecteur Type d'expérience (pré-filtre, v5.2) ──────────────────────
    playlist_values_all = _unique_sorted_values(dropdown_base, "playlist_ui")
    exp_values = _get_experience_type_options()
    _reconcile_filter_options(
        SK.FILTER_EXPERIENCE_TYPES,
        exp_values,
        "_experience_types_filter_mode",
        "_experience_types_exclusions",
    )
    experience_selected = render_checkbox_filter(
        label="Type",
        options=exp_values,
        session_key=SK.FILTER_EXPERIENCE_TYPES,
        expanded=False,
    )
    dropdown_base_filtered = _apply_experience_filter(
        dropdown_base,
        experience_selected,
        playlist_values_all,
    )

    # ── Options complètes (réconciliation + sauvegarde) ──────────────────────
    preferred_order = [
        t("playlist_quick_play"),
        t("playlist_ranked_arena"),
        t("playlist_ranked_assassin"),
    ]
    playlist_values = _apply_preferred_order(
        _unique_sorted_values(dropdown_base_filtered, "playlist_ui"),
        preferred_order,
    )
    mode_values = _unique_sorted_values(dropdown_base_filtered, "mode_ui")
    map_values = _unique_sorted_values(dropdown_base_filtered, "map_ui")

    for key, vals, mode_k, excl_k in [
        (SK.FILTER_PLAYLISTS, playlist_values, "_playlists_filter_mode", "_playlists_exclusions"),
        (SK.FILTER_MODES, mode_values, "_modes_filter_mode", "_modes_exclusions"),
        (SK.FILTER_MAPS, map_values, "_maps_filter_mode", "_maps_exclusions"),
    ]:
        _reconcile_filter_options(key, vals, mode_k, excl_k)

    # ── Rendu facetté (v5.3) ─────────────────────────────────────────────────
    pl_faceted, mode_faceted, map_faceted = _compute_faceted_options(
        dropdown_base_filtered,
        preferred_order,
    )
    playlists_selected = _render_dimension_with_restore(
        t("filter_playlists"),
        pl_faceted,
        SK.FILTER_PLAYLISTS,
        default_unchecked=get_firefight_playlists(pl_faceted),
    )
    modes_selected = _render_dimension_with_restore(
        t("filter_modes"),
        mode_faceted,
        SK.FILTER_MODES,
        use_hierarchical=True,
    )
    maps_selected = _render_dimension_with_restore(
        t("filter_maps"),
        map_faceted,
        SK.FILTER_MAPS,
    )

    return (
        playlists_selected,
        modes_selected,
        maps_selected,
        experience_selected,
        playlist_values,
        mode_values,
        map_values,
        exp_values,
    )
