"""Graphiques pour la page Match View - Expected vs Actual."""

from __future__ import annotations

import logging
from typing import Any

import plotly.graph_objects as go
import polars as pl
import streamlit as st
from plotly.subplots import make_subplots

from src.analysis.stats import compute_mode_category_averages, extract_mode_category, format_mmss
from src.config import HALO_COLORS
from src.ui.chart_utils import safe_chart_render
from src.ui.i18n import t
from src.ui.pages.match_view_helpers import os_card
from src.ui.streamlit_modern import PLOTLY_STATIC_CONFIG, fragment_if_available
from src.ui.vectorize_helpers import build_mapping
from src.visualization._compat import DataFrameLike, ensure_polars
from src.visualization.theme import apply_halo_plot_style, get_legend_horizontal_bottom

logger = logging.getLogger(__name__)


def _safe_numeric(value: Any) -> float:
    """Convertit une valeur en float, retourne NaN si impossible."""
    if value is None:
        return float("nan")
    try:
        v = float(value)
        return v
    except (ValueError, TypeError):
        return float("nan")


# =============================================================================
# Graphiques Expected vs Actual
# =============================================================================


@fragment_if_available
def render_expected_vs_actual(  # noqa: C901, PLR0912, PLR0913, PLR0915
    row: dict[str, Any],
    pm: dict,
    colors: dict,
    df_full: DataFrameLike | None = None,
    *,
    db_path: str | None = None,
    xuid: str | None = None,
) -> None:
    """Rend la section Réel vs Attendu avec moyenne historique par catégorie de mode."""
    logger.debug("charts: rendu expected vs actual, xuid=%s", xuid)
    # Normaliser df_full en Polars
    if df_full is not None:
        df_full = ensure_polars(df_full)
    team_mmr = pm.get("team_mmr")
    enemy_mmr = pm.get("enemy_mmr")
    delta_mmr = (team_mmr - enemy_mmr) if (team_mmr is not None and enemy_mmr is not None) else None

    mmr_cols = st.columns(3)
    with mmr_cols[0]:
        os_card(t("mvc_mmr_team"), f"{team_mmr:.1f}" if team_mmr is not None else "-")
    with mmr_cols[1]:
        os_card(t("mvc_mmr_enemy"), f"{enemy_mmr:.1f}" if enemy_mmr is not None else "-")
    with mmr_cols[2]:
        if delta_mmr is None:
            os_card(t("mvc_mmr_gap"), "-")
        else:
            dm = float(delta_mmr)
            col = (
                "#4CAF50"  # --color-win
                if dm > 0
                else ("#F44336" if dm < 0 else "#9E9E9E")  # --color-loss / --color-tie
            )
            os_card(t("mvc_mmr_gap"), f"{dm:+.1f}", accent=col, kpi_color=col)

    def _ev_card(title: str, perf: dict, *, mode: str) -> None:
        count = perf.get("count")
        expected = perf.get("expected")

        # Si count est disponible mais expected est None (DuckDB v4), afficher quand même la valeur réelle
        if count is None:
            os_card(title, "-", "")
            return

        # Si expected est None, afficher seulement la valeur réelle sans comparaison
        if expected is None:
            os_card(title, f"{float(count):.0f}", t("mvc_actual_only"))
            return

        delta = float(count) - float(expected)
        if delta == 0:
            delta_class = "text-neutral"
        else:
            good = delta > 0
            if mode == "inverse":
                good = not good
            delta_class = "text-positive" if good else "text-negative"

        arrow = "↑" if delta > 0 else ("↓" if delta < 0 else "→")
        sub = f"<span class='{delta_class} fw-bold'>{arrow} {delta:+.1f}</span>"
        os_card(title, f"{float(count):.0f} vs {float(expected):.1f}", sub)

    perf_k = pm.get("kills") or {}
    perf_d = pm.get("deaths") or {}
    perf_a = pm.get("assists") or {}

    st.subheader(t("mv_vs_expected"))
    av_cols = st.columns(3)
    with av_cols[0]:
        _ev_card(t("tm_kills"), perf_k, mode="normal")
    with av_cols[1]:
        _ev_card(t("tm_deaths"), perf_d, mode="inverse")
    with av_cols[2]:
        avg_life_last = row.get("average_life_seconds")
        os_card(t("col_avg_life_long"), format_mmss(avg_life_last), "")

    # Calculer la moyenne historique par catégorie de mode
    mode_category = extract_mode_category(row.get("pair_name"))
    hist_avgs: dict[str, float | None] = {
        "avg_kills": None,
        "avg_deaths": None,
        "avg_assists": None,
        "avg_ratio": None,
        "match_count": 0,
    }
    if df_full is not None and len(df_full) >= 10:
        hist_avgs = compute_mode_category_averages(df_full, mode_category)

    # Graphique F / D / A
    labels = [t("mvc_lbl_k"), t("mvc_lbl_d"), t("mvc_lbl_a")]
    actual_vals = [
        float(row.get("kills") or 0.0),
        float(row.get("deaths") or 0.0),
        float(row.get("assists") or 0.0),
    ]
    exp_vals = [
        perf_k.get("expected"),
        perf_d.get("expected"),
        perf_a.get("expected"),
    ]
    hist_vals = [
        hist_avgs.get("avg_kills"),
        hist_avgs.get("avg_deaths"),
        hist_avgs.get("avg_assists"),
    ]

    real_ratio = row.get("ratio")
    try:
        real_ratio_f = float(real_ratio) if real_ratio == real_ratio else None
    except Exception:
        logger.warning("charts: conversion ratio impossible, valeur=%s", real_ratio)
        real_ratio_f = None
    if real_ratio_f is None:
        denom = max(1.0, float(row.get("deaths") or 0.0))
        real_ratio_f = (float(row.get("kills") or 0.0) + float(row.get("assists") or 0.0)) / denom

    exp_fig = make_subplots(specs=[[{"secondary_y": True}]])

    bar_colors = [HALO_COLORS.green, HALO_COLORS.red, HALO_COLORS.cyan]

    # Barres "Réel" en premier (pleines)
    exp_fig.add_trace(
        go.Bar(
            x=labels,
            y=actual_vals,
            name=t("lbl_actual"),
            marker_color=bar_colors,
            opacity=0.90,
            hovertemplate=f"%{{x}} ({t('lbl_actual')}): %{{y:.0f}}<extra></extra>",
        ),
        secondary_y=False,
    )

    # Barres "Attendu" uniquement si données disponibles
    has_expected = any(v is not None for v in exp_vals)
    if has_expected:
        # Couleurs plus visibles pour les barres "Attendu" (bordure + remplissage + hachuré)
        exp_colors = ["rgba(0, 255, 128, 0.6)", "rgba(255, 80, 80, 0.6)", "rgba(0, 200, 255, 0.6)"]
        exp_fig.add_trace(
            go.Bar(
                x=labels,
                y=exp_vals,
                name=t("lbl_expected"),
                marker={
                    "color": exp_colors,
                    "line": {"color": bar_colors, "width": 2},
                    "pattern": {"shape": "/", "solidity": 0.4, "fgcolor": bar_colors},
                },
                hovertemplate=f"%{{x}} ({t('lbl_expected')}): %{{y:.1f}}<extra></extra>",
            ),
            secondary_y=False,
        )

    # Moyenne historique par catégorie (si disponible) -> en barres
    if hist_avgs.get("match_count", 0) >= 10:
        exp_fig.add_trace(
            go.Bar(
                x=labels,
                y=hist_vals,
                name=t("mvc_hist_avg", category=mode_category, n=hist_avgs["match_count"]),
                marker={
                    "color": bar_colors,
                    "pattern": {
                        "shape": ".",
                        "fgcolor": "rgba(255,255,255,0.75)",
                        "solidity": 0.10,
                    },
                },
                opacity=0.35,
                hovertemplate=f"%{{x}} ({t('mvc_hist_avg', category=mode_category, n=hist_avgs['match_count'])}): %{{y:.1f}}<extra></extra>",
            ),
            secondary_y=False,
        )

    exp_fig.update_layout(
        barmode="group",
        height=360,
        margin={"l": 40, "r": 20, "t": 50, "b": 90},  # Augmenter le top margin pour l'annotation
        legend=get_legend_horizontal_bottom(),
        annotations=[
            {
                "x": 1.0,  # Position à droite
                "y": 1.05,  # Au-dessus du graphique
                "xref": "paper",
                "yref": "paper",
                "text": f"{t('mvc_fda_ratio')}: <b>{real_ratio_f:.2f}</b>",
                "showarrow": False,
                "font": {"size": 14, "color": HALO_COLORS.amber},
                "bgcolor": "rgba(0,0,0,0.5)",
                "bordercolor": HALO_COLORS.amber,
                "borderwidth": 1,
                "borderpad": 4,
            }
        ],
    )
    exp_fig.update_yaxes(title_text=t("mvc_fda_title"), rangemode="tozero", secondary_y=False)
    exp_fig.update_yaxes(visible=False, secondary_y=True)

    # K/D/A et Folie meurtrière sur la même rangée
    chart_cols = st.columns(2)
    with chart_cols[0], safe_chart_render():
        if exp_fig is not None:
            st.plotly_chart(exp_fig, width="stretch", config=PLOTLY_STATIC_CONFIG)
        else:
            st.info(t("insufficient_data_chart"))
    with chart_cols[1]:
        _render_spree_headshots(
            row,
            df_full=df_full,
            db_path=db_path,
            xuid=xuid,
        )


def _render_spree_headshots(
    row: dict[str, Any],
    df_full: DataFrameLike | None = None,
    *,
    db_path: str | None = None,
    xuid: str | None = None,
) -> None:
    """Rend le graphique folie meurtrière / tirs à la tête / frags parfaits.

    Ajoute, si possible, une série de barres correspondant à la moyenne
    historique sur la même catégorie custom (alignée sidebar).
    Les frags parfaits sont comptés via les médailles Perfect (medals_earned).
    """
    # Normaliser df_full en Polars
    if df_full is not None:
        df_full = ensure_polars(df_full)

    spree_v = _safe_numeric(row.get("max_killing_spree"))
    headshots_v = _safe_numeric(row.get("headshot_kills"))
    mode_category = extract_mode_category(row.get("pair_name"))
    hist_avgs: dict[str, float | None] = {
        "avg_max_killing_spree": None,
        "avg_headshot_kills": None,
        "avg_perfect_kills": None,
        "match_count": 0,
    }
    if df_full is not None and len(df_full) >= 10:
        hist_avgs_full = compute_mode_category_averages(df_full, mode_category)
        hist_avgs["avg_max_killing_spree"] = hist_avgs_full.get("avg_max_killing_spree")
        hist_avgs["avg_headshot_kills"] = hist_avgs_full.get("avg_headshot_kills")
        hist_avgs["match_count"] = hist_avgs_full.get("match_count", 0)

    # Frags parfaits du match courant et moyenne historique (médailles Perfect)
    perfect_current = 0
    if db_path and xuid and str(db_path).endswith(".duckdb"):
        try:
            from src.data.repositories.duckdb_repo import DuckDBRepository

            repo = DuckDBRepository(db_path, str(xuid).strip())
            match_id = str(row.get("match_id", "")).strip()
            if match_id:
                counts = repo.count_perfect_kills_by_match([match_id])
                perfect_current = counts.get(match_id, 0)
            # Moyenne historique par catégorie (même filtre que spree/headshots)
            if df_full is not None and hist_avgs.get("match_count", 0) >= 10:
                _cat_map = build_mapping(df_full.get_column("pair_name"), extract_mode_category)
                filtered = df_full.filter(
                    pl.col("pair_name")
                    .cast(pl.Utf8)
                    .replace_strict(_cat_map, default="Other", return_dtype=pl.Utf8)
                    == mode_category
                )
                if len(filtered) >= 10:
                    match_ids = filtered["match_id"].cast(pl.Utf8).to_list()
                    perfect_counts = repo.count_perfect_kills_by_match(match_ids)
                    total = sum(perfect_counts.get(mid, 0) for mid in match_ids)
                    hist_avgs["avg_perfect_kills"] = total / len(match_ids)
        except Exception:
            logger.warning(
                "charts: erreur chargement perfect kills, match=%s",
                row.get("match_id"),
                exc_info=True,
            )

    has_spree_or_hs = (spree_v == spree_v) or (headshots_v == headshots_v)
    if has_spree_or_hs or (db_path and xuid):
        st.subheader(t("ts_spree"))
        fig_sh = go.Figure()

        x_labels = [t("tm_killing_spree"), t("tm_headshots"), t("tm_perfect_kills")]
        bar_colors = [HALO_COLORS.violet, HALO_COLORS.cyan, HALO_COLORS.green]
        real_vals = [
            float(spree_v) if (spree_v == spree_v) else 0.0,
            float(headshots_v) if (headshots_v == headshots_v) else 0.0,
            float(perfect_current),
        ]
        fig_sh.add_trace(
            go.Bar(
                x=x_labels,
                y=real_vals,
                name=t("lbl_actual"),
                marker_color=bar_colors,
                opacity=0.85,
                hovertemplate=f"%{{x}} ({t('lbl_actual')}): %{{y:.0f}}<extra></extra>",
            )
        )

        if hist_avgs.get("match_count", 0) >= 10:
            avg_perfect = hist_avgs.get("avg_perfect_kills")
            hist_vals = [
                float(hist_avgs.get("avg_max_killing_spree") or 0.0),
                float(hist_avgs.get("avg_headshot_kills") or 0.0),
                float(avg_perfect) if avg_perfect is not None else 0.0,
            ]
            fig_sh.add_trace(
                go.Bar(
                    x=x_labels,
                    y=hist_vals,
                    name=t("mvc_hist_avg", category=mode_category, n=hist_avgs["match_count"]),
                    marker={
                        "color": bar_colors,
                        "pattern": {
                            "shape": ".",
                            "fgcolor": "rgba(255,255,255,0.75)",
                            "solidity": 0.10,
                        },
                    },
                    opacity=0.35,
                    hovertemplate=f"%{{x}} ({t('mvc_hist_avg', category=mode_category, n=hist_avgs['match_count'])}): %{{y:.1f}}<extra></extra>",
                )
            )

        fig_sh.update_layout(
            barmode="group",
            height=260,
            margin={"l": 40, "r": 20, "t": 30, "b": 60},
            legend=get_legend_horizontal_bottom(),
        )
        fig_sh.update_yaxes(rangemode="tozero")
        with safe_chart_render():
            styled_fig = apply_halo_plot_style(fig_sh, height=260)
            if styled_fig is not None:
                st.plotly_chart(styled_fig, width="stretch", config=PLOTLY_STATIC_CONFIG)
            else:
                st.info(t("insufficient_data_chart"))


# =============================================================================
# Exports publics
# =============================================================================

__all__ = [
    "render_expected_vs_actual",
]
