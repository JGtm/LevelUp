"""Chaînes génériques partagées entre toutes les pages.

Ces chaînes apparaissent dans plusieurs fichiers src/ui/pages/ et src/app/.
En centralisant ici, on évite la duplication.

⚠️ ChatGPT : remplir toutes les valeurs marquées "TODO" ci-dessous.
   Règles : voir le prompt de la Phase 1b dans le plan i18n.
"""

from __future__ import annotations

STRINGS: dict[str, dict[str, str]] = {
    # ── Données manquantes / vides ──────────────────────────────────────────
    "no_matches": {
        "fr": "Aucun match à afficher. Vérifiez vos filtres ou synchronisez les données.",
        "en": "No matches to display. Check your filters or sync your data.",
    },
    "no_data": {
        "fr": "Aucune donnée disponible.",
        "en": "No data available.",
    },
    "no_data_filter": {
        "fr": "Aucune donnée disponible pour ce filtre.",
        "en": "No data available for this filter.",
    },
    "insufficient_data": {
        "fr": "Données insuffisantes.",
        "en": "Not enough data.",
    },
    "insufficient_data_chart": {
        "fr": "Données insuffisantes pour afficher ce graphique.",
        "en": "Not enough data to display this chart.",
    },
    "no_events": {
        "fr": "Aucun événement trouvé.",
        "en": "No events found.",
    },
    "no_media": {
        "fr": "Aucun média à afficher avec ces filtres.",
        "en": "No media to display with these filters.",
    },
    "no_match_found": {
        "fr": "Aucun match trouvé.",
        "en": "No match found.",
    },
    "no_medals": {
        "fr": "Aucune médaille.",
        "en": "No medals.",
    },
    "match_unknown": {
        "fr": "Match inconnu",
        "en": "Unknown match",
    },
    # ── Erreurs d'affichage ─────────────────────────────────────────────────
    "cannot_display": {
        "fr": "Impossible d'afficher ce graphique.",
        "en": "Unable to display this chart.",
    },
    "cannot_generate": {
        "fr": "Impossible de générer ce graphique.",
        "en": "Unable to generate this chart.",
    },
    "error_chart": {
        "fr": "Erreur lors de l'affichage du graphique : {error}",
        "en": "Error while displaying the chart: {error}",
    },
    "error_generic": {
        "fr": "Erreur : {error}",
        "en": "Error: {error}",
    },
    "error_loading": {
        "fr": "Erreur lors du chargement : {error}",
        "en": "Error while loading: {error}",
    },
    # ── Actions ─────────────────────────────────────────────────────────────
    "loading": {
        "fr": "Chargement…",
        "en": "Loading…",
    },
    "computing": {
        "fr": "Calcul en cours…",
        "en": "Computing…",
    },
    "syncing": {
        "fr": "Synchronisation en cours…",
        "en": "Sync in progress…",
    },
    "cache_cleared": {
        "fr": "Caches vidés.",
        "en": "Caches cleared.",
    },
    "not_associated": {
        "fr": "Match: non associé",
        "en": "Match: unassociated",
    },
    "no_thumbnail": {
        "fr": "(pas de miniature générée)",
        "en": "(no thumbnail generated)",
    },
    # ── Données temporelles ─────────────────────────────────────────────────
    "missing_time_data": {
        "fr": "Données temporelles manquantes.",
        "en": "Missing time data.",
    },
    "missing_outcome_data": {
        "fr": "Données de résultat manquantes.",
        "en": "Missing outcome data.",
    },
    # ── Minimum de matchs ───────────────────────────────────────────────────
    "not_enough_matches": {
        "fr": "Pas assez de données ({count} matchs). Il en faut au moins {min} pour afficher ce graphique.",
        "en": "Not enough data ({count} matches). Need at least {min} to display this chart.",
    },
    "at_least_n_matches": {
        "fr": "Au moins {n} matchs requis.",
        "en": "At least {n} matches required.",
    },
    # ── Médias ──────────────────────────────────────────────────────────────
    "file_missing": {
        "fr": "Fichier absent : {name}",
        "en": "Missing file: {name}",
    },
    "thumbnail_too_large": {
        "fr": "Miniature trop volumineuse : {name}",
        "en": "Thumbnail too large: {name}",
    },
    "media_disabled": {
        "fr": "Les médias sont désactivés dans Paramètres → Médias.",
        "en": "Media is disabled in Settings → Media.",
    },
    "media_no_folder": {
        "fr": "Configure au moins un dossier dans Paramètres → Médias (captures et/ou vidéos).",
        "en": "Configure at least one folder in Settings → Media (screenshots and/or videos).",
    },
    "media_not_found": {
        "fr": "Aucun média trouvé.",
        "en": "No media found.",
    },
    "indexing_error": {
        "fr": "Erreur lors de l'indexation : {error}",
        "en": "Indexing error: {error}",
    },
    # ── Résultats de match ──────────────────────────────────────────────────
    "outcome_win": {"fr": "Victoire", "en": "Win"},
    "outcome_loss": {"fr": "Défaite", "en": "Loss"},
    "outcome_draw": {"fr": "Égalité", "en": "Draw"},
    "outcome_dnf": {"fr": "Non terminé", "en": "Did Not Finish"},
    "outcome_unknown": {"fr": "?", "en": "?"},
    # ── Colonnes / métriques communes ───────────────────────────────────────
    "col_kills": {"fr": "Frags", "en": "Kills"},
    "col_deaths": {"fr": "Morts", "en": "Deaths"},
    "col_assists": {"fr": "Assistances", "en": "Assists"},
    "col_kda": {"fr": "FDA", "en": "KDA"},
    "col_accuracy": {"fr": "Précision", "en": "Accuracy"},
    "col_accuracy_pct": {"fr": "Précision (%)", "en": "Accuracy (%)"},
    "col_score": {"fr": "Score", "en": "Score"},
    "col_rank": {"fr": "Rang", "en": "Rank"},
    "col_player": {"fr": "Joueur", "en": "Player"},
    "col_date": {"fr": "Date", "en": "Date"},
    "col_time": {"fr": "Heure", "en": "Time"},
    "col_performance": {"fr": "Performance", "en": "Performance"},
    "col_map": {"fr": "Carte", "en": "Map"},
    "col_maps": {"fr": "Cartes", "en": "Maps"},
    "col_mode": {"fr": "Mode", "en": "Mode"},
    "col_modes": {"fr": "Modes", "en": "Modes"},
    "col_playlist": {"fr": "Playlist", "en": "Playlist"},
    "col_playlists": {"fr": "Playlists", "en": "Playlists"},
    "col_result": {"fr": "Résultat", "en": "Result"},
    "col_matches": {"fr": "Matchs", "en": "Matches"},
    "col_match": {"fr": "Match", "en": "Match"},
    "col_duration": {"fr": "Durée", "en": "Duration"},
    "col_start_date": {"fr": "Date de début", "en": "Start date"},
    "col_mmr_team": {"fr": "MMR équipe", "en": "Team MMR"},
    "col_mmr_enemy": {"fr": "MMR adverse", "en": "Enemy MMR"},
    "col_mmr_gap": {"fr": "Écart MMR", "en": "MMR gap"},
    "col_max_spree": {"fr": "Tuerie (max)", "en": "Spree (max)"},
    "col_headshots": {"fr": "Têtes", "en": "Headshots"},
    "col_avg_life": {"fr": "Durée de vie", "en": "Avg life"},
    "col_avg_life_long": {"fr": "Durée de vie moyenne", "en": "Average life span"},
    "col_kpm": {"fr": "Frags / min", "en": "Kills / min"},
    "col_dpm": {"fr": "Morts / min", "en": "Deaths / min"},
    "col_apm": {"fr": "Assistances / min", "en": "Assists / min"},
    "col_kills_per_match": {"fr": "Frags par partie", "en": "Kills per match"},
    "col_deaths_per_match": {"fr": "Morts par partie", "en": "Deaths per match"},
    "col_assists_per_match": {"fr": "Assistances par partie", "en": "Assists per match"},
    "col_win_rate": {"fr": "Taux de victoire", "en": "Win rate"},
    "col_loss_rate": {"fr": "Taux de défaite", "en": "Loss rate"},
    "col_ratio": {"fr": "Ratio", "en": "Ratio"},
    "col_win_rate_pct": {"fr": "Taux victoire (%)", "en": "Win rate (%)"},
    "col_loss_rate_pct": {"fr": "Taux défaite (%)", "en": "Loss rate (%)"},
    "col_global_ratio": {"fr": "Ratio global", "en": "Overall ratio"},
    "col_shots_fired": {"fr": "Tirs", "en": "Shots fired"},
    "col_shots_hit": {"fr": "Tirs au but", "en": "Shots hit"},
    "col_melee": {"fr": "Corps à corps", "en": "Melee"},
    "col_power_weapon": {"fr": "Armes lourdes", "en": "Power weapons"},
    "col_dmg_dealt": {"fr": "Dégâts infligés", "en": "Damage dealt"},
    "col_dmg_taken": {"fr": "Dégâts subis", "en": "Damage taken"},
    "col_perfect_kills": {"fr": "Frags parfaits", "en": "Perfect kills"},
    "col_killing_spree": {"fr": "Folie meurtrière", "en": "Killing spree"},
    "col_total_time": {"fr": "Temps joué", "en": "Time played"},
    "col_avg_duration": {"fr": "Durée moyenne / match", "en": "Avg duration / match"},
    "col_total_duration": {"fr": "Durée totale", "en": "Total duration"},
    "col_avg_accuracy": {"fr": "Précision moyenne", "en": "Average accuracy"},
    "col_avg_accuracy_pct": {"fr": "Précision moy. (%)", "en": "Avg accuracy (%)"},
    "col_avg_perf": {"fr": "Performance moy.", "en": "Avg performance"},
    # ── Actions génériques ──────────────────────────────────────────────────
    "btn_open": {"fr": "Ouvrir", "en": "Open"},
    "btn_search": {"fr": "Rechercher", "en": "Search"},
    "btn_confirm": {"fr": "Confirmer", "en": "Confirm"},
    "btn_cancel": {"fr": "Annuler", "en": "Cancel"},
    "btn_download_csv": {"fr": "Télécharger CSV", "en": "Download CSV"},
    "btn_save": {"fr": "Enregistrer", "en": "Save"},
    "btn_reload": {"fr": "Recharger depuis fichier", "en": "Reload from file"},
    "btn_select_all": {"fr": "✓ Tout", "en": "✓ All"},
    "btn_select_none": {"fr": "✗ Aucun", "en": "✗ None"},
    "btn_use_match": {"fr": "Utiliser ce match", "en": "Use this match"},
    # ── Jours de la semaine abrégés ─────────────────────────────────────────
    "day_mon": {"fr": "lun.", "en": "Mon"},
    "day_tue": {"fr": "mar.", "en": "Tue"},
    "day_wed": {"fr": "mer.", "en": "Wed"},
    "day_thu": {"fr": "jeu.", "en": "Thu"},
    "day_fri": {"fr": "ven.", "en": "Fri"},
    "day_sat": {"fr": "sam.", "en": "Sat"},
    "day_sun": {"fr": "dim.", "en": "Sun"},
    # ── Divers ──────────────────────────────────────────────────────────────
    "unknown_map": {"fr": "Carte inconnue", "en": "Unknown map"},
    "unranked": {"fr": "Non classé", "en": "Unranked"},
    "ranked": {"fr": "Classé", "en": "Ranked"},
    "lbl_hour": {"fr": "Heure", "en": "Hour"},
    "lbl_actual": {"fr": "Réel", "en": "Actual"},
    "lbl_expected": {"fr": "Attendu", "en": "Expected"},
    "lbl_several": {"fr": "Plusieurs", "en": "Multiple"},
    "lbl_parties": {"fr": "Parties", "en": "Matches"},
    "lbl_total_matches": {"fr": "parties", "en": "matches"},
    "lbl_all_sessions": {"fr": "toutes sessions", "en": "all sessions"},
    "lbl_nemesis": {"fr": "Némésis", "en": "Nemesis"},
    "lbl_victim": {"fr": "Souffre-douleur", "en": "Punching bag"},
    "lbl_validated": {"fr": "Validé", "en": "Verified"},
    "lbl_not_validated": {"fr": "Non validé", "en": "Not verified"},
    "sel_all": {"fr": "(toutes)", "en": "(all)"},
    "sel_none": {"fr": "(aucun)", "en": "(none)"},
    # col_killing_spree, col_shots_hit : déjà définis plus haut (L193/L199)
    "col_shots_fired_short": {"fr": "Tirs", "en": "Shots"},
    "col_assists_short": {"fr": "Assist.", "en": "Assists"},
    "lbl_you": {"fr": "Toi", "en": "You"},
}
