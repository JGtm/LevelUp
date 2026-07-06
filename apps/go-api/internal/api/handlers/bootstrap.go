// Package handlers — handlers GET /api/v1/bootstrap et GET /api/v1/players.
//
// BootstrapHandler MIGRÉ vers Huma (Phase 3b) : Mount crée humacore.NewAPI(r)
// sur le sous-routeur /api/v1 et enregistre le GET /bootstrap via huma.Get.
// Logique métier inchangée (BootstrapService), seul le wrapping HTTP change.
package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/service"
)

// BootstrapHandler gère GET /api/v1/bootstrap.
type BootstrapHandler struct {
	svc *service.BootstrapService
}

// NewBootstrapHandler crée un BootstrapHandler.
func NewBootstrapHandler(svc *service.BootstrapService) *BootstrapHandler {
	return &BootstrapHandler{svc: svc}
}

// Mount enregistre GET /bootstrap via Huma sur le sous-routeur chi /api/v1
// (chemin relatif identique à l'ancien r.Get("/bootstrap", ...)).
func (h *BootstrapHandler) Mount(r chi.Router) {
	api := humacore.NewAPI(r)
	huma.Get(api, "/bootstrap", h.handleBootstrap)
}

// bootstrapOutput : 200 avec le BootstrapResponse complet (contrat préservé —
// même corps JSON que l'ancien writeJSON(resp)).
type bootstrapOutput struct{ Body *domain.BootstrapResponse }

// handleBootstrap retourne le BootstrapResponse au shell React.
func (h *BootstrapHandler) handleBootstrap(ctx context.Context, _ *struct{}) (*bootstrapOutput, error) {
	sess := middleware.GetSession(ctx)
	resp, err := h.svc.Build(ctx, sess)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "bootstrap_error", err.Error())
	}
	return &bootstrapOutput{Body: resp}, nil
}

// PlayersHandler gère GET /api/v1/players.
type PlayersHandler struct {
	svc *service.BootstrapService
}

// NewPlayersHandler crée un PlayersHandler.
func NewPlayersHandler(svc *service.BootstrapService) *PlayersHandler {
	return &PlayersHandler{svc: svc}
}

// Mount enregistre GET /players via Huma sur le sous-routeur chi /api/v1
// (chemin relatif identique à l'ancien r.Get("/players", ...)).
func (h *PlayersHandler) Mount(r chi.Router) {
	api := humacore.NewAPI(r)
	huma.Get(api, "/players", h.handlePlayers)
}

// playersOutput : 200 avec la liste des joueurs disponibles (contrat préservé —
// même corps JSON que l'ancien writeJSON(resp)).
type playersOutput struct{ Body *domain.PlayersListResponse }

// handlePlayers retourne la liste des joueurs disponibles, restreinte aux profils
// possédés par l'utilisateur courant (S4, lot S — même filtrage que /bootstrap).
func (h *PlayersHandler) handlePlayers(ctx context.Context, _ *struct{}) (*playersOutput, error) {
	sess := middleware.GetSession(ctx)
	resp, err := h.svc.BuildPlayersList(ctx, sess)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "players_error", err.Error())
	}
	return &playersOutput{Body: resp}, nil
}
