"""Chaînes pour les widgets interactifs Streamlit.

Boutons, selectbox, expander, checkbox, radio, multiselect, captions de widgets.

⚠️ ChatGPT : remplir toutes les valeurs marquées "TODO" ci-dessous.
   Règles : voir le prompt de la Phase 1b dans le plan i18n.
"""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
    # ── Sélecteur de langue ──────────────────────────────────────────────────
    "lang_selector_label": {
        "fr": "🌐 Langue",
        "en": "🌐 Language",
    },
    # ── Boutons génériques ───────────────────────────────────────────────────
    "btn_sync": {
        "fr": "Actualiser",
        "en": "Refresh",
    },
    "btn_open_match": {
        "fr": "Ouvrir le match",
        "en": "Open match",
    },
    "btn_use_this_match": "btn_use_match",  # alias → common
    # btn_search : défini dans common.py
    "btn_view_fullscreen": {
        "fr": "Voir en grand",
        "en": "View fullscreen",
    },
    "btn_reset_media_index": {
        "fr": "Réinitialiser l'index médias",
        "en": "Reset media index",
    },
    "btn_rescan_folders": {
        "fr": "Re-scanner les dossiers",
        "en": "Rescan folders",
    },
    "btn_hide_thumbnail": {
        "fr": "Masquer miniature",
        "en": "Hide thumbnail",
    },
    "btn_show_thumbnail": {
        "fr": "Afficher miniature",
        "en": "Show thumbnail",
    },
    "btn_preview": {
        "fr": "Aperçu",
        "en": "Preview",
    },
    "btn_launch_openspartan": {
        "fr": "Lancer OpenSpartan Workshop",
        "en": "Launch OpenSpartan Workshop",
    },
    # btn_confirm, btn_cancel : définis dans common.py
    # ── Confirmations ────────────────────────────────────────────────────────
    "confirm_clear_selection": {
        "fr": "⚠️ Confirmer : vider toutes les sélections ?",
        "en": "⚠️ Confirm: clear all selections?",
    },
    # ── Expanders ───────────────────────────────────────────────────────────
    "exp_source": {
        "fr": "Source",
        "en": "Source",
    },
    "exp_sync": {
        "fr": "Synchronisation",
        "en": "Sync",
    },
    "exp_sync_button_options": {
        "fr": "Options du bouton Actualiser",
        "en": "Refresh button options",
    },
    "exp_media": {
        "fr": "Médias",
        "en": "Media",
    },
    "exp_experience": {
        "fr": "Expérience",
        "en": "Experience",
    },
    "exp_match_details": {
        "fr": "Détails des matchs (joueur vs joueur)",
        "en": "Match details (player vs player)",
    },
    "exp_filters": {
        "fr": "Filtres",
        "en": "Filters",
    },
    "exp_options": {
        "fr": "Options",
        "en": "Options",
    },
    "exp_datetime_search": {
        "fr": "Recherche par date/heure",
        "en": "Search by date/time",
    },
    "exp_advanced_analytics": {
        "fr": "📊 Analytics avancées (DuckDB)",
        "en": "📊 Advanced analytics (DuckDB)",
    },
    "exp_perf_score_info": {
        "fr": "ℹ️ À propos du score de performance",
        "en": "ℹ️ About the performance score",
    },
    "exp_history_detail": {
        "fr": "Historique détaillé",
        "en": "Detailed history",
    },
    "exp_current_ranking": {
        "fr": "📊 Classement actuel (instantané)",
        "en": "📊 Current rating snapshot",
    },
    # ── Checkboxes ──────────────────────────────────────────────────────────
    "chk_backfill_all": {
        "fr": "Tout backfiller",
        "en": "Backfill everything",
    },
    "chk_backfill_medals": {
        "fr": "Médailles",
        "en": "Medals",
    },
    "chk_backfill_events": {
        "fr": "Events (film)",
        "en": "Events (film)",
    },
    "chk_backfill_skill": {
        "fr": "Compétence (LUSR/CSR)",
        "en": "Skill (LUSR/CSR)",
    },
    "chk_backfill_personal_scores": {
        "fr": "Scores personnels",
        "en": "Personal scores",
    },
    "chk_backfill_performance_scores": {
        "fr": "Scores de performance",
        "en": "Performance scores",
    },
    "chk_backfill_aliases": {
        "fr": "Alias xuid",
        "en": "XUID aliases",
    },
    "chk_same_team_only": {
        "fr": "Même équipe uniquement",
        "en": "Same team only",
    },
    # ── Selectbox / Radio ─────────────────────────────────────────────────────
    "sel_category": {
        "fr": "Catégorie",
        "en": "Category",
    },
    "sel_all_categories": "sel_all",  # alias → common
    "sel_group": {
        "fr": "Groupe :",
        "en": "Group:",
    },
    "sel_metric": {
        "fr": "Métrique",
        "en": "Metric",
    },
    "sel_session_a": {
        "fr": "Session A",
        "en": "Session A",
    },
    "sel_session_b": {
        "fr": "Session B",
        "en": "Session B",
    },
    "sel_scope_filter": {
        "fr": "Moi (filtres actuels)",
        "en": "Me (current filters)",
    },
    "sel_scope_all": {
        "fr": "Moi (toutes les parties)",
        "en": "Me (all matches)",
    },
    # ── Tabs ────────────────────────────────────────────────────────────────
    "tab_map": {
        "fr": "🗺️ Carte",
        "en": "🗺️ Map",
    },
    "tab_rank": {
        "fr": "🏅 Rang",
        "en": "🏅 Rank",
    },
    "tab_axes": {
        "fr": "**Axes**",
        "en": "**Axes**",
    },
    # ── Sidebar ──────────────────────────────────────────────────────────────
    "sidebar_no_db_selected": {
        "fr": "Aucune base de données sélectionnée",
        "en": "No database selected",
    },
    "sidebar_quick_filters": {
        "fr": "🎮 Filtres rapides",
        "en": "🎮 Quick filters",
    },
    # ── Filtres ──────────────────────────────────────────────────────────────
    "filter_date_error": {
        "fr": "La date de début est après la date de fin.",
        "en": "Start date is after end date.",
    },
    "filter_mode_label": {
        "fr": "Mode de filtrage",
        "en": "Filter mode",
    },
    # ── Coéquipiers ──────────────────────────────────────────────────────────
    "tm_multiselect_label": {
        "fr": "Coéquipiers",
        "en": "Teammates",
    },
    # ── Match history ─────────────────────────────────────────────────────────
    "mh_no_matches": "no_matches",  # alias → common
    # ── Sidebar — sync ──────────────────────────────────────────────────────
    "sidebar_sync_btn": {"fr": "🔄 Synchroniser", "en": "🔄 Sync"},
    "sidebar_sync_help": {
        "fr": "Synchronise tous les joueurs (nouveaux matchs, highlights, aliases).",
        "en": "Sync all players (new matches, highlights, aliases).",
    },
    "sidebar_sync_spinner": {
        "fr": "Synchronisation en cours…",
        "en": "Syncing…",
    },
    "sidebar_backfill_running": {
        "fr": "Backfill des données manquantes…",
        "en": "Backfilling missing data…",
    },
    "sidebar_navigation": {"fr": "Navigation", "en": "Navigation"},
    "sidebar_db_not_found": {
        "fr": "Base introuvable : {path}",
        "en": "Database not found: {path}",
    },
    # ── Sidebar — résultats backfill ────────────────────────────────────────
    "backfill_medals": {"fr": "{n} médaille(s)", "en": "{n} medal(s)"},
    "backfill_events": {"fr": "{n} event(s)", "en": "{n} event(s)"},
    "backfill_scores": {"fr": "{n} score(s) perf", "en": "{n} perf score(s)"},
    "backfill_aliases": {"fr": "{n} alias(es)", "en": "{n} alias(es)"},
    "backfill_personal_scores": {"fr": "{n} personal_score(s)", "en": "{n} personal score(s)"},
    "backfill_skill": {"fr": "skill", "en": "skill"},
    "backfill_lusr": {"fr": "{n} LUSR calculé(s)", "en": "{n} LUSR computed"},
    # ── Filtres — labels supplémentaires ────────────────────────────────────
    "filter_header": {"fr": "Filtres", "en": "Filters"},
    "filter_selection": {"fr": "Sélection", "en": "Selection"},
    "filter_period": {"fr": "Période", "en": "Period"},
    "filter_sessions": {"fr": "Sessions", "en": "Sessions"},
    "filter_start": {"fr": "Début", "en": "Start"},
    "filter_end": {"fr": "Fin", "en": "End"},
    "filter_playlists": "col_playlists",  # alias → common
    "filter_modes": "col_modes",  # alias → common
    "filter_maps": "col_maps",  # alias → common
    "filter_last_session": {"fr": "Dernière session", "en": "Last session"},
    "filter_prev_session": {"fr": "Session précédente", "en": "Previous session"},
    "filter_session_label": {"fr": "Session", "en": "Session"},
    "filter_solo_title": {"fr": "En solo", "en": "Solo"},
    "filter_squad_title": {"fr": "Mon escouade", "en": "My Squad"},
    "filter_last_carnage": {"fr": "Dernier carnage", "en": "Last carnage"},
    "filter_prev_carnage": {"fr": "Carnage précédent", "en": "Previous carnage"},
    "filter_solo_session_label": {"fr": "Session solo", "en": "Solo session"},
    "filter_squad_session_label": {"fr": "Session escouade", "en": "Squad session"},
    "filter_squad_no_friends": {
        "fr": "Configurez vos coéquipiers dans l'onglet Teammates.",
        "en": "Configure your teammates in the Teammates tab.",
    },
    "filter_pvp_unranked": {"fr": "PVP non classé", "en": "Unranked PVP"},
    "filter_pvp_ranked": {"fr": "PVP classé", "en": "Ranked PVP"},
    "filter_pve": {"fr": "PVE", "en": "PVE"},
    # ── Settings — labels supplémentaires ───────────────────────────────────
    "settings_save": "btn_save",  # alias → common
    "settings_backfill_enable": {
        "fr": "Activer le backfill après synchronisation",
        "en": "Enable backfill after sync",
    },
    "settings_max_matches": {"fr": "Max matchs par sync", "en": "Max matches per sync"},
    "settings_api_rate": {"fr": "Requêtes API / seconde", "en": "API requests / second"},
    "settings_clear_cache_on_refresh": {
        "fr": "Le bouton Actualiser vide aussi les caches",
        "en": "Refresh also clears caches",
    },
    "settings_media_base_folder": {
        "fr": "Dossier de base des captures",
        "en": "Base folder for captures",
    },
    "settings_media_tolerance": {
        "fr": "Tolérance (minutes) autour du match",
        "en": "Tolerance (minutes) around the match",
    },
    "settings_source_help": {
        "fr": "Le fichier db_profiles.json contient la configuration des bases de données. Les tokens d'authentification sont dans .env.local.",
        "en": "The db_profiles.json file contains the database configuration. Authentication tokens are in .env.local.",
    },
    "settings_sync_help": {
        "fr": "Configure les paramètres de synchronisation avec l'API Halo.",
        "en": "Configure sync settings with the Halo API.",
    },
    # ── Experience types ────────────────────────────────────────────────────
    "exp_pvp_unranked": {"fr": "PVP non classé", "en": "Unranked PVP"},
    "exp_pvp_ranked": {"fr": "PVP classé", "en": "Ranked PVP"},
    "exp_pve": {"fr": "PVE", "en": "PVE"},
    # ── Sidebar — sélecteur joueur ───────────────────────────────────────────
    "sidebar_player_heading": {"fr": "👥 Joueur", "en": "👥 Player"},
    "sidebar_player_label": "col_player",  # alias → common
    # ── KPI — résumé parties sélectionnées ──────────────────────────────────
    "kpi_no_matches": {"fr": "Aucun match sélectionné", "en": "No match selected"},
    "kpi_selected_matches": {"fr": "Parties sélectionnées", "en": "Selected matches"},
    "kpi_wins": {"fr": "Victoires", "en": "Wins"},
    "kpi_losses": {"fr": "Défaites", "en": "Losses"},
    "kpi_ties": {"fr": "Égalités", "en": "Ties"},
    "kpi_no_finish": {"fr": "Non terminés", "en": "Unfinished"},
    # btn_select_all, btn_select_none : définis dans common.py
    # ── Durée ────────────────────────────────────────────────────────────────
    "dur_day_abbr": {"fr": "j", "en": "d"},
    # ── Métriques DuckDB analytics ───────────────────────────────────────────
    "metric_matches": "col_matches",  # alias → common
    "metric_win_rate": {"fr": "Taux victoires", "en": "Win rate"},
    "metric_avg_kda": {"fr": "FDA moyen", "en": "Avg KDA"},
    "metric_accuracy": "col_accuracy",  # alias → common
    "metric_kills": "col_kills",  # alias → common
    "metric_deaths": "col_deaths",  # alias → common
    "metric_assists": "col_assists",  # alias → common
    "metric_time_played": "col_total_time",  # alias → common
    # ── Filtres — playlists preferred order ──────────────────────────────────
    "playlist_quick_play": {"fr": "Partie rapide", "en": "Quick Play"},
    "playlist_ranked_arena": {"fr": "Arène classée", "en": "Ranked Arena"},
    "playlist_ranked_assassin": {"fr": "Assassin classé", "en": "Ranked Assassin"},
}
