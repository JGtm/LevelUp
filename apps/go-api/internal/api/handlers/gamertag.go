// Package handlers — GamertagHandler : GET /api/v1/directory/gamertags/search.
package handlers

import (
	"net/http"
	"strings"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// GamertagHandler gère GET /api/v1/directory/gamertags/search?q=.
type GamertagHandler struct {
	svc port.GamertagSearchService
}

// NewGamertagHandler crée un GamertagHandler.
func NewGamertagHandler(svc port.GamertagSearchService) *GamertagHandler {
	return &GamertagHandler{svc: svc}
}

// Search cherche les gamertags correspondant à la query ?q=.
func (h *GamertagHandler) Search(w http.ResponseWriter, r *http.Request) {
	// Sprint 49 : route inconditionnelle — 503 si shared DB absente.
	if h.svc == nil {
		writeError(w, http.StatusServiceUnavailable, "shared_db_unavailable", "gamertag search requires shared database")
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusOK, domain.GamertagSearchResponse{Query: q, Items: []domain.GamertagSearchResult{}})
		return
	}

	items, err := h.svc.Search(r.Context(), q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gamertag_search_error", err.Error())
		return
	}
	if items == nil {
		items = []domain.GamertagSearchResult{}
	}
	writeJSON(w, http.StatusOK, domain.GamertagSearchResponse{Query: q, Items: items})
}
