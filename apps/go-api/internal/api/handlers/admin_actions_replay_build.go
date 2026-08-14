// Package handlers — admin_actions_replay_build.go : action « construire le rejeu 2D
// d'un match » (job asynchrone via JobStore, patron admin_actions_convergence).
//
// POST /admin/actions/replay-build/run {match_id} → 202 + AsyncJobStatus.
// Le runner appelle la LIBRAIRIE internal/replaybuild (jamais un exec de CLI) ; le
// décodage est sérialisé par le verrou process filmdec, partagé avec killsource. Le
// single-flight applicatif vit dans le runner (replayBuildMu) — ici on refuse seulement
// un second job sur le MÊME match (409 avec le job_id actif, contrat convergence).
package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/jobs"
)

// ReplayBuildRunner exécute la construction (bloquant — appelé dans la goroutine du
// job). Implémenté par ServiceRegistry.RunReplayBuild.
type ReplayBuildRunner func(ctx context.Context, titleSlug, matchID string) (map[string]any, error)

// ReplayBuildEnqueuer met la construction en FILE DURABLE au lieu de la faire
// ici : le travail part alors à un ouvrier, éventuellement sur une autre machine.
// Implémenté par ServiceRegistry.EnqueueReplayBuild.
type ReplayBuildEnqueuer func(ctx context.Context, titleSlug, matchID string) (domain.BuildQueueJob, bool, error)

// AdminReplayBuildActionHandler porte les actions replay-build/{run,enqueue}.
type AdminReplayBuildActionHandler struct {
	run     ReplayBuildRunner
	enqueue ReplayBuildEnqueuer
	jobs    *jobs.Store
	bgCtx   context.Context
}

// NewAdminReplayBuildActionHandler construit le handler.
func NewAdminReplayBuildActionHandler(
	run ReplayBuildRunner, jobStore *jobs.Store, bgCtx context.Context,
) *AdminReplayBuildActionHandler {
	if bgCtx == nil {
		bgCtx = context.Background()
	}
	return &AdminReplayBuildActionHandler{run: run, jobs: jobStore, bgCtx: bgCtx}
}

// WithEnqueuer branche la mise en file (route /enqueue). Nil → la route répond
// 503 : sans store monitoring, il n'y a pas de file où mettre quoi que ce soit.
func (h *AdminReplayBuildActionHandler) WithEnqueuer(enqueue ReplayBuildEnqueuer) *AdminReplayBuildActionHandler {
	h.enqueue = enqueue
	return h
}

// Mount enregistre la route via Huma sur le sous-routeur chi /admin (middleware
// RequireAuth/RequireAdmin hérités). Body décodé maison → 400 invalid_input si JSON
// malformé OU match_id absent (contrat convergence préservé).
func (h *AdminReplayBuildActionHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/actions/replay-build/run", h.handleRun, humacore.Op(
		"postAdminActionReplayBuildRun",
		"Action admin — construit l'artefact de rejeu 2D d'un match depuis le cache film local (job asynchrone, décodage hors ligne, auth admin requis)",
		"admin"),
		humacore.DefaultStatus(http.StatusAccepted))
	huma.Post(api, "/actions/replay-build/enqueue", h.handleEnqueue, humacore.Op(
		"postAdminActionReplayBuildEnqueue",
		"Action admin — met la construction du rejeu 2D d'un match dans la file durable : le manifeste du film est résolu ici (tokens) et les URL CDN "+
			"pré-signées partent dans le job, qu'un ouvrier distant prendra (auth admin requis)",
		"admin"),
		humacore.DefaultStatus(http.StatusAccepted))
}

// replayBuildEnqueueOutput : 202 Accepted + le job de la file.
type replayBuildEnqueueOutput struct {
	Body ReplayBuildEnqueueResponse
}

// ReplayBuildEnqueueResponse : accusé de mise en file. Created=false signale un
// doublon absorbé (le match était déjà en file ou en cours) — ce n'est pas une
// erreur, c'est l'idempotence qui a fait son travail.
type ReplayBuildEnqueueResponse struct {
	Job     domain.BuildQueueJob `json:"job"`
	Created bool                 `json:"created"`
}

// handleEnqueue met la construction du rejeu d'un match en file durable.
// POST /admin/actions/replay-build/enqueue {match_id} → 202.
func (h *AdminReplayBuildActionHandler) handleEnqueue(ctx context.Context, in *replayBuildRunInput) (*replayBuildEnqueueOutput, error) {
	if h.enqueue == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, "build_queue_unavailable",
			"File de construction indisponible (store monitoring non ouvert).")
	}
	var req replayBuildRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil || req.MatchID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_input", "match_id requis.")
	}
	titleSlug := titleOrDefaultSlug(in.Title)
	job, created, err := h.enqueue(ctx, titleSlug, req.MatchID)
	if err != nil {
		slog.ErrorContext(ctx, "admin_actions: mise en file du rejeu échouée",
			"module", "monitoring", "match_id", req.MatchID, "title", titleSlug, "err", err)
		return nil, humacore.NewError(http.StatusBadRequest, "replay_enqueue_failed", err.Error())
	}
	slog.InfoContext(ctx, "admin_actions: rejeu 2D mis en file",
		"job_id", job.JobID, "match_id", req.MatchID, "title", titleSlug, "created", created)
	return &replayBuildEnqueueOutput{Body: ReplayBuildEnqueueResponse{Job: job, Created: created}}, nil
}

// replayBuildRunInput : ?title= optionnel + corps brut décodé maison.
type replayBuildRunInput struct {
	Title   string `query:"title"`
	RawBody []byte
}

// replayBuildRunOutput : 202 Accepted + AsyncJobStatus.
type replayBuildRunOutput struct {
	Body *domain.AsyncJobStatus
}

// replayBuildRequest : corps de POST /admin/actions/replay-build/run.
type replayBuildRequest struct {
	MatchID string `json:"match_id"`
}

// handleRun démarre la construction du rejeu d'un match.
// POST /admin/actions/replay-build/run {match_id} → 202 (409 si déjà en vol sur ce match).
func (h *AdminReplayBuildActionHandler) handleRun(ctx context.Context, in *replayBuildRunInput) (*replayBuildRunOutput, error) {
	if h.run == nil || h.jobs == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, "replay_build_unavailable",
			"Construction de rejeu indisponible (runner non câblé).")
	}
	var req replayBuildRequest
	if err := json.Unmarshal(in.RawBody, &req); err != nil || req.MatchID == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "invalid_input", "match_id requis.")
	}
	titleSlug := titleOrDefaultSlug(in.Title)

	if active := h.jobs.FindActiveJob(domain.JobTypeReplayBuild, req.MatchID); active != nil {
		return nil, &conflictWithJobError{
			Code:      actionBusyCode,
			Message:   "Une construction de rejeu est déjà en cours pour ce match.",
			Retryable: false,
			Details:   map[string]string{jobDetailKeyJobID: active.JobID},
		}
	}

	job := h.jobs.Create(domain.JobTypeReplayBuild, req.MatchID)
	step := "construction du rejeu 2D en cours (décodage hors ligne du film)"
	h.jobs.SetStatus(job.JobID, domain.JobStatusRunning, &step)
	go h.runBuild(h.bgCtx, job.JobID, titleSlug, req.MatchID)
	slog.InfoContext(ctx, "admin_actions: construction rejeu 2D démarrée",
		"job_id", job.JobID, "match_id", req.MatchID, "title", titleSlug)
	return &replayBuildRunOutput{Body: h.jobs.Get(job.JobID)}, nil
}

// runBuild exécute la construction dans la goroutine du job.
func (h *AdminReplayBuildActionHandler) runBuild(ctx context.Context, jobID, titleSlug, matchID string) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.ErrorContext(ctx, "admin_actions: panic pendant la construction du rejeu",
				"job_id", jobID, "panic", rec)
			h.jobs.Update(jobID, func(j *domain.AsyncJobStatus) {
				j.Error = &domain.JobErrorDetail{Code: jobErrorCodePanic, Message: "construction interrompue par une erreur interne", Retryable: true}
			})
			h.jobs.SetStatus(jobID, domain.JobStatusFailed, nil)
		}
	}()
	result, err := h.run(ctx, titleSlug, matchID)
	if err != nil {
		slog.ErrorContext(ctx, "admin_actions: construction rejeu 2D échouée",
			"module", "monitoring", "job_id", jobID, "match_id", matchID, "err", err)
		h.jobs.Update(jobID, func(j *domain.AsyncJobStatus) {
			j.Error = &domain.JobErrorDetail{Code: "replay_build_failed", Message: err.Error(), Retryable: false}
		})
		h.jobs.SetStatus(jobID, domain.JobStatusFailed, nil)
		return
	}
	h.jobs.Update(jobID, func(j *domain.AsyncJobStatus) { j.Result = result })
	h.jobs.SetStatus(jobID, domain.JobStatusSucceeded, nil)
}
