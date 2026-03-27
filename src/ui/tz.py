"""Gestion de la timezone utilisateur — source unique pour tout l'affichage.

- ``get_tz_name()`` lit ``user_timezone`` depuis ``st.session_state["app_settings"]``
  avec fallback ``"Europe/Paris"``.
- ``get_tz()`` retourne le ``ZoneInfo`` correspondant.
- ``CURATED_TZ_LIST`` : liste curated pour le sélecteur UI (~40 TZ IANA).
- ``utc_to_local()`` / ``local_to_utc()`` : conversions centralisées.

Le module importe ``streamlit`` en lazy (dans les fonctions) pour rester
importable hors contexte Streamlit (scripts CLI, tests).
"""

from __future__ import annotations

import logging
from datetime import datetime, timezone
from zoneinfo import ZoneInfo

logger = logging.getLogger(__name__)

_DEFAULT_TZ = "Europe/Paris"

# ─── Liste curated affichée dans le sélecteur Settings ──────────────────────

CURATED_TZ_LIST: list[str] = [
    # Europe
    "Europe/Paris",
    "Europe/London",
    "Europe/Berlin",
    "Europe/Madrid",
    "Europe/Rome",
    "Europe/Amsterdam",
    "Europe/Brussels",
    "Europe/Zurich",
    "Europe/Lisbon",
    "Europe/Stockholm",
    "Europe/Oslo",
    "Europe/Copenhagen",
    "Europe/Warsaw",
    "Europe/Prague",
    "Europe/Budapest",
    "Europe/Athens",
    "Europe/Helsinki",
    "Europe/Bucharest",
    "Europe/Moscow",
    "Europe/Istanbul",
    # Amériques
    "America/New_York",
    "America/Chicago",
    "America/Denver",
    "America/Los_Angeles",
    "America/Toronto",
    "America/Vancouver",
    "America/Sao_Paulo",
    "America/Argentina/Buenos_Aires",
    "America/Mexico_City",
    "America/Bogota",
    # Asie
    "Asia/Tokyo",
    "Asia/Seoul",
    "Asia/Shanghai",
    "Asia/Singapore",
    "Asia/Kolkata",
    "Asia/Dubai",
    "Asia/Karachi",
    # Australie / Pacifique
    "Australia/Sydney",
    "Australia/Perth",
    "Pacific/Auckland",
    # Universel
    "UTC",
]


def get_tz_name() -> str:
    """Retourne le nom IANA de la timezone utilisateur.

    Lit ``user_timezone`` depuis ``st.session_state["app_settings"]``.
    Fallback : ``"Europe/Paris"``.
    """
    try:
        import streamlit as st  # lazy — évite d'importer streamlit hors contexte UI

        settings = st.session_state.get("app_settings")
        if settings is not None:
            tz = settings.user_timezone.strip()
            if tz:
                return tz
    except Exception:
        logger.debug("get_tz_name: impossible de lire app_settings, fallback %s", _DEFAULT_TZ)
    return _DEFAULT_TZ


def get_tz() -> ZoneInfo:
    """Retourne un objet ``ZoneInfo`` pour la timezone utilisateur."""
    return ZoneInfo(get_tz_name())


# ─── Conversion UTC ↔ local ─────────────────────────────────────────────────


def db_ts_to_utc(dt: datetime) -> datetime:
    """Convertit un timestamp issu d'une colonne DuckDB TIMESTAMP en UTC-aware.

    DuckDB convertit les datetime(tzinfo=UTC) en heure locale système (CET/CEST)
    avant stockage dans les colonnes TIMESTAMP (sans timezone). Un datetime naïf
    lu depuis DuckDB représente donc l'heure locale, pas UTC.

    Comportement :
    - Naïf  → astimezone() utilise la TZ locale système (CET) → UTC correct
    - Aware → astimezone() convertit directement vers UTC

    Args:
        dt: Datetime naïf ou aware (typiquement issu d'une requête DuckDB).

    Returns:
        Datetime UTC-aware représentant le même instant.
    """
    return dt.astimezone(timezone.utc)


def utc_to_local(dt: datetime) -> datetime:
    """Convertit un datetime UTC (naïf ou aware) en timezone utilisateur.

    Convention UI : afficher en timezone utilisateur.
    Pour les timestamps issus de DuckDB, utiliser db_ts_to_utc() en amont.
    """
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    return dt.astimezone(get_tz())


def local_to_utc(dt: datetime) -> datetime:
    """Convertit un datetime local (naïf → assume TZ utilisateur) en UTC."""
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=get_tz())
    return dt.astimezone(timezone.utc)
