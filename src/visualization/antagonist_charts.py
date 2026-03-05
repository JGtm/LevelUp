"""Graphiques pour visualisation des antagonistes (némésis/souffre-douleur).

Façade re-exportant les fonctions depuis les sous-modules :
- _antagonist_kv : barres empilées, timeseries, heatmap
- _antagonist_duels : duels, résumé nemesis/victime, top antagonists, indicateur K/D
"""

from __future__ import annotations

# ── Constantes (importées par les sous-modules via import retardé) ───────────

COLORS = {
    "kills": "#009E73",
    "deaths": "#D55E00",
    "nemesis": "#E69F00",
    "victim": "#0072B2",
    "neutral": "#888888",
    "positive_kd": "#009E73",
    "negative_kd": "#D55E00",
    "team_alpha": "#0072B2",
    "team_bravo": "#E69F00",
    "highlight": "#F0E442",
}

PLAYER_COLORS = [
    "#0072B2",
    "#E69F00",
    "#009E73",
    "#D55E00",
    "#56B4E9",
    "#CC79A7",
    "#F0E442",
    "#999999",
    "#332288",
    "#44AA99",
    "#DDCC77",
    "#AA4499",
]

# ── Re-exports ───────────────────────────────────────────────────────────────

from src.visualization._antagonist_duels import (  # noqa: E402, F401
    create_kd_indicator,
    get_antagonist_chart_colors,
    plot_duel_history,
    plot_nemesis_victim_summary,
    plot_top_antagonists_bars,
)
from src.visualization._antagonist_kv import (  # noqa: E402, F401
    plot_kd_timeseries,
    plot_killer_victim_heatmap,
    plot_killer_victim_stacked_bars,
)

__all__ = [
    "COLORS",
    "PLAYER_COLORS",
    "create_kd_indicator",
    "get_antagonist_chart_colors",
    "plot_duel_history",
    "plot_kd_timeseries",
    "plot_killer_victim_heatmap",
    "plot_killer_victim_stacked_bars",
    "plot_nemesis_victim_summary",
    "plot_top_antagonists_bars",
]
