"""Rendu des KPIs pour l'application Streamlit."""

from __future__ import annotations

import logging

import polars as pl
import streamlit as st

logger = logging.getLogger(__name__)

from src.analysis.stats import format_mmss
from src.app.kpis import KPIStats, compute_kpi_stats
from src.ui.components import render_combined_kpi_cards
from src.ui.formatting import format_duration_dhm, format_duration_hms
from src.ui.i18n import get_lang, t
from src.utils.polars_compat import ensure_polars as _to_polars


def _trend(
    current: float | None,
    reference: float | None,
    *,
    higher_is_better: bool = True,
    threshold: float = 0.08,
) -> str:
    """Retourne 'above', 'near', 'below' ou 'none' selon l'écart relatif au all-time."""
    if current is None or reference is None or reference == 0:
        return "none"
    ratio = current / reference
    if higher_is_better:
        if ratio >= 1 + threshold:
            return "above"
        if ratio <= 1 - threshold:
            return "below"
    else:
        if ratio <= 1 - threshold:
            return "above"
        if ratio >= 1 + threshold:
            return "below"
    return "near"


def _build_kpi_cards(kpis: KPIStats, kpis_all: KPIStats | None) -> list[dict]:
    """Construit la liste de dicts pour render_combined_kpi_cards."""
    avg_match_txt = format_duration_hms(kpis.avg_match_seconds)
    total_play_txt = format_duration_dhm(kpis.total_play_seconds)

    def tr(current: float | None, attr: str, *, higher_is_better: bool = True) -> str:
        ref = getattr(kpis_all, attr, None) if kpis_all else None
        return _trend(current, ref, higher_is_better=higher_is_better)

    return [
        {
            "label": t("kpi_selected_matches"),
            "main": str(kpis.total_matches),
            "sub": f"{avg_match_txt}/match" if avg_match_txt else None,
            "trend": "none",
        },
        {
            "label": t("kpi_total_duration"),
            "main": total_play_txt if total_play_txt else "-",
            "sub": None,
            "trend": "none",
        },
        {
            "label": t("kpi_kills_per_match"),
            "main": f"{kpis.kills_per_game:.2f}" if kpis.kills_per_game is not None else "-",
            "sub": f"{kpis.kills_per_minute:.2f}/min" if kpis.kills_per_minute else None,
            "trend": tr(kpis.kills_per_game, "kills_per_game"),
        },
        {
            "label": t("kpi_deaths_per_match"),
            "main": f"{kpis.deaths_per_game:.2f}" if kpis.deaths_per_game is not None else "-",
            "sub": f"{kpis.deaths_per_minute:.2f}/min" if kpis.deaths_per_minute else None,
            "trend": tr(kpis.deaths_per_game, "deaths_per_game", higher_is_better=False),
        },
        {
            "label": t("kpi_assists_per_match"),
            "main": f"{kpis.assists_per_game:.2f}" if kpis.assists_per_game is not None else "-",
            "sub": f"{kpis.assists_per_minute:.2f}/min" if kpis.assists_per_minute else None,
            "trend": tr(kpis.assists_per_game, "assists_per_game"),
        },
        {
            "label": t("kpi_avg_accuracy"),
            "main": f"{kpis.avg_accuracy:.2f}%" if kpis.avg_accuracy is not None else "-",
            "sub": None,
            "trend": tr(kpis.avg_accuracy, "avg_accuracy"),
        },
        {
            "label": t("kpi_avg_lifespan"),
            "main": format_mmss(kpis.avg_life_seconds),
            "sub": None,
            "trend": tr(kpis.avg_life_seconds, "avg_life_seconds"),
        },
        {
            "label": t("mv_results"),
            "wide": True,
            "bar": [
                ("#3DFF9A", kpis.wins, f"{kpis.wins} {t('kpi_wins')}"),
                ("#FF5C5C", kpis.losses, f"{kpis.losses} {t('kpi_losses')}"),
                ("#A855F7", kpis.ties, f"{kpis.ties} {t('kpi_ties')}"),
                (
                    "rgba(182,196,214,0.45)",
                    kpis.no_finish,
                    f"{kpis.no_finish} {t('kpi_no_finish')}",
                ),
            ]
            if kpis.total_matches
            else None,
            "trend": "none",
        },
    ]


def render_kpis_section(dff: pl.DataFrame, df_all: pl.DataFrame | None = None) -> None:
    """Rend la section complète des KPIs.

    Args:
        dff: DataFrame filtré des matchs (sélection courante).
        df_all: Tous les matchs du joueur (référence all-time pour le code couleur).
    """
    from src.ui.perf import perf_section

    dff_pl = _to_polars(dff)
    logger.debug("KPIs calculés: %d matchs", len(dff_pl))

    with perf_section("kpis"):
        kpis = compute_kpi_stats(dff_pl)

    kpis_all = None
    if df_all is not None:
        df_all_pl = _to_polars(df_all)
        if len(df_all_pl) > len(dff_pl):
            with perf_section("kpis_all"):
                kpis_all = compute_kpi_stats(df_all_pl)

    render_combined_kpi_cards(_build_kpi_cards(kpis, kpis_all))


def render_performance_info() -> None:
    """Rend l'expander d'explication du score de performance."""
    from src.analysis.performance_config import get_performance_full_desc

    lang = get_lang()
    with st.expander(t("exp_perf_score_info"), expanded=False):
        st.markdown(get_performance_full_desc(lang))
