// Package handlers — MatchViewHandler : GET .../matches/{match_id}.
package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/port"
)

// MatchViewHandler gère GET /players/{player_slug}/matches/{match_id}.
type MatchViewHandler struct {
	newSvc ServiceFactory[port.MatchViewService]
}

// NewMatchViewHandler crée un MatchViewHandler.
func NewMatchViewHandler(newSvc ServiceFactory[port.MatchViewService]) *MatchViewHandler {
	return &MatchViewHandler{newSvc: newSvc}
}

// GetMatchView retourne la vue détaillée d'un match pour un joueur.
// GET /api/v1/players/{player_slug}/matches/{match_id}
func (h *MatchViewHandler) GetMatchView(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	matchID := chi.URLParam(r, "match_id")
	if matchID == "" {
		writeError(w, http.StatusBadRequest, "missing_match_id", "match_id est requis")
		return
	}

	resp, err := svc.GetMatchView(r.Context(), matchID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "match_view_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
