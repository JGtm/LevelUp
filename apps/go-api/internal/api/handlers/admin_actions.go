// Package handlers — admin_actions.go : actions correctives du dashboard
// monitoring admin (Phase 1 : data health check immédiat + cycle auto-sync
// forcé). Toutes in-process — JAMAIS de spawn de CLI (les CLIs ouvriraient
// les mêmes DuckDB en RW concurrent du serveur).
//
// Routes (montées sous /api/v1/admin/actions/, RequireAuth+RequireAdmin) :
//   - POST /data-health/run : audit data health synchrone (~s, lectures RO)
//   - POST /auto-sync/run   : cycle delta complet via JobStore (202 + job_id,
//     suivi par GET /jobs/{job_id} — RunOnce est réentrant, chaque joueur
//     claim le SyncGate individuellement, donc sans danger pendant un tick)
package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	"levelup/go-api/internal/scheduler"
)

// DataHealthRunNow exécute un audit data health immédiat (implémenté par
// ServiceRegistry.RunDataHealthNow).
type DataHealthRunNow func(ctx context.Context) (*domain.MonitoringDataHealth, error)

// forcedSyncCycleSlug est le PlayerSlug sentinelle des jobs « cycle complet »
// (le cycle couvre tous les joueurs — la dédup FindActiveJob reste par type).
const forcedSyncCycleSlug = "_all"

// AdminActionsHandler sert les actions correctives du dashboard monitoring.
type AdminActionsHandler struct {
	dataHealth DataHealthRunNow
	sched      *scheduler.AutoSyncScheduler
	jobs       *jobs.Store
	// bgCtx : parent des goroutines de jobs — dérivé du serverCtx (annulé au
	// shutdown AVANT duckdb.CloseAll), jamais le ctx de la requête HTTP.
	bgCtx context.Context
}

// NewAdminActionsHandler construit le handler. Chaque dépendance nil dégrade
// l'action correspondante en 503 explicite.
func NewAdminActionsHandler(
	dataHealth DataHealthRunNow,
	sched *scheduler.AutoSyncScheduler,
	jobStore *jobs.Store,
	bgCtx context.Context,
) *AdminActionsHandler {
	if bgCtx == nil {
		bgCtx = context.Background()
	}
	return &AdminActionsHandler{dataHealth: dataHealth, sched: sched, jobs: jobStore, bgCtx: bgCtx}
}

// RunDataHealth exécute l'audit data health et retourne ses compteurs.
// POST /admin/actions/data-health/run.
func (h *AdminActionsHandler) RunDataHealth(w http.ResponseWriter, r *http.Request) {
	if h.dataHealth == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "data_health_unavailable",
			"Audit data health indisponible (scheduler non câblé).")
		return
	}
	res, err := h.dataHealth(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "admin_actions: data health run failed", "err", err)
		writeError(r.Context(), w, http.StatusServiceUnavailable, "data_health_unavailable",
			"Audit data health indisponible.")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// RunSyncCycle force un cycle auto-sync complet, suivi via le JobStore.
// POST /admin/actions/auto-sync/run → 202 + AsyncJobStatus (409 si un cycle
// forcé est déjà en vol).
func (h *AdminActionsHandler) RunSyncCycle(w http.ResponseWriter, r *http.Request) {
	if h.sched == nil || h.jobs == nil {
		writeError(r.Context(), w, http.StatusServiceUnavailable, "scheduler_unavailable",
			"Scheduler auto-sync indisponible.")
		return
	}
	if active := h.jobs.FindActiveJob(domain.JobTypeForcedSyncCycle, forcedSyncCycleSlug); active != nil {
		// 409 en enveloppe d'erreur standard (le client front transforme tout
		// non-2xx en ApiError{code, message, details}) — job_id dans details
		// pour que le front suive directement l'exécution déjà en vol.
		writeJSON(w, http.StatusConflict, map[string]any{
			"code":      "already_running",
			"message":   "Un cycle auto-sync forcé est déjà en cours.",
			"retryable": false,
			"details":   map[string]string{"job_id": active.JobID},
		})
		return
	}
	job := h.jobs.Create(domain.JobTypeForcedSyncCycle, forcedSyncCycleSlug)
	step := "cycle auto-sync en cours"
	h.jobs.SetStatus(job.JobID, domain.JobStatusRunning, &step)
	go h.runForcedCycle(h.bgCtx, job.JobID)
	slog.InfoContext(r.Context(), "admin_actions: cycle auto-sync forcé démarré", "job_id", job.JobID)
	writeJSON(w, http.StatusAccepted, h.jobs.Get(job.JobID))
}

// runForcedCycle exécute le cycle dans une goroutine et reflète le résultat
// dans le job. Recover défensif : un panic dans une goroutine tuerait le
// process (le Recoverer chi ne couvre que les requêtes).
func (h *AdminActionsHandler) runForcedCycle(ctx context.Context, jobID string) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("admin_actions: panic pendant le cycle forcé", "job_id", jobID, "panic", rec)
			h.jobs.Update(jobID, func(j *domain.AsyncJobStatus) {
				j.Error = &domain.JobErrorDetail{Code: "panic", Message: "cycle interrompu par une erreur interne", Retryable: true}
			})
			h.jobs.SetStatus(jobID, domain.JobStatusFailed, nil)
		}
	}()
	res := h.sched.RunOnceTrigger(ctx, "manual")
	h.jobs.Update(jobID, func(j *domain.AsyncJobStatus) {
		j.Result = map[string]any{
			"total":       res.Total,
			"synced":      res.Synced,
			"skipped":     res.Skipped,
			"failed":      res.Failed,
			"duration_ms": res.Duration.Milliseconds(),
		}
	})
	// Le cycle a tourné : le job est « succeeded » même avec des joueurs en
	// échec (le détail par joueur vit dans le snapshot scheduler).
	h.jobs.SetStatus(jobID, domain.JobStatusSucceeded, nil)
}
