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
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"levelup/go-api/internal/api/handlers"
	"levelup/go-api/internal/api/humacore"
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
	}, humacore.Op("getChangelog", "Changelog Markdown du projet", "health"))
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
	}, humacore.Op("getJob", "Statut d'un job long", "jobs"))
}

// gamertagSearchHumaInput : query params du GET /directory/gamertags/search.
//   - q    : fragment recherché.
//   - live : arme le repli LIVE (résolution Xbox d'un joueur jamais croisé). Défaut
//     false = recherche locale seule, rapide (typeahead). true uniquement sur intention
//     explicite (bouton « Rechercher sur Xbox ») — challenge V72-24 (latence).
type gamertagSearchHumaInput struct {
	Q    string `query:"q"`
	Live bool   `query:"live"`
}

// gamertagSearchHumaOutput : corps = GamertagSearchResponse (writeJSON(200, resp)).
type gamertagSearchHumaOutput struct {
	Body domain.GamertagSearchResponse
}

// registerGamertagHuma migre GET /directory/gamertags/search vers Huma (shape
// query-param — Phase 3b). Contrat préservé : 200 {query, items} ; 503
// {code:shared_db_unavailable} si service absent ; 500 {code:gamertag_search_error}
// (message générique « internal error » sur 5xx, comme writeError).
func registerGamertagHuma(api huma.API, h *handlers.GamertagHandler) {
	huma.Get(api, "/directory/gamertags/search", func(ctx context.Context, in *gamertagSearchHumaInput) (*gamertagSearchHumaOutput, error) {
		resp, err := h.Query(ctx, in.Q, in.Live)
		if err != nil {
			if errors.Is(err, handlers.ErrGamertagSearchUnavailable) {
				return nil, newHumaError(http.StatusServiceUnavailable, "shared_db_unavailable", "gamertag search requires shared database")
			}
			return nil, newHumaError(http.StatusInternalServerError, "gamertag_search_error", err.Error())
		}
		return &gamertagSearchHumaOutput{Body: resp}, nil
	}, humacore.Op("searchGamertags", "Recherche floue de gamertags", "explorer"))
}
