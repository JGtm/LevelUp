package migration

// steps_shared_skill_v2.go — shared_create_skill_v2_tables (player_skill_state_v2 +
// lusr_hyperparams_v2 + vues) a été migré vers internal/games/halo_infinite/migrations/
// steps.go (Phase 1.5 b18, voie B). Le seed shared_seed_tier_boundaries_v2 (consommateur
// de lusr_hyperparams_v2) est déjà title-owned (b3). canonicalOrder ordonne désormais le
// créateur AVANT le seed (reorder escaladé, cf. order.go). Les noms restent dans
// internal/migration/order.go (canonicalOrder).
//
// L'éradication ART du watermark (colonne is_reset + vue _latest filtrée, sentinelle reset
// append-only au lieu de DELETE WHERE xuid) est portée par la migration globale séparée
// player_skill_state_v2_reset_marker_v1 (steps_shared_skill_v2_reset_marker.go), exécutée
// APRÈS shared_create_skill_v2_tables — aucune logique perdue.
