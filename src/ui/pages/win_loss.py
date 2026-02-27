"""Page Victoires/Défaites.

Analyse des victoires et défaites par période et par carte.
"""

from __future__ import annotations

import pandas as pd  # requis pour l'API .style de Streamlit
import polars as pl
import streamlit as st

from src.config import HALO_COLORS
from src.data.services.win_loss_service import WinLossService
from src.ui.i18n import get_lang, t
from src.ui.streamlit_modern import fragment_if_available
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


def _styler_map(styler, func, subset):
    """Applique un style en mode compatible pandas 1.x et 2.x."""
    try:
        return styler.map(func, subset=subset)
    except AttributeError:
        return styler.applymap(func, subset=subset)


def _to_float(v: object) -> float | None:
    """Convertit une valeur en float, ou None si impossible."""
    try:
        if v is None:
            return None
        x = float(v)
        return x if x == x else None
    except Exception:
        return None


def _style_map_table_row(row: pd.Series) -> pd.Series:
    """Style les lignes du tableau par carte."""
    green = str(getattr(HALO_COLORS, "green", "#2ECC71"))
    red = str(getattr(HALO_COLORS, "red", "#E74C3C"))
    violet = "#8E6CFF"

    col_win = t("wl_col_win_rate")
    col_loss = t("wl_col_loss_rate")
    col_ratio = t("wl_col_ratio")

    win_pct = _to_float(row.get(col_win))
    loss_pct = _to_float(row.get(col_loss))
    ratio_val = _to_float(row.get(col_ratio))

    styles: dict[str, str] = {str(c): "" for c in row.index}

    if win_pct is not None and loss_pct is not None:
        if win_pct > loss_pct:
            styles[col_win] = f"color: {green}; font-weight: 800;"
            styles[col_loss] = f"color: {red}; font-weight: 800;"
        elif win_pct < loss_pct:
            styles[col_win] = f"color: {red}; font-weight: 800;"
            styles[col_loss] = f"color: {green}; font-weight: 800;"
        else:
            styles[col_win] = f"color: {violet}; font-weight: 800;"
            styles[col_loss] = f"color: {violet}; font-weight: 800;"

    if ratio_val is not None:
        if ratio_val > 1.0:
            styles[col_ratio] = f"color: {green}; font-weight: 800;"
        elif ratio_val < 1.0:
            styles[col_ratio] = f"color: {red}; font-weight: 800;"
        else:
            styles[col_ratio] = f"color: {violet}; font-weight: 800;"

    return pd.Series(styles)


@fragment_if_available
def render_win_loss_page(
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
        _render_ratio_by_map_section(dff, base, db_path, xuid, db_key)


def _render_outcomes_over_time(dff: pl.DataFrame, is_session_scope: bool) -> str:
    """Affiche le graphe outcomes over time. Retourne le bucket_label."""
    try:
        fig_out, bucket_label = plot_outcomes_over_time(
            dff, session_style=is_session_scope, lang=get_lang()
        )
        st.markdown(t("wl_bucket_intro", bucket=bucket_label))
        if fig_out is not None:
            st.plotly_chart(fig_out, width="stretch", config={"displayModeBar": False})
        else:
            st.info(t("wl_insufficient_evolution"))
        return bucket_label
    except Exception as e:
        st.warning(t("wl_cannot_display_evolution", error=e))
        return t("wl_period_default")


@fragment_if_available
def _render_map_mode_breakdown(dff: pl.DataFrame) -> None:
    """Affiche les résultats par carte et mode (Sprint 5.4)."""
    st.divider()
    st.subheader(t("wl_results_by_map_mode"))
    col_by_map, col_by_mode = st.columns(2)

    with col_by_map:
        st.markdown(f"##### {t('wl_by_map')}")
        if "map_name" in dff.columns and "outcome" in dff.columns:
            try:
                fig_map = plot_stacked_outcomes_by_category(
                    dff,
                    "map_name",
                    title=None,
                    min_matches=2,
                    sort_by="total",
                    max_categories=12,
                    lang=get_lang(),
                )
                if fig_map is not None:
                    st.plotly_chart(fig_map, width="stretch", config={"staticPlot": True})
                else:
                    st.info(t("insufficient_data_chart"))
            except Exception as e:
                st.warning(t("error_chart", error=e))
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
            try:
                fig_mode = plot_stacked_outcomes_by_category(
                    dff,
                    mode_col,
                    title=None,
                    min_matches=2,
                    sort_by="total",
                    max_categories=10,
                    lang=get_lang(),
                )
                if fig_mode is not None:
                    st.plotly_chart(fig_mode, width="stretch", config={"staticPlot": True})
                else:
                    st.info(t("insufficient_data_chart"))
            except Exception as e:
                st.warning(t("error_chart", error=e))
        else:
            st.info(t("insufficient_data_chart"))


@fragment_if_available
def _render_heatmap_section(dff: pl.DataFrame) -> None:
    """Affiche la heatmap Win Rate par jour et heure."""
    st.divider()
    st.subheader(t("wl_heatmap_title"))
    st.caption(t("wl_heatmap_caption"))
    if "start_time" in dff.columns and "outcome" in dff.columns:
        try:
            fig_heat = plot_win_ratio_heatmap(dff, title=None, min_matches=1, lang=get_lang())
            if fig_heat is not None:
                st.plotly_chart(fig_heat, width="stretch", config={"staticPlot": True})
            else:
                st.info(t("insufficient_data_chart"))
        except Exception as e:
            st.warning(t("error_chart", error=e))
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
    try:
        rank_col = "rank" if "rank" in dff.columns else "outcome"
        fig_top = plot_matches_at_top_by_week(
            dff,
            title=None,
            rank_col=rank_col,
            top_n_ranks=1,
            lang=get_lang(),
        )
        if fig_top is not None:
            st.plotly_chart(fig_top, width="stretch", config={"displayModeBar": False})
        else:
            st.info(t("insufficient_data_chart"))
    except Exception as e:
        st.warning(t("error_chart", error=e))


@fragment_if_available
def _render_streak_section(dff: pl.DataFrame) -> None:
    """Affiche les séries de victoires/défaites (Sprint 7.2)."""
    st.divider()
    st.subheader(t("wl_streaks"))
    st.caption(t("wl_streaks_caption"))
    if "outcome" in dff.columns and "start_time" in dff.columns:
        try:
            fig_streak = plot_streak_chart(dff, title=None, lang=get_lang())
            if fig_streak is not None:
                st.plotly_chart(fig_streak, width="stretch", config={"displayModeBar": False})
            else:
                st.info(t("insufficient_data_chart"))
        except Exception as e:
            st.warning(t("error_chart", error=e))
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
        st.plotly_chart(fig_ps, width="stretch", config={"displayModeBar": False})
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
            x = float(v)  # noqa: F841
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
def _render_ratio_by_map_section(
    dff: pl.DataFrame,
    base: pl.DataFrame,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
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

    metric = st.selectbox(
        t("wl_metric_label"),
        options=[
            ("ratio_global", t("wl_metric_ratio")),
            ("win_rate", t("wl_metric_win_rate")),
            ("accuracy_avg", t("wl_metric_accuracy")),
        ],
        format_func=lambda x: x[1],
    )
    key, label = metric

    view = breakdown.head(20).reverse()
    try:
        if key == "ratio_global":
            fig = plot_map_ratio_with_winloss(view, title=label)
        else:
            fig = plot_map_comparison(view, key, title=label)

        if fig is not None:
            if key in ("win_rate",):
                fig.update_xaxes(tickformat=".0%")
            if key in ("accuracy_avg",):
                fig.update_xaxes(ticksuffix="%")
            st.plotly_chart(fig, width="stretch", config={"staticPlot": True})
        else:
            st.info(t("insufficient_data_chart"))
    except Exception as e:
        st.warning(t("error_chart", error=e))

    base_scope = base_scope_pl
    _render_map_table(breakdown, base_scope)


def _render_map_table(breakdown: pl.DataFrame, base_scope: pl.DataFrame) -> None:
    """Affiche le tableau détaillé par carte."""
    from src.ui.translations import translate_playlist_name

    # Transformations internes en Polars
    cols_expr = [
        (pl.col("win_rate") * 100).round(1).alias("win_rate"),
        (pl.col("loss_rate") * 100).round(1).alias("loss_rate"),
        pl.col("accuracy_avg").cast(pl.Float64, strict=False).round(2).alias("accuracy_avg"),
        pl.col("ratio_global").cast(pl.Float64, strict=False).round(2).alias("ratio_global"),
    ]
    if "performance_avg" in breakdown.columns:
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
        if len(vals) == 0:
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
        # Vectorisation: build_mapping + replace_strict au lieu de map_elements
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
        # Vectorisation: build_mapping + replace_strict au lieu de map_elements
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
    rename_map = {
        "map_name": t("wl_col_map"),
        "matches": t("wl_col_matches"),
        "accuracy_avg": t("wl_col_accuracy_avg"),
        "performance_avg": t("wl_col_performance_avg"),
        "win_rate": t("wl_col_win_rate"),
        "loss_rate": t("wl_col_loss_rate"),
        "ratio_global": t("wl_col_ratio"),
        "playlist_ctx": t("wl_col_playlist"),
        "mode_ctx": t("wl_col_mode"),
    }
    tbl = tbl.rename({k: v for k, v in rename_map.items() if k in tbl.columns})

    _col_map = t("wl_col_map")
    _col_playlist = t("wl_col_playlist")
    _col_mode = t("wl_col_mode")
    _col_matches = t("wl_col_matches")
    _col_acc = t("wl_col_accuracy_avg")
    _col_perf = t("wl_col_performance_avg")
    _col_win = t("wl_col_win_rate")
    _col_loss = t("wl_col_loss_rate")
    _col_ratio = t("wl_col_ratio")
    ordered_cols = [
        _col_map,
        _col_playlist,
        _col_mode,
        _col_matches,
        _col_acc,
        _col_perf,
        _col_win,
        _col_loss,
        _col_ratio,
    ]
    tbl = tbl.select([c for c in ordered_cols if c in tbl.columns])

    # Conversion pandas à la frontière .style
    tbl_pd = tbl.to_pandas()
    tbl_styled = tbl_pd.style.apply(_style_map_table_row, axis=1)
    st.dataframe(
        tbl_styled,
        width="stretch",
        hide_index=True,
        column_config={
            _col_matches: st.column_config.NumberColumn(_col_matches, format="%d"),
            _col_acc: st.column_config.NumberColumn(_col_acc, format="%.2f"),
            _col_perf: st.column_config.NumberColumn(_col_perf, format="%.1f"),
            _col_win: st.column_config.NumberColumn(_col_win, format="%.1f"),
            _col_loss: st.column_config.NumberColumn(_col_loss, format="%.1f"),
            _col_ratio: st.column_config.NumberColumn(_col_ratio, format="%.2f"),
        },
    )
