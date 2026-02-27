"""Chaînes de titres et messages spécifiques aux pages Streamlit.

Chaque section correspond à une page ou un sous-groupe de sections.

⚠️ ChatGPT : remplir toutes les valeurs marquées "TODO" ci-dessous.
   Règles : voir le prompt de la Phase 1b dans le plan i18n.
"""

from __future__ import annotations

STRINGS: dict[str, dict[str, str]] = {
    # ── Page Victoires/Défaites ──────────────────────────────────────────────
    "wl_results_by_map_mode": {
        "fr": "Résultats par carte et mode",
        "en": "Results by map and mode",
    },
    "wl_by_map": {
        "fr": "Par carte",
        "en": "By map",
    },
    "wl_by_mode": {
        "fr": "Par mode",
        "en": "By mode",
    },
    "wl_heatmap_title": {
        "fr": "Win Rate par jour et heure",
        "en": "Win rate by day and hour",
    },
    "wl_heatmap_caption": {
        "fr": "Identifie les créneaux horaires où tu performes le mieux. Les cellules affichent le nombre de matchs.",
        "en": "Find the time slots where you perform best. Cells show the number of matches.",
    },
    "wl_top_by_week": {
        "fr": "Matchs Top vs Total par semaine",
        "en": "Top matches vs total per week",
    },
    "wl_top_by_week_caption": {
        "fr": 'Compare le nombre de matchs où tu as terminé en tête (rang 1) par rapport au total. La ligne indique le taux de "Top 1".',
        "en": "Compare how many matches you finished on top (rank 1) versus the total. The line shows the Top 1 rate.",
    },
    "wl_streaks": {
        "fr": "Séries de victoires / défaites",
        "en": "Win / loss streaks",
    },
    "wl_streaks_caption": {
        "fr": "Visualise les séries consécutives de victoires (barres positives) et de défaites (barres négatives). Les séries longues indiquent les phases de momentum positif ou négatif.",
        "en": "Visualize consecutive win streaks (positive bars) and loss streaks (negative bars). Long streaks highlight momentum swings.",
    },
    "wl_personal_score": {
        "fr": "Score personnel par match",
        "en": "Personal score per match",
    },
    "wl_personal_score_caption": {
        "fr": "Barres colorées du score personnel pour chaque match, avec courbe de moyenne lissée.",
        "en": "Colored bars for personal score on each match, with a smoothed average line.",
    },
    "wl_period": {
        "fr": "Par période",
        "en": "By period",
    },
    "wl_ratio_by_map": {
        "fr": "Ratio par cartes",
        "en": "Ratio by map",
    },
    "wl_ratio_caption": {
        "fr": "Compare tes performances par map.",
        "en": "Compare your performance across maps.",
    },
    "wl_no_period_data": {
        "fr": "Aucune donnée pour construire le tableau.",
        "en": "No data to build the table.",
    },
    "wl_cannot_display_evolution": {
        "fr": "Impossible d'afficher l'évolution des résultats : {error}",
        "en": "Unable to display the results evolution: {error}",
    },
    "wl_insufficient_evolution": {
        "fr": "Données insuffisantes pour afficher l'évolution des résultats.",
        "en": "Not enough data to display the results evolution.",
    },
    "wl_period_default": {
        "fr": "période",
        "en": "period",
    },
    # ── Page Carrière ────────────────────────────────────────────────────────
    "career_header": {
        "fr": "Carrière",
        "en": "Career",
    },
    "career_no_data": {
        "fr": "Aucune donnée de carrière disponible. Synchronisez vos données pour voir votre progression de rang.",
        "en": "No career data available. Sync your data to see your rank progression.",
    },
    "career_rank_max": {
        "fr": "Rang maximum",
        "en": "Max rank",
    },
    "career_progression_to_hero": {
        "fr": "Progression vers Héros",
        "en": "Progression to Hero",
    },
    "career_rank_history_title": {
        "fr": "Historique détaillé",
        "en": "Detailed history",
    },
    "career_rank_history_no_data": {
        "fr": "Pas assez de données pour afficher l'historique.",
        "en": "Not enough data to display the history.",
    },
    "career_lusr_snapshot_title": {
        "fr": "📊 Classement actuel (instantané)",
        "en": "📊 Current rating snapshot",
    },
    "career_lusr_no_rating": {
        "fr": "Aucun rating LUSR/CSR calculé. Utilisez `--lusr` (non classé) ou `--csr` (classé) pour calculer.",
        "en": "No LUSR/CSR rating computed. Run `--lusr` (unranked) or `--csr` (ranked) to compute it.",
    },
    "career_lusr_rating_evolution": {
        "fr": "📈 Évolution du rating",
        "en": "📈 Rating over time",
    },
    "career_lusr_group_select": {
        "fr": "Groupe :",
        "en": "Group:",
    },
    "career_lusr_all_groups": {
        "fr": "Tous les groupes",
        "en": "All groups",
    },
    "career_gauge_error": {
        "fr": "Impossible d'afficher la jauge de progression : {error}",
        "en": "Unable to display the progression gauge: {error}",
    },
    "career_gauge_generate_error": {
        "fr": "Impossible de générer la jauge de progression.",
        "en": "Unable to generate the progression gauge.",
    },
    "career_hero_progress_error": {
        "fr": "Impossible d'afficher la progression vers Héros : {error}",
        "en": "Unable to display progression to Hero: {error}",
    },
    "career_history_error": {
        "fr": "Impossible d'afficher l'historique de progression : {error}",
        "en": "Unable to display progression history: {error}",
    },
    "career_lusr_error": {
        "fr": "Impossible d'afficher le graphe LUSR global : {error}",
        "en": "Unable to display the global LUSR chart: {error}",
    },
    "career_lusr_group_error": {
        "fr": "Impossible d'afficher le graphe : {error}",
        "en": "Unable to display the chart: {error}",
    },
    # ── Métriques carrière (v5.3) ────────────────────────────────────────────
    "career_metric_rank": {
        "fr": "Rang",
        "en": "Rank",
    },
    "career_metric_xp_total": {
        "fr": "XP total",
        "en": "Total XP",
    },
    "career_metric_current_xp": {
        "fr": "XP actuel",
        "en": "Current XP",
    },
    "career_metric_next_rank_xp": {
        "fr": "XP prochain rang",
        "en": "Next rank XP",
    },
    "career_metric_xp_earned": {
        "fr": "XP gagnée",
        "en": "XP earned",
    },
    "career_metric_xp_remaining": {
        "fr": "XP restante",
        "en": "XP remaining",
    },
    "career_metric_xp_required": {
        "fr": "Total requis",
        "en": "Required total",
    },
    # ── Page Timeseries ──────────────────────────────────────────────────────
    "ts_fda": {
        "fr": "FDA",
        "en": "KDA",
    },
    "ts_fda_unavailable": {
        "fr": "FDA indisponible sur ce filtre.",
        "en": "KDA is unavailable for this filter.",
    },
    "ts_cumulative": {
        "fr": "Performance cumulée & tendance",
        "en": "Cumulative performance & trend",
    },
    "ts_cumulative_caption": {
        "fr": "Net score et F/M cumulé au fil des matchs, F/M glissant, et tendance (début vs fin de période).",
        "en": "Net score and cumulative K/D over matches, rolling K/D, and trend (start vs end of the period).",
    },
    "ts_distributions": {
        "fr": "Distributions",
        "en": "Distributions",
    },
    "ts_distributions_caption": {
        "fr": "Visualise la répartition de tes statistiques.",
        "en": "Visualize the spread of your stats.",
    },
    "ts_correlations": {
        "fr": "Corrélations",
        "en": "Correlations",
    },
    "ts_correlations_caption": {
        "fr": "Analyse les relations entre tes métriques et le résultat du match.",
        "en": "Analyze relationships between your metrics and match outcomes.",
    },
    "ts_first_event": {
        "fr": "Temps du premier frag / première mort",
        "en": "Time to first kill / first death",
    },
    "ts_first_event_caption": {
        "fr": "Distribution des timestamps du premier frag et de la première mort. Visualise à quelle vitesse tu obtiens ton premier frag vs ta première mort.",
        "en": "Distribution of timestamps for your first kill and first death. See how fast you get your first kill vs your first death.",
    },
    "ts_first_event_no_data": {
        "fr": "Données d'événements non disponibles (premier frag / première mort). L'**Actualiser** récupère déjà ces données pour les **nouveaux** matchs. Pour les matchs déjà en base sans événements film, active dans **Paramètres** → **Options du bouton Actualiser** l'option **Backfill events**, puis **Actualiser**.",
        "en": "Event data is not available (first kill / first death). **Refresh** already fetches these for **new** matches. For existing matches without film events, enable **Backfill events** in **Settings** → **Refresh button options**, then **Refresh**.",
    },
    "ts_performance": {
        "fr": "Performance",
        "en": "Performance",
    },
    "ts_assists": {
        "fr": "Assistances",
        "en": "Assists",
    },
    "ts_per_minute": {
        "fr": "Stats par minute",
        "en": "Per-minute stats",
    },
    "ts_lifespan": {
        "fr": "Durée de vie moyenne",
        "en": "Average life span",
    },
    "ts_lifespan_unavailable": {
        "fr": "Average Life indisponible sur ce filtre.",
        "en": "Average Life is unavailable for this filter.",
    },
    "ts_spree": {
        "fr": "Folie meurtrière / Tirs à la tête / Frags parfaits",
        "en": "Killing Spree / Headshots / Perfect Kills",
    },
    "ts_shots": {
        "fr": "Tirs et précision",
        "en": "Shots & accuracy",
    },
    "ts_shots_caption": {
        "fr": "Tirs tirés vs touchés (barres groupées) et courbe de précision. La précision a été retirée du graphe Folie meurtrière pour une lecture plus claire.",
        "en": "Shots fired vs shots hit (grouped bars) and an accuracy line. Accuracy was removed from the Killing Spree chart for readability.",
    },
    "ts_damage": {
        "fr": "Dégâts",
        "en": "Damage",
    },
    "ts_damage_caption": {
        "fr": "Compare les dégâts infligés et subis par match. Un ratio élevé (infligés > subis) indique une bonne efficacité au combat.",
        "en": "Compare damage dealt and damage taken per match. A high ratio (dealt > taken) usually means strong combat efficiency.",
    },
    "ts_rank_score": {
        "fr": "Rang et score personnel",
        "en": "Rank and personal score",
    },
    "ts_rank_score_caption": {
        "fr": "Le score personnel en barres et le rang en ligne (axe Y inversé : rang 1 en haut). Un bon score associé à un rang élevé confirme l'impact.",
        "en": "Personal score as bars and rank as a line (inverted Y axis: rank 1 at the top). High score + high placement confirms impact.",
    },
    "ts_trend_min_matches": {
        "fr": "Tendance de session : au moins 4 matchs requis.",
        "en": "Session trend: at least 4 matches required.",
    },
    # ── Page Coéquipiers ─────────────────────────────────────────────────────
    "tm_by_map": {
        "fr": "Par carte — avec mes coéquipiers",
        "en": "By map — with my teammates",
    },
    "tm_history": {
        "fr": "Historique — matchs avec mes coéquipiers",
        "en": "History — matches with my teammates",
    },
    "tm_medals": {
        "fr": "Médailles (matchs partagés)",
        "en": "Medals (shared matches)",
    },
    "tm_medals_all": {
        "fr": "Médailles",
        "en": "Medals",
    },
    "tm_per_minute": {
        "fr": "Stats par minute",
        "en": "Per-minute stats",
    },
    "tm_no_matches_teammate": {
        "fr": "Aucun match trouvé avec ce coéquipier (selon le filtre).",
        "en": "No matches found with this teammate (based on the current filter).",
    },
    "tm_not_enough_matches": {
        "fr": "Pas assez de matchs avec tes coéquipiers (selon le filtre actuel).",
        "en": "Not enough matches with your teammates (based on the current filter).",
    },
    "tm_no_matches_filter": {
        "fr": "Aucun match trouvé avec tes coéquipiers (selon le filtre actuel).",
        "en": "No matches found with your teammates (based on the current filter).",
    },
    "tm_no_shared_medals": {
        "fr": "Aucun match partagé pour calculer les médailles.",
        "en": "No shared matches to compute medals.",
    },
    "tm_no_medals_aggregate": {
        "fr": "Impossible de déterminer la liste des matchs pour l'agrégation des médailles.",
        "en": "Unable to determine the match list for medals aggregation.",
    },
    "tm_trio_session": {
        "fr": "Dernière session trio détectée : {label}.",
        "en": "Latest trio session detected: {label}.",
    },
    "tm_trio_session_unknown": {
        "fr": "Impossible de déterminer une session trio (données insuffisantes).",
        "en": "Unable to determine a trio session (not enough data).",
    },
    "tm_impact_header": {
        "fr": "⚡ Impact",
        "en": "⚡ Impact",
    },
    "tm_impact_select_two": {
        "fr": "Sélectionnez au moins 2 coéquipiers pour voir l'analyse d'impact.",
        "en": "Select at least 2 teammates to see the impact analysis.",
    },
    "tm_impact_no_matches": {
        "fr": "Aucun match à analyser.",
        "en": "No matches to analyze.",
    },
    "tm_impact_no_events_matches": {
        "fr": "Aucun événement trouvé pour les matchs sélectionnés.",
        "en": "No impact events found for the selected matches.",
    },
    "tm_impact_no_events_players": {
        "fr": "Aucun événement d'impact trouvé pour les joueurs sélectionnés.",
        "en": "No impact events found for the selected players.",
    },
    "tm_impact_heatmap": {
        "fr": "Heatmap d'Impact",
        "en": "Impact heatmap",
    },
    "tm_impact_ranking": {
        "fr": "🏆 Classement",
        "en": "🏆 Ranking",
    },
    "tm_impact_legend": {
        "fr": "⚡ Premier sang | 🎯 Finisseur | 💀 Boulet | 🐌 Plus lent | 🪦 Première victime",
        "en": "⚡ First Blood | 🎯 Finisher | 💀 Dead Weight | 🐌 Slowest | 🪦 First Victim",
    },
    # ── Page Match View ──────────────────────────────────────────────────────
    "mv_citations": {
        "fr": "Citations",
        "en": "Commendations",
    },
    "mv_medals": {
        "fr": "Médailles",
        "en": "Medals",
    },
    "mv_lusr_no_data": {
        "fr": "Aucun rating LUSR/CSR calculé pour ce match. Lancez `--lusr` (non classé) ou `--csr` (classé) pour calculer.",
        "en": "No LUSR/CSR rating computed for this match. Run `--lusr` (unranked) or `--csr` (ranked) to compute it.",
    },
    "mv_lusr_modules_missing": {
        "fr": "Modules LUSR non disponibles.",
        "en": "LUSR modules are not available.",
    },
    # ── Page Médias ──────────────────────────────────────────────────────────
    "media_library_title": {
        "fr": "Bibliothèque médias",
        "en": "Media library",
    },
    "media_unassociated": {
        "fr": "Non associés",
        "en": "Unassociated",
    },
    "media_rescan": {
        "fr": "Re-scanner les dossiers",
        "en": "Rescan folders",
    },
    "media_error_indexing": {
        "fr": "Erreur lors de l'indexation : {error}",
        "en": "Indexing error: {error}",
    },
    "media_configure_video": {
        "fr": "Configure un dossier vidéos dans Paramètres → Médias.",
        "en": "Configure a videos folder in Settings → Media.",
    },
    # ── Page Paramètres ──────────────────────────────────────────────────────
    "settings_title": {
        "fr": "Paramètres",
        "en": "Settings",
    },
    "settings_backfill_data_label": {
        "fr": "Données à backfill :",
        "en": "Data to backfill:",
    },
    "backfill_all_data": {"fr": "Toutes les données", "en": "All data"},
    # ── Page Analyse Objectifs ───────────────────────────────────────────────
    "obj_analysis_title": {
        "fr": "📊 Analyse des Objectifs",
        "en": "📊 Objective analysis",
    },
    "obj_comparison_coming_soon": {
        "fr": "🔜 Comparaison (à venir)",
        "en": "🔜 Comparison (coming soon)",
    },
    # ── Page Session Compare ─────────────────────────────────────────────────
    "sc_performance_score": {
        "fr": "### 🏆 Score de performance",
        "en": "### 🏆 Performance score",
    },
    "sc_detailed_metrics": {
        "fr": "### 📊 Métriques détaillées",
        "en": "### 📊 Detailed metrics",
    },
    "sc_mmr_comparison": {
        "fr": "### 🎯 Comparaison MMR",
        "en": "### 🎯 MMR comparison",
    },
    "sc_comparative_charts": {
        "fr": "### 📈 Graphiques comparatifs",
        "en": "### 📈 Comparative charts",
    },
    "sc_match_history": {
        "fr": "### 📋 Historique des parties",
        "en": "### 📋 Match history",
    },
    "sc_participation_profile": {
        "fr": "#### 🎯 Évolution du profil de participation",
        "en": "#### 🎯 Participation profile over time",
    },
    "sc_radar_view": {
        "fr": "#### Vue radar",
        "en": "#### Radar view",
    },
    "sc_metric_comparison": {
        "fr": "#### Comparaison par métrique",
        "en": "#### Metric comparison",
    },
    "sc_metric_label": {
        "fr": "**Métrique**",
        "en": "**Metric**",
    },
    "sc_mmr_metric_label": {
        "fr": "**Métrique MMR**",
        "en": "**MMR metric**",
    },
    "sc_net_score_cumul": {
        "fr": "#### Net score cumulé par session",
        "en": "#### Cumulative net score per session",
    },
    # ── Page Dernière partie ─────────────────────────────────────────────────
    "last_match_search_by_datetime": {
        "fr": "Recherche par date/heure",
        "en": "Search by date/time",
    },
    # ── Multiplayer ──────────────────────────────────────────────────────────
    "multiplayer_player_label": {
        "fr": "#### 👥 Joueur",
        "en": "#### 👥 Player",
    },
    # ── OpenSpartan ──────────────────────────────────────────────────────────
    "openspartan_exe_not_found": {
        "fr": "Executable introuvable. Tu peux définir OPENSPARTAN_WORKSHOP_EXE si besoin.",
        "en": "Executable not found. You can set OPENSPARTAN_WORKSHOP_EXE if needed.",
    },
    "openspartan_launched": {
        "fr": "OpenSpartan Workshop lancé.",
        "en": "OpenSpartan Workshop launched.",
    },
    "openspartan_launch_error": {
        "fr": "Impossible de lancer OpenSpartan Workshop : {error}",
        "en": "Unable to launch OpenSpartan Workshop: {error}",
    },
    # ── Page Timeseries — spinners & labels manquants ─────────────────────────
    "ts_computing": {
        "fr": "Génération des graphes…",
        "en": "Generating charts…",
    },
    "ts_kda_mean_label": {
        "fr": "FDA moyen",
        "en": "Avg. KDA",
    },
    "ts_dist_accuracy_title": {
        "fr": "Distribution de la Précision",
        "en": "Accuracy distribution",
    },
    "ts_dist_kills_title": {
        "fr": "Distribution des Frags",
        "en": "Kill distribution",
    },
    "ts_dist_life_title": {
        "fr": "Distribution Durée de Vie",
        "en": "Life span distribution",
    },
    "ts_dist_perf_title": {
        "fr": "Distribution Score de Performance",
        "en": "Performance score distribution",
    },
    "ts_dist_score_per_min_title": {
        "fr": "Distribution Score Personnel / min",
        "en": "Personal score / min distribution",
    },
    "ts_dist_win_rate_title": {
        "fr": "Distribution Win Rate Glissant (5 matchs)",
        "en": "Rolling win rate distribution (5 matches)",
    },
    "ts_accuracy_label": {
        "fr": "Précision (%)",
        "en": "Accuracy (%)",
    },
    "ts_kills_label": {
        "fr": "Frags",
        "en": "Kills",
    },
    "ts_life_label": {
        "fr": "Durée (secondes)",
        "en": "Duration (seconds)",
    },
    "ts_score_label": {
        "fr": "Score",
        "en": "Score",
    },
    "ts_matches_label": {
        "fr": "Matchs",
        "en": "Matches",
    },
    "ts_score_per_min_label": {
        "fr": "Score / min",
        "en": "Score / min",
    },
    "ts_win_rate_label": {
        "fr": "Taux de victoire (%)",
        "en": "Win rate (%)",
    },
    "ts_frequency_label": {
        "fr": "Fréquence",
        "en": "Frequency",
    },
    "ts_col_missing_cumul": {
        "fr": "Colonnes start_time, kills ou deaths manquantes pour la performance cumulée.",
        "en": "Columns start_time, kills or deaths missing for cumulative performance.",
    },
    "ts_col_missing_score_per_min": {
        "fr": "Colonnes score personnel ou time_played non disponibles.",
        "en": "Columns personal_score or time_played not available.",
    },
    "ts_col_missing_outcome": {
        "fr": "Colonne outcome non disponible.",
        "en": "Column outcome not available.",
    },
    "ts_insufficient_win_rate": {
        "fr": "Au moins 5 matchs requis pour le win rate glissant.",
        "en": "At least 5 matches required for rolling win rate.",
    },
    "ts_insufficient_win_rate_dist": {
        "fr": "Pas assez de données pour la distribution du win rate glissant (5 matchs minimum par fenêtre).",
        "en": "Not enough data for rolling win rate distribution (5 matches minimum per window).",
    },
    "ts_insufficient_score_per_min": {
        "fr": "Pas assez de données pour la distribution du score par minute.",
        "en": "Not enough data for score per minute distribution.",
    },
    "ts_corr_gen_error": {
        "fr": "Impossible de générer la corrélation {title}.",
        "en": "Unable to generate the {title} correlation.",
    },
    # ── Page Win/Loss — spinners & labels manquants ───────────────────────────
    "wl_computing": {
        "fr": "Calcul des victoires/défaites…",
        "en": "Computing wins/losses…",
    },
    "wl_bucket_intro": {
        "fr": "Par **{bucket}** : on regroupe les parties par {bucket} et on compte le nombre de victoires/défaites (et autres statuts) pour suivre l'évolution.",
        "en": "By **{bucket}**: grouping matches by {bucket} and counting wins/losses (and other statuses) to track evolution.",
    },
    "wl_scope_label": {
        "fr": "Scope",
        "en": "Scope",
    },
    "wl_scope_me_filtered": {
        "fr": "Moi (filtres actuels)",
        "en": "Me (current filters)",
    },
    "wl_scope_me_all": {
        "fr": "Moi (toutes les parties)",
        "en": "Me (all matches)",
    },
    "wl_min_matches_map_slider": {
        "fr": "Minimum de matchs par carte",
        "en": "Minimum matches per map",
    },
    "wl_computing_map": {
        "fr": "Calcul des stats par carte…",
        "en": "Computing map stats…",
    },
    "wl_not_enough_map": {
        "fr": "Pas assez de matchs par map avec ces filtres.",
        "en": "Not enough matches per map with these filters.",
    },
    "wl_metric_label": {
        "fr": "Métrique",
        "en": "Metric",
    },
    "wl_metric_ratio": {
        "fr": "Ratio Victoire/défaite",
        "en": "Win/Loss Ratio",
    },
    "wl_metric_win_rate": {
        "fr": "Taux de victoires",
        "en": "Win rate",
    },
    "wl_metric_accuracy": {
        "fr": "Précision moyenne",
        "en": "Avg. accuracy",
    },
    "wl_personal_score_y_axis": {
        "fr": "Score personnel",
        "en": "Personal score",
    },
    "wl_personal_score_hover": {
        "fr": "Score",
        "en": "Score",
    },
    # ── Page Carrière — manquants ─────────────────────────────────────────────
    "career_computing": {
        "fr": "Chargement des données de carrière…",
        "en": "Loading career data…",
    },
    # ── Page Analyse Objectifs — manquants ────────────────────────────────────
    "obj_loading": {
        "fr": "Chargement des données…",
        "en": "Loading data…",
    },
    "obj_polars_missing": {
        "fr": "⚠️ Cette page nécessite Polars. Installez-le avec: `pip install polars`",
        "en": "⚠️ This page requires Polars. Install it with: `pip install polars`",
    },
    "obj_assists_caption": {
        "fr": "Décomposition des différents types d'assistances.",
        "en": "Breakdown of different assistance types.",
    },
    "obj_top_opponents_caption": {
        "fr": "Top joueurs rencontrés par contribution aux objectifs.",
        "en": "Top players encountered by objective contribution.",
    },
    "obj_no_assists": {
        "fr": "Aucune donnée d'assistance disponible.",
        "en": "No assistance data available.",
    },
    "obj_no_awards": {
        "fr": "Aucun award objectif enregistré.",
        "en": "No objective awards recorded.",
    },
    "obj_no_awards_generic": {
        "fr": "Aucun award enregistré.",
        "en": "No awards recorded.",
    },
    # ── Page Session Compare — manquants ─────────────────────────────────────
    "sc_no_sessions": {
        "fr": "Aucune session disponible.",
        "en": "No sessions available.",
    },
    "sc_need_two_sessions": {
        "fr": "Il faut au moins 2 sessions pour comparer.",
        "en": "At least 2 sessions needed to compare.",
    },
    "sc_loading_caption": {
        "fr": "Compare les performances entre deux sessions de jeu.",
        "en": "Compare performance between two game sessions.",
    },
    # ── Page Coéquipiers — manquants ─────────────────────────────────────────
    "tm_computing_teammate": {
        "fr": "Chargement des matchs avec ce coéquipier…",
        "en": "Loading matches with this teammate…",
    },
    "tm_computing_map": {
        "fr": "Calcul du ratio par carte (coéquipiers)…",
        "en": "Computing map ratio (teammates)…",
    },
    "tm_computing_stats": {
        "fr": "Chargement des stats des coéquipiers…",
        "en": "Loading teammate stats…",
    },
    "tm_computing_medals": {
        "fr": "Agrégation des médailles (moi + coéquipier)…",
        "en": "Aggregating medals (me + teammate)…",
    },
    "tm_computing_medals_all": {
        "fr": "Agrégation des médailles…",
        "en": "Aggregating medals…",
    },
    "tm_trio_header": {
        "fr": "Tous les trois — {f1} + {f2}",
        "en": "All three — {f1} + {f2}",
    },
    # ── Page Match View — manquants ───────────────────────────────────────────
    "mv_loading": {
        "fr": "Lecture des stats détaillées (attendu vs réel, médailles)…",
        "en": "Loading detailed stats (expected vs actual, medals)…",
    },
    "mv_playlist_group_caption": {
        "fr": "Groupe : {group}",
        "en": "Group: {group}",
    },
    # Nouveau format "LUSR Arena : 1528" — inclut type, groupe traduit et valeur
    "mv_playlist_group_rating_label": {
        "fr": "{type} {group} : {value}",
        "en": "{type} {group}: {value}",
    },
    # Noms traduits des groupes de playlist
    "mv_pg_ranked": {"fr": "Classé", "en": "Ranked"},
    "mv_pg_arena": {"fr": "Arena", "en": "Arena"},
    "mv_pg_btb": {"fr": "Grand Combat", "en": "Big Team Battle"},
    "mv_pg_tactical": {"fr": "Tactique", "en": "Tactical"},
    "mv_pg_social": {"fr": "Social", "en": "Social"},
    "mv_pg_fun": {"fr": "Fun", "en": "Fun"},
    # Barre de progression dans le sous-tier
    "mv_sub_tier_progress": {
        "fr": "Progression dans {tier_name} {sub_tier} : {pts_earned} / {tier_size} pts",
        "en": "Progress in {tier_name} {sub_tier}: {pts_earned} / {tier_size} pts",
    },
    "mv_citations_unavailable": {
        "fr": "Référentiel Citations indisponible.",
        "en": "Citations reference unavailable.",
    },
    "mv_citations_no_data": {
        "fr": "Citations indisponibles pour ce match.",
        "en": "No citations available for this match.",
    },
    # ── Page Médias — manquants ───────────────────────────────────────────────
    "media_options_expander": {
        "fr": "Options",
        "en": "Options",
    },
    "media_scanning": {
        "fr": "Indexation en cours…",
        "en": "Scanning…",
    },
    "media_generating_thumbnails": {
        "fr": "Génération des thumbnails…",
        "en": "Generating thumbnails…",
    },
    # ── Citations ─────────────────────────────────────────────────────────────
    "citations_halo5_title": {
        "fr": "Citations",
        "en": "Commendations",
    },
    "citations_medals_title": {
        "fr": "Médailles",
        "en": "Medals",
    },
    "citations_medals_caption": {
        "fr": "Médailles sur la sélection/filtres actuels.",
        "en": "Medals for the current selection/filters.",
    },
    "citations_no_medals": {
        "fr": "Aucune médaille trouvée (ou payload médailles absent).",
        "en": "No medals found (or medal payload missing).",
    },
    "citations_medals_distribution": {
        "fr": "Distribution des médailles",
        "en": "Medal distribution",
    },
    "citations_medals_grid": {
        "fr": "Grille de médailles",
        "en": "Medal grid",
    },
    "citations_no_progress": {
        "fr": "Aucune citation n'a progressé dans ce match.",
        "en": "No citation progressed in this match.",
    },
    # ── Match View ────────────────────────────────────────────────────────────
    "mv_match_dynamics": {
        "fr": "Dynamique du match",
        "en": "Match dynamics",
    },
    "mv_dynamics_computing": {
        "fr": "Analyse de la dynamique…",
        "en": "Analyzing dynamics…",
    },
    "mv_dynamics_no_data": {
        "fr": "Données insuffisantes pour afficher la dynamique du match.",
        "en": "Not enough data to display match dynamics.",
    },
    "mv_dynamics_no_roster": {
        "fr": "Roster introuvable — frise de dominance indisponible.",
        "en": "Roster not found — dominance timeline unavailable.",
    },
    "mv_dynamics_no_team": {
        "fr": "Équipe introuvable pour ce joueur — frise de dominance indisponible.",
        "en": "Team not found for this player — dominance timeline unavailable.",
    },
    "mv_dynamics_no_kills": {
        "fr": "Aucun kill enregistré pour ce match.",
        "en": "No kills recorded for this match.",
    },
    "mv_dynamics_no_dominance": {
        "fr": "Données insuffisantes pour la frise de dominance.",
        "en": "Not enough data for the dominance timeline.",
    },
    "mv_antagonists_title": {
        "fr": "Antagonistes du match",
        "en": "Match antagonists",
    },
    "mv_highlight_loading": {
        "fr": "Chargement des highlight events (film)…",
        "en": "Loading highlight events (film)…",
    },
    "mv_nemesis_no_data": {
        "fr": "Impossible de déterminer Némésis/Souffre-douleur (timeline insuffisante).",
        "en": "Unable to determine Nemesis/Punching Bag (insufficient timeline).",
    },
    "mv_interactions_no_data": {
        "fr": "Données insuffisantes pour les interactions eliminateur-victime.",
        "en": "Not enough data for killer-victim interactions.",
    },
    "mv_kills_over_time": {
        "fr": "Frags au fil du match",
        "en": "Kills over the match",
    },
    "mv_scoreboard": {
        "fr": "Tableau des scores",
        "en": "Scoreboard",
    },
    "mv_scoreboard_no_data": {
        "fr": "Statistiques des joueurs indisponibles pour ce match.",
        "en": "Player stats unavailable for this match.",
    },
    "mv_players_title": {
        "fr": "Joueurs",
        "en": "Players",
    },
    "mv_impact_title": {
        "fr": "Impact du match",
        "en": "Match impact",
    },
    "mv_impact_no_events": {
        "fr": "Données d'impact indisponibles (highlight events manquants).",
        "en": "Impact data unavailable (missing highlight events).",
    },
    "mv_impact_computing": {
        "fr": "Analyse de la timeline…",
        "en": "Analyzing timeline…",
    },
    "mv_impact_no_events_match": {
        "fr": "Aucun événement enregistré pour ce match.",
        "en": "No events recorded for this match.",
    },
    "mv_impact_too_few": {
        "fr": "Pas assez de données pour afficher la timeline.",
        "en": "Not enough data to display the timeline.",
    },
    "mv_medals_no_data": {
        "fr": "Médailles indisponibles pour ce match (ou aucune médaille).",
        "en": "Medals unavailable for this match (or no medals).",
    },
    "mv_match_id_missing": {
        "fr": "MatchId manquant.",
        "en": "MatchId missing.",
    },
    # ── Teammates ─────────────────────────────────────────────────────────────
    "tm_map_stats_no_data": {
        "fr": "Données insuffisantes pour le ratio par carte.",
        "en": "Not enough data for the map ratio.",
    },
    "tm_trio_warning": {
        "fr": "Impossible d'aligner les stats des 3 joueurs sur ces matchs.",
        "en": "Unable to align stats for 3 players on these matches.",
    },
    # ── Media ─────────────────────────────────────────────────────────────────
    "media_disabled": {
        "fr": "Les médias sont désactivés dans Paramètres → Médias.",
        "en": "Media is disabled in Settings → Media.",
    },
    "media_no_folder": {
        "fr": "Configure au moins un dossier dans Paramètres → Médias (captures et/ou vidéos).",
        "en": "Configure at least one folder in Settings → Media (screenshots and/or videos).",
    },
    "media_no_files": {
        "fr": "Aucun média trouvé.",
        "en": "No media found.",
    },
    "media_no_filter_result": {
        "fr": "Aucun média à afficher avec ces filtres.",
        "en": "No media to display with these filters.",
    },
    "media_unknown_match": {
        "fr": "Match inconnu",
        "en": "Unknown match",
    },
    "media_no_thumbnail": {
        "fr": "(pas de miniature générée)",
        "en": "(no thumbnail generated)",
    },
    "media_unassociated_match": {
        "fr": "Match: non associé",
        "en": "Match: unassociated",
    },
    # ── Settings ─────────────────────────────────────────────────────────────
    "settings_save_ok": {
        "fr": "Paramètres sauvegardés.",
        "en": "Settings saved.",
    },
    "settings_save_error": {
        "fr": "Erreur lors de la sauvegarde : {error}",
        "en": "Error while saving: {error}",
    },
    # ── Last Match ────────────────────────────────────────────────────────────
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
    "settings_index_reset": {
        "fr": "Index médias réinitialisé (joueur courant).",
        "en": "Media index reset (current player).",
    },
    "obj_no_player_data": {
        "fr": "⚠️ Aucune donnée pour le joueur (XUID: {xuid}).",
        "en": "⚠️ No data for this player (XUID: {xuid}).",
    },
    "obj_profile_label": {
        "fr": "Profil détecté",
        "en": "Detected profile",
    },
    "obj_analysis_detailed": {
        "fr": "📈 Analyse détaillée",
        "en": "📈 Detailed analysis",
    },
    "obj_correlation_title": {
        "fr": "Corrélation Objectifs / Kills",
        "en": "Objective / Kill correlation",
    },
    "obj_breakdown_title": {
        "fr": "Répartition par Catégorie",
        "en": "Category breakdown",
    },
    "obj_trend_title": {
        "fr": "Évolution dans le temps",
        "en": "Timeline",
    },
    "obj_assist_detail": {
        "fr": "Détail par type",
        "en": "Detail by type",
    },
    "obj_awards_frequent": {
        "fr": "🏅 Awards les plus fréquents",
        "en": "🏅 Most frequent awards",
    },
    "obj_tips": {
        "fr": "💡 Conseils personnalisés",
        "en": "💡 Personalized tips",
    },
    "obj_no_player_selected": {
        "fr": "⚠️ Veuillez d'abord sélectionner un profil joueur.",
        "en": "⚠️ Please select a player profile first.",
    },
    # ── Teammates ─────────────────────────────────────────────────────────────
    "tm_kd_half_caption": {
        "fr": "F/M 1ère moitié → 2nde moitié des matchs affichés.",
        "en": "K/D first half → second half of displayed matches.",
    },
    "tm_select_teammate": {
        "fr": "Sélectionne au moins un coéquipier.",
        "en": "Select at least one teammate.",
    },
    # ── Match View ────────────────────────────────────────────────────────────
    "mv_vs_expected": {
        "fr": "Réel vs attendu",
        "en": "Actual vs expected",
    },
    "mv_media_title": {
        "fr": "Médias",
        "en": "Media",
    },
    "mv_videos_title": {
        "fr": "Vidéos",
        "en": "Videos",
    },
    # ── Coéquipiers — titres graphiques ──────────────────────────────────────
    "tm_killing_spree": {"fr": "Folie meurtrière (max)", "en": "Killing Spree (max)"},
    "tm_headshots": {"fr": "Tirs à la tête", "en": "Headshots"},
    "tm_perfect_kills": {"fr": "Frags parfaits", "en": "Perfect Kills"},
    "tm_kills": {"fr": "Frags", "en": "Kills"},
    "tm_deaths": {"fr": "Morts", "en": "Deaths"},
    "tm_assists": {"fr": "Assists", "en": "Assists"},
    # ── Match View Participation ──────────────────────────────────────────────
    "mvp_participation_title": {"fr": "Participation au match", "en": "Match Participation"},
    "mvp_comparison_title": {
        "fr": "Comparaison de participation",
        "en": "Participation Comparison",
    },
    "mvp_axes_label": {"fr": "Axes", "en": "Axes"},
    # ── Session Compare ────────────────────────────────────────────────────────
    "sc_no_matches_in_session": {
        "fr": "Aucune partie dans {session_name}.",
        "en": "No matches in {session_name}.",
    },
    "sc_participation_comparison": {
        "fr": "Comparaison de la contribution au score entre les deux sessions.",
        "en": "Comparison of score contribution between the two sessions.",
    },
    # ── Analyse d'objectifs — labels métriques ────────────────────────────────
    "obj_overview_title": {"fr": "Vue d'ensemble", "en": "Overview"},
    "obj_score_label": {"fr": "Score Objectifs", "en": "Objective Score"},
    "obj_frag_score_label": {"fr": "Score Frags", "en": "Kill Score"},
    "obj_assist_score_label": {"fr": "Score Assistances", "en": "Assist Score"},
    "obj_ratio_label": {"fr": "Ratio Objectifs", "en": "Objective Ratio"},
    # ── Objective Analysis — onglets ─────────────────────────────────────────
    "obj_tab_scatter": {"fr": "Objectifs vs Frags", "en": "Objectives vs Kills"},
    "obj_tab_breakdown": {"fr": "Répartition du Score", "en": "Score Breakdown"},
    "obj_tab_trend": {"fr": "Évolution", "en": "Trend"},
    # ── Coéquipiers — métriques supplémentaires ──────────────────────────────
    "tm_kda": {"fr": "FDA", "en": "KDA"},
    "tm_accuracy": {"fr": "Précision", "en": "Accuracy"},
    "tm_avg_life": {"fr": "Durée de vie moyenne", "en": "Average Lifespan"},
    "tm_performance": {"fr": "Performance", "en": "Performance"},
    "tm_score": {"fr": "Score", "en": "Score"},
    "tm_seconds": {"fr": "Secondes", "en": "Seconds"},
    # ── Noms des pages (navigation / onglets) ────────────────────────────────
    "page_timeseries": {"fr": "Séries temporelles", "en": "Time Series"},
    "page_session_compare": {"fr": "Comparaison de sessions", "en": "Session Comparison"},
    "page_last_match": {"fr": "Dernier match", "en": "Last Match"},
    "page_match": {"fr": "Match", "en": "Match"},
    "page_media": {"fr": "Médias", "en": "Media"},
    "page_citations": {"fr": "Citations", "en": "Commendations"},
    "page_win_loss": {"fr": "Victoires/Défaites", "en": "Wins/Losses"},
    "page_teammates": {"fr": "Mes coéquipiers", "en": "My Teammates"},
    "page_match_history": {"fr": "Historique des parties", "en": "Match History"},
    "page_career": {"fr": "Carrière", "en": "Career"},
    "page_settings": {"fr": "Paramètres", "en": "Settings"},
    # ── KPIs ─────────────────────────────────────────────────────────────────
    "kpi_matches_header": {"fr": "Parties", "en": "Matches"},
    "kpi_career_header": {"fr": "Carrière", "en": "Career"},
    "kpi_avg_duration": {"fr": "Durée moyenne / match", "en": "Avg duration / match"},
    "kpi_total_duration": {"fr": "Durée totale", "en": "Total duration"},
    "kpi_kills_per_match": {"fr": "Frags par partie", "en": "Kills per match"},
    "kpi_deaths_per_match": {"fr": "Morts par partie", "en": "Deaths per match"},
    "kpi_assists_per_match": {"fr": "Assistances par partie", "en": "Assists per match"},
    "kpi_kills_per_min": {"fr": "Frags / min", "en": "Kills / min"},
    "kpi_deaths_per_min": {"fr": "Morts / min", "en": "Deaths / min"},
    "kpi_assists_per_min": {"fr": "Assistances / min", "en": "Assists / min"},
    "kpi_avg_accuracy": {"fr": "Précision moyenne", "en": "Average accuracy"},
    "kpi_avg_lifespan": {"fr": "Durée de vie moyenne", "en": "Average lifespan"},
    "kpi_win_rate": {"fr": "Taux de victoire", "en": "Win rate"},
    "kpi_loss_rate": {"fr": "Taux de défaite", "en": "Loss rate"},
    "kpi_ratio": {"fr": "Ratio", "en": "Ratio"},
    # ── Score de performance ─────────────────────────────────────────────────
    "perf_title": {"fr": "Score de performance", "en": "Performance Score"},
    "perf_short_desc": {"fr": "Relatif à ton historique", "en": "Relative to your history"},
    "perf_insufficient": {"fr": "Historique insuffisant", "en": "Insufficient history"},
    "perf_matches_count": {"fr": "parties", "en": "matches"},
    "perf_label_excellent": {"fr": "Excellent", "en": "Excellent"},
    "perf_label_good": {"fr": "Bon", "en": "Good"},
    "perf_label_average": {"fr": "Moyen", "en": "Average"},
    "perf_label_below": {"fr": "Faible", "en": "Below average"},
    "perf_label_bad": {"fr": "Difficile", "en": "Difficult"},
    "perf_score_exceptional": {"fr": "Exceptionnel", "en": "Exceptional"},
    "perf_score_good": {"fr": "Bon", "en": "Good"},
    "perf_score_normal": {"fr": "Normal", "en": "Normal"},
    "perf_score_below": {"fr": "Sous la moyenne", "en": "Below average"},
    "perf_score_difficult": {"fr": "Difficile", "en": "Difficult"},
    "perf_interp_excellent": {
        "fr": "Match exceptionnel pour toi",
        "en": "Exceptional match for you",
    },
    "perf_interp_good": {"fr": "Au-dessus de ta moyenne", "en": "Above your average"},
    "perf_interp_average": {"fr": "Performance typique", "en": "Typical performance"},
    "perf_interp_below": {"fr": "En-dessous de ta moyenne", "en": "Below your average"},
    "perf_interp_bad": {"fr": "Match difficile", "en": "Tough match"},
    # ── Match history ───────────────────────────────────────────────────────
    "mh_title": {"fr": "Historique des parties", "en": "Match history"},
    # ── Match view ──────────────────────────────────────────────────────────
    "mv_no_rating": {
        "fr": "Aucun rating LUSR/CSR calculé pour ce match. Lance un backfill rating pour la playlist concernée.",
        "en": "No LUSR/CSR rating calculated for this match. Run a rating backfill for the relevant playlist.",
    },
    "mv_performance": {"fr": "Performance", "en": "Performance"},
    "mv_results": {"fr": "Résultats", "en": "Results"},
    "mv_relative_history": {"fr": "Relatif à ton historique", "en": "Relative to your history"},
    "mv_insufficient_history": {"fr": "Historique insuffisant", "en": "Insufficient history"},
    "mv_thumbnail_unavailable": {
        "fr": "Miniature de carte indisponible.",
        "en": "Map thumbnail unavailable.",
    },
    "mv_stats_unavailable": {
        "fr": "Stats détaillées indisponibles pour ce match. Essaie un backfill.",
        "en": "Detailed stats unavailable for this match. Try a backfill.",
    },
    "mv_medal_fallback": {"fr": "Médaille #{n}", "en": "Medal #{n}"},
    "mv_open_waypoint": {"fr": "Ouvrir sur HaloWaypoint", "en": "Open on HaloWaypoint"},
    "mv_killer_victim_title": {"fr": "Éliminateur-Victime", "en": "Killer-Victim"},
    "mv_events_unavailable": {
        "fr": "Indisponible : la DB ne contient pas les highlight events pour ce match.",
        "en": "Unavailable: the DB does not contain highlight events for this match.",
    },
    "mv_scoreboard_avg_life": {"fr": "Durée de vie moy.", "en": "Avg life span"},
    "mv_dominance_legend": {
        "fr": "- **Barres ** : % des frags par tranche de 30 s — bleu = mon équipe, orange = adversaires\n- **Chiffres encadrés** : score cumulé de chaque équipe à l'instant T (encadré si en tête)\n- **Points ** : chaque point = un kill individuel\n- **Lignes reliées** : série d'un même joueur (≥ 3 kills consécutifs sans mourir)",
        "en": "- **Bars**: % of kills per 30s interval — blue = my team, orange = opponents\n- **Framed numbers**: cumulative team score at that point (framed if leading)\n- **Dots**: each dot = an individual kill\n- **Connected lines**: streak by a single player (≥ 3 consecutive kills without dying)",
    },
    "mv_nemesis_unavailable": {
        "fr": "Indisponible : la DB ne contient pas les highlight events. Si tu utilises une DB SPNKr, relance l'import avec `--with-highlight-events`.",
        "en": "Unavailable: the DB does not contain highlight events. If you are using a SPNKr DB, re-run the import with `--with-highlight-events`.",
    },
    "mv_deaths_count": {"fr": "{prefix}{n} morts", "en": "{prefix}{n} deaths"},
    "mv_killed_count": {"fr": "{prefix}Tué {n} fois", "en": "{prefix}Killed {n} times"},
    "mv_team_n": {"fr": "Équipe {n}", "en": "Team {n}"},
    "mv_team_unknown": {"fr": "Équipe inconnue", "en": "Unknown team"},
    "mv_team_label": {"fr": "Équipe {name}", "en": "Team {name}"},
    "mv_scoreboard_rank_note": {
        "fr": "ℹ️ Le rang est individuel au sein de chaque équipe — plusieurs joueurs peuvent partager le même rang (ex. : tous les membres d'une équipe vaincue).",
        "en": "ℹ️ Rank is individual within each team — multiple players can share the same rank (e.g., all members of a defeated team).",
    },
    "mv_roster_unavailable": {
        "fr": "Roster indisponible pour ce match (payload MatchStats manquant ou équipe introuvable).",
        "en": "Roster unavailable for this match (MatchStats payload missing or team not found).",
    },
    "mv_roster_my_team": {"fr": "Mon équipe — {name} ({n})", "en": "My team — {name} ({n})"},
    "mv_roster_enemy_team": {
        "fr": "Équipe adverse — {name} ({n})",
        "en": "Enemy team — {name} ({n})",
    },
    "mv_roster_opponents": {"fr": "Adversaires", "en": "Opponents"},
    # ── Match view charts ───────────────────────────────────────────────────
    "mvc_mmr_team": {"fr": "MMR d'équipe", "en": "Team MMR"},
    "mvc_mmr_enemy": {"fr": "MMR adverse", "en": "Enemy MMR"},
    "mvc_mmr_gap": {"fr": "Écart MMR", "en": "MMR gap"},
    "mvc_actual_only": {
        "fr": "Valeur réelle (comparaison indisponible)",
        "en": "Actual value (comparison unavailable)",
    },
    "mvc_hist_avg": {
        "fr": "Moyenne hist. {category} ({n} matchs)",
        "en": "Hist. avg {category} ({n} matches)",
    },
    "mvc_ratio_avg": {"fr": "Ratio moy. {category}", "en": "Avg ratio {category}"},
    "mvc_fda_title": {"fr": "F / D / A", "en": "K / D / A"},
    "mvc_ratio_title": {"fr": "Ratio", "en": "Ratio"},
    "mvc_this_match": {"fr": "Ce match", "en": "This match"},
    "mvc_mmr_gap_sub": {"fr": "équipe - adverse", "en": "team - enemy"},
    "mvc_fda_ratio": {"fr": "Ratio F/D/A", "en": "K/D/A Ratio"},
    "mvc_lbl_k": {"fr": "F", "en": "K"},
    "mvc_lbl_d": {"fr": "D", "en": "D"},
    "mvc_lbl_a": {"fr": "A", "en": "A"},
    # ── Career ──────────────────────────────────────────────────────────────
    "career_xp_total": {"fr": "XP total", "en": "Total XP"},
    "career_xp_progress": {"fr": "Progression XP", "en": "XP progression"},
    "career_rank_n": {"fr": "Rang {n}", "en": "Rank {n}"},
    "career_rank_hover": {
        "fr": "Rang {rank}: {label}<br>XP total: {xp}",
        "en": "Rank {rank}: {label}<br>Total XP: {xp}",
    },
    "career_max_rank": {"fr": "Rang maximum atteint", "en": "Maximum rank reached"},
    "career_hero_rank": {"fr": "Rang Héros atteint !", "en": "Hero rank reached!"},
    "career_ranked": {"fr": "Classé", "en": "Ranked"},
    "career_tactical": {"fr": "Tactique", "en": "Tactical"},
    # ── Citations ───────────────────────────────────────────────────────────
    "cit_obtained": {"fr": "Citations obtenues", "en": "Citations earned"},
    "cit_matches_analyzed": {"fr": "Matchs analysés", "en": "Matches analyzed"},
    "cit_distinct_medals": {"fr": "Médailles distinctes", "en": "Distinct medals"},
    "cit_total_medals": {"fr": "Total médailles", "en": "Total medals"},
    "cit_filter_category": {"fr": "Catégorie", "en": "Category"},
    "cit_filter_all": {"fr": "(toutes)", "en": "(all)"},
    "cit_search": {"fr": "Recherche", "en": "Search"},
    "cit_search_placeholder": {
        "fr": "ex: assassin, pilote, multifrag…",
        "en": "e.g. assassin, pilot, multikill…",
    },
    "cit_mastery_master": {"fr": "Maître", "en": "Master"},
    "cit_mastery_level": {"fr": "Niveau {level}", "en": "Level {level}"},
    # ── Last match ──────────────────────────────────────────────────────────
    "lm_quick_select": {
        "fr": "Sélection rapide (filtres actuels)",
        "en": "Quick select (current filters)",
    },
    "lm_search_datetime": {"fr": "Recherche par date/heure", "en": "Search by date/time"},
    "lm_date": {"fr": "Date", "en": "Date"},
    "lm_time": {"fr": "Heure", "en": "Time"},
    "lm_tolerance": {"fr": "Tolérance (minutes)", "en": "Tolerance (minutes)"},
    "lm_no_match_tol": {
        "fr": "Aucun match trouvé dans ±{tol} min autour de {dt}.",
        "en": "No match found within ±{tol} min of {dt}.",
    },
    # ── Session compare ─────────────────────────────────────────────────────
    "sc_session_a_ref": {"fr": "Session A (référence)", "en": "Session A (reference)"},
    "sc_session_b_cmp": {"fr": "Session B (à comparer)", "en": "Session B (to compare)"},
    "sc_session_a": {"fr": "Session A", "en": "Session A"},
    "sc_session_b": {"fr": "Session B", "en": "Session B"},
    "sc_match_count": {"fr": "Nombre de parties", "en": "Number of matches"},
    "sc_kda_label": {"fr": "FDA (Frags-Décès-Assists)", "en": "KDA (Kills-Deaths-Assists)"},
    "sc_win_rate": {"fr": "Taux de victoire", "en": "Win rate"},
    "sc_avg_life": {"fr": "Durée de vie moyenne", "en": "Average life span"},
    "sc_total_kills": {"fr": "Total des frags", "en": "Total kills"},
    "sc_total_deaths": {"fr": "Total des morts", "en": "Total deaths"},
    "sc_total_assists": {"fr": "Total des assistances", "en": "Total assists"},
    "sc_mmr_team_avg": {"fr": "MMR équipe (moy)", "en": "Team MMR (avg)"},
    "sc_mmr_enemy_avg": {"fr": "MMR adverse (moy)", "en": "Enemy MMR (avg)"},
    "sc_mmr_gap_avg": {"fr": "Écart MMR (moy)", "en": "MMR gap (avg)"},
    "sc_net_score_desc": {
        "fr": "Évolution du net score (Frags − Morts) match après match",
        "en": "Net score (Kills − Deaths) evolution match after match",
    },
    "sc_hist_avg_trace": {
        "fr": "Moy. historique ({n} sessions){suffix}",
        "en": "Hist. avg ({n} sessions){suffix}",
    },
    "sc_per_match_ratio": {"fr": "Par partie / Ratio", "en": "Per match / Ratio"},
    "sc_radar_kd": {"fr": "F/M", "en": "K/D"},
    "sc_radar_win": {"fr": "Victoire %", "en": "Win %"},
    "sc_kills_per_match": {"fr": "Frags / partie", "en": "Kills / match"},
    "sc_deaths_per_match": {"fr": "Morts / partie", "en": "Deaths / match"},
    "sc_kd_ratio": {"fr": "Ratio F/D", "en": "K/D ratio"},
    "sc_with_friends": {"fr": "avec {friends} 👥", "en": "with {friends} 👥"},
    "sc_with_n_friends": {"fr": "avec {n} ami(s) 👥", "en": "with {n} friend(s) 👥"},
    "sc_friends_unavailable": {"fr": "info amis indisponible", "en": "friends info unavailable"},
    "sc_solo": {"fr": "solo 🎮", "en": "solo 🎮"},
    "sc_compare_all": {
        "fr": "vs moy. {n} sessions {cat}",
        "en": "vs avg. {n} sessions {cat}",
    },
    "sc_compare_same_friends": {
        "fr": "vs moy. {n} sessions {cat} (mêmes amis)",
        "en": "vs avg. {n} sessions {cat} (same friends)",
    },
    "sc_compare_with_friends": {
        "fr": "vs moy. {n} sessions {cat} (avec amis)",
        "en": "vs avg. {n} sessions {cat} (with friends)",
    },
    "sc_compare_solo": {
        "fr": "vs moy. {n} sessions solo {cat}",
        "en": "vs avg. {n} solo sessions {cat}",
    },
    "sc_friends_suffix": {"fr": "ami(s) 👥", "en": "friend(s) 👥"},
    # ── Session compare charts ──────────────────────────────────────────────
    "scc_mmr_team": {"fr": "MMR Équipe", "en": "Team MMR"},
    "scc_mmr_enemy": {"fr": "MMR Adverse", "en": "Enemy MMR"},
    # ── Playlist categories ─────────────────────────────────────────────────
    "cat_btb": {"fr": "Grande bataille en équipe", "en": "Big Team Battle"},
    "cat_ranked": {"fr": "Classé", "en": "Ranked"},
    "cat_firefight": {"fr": "Baptême du feu", "en": "Firefight"},
    "cat_other": {"fr": "Autre", "en": "Other"},
    # ── Win/Loss ────────────────────────────────────────────────────────────
    "wl_me_filtered": {"fr": "Moi (filtres actuels)", "en": "Me (current filters)"},
    "wl_me_all": {"fr": "Moi (toutes les parties)", "en": "Me (all matches)"},
    "wl_col_map": {"fr": "Carte", "en": "Map"},
    "wl_col_matches": {"fr": "Parties", "en": "Matches"},
    "wl_col_accuracy_avg": {"fr": "Précision moy. (%)", "en": "Avg accuracy (%)"},
    "wl_col_performance_avg": {"fr": "Performance moy.", "en": "Avg performance"},
    "wl_col_win_rate": {"fr": "Taux victoire (%)", "en": "Win rate (%)"},
    "wl_col_loss_rate": {"fr": "Taux défaite (%)", "en": "Loss rate (%)"},
    "wl_col_ratio": {"fr": "Ratio global", "en": "Overall ratio"},
    "wl_col_playlist": {"fr": "Playlist", "en": "Playlist"},
    "wl_col_mode": {"fr": "Mode", "en": "Mode"},
    "wl_several": {"fr": "Plusieurs", "en": "Multiple"},
    "wl_period_col_wins": {"fr": "Victoires", "en": "Wins"},
    "wl_period_col_losses": {"fr": "Défaites", "en": "Losses"},
    "wl_period_col_draws": {"fr": "Égalités", "en": "Draws"},
    "wl_period_col_unfinished": {"fr": "Non terminés", "en": "Unfinished"},
    "wl_period_col_total": {"fr": "Total", "en": "Total"},
    "wl_period_col_win_rate": {"fr": "Taux de victoires", "en": "Win rate"},
    # ── Teammates ───────────────────────────────────────────────────────────
    "tm_no_matches_filters": {
        "fr": "Aucun match à afficher avec les filtres actuels.",
        "en": "No matches to display with current filters.",
    },
    "tm_min_matches_map": {"fr": "Minimum de matchs par carte", "en": "Min matches per map"},
    "tm_ratio_map_header": {
        "fr": "Ratio global par carte — avec mes coéquipiers (min {n} matchs)",
        "en": "Overall ratio by map — with my teammates (min {n} matches)",
    },
    "tm_no_trio_matches": {
        "fr": "Aucun match trouvé où vous êtes tous les trois dans l'équipe.",
        "en": "No matches found where all three of you are on the same team.",
    },
    "tm_match_details_title": {
        "fr": "Détails des matchs (joueur vs joueur)",
        "en": "Match details (player vs player)",
    },
    "tm_win_loss": {"fr": "Win/Loss", "en": "Win/Loss"},
    "tm_metric_matches": {"fr": "Matchs", "en": "Matches"},
    "tm_metric_global_ratio": {"fr": "Ratio global", "en": "Overall ratio"},
    "tm_metric_frags_min": {"fr": "Frags / min", "en": "Kills / min"},
    "tm_metric_deaths_min": {"fr": "Morts / min", "en": "Deaths / min"},
    "tm_metric_assists_min": {"fr": "Assistances / min", "en": "Assists / min"},
    "tm_loading_slow": {
        "fr": "Le chargement sera plus lent car les données sont lues depuis la base partagée.",
        "en": "Loading will be slower as data is read from the shared database.",
    },
    "tm_apply_filters": {
        "fr": "Appliquer les filtres actuels (période/sessions + map/playlist)",
        "en": "Apply current filters (period/sessions + map/playlist)",
    },
    "tm_same_team": {"fr": "Même équipe", "en": "Same team"},
    "tm_show_smoothed": {"fr": "Afficher les courbes lissées", "en": "Show smoothed curves"},
    "tm_smoothed_help": {
        "fr": "Active/désactive les courbes de moyenne mobile sur l'ensemble de la période.",
        "en": "Toggle rolling average curves over the entire period.",
    },
    "tm_select_teammates": {"fr": "Coéquipiers", "en": "Teammates"},
    "tm_session_trend": {"fr": "Tendance de session", "en": "Session trend"},
    # ── Teammates impact ────────────────────────────────────────────────────
    "tmi_first_blood": {"fr": "🟢 Premier Sang", "en": "🟢 First Blood"},
    "tmi_finisher": {"fr": "🟡 Finisseur", "en": "🟡 Finisher"},
    "tmi_liability": {"fr": "🔴 Boulet", "en": "🔴 Liability"},
    "tmi_matches_analyzed": {"fr": "📊 Matchs analysés", "en": "📊 Matches analyzed"},
    "tmi_badge": {"fr": "Badge", "en": "Badge"},
    "tmi_col_rank": {"fr": "Rang", "en": "Rank"},
    "tmi_col_player": {"fr": "Joueur", "en": "Player"},
    "tmi_col_score": {"fr": "Score", "en": "Score"},
    "tmi_col_first_blood": {"fr": "⚡ Premier Sang", "en": "⚡ First Blood"},
    "tmi_col_finisher": {"fr": "🎯 Finisseur", "en": "🎯 Finisher"},
    "tmi_col_casualty": {"fr": "💀 Boulet", "en": "💀 Liability"},
    "tmi_mvp_label": {"fr": "🏆 Brute de la Soirée : {mvp}", "en": "🏆 MVP of the Night: {mvp}"},
    "tmi_boulet_label": {"fr": "🍌 Maillon Faible : {boulet}", "en": "🍌 Weak Link: {boulet}"},
    "tmi_no_shared_db": {
        "fr": "Impossible d'accéder à shared_matches.duckdb pour les événements.",
        "en": "Cannot access shared_matches.duckdb for events.",
    },
    "tmi_no_events": {
        "fr": "Les données d'événements ne sont pas disponibles. Lance un backfill events.",
        "en": "Event data is not available. Run an events backfill.",
    },
    # ── Teammates synergy ───────────────────────────────────────────────────
    "tms_participation_title": {"fr": "Profil de participation", "en": "Participation profile"},
    "tms_axes": {"fr": "**Axes**", "en": "**Axes**"},
    "tms_trio_title": {"fr": "Complémentarité trio", "en": "Trio complementarity"},
    # ── Teammates helpers ───────────────────────────────────────────────────
    "tmh_waypoint": {"fr": "HaloWaypoint", "en": "HaloWaypoint"},
    "tmh_col_match": {"fr": "Match", "en": "Match"},
    "tmh_col_date": {"fr": "Date", "en": "Date"},
    "tmh_col_map": {"fr": "Carte", "en": "Map"},
    "tmh_col_playlist": {"fr": "Playlist", "en": "Playlist"},
    "tmh_col_mode": {"fr": "Mode", "en": "Mode"},
    "tmh_col_result": {"fr": "Résultat", "en": "Result"},
    "tmh_col_score": {"fr": "Score", "en": "Score"},
    "tmh_col_team_mmr": {"fr": "MMR équipe", "en": "Team MMR"},
    "tmh_col_enemy_mmr": {"fr": "MMR adverse", "en": "Enemy MMR"},
    "tmh_col_delta_mmr": {"fr": "Écart MMR", "en": "MMR gap"},
    "tmh_link_open": {"fr": "Ouvrir", "en": "Open"},
    "tmh_outcome_win": {"fr": "Victoire", "en": "Win"},
    "tmh_outcome_loss": {"fr": "Défaite", "en": "Loss"},
    "tmh_outcome_draw": {"fr": "Égalité", "en": "Draw"},
    "tmh_outcome_unfinished": {"fr": "Non terminé", "en": "Did Not Finish"},
    # ── Timeseries ──────────────────────────────────────────────────────────
    "ts_lifespan_vs_kills": {"fr": "Durée de vie vs frags", "en": "Life span vs kills"},
    "ts_lifespan_s": {"fr": "Durée de vie (s)", "en": "Life span (s)"},
    "ts_accuracy_vs_kda": {"fr": "Précision vs FDA", "en": "Accuracy vs KDA"},
    "ts_lifespan_vs_deaths": {"fr": "Durée de vie vs morts", "en": "Life span vs deaths"},
    "ts_deaths_label": {"fr": "Morts", "en": "Deaths"},
    "ts_kills_vs_deaths": {"fr": "Frags vs morts", "en": "Kills vs deaths"},
    "ts_mmr_team_vs_enemy": {"fr": "MMR Équipe vs MMR Adversaire", "en": "Team MMR vs Enemy MMR"},
    "ts_mmr_team": {"fr": "MMR Équipe", "en": "Team MMR"},
    "ts_mmr_enemy": {"fr": "MMR Adversaire", "en": "Enemy MMR"},
    "ts_events_unavailable": {
        "fr": "Données d'événements non disponibles. Lance un backfill events.",
        "en": "Event data not available. Run an events backfill.",
    },
    "ts_not_enough_dist": {
        "fr": "Pas assez de données ({count} matchs). Il en faut au moins {min} pour la distribution.",
        "en": "Not enough data ({count} matches). At least {min} are needed for the distribution.",
    },
    "ts_not_enough_corr": {
        "fr": "Pas assez de données ({count} matchs). Il en faut au moins {min} pour la corrélation.",
        "en": "Not enough data ({count} matches). At least {min} are needed for the correlation.",
    },
    # ── Media tab ───────────────────────────────────────────────────────────
    "media_view_full": {"fr": "Voir en grand", "en": "View full size"},
    "media_file_missing": {"fr": "Fichier absent : {file_name}", "en": "File missing: {file_name}"},
    "media_dialog_title": {"fr": "Média", "en": "Media"},
    "media_no_indexed": {
        "fr": "Aucun média indexé. Configure les dossiers dans Paramètres → Médias et lance un scan.",
        "en": "No media indexed. Configure folders in Settings → Media and run a scan.",
    },
    "media_filters": {"fr": "Filtres", "en": "Filters"},
    "media_type": {"fr": "Type", "en": "Type"},
    "media_filename": {"fr": "Nom de fichier", "en": "File name"},
    "media_columns": {"fr": "Colonnes", "en": "Columns"},
    "media_my_captures": {"fr": "Mes captures", "en": "My captures"},
    "media_captures_of": {"fr": "Captures de {gamertag}", "en": "Captures by {gamertag}"},
    "media_unmatched": {"fr": "Sans correspondance", "en": "Unmatched"},
    # ── Media library ───────────────────────────────────────────────────────
    "ml_open_match": {"fr": "Ouvrir le match", "en": "Open match"},
    "ml_click_thumbnail": {
        "fr": "Cliquer pour afficher la miniature",
        "en": "Click to show thumbnail",
    },
    "ml_hide_thumbnail": {"fr": "Masquer miniature", "en": "Hide thumbnail"},
    "ml_show_thumbnail": {"fr": "Afficher miniature", "en": "Show thumbnail"},
    "ml_preview": {"fr": "Aperçu", "en": "Preview"},
    "ml_options": {"fr": "Options", "en": "Options"},
    "ml_group_by_match": {"fr": "Grouper par match", "en": "Group by match"},
    "ml_show_unassociated": {"fr": "Afficher non associés", "en": "Show unassociated"},
    "ml_max_media": {"fr": "Max médias", "en": "Max media"},
    "ml_types": {"fr": "Types", "en": "Types"},
    "ml_filter_filename": {"fr": "Filtre nom de fichier", "en": "Filter filename"},
    "ml_rescan": {"fr": "Re-scanner les dossiers", "en": "Re-scan folders"},
    "ml_indexing_done": {
        "fr": "Indexation terminée : {n_new} nouveaux, {n_updated} mis à jour, {n_associated} association(s).",
        "en": "Indexing done: {n_new} new, {n_updated} updated, {n_associated} association(s).",
    },
    "ml_indexing_thumbnails": {
        "fr": " — {n_gen} miniature(s), {n_err} erreur(s)",
        "en": " — {n_gen} thumbnail(s), {n_err} error(s)",
    },
    "ml_generate_thumbnails": {"fr": "Générer les thumbnails", "en": "Generate thumbnails"},
    "ml_thumbnails_generated": {
        "fr": "{n_gen} thumbnail(s) généré(s)",
        "en": "{n_gen} thumbnail(s) generated",
    },
    "ml_unassigned_from_db": {
        "fr": "ℹ️ {count} média(s) non associé(s) depuis la BDD. Cliquez sur 'Re-scanner les dossiers' pour forcer l'association.",
        "en": "ℹ️ {count} unassociated media from DB. Click 'Re-scan folders' to force association.",
    },
    "ml_disk_fallback": {
        "fr": "ℹ️ Les médias sont chargés depuis le scan disque (pas encore indexés en BDD). Cliquez sur 'Re-scanner les dossiers' pour indexer.",
        "en": "ℹ️ Media loaded from disk scan (not yet indexed in DB). Click 'Re-scan folders' to index.",
    },
    "ml_no_match_windows": {
        "fr": "⚠️ Aucune fenêtre temporelle de match disponible pour l'association automatique.",
        "en": "⚠️ No match time windows available for automatic association.",
    },
    "ml_no_associations": {
        "fr": "⚠️ Aucun média associé à un match depuis la BDD. Tolérance actuelle : {tol} min.",
        "en": "⚠️ No media could be associated with a match from DB. Current tolerance: {tol} min.",
    },
    "ml_generate_help": {
        "fr": "Génère les miniatures pour les vidéos. Nécessite ffmpeg.",
        "en": "Generate thumbnails for videos. Requires ffmpeg.",
    },
    "ml_rescan_prompt": {
        "fr": "Cliquez sur 'Re-scanner les dossiers' pour forcer l'indexation.",
        "en": "Click 'Re-scan folders' to force indexing.",
    },
    "ml_rescan_index": {
        "fr": "Cliquez sur 'Re-scanner les dossiers' pour indexer en BDD.",
        "en": "Click 'Re-scan folders' to index into DB.",
    },
    "ml_tolerance": {
        "fr": "Tolérance actuelle : {tol} min",
        "en": "Current tolerance: {tol} min",
    },
    "ml_window": {"fr": "Fenêtre : {t0} → {t1}", "en": "Window: {t0} → {t1}"},
    "ml_captures": {"fr": "Captures", "en": "Captures"},
    "ml_video": {"fr": "Vidéo", "en": "Video"},
    # ── Settings ────────────────────────────────────────────────────────────
    "set_db_mgmt": {"fr": "Gestion de la base de données", "en": "Database management"},
    "set_db_empty": {
        "fr": "La base sélectionnée est vide (0 octet). Lancez une synchronisation.",
        "en": "The selected database is empty (0 bytes). Run a sync.",
    },
    "set_sync_title": {"fr": "Synchronisation", "en": "Sync"},
    "set_sync_source": {"fr": "Source de données", "en": "Data source"},
    "set_sync_max_matches": {"fr": "Max matchs à récupérer", "en": "Max matches to fetch"},
    "set_sync_rate": {"fr": "Délai entre appels API (sec)", "en": "API rate limit (sec)"},
    "set_refresh_options": {"fr": "Options du bouton Actualiser", "en": "Refresh button options"},
    "set_backfill_medals": {"fr": "Médailles", "en": "Medals"},
    "set_backfill_events": {"fr": "Événements", "en": "Events"},
    "set_backfill_scores": {"fr": "Scores de performance", "en": "Performance scores"},
    "set_backfill_aliases": {"fr": "Alias joueurs", "en": "Player aliases"},
    "set_backfill_personal_scores": {"fr": "Personal Scores", "en": "Personal Scores"},
    "set_backfill_sessions": {"fr": "Sessions", "en": "Sessions"},
    "set_backfill_citations": {"fr": "Citations", "en": "Citations"},
    "set_backfill_antagonists": {"fr": "Antagonistes", "en": "Antagonists"},
    "set_backfill_skill": {"fr": "Skill/Rating", "en": "Skill/Rating"},
    "set_backfill_killer_victim": {"fr": "Killer/Victim", "en": "Killer/Victim"},
    "set_backfill_shots": {"fr": "Shots (tirs)", "en": "Shots"},
    "set_backfill_participants_shots": {"fr": "Shots participants", "en": "Participants shots"},
    "set_backfill_pve": {"fr": "Stats PvE", "en": "PvE stats"},
    "set_media_title": {"fr": "Médias", "en": "Media"},
    "set_media_enable": {"fr": "Activer les médias", "en": "Enable media"},
    "set_media_screenshots": {"fr": "Dossier captures", "en": "Screenshots folder"},
    "set_media_videos": {"fr": "Dossier vidéos", "en": "Videos folder"},
    "set_media_tolerance": {
        "fr": "Tolérance association (minutes)",
        "en": "Association tolerance (minutes)",
    },
    "set_experience_title": {"fr": "Expérience", "en": "Experience"},
    "set_lang_title": {"fr": "Langue", "en": "Language"},
    "set_clear_cache_title": {
        "fr": "Vider le cache à chaque actualisation",
        "en": "Clear cache on every refresh",
    },
    # ── Match view helpers ──────────────────────────────────────────────────
    "mvh_my_captures": {"fr": "Mes captures", "en": "My captures"},
    "mvh_captures_of": {"fr": "Captures de {gt}", "en": "Captures by {gt}"},
    # ── Match view participation ────────────────────────────────────────────
    "mvp_this_match": {"fr": "Ce match", "en": "This match"},
    # ── Radar chart ─────────────────────────────────────────────────────────
    "radar_stats_per_min": {"fr": "Stats par minute", "en": "Stats per minute"},
    "radar_perf_profile": {"fr": "Profil de performance", "en": "Performance profile"},
    "radar_kpm": {"fr": "Frags/min", "en": "Kills/min"},
    "radar_dpm": {"fr": "Morts/min", "en": "Deaths/min"},
    "radar_apm": {"fr": "Assistances/min", "en": "Assists/min"},
    "radar_objective": {"fr": "Objectif", "en": "Objective"},
    "radar_survival": {"fr": "Survie", "en": "Survival"},
    "radar_participation_profile": {"fr": "Profil de participation", "en": "Participation profile"},
    "radar_objectives": {"fr": "Objectifs", "en": "Objectives"},
    "radar_combat": {"fr": "Combat", "en": "Combat"},
    "radar_support": {"fr": "Support", "en": "Support"},
    "radar_impact": {"fr": "Impact", "en": "Impact"},
    "radar_complementarity": {"fr": "Complémentarité", "en": "Complementarity"},
    "radar_kills_label": {"fr": "Frags", "en": "Kills"},
    "radar_assists_label": {"fr": "Assistances", "en": "Assists"},
    "radar_kills_pct": {"fr": "Frags %", "en": "Kills %"},
    "radar_assists_pct": {"fr": "Assistances %", "en": "Assists %"},
    "radar_obj_pct": {"fr": "Objectifs %", "en": "Objectives %"},
    "radar_kd": {"fr": "F/M", "en": "K/D"},
    "radar_session_evolution": {"fr": "Évolution du profil", "en": "Profile evolution"},
    "radar_win_rate": {"fr": "Taux victoires", "en": "Win rate"},
    "radar_avg_score": {"fr": "Score moy.", "en": "Avg score"},
    "radar_hover_deaths": {"fr": "morts", "en": "deaths"},
    "radar_hover_no_penalty": {"fr": "Aucune", "en": "None"},
    "radar_hover_survival_pct": {"fr": "survie", "en": "survival"},
    "radar_hover_pts_per_min": {"fr": "{pts:.1f} pts/min", "en": "{pts:.1f} pts/min"},
    # ── Radar — descriptions des axes ───────────────────────────────────────
    "radar_desc_objectives": {
        "fr": "**Objectifs** : contribution à la victoire (objectifs ou frags selon le mode)",
        "en": "**Objectives** : contribution to victory (objectives or kills depending on mode)",
    },
    "radar_desc_combat": {"fr": "**Combat** : éliminations directes", "en": "**Combat** : direct eliminations"},
    "radar_desc_support": {"fr": "**Support** : assists", "en": "**Support** : assists"},
    "radar_desc_score": {"fr": "**Score** : points totaux", "en": "**Score** : total points"},
    "radar_desc_impact": {"fr": "**Impact** : intensité (pts/min)", "en": "**Impact** : intensity (pts/min)"},
    "radar_desc_survival": {
        "fr": "**Survie** : moins de morts + durée de vie moyenne",
        "en": "**Survival** : fewer deaths + average lifespan",
    },
    # ── Career progress circle ──────────────────────────────────────────────
    # (uses career_max_rank, career_hero_rank from career section above)
    # ── Checkbox filter ─────────────────────────────────────────────────────
    "cbf_suffix_all": {"fr": "(tous)", "en": "(all)"},
    "cbf_suffix_none": {"fr": "(aucun)", "en": "(none)"},
    "cbf_confirm_clear": {
        "fr": "⚠️ Confirmer : vider toutes les sélections ?",
        "en": "⚠️ Confirm: clear all selections?",
    },
    # ── DuckDB analytics ────────────────────────────────────────────────────
    "dba_global_stats": {"fr": "📊 Stats globales (DuckDB)", "en": "📊 Global stats (DuckDB)"},
    "dba_win_rate": {"fr": "Taux victoires", "en": "Win rate"},
    "kpi_section_matches": {"fr": "Parties", "en": "Matches"},
    "kpi_section_career": {"fr": "Carrière", "en": "Career"},
    "flt_date_start": {"fr": "Début", "en": "Start"},
    "flt_date_end": {"fr": "Fin", "en": "End"},
    "flt_session_all": {"fr": "(toutes)", "en": "(all)"},
    "flt_session_last": {"fr": "Dernière session", "en": "Last session"},
    "flt_session_prev": {"fr": "Session précédente", "en": "Previous session"},
    "flt_session_trio": {"fr": "Dernière session en trio", "en": "Last trio session"},
    "flt_trio_caption": {"fr": "Trio : {label}", "en": "Trio: {label}"},
    "flt_session_select": {"fr": "Session", "en": "Session"},
    "hlp_db_empty_warning": {
        "fr": "La base sélectionnée est vide (0 octet). Basculement automatique vers une DB valide si possible.",
        "en": "The selected database is empty (0 bytes). Auto-switching to a valid DB if possible.",
    },
    "hlp_db_fallback_info": {"fr": "DB utilisée : {path}", "en": "DB used: {path}"},
    "hlp_btn_sync": {"fr": "🔄 Synchroniser", "en": "🔄 Sync"},
    "hlp_btn_sync_help": {
        "fr": "Synchronise tous les joueurs (nouveaux matchs, highlights, aliases).",
        "en": "Sync all players (new matches, highlights, aliases).",
    },
    "hlp_configure_db": {
        "fr": "Configure une DB et un joueur dans Paramètres.",
        "en": "Configure a DB and a player in Settings.",
    },
    "dba_avg_kda": {"fr": "FDA moyen", "en": "Avg KDA"},
    "dba_kda_evolution": {
        "fr": "📈 Évolution KDA (moyenne mobile {n} matchs)",
        "en": "📈 KDA evolution (rolling avg {n} matches)",
    },
    "dba_kda_recent": {"fr": "KDA (derniers {n})", "en": "KDA (last {n})"},
    "dba_kda_older": {"fr": "KDA (plus anciens {n})", "en": "KDA (older {n})"},
    "dba_trend": {"fr": "Tendance", "en": "Trend"},
    "dba_map_perf": {"fr": "🗺️ Performances par carte (DuckDB)", "en": "🗺️ Map performance (DuckDB)"},
    "dba_advanced": {
        "fr": "📊 Analytics avancées (DuckDB)",
        "en": "📊 Advanced analytics (DuckDB)",
    },
    "dba_advanced_help": {
        "fr": "Ces statistiques sont calculées via DuckDB directement.",
        "en": "These stats are calculated directly via DuckDB.",
    },
    "dba_no_parquet": {
        "fr": "Aucune donnée Parquet disponible.",
        "en": "No Parquet data available.",
    },
    # ── App helpers ─────────────────────────────────────────────────────────
    "app_db_used": {"fr": "DB utilisée : {path}", "en": "DB used: {path}"},
    "app_sync_all_help": {
        "fr": "Synchronise tous les joueurs configurés.",
        "en": "Sync all configured players.",
    },
    "app_no_match": {"fr": "Aucun match trouvé.", "en": "No match found."},
    "app_configure_db": {
        "fr": "Configure une DB et un joueur dans Paramètres.",
        "en": "Configure a DB and a player in Settings.",
    },
    "app_asset_error": {
        "fr": "Asset '{prefix}' non téléchargé : {err}",
        "en": "Asset '{prefix}' not downloaded: {err}",
    },
    "app_player_default": {"fr": "(joueur)", "en": "(player)"},
    "app_profile_api_err": {
        "fr": "Profil auto (SPNKr) : {err}",
        "en": "Profile auto (SPNKr): {err}",
    },
    # ── Objective analysis page (Phase 3) ───────────────────────────────────
    "obj_caption": {
        "fr": "Analysez votre contribution aux objectifs de jeu et découvrez votre profil de joueur.",
        "en": "Analyze your contribution to game objectives and discover your player profile.",
    },
    "obj_sync_hint": {
        "fr": "💡 Les tables `personal_score_awards` peuvent ne pas exister. Lancez une synchronisation pour les créer.",
        "en": "💡 The `personal_score_awards` tables may not exist. Please run a sync to create them.",
    },
    "obj_no_personal_score": {
        "fr": "⚠️ Aucune donnée de score personnel disponible. Synchronisez vos matchs pour obtenir ces données.",
        "en": "⚠️ No personal score data available. Sync your matches to get this data.",
    },
    "obj_help_obj_points": {
        "fr": "Points gagnés sur les objectifs de jeu",
        "en": "Points earned on game objectives",
    },
    "obj_help_kill_points": {
        "fr": "Points gagnés avec les éliminations",
        "en": "Points earned with eliminations",
    },
    "obj_help_assist_points": {
        "fr": "Points gagnés avec les assistances",
        "en": "Points earned with assists",
    },
    "obj_profile_desc_support": {
        "fr": "Vous contribuez fortement aux objectifs de l'équipe.",
        "en": "You contribute strongly to team objectives.",
    },
    "obj_profile_desc_balanced": {
        "fr": "Bon équilibre entre kills et objectifs.",
        "en": "Good balance between kills and objectives.",
    },
    "obj_profile_desc_slayer": {
        "fr": "Vous excellez dans les éliminations.",
        "en": "You excel at eliminations.",
    },
    "obj_scatter_caption": {
        "fr": "Chaque point représente un match. Les points au-dessus de la tendance indiquent une meilleure contribution aux objectifs.",
        "en": "Each point represents a match. Points above the trend line indicate better objective contribution.",
    },
    "obj_team_feature_hint": {
        "fr": "Cette fonctionnalité nécessite d'avoir synchronisé les données de tous les joueurs d'un match. Elle sera disponible dans une prochaine version.",
        "en": "This feature requires syncing all players' data from a match. It will be available in a future version.",
    },
    "obj_tip_improve_obj": {
        "fr": "🎯 **Pensez aux objectifs !**\n\nVotre ratio objectifs est faible. Dans les modes objectifs (CTF, Strongholds, etc.), contribuer aux objectifs rapporte plus de points à l'équipe.",
        "en": "🎯 **Think objectives!**\n\nYour objective ratio is low. In objective modes (CTF, Strongholds, etc.), focusing on objectives scores more points for the team.",
    },
    "obj_tip_great_support": {
        "fr": "🛡️ **Excellent joueur d'objectif !**\n\nVous êtes un pilier pour votre équipe sur les objectifs. Continuez à jouer le jeu d'équipe !",
        "en": "🛡️ **Excellent objective player!**\n\nYou are a cornerstone for your team on objectives. Keep playing the team game!",
    },
    "obj_tip_assists": {
        "fr": "🤝 **Grand fournisseur d'assists !**\n\nVous contribuez beaucoup aux éliminations de vos coéquipiers. Pensez à utiliser le ping et les EMP pour maximiser cet impact.",
        "en": "🤝 **Great assist provider!**\n\nYou contribute a lot to your teammates' eliminations. Consider using ping and EMP to maximize this impact.",
    },
    # ── Settings page (Phase 3) ──────────────────────────────────────────────
    "set_arch_v5_info": {
        "fr": "ℹ️ **Architecture v5** : La synchronisation récupère automatiquement TOUTES les données :\n- Nouveaux matchs (matchmaking uniquement)\n- Highlight events (clips)\n- Médailles\n- Stats skill/MMR\n- Personal score awards\n- Aliases XUID\n\nCes options ne sont plus configurables - tout est récupéré à chaque sync.",
        "en": "ℹ️ **Architecture v5**: The sync automatically fetches ALL data:\n- New matches (matchmaking only)\n- Highlight events (clips)\n- Medals\n- Skill/MMR stats\n- Personal score awards\n- XUID aliases\n\nThese options are no longer configurable — everything is fetched on each sync.",
    },
    "settings_backfill_caption": {
        "fr": "Configurez ce que fait le bouton 🔄 Synchroniser dans la sidebar. Le backfill remplit les données manquantes pour les matchs existants.",
        "en": "Configure what the 🔄 Sync button does in the sidebar. Backfill fills missing data for existing matches.",
    },
    "settings_backfill_all_help": {
        "fr": "Cochez pour backfill toutes les données, ou choisissez individuellement ci-dessous",
        "en": "Check to backfill all data, or choose individually below",
    },
    "set_backfill_score_help": {
        "fr": "Calcule les scores de performance manquants (peut être activé même sans backfill général)",
        "en": "Computes missing performance scores (can be enabled even without general backfill)",
    },
    "set_media_arch_info": {
        "fr": "ℹ️ **Architecture v5** : La section Médias est toujours active. Configurez le dossier de base et la tolérance temporelle.",
        "en": "ℹ️ **Architecture v5**: The Media section is always active. Configure the base folder and time tolerance.",
    },
    "set_media_root_help": {
        "fr": "Racine des captures. Un sous-dossier par joueur, nommé comme le gamertag (ex: D:/Captures/PlayerA/, D:/Captures/PlayerB/). Images et vidéos dans le même dossier.",
        "en": "Capture root. One subfolder per player, named like the gamertag (e.g. D:/Captures/PlayerA/, D:/Captures/PlayerB/). Images and videos in the same folder.",
    },
    # ── Match view helpers (Phase 3) ─────────────────────────────────────────
    "mvh_window_label": {
        "fr": "Fenêtre: {start} → {end}",
        "en": "Window: {start} → {end}",
    },
    "mvh_duration_estimated": {
        "fr": " *(durée estimée)*",
        "en": " *(estimated duration)*",
    },
    # ── Match view players (Phase 3) ─────────────────────────────────────────
    "mvp_attribution_deaths": {
        "fr": "Morts attribuées {assigned}/{total} (certain {certain}, estimé {estimated}, manquantes {missing})",
        "en": "Deaths attributed {assigned}/{total} (certain {certain}, estimated {estimated}, missing {missing})",
    },
    "mvp_attribution_kills": {
        "fr": "Kills attribués {assigned}/{total} (certain {certain}, estimé {estimated}, manquants {missing})",
        "en": "Kills attributed {assigned}/{total} (certain {certain}, estimated {estimated}, missing {missing})",
    },
    # ── Teammates charts (Phase 3) ───────────────────────────────────────────
    "tm_lifespan_with": {
        "fr": "{player} — Durée de vie (avec {partner})",
        "en": "{player} — Life span (with {partner})",
    },
}
