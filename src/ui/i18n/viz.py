"""Chaînes pour les graphiques Plotly (noms de traces, axes, titres).

Ces chaînes sont utilisées dans src/visualization/*.py.
Les fonctions Plotly acceptent un paramètre ``lang: str = "fr"``
et appelent ``viz_t(key, lang)`` pour résoudre la chaîne.

⚠️ ChatGPT : remplir toutes les valeurs marquées "TODO" ci-dessous.
   Règles : voir le prompt de la Phase 1b dans le plan i18n.
"""
from __future__ import annotations

STRINGS: dict[str, dict[str, str]] = {
    # ── Traces communes ──────────────────────────────────────────────────────
    "trace_kills": {
        "fr": "Frags",
        "en": "Kills",
    },
    "trace_deaths": {
        "fr": "Morts",
        "en": "Deaths",
    },
    "trace_ratio": {
        "fr": "Ratio",
        "en": "Ratio",
    },
    "trace_assists": {
        "fr": "Assistances",
        "en": "Assists",
    },
    "trace_matches": {
        "fr": "Matchs",
        "en": "Matches",
    },
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
    "trace_perfect_kills": {
        "fr": "Frags parfaits",
        "en": "Perfect kills",
    },
    "trace_performance": {
        "fr": "Performance",
        "en": "Performance",
    },
    "trace_dmg_dealt": {
        "fr": "Dégâts infligés",
        "en": "Damage dealt",
    },
    "trace_dmg_dealt_avg": {
        "fr": "Moy. infligés",
        "en": "Avg. dealt",
    },
    "trace_dmg_taken": {
        "fr": "Dégâts subis",
        "en": "Damage taken",
    },
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
    "trace_accuracy": {
        "fr": "Précision (%)",
        "en": "Accuracy (%)",
    },
    "trace_personal_score": {
        "fr": "Score personnel",
        "en": "Personal score",
    },
    "trace_rank": {
        "fr": "Rang",
        "en": "Rank",
    },
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
    # ── Axes génériques ──────────────────────────────────────────────────────
    "axis_matches": {
        "fr": "Matchs",
        "en": "Matches",
    },
    "axis_accuracy_pct": {
        "fr": "Précision (%)",
        "en": "Accuracy (%)",
    },
    "axis_duration_sec": {
        "fr": "Durée (secondes)",
        "en": "Duration (seconds)",
    },
    "axis_score": {
        "fr": "Score",
        "en": "Score",
    },
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
    "col_map": {
        "fr": "Carte",
        "en": "Map",
    },
    "col_matches": {
        "fr": "Parties",
        "en": "Matches",
    },
    "col_avg_accuracy": {
        "fr": "Précision moy. (%)",
        "en": "Avg. accuracy (%)",
    },
    "col_win_rate": {
        "fr": "Taux victoire (%)",
        "en": "Win rate (%)",
    },
    "col_loss_rate": {
        "fr": "Taux défaite (%)",
        "en": "Loss rate (%)",
    },
    "col_global_ratio": {
        "fr": "Ratio global",
        "en": "Overall ratio",
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
    # ── Axes / labels génériques ─────────────────────────────────────────────
    "axis_count": {
        "fr": "Nombre",
        "en": "Count",
    },
    "axis_hour": {
        "fr": "Heure",
        "en": "Hour",
    },
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
    # ── Labels statuts résultat (usage dans antagonists, summaries) ──────────
    "label_win": {
        "fr": "Victoire",
        "en": "Win",
    },
    "label_loss": {
        "fr": "Défaite",
        "en": "Loss",
    },
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
    "bucket_cap_match": {
        "fr": "Match",
        "en": "Match",
    },
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
    "axis_hour_label": {
        "fr": "Heure",
        "en": "Hour",
    },
    "axis_day_label": {
        "fr": "Jour",
        "en": "Day",
    },
}


def viz_t(key: str, lang: str = "fr", **kwargs: object) -> str:
    """Helper rapide pour les visualisations, sans dépendance Streamlit.

    Args:
        key:  Clé dans STRINGS.
        lang: ``"fr"`` ou ``"en"``.
        **kwargs: Variables pour str.format().

    Returns:
        La chaîne traduite ou la clé entre crochets.
    """
    entry = STRINGS.get(key)
    if entry is None:
        return f"[{key}]"
    text = entry.get(lang) or entry.get("fr") or f"[{key}]"
    if kwargs:
        try:
            text = text.format(**kwargs)
        except (KeyError, ValueError):
            pass
    return text
