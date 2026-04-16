// Package handlers — TeammatesHandler : POST /pages/teammates.
//
// Sprint 33 — contrat API Lot 4.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// TeammatesHandler gère POST /pages/teammates.
type TeammatesHandler struct {
	newSvc ContextFactory[port.TeammatesService]
}

// NewTeammatesHandler crée un TeammatesHandler.
func NewTeammatesHandler(newSvc ContextFactory[port.TeammatesService]) *TeammatesHandler {
	return &TeammatesHandler{newSvc: newSvc}
}

// GetPage traite POST /api/v1/players/{player_slug}/pages/teammates.
func (h *TeammatesHandler) GetPage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, xuid, _, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.TeammatesQueryRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "corps JSON invalide")
		return
	}

	resp, svcErr := svc.GetPage(r.Context(), xuid, req)
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, "teammates_error", svcErr.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
