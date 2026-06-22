package migration

// steps_metadata_catalog.go — add_catalog_playlists (catalogue Playlists/Pairs/Maps,
// 8 tables) a été migré vers internal/games/halo_infinite/migrations/steps.go avec
// la famille catalogue (drop_playlists_catalog_secondary_indexes +
// seed_ranked_playlists_catalog) (Phase 1.5 b12, voie B). Le nom reste dans
// internal/migration/order.go (canonicalOrder).
//
// Les tables mutées (playlists_catalog, game_variants_catalog, map_mode_pair_definitions,
// catalog_fetch_queue) y sont créées PK-only (pas d'index secondaire sur colonnes mutées
// = surface ART #23046) — aligné sur l'éradication ART. Drop/rebuild sur DB existantes :
// drop_metadata_art_surface_indexes_v1 + rebuild_catalog_fetch_queue_drop_art_indexes.
