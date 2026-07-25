// Package handlers — admin_title_diagnostic.go : GET /admin/titles/{slug}/diagnostic
// (PMT-14 volet A). Rapport de santé read-only d'un titre (config TOML + réalité DB).
// Monté sous le groupe /admin (RequireAuth + RequireAdmin).
//
// MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r) sur le sous-routeur
// /admin (middleware RequireAuth/RequireAdmin/NoStore hérités) et enregistre le
// GET via huma.Get. Logique métier inchangée (titleDiagnoser injecté), seul le
// wrapping HTTP change. Le chemin relatif est identique à la route chi d'origine.
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

// Mount enregistre la route via Huma sur le sous-routeur chi /admin (middleware
// RequireAuth/RequireAdmin/NoStore hérités).
func (h *AdminTitleDiagnosticHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/titles/{slug}/diagnostic", h.handleGet, humacore.Op("getAdminTitleDiagnostic", "Diagnostic santé d'un titre : présence des mappings TOML + réalité DB (auth admin requis)", "admin"))
}

// adminTitleDiagnosticInput : path param {slug}.
type adminTitleDiagnosticInput struct {
	Slug string `path:"slug"`
}

// adminTitleDiagnosticOutput : rapport de diagnostic (200).
type adminTitleDiagnosticOutput struct {
	Body *domain.TitleDiagnostic
}

// handleGet répond GET /admin/titles/{slug}/diagnostic.
func (h *AdminTitleDiagnosticHandler) handleGet(ctx context.Context, in *adminTitleDiagnosticInput) (*adminTitleDiagnosticOutput, error) {
	if in.Slug == "" {
		return nil, humacore.NewError(http.StatusBadRequest, "missing_slug", "title slug requis")
	}
	report, err := h.svc.Diagnose(ctx, in.Slug)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "diagnostic_failed", err.Error())
	}
	return &adminTitleDiagnosticOutput{Body: report}, nil
}
