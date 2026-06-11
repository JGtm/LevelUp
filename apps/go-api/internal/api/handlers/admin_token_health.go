// Package handlers — admin_token_health.go : dashboard admin « Santé des tokens »
// (MSAL / XSTS / Refresh par joueur).
//
// GET /admin/token-health?title={slug} → statuts token par joueur suivi, lus
// depuis le MultiUserTokenStore (ADR 0023) SANS refresh réseau.
package handlers

import (
	"context"
	"log/slog"
	"net/http"

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

// Get retourne la santé des tokens auth par joueur.
// GET /admin/token-health?title={slug} (défaut : titre par défaut).
func (h *AdminTokenHealthHandler) Get(w http.ResponseWriter, r *http.Request) {
	titleSlug := r.URL.Query().Get("title")
	if titleSlug == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	resp, err := h.run(r.Context(), titleSlug)
	if err != nil {
		slog.ErrorContext(r.Context(), "admin_token_health: run failed", "title", titleSlug, "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "token_health_error",
			"Impossible de lire la santé des tokens.")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
