package migration

// steps_player_prestige_campaign.go — create_improvement_campaign_schema (improvement_campaign
// + challenge.campaign_id) a été migré vers internal/games/halo_infinite/migrations/steps_player_base.go
// (Phase 1.5 b25, voie B). L'ordre (après create_prestige_player_schema qui crée challenge) est
// garanti par internal/migration/order.go (canonicalOrder).
//
// L'index ART idx_challenge_campaign (colonne campaign_id mutée par campaign_repo) N'EST PLUS
// créé par la migration title-owned (éradication ART #23046) ; il est retiré des DB existantes
// par drop_challenge_mutated_art_indexes_v1 (steps_player_drop_challenge_art_indexes.go).
