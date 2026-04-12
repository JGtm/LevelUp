"""Page Synthèse — vue d'ensemble stratégique.

Agrège les graphes existants (map/mode, heatmap temporelle, activité hebdo)
et ajoute une comparaison Solo vs Escouade.
"""

from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime, timedelta, timezone

import plotly.graph_objects as go
import polars as pl
import streamlit as st

from src.config import HALO_COLORS
from src.data.domain.refdata import Outcome
from src.ui.components.browser_storage import hints_visible
from src.ui.i18n import t
from src.ui.pages.explorer_data import load_is_with_friends
from src.ui.pages.win_loss import (
    _render_heatmap_section,
    _render_map_mode_breakdown,
    _render_top_by_week,
)
from src.ui.streamlit_modern import PLOTLY_STATIC_CONFIG, fragment_if_available
from src.visualization._compat import DataFrameLike, ensure_polars

_PERIOD_KEYS = ["all", "2y", "1y", "1m", "1w"]
_PERIOD_OFFSETS = {"2y": 730, "1y": 365, "1m": 30, "1w": 7}


@dataclass(frozen=True)
class ComparisonMetric:
    """Métrique Solo vs Escouade prête à tracer."""

    label: str
    solo_value: float
    squad_value: float
    solo_text: str
    squad_text: str


def _render_period_selector() -> str:
    """Retourne la période locale de la page Synthèse."""
    selected = st.segmented_control(
        t("encounters_period_label"),
        options=_PERIOD_KEYS,
        format_func=lambda key: t(f"encounters_period_{key}"),
        default="all",
        key="synthesis_period",
    )
    return str(selected or "all")


def _filter_by_period(df: pl.DataFrame, period: str) -> pl.DataFrame:
    """Filtre un DataFrame de matchs selon une période locale."""
    if df.is_empty() or period == "all" or "start_time" not in df.columns:
        return df

    days = _PERIOD_OFFSETS.get(period)
    if days is None:
        return df

    since = datetime.now(timezone.utc).replace(tzinfo=None) - timedelta(days=days)
    return df.filter(pl.col("start_time") >= since)


def _attach_is_with_friends(df: pl.DataFrame, db_path: str, xuid: str) -> pl.DataFrame:
    """Ajoute le flag solo/escouade si absent du DataFrame de matchs."""
    if df.is_empty() or "is_with_friends" in df.columns or "match_id" not in df.columns:
        return df

    match_ids = [str(match_id) for match_id in df.get_column("match_id").drop_nulls().to_list()]
    if not match_ids:
        return df

    flags = load_is_with_friends(db_path, xuid, match_ids)
    if not flags:
        return df

    return df.with_columns(
        pl.col("match_id")
        .cast(pl.String)
        .map_elements(lambda match_id: flags.get(match_id), return_dtype=pl.Boolean)
        .alias("is_with_friends")
    )


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


def _mean_or_none(df: pl.DataFrame, column: str) -> float | None:
    """Retourne la moyenne d'une colonne si disponible."""
    if column not in df.columns:
        return None
    values = df.get_column(column).cast(pl.Float64, strict=False).drop_nulls()
    if values.is_empty():
        return None
    return float(values.mean())


def _kills_per_min_or_none(df: pl.DataFrame) -> float | None:
    """Retourne les frags par minute agrégés si possible."""
    if "kills" not in df.columns or "time_played_seconds" not in df.columns:
        return None
    total_seconds = float(
        df["time_played_seconds"].cast(pl.Float64, strict=False).fill_null(0).sum()
    )
    if total_seconds <= 0:
        return None
    total_kills = float(df["kills"].cast(pl.Float64, strict=False).fill_null(0).sum())
    return total_kills / (total_seconds / 60.0)


def _append_metric(
    metrics: list[ComparisonMetric],
    *,
    label: str,
    solo_value: float | None,
    squad_value: float | None,
    formatter,
) -> None:
    """Ajoute une métrique si les deux valeurs sont présentes."""
    if solo_value is None or squad_value is None:
        return
    metrics.append(
        ComparisonMetric(
            label=label,
            solo_value=float(solo_value),
            squad_value=float(squad_value),
            solo_text=formatter(float(solo_value)),
            squad_text=formatter(float(squad_value)),
        )
    )


def _build_comparison_metrics(
    solo_df: pl.DataFrame, squad_df: pl.DataFrame
) -> list[ComparisonMetric]:
    """Construit les métriques à comparer entre Solo et Escouade."""
    solo_kd, solo_wr, solo_perf = _compute_group_stats(solo_df)
    squad_kd, squad_wr, squad_perf = _compute_group_stats(squad_df)
    metrics: list[ComparisonMetric] = []
    _append_metric(
        metrics,
        label="K/D",
        solo_value=solo_kd,
        squad_value=squad_kd,
        formatter=lambda value: f"{value:.2f}",
    )
    _append_metric(
        metrics,
        label=t("col_win_rate"),
        solo_value=solo_wr,
        squad_value=squad_wr,
        formatter=lambda value: f"{value:.1f}%",
    )
    _append_metric(
        metrics,
        label=t("col_accuracy"),
        solo_value=_mean_or_none(solo_df, "accuracy"),
        squad_value=_mean_or_none(squad_df, "accuracy"),
        formatter=lambda value: f"{value:.1f}%",
    )
    _append_metric(
        metrics,
        label=t("col_kpm"),
        solo_value=_kills_per_min_or_none(solo_df),
        squad_value=_kills_per_min_or_none(squad_df),
        formatter=lambda value: f"{value:.2f}",
    )
    _append_metric(
        metrics,
        label=t("col_avg_life"),
        solo_value=_mean_or_none(solo_df, "avg_life_seconds"),
        squad_value=_mean_or_none(squad_df, "avg_life_seconds"),
        formatter=lambda value: f"{value:.0f}s",
    )
    _append_metric(
        metrics,
        label=t("sc_performance_score"),
        solo_value=solo_perf,
        squad_value=squad_perf,
        formatter=lambda value: f"{value:.1f}",
    )
    return metrics


def _build_duel_chart(metrics: list[ComparisonMetric]) -> go.Figure:
    """Construit un comparatif visuel Solo vs Escouade auto-suffisant."""
    colors = HALO_COLORS.as_dict()
    ordered = list(reversed(metrics))
    scales = [max(metric.solo_value, metric.squad_value, 1.0) for metric in ordered]
    solo_x = [
        -(metric.solo_value / scale) * 100.0 for metric, scale in zip(ordered, scales, strict=False)
    ]
    squad_x = [
        (metric.squad_value / scale) * 100.0 for metric, scale in zip(ordered, scales, strict=False)
    ]
    labels = [metric.label for metric in ordered]

    fig = go.Figure()
    fig.add_trace(
        go.Bar(
            x=solo_x,
            y=labels,
            orientation="h",
            name=t("syn_solo"),
            marker_color=colors["cyan"],
            text=[metric.solo_text for metric in ordered],
            textposition="outside",
            cliponaxis=False,
            hovertemplate=f"{t('syn_solo')}<br>%{{y}}: %{{text}}<extra></extra>",
        )
    )
    fig.add_trace(
        go.Bar(
            x=squad_x,
            y=labels,
            orientation="h",
            name=t("syn_squad"),
            marker_color=colors["green"],
            text=[metric.squad_text for metric in ordered],
            textposition="outside",
            cliponaxis=False,
            hovertemplate=f"{t('syn_squad')}<br>%{{y}}: %{{text}}<extra></extra>",
        )
    )
    fig.add_vline(x=0, line_width=1, line_color=colors["slate"], opacity=0.8)
    fig.update_layout(
        barmode="overlay",
        height=max(320, 70 * len(metrics)),
        margin={"l": 24, "r": 24, "t": 16, "b": 16},
        showlegend=True,
        legend={"orientation": "h", "yanchor": "bottom", "y": 1.02, "x": 0.5, "xanchor": "center"},
    )
    fig.update_xaxes(
        showgrid=False, showticklabels=False, zeroline=False, range=[-120, 120], title=None
    )
    fig.update_yaxes(title=None, automargin=True)
    return fig


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

    if dff.get_column("is_with_friends").drop_nulls().is_empty():
        st.info(t("syn_no_data"))
        return

    solo_df = dff.filter(pl.col("is_with_friends") == pl.lit(False))
    squad_df = dff.filter(pl.col("is_with_friends") == pl.lit(True))

    if solo_df.is_empty() or squad_df.is_empty():
        st.info(t("syn_no_data"))
        return

    metrics = _build_comparison_metrics(solo_df, squad_df)
    if not metrics:
        st.info(t("syn_no_data"))
        return

    st.caption(t("syn_sample_split", solo=solo_df.height, squad=squad_df.height))
    st.plotly_chart(_build_duel_chart(metrics), width="stretch", config=PLOTLY_STATIC_CONFIG)


@fragment_if_available
def render_synthesis_page(
    dff: DataFrameLike,
    base: DataFrameLike,
    db_path: str,
    xuid: str,
    db_key: tuple[int, int] | None,
) -> None:
    """Affiche la page Synthèse sur tous les matchs, avec période locale."""
    all_matches_pl = ensure_polars(base)
    if all_matches_pl.is_empty():
        st.warning(t("no_matches"))
        return

    selected_period = _render_period_selector()
    period_df = _filter_by_period(all_matches_pl, selected_period)
    period_df = _attach_is_with_friends(period_df, db_path, xuid)
    if period_df.is_empty():
        st.info(t("no_matches"))
        return

    _render_map_mode_breakdown(period_df)
    _render_heatmap_section(period_df)
    _render_top_by_week(period_df)
    _render_solo_squad_compare(period_df)
