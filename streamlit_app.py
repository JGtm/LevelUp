"""LevelUp - Dashboard Streamlit.

Application de visualisation des statistiques Halo Infinite
depuis la base de données SPNKr.
"""

import contextlib
import logging
import os
import sys
import threading
import urllib.parse
from collections.abc import Callable
from typing import Any, NamedTuple

import streamlit as st

# Vérification : s'assurer que le script est exécuté via `streamlit run`
if not hasattr(st, "runtime") or not st.runtime.exists():
    print("\n[ERREUR] Ce script doit etre execute via Streamlit\n")
    print("Usage correct:")
    print("  streamlit run streamlit_app.py\n")
    print("ou via le launcher:")
    print("  python launcher.py\n")
    sys.exit(1)

# Suppression des warnings connus et non bloquants
logging.getLogger("streamlit.runtime.caching.cache_data_api").setLevel(logging.ERROR)

from src.app.data_loader import (
    ensure_h5g_commendations_repo,
    init_source_state,
)
from src.app.filters import (
    build_friends_opts_map,
)
from src.app.filters_render import (
    apply_filters,
    render_filters_sidebar,
)

# Phase 1 refactoring: Import des nouveaux modules app
# Phase 2 refactoring: Helpers et fonctions extraites
from src.app.helpers import (
    assign_player_colors,
    clean_asset_label,
    date_range,
    normalize_map_label,
    normalize_mode_label,
)

# Phase 5 refactoring: KPIs et Filtres
from src.app.kpis_render import (
    render_kpis_section,
    render_performance_info,
)
from src.app.main_helpers import (
    apply_settings_path_overrides as apply_settings_overrides_main,
)

# Phase 4 refactoring: Main helpers
from src.app.main_helpers import (
    load_match_dataframe,
    load_profile_api,
    propagate_identity_to_env,
    render_profile_hero,
    resolve_xuid_from_input,
    validate_and_fix_db_path,
)
from src.app.media_background import background_media_indexing

# Phase 4 refactoring: Page router
from src.app.page_router import (
    PAGE_KEYS,
    build_match_view_params,
    build_navigation,
    consume_pending_match_id,
    consume_pending_page,
    dispatch_page,
    get_page_label,
    render_page_selector,
    render_page_selector_nav,
)

# Imports depuis la nouvelle architecture
from src.config import (
    get_default_db_path,
)
from src.ui import (
    AppSettings,
    display_name_from_xuid,
    load_css,
    load_settings,
    save_settings,
)
from src.ui.cache import (
    cached_compute_sessions_db,
    cached_list_local_dbs,
    cached_load_highlight_events_for_match,
    cached_load_match_medals_for_player,
    cached_load_match_player_gamertags,
    cached_load_match_rosters,
    cached_load_player_match_result,
    clear_app_caches,
    db_cache_key,
    top_medals_smart,
)
from src.ui.filter_state import (
    _get_player_key,
    apply_filter_preferences,
    get_all_filter_keys_to_clear,
    save_filter_preferences,
)
from src.ui.formatting import (
    PARIS_TZ,
    format_datetime_fr_hm,
    format_score_label,
    score_css_color,
)
from src.ui.i18n import get_lang, set_lang, t
from src.ui.multiplayer import (
    get_gamertag_from_duckdb_v4_path,
    render_player_selector_unified,
)

# render_match_view est toujours nécessaire pour build_match_view_params
from src.ui.pages import render_match_view
from src.ui.streamlit_modern import HAS_NAVIGATION

# Sprint 8ter.5 : Imports page de rendu lazy (chargés à la demande via st.navigation)
# Les fonctions sont importées au moment où la page est visitée, pas au démarrage.
if not HAS_NAVIGATION:
    # Fallback : imports eager pour les versions < 1.36
    from src.ui.pages import (
        render_career_page,
        render_citations_page,
        render_last_match_page,
        render_match_history_page,
        render_match_search_page,
        render_media_tab,
        render_session_comparison_page,
        render_settings_page,
        render_teammates_page,
        render_timeseries_page,
        render_win_loss_page,
    )
from src.ui.perf import perf_reset_run, perf_section
from src.ui.sync import (
    cleanup_orphan_tmp_dbs,
    is_spnkr_db_path,
    render_sync_indicator,
    sync_all_players,
)
from src.visualization import (
    plot_multi_metric_bars_by_match,
)


def _qp_first(value: Any) -> str | None:
    """Extrait la première valeur d'un query parameter."""
    if value is None:
        return None
    if isinstance(value, list | tuple):
        return str(value[0]) if value else None
    s = str(value)
    return s if s.strip() else None


def _set_query_params(**kwargs: str) -> None:
    """Setter query params (compat multi-version Streamlit)."""
    clean: dict[str, str] = {
        k: str(v) for k, v in kwargs.items() if v is not None and str(v).strip()
    }
    try:
        st.query_params.clear()
        for k, v in clean.items():
            st.query_params[k] = v
    except Exception:
        # Fallback API legacy (compat)
        with contextlib.suppress(Exception):
            st.experimental_set_query_params(**clean)


def _app_url(page: str, **params: str) -> str:
    """Génère une URL avec query params pour navigation interne."""
    qp: dict[str, str] = {"page": page}
    for k, v in params.items():
        if v is None:
            continue
        s = str(v).strip()
        if s:
            qp[k] = s
    return "?" + urllib.parse.urlencode(qp)


def _clear_min_matches_maps_auto() -> None:
    st.session_state["_min_matches_maps_auto"] = False


def _clear_min_matches_maps_friends_auto() -> None:
    st.session_state["_min_matches_maps_friends_auto"] = False


# Alias pour les fonctions déplacées vers cache.py
_db_cache_key = db_cache_key
_top_medals = top_medals_smart
_clear_app_caches = clear_app_caches


def _aliases_cache_key() -> int | None:
    """Retourne toujours None depuis v5.2 (plus de fichier xuid_aliases.json)."""
    return None


# =============================================================================
# Contexte partagé entre les étapes de main()
# =============================================================================


class PageContext(NamedTuple):
    """Données partagées entre le chargement et le dispatch des pages."""

    dff: Any  # pl.DataFrame — matchs filtrés
    df: Any  # pl.DataFrame — tous les matchs
    base: Any  # pl.DataFrame — clone avant filtres
    db_path: str
    xuid: str
    db_key: Any
    aliases_key: int | None
    settings: AppSettings
    waypoint_player: str
    me_name: str
    gap_minutes: int
    picked_session_labels: Any
    match_view_params: dict[str, Any]


# =============================================================================
# Fonctions extraites de main()
# =============================================================================


def _tailscale_worker() -> None:
    """Démarre le funnel Tailscale + notification Discord."""
    try:
        from src.utils.discord_notifier import notify_app_started
        from src.utils.tailscale import start_funnel

        url = start_funnel()
        if url:
            notify_app_started(url)
    except Exception as _e:
        print(f"[Tailscale] worker erreur inattendue : {_e}", flush=True)


def _initialize_app() -> tuple[AppSettings, str, list[str], list[str]]:
    """Configure la page Streamlit, charge les settings et valide la config.

    Returns:
        Tuple (settings, DEFAULT_DB, cfg_warnings, cfg_errors).
    """
    st.set_page_config(page_title="LevelUp", page_icon="🎯", layout="wide")
    perf_reset_run()

    # Nettoyage des fichiers temporaires orphelins (une fois par session)
    cleanup_orphan_tmp_dbs()

    with perf_section("css"):
        st.markdown(load_css(), unsafe_allow_html=True)

    # Référentiel Citations (opt-in via env)
    if str(os.environ.get("OPENSPARTAN_CITATIONS_AUTOGEN") or "").strip() in {
        "1",
        "true",
        "True",
    }:
        ensure_h5g_commendations_repo()

    # Paramètres (persistés)
    settings: AppSettings = load_settings()
    st.session_state["app_settings"] = settings

    # Chargement des secrets (Doppler ou .env.local) — une seule fois par session
    if not st.session_state.get("_secrets_loaded"):
        st.session_state["_secrets_loaded"] = True
        try:
            from src.utils.secrets import load_doppler_secrets_to_env

            if bool(getattr(settings, "doppler_enabled", False)):
                load_doppler_secrets_to_env(
                    project=str(getattr(settings, "doppler_project", "") or ""),
                    config=str(getattr(settings, "doppler_config", "") or ""),
                )
        except Exception:
            pass

    # Langue UI (persistée) : session_state prime, sinon app_settings.json.
    if "lang" not in st.session_state:
        st.session_state["lang"] = getattr(settings, "lang", "fr") or "fr"

    # Propage les defaults depuis secrets vers l'env et applique les overrides de chemins
    propagate_identity_to_env()
    apply_settings_overrides_main(settings)

    # Validation de la configuration (calculée une fois par session)
    if "_startup_cfg_warnings" not in st.session_state:
        try:
            from src.utils.startup_check import check_app_settings

            _w, _e = check_app_settings(settings)
            st.session_state["_startup_cfg_warnings"] = _w
            st.session_state["_startup_cfg_errors"] = _e
        except Exception:
            st.session_state["_startup_cfg_warnings"] = []
            st.session_state["_startup_cfg_errors"] = []
    cfg_warnings: list[str] = st.session_state.get("_startup_cfg_warnings", [])
    cfg_errors: list[str] = st.session_state.get("_startup_cfg_errors", [])

    # Source (persistée via session_state)
    DEFAULT_DB = get_default_db_path()
    init_source_state(DEFAULT_DB, settings)

    return settings, DEFAULT_DB, cfg_warnings, cfg_errors


def _start_background_services(settings: AppSettings, DEFAULT_DB: str) -> None:
    """Lance les services d'arrière-plan (media indexing, Tailscale)."""
    background_media_indexing(settings, DEFAULT_DB)

    if not st.session_state.get("_tailscale_started") and bool(
        getattr(settings, "tailscale_funnel_enabled", False)
    ):
        st.session_state["_tailscale_started"] = True
        threading.Thread(target=_tailscale_worker, daemon=True, name="tailscale-funnel").start()


def _parse_query_params() -> None:
    """Consomme les query params (?page=...&match_id=...) pour navigation interne."""
    try:
        qp = dict(st.query_params)
        qp_page = _qp_first(qp.get("page"))
        qp_mid = _qp_first(qp.get("match_id"))
    except Exception:
        qp_page = None
        qp_mid = None
    qp_params = (str(qp_page or "").strip(), str(qp_mid or "").strip())
    if any(qp_params) and st.session_state.get("_consumed_query_params") != qp_params:
        st.session_state["_consumed_query_params"] = qp_params
        if qp_params[0]:
            st.session_state["_pending_page"] = qp_params[0]
        if qp_params[1]:
            st.session_state["_pending_match_id"] = qp_params[1]
        # Nettoie l'URL après consommation
        try:
            st.query_params.clear()
        except Exception:
            with contextlib.suppress(Exception):
                st.experimental_set_query_params()


def _render_main_sidebar(db_path: str, xuid: str, settings: AppSettings) -> tuple[str, str, str]:
    """Rendu de la sidebar principale (langue, logo, joueur, sync).

    Peut appeler ``st.rerun()`` lors d'un changement de joueur ou de langue.

    Returns:
        Tuple (db_path, xuid, waypoint_player) potentiellement mis à jour.
    """
    waypoint_player = str(st.session_state.get("waypoint_player", "") or "").strip()

    with st.sidebar:
        # Sélecteur de langue (discret) — au-dessus du logo
        _LANG_OPTIONS = {"fr": "FR", "en": "EN"}
        current_lang = str(get_lang() or "fr").strip().lower()
        if current_lang not in _LANG_OPTIONS:
            current_lang = "fr"

        lang_keys = list(_LANG_OPTIONS.keys())
        lang_idx = lang_keys.index(current_lang)
        picked = st.radio(
            "Langue",
            options=list(_LANG_OPTIONS.values()),
            index=lang_idx,
            key="_sidebar_lang_selector",
            horizontal=True,
            label_visibility="collapsed",
        )

        selected_lang = next(k for k, v in _LANG_OPTIONS.items() if v == picked)
        if selected_lang != current_lang:
            set_lang(selected_lang)
            settings.lang = selected_lang
            save_settings(settings)
            st.rerun()

        # Logo en haut de la sidebar
        logo_path = os.path.join(os.path.dirname(__file__), "static", "logo.png")
        if os.path.exists(logo_path):
            st.image(logo_path, width="stretch")

        st.markdown("<div class='os-sidebar-divider'></div>", unsafe_allow_html=True)

        # Indicateur de dernière synchronisation
        if db_path and os.path.exists(db_path):
            render_sync_indicator(db_path)

        # Sélecteur multi-joueurs
        if db_path and os.path.exists(db_path):
            new_db_path, new_xuid = render_player_selector_unified(
                db_path, xuid, key="sidebar_player_selector"
            )
            if new_db_path or new_xuid:
                # Changement de joueur : sauvegarder les filtres de l'ancien joueur
                old_xuid = xuid
                old_db_path = db_path
                with contextlib.suppress(Exception):
                    save_filter_preferences(old_xuid, old_db_path)

                # Nettoyer exhaustivement les filtres
                for key in get_all_filter_keys_to_clear(st.session_state):
                    del st.session_state[key]

                # Réinitialiser les flags de chargement et sauvegarde
                old_player_key = _get_player_key(old_xuid, old_db_path)
                old_filters_loaded_key = f"_filters_loaded_{old_player_key}"
                old_last_saved_key = f"_last_saved_player_{old_player_key}"
                if old_filters_loaded_key in st.session_state:
                    del st.session_state[old_filters_loaded_key]
                if old_last_saved_key in st.session_state:
                    del st.session_state[old_last_saved_key]

                # Mettre à jour db_path et xuid pour le nouveau joueur
                if new_db_path:
                    st.session_state["db_path"] = new_db_path
                    db_path = new_db_path
                    gamertag = get_gamertag_from_duckdb_v4_path(new_db_path)
                    if gamertag:
                        st.session_state["xuid_input"] = gamertag
                        st.session_state["waypoint_player"] = gamertag
                        xuid = gamertag
                if new_xuid:
                    st.session_state["xuid_input"] = new_xuid
                    xuid = new_xuid

                apply_filter_preferences(xuid, db_path)
                st.rerun()

        # Bouton Sync
        if db_path and is_spnkr_db_path(db_path) and os.path.exists(db_path):  # noqa: SIM102
            if st.button(
                t("sidebar_sync_btn"),
                key="sidebar_sync_button",
                help=t("sidebar_sync_help"),
                width="stretch",
            ):
                with st.spinner(t("sidebar_sync_spinner")):
                    ok, msg = sync_all_players(
                        db_path=db_path,
                        match_type=str(
                            getattr(settings, "spnkr_refresh_match_type", "matchmaking")
                            or "matchmaking"
                        ),
                        max_matches=int(getattr(settings, "spnkr_refresh_max_matches", 200) or 200),
                        rps=int(getattr(settings, "spnkr_refresh_rps", 5) or 5),
                        with_highlight_events=True,
                        with_aliases=True,
                        delta=True,
                        timeout_seconds=180,
                    )
                if ok:
                    st.success(msg)
                    _clear_app_caches()
                    st.session_state["_cache_buster"] = st.session_state.get("_cache_buster", 0) + 1
                    st.rerun()
                else:
                    st.error(msg)

    return db_path, xuid, waypoint_player


def _load_and_prepare_data(
    db_path: str,
    xuid: str,
    DEFAULT_DB: str,
    settings: AppSettings,
    waypoint_player: str,
    cfg_warnings: list[str],
    cfg_errors: list[str],
) -> PageContext | None:
    """Charge les données, applique les filtres et rend les KPIs.

    Returns:
        ``PageContext`` si des données existent, sinon ``None`` (page settings affichée).
    """
    # Validation du chemin DB
    db_path = validate_and_fix_db_path(db_path, DEFAULT_DB)

    # Résolution du XUID
    xuid = resolve_xuid_from_input(xuid, db_path)

    me_name = (
        display_name_from_xuid(xuid.strip(), db_path=db_path)
        if str(xuid or "").strip()
        else "(joueur)"
    )
    aliases_key = _aliases_cache_key()

    # Auto-profil (SPNKr) et rendu du hero
    api_app, _api_err = load_profile_api(xuid, settings, db_path=db_path)
    render_profile_hero(xuid, settings, api_app, db_path=db_path)

    # Chargement des données
    cache_buster = st.session_state.get("_cache_buster", 0)
    df, db_key = load_match_dataframe(db_path, xuid, cache_buster=cache_buster)

    # Debug conditionnel
    show_debug = st.session_state.get("_show_debug_info", False)
    if show_debug and len(df) > 0:
        st.info("🔍 **Mode Debug activé** - Informations sur les données chargées")
        with st.expander("🔍 Debug - DataFrame complet (avant filtres)", expanded=True):
            st.write(f"**Nombre total de matchs dans df** : {len(df)}")
            if "start_time" in df.columns:
                st.write(f"**Date min dans df** : {df['start_time'].min()}")
                st.write(f"**Date max dans df** : {df['start_time'].max()}")
                null_count = df["start_time"].is_null().sum()
                st.write(f"**Nombre de start_time NULL** : {null_count}")
                if null_count > 0:
                    st.warning("⚠️ Il y a des valeurs NULL dans start_time !")
                last_5_df = df.sort("start_time", descending=True).head(5)
                st.write("**5 derniers matchs dans df (par date) :**")
                for row in last_5_df.iter_rows(named=True):
                    st.write(
                        f"- {row.get('start_time')} | Match ID: {row.get('match_id')}"
                        f" | Map: {row.get('map_name')}"
                    )

    # Early return si aucun match
    if len(df) == 0:
        st.radio(
            t("sidebar_navigation"),
            options=[t("page_settings")],
            horizontal=True,
            key="page",
            label_visibility="collapsed",
        )
        from src.ui.pages import render_settings_page as _render_settings_empty

        _render_settings_empty(
            settings,
            get_local_dbs_fn=cached_list_local_dbs,
            on_clear_caches_fn=_clear_app_caches,
        )
        return None

    # Sidebar - Filtres
    with st.sidebar:
        for _err_msg in cfg_errors:
            st.error(_err_msg)
        for _warn_msg in cfg_warnings:
            st.warning(_warn_msg)

        filter_state = render_filters_sidebar(
            df=df,
            db_path=db_path,
            xuid=xuid,
            db_key=db_key,
            aliases_key=aliases_key,
            date_range_fn=date_range,
            clean_asset_label_fn=clean_asset_label,
            normalize_mode_label_fn=normalize_mode_label,
            normalize_map_label_fn=normalize_map_label,
            build_friends_opts_map_fn=build_friends_opts_map,
        )

    # Base "globale" : toutes les parties (après inclusion/exclusion Firefight)
    base = df.clone()

    # Application des filtres
    dff = apply_filters(
        dff=df,
        filter_state=filter_state,
        db_path=db_path,
        xuid=xuid,
        db_key=db_key,
        clean_asset_label_fn=clean_asset_label,
        normalize_mode_label_fn=normalize_mode_label,
        normalize_map_label_fn=normalize_map_label,
    )

    gap_minutes = filter_state.gap_minutes
    picked_session_labels = filter_state.picked_session_labels

    # KPIs
    render_kpis_section(dff)
    render_performance_info()

    # Paramètres communs pour les pages de match
    _match_view_params = build_match_view_params(
        db_path=db_path,
        xuid=xuid,
        waypoint_player=waypoint_player,
        db_key=db_key,
        settings=settings,
        df_full=df,
        render_match_view_fn=render_match_view,
        normalize_mode_label_fn=normalize_mode_label,
        format_score_label_fn=format_score_label,
        score_css_color_fn=score_css_color,
        format_datetime_fn=format_datetime_fr_hm,
        load_player_match_result_fn=cached_load_player_match_result,
        load_match_medals_fn=cached_load_match_medals_for_player,
        load_highlight_events_fn=cached_load_highlight_events_for_match,
        load_match_gamertags_fn=cached_load_match_player_gamertags,
        load_match_rosters_fn=cached_load_match_rosters,
        paris_tz=PARIS_TZ,
    )

    return PageContext(
        dff=dff,
        df=df,
        base=base,
        db_path=db_path,
        xuid=xuid,
        db_key=db_key,
        aliases_key=aliases_key,
        settings=settings,
        waypoint_player=waypoint_player,
        me_name=me_name,
        gap_minutes=gap_minutes,
        picked_session_labels=picked_session_labels,
        match_view_params=_match_view_params,
    )


def _dispatch_pages(ctx: PageContext) -> None:
    """Dispatch vers la page active (st.navigation ou fallback legacy)."""
    consume_pending_match_id()

    if HAS_NAVIGATION:
        _dispatch_navigation(ctx)
    else:
        _dispatch_legacy(ctx)


def _dispatch_navigation(ctx: PageContext) -> None:
    """Dispatch via st.navigation (Streamlit ≥ 1.36) avec lazy-loading."""

    def _page_timeseries() -> None:
        from src.ui.pages import render_timeseries_page

        render_timeseries_page(ctx.dff, df_full=ctx.df, db_path=ctx.db_path, xuid=ctx.xuid)

    def _page_session_compare() -> None:
        from src.app.filters import get_friends_xuids_for_sessions
        from src.app.page_router import _to_polars
        from src.ui.pages import render_session_comparison_page

        friends_tuple = get_friends_xuids_for_sessions(
            ctx.db_path,
            ctx.xuid.strip(),
            ctx.db_key,
            ctx.aliases_key,
        )
        all_sessions_df = cached_compute_sessions_db(
            ctx.db_path,
            ctx.xuid.strip(),
            ctx.db_key,
            True,
            ctx.gap_minutes,
            friends_xuids=friends_tuple,
        )
        all_sessions_pl = _to_polars(all_sessions_df)
        df_pl = _to_polars(ctx.df)
        if (
            not all_sessions_pl.is_empty()
            and "match_id" in df_pl.columns
            and "match_id" in all_sessions_pl.columns
        ):
            sess_cols = ["match_id", "session_id", "session_label"]
            drop_cols = [c for c in ("session_id", "session_label") if c in df_pl.columns]
            df_for_merge = df_pl.drop(drop_cols) if drop_cols else df_pl
            sessions_for_compare = df_for_merge.join(
                all_sessions_pl.select(sess_cols),
                on="match_id",
                how="inner",
            )
        else:
            sessions_for_compare = all_sessions_pl
        render_session_comparison_page(sessions_for_compare, df_full=ctx.df)

    def _page_last_match() -> None:
        from src.ui.pages import render_last_match_page

        render_last_match_page(dff=ctx.dff, **ctx.match_view_params)

    def _page_match_search() -> None:
        from src.ui.pages import render_match_search_page

        render_match_search_page(df=ctx.df, dff=ctx.dff, **ctx.match_view_params)

    def _page_media() -> None:
        from src.ui.pages import render_media_tab

        render_media_tab(df_full=ctx.df, settings=ctx.settings)

    def _page_citations() -> None:
        from src.ui.pages import render_citations_page

        render_citations_page(
            dff=ctx.dff,
            df_full=ctx.df,
            xuid=ctx.xuid,
            db_path=ctx.db_path,
            db_key=ctx.db_key,
            top_medals_fn=_top_medals,
        )

    def _page_win_loss() -> None:
        from src.ui.pages import render_win_loss_page

        render_win_loss_page(
            dff=ctx.dff,
            base=ctx.base,
            picked_session_labels=ctx.picked_session_labels,
            db_path=ctx.db_path,
            xuid=ctx.xuid,
            db_key=ctx.db_key,
        )

    def _page_teammates() -> None:
        from src.ui.pages import render_teammates_page

        render_teammates_page(
            df=ctx.df,
            dff=ctx.dff,
            base=ctx.base,
            me_name=ctx.me_name,
            xuid=ctx.xuid,
            db_path=ctx.db_path,
            db_key=ctx.db_key,
            aliases_key=ctx.aliases_key,
            settings=ctx.settings,
            picked_session_labels=ctx.picked_session_labels,
            include_firefight=True,
            waypoint_player=ctx.waypoint_player,
            build_friends_opts_map_fn=build_friends_opts_map,
            assign_player_colors_fn=assign_player_colors,
            plot_multi_metric_bars_fn=plot_multi_metric_bars_by_match,
            top_medals_fn=_top_medals,
        )

    def _page_match_history() -> None:
        from src.ui.pages import render_match_history_page

        render_match_history_page(
            dff=ctx.dff,
            waypoint_player=ctx.waypoint_player,
            db_path=ctx.db_path,
            xuid=ctx.xuid,
            db_key=ctx.db_key,
            df_full=ctx.df,
        )

    def _page_career() -> None:
        from src.ui.pages import render_career_page

        render_career_page(db_path=ctx.db_path, xuid=ctx.xuid, db_key=ctx.db_key)

    def _page_settings() -> None:
        from src.ui.pages import render_settings_page

        render_settings_page(
            ctx.settings,
            get_local_dbs_fn=cached_list_local_dbs,
            on_clear_caches_fn=_clear_app_caches,
        )

    page_callables: dict[str, Callable[[], None]] = {
        "timeseries": _page_timeseries,
        "session_compare": _page_session_compare,
        "last_match": _page_last_match,
        "match": _page_match_search,
        "media": _page_media,
        "citations": _page_citations,
        "win_loss": _page_win_loss,
        "teammates": _page_teammates,
        "match_history": _page_match_history,
        "career": _page_career,
        "settings": _page_settings,
    }

    pg, pages = build_navigation(page_callables)

    # Gérer les redirections en attente (liens depuis une autre page)
    pending_page = st.session_state.pop("_pending_page", None)
    if isinstance(pending_page, str):
        from src.app.page_router import _LEGACY_NAME_TO_SLUG

        slug = _LEGACY_NAME_TO_SLUG.get(pending_page, pending_page)
        label = get_page_label(slug) if slug in PAGE_KEYS else pending_page
        target = next((p for p in pages if p.title == label), None)
        if target is not None and target != pg:
            st.switch_page(target)

    render_page_selector_nav(pages, pg)
    pg.run()


def _dispatch_legacy(ctx: PageContext) -> None:
    """Dispatch legacy via page selector (Streamlit < 1.36)."""
    consume_pending_page()
    page = render_page_selector()

    dispatch_page(
        page=page,
        dff=ctx.dff,
        df=ctx.df,
        base=ctx.base,
        me_name=ctx.me_name,
        xuid=ctx.xuid,
        db_path=ctx.db_path,
        db_key=ctx.db_key,
        aliases_key=ctx.aliases_key,
        settings=ctx.settings,
        picked_session_labels=ctx.picked_session_labels,
        waypoint_player=ctx.waypoint_player,
        gap_minutes=ctx.gap_minutes,
        match_view_params=ctx.match_view_params,
        render_last_match_page_fn=render_last_match_page,
        render_match_search_page_fn=render_match_search_page,
        render_citations_page_fn=render_citations_page,
        render_session_comparison_page_fn=render_session_comparison_page,
        render_timeseries_page_fn=render_timeseries_page,
        render_win_loss_page_fn=render_win_loss_page,
        render_teammates_page_fn=render_teammates_page,
        render_match_history_page_fn=render_match_history_page,
        render_media_tab_fn=render_media_tab,
        render_career_page_fn=render_career_page,
        render_settings_page_fn=render_settings_page,
        cached_compute_sessions_db_fn=cached_compute_sessions_db,
        top_medals_fn=_top_medals,
        build_friends_opts_map_fn=build_friends_opts_map,
        assign_player_colors_fn=assign_player_colors,
        plot_multi_metric_bars_fn=plot_multi_metric_bars_by_match,
        get_local_dbs_fn=cached_list_local_dbs,
        clear_caches_fn=_clear_app_caches,
    )


# =============================================================================
# Application principale
# =============================================================================


def main() -> None:
    """Point d'entrée principal de l'application Streamlit."""
    # 1. Initialisation (config, CSS, settings, validation)
    settings, DEFAULT_DB, cfg_warnings, cfg_errors = _initialize_app()

    # 2. Services d'arrière-plan (media indexing, Tailscale)
    _start_background_services(settings, DEFAULT_DB)

    # 3. Query params (deep links)
    _parse_query_params()

    # 4. Sidebar principale (langue, joueur, sync)
    db_path = str(st.session_state.get("db_path", "") or "").strip()
    xuid = str(st.session_state.get("xuid_input", "") or "").strip()
    db_path, xuid, waypoint_player = _render_main_sidebar(db_path, xuid, settings)

    # 5. Chargement des données, filtres et KPIs
    ctx = _load_and_prepare_data(
        db_path,
        xuid,
        DEFAULT_DB,
        settings,
        waypoint_player,
        cfg_warnings,
        cfg_errors,
    )
    if ctx is None:
        return  # Aucun match — page settings déjà affichée

    # 6. Dispatch vers la page active
    _dispatch_pages(ctx)


if __name__ == "__main__":
    main()
