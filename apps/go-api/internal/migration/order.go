package migration

import "sort"

// order.go — ordre d'exécution EXPLICITE des migrations (Phase 1.5.0,
// title-agnostic ADR 0025). AVANT : l'ordre dépendait de l'ordre des init()
// (alphabétique par nom de fichier) + ordre des Register() — donc déplacer un
// steps_*.go pouvait réordonner les migrations et casser le boot (une migration
// avant sa dépendance). MAINTENANT : l'ordre est défini par canonicalOrder ;
// RunForDB trie dessus. Déplacer/renommer un fichier ne change plus rien.
//
// canonicalOrder a été généré depuis l'ordre d'enregistrement courant
// (2026-06-02) — la bascule est un no-op (cf. order_test.go). Toute NOUVELLE
// migration doit être ajoutée à cette liste (order_test.go échoue sinon).
var canonicalOrder = []string{
	"add_match_intensity_to_match_registry",                 // shared
	"add_asset_translations",                                // metadata
	"add_battlepass_asset_refs",                             // metadata
	"add_battlepass_metadata",                               // metadata
	"add_challenge_metadata",                                // metadata
	"add_medal_definitions",                                 // metadata
	"add_weapon_labels",                                     // metadata
	"add_weapon_registry",                                   // metadata
	"drop_legacy_translation_tables",                        // metadata
	"add_waypoint_assets_raw",                               // metadata
	"add_map_images_registry",                               // metadata
	"add_mode_name_tr",                                      // metadata
	"add_citation_mappings",                                 // metadata
	"add_citation_mappings_pk",                              // metadata
	"add_citation_mappings_v2_fields",                       // metadata
	"add_citation_name_display_en",                          // metadata
	"add_citation_description_en",                           // metadata
	"add_xbox_achievement_definitions",                      // metadata
	"add_career_rank_translations",                          // metadata
	"medal_definitions_add_indices",                         // metadata
	"enrich_medal_definitions_v2",                           // metadata
	"medal_definitions_add_personal_score",                  // metadata
	"seed_custom_vengeur_medal",                             // metadata
	"fix_meganaut_fr_description",                           // metadata
	"fix_super_fiesta_fr_label",                             // metadata
	"seed_playlist_fr_translations",                         // metadata
	"add_title_id_to_xbox_achievement_definitions",          // metadata
	"cleanup_xbox_achievement_definitions_unknown_title",    // metadata
	"add_xbox_title_id_to_xbox_achievement_definitions",     // metadata
	"add_service_config_id_to_xbox_achievement_definitions", // metadata
	"add_assists_model_coefs",                               // metadata
	"add_catalog_playlists",                                 // metadata
	"fix_citation_image_paths_double_encoded",               // metadata
	"normalize_citation_mappings_category_keys",             // metadata (2026-08-04 : libellés FR -> clés canoniques)
	"add_csr_placement_thresholds",                          // metadata
	"drop_metadata_art_surface_indexes_v1",                  // metadata
	"drop_metadata_art_surface_indexes_v2",                  // metadata
	"drop_metadata_art_surface_indexes_v3",                  // metadata
	"drop_metadata_art_surface_indexes_v4",                  // metadata
	"drop_playlists_catalog_secondary_indexes",              // metadata
	"create_milestone_catalog_metadata",                     // metadata
	"create_prestige_metadata_schema",                       // metadata
	"purge_weapons_name_fr_column",                          // metadata (V721-05.1 : rebuild CTAS-swap, doit suivre add_weapon_registry — cf. order_dependency_test.go)
	"rebuild_catalog_fetch_queue_drop_art_indexes",          // metadata
	"seed_ranked_playlists_catalog",                         // metadata
	"challenge_template_add_source_column",                  // metadata
	"add_template_tagging_columns",                          // metadata
	// Baseline squashée v1 (chantier N4, plan PLAN_MIGRATION_SQUASH_BASELINE_2026-07) :
	// remplace les 33 steps title-owned contigus create_base_player_schema..
	// player_append_only_csr_snapshots_v1 (bornes M3a). Cf. steps_player_baseline.go.
	"create_baseline_player_v1", // player
	// ALTER post-baseline additif : challenge_snapshots.locale (mirror du squashé
	// add_challenge_snapshots_display_path, désormais dans la baseline). Doit suivre
	// create_baseline_player_v1 (créateur de la table). Cf. steps_player_base.go.
	"add_challenge_snapshots_locale",                 // player
	"player_append_only_match_citations_v1",          // player
	"player_append_only_match_enrichment_v1",         // player
	"player_append_only_match_skill_rank_v1",         // player
	"msr_written_at_default_now_repair_v1",           // player
	"player_append_only_personal_score_awards_v1",    // player
	"create_streak_history_append_only",              // player
	"add_player_assists_model",                       // player
	"create_coach_proposal_player_schema",            // player
	"dedup_record_history_v1",                        // player
	"drop_challenge_mutated_art_indexes_v1",          // player
	"drop_coach_proposal_status_art_index_v1",        // player
	"drop_engagement_coefficients_xuid_art_index_v1", // player
	"fix_career_xp_total_default_zero",               // player
	"lusr_chain_rework_v1",                           // player
	"create_lusr_component_history",                  // player
	"player_append_only_lusr_component_history_v1",   // player
	"player_msr_view_lusr_over_v2_v1",                // player
	"player_msr_view_priority_csr_v1",                // player
	"create_notifications_in_shared_social",          // shared_social
	"drop_notifications_from_player_db",              // player
	"drop_idx_pn_xuid_unread",                        // shared_social
	"player_match_enrichment_performance_chain_v1",   // player
	"create_prestige_player_schema",                  // player
	// arc_titles : structure retirée le 2026-07-25 (décision produit 2026-07-18 — arcs,
	// défis et points de progression strictement indépendants par titre). L'entrée
	// historique create_arc_titles_join RESTE ici À DESSEIN : c'est un enregistrement
	// de schema_migrations présent sur toutes les bases joueur existantes, pas du code
	// mort exécutable — la retirer désynchroniserait le registre et ferait échouer
	// l'audit bidirectionnel (order_audit_test.go : « canonicalOrder référence X absent
	// de (global + title) »). NE PAS la supprimer. drop_arc_titles DOIT la suivre : sur
	// une base vierge le créateur tourne avant le dropper (cf. order_dependency_test.go).
	"create_arc_titles_join",                           // player (historique — table retirée par drop_arc_titles)
	"drop_arc_titles",                                  // player (V721-09 : retrait de la capacité multi-titres d'arc)
	"create_improvement_campaign_schema",               // player
	"create_progression_player_schema",                 // player
	"repair_rec_hist_achieved_index_canonical_v1",      // player (F4 : index record_history canonique)
	"prestige_add_source_columns_v1",                   // player (ADR 0020 : source challenge + telemetry)
	"player_match_enrichment_psa_checked_v1",           // player
	"rebuild_career_progression_defeat_art_corruption", // player
	"repair_player_match_enrichment_primary_key",       // player
	"repair_match_citations_primary_key",               // player
	"create_personal_score_awards_player_v1",           // player (autorité de schéma unique 2026-08-05 : table ex-Ensure-only)
	"create_player_csr_snapshots_player_v1",            // player (idem — cf. steps_player_schema_authority.go)
	"drop_career_xuid_art_index_v1",                    // player (DERNIER du bloc : doit suivre tout créateur de l'index, dont la baseline)
	"drop_psa_xuid_art_index_v1",                       // player (idem — miroir d'idx_career_xuid ; suit create_personal_score_awards_player_v1)
	"drop_psa_match_xuid_art_index_v1",                 // player (idem — préfixe redondant d'idx_psa_gen, posé par le seul PostSwap legacy)
	"create_base_shared_schema",                        // shared
	"add_film_match_start",                             // shared
	"add_highlight_events_autoincrement",               // shared
	"add_match_participants_columns",                   // shared
	"add_h5_kill_mechanics_columns",                    // shared (Halo 5 kill mechanics)
	"add_medals_bigint",                                // shared
	"add_mv_player_matches_fr_cols",                    // shared
	"add_mv_player_matches_view",                       // shared
	"add_shared_performance_indexes",                   // shared
	"add_playable_duration",                            // shared
	"add_playlist_fr_name_fallback",                    // shared
	"add_spnkr_version",                                // shared
	"add_team_ps_scores",                               // shared
	"add_weapon_kills",                                 // shared
	"add_weapon_kills_reconciled_as",                   // shared
	"drop_highlight_events_gamertag",                   // shared
	"fix_bot_xuid",                                     // shared
	"fix_bot_gamertags",                                // shared
	"fix_events_loaded_inconsistency",                  // shared
	"fix_mv_player_matches_scores",                     // shared
	"add_mv_player_matches_pair_id",                    // shared
	"migrate_weapon_kills_to_ubigint",                  // shared
	// L'éradication ART append-only de weapon_kills DOIT suivre la migration UBIGINT
	// (donc la création/ALTER de weapon_kills) — sinon sur DB FRAÎCHE le rebuild
	// no-ope (table absente). Name-keyed → no-op sur DB déjà migrées.
	"shared_append_only_weapon_kills_v1", // shared
	// Capture de la mecanique de kill Halo 5 (kill_kind) : ALTER weapon_kills +
	// recreation v_weapon_kills. DOIT suivre shared_append_only_weapon_kills_v1
	// (vue generationnelle + generation_id deja crees). Phase 1 (capture seule) ;
	// backfill + exploitation donut = Phase 2. Fichier steps_shared_h5_* → l'ordre
	// init (alphabetique) le place entre append_only_weapon_kills et rebuild_match_
	// participants, coherent avec cette position (TestSortByCanonicalIsNoOp).
	"shared_h5_weapon_kill_kind_v1",                   // shared
	"add_media_likes",                                 // shared
	"drop_media_likes_from_shared",                    // shared
	"add_match_registry_i18n_columns",                 // shared
	"add_match_registry_version_ids",                  // shared
	"add_start_time_utc_to_match_registry",            // shared
	"add_perf_indexes_shared",                         // shared
	"fix_mv_player_matches_i18n_cols",                 // shared
	"add_mv_player_matches_utc_cols",                  // shared
	"fix_start_time_utc_via_session_tz",               // shared
	"drop_assists_expected_halo_infinite",             // shared
	"upgrade_v_gamertag_lookup_bots_and_raw_fallback", // shared
	"repair_v_gamertag_lookup_bots_2026_05_16",        // shared
	"repair_v_gamertag_lookup_bots_2026_05_30",        // shared
	"shared_add_participation_info_booleans",          // shared
	"shared_add_participation_timestamps",             // shared
	"shared_add_t0_quality",                           // shared
	"shared_backfill_is_ranked_and_season",            // shared
	"shared_create_player_squad_offset",               // shared
	"add_shared_match_csrs",                           // shared
	// Fix fresh-provision (Halo 5 = 1er titre fraîchement sync, 2026-06-20) : la
	// conversion append-only de match_csrs DOIT suivre la création de la table
	// (add_shared_match_csrs) — sinon sur DB FRAÎCHE le rebuild no-ope (table absente)
	// et match_csrs reste en created_at, cassant l'index written_at d'EnsureSharedSchema
	// au 1er OpenSharedDB. Name-keyed → no-op sur DB déjà migrées (Infinite intact).
	// Même pattern documenté que le reorder skill_v2 ci-dessous.
	"shared_append_only_match_csrs_v1", // shared
	"add_pve_schema",                   // shared_pve
	// J4 : la position suit l'ordre d'enregistrement réel (init() par nom de
	// fichier — steps_shared_kill_events.go précède steps_shared_objective_events.go),
	// exigence de TestSortByCanonicalIsNoOpOnCurrentRegistry.
	"shared_match_kill_events_v1", // shared (1 ligne par mort, crédit kill-feed + source du dégât, append-only + vue _latest)
	// Inversion de préséance (2026-08-03) : base crédit + enrichissement film + orphelins.
	// Placée AVANT from_pairs parce que l'ordre d'init() suit le nom de fichier — from_pairs
	// ne trouve alors plus rien à reprendre, et c'est correct : la reprise credit-base couvre
	// tout ce qu'elle couvrait, dédupliqué sur l'identité (cf. §10-2 de la conception).
	"shared_kill_events_credit_base_v1",                // shared
	"shared_kill_events_from_pairs_v1",                 // shared (J4 : reprise dédupliquée de killer_victim_pairs -> match_kill_events + drop v_killer_victim_full ; la table source RESTE)
	"shared_objective_events_v1",                       // shared
	"shared_objective_score_v1_drop",                   // shared (v7.5 lot 3 : DROP match_objective_score_timeline ; remplace shared_objective_score_v1, dont le créateur est supprimé)
	"shared_match_player_positions_v1",                 // shared
	"shared_pve_append_only_v1",                        // shared_pve
	"rebuild_match_participants_defeat_art_corruption", // shared
	// Phase 1.5 b27 (reorder escaladé) : skill_v2 (créateur de lusr_hyperparams_v2)
	// AVANT le seed tier_boundaries (qui INSERT dedans). Corrige l'inversion 148/149
	// historique. Sûr : les 2 sont title-owned → n'affecte pas l'ordre du registre global
	// (TestSortByCanonicalIsNoOp). Name-keyed → no-op sur DB déjà migrées ; sur DB fraîche
	// le seed réussit dès le 1er boot (au lieu de converger sur 2 boots via backfill swallowed).
	"shared_create_skill_v2_tables",               // shared
	"shared_seed_tier_boundaries_v2",              // shared
	"player_skill_state_v2_reset_marker_v1",       // shared
	"create_base_shared_social_schema",            // shared_social
	"add_player_slug_to_media_files",              // shared_social
	"add_file_name_to_media_files",                // shared_social
	"add_missing_columns_to_media_files",          // shared_social
	"add_capture_start_indexed_at_to_media_files", // shared_social
	"add_is_manual_to_media_match_associations",   // shared_social
	"add_file_stem_ext_to_media_files",            // shared_social
	"align_media_files_legacy_schema",             // shared_social
	"drop_pn_unread_art_index_v2",                 // shared_social
	"shared_social_favorites_append_only_v1",      // shared_social
	"shared_social_likes_append_only_v1",          // shared_social
	"shared_social_media_assoc_append_only_v1",    // shared_social
	"media_files_drop_filepath_unique_v1",         // shared_social
	// APRÈS le rebuild CTAS ci-dessus : celui-ci énumère les colonnes existantes,
	// donc droper avant ferait juste un rebuild plus court — mais l'ordre naturel
	// (rebuild du schéma legacy, PUIS retrait de colonne) est le plus lisible.
	"drop_media_files_liked_columns_v1",                        // shared_social
	"shared_social_notif_prefs_append_only_v1",                 // shared_social
	"shared_social_notifications_append_only_v1",               // shared_social
	"create_prestige_shared_social_schema",                     // shared_social
	"purge_data_health_warning_notifs",                         // shared_social
	"create_player_records_history_append_only",                // shared_social
	"player_records_history_previous_cols_v1",                  // shared_social
	"extend_player_records_with_window",                        // shared_social
	"shared_social_squad_challenge_participant_append_only_v1", // shared_social
	"shared_social_squad_member_append_only_v1",                // shared_social
	"shared_social_squad_member_gamertag_v1",                   // shared_social
	"rekey_squad_member_xuid",                                  // shared_social
	"add_archived_at_to_squad_challenge",                       // shared_social
	"shared_social_user_prestige_append_only_v1",               // shared_social
	"create_world_player_season_stats",                         // shared
	"create_world_csr_leaderboard_snapshots",                   // shared
	"world_csr_leaderboard_latest_by_batch",                    // shared
	"add_title_slug_to_world_csr_leaderboard",                  // shared (PMT-7)
	"add_xuid_to_world_csr_leaderboard",                        // shared (B1)
	"shared_create_kill_positions",                             // shared (positions monde par kill, ref inter-titres)
	"shared_create_match_commendations",                        // shared (commendations natives par match, ref inter-titres, AXE B)
	"shared_match_commendations_add_progress",                  // shared (total à vie absolu au match — totaux commendations)
	"add_player_count_to_match_registry",                       // shared (roster API attendu — oracle d'intégrité, fix #10)
	"add_weapon_accuracy",                                      // shared (précision par arme/joueur/match, dérivée des events WeaponDrop H5)
	"add_events_empty_to_match_registry",                       // shared (statut distinct « chunk récupéré, 0 event légitime » — fin boucle parse_anomaly)
	"create_season_catalog",                                    // shared (C2 : noms+traductions des saisons CSR Waypoint, source scrape)
	"create_world_player_no_data",                              // shared (marqueur privés/sans-données classement mondial)
	"shared_create_objective_stats",                            // shared (V72-03 : stats objectifs CTF/Zones/Oddball par joueur/match, append-only)
	"shared_objective_stats_add_stockpile_extraction",          // shared (V721-02 : +18 colonnes Stockpile/Extraction/VIP + vue _latest recréée)
	"shared_weapon_kills_v3",                                   // shared (attribution d'arme par kill, voie v3 pur-film)
	"shared_match_weapon_shots_v1",                             // shared (J4 : ventilation des tirs par arme, append-only + vue _latest)
	// S2 — DERNIERS de l'ordre A DESSEIN : ils réparent le DEFAULT de `written_at` sur
	// les tables déjà créées, donc ils doivent suivre TOUTE création de table.
	"written_at_default_utc_shared",        // shared
	"written_at_default_utc_player",        // player
	"written_at_default_utc_shared_pve",    // shared_pve
	"written_at_default_utc_shared_social", // shared_social
	"written_at_default_utc_metadata",      // metadata
	// Lot S6 — clôture de la campagne : les MÊMES DEFAULT sensibles au fuseau sur les
	// autres colonnes d'horodatage (dont computed_at, qui arbitre
	// lusr_component_history_latest). Un step JUMEAU et non l'extension du prédicat
	// ci-dessus : le runner ne rejoue jamais un step déjà au ledger, donc élargir un step
	// déjà appliqué en prod n'aurait réparé aucune des bases qui portent le défaut.
	// Placés APRÈS leurs jumeaux written_at : sur une base vierge l'ordre est indifférent
	// (les deux sont des no-op sur un DDL déjà canonique), sur une base legacy le second
	// ne trouve plus que les colonnes que le premier n'a pas couvertes.
	"arbitration_clocks_default_utc_shared",        // shared
	"arbitration_clocks_default_utc_player",        // player
	"arbitration_clocks_default_utc_shared_pve",    // shared_pve
	"arbitration_clocks_default_utc_shared_social", // shared_social
	"arbitration_clocks_default_utc_metadata",      // metadata
}

var canonicalIndex = func() map[string]int {
	m := make(map[string]int, len(canonicalOrder))
	for i, n := range canonicalOrder {
		m[n] = i
	}
	return m
}()

// CanonicalOrder retourne une copie de l'ordre d'exécution canonique (noms).
// Exposé pour l'audit de complétude inter-packages : les steps title-owned
// (internal/games/{slug}/migrations) sont dans canonicalOrder mais PAS dans le
// registre global All() — l'audit bidirectionnel (global + title) vit donc dans
// le package du titre (halo_infinite/migrations/order_audit_test.go).
func CanonicalOrder() []string {
	return append([]string(nil), canonicalOrder...)
}

// canonicalRank retourne la position de `name` dans canonicalOrder, ou la fin
// (len) si absent — résilience runtime ; order_test.go garantit l'absence
// d'inconnu en CI.
func canonicalRank(name string) int {
	if idx, ok := canonicalIndex[name]; ok {
		return idx
	}
	return len(canonicalOrder)
}

// sortByCanonicalOrder réordonne les migrations selon canonicalOrder (tri
// stable : les inconnus gardent leur ordre relatif d'entrée, en fin de liste).
// Utilisé par le chemin par défaut (Halo) ; un titre non-défaut passe son propre
// ordre via sortByOrder.
func sortByCanonicalOrder(ms []Migration) {
	sortByOrder(canonicalOrder, ms)
}

// sortByOrder réordonne ms selon l'ordre des noms dans `order` (tri stable ; les
// inconnus vont en fin, ordre relatif préservé). Généralise sortByCanonicalOrder
// pour permettre à un TitleMigrationSet d'imposer SON propre ordre (PMT-9).
func sortByOrder(order []string, ms []Migration) {
	rank := make(map[string]int, len(order))
	for i, n := range order {
		rank[n] = i
	}
	rankOf := func(name string) int {
		if idx, ok := rank[name]; ok {
			return idx
		}
		return len(order)
	}
	sort.SliceStable(ms, func(i, j int) bool {
		return rankOf(ms[i].Name) < rankOf(ms[j].Name)
	})
}
