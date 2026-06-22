package migration

// steps_player_prestige.go — create_prestige_player_schema (arc, challenge, moment_card,
// prestige_telemetry, baseline_state) a été migré vers
// internal/games/halo_infinite/migrations/steps_player_base.go (Phase 1.5 b25, voie B).
// Le nom reste dans internal/migration/order.go (canonicalOrder).
//
// Éradication ART (#23046) : les index idx_ch_user_status (status muté) et idx_ch_arc
// (arc_id muté) sur challenge ne sont PLUS créés par la migration title-owned ; seul
// idx_ch_metric (metric jamais muté) reste. Drop sur DB existantes :
// drop_challenge_mutated_art_indexes_v1.
