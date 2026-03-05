"""Constantes de couleurs pour les graphiques antagonistes.

Module feuille (sans dépendance sur les sous-modules) importé par
``_antagonist_duels.py``, ``_antagonist_kv.py`` et ``antagonist_charts.py``.
Élimine la dépendance circulaire précédente (sous-modules ↔ façade).
"""

from __future__ import annotations

COLORS: dict[str, str] = {
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

PLAYER_COLORS: list[str] = [
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
