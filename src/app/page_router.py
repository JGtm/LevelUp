"""Routage des pages avec st.navigation (8ter.5).

Ce module centralise :
- La liste des pages disponibles
- La construction des paramètres pour les pages de match
- Le routage via st.navigation (lazy loading)
"""

from __future__ import annotations

import logging
from collections.abc import Callable

import polars as pl
import streamlit as st

from src.app._page_context import MatchViewParams
from src.ui.i18n import t
from src.ui.settings import AppSettings
from src.utils.polars_compat import ensure_polars as _to_polars

logger = logging.getLogger(__name__)

# Clés internes stables (slugs) — NE PAS TRADUIRE
PAGE_KEYS: list[str] = [
    "timeseries",
    "session_compare",
    "last_match",
    "explorer",
    "media",
    "citations",
    "win_loss",
    "teammates",
    "match_history",
    "career",
    "settings",
]

# Mapping slug → clé i18n pour le label traduit
_PAGE_I18N_KEYS: dict[str, str] = {
    "timeseries": "page_timeseries",
    "session_compare": "page_session_compare",
    "last_match": "page_last_match",
    "explorer": "page_explorer",
    "media": "page_media",
    "citations": "page_citations",
    "win_loss": "page_win_loss",
    "teammates": "page_teammates",
    "match_history": "page_match_history",
    "career": "page_career",
    "settings": "page_settings",
}

# Legacy: ancien nom FR → slug (pour migration session_state)
_LEGACY_NAME_TO_SLUG: dict[str, str] = {
    "Séries temporelles": "timeseries",
    "Comparaison de sessions": "session_compare",
    "Dernier match": "last_match",
    "Match": "explorer",
    "Explorer": "explorer",
    "Médias": "media",
    "Citations": "citations",
    "Victoires/Défaites": "win_loss",
    "Mes coéquipiers": "teammates",
    "Historique des parties": "match_history",
    "Carrière": "career",
    "Paramètres": "settings",
}


def get_page_label(slug: str) -> str:
    """Retourne le label traduit pour un slug de page."""
    i18n_key = _PAGE_I18N_KEYS.get(slug, "")
    return t(i18n_key) if i18n_key else slug


def get_page_labels() -> list[str]:
    """Retourne la liste des labels traduits dans l'ordre de PAGE_KEYS."""
    return [get_page_label(k) for k in PAGE_KEYS]


# Compatibilité descendante : PAGES est une propriété dynamique
# qui retourne les labels traduits (pour le code qui itère encore dessus).
@property  # type: ignore[misc]
def _pages_compat() -> list[str]:
    return get_page_labels()


# Garde l'ancienne variable pour éviter les imports cassés,
# mais elle retourne désormais les labels traduits.
PAGES = PAGE_KEYS  # Les consommateurs doivent utiliser get_page_label(slug)


def build_match_view_params(  # noqa: PLR0913
    db_path: str,
    xuid: str,
    waypoint_player: str,
    db_key: str | None,
    settings: AppSettings,
    df_full: pl.DataFrame,
    render_match_view_fn: Callable,
    normalize_mode_label_fn: Callable,
    format_score_label_fn: Callable,
    score_css_color_fn: Callable,
    format_datetime_fn: Callable,
    load_player_match_result_fn: Callable,
    load_match_medals_fn: Callable,
    load_highlight_events_fn: Callable,
    load_match_gamertags_fn: Callable,
    load_match_rosters_fn: Callable,
    paris_tz,
) -> MatchViewParams:
    """Construit les paramètres communs pour les pages de match."""
    return {
        "db_path": db_path,
        "xuid": xuid,
        "waypoint_player": waypoint_player,
        "db_key": db_key,
        "settings": settings,
        "df_full": df_full,
        "render_match_view_fn": render_match_view_fn,
        "normalize_mode_label_fn": normalize_mode_label_fn,
        "format_score_label_fn": format_score_label_fn,
        "score_css_color_fn": score_css_color_fn,
        "format_datetime_fn": format_datetime_fn,
        "load_player_match_result_fn": load_player_match_result_fn,
        "load_match_medals_fn": load_match_medals_fn,
        "load_highlight_events_fn": load_highlight_events_fn,
        "load_match_gamertags_fn": load_match_gamertags_fn,
        "load_match_rosters_fn": load_match_rosters_fn,
        "paris_tz": paris_tz,
    }


def consume_pending_page() -> None:
    """Consomme la page en attente si définie."""
    pending_page = st.session_state.pop("_pending_page", None)
    if isinstance(pending_page, str):
        # Accepter un slug OU un ancien nom FR
        slug = _LEGACY_NAME_TO_SLUG.get(pending_page, pending_page)
        if slug in PAGE_KEYS:
            st.session_state["page"] = slug
    if "page" not in st.session_state:
        st.session_state["page"] = PAGE_KEYS[0]  # timeseries


def consume_pending_match_id() -> None:
    """Consomme le match_id en attente si défini."""
    pending_mid = st.session_state.pop("_pending_match_id", None)
    if isinstance(pending_mid, str) and pending_mid.strip():
        logger.info("Ouverture match depuis pending: %s", pending_mid.strip())
        st.session_state["match_id_input"] = pending_mid.strip()


def render_page_selector() -> str:
    """Rend le sélecteur de page et retourne le slug de la page choisie.

    Note: Conservé pour compatibilité. Avec st.navigation, utiliser
    ``build_navigation`` + ``run_selected_page`` à la place.
    """
    labels = get_page_labels()
    selected_label = st.segmented_control(
        "Onglets",
        options=labels,
        key="page_label_selector",
        label_visibility="collapsed",
    )
    if selected_label:
        # Trouver le slug correspondant au label sélectionné
        idx = labels.index(selected_label) if selected_label in labels else 0
        slug = PAGE_KEYS[idx]
        st.session_state["page"] = slug
        return slug
    return st.session_state.get("page", PAGE_KEYS[0])


# ---------------------------------------------------------------------------
# st.navigation — routing moderne (8ter.5)
# ---------------------------------------------------------------------------

# Mapping slug → url_path (slugs URL-friendly)
_PAGE_URL_PATHS: dict[str, str] = {
    "timeseries": "timeseries",
    "session_compare": "session-compare",
    "last_match": "last-match",
    "explorer": "explorer",
    "media": "medias",
    "citations": "citations",
    "win_loss": "win-loss",
    "teammates": "teammates",
    "match_history": "history",
    "career": "career",
    "settings": "settings",
}

_PAGE_ICONS: dict[str, str] = {
    "timeseries": "📈",
    "session_compare": "🔄",
    "last_match": "🎯",
    "explorer": "🔍",
    "media": "🎬",
    "citations": "🏅",
    "win_loss": "📊",
    "teammates": "👥",
    "match_history": "📋",
    "career": "⭐",
    "settings": "⚙️",
}


def build_navigation(
    page_callables: dict[str, Callable[[], None]],
) -> tuple[st.Page, list[st.Page]]:
    """Construit les pages st.navigation et retourne (page sélectionnée, toutes les pages).

    Args:
        page_callables: Mapping slug → callback sans argument.

    Returns:
        Tuple (page_courante, liste_pages) prêt pour ``pg.run()``.
    """
    pages: list[st.Page] = []
    for slug in PAGE_KEYS:
        cb = page_callables.get(slug)
        if cb is None:
            continue
        url_path = _PAGE_URL_PATHS.get(slug, slug)
        icon = _PAGE_ICONS.get(slug)
        label = get_page_label(slug)
        pages.append(
            st.Page(cb, title=label, icon=icon, url_path=url_path),
        )

    pg = st.navigation(pages, position="hidden")
    return pg, pages


def render_page_selector_nav(
    pages: list[st.Page],
    current_page: st.Page,
) -> None:
    """Sélecteur de page visuel compatible st.navigation.

    Affiche un ``st.segmented_control`` et appelle ``st.switch_page``
    si l'utilisateur clique sur un onglet différent.
    """
    titles = [p.title for p in pages]
    current_title = current_page.title if current_page else titles[0]

    selected = st.segmented_control(
        "Onglets",
        options=titles,
        default=current_title,
        label_visibility="collapsed",
    )

    if selected and selected != current_title:
        logger.info("Navigation: %s → %s", current_title, selected)
        target = next((p for p in pages if p.title == selected), None)
        if target is not None:
            st.switch_page(target)


def dispatch_page(  # noqa: C901, PLR0912, PLR0913
    page: str,
    dff: pl.DataFrame,
    df: pl.DataFrame,
    base: pl.DataFrame,
    me_name: str,
    xuid: str,
    db_path: str,
    db_key: str | None,
    aliases_key: str | None,
    settings: AppSettings,
    picked_session_labels: list[str] | None,
    waypoint_player: str,
    gap_minutes: int,
    match_view_params: dict,
    # Fonctions de rendu
    render_last_match_page_fn: Callable,
    render_match_search_page_fn: Callable,
    render_citations_page_fn: Callable,
    render_session_comparison_page_fn: Callable,
    render_timeseries_page_fn: Callable,
    render_win_loss_page_fn: Callable,
    render_teammates_page_fn: Callable,
    render_match_history_page_fn: Callable,
    render_media_tab_fn: Callable,
    render_career_page_fn: Callable,
    render_settings_page_fn: Callable,
    # Fonctions utilitaires
    cached_compute_sessions_db_fn: Callable,
    top_medals_fn: Callable,
    build_friends_opts_map_fn: Callable,
    assign_player_colors_fn: Callable,
    plot_multi_metric_bars_fn: Callable,
    get_local_dbs_fn: Callable,
    clear_caches_fn: Callable,
) -> None:
    """Dispatch vers la page appropriée."""
    # Les fonctions de rendu attendent encore pandas, donc on garde df en l'état
    # La conversion se fera progressivement au niveau de chaque page
    logger.debug("Page exécutée: %s", page)

    if page == "last_match":
        render_last_match_page_fn(dff=dff, **match_view_params)

    elif page == "explorer":
        render_match_search_page_fn(df=df, dff=dff, **match_view_params)

    elif page == "citations":
        render_citations_page_fn(
            dff=dff,
            df_full=df,
            xuid=xuid,
            db_path=db_path,
            db_key=db_key,
            top_medals_fn=top_medals_fn,
        )

    elif page == "session_compare":
        from src.app.filters import get_friends_xuids_for_sessions

        friends_tuple = get_friends_xuids_for_sessions(db_path, xuid.strip(), db_key, aliases_key)
        all_sessions_df = cached_compute_sessions_db_fn(
            db_path, xuid.strip(), db_key, True, gap_minutes, friends_xuids=friends_tuple
        )
        # Normaliser en Polars
        all_sessions_pl = _to_polars(all_sessions_df)
        df_pl = _to_polars(df)

        # La page a besoin d'un DataFrame "sessions" enrichi : session_id/session_label
        # + toutes les stats match (kills, pair_name, etc.). En DuckDB v4, cached_compute_sessions_db
        # ne renvoie que match_id, start_time, session_id, session_label → on fusionne avec df.
        if (
            not all_sessions_pl.is_empty()
            and "match_id" in df_pl.columns
            and "match_id" in all_sessions_pl.columns
        ):
            sess_cols = ["match_id", "session_id", "session_label"]
            # Retirer session_id/session_label de df avant join pour éviter doublons
            drop_cols = [c for c in ("session_id", "session_label") if c in df_pl.columns]
            df_for_merge = df_pl.drop(drop_cols) if drop_cols else df_pl
            sessions_for_compare = df_for_merge.join(
                all_sessions_pl.select(sess_cols), on="match_id", how="inner"
            )
        else:
            sessions_for_compare = all_sessions_pl
        render_session_comparison_page_fn(sessions_for_compare, df_full=df)

    elif page == "timeseries":
        render_timeseries_page_fn(dff, df_full=df, db_path=db_path, xuid=xuid)

    elif page == "win_loss":
        render_win_loss_page_fn(
            dff=dff,
            base=base,
            picked_session_labels=picked_session_labels,
            db_path=db_path,
            xuid=xuid,
            db_key=db_key,
        )

    elif page == "teammates":
        render_teammates_page_fn(
            df=df,
            dff=dff,
            base=base,
            me_name=me_name,
            xuid=xuid,
            db_path=db_path,
            db_key=db_key,
            aliases_key=aliases_key,
            settings=settings,
            picked_session_labels=picked_session_labels,
            include_firefight=True,
            waypoint_player=waypoint_player,
            build_friends_opts_map_fn=build_friends_opts_map_fn,
            assign_player_colors_fn=assign_player_colors_fn,
            plot_multi_metric_bars_fn=plot_multi_metric_bars_fn,
            top_medals_fn=top_medals_fn,
        )

    elif page == "match_history":
        render_match_history_page_fn(
            dff=dff,
            waypoint_player=waypoint_player,
            db_path=db_path,
            xuid=xuid,
            db_key=db_key,
            df_full=df,
        )

    elif page == "media" or page == "Bibliothèque médias":
        render_media_tab_fn(
            df_full=df,
            settings=settings,
        )

    elif page == "career":
        render_career_page_fn(
            db_path=db_path,
            xuid=xuid,
            db_key=db_key,
        )

    elif page == "settings":
        render_settings_page_fn(
            settings,
            get_local_dbs_fn=get_local_dbs_fn,
            on_clear_caches_fn=clear_caches_fn,
        )
