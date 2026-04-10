"""Vues de rendu pour la page Coéquipiers (multi / escouade).

Extraites de teammates.py (Sprint 16 — refactoring Phase A).
La vue trio (1, 2 ou 3 coéquipiers) est dans ``_teammates_trio.py``.
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
from src.ui.components.browser_storage import hints_visible
from src.ui.i18n import get_lang, t
from src.ui.pages._teammates_trio import render_trio_view
from src.ui.pages._teammates_trio_helpers import _merge_trio_dataframes, _render_trio_medals
from src.ui.pages.teammates_charts import (
    render_metric_bar_charts,
)
from src.ui.pages.teammates_helpers import (
    render_friends_history_table,
)
from src.ui.pages.teammates_impact import render_impact_taquinerie
from src.ui.pages.teammates_map_charts import (
    render_map_charts_section,
    render_squad_heatmap,
    render_squad_timeline,
)
from src.ui.pages.teammates_views_shared import (
    _collect_friend_match_ids,
)
from src.ui.pages.teammates_weapons import render_weapon_kills_bar_chart
from src.ui.translations import resolve_map_display_names
from src.ui.tz import get_tz_name
from src.visualization._compat import DataFrameLike, ensure_polars

# Ré-exports pour compatibilité (tests importent depuis ce module)
__all__ = [
    "_merge_trio_dataframes",
    "render_multi_teammate_view",
    "render_trio_view",
]


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

    if len(picked_xuids) >= 1:
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

    # Médailles — toujours en dernier
    if rendered_bottom_charts and picked_xuids:
        _medal_match_ids = (
            sub_all["match_id"].cast(pl.Utf8).to_list() if not sub_all.is_empty() else []
        )
        f1_xuid = picked_xuids[0]
        f2_xuid = picked_xuids[1] if len(picked_xuids) >= 2 else None
        f3_xuid = picked_xuids[2] if len(picked_xuids) >= 3 else None
        f1_name = display_name_from_xuid(f1_xuid, db_path=db_path)
        f2_name = display_name_from_xuid(f2_xuid, db_path=db_path) if f2_xuid else None
        f3_name = display_name_from_xuid(f3_xuid, db_path=db_path) if f3_xuid else None
        _render_trio_medals(
            _medal_match_ids,
            db_path,
            ctx["xuid"],
            f1_xuid,
            f2_xuid,
            ctx["me_name"],
            f1_name,
            f2_name,
            db_key,
            top_medals_fn,
            f3_xuid=f3_xuid,
            f3_name=f3_name,
        )


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
    from src.visualization._chart_series import SquadRecordSet

    _show_records = bool(getattr(st.session_state.get("app_settings"), "show_records", False))
    _spree_record_set: SquadRecordSet | None = None
    _hspk_dict: dict | None = None
    if _show_records:
        from src.analysis.squad_records import compute_squad_records, get_dominant_pair_name

        _series_pl = [(n, ensure_polars(d)) for n, d in series]
        _dom_pair = get_dominant_pair_name([d for _, d in _series_pl])
        _spree_rec = compute_squad_records(_series_pl, [("max_killing_spree", False)], _dom_pair)
        _hspk_dfs = [
            (
                n,
                d.with_columns(
                    (
                        pl.col("headshot_kills").fill_null(0) + pl.col("perfect_kills").fill_null(0)
                    ).alias("hs_pk_total")
                ),
            )
            for n, d in _series_pl
            if "headshot_kills" in d.columns
        ]
        _hspk_rec = compute_squad_records(_hspk_dfs, [("hs_pk_total", False)], _dom_pair)
        _spree_record_set = SquadRecordSet(records=_spree_rec, per_map={})
        _hspk_dict = _hspk_rec
    render_metric_bar_charts(
        series=series,
        colors_by_name=colors_by_name,
        show_smooth=show_smooth,
        key_suffix=f"{len(series)}",
        plot_fn=plot_multi_metric_bars_fn,
        squad_records=_spree_record_set,
        hspk_records=_hspk_dict,
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
    if hints_visible():
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
                # Ajouter map_ui (traduit) si absent — même logique que _add_derived_columns
                if (
                    "map_ui" not in fr_sub.columns
                    and "map_id" in fr_sub.columns
                    and "map_name" in fr_sub.columns
                ):
                    _id_to_fallback = {
                        str(r["map_id"]): str(r["map_name"] or "")
                        for r in fr_sub.select(["map_id", "map_name"])
                        .drop_nulls(subset=["map_id"])
                        .unique(subset=["map_id"])
                        .iter_rows(named=True)
                    }
                    if _id_to_fallback:
                        _translated = resolve_map_display_names(_id_to_fallback, get_lang())
                        fr_sub = fr_sub.with_columns(
                            pl.col("map_id")
                            .cast(pl.Utf8)
                            .replace_strict(
                                _translated,
                                default=pl.col("map_name").cast(pl.Utf8),
                                return_dtype=pl.Utf8,
                            )
                            .alias("map_ui")
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
        # Désactivé temporairement: ne plus afficher ni charger les données
        # du profil de tempo synchronisé, tout en conservant le code en place.

    return sub_all, series, colors_by_name, rendered_bottom_charts


# ---------------------------------------------------------------------------
# Sous-fonctions privées
# ---------------------------------------------------------------------------
