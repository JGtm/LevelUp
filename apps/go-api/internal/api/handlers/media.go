// Package handlers — media.go : handler HTTP pour la galerie médias.
//
// Endpoints :
//
//	POST /api/v1/players/{player_slug}/pages/media   → MediaPageResponse
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
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
// POST /api/v1/players/{player_slug}/pages/media
// Body (optionnel) : { "page": 1, "page_size": 24, "kind": "clip" }
func (h *MediaHandler) GetMediaLibrary(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	// Valeurs par défaut.
	req := domain.MediaPageRequest{Page: 1}
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
			return
		}
	}
	if req.Page < 1 {
		req.Page = 1
	}

	repo := duckdb.NewMediaRepo(pdb)
	svc := service.NewMediaService(repo)

	resp, err := svc.GetMediaPage(r.Context(), req.Page)
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
