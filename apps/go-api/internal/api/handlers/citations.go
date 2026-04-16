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

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/service"
)

// CitationsHandler gère les endpoints des pages Citations et Commendations.
type CitationsHandler struct {
	cfg *config.AppConfig
}

// NewCitationsHandler crée un CitationsHandler.
func NewCitationsHandler(cfg *config.AppConfig) *CitationsHandler {
	return &CitationsHandler{cfg: cfg}
}

// GetCitations retourne la page Citations (accomplissements personnels).
// POST /api/v1/players/{player_slug}/pages/citations
// Body (optionnel) : { "category": "..." }
func (h *CitationsHandler) GetCitations(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.CitationsPageRequest
	// Body optionnel : décoder uniquement si présent.
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
			return
		}
	}

	repo := duckdb.NewCitationsRepo(pdb)
	svc := service.NewCitationsService(repo)

	page, err := svc.GetCitationsPage(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "citations_page_error", err.Error())
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
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.CommendationsPageRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
			return
		}
	}

	repo := duckdb.NewCitationsRepo(pdb)
	svc := service.NewCitationsService(repo)

	page, err := svc.GetCommendationsPage(r.Context(), pdb.XUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "commendations_page_error", err.Error())
		return
	}

	// Filtrage par catégorie si demandé.
	if req.Category != "" {
		page = filterCommendationsByCategory(page, req.Category)
	}

	writeJSON(w, http.StatusOK, page)
}

// resolvePlayer traduit le slug URL en PlayerDB.
func (h *CitationsHandler) resolvePlayer(r *http.Request) (*duckdb.PlayerDB, error) {
	slug := chi.URLParam(r, "player_slug")
	return config.ResolvePlayer(r.Context(), h.cfg, slug)
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
