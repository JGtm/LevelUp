"""Utilitaires temporels pour la bibliothèque médias (extraits de media_library_data.py)."""

from __future__ import annotations

from datetime import datetime

from src.ui.formatting import PARIS_TZ


def epoch_seconds_paris(dt_value: datetime | None) -> float | None:
    """Convertit un datetime en secondes epoch (fuseau Paris)."""
    if dt_value is None:
        return None
    try:
        aware = (
            PARIS_TZ.localize(dt_value)
            if dt_value.tzinfo is None
            else dt_value.astimezone(PARIS_TZ)
        )
        return float(aware.timestamp())
    except Exception:
        return None


def to_paris_naive(dt_value: object) -> datetime | None:
    """Convertit une valeur datetime en datetime naïve (fuseau Paris)."""
    try:
        if dt_value is None:
            return None
        if isinstance(dt_value, datetime):
            ts = dt_value
        elif isinstance(dt_value, str):
            s = str(dt_value).strip()
            if not s:
                return None
            if s.endswith("Z"):
                s = s[:-1] + "+00:00"
            ts = datetime.fromisoformat(s)
        else:
            return None
        if ts.tzinfo is None:
            return ts
        return ts.astimezone(PARIS_TZ).replace(tzinfo=None)
    except Exception:
        return None
