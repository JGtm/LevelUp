"""Chaînes pour les widgets interactifs Streamlit.

Boutons, selectbox, expander, checkbox, radio, multiselect, captions de widgets.

⚠️ ChatGPT : remplir toutes les valeurs marquées "TODO" ci-dessous.
   Règles : voir le prompt de la Phase 1b dans le plan i18n.
"""
from __future__ import annotations

STRINGS: dict[str, dict[str, str]] = {
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
    "btn_use_this_match": {
        "fr": "Utiliser ce match",
        "en": "Use this match",
    },
    "btn_search": {
        "fr": "Rechercher",
        "en": "Search",
    },
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
    "btn_confirm": {
        "fr": "Confirmer",
        "en": "Confirm",
    },
    "btn_cancel": {
        "fr": "Annuler",
        "en": "Cancel",
    },
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
    "sel_all_categories": {
        "fr": "(toutes)",
        "en": "(all)",
    },
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
    "tm_session_trend": {
        "fr": "Tendance de session",
        "en": "Session trend",
    },
    # ── Match history ─────────────────────────────────────────────────────────
    "mh_no_matches": {
        "fr": "Aucun match à afficher. Vérifiez vos filtres ou synchronisez les données.",
        "en": "No matches to display. Check your filters or sync your data.",
    },
}
