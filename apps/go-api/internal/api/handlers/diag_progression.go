// Package handlers — DiagProgressionHandler : GET /api/v1/_diag/progression/{player_slug}.
//
// Phase 4 plan stabilisation 2026-05-22. Endpoint admin/dev qui expose
// l'état des tables progression V2 Ascension pour un joueur (streaks,
// records, milestones). Utile pour valider qu'EvaluateProgressionAfterSync
// tourne bien sur l'auto-sync — avant ce fix, ces tables restaient vides
// (cf. AUDIT_ASCENSION_PIPELINE_DISCONNECTED_2026-05-21).
//
// Usage typique post-déploiement :
//
//	curl -s http://127.0.0.1:8000/api/v1/_diag/progression/JGtm | jq
//
// Si streak_count + player_records_count + milestone_earned_count = 0
// après plusieurs cycles auto-sync (15 min × N), le pipeline V2 n'est
// PAS câblé correctement — investiguer cmd/server/main.go::WithPostSyncRunner.
package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
)

// ProgressionDiagProvider est l'interface attendue par le handler.
// Implémentée par duckdb.ProgressionDiagRepo.
type ProgressionDiagProvider interface {
	GetProgressionDiag(ctx context.Context, slug string) (*domain.ProgressionDiag, error)
}

// ProgressionDiagFactory : résout slug → provider.
type ProgressionDiagFactory func(ctx context.Context, slug string) (ProgressionDiagProvider, error)

// DiagProgressionHandler expose le endpoint progression diag.
type DiagProgressionHandler struct {
	newProvider ProgressionDiagFactory
}

// NewDiagProgressionHandler crée un handler avec une factory injectée.
func NewDiagProgressionHandler(newProvider ProgressionDiagFactory) *DiagProgressionHandler {
	return &DiagProgressionHandler{newProvider: newProvider}
}

// GetDiag : GET /api/v1/_diag/progression/{player_slug}.
func (h *DiagProgressionHandler) GetDiag(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	if slug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_slug", "player_slug requis")
		return
	}
	provider, err := h.newProvider(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}
	diag, err := provider.GetProgressionDiag(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "diag_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, diag)
}
