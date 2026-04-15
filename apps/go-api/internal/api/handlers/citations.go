// Package handlers — citations.go : handlers HTTP pour les pages Citations et Commendations.
//
// Endpoints :
//   GET /api/v1/players/{player_slug}/pages/citations        → CitationsPageResponse
//   GET /api/v1/players/{player_slug}/pages/commendations    → CommendationsPageResponse
package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/config"
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
// GET /api/v1/players/{player_slug}/pages/citations
func (h *CitationsHandler) GetCitations(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	repo := duckdb.NewCitationsRepo(pdb)
	svc := service.NewCitationsService(repo)

	page, err := svc.GetCitationsPage(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "citations_page_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, page)
}

// GetCommendations retourne la page Commendations (médailles gagnées).
// GET /api/v1/players/{player_slug}/pages/commendations
func (h *CitationsHandler) GetCommendations(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	repo := duckdb.NewCitationsRepo(pdb)
	svc := service.NewCitationsService(repo)

	page, err := svc.GetCommendationsPage(r.Context(), pdb.XUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "commendations_page_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, page)
}

// resolvePlayer traduit le slug URL en PlayerDB.
func (h *CitationsHandler) resolvePlayer(r *http.Request) (*duckdb.PlayerDB, error) {
	slug := chi.URLParam(r, "player_slug")
	return config.ResolvePlayer(r.Context(), h.cfg, slug)
}
