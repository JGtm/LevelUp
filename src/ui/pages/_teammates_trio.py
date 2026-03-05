"""Vue trio (moi + 2 coéquipiers) pour la page Coéquipiers.

Extraite de teammates_views.py — regroupe render_trio_view et ses helpers privés.
"""

from __future__ import annotations

import polars as pl
import streamlit as st

from src.analysis import compute_aggregated_stats
from src.ui import display_name_from_xuid
from src.ui.cache import (
    cached_compute_sessions_db,
    cached_same_team_match_ids_with_friend,
)
from src.ui.chart_utils import safe_chart_render
from src.ui.i18n import t
from src.ui.medals import render_medals_grid
from src.ui.pages.teammates_charts import render_metric_bar_charts, render_trio_charts
from src.ui.pages.teammates_synergy import render_trio_synergy_radar
from src.ui.streamlit_modern import PLOTLY_STATIC_CONFIG
from src.visualization._compat import DataFrameLike, ensure_polars

# ---------------------------------------------------------------------------
# Vue trio publique
# ---------------------------------------------------------------------------


def render_trio_view(  # noqa: PLR0913
    df: DataFrameLike,
    dff: DataFrameLike,
    base: DataFrameLike,
    me_name: str,
    xuid: str,
    db_path: str,
    db_key: tuple[int, int] | None,
    aliases_key: int | None,
    picked_xuids: list[str],
    apply_current_filters: bool,
    include_firefight: bool,
    series: list[tuple[str, DataFrameLike]],
    colors_by_name: dict[str, str],
    show_smooth: bool,
    assign_player_colors_fn,
    plot_multi_metric_bars_fn,
    top_medals_fn,
    load_teammate_stats_fn,
    enrich_series_fn,
) -> bool:
    """Affiche la vue trio (moi + 2 coéquipiers). Retourne True si les graphes du bas sont rendus."""
    df = ensure_polars(df)
    dff = ensure_polars(dff)
    base = ensure_polars(base)
    f1_xuid, f2_xuid = picked_xuids[0], picked_xuids[1]
    f1_name = display_name_from_xuid(f1_xuid, db_path=db_path)
    f2_name = display_name_from_xuid(f2_xuid, db_path=db_path)
    st.subheader(t("tm_trio_header", f1=f1_name, f2=f2_name))

    ids_m = set(
        cached_same_team_match_ids_with_friend(db_path, xuid.strip(), f1_xuid, db_key=db_key)
    )
    ids_c = set(
        cached_same_team_match_ids_with_friend(db_path, xuid.strip(), f2_xuid, db_key=db_key)
    )
    trio_ids = ids_m & ids_c

    base_for_trio = dff if apply_current_filters else df
    trio_ids = trio_ids & set(base_for_trio["match_id"].cast(pl.Utf8).to_list())

    if not trio_ids:
        st.warning(t("tm_no_trio_matches"))
        return False

    trio_ids_set = {str(x) for x in trio_ids}

    _detect_trio_session(db_path, xuid, db_key, include_firefight, aliases_key, trio_ids_set)

    me_df = base_for_trio.filter(pl.col("match_id").is_in(list(trio_ids)))

    # Charger les stats des coéquipiers depuis LEURS propres DBs
    f1_df = ensure_polars(load_teammate_stats_fn(f1_name, trio_ids_set, db_path))
    f2_df = ensure_polars(load_teammate_stats_fn(f2_name, trio_ids_set, db_path))

    # Filtrer les DataFrames des coéquipiers pour ne garder que les match_ids présents dans me_df
    filtered_match_ids = me_df["match_id"].cast(pl.Utf8).to_list() if not me_df.is_empty() else []
    if not f1_df.is_empty() and "match_id" in f1_df.columns and filtered_match_ids:
        f1_df = f1_df.filter(pl.col("match_id").cast(pl.Utf8).is_in(filtered_match_ids))
    if not f2_df.is_empty() and "match_id" in f2_df.columns and filtered_match_ids:
        f2_df = f2_df.filter(pl.col("match_id").cast(pl.Utf8).is_in(filtered_match_ids))

    me_df = me_df.sort("start_time")

    _render_per_minute_stats(me_df, f1_df, f2_df, me_name, f1_name, f2_name, colors_by_name)

    # Radar de complémentarité trio
    render_trio_synergy_radar(
        me_df=me_df,
        f1_df=f1_df,
        f2_df=f2_df,
        me_name=me_name,
        f1_name=f1_name,
        f2_name=f2_name,
        colors_by_name=colors_by_name,
        db_path=db_path,
    )

    # Merge et performance
    merged = _merge_trio_dataframes(me_df, f1_df, f2_df)
    if merged.is_empty():
        st.warning(t("tm_trio_warning"))
        return False

    _render_trio_performance_charts(merged, me_name, f1_name, f2_name, f1_xuid, f2_xuid)

    # Graphes de barres - reconstruire series avec les DataFrames du trio
    series = [(me_name, me_df)]
    if not f1_df.is_empty():
        series.append((f1_name, f1_df))
    if not f2_df.is_empty():
        series.append((f2_name, f2_df))
    colors_by_name = assign_player_colors_fn([n for n, _ in series])
    series = enrich_series_fn(series, db_path)
    render_metric_bar_charts(
        series=series,
        colors_by_name=colors_by_name,
        show_smooth=show_smooth,
        key_suffix=f"{len(series)}",
        plot_fn=plot_multi_metric_bars_fn,
    )

    # Médailles trio
    _render_trio_medals(
        merged,
        db_path,
        xuid,
        f1_xuid,
        f2_xuid,
        me_name,
        f1_name,
        f2_name,
        db_key,
        top_medals_fn,
    )

    return True


# ---------------------------------------------------------------------------
# Helpers privés du trio
# ---------------------------------------------------------------------------


def _detect_trio_session(  # noqa: PLR0913
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    include_firefight: bool,
    aliases_key: int | None,
    trio_ids_set: set[str],
) -> None:
    """Détecte et affiche la dernière session trio."""
    from src.app.filters import get_friends_xuids_for_sessions

    friends_tuple = get_friends_xuids_for_sessions(db_path, xuid.strip(), db_key, aliases_key)
    base_s_trio = ensure_polars(
        cached_compute_sessions_db(
            db_path,
            xuid.strip(),
            db_key,
            include_firefight,
            120,  # gap figé
            friends_xuids=friends_tuple,
        )
    )
    trio_rows = base_s_trio.filter(pl.col("match_id").cast(pl.Utf8).is_in(list(trio_ids_set)))
    latest_label = None
    if not trio_rows.is_empty():
        # session_id peut être string ou int selon le contexte, comparer en string
        latest_sid = trio_rows["session_id"].max()
        latest_labels = (
            trio_rows.filter(pl.col("session_id").cast(pl.Utf8) == str(latest_sid))
            .select("session_label")
            .drop_nulls()
            .unique()
            .to_series()
            .to_list()
        )
        latest_label = latest_labels[0] if latest_labels else None

    st.session_state["_trio_latest_session_label"] = latest_label
    if latest_label:
        st.caption(t("tm_trio_session", label=latest_label))
    else:
        st.caption(t("tm_trio_session_unknown"))


def _render_per_minute_stats(  # noqa: PLR0913
    me_df: DataFrameLike,
    f1_df: DataFrameLike,
    f2_df: DataFrameLike,
    me_name: str,
    f1_name: str,
    f2_name: str,
    colors_by_name: dict[str, str],
) -> None:
    """Affiche le graphe barres groupées stats/min pour le trio."""
    import plotly.graph_objects as go

    from src.visualization.theme import apply_halo_plot_style

    me_stats = compute_aggregated_stats(me_df)
    f1_stats = compute_aggregated_stats(f1_df)
    f2_stats = compute_aggregated_stats(f2_df)
    st.subheader(t("tm_per_minute"))

    _pm_metrics = [t("tm_metric_frags_min"), t("tm_metric_deaths_min"), t("tm_metric_assists_min")]
    _pm_players = [
        (me_name, me_stats, colors_by_name.get(me_name, "#636EFA")),
        (f1_name, f1_stats, colors_by_name.get(f1_name, "#EF553B")),
        (f2_name, f2_stats, colors_by_name.get(f2_name, "#00CC96")),
    ]
    fig_pm = go.Figure()
    for _pm_name, _pm_st, _pm_color in _pm_players:
        _pm_vals = [
            round(float(_pm_st.kills_per_minute), 2) if _pm_st.kills_per_minute else 0,
            round(float(_pm_st.deaths_per_minute), 2) if _pm_st.deaths_per_minute else 0,
            round(float(_pm_st.assists_per_minute), 2) if _pm_st.assists_per_minute else 0,
        ]
        fig_pm.add_trace(
            go.Bar(
                name=_pm_name,
                x=_pm_metrics,
                y=_pm_vals,
                marker_color=_pm_color,
                text=[f"{v:.2f}" for v in _pm_vals],
                textposition="auto",
            )
        )
    fig_pm.update_layout(
        barmode="group",
        height=350,
        margin={"l": 40, "r": 20, "t": 30, "b": 40},
        legend={"orientation": "h", "yanchor": "bottom", "y": 1.02, "x": 0.5, "xanchor": "center"},
    )
    fig_pm = apply_halo_plot_style(fig_pm, title=None, height=None)
    with safe_chart_render():
        if fig_pm is not None:
            st.plotly_chart(fig_pm, width="stretch", config=PLOTLY_STATIC_CONFIG)
        else:
            st.info(t("insufficient_data_chart"))


def _merge_trio_dataframes(
    me_df: DataFrameLike,
    f1_df: DataFrameLike,
    f2_df: DataFrameLike,
) -> pl.DataFrame:
    """Merge les DataFrames des 3 joueurs sur match_id."""
    me_df = ensure_polars(me_df)
    f1_df = ensure_polars(f1_df)
    f2_df = ensure_polars(f2_df)
    friend_cols = [
        "match_id",
        "kills",
        "deaths",
        "assists",
        "accuracy",
        "ratio",
        "average_life_seconds",
    ]
    me_cols = [
        "match_id",
        "start_time",
        "kills",
        "deaths",
        "assists",
        "accuracy",
        "ratio",
        "average_life_seconds",
        "time_played_seconds",
    ]
    # Vérifier que tous les DataFrames ont les colonnes requises
    missing_me = [c for c in me_cols if c not in me_df.columns]
    missing_f1 = [c for c in friend_cols if c not in f1_df.columns]
    missing_f2 = [c for c in friend_cols if c not in f2_df.columns]
    if (
        missing_me
        or missing_f1
        or missing_f2
        or me_df.is_empty()
        or f1_df.is_empty()
        or f2_df.is_empty()
    ):
        return pl.DataFrame()
    f1_sel = f1_df.select(friend_cols).rename({c: f"f1_{c}" for c in friend_cols})
    f2_sel = f2_df.select(friend_cols).rename({c: f"f2_{c}" for c in friend_cols})
    return (
        me_df.select(me_cols)
        .join(f1_sel, left_on="match_id", right_on="f1_match_id", how="inner")
        .join(f2_sel, left_on="match_id", right_on="f2_match_id", how="inner")
    )


def _render_trio_performance_charts(  # noqa: PLR0913
    merged: DataFrameLike,
    me_name: str,
    f1_name: str,
    f2_name: str,
    f1_xuid: str,
    f2_xuid: str,
) -> None:
    """Calcule les scores de performance et affiche les graphes trio."""
    from src.analysis.performance_score import compute_performance_series

    merged = ensure_polars(merged).sort("start_time")
    d_self = merged.select(
        "start_time",
        "kills",
        "deaths",
        "assists",
        "ratio",
        "accuracy",
        "average_life_seconds",
        "time_played_seconds",
    )
    d_f1 = merged.select(
        "start_time",
        "f1_kills",
        "f1_deaths",
        "f1_assists",
        "f1_ratio",
        "f1_accuracy",
        "f1_average_life_seconds",
        "time_played_seconds",
    ).rename(
        {
            "f1_kills": "kills",
            "f1_deaths": "deaths",
            "f1_assists": "assists",
            "f1_ratio": "ratio",
            "f1_accuracy": "accuracy",
            "f1_average_life_seconds": "average_life_seconds",
        }
    )
    d_f2 = merged.select(
        "start_time",
        "f2_kills",
        "f2_deaths",
        "f2_assists",
        "f2_ratio",
        "f2_accuracy",
        "f2_average_life_seconds",
        "time_played_seconds",
    ).rename(
        {
            "f2_kills": "kills",
            "f2_deaths": "deaths",
            "f2_assists": "assists",
            "f2_ratio": "ratio",
            "f2_accuracy": "accuracy",
            "f2_average_life_seconds": "average_life_seconds",
        }
    )

    d_self = d_self.with_columns(
        pl.Series("performance", compute_performance_series(d_self, d_self))
    )
    d_f1 = d_f1.with_columns(pl.Series("performance", compute_performance_series(d_f1, d_f1)))
    d_f2 = d_f2.with_columns(pl.Series("performance", compute_performance_series(d_f2, d_f2)))

    render_trio_charts(d_self, d_f1, d_f2, me_name, f1_name, f2_name, f1_xuid, f2_xuid)


def _render_trio_medals(  # noqa: PLR0913
    merged: DataFrameLike,
    db_path: str,
    xuid: str,
    f1_xuid: str,
    f2_xuid: str,
    me_name: str,
    f1_name: str,
    f2_name: str,
    db_key: tuple[int, int] | None,
    top_medals_fn,
) -> None:
    """Affiche la section médailles du trio."""
    st.subheader(t("tm_medals_all"))
    trio_match_ids = (
        ensure_polars(merged).select("match_id").drop_nulls().to_series().cast(pl.Utf8).to_list()
    )
    if not trio_match_ids:
        st.info(t("tm_no_medals_aggregate"))
        return

    with st.spinner(t("tm_computing_medals_all")):
        top_self = top_medals_fn(db_path, xuid.strip(), trio_match_ids, top_n=12, db_key=db_key)
        top_f1 = top_medals_fn(db_path, f1_xuid, trio_match_ids, top_n=12, db_key=db_key)
        top_f2 = top_medals_fn(db_path, f2_xuid, trio_match_ids, top_n=12, db_key=db_key)

    c1, c2, c3 = st.columns(3)
    with c1, st.expander(f"{me_name}", expanded=True):
        render_medals_grid(
            [{"name_id": int(n), "count": int(c)} for n, c in (top_self or [])],
            cols_per_row=6,
        )
    with c2, st.expander(f"{f1_name}", expanded=True):
        render_medals_grid(
            [{"name_id": int(n), "count": int(c)} for n, c in (top_f1 or [])],
            cols_per_row=6,
        )
    with c3, st.expander(f"{f2_name}", expanded=True):
        render_medals_grid(
            [{"name_id": int(n), "count": int(c)} for n, c in (top_f2 or [])],
            cols_per_row=6,
        )
