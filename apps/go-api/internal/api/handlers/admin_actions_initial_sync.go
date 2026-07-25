// Package handlers — admin_actions_initial_sync.go : action « synchronisation
// initiale (re-import complet) d'un joueur au choix » (job asynchrone).
//
// POST /admin/actions/initial-sync/run {player_slug, title_slug?, max_matches?}
//
//	→ 202 + AsyncJobStatus.
//
// Miroir de admin_actions_convergence.go : le runner (RunPlayerInitialSync) fait
// un RunFull (tout l'historique jusqu'au plafond) au lieu du RunDelta de la
// convergence, et claim le SyncGate — joueur déjà en sync → job failed avec
// raison explicite (re-tenter après). Contrairement à /sync/initial (flux
// self-service gardé par la session courante), les tokens viennent du pool
// unifié (ADR 0023) : l'admin peut re-importer n'importe quel joueur déclaré.
//
// Mount enregistre le POST via humacore.NewAPI(r) sur le sous-routeur /admin
// (middleware RequireAuth/RequireAdmin hérités). Réutilise conflictWithJobError
// et titleOrDefaultSlug du package (définis avec la convergence / le monitoring).
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
)

// PlayerInitialSyncRunner exécute le re-import initial (bloquant — appelé dans la
// goroutine du job). Implémenté par ServiceRegistry.RunPlayerInitialSync.
type PlayerInitialSyncRunner func(ctx context.Context, titleSlug, playerSlug string, maxMatches int) (map[string]any, error)

// AdminInitialSyncActionHandler porte l'action initial-sync/run.
type AdminInitialSyncActionHandler struct {
	run   PlayerInitialSyncRunner
	jobs  *jobs.Store
	bgCtx context.Context
	// inFlightErr : sentinelle ErrSyncInFlight du package wire (injectée pour
	// éviter le cycle d'import).
	inFlightErr error
}

// NewAdminInitialSyncActionHandler construit le handler.
func NewAdminInitialSyncActionHandler(
	run PlayerInitialSyncRunner, jobStore *jobs.Store, bgCtx context.Context, inFlightErr error,
) *AdminInitialSyncActionHandler {
	if bgCtx == nil {
		bgCtx = context.Background()
	}
	return &AdminInitialSyncActionHandler{run: run, jobs: jobStore, bgCtx: bgCtx, inFlightErr: inFlightErr}
}

// Mount enregistre la route via Huma sur le sous-routeur chi /admin (middleware
// RequireAuth/RequireAdmin hérités). Le body est REQUIS (décodage maison →
// 400 invalid_input si JSON malformé OU player_slug absent, contrat aligné sur
// la convergence).
func (h *AdminInitialSyncActionHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/actions/initial-sync/run", h.handleRun)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// initialSyncRunInput : ?title= optionnel (fallback titre par défaut, remplacé
// par title_slug du corps s'il est fourni) + corps brut décodé maison. RawBody
// (pas Body typé) → préserve le contrat 400 {invalid_input} sur JSON malformé
// OU player_slug absent (un Body typé renverrait le 422 de validation Huma).
type initialSyncRunInput struct {
	Title   string `query:"title"`
	RawBody []byte
}

// initialSyncRunOutput : 202 Accepted + AsyncJobStatus (job async démarré).
type initialSyncRunOutput struct {
	Status int
	Body   *domain.AsyncJobStatus
}

// handleRun démarre le re-import initial d'un joueur.
// POST /admin/actions/initial-sync/run → 202 (409 si déjà en vol).
func (h *AdminInitialSyncActionHandler) handleRun(ctx context.Context, in *initialSyncRunInput) (*initialSyncRunOutput, error) {
	if h.run == nil || h.jobs == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, "initial_sync_unavailable",
			"Synchronisation initiale indisponible (scheduler non câblé).")
	}
	var req domain.InitialSyncStartRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil || req.PlayerSlug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_input", "player_slug requis.")
	}
	// Titre cible : title_slug du corps (multi-titre explicite) sinon ?title=
	// sinon titre par défaut — jamais de comparaison slug== (title-agnostic).
	titleSlug := req.TitleSlug
	if titleSlug == "" {
		titleSlug = titleOrDefaultSlug(in.Title)
	}

	if active := h.jobs.FindActiveJob(domain.JobTypeAdminInitialSync, req.PlayerSlug); active != nil {
		return nil, &conflictWithJobError{
			Code:      actionBusyCode,
			Message:   "Une synchronisation initiale est déjà en cours pour ce joueur.",
			Retryable: false,
			Details:   map[string]string{jobDetailKeyJobID: active.JobID},
		}
	}

	job := h.jobs.Create(domain.JobTypeAdminInitialSync, req.PlayerSlug)
	step := "synchronisation initiale en cours (re-import complet)"
	h.jobs.SetStatus(job.JobID, domain.JobStatusRunning, &step)
	go h.runInitialSync(h.bgCtx, job.JobID, titleSlug, req.PlayerSlug, req.MaxMatches)
	slog.InfoContext(ctx, "admin_actions: sync initiale joueur démarrée",
		"job_id", job.JobID, "player_slug", req.PlayerSlug, "title", titleSlug, "max_matches", req.MaxMatches)
	return &initialSyncRunOutput{Status: http.StatusAccepted, Body: h.jobs.Get(job.JobID)}, nil
}

// runInitialSync exécute le re-import initial dans la goroutine du job.
func (h *AdminInitialSyncActionHandler) runInitialSync(ctx context.Context, jobID, titleSlug, playerSlug string, maxMatches int) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.ErrorContext(ctx, "admin_actions: panic pendant la sync initiale", "job_id", jobID, "panic", rec)
			h.jobs.Update(jobID, func(j *domain.AsyncJobStatus) {
				j.Error = &domain.JobErrorDetail{Code: jobErrorCodePanic, Message: "synchronisation initiale interrompue par une erreur interne", Retryable: true}
			})
			h.jobs.SetStatus(jobID, domain.JobStatusFailed, nil)
		}
	}()
	result, err := h.run(ctx, titleSlug, playerSlug, maxMatches)
	if err != nil {
		code := "initial_sync_failed"
		retryable := false
		if h.inFlightErr != nil && errors.Is(err, h.inFlightErr) {
			code, retryable = "sync_in_flight", true
			slog.WarnContext(ctx, "admin_actions: sync initiale cédée (sync déjà en vol)",
				"module", "monitoring", "job_id", jobID, "player_slug", playerSlug)
		} else {
			slog.ErrorContext(ctx, "admin_actions: sync initiale échouée",
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
