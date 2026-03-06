"""Fonctions utilitaires de conversion/clamping centralisées.

Remplace les copies de ``_safe_float``, ``_safe_int``, ``_clamp``
dispersées dans le codebase.
"""

from __future__ import annotations

import math
from typing import Any


def safe_float(value: Any) -> float | None:
    """Convertit une valeur en float, retourne None si impossible/non-fini.

    Gère ``None``, ``NaN``, ``Inf`` et les types non-numériques.
    """
    if value is None:
        return None
    try:
        f = float(value)
        if math.isnan(f) or math.isinf(f):
            return None
        return f
    except (TypeError, ValueError):
        return None


def safe_int(value: Any) -> int | None:
    """Convertit une valeur en int, retourne None si impossible/non-fini."""
    if value is None:
        return None
    try:
        f = float(value)
        if math.isnan(f) or math.isinf(f):
            return None
        return int(f)
    except (TypeError, ValueError):
        return None


def clamp(value: float, lo: float = 0.0, hi: float = 100.0) -> float:
    """Contraint ``value`` dans l'intervalle ``[lo, hi]``."""
    return max(lo, min(hi, value))
