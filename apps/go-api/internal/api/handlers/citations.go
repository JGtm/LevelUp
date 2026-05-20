// Package handlers — citations.go : handlers HTTP pour les pages Citations et Commendations.
//
// Endpoints :
//
//	POST /api/v1/players/{player_slug}/pages/citations        → CitationsPageResponse
//	POST /api/v1/players/{player_slug}/pages/commendations    → CommendationsPageResponse
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// CitationsHandler gère les endpoints des pages Citations et Commendations.
type CitationsHandler struct {
	newSvc ContextFactory[port.CitationsService]
}

// NewCitationsHandler crée un CitationsHandler.
func NewCitationsHandler(newSvc ContextFactory[port.CitationsService]) *CitationsHandler {
	return &CitationsHandler{newSvc: newSvc}
}

// GetCitations retourne la page Citations (accomplissements personnels).
// POST /api/v1/players/{player_slug}/pages/citations
// Body (optionnel) : { "category": "..." }
func (h *CitationsHandler) GetCitations(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, _, _, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.CitationsPageRequest
	// Body optionnel : décoder uniquement si présent.
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
			return
		}
	}

	page, err := svc.GetCitationsPage(r.Context())
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "citations_page_error", err.Error())
		return
	}

	// Filtrage par catégorie si demandé.
	if req.Category != "" {
		page = filterCitationsByCategory(page, req.Category)
	}

	writeJSON(w, http.StatusOK, page)
}

// GetCommendations retourne la page Commendations (médailles gagnées).
// POST /api/v1/players/{player_slug}/pages/commendations
// Body (optionnel) : { "category": "..." }
func (h *CitationsHandler) GetCommendations(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, xuid, _, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.CommendationsPageRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(r.Context(), w, http.StatusBadRequest, "invalid_body", err.Error())
			return
		}
	}

	page, err := svc.GetCommendationsPage(r.Context(), xuid)
	if err != nil {
		writeError(r.Context(), w, http.StatusInternalServerError, "commendations_page_error", err.Error())
		return
	}

	// Filtrage par catégorie si demandé.
	if req.Category != "" {
		page = filterCommendationsByCategory(page, req.Category)
	}

	writeJSON(w, http.StatusOK, page)
}

// ---------------------------------------------------------------------------
// Helpers de filtrage
// ---------------------------------------------------------------------------

// filterCitationsByCategory filtre les items Citations sur une catégorie.
func filterCitationsByCategory(page *domain.CitationsPageResponse, category string) *domain.CitationsPageResponse {
	filtered := make([]domain.CitationItem, 0, len(page.Citations))
	for _, c := range page.Citations {
		if c.Category == category {
			filtered = append(filtered, c)
		}
	}
	return &domain.CitationsPageResponse{
		Citations:  filtered,
		Categories: page.Categories,
		TotalCount: len(filtered),
	}
}

// filterCommendationsByCategory filtre les Commendations sur une catégorie.
func filterCommendationsByCategory(page *domain.CommendationsPageResponse, category string) *domain.CommendationsPageResponse {
	filtered := make([]domain.CommendationCategory, 0)
	total := 0
	for _, cat := range page.Categories {
		if cat.Category == category {
			filtered = append(filtered, cat)
			total += cat.Total
		}
	}
	return &domain.CommendationsPageResponse{
		Categories: filtered,
		TotalCount: total,
	}
}
