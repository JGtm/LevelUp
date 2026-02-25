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
        "en": "Quotes",
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
        "fr": "Citations (Commendations Halo 5)",
        "en": "Citations (Halo 5 Commendations)",
    },
    "citations_medals_title": {
        "fr": "Médailles (Halo Infinite)",
        "en": "Medals (Halo Infinite)",
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
}
