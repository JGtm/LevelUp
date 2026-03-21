"""Chaînes viz — noms de traces Plotly (56 clés)."""

from __future__ import annotations

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
        "fr": "Tendance lissée",
        "en": "Trend (smoothed)",
    },
    "trace_trend_r2": {
        "fr": "Tendance",
        "en": "Trend",
    },
    "trace_avg_smoothed": {
        "fr": "Moy. lissée",
        "en": "Average (smoothed)",
    },
    "trace_avg_3_smoothed": {
        "fr": "Moy. lissée (×3)",
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
        "fr": "CSR / LevelUp Skill Rank",
        "en": "CSR / LevelUp Skill Rank",
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
    # ── Traces supplémentaires ───────────────────────────────────────────────
    "trace_others": {
        "fr": "Autres",
        "en": "Others",
    },
    "trace_top_rate": {
        "fr": "Taux Top (%)",
        "en": "Top Rate (%)",
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
    "trace_kd_cumul": {
        "fr": "F/D Cumulé",
        "en": "Cumulative K/D",
    },
    "trace_kd_match": {
        "fr": "F/D du Match",
        "en": "Match K/D",
    },
    "trace_kd_rolling": {
        "fr": "F/D Glissant ({window})",
        "en": "Rolling K/D ({window})",
    },
    # ── Progression avancée ─────────────────────────────────────────────────────
    "trace_kd_ci_band": {"fr": "Zone de stabilité F/D", "en": "K/D Stability Band"},
    "trace_kd_ewma": {"fr": "F/D Lissé", "en": "Smoothed K/D"},
    "trace_regression_line": {"fr": "Tendance", "en": "Trend"},
    "trace_nph_positive": {"fr": "Efficacité positive", "en": "Positive efficiency"},
    "trace_nph_negative": {"fr": "Efficacité négative", "en": "Negative efficiency"},
    "trace_nph_raw": {"fr": "Score/h (match)", "en": "Score/h (per match)"},
    "trace_nph_rolling": {"fr": "Score/h (courbe)", "en": "Score/h (smoothed)"},
    "trace_outcome_markers": {"fr": "Résultats", "en": "Outcomes"},
    # ── Traces taux / objectifs ──────────────────────────────────────────────
    "trace_win_rate": {"fr": "Victoires", "en": "Win Rate"},
    "trace_loss_rate": {"fr": "Défaites", "en": "Loss Rate"},
    "trace_others_tie_unfinished": {"fr": "Égalités / Non terminés", "en": "Ties / Unfinished"},
    "trace_actual": "lbl_actual",
    "trace_expected": "lbl_expected",
    "trace_obj_score": {"fr": "Score objectif", "en": "Objective Score"},
    "trace_total_score": {"fr": "Score total", "en": "Total Score"},
    # ── Traces squad timeline ─────────────────────────────────────────────────
    "trace_squad_perf": {"fr": "Perf. escouade", "en": "Squad perf."},
    "trace_team_mmr": {"fr": "MMR équipe", "en": "Team MMR"},
    "trace_match_count": {"fr": "Matchs", "en": "Matches"},
}
