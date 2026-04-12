"""Helpers de rendu HTML pour Mission Control V7."""

from __future__ import annotations

import base64
from collections.abc import Callable, Sequence
from html import escape
from typing import Any

from src.ui.formatting import format_datetime_fr_hm, format_duration_dhm
from src.ui.i18n import get_lang, t
from src.ui.pages.home_mission_control_api import HomeChallengeSummary
from src.ui.pages.home_mission_control_logic import (
    HomeActionCard,
    HomeHighlight,
    HomeMediaEntry,
    HomeRecentMatch,
    HomeSessionSummary,
    _compute_trend_snapshot,
    _format_percent,
    _format_ratio,
    _select_recent_matches,
)


def _image_data_url(image_bytes: bytes | None) -> str | None:
    """Encode une image en data URL pour un rendu inline."""
    if not image_bytes:
        return None
    encoded = base64.b64encode(image_bytes).decode("ascii")
    return f"data:image/png;base64,{encoded}"


def _build_hero_chips(
    recent_matches: list[HomeRecentMatch],
    solo_summary: HomeSessionSummary | None,
    squad_summary: HomeSessionSummary | None,
) -> str:
    """Construit les marqueurs de contexte affichés dans le hero home."""
    chips: list[str] = []
    if recent_matches:
        chips.append(
            "<span class='v7-chip'>"
            f"<strong>{escape(t('v7_home_last_match'))}</strong>{escape(recent_matches[0].title)}"
            "</span>"
        )
    if solo_summary is not None:
        chips.append(
            "<span class='v7-chip'>"
            f"<strong>{escape(t('v7_home_recent_solo'))}</strong>{escape(solo_summary.session_label)}"
            "</span>"
        )
    if squad_summary is not None:
        chips.append(
            "<span class='v7-chip'>"
            f"<strong>{escape(t('v7_home_recent_squad'))}</strong>{escape(squad_summary.session_label)}"
            "</span>"
        )
    if not chips:
        return ""
    return f"<div class='v7-home-chip-row'>{''.join(chips)}</div>"


def build_home_hero_html(
    *,
    player_name: str,
    matches_df: Any,
    solo_summary: HomeSessionSummary | None,
    squad_summary: HomeSessionSummary | None,
) -> str:
    """Construit le HTML du briefing principal de l'accueil."""
    recent_matches = _select_recent_matches(matches_df, limit=1)
    trend_snapshot = _compute_trend_snapshot(matches_df)

    latest_line = t("v7_home_hero_summary_empty")
    if recent_matches:
        latest = recent_matches[0]
        latest_line = f"{latest.title} · {latest.detail}"

    trend_line = t("v7_home_trend_na")
    stats_html = ""
    if trend_snapshot is not None:
        stats = trend_snapshot.current_kpis
        stats_html = "".join(
            [
                "<div class='v7-home-hero-stats'>",
                f"<span class='v7-home-hero-stat'><strong>{_format_ratio(stats.global_ratio)}</strong> KD</span>",
                f"<span class='v7-home-hero-stat'><strong>{_format_percent(stats.avg_accuracy)}</strong> {escape(t('col_avg_accuracy'))}</span>",
                f"<span class='v7-home-hero-stat'><strong>{_format_percent(stats.win_rate * 100)}</strong> WR</span>",
                f"<span class='v7-home-hero-stat'><strong>{stats.total_matches}</strong> {escape(t('lbl_parties'))}</span>",
                "</div>",
            ]
        )
        if trend_snapshot.ratio_delta is not None:
            trend_line = (
                f"KD {trend_snapshot.ratio_delta:+.2f} · "
                f"ACC {trend_snapshot.accuracy_delta:+.0f}% · "
                f"WR {trend_snapshot.win_rate_delta * 100:+.0f}%"
            )

    return "".join(
        [
            "<div class='v7-home-hero-kicker'>Mission Control</div>",
            f"<div class='v7-home-hero-title'>{escape(player_name)}</div>",
            f"<div class='v7-home-hero-summary'>{escape(t('v7_home_hero_title'))}</div>",
            f"<div class='v7-home-hero-brief'>{escape(latest_line)}</div>",
            _build_hero_chips(recent_matches, solo_summary, squad_summary),
            stats_html,
            f"<div class='v7-home-hero-trend'>{escape(trend_line)}</div>",
        ]
    )


def build_recent_highlights_html(highlights: list[HomeHighlight]) -> str:
    """Construit le HTML de la carte de signaux récents."""
    if not highlights:
        return "".join(
            [
                f"<div class='v7-subshell-title'>{escape(t('v7_home_recent_highlights'))}</div>",
                f"<div class='v7-inline-note'>{escape(t('v7_home_hero_summary_empty'))}</div>",
            ]
        )

    rows = [
        f"<div class='v7-subshell-title'>{escape(t('v7_home_recent_highlights'))}</div>",
        "<div class='v7-home-signal-grid'>",
    ]
    for highlight in highlights:
        rows.append(
            "".join(
                [
                    "<div class='v7-home-signal-item'>",
                    f"<div class='v7-home-signal-title'>{escape(highlight.title)}</div>",
                    f"<div class='v7-home-signal-value'>{escape(highlight.value)}</div>",
                    f"<div class='v7-home-signal-detail'>{escape(highlight.detail)}</div>",
                    "</div>",
                ]
            )
        )
    rows.append("</div>")
    return "".join(rows)


def build_challenges_card_html(summary: HomeChallengeSummary | None) -> str:
    """Construit le HTML de la carte des défis actifs."""
    title = t("v7_home_challenges")
    if summary is None:
        return "".join(
            [
                f"<div class='v7-subshell-title'>{escape(title)}</div>",
                f"<div class='v7-inline-note'>{escape(t('v7_home_api_unavailable'))}</div>",
            ]
        )
    if summary.total == 0:
        return "".join(
            [
                f"<div class='v7-subshell-title'>{escape(title)}</div>",
                f"<div class='v7-inline-note'>{escape(t('v7_home_challenges_empty'))}</div>",
            ]
        )

    badge_html = ""
    badge_url = _image_data_url(summary.badge_bytes)
    if badge_url is not None:
        badge_html = f"<div class='v7-home-card-badge'><img src='{badge_url}' alt=''></div>"

    rows = [
        "<div class='v7-home-card-head'>",
        "<div>",
        f"<div class='v7-subshell-title'>{escape(title)}</div>",
    ]
    if summary.title:
        rows.append(f"<div class='v7-home-meta'><strong>{escape(summary.title)}</strong></div>")
    if summary.description:
        rows.append(f"<div class='v7-home-meta'>{escape(summary.description)}</div>")
    rows.extend(["</div>", badge_html, "</div>"])
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
    return "".join(rows)


def build_action_grid_html(
    cards: Sequence[HomeActionCard],
    href_for: Callable[[HomeActionCard], str],
) -> str:
    """Construit la grille d'actions rapides en HTML pur."""
    rows = ["<div class='v7-home-action-grid'>"]
    for card in cards:
        href = href_for(card)
        rows.append(
            "".join(
                [
                    f"<a class='v7-home-action-link' href='{escape(href, quote=True)}' target='_self'>",
                    f"<div class='v7-home-action-title'>{escape(card.title)}</div>",
                    f"<div class='v7-home-action-body'>{escape(card.description)}</div>",
                    f"<span class='v7-home-card-cta'>{escape(card.button_label)}</span>",
                    "</a>",
                ]
            )
        )
    rows.append("</div>")
    return "".join(rows)


def build_recent_form_card_html(matches_df: Any) -> str:
    """Construit la carte de forme récente de la home."""
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

    return "".join(
        [
            f"<div class='v7-subshell-title'>{escape(t('v7_home_recent_form'))}</div>",
            f"<div class='v7-home-meta'>{escape(latest_line)}</div>",
            f"<div class='v7-home-hero-trend'>{escape(trend_line)}</div>",
            stats_html,
        ]
    )


def build_session_summary_card_html(
    *,
    title: str,
    summary: HomeSessionSummary | None,
    empty_text: str,
    cta_label: str,
    href: str,
) -> str:
    """Construit une carte de session avec CTA HTML."""
    lang = get_lang()
    if summary is None:
        body = "".join(
            [
                f"<div class='v7-subshell-title'>{escape(title)}</div>",
                f"<div class='v7-inline-note'>{escape(empty_text)}</div>",
            ]
        )
    else:
        started_at = format_datetime_fr_hm(summary.started_at, lang=lang)
        duration_txt = format_duration_dhm(summary.kpis.total_play_seconds, lang=lang)
        body = "".join(
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
    return (
        body
        + f"<a class='v7-home-card-cta' href='{escape(href, quote=True)}' target='_self'>{escape(cta_label)}</a>"
    )


def build_recent_activity_html(
    matches: Sequence[HomeRecentMatch],
    href_for: Callable[[HomeRecentMatch], str],
) -> str:
    """Construit la timeline récente en HTML pur."""
    rows = [f"<div class='v7-subshell-title'>{escape(t('v7_home_recent_activity'))}</div>"]
    if not matches:
        rows.append(f"<div class='v7-inline-note'>{escape(t('v7_home_no_recent_activity'))}</div>")
        return "".join(rows)

    rows.append("<div class='v7-home-activity-list'>")
    for match in matches[:4]:
        pill_class = f"v7-home-timeline-pill v7-home-timeline-pill--{escape(match.outcome_tone)}"
        rows.append(
            "".join(
                [
                    f"<a class='v7-home-activity-link' href='{escape(href_for(match), quote=True)}' target='_self'>",
                    "<div class='v7-home-timeline-item'>",
                    f"<span class='{pill_class}'>{escape(match.outcome_label)}</span>",
                    f"<div class='v7-home-timeline-main'>{escape(match.title)}</div>",
                    f"<div class='v7-home-timeline-meta'>{escape(match.detail)}</div>",
                    "</div>",
                    "</a>",
                ]
            )
        )
    rows.append("</div>")
    return "".join(rows)


def build_media_block_html(entries: Sequence[HomeMediaEntry], open_href: str) -> str:
    """Construit le bloc médias récents avec CTA HTML."""
    lang = get_lang()
    rows = [f"<div class='v7-subshell-title'>{escape(t('v7_home_recent_media'))}</div>"]
    if not entries:
        rows.append(f"<div class='v7-inline-note'>{escape(t('v7_home_no_recent_media'))}</div>")
        rows.append(
            f"<a class='v7-home-card-cta' href='{escape(open_href, quote=True)}' target='_self'>{escape(t('v7_home_open_section'))}</a>"
        )
        return "".join(rows)

    rows.append("<div class='v7-home-media-list'>")
    for entry in entries:
        date_txt = format_datetime_fr_hm(entry.match_start_time, lang=lang)
        match_txt = entry.match_id or "-"
        rows.append(
            "<div class='v7-home-media-item'>"
            f"<strong>{escape(entry.basename)}</strong>"
            f"<span>{escape(date_txt)} · {escape(match_txt)}</span>"
            "</div>"
        )
    rows.append("</div>")
    rows.append(
        f"<a class='v7-home-card-cta' href='{escape(open_href, quote=True)}' target='_self'>{escape(t('v7_home_open_section'))}</a>"
    )
    return "".join(rows)


def _format_challenge_progress(summary: HomeChallengeSummary) -> str:
    """Formate la progression du défi principal pour la carte home."""
    if summary.progress_target is None:
        return "-"
    current = summary.progress_current or 0
    return f"{current}/{summary.progress_target}"
