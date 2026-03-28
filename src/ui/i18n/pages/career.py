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
    "career_xp_estimated_note": {
        "fr": "\u26a0\ufe0f La courbe pointillée est une estimation. Les Career Ranks ont été introduits le 20 juin 2023 — tous les joueurs partaient de 0 à cette date. Seuls les matchs joués après cette date sont pris en compte dans le calcul.",
        "en": "\u26a0\ufe0f The dotted curve is an estimate. Career Ranks were introduced on June 20, 2023 — all players started at 0 on that date. Only matches played after that date are included in the calculation.",
    },
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
        "fr": "Projection optimiste (défis + boosts x2)",
        "en": "Optimistic (challenges + boosts x2)",
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
    "career_xp_other_estimated": {
        "fr": "{gamertag} (estimé)",
        "en": "{gamertag} (est.)",
    },
    "career_xp_other_estimated_hover": {
        "fr": "{gamertag} · {date} (estimé)<br>~{xp} XP",
        "en": "{gamertag} · {date} (est.)<br>~{xp} XP",
    },
    "career_projection_other_hero": {
        "fr": "Proj. {gamertag} → Héros",
        "en": "Proj. {gamertag} → Hero",
    },
    "career_projection_other_optimistic": {
        "fr": "Proj. {gamertag} (optimiste)",
        "en": "Proj. {gamertag} (optimistic)",
    },
    "career_projection_other_hover": {
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
    # ── Top 10 matchs marquants ─────────────────────────────────────────────
    "career_top_matches_header": {
        "fr": "Matchs marquants",
        "en": "Memorable Matches",
    },
    "career_top_btb_excluded": {
        "fr": "BTB exclus",
        "en": "BTB excluded",
    },
    "career_top_best_title": {
        "fr": "🏆 Meilleures performances",
        "en": "🏆 Best Performances",
    },
    "career_top_worst_title": {
        "fr": "💀 Pires performances",
        "en": "💀 Worst Performances",
    },
    "career_top_no_data": {
        "fr": "Pas assez de matchs pour établir un classement.",
        "en": "Not enough matches to build a ranking.",
    },
    "career_top_col_match_id": {
        "fr": "Match",
        "en": "Match",
    },
    "career_top_col_date": {
        "fr": "Date",
        "en": "Date",
    },
    "career_top_col_map": {
        "fr": "Carte",
        "en": "Map",
    },
    "career_top_col_mode": {
        "fr": "Mode",
        "en": "Mode",
    },
    "career_top_col_score": {
        "fr": "Score",
        "en": "Score",
    },
    "career_top_col_kda": {
        "fr": "F/D/A",
        "en": "K/D/A",
    },
    "career_top_col_kd": {
        "fr": "Ratio",
        "en": "K/D",
    },
    "career_top_col_duration": {
        "fr": "Durée",
        "en": "Duration",
    },
    "career_top_badge_domination": {
        "fr": "Domination",
        "en": "Domination",
    },
    "career_top_badge_humiliation": {
        "fr": "Humiliation",
        "en": "Humiliation",
    },
    "career_top_legend_domination": {
        "fr": 'Votre équipe a reçu la médaille "À table"',
        "en": 'Your team received the "Steaktacular" medal',
    },
    "career_top_legend_humiliation": {
        "fr": 'L\'équipe adverse a reçu la médaille "À table"',
        "en": 'The enemy team received the "Steaktacular" medal',
    },
    "career_top_badge_remontada": {
        "fr": "Remontada",
        "en": "Remontada",
    },
    "career_top_badge_debandade": {
        "fr": "Débandade",
        "en": "Collapse",
    },
    "career_top_badge_contre_remontada": {
        "fr": "Contre-Remontada",
        "en": "Held On",
    },
    "career_top_legend_remontada": {
        "fr": "Victoire après avoir été menés de manière significative durant un match",
        "en": "Victory after trailing significantly during a match",
    },
    "career_top_legend_debandade": {
        "fr": "Défaite après avoir mené le match significativement",
        "en": "Loss after leading significantly during a match",
    },
    "career_top_legend_contre_remontada": {
        "fr": "Victoire alors que le score des adversaires remontait fortement",
        "en": "Victory while the enemy mounted a significant comeback",
    },
    # ── Citations ───────────────────────────────────────────────────────────
}
