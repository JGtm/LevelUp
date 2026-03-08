"""Radar de participation unifié - 6 axes — rendu Plotly + re-exports calcul.

Le calcul pur (seuils, normalisation, profils) est dans src/analysis/participation_radar.py.
Ce module garde uniquement get_radar_axis_lines() (i18n) et re-exporte le reste pour compat.

Voir : .ai/features/RADAR_PARTICIPATION_UNIFIE_PLAN.md
"""

from __future__ import annotations

from src.analysis.participation_radar import (
    RADAR_THRESHOLDS,
    compute_global_radar_thresholds,
    compute_participation_profile,
    get_radar_thresholds,
)
from src.ui.i18n import t


# Légende des axes (une ligne par axe, pour affichage à côté du radar)
def get_radar_axis_lines(lang: str | None = None) -> list[str]:
    """Retourne les descriptions des axes radar dans la langue active.

    Args:
        lang: Langue cible (``"fr"`` / ``"en"``). Si None, utilise ``get_lang()``.
    """
    return [
        t("radar_desc_objectives", lang=lang),
        t("radar_desc_combat", lang=lang),
        t("radar_desc_support", lang=lang),
        t("radar_desc_score", lang=lang),
        t("radar_desc_impact", lang=lang),
        t("radar_desc_survival", lang=lang),
    ]


__all__ = [
    "RADAR_THRESHOLDS",
    "get_radar_axis_lines",
    "compute_participation_profile",
    "compute_global_radar_thresholds",
    "get_radar_thresholds",
]
