"""Chaînes i18n partagées entre pages (navigation, KPIs, radar, citations, analytics)."""

from __future__ import annotations

STRINGS: dict[str, dict[str, str] | str] = {
    "multiplayer_player_label": {
        "fr": "#### 👥 Joueur",
        "en": "#### 👥 Player",
    },
    # ── Page Timeseries — spinners & labels manquants ─────────────────────────
    "citations_halo5_title": {
        "fr": "Citations",
        "en": "Commendations",
    },
    "citations_medals_title": {
        "fr": "Médailles",
        "en": "Medals",
    },
    "citations_medals_caption": {
        "fr": "Médailles sur la sélection/filtres actuels.",
        "en": "Medals for the current selection/filters.",
    },
    "citations_no_medals": {
        "fr": "Aucune médaille trouvée (ou payload médailles absent).",
        "en": "No medals found (or medal payload missing).",
    },
    "citations_medals_distribution": {
        "fr": "Distribution des médailles",
        "en": "Medal distribution",
    },
    "citations_medals_grid": {
        "fr": "Grille de médailles",
        "en": "Medal grid",
    },
    "citations_no_progress": {
        "fr": "Aucune citation n'a progressé dans ce match.",
        "en": "No citation progressed in this match.",
    },
    # ── Match View ────────────────────────────────────────────────────────────
    "page_timeseries": {"fr": "Séries", "en": "Series"},
    "page_session_compare": {"fr": "Sessions", "en": "Sessions"},
    "page_last_match": {"fr": "Dernier match", "en": "Last Match"},
    "page_match": "col_match",  # alias → common
    "page_media": {"fr": "Médias", "en": "Media"},
    "page_citations": {"fr": "Citations", "en": "Commendations"},
    "page_win_loss": {"fr": "Victoires/Défaites", "en": "Wins/Losses"},
    "page_teammates": {"fr": "Escouade", "en": "Squad"},
    "page_match_history": {"fr": "Historique", "en": "History"},
    "page_career": {"fr": "Carrière", "en": "Career"},
    "page_settings": {"fr": "Paramètres", "en": "Settings"},
    "v7_nav_home": {"fr": "Accueil", "en": "Home"},
    "v7_nav_stats": {"fr": "Stats", "en": "Stats"},
    "v7_nav_profile": {"fr": "Profil", "en": "Profile"},
    "v7_section_context_stats": {"fr": "Contexte Stats", "en": "Stats Context"},
    "v7_section_context_squad": {"fr": "Contexte Escouade", "en": "Squad Context"},
    "v7_filters_none": {"fr": "Aucun filtre actif", "en": "No active filters"},
    "v7_filters_reset": {"fr": "Réinitialiser", "en": "Reset"},
    "v7_chip_playlists": {"fr": "Playlist", "en": "Playlist"},
    "v7_chip_modes": {"fr": "Mode", "en": "Mode"},
    "v7_chip_maps": {"fr": "Carte", "en": "Map"},
    "v7_chip_period": {"fr": "Période", "en": "Period"},
    "v7_chip_sessions": {"fr": "Sessions", "en": "Sessions"},
    "v7_chip_scope": {"fr": "Scope", "en": "Scope"},
    "v7_home_quick_actions": {"fr": "Accès rapides", "en": "Quick actions"},
    "v7_home_recent_solo": {"fr": "Dernière session solo", "en": "Latest solo session"},
    "v7_home_recent_squad": {"fr": "Dernière session escouade", "en": "Latest squad session"},
    "v7_home_no_recent_solo": {
        "fr": "Aucune session solo récente disponible.",
        "en": "No recent solo session available.",
    },
    "v7_home_no_recent_squad": {
        "fr": "Aucune session escouade récente disponible.",
        "en": "No recent squad session available.",
    },
    "v7_home_recent_media": {"fr": "Derniers médias liés", "en": "Latest linked media"},
    "v7_home_no_recent_media": {
        "fr": "Aucun média récent associé.",
        "en": "No recent linked media.",
    },
    "v7_home_last_match": {"fr": "Dernier match", "en": "Latest match"},
    # ── KPIs ─────────────────────────────────────────────────────────────────
    "kpi_matches_header": "lbl_parties",  # alias → common
    "kpi_career_header": {"fr": "Carrière", "en": "Career"},
    "kpi_avg_duration": "col_avg_duration",  # alias → common
    "kpi_total_duration": "col_total_duration",  # alias → common
    "kpi_kills_per_match": "col_kills_per_match",  # alias → common
    "kpi_deaths_per_match": "col_deaths_per_match",  # alias → common
    "kpi_assists_per_match": "col_assists_per_match",  # alias → common
    "kpi_kills_per_min": "col_kpm",  # alias → common
    "kpi_deaths_per_min": "col_dpm",  # alias → common
    "kpi_assists_per_min": "col_apm",  # alias → common
    "kpi_avg_accuracy": "col_avg_accuracy",  # alias → common
    "kpi_avg_lifespan": "col_avg_life_long",  # alias → common
    "kpi_win_rate": "col_win_rate",  # alias → common
    "kpi_loss_rate": "col_loss_rate",  # alias → common
    "kpi_ratio": "col_ratio",  # alias → common
    # ── Score de performance ─────────────────────────────────────────────────
    "perf_title": {"fr": "Score de performance", "en": "Performance Score"},
    "perf_short_desc": {"fr": "Relatif à ton historique", "en": "Relative to your history"},
    "perf_insufficient": {"fr": "Historique insuffisant", "en": "Insufficient history"},
    "perf_matches_count": "lbl_total_matches",  # alias → common
    "perf_label_excellent": {"fr": "Excellent", "en": "Excellent"},
    "perf_label_good": {"fr": "Solide", "en": "Solid"},
    "perf_label_average": {"fr": "Correct", "en": "Decent"},
    "perf_label_below": {"fr": "Mauvais", "en": "Bad"},
    "perf_label_bad": {"fr": "Catastrophique", "en": "Catastrophic"},
    "perf_score_exceptional": {"fr": "Excellent", "en": "Excellent"},
    "perf_score_good": {"fr": "Solide", "en": "Solid"},
    "perf_score_normal": {"fr": "Correct", "en": "Decent"},
    "perf_score_below": {"fr": "Mauvais", "en": "Bad"},
    "perf_score_difficult": {"fr": "Catastrophique", "en": "Catastrophic"},
    "perf_interp_excellent": {
        "fr": "Match exceptionnel pour toi",
        "en": "Exceptional match for you",
    },
    "perf_interp_good": {"fr": "Au-dessus de ta moyenne", "en": "Above your average"},
    "perf_interp_average": {"fr": "Performance typique", "en": "Typical performance"},
    "perf_interp_below": {"fr": "En-dessous de ta moyenne", "en": "Below your average"},
    "perf_interp_bad": {"fr": "Match difficile", "en": "Tough match"},
    # ── Match history ───────────────────────────────────────────────────────
    "cit_obtained": {"fr": "Citations obtenues", "en": "Citations earned"},
    "cit_matches_analyzed": {"fr": "Matchs analysés", "en": "Matches analyzed"},
    "cit_distinct_medals": {"fr": "Médailles distinctes", "en": "Distinct medals"},
    "cit_total_medals": {"fr": "Total médailles", "en": "Total medals"},
    "cit_search": {"fr": "Recherche", "en": "Search"},
    "cit_search_placeholder": {
        "fr": "ex: assassin, pilote, multifrag…",
        "en": "e.g. assassin, pilot, multikill…",
    },
    "cit_mastery_master": {"fr": "Maître", "en": "Master"},
    "cit_mastery_level": {"fr": "Niveau {level}", "en": "Level {level}"},
    # ── Last match ──────────────────────────────────────────────────────────
    "cat_btb": {"fr": "Grande bataille en équipe", "en": "Big Team Battle"},
    "cat_ranked": "ranked",  # alias → common
    "cat_firefight": {"fr": "Baptême du feu", "en": "Firefight"},
    "cat_other": {"fr": "Autre", "en": "Other"},
    # ── Win/Loss ────────────────────────────────────────────────────────────
    "radar_stats_per_min": {"fr": "Stats par minute", "en": "Stats per minute"},
    "radar_perf_profile": {"fr": "Profil de performance", "en": "Performance profile"},
    "radar_kpm": {"fr": "Frags/min", "en": "Kills/min"},
    "radar_dpm": {"fr": "Morts/min", "en": "Deaths/min"},
    "radar_apm": {"fr": "Assistances/min", "en": "Assists/min"},
    "radar_objective": {"fr": "Objectif", "en": "Objective"},
    "radar_survival": {"fr": "Survie", "en": "Survival"},
    "radar_participation_profile": {"fr": "Profil de participation", "en": "Participation profile"},
    "radar_objectives": {"fr": "Objectifs", "en": "Objectives"},
    "radar_combat": {"fr": "Combat", "en": "Combat"},
    "radar_support": {"fr": "Support", "en": "Support"},
    "radar_impact": {"fr": "Impact", "en": "Impact"},
    "radar_complementarity": {"fr": "Complémentarité", "en": "Complementarity"},
    "radar_kills_label": "col_kills",  # alias → common
    "radar_assists_label": "col_assists",  # alias → common
    "radar_kills_pct": {"fr": "Frags %", "en": "Kills %"},
    "radar_assists_pct": {"fr": "Assistances %", "en": "Assists %"},
    "radar_obj_pct": {"fr": "Objectifs %", "en": "Objectives %"},
    "radar_kd": {"fr": "F/M", "en": "K/D"},
    "radar_session_evolution": {"fr": "Évolution du profil", "en": "Profile evolution"},
    "radar_win_rate": {"fr": "Taux victoires", "en": "Win rate"},
    "radar_avg_score": {"fr": "Score moy.", "en": "Avg score"},
    "radar_hover_deaths": {"fr": "morts", "en": "deaths"},
    "radar_hover_no_penalty": {"fr": "Aucune", "en": "None"},
    "radar_hover_survival_pct": {"fr": "survie", "en": "survival"},
    "radar_hover_pts_per_min": {"fr": "{pts:.1f} pts/min", "en": "{pts:.1f} pts/min"},
    # ── Radar — descriptions des axes ───────────────────────────────────────
    "radar_desc_objectives": {
        "fr": "**Objectifs** : contribution à la victoire (objectifs ou frags selon le mode)",
        "en": "**Objectives** : contribution to victory (objectives or kills depending on mode)",
    },
    "radar_desc_combat": {
        "fr": "**Combat** : éliminations directes",
        "en": "**Combat** : direct eliminations",
    },
    "radar_desc_support": {"fr": "**Support** : assists", "en": "**Support** : assists"},
    "radar_desc_score": {"fr": "**Score** : points totaux", "en": "**Score** : total points"},
    "radar_desc_impact": {
        "fr": "**Impact** : intensité (pts/min)",
        "en": "**Impact** : intensity (pts/min)",
    },
    "radar_desc_survival": {
        "fr": "**Survie** : moins de morts + durée de vie moyenne",
        "en": "**Survival** : fewer deaths + average lifespan",
    },
    # ── Career progress circle ──────────────────────────────────────────────
    # (uses career_max_rank, career_hero_rank from career section above)
    # ── Checkbox filter ─────────────────────────────────────────────────────
    "cbf_suffix_all": {"fr": "(tous)", "en": "(all)"},
    "cbf_suffix_none": "sel_none",  # alias → common
    "cbf_confirm_clear": {
        "fr": "⚠️ Confirmer : vider toutes les sélections ?",
        "en": "⚠️ Confirm: clear all selections?",
    },
    # ── DuckDB analytics ────────────────────────────────────────────────────
    "dba_global_stats": {"fr": "📊 Stats globales (DuckDB)", "en": "📊 Global stats (DuckDB)"},
    "dba_win_rate": {"fr": "Taux victoires", "en": "Win rate"},
    "flt_date_start": {"fr": "Début", "en": "Start"},
    "flt_date_end": {"fr": "Fin", "en": "End"},
    "flt_session_all": "sel_all",  # alias → common
    "flt_session_last": {"fr": "Dernière session", "en": "Last session"},
    "flt_session_prev": {"fr": "Session précédente", "en": "Previous session"},
    "flt_session_trio": {"fr": "Dernière session en trio", "en": "Last trio session"},
    "flt_trio_caption": {"fr": "Trio : {label}", "en": "Trio: {label}"},
    "flt_session_select": {"fr": "Session", "en": "Session"},
    "hlp_db_empty_warning": {
        "fr": "La base sélectionnée est vide (0 octet). Basculement automatique vers une DB valide si possible.",
        "en": "The selected database is empty (0 bytes). Auto-switching to a valid DB if possible.",
    },
    "hlp_db_fallback_info": {"fr": "DB utilisée : {path}", "en": "DB used: {path}"},
    "hlp_btn_sync": {"fr": "🔄 Synchroniser", "en": "🔄 Sync"},
    "hlp_btn_sync_help": {
        "fr": "Synchronise tous les joueurs (nouveaux matchs, highlights, aliases).",
        "en": "Sync all players (new matches, highlights, aliases).",
    },
    "hlp_configure_db": {
        "fr": "Configure une DB et un joueur dans Paramètres.",
        "en": "Configure a DB and a player in Settings.",
    },
    "dba_avg_kda": {"fr": "FDA moyen", "en": "Avg KDA"},
    "dba_kda_evolution": {
        "fr": "📈 Évolution KDA (moyenne mobile {n} matchs)",
        "en": "📈 KDA evolution (rolling avg {n} matches)",
    },
    "dba_kda_recent": {"fr": "KDA (derniers {n})", "en": "KDA (last {n})"},
    "dba_kda_older": {"fr": "KDA (plus anciens {n})", "en": "KDA (older {n})"},
    "dba_trend": {"fr": "Tendance", "en": "Trend"},
    "dba_map_perf": {"fr": "🗺️ Performances par carte (DuckDB)", "en": "🗺️ Map performance (DuckDB)"},
    "dba_advanced": {
        "fr": "📊 Analytics avancées (DuckDB)",
        "en": "📊 Advanced analytics (DuckDB)",
    },
    "dba_advanced_help": {
        "fr": "Ces statistiques sont calculées via DuckDB directement.",
        "en": "These stats are calculated directly via DuckDB.",
    },
    "dba_no_parquet": {
        "fr": "Aucune donnée Parquet disponible.",
        "en": "No Parquet data available.",
    },
    # ── App helpers ─────────────────────────────────────────────────────────
    "app_db_used": {"fr": "DB utilisée : {path}", "en": "DB used: {path}"},
    "app_sync_all_help": {
        "fr": "Synchronise tous les joueurs configurés.",
        "en": "Sync all configured players.",
    },
    "app_no_match": "no_match_found",  # alias → common
    "app_configure_db": {
        "fr": "Configure une DB et un joueur dans Paramètres.",
        "en": "Configure a DB and a player in Settings.",
    },
    "app_asset_error": {
        "fr": "Asset '{prefix}' non téléchargé : {err}",
        "en": "Asset '{prefix}' not downloaded: {err}",
    },
    "app_player_default": {"fr": "(joueur)", "en": "(player)"},
    "app_profile_api_err": {
        "fr": "Profil auto (SPNKr) : {err}",
        "en": "Profile auto (SPNKr): {err}",
    },
    # ── Objective analysis page (Phase 3) ───────────────────────────────────
}
