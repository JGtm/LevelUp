// Package handlers — commendation_totals.go : page Totaux des commendations NATIVES
// (Halo 5, AXE B). Le total à vie d'une commendation = le dernier progress absolu du
// joueur, groupé par catégorie.
//
//	GET /api/v1/players/{player_slug}/commendations/totals → NativeCommendationsTotalsResponse
//
// Title-agnostic : le service est construit avec un loader type-asserté depuis
// l'adapter du titre (seul Halo 5 l'implémente). Un titre sans commendations natives
// renvoie une réponse VIDE (pas une erreur) — pas de gating par slug.
package handlers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/api/humacore"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// CommendationTotalsHandler gère la page Totaux des commendations natives.
type CommendationTotalsHandler struct {
	newSvc ContextFactory[port.CommendationTotalsService]
}

// NewCommendationTotalsHandler crée un CommendationTotalsHandler.
func NewCommendationTotalsHandler(newSvc ContextFactory[port.CommendationTotalsService]) *CommendationTotalsHandler {
	return &CommendationTotalsHandler{newSvc: newSvc}
}

// Mount enregistre la route via Huma sur le sous-routeur chi (préfixe
// /players/{player_slug} + middleware ownership/title hérités).
func (h *CommendationTotalsHandler) Mount(r chi.Router, opts ...humacore.MountOption) {
	api := humacore.NewAPI(r, opts...)
	huma.Get(api, "/commendations/totals", h.handleGetTotals)
}

type commendationTotalsInput struct {
	PlayerSlug string `path:"player_slug"`
}

type commendationTotalsOutput struct {
	Body *domain.NativeCommendationsTotalsResponse
}

// handleGetTotals retourne les totaux à vie des commendations natives du joueur.
// GET /api/v1/players/{player_slug}/commendations/totals
func (h *CommendationTotalsHandler) handleGetTotals(ctx context.Context, in *commendationTotalsInput) (*commendationTotalsOutput, error) {
	svc, _, _, err := h.newSvc(ctx, in.PlayerSlug)
	if err != nil {
		return nil, humacore.NewError(http.StatusNotFound, "player_not_found", err.Error())
	}
	resp, err := svc.GetTotals(ctx)
	if err != nil {
		return nil, humacore.NewError(http.StatusInternalServerError, "commendation_totals_error", err.Error())
	}
	return &commendationTotalsOutput{Body: resp}, nil
}
