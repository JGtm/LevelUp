package migration

// steps_world_csr_leaderboard.go — create_world_csr_leaderboard_snapshots a été migré
// vers internal/games/halo_infinite/migrations/steps.go (Phase 1.5 b18, voie B), avec
// world_csr_leaderboard_latest_by_batch (paire atomique sur la vue _latest). Le nom
// reste dans internal/migration/order.go (canonicalOrder).
