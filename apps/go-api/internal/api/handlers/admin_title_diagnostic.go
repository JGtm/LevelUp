// Package handlers — admin_title_diagnostic.go : GET /admin/titles/{slug}/diagnostic
// (PMT-14 volet A). Rapport de santé read-only d'un titre (config TOML + réalité DB).
// Monté sous le groupe /admin (RequireAuth + RequireAdmin).
package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
)

// titleDiagnoser produit le rapport de diagnostic d'un titre. Implémenté par
// *service.TitleDiagnosticService.
type titleDiagnoser interface {
	Diagnose(ctx context.Context, titleSlug string) (*domain.TitleDiagnostic, error)
}

// AdminTitleDiagnosticHandler sert le diagnostic de titre côté admin.
type AdminTitleDiagnosticHandler struct {
	svc    titleDiagnoser
	logger *slog.Logger
}

// NewAdminTitleDiagnosticHandler crée le handler.
func NewAdminTitleDiagnosticHandler(svc titleDiagnoser, logger *slog.Logger) *AdminTitleDiagnosticHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &AdminTitleDiagnosticHandler{svc: svc, logger: logger}
}

// Get répond GET /admin/titles/{slug}/diagnostic.
func (h *AdminTitleDiagnosticHandler) Get(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(r.Context(), w, http.StatusBadRequest, "missing_slug", "title slug requis")
		return
	}
	report, err := h.svc.Diagnose(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "diagnostic_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}
