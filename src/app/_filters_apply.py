"""Application des filtres au DataFrame — logique extraite de filters_render.py."""

from __future__ import annotations

import logging
from collections.abc import Callable

import polars as pl
import streamlit as st

from src.app._filters_cascade import _apply_experience_filter, _get_experience_type_options
from src.app._filters_shared import safe_to_date as _safe_to_date
from src.ui import translate_pair_name, translate_playlist_name
from src.ui.cache import cached_compute_sessions_db
from src.ui.i18n import get_lang
from src.ui.translations import resolve_map_display_names
from src.ui.vectorize_helpers import build_mapping
from src.utils.polars_compat import ensure_polars as _to_polars

logger = logging.getLogger(__name__)


def _warn_session_filter_empty(
    labels: list[str],
    base_s: pl.DataFrame,
    dff: pl.DataFrame,
) -> None:
    """Journalise l'incohérence quand le filtre session retourne un candidat vide.

    Indique que base_s_ui contient le label demandé mais que les match_ids
    correspondants sont absents de dff (désynchronisation de cache).
    La navigation sera corrigée via la clé session_key dans _resolve_nav_index.
    """
    logger.warning(
        "Filtre session vide pour labels=%s (base_s=%d lignes, dff=%d matchs). "
        "L'index de navigation sera réinitialisé via session_key.",
        labels,
        len(base_s),
        len(dff),
    )


def apply_filters(  # noqa: C901, PLR0912, PLR0913, PLR0915
    dff: pl.DataFrame,
    filter_state: FilterState,  # noqa: F821 — forward ref, importé lazy
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
    from src.app.filters_render import FilterState
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
                elif filter_state.picked_session_labels:
                    _warn_session_filter_empty(filter_state.picked_session_labels, base_s, dff)
        else:
            dff = dff.clone()

        _dff_after_session = len(dff)

        # Colonnes dérivées (nécessitent playlist_name, pair_name, map_name)
        # Vectorisation: build_mapping + replace_strict au lieu de map_elements
        dff = _add_derived_columns(
            dff, clean_asset_label_fn, normalize_mode_label_fn, normalize_map_label_fn
        )

    # Debug: Afficher l'état des filtres avant application
    show_debug = st.session_state.get("_show_debug_info", False)
    if show_debug:
        _show_debug_info_before(dff, filter_state, normalize_mode_label_fn, normalize_map_label_fn)

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
        _cand_exp = _apply_experience_filter(
            dff, filter_state.experience_types_selected, _exp_all_pls
        )
        if not _cand_exp.is_empty():
            dff = _cand_exp
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
        _show_empty_diagnostic(
            filter_state,
            _dff_initial_len,
            _dff_after_session,
            _dff_after_experience,
            _dff_after_playlists,
            _dff_after_modes,
            _dff_after_maps,
            _playlists_list,
            _modes_list,
            _maps_list,
        )

    logger.info("Filtres appliqués: %d → %d matchs", _dff_initial_len, len(dff))
    return dff


def _add_derived_columns(  # noqa: C901, PLR0912
    dff: pl.DataFrame,
    clean_asset_label_fn: Callable[[str], str],
    normalize_mode_label_fn: Callable[[str], str],
    normalize_map_label_fn: Callable[[str], str],
) -> pl.DataFrame:
    """Ajoute les colonnes dérivées UI (playlist_fr, mode_ui, map_ui, etc.)."""

    def _identity(s: str) -> str:
        return s

    derived_exprs: list[pl.Expr] = []
    if "playlist_name" in dff.columns:
        if "playlist_fr" not in dff.columns:
            # Utiliser playlist_name_fr (depuis v_match_full) si disponible, sinon passthrough
            if "playlist_name_fr" in dff.columns:
                derived_exprs.append(
                    pl.coalesce([
                        pl.col("playlist_name_fr").cast(pl.Utf8),
                        pl.col("playlist_name").cast(pl.Utf8),
                    ]).alias("playlist_fr")
                )
            else:
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
            # Utiliser playlist_name_fr si disponible et langue FR, sinon passthrough
            if "playlist_name_fr" in dff.columns and get_lang() == "fr":
                derived_exprs.append(
                    pl.coalesce([
                        pl.col("playlist_name_fr").cast(pl.Utf8),
                        pl.col("playlist_name").cast(pl.Utf8),
                    ]).alias("playlist_ui")
                )
            else:
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
            # Utiliser pair_name_fr (depuis v_match_full) si disponible, sinon translate
            if "pair_name_fr" in dff.columns:
                derived_exprs.append(
                    pl.coalesce([
                        pl.col("pair_name_fr").cast(pl.Utf8),
                        pl.col("pair_name").cast(pl.Utf8),
                    ]).alias("pair_fr")
                )
            else:
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
            # mode_ui : préférer game_variant_name_fr (traduction directe API),
            # sinon pair_name traduit dans la langue courante
            if "game_variant_name_fr" in dff.columns and get_lang() == "fr":
                derived_exprs.append(
                    pl.coalesce([
                        pl.col("game_variant_name_fr").cast(pl.Utf8),
                        pl.col("pair_name").cast(pl.Utf8),
                    ]).alias("mode_ui")
                )
            else:
                _mui_tr_map = build_mapping(
                    dff["pair_name"], lambda x: translate_pair_name(x, lang=get_lang())
                )
                derived_exprs.append(
                    pl.col("pair_name")
                    .cast(pl.Utf8)
                    .replace_strict(
                        _mui_tr_map,
                        default=pl.col("pair_name").cast(pl.Utf8),
                        return_dtype=pl.Utf8,
                    )
                    .alias("mode_ui")
                )
    if "map_name" in dff.columns and "map_ui" not in dff.columns:
        # Priorité 1 : map_name_fr direct (depuis mv_player_matches / v_match_full)
        if "map_name_fr" in dff.columns and get_lang() == "fr":
            derived_exprs.append(
                pl.coalesce([
                    pl.col("map_name_fr").cast(pl.Utf8),
                    pl.col("map_name").cast(pl.Utf8),
                ]).alias("map_ui")
            )
        elif "map_id" in dff.columns:
            # Traduction i18n via asset_translations (requête batch)
            lang = get_lang()
            id_to_fallback: dict[str, str] = {
                str(r["map_id"]): str(r["map_name"] or "")
                for r in dff.select(["map_id", "map_name"])
                .drop_nulls(subset=["map_id"])
                .unique(subset=["map_id"])
                .iter_rows(named=True)
            }
            if id_to_fallback:
                translated_maps = resolve_map_display_names(id_to_fallback, lang)
                n_translated = sum(1 for k, v in translated_maps.items() if v != id_to_fallback.get(k))
                logger.debug(
                    "map_ui i18n: %d/%d cartes traduites (lang=%s)",
                    n_translated,
                    len(id_to_fallback),
                    lang,
                )
                derived_exprs.append(
                    pl.col("map_id")
                    .cast(pl.Utf8)
                    .replace_strict(
                        translated_maps,
                        default=pl.col("map_name").cast(pl.Utf8),
                        return_dtype=pl.Utf8,
                    )
                    .alias("map_ui")
                )
            else:
                derived_exprs.append(pl.col("map_name").cast(pl.Utf8).alias("map_ui"))
        elif normalize_map_label_fn is _identity:
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
    return dff


def _show_debug_info_before(
    dff: pl.DataFrame,
    filter_state: FilterState,  # noqa: F821
    normalize_mode_label_fn: Callable[[str], str],
    normalize_map_label_fn: Callable[[str], str],
) -> None:
    """Affiche les infos de debug avant application des filtres checkboxes."""
    st.write(f"🔍 **Debug filtres** - Avant application des filtres checkboxes: {len(dff)} matchs")
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


def _show_empty_diagnostic(  # noqa: PLR0913
    filter_state: FilterState,  # noqa: F821
    initial_len: int,
    after_session: int,
    after_experience: int,
    after_playlists: int,
    after_modes: int,
    after_maps: int,
    playlists_list: list[str],
    modes_list: list[str],
    maps_list: list[str],
) -> None:
    """Affiche le diagnostic lorsque le DataFrame filtré est vide."""
    with st.expander("⚠️ Diagnostic : aucune donnée après filtrage", expanded=True):
        st.write(f"**Filter mode** : `{filter_state.filter_mode}`")
        st.write(f"**Matchs initiaux** : {initial_len}")
        st.write(f"**Après filtre sessions** : {after_session}")
        st.write(f"**Après filtre expérience** : {after_experience}")
        st.write(f"**Après filtre playlists** : {after_playlists}")
        st.write(f"**Après filtre modes** : {after_modes}")
        st.write(f"**Après filtre cartes** : {after_maps}")
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
            f"**Playlists sélectionnées** ({type(filter_state.playlists_selected).__name__}) : {playlists_list[:5]}"
        )
        st.write(
            f"**Modes sélectionnés** ({type(filter_state.modes_selected).__name__}) : {modes_list[:5]}"
        )
        st.write(
            f"**Cartes sélectionnées** ({type(filter_state.maps_selected).__name__}) : {maps_list[:5]}"
        )
        st.write(f"**Experience types** : {filter_state.experience_types_selected}")
