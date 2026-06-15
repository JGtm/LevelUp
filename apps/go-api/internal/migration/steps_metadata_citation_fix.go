package migration

// steps_metadata_citation_fix.go — `fix_citation_image_paths_double_encoded` a été
// migré vers internal/games/halo_infinite/migrations/steps.go (Phase 1.5 b5,
// famille citation_mappings : il UPDATE citation_mappings et doit rester collé à
// la chaîne base→pk→v2_fields, sinon il échoue en run global-only).
// Le nom reste dans internal/migration/order.go (canonicalOrder).
