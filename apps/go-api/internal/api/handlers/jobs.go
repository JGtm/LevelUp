// Package handlers — jobs.go : polling de statut des jobs asynchrones (Sprint 17).
//
// GET /jobs/{job_id} → MIGRÉ vers Huma (Phase 3b, registerJobsHuma dans le
// package api). La logique métier (lookup store) reste ici via Lookup ; le
// wrapping HTTP (path param + mapping 404) vit dans api/huma_routes.go.
package handlers

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
)

// JobsHandler gère le polling des jobs asynchrones.
type JobsHandler struct {
	store *jobs.Store
}

// NewJobsHandler crée un JobsHandler.
func NewJobsHandler(store *jobs.Store) *JobsHandler {
	return &JobsHandler{store: store}
}

// Lookup retourne le statut d'un job par son ID, ou nil si inconnu/expiré.
func (h *JobsHandler) Lookup(jobID string) *domain.AsyncJobStatus {
	return h.store.Get(jobID)
}
