// Package domain — types métier pour les jobs asynchrones longs.
// Sprint 17 : JobStore persistant + GET /jobs/{job_id} + POST /sync/initial.
package domain

import "time"

// JobStatus représente les états possibles d'un job.
type JobStatus string

const (
	JobStatusQueued      JobStatus = "queued"
	JobStatusRunning     JobStatus = "running"
	JobStatusSucceeded   JobStatus = "succeeded"
	JobStatusFailed      JobStatus = "failed"
	JobStatusCancelled   JobStatus = "cancelled"
	JobStatusInterrupted JobStatus = "interrupted" // running → interrupted au redémarrage
)

// JobType représente les types de jobs supportés.
type JobType string

const (
	JobTypeSetupSmokeTest JobType = "setup_smoke_test"
	JobTypeInitialSync    JobType = "initial_sync"
	JobTypeBackfill       JobType = "backfill"
	JobTypeReindexMedia   JobType = "reindex_media"
	JobTypeOther          JobType = "other"
)

// JobErrorDetail encapsule les détails d'erreur d'un job.
type JobErrorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

// AsyncJobStatus représente le statut complet d'un job asynchrone.
// Miroir de Python AsyncJobStatus (schemas/common.py).
type AsyncJobStatus struct {
	JobID    string    `json:"job_id"`
	JobType  string    `json:"job_type"`
	Status   JobStatus `json:"status"`
	Metadata JobMeta   `json:"metadata,omitempty"`

	ProgressPct   *int    `json:"progress_pct,omitempty"`
	CurrentStep   *string `json:"current_step,omitempty"`

	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`

	Result map[string]any  `json:"result,omitempty"`
	Error  *JobErrorDetail `json:"error,omitempty"`

	// Enrichissements sync initiale (Sprint 3)
	PhaseKey      *string `json:"phase_key,omitempty"`
	PhaseLabel    *string `json:"phase_label,omitempty"`
	MatchesDone   *int    `json:"matches_done,omitempty"`
	MatchesTotal  *int    `json:"matches_total,omitempty"`
	SubtasksDone  *int    `json:"subtasks_done,omitempty"`
	SubtasksTotal *int    `json:"subtasks_total,omitempty"`
	ETASeconds    *int    `json:"eta_seconds,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`

	// Champs internes (non exposés via JSON dans la réponse API)
	PlayerSlug string `json:"player_slug,omitempty"` // pour FindActiveInitialSync
}

// JobMeta contient les métadonnées arbitraires associées à un job.
type JobMeta map[string]any

// IsTerminal retourne vrai si le job est dans un état terminal (fini).
func (j *AsyncJobStatus) IsTerminal() bool {
	switch j.Status {
	case JobStatusSucceeded, JobStatusFailed, JobStatusCancelled, JobStatusInterrupted:
		return true
	default:
		return false
	}
}
