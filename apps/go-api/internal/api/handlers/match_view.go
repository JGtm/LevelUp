// Package handlers — MatchViewHandler : GET .../matches/{match_id}.
package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
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
		var apiErr *domain.APIError
		if errors.As(err, &apiErr) && apiErr.Code == "not_found" {
			writeError(w, http.StatusNotFound, "match_not_found", apiErr.Message)
			return
		}
		if strings.Contains(err.Error(), "no rows") || strings.Contains(err.Error(), "no rows in result set") {
			writeError(w, http.StatusNotFound, "match_not_found", "match introuvable : "+matchID)
			return
		}
		writeError(w, http.StatusInternalServerError, "match_view_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetMatchNeighbors retourne les matchs adjacents pour la navigation prev/next.
// GET /api/v1/players/{player_slug}/matches/{match_id}/neighbors
func (h *MatchViewHandler) GetMatchNeighbors(w http.ResponseWriter, r *http.Request) {
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

	resp, err := svc.GetMatchNeighbors(r.Context(), matchID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "neighbors_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
