"""Page Victoires/Défaites.

Analyse des victoires et défaites par période et par carte.
"""

from __future__ import annotations

import polars as pl
import streamlit as st

from src.config import HALO_COLORS
from src.data.services.win_loss_service import WinLossService
from src.ui.chart_utils import safe_chart_render
from src.ui.i18n import get_lang, t
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, PLOTLY_STATIC_CONFIG, fragment_if_available
from src.ui.tz import get_tz_name
from src.visualization import (
    plot_map_comparison,
    plot_map_outcome_timeline,
    plot_map_perf_vs_history,
    plot_map_winrate_bullet,
    plot_matches_at_top_by_week,
    plot_metric_bars_by_match,
    plot_outcomes_over_time,
    plot_stacked_outcomes_by_category,
    plot_streak_chart,
    plot_win_ratio_heatmap,
)
from src.visualization._compat import DataFrameLike, ensure_polars
from src.visualization._plot_options import PlotOptions


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

        _render_outcomes_over_time(dff, is_session_scope)
        _render_map_mode_breakdown(dff)
        _render_heatmap_section(dff)
        _render_top_by_week(dff)
        _render_streak_section(dff)
        _render_personal_score_section(dff)
        _render_winrate_perf_vs_history(dff, base)
        # _render_ratio_by_map_section(dff, base, is_session_scope)  # DISABLED


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
            _wl_map_col = "map_ui" if "map_ui" in dff.columns else "map_name"
            with safe_chart_render():
                fig_map = plot_stacked_outcomes_by_category(
                    dff,
                    _wl_map_col,
                    min_matches=1,
                    sort_by="total",
                    max_categories=12,
                    opts=PlotOptions(lang=get_lang()),
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
                    min_matches=1,
                    sort_by="total",
                    max_categories=10,
                    opts=PlotOptions(lang=get_lang()),
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
            fig_heat = plot_win_ratio_heatmap(dff, min_matches=1, lang=get_lang())
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
            fig_streak = plot_streak_chart(dff, lang=get_lang())
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


@fragment_if_available
def _render_winrate_perf_vs_history(dff: pl.DataFrame, base: pl.DataFrame) -> None:
    """Affiche les graphes Taux de victoires vs historique et Performance vs historique."""
    from src.analysis import compute_map_breakdown

    # Assurer cohérence map_ui : dff a map_ui (FR via _add_derived_columns), base brut n'en a pas.
    # Sans cela l'inner join sur map_name dans _prepare_bullet_joined_data échoue pour les 4 cartes
    # dont le nom FR diffère du nom EN (Cliffhanger, Fortress, Nemesis, The Pit).
    if "map_ui" not in base.columns and "map_name_fr" in base.columns and get_lang() == "fr":
        base = base.with_columns(
            pl.coalesce(
                [pl.col("map_name_fr").cast(pl.Utf8), pl.col("map_name").cast(pl.Utf8)]
            ).alias("map_ui")
        )

    bd_current = compute_map_breakdown(dff, df_history=base)
    bd_history = compute_map_breakdown(base)
    if bd_current.is_empty() or bd_history.is_empty():
        return

    lang = get_lang()
    _wl_order_col = "map_ui" if "map_ui" in dff.columns else "map_name"
    map_order: list[str] | None = None
    if "start_time" in dff.columns:
        map_order = (
            dff.sort("start_time")
            .unique(subset=[_wl_order_col], keep="first", maintain_order=True)
            .filter(pl.col(_wl_order_col).is_not_null())[_wl_order_col]
            .to_list()
        )

    view = bd_current.head(20).reverse()

    st.divider()
    st.markdown(f"##### {t('wl_map_bullet_title')}")
    st.caption(t("wl_map_bullet_caption"))
    with safe_chart_render():
        fig_bullet = plot_map_winrate_bullet(view, bd_history, lang=lang, map_order=map_order)
        if fig_bullet is not None:
            st.plotly_chart(fig_bullet, width="stretch", config=PLOTLY_STATIC_CONFIG)

    st.markdown(f"##### {t('wl_perf_vs_history_title')}")
    with safe_chart_render():
        fig_perf = plot_map_perf_vs_history(view, bd_history, lang=lang, map_order=map_order)
        if fig_perf is not None:
            st.plotly_chart(fig_perf, width="stretch", config=PLOTLY_STATIC_CONFIG)


def _compute_session_map_order(dff: pl.DataFrame) -> list[str] | None:
    """Retourne l'ordre chronologique des cartes depuis les matchs d'une session."""
    if "start_time" not in dff.columns:
        return None
    _col = "map_ui" if "map_ui" in dff.columns else "map_name"
    return (
        dff.sort("start_time")
        .unique(subset=[_col], keep="first", maintain_order=True)
        .filter(pl.col(_col).is_not_null())[_col]
        .to_list()
    )


def _render_ratio_by_map_section(
    dff: pl.DataFrame,
    base: pl.DataFrame,
    is_session_scope: bool = False,
) -> None:
    """Affiche la section par carte : lollipop, timeline, bullet, perf vs historique."""
    st.divider()
    st.subheader(t("wl_ratio_by_map"))
    st.caption(t("wl_ratio_caption"))
    if "min_matches_maps" not in st.session_state:
        st.session_state["min_matches_maps"] = 1
    min_matches = st.slider(
        t("wl_min_matches_map_slider"),
        1,
        30,
        step=1,
        key="min_matches_maps",
        on_change=_clear_min_matches_maps_auto,
    )

    with st.spinner(t("wl_computing_map")):
        # Assurer cohérence map_ui : base brut n'a pas map_ui, dff l'a (FR).
        # Nécessaire pour que l'inner join map_name dans _prepare_bullet_joined_data
        # retrouve les 4 cartes dont le nom FR ≠ EN (Cliffhanger, Fortress, Nemesis, The Pit).
        if "map_ui" not in base.columns and "map_name_fr" in base.columns and get_lang() == "fr":
            base = base.with_columns(
                pl.coalesce(
                    [pl.col("map_name_fr").cast(pl.Utf8), pl.col("map_name").cast(pl.Utf8)]
                ).alias("map_ui")
            )
        map_result = WinLossService.compute_map_breakdown(dff, min_matches, df_history=base)
        breakdown_history = WinLossService.compute_map_breakdown(base, 1).breakdown

    if map_result.is_empty:
        st.warning(t("wl_not_enough_map"))
        return

    view = map_result.breakdown.head(20).reverse()
    session_ids = dff["match_id"].cast(pl.Utf8).to_list()
    lang = get_lang()

    # B — Timeline chronologique (DISABLED: désactivé temporairement, conserver le code)
    if False:  # noqa: SIM210
        st.markdown(f"##### {t('wl_map_timeline_title')}")
        st.caption(t("wl_map_timeline_caption"))
        with safe_chart_render():
            fig_tl = plot_map_outcome_timeline(base, session_match_ids=session_ids, lang=lang)
            if fig_tl is not None:
                st.plotly_chart(fig_tl, width="stretch", config=PLOTLY_CLEAN_CONFIG)
            else:
                st.info(t("insufficient_data_chart"))

    if is_session_scope:
        return

    # C — Bullet win rate vs historique
    st.markdown(f"##### {t('wl_map_bullet_title')}")
    st.caption(t("wl_map_bullet_caption"))
    with safe_chart_render():
        fig_bullet = plot_map_winrate_bullet(view, breakdown_history, lang=lang)
        if fig_bullet is not None:
            st.plotly_chart(fig_bullet, width="stretch", config=PLOTLY_STATIC_CONFIG)

    # Feature 2 — Performance vs historique
    st.markdown(f"##### {t('wl_perf_vs_history_title')}")
    with safe_chart_render():
        fig_perf = plot_map_perf_vs_history(view, breakdown_history, lang=lang)
        if fig_perf is not None:
            st.plotly_chart(fig_perf, width="stretch", config=PLOTLY_STATIC_CONFIG)

    _render_map_ratio_accuracy(view, lang)


def _render_map_ratio_accuracy(view: pl.DataFrame, lang: str) -> None:
    """Affiche les charts ratio F/M et précision par carte."""
    with safe_chart_render():
        st.subheader(t("wl_metric_ratio"))
        fig_r = plot_map_comparison(view, "ratio_global")
        st.plotly_chart(fig_r, width="stretch", config=PLOTLY_STATIC_CONFIG)
    with safe_chart_render():
        if view.filter(pl.col("accuracy_avg").is_not_null()).is_empty():
            return
        st.subheader(t("wl_metric_accuracy"))
        fig_a = plot_map_comparison(view, "accuracy_avg")
        st.plotly_chart(fig_a, width="stretch", config=PLOTLY_STATIC_CONFIG)
