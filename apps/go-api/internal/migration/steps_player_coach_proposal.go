package migration

// steps_player_coach_proposal.go — create_coach_proposal_player_schema (table
// coach_proposal, ADR 0020) a été migré vers internal/games/halo_infinite/migrations/steps.go
// (Phase 1.5 b14, voie B). Le nom reste dans internal/migration/order.go (canonicalOrder).
//
// L'index ART idx_coach_proposal_user_status (colonne status mutée) N'EST PLUS créé par
// la migration title-owned (éradication ART #23046) ; il est retiré des DB existantes par
// drop_coach_proposal_status_art_index_v1 (steps_player_drop_coach_proposal_status_index.go).
