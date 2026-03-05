"""Graphiques de performance cumulée — façade de réexport.

Modules internes :
- _perf_cumulative : graphiques cumulatifs séquentiels (net score, K/D, rolling)
- _perf_session    : tendance de session, comparaison, indicateurs agrégés
"""

from src.visualization._perf_cumulative import (
    PERFORMANCE_COLORS,
    plot_cumulative_kd,
    plot_cumulative_net_score,
    plot_rolling_kd,
)
from src.visualization._perf_session import (
    create_cumulative_metrics_indicator,
    get_performance_colors,
    plot_cumulative_comparison,
    plot_session_trend,
)

__all__ = [
    "PERFORMANCE_COLORS",
    "create_cumulative_metrics_indicator",
    "get_performance_colors",
    "plot_cumulative_comparison",
    "plot_cumulative_kd",
    "plot_cumulative_net_score",
    "plot_rolling_kd",
    "plot_session_trend",
]
