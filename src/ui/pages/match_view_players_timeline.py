"""Sections timeline pour Match View — dominance d'équipe et frags cumulés.

Extraites de match_view_players.py — regroupe render_team_dominance_section
et render_kd_timeline_section.
"""

from __future__ import annotations

import contextlib
import logging
from collections.abc import Callable

import streamlit as st

from src.ui.i18n import get_lang, t
from src.ui.pages.match_view_players_data import (
    has_table_duckdb as _has_table_duckdb,
)
from src.ui.pages.match_view_players_data import (
    load_match_players_stats as _load_match_players_stats,
)
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, fragment_if_available
from src.utils import parse_xuid_input
from src.visualization.match_impact_timeline import plot_all_players_frags_timeline
from src.visualization.team_dominance_timeline import (
    compute_dominance_buckets,
    detect_streaks,
    plot_dominance_chart,
)

logger = logging.getLogger(__name__)


@fragment_if_available
def render_team_dominance_section(  # noqa: PLR0913
    *,
    match_id: str,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    is_firefight: bool,
    load_highlight_events_fn: Callable,
) -> None:
    """Rend la frise chronologique de dominance d'équipe (PvP uniquement).

    Affiche deux panneaux liés par l'axe temps :
    - Barres de dominance (tug-of-war) par tranche de 30s.
    - Kill feed individuel avec séries annotées.
    """
    if is_firefight:
        logger.debug("dominance: mode firefight, section ignorée")
        return

    if not (match_id and match_id.strip() and _has_table_duckdb(db_path, "highlight_events")):
        logger.debug("dominance: table highlight_events absente pour match=%s", match_id)
        return

    st.subheader(t("mv_match_dynamics"))

    with st.spinner(t("mv_dynamics_computing")):
        he = load_highlight_events_fn(db_path, match_id.strip(), db_key=db_key)

    if not he:
        st.info(t("mv_dynamics_no_data"))
        return

    all_players = _load_match_players_stats(db_path, match_id.strip())
    if not all_players:
        st.info(t("mv_dynamics_no_roster"))
        return

    me_xuid = str(parse_xuid_input(str(xuid or "").strip()) or str(xuid or "").strip()).strip()

    xuid_to_team: dict[str, int] = {
        str(p.get("xuid", "")).strip(): int(p["team_id"])
        for p in all_players
        if p.get("team_id") is not None and p.get("xuid")
    }
    xuid_to_gamertag: dict[str, str] = {}
    for _p in all_players:
        _xu = str(_p.get("xuid", "")).strip()
        _gt = str(_p.get("gamertag") or "").strip()
        if _xu and _gt and _gt != _xu and not _gt.isdigit() and not _gt.lower().startswith("xuid("):
            xuid_to_gamertag[_xu] = _gt

    if len(set(xuid_to_team.values())) < 2:
        return

    my_team_id = xuid_to_team.get(me_xuid)
    if my_team_id is None:
        st.info(t("mv_dynamics_no_team"))
        return

    kill_events = [
        e
        for e in he
        if str(e.get("event_type", "")).lower() == "kill" and e.get("time_ms") is not None
    ]
    if not kill_events:
        st.info(t("mv_dynamics_no_kills"))
        return

    duration_s = max(int(e["time_ms"]) for e in kill_events) / 1000.0 + 20.0

    buckets = compute_dominance_buckets(he, xuid_to_team, my_team_id, duration_s)
    streaks = detect_streaks(he, xuid_to_team, xuid_to_gamertag)

    fig = plot_dominance_chart(
        buckets=buckets,
        streaks=streaks,
        kill_events=kill_events,
        xuid_to_team=xuid_to_team,
        my_team_id=my_team_id,
        duration_s=duration_s,
        height=360,
    )

    if fig is not None:
        st.plotly_chart(fig, width="stretch", config=PLOTLY_CLEAN_CONFIG)
        st.markdown(t("mv_dominance_legend"))
    else:
        st.info(t("mv_dynamics_no_dominance"))


def render_kd_timeline_section(  # noqa: PLR0913
    *,
    match_id: str,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    load_highlight_events_fn: Callable,
    load_match_gamertags_fn: Callable,
) -> None:
    """Affiche le graphe des frags cumulés de tous les joueurs au fil du match."""
    if not (match_id and match_id.strip() and _has_table_duckdb(db_path, "highlight_events")):
        logger.debug("kd_timeline: table highlight_events absente pour match=%s", match_id)
        return

    try:
        he = load_highlight_events_fn(db_path, match_id.strip(), db_key=db_key)
    except Exception:
        logger.warning("kd_timeline: erreur chargement highlight_events match=%s", match_id)
        he = None

    if not he:
        return

    me_xu = str(parse_xuid_input(str(xuid or "").strip()) or str(xuid or "").strip()).strip()

    gt_map: dict[str, str] = {}
    with contextlib.suppress(Exception):
        gt_map = load_match_gamertags_fn(db_path, match_id.strip(), db_key=db_key) or {}

    fig = plot_all_players_frags_timeline(
        he,
        me_xu,
        gt_map=gt_map,
        height=380,
        lang=get_lang(),
    )
    if fig is not None:
        st.subheader(t("mv_kills_over_time"))
        st.plotly_chart(fig, width="stretch", config=PLOTLY_CLEAN_CONFIG)
