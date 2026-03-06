"""Constantes de formatage des dates — LevelUp.

Centralise tous les formats ``strftime`` utilisés dans le dashboard.
Évite les duplications de chaînes littérales et assure la cohérence.
"""

from __future__ import annotations

# --- Formats FR (affichage utilisateur) ---
FMT_DATE_FR = "%d/%m/%Y"  # 02/03/2026
FMT_DATE_FR_SHORT_YEAR = "%d/%m/%y"  # 02/03/26
FMT_SHORT_DATE_FR = "%d/%m"  # 02/03
FMT_DATETIME_FR = "%d/%m/%Y %H:%M"  # 02/03/2026 14:30
FMT_DATETIME_FR_SHORT_YEAR = "%d/%m/%y %H:%M"  # 02/03/26 14:30
FMT_SHORT_DATETIME_FR = "%d/%m %H:%M"  # 02/03 14:30

# --- Formats ISO (calcul / clés) ---
FMT_DATE_ISO = "%Y-%m-%d"  # 2026-03-02
FMT_MONTH_ISO = "%Y-%m"  # 2026-03
FMT_ISO8601 = "%Y-%m-%dT%H:%M:%S"  # 2026-03-02T14:30:00

# --- Formats heure ---
FMT_TIME = "%H:%M"  # 14:30
FMT_TIME_FULL = "%H:%M:%S"  # 14:30:00

# --- Formats tick Plotly ---
FMT_TICK_DATETIME = "%d/%m %H:%M"  # 02/03 14:30
