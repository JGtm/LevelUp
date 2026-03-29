"""Chaînes i18n — page Coéquipiers (impact, synergy, helpers)."""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
    "tm_by_map": {
        "fr": "Par carte — avec mes coéquipiers",
        "en": "By map — with my teammates",
    },
    "tm_history": {
        "fr": "Historique — matchs avec mes coéquipiers",
        "en": "History — matches with my teammates",
    },
    "tm_history_tz_caption": {
        "fr": "Dates et heures en heure locale ({tz}).",
        "en": "Dates and times displayed in local time ({tz}).",
    },
    "tm_medals": {
        "fr": "Médailles (matchs partagés)",
        "en": "Medals (shared matches)",
    },
    "tm_medals_all": {
        "fr": "Médailles",
        "en": "Medals",
    },
    "tm_per_minute": {
        "fr": "Stats par minute",
        "en": "Per-minute stats",
    },
    "tm_no_matches_teammate": {
        "fr": "Aucun match trouvé avec ce coéquipier (selon le filtre).",
        "en": "No matches found with this teammate (based on the current filter).",
    },
    "tm_not_enough_matches": {
        "fr": "Pas assez de matchs avec tes coéquipiers (selon le filtre actuel).",
        "en": "Not enough matches with your teammates (based on the current filter).",
    },
    "tm_no_matches_filter": {
        "fr": "Aucun match trouvé avec tes coéquipiers (selon le filtre actuel).",
        "en": "No matches found with your teammates (based on the current filter).",
    },
    "tm_no_shared_medals": {
        "fr": "Aucun match partagé pour calculer les médailles.",
        "en": "No shared matches to compute medals.",
    },
    "tm_no_medals_aggregate": {
        "fr": "Impossible de déterminer la liste des matchs pour l'agrégation des médailles.",
        "en": "Unable to determine the match list for medals aggregation.",
    },
    "tm_trio_session": {
        "fr": "Dernière session détectée : {label}.",
        "en": "Latest session detected: {label}.",
    },
    "tm_trio_session_unknown": {
        "fr": "Impossible de déterminer une session trio (données insuffisantes).",
        "en": "Unable to determine a trio session (not enough data).",
    },
    "tm_impact_header": {
        "fr": "⚡ Impact",
        "en": "⚡ Impact",
    },
    "tm_impact_select_two": {
        "fr": "Sélectionnez au moins 2 coéquipiers pour voir l'analyse d'impact.",
        "en": "Select at least 2 teammates to see the impact analysis.",
    },
    "tm_impact_no_matches": {
        "fr": "Aucun match à analyser.",
        "en": "No matches to analyze.",
    },
    "tm_impact_no_events_matches": {
        "fr": "Aucun événement trouvé pour les matchs sélectionnés.",
        "en": "No impact events found for the selected matches.",
    },
    "tm_impact_no_events_players": {
        "fr": "Aucun événement d'impact trouvé pour les joueurs sélectionnés.",
        "en": "No impact events found for the selected players.",
    },
    "tm_impact_heatmap": {
        "fr": "Matrice d'Impact",
        "en": "Impact heatmap",
    },
    "tmi_viz_heatmap": {
        "fr": "Matrice (original)",
        "en": "Matrix (original)",
    },
    "tmi_viz_scatter": {
        "fr": "Points (symboles)",
        "en": "Dots (symbols)",
    },
    "tm_impact_ranking": {
        "fr": "Matrice d'impact",
        "en": "Impact matrix",
    },
    "tm_impact_legend": {
        "fr": "- ⚡ **Premier sang** — premier kill du match, toutes équipes confondues\n- 🎯 **Finisseur** — dernier kill d'un match remporté\n- 💀 **Boulet** — dernière mort d'une défaite\n- 🐌 **Touriste** — dernier à obtenir son 1er kill\n- 🪦 **Première victime** — première mort du match, toutes équipes confondues\n- 🛡️ **Héros silencieux** — plus d'assists et moins de morts dans l'équipe (victoire)\n- 🗡️ **Faux-frère** — plus de morts et moins d'assists dans l'équipe (défaite)",
        "en": "- ⚡ **First Blood** — first kill of the match, across all teams\n- 🎯 **Finisher** — last kill of a won match\n- 💀 **Dead Weight** — last death of a lost match\n- 🐌 **Tourist** — last to get their first kill\n- 🪦 **First Casualty** — first death of the match, across all teams\n- 🛡️ **Silent Hero** — most assists and fewest deaths in the team (win only)\n- 🗡️ **False Brother** — most deaths and fewest assists in the team (loss only)",
    },
    # ── Page Match View ──────────────────────────────────────────────────────
    "tm_computing_teammate": {
        "fr": "Chargement des matchs avec ce coéquipier…",
        "en": "Loading matches with this teammate…",
    },
    "tm_computing_map": {
        "fr": "Calcul du ratio par carte (coéquipiers)…",
        "en": "Computing map ratio (teammates)…",
    },
    "tm_computing_stats": {
        "fr": "Chargement des stats des coéquipiers…",
        "en": "Loading teammate stats…",
    },
    "tm_computing_medals": {
        "fr": "Agrégation des médailles (moi + coéquipier)…",
        "en": "Aggregating medals (me + teammate)…",
    },
    "tm_computing_medals_all": {
        "fr": "Agrégation des médailles…",
        "en": "Aggregating medals…",
    },
    "tm_trio_header": {
        "fr": "Tous les trois — {f1} + {f2}",
        "en": "All three — {f1} + {f2}",
    },
    "tm_squad_header": {
        "fr": "Escouade — {names}",
        "en": "Squad — {names}",
    },
    # ── Page Match View — manquants ───────────────────────────────────────────
    "tm_map_stats_no_data": {
        "fr": "Données insuffisantes pour le ratio par carte.",
        "en": "Not enough data for the map ratio.",
    },
    "tm_trio_warning": {
        "fr": "Impossible d'aligner les stats des 3 joueurs sur ces matchs.",
        "en": "Unable to align stats for 3 players on these matches.",
    },
    # ── Media ─────────────────────────────────────────────────────────────────
    # media_disabled, media_no_folder : définis dans common.py
    "tm_kd_half_caption": {
        "fr": "F/M 1ère moitié → 2nde moitié des matchs affichés.",
        "en": "K/D first half → second half of displayed matches.",
    },
    "tm_select_teammate": {
        "fr": "Sélectionne au moins un coéquipier.",
        "en": "Select at least one teammate.",
    },
    # ── Match View ────────────────────────────────────────────────────────────
    "tm_weapon_kills_chart": {"fr": "Outils de destruction", "en": "Tools of destruction"},
    "tm_killing_spree": {"fr": "Folie meurtrière (max)", "en": "Killing Spree (max)"},
    "tm_headshots": {"fr": "Tirs à la tête", "en": "Headshots"},
    "tm_perfect_kills": {"fr": "Frags parfaits", "en": "Perfect kills"},
    "tm_kills": "col_kills",  # alias → common
    "tm_deaths": "col_deaths",  # alias → common
    "tm_kills_deaths": {"fr": "Frags ↑ | ↓ Morts", "en": "Kills ↑ | ↓ Deaths"},
    "tm_assists": {"fr": "Assistances", "en": "Assists"},
    # ── Match View Participation ──────────────────────────────────────────────
    "tm_kda": "col_kda",  # alias → common
    "tm_accuracy": "col_accuracy",  # alias → common
    "tm_avg_life": "col_avg_life_long",  # alias → common
    "tm_performance": "col_performance",  # alias → common
    "tm_score": "col_score",  # alias → common
    "tm_seconds": {"fr": "Secondes", "en": "Seconds"},
    # ── Noms des pages (navigation / onglets) ────────────────────────────────
    "tm_no_matches_filters": {
        "fr": "Aucun match à afficher avec les filtres actuels.",
        "en": "No matches to display with current filters.",
    },
    "tm_min_matches_map": {"fr": "Minimum de matchs par carte", "en": "Min matches per map"},
    "tm_ratio_map_header": {
        "fr": "Ratio global par carte — avec mes coéquipiers (min {n} matchs)",
        "en": "Overall ratio by map — with my teammates (min {n} matches)",
    },
    "tm_no_trio_matches": {
        "fr": "Aucun match trouvé où vous êtes tous les trois dans l'équipe.",
        "en": "No matches found where all three of you are on the same team.",
    },
    "tm_match_details_title": {
        "fr": "Détails des matchs (joueur vs joueur)",
        "en": "Match details (player vs player)",
    },
    "tm_win_loss": {"fr": "Win/Loss", "en": "Win/Loss"},
    "tm_metric_matches": "col_matches",  # alias → common
    "tm_metric_global_ratio": "col_global_ratio",  # alias → common
    "tm_metric_frags_min": "col_kpm",  # alias → common
    "tm_metric_deaths_min": "col_dpm",  # alias → common
    "tm_metric_assists_min": "col_apm",  # alias → common
    "tm_loading_slow": {
        "fr": "Le chargement sera plus lent car les données sont lues depuis la base partagée.",
        "en": "Loading will be slower as data is read from the shared database.",
    },
    "tm_apply_filters": {
        "fr": "Appliquer les filtres actuels (période/sessions + map/playlist)",
        "en": "Apply current filters (period/sessions + map/playlist)",
    },
    "tm_same_team": {"fr": "Même équipe", "en": "Same team"},
    "tm_show_smoothed": {"fr": "Afficher les courbes lissées", "en": "Show smoothed curves"},
    "tm_smoothed_help": {
        "fr": "Active/désactive les courbes de moyenne mobile sur l'ensemble de la période.",
        "en": "Toggle rolling average curves over the entire period.",
    },
    "tm_select_teammates": {"fr": "Coéquipiers", "en": "Teammates"},
    "tm_search_placeholder": {"fr": "Taper pour filtrer...", "en": "Type to filter..."},
    "tm_session_trend": {"fr": "Tendance de session", "en": "Session trend"},
    # ── En-tête session escouade ──────────────────────────────────────────────
    "squad_score_header": {"fr": "Score d'équipe", "en": "Squad Score"},
    "squad_score_bonus": {
        "fr": "moy. {base} (+{bonus} collectif)",
        "en": "avg. {base} (+{bonus} squad)",
    },
    "squad_legend": {
        "fr": "▲ au-dessus de la moyenne d'équipe · ▼ en dessous",
        "en": "▲ above squad average · ▼ below",
    },
    "squad_grade_legendaire": {"fr": "LEGENDAIRE", "en": "LEGENDARY"},
    "squad_grade_carnage": {"fr": "CARNAGE", "en": "CARNAGE"},
    "squad_grade_solide": {"fr": "SOLIDE", "en": "SOLID"},
    "squad_grade_moyen": {"fr": "MOYEN", "en": "AVERAGE"},
    "squad_grade_difficile": {"fr": "DIFFICILE", "en": "ROUGH"},
    "squad_grade_catastrophique": {"fr": "CATASTROPHIQUE", "en": "CATASTROPHIC"},
    "tm_solo_session_info": {
        "fr": "Cette session est en solo — les stats de coéquipiers ne sont pas disponibles.",
        "en": "You played solo this session — teammate stats are not available.",
    },
    # ── Teammates impact ────────────────────────────────────────────────────
    "tmi_first_blood": {"fr": "🟢 Premier Sang", "en": "🟢 First Blood"},
    "tmi_finisher": {"fr": "🟡 Finisseur", "en": "🟡 Finisher"},
    "tmi_liability": {"fr": "🔴 Boulet", "en": "🔴 Liability"},
    "tmi_matches_analyzed": {"fr": "📊 Matchs analysés", "en": "📊 Matches analyzed"},
    "tmi_badge": {"fr": "Badge", "en": "Badge"},
    "tmi_col_rank": "col_rank",  # alias → common
    "tmi_col_player": "col_player",  # alias → common
    "tmi_col_score": "col_score",  # alias → common
    "tmi_col_first_blood": {"fr": "⚡", "en": "⚡"},
    "tmi_col_finisher": {"fr": "🎯", "en": "🎯"},
    "tmi_col_casualty": {"fr": "💀", "en": "💀"},
    "tmi_mvp_label": {"fr": "🏆 Champion : {mvp}", "en": "🏆 MVP of the Night: {mvp}"},
    "tmi_boulet_label": {"fr": "🍌 Maillon Faible : {boulet}", "en": "🍌 Weak Link: {boulet}"},
    "tmi_no_shared_db": {
        "fr": "Impossible d'accéder à shared_matches.duckdb pour les événements.",
        "en": "Cannot access shared_matches.duckdb for events.",
    },
    "tmi_no_events": {
        "fr": "Les données d'événements ne sont pas disponibles. Lance un backfill events.",
        "en": "Event data is not available. Run an events backfill.",
    },
    # ── Teammates synergy ───────────────────────────────────────────────────
    "tms_participation_title": {"fr": "Profil de participation", "en": "Participation profile"},
    "tms_axes": {"fr": "**Axes**", "en": "**Axes**"},
    "tms_trio_title": {"fr": "Complémentarité de l'escouade", "en": "Squad complementarity"},
    # ── Teammates helpers ───────────────────────────────────────────────────
    "tmh_waypoint": {"fr": "HaloWaypoint", "en": "HaloWaypoint"},
    "tmh_col_match": "col_match",  # alias → common
    "tmh_col_date": "col_date",  # alias → common
    "tmh_col_map": "col_map",  # alias → common
    "tmh_col_playlist": "col_playlist",  # alias → common
    "tmh_col_mode": "col_mode",  # alias → common
    "tmh_col_result": "col_result",  # alias → common
    "tmh_col_score": "col_score",  # alias → common
    "tmh_col_team_mmr": "col_mmr_team",  # alias → common
    "tmh_col_enemy_mmr": "col_mmr_enemy",  # alias → common
    "tmh_col_delta_mmr": "col_mmr_gap",  # alias → common
    "tmh_link_open": "btn_open",  # alias → common
    "tmh_outcome_win": "outcome_win",  # alias → common
    "tmh_outcome_loss": "outcome_loss",  # alias → common
    "tmh_outcome_draw": "outcome_draw",  # alias → common
    "tmh_outcome_unfinished": "outcome_dnf",  # alias → common
    # ── Timeseries ──────────────────────────────────────────────────────────
    "tm_lifespan_with": {
        "fr": "{player} — Durée de vie (avec {partner})",
        "en": "{player} — Life span (with {partner})",
    },
    # ── Historique des rencontres (v5.4) ─────────────────────────────────────
    # ── Nouveaux graphiques par carte (coéquipiers) ──────────────────────────
    "tm_map_lollipop_title": {
        "fr": "V/D par carte",
        "en": "W/L by map",
    },
    "tm_map_timeline_title": {
        "fr": "Évolution chronologique par carte",
        "en": "Chronological map evolution",
    },
    "tm_map_timeline_caption": {
        "fr": "Chaque cercle = 1 match ensemble. La sélection courante est mise en évidence.",
        "en": "Each dot = 1 match together. Current selection highlighted.",
    },
    "tm_map_bullet_title": {
        "fr": "Taux de victoires vs historique",
        "en": "Session win rate vs history",
    },
    "tm_perf_vs_history_title": {
        "fr": "Performance vs historique",
        "en": "Performance vs history",
    },
    "tm_map_squad_heatmap_title": {
        "fr": "Performance par joueur × carte",
        "en": "Performance by player × map",
    },
}
