package migration

// steps_metadata_prestige.go — create_prestige_metadata_schema a été migré vers
// internal/games/halo_infinite/migrations/steps.go ; les seeds TOML
// (seed_prestige_catalog_v1) vers prestige.go (Phase 1.5 b10, voie B).
// Les noms restent dans internal/migration/order.go (canonicalOrder).
//
// Les tables challenge_template / preset_arc y sont créées PK-only (pas d'index sur
// title_slug/cadence/metric, colonnes mutées par les *Repo.Replace = surface ART #23046)
// — aligné sur l'éradication ART. Drop sur DB existantes :
// drop_metadata_art_surface_indexes_v3.
