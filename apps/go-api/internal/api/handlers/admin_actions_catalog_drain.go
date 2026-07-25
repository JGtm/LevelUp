// Package handlers — admin_actions_catalog_drain.go : action « drain
// DiscoveryUGC » (job asynchrone via JobStore). Réseau, rate-limité — peut
// durer plusieurs minutes, d'où l'exécution en job.
//
// POST /admin/actions/catalog/ugc-drain → 202 + AsyncJobStatus (409 si un drain
// est déjà en cours pour ce titre).
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le routeur chi
// (point de montage /admin/actions/catalog, middleware RequireAuth/RequireAdmin
// hérités) et enregistre le POST via huma.Post. Logique métier inchangée (drain
// en goroutine de job), seul le wrapping HTTP change. Le 409 est un corps de
// CONFLIT (pas une erreur Huma standard) → Status explicite + Body map pour
// préserver le champ details.job_id du contrat d'origine.
package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
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

// Mount enregistre la route via Huma sur le routeur chi (point de montage
// /admin/actions/catalog, middleware RequireAuth/RequireAdmin hérités du groupe).
func (h *AdminCatalogDrainHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Post(api, "/actions/catalog/ugc-drain", h.handleRun, humacore.Op(
		"postAdminActionCatalogUGCDrain",
		"Action admin — recense les asset IDs vus en jeu (catalog_fetch_queue) puis les résout via l'API DiscoveryUGC (réseau, rate-limité) ; complète "+
			"le catalog/refresh zéro-réseau pour les assets absents de match_registry ; job asynchrone (auth admin requis)",
		"admin"),
		// Statut NOMINAL du document ; le champ Status de catalogDrainOutput reste
		// le mécanisme runtime (202 drain lancé / 409 déjà en cours).
		humacore.DefaultStatus(http.StatusAccepted))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// catalogDrainInput : ?title= optionnel (fallback titre par défaut via
// titleOrDefaultSlug, comme l'ancien titleOrDefault(r)).
type catalogDrainInput struct {
	Title string `query:"title"`
}

// catalogDrainOutput : 202 + AsyncJobStatus au lancement, OU 409 + corps de
// conflit (code/message/retryable/details). Le 409 n'est PAS une erreur Huma
// standard (NewError ne porte pas details.job_id) → Status explicite + Body map
// pour préserver byte-identique l'enveloppe d'origine.
//
// Statut DYNAMIQUE : le champ Status est le mécanisme runtime et ne peut pas être
// supprimé ; le statut nominal du document vient de humacore.DefaultStatus (Mount).
type catalogDrainOutput struct {
	Status int
	Body   any
}

// ─── Endpoint ────────────────────────────────────────────────────────────────

// handleRun démarre le drain UGC.
// POST /admin/actions/catalog/ugc-drain → 202 (409 si déjà en cours).
func (h *AdminCatalogDrainHandler) handleRun(ctx context.Context, in *catalogDrainInput) (*catalogDrainOutput, error) {
	if h.run == nil || h.jobs == nil {
		return nil, humacore.NewError(http.StatusServiceUnavailable, "catalog_drain_unavailable",
			"Drain UGC indisponible.")
	}
	titleSlug := titleOrDefaultSlug(in.Title)

	if active := h.jobs.FindActiveJob(domain.JobTypeCatalogUGCDrain, titleSlug); active != nil {
		return &catalogDrainOutput{Status: http.StatusConflict, Body: map[string]any{
			"code":      actionBusyCode,
			"message":   "Un drain UGC est déjà en cours pour ce titre.",
			"retryable": false,
			"details":   map[string]string{jobDetailKeyJobID: active.JobID},
		}}, nil
	}

	job := h.jobs.Create(domain.JobTypeCatalogUGCDrain, titleSlug)
	step := "drain DiscoveryUGC (réseau, rate-limité)"
	h.jobs.SetStatus(job.JobID, domain.JobStatusRunning, &step)
	go h.runDrain(h.bgCtx, job.JobID, titleSlug)
	slog.InfoContext(ctx, "admin_actions: drain UGC démarré", "job_id", job.JobID, "title", titleSlug)
	return &catalogDrainOutput{Status: http.StatusAccepted, Body: h.jobs.Get(job.JobID)}, nil
}

// runDrain exécute le drain dans la goroutine du job.
func (h *AdminCatalogDrainHandler) runDrain(ctx context.Context, jobID, titleSlug string) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.ErrorContext(ctx, "admin_actions: panic pendant le drain UGC", "job_id", jobID, "panic", rec)
			h.jobs.Update(jobID, func(j *domain.AsyncJobStatus) {
				j.Error = &domain.JobErrorDetail{Code: jobErrorCodePanic, Message: "drain interrompu par une erreur interne", Retryable: true}
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
