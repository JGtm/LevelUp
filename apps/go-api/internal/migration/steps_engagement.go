package migration

// steps_engagement.go — la famille EngagementScore (5 steps) a été migrée vers
// internal/games/halo_infinite/migrations/ (Phase 1.5 b17, voie B) :
//   - 4 steps player (add_engagement_score_columns_to_player_match_enrichment,
//     create_engagement_coefficients_table, repair_engagement_coefficients_primary_key,
//     add_engagement_pace_columns_to_player_match_enrichment) → steps_player.go
//   - 1 step shared (add_match_intensity_to_match_registry) → steps.go (section shared)
// Tous consommateurs/self-contained (ALTER player_match_enrichment / match_registry —
// racines globales ; engagement_coefficients create+repair = paire atomique). Les noms
// restent dans internal/migration/order.go (canonicalOrder).
