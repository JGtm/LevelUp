"""Chaînes i18n — page Carrière."""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
    "career_header": {
        "fr": "Carrière",
        "en": "Career",
    },
    "career_no_data": {
        "fr": "Progression de rang non disponible. Synchronisez avec `--career` pour afficher votre rang et historique XP.",
        "en": "Rank progression not available. Sync with `--career` to display your rank and XP history.",
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
    "career_metric_rank": "col_rank",  # alias → common
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
    "career_computing": {
        "fr": "Chargement des données de carrière…",
        "en": "Loading career data…",
    },
    # ── Page Analyse Objectifs — manquants ────────────────────────────────────
    "career_xp_total": {"fr": "XP total", "en": "Total XP"},
    "career_xp_progress": {"fr": "Progression XP", "en": "XP progression"},
    "career_rank_n": {"fr": "Rang {n}", "en": "Rank {n}"},
    "career_rank_hover": {
        "fr": "Rang {rank}: {label}<br>XP total: {xp}",
        "en": "Rank {rank}: {label}<br>Total XP: {xp}",
    },
    "career_max_rank": {"fr": "Rang maximum atteint", "en": "Maximum rank reached"},
    "career_hero_rank": {"fr": "Rang Héros atteint !", "en": "Hero rank reached!"},
    "career_xp_estimated": {"fr": "XP estimé (pré-sync)", "en": "Estimated XP (pre-sync)"},
    "career_xp_estimated_hover": {
        "fr": "Estimation · {date}<br>~{xp} XP",
        "en": "Estimate · {date}<br>~{xp} XP",
    },
    "career_projection_hero": {"fr": "Projection → Héros", "en": "Projection → Hero"},
    "career_projection_hero_hover": {
        "fr": "Projection · {date}<br>{xp} XP",
        "en": "Projection · {date}<br>{xp} XP",
    },
    "career_projection_optimistic": {
        "fr": "Projection optimiste (défis + x2)",
        "en": "Optimistic (challenges + x2)",
    },
    "career_projection_optimistic_hover": {
        "fr": "Optimiste · {date}<br>{xp} XP",
        "en": "Optimistic · {date}<br>{xp} XP",
    },
    "career_hero_threshold": {"fr": "Seuil Héros", "en": "Hero threshold"},
    "career_xp_other_player": {"fr": "{gamertag}", "en": "{gamertag}"},
    "career_xp_other_player_hover": {
        "fr": "{gamertag} · {date}<br>{xp} XP",
        "en": "{gamertag} · {date}<br>{xp} XP",
    },
    "career_ranked": "ranked",  # alias → common
    "career_tactical": {"fr": "Tactique", "en": "Tactical"},
    # ── Section Rencontres & Antagonistes ────────────────────────────────────
    "career_encounters_header": {
        "fr": "🤝 Joueurs les plus croisés",
        "en": "🤝 Most encountered players",
    },
    "career_nemesis_header": {
        "fr": "💀 Top Némésis",
        "en": "💀 Top Nemesis",
    },
    "career_victims_header": {
        "fr": "🎯 Top Souffre-douleurs",
        "en": "🎯 Top Punching Bags",
    },
    "career_encounters_no_data": {
        "fr": "Pas de données de rencontres disponibles.",
        "en": "No encounter data available.",
    },
    "career_antagonists_no_data": {
        "fr": "Pas de données d'antagonistes. Lancez le backfill antagonistes pour calculer.",
        "en": "No antagonist data. Run the antagonist backfill to compute.",
    },
    "career_encounters_error": {
        "fr": "Impossible d'afficher les rencontres : {error}",
        "en": "Unable to display encounters: {error}",
    },
    # ── Filtre période rencontres ────────────────────────────────────────────
    "encounters_period_label": {
        "fr": "Période",
        "en": "Period",
    },
    "encounters_period_all": {
        "fr": "Tout",
        "en": "All time",
    },
    "encounters_period_2y": {
        "fr": "2 dernières années",
        "en": "Last 2 years",
    },
    "encounters_period_1y": {
        "fr": "Dernière année",
        "en": "Last year",
    },
    "encounters_period_1m": {
        "fr": "Dernier mois",
        "en": "Last month",
    },
    "encounters_period_1w": {
        "fr": "Dernière semaine",
        "en": "Last week",
    },
    "col_times_killed": {
        "fr": "Tués",
        "en": "Killed",
    },
    "col_times_killed_by": {
        "fr": "Mort par",
        "en": "Killed by",
    },
    "col_net_kills": {
        "fr": "Bilan",
        "en": "Net",
    },
    "col_matches_against": {
        "fr": "Matchs",
        "en": "Matches",
    },
    # ── Citations ───────────────────────────────────────────────────────────
}
