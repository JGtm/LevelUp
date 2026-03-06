"""Utilitaires partagés pour les filtres applicatifs.

Centralise ``safe_to_date`` utilisé dans ``filters.py``,
``filters_render.py`` et ``_filters_period.py``.
"""

from __future__ import annotations

from datetime import date


def safe_to_date(val: object) -> date:
    """Convertit une valeur en ``date`` Python, ``date.today()`` si invalide."""
    if isinstance(val, date):
        return val
    try:
        from dateutil.parser import parse as _parse_dt

        return _parse_dt(str(val)).date()
    except (ValueError, TypeError, ImportError):
        return date.today()
