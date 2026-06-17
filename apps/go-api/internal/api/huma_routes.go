// Package api — huma_routes.go : enregistrement des routes migrées vers Huma
// (Phase 3b). Chaque route migrée garde son contrat JSON et son format d'erreur
// (newHumaError → {code, message, retryable}, cf. huma_setup.go). La logique
// métier reste dans handlers/ ; ici seul le wrapping Input/Output + registration.
//
// Migration INCRÉMENTALE : tant que toutes les routes ne sont pas migrées,
// l'openapi.yaml MANUEL reste la source de vérité contractuelle (l'OpenAPI Huma
// vit sur un path interne neutre, cf. newHumaAPI). À la fin de la migration
// seulement, on basculera openapi.yaml en généré + régén du client front.
package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/domain"
)

// changelogHumaOutput : contrat de réponse GET /api/v1/changelog ({content}).
type changelogHumaOutput struct {
	Body struct {
		Content string `json:"content"`
	}
}

// registerChangelogHuma migre GET /changelog vers Huma (1re route pilote Phase 3b).
// Contrat strictement préservé : 200 {content} ; 404 {code:CHANGELOG_NOT_FOUND,...}.
func registerChangelogHuma(api huma.API, h *handlers.ChangelogHandler) {
	huma.Get(api, "/changelog", func(_ context.Context, _ *struct{}) (*changelogHumaOutput, error) {
		content, err := h.Content()
		if err != nil {
			return nil, newHumaError(http.StatusNotFound, "CHANGELOG_NOT_FOUND", "Changelog introuvable")
		}
		out := &changelogHumaOutput{}
		out.Body.Content = content
		return out, nil
	})
}

// jobStatusHumaInput : path param du GET /jobs/{job_id}.
type jobStatusHumaInput struct {
	JobID string `path:"job_id"`
}

// jobStatusHumaOutput : corps = AsyncJobStatus tel quel (writeJSON(200, job)).
type jobStatusHumaOutput struct {
	Body *domain.AsyncJobStatus
}

// registerJobsHuma migre GET /jobs/{job_id} vers Huma (shape path-param — Phase
// 3b). Contrat préservé : 200 AsyncJobStatus ; 404 {code:job_not_found,...}.
func registerJobsHuma(api huma.API, h *handlers.JobsHandler) {
	huma.Get(api, "/jobs/{job_id}", func(_ context.Context, in *jobStatusHumaInput) (*jobStatusHumaOutput, error) {
		if in.JobID == "" {
			return nil, newHumaError(http.StatusBadRequest, "missing_job_id", "Identifiant de job manquant.")
		}
		job := h.Lookup(in.JobID)
		if job == nil {
			return nil, newHumaError(http.StatusNotFound, "job_not_found", "Job introuvable ou expiré : "+in.JobID)
		}
		return &jobStatusHumaOutput{Body: job}, nil
	})
}
