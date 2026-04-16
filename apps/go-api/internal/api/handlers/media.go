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

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// MediaHandler gère les endpoints de la galerie médias.
type MediaHandler struct {
	newSvc ServiceFactory[port.MediaService]
}

// NewMediaHandler crée un MediaHandler.
func NewMediaHandler(newSvc ServiceFactory[port.MediaService]) *MediaHandler {
	return &MediaHandler{newSvc: newSvc}
}

// GetMediaLibrary retourne la page paginée de la galerie médias.
// POST /api/v1/players/{player_slug}/pages/media
// Body (optionnel) : { "page": 1, "page_size": 24, "kind": "clip" }
func (h *MediaHandler) GetMediaLibrary(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
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

	resp, err := svc.GetMediaPage(r.Context(), req.Page)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "media_page_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
