"""Chaînes i18n — page Victoires/Défaites."""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
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
        "fr": "Taux de victoires par jour et heure",
        "en": "Win rate by day and hour",
    },
    "wl_heatmap_caption": {
        "fr": "Identifie les créneaux horaires où tu performes le mieux (heure locale {tz}). Les cellules affichent le nombre de matchs.",
        "en": "Find the time slots where you perform best ({tz} local time). Cells show the number of matches.",
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
    "wl_metric_performance": {
        "fr": "Performance moyenne",
        "en": "Avg. performance",
    },
    "wl_session_map_note": {
        "fr": "ℹ️ En vue session (peu de matchs par carte), le ratio et le taux de victoires ne sont pas représentatifs — affiché : métriques de performance.",
        "en": "ℹ️ In session view (few matches per map), win ratio and win rate are not meaningful — showing performance metrics instead.",
    },
    "wl_personal_score_y_axis": {
        "fr": "Score personnel",
        "en": "Personal score",
    },
    "wl_personal_score_hover": "col_score",  # alias → common
    # ── Page Carrière — manquants ─────────────────────────────────────────────
    "wl_me_filtered": {"fr": "Moi (filtres actuels)", "en": "Me (current filters)"},
    "wl_me_all": {"fr": "Moi (toutes les parties)", "en": "Me (all matches)"},
    "wl_col_map": "col_map",  # alias → common
    "wl_col_matches": "lbl_parties",  # alias → common
    "wl_col_accuracy_avg": "col_avg_accuracy_pct",  # alias → common
    "wl_col_performance_avg": "col_avg_perf",  # alias → common
    "wl_col_win_rate": "col_win_rate_pct",  # alias → common
    "wl_col_loss_rate": "col_loss_rate_pct",  # alias → common
    "wl_col_ratio": "col_global_ratio",  # alias → common
    "wl_col_playlist": "col_playlist",  # alias → common
    "wl_col_mode": "col_mode",  # alias → common
    "wl_several": "lbl_several",  # alias → common
    "wl_period_col_wins": {"fr": "Victoires", "en": "Wins"},
    "wl_period_col_losses": {"fr": "Défaites", "en": "Losses"},
    "wl_period_col_draws": {"fr": "Égalités", "en": "Draws"},
    "wl_period_col_unfinished": {"fr": "Non terminés", "en": "Unfinished"},
    "wl_period_col_total": {"fr": "Total", "en": "Total"},
    "wl_period_col_win_rate": {"fr": "Taux de victoires", "en": "Win rate"},
    # ── Nouveaux graphiques par carte ────────────────────────────────────────
    "wl_map_lollipop_title": {
        "fr": "V/D par carte",
        "en": "W/L by map",
    },
    "wl_map_timeline_title": {
        "fr": "Évolution chronologique par carte",
        "en": "Chronological map evolution",
    },
    "wl_map_timeline_caption": {
        "fr": "Chaque cercle = 1 match. Les matchs de la sélection courante sont mis en évidence (plus grands, contour blanc).",
        "en": "Each dot = 1 match. Current selection highlighted (larger, white border).",
    },
    "wl_map_bullet_title": {
        "fr": "Taux de victoires vs historique",
        "en": "Session win rate vs history",
    },
    "wl_map_bullet_caption": {
        "fr": "Barre grise = taux de victoires sur toute ta carrière. Barre fine colorée = sélection actuelle (vert = meilleur, rouge = moins bien, ambre = équivalent).",
        "en": "Gray bar = lifetime win rate. Thin colored bar = current selection (green = better, red = worse, amber = similar).",
    },
    "wl_perf_vs_history_title": {
        "fr": "Performance vs historique",
        "en": "Performance vs history",
    },
    # ── Teammates ───────────────────────────────────────────────────────────
}
