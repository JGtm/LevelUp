// Package handlers — leaderboard.go : handler HTTP pour le classement CSR.
//
// Endpoint :
//
//	GET /api/v1/players/{player_slug}/pages/leaderboard → LeaderboardResponse
//
// Sprint 54 E.
package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// LeaderboardHandler gère l'endpoint du classement CSR.
type LeaderboardHandler struct {
	newSvc ContextFactory[port.LeaderboardService]
}

// NewLeaderboardHandler crée un LeaderboardHandler.
func NewLeaderboardHandler(newSvc ContextFactory[port.LeaderboardService]) *LeaderboardHandler {
	return &LeaderboardHandler{newSvc: newSvc}
}

// GetLeaderboardPage retourne le classement CSR.
// GET /api/v1/players/{player_slug}/pages/leaderboard?season=...&playlist=...
func (h *LeaderboardHandler) GetLeaderboardPage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, _, _, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	req := domain.LeaderboardRequest{
		Season:    r.URL.Query().Get("season"),
		Playlist:  r.URL.Query().Get("playlist"),
		TitleSlug: r.URL.Query().Get("title_slug"),
	}

	resp, err := svc.GetPage(r.Context(), req)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "leaderboard_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
