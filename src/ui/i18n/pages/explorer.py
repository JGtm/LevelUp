"""Traductions pour la page Explorer."""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
    # ── Navigation ────────────────────────────────────────────────────────
    "page_explorer": {"fr": "Explorer", "en": "Explorer"},
    # ── Filtres ───────────────────────────────────────────────────────────
    "exp_filter_date": "col_date",  # alias → common
    "exp_filter_squad": {"fr": "Type de groupe", "en": "Group type"},
    "exp_squad_all": {"fr": "Tous", "en": "All"},
    "exp_squad_solo": {"fr": "Solo", "en": "Solo"},
    "exp_squad_squad": {"fr": "Escouade", "en": "Squad"},
    "exp_filter_type": {"fr": "Type", "en": "Type"},
    "exp_filter_all": {"fr": "Tous", "en": "All"},
    "exp_filter_playlist": "col_playlist",  # alias
    "exp_filter_mode": "col_mode",  # alias
    "exp_filter_map": "col_map",  # alias
    "exp_match_select": {"fr": "Sélectionner un match", "en": "Select a match"},
    # ── Recherche joueur ──────────────────────────────────────────────────
    "exp_player_search": {"fr": "Rechercher un joueur", "en": "Search for a player"},
    "exp_player_hint": {"fr": "Nom du joueur…", "en": "Player name…"},
    "exp_player_suggestions": {"fr": "Suggestions", "en": "Suggestions"},
    # ── Résultats ─────────────────────────────────────────────────────────
    "exp_no_match_date": {
        "fr": "Aucun match à cette date. Date la plus proche : {date}",
        "en": "No match on this date. Closest date: {date}",
    },
    "exp_results_ally": {
        "fr": "Matchs en alliés ({count})",
        "en": "Matches as allies ({count})",
    },
    "exp_results_enemy": {
        "fr": "Matchs en adversaires ({count})",
        "en": "Matches as opponents ({count})",
    },
    "exp_player_summary": {
        "fr": "Bilan avec {gamertag}",
        "en": "Summary with {gamertag}",
    },
    "exp_no_results": {"fr": "Aucun résultat", "en": "No results"},
    "exp_no_match_id": {
        "fr": "Aucun match trouvé pour cet identifiant.",
        "en": "No match found for this ID.",
    },
    "exp_results_title": {"fr": "Résultats ({count})", "en": "Results ({count})"},
    "exp_select_match_hint": {
        "fr": "Cliquez sur « Rechercher » pour afficher le détail du match.",
        "en": "Click 'Search' to display match details.",
    },
    "exp_player_not_found": {
        "fr": "Joueur introuvable : {gamertag}",
        "en": "Player not found: {gamertag}",
    },
}
