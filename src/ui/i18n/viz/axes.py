"""Chaînes viz — labels d'axes Plotly (45 clés)."""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
    # ── Axes génériques (alias → common.py où possible) ──────────────────────
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
    # ── Performance cumulée ───────────────────────────────────────────────────
    "axis_net_score": {
        "fr": "Net Score (Frags - Morts)",
        "en": "Net Score (Kills - Deaths)",
    },
    "axis_kd_ratio": {
        "fr": "F/D Ratio",
        "en": "K/D Ratio",
    },
    "axis_match_number": {
        "fr": "Match #",
        "en": "Match #",
    },
    "axis_cumul_net_score": {
        "fr": "Net Score Cumulé",
        "en": "Cumulative Net Score",
    },
    # ── Axes supplémentaires (batch) ─────────────────────────────────────────
    "axis_kills": "col_kills",
    "axis_deaths": "col_deaths",
    "axis_kills_deaths": {"fr": "Frags / Morts", "en": "Kills / Deaths"},
    "axis_ratio": {"fr": "Ratio F/D", "en": "K/D Ratio"},
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
    # ── Labels assists ────────────────────────────────────────────────────────
    "axis_assists": "col_assists",
    "axis_per_minute": {"fr": "Par minute", "en": "Per minute"},
    # ── Axe temps du match ────────────────────────────────────────────────────
    "axis_match_time": {"fr": "Temps du match", "en": "Match time"},
    # ── Axe score différentiel K/D (timeline match) ──────────────────────────
    "axis_kd_score": {"fr": "Score Frags-Décès", "en": "K/D Score"},
    # ── Progression avancée ───────────────────────────────────────────────────
    "axis_net_per_hour": {"fr": "Net Score / heure", "en": "Net Score / hour"},
    # ── Axes squad timeline ───────────────────────────────────────────────────
    "axis_performance": {"fr": "Performance (0-100)", "en": "Performance (0-100)"},
    "axis_team_mmr": {"fr": "MMR Équipe", "en": "Team MMR"},
    "axis_sessions": {"fr": "Sessions", "en": "Sessions"},
}
