// Package handlers — admin_appearance_diag.go : endpoint admin du diagnostic
// apparence Spartan ID (volet 2 du plan .ai/PLAN_DIAG_APPARENCE_ADMIN_2026-07.md,
// Lot F).
//
// GET /admin/diag/appearance/{player_slug} → verdict par composant (bannière,
// emblème, backdrop, service tag) d'un joueur suivi, à la demande. ZÉRO logique
// métier ici : le runner (ServiceRegistry.DiagnoseAppearance) porte tout le calcul.
// Montée via Huma sur le sous-routeur /admin (RequireAuth/RequireAdmin + NoStore
// hérités), modèle admin_invariants.go.
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/service"
)

// AppearanceDiagRunner exécute le diagnostic apparence d'un joueur (implémenté par
// ServiceRegistry.DiagnoseAppearance — injecté pour éviter le cycle d'import).
type AppearanceDiagRunner func(ctx context.Context, playerSlug string) (domain.AppearanceDiagnosisResponse, error)

// AdminAppearanceDiagHandler sert le diagnostic apparence Spartan ID.
type AdminAppearanceDiagHandler struct {
	run AppearanceDiagRunner
}

// NewAdminAppearanceDiagHandler construit le handler.
func NewAdminAppearanceDiagHandler(run AppearanceDiagRunner) *AdminAppearanceDiagHandler {
	return &AdminAppearanceDiagHandler{run: run}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe /admin +
// RequireAuth/RequireAdmin + NoStore hérités).
func (h *AdminAppearanceDiagHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/diag/appearance/{player_slug}", h.handleGet, humacore.Op(
		"getAdminDiagAppearance",
		"Diagnostic apparence Spartan ID — verdict par composant (bannière/emblème/backdrop/service tag) d'un joueur suivi, à la demande (auth admin "+
			"requis)",
		"admin"))
}

// appearanceDiagInput : path param {player_slug}.
type appearanceDiagInput struct {
	PlayerSlug string `path:"player_slug"`
}

type appearanceDiagOutput struct {
	Body domain.AppearanceDiagnosisResponse
}

// handleGet retourne le diagnostic des 4 composants du Spartan ID.
// GET /admin/diag/appearance/{player_slug} (titre courant résolu via ctx).
func (h *AdminAppearanceDiagHandler) handleGet(ctx context.Context, in *appearanceDiagInput) (*appearanceDiagOutput, error) {
	resp, err := h.run(ctx, in.PlayerSlug)
	if err != nil {
		if errors.Is(err, service.ErrProfileNotFound) {
			return nil, humacore.NewError(http.StatusNotFound, "player_not_found",
				"Joueur suivi introuvable : "+in.PlayerSlug)
		}
		slog.ErrorContext(ctx, "admin_appearance_diag: échec", "player", in.PlayerSlug, "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "appearance_diag_error",
			"Impossible de diagnostiquer l'apparence.")
	}
	return &appearanceDiagOutput{Body: resp}, nil
}
