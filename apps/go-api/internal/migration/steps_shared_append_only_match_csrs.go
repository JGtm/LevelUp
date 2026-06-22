package migration

// steps_shared_append_only_match_csrs.go — shared_append_only_match_csrs_v1 a été migré
// vers internal/games/halo_infinite/migrations/steps_appendonly_misc.go (Phase 1.5 b22,
// voie B). Le nom reste dans internal/migration/order.go (canonicalOrder).
//
// La conversion append-only (CTAS swap id PK + written_at + vue match_csrs_latest,
// éradication ART) est portée à l'identique dans le fichier title-owned ci-dessus — aucune
// logique perdue par la relocation.
