// Package handlers — LastMatchHandler : POST /pages/last-match/resolve.
//
// Sprint 33 — contrat API Lot 5.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// LastMatchHandler gère POST /pages/last-match/resolve.
type LastMatchHandler struct {
	newSvc ServiceFactory[port.LastMatchService]
}

// NewLastMatchHandler crée un LastMatchHandler.
func NewLastMatchHandler(newSvc ServiceFactory[port.LastMatchService]) *LastMatchHandler {
	return &LastMatchHandler{newSvc: newSvc}
}

// Resolve traite POST /api/v1/players/{player_slug}/pages/last-match/resolve.
func (h *LastMatchHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.LastMatchResolveRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "corps JSON invalide")
		return
	}

	resp, svcErr := svc.Resolve(r.Context(), req)
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, "last_match_error", svcErr.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
