// Package handlers — admin_actions_convergence.go : action « relancer la
// convergence d'un joueur » (job asynchrone via JobStore).
//
// POST /admin/actions/convergence/run {player_slug} → 202 + AsyncJobStatus.
// La convergence = un RunDelta complet du joueur (le post-sync se déclenche
// via hasConvergenceBacklog même à 0 insert). Le runner claim le SyncGate —
// joueur déjà en sync → job failed avec raison explicite (re-tenter après).
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// /admin (middleware RequireAuth/RequireAdmin hérités) et enregistre le POST via
// huma.Post. Logique métier inchangée (runner + JobStore injectés), seul le
// wrapping HTTP change. Le chemin relatif est identique à la route chi d'origine
// (montée sous /admin par server_admin_monitoring.go).
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

// Mount enregistre la route via Huma sur le sous-routeur chi /admin (middleware
// RequireAuth/RequireAdmin hérités). Le body est REQUIS (décodage maison →
// 400 invalid_input si JSON malformé OU player_slug absent, contrat préservé).
func (h *AdminConvergenceActionHandler) Mount(r chi.Router) {
	api := humacore.NewAPI(r)
	huma.Post(api, "/actions/convergence/run", h.handleRun)
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// convergenceRunInput : ?title= optionnel (fallback titre par défaut) + corps
// brut décodé maison. RawBody (pas Body typé) → préserve le contrat 400
// {invalid_input} sur JSON malformé OU player_slug absent (un Body typé
// renverrait le 422 de validation Huma).
type convergenceRunInput struct {
	Title   string `query:"title"`
	RawBody []byte
}

// convergenceRunOutput : 202 Accepted + AsyncJobStatus (job async démarré).
type convergenceRunOutput struct {
	Status int
	Body   *domain.AsyncJobStatus
}

// conflictWithJobError reproduit EXACTEMENT le corps 409 d'origine
// (writeJSON 409 {code, message, retryable, details:{job_id}}) — humacore.NewError
// ne porte pas de champ `details`, donc un type d'erreur dédié est nécessaire pour
// préserver le job_id que le front consomme. Implémente huma.StatusError → Huma le
// sérialise tel quel via le format byte-identique writeJSON.
type conflictWithJobError struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Retryable bool              `json:"retryable"`
	Details   map[string]string `json:"details"`
}

func (e *conflictWithJobError) Error() string  { return e.Message }
func (e *conflictWithJobError) GetStatus() int { return http.StatusConflict }

// handleRun démarre la convergence d'un joueur.
// POST /admin/actions/convergence/run {player_slug} → 202 (409 si déjà en vol).
func (h *AdminConvergenceActionHandler) handleRun(ctx context.Context, in *convergenceRunInput) (*convergenceRunOutput, error) {
	if h.run == nil || h.jobs == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, "convergence_unavailable",
			"Convergence indisponible (scheduler non câblé).")
	}
	var req domain.PlayerConvergenceRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil || req.PlayerSlug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_input", "player_slug requis.")
	}
	titleSlug := titleOrDefaultSlug(in.Title)

	if active := h.jobs.FindActiveJob(domain.JobTypePlayerConvergence, req.PlayerSlug); active != nil {
		return nil, &conflictWithJobError{
			Code:      actionBusyCode,
			Message:   "Une convergence est déjà en cours pour ce joueur.",
			Retryable: false,
			Details:   map[string]string{jobDetailKeyJobID: active.JobID},
		}
	}

	job := h.jobs.Create(domain.JobTypePlayerConvergence, req.PlayerSlug)
	step := "convergence en cours (sync delta + post-sync)"
	h.jobs.SetStatus(job.JobID, domain.JobStatusRunning, &step)
	go h.runConvergence(h.bgCtx, job.JobID, titleSlug, req.PlayerSlug)
	slog.InfoContext(ctx, "admin_actions: convergence joueur démarrée",
		"job_id", job.JobID, "player_slug", req.PlayerSlug, "title", titleSlug)
	return &convergenceRunOutput{Status: http.StatusAccepted, Body: h.jobs.Get(job.JobID)}, nil
}

// runConvergence exécute la convergence dans la goroutine du job.
func (h *AdminConvergenceActionHandler) runConvergence(ctx context.Context, jobID, titleSlug, playerSlug string) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.ErrorContext(ctx, "admin_actions: panic pendant la convergence", "job_id", jobID, "panic", rec)
			h.jobs.Update(jobID, func(j *domain.AsyncJobStatus) {
				j.Error = &domain.JobErrorDetail{Code: jobErrorCodePanic, Message: "convergence interrompue par une erreur interne", Retryable: true}
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
