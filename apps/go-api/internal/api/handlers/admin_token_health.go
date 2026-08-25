// Package handlers — admin_token_health.go : dashboard admin « Santé des tokens »
// (Accès / XSTS / Refresh par joueur).
//
// GET /admin/token-health?title={slug} → statuts token par joueur suivi, lus
// depuis le MultiUserTokenStore (ADR 0023) SANS refresh réseau.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// /admin (middleware RequireAuth/RequireAdmin + NoStore hérités) et enregistre la
// route via huma.Get. Logique métier inchangée (TokenHealthRunner), seul le
// wrapping HTTP change.
package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
)

// TokenHealthRunner retourne la santé des tokens par joueur (implémenté par
// ServiceRegistry.TokenHealth — injecté pour éviter le cycle d'import).
type TokenHealthRunner func(ctx context.Context, titleSlug string) (domain.TokenHealthResponse, error)

// AdminTokenHealthHandler sert le dashboard admin « Santé des tokens ».
type AdminTokenHealthHandler struct {
	run TokenHealthRunner
}

// NewAdminTokenHealthHandler construit le handler.
func NewAdminTokenHealthHandler(run TokenHealthRunner) *AdminTokenHealthHandler {
	return &AdminTokenHealthHandler{run: run}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe /admin +
// middleware RequireAuth/RequireAdmin/NoStore hérités).
func (h *AdminTokenHealthHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/token-health", h.handleGet, humacore.Op("getAdminTokenHealth", "Santé des tokens auth (Accès / XSTS / Refresh) par joueur (auth admin requis)", "admin"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

// adminTokenHealthInput : ?title= optionnel (vide → titre par défaut).
type adminTokenHealthInput struct {
	Title string `query:"title"`
}

type adminTokenHealthOutput struct {
	Body domain.TokenHealthResponse
}

// ─── Endpoint ────────────────────────────────────────────────────────────────

// handleGet retourne la santé des tokens auth par joueur.
// GET /admin/token-health?title={slug} (défaut : titre par défaut).
func (h *AdminTokenHealthHandler) handleGet(ctx context.Context, in *adminTokenHealthInput) (*adminTokenHealthOutput, error) {
	titleSlug := in.Title
	if titleSlug == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	resp, err := h.run(ctx, titleSlug)
	if err != nil {
		slog.ErrorContext(ctx, "admin_token_health: run failed", "title", titleSlug, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "token_health_error",
			"Impossible de lire la santé des tokens.")
	}
	return &adminTokenHealthOutput{Body: resp}, nil
}
