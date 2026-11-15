package migration

// steps_player_append_only_streak.go — create_streak_history_append_only a été migré
// vers internal/games/halo_infinite/migrations/steps_player.go (Phase 1.5 b16, voie B) :
// fournisseur de streak_history ; la table héritée `streak` est retirée par drop_streak_legacy_v1
// (créée à l'origine par create_progression_player_schema, racine restée globale). Le nom reste dans internal/migration/order.go (canonicalOrder).
