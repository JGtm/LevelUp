package migration

// steps_shared_pve_append_only.go — shared_pve_append_only_v1 a été migré vers
// internal/games/halo_infinite/migrations/steps_appendonly_misc.go (Phase 1.5 b22, voie B),
// rejoint add_pve_schema (déjà title b3). Le nom reste dans internal/migration/order.go.
//
// La conversion append-only (CTAS swap id PK + written_at + vue pve_match_stats_latest,
// éradication ART) est portée à l'identique dans le fichier title-owned ci-dessus — aucune
// logique perdue par la relocation.
