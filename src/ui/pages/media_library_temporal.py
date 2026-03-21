"""Utilitaires temporels pour la bibliothèque médias (extraits de media_library_data.py)."""

from __future__ import annotations

from datetime import datetime, timezone

from src.ui.formatting import _get_user_tz


def epoch_seconds_paris(dt_value: datetime | None) -> float | None:
    """Convertit un datetime en secondes epoch (TZ utilisateur)."""
    if dt_value is None:
        return None
    try:
        user_tz = _get_user_tz()
        aware = (
            dt_value.replace(tzinfo=user_tz)
            if dt_value.tzinfo is None
            else dt_value.astimezone(user_tz)
        )
        return float(aware.timestamp())
    except Exception:
        return None


def to_paris_naive(dt_value: object) -> datetime | None:
    """Convertit une valeur datetime en datetime naïve (TZ utilisateur).

    Convention DB : les naïfs sont en UTC.
    """
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
        user_tz = _get_user_tz()
        if ts.tzinfo is None:
            ts = ts.replace(tzinfo=timezone.utc)
        return ts.astimezone(user_tz).replace(tzinfo=None)
    except Exception:
        return None
