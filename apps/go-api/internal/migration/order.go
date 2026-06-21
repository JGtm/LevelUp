package migration

import "sort"

// order.go â€” ordre d'exÃ©cution EXPLICITE des migrations (Phase 1.5.0,
// title-agnostic ADR 0025). AVANT : l'ordre dÃ©pendait de l'ordre des init()
// (alphabÃ©tique par nom de fichier) + ordre des Register() â€” donc dÃ©placer un
// steps_*.go pouvait rÃ©ordonner les migrations et casser le boot (une migration
// avant sa dÃ©pendance). MAINTENANT : l'ordre est dÃ©fini par canonicalOrder ;
// RunForDB trie dessus. DÃ©placer/renommer un fichier ne change plus rien.
//
// canonicalOrder a Ã©tÃ© gÃ©nÃ©rÃ© depuis l'ordre d'enregistrement courant
// (2026-06-02) â€” la bascule est un no-op (cf. order_test.go). Toute NOUVELLE
// migration doit Ãªtre ajoutÃ©e Ã  cette liste (order_test.go Ã©choue sinon).
var canonicalOrder = []string{
	"add_match_intensity_to_match_registry",                 // shared
	"add_asset_translations",                                // metadata
	"add_battlepass_asset_refs",                             // metadata
	"add_battlepass_metadata",                               // metadata
	"add_challenge_metadata",                                // metadata
	"add_medal_definitions",                                 // metadata
	"add_weapon_labels",                                     // metadata
	"drop_legacy_translation_tables",                        // metadata
	"add_waypoint_assets_raw",                               // metadata
	"add_map_images_registry",                               // metadata
	"add_mode_name_tr",                                      // metadata
	"add_citation_mappings",                                 // metadata
	"add_citation_mappings_pk",                              // metadata
	"add_citation_mappings_v2_fields",                       // metadata
	"add_xbox_achievement_definitions",                      // metadata
	"add_career_rank_translations",                          // metadata
	"medal_definitions_add_indices",                         // metadata
	"enrich_medal_definitions_v2",                           // metadata
	"medal_definitions_add_personal_score",                  // metadata
	"fix_super_fiesta_fr_label",                             // metadata
	"seed_playlist_fr_translations",                         // metadata
	"add_title_id_to_xbox_achievement_definitions",          // metadata
	"cleanup_xbox_achievement_definitions_unknown_title",    // metadata
	"add_xbox_title_id_to_xbox_achievement_definitions",     // metadata
	"add_service_config_id_to_xbox_achievement_definitions", // metadata
	"add_assists_model_coefs",                               // metadata
	"add_catalog_playlists",                                 // metadata
	"fix_citation_image_paths_double_encoded",               // metadata
	"add_csr_placement_thresholds",                          // metadata
	"drop_playlists_catalog_secondary_indexes",              // metadata
	"create_milestone_catalog_metadata",                     // metadata
	"create_prestige_metadata_schema",                       // metadata
	"seed_ranked_playlists_catalog",                         // metadata
	"challenge_template_add_source_column",                  // metadata
	"add_template_tagging_columns",                          // metadata
	"create_base_player_schema",                             // player
	// Les ALTER d'enrichissement DOIVENT suivre create_base_player_schema : sur une
	// DB fraîche (1er provisioning d'un titre, ex. Halo 5) elles étaient ordonnées
	// AVANT la création de player_match_enrichment → l'ALTER no-opait (table absente)
	// et engagement_score_brut/pace n'étaient jamais ajoutées (bug match history h5).
	// Name-keyed → Infinite (déjà migré incrémentalement) intact.
	"add_engagement_score_columns_to_player_match_enrichment", // player
	"create_engagement_coefficients_table",                    // player
	"repair_engagement_coefficients_primary_key",              // player
	"add_engagement_pace_columns_to_player_match_enrichment",  // player
	"add_bot_teammate_column",                                 // player
	"add_career_progression_sequence",                         // player
	"add_career_identity_assets",                              // player
	"add_career_banner_image",                                 // player
	"add_career_last_fetch_status",                            // player
	"add_challenge_snapshots",                                 // player
	"add_battlepass_snapshots",                                // player
	"add_dominance_flag_column",                               // player
	"add_media_discord_notified",                              // player
	"add_media_like_columns",                                  // player
	"add_media_capture_start_utc",                             // player
	"add_performance_score",                                   // player
	"add_player_performance_indexes",                          // player
	"add_pme_session_label",                                   // player
	"add_pme_session_index",                                   // player
	"add_skill_rating_table",                                  // player
	"fix_mv_session_stats_varchar",                            // player
	"add_match_exclusion_flag",                                // player
	"add_player_privacy_state",                                // player
	"drop_media_from_player_db",                               // player
	"add_player_achievements",                                 // player
	"fix_match_citations_schema",                              // player
	"cleanup_spartan_customization_garbage_urls",              // player
	"add_msr_measurement_matches_remaining",                   // player
	"player_add_expected_win_prob",                            // player
	"player_append_only_csr_snapshots_v1",                     // player
	"player_append_only_match_skill_rank_v1",                  // player
	"msr_written_at_default_now_repair_v1",                    // player
	"create_streak_history_append_only",                       // player
	"add_player_assists_model",                                // player
	"create_coach_proposal_player_schema",                     // player
	"dedup_record_history_v1",                                 // player
	"fix_career_xp_total_default_zero",                        // player
	"lusr_chain_rework_v1",                                    // player
	"create_lusr_component_history",                           // player
	"player_msr_view_lusr_over_v2_v1",                         // player
	"player_msr_view_priority_csr_v1",                         // player
	"create_notifications_in_shared_social",                   // shared_social
	"drop_notifications_from_player_db",                       // player
	"drop_idx_pn_xuid_unread",                                 // shared_social
	"player_match_enrichment_performance_chain_v1",            // player
	"create_prestige_player_schema",                           // player
	"create_arc_titles_join",                                  // player (cross-titre arcs backend)
	"create_improvement_campaign_schema",                      // player
	"create_progression_player_schema",                        // player
	"player_match_enrichment_psa_checked_v1",                  // player
	"rebuild_career_progression_defeat_art_corruption",        // player
	"repair_player_match_enrichment_primary_key",              // player
	"repair_match_citations_primary_key",                      // player
	"create_base_shared_schema",                               // shared
	"add_film_match_start",                                    // shared
	"add_highlight_events_autoincrement",                      // shared
	"add_match_participants_columns",                          // shared
	"add_medals_bigint",                                       // shared
	"add_mv_player_matches_fr_cols",                           // shared
	"add_mv_player_matches_view",                              // shared
	"add_shared_performance_indexes",                          // shared
	"add_playable_duration",                                   // shared
	"add_playlist_fr_name_fallback",                           // shared
	"add_spnkr_version",                                       // shared
	"add_team_ps_scores",                                      // shared
	"add_weapon_kills",                                        // shared
	"add_weapon_kills_reconciled_as",                          // shared
	"drop_highlight_events_gamertag",                          // shared
	"fix_bot_xuid",                                            // shared
	"fix_bot_gamertags",                                       // shared
	"fix_events_loaded_inconsistency",                         // shared
	"fix_mv_player_matches_scores",                            // shared
	"add_mv_player_matches_pair_id",                           // shared
	"migrate_weapon_kills_to_ubigint",                         // shared
	"add_media_likes",                                         // shared
	"drop_media_likes_from_shared",                            // shared
	"add_match_registry_i18n_columns",                         // shared
	"add_match_registry_version_ids",                          // shared
	"add_start_time_utc_to_match_registry",                    // shared
	"add_perf_indexes_shared",                                 // shared
	"fix_mv_player_matches_i18n_cols",                         // shared
	"add_mv_player_matches_utc_cols",                          // shared
	"fix_start_time_utc_via_session_tz",                       // shared
	"drop_assists_expected_halo_infinite",                     // shared
	"upgrade_v_gamertag_lookup_bots_and_raw_fallback",         // shared
	"repair_v_gamertag_lookup_bots_2026_05_16",                // shared
	"repair_v_gamertag_lookup_bots_2026_05_30",                // shared
	"shared_add_participation_info_booleans",                  // shared
	"shared_add_participation_timestamps",                     // shared
	"shared_add_t0_quality",                                   // shared
	"shared_backfill_is_ranked_and_season",                    // shared
	"shared_create_player_squad_offset",                       // shared
	"add_shared_match_csrs",                                   // shared
	// Fix fresh-provision (Halo 5 = 1er titre fraîchement sync, 2026-06-20) : la
	// conversion append-only de match_csrs DOIT suivre la création de la table
	// (add_shared_match_csrs) — sinon sur DB FRAÎCHE le rebuild no-ope (table absente)
	// et match_csrs reste en created_at, cassant l'index written_at d'EnsureSharedSchema
	// au 1er OpenSharedDB. Name-keyed → no-op sur DB déjà migrées (Infinite intact).
	// Même pattern documenté que le reorder skill_v2 ci-dessous.
	"shared_append_only_match_csrs_v1",                 // shared
	"add_pve_schema",                                   // shared_pve
	"shared_pve_append_only_v1",                        // shared_pve
	"rebuild_match_participants_defeat_art_corruption", // shared
	// Phase 1.5 b27 (reorder escaladÃ©) : skill_v2 (crÃ©ateur de lusr_hyperparams_v2)
	// AVANT le seed tier_boundaries (qui INSERT dedans). Corrige l'inversion 148/149
	// historique. SÃ»r : les 2 sont title-owned â†’ n'affecte pas l'ordre du registre global
	// (TestSortByCanonicalIsNoOp). Name-keyed â†’ no-op sur DB dÃ©jÃ  migrÃ©es ; sur DB fraÃ®che
	// le seed rÃ©ussit dÃ¨s le 1er boot (au lieu de converger sur 2 boots via backfill swallowed).
	"shared_create_skill_v2_tables",               // shared
	"shared_seed_tier_boundaries_v2",              // shared
	"create_base_shared_social_schema",            // shared_social
	"add_player_slug_to_media_files",              // shared_social
	"add_file_name_to_media_files",                // shared_social
	"add_missing_columns_to_media_files",          // shared_social
	"add_capture_start_indexed_at_to_media_files", // shared_social
	"add_is_manual_to_media_match_associations",   // shared_social
	"add_file_stem_ext_to_media_files",            // shared_social
	"align_media_files_legacy_schema",             // shared_social
	"create_prestige_shared_social_schema",        // shared_social
	"purge_data_health_warning_notifs",            // shared_social
	"create_player_records_history_append_only",   // shared_social
	"player_records_history_previous_cols_v1",     // shared_social
	"extend_player_records_with_window",           // shared_social
	"rekey_squad_member_xuid",                     // shared_social
	"create_world_player_season_stats",            // shared
	"create_world_csr_leaderboard_snapshots",      // shared
	"world_csr_leaderboard_latest_by_batch",       // shared
	"add_title_slug_to_world_csr_leaderboard",     // shared (PMT-7)
	"shared_create_kill_positions",                // shared (positions monde par kill, ref inter-titres)
}

var canonicalIndex = func() map[string]int {
	m := make(map[string]int, len(canonicalOrder))
	for i, n := range canonicalOrder {
		m[n] = i
	}
	return m
}()

// CanonicalOrder retourne une copie de l'ordre d'exÃ©cution canonique (noms).
// ExposÃ© pour l'audit de complÃ©tude inter-packages : les steps title-owned
// (internal/games/{slug}/migrations) sont dans canonicalOrder mais PAS dans le
// registre global All() â€” l'audit bidirectionnel (global + title) vit donc dans
// le package du titre (halo_infinite/migrations/order_audit_test.go).
func CanonicalOrder() []string {
	return append([]string(nil), canonicalOrder...)
}

// canonicalRank retourne la position de `name` dans canonicalOrder, ou la fin
// (len) si absent â€” rÃ©silience runtime ; order_test.go garantit l'absence
// d'inconnu en CI.
func canonicalRank(name string) int {
	if idx, ok := canonicalIndex[name]; ok {
		return idx
	}
	return len(canonicalOrder)
}

// sortByCanonicalOrder rÃ©ordonne les migrations selon canonicalOrder (tri
// stable : les inconnus gardent leur ordre relatif d'entrÃ©e, en fin de liste).
// UtilisÃ© par le chemin par dÃ©faut (Halo) ; un titre non-dÃ©faut passe son propre
// ordre via sortByOrder.
func sortByCanonicalOrder(ms []Migration) {
	sortByOrder(canonicalOrder, ms)
}

// sortByOrder rÃ©ordonne ms selon l'ordre des noms dans `order` (tri stable ; les
// inconnus vont en fin, ordre relatif prÃ©servÃ©). GÃ©nÃ©ralise sortByCanonicalOrder
// pour permettre Ã  un TitleMigrationSet d'imposer SON propre ordre (PMT-9).
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
