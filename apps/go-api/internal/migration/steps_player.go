package migration

// steps_player.go — les 25 migrations de base stats.duckdb (create_base_player_schema …
// add_msr_measurement_matches_remaining) + leurs helpers (applyCareerProgressionSequence,
// applyCareerProgressionIdentityAssets, applyFixMvSessionStats) ont été migrés vers
// internal/games/halo_infinite/migrations/steps_player_base.go (Phase 1.5 b25, voie B —
// RACINE player déplacée après tous ses consommateurs). Les noms restent dans
// internal/migration/order.go (canonicalOrder).
