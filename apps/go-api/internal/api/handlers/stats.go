// Package handlers — StatsHandler : endpoint des séries temporelles et stats analytiques.
package handlers

import (
	"encoding/json"
	"net/http"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"

	"github.com/go-chi/chi/v5"
)

// StatsHandler gère les requêtes de stats/séries analytiques.
type StatsHandler struct {
	newSvc ServiceFactory[port.StatsService]
}

// NewStatsHandler crée un StatsHandler.
func NewStatsHandler(newSvc ServiceFactory[port.StatsService]) *StatsHandler {
	return &StatsHandler{newSvc: newSvc}
}

// GetPage traite POST /api/v1/players/{player_slug}/pages/stats/query.
//
// Corps JSON : { "tab": "win_loss|accuracy|objective|form|lusr|all", "mode": "period|sessions" }
func (h *StatsHandler) GetPage(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "player_slug")
	svc, err := h.newSvc(r.Context(), slug)
	if err != nil {
		writeError(w, http.StatusBadRequest, "player_not_found", "joueur introuvable")
		return
	}

	var req domain.StatsQueryRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "corps JSON invalide")
		return
	}
	if req.Tab == "" {
		req.Tab = "win_loss"
	}

	resp, svcErr := svc.GetPage(r.Context(), req)
	if svcErr != nil {
		writeError(w, http.StatusInternalServerError, "stats_error", "erreur chargement stats")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
