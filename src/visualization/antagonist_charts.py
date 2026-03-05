"""Graphiques pour visualisation des antagonistes (némésis/souffre-douleur).

Façade re-exportant les fonctions depuis les sous-modules :
- _antagonist_kv : barres empilées, timeseries, heatmap
- _antagonist_duels : duels, résumé nemesis/victime, top antagonists, indicateur K/D
"""

from __future__ import annotations

# ── Constantes (module feuille, pas de circularité) ──────────────────────────
from src.visualization._antagonist_colors import COLORS, PLAYER_COLORS

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
