// Package handlers — FiltersHandler : POST /api/v1/players/{player_slug}/filters/resolve.
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

// FiltersHandler gère POST .../filters/resolve.
type FiltersHandler struct {
	cfg *config.AppConfig
}

// NewFiltersHandler crée un FiltersHandler.
func NewFiltersHandler(cfg *config.AppConfig) *FiltersHandler {
	return &FiltersHandler{cfg: cfg}
}

// Resolve applique le filtre et retourne les options disponibles.
func (h *FiltersHandler) Resolve(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	pdb, err := config.ResolvePlayer(r.Context(), h.cfg, slug)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var input domain.FilterContextInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	repo := duckdb.NewFiltersRepo(pdb)
	svc := service.NewFiltersService(repo)
	result, err := svc.Resolve(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "filters_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
