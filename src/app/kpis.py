"""Calcul des KPIs pour l'application Streamlit.

Ce module gère :
- La structure de données KPIStats
- Le calcul des statistiques agrégées (compute_kpi_stats)
"""

from __future__ import annotations

from typing import NamedTuple

import polars as pl

from src.analysis import compute_aggregated_stats, compute_outcome_rates
from src.app.helpers import avg_match_duration_seconds, compute_session_span_seconds
from src.utils.polars_compat import ensure_polars as _to_polars

# =============================================================================
# Statistiques KPI
# =============================================================================


class KPIStats(NamedTuple):
    """Statistiques KPI calculées."""

    # Outcomes
    win_rate: float
    loss_rate: float
    total_matches: int
    wins: int
    losses: int
    ties: int

    # Performance
    # no_finish exposé en dernier (défaut = 0 pour rétro-compatibilité)
    avg_accuracy: float | None
    global_ratio: float | None
    avg_life_seconds: float | None

    # Per-game averages
    kills_per_game: float | None
    deaths_per_game: float | None
    assists_per_game: float | None

    # Per-minute rates
    kills_per_minute: float | None
    deaths_per_minute: float | None
    assists_per_minute: float | None

    # Time
    avg_match_seconds: float | None
    total_play_seconds: float | None
    no_finish: int = 0


def compute_kpi_stats(df: pl.DataFrame) -> KPIStats:
    """Calcule toutes les statistiques KPI.

    Args:
        df: DataFrame (Pandas ou Polars) des matchs filtrés.

    Returns:
        KPIStats avec toutes les statistiques calculées.
    """
    df_pl = _to_polars(df)

    # Outcomes
    rates = compute_outcome_rates(df_pl)
    total_outcomes = max(1, rates.total)
    win_rate = rates.wins / total_outcomes
    loss_rate = rates.losses / total_outcomes

    # Performance
    avg_acc = None
    if not df_pl.is_empty() and "accuracy" in df_pl.columns:
        avg_acc = df_pl.select(pl.col("accuracy").drop_nulls().mean()).item()
    global_ratio = None
    if not df_pl.is_empty() and "ratio" in df_pl.columns:
        global_ratio = df_pl.select(pl.col("ratio").drop_nulls().mean()).item()
    avg_life = None
    if not df_pl.is_empty() and "average_life_seconds" in df_pl.columns:
        avg_life = df_pl.select(pl.col("average_life_seconds").drop_nulls().mean()).item()

    # Per-game averages
    kpg = df_pl.select(pl.col("kills").mean()).item() if not df_pl.is_empty() else None
    dpg = df_pl.select(pl.col("deaths").mean()).item() if not df_pl.is_empty() else None
    apg = df_pl.select(pl.col("assists").mean()).item() if not df_pl.is_empty() else None

    # Per-minute rates
    stats = compute_aggregated_stats(df_pl)

    # Time
    avg_match_s = avg_match_duration_seconds(df_pl)
    total_play_s = compute_session_span_seconds(df_pl)

    return KPIStats(
        win_rate=win_rate,
        loss_rate=loss_rate,
        total_matches=rates.total,
        wins=rates.wins,
        losses=rates.losses,
        ties=rates.ties,
        avg_accuracy=avg_acc,
        global_ratio=global_ratio,
        avg_life_seconds=avg_life,
        kills_per_game=kpg,
        deaths_per_game=dpg,
        assists_per_game=apg,
        kills_per_minute=stats.kills_per_minute,
        deaths_per_minute=stats.deaths_per_minute,
        assists_per_minute=stats.assists_per_minute,
        avg_match_seconds=avg_match_s,
        total_play_seconds=total_play_s,
        no_finish=rates.no_finish,
    )
