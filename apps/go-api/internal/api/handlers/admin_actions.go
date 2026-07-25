// Package handlers — admin_actions.go : actions correctives du dashboard
// monitoring admin (Phase 1 : data health check immédiat + cycle auto-sync
// forcé). Toutes in-process — JAMAIS de spawn de CLI (les CLIs ouvriraient
// les mêmes DuckDB en RW concurrent du serveur).
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// /admin (middleware RequireAuth/RequireAdmin hérités) et enregistre les 2 POST
// via huma.Post. Logique métier inchangée (runner data health + scheduler +
// JobStore), seul le wrapping HTTP change. Les chemins relatifs sont identiques
// aux routes chi d'origine (montées sous /admin par server.go).
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
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
	"levelup/go-api/internal/scheduler"
)

// DataHealthRunNow exécute un audit data health immédiat (implémenté par
// ServiceRegistry.RunDataHealthNow).
type DataHealthRunNow func(ctx context.Context) (*domain.MonitoringDataHealth, error)

// ActionJournalReporter retourne le journal des actions globales (dernière
// exécution / issue / déclencheur par action). Implémenté par
// ServiceRegistry.ActionJournalReport. Nil → l'endpoint sert un journal vide.
type ActionJournalReporter func(ctx context.Context) domain.AdminActionJournalResponse

// forcedSyncCycleSlug est le PlayerSlug sentinelle des jobs « cycle complet »
// (le cycle couvre tous les joueurs — la dédup FindActiveJob reste par type).
const forcedSyncCycleSlug = "_all"

// Clés/codes partagés par les handlers d'actions admin (goconst).
const (
	jobDetailKeyJobID = "job_id"
	jobErrorCodePanic = "panic"
)

// AdminActionsHandler sert les actions correctives du dashboard monitoring.
type AdminActionsHandler struct {
	dataHealth DataHealthRunNow
	journal    ActionJournalReporter // nil → GET /actions/journal sert un journal vide
	sched      *scheduler.AutoSyncScheduler
	jobs       *jobs.Store
	// bgCtx : parent des goroutines de jobs — dérivé du serverCtx (annulé au
	// shutdown AVANT duckdb.CloseAll), jamais le ctx de la requête HTTP.
	bgCtx context.Context
}

// NewAdminActionsHandler construit le handler. Chaque dépendance nil dégrade
// l'action correspondante en 503 explicite (journal nil → journal vide).
func NewAdminActionsHandler(
	dataHealth DataHealthRunNow,
	journal ActionJournalReporter,
	sched *scheduler.AutoSyncScheduler,
	jobStore *jobs.Store,
	bgCtx context.Context,
) *AdminActionsHandler {
	if bgCtx == nil {
		bgCtx = context.Background()
	}
	return &AdminActionsHandler{dataHealth: dataHealth, journal: journal, sched: sched, jobs: jobStore, bgCtx: bgCtx}
}

// Mount enregistre les 2 actions via Huma sur le sous-routeur chi (préfixe /admin
// + middleware RequireAuth/RequireAdmin hérités). Aucun body de requête (POST
// sans corps) — les deux actions sont déclenchées sans payload.
func (h *AdminActionsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/actions/data-health/run", h.handleRunDataHealth)
	huma.Post(api, "/actions/auto-sync/run", h.handleRunSyncCycle)
	huma.Get(api, "/actions/journal", h.handleGetJournal)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// runDataHealthOutput : 200 + compteurs data health (byte-identique à
// writeJSON(w, 200, *domain.MonitoringDataHealth)).
type runDataHealthOutput struct {
	Body *domain.MonitoringDataHealth
}

// runSyncCycleOutput : statut dynamique (202 job lancé / 409 déjà en cours).
// Body any porte soit *domain.AsyncJobStatus (202), soit le map d'erreur 409
// {code, message, retryable, details} — byte-identique à writeJSON d'origine.
type runSyncCycleOutput struct {
	Status int
	Body   any
}

// actionJournalOutput : 200 + journal des actions globales (dernière exécution
// par action). NoStore posé en interne (état courant, jamais de cache).
type actionJournalOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         domain.AdminActionJournalResponse
}

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleRunDataHealth exécute l'audit data health et retourne ses compteurs.
// POST /admin/actions/data-health/run.
func (h *AdminActionsHandler) handleRunDataHealth(ctx context.Context, _ *struct{}) (*runDataHealthOutput, error) {
	if h.dataHealth == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, "data_health_unavailable",
			"Audit data health indisponible (scheduler non câblé).")
	}
	res, err := h.dataHealth(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "admin_actions: data health run failed", "err", err)
		return nil, humacore.NewError(http.StatusServiceUnavailable, "data_health_unavailable",
			"Audit data health indisponible.")
	}
	return &runDataHealthOutput{Body: res}, nil
}

// handleRunSyncCycle force un cycle auto-sync complet, suivi via le JobStore.
// POST /admin/actions/auto-sync/run → 202 + AsyncJobStatus (409 si un cycle
// forcé est déjà en vol).
func (h *AdminActionsHandler) handleRunSyncCycle(ctx context.Context, _ *struct{}) (*runSyncCycleOutput, error) {
	if h.sched == nil || h.jobs == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, "scheduler_unavailable",
			"Scheduler auto-sync indisponible.")
	}
	if active := h.jobs.FindActiveJob(domain.JobTypeForcedSyncCycle, forcedSyncCycleSlug); active != nil {
		// 409 en enveloppe d'erreur standard (le client front transforme tout
		// non-2xx en ApiError{code, message, details}) — job_id dans details
		// pour que le front suive directement l'exécution déjà en vol.
		return &runSyncCycleOutput{
			Status: http.StatusConflict,
			Body: map[string]any{
				"code":      actionBusyCode,
				"message":   "Un cycle auto-sync forcé est déjà en cours.",
				"retryable": false,
				"details":   map[string]string{jobDetailKeyJobID: active.JobID},
			},
		}, nil
	}
	job := h.jobs.Create(domain.JobTypeForcedSyncCycle, forcedSyncCycleSlug)
	step := "cycle auto-sync en cours"
	h.jobs.SetStatus(job.JobID, domain.JobStatusRunning, &step)
	go h.runForcedCycle(h.bgCtx, job.JobID)
	slog.InfoContext(ctx, "admin_actions: cycle auto-sync forcé démarré", "job_id", job.JobID)
	return &runSyncCycleOutput{Status: http.StatusAccepted, Body: h.jobs.Get(job.JobID)}, nil
}

// runForcedCycle exécute le cycle dans une goroutine et reflète le résultat
// dans le job. Recover défensif : un panic dans une goroutine tuerait le
// process (le Recoverer chi ne couvre que les requêtes).
func (h *AdminActionsHandler) runForcedCycle(ctx context.Context, jobID string) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.ErrorContext(ctx, "admin_actions: panic pendant le cycle forcé", "job_id", jobID, "panic", rec)
			h.jobs.Update(jobID, func(j *domain.AsyncJobStatus) {
				j.Error = &domain.JobErrorDetail{Code: jobErrorCodePanic, Message: "cycle interrompu par une erreur interne", Retryable: true}
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

// handleGetJournal retourne le journal des actions globales (dernière exécution
// / issue / déclencheur par action) — survit au reboot (C2).
// GET /admin/actions/journal.
func (h *AdminActionsHandler) handleGetJournal(ctx context.Context, _ *struct{}) (*actionJournalOutput, error) {
	if h.journal == nil {
		return &actionJournalOutput{CacheControl: noStoreCacheControl, Body: domain.AdminActionJournalResponse{
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
			Actions:     []domain.AdminActionJournalEntry{},
		}}, nil
	}
	return &actionJournalOutput{CacheControl: noStoreCacheControl, Body: h.journal(ctx)}, nil
}
