// Package handlers — TimeseriesHandler : POST /pages/timeseries.
//
// Sprint 33 — contrat API Lot 4.
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// TimeseriesHandler gère POST /pages/timeseries.
type TimeseriesHandler struct {
	newSvc ServiceFactory[port.TimeseriesService]
}

// NewTimeseriesHandler crée un TimeseriesHandler.
func NewTimeseriesHandler(newSvc ServiceFactory[port.TimeseriesService]) *TimeseriesHandler {
	return &TimeseriesHandler{newSvc: newSvc}
}

// GetPage traite POST /api/v1/players/{player_slug}/pages/timeseries.
func (h *TimeseriesHandler) GetPage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), w, http.StatusNotFound, "player_not_found", err.Error())
		return
	}

	var req domain.TimeseriesQueryRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(r.Context(), w, http.StatusBadRequest, "invalid_json", "corps JSON invalide")
		return
	}

	resp, svcErr := svc.GetPage(r.Context(), req)
	if svcErr != nil {
		slog.ErrorContext(r.Context(), "timeseries: erreur service", "player", slug, "err", svcErr)
		writeError(r.Context(), w, http.StatusInternalServerError, "timeseries_error", svcErr.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
