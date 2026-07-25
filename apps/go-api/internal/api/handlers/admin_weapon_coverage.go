// Package handlers — admin_weapon_coverage.go : endpoint monitoring « couverture
// de résolution d'arme » (GET /admin/monitoring/weapon-coverage?title=). Détecte
// les weapon_id non mappés (registre / weapon_labels / non résolus). Runner injecté.
package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
)

// WeaponCoverageRunner retourne la couverture de résolution d'arme d'un titre
// (implémenté par ServiceRegistry.WeaponCoverage — injecté pour éviter le cycle).
type WeaponCoverageRunner func(ctx context.Context, titleSlug string) (domain.AdminWeaponCoverage, error)

// AdminWeaponCoverageHandler sert l'endpoint de couverture.
type AdminWeaponCoverageHandler struct {
	coverage WeaponCoverageRunner
}

// NewAdminWeaponCoverageHandler construit le handler.
func NewAdminWeaponCoverageHandler(coverage WeaponCoverageRunner) *AdminWeaponCoverageHandler {
	return &AdminWeaponCoverageHandler{coverage: coverage}
}

// Mount enregistre la route Huma sur le sous-routeur /admin.
func (h *AdminWeaponCoverageHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/monitoring/weapon-coverage", h.handleGet, humacore.Op(
		"getAdminMonitoringWeaponCoverage",
		"Dashboard monitoring — couverture de résolution d'arme (registre vs weapon_labels vs non résolu) par titre (auth admin requis)",
		"admin"))
}

type adminWeaponCoverageOutput struct {
	Body domain.AdminWeaponCoverage
}

// handleGet — GET /admin/monitoring/weapon-coverage?title={slug}.
func (h *AdminWeaponCoverageHandler) handleGet(ctx context.Context, in *titleInput) (*adminWeaponCoverageOutput, error) {
	titleSlug := titleOrDefaultSlug(in.Title)
	resp, err := h.coverage(ctx, titleSlug)
	if err != nil {
		slog.ErrorContext(ctx, "admin_monitoring: weapon coverage failed", "title", titleSlug, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "weapon_coverage_error",
			"Impossible de calculer la couverture de résolution d'arme.")
	}
	return &adminWeaponCoverageOutput{Body: resp}, nil
}
