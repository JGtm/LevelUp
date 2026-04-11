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

logger = logging.getLogger(__name__)

# Clés internes stables (slugs) — NE PAS TRADUIRE
PAGE_KEYS: list[str] = [
    "timeseries",
    "session_compare",
    "teammates",
    "last_match",
    "media",
    "citations",
    "explorer",
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
    # ↓ Redirigé vers timeseries depuis la fusion v6.3 (ancienne page Victoires/Défaites)
    "Victoires/Défaites": "timeseries",
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
    return MatchViewParams(
        db_path=db_path,
        xuid=xuid,
        waypoint_player=waypoint_player,
        db_key=db_key,
        settings=settings,
        df_full=df_full,
        render_match_view_fn=render_match_view_fn,
        format_score_label_fn=format_score_label_fn,
        score_css_color_fn=score_css_color_fn,
        format_datetime_fn=format_datetime_fn,
        load_player_match_result_fn=load_player_match_result_fn,
        load_match_medals_fn=load_match_medals_fn,
        load_highlight_events_fn=load_highlight_events_fn,
        load_match_gamertags_fn=load_match_gamertags_fn,
        load_match_rosters_fn=load_match_rosters_fn,
        paris_tz=paris_tz,
    )


def consume_pending_match_id() -> None:
    """Consomme le match_id en attente si défini."""
    pending_mid = st.session_state.pop("_pending_match_id", None)
    if isinstance(pending_mid, str) and pending_mid.strip():
        logger.info("Ouverture match depuis pending: %s", pending_mid.strip())
        st.session_state["match_id_input"] = pending_mid.strip()


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
    "teammates": "👥",
    "match_history": "📋",
    "career": "⭐",
    "settings": "⚙️",
}

V7_SECTION_KEYS: list[str] = [
    "home",
    "stats",
    "squad",
    "explorer",
    "media",
    "profile",
]

_V7_SECTION_I18N_KEYS: dict[str, str] = {
    "home": "v7_nav_home",
    "stats": "v7_nav_stats",
    "squad": "page_teammates",
    "explorer": "page_explorer",
    "media": "page_media",
    "profile": "v7_nav_profile",
}

_URL_PATH_TO_PAGE: dict[str, str] = {url_path: slug for slug, url_path in _PAGE_URL_PATHS.items()}

# URLs propres pour les sections v7 (st.navigation)
# La section "home" utilise url_path="" pour être à la racine (/).
V7_SECTION_URL_PATHS: dict[str, str] = {
    "home": "",
    "stats": "stats",
    "squad": "squad",
    "explorer": "explorer",
    "media": "media",
    "profile": "profile",
}

_V7_LEGACY_PAGE_TO_SECTION: dict[str, str] = {
    "last_match": "home",
    "timeseries": "stats",
    "session_compare": "stats",
    "match_history": "stats",
    "teammates": "squad",
    "explorer": "explorer",
    "media": "media",
    "career": "profile",
    "citations": "profile",
    "settings": "profile",
}

_V7_LEGACY_PAGE_TO_SUBVIEW: dict[str, str] = {
    "timeseries": "timeseries",
    "session_compare": "session_compare",
    "match_history": "match_history",
    "career": "career",
    "citations": "citations",
    "settings": "settings",
}


def get_v7_section_label(section: str) -> str:
    """Retourne le label traduit d'une section v7."""
    i18n_key = _V7_SECTION_I18N_KEYS.get(section, "")
    return t(i18n_key) if i18n_key else section


def get_v7_section_i18n_key(section: str) -> str:
    """Retourne la clé i18n du libellé d'une section v7."""
    return _V7_SECTION_I18N_KEYS.get(section, section)


def normalize_v7_section(section: str | None) -> str:
    """Normalise une section v7 ou retourne la section par défaut."""
    candidate = str(section or "").strip().lower()
    return candidate if candidate in V7_SECTION_KEYS else "home"


def map_legacy_page_to_v7_location(value: str | None) -> tuple[str, str | None]:
    """Mappe une page legacy ou un url_path vers une section/sous-vue v7."""
    raw = str(value or "").strip()
    if not raw:
        return "home", None

    slug = _LEGACY_NAME_TO_SLUG.get(raw, raw)
    slug = _URL_PATH_TO_PAGE.get(slug, slug)
    section = _V7_LEGACY_PAGE_TO_SECTION.get(slug, "home")
    subview = _V7_LEGACY_PAGE_TO_SUBVIEW.get(slug)
    return section, subview


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
        width="stretch",
    )

    if selected and selected != current_title:
        logger.info("Navigation: %s → %s", current_title, selected)
        target = next((p for p in pages if p.title == selected), None)
        if target is not None:
            st.switch_page(target)
