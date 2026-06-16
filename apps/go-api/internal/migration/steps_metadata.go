package migration

// steps_metadata.go — add_asset_translations (asset_translations + medal_translations) +
// fix_super_fiesta_fr_label ont été migrés vers
// internal/games/halo_infinite/migrations/steps_metadata_root.go (Phase 1.5 b26, voie B —
// DERNIER root metadata). Tout le tier metadata est désormais title-owned. Les noms restent
// dans internal/migration/order.go (canonicalOrder).
