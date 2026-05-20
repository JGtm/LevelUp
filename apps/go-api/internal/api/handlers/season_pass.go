// Package handlers — season_pass.go : handler HTTP pour la page Season Pass.
//
// Endpoint :
//
//	GET /api/v1/players/{player_slug}/pages/palmares/season-pass → SeasonPassPageResponse
package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/port"
)

// SeasonPassAuthFactory est une factory qui retourne un SeasonPassService + contexte enrichi.
type SeasonPassAuthFactory func(ctx context.Context, slug string) (svc port.SeasonPassService, enrichedCtx context.Context, err error)

// SeasonPassHandler gère l'endpoint /pages/palmares/season-pass.
type SeasonPassHandler struct {
	newSvc SeasonPassAuthFactory
}

// NewSeasonPassHandler crée un SeasonPassHandler.
func NewSeasonPassHandler(newSvc SeasonPassAuthFactory) *SeasonPassHandler {
	return &SeasonPassHandler{newSvc: newSvc}
}

// GetSeasonPass retourne la page Season Pass complète.
// GET /api/v1/players/{player_slug}/pages/palmares/season-pass
func (h *SeasonPassHandler) GetSeasonPass(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, ctx, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", "joueur introuvable")
		return
	}

	page, err := svc.GetSeasonPassPage(ctx)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "season_pass_error", "erreur chargement page Season Pass")
		return
	}

	writeJSON(w, http.StatusOK, page)
}
