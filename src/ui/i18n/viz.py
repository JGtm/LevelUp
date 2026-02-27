"""Chaînes pour les graphiques Plotly (noms de traces, axes, titres).

Ces chaînes sont utilisées dans src/visualization/*.py.
Les fonctions Plotly acceptent un paramètre ``lang: str = "fr"``
et appelent ``viz_t(key, lang)`` pour résoudre la chaîne.

⚠️ ChatGPT : remplir toutes les valeurs marquées "TODO" ci-dessous.
   Règles : voir le prompt de la Phase 1b dans le plan i18n.
"""

from __future__ import annotations

import contextlib

STRINGS: dict[str, dict[str, str] | str] = {
    # ── Traces communes (alias → common.py) ──────────────────────────────────
    "trace_kills": "col_kills",
    "trace_deaths": "col_deaths",
    "trace_ratio": "col_ratio",
    "trace_assists": "col_assists",
    "trace_matches": "col_matches",
    "trace_density": {
        "fr": "Densité (KDE)",
        "en": "Density (KDE)",
    },
    "trace_density_short": {
        "fr": "Densité",
        "en": "Density",
    },
    "trace_trend": {
        "fr": "Tendance (lissée)",
        "en": "Trend (smoothed)",
    },
    "trace_trend_r2": {
        "fr": "Tendance (R²={r2:.2f})",
        "en": "Trend (R²={r2:.2f})",
    },
    "trace_avg_smoothed": {
        "fr": "Moyenne (lissée)",
        "en": "Average (smoothed)",
    },
    "trace_avg_3_smoothed": {
        "fr": "Moyenne (3) lissée",
        "en": "Average (3) smoothed",
    },
    "trace_confidence": {
        "fr": "Confiance (σ)",
        "en": "Confidence (σ)",
    },
    "trace_first_kill": {
        "fr": "Premier frag",
        "en": "First kill",
    },
    "trace_first_death": {
        "fr": "Première mort",
        "en": "First death",
    },
    # ── Combat ───────────────────────────────────────────────────────────────
    "trace_lifespan": {
        "fr": "Durée de vie (s)",
        "en": "Life span (s)",
    },
    "trace_killing_spree": {
        "fr": "Folie meurtrière (max)",
        "en": "Killing Spree (max)",
    },
    "trace_headshots": {
        "fr": "Tirs à la tête",
        "en": "Headshots",
    },
    "trace_perfect_kills": "col_perfect_kills",
    "trace_performance": "col_performance",
    "trace_dmg_dealt": "col_dmg_dealt",
    "trace_dmg_dealt_avg": {
        "fr": "Moy. infligés",
        "en": "Avg. dealt",
    },
    "trace_dmg_taken": "col_dmg_taken",
    "trace_dmg_taken_avg": {
        "fr": "Moy. subis",
        "en": "Avg. taken",
    },
    "trace_shots_fired": {
        "fr": "Tirs tirés",
        "en": "Shots fired",
    },
    "trace_shots_hit": {
        "fr": "Tirs touchés",
        "en": "Shots hit",
    },
    "trace_accuracy": "col_accuracy_pct",
    "trace_personal_score": {
        "fr": "Score personnel",
        "en": "Personal score",
    },
    "trace_rank": "col_rank",
    # ── Stats par minute ─────────────────────────────────────────────────────
    "trace_kills_per_min": {
        "fr": "Frags/min",
        "en": "Kills/min",
    },
    "trace_deaths_per_min": {
        "fr": "Morts/min",
        "en": "Deaths/min",
    },
    "trace_assists_per_min": {
        "fr": "Assist./min",
        "en": "Assists/min",
    },
    "trace_avg_kills_per_min": {
        "fr": "Moy. frags/min",
        "en": "Avg. kills/min",
    },
    "trace_avg_deaths_per_min": {
        "fr": "Moy. morts/min",
        "en": "Avg. deaths/min",
    },
    "trace_avg_assists_per_min": {
        "fr": "Moy. assist./min",
        "en": "Avg. assists/min",
    },
    # ── LUSR / CSR ───────────────────────────────────────────────────────────
    "trace_lusr_default_title": {
        "fr": "LUSR — LevelUp Skill Rank",
        "en": "LUSR — LevelUp Skill Rank",
    },
    "trace_lusr_axis": {
        "fr": "Rating LUSR / CSR",
        "en": "LUSR / CSR rating",
    },
    # ── Équipes ──────────────────────────────────────────────────────────────
    "trace_my_team": {
        "fr": "Mon équipe",
        "en": "My team",
    },
    "trace_opponents": {
        "fr": "Adversaires",
        "en": "Opponents",
    },
    # ── Axes génériques (alias → common.py où possible) ──────────────────────────
    "axis_matches": "col_matches",
    "axis_accuracy_pct": "col_accuracy_pct",
    "axis_duration_sec": {
        "fr": "Durée (secondes)",
        "en": "Duration (seconds)",
    },
    "axis_score": "col_score",
    "axis_score_per_min": {
        "fr": "Score / min",
        "en": "Score / min",
    },
    "axis_win_rate_pct": {
        "fr": "Taux de victoire (%)",
        "en": "Win rate (%)",
    },
    "axis_frequency": {
        "fr": "Fréquence",
        "en": "Frequency",
    },
    "axis_personal_score": {
        "fr": "Score personnel",
        "en": "Personal score",
    },
    # ── Tableaux de données ──────────────────────────────────────────────────
    # NOTE: Les clés col_map, col_matches, col_avg_accuracy, col_win_rate,
    # col_loss_rate, col_global_ratio ont été supprimées (doublons de common.py
    # ou inutilisées via viz_t). viz_t() retombe sur common.STRINGS en fallback.
    # ── Traces victoires / défaites ──────────────────────────────────────────
    "trace_wins": {
        "fr": "Victoires",
        "en": "Wins",
    },
    "trace_losses": {
        "fr": "Défaites",
        "en": "Losses",
    },
    "trace_ties": {
        "fr": "Égalités",
        "en": "Ties",
    },
    "trace_unfinished": {
        "fr": "Non terminés",
        "en": "Unfinished",
    },
    # ── Axes / labels génériques ─────────────────────────────────────────────
    "axis_count": {
        "fr": "Nombre",
        "en": "Count",
    },
    "axis_hour": "lbl_hour",
    "axis_day_of_week": {
        "fr": "Jour",
        "en": "Day",
    },
    "axis_win_rate": {
        "fr": "Win Rate",
        "en": "Win Rate",
    },
    # ── Buckets temporels (labels retournés par les fonctions viz) ───────────
    "bucket_match": {
        "fr": "partie",
        "en": "match",
    },
    "bucket_hour": {
        "fr": "heure",
        "en": "hour",
    },
    "bucket_day": {
        "fr": "jour",
        "en": "day",
    },
    "bucket_week": {
        "fr": "semaine",
        "en": "week",
    },
    "bucket_month": {
        "fr": "mois",
        "en": "month",
    },
    "bucket_period": {
        "fr": "période",
        "en": "period",
    },
    # ── Labels statuts résultat (alias → common.py) ──────────────────────────
    "label_win": "outcome_win",
    "label_loss": "outcome_loss",
    "label_tie": {
        "fr": "Égalité",
        "en": "Tie",
    },
    # ── Hover templates (parties variables) ─────────────────────────────────
    "hover_wins": {
        "fr": "Victoires",
        "en": "Wins",
    },
    "hover_losses": {
        "fr": "Défaites",
        "en": "Losses",
    },
    "hover_ties": {
        "fr": "Égalités",
        "en": "Ties",
    },
    "hover_unfinished": {
        "fr": "Non terminés",
        "en": "Unfinished",
    },
    "hover_win_rate": {
        "fr": "Win Rate",
        "en": "Win Rate",
    },
    # ── Labels temporels (majuscule) — x-axis ────────────────────────────────
    "bucket_cap_match": "col_match",
    "bucket_cap_day": {
        "fr": "Jour",
        "en": "Day",
    },
    "bucket_cap_week": {
        "fr": "Semaine",
        "en": "Week",
    },
    # ── Traces supplémentaires ───────────────────────────────────────────────
    "trace_others": {
        "fr": "Autres",
        "en": "Others",
    },
    "trace_top_rate": {
        "fr": "Taux Top (%)",
        "en": "Top Rate (%)",
    },
    # ── Axes supplémentaires ─────────────────────────────────────────────────
    "axis_rate_pct": {
        "fr": "Taux (%)",
        "en": "Rate (%)",
    },
    "axis_hour_label": "lbl_hour",
    "axis_day_label": {
        "fr": "Jour",
        "en": "Day",
    },
    # ── Axes combat séries temporelles ───────────────────────────────────────
    "axis_seconds": {
        "fr": "Durée (s)",
        "en": "Duration (s)",
    },
    "axis_chronological": {
        "fr": "Match (chronologique)",
        "en": "Match (chronological)",
    },
    "axis_spree_headshots": {
        "fr": "Folie meurtrière / Tirs à la tête",
        "en": "Killing Spree / Headshots",
    },
    "axis_damage": {
        "fr": "Dégâts",
        "en": "Damage",
    },
    "axis_shots": "col_shots_fired_short",
    "axis_streak": {
        "fr": "Série",
        "en": "Streak",
    },
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
    "empty_no_streak_data": {
        "fr": "Aucune donnée de série disponible",
        "en": "No streak data available",
    },
    # ── Performance cumulée ───────────────────────────────────────────────────
    "trace_net_score_cumul": {
        "fr": "Net Score Cumulé",
        "en": "Cumulative Net Score",
    },
    "trace_net_score_match": {
        "fr": "Net Score du Match",
        "en": "Match Net Score",
    },
    "label_balance": {
        "fr": "Équilibre",
        "en": "Balance",
    },
    "axis_net_score": {
        "fr": "Net Score (Frags - Morts)",
        "en": "Net Score (Kills - Deaths)",
    },
    "trace_kd_cumul": {
        "fr": "K/D Cumulé",
        "en": "Cumulative K/D",
    },
    "trace_kd_match": {
        "fr": "K/D du Match",
        "en": "Match K/D",
    },
    "label_target": {
        "fr": "Cible : {value}",
        "en": "Target: {value}",
    },
    "axis_kd_ratio": {
        "fr": "K/D Ratio",
        "en": "K/D Ratio",
    },
    "trace_kd_rolling": {
        "fr": "K/D Glissant ({window})",
        "en": "Rolling K/D ({window})",
    },
    "label_improving": {
        "fr": "En progression",
        "en": "Improving",
    },
    "label_declining": {
        "fr": "En régression",
        "en": "Declining",
    },
    "label_stable": {
        "fr": "Stable",
        "en": "Stable",
    },
    "label_session_start": {
        "fr": "Début de session",
        "en": "Session start",
    },
    "label_session_end": {
        "fr": "Fin de session",
        "en": "Session end",
    },
    "label_kd_fm": {
        "fr": "F/M",
        "en": "K/D",
    },
    "label_kd_ref": {
        "fr": "K/D = 1.0",
        "en": "K/D = 1.0",
    },
    "title_rolling_kd": {
        "fr": "K/D Glissant ({window} matchs)",
        "en": "Rolling K/D ({window} matches)",
    },
    "title_session_comparison": {
        "fr": "Comparaison des Sessions",
        "en": "Session Comparison",
    },
    "axis_match_number": {
        "fr": "Match #",
        "en": "Match #",
    },
    "axis_cumul_net_score": {
        "fr": "Net Score Cumulé",
        "en": "Cumulative Net Score",
    },
    # ── Axes supplémentaires ─────────────────────────────────────────────────
    "axis_kills": "col_kills",
    "axis_deaths": "col_deaths",
    "axis_kills_deaths": {"fr": "Frags / Morts", "en": "Kills / Deaths"},
    "axis_ratio": {"fr": "Ratio K/D", "en": "K/D Ratio"},
    "axis_fda": {"fr": "FDA", "en": "FDA"},
    "axis_per_min": {"fr": "Par minute", "en": "Per Minute"},
    "axis_time_seconds": {"fr": "Durée (secondes)", "en": "Duration (seconds)"},
    "axis_match_count": {"fr": "Nombre de matchs", "en": "Number of Matches"},
    "axis_cumul": {"fr": "Cumulé", "en": "Cumulative"},
    "axis_frag_count": {"fr": "Nombre de frags", "en": "Frag Count"},
    "axis_minute": {"fr": "Minute", "en": "Minute"},
    "axis_points": {"fr": "Points", "en": "Points"},
    "axis_category": {"fr": "Catégorie", "en": "Category"},
    "axis_total_points": {"fr": "Points totaux", "en": "Total Points"},
    "axis_killer": {"fr": "Tueur", "en": "Killer"},
    "axis_victim": {"fr": "Victime", "en": "Victim"},
    "axis_kills_deaths_ratio": {"fr": "Frags — Morts", "en": "Kills — Deaths"},
    # ── Titres supplémentaires ────────────────────────────────────────────────
    "title_kda": {"fr": "Frags / Morts / Assists", "en": "Kills / Deaths / Assists"},
    "title_assists": {"fr": "Assists par match", "en": "Assists per Match"},
    "title_permin": {"fr": "Frags / Morts par minute", "en": "Kills / Deaths per Minute"},
    "title_killer_victim": {"fr": "Duels — Killer vs Victime", "en": "Duels — Killer vs Victim"},
    "title_kd_per_min": {"fr": "K/D par minute", "en": "K/D per Minute"},
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
    # ── Traces supplémentaires ────────────────────────────────────────────────
    "trace_win_rate": {"fr": "Victoires", "en": "Win Rate"},
    "trace_loss_rate": {"fr": "Défaites", "en": "Loss Rate"},
    "trace_others_tie_unfinished": {"fr": "Égalités / Non terminés", "en": "Ties / Unfinished"},
    "trace_actual": "lbl_actual",
    "trace_expected": "lbl_expected",
    "trace_obj_score": {"fr": "Score objectif", "en": "Objective Score"},
    "trace_total_score": {"fr": "Score total", "en": "Total Score"},
    # ── Hover supplémentaires ─────────────────────────────────────────────────
    "hover_kda": {
        "fr": "Frags: %{customdata[0]}<br>Morts: %{customdata[1]}<br>Assists: %{customdata[2]}",
        "en": "Kills: %{customdata[0]}<br>Deaths: %{customdata[1]}<br>Assists: %{customdata[2]}",
    },
    "hover_kill_ordinal": {"fr": "Frag n°%{x}", "en": "Kill #%{x}"},
    "hover_death_ordinal": {"fr": "Mort n°%{x}", "en": "Death #%{x}"},
    # ── Labels supplémentaires ────────────────────────────────────────────────
    "label_nemesis": "lbl_nemesis",
    "label_punching_bag": {"fr": "Punching bag", "en": "Punching Bag"},
    "label_session_a": {"fr": "Session A", "en": "Session A"},
    "label_session_b": {"fr": "Session B", "en": "Session B"},
    "label_this_match": {"fr": "Ce match", "en": "This Match"},
    # ── Catégories de participation ────────────────────────────────────────────
    "cat_label_kill": "col_kills",
    "cat_label_assist": {"fr": "Assists", "en": "Assists"},
    "cat_label_objective": {"fr": "Objectifs", "en": "Objectives"},
    "cat_label_vehicle": {"fr": "Véhicules", "en": "Vehicles"},
    "cat_label_penalty": {"fr": "Pénalités", "en": "Penalties"},
    "cat_label_other": {"fr": "Autre", "en": "Other"},
    # ── Impact timeline ────────────────────────────────────────────────────────
    "impact_first_blood": {"fr": "Premier sang", "en": "First Blood"},
    "impact_clutch_finisher": {"fr": "Clutch Finisher", "en": "Clutch Finisher"},
    "impact_last_group_kill": {"fr": "Dernier du groupe", "en": "Last Group Kill"},
    "impact_first_group_death": {"fr": "Premier mort du groupe", "en": "First Group Death"},
    # ── Messages vides ────────────────────────────────────────────────────────
    "empty_no_data": "no_data",
    "empty_no_duel": {"fr": "Aucun duel trouvé.", "en": "No duel found."},
    "empty_no_match_data": {
        "fr": "Aucune donnée de match disponible.",
        "en": "No match data available.",
    },
    "empty_no_impact_events": {
        "fr": "Aucun événement d'impact détecté.",
        "en": "No impact events detected.",
    },
    # ── Labels assists ────────────────────────────────────────────────────────
    "label_kill_assists": {"fr": "Kill Assists", "en": "Kill Assists"},
    "label_mark_assists": {"fr": "Mark Assists", "en": "Mark Assists"},
    "label_emp_assists": {"fr": "EMP Assists", "en": "EMP Assists"},
    "axis_assists": "col_assists",
    "axis_per_minute": {"fr": "Par minute", "en": "Per minute"},
    # ── Labels duels / net ────────────────────────────────────────────────────
    "label_net": {"fr": "Net", "en": "Net"},
    "label_cumul": {"fr": "Cumul", "en": "Count"},
    # ── Axe temps du match ────────────────────────────────────────────────────
    "axis_match_time": {"fr": "Temps du match", "en": "Match time"},
    # ── Suffixes indicateurs ──────────────────────────────────────────────────
    "suffix_deaths": {"fr": "morts", "en": "deaths"},
    "suffix_kills": {"fr": "kills", "en": "kills"},
    "suffix_smoothed": {"fr": "(moy. lissée)", "en": "(smoothed)"},
    # ── Titres supplémentaires (Phase 3) ─────────────────────────────────────
    "title_cumul_net_score": {"fr": "Net Score Cumulé", "en": "Cumulative Net Score"},
    "title_cumul_kd": {"fr": "K/D Cumulé", "en": "Cumulative K/D"},
    "title_session_trend": {"fr": "Tendance de la Session", "en": "Session Trend"},
    "title_nemesis_victim": {"fr": "Némésis et Souffre-douleur", "en": "Nemesis & Punching Bag"},
    "title_killer_victim_matrix": {"fr": "Matrice Killer-Victim", "en": "Killer-Victim Matrix"},
    "title_top_antagonists": {"fr": "Top Antagonistes", "en": "Top Antagonists"},
    "title_kd_timeline": {"fr": "K/D", "en": "K/D"},
    "title_elim_victim": {"fr": "Eliminateur-Victime", "en": "Eliminator-Victim"},
    # ── Hovertemplates (Phase 3) ───────────────────────────────────────────────
    "hover_kda_combined": {
        "fr": "frags=%{customdata[0]} morts=%{customdata[1]} assistances=%{customdata[2]}<br>précision=%{customdata[3]}% ratio=%{customdata[4]:.3f}<extra></extra>",
        "en": "kills=%{customdata[0]} deaths=%{customdata[1]} assists=%{customdata[2]}<br>accuracy=%{customdata[3]}% ratio=%{customdata[4]:.3f}<extra></extra>",
    },
    "hover_assists_combined": {
        "fr": "assistances=%{y}<br>frags=%{customdata[0]} morts=%{customdata[1]}<br>précision=%{customdata[3]}% ratio=%{customdata[4]:.3f}<extra></extra>",
        "en": "assists=%{y}<br>kills=%{customdata[0]} deaths=%{customdata[1]}<br>accuracy=%{customdata[3]}% ratio=%{customdata[4]:.3f}<extra></extra>",
    },
    "hover_avg_smoothed": {
        "fr": "moyenne=%{y:.2f}<extra></extra>",
        "en": "average=%{y:.2f}<extra></extra>",
    },
    "hover_avg_smoothed_s": {
        "fr": "moyenne=%{y:.2f}s<extra></extra>",
        "en": "average=%{y:.2f}s<extra></extra>",
    },
    "hover_avg_s1": {
        "fr": "moyenne=%{y:.1f}<extra></extra>",
        "en": "average=%{y:.1f}<extra></extra>",
    },
    "hover_perfect_sprees": {
        "fr": "frags parfaits=%{y}<extra></extra>",
        "en": "perfect sprees=%{y}<extra></extra>",
    },
    "hover_avg": {
        "fr": "moy=%{y:.2f}<extra></extra>",
        "en": "avg=%{y:.2f}<extra></extra>",
    },
    "hover_avg0": {
        "fr": "moy=%{y:.0f}<extra></extra>",
        "en": "avg=%{y:.0f}<extra></extra>",
    },
    "hover_kpm": {
        "fr": "frags/min=%{y:.2f}<br>temps joué=%{customdata[0]:.0f}s (frags=%{customdata[1]:.0f})<extra></extra>",
        "en": "kills/min=%{y:.2f}<br>time played=%{customdata[0]:.0f}s (kills=%{customdata[1]:.0f})<extra></extra>",
    },
    "hover_dpm": {
        "fr": "morts/min=%{y:.2f}<br>temps joué=%{customdata[0]:.0f}s (morts=%{customdata[2]:.0f})<extra></extra>",
        "en": "deaths/min=%{y:.2f}<br>time played=%{customdata[0]:.0f}s (deaths=%{customdata[2]:.0f})<extra></extra>",
    },
    "hover_apm": {
        "fr": "assist./min=%{y:.2f}<br>temps joué=%{customdata[0]:.0f}s (assistances=%{customdata[3]:.0f})<extra></extra>",
        "en": "assists/min=%{y:.2f}<br>time played=%{customdata[0]:.0f}s (assists=%{customdata[3]:.0f})<extra></extra>",
    },
    "hover_accuracy_pct": {
        "fr": "précision=%{y:.2f}%<extra></extra>",
        "en": "accuracy=%{y:.2f}%<extra></extra>",
    },
    "hover_lifespan": {
        "fr": "durée de vie moy.=%{y:.1f}s<br>morts=%{customdata[0]}<br>temps joué=%{customdata[1]:.0f}s<extra></extra>",
        "en": "avg lifespan=%{y:.1f}s<br>deaths=%{customdata[0]}<br>time played=%{customdata[1]:.0f}s<extra></extra>",
    },
    "hover_killing_spree": {
        "fr": "folie meurtrière=%{y}<extra></extra>",
        "en": "killing spree=%{y}<extra></extra>",
    },
    "hover_headshots": {
        "fr": "tirs à la tête=%{y}<extra></extra>",
        "en": "headshots=%{y}<extra></extra>",
    },
    "hover_streak": {
        "fr": "série=%{y}<br>date=%{customdata}<extra></extra>",
        "en": "streak=%{y}<br>date=%{customdata}<extra></extra>",
    },
    "hover_dmg_dealt": {
        "fr": "infligés=%{y:.0f}<extra></extra>",
        "en": "dealt=%{y:.0f}<extra></extra>",
    },
    "hover_dmg_taken": {
        "fr": "subis=%{y:.0f}<extra></extra>",
        "en": "taken=%{y:.0f}<extra></extra>",
    },
    "hover_shots_fired": {
        "fr": "tirés=%{y:.0f}<extra></extra>",
        "en": "fired=%{y:.0f}<extra></extra>",
    },
    "hover_shots_hit": {
        "fr": "touchés=%{y:.0f}<extra></extra>",
        "en": "hit=%{y:.0f}<extra></extra>",
    },
    "hover_net_kd_cumul": {
        "fr": "Min %{x}<br>Net K/D cumulé: %{y:+d}<extra></extra>",
        "en": "Round %{x}<br>Net K/D cumul: %{y:+d}<extra></extra>",
    },
    "hover_killed_by": {
        "fr": "<b>%{y}</b><br>M'a tué: %{x} fois<extra></extra>",
        "en": "<b>%{y}</b><br>Killed me: %{x} times<extra></extra>",
    },
    "hover_i_killed": {
        "fr": "<b>%{y}</b><br>Tué: %{x} fois<extra></extra>",
        "en": "<b>%{y}</b><br>Killed: %{x} times<extra></extra>",
    },
    "hover_kde": {
        "fr": "FDA=%{x:.2f}<br>densité=%{y:.3f}<extra></extra>",
        "en": "CDF=%{x:.2f}<br>density=%{y:.3f}<extra></extra>",
    },
    "hover_first_event": {
        "fr": "Temps: %{x:.0f}s<br>Matchs: %{y}<extra></extra>",
        "en": "Time: %{x:.0f}s<br>Matches: %{y}<extra></extra>",
    },
    "hover_weapons": {
        "fr": "%{y}<br>Frags: %{x}<br>Headshot: %{customdata[0]:.1f}%<br>Précision: %{customdata[1]:.1f}%<extra></extra>",
        "en": "%{y}<br>Kills: %{x}<br>Headshot: %{customdata[0]:.1f}%<br>Accuracy: %{customdata[1]:.1f}%<extra></extra>",
    },
    "hover_cumul_score": {
        "fr": "<b>%{x}</b><br>Cumulé: %{y:+d}<extra></extra>",
        "en": "<b>%{x}</b><br>Cumul: %{y:+d}<extra></extra>",
    },
    "hover_kd_cumul_line": {
        "fr": "<b>%{x}</b><br>K/D Cumulé: %{y:.2f}<extra></extra>",
        "en": "<b>%{x}</b><br>K/D Cumul: %{y:.2f}<extra></extra>",
    },
    "hover_kd_rolling_line": {
        "fr": "<b>%{x}</b><br>K/D Glissant: %{y:.2f}<extra></extra>",
        "en": "<b>%{x}</b><br>Rolling K/D: %{y:.2f}<extra></extra>",
    },
    "hover_match_cumul": {
        "fr": "<b>{label}</b><br>Match #%{{x}}<br>Cumulé: %{{y:+d}}<extra></extra>",
        "en": "<b>{label}</b><br>Match #%{{x}}<br>Cumul: %{{y:+d}}<extra></extra>",
    },
    "text_kills_count": {
        "fr": "{k} frags",
        "en": "{k} kills",
    },
    # ── Annotations (Phase 3) ─────────────────────────────────────────────────
    "annot_median": {
        "fr": "Médiane: {val}",
        "en": "Median: {val}",
    },
    "annot_avg_kill": {
        "fr": "Moy. frag: {val}s",
        "en": "Avg kill: {val}s",
    },
    "annot_avg_death": {
        "fr": "Moy. mort: {val}s",
        "en": "Avg death: {val}s",
    },
    "annot_med_kill": {
        "fr": "Méd. frag: {val}s",
        "en": "Med. kill: {val}s",
    },
    "annot_med_death": {
        "fr": "Méd. mort: {val}s",
        "en": "Med. death: {val}s",
    },
    "annot_penalties": {
        "fr": "Pénalités: {pts} pts",
        "en": "Penalties: {pts} pts",
    },
    # ── Messages vides supplémentaires (Phase 3) ──────────────────────────────
    "empty_not_enough_matches": {
        "fr": "Pas assez de matchs pour analyser la tendance (min: 4)",
        "en": "Not enough matches to analyze trend (min: 4)",
    },
    # ── Team dominance timeline ───────────────────────────────────────────────
    "label_streak": {"fr": "série", "en": "streak"},
    # ── Impact timeline hovertemplates ───────────────────────────────────────
    "hover_kill_cumul": {
        "fr": "<b>Frag #%{y}</b><br>%{text}<extra></extra>",
        "en": "<b>Kill #%{y}</b><br>%{text}<extra></extra>",
    },
    "hover_death_cumul": {
        "fr": "<b>Mort #%{y}</b><br>%{text}<extra></extra>",
        "en": "<b>Death #%{y}</b><br>%{text}<extra></extra>",
    },
}


def viz_t(key: str, lang: str = "fr", **kwargs: object) -> str:
    """Helper rapide pour les visualisations, sans dépendance Streamlit.

    Résout d'abord dans ``viz.STRINGS``, puis tombe en fallback sur
    ``common.STRINGS`` pour éviter de dupliquer les labels partagés
    (métriques, colonnes, résultats…).

    Supporte les **alias** : si la valeur dans STRINGS est un ``str``
    au lieu d'un ``dict``, elle est utilisée comme clé à résoudre
    (dans STRINGS d'abord, puis common.STRINGS).

    Args:
        key:  Clé dans STRINGS (ou common.STRINGS en fallback).
        lang: ``"fr"`` ou ``"en"``.
        **kwargs: Variables pour str.format().

    Returns:
        La chaîne traduite ou la clé entre crochets.
    """
    entry = STRINGS.get(key)

    # Résolution d'alias (str → clé cible)
    if isinstance(entry, str):
        target = entry
        entry = STRINGS.get(target)
        if entry is None or isinstance(entry, str):
            from src.ui.i18n.common import STRINGS as COMMON_STRINGS

            entry = COMMON_STRINGS.get(target)

    if entry is None:
        # Fallback vers common.STRINGS (import paresseux pour éviter circular)
        from src.ui.i18n.common import STRINGS as COMMON_STRINGS

        entry = COMMON_STRINGS.get(key)

    if entry is None:
        return f"[{key}]"
    if isinstance(entry, str):
        # Alias non résolu dans common — ne devrait pas arriver
        return entry
    text = entry.get(lang) or entry.get("fr") or f"[{key}]"
    if kwargs:
        with contextlib.suppress(KeyError, ValueError):
            text = text.format(**kwargs)
    return text
