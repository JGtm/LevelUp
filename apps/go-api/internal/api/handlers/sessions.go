// Package handlers — SessionsHandler : endpoint de calcul des sessions.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// (préfixe /players/{player_slug} + middleware ownership/title hérités, lit
// {player_slug} parent) et enregistre le GET via huma.Get. Logique métier
// inchangée (SessionsService), seul le wrapping HTTP change.
package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// SessionsHandler gère les requêtes de calcul des sessions.
type SessionsHandler struct {
	newSvc ServiceFactory[port.SessionsService]
}

// NewSessionsHandler crée un SessionsHandler.
func NewSessionsHandler(newSvc ServiceFactory[port.SessionsService]) *SessionsHandler {
	return &SessionsHandler{newSvc: newSvc}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *SessionsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/pages/sessions", h.handleGetSessions, humacore.Op("getSessions", "Sessions (Go-only — à réconcilier avec /pages/timeseries FastAPI au Sprint 32)", "timeseries"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// sessionsInput : {player_slug} parent + query params optionnels (tous tolérés
// vides ou invalides — les valeurs par défaut raisonnables sont conservées,
// contrat identique à l'ancien parse manuel).
type sessionsInput struct {
	PlayerSlug  string `path:"player_slug"`
	GapMinutes  string `query:"gap_minutes"`
	Mode        string `query:"mode"`
	SplitRanked string `query:"split_ranked"`
}

type sessionsOutput struct{ Body domain.SessionsResponse }

// ─── Endpoints ───────────────────────────────────────────────────────────────

// handleGetSessions traite GET /api/v1/players/{player_slug}/pages/sessions.
func (h *SessionsHandler) handleGetSessions(ctx context.Context, in *sessionsInput) (*sessionsOutput, error) {
	svc, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusBadRequest, "player_not_found", "joueur introuvable")
	}

	// Options depuis query params (optionnels — valeurs par défaut raisonnables).
	opts := analysis.DefaultSessionOptions()
	if in.GapMinutes != "" {
		var gap int
		if _, parseErr := fmt.Sscanf(in.GapMinutes, "%d", &gap); parseErr == nil && gap > 0 {
			opts.GapMinutes = gap
		}
	}
	if in.Mode != "" {
		opts.Mode = domain.SessionComputeMode(in.Mode)
	}
	if in.SplitRanked == jsonBoolTrueStr {
		opts.SplitOnRankedChange = true
	}

	resp, svcErr := svc.GetSessions(ctx, opts)
	if svcErr != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "sessions_error", "erreur calcul sessions")
	}

	return &sessionsOutput{Body: resp}, nil
}
