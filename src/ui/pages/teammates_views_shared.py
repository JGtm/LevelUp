"""Helpers partagés extraits de teammates_views.py."""

from __future__ import annotations

import streamlit as st

from src.analysis import (
    compute_aggregated_stats,
    compute_global_ratio,
    compute_outcome_rates,
)
from src.ui.cache import (
    cached_query_matches_with_friend,
    cached_same_team_match_ids_with_friend,
)
from src.ui.i18n import t
from src.ui.medals import render_medals_grid
from src.visualization._compat import DataFrameLike, ensure_polars, to_pandas_for_st


def _render_match_details_expander(dfr: DataFrameLike) -> None:
    """Affiche l'expander avec détails des matchs joueur vs joueur."""
    dfr = ensure_polars(dfr)
    with st.expander(t("tm_match_details_title"), expanded=False):
        st.dataframe(
            to_pandas_for_st(
                dfr.select(
                    "start_time",
                    "playlist_name",
                    "pair_name",
                    "same_team",
                    "my_team_id",
                    "my_outcome",
                    "friend_team_id",
                    "friend_outcome",
                    "match_id",
                )
            ),
            width="stretch",
            hide_index=True,
        )


def _render_shared_stats_metrics(sub: DataFrameLike) -> None:
    """Affiche les métriques KPI pour les matchs partagés."""
    rates_sub = compute_outcome_rates(sub)
    total_out = max(1, rates_sub.total)
    win_rate_sub = rates_sub.wins / total_out
    loss_rate_sub = rates_sub.losses / total_out
    global_ratio_sub = compute_global_ratio(sub)

    k = st.columns(3)
    k[0].metric(t("tm_metric_matches"), f"{len(sub)}")
    k[1].metric(t("tm_win_loss"), f"{win_rate_sub * 100:.1f}% / {loss_rate_sub * 100:.1f}%")
    k[2].metric(
        t("tm_metric_global_ratio"),
        f"{global_ratio_sub:.2f}" if global_ratio_sub is not None else "-",
    )

    stats_sub = compute_aggregated_stats(sub)
    per_min = st.columns(3)
    per_min[0].metric(
        t("tm_metric_frags_min"),
        f"{stats_sub.kills_per_minute:.2f}" if stats_sub.kills_per_minute else "-",
    )
    per_min[1].metric(
        t("tm_metric_deaths_min"),
        f"{stats_sub.deaths_per_minute:.2f}" if stats_sub.deaths_per_minute else "-",
    )
    per_min[2].metric(
        t("tm_metric_assists_min"),
        f"{stats_sub.assists_per_minute:.2f}" if stats_sub.assists_per_minute else "-",
    )


def _render_shared_medals(  # noqa: PLR0913
    db_path: str,
    xuid: str,
    friend_xuid: str,
    me_name: str,
    friend_name: str,
    shared_ids: set[str],
    db_key: tuple[int, int] | None,
    top_medals_fn,
) -> None:
    """Affiche la section médailles partagées (single view)."""
    st.subheader(t("tm_medals"))
    shared_list = sorted({str(x) for x in shared_ids if str(x).strip()})
    if not shared_list:
        st.info(t("tm_no_shared_medals"))
        return

    with st.spinner(t("tm_computing_medals")):
        my_top = top_medals_fn(db_path, xuid.strip(), shared_list, top_n=12, db_key=db_key)
        fr_top = top_medals_fn(db_path, friend_xuid, shared_list, top_n=12, db_key=db_key)

    m1, m2 = st.columns(2)
    with m1:
        st.caption(f"{me_name}")
        render_medals_grid(
            [{"name_id": int(n), "count": int(c)} for n, c in (my_top or [])],
            cols_per_row=6,
        )
    with m2:
        st.caption(f"{friend_name}")
        render_medals_grid(
            [{"name_id": int(n), "count": int(c)} for n, c in (fr_top or [])],
            cols_per_row=6,
        )


def _collect_friend_match_ids(
    db_path: str,
    xuid: str,
    picked_xuids: list[str],
    same_team_only: bool,
    db_key: tuple[int, int] | None,
) -> tuple[set[str], dict[str, set[str]]]:
    """Collecte les match_ids par coéquipier et l'union totale."""
    all_match_ids: set[str] = set()
    per_friend_ids: dict[str, set[str]] = {}
    for fx in picked_xuids:
        ids: set[str] = set()
        if bool(same_team_only):
            ids = {
                str(x)
                for x in cached_same_team_match_ids_with_friend(
                    db_path, xuid.strip(), fx, db_key=db_key
                )
            }
        else:
            rows = cached_query_matches_with_friend(db_path, xuid.strip(), fx, db_key=db_key)
            # DuckDB v4 : rows est une list[str] de match_ids ; chemin legacy : objets avec .match_id
            ids = {str(r) if isinstance(r, str) else str(r.match_id) for r in rows}
        per_friend_ids[str(fx)] = ids
        all_match_ids.update(ids)
    return all_match_ids, per_friend_ids
