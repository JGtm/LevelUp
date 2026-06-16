package migration

// steps_player_append_only_streak.go — create_streak_history_append_only a été migré
// vers internal/games/halo_infinite/migrations/steps_player.go (Phase 1.5 b16, voie B) :
// consommateur de streak (créée par create_progression_player_schema, racine restée
// globale). Le nom reste dans internal/migration/order.go (canonicalOrder).
