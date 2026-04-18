// Package handlers — compare.go : handler HTTP pour la comparaison joueur vs joueur.
//
// Endpoint :
//
//	POST /api/v1/players/{player_slug}/pages/compare → CompareResponse
//
// Sprint 54 C.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// CompareHandler gère l'endpoint de comparaison joueur vs joueur.
type CompareHandler struct {
	newSvc ContextFactory[port.CompareService]
}

// NewCompareHandler crée un CompareHandler.
func NewCompareHandler(newSvc ContextFactory[port.CompareService]) *CompareHandler {
	return &CompareHandler{newSvc: newSvc}
}

// PostComparePage compare le joueur local (slug) avec un joueur cible (body).
// POST /api/v1/players/{player_slug}/pages/compare
func (h *CompareHandler) PostComparePage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, _, _, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.CompareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	resp, err := svc.GetPage(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "compare_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
