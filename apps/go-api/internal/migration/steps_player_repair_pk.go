package migration

// steps_player_repair_pk.go — repair_player_match_enrichment_primary_key +
// repair_match_citations_primary_key (+ helpers repairPlayerMatchEnrichmentPK /
// repairMatchCitationsPK) ont été migrés vers
// internal/games/halo_infinite/migrations/steps_player_repairs.go (Phase 1.5 b21, voie B).
// L'util RebuildPlayerMatchEnrichmentART (steps_player_rebuild_match_enrichment.go) reste
// dans ce package. Les noms restent dans internal/migration/order.go (canonicalOrder).
//
// Append-only #23046 : ces repairs sont append-only-aware côté title-owned —
// repairPlayerMatchEnrichmentPK no-ope si la colonne `id` existe (jamais de PK match_id),
// repairMatchCitationsPK no-ope si `generation_id` existe (table append-only).
