"""Chaînes i18n — page Analyse des Objectifs."""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
    "obj_analysis_title": {
        "fr": "📊 Analyse des Objectifs",
        "en": "📊 Objective analysis",
    },
    "obj_comparison_coming_soon": {
        "fr": "🔜 Comparaison (à venir)",
        "en": "🔜 Comparison (coming soon)",
    },
    # ── Page Session Compare ─────────────────────────────────────────────────
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
}
