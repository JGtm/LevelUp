"""Rendu Streamlit de l'accueil Mission Control V7."""

from __future__ import annotations

import logging
from html import escape
from typing import Any
from urllib.parse import urlencode

import streamlit as st

from src.app.page_router import V7_SECTION_URL_PATHS
from src.ui.i18n import get_lang, t
from src.ui.layout.kpi_bar import render_kpi_bar
from src.ui.pages.home_mission_control_api import (
    HomeBattlepassInfo,
    HomeChallengeSummary,
    fetch_home_progressions,
)
from src.ui.pages.home_mission_control_battlepass_render import render_battlepass_card
from src.ui.pages.home_mission_control_cards import (
    build_action_grid_html,
    build_challenges_card_html,
    build_home_hero_html,
    build_media_block_html,
    build_recent_activity_html,
    build_recent_form_card_html,
    build_recent_highlights_html,
    build_session_summary_card_html,
)
from src.ui.pages.home_mission_control_logic import (
    HomeActionCard,
    HomeMediaEntry,
    HomeRecentMatch,
    HomeSessionSummary,
    SessionCardConfig,
    _build_navigation_state,
    _build_recent_highlights,
    _build_session_summary,
    _format_percent,
    _select_recent_matches,
    _select_recent_media,
)
from src.ui.pages.last_match import render_last_match_page
from src.ui.pages.media_library_data import gamertag_from_db_path, load_media_from_db

logger = logging.getLogger(__name__)


def _section_href(section: str) -> str:
    """Construit le chemin URL d'une section v7."""
    url_path = V7_SECTION_URL_PATHS.get(section, "")
    return "/" if not url_path else f"/{url_path}"


def _home_href(
    section: str,
    *,
    stats_view: str | None = None,
    session_label: str | None = None,
    squad_mode: bool | None = None,
    pending_match_id: str | None = None,
) -> str:
    """Construit une URL interne pour les CTA HTML de la home."""
    params: dict[str, str] = {}
    if stats_view:
        params["stats_view"] = stats_view
    if session_label:
        params["session"] = session_label
        if squad_mode is True:
            params["scope"] = "squad"
        elif squad_mode is False:
            params["scope"] = "solo"
    if pending_match_id:
        params["match_id"] = pending_match_id
    query = urlencode(params)
    base = _section_href(section)
    return base if not query else f"{base}?{query}"


def _set_section(
    section: str,
    *,
    stats_view: str | None = None,
    session_label: str | None = None,
    squad_mode: bool | None = None,
    pending_match_id: str | None = None,
) -> None:
    """Met à jour la navigation V7 et relance l'application."""
    updates = _build_navigation_state(
        section,
        stats_view=stats_view,
        session_label=session_label,
        squad_mode=squad_mode,
        pending_match_id=pending_match_id,
    )
    st.session_state.update(updates)
    logger.info(
        "Mission Control V7: navigation vers %s stats_view=%s session=%s match=%s",
        section,
        stats_view,
        session_label,
        pending_match_id,
    )
    # Navigation via URL (st.navigation) — fallback rerun si pages non disponibles
    pages_dict = st.session_state.get("_v7_pages", {})
    target = pages_dict.get(section)
    if target is not None:
        st.switch_page(target)
    else:
        st.rerun()


def _render_home_card(content: str, *, extra_class: str = "") -> None:
    """Rend une card V7 compacte."""
    classes = "v7-subshell v7-home-card"
    if extra_class:
        classes = f"{classes} {extra_class}"
    st.markdown(f"<div class='{classes}'>{content}</div>", unsafe_allow_html=True)


def _build_action_cards(
    solo_summary: HomeSessionSummary | None,
    squad_summary: HomeSessionSummary | None,
    recent_matches: list[HomeRecentMatch],
) -> list[HomeActionCard]:
    """Construit les cartes d'action rapide contextuelles."""
    stats_description = t("v7_home_action_stats_hint")
    squad_description = t("v7_home_action_squad_hint")
    explorer_description = t("v7_home_action_explorer_hint")
    media_description = t("v7_home_action_media_hint")

    if solo_summary is not None:
        stats_description = (
            f"{solo_summary.session_label} · {solo_summary.match_count} {t('lbl_parties')}"
        )
    if squad_summary is not None:
        squad_description = (
            f"{squad_summary.session_label} · WR "
            f"{_format_percent(squad_summary.kpis.win_rate * 100)}"
        )
    if recent_matches:
        explorer_description = recent_matches[0].title

    return [
        HomeActionCard(
            title=t("v7_nav_stats"),
            description=stats_description,
            button_label=t("v7_home_open_section"),
            button_key="v7_home_go_stats",
            target_section="stats",
            stats_view="timeseries",
            session_label=solo_summary.session_label if solo_summary else None,
            squad_mode=False,
        ),
        HomeActionCard(
            title=t("page_teammates"),
            description=squad_description,
            button_label=t("v7_home_open_section"),
            button_key="v7_home_go_squad",
            target_section="squad",
            session_label=squad_summary.session_label if squad_summary else None,
            squad_mode=True,
        ),
        HomeActionCard(
            title=t("page_explorer"),
            description=explorer_description,
            button_label=t("v7_home_open_match"),
            button_key="v7_home_go_explorer",
            target_section="explorer",
            pending_match_id=recent_matches[0].match_id if recent_matches else None,
        ),
        HomeActionCard(
            title=t("page_media"),
            description=media_description,
            button_label=t("v7_home_open_section"),
            button_key="v7_home_go_media",
            target_section="media",
        ),
    ]


def _render_recent_form_card(matches_df: Any) -> None:
    """Affiche la carte de forme récente (5 derniers matchs) pour l'accueil."""
    _render_home_card(build_recent_form_card_html(matches_df))


def _render_battlepass_card(info: HomeBattlepassInfo | None) -> None:
    """Affiche la carte de progression du pass de combat."""
    render_battlepass_card(info, _render_home_card)


def _render_challenges_card(summary: HomeChallengeSummary | None) -> None:
    """Affiche la carte des défis actifs."""
    _render_home_card(build_challenges_card_html(summary))


def _format_challenge_progress(summary: HomeChallengeSummary) -> str:
    """Compatibilité tests: délègue le formatage au module de cartes home."""
    from src.ui.pages.home_mission_control_cards import _format_challenge_progress as _delegate

    return _delegate(summary)


def _render_action_cards(cards: list[HomeActionCard]) -> None:
    """Affiche les cartes d'action rapide."""

    def _card_href(card: HomeActionCard) -> str:
        return _home_href(
            card.target_section,
            stats_view=card.stats_view,
            session_label=card.session_label,
            squad_mode=card.squad_mode,
            pending_match_id=card.pending_match_id,
        )

    st.markdown(
        f"<div class='v7-section-title'>{escape(t('v7_home_quick_actions'))}</div>",
        unsafe_allow_html=True,
    )
    _render_home_card(build_action_grid_html(cards, _card_href), extra_class="v7-home-action-shell")


def _render_session_summary_card(
    summary: HomeSessionSummary | None, config: SessionCardConfig
) -> None:
    """Affiche une carte de session récente avec CTA."""
    href = _home_href(
        config.target_section,
        stats_view="timeseries" if config.target_section == "stats" else None,
        session_label=summary.session_label if summary else None,
        squad_mode=config.squad_mode,
    )
    _render_home_card(
        build_session_summary_card_html(
            title=config.title,
            summary=summary,
            empty_text=config.empty_text,
            cta_label=config.button_label,
            href=href,
        )
    )


def _render_recent_activity_card(matches: list[HomeRecentMatch]) -> None:
    """Affiche une timeline compacte des derniers matchs."""

    def _match_href(match: HomeRecentMatch) -> str:
        return _home_href("explorer", pending_match_id=match.match_id)

    _render_home_card(build_recent_activity_html(matches, _match_href))


def _render_recent_media_block(entries: list[HomeMediaEntry]) -> None:
    """Affiche les médias récents liés au joueur courant."""
    st.markdown(
        f"<div class='v7-section-title'>{escape(t('v7_home_recent_media'))}</div>",
        unsafe_allow_html=True,
    )
    _render_home_card(build_media_block_html(entries, _home_href("media")))


def render_home_mission_control(ctx: Any) -> None:
    """Rend l'accueil V7 Mission Control."""
    media_df = load_media_from_db(
        ctx.db_path,
        xuid=ctx.xuid,
        gamertag=gamertag_from_db_path(ctx.db_path),
    )
    player_name = gamertag_from_db_path(ctx.db_path) or str(ctx.xuid or "-")
    media_entries = _select_recent_media(media_df)
    recent_matches = _select_recent_matches(ctx.df)
    solo_summary = _build_session_summary(ctx.df, ctx.base_s_ui, squad_mode=False)
    squad_summary = _build_session_summary(ctx.df, ctx.base_s_ui, squad_mode=True)
    highlights = _build_recent_highlights(ctx.df, ctx.base_s_ui)

    hero_col, signal_col = st.columns([1.7, 1.0], gap="large")
    with hero_col:
        _render_home_card(
            build_home_hero_html(
                player_name=player_name,
                matches_df=ctx.df,
                solo_summary=solo_summary,
                squad_summary=squad_summary,
            ),
            extra_class="v7-home-hero",
        )
    with signal_col:
        _render_home_card(
            build_recent_highlights_html(highlights),
            extra_class="v7-home-signals-card",
        )

    render_kpi_bar(ctx.df)

    # Accès rapides
    _render_action_cards(_build_action_cards(solo_summary, squad_summary, recent_matches))

    # Row 1 : Forme récente | Dernière session escouade | Activité récente
    form_col, squad_col, activity_col = st.columns([1.0, 1.0, 1.2], gap="large")
    with form_col:
        _render_recent_form_card(ctx.df)
    with squad_col:
        _render_session_summary_card(
            squad_summary,
            SessionCardConfig(
                title=t("v7_home_recent_squad"),
                empty_text=t("v7_home_no_recent_squad"),
                button_label=t("v7_home_open_scope"),
                button_key="v7_home_open_squad",
                target_section="squad",
                squad_mode=True,
            ),
        )
    with activity_col:
        _render_recent_activity_card(recent_matches)

    # Row 2 : Pass de combat | Défis actifs
    bp_info, challenges = fetch_home_progressions(
        ctx.db_path,
        ctx.xuid,
        gamertag=gamertag_from_db_path(ctx.db_path),
        lang=get_lang(),
    )
    bp_col, chal_col = st.columns(2)
    with bp_col:
        _render_battlepass_card(bp_info)
    with chal_col:
        _render_challenges_card(challenges)

    # Row 3 : Dernier match
    st.markdown(
        f"<div class='v7-section-title'>{escape(t('v7_home_last_match'))}</div>",
        unsafe_allow_html=True,
    )
    render_last_match_page(dff=ctx.df, params=ctx.match_view_params)

    # Row 4 : Médias récents
    _render_recent_media_block(media_entries)
