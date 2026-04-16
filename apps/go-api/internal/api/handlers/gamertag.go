// Package handlers — GamertagHandler : GET /api/v1/directory/gamertags/search.
package handlers

import (
	"net/http"
	"strings"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
)

// GamertagHandler gère GET /api/v1/directory/gamertags/search?q=.
type GamertagHandler struct {
	cfg *config.AppConfig
}

// NewGamertagHandler crée un GamertagHandler.
func NewGamertagHandler(cfg *config.AppConfig) *GamertagHandler {
	return &GamertagHandler{cfg: cfg}
}

// Search cherche les gamertags correspondant à la query ?q=.
func (h *GamertagHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusOK, domain.GamertagSearchResponse{Query: q, Items: []domain.GamertagSearchResult{}})
		return
	}

	sharedPath := config.SharedDBPath(h.cfg)
	db, err := duckdb.OpenReadOnly(sharedPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db_error", err.Error())
		return
	}
	defer db.Close()

	repo := duckdb.NewGamertagRepo(db)
	items, err := repo.Search(r.Context(), q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gamertag_search_error", err.Error())
		return
	}
	if items == nil {
		items = []domain.GamertagSearchResult{}
	}
	writeJSON(w, http.StatusOK, domain.GamertagSearchResponse{Query: q, Items: items})
}
