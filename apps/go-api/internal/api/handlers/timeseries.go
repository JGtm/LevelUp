// Package handlers — TimeseriesHandler : POST /pages/timeseries.
//
// Sprint 33 — contrat API Lot 4.
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

// TimeseriesHandler gère POST /pages/timeseries.
type TimeseriesHandler struct {
	cfg *config.AppConfig
}

// NewTimeseriesHandler crée un TimeseriesHandler.
func NewTimeseriesHandler(cfg *config.AppConfig) *TimeseriesHandler {
	return &TimeseriesHandler{cfg: cfg}
}

// GetPage traite POST /api/v1/players/{player_slug}/pages/timeseries.
func (h *TimeseriesHandler) GetPage(w http.ResponseWriter, r *http.Request) {
	pdb, err := h.resolvePlayer(r)
	if err != nil {
		writeError(w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.TimeseriesQueryRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "corps JSON invalide")
		return
	}

	repo := duckdb.NewStatsRepo(pdb)
	svc := service.NewTimeseriesService(repo)

	resp, svcErr := svc.GetPage(r.Context(), req)
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, "timeseries_error", svcErr.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *TimeseriesHandler) resolvePlayer(r *http.Request) (*duckdb.PlayerDB, error) {
	slug := chi.URLParam(r, "player_slug")
	return config.ResolvePlayer(r.Context(), h.cfg, slug)
}
