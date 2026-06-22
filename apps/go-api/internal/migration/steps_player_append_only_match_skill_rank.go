package migration

// steps_player_append_only_match_skill_rank.go — player_append_only_match_skill_rank_v1
// (applyAppendOnlyMatchSkillRank) a été migré vers
// internal/games/halo_infinite/migrations/steps_player_match_skill_rank.go (Phase 1.5 b20,
// voie B). Le nom reste dans internal/migration/order.go (canonicalOrder).
//
// La conversion append-only (CTAS swap id PK + written_at + vue match_skill_rank_latest,
// éradication ART) est portée à l'identique dans le fichier title-owned ci-dessus — aucune
// logique perdue par la relocation.
