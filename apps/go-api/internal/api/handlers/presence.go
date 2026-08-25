// Package handlers — presence.go : GET /api/v1/presence.
//
// Qui est en jeu en ce moment (joueurs suivis accessibles à l'utilisateur) et
// combien d'amis le sont. Sert la manette du sélecteur de joueur et son badge
// « N amis en jeu » dans la navigation.
//
// Monté sous RequireAuth + NoStore — PAS RequireAdmin : c'est une information
// de confort pour l'utilisateur connecté, sur SES joueurs (ADR 0029).
// /watcher/status, lui, reste admin : il expose l'état interne du daemon.
//
// Handler mince par construction : tout le calcul vit dans service.PresenceService.
package handlers

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/api/middleware"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/service"
)

// PresenceHandler gère GET /api/v1/presence.
type PresenceHandler struct {
	svc *service.PresenceService
}

// NewPresenceHandler crée un PresenceHandler.
func NewPresenceHandler(svc *service.PresenceService) *PresenceHandler {
	return &PresenceHandler{svc: svc}
}

// Mount enregistre GET /presence via Huma sur le sous-routeur chi /api/v1.
func (h *PresenceHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/presence", h.handleGetPresence,
		humacore.Op("getPresence", "Présence en jeu des joueurs suivis et des amis", "bootstrap"))
}

// presenceOutput : 200 avec le PresenceSnapshot.
type presenceOutput struct{ Body *domain.PresenceSnapshot }

// handleGetPresence retourne l'état de présence courant.
//
// Toujours 200 : une source indisponible (watcher éteint, Xbox injoignable) rend
// une réponse vide, jamais une erreur — le shell interroge cet endpoint toutes
// les 30 s et n'a rien à faire d'un échec.
func (h *PresenceHandler) handleGetPresence(ctx context.Context, _ *struct{}) (*presenceOutput, error) {
	return &presenceOutput{Body: h.svc.GetSnapshot(ctx, middleware.GetSession(ctx))}, nil
}
