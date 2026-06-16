// Package migration — steps_metadata_seed_ranked_playlists.go : seed_ranked_playlists_catalog
// (applyRankedPlaylistSeeds + seedRankedPlaylistFR) a été migré vers
// internal/games/halo_infinite/migrations/ranked_playlists.go avec la famille catalogue
// (Phase 1.5 b12, voie B). Le step est statique dans Steps() (ApplySchema:
// applyRankedPlaylistSeeds). Le nom reste dans internal/migration/order.go (canonicalOrder).
package migration
