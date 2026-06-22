//go:build integration

// steps_metadata_catalog_test.go — tests du catalogue Playlists/Pairs/Maps déplacés
// vers internal/games/halo_infinite/migrations/catalog_test.go (Phase 1.5 b12) :
// add_catalog_playlists est title-owned, RunForDB nécessite le provider StepsFor.
//
// Le test de dédup catalog_fetch_queue y suit désormais le pattern SELECT-then-INSERT
// (NOT EXISTS) : rebuild_catalog_fetch_queue_drop_art_indexes retire la PK (surface ART
// du drain), donc INSERT OR IGNORE ne s'applique plus.
package migration
