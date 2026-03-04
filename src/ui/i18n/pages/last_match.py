"""Chaînes i18n — page Dernière partie."""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
    "last_match_search_by_datetime": {
        "fr": "Recherche par date/heure",
        "en": "Search by date/time",
    },
    # ── Multiplayer ──────────────────────────────────────────────────────────
    "last_match_caption": {
        "fr": "Dernière partie selon la sélection/filtres actuels.",
        "en": "Last match based on current selection/filters.",
    },
    "last_match_select_caption": {
        "fr": "Afficher un match précis via un MatchId, une date/heure, ou une sélection.",
        "en": "Display a specific match using a MatchId, a date/time, or a selection.",
    },
    "last_match_no_date": {
        "fr": "Aucune date exploitable dans la DB.",
        "en": "No usable date found in the DB.",
    },
    "last_match_not_found": {
        "fr": "MatchId introuvable dans la DB actuelle.",
        "en": "MatchId not found in the current DB.",
    },
    "last_match_enter_id": {
        "fr": "Renseigne un MatchId ou utilise la sélection/recherche ci-dessus.",
        "en": "Enter a MatchId or use the selection/search above.",
    },
    # ── Objective Analysis (sections) ─────────────────────────────────────────
    "lm_quick_select": {
        "fr": "Sélection rapide (filtres actuels)",
        "en": "Quick select (current filters)",
    },
    "lm_search_datetime": {"fr": "Recherche par date/heure", "en": "Search by date/time"},
    "lm_date": "col_date",  # alias → common
    "lm_time": "col_time",  # alias → common
    "lm_tolerance": {"fr": "Tolérance (minutes)", "en": "Tolerance (minutes)"},
    "lm_no_match_tol": {
        "fr": "Aucun match trouvé dans ±{tol} min autour de {dt}.",
        "en": "No match found within ±{tol} min of {dt}.",
    },
    # ── Session compare ─────────────────────────────────────────────────────
}
