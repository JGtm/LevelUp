"""Chaînes i18n — page Séries temporelles."""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
    "ts_fda": "col_kda",  # alias → common
    "ts_fda_unavailable": {
        "fr": "FDA indisponible sur ce filtre.",
        "en": "KDA is unavailable for this filter.",
    },
    "ts_cumulative": {
        "fr": "Performance cumulée & tendance",
        "en": "Cumulative performance & trend",
    },
    "ts_cumulative_caption": {
        "fr": "Efficacité (net score/heure), F/D cumulé avec intervalle de confiance à 90 %, F/D lissé EWMA et indicateurs de régression.",
        "en": "Efficiency (net score/hour), cumulative K/D with 90 % CI, EWMA smoothed K/D, and regression indicators.",
    },
    # ── Contrôles section Progression ────────────────────────────────────────
    "ts_ewma_alpha_label": {
        "fr": "Réactivité du lissage (α)",
        "en": "Smoothing reactivity (α)",
    },
    "ts_ewma_alpha_help": {
        "fr": "Petit α (0.10) = très lissé, tend vers la moyenne de long terme.\nGrand α (0.50) = très réactif, reflète ta forme récente.",
        "en": "Small α (0.10) = very smooth, tends toward long-term average.\nLarge α (0.50) = very reactive, reflects recent form.",
    },
    "ts_show_outcome_markers": {
        "fr": "Afficher V/D sur l'axe",
        "en": "Show W/L on axis",
    },
    "ts_section_cumulative": {
        "fr": "Bilan cumulatif",
        "en": "Cumulative overview",
    },
    "ts_section_recent": {
        "fr": "Forme récente",
        "en": "Recent form",
    },
    "ts_nph_unavailable": {
        "fr": "Net score/heure indisponible (colonne time_played_seconds manquante).",
        "en": "Net score/hour unavailable (missing time_played_seconds column).",
    },
    "ts_distributions": {
        "fr": "Distributions",
        "en": "Distributions",
    },
    "ts_distributions_caption": {
        "fr": "Visualise la répartition de tes statistiques.",
        "en": "Visualize the spread of your stats.",
    },
    "ts_correlations": {
        "fr": "Corrélations",
        "en": "Correlations",
    },
    "ts_correlations_caption": {
        "fr": "Analyse les relations entre tes métriques et le résultat du match.",
        "en": "Analyze relationships between your metrics and match outcomes.",
    },
    "ts_first_event": {
        "fr": "Temps du premier frag / première mort",
        "en": "Time to first kill / first death",
    },
    "ts_first_event_caption": {
        "fr": "Distribution des timestamps du premier frag et de la première mort. Visualise à quelle vitesse tu obtiens ton premier frag vs ta première mort.",
        "en": "Distribution of timestamps for your first kill and first death. See how fast you get your first kill vs your first death.",
    },
    "ts_first_event_no_data": {
        "fr": "Données d'événements non disponibles (premier frag / première mort). L'**Actualiser** récupère déjà ces données pour les **nouveaux** matchs. Pour les matchs déjà en base sans événements film, active dans **Paramètres** → **Options du bouton Actualiser** l'option **Backfill events**, puis **Actualiser**.",
        "en": "Event data is not available (first kill / first death). **Refresh** already fetches these for **new** matches. For existing matches without film events, enable **Backfill events** in **Settings** → **Refresh button options**, then **Refresh**.",
    },
    "ts_performance": "col_performance",  # alias → common
    "ts_assists": "col_assists",  # alias → common
    "ts_per_minute": {
        "fr": "Stats par minute",
        "en": "Per-minute stats",
    },
    "ts_lifespan": "col_avg_life_long",  # alias → common
    "ts_lifespan_unavailable": {
        "fr": "Average Life indisponible sur ce filtre.",
        "en": "Average Life is unavailable for this filter.",
    },
    "ts_spree": {
        "fr": "Folie meurtrière / Tirs à la tête / Frags parfaits",
        "en": "Killing Spree / Headshots / Perfect Kills",
    },
    "ts_shots": {
        "fr": "Tirs et précision",
        "en": "Shots & accuracy",
    },
    "ts_shots_caption": {
        "fr": "Tirs tirés vs touchés (barres groupées) et courbe de précision. La précision a été retirée du graphe Folie meurtrière pour une lecture plus claire.",
        "en": "Shots fired vs shots hit (grouped bars) and an accuracy line. Accuracy was removed from the Killing Spree chart for readability.",
    },
    "ts_damage": {
        "fr": "Dégâts",
        "en": "Damage",
    },
    "ts_damage_caption": {
        "fr": "Compare les dégâts infligés et subis par match. Un ratio élevé (infligés > subis) indique une bonne efficacité au combat.",
        "en": "Compare damage dealt and damage taken per match. A high ratio (dealt > taken) usually means strong combat efficiency.",
    },
    "ts_rank_score": {
        "fr": "Rang et score personnel",
        "en": "Rank and personal score",
    },
    "ts_rank_score_caption": {
        "fr": "Le score personnel en barres et le rang en ligne (axe Y inversé : rang 1 en haut). Un bon score associé à un rang élevé confirme l'impact.",
        "en": "Personal score as bars and rank as a line (inverted Y axis: rank 1 at the top). High score + high placement confirms impact.",
    },
    "ts_trend_min_matches": {
        "fr": "Tendance de session : au moins 4 matchs requis.",
        "en": "Session trend: at least 4 matches required.",
    },
    # ── Notes explicatives sous les graphes de progression ───────────────────
    "ts_note_nph": {
        "fr": "- Courbe qui **monte** → tu crées plus de valeur que tu n'en perds — bonne dynamique\n"
        "- **En dessous de 0** → tu meurs plus vite que tu ne fragges\n"
        "- Un pic isolé ne dit rien ; c'est la direction générale sur plusieurs matchs qui compte",
        "en": "- **Rising** → you generate more value than you lose — good momentum\n"
        "- **Below 0** → you die faster than you score kills\n"
        "- An isolated spike means nothing; the overall direction over several games is what matters",
    },
    "ts_note_ci": {
        "fr": "- Courbe en hausse en fin de session → tu t'es vraiment amélioré par rapport au début\n"
        "- **Bande étroite** → niveau stable et reproductible ; **bande large** → trop tôt pour conclure\\n"
        "- Cercle très éloigné de la courbe → partie atypique qui tire la moyenne",
        "en": "- Rising curve toward session end → genuine improvement relative to your start\n"
        "- **Narrow CI** → stable, reproducible level; **wide CI** → too early to draw conclusions\n"
        "- Circle far from the curve → atypical game skewing the average",
    },
    "ts_note_ewma": {
        "fr": "- Regarde la **direction générale** de la courbe lissée, pas les sommets ou creux individuels\n"
        "- La droite pointillée confirme une **vraie tendance** si R² ≥ 0,3 — en dessous, c'est du bruit\n"
        "- α élevé : utile pour détecter rapidement une rupture de forme dans la session",
        "en": "- Focus on the **general direction** of the smoothed curve, not individual peaks or dips\n"
        "- The dotted line confirms a **real trend** if R² ≥ 0.3 — below that threshold, it's noise\n"
        "- High α: useful for quickly detecting a shift in form during the session",
    },
    "ts_note_regression": {
        "fr": "- **Pente positive + R² ≥ 0,3** → progression réelle sur la session, pas un hasard\n"
        "- **R² < 0,3** → résultats trop éparpillés, impossible de tirer une conclusion fiable\n"
        "- Win rate en hausse + pente positive → double confirmation que la forme et les résultats s'alignent",
        "en": "- **Positive slope + R² ≥ 0.3** → genuine improvement across the session, not luck\n"
        "- **R² < 0.3** → results too scattered, no reliable conclusion possible\n"
        "- Rising win rate + positive slope → double confirmation that form and results align",
    },
    "ts_regression_subheader": {
        "fr": "Tendance (régression linéaire)",
        "en": "Trend (linear regression)",
    },
    # ── Page Coéquipiers ─────────────────────────────────────────────────────
    "ts_computing": {
        "fr": "Génération des graphes…",
        "en": "Generating charts…",
    },
    "ts_kda_mean_label": {
        "fr": "FDA moyen",
        "en": "Avg. KDA",
    },
    "ts_dist_accuracy_title": {
        "fr": "Distribution de la Précision",
        "en": "Accuracy distribution",
    },
    "ts_dist_kills_title": {
        "fr": "Distribution des Frags",
        "en": "Kill distribution",
    },
    "ts_dist_life_title": {
        "fr": "Distribution Durée de Vie",
        "en": "Life span distribution",
    },
    "ts_dist_perf_title": {
        "fr": "Distribution Score de Performance",
        "en": "Performance score distribution",
    },
    "ts_dist_score_per_min_title": {
        "fr": "Distribution Score Personnel / min",
        "en": "Personal score / min distribution",
    },
    "ts_dist_win_rate_title": {
        "fr": "Distribution Win Rate Glissant (5 matchs)",
        "en": "Rolling win rate distribution (5 matches)",
    },
    "ts_accuracy_label": "col_accuracy_pct",  # alias → common
    "ts_kills_label": "col_kills",  # alias → common
    "ts_life_label": {
        "fr": "Durée (secondes)",
        "en": "Duration (seconds)",
    },
    "ts_score_label": "col_score",  # alias → common
    "ts_matches_label": "col_matches",  # alias → common
    "ts_score_per_min_label": {
        "fr": "Score / min",
        "en": "Score / min",
    },
    "ts_win_rate_label": {
        "fr": "Taux de victoire (%)",
        "en": "Win rate (%)",
    },
    "ts_frequency_label": {
        "fr": "Fréquence",
        "en": "Frequency",
    },
    "ts_col_missing_cumul": {
        "fr": "Colonnes start_time, kills ou deaths manquantes pour la performance cumulée.",
        "en": "Columns start_time, kills or deaths missing for cumulative performance.",
    },
    "ts_col_missing_score_per_min": {
        "fr": "Colonnes score personnel ou time_played non disponibles.",
        "en": "Columns personal_score or time_played not available.",
    },
    "ts_col_missing_outcome": {
        "fr": "Colonne outcome non disponible.",
        "en": "Column outcome not available.",
    },
    "ts_insufficient_win_rate": {
        "fr": "Au moins 5 matchs requis pour le win rate glissant.",
        "en": "At least 5 matches required for rolling win rate.",
    },
    "ts_insufficient_win_rate_dist": {
        "fr": "Pas assez de données pour la distribution du win rate glissant (5 matchs minimum par fenêtre).",
        "en": "Not enough data for rolling win rate distribution (5 matches minimum per window).",
    },
    "ts_insufficient_score_per_min": {
        "fr": "Pas assez de données pour la distribution du score par minute.",
        "en": "Not enough data for score per minute distribution.",
    },
    "ts_corr_gen_error": {
        "fr": "Impossible de générer la corrélation {title}.",
        "en": "Unable to generate the {title} correlation.",
    },
    # ── Page Win/Loss — spinners & labels manquants ───────────────────────────
    "ts_lifespan_vs_kills": {"fr": "Durée de vie vs frags", "en": "Life span vs kills"},
    "ts_lifespan_s": {"fr": "Durée de vie (s)", "en": "Life span (s)"},
    "ts_accuracy_vs_kda": {"fr": "Précision vs FDA", "en": "Accuracy vs KDA"},
    "ts_lifespan_vs_deaths": {"fr": "Durée de vie vs morts", "en": "Life span vs deaths"},
    "ts_deaths_label": "col_deaths",  # alias → common
    "ts_kills_vs_deaths": {"fr": "Frags vs morts", "en": "Kills vs deaths"},
    "ts_mmr_team_vs_enemy": {"fr": "MMR Équipe vs MMR Adversaire", "en": "Team MMR vs Enemy MMR"},
    "ts_mmr_team": {"fr": "MMR Équipe", "en": "Team MMR"},
    "ts_mmr_enemy": {"fr": "MMR Adversaire", "en": "Enemy MMR"},
    "ts_events_unavailable": {
        "fr": "Données d'événements non disponibles. Lance un backfill events.",
        "en": "Event data not available. Run an events backfill.",
    },
    "ts_not_enough_dist": {
        "fr": "Pas assez de données ({count} matchs). Il en faut au moins {min} pour la distribution.",
        "en": "Not enough data ({count} matches). At least {min} are needed for the distribution.",
    },
    "ts_not_enough_corr": {
        "fr": "Pas assez de données ({count} matchs). Il en faut au moins {min} pour la corrélation.",
        "en": "Not enough data ({count} matches). At least {min} are needed for the correlation.",
    },
    # ── Armes (onglet K/D/A)
    "ts_top_weapons_title": {
        "fr": "Outils de destruction",
        "en": "Tools of destruction",
    },
    # ── Onglets (navigation)
    "ts_tab_kda": {"fr": "⚔️ F/D/A", "en": "⚔️ K/D/A"},
    "ts_tab_progression": {"fr": "📈 Progression", "en": "📈 Progression"},
    "ts_tab_distributions": {"fr": "📊 Distributions", "en": "📊 Distributions"},
    "ts_tab_advanced": {"fr": "🎯 Avancé", "en": "🎯 Advanced"},
}
