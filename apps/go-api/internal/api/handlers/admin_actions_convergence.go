// Package handlers — admin_actions_convergence.go : action « relancer la
// convergence d'un joueur » (job asynchrone via JobStore).
//
// POST /admin/actions/convergence/run {player_slug} → 202 + AsyncJobStatus.
// La convergence = un RunDelta complet du joueur (le post-sync se déclenche
// via hasConvergenceBacklog même à 0 insert). Le runner claim le SyncGate —
// joueur déjà en sync → job failed avec raison explicite (re-tenter après).
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
)

// PlayerConvergenceRunner exécute la convergence (bloquant — appelé dans la
// goroutine du job). Implémenté par ServiceRegistry.RunPlayerConvergence.
type PlayerConvergenceRunner func(ctx context.Context, titleSlug, playerSlug string) (map[string]any, error)

// AdminConvergenceActionHandler porte l'action convergence/run.
type AdminConvergenceActionHandler struct {
	run   PlayerConvergenceRunner
	jobs  *jobs.Store
	bgCtx context.Context
	// inFlightErr : sentinelle ErrSyncInFlight du package api (injectée pour
	// éviter le cycle d'import).
	inFlightErr error
}

// NewAdminConvergenceActionHandler construit le handler.
func NewAdminConvergenceActionHandler(
	run PlayerConvergenceRunner, jobStore *jobs.Store, bgCtx context.Context, inFlightErr error,
) *AdminConvergenceActionHandler {
	if bgCtx == nil {
		bgCtx = context.Background()
	}
	return &AdminConvergenceActionHandler{run: run, jobs: jobStore, bgCtx: bgCtx, inFlightErr: inFlightErr}
}

// Run démarre la convergence d'un joueur.
// POST /admin/actions/convergence/run {player_slug} → 202 (409 si déjà en vol).
func (h *AdminConvergenceActionHandler) Run(w http.ResponseWriter, r *http.Request) {
	if h.run == nil || h.jobs == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "convergence_unavailable",
			"Convergence indisponible (scheduler non câblé).")
		return
	}
	var req domain.PlayerConvergenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PlayerSlug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_input", "player_slug requis.")
		return
	}
	titleSlug := titleOrDefault(r)

	if active := h.jobs.FindActiveJob(domain.JobTypePlayerConvergence, req.PlayerSlug); active != nil {
		writeJSON(w, http.StatusConflict, map[string]any{
			"code":      "already_running",
			"message":   "Une convergence est déjà en cours pour ce joueur.",
			"retryable": false,
			"details":   map[string]string{"job_id": active.JobID},
		})
		return
	}

	job := h.jobs.Create(domain.JobTypePlayerConvergence, req.PlayerSlug)
	step := "convergence en cours (sync delta + post-sync)"
	h.jobs.SetStatus(job.JobID, domain.JobStatusRunning, &step)
	go h.runConvergence(h.bgCtx, job.JobID, titleSlug, req.PlayerSlug)
	slog.InfoContext(r.Context(), "admin_actions: convergence joueur démarrée",
		"job_id", job.JobID, "player_slug", req.PlayerSlug, "title", titleSlug)
	writeJSON(w, http.StatusAccepted, h.jobs.Get(job.JobID))
}

// runConvergence exécute la convergence dans la goroutine du job.
func (h *AdminConvergenceActionHandler) runConvergence(ctx context.Context, jobID, titleSlug, playerSlug string) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("admin_actions: panic pendant la convergence", "job_id", jobID, "panic", rec)
			h.jobs.Update(jobID, func(j *domain.AsyncJobStatus) {
				j.Error = &domain.JobErrorDetail{Code: "panic", Message: "convergence interrompue par une erreur interne", Retryable: true}
			})
			h.jobs.SetStatus(jobID, domain.JobStatusFailed, nil)
		}
	}()
	result, err := h.run(ctx, titleSlug, playerSlug)
	if err != nil {
		code := "convergence_failed"
		retryable := false
		if h.inFlightErr != nil && errors.Is(err, h.inFlightErr) {
			code, retryable = "sync_in_flight", true
			slog.WarnContext(ctx, "admin_actions: convergence cédée (sync déjà en vol)",
				"module", "monitoring", "job_id", jobID, "player_slug", playerSlug)
		} else {
			slog.ErrorContext(ctx, "admin_actions: convergence échouée",
				"module", "monitoring", "job_id", jobID, "player_slug", playerSlug, "err", err)
		}
		h.jobs.Update(jobID, func(j *domain.AsyncJobStatus) {
			j.Error = &domain.JobErrorDetail{Code: code, Message: err.Error(), Retryable: retryable}
		})
		h.jobs.SetStatus(jobID, domain.JobStatusFailed, nil)
		return
	}
	h.jobs.Update(jobID, func(j *domain.AsyncJobStatus) { j.Result = result })
	h.jobs.SetStatus(jobID, domain.JobStatusSucceeded, nil)
}
