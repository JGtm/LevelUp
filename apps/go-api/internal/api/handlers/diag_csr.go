// Package handlers — DiagCSRHandler : GET /api/v1/_diag/csr-coverage/{player_slug}.
//
// Phase 9 du plan pipeline CSR. Endpoint admin/dev permettant de vérifier
// rapidement la couverture CSR d'un joueur : snapshots Waypoint, MSR CSR
// (matured + placement), gap vs match_registry.
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

	"github.com/go-chi/chi/v5"

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

// GetCoverage : GET /api/v1/_diag/csr-coverage/{player_slug}.
func (h *DiagCSRHandler) GetCoverage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	if slug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_slug", "player_slug requis")
		return
	}
	provider, xuid, err := h.newProvider(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	cov, err := provider.GetCoverage(r.Context(), slug, xuid)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "coverage_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, cov)
}
