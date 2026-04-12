"""Page Synthèse — vue d'ensemble stratégique.

Agrège les graphes existants (map/mode, heatmap temporelle, activité hebdo)
et ajoute une comparaison Solo vs Escouade.
"""

from __future__ import annotations

import polars as pl
import streamlit as st

from src.data.domain.refdata import Outcome
from src.ui.components.browser_storage import hints_visible
from src.ui.i18n import t
from src.ui.pages.win_loss import (
    _render_heatmap_section,
    _render_map_mode_breakdown,
    _render_top_by_week,
)
from src.ui.streamlit_modern import fragment_if_available
from src.visualization._compat import DataFrameLike, ensure_polars


def _compute_group_stats(df: pl.DataFrame) -> tuple[float, float, float | None]:
    """Retourne (kd, winrate_pct, perf_moy) pour un groupe de matchs."""
    kills = df["kills"].fill_null(0).sum() if "kills" in df.columns else 0
    deaths = df["deaths"].fill_null(0).sum() if "deaths" in df.columns else 0
    kd = round(kills / deaths, 2) if deaths else float(kills)

    if "outcome" in df.columns and not df.is_empty():
        total = len(df)
        wins = int((df["outcome"] == Outcome.WIN).sum())
        winrate = round(wins / total * 100, 1)
    else:
        winrate = 0.0

    perf: float | None = None
    if "performance_score" in df.columns:
        vals = df["performance_score"].drop_nulls()
        if not vals.is_empty():
            perf = round(float(vals.mean()), 1)

    return kd, winrate, perf


def _render_group_column(label: str, count: int, kd: float, wr: float, perf: float | None) -> None:
    """Affiche les métriques d'un groupe dans une colonne."""
    st.markdown(f"#### {label} ({count})")
    st.metric("K/D", kd)
    st.metric(t("col_win_rate"), f"{wr}%")
    if perf is not None:
        st.metric(t("sc_performance_score"), perf)


@fragment_if_available
def _render_solo_squad_compare(dff: pl.DataFrame) -> None:
    """Affiche la comparaison Solo vs Escouade."""
    st.divider()
    st.subheader(t("syn_solo_squad_title"))
    if hints_visible():
        st.caption(t("syn_solo_squad_caption"))

    if "is_with_friends" not in dff.columns or dff.is_empty():
        st.info(t("syn_no_data"))
        return

    solo_df = dff.filter(pl.col("is_with_friends").fill_null(False).not_())
    squad_df = dff.filter(pl.col("is_with_friends").fill_null(False))

    if solo_df.is_empty() or squad_df.is_empty():
        st.info(t("syn_no_data"))
        return

    solo_kd, solo_wr, solo_perf = _compute_group_stats(solo_df)
    squad_kd, squad_wr, squad_perf = _compute_group_stats(squad_df)

    col_solo, col_squad = st.columns(2)
    with col_solo:
        _render_group_column(t("syn_solo"), len(solo_df), solo_kd, solo_wr, solo_perf)
    with col_squad:
        _render_group_column(t("syn_squad"), len(squad_df), squad_kd, squad_wr, squad_perf)


@fragment_if_available
def render_synthesis_page(
    dff: DataFrameLike,
    base: DataFrameLike,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
) -> None:
    """Affiche la page Synthèse : résultats, heatmap, activité, solo vs escouade."""
    dff_pl = ensure_polars(dff)
    if dff_pl.is_empty():
        st.warning(t("no_matches"))
        return

    _render_map_mode_breakdown(dff_pl)
    _render_heatmap_section(dff_pl)
    _render_top_by_week(dff_pl)
    _render_solo_squad_compare(dff_pl)
