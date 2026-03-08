"""Helpers de parsing et conversion de types pour les transformers.

Coercions de types sûres (_safe_float, _safe_int, etc.) et parsing de formats
standardisés (ISO 8601, durées). Ces fonctions n'ont aucune dépendance externe.
"""

from __future__ import annotations

import math
import re
from datetime import datetime
from typing import Any

# Regex XUID Xbox (12-20 chiffres)
XUID_RE = re.compile(r"(\d{12,20})")


def _safe_float(v: Any) -> float | None:
    """Convertit une valeur en float, gérant NaN et None."""
    if v is None:
        return None
    try:
        f = float(v)
        if math.isnan(f) or math.isinf(f):
            return None
        return f
    except (TypeError, ValueError):
        return None


def _safe_int(v: Any) -> int | None:
    """Convertit une valeur en int, gérant NaN et None."""
    if v is None:
        return None
    try:
        f = float(v)
        if math.isnan(f) or math.isinf(f):
            return None
        return int(f)
    except (TypeError, ValueError):
        return None


def _safe_str(v: Any) -> str | None:
    """Convertit une valeur en str, gérant None."""
    if v is None:
        return None
    try:
        s = str(v)
        if s == "nan" or s == "None":
            return None
        return s
    except Exception:
        return None


def _parse_iso_utc(s: str | None) -> datetime | None:
    """Parse un timestamp ISO 8601 en datetime UTC."""
    if not s or not isinstance(s, str):
        return None
    try:
        # Gérer les formats avec ou sans 'Z'
        s = s.replace("Z", "+00:00")
        if "+" not in s and "-" not in s[10:]:
            s += "+00:00"
        return datetime.fromisoformat(s)
    except Exception:
        return None


def _parse_duration_to_seconds(duration_str: str) -> int | None:
    """Parse une durée ISO 8601 (PT1H30M45S) en secondes."""
    if not duration_str or not isinstance(duration_str, str):
        return None

    # Format: PT{hours}H{minutes}M{seconds}S ou variations
    pattern = r"PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?"
    match = re.match(pattern, duration_str, re.IGNORECASE)
    if not match:
        return None

    hours = int(match.group(1) or 0)
    minutes = int(match.group(2) or 0)
    seconds = float(match.group(3) or 0)

    return int(hours * 3600 + minutes * 60 + seconds)
