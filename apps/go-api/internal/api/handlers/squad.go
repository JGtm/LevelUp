// Package handlers — squad.go : handlers HTTP pour les pages Escouade et Synthèse.
//
// Endpoints :
//
//	GET  /api/v1/players/{player_slug}/pages/squad             → SquadPageResponse
//	GET  /api/v1/players/{player_slug}/pages/squad?teammate=xuid
//	POST /api/v1/players/{player_slug}/pages/synthesis         → SynthesisPageResponse
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// SquadHandler gère les endpoints de la page Escouade et Synthèse.
type SquadHandler struct {
	newSvc ContextFactory[port.SquadService]
}

// NewSquadHandler crée un SquadHandler.
func NewSquadHandler(newSvc ContextFactory[port.SquadService]) *SquadHandler {
	return &SquadHandler{newSvc: newSvc}
}

// GetSquadPage retourne la page Escouade.
// GET /api/v1/players/{player_slug}/pages/squad[?teammate=xuid]
func (h *SquadHandler) GetSquadPage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, xuid, gamertag, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	teammateXUID := r.URL.Query().Get("teammate")

	page, err := svc.GetSquadPage(r.Context(), xuid, gamertag, teammateXUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "squad_page_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, page)
}

// GetSynthesisPage retourne la page Synthèse (heatmap + top semaines).
// POST /api/v1/players/{player_slug}/pages/synthesis
// Body (optionnel) : { "filters": {...} }
func (h *SquadHandler) GetSynthesisPage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, xuid, _, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	// Body optionnel — filters déclarés mais utilisés en Sprint 33.
	var req domain.SynthesisPageRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
			return
		}
	}
	_ = req // filtres Sprint 33

	page, err := svc.GetSynthesisPage(r.Context(), xuid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "synthesis_page_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, page)
}
