"""Graphiques de performance cumulée — façade de réexport.

Modules internes :
- _perf_cumulative  : graphiques cumulatifs séquentiels (net score, K/D, rolling)
- _perf_session     : tendance de session, comparaison, indicateurs agrégés
- _perf_progression : progression avancée (EWMA, IC 90 %, net score/h)
"""

from src.visualization._perf_cumulative import (
    PERFORMANCE_COLORS,
    plot_cumulative_kd,
    plot_cumulative_net_score,
    plot_rolling_kd,
)
from src.visualization._perf_progression import (
    _add_outcome_markers,
    plot_cumulative_kd_with_ci,
    plot_ewma_kd,
    plot_net_score_per_hour,
)
from src.visualization._perf_session import (
    create_cumulative_metrics_indicator,
    get_performance_colors,
    plot_cumulative_comparison,
    plot_regression_trend,
    plot_session_trend,
)

__all__ = [
    "PERFORMANCE_COLORS",
    "_add_outcome_markers",
    "create_cumulative_metrics_indicator",
    "get_performance_colors",
    "plot_cumulative_comparison",
    "plot_cumulative_kd",
    "plot_cumulative_kd_with_ci",
    "plot_cumulative_net_score",
    "plot_ewma_kd",
    "plot_net_score_per_hour",
    "plot_regression_trend",
    "plot_rolling_kd",
    "plot_session_trend",
]


__all__ = [
    "PERFORMANCE_COLORS",
    "_add_outcome_markers",
    "create_cumulative_metrics_indicator",
    "get_performance_colors",
    "plot_cumulative_comparison",
    "plot_cumulative_kd",
    "plot_cumulative_kd_with_ci",
    "plot_cumulative_net_score",
    "plot_ewma_kd",
    "plot_net_score_per_hour",
    "plot_regression_trend",
    "plot_rolling_kd",
    "plot_session_trend",
]
