package migration

// steps_player_append_only_csr_snapshots.go — player_append_only_csr_snapshots_v1 a été
// migré vers internal/games/halo_infinite/migrations/steps_appendonly_misc.go (Phase 1.5
// b22, voie B). Le nom reste dans internal/migration/order.go (canonicalOrder).
//
// La conversion append-only (CTAS swap id PK + written_at + vue _latest, éradication ART)
// est portée à l'identique dans le fichier title-owned ci-dessus — aucune logique perdue
// par la relocation. Le helper applyAppendOnlyRebuild (append_only_rebuild.go) reste utilisé
// par les conversions append-only restées globales (match_citations, personal_score_awards,
// lusr_component_history).
