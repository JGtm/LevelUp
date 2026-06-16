package migration

// steps_player_dedup_record_history.go — dedup_record_history_v1 a été migré vers
// internal/games/halo_infinite/migrations/steps_player.go (Phase 1.5 b16, voie B) :
// consommateur de record_history (créée par create_progression_player_schema, racine
// restée globale). Le nom reste dans internal/migration/order.go (canonicalOrder).
