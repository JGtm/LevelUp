// Package handlers — DiagProgressionHandler : GET /api/v1/_diag/progression/{player_slug}.
//
// Phase 4 plan stabilisation 2026-05-22. Endpoint admin/dev qui expose
// l'état des tables progression V2 Ascension pour un joueur (streaks,
// records, milestones). Utile pour valider qu'EvaluateProgressionAfterSync
// tourne bien sur l'auto-sync — avant ce fix, ces tables restaient vides
// (cf. AUDIT_ASCENSION_PIPELINE_DISCONNECTED_2026-05-21).
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) au point de
// montage (préfixe /api/v1, middleware NoStore hérité) et enregistre le GET
// via huma.Get. Logique métier inchangée, seul le wrapping HTTP change.
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

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
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

// Mount enregistre la route via Huma au point de montage chi (préfixe /api/v1
// + middleware NoStore hérité). Lit {player_slug} dans son propre path.
func (h *DiagProgressionHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/_diag/progression/{player_slug}", h.handleGetDiag, humacore.Op("getDiagProgression", "Diag pipeline progression V2 (Ascension)", "diagnostics"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// diagProgressionInput : path param {player_slug} (présent dans le path de la route).
type diagProgressionInput struct {
	PlayerSlug string `path:"player_slug"`
}

type diagProgressionOutput struct{ Body *domain.ProgressionDiag }

// ─── Endpoint ────────────────────────────────────────────────────────────────

// handleGetDiag : GET /api/v1/_diag/progression/{player_slug}.
func (h *DiagProgressionHandler) handleGetDiag(ctx context.Context, in *diagProgressionInput) (*diagProgressionOutput, error) {
	if in.PlayerSlug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_slug", "player_slug requis")
	}
	provider, err := h.newProvider(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	diag, err := provider.GetProgressionDiag(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "diag_error", err.Error())
	}
	return &diagProgressionOutput{Body: diag}, nil
}
