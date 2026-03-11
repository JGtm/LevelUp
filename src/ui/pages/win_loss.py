"""Page Victoires/Défaites.

Analyse des victoires et défaites par période et par carte.
"""

from __future__ import annotations

import html as html_lib

import polars as pl
import streamlit as st

from src.config import HALO_COLORS
from src.data.services.win_loss_service import WinLossService
from src.ui.chart_utils import safe_chart_render
from src.ui.i18n import get_lang, t
from src.ui.pages.win_loss_table_style import (
    _styler_map,
    _to_float,
    loss_rate_cell_html,
    map_name_cell_html,
    perf_cell_html,
    plain_cell_html,
    ratio_cell_html,
    win_rate_cell_html,
)
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, PLOTLY_STATIC_CONFIG, fragment_if_available
from src.ui.tz import get_tz_name
from src.ui.vectorize_helpers import build_mapping
from src.visualization import (
    plot_map_comparison,
    plot_map_ratio_with_winloss,
    plot_matches_at_top_by_week,
    plot_metric_bars_by_match,
    plot_outcomes_over_time,
    plot_stacked_outcomes_by_category,
    plot_streak_chart,
    plot_win_ratio_heatmap,
)
from src.visualization._compat import DataFrameLike, ensure_polars


def _clear_min_matches_maps_auto() -> None:
    """Callback pour désactiver le mode auto du slider."""
    st.session_state["_min_matches_maps_auto"] = False


@fragment_if_available
def render_win_loss_page(  # noqa: PLR0913
    dff: DataFrameLike,
    base: DataFrameLike,
    picked_session_labels: list[str] | None,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
) -> None:
    """Affiche la page Victoires/Défaites.

    Args:
        dff: DataFrame filtré des matchs.
        base: DataFrame de base (toutes les parties après filtres Firefight).
        picked_session_labels: Labels des sessions sélectionnées.
        db_path: Chemin vers la base de données.
        xuid: XUID du joueur.
        db_key: Clé de cache de la DB.
    """
    dff = ensure_polars(dff)
    base = ensure_polars(base)
    if dff.is_empty():
        st.warning(t("no_matches"))
        return

    with st.spinner(t("wl_computing")):
        current_mode = st.session_state.get("filter_mode")
        is_session_scope = bool(current_mode == "Sessions" and picked_session_labels)

        bucket_label = _render_outcomes_over_time(dff, is_session_scope)
        _render_map_mode_breakdown(dff)
        _render_heatmap_section(dff)
        _render_top_by_week(dff)
        _render_streak_section(dff)
        _render_personal_score_section(dff)
        _render_period_section(dff, bucket_label, is_session_scope)
        _render_ratio_by_map_section(dff, base, db_path, xuid, db_key, is_session_scope)


def _render_outcomes_over_time(dff: pl.DataFrame, is_session_scope: bool) -> str:
    """Affiche le graphe outcomes over time. Retourne le bucket_label."""
    bucket_label = t("wl_period_default")
    with safe_chart_render("wl_cannot_display_evolution"):
        fig_out, bucket_label = plot_outcomes_over_time(
            dff, session_style=is_session_scope, lang=get_lang()
        )
        st.markdown(t("wl_bucket_intro", bucket=bucket_label))
        if fig_out is not None:
            st.plotly_chart(fig_out, width="stretch", config=PLOTLY_CLEAN_CONFIG)
        else:
            st.info(t("wl_insufficient_evolution"))
    return bucket_label


@fragment_if_available
def _render_map_mode_breakdown(dff: pl.DataFrame) -> None:
    """Affiche les résultats par carte et mode (Sprint 5.4)."""
    st.divider()
    st.subheader(t("wl_results_by_map_mode"))
    col_by_map, col_by_mode = st.columns(2)

    with col_by_map:
        st.markdown(f"##### {t('wl_by_map')}")
        if "map_name" in dff.columns and "outcome" in dff.columns:
            with safe_chart_render():
                fig_map = plot_stacked_outcomes_by_category(
                    dff,
                    "map_name",
                    title=None,
                    min_matches=1,
                    sort_by="total",
                    max_categories=12,
                    lang=get_lang(),
                )
                if fig_map is not None:
                    fig_map.update_layout(
                        legend={
                            "orientation": "h",
                            "yanchor": "bottom",
                            "y": 1.02,
                            "xanchor": "right",
                            "x": 1,
                        },
                        margin={"l": 40, "r": 20, "t": 30, "b": 80},
                    )
                    st.plotly_chart(fig_map, width="stretch", config=PLOTLY_STATIC_CONFIG)
                else:
                    st.info(t("insufficient_data_chart"))
        else:
            st.info(t("insufficient_data_chart"))

    with col_by_mode:
        st.markdown(f"##### {t('wl_by_mode')}")
        mode_col = (
            "mode_ui"
            if "mode_ui" in dff.columns
            else ("mode_category" if "mode_category" in dff.columns else "pair_name")
        )
        if mode_col in dff.columns and "outcome" in dff.columns:
            with safe_chart_render():
                # Extraire la partie après " : " pour raccourcir les noms de modes
                # (ex. "Arène : Assassin" → "Assassin") et regrouper les variantes
                dff_mode = dff.with_columns(
                    pl.col(mode_col)
                    .map_elements(
                        lambda s: s.split(" : ", 1)[1] if isinstance(s, str) and " : " in s else s,
                        return_dtype=pl.String,
                    )
                    .alias("_mode_short")
                )
                fig_mode = plot_stacked_outcomes_by_category(
                    dff_mode,
                    "_mode_short",
                    title=None,
                    min_matches=2,
                    sort_by="total",
                    max_categories=10,
                    lang=get_lang(),
                )
                if fig_mode is not None:
                    fig_mode.update_layout(
                        legend={
                            "orientation": "h",
                            "yanchor": "bottom",
                            "y": 1.02,
                            "xanchor": "right",
                            "x": 1,
                        },
                        margin={"l": 40, "r": 20, "t": 30, "b": 60},
                    )
                    st.plotly_chart(fig_mode, width="stretch", config=PLOTLY_STATIC_CONFIG)
                else:
                    st.info(t("insufficient_data_chart"))
        else:
            st.info(t("insufficient_data_chart"))


@fragment_if_available
def _render_heatmap_section(dff: pl.DataFrame) -> None:
    """Affiche la heatmap Win Rate par jour et heure."""
    st.divider()
    st.subheader(t("wl_heatmap_title"))
    st.caption(t("wl_heatmap_caption", tz=get_tz_name()))
    if "start_time" in dff.columns and "outcome" in dff.columns:
        with safe_chart_render():
            fig_heat = plot_win_ratio_heatmap(dff, title=None, min_matches=1, lang=get_lang())
            if fig_heat is not None:
                st.plotly_chart(fig_heat, width="stretch", config=PLOTLY_STATIC_CONFIG)
            else:
                st.info(t("insufficient_data_chart"))
    else:
        st.info(t("missing_time_data"))


@fragment_if_available
def _render_top_by_week(dff: pl.DataFrame) -> None:
    """Affiche Matchs Top vs Total par semaine (Sprint 5.4.7)."""
    st.divider()
    st.subheader(t("wl_top_by_week"))
    st.caption(t("wl_top_by_week_caption"))
    if "start_time" not in dff.columns:
        st.info(t("missing_time_data"))
        return
    rank_col = "rank" if "rank" in dff.columns else "outcome"
    with safe_chart_render():
        fig_top = plot_matches_at_top_by_week(
            dff,
            title=None,
            rank_col=rank_col,
            top_n_ranks=1,
            lang=get_lang(),
        )
        if fig_top is not None:
            st.plotly_chart(fig_top, width="stretch", config=PLOTLY_CLEAN_CONFIG)
        else:
            st.info(t("insufficient_data_chart"))


@fragment_if_available
def _render_streak_section(dff: pl.DataFrame) -> None:
    """Affiche les séries de victoires/défaites (Sprint 7.2)."""
    st.divider()
    st.subheader(t("wl_streaks"))
    st.caption(t("wl_streaks_caption"))
    if "outcome" in dff.columns and "start_time" in dff.columns:
        with safe_chart_render():
            fig_streak = plot_streak_chart(dff, title=None, lang=get_lang())
            if fig_streak is not None:
                st.plotly_chart(fig_streak, width="stretch", config=PLOTLY_CLEAN_CONFIG)
            else:
                st.info(t("insufficient_data_chart"))
    else:
        st.info(t("missing_outcome_data"))


def _render_personal_score_section(dff: pl.DataFrame) -> None:
    """Affiche le score personnel par match (Sprint 7.1)."""
    if "personal_score" not in dff.columns:
        return
    st.divider()
    st.subheader(t("wl_personal_score"))
    st.caption(t("wl_personal_score_caption"))
    colors = HALO_COLORS.as_dict()
    fig_ps = plot_metric_bars_by_match(
        dff,
        metric_col="personal_score",
        title=None,
        y_axis_title=t("wl_personal_score_y_axis"),
        hover_label=t("wl_personal_score_hover"),
        bar_color=colors["amber"],
        smooth_color=colors["violet"],
        smooth_window=10,
    )
    if fig_ps is not None:
        st.plotly_chart(fig_ps, width="stretch", config=PLOTLY_CLEAN_CONFIG)
    else:
        st.info(t("insufficient_data_chart"))


def _render_period_section(
    dff: pl.DataFrame,
    bucket_label: str,
    is_session_scope: bool,
) -> None:
    """Affiche le tableau par période."""
    st.divider()
    st.subheader(t("wl_period"))
    # compute_period_table accepte directement un pl.DataFrame
    period = WinLossService.compute_period_table(
        dff, bucket_label, is_session_scope, lang=get_lang()
    )
    if period.is_empty:
        st.info(t("wl_no_period_data"))
        return

    # Conversion pandas à la frontière pour le styling .style
    out_tbl = period.table.to_pandas()

    def _style_pct(v) -> str:
        try:
            float(v)
        except Exception:
            return ""
        return "color: #E0E0E0; font-weight: 700;"

    win_rate_col = (
        t("wl_period_col_win_rate") if t("wl_period_col_win_rate") in out_tbl.columns else None
    )
    if win_rate_col:
        out_styled = _styler_map(out_tbl.style, _style_pct, subset=[win_rate_col])
        col_cfg = {win_rate_col: st.column_config.NumberColumn(win_rate_col, format="%.1f%%")}
    else:
        out_styled = out_tbl.style
        col_cfg = {}

    st.dataframe(out_styled, width="stretch", hide_index=True, column_config=col_cfg)


@fragment_if_available
def _render_ratio_by_map_section(  # noqa: PLR0913
    dff: pl.DataFrame,
    base: pl.DataFrame,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
    is_session_scope: bool = False,
) -> None:
    """Affiche le ratio par cartes avec sélection du scope."""
    st.divider()
    st.subheader(t("wl_ratio_by_map"))
    st.caption(t("wl_ratio_caption"))

    _scope_labels: dict[str, str] = {
        "Moi (filtres actuels)": t("wl_scope_me_filtered"),
        "Moi (toutes les parties)": t("wl_scope_me_all"),
    }
    scope = st.radio(
        t("wl_scope_label"),
        options=[
            "Moi (filtres actuels)",
            "Moi (toutes les parties)",
            "Avec SpartanA",
            "Avec SpartanB",
        ],
        format_func=lambda k: _scope_labels.get(k, k),
        horizontal=True,
    )
    min_matches = st.slider(
        t("wl_min_matches_map_slider"),
        1,
        30,
        1,
        step=1,
        key="min_matches_maps",
        on_change=_clear_min_matches_maps_auto,
    )

    base_scope_pl = WinLossService.get_friend_scope_df(
        scope,
        dff,
        base,
        db_path,
        xuid,
        db_key,
    )

    with st.spinner(t("wl_computing_map")):
        map_result = WinLossService.compute_map_breakdown(base_scope_pl, min_matches)
        breakdown = map_result.breakdown if not map_result.is_empty else pl.DataFrame()

    if map_result.is_empty:
        st.warning(t("wl_not_enough_map"))
        return

    if is_session_scope:
        st.caption(t("wl_session_map_note"))
        metric_options = [
            ("performance_avg", t("wl_metric_performance")),
            ("accuracy_avg", t("wl_metric_accuracy")),
        ]
    else:
        metric_options = [
            ("performance_avg", t("wl_metric_performance")),
            ("ratio_global", t("wl_metric_ratio")),
            ("win_rate", t("wl_metric_win_rate")),
            ("accuracy_avg", t("wl_metric_accuracy")),
        ]
    metric = st.selectbox(
        t("wl_metric_label"),
        options=metric_options,
        format_func=lambda x: x[1],
    )
    key, label = metric

    view = breakdown.head(20).reverse()
    with safe_chart_render():
        if key == "ratio_global":
            fig = plot_map_ratio_with_winloss(view, title=label, absolute_counts=True)
        else:
            fig = plot_map_comparison(
                view, key, title=label, color_by_sign=(key == "performance_avg")
            )

        if fig is not None:
            if key in ("win_rate",):
                fig.update_xaxes(tickformat=".0%")
            if key in ("accuracy_avg",):
                fig.update_xaxes(ticksuffix="%")
            st.plotly_chart(fig, width="stretch", config=PLOTLY_STATIC_CONFIG)
        else:
            st.info(t("insufficient_data_chart"))

    base_scope = base_scope_pl
    _render_map_table(breakdown, base_scope)


def _render_map_table(breakdown: pl.DataFrame, base_scope: pl.DataFrame) -> None:  # noqa: PLR0912, PLR0915, C901
    """Affiche le tableau détaillé par carte (HTML pur, sans pandas)."""
    from src.ui.translations import translate_playlist_name

    # -- Transformations numériques (Polars)
    has_perf = "performance_avg" in breakdown.columns
    cols_expr = [
        (pl.col("win_rate") * 100).round(1).alias("win_rate"),
        (pl.col("loss_rate") * 100).round(1).alias("loss_rate"),
        pl.col("accuracy_avg").cast(pl.Float64, strict=False).round(2).alias("accuracy_avg"),
        pl.col("ratio_global").cast(pl.Float64, strict=False).round(2).alias("ratio_global"),
    ]
    if has_perf:
        cols_expr.append(
            pl.col("performance_avg")
            .cast(pl.Float64, strict=False)
            .round(1)
            .alias("performance_avg")
        )
    tbl = breakdown.with_columns(cols_expr)

    def _single_or_multi_label(values: list) -> str:
        """Détermine un label unique ou 'Plusieurs' à partir d'une liste."""
        try:
            vals = sorted({str(x).strip() for x in values if x is not None and str(x).strip()})
        except Exception:
            return "-"
        if not vals:
            return "-"
        if len(vals) == 1:
            return vals[0]
        return t("wl_several")

    def _clean_asset_label(s: str | None) -> str:
        """Nettoie un label d'asset."""
        if not s:
            return ""
        return str(s).split("/")[-1].replace("-", " ").strip().title()

    def _normalize_mode_label(p: str | None) -> str | None:
        """Normalise un label de mode de jeu."""
        from src.ui.translations import translate_pair_name

        return translate_pair_name(p, lang=get_lang()) if p else None

    if "playlist_ui" in base_scope.columns:
        playlist_ctx = _single_or_multi_label(base_scope["playlist_ui"].drop_nulls().to_list())
    else:

        def _clean_then_translate(s: str | None) -> str | None:
            return translate_playlist_name(_clean_asset_label(s), lang=get_lang())

        _pl_map = build_mapping(base_scope["playlist_name"], _clean_then_translate)
        playlist_vals = (
            base_scope["playlist_name"]
            .cast(pl.Utf8)
            .replace_strict(_pl_map, default=None, return_dtype=pl.Utf8)
            .drop_nulls()
            .to_list()
        )
        playlist_ctx = _single_or_multi_label(playlist_vals)

    if "mode_ui" in base_scope.columns:
        mode_ctx = _single_or_multi_label(base_scope["mode_ui"].drop_nulls().to_list())
    else:
        _mode_map = build_mapping(base_scope["pair_name"], _normalize_mode_label)
        mode_vals = (
            base_scope["pair_name"]
            .cast(pl.Utf8)
            .replace_strict(_mode_map, default=None, return_dtype=pl.Utf8)
            .drop_nulls()
            .to_list()
        )
        mode_ctx = _single_or_multi_label(mode_vals)

    tbl = tbl.with_columns(
        [
            pl.lit(playlist_ctx).alias("playlist_ctx"),
            pl.lit(mode_ctx).alias("mode_ctx"),
        ]
    )

    # -- Définition des colonnes (clé interne → label traduit)
    col_defs: list[tuple[str, str]] = [
        ("map_name", t("wl_col_map")),
        ("playlist_ctx", t("wl_col_playlist")),
        ("mode_ctx", t("wl_col_mode")),
        ("matches", t("wl_col_matches")),
        ("accuracy_avg", t("wl_col_accuracy_avg")),
    ]
    if has_perf:
        col_defs.append(("performance_avg", t("wl_col_performance_avg")))
    col_defs += [
        ("win_rate", t("wl_col_win_rate")),
        ("loss_rate", t("wl_col_loss_rate")),
        ("ratio_global", t("wl_col_ratio")),
    ]
    col_defs = [(k, lbl) for k, lbl in col_defs if k in tbl.columns]

    # -- En-têtes
    head = (
        "<thead><tr>"
        + "".join(f"<th>{html_lib.escape(lbl)}</th>" for _, lbl in col_defs)
        + "</tr></thead>"
    )

    # -- Lignes
    rows_html: list[str] = []
    for r in tbl.to_dicts():
        win_pct = _to_float(r.get("win_rate"))
        loss_pct = _to_float(r.get("loss_rate"))
        tds: list[str] = []
        for key, _ in col_defs:
            if key == "map_name":
                tds.append(map_name_cell_html(r.get(key)))
            elif key == "win_rate":
                display = f"{win_pct:.1f}" if win_pct is not None else "-"
                tds.append(win_rate_cell_html(win_pct, loss_pct, display))
            elif key == "loss_rate":
                display = f"{loss_pct:.1f}" if loss_pct is not None else "-"
                tds.append(loss_rate_cell_html(win_pct, loss_pct, display))
            elif key == "ratio_global":
                rv = _to_float(r.get(key))
                display = f"{rv:.2f}" if rv is not None else "-"
                tds.append(ratio_cell_html(rv, display))
            elif key == "performance_avg":
                tds.append(perf_cell_html(r.get(key)))
            elif key == "matches":
                tds.append(plain_cell_html(r.get(key), fmt="%d"))
            elif key == "accuracy_avg":
                tds.append(plain_cell_html(r.get(key), fmt="%.2f"))
            else:
                tds.append(plain_cell_html(r.get(key)))
        rows_html.append("<tr>" + "".join(tds) + "</tr>")

    body = "<tbody>" + "".join(rows_html) + "</tbody>"
    st.markdown(
        f"<div class='os-table-wrap'><table class='os-table'>{head}{body}</table></div>",
        unsafe_allow_html=True,
    )
