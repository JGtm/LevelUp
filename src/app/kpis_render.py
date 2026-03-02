"""Rendu des KPIs extraits de main() pour simplification.

Ce module gère:
- Le calcul des métriques de base (win rate, KPIs)
- Le rendu du bandeau résumé
- Le rendu des cartes KPI carrière
"""

from __future__ import annotations

from typing import TYPE_CHECKING

import polars as pl
import streamlit as st

from src.analysis import (
    compute_aggregated_stats,
    compute_global_ratio,
    compute_outcome_rates,
)
from src.analysis.stats import format_mmss
from src.app.helpers import (
    avg_match_duration_seconds,
    compute_total_play_seconds,
)
from src.ui.components import (
    render_kpi_cards,
    render_top_summary,
)
from src.ui.formatting import (
    format_duration_dhm,
    format_duration_hms,
)
from src.ui.i18n import get_lang, t
from src.utils.polars_compat import ensure_polars as _to_polars

if TYPE_CHECKING:
    pass


def render_kpis_section(dff: pl.DataFrame) -> None:
    """Rend la section complète des KPIs.

    Args:
        dff: DataFrame (Pandas ou Polars) filtré des matchs.
    """
    from src.ui.perf import perf_section

    dff_pl = _to_polars(dff)

    with perf_section("kpis"):
        rates = compute_outcome_rates(dff_pl)
        total_outcomes = max(1, rates.total)
        win_rate = rates.wins / total_outcomes
        loss_rate = rates.losses / total_outcomes

        avg_acc = None
        if not dff_pl.is_empty() and "accuracy" in dff_pl.columns:
            avg_acc = dff_pl.select(pl.col("accuracy").drop_nulls().mean()).item()
        global_ratio = compute_global_ratio(dff_pl)
        avg_life = None
        if not dff_pl.is_empty() and "average_life_seconds" in dff_pl.columns:
            avg_life = dff_pl.select(pl.col("average_life_seconds").drop_nulls().mean()).item()

    # Durées
    avg_match_seconds = avg_match_duration_seconds(dff_pl)
    total_play_seconds = compute_total_play_seconds(dff_pl)
    avg_match_txt = format_duration_hms(avg_match_seconds)
    total_play_txt = format_duration_dhm(total_play_seconds)

    # Stats par minute / totaux
    stats = compute_aggregated_stats(dff_pl)

    # Moyennes par partie
    kpg = dff_pl.select(pl.col("kills").mean()).item() if not dff_pl.is_empty() else None
    dpg = dff_pl.select(pl.col("deaths").mean()).item() if not dff_pl.is_empty() else None
    apg = dff_pl.select(pl.col("assists").mean()).item() if not dff_pl.is_empty() else None

    # Rendu des sections
    st.subheader(t("kpi_matches_header"))
    render_top_summary(len(dff_pl), rates)
    render_kpi_cards(
        [
            (t("kpi_avg_duration"), avg_match_txt),
            (t("kpi_total_duration"), total_play_txt),
        ]
    )

    st.subheader(t("kpi_career_header"))
    render_kpi_cards(
        [
            (t("kpi_avg_duration"), avg_match_txt),
            (t("kpi_kills_per_match"), f"{kpg:.2f}" if kpg is not None else "-"),
            (t("kpi_deaths_per_match"), f"{dpg:.2f}" if dpg is not None else "-"),
            (
                t("kpi_assists_per_match"),
                f"{apg:.2f}" if apg is not None else "-",
            ),
        ],
        dense=False,
    )
    render_kpi_cards(
        [
            (
                t("kpi_kills_per_min"),
                f"{stats.kills_per_minute:.2f}" if stats.kills_per_minute else "-",
            ),
            (
                t("kpi_deaths_per_min"),
                f"{stats.deaths_per_minute:.2f}" if stats.deaths_per_minute else "-",
            ),
            (
                t("kpi_assists_per_min"),
                f"{stats.assists_per_minute:.2f}" if stats.assists_per_minute else "-",
            ),
            (t("kpi_avg_accuracy"), f"{avg_acc:.2f}%" if avg_acc is not None else "-"),
            (t("kpi_avg_lifespan"), format_mmss(avg_life)),
            (t("kpi_win_rate"), f"{win_rate*100:.1f}%" if rates.total else "-"),
            (t("kpi_loss_rate"), f"{loss_rate*100:.1f}%" if rates.total else "-"),
            (t("kpi_ratio"), f"{global_ratio:.2f}" if global_ratio is not None else "-"),
        ],
        dense=False,
    )


def render_performance_info() -> None:
    """Rend l'expander d'explication du score de performance."""
    from src.analysis.performance_config import get_performance_full_desc

    lang = get_lang()
    with st.expander(t("exp_perf_score_info"), expanded=False):
        st.markdown(get_performance_full_desc(lang))
