"""Sections distributions et corrélations de la page Séries temporelles.

Extraites de timeseries.py pour respecter la limite de 500L par module.
"""

from __future__ import annotations

import polars as pl
import streamlit as st

from src.config import HALO_COLORS
from src.data.services.timeseries_service import TimeseriesService
from src.ui.chart_utils import safe_chart_render
from src.ui.i18n import t
from src.ui.streamlit_modern import PLOTLY_CLEAN_CONFIG, PLOTLY_STATIC_CONFIG, fragment_if_available
from src.visualization.distributions import (
    plot_correlation_scatter,
    plot_histogram,
)

# =============================================================================
# Distributions (histogrammes)
# =============================================================================


@fragment_if_available
def render_distributions(dff: pl.DataFrame, lang: str = "fr") -> None:
    """Affiche les distributions statistiques (Sprint 5.4.3 + Sprint 6)."""
    st.subheader(t("ts_distributions"))
    st.caption(t("ts_distributions_caption"))

    colors = HALO_COLORS.as_dict()
    _render_distribution_row1(dff, colors, lang=lang)
    _render_distribution_row2(dff, colors, lang=lang)
    _render_distribution_row3(dff, colors, lang=lang)


def _render_distribution_row1(dff: pl.DataFrame, colors: dict, lang: str = "fr") -> None:
    """Ligne 1 : précision + kills."""
    col1, col2 = st.columns(2)
    with col1:
        _render_single_histogram(
            dff,
            "accuracy",
            t("ts_dist_accuracy_title"),
            t("ts_accuracy_label"),
            colors["cyan"],
            lang=lang,
        )
    with col2:
        _render_single_histogram(
            dff,
            "kills",
            t("ts_dist_kills_title"),
            t("ts_kills_label"),
            colors["green"],
            lang=lang,
        )


def _render_distribution_row2(dff: pl.DataFrame, colors: dict, lang: str = "fr") -> None:
    """Ligne 2 : durée de vie + score de performance."""
    col3, col4 = st.columns(2)
    with col3:
        life_col = (
            "avg_life_seconds" if "avg_life_seconds" in dff.columns else "average_life_seconds"
        )
        _render_single_histogram(
            dff,
            life_col,
            t("ts_dist_life_title"),
            t("ts_life_label"),
            colors["amber"],
            lang=lang,
        )
    with col4:
        _render_single_histogram(
            dff,
            "performance_score",
            t("ts_dist_perf_title"),
            t("ts_score_label"),
            colors["violet"],
            lang=lang,
        )


def _render_distribution_row3(dff: pl.DataFrame, colors: dict, lang: str = "fr") -> None:
    """Ligne 3 : score/min + win rate glissant (Sprint 6)."""
    col5, col6 = st.columns(2)
    with col5:
        spm_data = TimeseriesService.compute_score_per_minute(dff)
        if spm_data.has_data:
            fig_spm = plot_histogram(
                spm_data.values,
                title=t("ts_dist_score_per_min_title"),
                x_label=t("ts_score_per_min_label"),
                y_label=t("ts_frequency_label"),
                show_kde=True,
                color=colors["amber"],
                lang=lang,
            )
            st.plotly_chart(fig_spm, width="stretch", config=PLOTLY_STATIC_CONFIG)
        elif "personal_score" not in dff.columns or "time_played_seconds" not in dff.columns:
            st.info(t("ts_col_missing_score_per_min"))
        else:
            st.info(t("ts_insufficient_score_per_min"))

    with col6:
        wr_data = TimeseriesService.compute_rolling_win_rate(dff)
        if wr_data.has_data:
            fig_wr = plot_histogram(
                wr_data.values,
                title=t("ts_dist_win_rate_title"),
                x_label=t("ts_win_rate_label"),
                y_label=t("ts_frequency_label"),
                show_kde=True,
                color=colors["green"],
                lang=lang,
            )
            st.plotly_chart(fig_wr, width="stretch", config=PLOTLY_STATIC_CONFIG)
        elif wr_data.missing_column:
            st.info(t("ts_col_missing_outcome"))
        elif wr_data.not_enough_matches:
            st.info(t("ts_insufficient_win_rate"))
        else:
            st.info(t("ts_insufficient_win_rate_dist"))


def _render_single_histogram(  # noqa: PLR0913
    dff: pl.DataFrame,
    column: str,
    title: str,
    x_label: str,
    color: str,
    min_data: int = 6,
    lang: str = "fr",
) -> None:
    """Affiche un histogramme simple pour une colonne donnée."""
    if column not in dff.columns:
        st.info(t("cannot_display"))
        return
    data = dff[column].drop_nulls()
    if len(data) > min_data - 1:
        fig = plot_histogram(
            data,
            title=title,
            x_label=x_label,
            y_label=t("ts_frequency_label"),
            show_kde=True,
            color=color,
            lang=lang,
        )
        st.plotly_chart(fig, width="stretch", config=PLOTLY_STATIC_CONFIG)
    elif len(data) == 0:
        st.info(t("no_data_filter"))
    else:
        st.info(t("ts_not_enough_dist", count=len(data), min=min_data))


# =============================================================================
# Corrélations (scatter plots)
# =============================================================================


@fragment_if_available
def render_correlations(dff: pl.DataFrame, lang: str = "fr") -> None:
    """Affiche les graphes de corrélation (Sprint 5.4.5 + Sprint 6)."""
    st.divider()
    st.subheader(t("ts_correlations"))
    st.caption(t("ts_correlations_caption"))

    _render_correlation_row1(dff, lang=lang)
    _render_correlation_row2(dff, lang=lang)
    _render_mmr_correlation(dff, lang=lang)


def _render_correlation_row1(dff: pl.DataFrame, lang: str = "fr") -> None:
    """Durée de vie vs Frags + Précision vs FDA."""
    col1, col2 = st.columns(2)
    life_col = "avg_life_seconds" if "avg_life_seconds" in dff.columns else "average_life_seconds"
    with col1:
        _render_scatter(
            dff,
            life_col,
            "kills",
            "outcome",
            t("ts_lifespan_vs_kills"),
            t("ts_lifespan_s"),
            t("ts_kills_label"),
            lang=lang,
        )
    with col2:
        _render_scatter(
            dff,
            "accuracy",
            "kda",
            "outcome",
            t("ts_accuracy_vs_kda"),
            t("ts_accuracy_label"),
            t("ts_fda"),
            lang=lang,
        )


def _render_correlation_row2(dff: pl.DataFrame, lang: str = "fr") -> None:
    """Durée de vie vs Morts + Frags vs Morts (Sprint 6)."""
    col3, col4 = st.columns(2)
    life_col = "avg_life_seconds" if "avg_life_seconds" in dff.columns else "average_life_seconds"
    with col3:
        _render_scatter(
            dff,
            life_col,
            "deaths",
            "outcome",
            t("ts_lifespan_vs_deaths"),
            t("ts_lifespan_s"),
            t("ts_deaths_label"),
            lang=lang,
        )
    with col4:
        _render_scatter(
            dff,
            "kills",
            "deaths",
            "outcome",
            t("ts_kills_vs_deaths"),
            t("ts_kills_label"),
            t("ts_deaths_label"),
            lang=lang,
        )


def _render_mmr_correlation(dff: pl.DataFrame, lang: str = "fr") -> None:
    """Team MMR vs Enemy MMR (Sprint 6)."""
    _render_scatter(
        dff,
        "team_mmr",
        "enemy_mmr",
        "outcome",
        t("ts_mmr_team_vs_enemy"),
        t("ts_mmr_team"),
        t("ts_mmr_enemy"),
        lang=lang,
    )


def _render_scatter(  # noqa: PLR0913
    dff: pl.DataFrame,
    x_col: str,
    y_col: str,
    color_col: str,
    title: str,
    x_label: str,
    y_label: str,
    min_data: int = 6,
    lang: str = "fr",
) -> None:
    """Affiche un scatter de corrélation avec validation des données."""
    if x_col not in dff.columns or y_col not in dff.columns:
        st.info(t("insufficient_data_chart"))
        return
    valid = dff.drop_nulls(subset=[x_col, y_col])
    if len(valid) <= min_data - 1:
        st.info(t("ts_not_enough_corr", count=len(valid), min=min_data))
        return
    with safe_chart_render():
        fig = plot_correlation_scatter(
            dff,
            x_col,
            y_col,
            color_col=color_col,
            title=title,
            x_label=x_label,
            y_label=y_label,
            show_trendline=True,
            lang=lang,
        )
        if fig is not None:
            st.plotly_chart(fig, width="stretch", config=PLOTLY_CLEAN_CONFIG)
        else:
            st.info(t("ts_corr_gen_error", title=title))
