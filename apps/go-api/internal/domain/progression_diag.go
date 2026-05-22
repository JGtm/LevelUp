// Package domain — progression_diag.go : type métier pour le handler diag
// progression V2 Ascension (/api/v1/_diag/progression/{slug}).
//
// Phase 4 plan stabilisation 2026-05-22.
package domain

// ProgressionDiag reflète l'état des tables progression V2 pour un joueur.
// Émis par /api/v1/_diag/progression/{slug} ; permet de vérifier qu'
// EvaluateProgressionAfterSync tourne bien sur l'auto-sync (avant le fix
// Phase 4, ces tables restaient vides). Counts uniquement, pas de détail.
type ProgressionDiag struct {
	PlayerSlug            string `json:"player_slug"`
	StreakCount           int    `json:"streak_count"`
	PlayerRecordsCount    int    `json:"player_records_count"` // PB dans shared_social
	RecordHistoryCount    int    `json:"record_history_count"`
	MilestoneEarnedCount  int    `json:"milestone_earned_count"`
	MilestoneCatalogCount int    `json:"milestone_catalog_count"` // metadata.duckdb
	// PipelineWiredAt : timestamp du dernier post-sync delta enregistré dans
	// sync_meta. Empty si jamais.
	PipelineWiredAt string `json:"pipeline_wired_at,omitempty"`
}
