"""Vues de rendu pour la page Coéquipiers (single, multi).

Extraites de teammates.py (Sprint 16 — refactoring Phase A).
La vue trio est dans ``_teammates_trio.py``.
"""

from __future__ import annotations

import polars as pl
import streamlit as st

from src.analysis import (
    compute_map_breakdown,
)
from src.app._page_context import TeammateCallbacks, TeammateContext, TeammateFilterOptions
from src.data.services.teammates_service import TeammatesService
from src.ui import display_name_from_xuid
from src.ui.cache import (
    cached_friend_matches_df,
)
from src.ui.i18n import get_lang, t
from src.ui.pages._teammates_trio import render_trio_view
from src.ui.pages._teammates_trio_helpers import _merge_trio_dataframes
from src.ui.pages.teammates_charts import (
    render_comparison_charts,
    render_metric_bar_charts,
    render_outcome_bar_chart,
)
from src.ui.pages.teammates_helpers import (
    render_friends_history_table,
)
from src.ui.pages.teammates_impact import render_impact_taquinerie
from src.ui.pages.teammates_map_charts import (
    render_map_charts_section,
    render_single_map_section,
    render_squad_heatmap,
    render_squad_timeline,
)
from src.ui.pages.teammates_synergy import render_synergy_radar
from src.ui.pages.teammates_views_shared import (
    _collect_friend_match_ids,
    _render_match_details_expander,
    _render_shared_medals,
    _render_shared_stats_metrics,
)
from src.ui.pages.teammates_weapons import render_weapon_kills_bar_chart
from src.ui.tz import get_tz_name
from src.visualization._compat import DataFrameLike, ensure_polars

# Ré-exports pour compatibilité (tests importent depuis ce module)
__all__ = [
    "_merge_trio_dataframes",
    "render_multi_teammate_view",
    "render_single_teammate_view",
    "render_trio_view",
]


def render_single_teammate_view(
    df: DataFrameLike,
    dff: DataFrameLike,
    ctx: TeammateContext,
    filters: TeammateFilterOptions,
    callbacks: TeammateCallbacks,
) -> None:
    """Vue pour un seul coéquipier sélectionné."""
    xuid = ctx["xuid"]
    db_path = ctx["db_path"]
    db_key = ctx["db_key"]
    picked_xuids = ctx["picked_xuids"]
    apply_current_filters = filters["apply_current_filters"]
    same_team_only = filters["same_team_only"]
    show_smooth = filters["show_smooth"]
    load_teammate_stats_fn = callbacks["load_teammate_stats_fn"]

    df = ensure_polars(df)
    dff = ensure_polars(dff)
    friend_xuid = picked_xuids[0]
    with st.spinner(t("tm_computing_teammate")):
        dfr = ensure_polars(
            cached_friend_matches_df(
                db_path,
                xuid.strip(),
                friend_xuid,
                same_team_only=bool(same_team_only),
                db_key=db_key,
                tz_name=get_tz_name(),
            )
        )
        if dfr.is_empty():
            st.warning(t("tm_no_matches_teammate"))
            return

        render_outcome_bar_chart(dfr)

        _render_match_details_expander(dfr)

        base_for_friend = dff if apply_current_filters else df
        shared_ids = set(dfr["match_id"].cast(pl.Utf8).to_list())
        sub = base_for_friend.filter(pl.col("match_id").cast(pl.Utf8).is_in(shared_ids))

        if sub.is_empty():
            st.info(t("tm_no_matches_filters"))
            return

        name = display_name_from_xuid(friend_xuid, db_path=db_path)

        _render_shared_stats_metrics(sub)
        friend_sub = ensure_polars(load_teammate_stats_fn(name, shared_ids, db_path))
        _render_single_teammate_details(  # noqa: PLR0913
            sub=sub,
            dfr=dfr,
            friend_sub=friend_sub,
            shared_ids=shared_ids,
            name=name,
            friend_xuid=friend_xuid,
            xuid=xuid,
            db_path=db_path,
            db_key=db_key,
            show_smooth=show_smooth,
            callbacks=callbacks,
        )


def _render_single_teammate_details(  # noqa: PLR0913
    sub: pl.DataFrame,
    dfr: pl.DataFrame,
    friend_sub: pl.DataFrame,
    shared_ids: set[str],
    name: str,
    friend_xuid: str,
    xuid: str,
    db_path: str,
    db_key: tuple[int, int] | None,
    show_smooth: bool,
    callbacks: TeammateCallbacks,
) -> None:
    assign_player_colors_fn = callbacks["assign_player_colors_fn"]
    plot_multi_metric_bars_fn = callbacks["plot_multi_metric_bars_fn"]
    top_medals_fn = callbacks["top_medals_fn"]
    enrich_series_fn = callbacks["enrich_series_fn"]
    if not friend_sub.is_empty() and "match_id" in friend_sub.columns:
        filtered_match_ids = sub["match_id"].cast(pl.Utf8).to_list()
        friend_sub = friend_sub.filter(pl.col("match_id").cast(pl.Utf8).is_in(filtered_match_ids))

    me_name = display_name_from_xuid(xuid.strip(), db_path=db_path)
    sub = TeammatesService.enrich_with_performance_score(sub, me_name, db_path, is_main=True)
    if not friend_sub.is_empty():
        friend_sub = TeammatesService.enrich_with_performance_score(friend_sub, name, db_path)

    render_comparison_charts(
        sub=sub,
        friend_sub=friend_sub,
        me_name=me_name,
        friend_name=name,
        friend_xuid=friend_xuid,
        show_smooth=show_smooth,
    )
    series = [(me_name, sub)]
    if not friend_sub.is_empty():
        series.append((name, friend_sub))
    colors_by_name = assign_player_colors_fn([n for n, _ in series])
    series = enrich_series_fn(series, db_path)

    render_weapon_kills_bar_chart(
        player_infos=[(me_name, xuid, list(shared_ids)), (name, friend_xuid, list(shared_ids))],
        colors_by_name=colors_by_name,
        db_path=db_path,
        key_suffix=friend_xuid,
    )
    render_metric_bar_charts(
        series=series,
        colors_by_name=colors_by_name,
        show_smooth=show_smooth,
        key_suffix=friend_xuid,
        plot_fn=plot_multi_metric_bars_fn,
    )
    render_synergy_radar(
        sub=sub,
        friend_sub=friend_sub,
        me_name=me_name,
        friend_name=name,
        colors_by_name=colors_by_name,
    )
    _render_shared_medals(
        db_path,
        xuid,
        friend_xuid,
        me_name,
        name,
        shared_ids,
        db_key,
        top_medals_fn,
    )

    _render_single_teammate_weapon_and_map(
        sub=sub,
        dfr=dfr,
        friend_sub=friend_sub,
        me_name=me_name,
        name=name,
        db_path=db_path,
        friend_xuid=friend_xuid,
        shared_ids=shared_ids,
    )


def _render_single_teammate_weapon_and_map(  # noqa: PLR0913
    sub: pl.DataFrame,
    dfr: pl.DataFrame,
    friend_sub: pl.DataFrame,
    me_name: str,
    name: str,
    db_path: str,
    friend_xuid: str,
    shared_ids: set[str],
) -> None:
    """Rend le tableau d'armes et les graphes par carte en vue single teammate."""
    from src.ui.pages.teammates_weapons import render_weapon_kills_table

    render_weapon_kills_table(db_path, friend_xuid, list(shared_ids))
    render_single_map_section(
        sub=sub,
        dfr=dfr,
        series=[(me_name, sub)] + ([(name, friend_sub)] if not friend_sub.is_empty() else []),
        lang=get_lang(),
    )
    render_squad_timeline(
        db_path=db_path,
        me_name=me_name,
        friend_names=[name],
        all_match_ids=list(shared_ids),
        lang=get_lang(),
    )


def render_multi_teammate_view(  # noqa: PLR0913
    df: DataFrameLike,
    dff: DataFrameLike,
    base: DataFrameLike,
    ctx: TeammateContext,
    filters: TeammateFilterOptions,
    callbacks: TeammateCallbacks,
) -> None:
    """Vue pour plusieurs coéquipiers sélectionnés."""
    db_path = ctx["db_path"]
    db_key = ctx["db_key"]
    picked_xuids = ctx["picked_xuids"]
    aliases_key = ctx["aliases_key"]
    include_firefight = ctx["include_firefight"]
    show_smooth = filters["show_smooth"]
    assign_player_colors_fn = callbacks["assign_player_colors_fn"]
    plot_multi_metric_bars_fn = callbacks["plot_multi_metric_bars_fn"]
    top_medals_fn = callbacks["top_medals_fn"]
    load_teammate_stats_fn = callbacks["load_teammate_stats_fn"]
    enrich_series_fn = callbacks["enrich_series_fn"]

    df = ensure_polars(df)
    dff = ensure_polars(dff)
    base = ensure_polars(base)

    sub_all, series, colors_by_name, rendered_bottom_charts = _render_map_history_section(
        df=df,
        dff=dff,
        ctx=ctx,
        filters=filters,
        callbacks=callbacks,
    )

    if len(picked_xuids) >= 2:
        rendered_bottom_charts = render_trio_view(
            df=df,
            dff=dff,
            base=base,
            me_name=ctx["me_name"],
            xuid=ctx["xuid"],
            db_path=db_path,
            db_key=db_key,
            aliases_key=aliases_key,
            picked_xuids=picked_xuids,
            apply_current_filters=filters["apply_current_filters"],
            include_firefight=include_firefight,
            series=series,
            colors_by_name=colors_by_name,
            show_smooth=show_smooth,
            assign_player_colors_fn=assign_player_colors_fn,
            plot_multi_metric_bars_fn=plot_multi_metric_bars_fn,
            top_medals_fn=top_medals_fn,
            load_teammate_stats_fn=load_teammate_stats_fn,
            enrich_series_fn=enrich_series_fn,
        )

    if len(picked_xuids) >= 2:
        impact_match_ids = (
            sub_all.sort("start_time")["match_id"]
            .cast(pl.Utf8)
            .unique(maintain_order=True)
            .to_list()
            if not sub_all.is_empty() and "start_time" in sub_all.columns
            else sub_all["match_id"].cast(pl.Utf8).unique().to_list()
            if not sub_all.is_empty()
            else []
        )
        render_impact_taquinerie(
            db_path=db_path,
            xuid=ctx["xuid"],
            match_ids=impact_match_ids,
            friend_xuids=picked_xuids,
            db_key=db_key,
        )

    if not rendered_bottom_charts:
        _render_bottom_charts(sub_all, series, colors_by_name, ctx, filters, callbacks)


def _render_bottom_charts(  # noqa: PLR0913
    sub_all: DataFrameLike,
    series: list,
    colors_by_name: dict,
    ctx: TeammateContext,
    filters: TeammateFilterOptions,
    callbacks: TeammateCallbacks,
) -> None:
    """Affiche les graphes d'armes et de métriques en bas de la vue multi."""
    db_path = ctx["db_path"]
    show_smooth = filters["show_smooth"]
    enrich_series_fn = callbacks["enrich_series_fn"]
    plot_multi_metric_bars_fn = callbacks["plot_multi_metric_bars_fn"]

    series = enrich_series_fn(series, db_path)
    _match_ids = sub_all["match_id"].cast(pl.Utf8).to_list() if not sub_all.is_empty() else []
    if _match_ids:
        _player_infos = [(ctx["me_name"], ctx["xuid"], _match_ids)]
        for fx in ctx["picked_xuids"]:
            fx_name = display_name_from_xuid(str(fx), db_path=db_path)
            _player_infos.append((fx_name, str(fx), _match_ids))
        render_weapon_kills_bar_chart(
            player_infos=_player_infos,
            colors_by_name=colors_by_name,
            db_path=db_path,
            key_suffix=f"multi_{len(series)}",
        )
    render_metric_bar_charts(
        series=series,
        colors_by_name=colors_by_name,
        show_smooth=show_smooth,
        key_suffix=f"{len(series)}",
        plot_fn=plot_multi_metric_bars_fn,
    )


def _render_map_breakdown(
    sub_all: DataFrameLike,
    full_squad_df: DataFrameLike,
    breakdown_all: DataFrameLike,
    ctx: TeammateContext,
) -> None:
    """Affiche lollipop + timeline + bullet + perf vs historique, puis tableau d'historique."""
    db_path = ctx["db_path"]
    xuid = ctx["xuid"]
    db_key = ctx["db_key"]
    waypoint_player = ctx["waypoint_player"]

    render_map_charts_section(sub_all, full_squad_df, breakdown_all, lang=get_lang())

    st.subheader(t("tm_history"))
    st.caption(t("tm_history_tz_caption", tz=get_tz_name()))
    if ensure_polars(sub_all).is_empty():
        st.info(t("tm_no_matches_filter"))
    else:
        render_friends_history_table(
            sub_all, db_path, xuid, db_key, waypoint_player, full_df=full_squad_df
        )


def _render_map_history_section(
    df: DataFrameLike,
    dff: DataFrameLike,
    ctx: TeammateContext,
    filters: TeammateFilterOptions,
    callbacks: TeammateCallbacks,
) -> tuple[DataFrameLike, list, dict, bool]:
    """Rend la carte W/L et l'historique multi-coéquipiers.

    Retourne (sub_all, series, colors_by_name, rendered_bottom_charts).
    """
    me_name = ctx["me_name"]
    xuid = ctx["xuid"]
    db_path = ctx["db_path"]
    db_key = ctx["db_key"]
    picked_xuids = ctx["picked_xuids"]
    apply_current_filters = filters["apply_current_filters"]
    same_team_only = filters["same_team_only"]
    assign_player_colors_fn = callbacks["assign_player_colors_fn"]
    load_teammate_stats_fn = callbacks["load_teammate_stats_fn"]

    df = ensure_polars(df)
    dff = ensure_polars(dff)
    rendered_bottom_charts = False

    with st.spinner(t("tm_computing_map")):
        base_for_friends_all = dff if apply_current_filters else df
        all_match_ids, per_friend_ids = _collect_friend_match_ids(
            db_path, xuid, picked_xuids, same_team_only, db_key
        )
        sub_all = base_for_friends_all.filter(
            pl.col("match_id").cast(pl.Utf8).is_in(list(all_match_ids))
        )

        sub_all = TeammatesService.enrich_with_performance_score(
            sub_all, me_name, db_path, is_main=True
        )
        series: list[tuple[str, DataFrameLike]] = [(me_name, sub_all)]
        filtered_match_ids = (
            sub_all["match_id"].cast(pl.Utf8).to_list() if not sub_all.is_empty() else []
        )

        with st.spinner(t("tm_computing_stats")):
            for fx in picked_xuids:
                ids = per_friend_ids.get(str(fx), set())
                if not ids:
                    continue
                fx_gamertag = display_name_from_xuid(str(fx), db_path=db_path)
                fr_sub = ensure_polars(
                    load_teammate_stats_fn(fx_gamertag, {str(x) for x in ids}, db_path)
                )
                if fr_sub.is_empty():
                    continue
                if "match_id" in fr_sub.columns and filtered_match_ids:
                    fr_sub = fr_sub.filter(
                        pl.col("match_id").cast(pl.Utf8).is_in(filtered_match_ids)
                    )
                if fr_sub.is_empty():
                    continue
                fr_sub = TeammatesService.enrich_with_performance_score(
                    fr_sub, fx_gamertag, db_path
                )
                series.append((fx_gamertag, fr_sub))

        colors_by_name = assign_player_colors_fn([n for n, _ in series])
        breakdown_all = ensure_polars(compute_map_breakdown(sub_all))

        # full_squad_df = tous les matchs avec ces amis (non filtré par session/date)
        full_squad_df = df.filter(pl.col("match_id").cast(pl.Utf8).is_in(list(all_match_ids)))
        _render_map_breakdown(sub_all, full_squad_df, breakdown_all, ctx)
        render_squad_heatmap(series, lang=get_lang())
        render_squad_timeline(
            db_path=db_path,
            me_name=me_name,
            friend_names=[display_name_from_xuid(str(fx), db_path=db_path) for fx in picked_xuids],
            all_match_ids=list(all_match_ids),
            lang=get_lang(),
        )

    return sub_all, series, colors_by_name, rendered_bottom_charts


# ---------------------------------------------------------------------------
# Sous-fonctions privées
# ---------------------------------------------------------------------------
