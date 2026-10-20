// Package handlers — admin_identities.go : l'annuaire des joueurs vu par
// l'administrateur (ADR 0035 D7).
//
// GET /admin/identities → une ligne par identité (xuid), avec ce que chacun des
// quatre registres en sait — compte, profils, credentials, suivi live — et ses
// anomalies typées.
//
// C'est la vue qui aurait montré le compte du 2026-07-23 le soir même : jusqu'à
// elle, la page Gestion listait les COMPTES, les pages de monitoring listaient
// les PROFILS, et un compte sans profil n'apparaissait nulle part.
//
// Aucune logique ici : la composition des registres vit dans
// `internal/service/playerdirectory` derrière port.PlayerDirectory.
package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// AdminIdentitiesHandler sert l'annuaire des joueurs.
type AdminIdentitiesHandler struct {
	directory port.PlayerDirectory
}

// NewAdminIdentitiesHandler construit le handler.
func NewAdminIdentitiesHandler(directory port.PlayerDirectory) *AdminIdentitiesHandler {
	return &AdminIdentitiesHandler{directory: directory}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe /admin +
// middleware RequireAuth/RequireAdmin/NoStore hérités, comme /admin/monitoring).
func (h *AdminIdentitiesHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/identities", h.handleGet,
		humacore.Op("getAdminIdentities",
			"Annuaire des joueurs : comptes, profils, credentials, suivi live et anomalies (auth admin requis)",
			"admin"))
}

// ─── Inputs/Outputs Huma ─────────────────────────────────────────────────────

type adminIdentitiesOutput struct {
	Body domain.AdminIdentitiesResponse
}

// ─── Endpoint ────────────────────────────────────────────────────────────────

// handleGet retourne l'annuaire complet.
// GET /admin/identities
func (h *AdminIdentitiesHandler) handleGet(ctx context.Context, _ *struct{}) (*adminIdentitiesOutput, error) {
	if h.directory == nil {
		slog.ErrorContext(ctx, "admin_identities: annuaire non câblé")
		return nil, humacore.NewError(http.StatusServiceUnavailable, "directory_unavailable",
			"Annuaire indisponible.")
	}
	resp, err := h.directory.List(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "admin_identities: lecture de l'annuaire impossible", "err", err)
		return nil, humacore.NewError(http.StatusInternalServerError, "identities_error",
			"Lecture de l'annuaire impossible.")
	}
	return &adminIdentitiesOutput{Body: resp}, nil
}
