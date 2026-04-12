"""Rendu Streamlit de l'accueil Mission Control V7."""

from __future__ import annotations

import logging
from html import escape
from typing import Any

import streamlit as st

from src.ui.formatting import format_datetime_fr_hm, format_duration_dhm
from src.ui.i18n import get_lang, t
from src.ui.pages.home_mission_control_api import (
    HomeBattlepassInfo,
    HomeChallengeSummary,
    fetch_home_progressions,
)
from src.ui.pages.home_mission_control_logic import (
    HomeActionCard,
    HomeMediaEntry,
    HomeRecentMatch,
    HomeSessionSummary,
    SessionCardConfig,
    _build_navigation_state,
    _build_session_summary,
    _compute_trend_snapshot,
    _format_percent,
    _format_ratio,
    _select_recent_matches,
    _select_recent_media,
)
from src.ui.pages.last_match import render_last_match_page
from src.ui.pages.media_library_data import gamertag_from_db_path, load_media_from_db

logger = logging.getLogger(__name__)


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
    recent_matches = _select_recent_matches(matches_df, limit=1)
    trend_snapshot = _compute_trend_snapshot(matches_df)

    latest_line = t("v7_home_hero_summary_empty")
    if recent_matches:
        latest = recent_matches[0]
        latest_line = f"{latest.title} · {latest.detail}"

    stats_html = ""
    trend_line = t("v7_home_trend_na")
    if trend_snapshot is not None:
        stats = trend_snapshot.current_kpis
        stats_html = "".join(
            [
                "<div class='v7-home-stats'>",
                f"<span class='v7-home-stat'><strong>{_format_ratio(stats.global_ratio)}</strong> KD</span>",
                f"<span class='v7-home-stat'><strong>{_format_percent(stats.avg_accuracy)}</strong> {escape(t('col_avg_accuracy'))}</span>",
                f"<span class='v7-home-stat'><strong>{_format_percent(stats.win_rate * 100)}</strong> WR</span>",
                "</div>",
            ]
        )
        if trend_snapshot.ratio_delta is not None:
            trend_line = (
                f"KD {trend_snapshot.ratio_delta:+.2f} · "
                f"ACC {trend_snapshot.accuracy_delta:+.0f}% · "
                f"WR {trend_snapshot.win_rate_delta * 100:+.0f}%"
            )

    _render_home_card(
        "".join(
            [
                f"<div class='v7-subshell-title'>{escape(t('v7_home_recent_form'))}</div>",
                f"<div class='v7-home-meta'>{escape(latest_line)}</div>",
                f"<div class='v7-home-hero-trend'>{escape(trend_line)}</div>",
                stats_html,
            ]
        ),
    )


def _render_battlepass_card(info: HomeBattlepassInfo | None) -> None:
    """Affiche la carte de progression du pass de combat."""
    title = t("v7_home_battlepass")
    if info is None:
        _render_home_card(
            "".join(
                [
                    f"<div class='v7-subshell-title'>{escape(title)}</div>",
                    f"<div class='v7-inline-note'>{escape(t('v7_home_api_unavailable'))}</div>",
                ]
            )
        )
        return
    if info.track_image_bytes:
        st.image(info.track_image_bytes, width="stretch")
    tier_label = t("v7_home_battlepass_premium") if info.is_owned else t("v7_home_battlepass_free")
    career_str = info.career_rank_label or f"Rang {info.career_rank}"
    _render_home_card(
        "".join(
            [
                f"<div class='v7-subshell-title'>{escape(title)}</div>",
                f"<div class='v7-home-meta'>{escape(info.track_name)} · {escape(tier_label)}</div>",
                "<div class='v7-home-stats'>",
                f"<span class='v7-home-stat'><strong>Niv. {info.op_rank}</strong></span>",
                f"<span class='v7-home-stat'>{escape(career_str)}</span>",
                "</div>",
            ]
        )
    )


def _render_challenges_card(summary: HomeChallengeSummary | None) -> None:
    """Affiche la carte des défis actifs."""
    title = t("v7_home_challenges")
    if summary is None:
        _render_home_card(
            "".join(
                [
                    f"<div class='v7-subshell-title'>{escape(title)}</div>",
                    f"<div class='v7-inline-note'>{escape(t('v7_home_api_unavailable'))}</div>",
                ]
            )
        )
        return
    if summary.total == 0:
        _render_home_card(
            "".join(
                [
                    f"<div class='v7-subshell-title'>{escape(title)}</div>",
                    f"<div class='v7-inline-note'>{escape(t('v7_home_challenges_empty'))}</div>",
                ]
            )
        )
        return
    if summary.badge_bytes:
        st.image(summary.badge_bytes, width=88)
    rows = [
        f"<div class='v7-subshell-title'>{escape(title)}</div>",
    ]
    if summary.title:
        rows.append(f"<div class='v7-home-meta'><strong>{escape(summary.title)}</strong></div>")
    if summary.description:
        rows.append(f"<div class='v7-home-meta'>{escape(summary.description)}</div>")
    rows.extend(
        [
            "<div class='v7-home-stats'>",
            f"<span class='v7-home-stat'><strong>{summary.completed}/{summary.total}</strong> {escape(t('v7_home_challenges_done'))}</span>",
            f"<span class='v7-home-stat'><strong>{_format_challenge_progress(summary)}</strong> {escape(t('v7_home_challenges_progress'))}</span>",
            f"<span class='v7-home-stat'><strong>+{summary.xp_available:,}</strong> XP</span>",
            "</div>",
        ]
    )
    if summary.next_expiry:
        expiry_txt = t("v7_home_challenges_expiry").format(date=summary.next_expiry)
        rows.append(f"<div class='v7-home-meta'>{escape(expiry_txt)}</div>")
    _render_home_card("".join(rows))


def _format_challenge_progress(summary: HomeChallengeSummary) -> str:
    """Formate la progression du défi principal pour la carte home."""
    if summary.progress_target is None:
        return "-"
    current = summary.progress_current or 0
    return f"{current}/{summary.progress_target}"


def _render_action_cards(cards: list[HomeActionCard]) -> None:
    """Affiche les cartes d'action rapide."""
    st.markdown(
        f"<div class='v7-section-title'>{escape(t('v7_home_quick_actions'))}</div>",
        unsafe_allow_html=True,
    )
    for column, card in zip(st.columns(len(cards)), cards, strict=False):
        with column:
            _render_home_card(
                "".join(
                    [
                        f"<div class='v7-home-action-kicker'>{escape(card.title)}</div>",
                        f"<div class='v7-home-action-title'>{escape(card.title)}</div>",
                        f"<div class='v7-home-action-body'>{escape(card.description)}</div>",
                    ]
                ),
                extra_class="v7-home-action-card",
            )
            if st.button(card.button_label, key=card.button_key, width="stretch"):
                _set_section(
                    card.target_section,
                    stats_view=card.stats_view,
                    session_label=card.session_label,
                    squad_mode=card.squad_mode,
                    pending_match_id=card.pending_match_id,
                )


def _session_card_html(title: str, summary: HomeSessionSummary) -> str:
    """Construit le HTML d'une carte session."""
    started_at = format_datetime_fr_hm(summary.started_at, lang=get_lang())
    duration_txt = format_duration_dhm(summary.kpis.total_play_seconds, lang=get_lang())
    return "".join(
        [
            f"<div class='v7-subshell-title'>{escape(title)}</div>",
            f"<div class='v7-home-meta'>{escape(summary.session_label)} · {escape(started_at)}</div>",
            "<div class='v7-home-stats'>",
            f"<span class='v7-home-stat'><strong>{summary.match_count}</strong> {escape(t('lbl_parties'))}</span>",
            f"<span class='v7-home-stat'><strong>{escape(duration_txt)}</strong> {escape(t('col_total_duration'))}</span>",
            f"<span class='v7-home-stat'><strong>{_format_ratio(summary.kpis.global_ratio)}</strong> KD</span>",
            f"<span class='v7-home-stat'><strong>{_format_percent(summary.kpis.avg_accuracy)}</strong> {escape(t('col_avg_accuracy'))}</span>",
            f"<span class='v7-home-stat'><strong>{_format_percent(summary.kpis.win_rate * 100)}</strong> WR</span>",
            "</div>",
        ]
    )


def _render_session_summary_card(
    summary: HomeSessionSummary | None, config: SessionCardConfig
) -> None:
    """Affiche une carte de session récente avec CTA."""
    if summary is None:
        _render_home_card(
            "".join(
                [
                    f"<div class='v7-subshell-title'>{escape(config.title)}</div>",
                    f"<div class='v7-inline-note'>{escape(config.empty_text)}</div>",
                ]
            )
        )
    else:
        _render_home_card(_session_card_html(config.title, summary))

    if st.button(config.button_label, key=config.button_key, width="stretch"):
        _set_section(
            config.target_section,
            stats_view="timeseries" if config.target_section == "stats" else None,
            session_label=summary.session_label if summary else None,
            squad_mode=config.squad_mode,
        )


def _render_recent_media_block(entries: list[HomeMediaEntry]) -> None:
    """Affiche les médias récents liés au joueur courant."""
    st.markdown(
        f"<div class='v7-section-title'>{escape(t('v7_home_recent_media'))}</div>",
        unsafe_allow_html=True,
    )
    if not entries:
        _render_home_card(
            f"<div class='v7-inline-note'>{escape(t('v7_home_no_recent_media'))}</div>"
        )
    else:
        rows = ["<div class='v7-home-media-list'>"]
        for entry in entries:
            date_txt = format_datetime_fr_hm(entry.match_start_time, lang=get_lang())
            match_txt = entry.match_id or "-"
            rows.append(
                "<div class='v7-home-media-item'>"
                f"<strong>{escape(entry.basename)}</strong>"
                f"<span>{escape(date_txt)} · {escape(match_txt)}</span>"
                "</div>"
            )
        rows.append("</div>")
        _render_home_card("".join(rows))

    if st.button(t("v7_home_open_section"), key="v7_home_open_media", width="stretch"):
        _set_section("media")


def render_home_mission_control(ctx: Any) -> None:
    """Rend l'accueil V7 Mission Control."""
    media_df = load_media_from_db(
        ctx.db_path,
        xuid=ctx.xuid,
        gamertag=gamertag_from_db_path(ctx.db_path),
    )
    media_entries = _select_recent_media(media_df)
    recent_matches = _select_recent_matches(ctx.df)
    solo_summary = _build_session_summary(ctx.df, ctx.base_s_ui, squad_mode=False)
    squad_summary = _build_session_summary(ctx.df, ctx.base_s_ui, squad_mode=True)

    # Accès rapides
    _render_action_cards(_build_action_cards(solo_summary, squad_summary, recent_matches))

    # Row 1 : Forme récente | Dernière session escouade
    form_col, squad_col = st.columns(2)
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
