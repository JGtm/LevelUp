// Package handlers — DiagCSRHandler : GET /api/v1/_diag/csr-coverage/{player_slug}.
//
// Phase 9 du plan pipeline CSR. Endpoint admin/dev permettant de vérifier
// rapidement la couverture CSR d'un joueur : snapshots Waypoint, MSR CSR
// (matured + placement), gap vs match_registry.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) au point de
// montage (préfixe /api/v1, middleware NoStore hérité) et enregistre le GET
// via huma.Get. Logique métier inchangée, seul le wrapping HTTP change.
//
// Usage typique post-backfill :
//
//	curl -s http://127.0.0.1:8000/api/v1/_diag/csr-coverage/JGtm | jq
//
// Si needs_backfill=true → lancer `levelup backfill --gamertag X --csr --force`.
package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
)

// CSRCoverageProvider est l'interface attendue par le handler. Implémentée
// par duckdb.CSRCoverageRepo et mockable dans les tests.
type CSRCoverageProvider interface {
	GetCoverage(ctx context.Context, playerSlug, xuid string) (*domain.CSRCoverage, error)
}

// CSRCoverageFactory : résout slug → CSRCoverageProvider + xuid (player-scoped).
type CSRCoverageFactory func(ctx context.Context, slug string) (provider CSRCoverageProvider, xuid string, err error)

// DiagCSRHandler expose le endpoint coverage CSR.
type DiagCSRHandler struct {
	newProvider CSRCoverageFactory
}

// NewDiagCSRHandler crée un handler avec une factory de provider injectée.
func NewDiagCSRHandler(newProvider CSRCoverageFactory) *DiagCSRHandler {
	return &DiagCSRHandler{newProvider: newProvider}
}

// Mount enregistre la route via Huma au point de montage chi `r` (préfixe
// /api/v1 + middleware NoStore hérités). {player_slug} est lu dans le path de
// la route via le champ Input `path:"player_slug"`.
func (h *DiagCSRHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/_diag/csr-coverage/{player_slug}", h.GetCoverage, humacore.Op("getDiagCSRCoverage", "Diag couverture CSR pour un joueur", "diagnostics"))
}

// diagCSRInput : path param {player_slug} (présent dans le path de la route).
type diagCSRInput struct {
	PlayerSlug string `path:"player_slug"`
}

// diagCSROutput : payload coverage CSR sérialisé en 200.
type diagCSROutput struct {
	Body *domain.CSRCoverage
}

// GetCoverage : GET /api/v1/_diag/csr-coverage/{player_slug}.
func (h *DiagCSRHandler) GetCoverage(ctx context.Context, in *diagCSRInput) (*diagCSROutput, error) {
	slug := in.PlayerSlug
	if slug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_slug", "player_slug requis")
	}
	provider, xuid, err := h.newProvider(ctx, slug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	cov, err := provider.GetCoverage(ctx, slug, xuid)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "coverage_error", err.Error())
	}
	return &diagCSROutput{Body: cov}, nil
}
