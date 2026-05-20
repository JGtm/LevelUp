// Package handlers — jobs.go : polling de statut des jobs asynchrones (Sprint 17).
//
// GET /jobs/{job_id} → retourne le statut d'un job ou 404 si inconnu/expiré.
package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

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

// GetJob retourne le statut d'un job par son ID.
// GET /jobs/{job_id} → 200 AsyncJobStatus ou 404.
func (h *JobsHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "job_id")
	if jobID == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_job_id", "Identifiant de job manquant.")
		return
	}

	job := h.store.Get(jobID)
	if job == nil {
		writeError(r.Context(), w, http.StatusNotFound, "job_not_found",
			"Job introuvable ou expiré : "+jobID)
		return
	}

	writeJSON(w, http.StatusOK, job)
}
