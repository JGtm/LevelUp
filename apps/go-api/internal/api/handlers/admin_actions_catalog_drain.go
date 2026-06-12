// Package handlers — admin_actions_catalog_drain.go : action « drain
// DiscoveryUGC » (job asynchrone via JobStore). Réseau, rate-limité — peut
// durer plusieurs minutes, d'où l'exécution en job.
//
// POST /admin/actions/catalog/ugc-drain → 202 + AsyncJobStatus (409 si un drain
// est déjà en cours pour ce titre).
package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
)

// CatalogDrainRunner exécute le drain UGC (bloquant — appelé dans la goroutine
// du job). Implémenté par ServiceRegistry.RunCatalogUGCDrain.
type CatalogDrainRunner func(ctx context.Context, titleSlug string) (domain.CatalogUGCDrainResult, error)

// AdminCatalogDrainHandler porte l'action catalog/ugc-drain.
type AdminCatalogDrainHandler struct {
	run   CatalogDrainRunner
	jobs  *jobs.Store
	bgCtx context.Context
}

// NewAdminCatalogDrainHandler construit le handler.
func NewAdminCatalogDrainHandler(run CatalogDrainRunner, jobStore *jobs.Store, bgCtx context.Context) *AdminCatalogDrainHandler {
	if bgCtx == nil {
		bgCtx = context.Background()
	}
	return &AdminCatalogDrainHandler{run: run, jobs: jobStore, bgCtx: bgCtx}
}

// Run démarre le drain UGC.
// POST /admin/actions/catalog/ugc-drain → 202 (409 si déjà en cours).
func (h *AdminCatalogDrainHandler) Run(w http.ResponseWriter, r *http.Request) {
	if h.run == nil || h.jobs == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "catalog_drain_unavailable",
			"Drain UGC indisponible.")
		return
	}
	titleSlug := titleOrDefault(r)

	if active := h.jobs.FindActiveJob(domain.JobTypeCatalogUGCDrain, titleSlug); active != nil {
		writeJSON(w, http.StatusConflict, map[string]any{
			"code":      "already_running",
			"message":   "Un drain UGC est déjà en cours pour ce titre.",
			"retryable": false,
			"details":   map[string]string{"job_id": active.JobID},
		})
		return
	}

	job := h.jobs.Create(domain.JobTypeCatalogUGCDrain, titleSlug)
	step := "drain DiscoveryUGC (réseau, rate-limité)"
	h.jobs.SetStatus(job.JobID, domain.JobStatusRunning, &step)
	go h.runDrain(h.bgCtx, job.JobID, titleSlug)
	slog.InfoContext(r.Context(), "admin_actions: drain UGC démarré", "job_id", job.JobID, "title", titleSlug)
	writeJSON(w, http.StatusAccepted, h.jobs.Get(job.JobID))
}

// runDrain exécute le drain dans la goroutine du job.
func (h *AdminCatalogDrainHandler) runDrain(ctx context.Context, jobID, titleSlug string) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("admin_actions: panic pendant le drain UGC", "job_id", jobID, "panic", rec)
			h.jobs.Update(jobID, func(j *domain.AsyncJobStatus) {
				j.Error = &domain.JobErrorDetail{Code: "panic", Message: "drain interrompu par une erreur interne", Retryable: true}
			})
			h.jobs.SetStatus(jobID, domain.JobStatusFailed, nil)
		}
	}()
	result, err := h.run(ctx, titleSlug)
	if err != nil {
		slog.ErrorContext(ctx, "admin_actions: drain UGC échoué",
			"module", "monitoring", "job_id", jobID, "title", titleSlug, "err", err)
		h.jobs.Update(jobID, func(j *domain.AsyncJobStatus) {
			j.Error = &domain.JobErrorDetail{Code: "catalog_drain_failed", Message: err.Error(), Retryable: true}
		})
		h.jobs.SetStatus(jobID, domain.JobStatusFailed, nil)
		return
	}
	h.jobs.Update(jobID, func(j *domain.AsyncJobStatus) {
		j.Result = map[string]any{
			"seeded":        result.Seeded,
			"playlists":     result.Playlists,
			"pairs":         result.Pairs,
			"maps":          result.Maps,
			"game_variants": result.GameVariants,
			"errors":        result.Errors,
		}
	})
	h.jobs.SetStatus(jobID, domain.JobStatusSucceeded, nil)
}
