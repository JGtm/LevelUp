// Package handlers — admin_invariants.go : dashboard admin « Intégrité des
// données » (Phase 4 du plan .ai/PLAN_SYNC_INVARIANTS_GATE.md).
//
// GET /admin/invariants?title={slug} → exécute les invariants de données
// déclarés (internal/sync/invariants) pour chaque joueur suivi et retourne
// les violations par joueur. Lectures seules, best-effort par joueur.
package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"levelup/go-api/internal/domain"
	titlePkg "levelup/go-api/internal/domain/title"
)

// InvariantsRunner exécute les invariants pour un titre (implémenté par
// ServiceRegistry.RunDataInvariants — injecté pour éviter le cycle d'import).
type InvariantsRunner func(ctx context.Context, titleSlug string) (domain.AdminInvariantsResponse, error)

// AdminInvariantsHandler sert le dashboard d'intégrité des données.
type AdminInvariantsHandler struct {
	run InvariantsRunner
}

// NewAdminInvariantsHandler construit le handler.
func NewAdminInvariantsHandler(run InvariantsRunner) *AdminInvariantsHandler {
	return &AdminInvariantsHandler{run: run}
}

// Get retourne les rapports d'invariants par joueur.
// GET /admin/invariants?title={slug} (défaut : titre par défaut).
func (h *AdminInvariantsHandler) Get(w http.ResponseWriter, r *http.Request) {
	titleSlug := r.URL.Query().Get("title")
	if titleSlug == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	resp, err := h.run(r.Context(), titleSlug)
	if err != nil {
		slog.ErrorContext(r.Context(), "admin_invariants: run failed", "title", titleSlug, "err", err)
		writeError(r.Context(), w, http.StatusInternalServerError, "invariants_error",
			"Impossible d'exécuter les invariants de données.")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
