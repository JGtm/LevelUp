"""Vue escouade (moi + 2 ou 3 coéquipiers) pour la page Escouade/Squad.

Extraite de teammates_views.py — regroupe render_trio_view et ses helpers privés.
"""

from __future__ import annotations

import logging

import polars as pl
import streamlit as st

logger = logging.getLogger(__name__)

from src.analysis import compute_aggregated_stats
from src.config import OKABE_ITO_PALETTE
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


def render_trio_view(  # noqa: PLR0913, PLR0915
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
    """Affiche la vue escouade (moi + 2 ou 3 coéquipiers). Retourne True si les graphes du bas sont rendus."""
    df = ensure_polars(df)
    dff = ensure_polars(dff)
    base = ensure_polars(base)
    f1_xuid = picked_xuids[0]
    f2_xuid = picked_xuids[1]
    f3_xuid = picked_xuids[2] if len(picked_xuids) >= 3 else None

    f1_name = display_name_from_xuid(f1_xuid, db_path=db_path)
    f2_name = display_name_from_xuid(f2_xuid, db_path=db_path)
    f3_name = display_name_from_xuid(f3_xuid, db_path=db_path) if f3_xuid else None

    squad_size = 3 + (1 if f3_xuid else 0)
    logger.info(
        "render_trio_view: escouade %d joueurs — %s",
        squad_size,
        [me_name, f1_name, f2_name] + ([f3_name] if f3_name else []),
    )

    friend_names_all = [f1_name, f2_name] + ([f3_name] if f3_name else [])
    st.subheader(t("tm_squad_header", names=" + ".join(friend_names_all)))

    ids_m = set(
        cached_same_team_match_ids_with_friend(db_path, xuid.strip(), f1_xuid, db_key=db_key)
    )
    ids_c = set(
        cached_same_team_match_ids_with_friend(db_path, xuid.strip(), f2_xuid, db_key=db_key)
    )
    squad_ids = ids_m & ids_c
    if f3_xuid:
        ids_d = set(
            cached_same_team_match_ids_with_friend(db_path, xuid.strip(), f3_xuid, db_key=db_key)
        )
        squad_ids = squad_ids & ids_d

    base_for_trio = dff if apply_current_filters else df
    squad_ids = squad_ids & set(base_for_trio["match_id"].cast(pl.Utf8).to_list())

    logger.info("render_trio_view: %d matchs d'escouade trouvés", len(squad_ids))
    if not squad_ids:
        st.warning(t("tm_no_trio_matches"))
        return False

    trio_ids_set = {str(x) for x in squad_ids}

    _detect_trio_session(db_path, xuid, db_key, include_firefight, aliases_key, trio_ids_set)

    me_df = base_for_trio.filter(pl.col("match_id").is_in(list(squad_ids)))

    # Charger les stats des coéquipiers depuis leurs propres DBs
    f1_df = ensure_polars(load_teammate_stats_fn(f1_name, trio_ids_set, db_path))
    f2_df = ensure_polars(load_teammate_stats_fn(f2_name, trio_ids_set, db_path))
    f3_df: pl.DataFrame | None = (
        ensure_polars(load_teammate_stats_fn(f3_name, trio_ids_set, db_path)) if f3_name else None
    )

    # Filtrer les DataFrames pour ne garder que les match_ids présents dans me_df
    filtered_match_ids = me_df["match_id"].cast(pl.Utf8).to_list() if not me_df.is_empty() else []
    if not f1_df.is_empty() and "match_id" in f1_df.columns and filtered_match_ids:
        f1_df = f1_df.filter(pl.col("match_id").cast(pl.Utf8).is_in(filtered_match_ids))
    if not f2_df.is_empty() and "match_id" in f2_df.columns and filtered_match_ids:
        f2_df = f2_df.filter(pl.col("match_id").cast(pl.Utf8).is_in(filtered_match_ids))
    if (
        f3_df is not None
        and not f3_df.is_empty()
        and "match_id" in f3_df.columns
        and filtered_match_ids
    ):
        f3_df = f3_df.filter(pl.col("match_id").cast(pl.Utf8).is_in(filtered_match_ids))

    me_df = me_df.sort("start_time")

    _render_per_minute_stats(
        me_df,
        f1_df,
        f2_df,
        me_name,
        f1_name,
        f2_name,
        colors_by_name,
        f3_df=f3_df,
        f3_name=f3_name,
    )

    # Radar de complémentarité escouade
    render_trio_synergy_radar(
        me_df=me_df,
        f1_df=f1_df,
        f2_df=f2_df,
        me_name=me_name,
        f1_name=f1_name,
        f2_name=f2_name,
        colors_by_name=colors_by_name,
        db_path=db_path,
        f3_df=f3_df,
        f3_name=f3_name,
    )

    # Vérification alignment : besoin d'au moins me + f1 + f2
    if me_df.is_empty() or f1_df.is_empty() or f2_df.is_empty():
        st.warning(t("tm_trio_warning"))
        return False

    # Si f3 était attendu mais vide, on le retire silencieusement
    if f3_xuid and (f3_df is None or f3_df.is_empty()):
        logger.warning(
            "render_trio_view: f3 (%s) vide après chargement — retiré de l'escouade", f3_name
        )
        f3_df = None
        f3_xuid = None
        f3_name = None

    _render_trio_performance_charts(
        me_df,
        f1_df,
        f2_df,
        me_name,
        f1_name,
        f2_name,
        f1_xuid,
        f2_xuid,
        f3_df=f3_df,
        f3_name=f3_name,
        f3_xuid=f3_xuid,
        colors_by_name=colors_by_name,
    )

    # Graphes de barres - reconstruire series avec les DataFrames de l'escouade
    series = [(me_name, me_df)]
    if not f1_df.is_empty():
        series.append((f1_name, f1_df))
    if not f2_df.is_empty():
        series.append((f2_name, f2_df))
    if f3_name and f3_df is not None and not f3_df.is_empty():
        series.append((f3_name, f3_df))
    colors_by_name = assign_player_colors_fn([n for n, _ in series])
    series = enrich_series_fn(series, db_path)
    render_metric_bar_charts(
        series=series,
        colors_by_name=colors_by_name,
        show_smooth=show_smooth,
        key_suffix=f"{len(series)}",
        plot_fn=plot_multi_metric_bars_fn,
    )

    # Médailles escouade
    squad_match_ids = me_df["match_id"].cast(pl.Utf8).to_list() if not me_df.is_empty() else []
    _render_trio_medals(
        squad_match_ids,
        db_path,
        xuid.strip(),
        f1_xuid,
        f2_xuid,
        me_name,
        f1_name,
        f2_name,
        db_key,
        top_medals_fn,
        f3_xuid=f3_xuid,
        f3_name=f3_name,
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
    *,
    f3_df: DataFrameLike | None = None,
    f3_name: str | None = None,
) -> None:
    """Affiche le graphe barres groupées stats/min pour l'escouade (3 ou 4 joueurs)."""
    import plotly.graph_objects as go

    from src.visualization.theme import apply_halo_plot_style

    _pm_metrics = [t("tm_metric_frags_min"), t("tm_metric_deaths_min"), t("tm_metric_assists_min")]
    _pm_players = [
        (
            me_name,
            compute_aggregated_stats(me_df),
            colors_by_name.get(me_name, OKABE_ITO_PALETTE[0]),
        ),
        (
            f1_name,
            compute_aggregated_stats(f1_df),
            colors_by_name.get(f1_name, OKABE_ITO_PALETTE[1]),
        ),
        (
            f2_name,
            compute_aggregated_stats(f2_df),
            colors_by_name.get(f2_name, OKABE_ITO_PALETTE[2]),
        ),
    ]
    if f3_name and f3_df is not None:
        _pm_players.append(
            (
                f3_name,
                compute_aggregated_stats(f3_df),
                colors_by_name.get(f3_name, OKABE_ITO_PALETTE[3]),
            )
        )
    st.subheader(t("tm_per_minute"))
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
    me_df: DataFrameLike,
    f1_df: DataFrameLike,
    f2_df: DataFrameLike,
    me_name: str,
    f1_name: str,
    f2_name: str,
    f1_xuid: str,
    f2_xuid: str,
    *,
    f3_df: DataFrameLike | None = None,
    f3_name: str | None = None,
    f3_xuid: str | None = None,
    colors_by_name: dict[str, str] | None = None,
) -> None:
    """Calcule les scores de performance et affiche les graphes d'escouade.

    Utilise _merge_trio_dataframes (inner join sur match_id) pour garantir
    que les start_times sont identiques entre tous les joueurs — évite les
    incompatibilités de type Datetime qui rendraient plot_trio_metric vide.
    """
    from src.analysis.performance_score import compute_performance_series

    merged = _merge_trio_dataframes(me_df, f1_df, f2_df)
    if merged.is_empty():
        logger.warning("_render_trio_performance_charts: merged vide, aucun match aligné")
        return

    _stat_cols = [
        "start_time",
        "kills",
        "deaths",
        "assists",
        "ratio",
        "accuracy",
        "average_life_seconds",
        "time_played_seconds",
    ]
    _f_rename = {
        "kills": "kills",
        "deaths": "deaths",
        "assists": "assists",
        "ratio": "ratio",
        "accuracy": "accuracy",
        "average_life_seconds": "average_life_seconds",
    }

    def _extract_player(prefix: str | None) -> pl.DataFrame:
        """Extrait les colonnes d'un joueur (me = pas de prefix, f1/f2 = prefix)."""
        if prefix is None:
            cols = {c: c for c in _stat_cols if c in merged.columns}
        else:
            cols = {"start_time": "start_time", "time_played_seconds": "time_played_seconds"}
            for k, v in _f_rename.items():
                src = f"{prefix}_{k}"
                if src in merged.columns:
                    cols[src] = v
        out = merged.select(list(cols.keys())).rename(cols)
        return out

    d_self = _extract_player(None)
    d_f1 = _extract_player("f1")
    d_f2 = _extract_player("f2")

    d_self = d_self.with_columns(
        pl.Series("performance", compute_performance_series(d_self, d_self))
    )
    d_f1 = d_f1.with_columns(pl.Series("performance", compute_performance_series(d_f1, d_f1)))
    d_f2 = d_f2.with_columns(pl.Series("performance", compute_performance_series(d_f2, d_f2)))

    # Pour f3 : aligner sur les match_id du merged via start_time partagée
    d_f3_p: pl.DataFrame | None = None
    if f3_df is not None:
        f3_polars = ensure_polars(f3_df)
        if not f3_polars.is_empty() and "match_id" in f3_polars.columns:
            merged_ids = set(merged.select("match_id").to_series().cast(pl.Utf8).to_list())
            f3_filtered = f3_polars.filter(pl.col("match_id").cast(pl.Utf8).is_in(merged_ids))
            if not f3_filtered.is_empty():
                # Injecter la start_time depuis merged (même référence que les autres joueurs)
                time_map = merged.select(["match_id", "start_time"])
                f3_cols = [c for c in _stat_cols if c in f3_filtered.columns and c != "start_time"]
                f3_sel = f3_filtered.select(["match_id"] + f3_cols)
                f3_aligned = time_map.join(f3_sel, on="match_id", how="inner").drop("match_id")
                d_f3_p = f3_aligned.with_columns(
                    pl.Series("performance", compute_performance_series(f3_aligned, f3_aligned))
                )

    render_trio_charts(
        d_self,
        d_f1,
        d_f2,
        me_name,
        f1_name,
        f2_name,
        f1_xuid,
        f2_xuid,
        d_f3=d_f3_p,
        f3_name=f3_name,
        f3_xuid=f3_xuid,
        colors_by_name=colors_by_name,
    )


def _render_trio_medals(  # noqa: PLR0913
    match_ids: list[str],
    db_path: str,
    xuid: str,
    f1_xuid: str,
    f2_xuid: str,
    me_name: str,
    f1_name: str,
    f2_name: str,
    db_key: tuple[int, int] | None,
    top_medals_fn,
    *,
    f3_xuid: str | None = None,
    f3_name: str | None = None,
) -> None:
    """Affiche la section médailles de l'escouade (3 ou 4 joueurs)."""
    st.subheader(t("tm_medals_all"))
    if not match_ids:
        st.info(t("tm_no_medals_aggregate"))
        return

    with st.spinner(t("tm_computing_medals_all")):
        top_self = top_medals_fn(db_path, xuid.strip(), match_ids, top_n=12, db_key=db_key)
        top_f1 = top_medals_fn(db_path, f1_xuid, match_ids, top_n=12, db_key=db_key)
        top_f2 = top_medals_fn(db_path, f2_xuid, match_ids, top_n=12, db_key=db_key)
        top_f3 = (
            top_medals_fn(db_path, f3_xuid, match_ids, top_n=12, db_key=db_key) if f3_xuid else None
        )

    players_medals = [
        (me_name, top_self),
        (f1_name, top_f1),
        (f2_name, top_f2),
    ]
    if f3_xuid and f3_name and top_f3 is not None:
        players_medals.append((f3_name, top_f3))

    cols = st.columns(len(players_medals))
    for col, (name, top) in zip(cols, players_medals, strict=False):
        with col, st.expander(name, expanded=True):
            render_medals_grid(
                [{"name_id": int(n), "count": int(c)} for n, c in (top or [])],
                cols_per_row=6,
            )
