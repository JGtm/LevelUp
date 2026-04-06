"""Chaînes viz — titres de graphiques Plotly (38 clés)."""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
    # ── Titres graphiques combat ──────────────────────────────────────────────
    "title_avg_life": "col_avg_life_long",
    "title_performance": {
        "fr": "Score de performance",
        "en": "Performance score",
    },
    "title_streaks": {
        "fr": "Séries de victoires / défaites",
        "en": "Win / loss streaks",
    },
    "title_damage": {
        "fr": "Dégâts infligés et subis",
        "en": "Damage dealt and taken",
    },
    "title_shots": {
        "fr": "Tirs et précision",
        "en": "Shots and accuracy",
    },
    "title_rank_score": {
        "fr": "Rang et score personnel",
        "en": "Rank and personal score",
    },
    "title_rolling_kd": {
        "fr": "F/D Glissant ({window} matchs)",
        "en": "Rolling K/D ({window} matches)",
    },
    "title_session_comparison": {
        "fr": "Comparaison des Sessions",
        "en": "Session Comparison",
    },
    # ── Titres supplémentaires ────────────────────────────────────────────────
    "title_kda": {"fr": "Frags / Morts / Assists", "en": "Kills / Deaths / Assists"},
    "title_assists": {"fr": "Assists par match", "en": "Assists per Match"},
    "title_permin": {"fr": "Frags / Morts par minute", "en": "Kills / Deaths per Minute"},
    "title_killer_victim": {"fr": "Duels — Killer vs Victime", "en": "Duels — Killer vs Victim"},
    "title_kd_per_min": {"fr": "F/D par minute", "en": "K/D per Minute"},
    "title_score_breakdown": {"fr": "Score par catégorie", "en": "Score Breakdown"},
    "title_action_detail": {"fr": "Détail des actions", "en": "Action Detail"},
    "title_participation": {"fr": "Participation par match", "en": "Participation per Match"},
    "title_participation_by_match": {
        "fr": "Participation par match",
        "en": "Participation by Match",
    },
    "title_participation_detail": {
        "fr": "Détail de la participation",
        "en": "Participation Detail",
    },
    "title_participation_profile": {"fr": "Profil de participation", "en": "Participation Profile"},
    "title_obj_vs_kills": {"fr": "Objectifs vs Frags", "en": "Objectives vs Kills"},
    "title_obj_vs_kills_per_match": {
        "fr": "Objectifs vs Frags par match",
        "en": "Objectives vs Kills per Match",
    },
    "title_score_by_category": {"fr": "Score par catégorie", "en": "Score by Category"},
    "title_total_points_by_cat": {
        "fr": "Points totaux par catégorie",
        "en": "Total Points by Category",
    },
    "title_obj_ratio": {"fr": "Ratio objectifs", "en": "Objective Ratio"},
    "title_obj_ratio_pct": {"fr": "Ratio objectifs (%)", "en": "Objective Ratio (%)"},
    "title_assist_breakdown": {"fr": "Détail des assists", "en": "Assist Breakdown"},
    "title_assist_types": {"fr": "Types d'assists", "en": "Assist Types"},
    "title_obj_trend": {"fr": "Tendance objectifs", "en": "Objective Trend"},
    "title_obj_trend_per_match": {
        "fr": "Tendance objectifs par match",
        "en": "Objective Trend per Match",
    },
    "title_top_players_obj": {"fr": "Top joueurs — objectifs", "en": "Top Players — Objectives"},
    # ── Titres supplémentaires (Phase 3) ─────────────────────────────────────
    "title_cumul_net_score": {"fr": "Net Score Cumulé", "en": "Cumulative Net Score"},
    "title_cumul_kd": {"fr": "F/D Cumulé", "en": "Cumulative K/D"},
    "title_session_trend": {"fr": "Tendance de la Session", "en": "Session Trend"},
    "title_nemesis_victim": {"fr": "Némésis et Souffre-douleur", "en": "Nemesis & Punching Bag"},
    # ── Progression avancée ───────────────────────────────────────────────────
    "title_cumul_kd_ci": {"fr": "F/D Cumulé", "en": "Cumulative K/D"},
    "title_net_score_per_hour": {
        "fr": "Efficacité — Score / heure",
        "en": "Efficiency — Score / hour",
    },
    "title_ewma_kd": {"fr": "F/D Lissé", "en": "Smoothed K/D"},
    "title_regression_trend": {
        "fr": "Tendance",
        "en": "Trend",
    },
    "title_killer_victim_matrix": {"fr": "Matrice Killer-Victim", "en": "Killer-Victim Matrix"},
    "title_top_antagonists": {"fr": "Top Antagonistes", "en": "Top Antagonists"},
    "title_kd_timeline": {"fr": "F/D", "en": "K/D"},
    "title_elim_victim": {"fr": "Eliminateur-Victime", "en": "Eliminator-Victim"},
    # ── Squad timeline ────────────────────────────────────────────────────────
    "title_squad_timeline": {
        "fr": "Évolution de la performance d'escouade",
        "en": "Squad performance evolution",
    },
    # ── Graphe combiné HS + PK ────────────────────────────────────────────────
    "hs_pk_combined_title": {
        "fr": "Tirs à la tête & Frags parfaits",
        "en": "Headshots & Perfect Kills",
    },
    "hs_pk_y_axis": {
        "fr": "Nombre de kills",
        "en": "Kill count",
    },
    # ── Titres cadence / intensité ────────────────────────────────────────────
    "title_intensity_heatmap": {
        "fr": "Profil d'intensité par match",
        "en": "Intensity profile per match",
    },
    "title_squad_cadence": {
        "fr": "Profil de tempo synchronisé",
        "en": "Synchronized tempo profile",
    },
}
