"""Helpers privés pour la vue escouade (moi + 2 ou 3 coéquipiers).

Extraits de _teammates_trio.py pour garder chaque module sous 500L.
"""

from __future__ import annotations

import logging

import plotly.graph_objects as go
import polars as pl
import streamlit as st

logger = logging.getLogger(__name__)

from src.analysis import compute_aggregated_stats
from src.config import OKABE_ITO_PALETTE
from src.ui.cache import cached_compute_sessions_db
from src.ui.chart_utils import safe_chart_render
from src.ui.i18n import t
from src.ui.medals import render_medals_grid
from src.ui.pages.teammates_charts import render_trio_charts
from src.ui.streamlit_modern import PLOTLY_STATIC_CONFIG
from src.visualization._chart_series import ChartData, MatchSeries
from src.visualization._compat import DataFrameLike, ensure_polars
from src.visualization.theme import apply_halo_plot_style, get_legend_horizontal_bottom
from src.visualization.trio import _negative_color

# ---------------------------------------------------------------------------
# Session trio
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
        try:
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
        except Exception:
            logger.debug(
                "_detect_trio_session: session_id conversion échouée (type=%s)",
                trio_rows["session_id"].dtype,
                exc_info=True,
            )
            latest_label = None

    st.session_state["_trio_latest_session_label"] = latest_label
    if latest_label:
        st.caption(t("tm_trio_session", label=latest_label))
    else:
        st.caption(t("tm_trio_session_unknown"))


# ---------------------------------------------------------------------------
# Stats par minute
# ---------------------------------------------------------------------------



def _render_per_minute_stats(  # noqa: PLR0913
    me_df: DataFrameLike,
    f1_df: DataFrameLike,
    f2_df: DataFrameLike | None,
    me_name: str,
    f1_name: str,
    f2_name: str | None,
    colors_by_name: dict[str, str],
    *,
    f3_df: DataFrameLike | None = None,
    f3_name: str | None = None,
    pm_records: dict[str, tuple[float | None, float | None, float | None]] | None = None,
) -> None:
    """Affiche le graphe barres groupées stats/min pour l'escouade (2, 3 ou 4 joueurs)."""
    _pm_metrics = [t("tm_metric_frags_min"), t("tm_metric_deaths_min"), t("tm_metric_assists_min")]
    _pm_raw: list[tuple[str, pl.DataFrame]] = [
        (me_name, ensure_polars(me_df)),
        (f1_name, ensure_polars(f1_df)),
    ]
    if f2_name and f2_df is not None:
        _pm_raw.append((f2_name, ensure_polars(f2_df)))
    if f3_name and f3_df is not None:
        _pm_raw.append((f3_name, ensure_polars(f3_df)))

    _pm_players = [
        (name, compute_aggregated_stats(df), colors_by_name.get(name, OKABE_ITO_PALETTE[i]))
        for i, (name, df) in enumerate(_pm_raw)
    ]
    st.subheader(t("tm_per_minute"))
    fig_pm = go.Figure()
    for _pm_name, _pm_st, _pm_color in _pm_players:
        _kpm = round(float(_pm_st.kills_per_minute), 2) if _pm_st.kills_per_minute else 0
        _dpm = round(float(_pm_st.deaths_per_minute), 2) if _pm_st.deaths_per_minute else 0
        _apm = round(float(_pm_st.assists_per_minute), 2) if _pm_st.assists_per_minute else 0
        _pm_y = [_kpm, -_dpm, _apm]  # morts sous l'axe X
        _pm_text = [f"{_kpm:.2f}", f"{_dpm:.2f}", f"{_apm:.2f}"]  # labels absolus
        # Frags/Assists → couleur normale ; Morts → teinte négative Okabe-Ito du joueur
        _bar_colors = [_pm_color, _negative_color(_pm_color), _pm_color]
        fig_pm.add_trace(
            go.Bar(
                name=_pm_name,
                x=_pm_metrics,
                y=_pm_y,
                marker_color=_bar_colors,
                text=_pm_text,
                textposition="auto",
                offsetgroup=_pm_name,
            )
        )
    # ── Records par-minute (traces fantômes hachurées, axe catégoriel) ───────
    if pm_records:
        _player_names_pm = [n for n, _, _ in _pm_players]
        ChartData(
            series=[
                MatchSeries(
                    name=n,
                    x=[],
                    y=[],
                    color=colors_by_name.get(n, OKABE_ITO_PALETTE[i]),
                    map_names=[],
                )
                for i, (n, _, _) in enumerate(_pm_players)
            ],
            x_labels=_pm_metrics,
            barmode="categorical",
            global_records={
                p_name: rec
                for p_name, rec in pm_records.items()
                if p_name in _player_names_pm
            },
        ).add_record_overlays(fig_pm)
    fig_pm.update_layout(
        barmode="group",
        height=350,
        margin={"l": 40, "r": 20, "t": 30, "b": 80},
        legend=get_legend_horizontal_bottom(),
    )
    fig_pm = apply_halo_plot_style(fig_pm, title=None, height=None)
    # Forcer l'axe zéro en gras blanc (apply_halo_plot_style le désactive via theme.py)
    fig_pm.update_yaxes(zeroline=True, zerolinecolor="rgba(255,255,255,0.75)", zerolinewidth=2)
    with safe_chart_render():
        if fig_pm is not None:
            st.plotly_chart(fig_pm, width="stretch", config=PLOTLY_STATIC_CONFIG)
        else:
            st.info(t("insufficient_data_chart"))


# ---------------------------------------------------------------------------
# Merge DataFrames
# ---------------------------------------------------------------------------


def _merge_trio_dataframes(
    me_df: DataFrameLike,
    f1_df: DataFrameLike,
    f2_df: DataFrameLike | None,
) -> pl.DataFrame:
    """Merge les DataFrames des joueurs sur match_id (f2 optionnel)."""
    me_df = ensure_polars(me_df)
    f1_df = ensure_polars(f1_df)
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
        "map_name",
        "kills",
        "deaths",
        "assists",
        "accuracy",
        "ratio",
        "average_life_seconds",
        "time_played_seconds",
    ]
    _me_required = [c for c in me_cols if c != "map_name"]
    missing_me = [c for c in _me_required if c not in me_df.columns]
    missing_f1 = [c for c in friend_cols if c not in f1_df.columns]
    if missing_me or missing_f1 or me_df.is_empty() or f1_df.is_empty():
        return pl.DataFrame()
    _opt = ["performance_score"]
    me_opt = [c for c in _opt if c in me_df.columns]
    me_cols_actual = [c for c in me_cols if c in me_df.columns]
    f1_ext = friend_cols + [c for c in _opt if c in f1_df.columns]
    f1_sel = f1_df.select(f1_ext).rename({c: f"f1_{c}" for c in f1_ext})
    merged = me_df.select(me_cols_actual + me_opt).join(
        f1_sel, left_on="match_id", right_on="f1_match_id", how="inner"
    )
    if f2_df is not None:
        f2_polars = ensure_polars(f2_df)
        missing_f2 = [c for c in friend_cols if c not in f2_polars.columns]
        if not missing_f2 and not f2_polars.is_empty():
            f2_ext = friend_cols + [c for c in _opt if c in f2_polars.columns]
            f2_sel = f2_polars.select(f2_ext).rename({c: f"f2_{c}" for c in f2_ext})
            merged = merged.join(f2_sel, left_on="match_id", right_on="f2_match_id", how="inner")
    return merged


# ---------------------------------------------------------------------------
# Graphes de performance trio
# ---------------------------------------------------------------------------

_STAT_COLS = [
    "start_time",
    "map_name",
    "kills",
    "deaths",
    "assists",
    "ratio",
    "accuracy",
    "average_life_seconds",
    "time_played_seconds",
    "performance_score",
]

_F_RENAME = {
    "kills": "kills",
    "deaths": "deaths",
    "assists": "assists",
    "ratio": "ratio",
    "accuracy": "accuracy",
    "average_life_seconds": "average_life_seconds",
    "performance_score": "performance_score",
}


def _extract_player_df(merged: pl.DataFrame, prefix: str | None) -> pl.DataFrame:
    """Extrait les colonnes d'un joueur depuis le merged (me = None, f1/f2 = prefix)."""
    if prefix is None:
        cols = {c: c for c in _STAT_COLS if c in merged.columns}
    else:
        cols = {"start_time": "start_time", "time_played_seconds": "time_played_seconds"}
        for k, v in _F_RENAME.items():
            src = f"{prefix}_{k}"
            if src in merged.columns:
                cols[src] = v
    return merged.select(list(cols.keys())).rename(cols)


def _align_f3_to_merged(f3_df: DataFrameLike | None, merged: pl.DataFrame) -> pl.DataFrame | None:
    """Aligne le DataFrame f3 sur les match_ids du merged, injecte start_time."""
    if f3_df is None:
        return None
    f3_polars = ensure_polars(f3_df)
    if f3_polars.is_empty() or "match_id" not in f3_polars.columns:
        return None
    merged_ids = set(merged.select("match_id").to_series().cast(pl.Utf8).to_list())
    f3_filtered = f3_polars.filter(pl.col("match_id").cast(pl.Utf8).is_in(merged_ids))
    if f3_filtered.is_empty():
        return None
    time_map = merged.select(["match_id", "start_time"])
    f3_cols = [c for c in _STAT_COLS if c in f3_filtered.columns and c != "start_time"]
    f3_sel = f3_filtered.select(["match_id"] + f3_cols)
    return time_map.join(f3_sel, on="match_id", how="inner").drop("match_id")


def _use_or_compute_performance(df: pl.DataFrame) -> pl.DataFrame:
    """Utilise performance_score stocké si disponible, sinon recalcule sur l'historique local."""
    from src.analysis.performance_score import compute_performance_series

    if "performance_score" in df.columns and df["performance_score"].drop_nulls().len() > 0:
        return df.with_columns(pl.col("performance_score").alias("performance"))
    logger.debug(
        "_use_or_compute_performance: performance_score absent ou vide (%d lignes) — recalcul",
        len(df),
    )
    return df.with_columns(pl.Series("performance", compute_performance_series(df, df)))


def _render_trio_performance_charts(  # noqa: PLR0913
    me_df: DataFrameLike,
    f1_df: DataFrameLike,
    f2_df: DataFrameLike | None,
    me_name: str,
    f1_name: str,
    f2_name: str | None,
    f1_xuid: str,
    f2_xuid: str | None,
    *,
    f3_df: DataFrameLike | None = None,
    f3_name: str | None = None,
    f3_xuid: str | None = None,
    colors_by_name: dict[str, str] | None = None,
    records: dict[str, dict[str, float | None]] | None = None,
    records_per_map: dict | None = None,
) -> None:
    """Affiche les graphes de performance escouade (2, 3 ou 4 joueurs).

    Utilise performance_score stocké dans player_match_enrichment si disponible
    (injecté en amont via TeammatesService.enrich_with_performance_score).
    Sinon recalcule sur le sous-ensemble de matchs (fallback).

    Args:
        records: Records historiques pré-calculés depuis l'historique complet.
            Si None, aucun record n'est affiché sur les graphes.
    """
    merged = _merge_trio_dataframes(me_df, f1_df, f2_df)
    if merged.is_empty():
        logger.warning("_render_trio_performance_charts: merged vide, aucun match aligné")
        return

    d_self = _use_or_compute_performance(_extract_player_df(merged, None))
    d_f1 = _use_or_compute_performance(_extract_player_df(merged, "f1"))
    # f2 uniquement si le merge a produit des colonnes f2_* (f2 présent à l'escouade)
    has_f2_cols = any(c.startswith("f2_") for c in merged.columns)
    d_f2 = _use_or_compute_performance(_extract_player_df(merged, "f2")) if has_f2_cols else None

    d_f3_p = _align_f3_to_merged(f3_df, merged)
    if d_f3_p is not None:
        d_f3_p = _use_or_compute_performance(d_f3_p)

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
        records=records,
        records_per_map=records_per_map,
    )


# ---------------------------------------------------------------------------
# Médailles escouade
# ---------------------------------------------------------------------------


def _render_trio_medals(  # noqa: PLR0913
    match_ids: list[str],
    db_path: str,
    xuid: str,
    f1_xuid: str,
    f2_xuid: str | None,
    me_name: str,
    f1_name: str,
    f2_name: str | None,
    db_key: tuple[int, int] | None,
    top_medals_fn,
    *,
    f3_xuid: str | None = None,
    f3_name: str | None = None,
) -> None:
    """Affiche la section médailles de l'escouade (2, 3 ou 4 joueurs)."""
    st.subheader(t("tm_medals_all"))
    if not match_ids:
        st.info(t("tm_no_medals_aggregate"))
        return

    with st.spinner(t("tm_computing_medals_all")):
        top_self = top_medals_fn(db_path, xuid.strip(), match_ids, top_n=12, db_key=db_key)
        top_f1 = top_medals_fn(db_path, f1_xuid, match_ids, top_n=12, db_key=db_key)
        top_f2 = (
            top_medals_fn(db_path, f2_xuid, match_ids, top_n=12, db_key=db_key) if f2_xuid else None
        )
        top_f3 = (
            top_medals_fn(db_path, f3_xuid, match_ids, top_n=12, db_key=db_key) if f3_xuid else None
        )

    players_medals = [(me_name, top_self), (f1_name, top_f1)]
    if f2_xuid and f2_name and top_f2 is not None:
        players_medals.append((f2_name, top_f2))
    if f3_xuid and f3_name and top_f3 is not None:
        players_medals.append((f3_name, top_f3))

    cols = st.columns(len(players_medals))
    for col, (name, top) in zip(cols, players_medals, strict=False):
        with col, st.expander(name, expanded=True):
            render_medals_grid(
                [{"name_id": int(n), "count": int(c)} for n, c in (top or [])],
                cols_per_row=6,
            )
