"""Onglet Impact pour la page Coéquipiers.

Extrait de teammates.py (Sprint 16 — refactoring Phase A).
Heatmap des événements clés + tableau de ranking MVP/Boulet.
"""

from __future__ import annotations

import polars as pl
import streamlit as st

from src.analysis.friends_impact import (
    build_impact_matrix,
    get_all_impact_events,
)
from src.data.repositories import DuckDBRepository
from src.ui.i18n import t
from src.ui.streamlit_modern import PLOTLY_STATIC_CONFIG
from src.utils.db import ensure_shared_attached
from src.utils.paths import get_shared_matches_path_from_player
from src.visualization.friends_impact_heatmap import (
    build_impact_ranking_df,
    count_events_by_player,
    plot_friends_impact_heatmap,
    render_impact_summary_stats,
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


def _render_impact_stats(
    first_bloods: dict,
    clutch_finishers: dict,
    last_casualties: dict,
) -> None:
    """Affiche les métriques résumées d'impact."""
    stats = render_impact_summary_stats(first_bloods, clutch_finishers, last_casualties)
    cols = st.columns(4)
    cols[0].metric(t("tmi_first_blood"), stats["total_fb"])
    cols[1].metric(t("tmi_finisher"), stats["total_clutch"])
    cols[2].metric(t("tmi_liability"), stats["total_casualty"])
    cols[3].metric(t("tmi_matches_analyzed"), stats["total_matches"])


def _render_ranking_table(
    scores: dict,
    first_bloods: dict,
    clutch_finishers: dict,
    last_casualties: dict,
) -> None:
    """Affiche le tableau de classement MVP/Boulet."""
    fb_counts = count_events_by_player(first_bloods)
    clutch_counts = count_events_by_player(clutch_finishers)
    casualty_counts = count_events_by_player(last_casualties)

    ranking_df = build_impact_ranking_df(
        scores,
        first_blood_counts=fb_counts,
        clutch_counts=clutch_counts,
        casualty_counts=casualty_counts,
    )

    if not ranking_df.is_empty():
        # Sprint 19 : renommer les colonnes en Polars sans conversion Pandas
        display_df = ranking_df.rename(
            dict(
                zip(
                    ranking_df.columns,
                    [
                        t("tmi_col_rank"),
                        t("tmi_col_player"),
                        t("tmi_col_score"),
                        t("tmi_col_first_blood"),
                        t("tmi_col_finisher"),
                        t("tmi_col_casualty"),
                        t("tmi_badge"),
                    ],
                    strict=False,
                )
            )
        )
        st.dataframe(display_df, width="stretch", hide_index=True)

        mvp = ranking_df[0, "gamertag"] if len(ranking_df) > 0 else None
        boulet = (
            ranking_df[-1, "gamertag"]
            if len(ranking_df) > 1 and ranking_df[-1, "score"] < 0
            else None
        )

        summary_cols = st.columns(2)
        if mvp:
            summary_cols[0].success(t("tmi_mvp_label", mvp=mvp))
        if boulet:
            summary_cols[1].error(t("tmi_boulet_label", boulet=boulet))


def _render_impact_from_events(
    events_df: pl.DataFrame,
    matches_df: pl.DataFrame,
    match_ids: list[str],
    friend_xuids: list[str],
    xuid: str,
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

    if not scores:
        st.info(t("tm_impact_no_events_players"))
        return

    gamertags = list(scores.keys())
    impact_match_set = set(
        list(first_bloods.keys())
        + list(clutch_finishers.keys())
        + list(last_casualties.keys())
        + list(last_group_kills.keys())
        + list(first_group_deaths.keys())
    )
    sorted_match_ids = [m for m in match_ids if m in impact_match_set]

    match_outcomes: dict[str, int] = {}
    if not matches_df.is_empty():
        for row in matches_df.iter_rows(named=True):
            match_outcomes[str(row["match_id"])] = int(row["outcome"])

    impact_matrix = build_impact_matrix(
        first_bloods,
        clutch_finishers,
        last_casualties,
        last_group_kills,
        first_group_deaths,
        match_ids=sorted_match_ids,
        gamertags=gamertags,
        match_outcomes=match_outcomes,
    )

    st.subheader(t("tm_impact_heatmap"))
    fig = plot_friends_impact_heatmap(
        impact_matrix,
        title=None,
        max_matches=len(sorted_match_ids),
    )
    st.plotly_chart(fig, width="stretch", config=PLOTLY_STATIC_CONFIG)

    st.subheader(t("tm_impact_ranking"))
    _render_ranking_table(scores, first_bloods, clutch_finishers, last_casualties)


def render_impact_taquinerie(
    db_path: str,
    xuid: str,
    match_ids: list[str],
    friend_xuids: list[str],
    db_key: tuple[int, int] | None = None,
) -> None:
    """Affiche l'onglet Impact (Sprint 12)."""
    st.subheader(t("tm_impact_header"))

    if len(friend_xuids) < 2:
        st.info(t("tm_impact_select_two"))
        return

    if not match_ids:
        st.warning(t("tm_impact_no_matches"))
        return

    st.caption(t("tm_impact_legend"))

    try:
        repo = DuckDBRepository(db_path, xuid.strip())
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
        _render_impact_from_events(events_df, matches_df, match_ids, friend_xuids, xuid)

    except Exception as e:
        st.warning(t("error_chart", error=e))
