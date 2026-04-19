// Package handlers — synthesis.go : handler HTTP pour la page Synthèse.
//
// Sprint 55 D1 : extrait de squad.go — SynthesisHandler devient autonome.
//
// Endpoint :
//
//	POST /api/v1/players/{player_slug}/pages/synthesis → SynthesisPageV2Response
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// SynthesisHandler gère l'endpoint de la page Synthèse.
// Sprint 55 D1 : séparé de SquadHandler pour refléter la frontière produit.
type SynthesisHandler struct {
	newSvc ContextFactory[port.SynthesisService]
}

// NewSynthesisHandler crée un SynthesisHandler.
func NewSynthesisHandler(newSvc ContextFactory[port.SynthesisService]) *SynthesisHandler {
	return &SynthesisHandler{newSvc: newSvc}
}

// GetSynthesisPage retourne la page Synthèse avec scope explicite et filtres appliqués.
// POST /api/v1/players/{player_slug}/pages/synthesis
// Body (optionnel) : { "period": "1m", "filters": {...} }
func (h *SynthesisHandler) GetSynthesisPage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, xuid, _, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.SynthesisRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
			return
		}
	}

	page, err := svc.GetSynthesisPage(r.Context(), xuid, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "synthesis_page_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, page)
}
