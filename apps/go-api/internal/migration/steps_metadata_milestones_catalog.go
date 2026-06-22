package migration

// steps_metadata_milestones_catalog.go — create_milestone_catalog_metadata a été
// migré vers internal/games/halo_infinite/migrations/steps.go ; le seed TOML
// (seed_milestone_catalog_v1) vers milestones.go (Phase 1.5 b11, voie B).
// Les noms restent dans internal/migration/order.go (canonicalOrder).
//
// La table milestone_catalog y est créée PK-only (pas d'index secondaire sur
// title_slug/metric, colonnes mutées par MilestoneCatalogRepo.Upsert = surface ART
// #23046) — aligné sur l'éradication ART. Drop sur DB existantes :
// drop_metadata_art_surface_indexes_v2.
