"""Traductions de la page Synthèse."""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
    "syn_solo_squad_title": {"fr": "Solo vs Escouade", "en": "Solo vs Squad"},
    "syn_solo_squad_caption": {
        "fr": "Comparaison des métriques clés selon le type de session.",
        "en": "Key metrics comparison by session type.",
    },
    "syn_solo": {"fr": "Solo", "en": "Solo"},
    "syn_squad": {"fr": "Escouade", "en": "Squad"},
    "syn_no_data": {
        "fr": "Pas assez de données pour comparer solo et escouade.",
        "en": "Not enough data to compare solo vs squad.",
    },
}
