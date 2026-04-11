"""Sections temporaires v7 construites a partir des pages existantes."""

from __future__ import annotations

import logging
from collections.abc import Callable
from typing import Any

import streamlit as st

from src.app._filters_helpers import _to_polars
from src.app.filters import build_friends_opts_map, get_friends_xuids_for_sessions
from src.app.helpers import assign_player_colors
from src.app.page_router import get_page_label
from src.app.session_keys import SK
from src.ui.cache import cached_compute_sessions_db, top_medals_smart
from src.ui.i18n import t
from src.ui.pages import (
    render_career_page,
    render_citations_page,
    render_explorer_page,
    render_match_history_page,
    render_media_v2,
    render_session_comparison_page,
    render_settings_page,
    render_teammates_page,
    render_timeseries_page,
)
from src.ui.pages.home_mission_control import render_home_mission_control
from src.visualization import plot_multi_metric_bars_by_match

logger = logging.getLogger(__name__)


def _render_section_title(title: str) -> None:
    """Affiche un intertitre discret de section."""
    st.markdown(f"<div class='v7-section-title'>{title}</div>", unsafe_allow_html=True)


def _render_segmented_view(
    *,
    state_key: str,
    options: list[str],
    label_fn: Callable[[str], str],
) -> str:
    """Rend une navigation secondaire via segmented control."""
    current = st.session_state.get(state_key, options[0])
    if current not in options:
        current = options[0]
    labels = {option: label_fn(option) for option in options}
    selected_label = st.segmented_control(
        label_fn(current),
        options=list(labels.values()),
        default=labels[current],
        key=f"{state_key}_widget",
        label_visibility="collapsed",
        width="stretch",
    )
    reverse_map = {label: option for option, label in labels.items()}
    selected = reverse_map.get(selected_label, current)
    if selected != current:
        logger.info("Navigation secondaire V7: %s %s -> %s", state_key, current, selected)
    st.session_state[state_key] = selected
    return selected


def _build_sessions_for_compare(ctx: Any):
    """Construit le DataFrame de sessions attendu par la page de comparaison."""
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
        drop_cols = [
            column for column in ("session_id", "session_label") if column in df_pl.columns
        ]
        df_for_merge = df_pl.drop(drop_cols) if drop_cols else df_pl
        sessions_for_compare = df_for_merge.join(
            all_sessions_pl.select(sess_cols),
            on="match_id",
            how="inner",
        )
        n_sessions = (
            sessions_for_compare.select("session_label").n_unique()
            if "session_label" in sessions_for_compare.columns
            and not sessions_for_compare.is_empty()
            else 0
        )
        if n_sessions < 2:
            return all_sessions_pl
        return sessions_for_compare
    return all_sessions_pl


def render_stats_section(ctx: Any) -> None:
    """Rend la section Stats v7 a partir des vues existantes."""
    _render_section_title(t("v7_nav_stats"))
    view = _render_segmented_view(
        state_key=SK.V7_STATS_VIEW,
        options=["timeseries", "session_compare", "match_history"],
        label_fn=get_page_label,
    )
    logger.debug("Rendu section V7 stats: vue=%s", view)

    with st.container(border=True):
        if view == "session_compare":
            render_session_comparison_page(_build_sessions_for_compare(ctx), df_full=ctx.dff)
            return
        if view == "match_history":
            render_match_history_page(
                dff=ctx.dff,
                waypoint_player=ctx.waypoint_player,
                db_path=ctx.db_path,
                xuid=ctx.xuid,
                db_key=ctx.db_key,
                df_full=ctx.df,
            )
            return

        render_timeseries_page(
            ctx.dff,
            df_full=ctx.df,
            base=ctx.base,
            picked_session_labels=ctx.picked_session_labels,
            db_path=ctx.db_path,
            xuid=ctx.xuid,
            db_key=ctx.db_key,
        )


def render_profile_section(ctx: Any) -> None:
    """Rend la section Profil v7 a partir des vues existantes."""
    _render_section_title(t("v7_nav_profile"))
    view = _render_segmented_view(
        state_key=SK.V7_PROFILE_VIEW,
        options=["career", "citations", "settings"],
        label_fn=get_page_label,
    )
    logger.debug("Rendu section V7 profile: vue=%s", view)

    with st.container(border=True):
        if view == "citations":
            render_citations_page(
                dff=ctx.dff,
                df_full=ctx.df,
                xuid=ctx.xuid,
                db_path=ctx.db_path,
                db_key=ctx.db_key,
                top_medals_fn=top_medals_smart,
            )
            return

        if view == "settings":
            render_settings_page(ctx.settings)
            return

        render_career_page(
            db_path=ctx.db_path,
            xuid=ctx.xuid,
            db_key=ctx.db_key,
            waypoint_player=ctx.waypoint_player,
        )


def render_v7_section(active_section: str, ctx: Any) -> None:
    """Dispatch principal des six sections v7."""
    logger.debug("Dispatch section V7: %s", active_section)
    if active_section == "home":
        _render_section_title(t("v7_nav_home"))
        render_home_mission_control(ctx)
        return

    if active_section == "stats":
        render_stats_section(ctx)
        return

    if active_section == "squad":
        _render_section_title(get_page_label("teammates"))
        with st.container(border=True):
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
                base_s_ui=ctx.base_s_ui,
                include_firefight=True,
                waypoint_player=ctx.waypoint_player,
                build_friends_opts_map_fn=build_friends_opts_map,
                assign_player_colors_fn=assign_player_colors,
                plot_multi_metric_bars_fn=plot_multi_metric_bars_by_match,
                top_medals_fn=top_medals_smart,
            )
        return

    if active_section == "explorer":
        _render_section_title(get_page_label("explorer"))
        with st.container(border=True):
            render_explorer_page(df=ctx.df, dff=ctx.df, params=ctx.match_view_params)
        return

    if active_section == "media":
        _render_section_title(get_page_label("media"))
        with st.container(border=True):
            render_media_v2(df_full=ctx.df, settings=ctx.settings)
        return

    render_profile_section(ctx)
