// Package handlers — media.go : handler HTTP pour la galerie médias.
//
// Endpoints :
//   GET /api/v1/players/{player_slug}/pages/media[?page=N]   → MediaPageResponse
package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/service"
)

// MediaHandler gère les endpoints de la galerie médias.
type MediaHandler struct {
	cfg *config.AppConfig
}

// NewMediaHandler crée un MediaHandler.
func NewMediaHandler(cfg *config.AppConfig) *MediaHandler {
	return &MediaHandler{cfg: cfg}
}

// GetMediaLibrary retourne la page paginée de la galerie médias.
// GET /api/v1/players/{player_slug}/pages/media[?page=N]
func (h *MediaHandler) GetMediaLibrary(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}

	repo := duckdb.NewMediaRepo(pdb)
	svc := service.NewMediaService(repo)

	resp, err := svc.GetMediaPage(r.Context(), page)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "media_page_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// resolvePlayer traduit le slug URL en PlayerDB.
func (h *MediaHandler) resolvePlayer(r *http.Request) (*duckdb.PlayerDB, error) {
	slug := chi.URLParam(r, "player_slug")
	return config.ResolvePlayer(r.Context(), h.cfg, slug)
}
