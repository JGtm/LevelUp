package migration

// steps_shared_skill_v2.go — shared_create_skill_v2_tables (player_skill_state_v2 +
// lusr_hyperparams_v2 + vues) a été migré vers internal/games/halo_infinite/migrations/
// steps.go (Phase 1.5 b18, voie B). Le seed shared_seed_tier_boundaries_v2 (consommateur
// de lusr_hyperparams_v2) est déjà title-owned (b3). Inversion canonicalOrder
// (seed avant créateur) inchangée — fix réservé au reorder escaladé. Les noms restent
// dans internal/migration/order.go (canonicalOrder).
