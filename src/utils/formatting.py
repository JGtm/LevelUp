"""Utilitaires de formatage purs (sans dépendance UI/Streamlit).

Ce module centralise les formateurs utilisables à tous les niveaux de l'architecture
(analysis, data, ui) sans créer de couplage vers src.ui.
"""

from __future__ import annotations

from datetime import timedelta


def format_mmss(seconds: float | int | None) -> str:
    """Formate une durée en mm:ss.

    Args:
        seconds: Durée en secondes.

    Returns:
        Chaîne formatée "mm:ss" ou "-" si invalide.
    """
    if seconds is None:
        return "-"
    if isinstance(seconds, float) and seconds != seconds:  # NaN
        return "-"
    try:
        secs = int(seconds)
        if secs < 0:
            return "-"
        td = timedelta(seconds=secs)
        total_minutes = td.seconds // 60
        remaining_seconds = td.seconds % 60
        return f"{total_minutes:02d}:{remaining_seconds:02d}"
    except (ValueError, TypeError):
        return "-"
