"""Accueil Mission Control pour le cockpit V7."""

from __future__ import annotations

import logging
from dataclasses import dataclass
from html import escape
from typing import Any

import polars as pl
import streamlit as st

from src.app.kpis import KPIStats, compute_kpi_stats
from src.app.session_keys import SK
from src.ui.formatting import format_datetime_fr_hm, format_duration_dhm
from src.ui.i18n import get_lang, t
from src.ui.pages.last_match import render_last_match_page
from src.ui.pages.media_library_data import gamertag_from_db_path, load_media_from_db
from src.visualization._compat import ensure_polars

logger = logging.getLogger(__name__)


@dataclass(frozen=True)
class HomeSessionSummary:
    """Résumé compact d'une session récente."""

    session_label: str
    started_at: object | None
    match_count: int
    kpis: KPIStats


@dataclass(frozen=True)
class HomeMediaEntry:
    """Entrée compacte de média récent."""

    basename: str
    match_id: str | None
    match_start_time: object | None


@dataclass(frozen=True)
class SessionCardConfig:
    """Configuration de rendu d'une carte session."""

    title: str
    empty_text: str
    button_label: str
    button_key: str
    target_section: str


def _format_ratio(value: float | None) -> str:
    """Formate un ratio avec fallback neutre."""
    if value is None:
        return "-"
    return f"{value:.2f}"


def _format_percent(value: float | None) -> str:
    """Formate un pourcentage avec fallback neutre."""
    if value is None:
        return "-"
    return f"{value:.0f}%"


def _set_section(section: str, *, stats_view: str | None = None) -> None:
    """Met à jour la navigation V7 et relance l'application."""
    st.session_state[SK.V7_CURRENT_SECTION] = section
    if stats_view is not None:
        st.session_state[SK.V7_STATS_VIEW] = stats_view
    logger.info("Mission Control V7: navigation vers %s", section)
    st.rerun()


def _get_scope_sessions(sessions_df: Any, *, squad_mode: bool) -> pl.DataFrame:
    """Retourne les sessions d'un scope (solo ou escouade)."""
    sessions_pl = ensure_polars(sessions_df)
    if sessions_pl.is_empty():
        return sessions_pl

    required = {"match_id", "session_label", "start_time"}
    if not required.issubset(set(sessions_pl.columns)):
        return pl.DataFrame()

    if "is_with_friends" not in sessions_pl.columns:
        return pl.DataFrame() if squad_mode else sessions_pl.drop_nulls(subset=["session_label"])

    return sessions_pl.filter(pl.col("is_with_friends") == squad_mode).drop_nulls(
        subset=["session_label"]
    )


def _build_session_summary(
    matches_df: Any,
    sessions_df: Any,
    *,
    squad_mode: bool,
) -> HomeSessionSummary | None:
    """Construit le résumé de la dernière session d'un scope."""
    matches_pl = ensure_polars(matches_df)
    scope_sessions = _get_scope_sessions(sessions_df, squad_mode=squad_mode)
    if matches_pl.is_empty() or scope_sessions.is_empty() or "match_id" not in matches_pl.columns:
        return None

    latest_session = (
        scope_sessions.group_by("session_label")
        .agg(pl.col("start_time").max().alias("last_start"))
        .sort("last_start", descending=True)
    )
    if latest_session.is_empty():
        return None

    latest_label = latest_session.row(0, named=True).get("session_label")
    if latest_label is None:
        return None

    session_match_ids = (
        scope_sessions.filter(pl.col("session_label") == latest_label)
        .get_column("match_id")
        .drop_nulls()
        .unique()
        .to_list()
    )
    if not session_match_ids:
        return None

    session_matches = matches_pl.filter(pl.col("match_id").is_in(session_match_ids))
    if session_matches.is_empty():
        return None

    started_at = None
    if "start_time" in session_matches.columns:
        started_at = session_matches.select(pl.col("start_time").min()).item()

    return HomeSessionSummary(
        session_label=str(latest_label),
        started_at=started_at,
        match_count=len(session_matches),
        kpis=compute_kpi_stats(session_matches),
    )


def _select_recent_media(media_df: Any, limit: int = 3) -> list[HomeMediaEntry]:
    """Sélectionne les médias récents à afficher dans l'accueil."""
    media_pl = ensure_polars(media_df)
    if media_pl.is_empty() or "basename" not in media_pl.columns:
        return []

    sort_column = "mtime_paris_epoch" if "mtime_paris_epoch" in media_pl.columns else None
    if sort_column is None and "match_start_time" in media_pl.columns:
        sort_column = "match_start_time"

    if sort_column:
        media_pl = media_pl.sort(sort_column, descending=True)

    media_pl = media_pl.unique(
        subset=["path"] if "path" in media_pl.columns else ["basename"],
        keep="first",
        maintain_order=True,
    )
    rows = media_pl.head(limit).iter_rows(named=True)
    return [
        HomeMediaEntry(
            basename=str(row.get("basename") or "-"),
            match_id=str(row.get("match_id") or "").strip() or None,
            match_start_time=row.get("match_start_time"),
        )
        for row in rows
    ]


def _render_home_card(content: str) -> None:
    """Rend une card V7 compacte."""
    st.markdown(f"<div class='v7-subshell v7-home-card'>{content}</div>", unsafe_allow_html=True)


def _render_quick_actions() -> None:
    """Affiche les raccourcis principaux de l'accueil."""
    st.markdown(
        f"<div class='v7-section-title'>{escape(t('v7_home_quick_actions'))}</div>",
        unsafe_allow_html=True,
    )
    stats_col, squad_col, explorer_col, media_col = st.columns(4)
    with stats_col:
        if st.button(t("v7_nav_stats"), key="v7_home_go_stats", width="stretch"):
            _set_section("stats", stats_view="timeseries")
    with squad_col:
        if st.button(t("page_teammates"), key="v7_home_go_squad", width="stretch"):
            _set_section("squad")
    with explorer_col:
        if st.button(t("page_explorer"), key="v7_home_go_explorer", width="stretch"):
            _set_section("explorer")
    with media_col:
        if st.button(t("page_media"), key="v7_home_go_media", width="stretch"):
            _set_section("media")


def _session_card_html(title: str, summary: HomeSessionSummary) -> str:
    """Construit le HTML d'une carte session."""
    lang = get_lang()
    started_at = format_datetime_fr_hm(summary.started_at, lang=lang)
    win_label = "V%" if lang == "fr" else "W%"
    duration_txt = format_duration_dhm(summary.kpis.total_play_seconds, lang=lang)
    return "".join(
        [
            f"<div class='v7-subshell-title'>{escape(title)}</div>",
            f"<div class='v7-home-meta'>{escape(summary.session_label)} · {escape(started_at)}</div>",
            "<div class='v7-home-stats'>",
            f"<span class='v7-home-stat'><strong>{summary.match_count}</strong> {escape(t('lbl_parties'))}</span>",
            f"<span class='v7-home-stat'><strong>{escape(duration_txt)}</strong> {escape(t('col_total_duration'))}</span>",
            f"<span class='v7-home-stat'><strong>{_format_ratio(summary.kpis.global_ratio)}</strong> KD</span>",
            f"<span class='v7-home-stat'><strong>{_format_percent(summary.kpis.avg_accuracy)}</strong> {escape(t('col_avg_accuracy'))}</span>",
            f"<span class='v7-home-stat'><strong>{_format_percent(summary.kpis.win_rate * 100)}</strong> {win_label}</span>",
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
        )


def _render_recent_media_block(db_path: str, xuid: str) -> None:
    """Affiche les médias récents liés au joueur courant."""
    st.markdown(
        f"<div class='v7-section-title'>{escape(t('v7_home_recent_media'))}</div>",
        unsafe_allow_html=True,
    )
    media_df = load_media_from_db(
        db_path,
        xuid=xuid,
        gamertag=gamertag_from_db_path(db_path),
    )
    entries = _select_recent_media(media_df)
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

    if st.button(t("page_media"), key="v7_home_open_media", width="stretch"):
        _set_section("media")


def render_home_mission_control(ctx: Any) -> None:
    """Rend l'accueil V7 enrichi autour du dernier match."""
    solo_summary = _build_session_summary(ctx.df, ctx.base_s_ui, squad_mode=False)
    squad_summary = _build_session_summary(ctx.df, ctx.base_s_ui, squad_mode=True)

    _render_quick_actions()

    solo_col, squad_col = st.columns(2)
    with solo_col:
        _render_session_summary_card(
            solo_summary,
            SessionCardConfig(
                title=t("v7_home_recent_solo"),
                empty_text=t("v7_home_no_recent_solo"),
                button_label=t("v7_nav_stats"),
                button_key="v7_home_open_stats",
                target_section="stats",
            ),
        )
    with squad_col:
        _render_session_summary_card(
            squad_summary,
            SessionCardConfig(
                title=t("v7_home_recent_squad"),
                empty_text=t("v7_home_no_recent_squad"),
                button_label=t("page_teammates"),
                button_key="v7_home_open_squad",
                target_section="squad",
            ),
        )

    _render_recent_media_block(ctx.db_path, ctx.xuid)

    st.markdown(
        f"<div class='v7-section-title'>{escape(t('v7_home_last_match'))}</div>",
        unsafe_allow_html=True,
    )
    render_last_match_page(dff=ctx.dff, params=ctx.match_view_params)
