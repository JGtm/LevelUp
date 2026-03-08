"""Chaînes i18n — page Comparaison de sessions."""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
    "sc_performance_score": {
        "fr": "### 🏆 Score de performance",
        "en": "### 🏆 Performance score",
    },
    "sc_detailed_metrics": {
        "fr": "### 📊 Métriques détaillées",
        "en": "### 📊 Detailed metrics",
    },
    "sc_mmr_comparison": {
        "fr": "### 🎯 Comparaison MMR",
        "en": "### 🎯 MMR comparison",
    },
    "sc_comparative_charts": {
        "fr": "### 📈 Graphiques comparatifs",
        "en": "### 📈 Comparative charts",
    },
    "sc_match_history": {
        "fr": "### 📋 Historique des parties",
        "en": "### 📋 Match history",
    },
    "sc_participation_profile": {
        "fr": "#### 🎯 Évolution du profil de participation",
        "en": "#### 🎯 Participation profile over time",
    },
    "sc_radar_view": {
        "fr": "#### Vue radar",
        "en": "#### Radar view",
    },
    "sc_metric_comparison": {
        "fr": "#### Comparaison par métrique",
        "en": "#### Metric comparison",
    },
    "sc_metric_label": {
        "fr": "**Métrique**",
        "en": "**Metric**",
    },
    "sc_mmr_metric_label": {
        "fr": "**Métrique MMR**",
        "en": "**MMR metric**",
    },
    "sc_net_score_cumul": {
        "fr": "#### Net score cumulé par session",
        "en": "#### Cumulative net score per session",
    },
    # ── Page Dernière partie ─────────────────────────────────────────────────
    "sc_no_sessions": {
        "fr": "Aucune session disponible.",
        "en": "No sessions available.",
    },
    "sc_need_two_sessions": {
        "fr": "Il faut au moins 2 sessions pour comparer.",
        "en": "At least 2 sessions needed to compare.",
    },
    "sc_loading_caption": {
        "fr": "Compare les performances entre deux sessions de jeu.",
        "en": "Compare performance between two game sessions.",
    },
    # ── Page Coéquipiers — manquants ─────────────────────────────────────────
    "sc_no_matches_in_session": {
        "fr": "Aucune partie dans {session_name}.",
        "en": "No matches in {session_name}.",
    },
    "sc_participation_comparison": {
        "fr": "Comparaison de la contribution au score entre les deux sessions.",
        "en": "Comparison of score contribution between the two sessions.",
    },
    # ── Analyse d'objectifs — labels métriques ────────────────────────────────
    "sc_session_a_ref": {"fr": "Session A (référence)", "en": "Session A (reference)"},
    "sc_session_b_cmp": {"fr": "Session B (à comparer)", "en": "Session B (to compare)"},
    "sc_session_a": {"fr": "Session A", "en": "Session A"},
    "sc_session_b": {"fr": "Session B", "en": "Session B"},
    "sc_match_count": {"fr": "Nombre de parties", "en": "Number of matches"},
    "sc_kda_label": {"fr": "FDA (Frags-Décès-Assists)", "en": "KDA (Kills-Deaths-Assists)"},
    "sc_win_rate": "col_win_rate",  # alias → common
    "sc_avg_life": "col_avg_life_long",  # alias → common
    "sc_total_kills": {"fr": "Total des frags", "en": "Total kills"},
    "sc_total_deaths": {"fr": "Total des morts", "en": "Total deaths"},
    "sc_total_assists": {"fr": "Total des assistances", "en": "Total assists"},
    "sc_mmr_team_avg": {"fr": "MMR équipe (moy)", "en": "Team MMR (avg)"},
    "sc_mmr_enemy_avg": {"fr": "MMR adverse (moy)", "en": "Enemy MMR (avg)"},
    "sc_mmr_gap_avg": {"fr": "Écart MMR (moy)", "en": "MMR gap (avg)"},
    "sc_net_score_desc": {
        "fr": "Évolution du net score (Frags − Morts) match après match",
        "en": "Net score (Kills − Deaths) evolution match after match",
    },
    "sc_hist_avg_trace": {
        "fr": "Moy. historique ({n} sessions){suffix}",
        "en": "Hist. avg ({n} sessions){suffix}",
    },
    "sc_per_match_ratio": {"fr": "Par partie / Ratio", "en": "Per match / Ratio"},
    "sc_radar_kd": {"fr": "F/M", "en": "K/D"},
    "sc_radar_win": {"fr": "Victoire %", "en": "Win %"},
    "sc_kills_per_match": {"fr": "Frags / partie", "en": "Kills / match"},
    "sc_deaths_per_match": {"fr": "Morts / partie", "en": "Deaths / match"},
    "sc_kd_ratio": {"fr": "Ratio F/D", "en": "K/D ratio"},
    "sc_with_friends": {"fr": "avec {friends} 👥", "en": "with {friends} 👥"},
    "sc_with_n_friends": {"fr": "avec {n} ami(s) 👥", "en": "with {n} friend(s) 👥"},
    "sc_friends_unavailable": {"fr": "info amis indisponible", "en": "friends info unavailable"},
    "sc_solo": {"fr": "solo 🎮", "en": "solo 🎮"},
    "sc_compare_all": {
        "fr": "vs moy. {n} sessions {cat}",
        "en": "vs avg. {n} sessions {cat}",
    },
    "sc_compare_same_friends": {
        "fr": "vs moy. {n} sessions {cat} (mêmes amis)",
        "en": "vs avg. {n} sessions {cat} (same friends)",
    },
    "sc_compare_with_friends": {
        "fr": "vs moy. {n} sessions {cat} (avec amis)",
        "en": "vs avg. {n} sessions {cat} (with friends)",
    },
    "sc_compare_solo": {
        "fr": "vs moy. {n} sessions solo {cat}",
        "en": "vs avg. {n} solo sessions {cat}",
    },
    "sc_friends_suffix": {"fr": "ami(s) 👥", "en": "friend(s) 👥"},
    # ── Session compare charts ──────────────────────────────────────────────
    "scc_mmr_team": {"fr": "MMR Équipe", "en": "Team MMR"},
    "scc_mmr_enemy": {"fr": "MMR Adverse", "en": "Enemy MMR"},
    # ── Nouvelles sections graphiques ───────────────────────────────────────
    "sc_outcomes_distribution": {
        "fr": "### 🎯 Résultats des parties",
        "en": "### 🎯 Match results",
    },
    "sc_kd_progression": {
        "fr": "#### 📈 Évolution F/D par partie",
        "en": "#### 📈 K/D progression per match",
    },
    "sc_match_index": {"fr": "Partie", "en": "Match"},
    "sc_fd_ratio": {"fr": "F/D", "en": "K/D"},
    "sc_modes_breakdown": {
        "fr": "#### 🕹️ Modes joués",
        "en": "#### 🕹️ Modes played",
    },
    "sc_match_highlights": {
        "fr": "#### ⭐ Highlights",
        "en": "#### ⭐ Highlights",
    },
    "sc_best_match": {"fr": "🏆 Meilleur", "en": "🏆 Best"},
    "sc_worst_match": {"fr": "💀 Pire", "en": "💀 Worst"},
    "sc_map_table": {
        "fr": "#### 🗺️ Stats par carte",
        "en": "#### 🗺️ Map stats",
    },
    "sc_map_col": {"fr": "Carte", "en": "Map"},
    # ── Playlist categories ─────────────────────────────────────────────────
}
