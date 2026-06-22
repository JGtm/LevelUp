package migration

// steps_player_lusr_chain_rework.go — lusr_chain_rework_v1 a été migré vers
// internal/games/halo_infinite/migrations/steps_player_match_skill_rank.go (Phase 1.5 b20,
// voie B). Le nom reste dans internal/migration/order.go (canonicalOrder).
//
// La purge LUSR y est faite via rebuild CTAS append-only (PAS de DELETE indexé,
// éradication ART #23046) — la logique ART-safe (fonction lusrChainRework) est portée
// dans le fichier title-owned, pas par un DELETE brut.
