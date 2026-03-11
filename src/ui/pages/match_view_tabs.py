"""Rendus des onglets de la vue match (extraits de match_view.py)."""

from __future__ import annotations

from typing import Any

import streamlit as st

from src.ui.i18n import t
from src.ui.pages.match_view_charts import render_expected_vs_actual
from src.ui.pages.match_view_citations import (
    render_match_citations_section as _render_match_citations_section,
)
from src.ui.pages.match_view_citations import (
    render_medals_tab as _render_medals_tab,
)
from src.ui.pages.match_view_encounters import render_encounter_section
from src.ui.pages.match_view_helpers import render_media_section
from src.ui.pages.match_view_participation import render_participation_section
from src.ui.pages.match_view_players import (
    render_kd_timeline_section,
    render_match_impact_section,
    render_match_scoreboard,
    render_nemesis_section,
    render_team_dominance_section,
)
from src.ui.pages.match_view_weapon_kills import render_weapon_kills_section


def _render_summary_tab(  # noqa: PLR0913
    pm: Any,
    row: dict,
    colors: dict,
    df_full: Any,
    db_path: str,
    xuid: str,
    match_id: str,
    db_key: str,
    *,
    is_abandoned: bool = False,
) -> None:
    """Contenu de l'onglet Résumé."""
    if is_abandoned:
        st.info(t("mv_abandoned_match_desc"))
    elif not pm:
        st.info(t("mv_stats_unavailable"))
    else:
        render_expected_vs_actual(row, pm, colors, df_full=df_full, db_path=db_path, xuid=xuid)
    render_weapon_kills_section(
        db_path=db_path,
        match_id=match_id,
        xuid=xuid,
        colors=colors,
    )
    render_participation_section(
        db_path=db_path,
        match_id=match_id,
        xuid=xuid,
        db_key=db_key,
        match_row=row,
    )


def _render_combat_tab(  # noqa: PLR0913
    match_id: str,
    db_path: str,
    xuid: str,
    db_key: str,
    outcome_code: int,
    row: dict,
    load_highlight_events_fn: Any,
    load_match_gamertags_fn: Any,
    colors: dict,
    *,
    is_abandoned: bool = False,
) -> None:
    """Contenu de l'onglet Combat."""
    if is_abandoned:
        st.info(t("mv_abandoned_match_desc"))
        return
    render_match_impact_section(
        match_id=match_id,
        db_path=db_path,
        xuid=xuid,
        db_key=db_key,
        outcome=outcome_code,
        load_highlight_events_fn=load_highlight_events_fn,
        load_match_gamertags_fn=load_match_gamertags_fn,
    )
    render_team_dominance_section(
        match_id=match_id,
        db_path=db_path,
        xuid=xuid,
        db_key=db_key,
        is_firefight=bool(row.get("is_firefight")),
        load_highlight_events_fn=load_highlight_events_fn,
    )
    render_nemesis_section(
        match_id=match_id,
        db_path=db_path,
        xuid=xuid,
        db_key=db_key,
        colors=colors,
        load_highlight_events_fn=load_highlight_events_fn,
        load_match_gamertags_fn=load_match_gamertags_fn,
    )
    render_kd_timeline_section(
        match_id=match_id,
        db_path=db_path,
        xuid=xuid,
        db_key=db_key,
        load_highlight_events_fn=load_highlight_events_fn,
        load_match_gamertags_fn=load_match_gamertags_fn,
    )


def _render_team_tab(
    match_id: str,
    db_path: str,
    xuid: str,
    db_key: str,
    load_match_gamertags_fn: Any,
) -> None:
    """Contenu de l'onglet Équipe."""
    render_match_scoreboard(
        match_id=match_id,
        db_path=db_path,
        xuid=xuid,
        db_key=db_key,
        load_match_gamertags_fn=load_match_gamertags_fn,
    )
    render_encounter_section(match_id=match_id, self_xuid=xuid, db_path=db_path)


def _render_citations_tab(
    match_id: str,
    db_path: str,
    xuid: str,
    medals_last: Any,
) -> None:
    """Contenu de l'onglet Citations & Médailles."""
    st.subheader(t("mv_citations"))
    _render_match_citations_section(
        match_id=match_id,
        db_path=db_path,
        xuid=xuid.strip(),
    )
    _render_medals_tab(medals_last)


def _render_media_tab(  # noqa: PLR0913
    row: dict,
    settings: Any,
    format_datetime_fn: Any,
    paris_tz: Any,
    gamertag: str | None,
    db_path: str,
    xuid: str,
    match_url: str | None,
) -> None:
    """Contenu de l'onglet Médias."""
    render_media_section(
        row=row,
        settings=settings,
        format_datetime_fn=format_datetime_fn,
        paris_tz=paris_tz,
        gamertag=gamertag,
        db_path=db_path,
        current_xuid=xuid,
    )
    if match_url:
        st.write("")
        st.link_button(t("mv_open_waypoint"), match_url, width="stretch")
