"""Chaînes viz — labels, buckets, catégories, impacts, annotations, etc. (59 clés)."""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
    # ── Labels statuts résultat (alias → common.py) ──────────────────────────
    "label_win": "outcome_win",
    "label_loss": "outcome_loss",
    "label_tie": {
        "fr": "Égalité",
        "en": "Tie",
    },
    # ── Labels assists ────────────────────────────────────────────────────────
    "label_kill_assists": {"fr": "Kill Assists", "en": "Kill Assists"},
    "label_mark_assists": {"fr": "Mark Assists", "en": "Mark Assists"},
    "label_emp_assists": {"fr": "EMP Assists", "en": "EMP Assists"},
    # ── Labels duels / net ────────────────────────────────────────────────────
    "label_net": {"fr": "Net", "en": "Net"},
    "label_cumul": {"fr": "Cumul", "en": "Count"},
    # ── Performance cumulée ───────────────────────────────────────────────────
    "label_balance": {
        "fr": "Équilibre",
        "en": "Balance",
    },
    "label_target": {
        "fr": "Cible : {value}",
        "en": "Target: {value}",
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
    # ── Labels supplémentaires ────────────────────────────────────────────────
    "label_nemesis": "lbl_nemesis",
    "label_punching_bag": {"fr": "Punching bag", "en": "Punching Bag"},
    "label_session_a": {"fr": "Session A", "en": "Session A"},
    "label_session_b": {"fr": "Session B", "en": "Session B"},
    "label_this_match": {"fr": "Ce match", "en": "This Match"},
    # ── Team dominance timeline ───────────────────────────────────────────────
    "label_streak": {"fr": "série", "en": "streak"},
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
    # ── Catégories de participation ────────────────────────────────────────────
    "cat_label_kill": "col_kills",
    "cat_label_assist": {"fr": "Assists", "en": "Assists"},
    "cat_label_objective": {"fr": "Objectifs", "en": "Objectives"},
    "cat_label_vehicle": {"fr": "Véhicules", "en": "Vehicles"},
    "cat_label_penalty": {"fr": "Pénalités", "en": "Penalties"},
    "cat_label_other": {"fr": "Autre", "en": "Other"},
    # ── Impact timeline ────────────────────────────────────────────────────────
    "impact_first_blood": {"fr": "Premier sang", "en": "First Blood"},
    "impact_clutch_finisher": {"fr": "Finisseur", "en": "Finisher"},
    "impact_last_group_kill": {"fr": "Plus lent", "en": "Slowest"},
    "impact_first_group_death": {"fr": "Première victime", "en": "First Casualty"},
    "impact_last_casualty": {"fr": "Boulet", "en": "Weakest Link"},
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
    "empty_not_enough_matches": {
        "fr": "Pas assez de matchs pour analyser la tendance (min: 4)",
        "en": "Not enough matches to analyze trend (min: 4)",
    },
    "empty_no_streak_data": {
        "fr": "Aucune donnée de série disponible",
        "en": "No streak data available",
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
    # ── Suffixes indicateurs ──────────────────────────────────────────────────
    "suffix_deaths": {"fr": "morts", "en": "deaths"},
    "suffix_kills": {"fr": "kills", "en": "kills"},
    "suffix_smoothed": {"fr": "(lissée)", "en": "(smoothed)"},
    # ── Texte divers ──────────────────────────────────────────────────────────
    "text_kills_count": {
        "fr": "{k} frags",
        "en": "{k} kills",
    },
}
