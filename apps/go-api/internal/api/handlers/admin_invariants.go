// Package handlers — admin_invariants.go : dashboard admin « Intégrité des
// données » (Phase 4 du plan .ai/PLAN_SYNC_INVARIANTS_GATE.md).
//
// GET /admin/invariants?title={slug} → exécute les invariants de données
// déclarés (internal/sync/invariants) pour chaque joueur suivi et retourne
// les violations par joueur. Lectures seules, best-effort par joueur.
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// /admin (middleware RequireAuth/RequireAdmin + NoStore hérités) et enregistre
// la route via huma.Get. Logique métier inchangée (InvariantsRunner), seul le
// wrapping HTTP change. Le chemin relatif est identique à la route chi d'origine
// (montée sous /admin par server.go).
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

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe /admin +
// middleware RequireAuth/RequireAdmin + NoStore hérités).
func (h *AdminInvariantsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/invariants", h.handleGet, humacore.Op("getAdminInvariants", "Intégrité des données — invariants du pipeline sync par joueur (auth admin requis)", "admin"))
}

// adminInvariantsInput : ?title= optionnel (défaut : titre par défaut).
type adminInvariantsInput struct {
	Title string `query:"title"`
}

type adminInvariantsOutput struct {
	Body domain.AdminInvariantsResponse
}

// handleGet retourne les rapports d'invariants par joueur.
// GET /admin/invariants?title={slug} (défaut : titre par défaut).
func (h *AdminInvariantsHandler) handleGet(ctx context.Context, in *adminInvariantsInput) (*adminInvariantsOutput, error) {
	titleSlug := in.Title
	if titleSlug == "" {
		titleSlug = titlePkg.DefaultSlug
	}
	resp, err := h.run(ctx, titleSlug)
	if err != nil {
		slog.ErrorContext(ctx, "admin_invariants: run failed", "title", titleSlug, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "invariants_error",
			"Impossible d'exécuter les invariants de données.")
	}
	return &adminInvariantsOutput{Body: resp}, nil
}
