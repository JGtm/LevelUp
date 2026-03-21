"""Helpers pour le graphe stats par minute (plot_per_minute_timeseries).

Séparé de timeseries.py pour respecter la limite de 500 lignes par module.
"""

from __future__ import annotations

import math


def build_symmetric_abs_ticks(
    max_pos: float, max_neg: float, n_steps: int = 5
) -> tuple[list[float], list[str]]:
    """Calcule des ticks symétriques avec labels absolus pour un axe Y miroir.

    Les valeurs négatives (deaths) sont affichées avec des labels positifs.

    Args:
        max_pos: Valeur maximale positive (kills/assists par minute).
        max_neg: Valeur maximale absolue côté négatif (deaths par minute).
        n_steps: Nombre approximatif de graduations de chaque côté.

    Returns:
        (tickvals, ticktext) pour fig.update_yaxes(tickvals=..., ticktext=...).
    """
    y_max = max(max_pos, max_neg, 0.1)
    raw_step = y_max / n_steps
    # Arrondir le step à une valeur "propre"
    magnitude = 10 ** math.floor(math.log10(raw_step))
    step = round(math.ceil(raw_step / magnitude) * magnitude, 10)

    ticks_pos = [round(i * step, 4) for i in range(n_steps + 2)]
    ticks_neg = [-v for v in ticks_pos[1:]]

    tickvals = sorted(set(ticks_neg + ticks_pos))
    ticktext = [str(abs(v)) if abs(v) == int(abs(v)) else f"{abs(v):.2f}" for v in tickvals]
    return tickvals, ticktext
