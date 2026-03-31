"""Onglet Impact pour la page Coéquipiers.

Extrait de teammates.py (Sprint 16 — refactoring Phase A).
Heatmap des événements clés + tableau de ranking MVP/Boulet.
"""

from __future__ import annotations

import html

import polars as pl
import streamlit as st

from src.analysis.friends_impact import (
    SCORE_FALSE_BROTHER,
    SCORE_SILENT_HERO,
    SCORE_TOP_KILLER,
    ImpactEventSets,
    build_impact_matrix,
    get_all_impact_events,
    identify_false_brother_multi,
    identify_silent_hero_multi,
    identify_top_killer_multi,
)
from src.data.repositories import DuckDBRepository
from src.ui.i18n import get_lang, t
from src.utils.db import ensure_shared_attached
from src.utils.paths import get_shared_matches_path_from_player


def _load_match_participants(
    conn,
    match_ids: list[str],
    shared_alias: str,
) -> pl.DataFrame | None:
    """Charge les stats assists/deaths depuis shared.match_participants."""
    try:
        conn.execute(f"SELECT 1 FROM {shared_alias}.match_participants LIMIT 1")
    except Exception:
        return None
    query = (
        f"SELECT p.match_id, p.xuid::TEXT as xuid, "
        f"COALESCE(vg.gamertag, p.gamertag, p.xuid::TEXT) as gamertag, "
        f"p.assists, p.deaths, p.kills "
        f"FROM {shared_alias}.match_participants p "
        f"LEFT JOIN {shared_alias}.v_gamertag_lookup vg ON vg.xuid = p.xuid::TEXT "
        f"WHERE p.match_id IN ({', '.join(['?' for _ in match_ids])})"
    )
    rows = conn.execute(query, match_ids).fetchall()
    if not rows:
        return pl.DataFrame()
    return pl.DataFrame(
        {
            "match_id": [str(r[0]) for r in rows],
            "xuid": [str(r[1]) for r in rows],
            "gamertag": [r[2] or "Unknown" for r in rows],
            "assists": [int(r[3] or 0) for r in rows],
            "deaths": [int(r[4] or 0) for r in rows],
            "kills": [int(r[5] or 0) for r in rows],
        }
    )


def _load_highlight_events(
    conn,
    match_ids: list[str],
    shared_alias: str,
) -> pl.DataFrame | None:
    """Charge les événements highlight depuis shared_matches.duckdb.

    Args:
        conn: Connexion DuckDB.
        match_ids: Liste des match_id.
        shared_alias: Alias de la DB shared attachée.

    Returns:
        DataFrame Polars des événements, ou None si indisponible.
    """
    # V5 : highlight_events est dans shared_matches.duckdb
    try:
        conn.execute(f"SELECT 1 FROM {shared_alias}.highlight_events LIMIT 1")
    except Exception:
        return None

    # v6 : v_gamertag_lookup est toujours présente (ensure_v6_views)
    events_query = (
        f"SELECT he.match_id, he.xuid::TEXT as xuid, "
        f"vg.gamertag as gamertag, "
        f"he.event_type, he.time_ms "
        f"FROM {shared_alias}.highlight_events he "
        f"LEFT JOIN {shared_alias}.v_gamertag_lookup vg ON vg.xuid = he.xuid::TEXT "
        f"WHERE he.match_id IN ({', '.join(['?' for _ in match_ids])})"
    )

    events_result = conn.execute(events_query, match_ids).fetchall()

    if not events_result:
        return pl.DataFrame()

    return pl.DataFrame(
        {
            "match_id": [str(r[0]) for r in events_result],
            "xuid": [str(r[1]) for r in events_result],
            "gamertag": [r[2] or "Unknown" for r in events_result],
            "event_type": [r[3] for r in events_result],
            "time_ms": [int(r[4] or 0) for r in events_result],
        }
    )


def _load_match_outcomes(
    conn,
    match_ids: list[str],
    xuid: str,
    shared_alias: str,
) -> pl.DataFrame:
    """Charge les outcomes des matchs depuis shared.match_participants.

    Args:
        conn: Connexion DuckDB.
        match_ids: Liste des match_id.
        xuid: XUID du joueur principal.
        shared_alias: Alias de la DB shared attachée.

    Returns:
        DataFrame avec match_id et outcome du joueur principal.
    """
    # V5 : match_stats n'existe plus, lire depuis shared.match_participants
    matches_query = f"""
        SELECT match_id, outcome
        FROM {shared_alias}.match_participants
        WHERE match_id IN ({{}}) AND xuid = ?
    """.format(", ".join(["?" for _ in match_ids]))

    matches_result = conn.execute(matches_query, match_ids + [xuid]).fetchall()

    return pl.DataFrame(
        {
            "match_id": [str(r[0]) for r in matches_result],
            "outcome": [int(r[1] or 0) for r in matches_result],
        }
    )


_EVENT_TO_EMOJI: dict[str, str] = {
    "first_blood": "⚡",
    "clutch_finisher": "🎯",
    "last_casualty": "💀",
    "last_group_kill": "🐌",
    "first_group_death": "🪦",
    "silent_hero": "🛡️",
    "false_brother": "🗡️",
    "top_killer": "💥",
}
_AGG_KEYS: list[str] = list(_EVENT_TO_EMOJI.keys())
_IMPACT_INVERTED: set[str] = {
    "last_casualty",
    "last_group_kill",
    "first_group_death",
    "false_brother",
}
_OUTCOME_BG: dict[int, str] = {2: "rgba(0,158,115,0.30)", 3: "rgba(213,94,0,0.30)"}
_OUTCOME_BG_TIE = "rgba(100,100,130,0.15)"


def _pivot_matrix_cells(
    impact_matrix: pl.DataFrame, gamertags: list[str], match_ids: list[str]
) -> dict[str, dict[str, str]]:
    """Construit {gamertag: {match_id: emojis}} depuis impact_matrix."""
    cells: dict[str, dict[str, str]] = {gt: {} for gt in gamertags}
    mid_set = set(match_ids)
    for row in impact_matrix.filter(pl.col("gamertag") != "Résultat").iter_rows(named=True):
        gt, mid = row["gamertag"], str(row["match_id"])
        if gt not in cells or mid not in mid_set:
            continue
        emoji_list = [
            _EVENT_TO_EMOJI[e["event"]]
            for e in (row["events"] or [])
            if e["event"] in _EVENT_TO_EMOJI
        ]
        parts: list[str] = []
        for i, e in enumerate(emoji_list):
            parts.append(e)
            if (i + 1) % 2 == 0 and i < len(emoji_list) - 1:
                parts.append("<br>")
        cells[gt][mid] = "".join(parts)
    return cells


def _player_agg_counts(impact_matrix: pl.DataFrame, gamertag: str) -> dict[str, int]:
    """Compte les events par type pour un joueur."""
    counts = dict.fromkeys(_AGG_KEYS, 0)
    for row in impact_matrix.filter(pl.col("gamertag") == gamertag).iter_rows(named=True):
        for ev in row["events"] or []:
            if ev["event"] in counts:
                counts[ev["event"]] += 1
    return counts


def _impact_extremes_from_agg(all_agg: dict[str, dict[str, int]]) -> dict[str, tuple[int, int]]:
    """Min/max par colonne d'agrégat (≥2 valeurs distinctes non nulles)."""
    result: dict[str, tuple[int, int]] = {}
    for key in _AGG_KEYS:
        vals = [agg[key] for agg in all_agg.values() if agg[key] != 0]
        if len(vals) >= 2 and (mn := min(vals)) != (mx := max(vals)):
            result[key] = (mn, mx)
    return result


def _impact_td_class(
    key: str, val: int | float, extremes: dict[str, tuple[int | float, int | float]]
) -> str:
    """Classe CSS best/worst pour une cellule agrégat Impact."""
    if key not in extremes or val == 0:
        return ""
    mn, mx = extremes[key]
    if key in _IMPACT_INVERTED:
        return " os-sb-td--worst" if val == mx else (" os-sb-td--best" if val == mn else "")
    return " os-sb-td--best" if val == mx else (" os-sb-td--worst" if val == mn else "")


def _render_impact_ranking_html(
    impact_matrix: pl.DataFrame,
    scores: dict,
    match_ids_order: list[str],
) -> None:
    """Tableau fusionné : grille match×joueur + totaux par badge + ranking."""
    if not scores or impact_matrix.is_empty():
        return

    outcomes: dict[str, int] = {}
    for row in impact_matrix.filter(pl.col("gamertag") == "Résultat").iter_rows(named=True):
        outcomes[str(row["match_id"])] = int(row["outcome"] or 0)

    sorted_players = sorted(scores.items(), key=lambda x: x[1], reverse=True)
    gamertags = [gt for gt, _ in sorted_players]
    n, last_score = len(gamertags), sorted_players[-1][1] if sorted_players else 0

    cells = _pivot_matrix_cells(impact_matrix, gamertags, match_ids_order)
    all_agg = {gt: _player_agg_counts(impact_matrix, gt) for gt in gamertags}
    extremes = _impact_extremes_from_agg(all_agg)
    score_vals = [s for _, s in sorted_players if s != 0]
    if len(score_vals) >= 2 and (mn := min(score_vals)) != (mx := max(score_vals)):
        extremes["score"] = (mn, mx)

    n_total = 1 + len(match_ids_order) + len(_AGG_KEYS) + 2
    match_ths = "".join(
        f"<th class='os-sb-th' style='width:34px;"
        f"background:{_OUTCOME_BG.get(outcomes.get(mid, 0), _OUTCOME_BG_TIE)}'>{i + 1}</th>"
        for i, mid in enumerate(match_ids_order)
    )
    agg_ths = "".join(f"<th class='os-sb-th'>{_EVENT_TO_EMOJI[k]}</th>" for k in _AGG_KEYS)
    th_row = (
        f"<th class='os-sb-th' style='text-align:left'>{t('tmi_col_player')}</th>"
        f"{match_ths}{agg_ths}"
        f"<th class='os-sb-th'>{t('tmi_col_score')}</th>"
        f"<th class='os-sb-th'>{t('tmi_badge')}</th>"
    )
    header_html = (
        "<tbody class='os-sb-head'>"
        f"<tr><th class='os-sb-team os-sb-team--mine' colspan='{n_total}'>"
        f"{html.escape(t('tm_impact_ranking'))}</th></tr>"
        f"<tr>{th_row}</tr></tbody>"
    )

    body_rows = []
    for rank, (gt, score) in enumerate(sorted_players, start=1):
        if rank == 1:
            row_class, badge = " os-sb-row--mvp", "🏆 Champion"
        elif rank == n and last_score < 0:
            row_class, badge = " os-sb-row--lvp", "🍌 Maillon faible"
        elif rank == n:
            row_class, badge = "", "📉 Passager clandestin"
        else:
            row_class, badge = "", ""
        score_str = f"+{score:g}" if score > 0 else f"{score:g}"
        match_tds = "".join(
            f"<td class='os-sb-td' style='text-align:center'>{cells.get(gt, {}).get(mid, '')}</td>"
            for mid in match_ids_order
        )
        agg_tds = "".join(
            f"<td class='os-sb-td{_impact_td_class(k, all_agg[gt][k], extremes)}'>"
            f"{all_agg[gt][k] if all_agg[gt][k] else '—'}</td>"
            for k in _AGG_KEYS
        )
        tds = (
            f"<td class='os-sb-td' style='text-align:left'>{html.escape(gt)}</td>"
            f"{match_tds}{agg_tds}"
            f"<td class='os-sb-td{_impact_td_class('score', score, extremes)}'>{html.escape(score_str)}</td>"
            f"<td class='os-sb-td'>{html.escape(badge)}</td>"
        )
        body_rows.append(
            f"<tbody class='os-sb-player'><tr class='os-sb-row{row_class}'>{tds}</tr></tbody>"
        )

    table_html = (
        "<div class='os-table-wrap os-sb-wrap'>"
        f"<table class='os-table os-scoreboard os-impact-table'>"
        f"{header_html}{''.join(body_rows)}</table></div>"
    )
    st.markdown(table_html, unsafe_allow_html=True)


def _render_impact_from_events(  # noqa: PLR0913
    events_df: pl.DataFrame,
    matches_df: pl.DataFrame,
    match_ids: list[str],
    friend_xuids: list[str],
    xuid: str,
    participants_df: pl.DataFrame | None = None,
) -> None:
    """Calcule les métriques d'impact et affiche heatmap + ranking."""
    all_friend_xuids = {str(x) for x in friend_xuids}
    all_friend_xuids.add(str(xuid).strip())

    (
        first_bloods,
        clutch_finishers,
        last_casualties,
        last_group_kills,
        first_group_deaths,
        scores,
    ) = get_all_impact_events(events_df, matches_df, friend_xuids=all_friend_xuids)

    # Badges stats-only depuis match_participants
    silent_heroes: dict = {}
    false_brothers: dict = {}
    top_killers: dict = {}
    if participants_df is not None and not participants_df.is_empty():
        silent_heroes = identify_silent_hero_multi(participants_df, matches_df, all_friend_xuids)
        false_brothers = identify_false_brother_multi(participants_df, matches_df, all_friend_xuids)
        top_killers = identify_top_killer_multi(participants_df, all_friend_xuids)
        # Ajouter les scores des badges stats-only au classement
        for ev in silent_heroes.values():
            scores[ev.gamertag] = scores.get(ev.gamertag, 0.0) + SCORE_SILENT_HERO
        for ev in false_brothers.values():
            scores[ev.gamertag] = scores.get(ev.gamertag, 0.0) + SCORE_FALSE_BROTHER
        for ev in top_killers.values():
            scores[ev.gamertag] = scores.get(ev.gamertag, 0.0) + SCORE_TOP_KILLER
        scores = dict(sorted(scores.items(), key=lambda x: x[1], reverse=True))

    if not scores:
        st.info(t("tm_impact_no_events_players"))
        return

    gamertags = list(scores.keys())
    # Ajouter les gamertags des nouveaux badges s'ils ne sont pas déjà dans la liste
    for ev in list(silent_heroes.values()) + list(false_brothers.values()) + list(top_killers.values()):
        if ev.gamertag not in gamertags:
            gamertags.append(ev.gamertag)

    impact_match_set = set(
        list(first_bloods.keys())
        + list(clutch_finishers.keys())
        + list(last_casualties.keys())
        + list(last_group_kills.keys())
        + list(first_group_deaths.keys())
        + list(silent_heroes.keys())
        + list(false_brothers.keys())
        + list(top_killers.keys())
    )
    sorted_match_ids = [m for m in match_ids if m in impact_match_set]

    match_outcomes: dict[str, int] = {}
    if not matches_df.is_empty():
        for row in matches_df.iter_rows(named=True):
            match_outcomes[str(row["match_id"])] = int(row["outcome"])

    impact_matrix = build_impact_matrix(
        ImpactEventSets(
            first_bloods=first_bloods,
            clutch_finishers=clutch_finishers,
            last_casualties=last_casualties,
            last_group_kills=last_group_kills,
            first_group_deaths=first_group_deaths,
            silent_heroes=silent_heroes,
            false_brothers=false_brothers,
            top_killers=top_killers,
        ),
        match_ids=sorted_match_ids,
        gamertags=gamertags,
        match_outcomes=match_outcomes,
    )

    _render_impact_ranking_html(impact_matrix, scores, sorted_match_ids)
    _legend_label = "ℹ️ Légende" if get_lang() == "fr" else "ℹ️ Legend"
    with st.expander(_legend_label, expanded=False):
        st.markdown(t("tm_impact_legend"))


def render_impact_taquinerie(
    db_path: str,
    xuid: str,
    match_ids: list[str],
    friend_xuids: list[str],
    db_key: tuple[int, int] | None = None,
) -> None:
    """Affiche l'onglet Impact (Sprint 12)."""
    if not friend_xuids:
        st.info(t("tm_impact_select_two"))
        return

    if not match_ids:
        st.warning(t("tm_impact_no_matches"))
        return

    try:
        repo = DuckDBRepository(db_path, xuid.strip())
        # _get_connection() légitime : conn est passée aux helpers _load_highlight_events
        # et _load_match_outcomes qui effectuent les requêtes via shared attaché.
        # Pas de query inline ici — rôle purement d'orchestration.
        conn = repo._get_connection()

        _shared_db = get_shared_matches_path_from_player(db_path)
        shared_alias = (
            ensure_shared_attached(conn, _shared_db) if _shared_db and _shared_db.exists() else None
        )
        if not shared_alias:
            st.warning(t("tmi_no_shared_db"))
            return

        events_df = _load_highlight_events(conn, match_ids, shared_alias)
        if events_df is None:
            st.info(t("tmi_no_events"))
            return
        if events_df.is_empty():
            st.info(t("tm_impact_no_events_matches"))
            return

        matches_df = _load_match_outcomes(conn, match_ids, xuid, shared_alias)
        participants_df = _load_match_participants(conn, match_ids, shared_alias)
        _render_impact_from_events(
            events_df, matches_df, match_ids, friend_xuids, xuid, participants_df
        )

    except Exception as e:
        st.warning(t("error_chart", error=e))
