"""Rendu des KPIs pour l'application Streamlit."""

from __future__ import annotations

import logging

import polars as pl
import streamlit as st

logger = logging.getLogger(__name__)

from src.analysis.stats import format_mmss
from src.app.kpis import compute_kpi_stats
from src.ui.components import render_kpi_cards, render_top_summary
from src.ui.formatting import format_duration_dhm, format_duration_hms
from src.ui.i18n import get_lang, t
from src.utils.polars_compat import ensure_polars as _to_polars


def render_kpis_section(dff: pl.DataFrame) -> None:
    """Rend la section complète des KPIs.

    Args:
        dff: DataFrame (Pandas ou Polars) filtré des matchs.
    """
    from src.ui.perf import perf_section

    dff_pl = _to_polars(dff)
    logger.debug("KPIs calculés: %d matchs", len(dff_pl))

    with perf_section("kpis"):
        kpis = compute_kpi_stats(dff_pl)

    avg_match_txt = format_duration_hms(kpis.avg_match_seconds)
    total_play_txt = format_duration_dhm(kpis.total_play_seconds)

    st.subheader(t("kpi_matches_header"))
    render_top_summary(len(dff_pl), kpis, avg_duration=avg_match_txt, total_duration=total_play_txt)

    st.subheader(t("kpi_career_header"))
    render_kpi_cards(
        [
            (
                t("kpi_kills_per_match"),
                f"{kpis.kills_per_game:.2f}" if kpis.kills_per_game is not None else "-",
            ),
            (
                t("kpi_deaths_per_match"),
                f"{kpis.deaths_per_game:.2f}" if kpis.deaths_per_game is not None else "-",
            ),
            (
                t("kpi_assists_per_match"),
                f"{kpis.assists_per_game:.2f}" if kpis.assists_per_game is not None else "-",
            ),
            (
                t("kpi_kills_per_min"),
                f"{kpis.kills_per_minute:.2f}" if kpis.kills_per_minute else "-",
            ),
            (
                t("kpi_deaths_per_min"),
                f"{kpis.deaths_per_minute:.2f}" if kpis.deaths_per_minute else "-",
            ),
            (
                t("kpi_assists_per_min"),
                f"{kpis.assists_per_minute:.2f}" if kpis.assists_per_minute else "-",
            ),
        ],
        dense=False,
    )
    render_kpi_cards(
        [
            (
                t("kpi_avg_accuracy"),
                f"{kpis.avg_accuracy:.2f}%" if kpis.avg_accuracy is not None else "-",
            ),
            (t("kpi_avg_lifespan"), format_mmss(kpis.avg_life_seconds)),
            (t("kpi_win_rate"), f"{kpis.win_rate * 100:.1f}%" if kpis.total_matches else "-"),
            (t("kpi_loss_rate"), f"{kpis.loss_rate * 100:.1f}%" if kpis.total_matches else "-"),
            (t("kpi_ratio"), f"{kpis.global_ratio:.2f}" if kpis.global_ratio is not None else "-"),
        ],
        dense=False,
    )


def render_performance_info() -> None:
    """Rend l'expander d'explication du score de performance."""
    from src.analysis.performance_config import get_performance_full_desc

    lang = get_lang()
    with st.expander(t("exp_perf_score_info"), expanded=False):
        st.markdown(get_performance_full_desc(lang))
