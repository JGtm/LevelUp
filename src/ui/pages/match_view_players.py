"""Section joueurs pour la page Match View — façade de réexport.

Modules internes :
- match_view_players_nemesis   : render_nemesis_section + graphique KV
- match_view_players_timeline  : render_team_dominance_section + render_kd_timeline_section
- (ce fichier)                 : render_roster_section + render_match_impact_section
"""

from __future__ import annotations

import html
import logging
from collections.abc import Callable

import streamlit as st

from src.config import TEAM_MAP, get_bot_name
from src.ui import display_name_from_xuid
from src.ui.formatting import format_time_ms as _format_time
from src.ui.i18n import get_lang, t
from src.ui.pages.match_table_html import gamertag_link
from src.ui.pages.match_view_players_data import (
    has_table_duckdb as _has_table_duckdb,
)
from src.ui.pages.match_view_players_data import (
    load_match_players_stats as _load_match_players_stats,
)
from src.ui.pages.match_view_players_nemesis import render_nemesis_section  # noqa: F401
from src.ui.pages.match_view_players_timeline import (  # noqa: F401
    render_kd_timeline_section,
    render_team_dominance_section,
)
from src.ui.pages.match_view_scoreboard import (
    render_match_scoreboard,  # noqa: F401 — re-export
)
from src.ui.streamlit_modern import PLOTLY_STATIC_CONFIG, fragment_if_available
from src.utils import parse_xuid_input
from src.visualization.match_impact_timeline import (
    compute_single_match_impact,
    get_impact_labels,
    plot_match_kill_death_timeline,
)

logger = logging.getLogger(__name__)


# =============================================================================
# Section Roster
# =============================================================================


def render_roster_section(  # noqa: C901, PLR0913
    *,
    match_id: str,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    load_match_rosters_fn: Callable,
    load_match_gamertags_fn: Callable,
) -> None:
    """Rend la section Joueurs (roster)."""
    st.subheader(t("mv_players_title"))
    rosters = load_match_rosters_fn(db_path, match_id.strip(), xuid.strip(), db_key=db_key)
    if not rosters:
        st.info(t("mv_roster_unavailable"))
        return

    gt_map = load_match_gamertags_fn(db_path, match_id.strip(), db_key=db_key)
    me_xu = str(parse_xuid_input(str(xuid or "").strip()) or str(xuid or "").strip()).strip()

    my_team_id = rosters.get("my_team_id")
    my_team_name = rosters.get("my_team_name")
    enemy_team_ids = rosters.get("enemy_team_ids") or []
    enemy_team_names = rosters.get("enemy_team_names") or []

    def _team_label(team_id_value) -> str:
        try:
            tid = int(team_id_value)
        except Exception:
            return "-"
        return TEAM_MAP.get(tid) or f"Team {tid}"

    def _roster_name(xu: str, gt: str | None) -> str:
        xu_s = str(parse_xuid_input(str(xu or "").strip()) or str(xu or "").strip()).strip()

        if xu_s:
            bot_key = xu_s.strip()
            if bot_key.lower().startswith("bid("):
                bot_name = get_bot_name(bot_key)
                if isinstance(bot_name, str) and bot_name.strip():
                    return bot_name.strip()

        if xu_s and isinstance(gt_map, dict):
            mapped = gt_map.get(xu_s)
            if isinstance(mapped, str) and mapped.strip():
                return mapped.strip()

        g = str(gt or "").strip()
        if g and g != "?" and (not g.isdigit()) and (not g.lower().startswith("xuid(")):
            return g

        if xu_s:
            return display_name_from_xuid(xu_s)
        return "-"

    my_rows = rosters.get("my_team") or []
    en_rows = rosters.get("enemy_team") or []

    my_names: list[tuple[str, bool]] = []
    en_names: list[tuple[str, bool]] = []

    for r in my_rows:
        xu = str(r.get("xuid") or "").strip()
        name = str(r.get("display_name") or "").strip() or _roster_name(xu, r.get("gamertag"))
        is_self = bool(me_xu and xu and (str(parse_xuid_input(xu) or xu).strip() == me_xu)) or bool(
            r.get("is_me")
        )
        my_names.append((name, is_self))

    for r in en_rows:
        xu = str(r.get("xuid") or "").strip()
        name = str(r.get("display_name") or "").strip() or _roster_name(xu, r.get("gamertag"))
        en_names.append((name, False))

    rows_n = max(len(my_names), len(en_names), 1)
    my_names += [("", False)] * (rows_n - len(my_names))
    en_names += [("", False)] * (rows_n - len(en_names))

    def _pill_html(name: str, *, side: str, is_self: bool) -> str:
        if not name:
            return "<span class='os-roster-empty'>—</span>"
        display = gamertag_link(name) if not is_self else html.escape(str(name))
        extra = " os-roster-pill--self" if is_self else ""
        return (
            f"<span class='os-roster-pill os-roster-pill--{side}{extra}'>"
            "<span class='os-roster-pill__dot'></span>"
            f"<span>{display}</span>"
            "</span>"
        )

    body_rows = []
    for i in range(rows_n):
        n_me, is_self = my_names[i]
        n_en, _ = en_names[i]
        body_rows.append(
            "<tr>"
            f"<td>{_pill_html(n_me, side='me', is_self=is_self)}</td>"
            f"<td>{_pill_html(n_en, side='enemy', is_self=False)}</td>"
            "</tr>"
        )

    _my_team_display = html.escape(str(my_team_name or _team_label(my_team_id)))
    _my_count = len([n for n, _ in my_names if n])
    _enemy_raw = (
        enemy_team_names[0]
        if (
            isinstance(enemy_team_names, list)
            and len(enemy_team_names) == 1
            and enemy_team_names[0]
        )
        else (
            " / ".join([_team_label(t_id) for t_id in enemy_team_ids])
            if enemy_team_ids
            else t("mv_roster_opponents")
        )
    )
    _enemy_display = html.escape(str(_enemy_raw))
    _enemy_count = len([n for n, _ in en_names if n])

    st.markdown(
        "<div class='os-table-wrap os-roster-wrap'>"
        "<table class='os-table os-roster'>"
        "<thead><tr>"
        f"<th class='os-roster-th os-roster-th--me'>{t('mv_roster_my_team', name=_my_team_display, n=_my_count)}</th>"
        f"<th class='os-roster-th os-roster-th--enemy'>{t('mv_roster_enemy_team', name=_enemy_display, n=_enemy_count)}</th>"
        "</tr></thead>"
        "<tbody>" + "".join(body_rows) + "</tbody>"
        "</table>"
        "</div>",
        unsafe_allow_html=True,
    )


# =============================================================================
# Section Impact & Timeline
# =============================================================================


@fragment_if_available
def render_match_impact_section(  # noqa: PLR0913
    *,
    match_id: str,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    outcome: int | None,
    load_highlight_events_fn: Callable,
    load_match_gamertags_fn: Callable,
) -> None:
    """Rend la section Impact & Timeline pour un match unique.

    Affiche un graphe chronologique kills/deaths cumulées du joueur,
    avec annotations des événements d'impact (premier sang, finisseur,
    touriste, première victime).
    """
    st.subheader(t("mv_impact_title"))

    if not (match_id and match_id.strip() and _has_table_duckdb(db_path, "highlight_events")):
        logger.debug("impact: table highlight_events absente pour match=%s", match_id)
        st.caption(t("mv_impact_no_events"))
        return

    with st.spinner(t("mv_impact_computing")):
        he = load_highlight_events_fn(db_path, match_id.strip(), db_key=db_key)

    if not he:
        st.info(t("mv_impact_no_events_match"))
        return

    me_xuid = str(parse_xuid_input(str(xuid or "").strip()) or str(xuid or "").strip()).strip()
    gt_map = load_match_gamertags_fn(db_path, match_id.strip(), db_key=db_key)

    # Résoudre les xuids de l'équipe alliée pour filtrer les events d'impact
    team_xuids: set[str] | None = None
    all_players = _load_match_players_stats(db_path, match_id.strip())
    if all_players:
        xuid_to_team = {
            str(p.get("xuid", "")).strip(): p.get("team_id") for p in all_players if p.get("xuid")
        }
        my_team_id = xuid_to_team.get(me_xuid)
        if my_team_id is not None:
            team_xuids = {xu for xu, tid in xuid_to_team.items() if tid == my_team_id}

    # Identifier les événements d'impact (filtrés par équipe alliée)
    impact_events = compute_single_match_impact(
        he,
        me_xuid,
        outcome=outcome,
        team_xuids=team_xuids,
        participants_stats=all_players or None,
        lang=get_lang(),
    )

    # Enrichir les gamertags via gt_map
    if gt_map and isinstance(gt_map, dict):
        enriched = []
        for ie in impact_events:
            resolved = gt_map.get(ie.xuid, ie.gamertag)
            if resolved and resolved != ie.gamertag:
                from src.visualization.match_impact_timeline import MatchImpactEvent

                ie = MatchImpactEvent(
                    event_type=ie.event_type,
                    xuid=ie.xuid,
                    gamertag=resolved,
                    time_ms=ie.time_ms,
                    is_me=ie.is_me,
                    extra_label=ie.extra_label,
                )
            enriched.append(ie)
        impact_events = enriched

    # Badges d'impact — flexbox HTML unique pour contrôler gap et padding
    if impact_events:
        _impact_labels = get_impact_labels(get_lang())
        cards_html: list[str] = []
        for ie in impact_events:
            label_info = _impact_labels.get(ie.event_type)
            if not label_info:
                continue
            _icon, label_fr = label_info
            display_name = ie.gamertag
            display_html = gamertag_link(display_name) if not ie.is_me else html.escape(display_name)
            accent = "#3DFFB5" if ie.is_me else "#FFB703"  # vert si moi, ambre sinon
            time_str = html.escape(str(ie.extra_label if ie.extra_label else _format_time(ie.time_ms)))
            icon_label = html.escape(label_fr)
            cards_html.append(
                f"<div class='os-card' style='padding:10px; min-height:80px; flex:1; min-width:90px;"
                f" border-color:{accent}66;'>"
                f"<div class='os-card-title' style='font-size:14px'>{icon_label}</div>"
                f"<div class='os-card-kpi' style='color:{accent};font-size:15px'>{display_html}</div>"
                f"<div class='os-card-sub'>{time_str}</div>"
                f"</div>"
            )
        if cards_html:
            st.markdown(
                "<div style='display:flex; gap:8px; flex-wrap:wrap; margin-bottom:10px;'>"
                + "".join(cards_html)
                + "</div>",
                unsafe_allow_html=True,
            )

    # Graphe timeline kills/deaths
    fig = plot_match_kill_death_timeline(
        he,
        me_xuid,
        impact_events,
        height=340,
        lang=get_lang(),
    )
    if fig is not None:
        st.plotly_chart(fig, width="stretch", config=PLOTLY_STATIC_CONFIG)
    else:
        st.info(t("mv_impact_too_few"))
    _legend_label = "ℹ️ Légende" if get_lang() == "fr" else "ℹ️ Legend"
    with st.expander(_legend_label, expanded=False):
        st.markdown(t("mv_impact_legend"))


# =============================================================================
# Exports publics
# =============================================================================

__all__ = [
    "render_team_dominance_section",
    "render_nemesis_section",
    "render_roster_section",
    "render_match_impact_section",
    "render_kd_timeline_section",
    "render_match_scoreboard",
]
