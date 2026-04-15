// Package handlers — squad.go : handlers HTTP pour les pages Escouade et Synthèse.
//
// Endpoints :
//   GET /api/v1/players/{player_slug}/pages/squad             → SquadPageResponse
//   GET /api/v1/players/{player_slug}/pages/squad?teammate=xuid
//   GET /api/v1/players/{player_slug}/pages/synthesis        → SynthesisPageResponse
package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/service"
)

// SquadHandler gère les endpoints de la page Escouade et Synthèse.
type SquadHandler struct {
	cfg *config.AppConfig
}

// NewSquadHandler crée un SquadHandler.
func NewSquadHandler(cfg *config.AppConfig) *SquadHandler {
	return &SquadHandler{cfg: cfg}
}

// GetSquadPage retourne la page Escouade.
// GET /api/v1/players/{player_slug}/pages/squad[?teammate=xuid]
func (h *SquadHandler) GetSquadPage(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	teammateXUID := r.URL.Query().Get("teammate")

	repo := duckdb.NewSquadRepo(pdb)
	svc := service.NewSquadService(repo)

	page, err := svc.GetSquadPage(r.Context(), pdb.XUID, pdb.Gamertag, teammateXUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "squad_page_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, page)
}

// GetSynthesisPage retourne la page Synthèse (heatmap + top semaines).
// GET /api/v1/players/{player_slug}/pages/synthesis
func (h *SquadHandler) GetSynthesisPage(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	repo := duckdb.NewSquadRepo(pdb)
	svc := service.NewSquadService(repo)

	page, err := svc.GetSynthesisPage(r.Context(), pdb.XUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "synthesis_page_error", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, page)
}

// resolvePlayer traduit le slug URL en PlayerDB.
func (h *SquadHandler) resolvePlayer(r *http.Request) (*duckdb.PlayerDB, error) {
	slug := chi.URLParam(r, "player_slug")
	return config.ResolvePlayer(r.Context(), h.cfg, slug)
}
