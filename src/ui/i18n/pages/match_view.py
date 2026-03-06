"""Chaînes i18n — page Match View (scoreboard, dynamics, roster, charts)."""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
    "mv_citations": {
        "fr": "Citations",
        "en": "Commendations",
    },
    "mv_medals": {
        "fr": "Médailles",
        "en": "Medals",
    },
    "mv_lusr_no_data": {
        "fr": "Aucun rating LUSR/CSR calculé pour ce match. Lancez `--lusr` (non classé) ou `--csr` (classé) pour calculer.",
        "en": "No LUSR/CSR rating computed for this match. Run `--lusr` (unranked) or `--csr` (ranked) to compute it.",
    },
    "mv_lusr_modules_missing": {
        "fr": "Modules LUSR non disponibles.",
        "en": "LUSR modules are not available.",
    },
    # ── Page Médias ──────────────────────────────────────────────────────────
    "mv_loading": {
        "fr": "Lecture des stats détaillées (attendu vs réel, médailles)…",
        "en": "Loading detailed stats (expected vs actual, medals)…",
    },
    "mv_playlist_group_caption": {
        "fr": "Groupe : {group}",
        "en": "Group: {group}",
    },
    # Nouveau format "LUSR Arena : 1528" — inclut type, groupe traduit et valeur
    "mv_playlist_group_rating_label": {
        "fr": "{type} {group} : {value}",
        "en": "{type} {group}: {value}",
    },
    # Noms traduits des groupes de playlist
    "mv_pg_ranked": "ranked",  # alias → common
    "mv_pg_arena": {"fr": "Arena", "en": "Arena"},
    "mv_pg_btb": {"fr": "Grand Combat", "en": "Big Team Battle"},
    "mv_pg_tactical": {"fr": "Tactique", "en": "Tactical"},
    "mv_pg_social": {"fr": "Social", "en": "Social"},
    "mv_pg_fun": {"fr": "Fun", "en": "Fun"},
    # Barre de progression dans le sous-tier
    "mv_sub_tier_progress": {
        "fr": "Progression dans {tier_name} {sub_tier} : {pts_earned} / {tier_size} pts",
        "en": "Progress in {tier_name} {sub_tier}: {pts_earned} / {tier_size} pts",
    },
    "mv_citations_unavailable": {
        "fr": "Référentiel Citations indisponible.",
        "en": "Citations reference unavailable.",
    },
    "mv_citations_no_data": {
        "fr": "Citations indisponibles pour ce match.",
        "en": "No citations available for this match.",
    },
    # ── Page Médias — manquants ───────────────────────────────────────────────
    "mv_match_dynamics": {
        "fr": "Dynamique du match",
        "en": "Match dynamics",
    },
    "mv_dynamics_computing": {
        "fr": "Analyse de la dynamique…",
        "en": "Analyzing dynamics…",
    },
    "mv_dynamics_no_data": {
        "fr": "Données insuffisantes pour afficher la dynamique du match.",
        "en": "Not enough data to display match dynamics.",
    },
    "mv_dynamics_no_roster": {
        "fr": "Roster introuvable — frise de dominance indisponible.",
        "en": "Roster not found — dominance timeline unavailable.",
    },
    "mv_dynamics_no_team": {
        "fr": "Équipe introuvable pour ce joueur — frise de dominance indisponible.",
        "en": "Team not found for this player — dominance timeline unavailable.",
    },
    "mv_dynamics_no_kills": {
        "fr": "Aucun kill enregistré pour ce match.",
        "en": "No kills recorded for this match.",
    },
    "mv_dynamics_no_dominance": {
        "fr": "Données insuffisantes pour la frise de dominance.",
        "en": "Not enough data for the dominance timeline.",
    },
    "mv_antagonists_title": {
        "fr": "Antagonistes du match",
        "en": "Match antagonists",
    },
    "mv_highlight_loading": {
        "fr": "Chargement des highlight events (film)…",
        "en": "Loading highlight events (film)…",
    },
    "mv_nemesis_no_data": {
        "fr": "Impossible de déterminer Némésis/Souffre-douleur (timeline insuffisante).",
        "en": "Unable to determine Nemesis/Punching Bag (insufficient timeline).",
    },
    "mv_interactions_no_data": {
        "fr": "Données insuffisantes pour les interactions eliminateur-victime.",
        "en": "Not enough data for killer-victim interactions.",
    },
    "mv_kills_over_time": {
        "fr": "Score Frags-Décès au fil du match",
        "en": "K/D Score over the match",
    },
    "mv_scoreboard": {
        "fr": "Tableau des scores",
        "en": "Scoreboard",
    },
    "mv_scoreboard_no_data": {
        "fr": "Statistiques des joueurs indisponibles pour ce match.",
        "en": "Player stats unavailable for this match.",
    },
    "mv_players_title": {
        "fr": "Joueurs",
        "en": "Players",
    },
    "mv_impact_title": {
        "fr": "Impact du match",
        "en": "Match impact",
    },
    "mv_impact_no_events": {
        "fr": "Données d'impact indisponibles (highlight events manquants).",
        "en": "Impact data unavailable (missing highlight events).",
    },
    "mv_impact_computing": {
        "fr": "Analyse de la timeline…",
        "en": "Analyzing timeline…",
    },
    "mv_impact_no_events_match": {
        "fr": "Aucun événement enregistré pour ce match.",
        "en": "No events recorded for this match.",
    },
    "mv_impact_too_few": {
        "fr": "Pas assez de données pour afficher la timeline.",
        "en": "Not enough data to display the timeline.",
    },
    "mv_medals_no_data": {
        "fr": "Médailles indisponibles pour ce match (ou aucune médaille).",
        "en": "Medals unavailable for this match (or no medals).",
    },
    "mv_match_id_missing": {
        "fr": "MatchId manquant.",
        "en": "MatchId missing.",
    },
    # ── Teammates ─────────────────────────────────────────────────────────────
    "mv_vs_expected": {
        "fr": "Réel vs attendu",
        "en": "Actual vs expected",
    },
    "mv_media_title": {
        "fr": "Médias",
        "en": "Media",
    },
    "mv_videos_title": {
        "fr": "Vidéos",
        "en": "Videos",
    },
    # ── Coéquipiers — titres graphiques ──────────────────────────────────────
    "mvp_participation_title": {"fr": "Participation au match", "en": "Match Participation"},
    "mvp_comparison_title": {
        "fr": "Comparaison de participation",
        "en": "Participation Comparison",
    },
    "mvp_axes_label": {"fr": "Axes", "en": "Axes"},
    # ── Session Compare ────────────────────────────────────────────────────────
    "mh_title": {"fr": "Historique des parties", "en": "Match history"},
    # ── Match view ──────────────────────────────────────────────────────────
    "mv_no_rating": {
        "fr": "Aucun rating LUSR/CSR calculé pour ce match. Lance un backfill rating pour la playlist concernée.",
        "en": "No LUSR/CSR rating calculated for this match. Run a rating backfill for the relevant playlist.",
    },
    "mv_rating_pending": {
        "fr": "Rating non encore calculé",
        "en": "Rating not yet calculated",
    },
    "mv_rating_pending_hint": {
        "fr": "LUSR calculé automatiquement au prochain sync",
        "en": "LUSR will be computed automatically on next sync",
    },
    "mv_performance": "col_performance",  # alias → common
    "mv_results": {"fr": "Résultats", "en": "Results"},
    "mv_relative_history": {"fr": "Relatif à ton historique", "en": "Relative to your history"},
    "mv_insufficient_history": {"fr": "Historique insuffisant", "en": "Insufficient history"},
    "mv_bot_teammate_note": {
        "fr": "Équipe incomplète (+5 pts)",
        "en": "Incomplete team (+5 pts)",
    },
    "mv_bot_teammate_win_note": {
        "fr": "Victoire malgré équipe incomplète",
        "en": "Won despite incomplete team",
    },
    "mv_lusr_bot_loss": {
        "fr": "Défaite avec équipe incomplète (non représentative)",
        "en": "Loss with incomplete team (unrepresentative)",
    },
    "mv_lusr_bot_win": {
        "fr": "Victoire malgré équipe incomplète",
        "en": "Won despite incomplete team",
    },
    "mv_thumbnail_unavailable": {
        "fr": "Miniature de carte indisponible.",
        "en": "Map thumbnail unavailable.",
    },
    "mv_stats_unavailable": {
        "fr": "Stats détaillées indisponibles pour ce match. Essaie un backfill.",
        "en": "Detailed stats unavailable for this match. Try a backfill.",
    },
    "mv_medal_fallback": {"fr": "Médaille #{n}", "en": "Medal #{n}"},
    "mv_open_waypoint": {"fr": "Ouvrir sur HaloWaypoint", "en": "Open on HaloWaypoint"},
    "mv_killer_victim_title": {"fr": "Éliminateur-Victime", "en": "Killer-Victim"},
    "mv_events_unavailable": {
        "fr": "Indisponible : la DB ne contient pas les highlight events pour ce match.",
        "en": "Unavailable: the DB does not contain highlight events for this match.",
    },
    "mv_scoreboard_avg_life": {"fr": "Durée de vie moy.", "en": "Avg life span"},
    "mv_dominance_legend": {
        "fr": "- **Barres ** : % des frags par tranche de 30 s — bleu = mon équipe, orange = adversaires\n- **Chiffres encadrés** : score cumulé de chaque équipe à l'instant T (encadré si en tête)\n- **Points ** : chaque point = un kill individuel\n- **Lignes reliées** : série d'un même joueur (≥ 3 kills consécutifs sans mourir)",
        "en": "- **Bars**: % of kills per 30s interval — blue = my team, orange = opponents\n- **Framed numbers**: cumulative team score at that point (framed if leading)\n- **Dots**: each dot = an individual kill\n- **Connected lines**: streak by a single player (≥ 3 consecutive kills without dying)",
    },
    "mv_nemesis_unavailable": {
        "fr": "Indisponible : la DB ne contient pas les highlight events. Si tu utilises une DB SPNKr, relance l'import avec `--with-highlight-events`.",
        "en": "Unavailable: the DB does not contain highlight events. If you are using a SPNKr DB, re-run the import with `--with-highlight-events`.",
    },
    "mv_deaths_count": {"fr": "{prefix}{n} morts", "en": "{prefix}{n} deaths"},
    "mv_killed_count": {"fr": "{prefix}Tué {n} fois", "en": "{prefix}Killed {n} times"},
    "mv_team_n": {"fr": "Équipe {n}", "en": "Team {n}"},
    "mv_team_unknown": {"fr": "Équipe inconnue", "en": "Unknown team"},
    "mv_team_label": {"fr": "Équipe {name}", "en": "Team {name}"},
    "mv_scoreboard_rank_note": {
        "fr": "ℹ️ Le rang est individuel au sein de chaque équipe — plusieurs joueurs peuvent partager le même rang (ex. : tous les membres d'une équipe vaincue).",
        "en": "ℹ️ Rank is individual within each team — multiple players can share the same rank (e.g., all members of a defeated team).",
    },
    "mv_roster_unavailable": {
        "fr": "Roster indisponible pour ce match (payload MatchStats manquant ou équipe introuvable).",
        "en": "Roster unavailable for this match (MatchStats payload missing or team not found).",
    },
    "mv_roster_my_team": {"fr": "Mon équipe — {name} ({n})", "en": "My team — {name} ({n})"},
    "mv_roster_enemy_team": {
        "fr": "Équipe adverse — {name} ({n})",
        "en": "Enemy team — {name} ({n})",
    },
    "mv_roster_opponents": {"fr": "Adversaires", "en": "Opponents"},
    # ── Match view charts ───────────────────────────────────────────────────
    "mvc_mmr_team": {"fr": "MMR d'équipe", "en": "Team MMR"},
    "mvc_mmr_enemy": "col_mmr_enemy",  # alias → common
    "mvc_mmr_gap": "col_mmr_gap",  # alias → common
    "mvc_actual_only": {
        "fr": "Valeur réelle (comparaison indisponible)",
        "en": "Actual value (comparison unavailable)",
    },
    "mvc_hist_avg": {
        "fr": "Moyenne hist. {category} ({n} matchs)",
        "en": "Hist. avg {category} ({n} matches)",
    },
    "mvc_ratio_avg": {"fr": "Ratio moy. {category}", "en": "Avg ratio {category}"},
    "mvc_fda_title": {"fr": "F / D / A", "en": "K / D / A"},
    "mvc_ratio_title": "col_ratio",  # alias → common
    "mvc_this_match": {"fr": "Ce match", "en": "This match"},
    "mvc_mmr_gap_sub": {"fr": "équipe - adverse", "en": "team - enemy"},
    "mvc_fda_ratio": {"fr": "Ratio F/D/A", "en": "K/D/A Ratio"},
    "mvc_lbl_k": {"fr": "F", "en": "K"},
    "mvc_lbl_d": {"fr": "D", "en": "D"},
    "mvc_lbl_a": {"fr": "A", "en": "A"},
    # ── Career ──────────────────────────────────────────────────────────────
    "mvh_my_captures": {"fr": "Mes captures", "en": "My captures"},
    "mvh_captures_of": {"fr": "Captures de {gt}", "en": "Captures by {gt}"},
    # ── Match view participation ────────────────────────────────────────────
    "mvp_this_match": {"fr": "Ce match", "en": "This match"},
    # ── Radar chart ─────────────────────────────────────────────────────────
    "mvh_window_label": {
        "fr": "Fenêtre: {start} → {end}",
        "en": "Window: {start} → {end}",
    },
    "mvh_duration_estimated": {
        "fr": " *(durée estimée)*",
        "en": " *(estimated duration)*",
    },
    # ── Match view players (Phase 3) ─────────────────────────────────────────
    "mvp_attribution_deaths": {
        "fr": "Morts attribuées {assigned}/{total} (certain {certain}, estimé {estimated}, manquantes {missing})",
        "en": "Deaths attributed {assigned}/{total} (certain {certain}, estimated {estimated}, missing {missing})",
    },
    "mvp_attribution_kills": {
        "fr": "Kills attribués {assigned}/{total} (certain {certain}, estimé {estimated}, manquants {missing})",
        "en": "Kills attributed {assigned}/{total} (certain {certain}, estimated {estimated}, missing {missing})",
    },
    # ── Teammates charts (Phase 3) ───────────────────────────────────────────
    "mv_encounter_history": {
        "fr": "Historique des rencontres",
        "en": "Encounter History",
    },
    "col_role": {"fr": "Rôle", "en": "Side"},
    "col_encounters": {"fr": "Rencontres", "en": "Encounters"},
    "col_wr_ally": {"fr": "% Vict. allié", "en": "Win% (ally)"},
    "col_wr_enemy": {"fr": "% Vict. ennemi", "en": "Win% (enemy)"},
    "col_kd_cross": {"fr": "Frags/Morts", "en": "Kills/Deaths"},
    "col_last_seen": {"fr": "Dernier match", "en": "Last match"},
    "mv_encounter_legend": {
        "fr": (
            "**% Vict. allié / ennemi** — ton taux de victoires sur les matchs joués avec ou contre ce joueur · "
            "**Frags/Morts** — nombre de fois où tu l'as éliminé / où il t'a éliminé dans vos matchs communs · "
            "**Dernier match** — date du dernier match partagé"
        ),
        "en": (
            "**Win% (ally/enemy)** — your win rate on shared matches, by team side · "
            "**Kills/Deaths** — how many times you eliminated them / they eliminated you across shared matches · "
            "**Last match** — date of the last shared match"
        ),
    },
    "legend_badge_tough_nut": {
        "fr": "Ce joueur te tue au moins 2× plus que tu ne le tues (min. 3 morts)",
        "en": "This player kills you at least 2× more than you kill them (min. 3 deaths)",
    },
    "legend_badge_ally_plus": {
        "fr": "Tu gagnes souvent avec ce joueur dans ton équipe (≥ 65% de victoires, min. 2 matchs)",
        "en": "You win often with this player on your team (≥ 65% win rate, min. 2 games)",
    },
    "legend_badge_coriace": {
        "fr": "Tu as du mal à battre ce joueur quand il est en face (≤ 35% de victoires, min. 3 duels)",
        "en": "You struggle to beat this player as an opponent (≤ 35% win rate, min. 3 duels)",
    },
    # Ordinal de rencontre — "{ordinal} rencontre" / "{ordinal} encounter"
    "encounter_ordinal": {"fr": "{ordinal} rencontre", "en": "{ordinal} encounter"},
    # Dates relatives (tableau encounter history)
    "rel_date_upcoming": {"fr": "à venir", "en": "upcoming"},
    "rel_date_today": {"fr": "aujourd'hui", "en": "today"},
    "rel_date_yesterday": {"fr": "hier", "en": "yesterday"},
    "rel_date_days_ago": {"fr": "il y a {days} j", "en": "{days}d ago"},
    "rel_date_weeks_ago": {"fr": "il y a {weeks} sem.", "en": "{weeks}w ago"},
    "rel_date_months_ago": {"fr": "il y a {months} mois", "en": "{months}mo ago"},
    "rel_date_years_ago": {"fr": "il y a {years} an", "en": "{years}y ago"},
    "rel_date_years_ago_pl": {"fr": "il y a {years} ans", "en": "{years}y ago"},
    "badge_tough_nut": {"fr": "Dur à cuire", "en": "Tough Nut"},
    "badge_ally_plus": {"fr": "Allié+", "en": "Ally+"},
    "badge_coriace": {"fr": "Coriace", "en": "Tough Guy"},
    "role_ally": {"fr": "Allié", "en": "Ally"},
    "role_enemy": {"fr": "Ennemi", "en": "Enemy"},
    "tooltip_tough_nut_kd": {
        "fr": "Il t'a tué {killed}× pour {kills} kill de ta part",
        "en": "Killed you {killed}× for only {kills} kill back",
    },
    "tooltip_tough_nut_zero": {
        "fr": "Il t'a tué {killed}× sans riposte",
        "en": "Killed you {killed}× with no retaliation",
    },
    "tooltip_ally_plus": {
        "fr": "{pct}% de victoires ensemble ({count} matchs)",
        "en": "{pct}% win rate together ({count} games)",
    },
    "tooltip_coriace": {
        "fr": "Seulement {pct}% de victoires face à lui ({count} duels)",
        "en": "Only {pct}% win rate against them ({count} duels)",
    },
    # Onglets de la vue détaillée (P2)
    "mv_tab_summary": {"fr": "📊 Résumé", "en": "📊 Summary"},
    "mv_tab_combat": {"fr": "⚔️ Combat", "en": "⚔️ Combat"},
    "mv_tab_team": {"fr": "👥 Équipe", "en": "👥 Team"},
    "mv_tab_citations_medals": {"fr": "🏅 Citations & Médailles", "en": "🏅 Citations & Medals"},
    "mv_tab_media": {"fr": "🎬 Médias", "en": "🎬 Media"},
}
